package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"tkbastion/pkg/constant"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/pkg/totp"
	"tkbastion/pkg/validator"
	"tkbastion/server/model"
	"tkbastion/server/repository"
	"tkbastion/server/service"
	"tkbastion/server/utils"

	"github.com/go-ldap/ldap/v3"

	"github.com/labstack/echo/v4"
)

type LoginAccount struct {
	Username string `json:"username"      label:"账号"  validate:"required,max=64"                               `
	Password string `json:"password"      label:"密码"  validate:"required,max=64"                               `
	Remember bool   `json:"remember"`
	TOTP     string `json:"totp"`
}

type LoginAccountWithAuth struct {
	Username     string `json:"username"      label:"账号"  validate:"required,max=64"                               `
	Password     string `json:"password"      label:"密码"  validate:"required,max=64"                               `
	AuthPassword string `json:"authPassword"      label:"认证密码"  validate:"required,max=64"                               `
	Remember     bool   `json:"remember"`
	TOTP         string `json:"totp"`
	// 指纹
	Fingerprint bool `json:"fingerprint"`
}

type ConfirmTOTP struct {
	Secret string `json:"secret"`
	TOTP   string `json:"totp"`
}

func LoginEndpointNew(c echo.Context) error {
	var loginAccount LoginAccount
	if err := c.Bind(&loginAccount); err != nil {
		log.Errorf("Bind Error: %v", err)
		return err
	}
	//数据校验
	if err := c.Validate(loginAccount); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	// 去掉前后空格
	loginAccount.Username = strings.TrimSpace(loginAccount.Username)
	userNew, err := userNewRepository.FindByName(loginAccount.Username)
	if err != nil {
		return NotFound(c, "用户名或密码错误")
	}

	protocol := "http"
	if c.IsTLS() {
		protocol = "https"
	}

	token := strings.Join([]string{utils.UUID(), utils.UUID(), utils.UUID(), utils.UUID()}, "")
	loginLog := model.LoginLog{
		ID:              token,
		UserId:          userNew.ID,
		Username:        userNew.Username,
		Nickname:        userNew.Nickname,
		ClientIP:        c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LoginTime:       utils.NowJsonTime(),
		Remember:        loginAccount.Remember,
		DepartmentName:  userNew.DepartmentName,
		DepartmentId:    userNew.DepartmentId,
		Protocol:        protocol,
		LoginType:       userNew.AuthenticationWay,
	}
	if userNew.Status == constant.Expiration {
		// 保存登录日志
		loginLog.LoginResult = "失败"
		loginLog.Description = "用户" + constant.Expiration

		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			return err
		}

		err = userAccessStatisticsRepository.AddUserAccessStatisticsByProtocol(context.TODO(), &userNew, constant.LOGIN, "", c.RealIP(), "失败", "用户"+constant.Expiration)
		if err != nil {
			log.Errorf("用户协议访问记录失败: %v", err.Error())
		}

		return Fail(c, -1, "用户已过期")
	}
	if userNew.Status == constant.Disable {
		// 保存登录日志
		loginLog.LoginResult = "失败"
		loginLog.Description = "用户" + constant.Disable

		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			return err
		}

		err = userAccessStatisticsRepository.AddUserAccessStatisticsByProtocol(context.TODO(), &userNew, constant.LOGIN, "", c.RealIP(), "失败", "用户"+constant.Disable)
		if err != nil {
			log.Errorf("用户协议访问记录失败: %v", err.Error())
		}

		return Fail(c, -1, "用户已禁用")
	}

	// 存储登录失败次数信息
	item, err := repository.IdentityConfigDao.FindLonginConfig()
	var loginFailCountKey string
	if item.LoginLockWay == "user" {
		loginFailCountKey = loginAccount.Username
	} else {
		loginFailCountKey = c.RealIP()
	}
	v, ok := global.Cache.Get(loginFailCountKey)
	if !ok {
		v = 0
	}
	count := v.(int)
	loginFailTime := item.AttemptTimes
	loginFailTimes := strconv.Itoa(loginFailTime)
	if count > loginFailTime-1 {
		// 保存登录日志
		loginLog.LoginResult = "失败"
		loginLog.Description = "连续登录失败数超过最大登录失败次数" + loginFailTimes

		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			return err
		}

		err = userAccessStatisticsRepository.AddUserAccessStatisticsByProtocol(context.TODO(), &userNew, constant.LOGIN, "", c.RealIP(), "失败", "连续登录失败数超过最大登录失败次数"+loginFailTimes)
		if err != nil {
			log.Errorf("用户协议访问记录失败: %v", err.Error())
		}

		if item.LoginLockWay == "ip" {
			if err := identityConfigRepository.RecordIp(c.RealIP()); err != nil {
				log.Errorf("RecordIp Error: %v", err.Error())
			}
			return Fail(c, -1, "连续登录失败数超过最大登录失败次数"+loginFailTimes+", 您的IP已被锁定")
		} else {
			return Fail(c, -1, "连续登录失败数超过最大登录失败次数"+loginFailTimes+", 您的账号已被锁定")
		}
	}

	// 远程登录地址检测
	ipWhiteList, err := propertyRepository.GetRemoteManageHost()
	if err != nil {
		log.Error("远程管理主机地址获取失败")
	}
	if len(ipWhiteList) > 0 {
		// 检测IP是否在白名单中，白名单包括IP和IP段
		ip := c.RealIP()
		if !utils.CheckIPInWhiteList(ip, ipWhiteList) {
			count++
			global.Cache.Set(loginFailCountKey, count, time.Minute*time.Duration(item.ContinuousTime))
			// 保存登录日志
			loginLog.LoginResult = "失败"
			loginLog.Description = "IP地址不在远程管理主机地址名单中"

			err = loginLogRepository.Create(&loginLog)
			if nil != err {
				return err
			}

			err = userAccessStatisticsRepository.AddUserAccessStatisticsByProtocol(context.TODO(), &userNew, constant.LOGIN, "", c.RealIP(), "失败", "IP地址不在远程管理主机地址名单中")
			if err != nil {
				log.Error("用户协议访问记录失败")
			}

			// TODO: 提示信息需要修改
			return Fail(c, -1, "IP不在白名单中")
		}
	}

	// 验证用户策略
	result, message := JudgeUserStrategy(userNew.ID, c.RealIP())
	if !result {
		count++
		global.Cache.Set(loginFailCountKey, count, time.Minute*time.Duration(item.ContinuousTime))
		// 保存登录日志
		loginLog.LoginResult = "失败"
		loginLog.Description = message
		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			return err
		}

		err = userAccessStatisticsRepository.AddUserAccessStatisticsByProtocol(context.TODO(), &userNew, constant.LOGIN, "", c.RealIP(), "失败", message)
		if err != nil {
			log.Errorf("用户协议访问记录失败: %v", err.Error())
		}

		return Fail(c, -1, message)
	}

	// 认证检测
	retCode, mess := isAllowLogin(c, loginAccount, userNew)
	switch retCode {
	case -1:
		// 认证失败
		count++
		global.Cache.Set(loginFailCountKey, count, time.Minute*time.Duration(item.ContinuousTime))
		err = userAccessStatisticsRepository.AddUserAccessStatisticsByProtocol(context.TODO(), &userNew, constant.LOGIN, "", c.RealIP(), "失败", mess)
		if err != nil {
			log.Errorf("用户协议访问记录失败: %v", err.Error())
		}
		return Fail(c, -1, mess)
	case -5:
		// 还需RADIUS认证
		return Fail(c, -5, mess)
	case -6:
		// 还需LDAP/AD认证
		return Fail(c, -6, mess)
	case -7:
		// 还需邮件认证
		return Fail(c, -7, mess)
	case -8:
		// 还需TOTP认证
		return Fail(c, -8, mess)
	case -9:
		// 需修改密码
		return Fail(c, -9, mess)
	case -11:
		// 需要指纹认证
		return Fail(c, -11, mess)
	}

	// 在线用户数限制检测
	isOver := onlineUserCountCheck(loginAccount.Username)
	if isOver {
		err = userAccessStatisticsRepository.AddUserAccessStatisticsByProtocol(context.TODO(), &userNew, constant.LOGIN, "", c.RealIP(), "失败", "在线用户数超过最大限制")
		if err != nil {
			log.Errorf("用户协议访问记录失败: %v", err.Error())
		}
		return Fail(c, -1, "此用户登录数已达到最大限制, 请稍后重试或联系系统管理员修改在线用户数限制")
	}

	err = LoginSuccessNew(c, loginAccount, userNew, token)
	if err != nil {
		return err
	}

	////TODO 密码更换周期, 显示用户密码过期时间
	expire, err := repository.IdentityConfigDao.FindPasswordConfig()
	if err != nil {
		log.Errorf("获取密码过期时间失败: %v", err.Error())
	}
	if expire != nil && expire.PasswordCycle != 0 {
		day := int(time.Now().Sub(userNew.PasswordUpdated.Time).Hours()) / 24
		if expire.PasswordCycle-day < 0 {
			return Fail(c, -9, "您的密码已过期，账号存在风险，请立即重置密码")
		}
		if expire.PasswordRemind != 0 && expire.PasswordCycle-day <= expire.PasswordRemind {
			if err := messageRepository.Create(&model.Message{
				ID:        utils.UUID(),
				ReceiveId: userNew.ID,
				Theme:     "密码过期提醒",
				Level:     "high",
				Content:   fmt.Sprintf("您的密码将在%d天后过期，请您及时前往个人中心修改密码", expire.PasswordCycle-day),
				Status:    false,
				Type:      constant.NoticeMessage,
				Created:   utils.NowJsonTime(),
			}); err != nil {
				log.Errorf("创建消息失败: %v", err.Error())
			}
		}
	}
	err = userAccessStatisticsRepository.AddUserAccessStatisticsByProtocol(context.TODO(), &userNew, constant.LOGIN, "", c.RealIP(), "成功", "登陆成功")
	if err != nil {
		log.Errorf("用户协议访问记录失败: %v", err.Error())
	}
	return Success(c, token)

}

func onlineUserCountCheck(userName string) bool {
	var onlineUserNameArr []string

	cacheM := global.Cache.Items()
	for k, v := range cacheM {
		if strings.Contains(k, constant.Token) {
			user := v.Object.(global.AuthorizationNew).UserNew
			onlineUserNameArr = append(onlineUserNameArr, user.Username)
		}
	}

	item, err := propertyRepository.FindByName("login-session-online-count")
	if nil != err {
		log.Errorf("DB Error: %v", err.Error())
		return true
	}
	onlineUserCountLimit, err := strconv.Atoi(item.Value)
	if nil != err {
		log.Errorf("Atoi Error: %v", err.Error())
		return true
	}

	count := 0
	for i := range onlineUserNameArr {
		if userName == onlineUserNameArr[i] {
			count++
		}
	}

	if count < onlineUserCountLimit || 0 == onlineUserCountLimit {
		return false
	}
	return true
}

func LoginWithAuthEndpointNew(c echo.Context) error {
	var loginAccountWithAuth LoginAccountWithAuth
	if err := c.Bind(&loginAccountWithAuth); err != nil {
		log.Errorf("Bind Error: %v", err)
		return err
	}
	//数据校验
	if err := c.Validate(loginAccountWithAuth); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}
	// 去掉前后空格
	loginAccountWithAuth.Username = strings.TrimSpace(loginAccountWithAuth.Username)
	userNew, err := userNewRepository.FindByName(loginAccountWithAuth.Username)
	if err != nil {
		return NotFound(c, "用户名或密码错误")
	}

	protocol := "http"
	if c.IsTLS() {
		protocol = "https"
	}

	token := strings.Join([]string{utils.UUID(), utils.UUID(), utils.UUID(), utils.UUID()}, "")
	loginLog := model.LoginLog{
		ID:              token,
		UserId:          userNew.ID,
		Username:        userNew.Username,
		Nickname:        userNew.Nickname,
		ClientIP:        c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LoginTime:       utils.NowJsonTime(),
		Remember:        false,
		DepartmentName:  userNew.DepartmentName,
		DepartmentId:    userNew.DepartmentId,
		Protocol:        protocol,
		LoginType:       userNew.AuthenticationWay,
	}
	if userNew.Status == constant.Expiration {
		// 保存登录日志
		loginLog.LoginResult = "失败"
		loginLog.Description = "用户" + constant.Expiration

		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			return err
		}

		err = userAccessStatisticsRepository.AddUserAccessStatisticsByProtocol(context.TODO(), &userNew, constant.LOGIN, "", c.RealIP(), "失败", "用户"+constant.Expiration)
		if err != nil {
			log.Errorf("用户协议访问记录失败: %v", err.Error())
		}

		return Fail(c, -1, "用户已过期")
	}
	if userNew.Status == constant.Disable {
		// 保存登录日志
		loginLog.LoginResult = "失败"
		loginLog.Description = "用户" + constant.Disable

		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			return err
		}

		err = userAccessStatisticsRepository.AddUserAccessStatisticsByProtocol(context.TODO(), &userNew, constant.LOGIN, "", c.RealIP(), "失败", "用户"+constant.Disable)
		if err != nil {
			log.Errorf("用户协议访问记录失败: %v", err.Error())
		}

		return Fail(c, -1, "用户已禁用")
	}

	// 存储登录失败次数信息
	item, err := repository.IdentityConfigDao.FindLonginConfig()
	var loginFailCountKey string
	if item.LoginLockWay == "user" {
		loginFailCountKey = loginAccountWithAuth.Username
	} else {
		loginFailCountKey = c.RealIP()
	}
	v, ok := global.Cache.Get(loginFailCountKey)
	if !ok {
		v = 0
	}
	count := v.(int)
	loginFailTime := item.AttemptTimes
	loginFailTimes := strconv.Itoa(loginFailTime)
	if count > loginFailTime-1 {
		// 保存登录日志
		loginLog.LoginResult = "失败"
		loginLog.Description = "连续登录失败数超过最大登录失败次数" + loginFailTimes

		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			return err
		}

		err = userAccessStatisticsRepository.AddUserAccessStatisticsByProtocol(context.TODO(), &userNew, constant.LOGIN, "", c.RealIP(), "失败", "连续登录失败数超过最大登录失败次数"+loginFailTimes)
		if err != nil {
			log.Errorf("用户协议访问记录失败: %v", err.Error())
		}

		if item.LoginLockWay == "ip" {
			if err := identityConfigRepository.RecordIp(c.RealIP()); err != nil {
				log.Errorf("RecordIp Error: %v", err.Error())
			}
			return Fail(c, -1, "连续登录失败数超过最大登录失败次数"+loginFailTimes+", 您的IP已被锁定")
		} else {
			return Fail(c, -1, "连续登录失败数超过最大登录失败次数"+loginFailTimes+", 您的账号已被锁定")
		}
	}

	// 远程登录地址检测
	ipWhiteList, err := propertyRepository.GetRemoteManageHost()
	if err != nil {
		log.Errorf("获取IP白名单失败: %v", err.Error())
	}
	if len(ipWhiteList) > 0 {
		// 检测IP是否在白名单中，白名单包括IP和IP段
		ip := c.RealIP()
		if !utils.CheckIPInWhiteList(ip, ipWhiteList) {
			count++
			global.Cache.Set(loginFailCountKey, count, time.Minute*time.Duration(item.ContinuousTime))
			// 保存登录日志
			loginLog.LoginResult = "失败"
			loginLog.Description = "IP不在白名单中"

			err = loginLogRepository.Create(&loginLog)
			if nil != err {
				return err
			}

			err = userAccessStatisticsRepository.AddUserAccessStatisticsByProtocol(context.TODO(), &userNew, constant.LOGIN, "", c.RealIP(), "失败", "IP不在白名单中")
			if err != nil {
				log.Errorf("用户协议访问记录失败: %v", err.Error())
			}

			return Fail(c, -1, "IP不在白名单中")
		}
	}

	// 验证用户策略
	result, message := JudgeUserStrategy(userNew.ID, c.RealIP())
	if !result {
		count++
		global.Cache.Set(loginFailCountKey, count, time.Minute*time.Duration(item.ContinuousTime))
		// 保存登录日志
		loginLog.LoginResult = "失败"
		loginLog.Description = message
		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			return err
		}

		err = userAccessStatisticsRepository.AddUserAccessStatisticsByProtocol(context.TODO(), &userNew, constant.LOGIN, "", c.RealIP(), "失败", message)
		if err != nil {
			log.Errorf("用户协议访问记录失败: %v", err.Error())
		}

		return Fail(c, -1, message)
	}

	// 认证检测
	retCode, mess := isAllowLoginWithAuth(c, loginAccountWithAuth, userNew)
	if -1 == retCode {
		// 认证失败
		count++
		global.Cache.Set(loginFailCountKey, count, time.Minute*time.Duration(item.ContinuousTime))

		err = userAccessStatisticsRepository.AddUserAccessStatisticsByProtocol(context.TODO(), &userNew, constant.LOGIN, "", c.RealIP(), "失败", mess)
		if err != nil {
			log.Errorf("用户协议访问记录失败: %v", err.Error())
		}

		return Fail(c, -1, mess)
	}

	// 在线用户数限制检测
	isOver := onlineUserCountCheck(loginAccountWithAuth.Username)
	if isOver {
		count++
		global.Cache.Set(loginFailCountKey, count, time.Minute*time.Duration(item.ContinuousTime))
		err = userAccessStatisticsRepository.AddUserAccessStatisticsByProtocol(context.TODO(), &userNew, constant.LOGIN, "", c.RealIP(), "失败", "在线用户数超过最大限制")
		if err != nil {
			log.Errorf("用户协议访问记录失败: %v", err.Error())
		}

		return Fail(c, -1, "此用户登录数已达到最大限制, 请稍后重试或联系系统管理员修改在线用户数限制")
	}

	err = LoginSuccessNew(c, LoginAccount{Username: loginAccountWithAuth.Username, Password: loginAccountWithAuth.Password, Remember: loginAccountWithAuth.Remember, TOTP: loginAccountWithAuth.TOTP}, userNew, token)
	if err != nil {
		return err
	}

	err = userAccessStatisticsRepository.AddUserAccessStatisticsByProtocol(context.TODO(), &userNew, constant.LOGIN, "", c.RealIP(), "成功", "TOTP认证成功")
	if err != nil {
		log.Errorf("用户协议访问记录失败: %v", err.Error())
	}

	////TODO 密码更换周期, 显示用户密码过期时间
	expire, err := repository.IdentityConfigDao.FindPasswordConfig()
	if err != nil {
		log.Errorf("获取密码过期时间失败: %v", err.Error())
	}
	if expire != nil && expire.PasswordCycle != 0 {
		day := int(time.Now().Sub(userNew.PasswordUpdated.Time).Hours()) / 24
		if expire.PasswordCycle-day < 0 {
			return Fail(c, -9, "您的密码已过期，账号存在风险，请立即重置密码")
		}
		if expire.PasswordRemind != 0 && expire.PasswordCycle-day <= expire.PasswordRemind {
			if err := messageRepository.Create(&model.Message{
				ID:        utils.UUID(),
				ReceiveId: userNew.ID,
				Theme:     "密码过期提醒",
				Level:     "high",
				Content:   fmt.Sprintf("您的密码将在%d天后过期，请您及时前往个人中心修改密码", expire.PasswordCycle-day),
				Status:    false,
				Type:      constant.NoticeMessage,
				Created:   utils.NowJsonTime(),
			}); err != nil {
				log.Errorf("创建消息失败: %v", err.Error())
			}
		}
	}
	return Success(c, token)
}

// UserLoginTypeEndpoint 1. 最初的登录样式->不变
// 2. 3个框->最初的登录样式
func UserLoginTypeEndpoint(c echo.Context) error {
	userName := c.QueryParam("userName")
	user, err := userNewRepository.FindByName(userName)
	if nil != err {
		return FailWithDataOperate(c, -1, "", "", nil)
	}

	if "邮件认证" == user.AuthenticationWay {
		verifyMailId := utils.UUID()
		verifyMailCode := rand.New(rand.NewSource(time.Now().UnixNano())).Int31n(10000)
		global.Cache.Set(verifyMailId, verifyMailCode, time.Minute*time.Duration(5))

		err = sysConfigService.SendMail([]string{user.Mail}, "[Tkbastion] 邮件认证", "您的认证码为: "+strconv.Itoa(int(verifyMailCode))+", 请在有效期5分钟内输入认证码进行登录认证, 过期后需重新接收新的认证码.")
		if nil != err {
			log.Errorf("SendMail Error: %v", err.Error())
			return FailWithDataOperate(c, 1, "邮件认证码发送失败, 请联系系统管理员查看邮件配置是否正确", "", verifyMailId)
		}

		return FailWithDataOperate(c, 1, "我们已向您的邮箱"+user.Mail+"发送了认证码, 请您注意查收并在5分钟内使用该认证码进行登录认证", "", verifyMailId)
	} else if "TOTP认证" == user.AuthenticationWay {
		return FailWithDataOperate(c, 2, "请输入TOTP APP上显示的认证码", "", nil)
	} else if "RADIUS认证" == user.AuthenticationWay {
		return FailWithDataOperate(c, 3, "请输入RADIUS认证密码", "", nil)
	} else if "LDAP/AD认证" == user.AuthenticationWay {
		return FailWithDataOperate(c, 4, "请输入LDAP/AD认证密码", "", nil)
	}

	return FailWithDataOperate(c, -1, "", "", nil)
}

// 认证并记录认证失败时登录日志
// 返回  0 代表允许登录
// 返回 -1 代表认证(静态密码、RADIUS、LDAP/AD、邮件、TOTP)错误, 需更新记录登录失败次数的count值
// 返回 -5 代表此用户还需进行RADIUS认证
// 返回 -6 代表此用户还需进行LDAP/AD认证
// 返回 -7 代表此用户还需进行邮件认证
// 返回 -8 代表此用户还需进行TOTP认证
// 返回 -9 代表此用户是首次登录/或密码到期，需要进行密码修改
func isAllowLogin(c echo.Context, loginAccount LoginAccount, userNew model.UserNew) (retCode int, message string) {
	passConfig, _ := repository.IdentityConfigDao.FindPasswordConfig()
	switch userNew.AuthenticationWay {
	case "静态密码":
		if userNew.LoginNumbers == 0 && *passConfig.ForceChangePassword {
			return -9, "首次登录, 请您修改密码后重新登录"
		}
		return password(c, loginAccount, userNew)
	case "RADIUS认证":
		return radius(c, loginAccount, userNew)
	case "LDAP/AD认证":
		return ldapAd(c, loginAccount, userNew)
	case "邮件认证":
		return mail(c, loginAccount, userNew)
	case "TOTP认证":
		return totpAuth(c, loginAccount, userNew)
	case "RADIUS认证+静态密码":
		if userNew.LoginNumbers == 0 && *passConfig.ForceChangePassword {
			return -9, "首次登录, 请您修改密码后重新登录"
		}
		retCode, message = password(c, loginAccount, userNew)
		if -1 == retCode {
			return
		}

		return -5, "请输入RADIUS认证密码"
	case "LDAP/AD认证+静态密码":
		if userNew.LoginNumbers == 0 && *passConfig.ForceChangePassword {
			return -9, "首次登录, 请您修改密码后重新登录"
		}
		retCode, message = password(c, loginAccount, userNew)
		if -1 == retCode {
			return
		}
		return -6, "请输入LDAP/AD认证密码"
	case "邮件认证+静态密码":
		if userNew.LoginNumbers == 0 && *passConfig.ForceChangePassword {
			return -9, "首次登录, 请您修改密码后重新登录"
		}
		retCode, message = password(c, loginAccount, userNew)
		if -1 == retCode {
			return
		}

		verifyMailId := utils.UUID()
		verifyMailCode := rand.New(rand.NewSource(time.Now().UnixNano())).Int31n(10000)
		global.Cache.Set(verifyMailId, verifyMailCode, time.Minute*time.Duration(5))

		err := sysConfigService.SendMail([]string{userNew.Mail}, "[Tkbastion] 邮件认证", "您的认证码为: "+strconv.Itoa(int(verifyMailCode))+", 请在有效期5分钟内输入认证码进行登录认证, 过期后需重新接收新的认证码.")
		if nil != err {
			log.Errorf("SendMail Error: %v", err.Error())
			return -1, "邮件认证码发送失败, 请联系系统管理员查看邮件配置是否正确"
		}

		return -7, verifyMailId + "~我们已向您的邮箱" + userNew.Mail + "发送了认证码, 请您注意查收并在5分钟内使用该认证码进行登录认证"
	case "TOTP认证+静态密码":
		if userNew.LoginNumbers == 0 && *passConfig.ForceChangePassword {
			return -9, "首次登录, 请您修改密码后重新登录"
		}
		retCode, message = password(c, loginAccount, userNew)
		if -1 == retCode {
			return
		}

		return -8, "请输入TOTP APP上显示的认证码进行登录认证"
	case "指纹认证+静态密码":
		if userNew.LoginNumbers == 0 && *passConfig.ForceChangePassword {
			return -9, "首次登录, 请您修改密码后重新登录"
		}
		retCode, message = password(c, loginAccount, userNew)
		if -1 == retCode {
			return
		}

		return -11, "请使用指纹进行登录认证"

	}
	return -10, "你们不会看到我"
}

// 认证并记录认证失败时登录日志
// 返回  0 代表允许登录
// 返回 -1 代表认证(静态密码、RADIUS、LDAP/AD、邮件、TOTP)错误, 需更新记录登录失败次数的count值
func isAllowLoginWithAuth(c echo.Context, loginAccountWithAuth LoginAccountWithAuth, userNew model.UserNew) (retCode int, message string) {
	switch userNew.AuthenticationWay {
	case "RADIUS认证+静态密码":
		retCode, message = password(c, LoginAccount{Username: loginAccountWithAuth.Username, Password: loginAccountWithAuth.Password, Remember: loginAccountWithAuth.Remember, TOTP: loginAccountWithAuth.TOTP}, userNew)
		if 0 != retCode {
			return
		}
		return radius(c, LoginAccount{Username: loginAccountWithAuth.Username, Password: loginAccountWithAuth.AuthPassword, Remember: loginAccountWithAuth.Remember, TOTP: loginAccountWithAuth.TOTP}, userNew)
	case "LDAP/AD认证+静态密码":
		retCode, message = password(c, LoginAccount{Username: loginAccountWithAuth.Username, Password: loginAccountWithAuth.Password, Remember: loginAccountWithAuth.Remember, TOTP: loginAccountWithAuth.TOTP}, userNew)
		if 0 != retCode {
			return
		}
		return ldapAd(c, LoginAccount{Username: loginAccountWithAuth.Username, Password: loginAccountWithAuth.AuthPassword, Remember: loginAccountWithAuth.Remember, TOTP: loginAccountWithAuth.TOTP}, userNew)
	case "邮件认证+静态密码":
		retCode, message = password(c, LoginAccount{Username: loginAccountWithAuth.Username, Password: loginAccountWithAuth.Password, Remember: loginAccountWithAuth.Remember, TOTP: loginAccountWithAuth.TOTP}, userNew)
		if 0 != retCode {
			return
		}
		return mail(c, LoginAccount{Username: loginAccountWithAuth.Username, Password: loginAccountWithAuth.AuthPassword, Remember: loginAccountWithAuth.Remember, TOTP: loginAccountWithAuth.TOTP}, userNew)
	case "TOTP认证+静态密码":
		retCode, message = password(c, LoginAccount{Username: loginAccountWithAuth.Username, Password: loginAccountWithAuth.Password, Remember: loginAccountWithAuth.Remember, TOTP: loginAccountWithAuth.TOTP}, userNew)
		if 0 != retCode {
			return
		}
		return totpAuth(c, LoginAccount{Username: loginAccountWithAuth.Username, Password: loginAccountWithAuth.AuthPassword, Remember: loginAccountWithAuth.Remember, TOTP: loginAccountWithAuth.TOTP}, userNew)
	case "指纹认证+静态密码":
		return 0, ""
	}
	return 1024, "你们不会看到我"
}

func mail(c echo.Context, loginAccount LoginAccount, userNew model.UserNew) (retCode int, message string) {
	get, ok := global.Cache.Get(c.QueryParam("verifyMailId"))
	if !ok {
		saveLoginFailLog(c, userNew, constant.ExpireVerifyMailCode)
		verifyMailId := c.QueryParam("verifyMailId")
		verifyMailCode := rand.New(rand.NewSource(time.Now().UnixNano())).Int31n(10000)
		global.Cache.Set(verifyMailId, verifyMailCode, time.Minute*time.Duration(5))

		err := sysConfigService.SendMail([]string{userNew.Mail}, "[Tkbastion] 邮件认证", "您的认证码为: "+strconv.Itoa(int(verifyMailCode))+", 请在有效期5分钟内输入认证码进行登录认证, 过期后需重新接收新的认证码.")
		if nil != err {
			log.Errorf("SendMail Error: %v", err.Error())
			return -1, "邮件认证码已过期, 新的认证码发送失败, 请联系系统管理员查看邮件配置是否正确"
		}

		return -1, "邮件认证码已过期, 我们已向您的邮箱" + userNew.Mail + "发送了新的认证码, 请注意查收并使用新的认证码进行登录认证"
	}
	verifyMailCode := get.(int32)
	code, err := strconv.Atoi(loginAccount.Password)
	if nil != err {
		log.Errorf("Atoi Error: %v", err.Error())
		return -1, "邮件认证失败"
	}
	if verifyMailCode != int32(code) {
		saveLoginFailLog(c, userNew, constant.FailVerifyMailCode)
		return -1, constant.FailVerifyMailCode
	}

	return 0, ""
}

func totpAuth(c echo.Context, loginAccount LoginAccount, userNew model.UserNew) (retCode int, message string) {
	if !totp.Validate(loginAccount.Password, userNew.TOTPSecret) {
		saveLoginFailLog(c, userNew, constant.FailTwoFactorAuthToken)
		return -1, constant.FailTwoFactorAuthToken
	}

	return 0, ""
}

func ldapAd(c echo.Context, loginAccount LoginAccount, userNew model.UserNew) (retCode int, message string) {
	item, err := ldapAdAuthRepository.FindById(userNew.AuthenticationServerId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return -1, "LDAP/AD认证失败"
	}

	if "ldap" == item.LdapAdType {
		return ldapAuth(c, loginAccount, userNew, item)
	}

	return adAuth(c, loginAccount, userNew, item)
}

func ldapAuth(c echo.Context, loginAccount LoginAccount, userNew model.UserNew, item model.LdapAdAuth) (retCode int, message string) {
	conn, err := ldap.DialURL("ldap://" + item.LdapAdServerAddress + ":" + strconv.Itoa(item.LdapAdPort))
	if err != nil {
		log.Errorf("DialURL Error: %v", err)
		saveLoginFailLog(c, userNew, "LDAP/AD认证失败")
		return -1, "LDAP/AD认证失败"
	}
	defer conn.Close()

	if "true" == item.LdapAdTls {
		err = conn.StartTLS(&tls.Config{InsecureSkipVerify: true})
		if err != nil {
			log.Errorf("StartTLS Error: %v", err)
			saveLoginFailLog(c, userNew, "LDAP/AD认证失败")
			return -1, "LDAP/AD认证失败"
		}
	}

	err = conn.Bind(userNew.Dn, loginAccount.Password)
	if err != nil {
		log.Errorf("Bind Error: %v", err)
		saveLoginFailLog(c, userNew, "LDAP/AD认证失败")
		return -1, "LDAP/AD认证失败"
	}

	return 0, ""
}

func adAuth(c echo.Context, loginAccount LoginAccount, userNew model.UserNew, item model.LdapAdAuth) (retCode int, message string) {
	conn, err := ldap.DialURL("ldap://" + item.LdapAdServerAddress + ":" + strconv.Itoa(item.LdapAdPort))
	if err != nil {
		log.Errorf("DialURL Error: %v", err)
		saveLoginFailLog(c, userNew, "LDAP/AD认证失败")
		return -1, "LDAP/AD认证失败"
	}
	defer conn.Close()

	if "true" == item.LdapAdTls {
		err = conn.StartTLS(&tls.Config{InsecureSkipVerify: true})
		if err != nil {
			log.Errorf("StartTLS Error: %v", err)
			saveLoginFailLog(c, userNew, "LDAP/AD认证失败")
			return -1, "LDAP/AD认证失败"
		}
	}

	err = conn.Bind(loginAccount.Username+"@"+item.LdapAdDomain, loginAccount.Password)
	if err != nil {
		log.Errorf("Bind Error: %v", err)
		return -1, "LDAP/AD认证失败"
	}

	return 0, ""
}

func password(c echo.Context, loginAccount LoginAccount, userNew model.UserNew) (retCode int, message string) {
	if err := utils.Encoder.Match([]byte(userNew.Password), []byte(loginAccount.Password)); err != nil {
		saveLoginFailLog(c, userNew, constant.FailPassword)
		return -1, constant.FailPassword
	}
	return 0, ""
}

func radius(c echo.Context, loginAccount LoginAccount, userNew model.UserNew) (retCode int, message string) {
	item := propertyRepository.FindAuMap("radius")

	//radtest -t chap ldj 123456 192.168.28.179 1812 admin
	radiusAuthCommand := "radtest" + " -t " + item["radius-auth-protocol"] + " " + loginAccount.Username + " " + loginAccount.Password + " " + item["radius-server-address"] + " " + item["radius-port"] + " " + item["radius-auth-share-secret"]
	authResult, err := utils.ExecShell(radiusAuthCommand)
	if err != nil {
		log.Errorf("Exec_shell Error: %v", err)
		return -1, "RADIUS认证失败"
	}
	if strings.Contains(authResult, "Received Access-Accept") {
		return 0, ""
	}
	saveLoginFailLog(c, userNew, "RADIUS认证失败")
	return -1, "RADIUS认证失败"
}

func saveLoginFailLog(c echo.Context, userNew model.UserNew, description string) {
	protocol := "http"
	if c.IsTLS() {
		protocol = "https"
	}

	loginLog := model.LoginLog{
		ID:              strings.Join([]string{utils.UUID(), utils.UUID(), utils.UUID(), utils.UUID()}, ""),
		UserId:          userNew.ID,
		Username:        userNew.Username,
		Nickname:        userNew.Nickname,
		ClientIP:        c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LoginTime:       utils.NowJsonTime(),
		Remember:        false,
		DepartmentName:  userNew.DepartmentName,
		DepartmentId:    userNew.DepartmentId,
		Protocol:        protocol,
		LoginType:       userNew.AuthenticationWay,
		LoginResult:     "失败",
		Description:     description,
	}

	err := loginLogRepository.Create(&loginLog)
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}
}

func LoginSuccessNew(c echo.Context, loginAccount LoginAccount, userNew model.UserNew, token string) error {
	var err error
	loginTime := utils.NowJsonTime()
	authorizationNew := global.AuthorizationNew{
		Token:          token,
		Remember:       loginAccount.Remember,
		UserNew:        userNew,
		LoginTime:      loginTime,
		LastActiveTime: loginTime,
		LoginAddress:   c.RealIP(),
	}

	cacheKey := BuildCacheKeyByToken(token)

	item, err := propertyRepository.FindByName("login-session-overtime")
	if nil != err {
		log.Errorf("DB Error: %v", err.Error())
	}
	iExpirationTime, err := strconv.Atoi(item.Value)
	if nil != err {
		log.Errorf("Atoi Error: %v", err.Error())
	}
	rememberEffectiveTime := time.Minute * time.Duration(iExpirationTime)
	global.Cache.Set(cacheKey, authorizationNew, rememberEffectiveTime)

	protocol := "http"
	if c.IsTLS() {
		protocol = "https"
	}

	// 保存登录日志
	loginLog := model.LoginLog{
		ID:              token,
		UserId:          userNew.ID,
		Username:        userNew.Username,
		Nickname:        userNew.Nickname,
		ClientIP:        c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LoginTime:       loginTime,
		Remember:        loginAccount.Remember,
		DepartmentName:  userNew.DepartmentName,
		DepartmentId:    userNew.DepartmentId,
		Source:          userNew.AuthenticationWay,
		LoginResult:     constant.LoginSuccess,
		Protocol:        protocol,
		Description:     "登录成功",
		LoginType:       userNew.AuthenticationWay,
	}

	if loginLogRepository.Create(&loginLog) != nil {
		return err
	}

	// 修改登录状态和用户登录次数
	err = userNewRepository.UpdateStructById(model.UserNew{Online: true, LoginNumbers: userNew.LoginNumbers + 1}, userNew.ID)
	if nil != err {
		return err
	}

	return err
}

func LoginSuccessMail(c echo.Context, loginAccount LoginAccountMail, user model.UserNew, token string) error {
	var err error
	authorization := global.AuthorizationNew{
		Token:    token,
		Remember: loginAccount.Remember,
		UserNew:  user,
	}

	cacheKey := BuildCacheKeyByToken(token)
	item, err := repository.IdentityConfigDao.FindLonginConfig()
	expirationTime := item.LockTime
	RememberEffectiveTime := time.Minute * time.Duration(expirationTime)
	global.Cache.Set(cacheKey, authorization, RememberEffectiveTime)

	// 保存登录日志
	loginLog := model.LoginLog{
		ID:              token,
		UserId:          user.ID,
		ClientIP:        c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LoginTime:       utils.NowJsonTime(),
		Remember:        authorization.Remember,
		LoginResult:     constant.LoginSuccess,
	}

	if loginLogRepository.Create(&loginLog) != nil {
		return err
	}

	// 修改登录状态,更新该用户登录次数
	err = userNewRepository.UpdateStructById(model.UserNew{Online: true, LoginNumbers: user.LoginNumbers + 1}, user.ID)
	if nil != err {
		return err
	}

	return err
}

func BuildCacheKeyByToken(token string) string {
	cacheKey := strings.Join([]string{constant.Token, token}, ":")
	return cacheKey
}

func GetTokenFormCacheKey(cacheKey string) string {
	token := strings.Split(cacheKey, ":")[1]
	return token
}

func loginWithTotpEndpoint(c echo.Context) error {
	var loginAccount LoginAccount
	if err := c.Bind(&loginAccount); err != nil {
		return err
	}
	// 去除空格
	loginAccount.Username = strings.TrimSpace(loginAccount.Username)
	user, err := userNewRepository.FindByName(loginAccount.Username)
	token := strings.Join([]string{utils.UUID(), utils.UUID(), utils.UUID(), utils.UUID()}, "")
	protocol := "http"
	if c.IsTLS() {
		protocol = "https"
	}
	loginLog := model.LoginLog{
		ID:              token,
		UserId:          user.ID,
		ClientIP:        c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LoginTime:       utils.NowJsonTime(),
		Remember:        loginAccount.Remember,
		Nickname:        user.Nickname,
		DepartmentName:  user.DepartmentName,
		DepartmentId:    user.DepartmentId,
		Protocol:        protocol,
		LoginType:       user.AuthenticationWay,
	}

	// 存储登录失败次数信息
	loginFailCountKey := loginAccount.Username
	v, ok := global.Cache.Get(loginFailCountKey)
	if !ok {
		v = 0
	}
	count := v.(int)
	item, err := repository.IdentityConfigDao.FindLonginConfig()
	loginFailTime := item.LockTime
	loginFailTimes := strconv.Itoa(loginFailTime)
	if count > loginFailTime-1 {
		// 保存登录日志
		loginLog.LoginResult = "失败:" + "登录失败数超过最大登录失败次数" + loginFailTimes

		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			return err
		}

		return Fail(c, -1, "登录失败次数过多,请稍后再试")
	}

	if err != nil {
		count++
		global.Cache.Set(loginFailCountKey, count, time.Minute*time.Duration(item.ContinuousTime))

		// 保存登录日志
		loginLog.LoginResult = "失败:" + constant.FailUserName

		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			return err
		}
		return FailWithData(c, -1, "您输入的账号或密码不正确", count)
	}

	if err := utils.Encoder.Match([]byte(user.Password), []byte(loginAccount.Password)); err != nil {
		count++
		global.Cache.Set(loginFailCountKey, count, time.Minute*time.Duration(item.ContinuousTime))

		// 保存登录日志
		loginLog.LoginResult = "失败:" + constant.FailPassword

		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			return err
		}
		return FailWithData(c, -1, "您输入的账号或密码不正确", count)
	}

	if !totp.Validate(loginAccount.TOTP, user.TOTPSecret) {
		count++
		global.Cache.Set(loginFailCountKey, count, time.Minute*time.Duration(item.ContinuousTime))

		// 保存登录日志
		loginLog.LoginResult = "失败:" + constant.FailTwoFactorAuthToken

		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			return err
		}
		return FailWithData(c, -1, "您输入双因素认证授权码不正确", count)
	}

	// 邮箱验证开启则发送邮箱验证信息
	if *user.VerifyMailState {
		verifyMailId := utils.UUID()
		verifyMailCode := rand.New(rand.NewSource(time.Now().UnixNano())).Int31n(10000)
		global.Cache.Set(verifyMailId, verifyMailCode, time.Minute*time.Duration(5))
		propertiesMap := propertyRepository.FindAllMap()
		host := propertiesMap[constant.MailHost]
		port := propertiesMap[constant.MailPort]
		username := propertiesMap[constant.MailUsername]
		password := propertiesMap[constant.MailPassword]
		err := mailService.NewSendMail(host, port, username, password, []string{user.Mail}, "[Tkbastion] 邮箱认证", "您的认证码为:"+strconv.Itoa(int(verifyMailCode))+",请在有效期5分钟内输入认证码,过期后需重新执行登录操作.")
		if nil != err {
			log.Errorf("NewSendMail Error: %v", err)
			return FailWithDataOperate(c, 500, "登录失败", "", err)
		}

		return FailWithData(c, 2, "我们已向您的邮箱"+user.Mail+"发送了验证码,请您注意查收并在5分钟内将验证码填写到下面的输入框进行邮箱验证", verifyMailId)
	}

	// 验证用户策略
	result, message := JudgeUserStrategy(user.ID, c.RealIP())
	if !result {
		// 保存登录日志
		loginLog.LoginResult = "失败"
		loginLog.Description = message
		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			return err
		}
		return Fail(c, -1, message)
	}

	err = LoginSuccessNew(c, loginAccount, user, token)
	if nil != err {
		return err
	}
	////TODO 密码更换周期, 显示用户密码过期时间
	expire, err := repository.IdentityConfigDao.FindPasswordConfig()
	if err != nil {
		log.Errorf("获取密码过期时间失败: %v", err.Error())
	}
	if expire != nil && expire.PasswordCycle != 0 {
		day := int(time.Now().Sub(user.PasswordUpdated.Time).Hours()) / 24
		if expire.PasswordCycle-day < 0 {
			return Fail(c, -9, "您的密码已过期，账号存在风险，请立即重置密码")
		}
		if expire.PasswordRemind != 0 && expire.PasswordCycle-day <= expire.PasswordRemind {
			if err := messageRepository.Create(&model.Message{
				ID:        utils.UUID(),
				ReceiveId: user.ID,
				Theme:     "密码过期提醒",
				Level:     "high",
				Content:   fmt.Sprintf("您的密码将在%d天后过期，请您及时前往个人中心修改密码", expire.PasswordCycle-day),
				Status:    false,
				Type:      constant.NoticeMessage,
				Created:   utils.NowJsonTime(),
			}); err != nil {
				log.Errorf("创建消息失败: %v", err.Error())
			}
		}
	}
	return Success(c, token)
}

type LoginAccountMail struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	Remember       bool   `json:"remember"`
	VerifyMailId   string `json:"verifyMailId"`
	VerifyMailCode string `json:"verifyMailCode"`
}

func loginWithMailEndpoint(c echo.Context) error {
	var loginAccountMail LoginAccountMail
	if err := c.Bind(&loginAccountMail); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, -1, "登录失败", "", err)
	}
	// 去除空格
	loginAccountMail.Username = strings.TrimSpace(loginAccountMail.Username)
	user, err := userNewRepository.FindByName(loginAccountMail.Username)
	token := strings.Join([]string{utils.UUID(), utils.UUID(), utils.UUID(), utils.UUID()}, "")
	protocol := "http"
	if c.IsTLS() {
		protocol = "https"
	}
	loginLog := model.LoginLog{
		ID:              token,
		UserId:          user.ID,
		ClientIP:        c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LoginTime:       utils.NowJsonTime(),
		Remember:        loginAccountMail.Remember,
		Nickname:        user.Nickname,
		DepartmentName:  user.DepartmentName,
		DepartmentId:    user.DepartmentId,
		Protocol:        protocol,
		LoginType:       user.AuthenticationWay,
	}
	// 存储登录失败次数信息
	loginFailCountKey := loginAccountMail.Username
	v, ok := global.Cache.Get(loginFailCountKey)
	if !ok {
		v = 0
	}
	count := v.(int)
	item, err := repository.IdentityConfigDao.FindLonginConfig()
	loginFailTime := item.LockTime
	MoreThanFailNumber := strconv.Itoa(loginFailTime)
	if count > loginFailTime-1 {
		// 保存登录日志
		loginLog.LoginResult = "失败:" + "登录失败数超过最大登录失败次数" + MoreThanFailNumber

		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			return err
		}

		return Fail(c, -1, "登录失败次数过多,请稍后再试")
	}

	if err != nil {
		count++
		global.Cache.Set(loginFailCountKey, count, time.Minute*time.Duration(item.ContinuousTime))

		// 保存登录日志
		loginLog.LoginResult = "失败:" + constant.FailUserName

		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			return err
		}
		return FailWithData(c, -1, "您输入的账号或密码不正确", count)
	}

	if err := utils.Encoder.Match([]byte(user.Password), []byte(loginAccountMail.Password)); err != nil {
		count++
		global.Cache.Set(loginFailCountKey, count, time.Minute*time.Duration(item.ContinuousTime))

		// 保存登录日志
		loginLog.LoginResult = "失败:" + constant.FailPassword

		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			return err
		}
		return FailWithData(c, -1, "您输入的账号或密码不正确", count)
	}

	get, ok := global.Cache.Get(loginAccountMail.VerifyMailId)
	if !ok {
		// 保存登录日志
		loginLog.LoginResult = "失败:" + constant.ExpireVerifyMailCode
		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			log.Errorf("DB Error: %v", err)
		}
		return FailWithDataOperate(c, -1, "邮箱验证码过期", "", err)
	}
	verifyMailCode := get.(int32)
	code, err := strconv.Atoi(loginAccountMail.VerifyMailCode)
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		return FailWithDataOperate(c, -1, "登录失败", "", err)
	}
	if verifyMailCode != int32(code) {
		global.Cache.Delete(loginAccountMail.VerifyMailId)

		// 保存登录日志
		loginLog.LoginResult = "失败:" + constant.FailVerifyMailCode
		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			log.Errorf("DB Error: %v", err)
		}
		return FailWithDataOperate(c, -1, "邮箱验证码不正确", "", err)
	}

	// 验证用户策略
	result, message := JudgeUserStrategy(user.ID, c.RealIP())
	if !result {
		// 保存登录日志
		loginLog.LoginResult = "失败"
		loginLog.Description = message
		err = loginLogRepository.Create(&loginLog)
		if nil != err {
			return err
		}
		return Fail(c, -1, message)
	}

	err = LoginSuccessMail(c, loginAccountMail, user, token)
	if nil != err {
		return err
	}
	////TODO 密码更换周期, 显示用户密码过期时间
	expire, err := repository.IdentityConfigDao.FindPasswordConfig()
	if err != nil {
		log.Errorf("获取密码过期时间失败: %v", err.Error())
	}
	if expire != nil && expire.PasswordCycle != 0 {
		day := int(time.Now().Sub(user.PasswordUpdated.Time).Hours()) / 24
		if expire.PasswordCycle-day < 0 {
			return Fail(c, -9, "您的密码已过期，账号存在风险，请立即重置密码")
		}
		if expire.PasswordRemind != 0 && expire.PasswordCycle-day <= expire.PasswordRemind {
			if err := messageRepository.Create(&model.Message{
				ID:        utils.UUID(),
				ReceiveId: user.ID,
				Theme:     "密码过期提醒",
				Level:     "high",
				Content:   fmt.Sprintf("您的密码将在%d天后过期，请您及时前往个人中心修改密码", expire.PasswordCycle-day),
				Status:    false,
				Type:      constant.NoticeMessage,
				Created:   utils.NowJsonTime(),
			}); err != nil {
				log.Errorf("创建消息失败: %v", err.Error())
			}
		}
	}
	return Success(c, token)
}

func LogoutEndpoint(c echo.Context) error {
	token := GetToken(c)
	cacheKey := BuildCacheKeyByToken(token)
	global.Cache.Delete(cacheKey)
	err := userService.Logout(token)
	if err != nil {
		return err
	}
	return Success(c, nil)
}

func ConfirmTOTPEndpoint(c echo.Context) error {
	var confirmTOTP ConfirmTOTP
	if err := c.Bind(&confirmTOTP); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 500, "用户信息已过期，请重新登录")
	}

	if !totp.Validate(confirmTOTP.TOTP, confirmTOTP.Secret) {
		return FailWithDataOperate(c, 400, "TOTP 验证失败, 请重试", "用户列表-新增/修改: 失败原因[TOTP验证失败]", nil)
	}

	u := model.UserNew{
		TOTPSecret: confirmTOTP.Secret,
		ID:         account.ID,
	}

	if err := userNewRepository.UpdateStructById(u, account.ID); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "配置失败", "", err)
	}

	return SuccessWithOperate(c, "用户列表-新增/修改: [TOTP验证成功]", nil)
}

func ReloadTOTPEndpoint(c echo.Context) error {
	userName := c.QueryParam("userName")
	key, err := totp.NewTOTP(totp.GenerateOpts{
		Issuer:      c.Request().Host,
		AccountName: userName,
	})
	if nil != err {
		log.Errorf("NewTOTP Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "获取TOTP二维码失败", "", nil)
	}

	qrcode, err := key.Image(200, 200)
	if err != nil {
		log.Errorf("Image Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "获取TOTP二维码失败", "", nil)
	}

	qrEncode, err := utils.ImageToBase64Encode(qrcode)
	if err != nil {
		log.Errorf("ImageToBase64Encode Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "获取TOTP二维码失败", "", nil)
	}

	return SuccessWithOperate(c, "", map[string]string{
		"qr":     qrEncode,
		"secret": key.Secret(),
	})
}
func ValidTimePasswordEndpoint(c echo.Context) error {
	loginLockConfig, err := identityConfigRepository.FindPasswordConfig()
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, -1, "获取用户信息失效,请重新登录")
	}
	var validTime string
	if loginLockConfig.PasswordCycle == 0 {
		validTime = "永久有效"
	} else {
		day := int(time.Now().Sub(account.PasswordUpdated.Time).Hours()) / 24
		if loginLockConfig.PasswordCycle-day >= 0 {
			validTime = fmt.Sprintf("%d天", loginLockConfig.PasswordCycle-day)
		} else {
			validTime = "已过期"
		}
	}
	return Success(c, validTime)
}

func ChangePasswordEndpoint(c echo.Context) error {
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, -1, "获取用户信息失效,请重新登录")
	}
	var changePassword model.ChangePassword
	if err := c.Bind(&changePassword); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	//数据校验
	if err := c.Validate(changePassword); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetErrLogMsg(err, errs.ChangePasswordLog)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}
	// 密码为空则禁止修改
	if changePassword.NewPassword == "" {
		return Fail(c, -1, "新密码为空")
	}
	// 验证两次密码是否一致
	if changePassword.NewPassword != changePassword.ConfirmPassword {
		return FailWithDataOperate(c, -1, "两次输入的密码不一致", "", nil)
	}
	if !strings.Contains(account.AuthenticationWay, constant.StaticPassword) {
		return FailWithDataOperate(c, -1, "禁止修改", "个人中心-密码修改: 非密码用户修改密码", nil)
	}
	if err := utils.Encoder.Match([]byte(account.Password), []byte(changePassword.OldPassword)); err != nil {
		return FailWithDataOperate(c, -1, "您输入的原密码不正确", "个人中心-密码修改: 原密码不正确", nil)
	}
	if ok := CheckPasswordRepeatTimes(account.SamePwdJudge, changePassword.NewPassword); !ok {
		return FailWithDataOperate(c, -1, "新密码不可与最近使用密码相同", "个人中心-密码修改: [修改密码:失败]失败原因: 新密码与最近使用密码相同", nil)
	}

	//密码复杂度校验
	result, code, msg, err := CheckPasswordComplexity(changePassword.NewPassword)
	if !result {
		return FailWithDataOperate(c, code, msg, "", err)
	}

	passwd, err := utils.Encoder.Encode([]byte(changePassword.NewPassword))
	if err != nil {
		log.Errorf("Encode Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	UpdateTime := utils.NewJsonTime(time.Now())
	u := model.UserNew{
		Password:        string(passwd),
		PasswordUpdated: UpdateTime,
		ID:              account.ID,
		SamePwdJudge:    utils.StrJoin(account.SamePwdJudge, string(passwd)),
	}

	if err := userNewRepository.UpdateStructById(u, account.ID); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "用户日志",
		Result:          "成功",
		LogContents:     "个人中心-密码修改: 密码修改",
		Created:         utils.NowJsonTime(),
		Users:           account.Username,
	}
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	return LogoutEndpoint(c)
}

func GetSelfInfoEndpoint(c echo.Context) error {
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, -1, "获取用户信息失效,请重新登录")
	}
	user, err := userNewRepository.FindById(account.ID)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", err)
	}
	var result = model.ChangePersonalInfo{
		Nickname: user.Nickname,
		QQ:       user.QQ,
		Phone:    user.Phone,
		Email:    user.Mail,
	}
	return Success(c, result)
}

func ChangePersonalInformation(c echo.Context) error {
	var personalInfo model.ChangePersonalInfo
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, -1, "获取用户信息失效,请重新登录")
	}
	if err := c.Bind(&personalInfo); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	var user = model.UserNew{
		Nickname:         personalInfo.Nickname,
		Mail:             personalInfo.Email,
		Phone:            personalInfo.Phone,
		QQ:               personalInfo.QQ,
		TOTPSecret:       account.TOTPSecret,
		Online:           account.Online,
		DepartmentId:     account.DepartmentId,
		IsRandomPassword: account.IsRandomPassword,
		SendWay:          account.SendWay,
	}
	if err := userNewRepository.UpdateMapById(user, account.ID); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	//if err := userNewRepository.UpdateStructById(model.UserNew{
	//	Nickname: personalInfo.Nickname,
	//	Mail:     personalInfo.Email,
	//	Phone:    personalInfo.Phone,
	//	QQ:       personalInfo.QQ,
	//}, account.ID); err != nil {
	//	log.Errorf("DB Error: %v", err)
	//	return FailWithDataOperate(c, 500, "修改失败", "", err)
	//}
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "用户日志",
		Result:          "成功",
		LogContents:     "个人中心-修改: [个人信息修改]",
		Created:         utils.NowJsonTime(),
		Users:           account.Username,
	}
	err := global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "创建日志失败", "", err)
	}
	return Success(c, "修改成功")
}

type AccountInfoNew struct {
	Id         string `json:"id"`
	Username   string `json:"username"`
	Nickname   string `json:"nickname"`
	Role       string `json:"role"`
	EnableTotp bool   `json:"enableTotp"`
}

func InfoEndpointNew(c echo.Context) error {
	account, _ := GetCurrentAccountNew(c)
	userNew, err := userNewRepository.FindById(account.ID)
	if err != nil {
		return err
	}

	info := AccountInfoNew{
		Id:         userNew.ID,
		Username:   userNew.Username,
		Nickname:   userNew.Nickname,
		Role:       userNew.RoleName,
		EnableTotp: userNew.TOTPSecret != "" && userNew.TOTPSecret != "-",
	}
	return Success(c, info)
}

// --------------------分割线-----------------

func JudgeUserStrategy(userId string, ip string) (result bool, message string) {
	user, err := userNewRepository.FindById(userId)
	if err != err {
		return false, "user not found"
	}
	var userStrategyId = make([]string, 0)
	_ = userStrategyRepository.DB.Table("user_strategy_users").Select("user_strategy_id").Where("user_id = ?", user.ID).Find(&userStrategyId)
	// 找到此用户对应的用户组id
	userGroupId, _ := userGroupMemberRepository.FindUserGroupIdsByUserId(userId)
	// 找到用户组对应的策略id
	var userGroupStrategyIds []string
	_ = userStrategyRepository.DB.Table("user_strategy_user_group").Select("user_strategy_id").Where("user_group_id in  ?", userGroupId).Find(&userGroupStrategyIds)
	// 合并用户策略id和用户组策略id并去重
	userStrategyId = append(userStrategyId, userGroupStrategyIds...)
	userStrategyId = utils.RemoveDuplicatesAndEmpty(userStrategyId)
	if len(userStrategyId) == 0 { // 没有策略限制可直接登录
		return true, "strategy not found"
	}
	// 找到生效策略
	var userStrategyValid model.UserStrategy
	if len(userStrategyId) > 0 {
		// 找到用户策略id对应的策略
		userStrategy := make([]model.UserStrategy, 0)
		for _, v := range userStrategyId {
			userPolicyTemp, err := userStrategyRepository.FindById(v)
			if err != nil {
				log.Errorf("查询用户策略失败:%v", err)
				continue
			}
			// 筛掉过期的不生效的策略,并且已启用
			if !*userPolicyTemp.IsPermanent && (userPolicyTemp.BeginValidTime.After(time.Now()) || userPolicyTemp.EndValidTime.Before(time.Now())) {
				continue
			}
			if userPolicyTemp.Status != constant.Enable {
				continue
			}
			userStrategy = append(userStrategy, userPolicyTemp)
		}
		// 先根据优先级排序再根据部门机构深度排序
		if len(userStrategy) == 0 {
			return true, "valid strategy not found"
		}
		if len(userStrategy) > 0 {
			sort.Slice(userStrategy, func(i, j int) bool {
				if userStrategy[i].Priority == userStrategy[j].Priority {
					return userStrategy[i].DepartmentDepth < userStrategy[j].DepartmentDepth
				}
				return userStrategy[i].Priority < userStrategy[j].Priority
			})
			userStrategyValid = userStrategy[0]
		}
	}
	// 检测最高优先级的策略的时间是否在登录期间
	if &userStrategyValid != nil {
		ok, err := service.ExpDateService.JudgeExpUserStrategy(&userStrategyValid)
		if err != nil {
			return false, "JudgeExpUserStrategy error"
		}
		if !ok {
			// 当前时间未在此策略的授权时间段内 TODO
			// 无策略默认允许登录不受限制
			return false, "当前登录时间未在授权时间段内"
		}
		// 检测IP限制
		var limit = true
		if userStrategyValid.IpLimitType == "blackList" {
			if isIpsContainIp(ip, userStrategyValid.IpLimitList) {
				// 当前登录IP属于黑名单列表
				limit = false
			}
		}
		if userStrategyValid.IpLimitType == "whiteList" {
			if !isIpsContainIp(ip, userStrategyValid.IpLimitList) {
				// 当前登录IP不属于白名单列表
				limit = false
			}
		}
		if !limit {
			// 当前时间未在此策略的授权时间段内
			return false, "当前登录IP被禁止登录"
		}
	}
	return true, "success"
}

// CheckPasswordComplexity 密码复杂度校验
func CheckPasswordComplexity(password string) (bool, int, string, error) {
	//密码复杂度校验
	pwd := password
	numReg := `[0-9]{1}`
	lowerLetterReg := `[a-z]{1}`
	upperLetterReg := `[A-Z]{1}`
	symbolReg := `[!@#~$%^&*()+|_]{1}`
	//获取当前数据库数据
	passConfig, err := repository.IdentityConfigDao.FindPasswordConfig()
	if err != nil {
		return false, 500, "获取密码配置失败", err
	}
	P := strconv.Itoa(passConfig.PasswordLength)
	if len(pwd) < passConfig.PasswordLength {
		return false, 422, " 用户密码长度不能小于 " + P + " 位数", err
	}
	if len(pwd) > 32 {
		return false, 422, " 用户密码长度不能大于 32 位数", err
	}
	if *passConfig.PasswordCheck {
		num, err := regexp.MatchString(numReg, pwd)
		if err != nil || !num {
			return false, 422, "密码需包含数字", err
		}
		lower, err := regexp.MatchString(lowerLetterReg, pwd)
		if err != nil || !lower {
			return false, 422, "密码需包含小写字母", err
		}
		upper, err := regexp.MatchString(upperLetterReg, pwd)
		if err != nil || !upper {
			return false, 422, "密码需包含大写字母", err
		}
		symbol, err := regexp.MatchString(symbolReg, pwd)
		if err != nil || !symbol {
			return false, 422, "密码需包含特殊字符", err
		}
	}
	return true, 200, "", err
}

// CheckPasswordRepeatTimes 密码重复次数校验
func CheckPasswordRepeatTimes(passwordRecord string, password string) bool {
	if passwordRecord == "" {
		return true
	}
	passConfig, err := repository.IdentityConfigDao.FindPasswordConfig()
	if err != nil {
		return true
	}
	passwordRecordList := strings.Split(passwordRecord, ";")
	for i, v := range passwordRecordList {
		if i+1 < passConfig.PasswordSameTimes {
			if err := utils.Encoder.Match([]byte(v), []byte(password)); err == nil {
				return false
			}
		} else {
			break
		}
	}
	return true
}

type PasswordFirstChange struct {
	Username        string `json:"username" binding:"required"`
	Password        string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirmPassword" binding:"required"`
}

// FirstLoginForceChangePasswordEndpoint 修改密码
func FirstLoginForceChangePasswordEndpoint(c echo.Context) error {
	var passwordFirstChange PasswordFirstChange
	if err := c.Bind(&passwordFirstChange); err != nil {
		log.Errorf("FirstLoginForceChangePassword bind error: %v", err)
		return FailWithDataOperate(c, 500, "参数错误", "", err)
	}
	if passwordFirstChange.Password != passwordFirstChange.ConfirmPassword {
		log.Errorf("FirstLoginForceChangePassword password not equal confirmPassword")
		return FailWithDataOperate(c, 500, "两次密码不一致", "", nil)
	}
	user, err := userNewRepository.FindByName(passwordFirstChange.Username)
	if err != nil {
		log.Errorf("FirstLoginForceChangePassword FindByName error: %v", err)
		return FailWithDataOperate(c, 500, "用户不存在", "", err)
	}

	ok, _, message, err := CheckPasswordComplexity(passwordFirstChange.Password)
	if !ok {
		return FailWithDataOperate(c, 500, message, "", err)
	}
	if err != nil {
		log.Errorf("FirstLoginForceChangePassword CheckPasswordComplexity error: %v", err)
		return FailWithDataOperate(c, 500, "密码复杂度校验失败", "", err)
	}
	pass, err := utils.Encoder.Encode([]byte(passwordFirstChange.Password)) // 加密存储密码
	userNew := model.UserNew{
		PasswordUpdated: utils.NowJsonTime(),
		Password:        string(pass),
		VerifyPassword:  string(pass),
		LoginNumbers:    1,
		SamePwdJudge:    utils.StrJoin(user.SamePwdJudge, string(pass)),
	}
	if err := userNewRepository.UpdateStructById(userNew, user.ID); err != nil {
		log.Errorf("FirstLoginForceChangePassword UpdateStructById error: %v", err)
		return FailWithDataOperate(c, 500, "修改密码失败", "", err)
	}
	return Success(c, nil)
}
