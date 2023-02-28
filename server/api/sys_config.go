package api

import (
	"github.com/labstack/echo/v4"
	"github.com/robfig/cron/v3"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"tkbastion/pkg/config"
	"tkbastion/pkg/constant"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/pkg/service"
	"tkbastion/pkg/validator"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"gorm.io/gorm"
)

// 系统时间
type SysTime struct {
	AutoSyncTime   string `json:"autoSyncTime"`
	SysCurrentTime string `json:"sysCurrentTime"`
	NtpServer      string `json:"ntpServer"`
}

func SysTimeGetEndpoint(c echo.Context) error {
	autoSyncTime, err := propertyRepository.FindByName("auto-sync-time")
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	ntpServer, err := propertyRepository.FindByName("ntp-server")
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}

	return SuccessWithOperate(c, "", SysTime{AutoSyncTime: autoSyncTime.Value, SysCurrentTime: utils.NowJsonTime().Format("2006-01-02 15:04:05"), NtpServer: ntpServer.Value})
}

func SysTimeSetTimePutEndpoint(c echo.Context) error {
	time := c.QueryParam("time")
	updateTimeCommand := "date -s \"" + time + "\""
	err := exec.Command("bash", "-c", updateTimeCommand).Run()
	if nil != err {
		log.Errorf("Run Error: %v", err)
		return FailWithDataOperate(c, 500, "设置时间失败", "", err)
	}

	return SuccessWithOperate(c, "系统时间-修改: 时间["+time+"]", nil)
}

func SysTimeSyncTimePutEndpoint(c echo.Context) error {
	ntpServer := c.QueryParam("ntp-server")
	err := propertyRepository.UpdateByName(&model.Property{Name: "ntp-server", Value: ntpServer}, "ntp-server")
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "同步时间失败", "", err)
	}

	syncTimeCommand := "ntpdate " + ntpServer
	err = exec.Command("bash", "-c", syncTimeCommand).Run()
	if nil != err {
		log.Errorf("Run Error: %v", err)
		return FailWithDataOperate(c, 500, "同步时间失败", "", err)
	}

	return SuccessWithOperate(c, "系统时间-同步时间: 同步NTP服务器["+ntpServer+"]时间", nil)
}

func SysTimeAutoSyncTimePutEndpoint(c echo.Context) error {
	autoSyncTime := c.QueryParam("auto-sync-time")
	err := propertyRepository.UpdateByName(&model.Property{Name: "auto-sync-time", Value: autoSyncTime}, "auto-sync-time")
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}
	ntpServer := c.QueryParam("ntp-server")
	err = propertyRepository.UpdateByName(&model.Property{Name: "ntp-server", Value: ntpServer}, "ntp-server")
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}

	if "true" == autoSyncTime {
		err = jobService.AutoSyncTime()
		if nil != err {
			log.Errorf("AutoSyncTime Error: %v", err)
			return FailWithDataOperate(c, 500, "操作失败", "", err)
		}

		return SuccessWithOperate(c, "系统时间-修改: 开启自动同步时间", nil)
	} else {
		global.Cron.Remove(cron.EntryID(service.JobsRecords["自动同步时间"]))
		log.Infof("删除计划任务[自动同步时间], 运行中计划任务数量「%v」", len(global.Cron.Entries()))
	}

	return SuccessWithOperate(c, "系统时间-修改: 关闭自动同步时间", nil)
}

// 认证配置-Radius
type RadiusConfig struct {
	RadiusState           string `json:"radiusState"`
	RadiusServerAddress   string `json:"radiusServerAddress" validate:"required,ip|fqdn"`
	RadiusPort            int    `json:"radiusPort" validate:"required,min=1,max=65535"`
	RadiusAuthProtocol    string `json:"radiusAuthProtocol"`
	RadiusAuthShareSecret string `json:"radiusAuthShareSecret"`
	RadiusAuthTimeOut     int    `json:"radiusAuthTimeOut" validate:"required,min=5,max=30"`
}

func AuthConfigRadiusGetEndpoint(c echo.Context) error {
	item := propertyRepository.FindAuMap("radius")
	var radiusConfig RadiusConfig
	radiusConfig.RadiusState = item["radius-state"]
	radiusConfig.RadiusServerAddress = item["radius-server-address"]
	iRadiusPort, err := strconv.Atoi(item["radius-port"])
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
	}
	radiusConfig.RadiusPort = iRadiusPort
	radiusConfig.RadiusAuthProtocol = item["radius-auth-protocol"]
	radiusConfig.RadiusAuthShareSecret = item["radius-auth-share-secret"]
	iRadiusAuthTimeOut, err := strconv.Atoi(item["radius-auth-time-out"])
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
	}
	radiusConfig.RadiusAuthTimeOut = iRadiusAuthTimeOut

	return SuccessWithOperate(c, "", radiusConfig)
}

func AuthConfigRadiusUpdateEndpoint(c echo.Context) error {
	var radiusConfig RadiusConfig
	if err := c.Bind(&radiusConfig); err != nil {
		log.Errorf("Bind Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if err := c.Validate(&radiusConfig); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	if err := propertyRepository.UpdateByName(&model.Property{Name: "radius-state", Value: radiusConfig.RadiusState}, "radius-state"); nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	var state string
	if "true" == radiusConfig.RadiusState {
		state = "开启"

		if err := propertyRepository.UpdateByName(&model.Property{Name: "radius-server-address", Value: radiusConfig.RadiusServerAddress}, "radius-server-address"); nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
		if err := propertyRepository.UpdateByName(&model.Property{Name: "radius-port", Value: strconv.Itoa(radiusConfig.RadiusPort)}, "radius-port"); nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
		if err := propertyRepository.UpdateByName(&model.Property{Name: "radius-auth-protocol", Value: radiusConfig.RadiusAuthProtocol}, "radius-auth-protocol"); nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
		if err := propertyRepository.UpdateByName(&model.Property{Name: "radius-auth-share-secret", Value: radiusConfig.RadiusAuthShareSecret}, "radius-auth-share-secret"); nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
		if err := propertyRepository.UpdateByName(&model.Property{Name: "radius-auth-time-out", Value: strconv.Itoa(radiusConfig.RadiusAuthTimeOut)}, "radius-auth-time-out"); nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
	} else {
		state = "关闭"
		// 删除radius认证关联的用户
		err := DeleteRadiusUsers()
		if nil != err {
			log.Errorf("DeleteRadiusUsers Error: %v", err)
			log.Error("关闭radius认证成功, 删除radius关联用户失败")
			return FailWithDataOperate(c, 500, "删除关联用户失败", "", nil)
		}
	}

	return SuccessWithOperate(c, "RADIUS认证配置-修改: "+state+"RADIUS认证", nil)
}

// 认证配置-Ldap/Ad
func AuthConfigLdapAdPagingEndpoint(c echo.Context) error {
	ldapAdAuthArr, err := ldapAdAuthRepository.Find()
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	return SuccessWithOperate(c, "", ldapAdAuthArr)
}

// 这里规定对一个ID对应的认证服务器信息进行修改时, 不删除与之关联的用户(该用户创建时选择LDAP/AD认证选择了此ID对应的服务器 或 该用户是通过此ID对应的服务器同步用户创建的)
// 即用户只和 服务器ID相对应, 就算修改该ID对应的服务器IP, DN等信息, 也不会删除用户, 这些用户下次认证时使用修改后的相关信息认证即可
func AuthConfigLdapAdUpdateEndpoint(c echo.Context) error {
	id := c.Param("id")
	iId, err := strconv.Atoi(id)
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", nil)
	}
	var item model.LdapAdAuth
	if err := c.Bind(&item); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if err := c.Validate(&item); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	ldapAdAuth, err := ldapAdAuthRepository.FindById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", nil)
	}

	err = ldapAdAuthRepository.UpdateById(int64(iId), &item)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", nil)
	}

	global.Cron.Remove(cron.EntryID(service.JobsRecords["LDAP/AD用户同步"+id]))
	log.Debugf("删除计划任务 [LDAP/AD用户同步"+id+"], 运行中计划任务数量[%v]", len(global.Cron.Entries()))

	if "auto" == item.LdapAdSyncType {
		err = jobService.CreateLdapAdSyncUserJob(ldapAdAuth.ID)
		if nil != err {
			log.Errorf("CreateLdapAdSyncUserJob Error: %v", err)
			return FailWithDataOperate(c, 500, "自动同步用户定时任务启动失败", "", err)
		}
	}

	return SuccessWithOperate(c, "LDAP/AD认证配置-修改: 域服务器地址["+ldapAdAuth.LdapAdServerAddress+"->"+item.LdapAdServerAddress+"]", nil)
}

func AuthConfigLdapAdDeleteEndpoint(c echo.Context) error {
	id := c.Param("id")
	iId, err := strconv.Atoi(id)
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "", nil)
	}

	ldapAdAuth, err := ldapAdAuthRepository.FindById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "", nil)
	}
	err = ldapAdAuthRepository.DeleteById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "", nil)
	}

	global.Cron.Remove(cron.EntryID(service.JobsRecords["LDAP/AD用户同步"+id]))
	log.Debugf("删除计划任务 [LDAP/AD用户同步"+id+"], 运行中计划任务数量[%v]", len(global.Cron.Entries()))

	// 删除此认证服务器关联的用户
	err = DeleteUserByAuthServerId(int64(iId))
	if nil != err {
		log.Errorf("DeleteUserByAuthServerId Error: %v", err)
		log.Error("删除LDAP/AD认证服务器成功, 删除关联用户失败")
		return FailWithDataOperate(c, 500, "删除关联用户失败", "", nil)
	}

	return SuccessWithOperate(c, "LDAP/AD认证配置-删除: 域服务器地址["+ldapAdAuth.LdapAdServerAddress+"]", nil)
}

func AuthConfigLdapAdSyncAccountEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	iId, err := strconv.Atoi(id)
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		return FailWithDataOperate(c, 500, "同步账号失败", "", nil)
	}

	ldapAdAuth, err := ldapAdAuthRepository.FindById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "同步账号失败", "", nil)
	}

	err = authenticationService.LdapAdSyncUser(ldapAdAuth.ID)
	if nil != err {
		log.Errorf("LdapAdSyncUser Error: %v", err)
		return FailWithDataOperate(c, 500, "同步账号失败", "", nil)
	}

	return SuccessWithOperate(c, "LDAP/AD认证配置-同步账号: 域服务器地址["+ldapAdAuth.LdapAdServerAddress+"]", nil)
}

func AuthConfigLdapAdGetEndpoint(c echo.Context) error {
	id := c.Param("id")
	iId, err := strconv.Atoi(id)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	item, err := ldapAdAuthRepository.FindById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return err
	}

	return SuccessWithOperate(c, "", item)
}

func AuthConfigLdapAdCreateEndpoint(c echo.Context) error {
	var item model.LdapAdAuth
	if err := c.Bind(&item); nil != err {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}

	err := ldapAdAuthRepository.Create(&item)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", nil)
	}

	if "auto" == item.LdapAdSyncType {
		var ldapAdAuth model.LdapAdAuth
		err = ldapAdAuthRepository.DB.Order("id desc").First(&ldapAdAuth).Error
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "自动同步用户定时任务启动失败", "", nil)
		}

		err = jobService.CreateLdapAdSyncUserJob(ldapAdAuth.ID)
		if nil != err {
			log.Errorf("CreateLdapAdSyncUserJob Error: %v", err)
			return FailWithDataOperate(c, 500, "自动同步用户定时任务启动失败", "", err)
		}
	}

	return SuccessWithOperate(c, "LDAP/AD认证配置-新增: 域服务器地址["+item.LdapAdServerAddress+"]", nil)
}

// 认证配置-指纹认证

func AuthConfigFingerprintGetEndpoint(c echo.Context) error {
	item := propertyRepository.FindAuMap("finger-print")
	return SuccessWithOperate(c, "", item)
}

func AuthConfigFingerprintUpdateEndpoint(c echo.Context) error {
	var item map[string]interface{}
	if err := c.Bind(&item); nil != err {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "更新失败", "", err)
	}

	err := propertyRepository.UpdateByName(&model.Property{
		Name:  "finger-print",
		Value: item["finger-print"].(string),
	}, "finger-print")
	if nil != err {
		log.Error("DB Error: ", err)
		return FailWithDataOperate(c, 500, "更新失败", "", err)
	}

	return SuccessWithOperate(c, "指纹认证配置-更新: 更新指纹认证配置", nil)
}

// 外发配置-邮件

type MailConfig struct {
	MailState              string `json:"mailState"`
	MailSendMailServer     string `json:"mailSendMailServer" validate:"required,email,contains=smtp"`
	MailSecretType         string `json:"mailSecretType"`
	MailPort               int    `json:"mailPort" validate:"required,min=1,max=65535"`
	MailAccount            string `json:"mailAccount" validate:"required,email"`
	MailPassword           string `json:"mailPassword"`
	MailTestMailboxAddress string `json:"mailTestMailboxAddress" validate:"required,email"`
}

func OutSendMailGetEndpoint(c echo.Context) error {
	item := propertyRepository.FindAuMap("mail")
	var mailConfig MailConfig
	mailConfig.MailState = item["mail-state"]
	mailConfig.MailSendMailServer = item["mail-send-mail-server"]
	mailConfig.MailSecretType = item["mail-secret-type"]
	iMailPort, err := strconv.Atoi(item["mail-port"])
	if nil != err {
		if "ssl" == mailConfig.MailSecretType {
			iMailPort = 465
		} else {
			iMailPort = 25
		}

		log.Errorf("Atoi Error: %v", err)
	}
	mailConfig.MailPort = iMailPort
	mailConfig.MailAccount = item["mail-account"]
	mailConfig.MailPassword = item["mail-password"]
	mailConfig.MailTestMailboxAddress = item["mail-test-mailbox-address"]

	return SuccessWithOperate(c, "", mailConfig)
}

func OutSendMailUpdateEndpoint(c echo.Context) error {
	var mailConfig MailConfig
	if err := c.Bind(&mailConfig); err != nil {
		log.Errorf("Bind Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if err := c.Validate(&mailConfig); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	if err := propertyRepository.UpdateByName(&model.Property{Name: "mail-state", Value: mailConfig.MailState}, "mail-state"); nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	var state string
	if "true" == mailConfig.MailState {
		state = "开启"

		if err := propertyRepository.UpdateByName(&model.Property{Name: "mail-send-mail-server", Value: mailConfig.MailSendMailServer}, "mail-send-mail-server"); nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
		if err := propertyRepository.UpdateByName(&model.Property{Name: "mail-secret-type", Value: mailConfig.MailSecretType}, "mail-secret-type"); nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
		if err := propertyRepository.UpdateByName(&model.Property{Name: "mail-port", Value: strconv.Itoa(mailConfig.MailPort)}, "mail-port"); nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
		if err := propertyRepository.UpdateByName(&model.Property{Name: "mail-account", Value: mailConfig.MailAccount}, "mail-account"); nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
		if err := propertyRepository.UpdateByName(&model.Property{Name: "mail-password", Value: mailConfig.MailPassword}, "mail-password"); nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
		if err := propertyRepository.UpdateByName(&model.Property{Name: "mail-test-mailbox-address", Value: mailConfig.MailTestMailboxAddress}, "mail-test-mailbox-address"); nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
	} else {
		state = "关闭"
		// 删除邮件认证关联的用户
		err := DeleteMailUsers()
		if nil != err {
			log.Errorf("DeleteMailUsers Error: %v", err.Error())
			log.Error("关闭邮件配置, 删除邮件认证关联用户失败")
			return FailWithDataOperate(c, 500, "删除邮件认证关联用户失败", "", nil)
		}
	}

	return SuccessWithOperate(c, "邮件配置-修改: "+state+"邮件配置", nil)
}

func OutSendMailSendTestMailEndpoint(c echo.Context) error {
	var mailConfig MailConfig
	if err := c.Bind(&mailConfig); err != nil {
		log.Errorf("Bind Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if err := c.Validate(&mailConfig); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	host := mailConfig.MailSendMailServer
	ssl := false
	if "ssl" == mailConfig.MailSecretType {
		ssl = true
	}
	port := mailConfig.MailPort
	username := mailConfig.MailAccount
	password := mailConfig.MailPassword
	receiver := mailConfig.MailTestMailboxAddress

	err := sysConfigService.SendTestMail(host, username, password, port, ssl, []string{receiver}, "邮件配置", "邮件配置测试邮件")
	if nil != err {
		log.Errorf("NewNewSendMail Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "发送测试邮件失败", "", err)
	}
	return SuccessWithOperate(c, "邮件配置-发送测试邮件: 发件服务器["+host+":"+strconv.Itoa(port)+"], 测试邮箱["+mailConfig.MailTestMailboxAddress+"]", nil)
}

// 外发配置-短信配置

var SMS = []string{"sms_type", "sms_api_id", "sms_api_secret", "sms_sign_name", "sms_test_phone_number", "sms_template_code"}

func OutSendSmsGetEndpoint(c echo.Context) error {
	var smsConfig dto.SmsConfig
	smsConfig, err := propertyRepository.GetSmsProperty()
	if nil != err {
		log.Error("获取短信配置失败: ", err.Error())
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}

	return Success(c, smsConfig)
}

func OutSendSmsUpdateEndpoint(c echo.Context) error {
	var smsConfig dto.SmsConfig
	if err := c.Bind(&smsConfig); err != nil {
		log.Errorf("Bind Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if err := c.Validate(&smsConfig); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	if err := propertyRepository.UpdateByName(&model.Property{Name: "mail_state", Value: smsConfig.SmsState}, "sms_state"); nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	var state string
	if "true" == smsConfig.SmsState {
		state = "开启"
		if err := propertyRepository.DeleteByNames(SMS); nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}

		// 加密
		if err := smsConfig.Encrypt(); err != nil {
			log.Errorf("Encrypt Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}

		smsMap := utils.Struct2MapByStructTag(smsConfig)
		delete(smsMap, "sms_state")

		if err := propertyRepository.CreateByMap(smsMap); nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
	} else {
		state = "关闭"
		// 删除短信认证关联的用户
		if err := propertyRepository.DeleteByNames(SMS); nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
	}

	return SuccessWithOperate(c, "短信配置-修改: "+state+"短信配置", nil)
}

func OutSendSmsSendTestSmsEndpoint(c echo.Context) error {
	var smsConfig dto.SmsConfig
	if err := c.Bind(&smsConfig); err != nil {
		log.Errorf("Bind Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if err := c.Validate(&smsConfig); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	if smsConfig.SmsApiId == "" || smsConfig.SmsApiSecret == "" || smsConfig.SmsSignName == "" || smsConfig.SmsTestPhoneNumber == "" || smsConfig.SmsTemplateCode == "" {
		return FailWithDataOperate(c, 500, "发送测试短信失败, 短信配置不完整", "", nil)
	}

	err := utils.SendSms(smsConfig.SmsTestPhoneNumber, "054345", smsConfig.SmsApiId, smsConfig.SmsApiSecret, smsConfig.SmsSignName, smsConfig.SmsTemplateCode) //"短信配置测试信息")
	if nil != err {
		log.Errorf("NewNewSendMail Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "发送测试短信失败", "", err)
	}
	return SuccessWithOperate(c, "短信配置-发送测试短信: 短信模板["+smsConfig.SmsTemplateCode+"], 测试手机号["+smsConfig.SmsTestPhoneNumber+"]", nil)
}

// OutSendSnmpGetEndpoint SNMP配置
func OutSendSnmpGetEndpoint(c echo.Context) error {
	// 获取SNMP配置
	item := propertyRepository.FindAuMap("snmp")
	var snmpConfig model.SnmpConfigGet
	snmpConfig.SnmpState = item["snmp-state"]
	port, _ := strconv.Atoi(item["snmp-port"])
	snmpConfig.Port = port
	snmpConfig.PhysicalLocationInfo = item["snmp-physical-location-info"]
	snmpConfig.ContactInfo = item["snmp-contact-info"]
	snmpConfig.SysInfo = item["snmp-sys-info"]
	snmpConfig.Version = item["snmp-version"]
	snmpConfig.V2CName = item["snmp-v2c-name"]
	snmpConfig.V2CRWAuth = item["snmp-v2c-rw-auth"]
	snmpConfig.V3Name = item["snmp-v3-name"]
	snmpConfig.V3RWAuth = item["snmp-v3-rw-auth"]
	snmpConfig.V3CertificationType = item["snmp-v3-certification-type"]
	snmpConfig.V3CertificationMode = item["snmp-v3-certification-mode"]
	snmpConfig.V3EncryptionMode = item["snmp-v3-encryption-mode"]
	snmpConfig.AuthIp = item["snmp-auth-ip"]
	return Success(c, snmpConfig)
}

func OutSendSnmpUpdateEndpoint(c echo.Context) error {
	var snmpConfig model.SnmpConfig
	if err := c.Bind(&snmpConfig); err != nil {
		log.Errorf("Bind Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	snmpConfigMap := make(map[string]string)
	snmpConfigMap["snmp-state"] = snmpConfig.SnmpState
	if snmpConfig.Port == 0 {
		temp, _ := propertyRepository.FindByName("snmp-port")
		snmpConfigMap["snmp-port"] = temp.Value
	} else {
		snmpConfigMap["snmp-port"] = strconv.Itoa(snmpConfig.Port)
	}
	snmpConfigMap["snmp-physical-location-info"] = snmpConfig.PhysicalLocationInfo
	snmpConfigMap["snmp-contact-info"] = snmpConfig.ContactInfo
	snmpConfigMap["snmp-sys-info"] = snmpConfig.SysInfo
	snmpConfigMap["snmp-version"] = snmpConfig.Version
	snmpConfigMap["snmp-v2c-name"] = snmpConfig.V2CName
	snmpConfigMap["snmp-v2c-rw-auth"] = snmpConfig.V2CRWAuth
	snmpConfigMap["snmp-v3-name"] = snmpConfig.V3Name
	snmpConfigMap["snmp-v3-rw-auth"] = snmpConfig.V3RWAuth
	snmpConfig.V3CertificationPass, _ = utils.AesEncryptECB(snmpConfig.V3CertificationPass, "snmp_v3_password")
	snmpConfigMap["snmp-v3-certification-pass"] = snmpConfig.V3CertificationPass
	//snmpConfigMap["snmp-v3-certification-pass"] = snmpConfig.V3CertificationPass
	snmpConfigMap["snmp-v3-certification-type"] = snmpConfig.V3CertificationType
	snmpConfig.V3EncryptionPass, _ = utils.AesEncryptECB(snmpConfig.V3EncryptionPass, "snmp_v3_password")
	snmpConfigMap["snmp-v3-encryption-pass"] = snmpConfig.V3EncryptionPass
	//snmpConfigMap["snmp-v3-encryption-pass"] = snmpConfig.V3EncryptionPass
	snmpConfigMap["snmp-v3-certification-mode"] = snmpConfig.V3CertificationMode
	snmpConfigMap["snmp-v3-encryption-mode"] = snmpConfig.V3EncryptionMode
	snmpConfigMap["snmp-auth-ip"] = snmpConfig.AuthIp
	for k, v := range snmpConfigMap {
		if err := propertyRepository.UpdateByName(&model.Property{Name: k, Value: v}, k); nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
	}
	// 关闭SNMP服务
	if snmpConfigMap["snmp-state"] == "false" {
		err := snmpService.StopSnmp()
		if err != nil {
			return err
		}
	}
	// 启动SNMP服务
	if snmpConfigMap["snmp-state"] == "true" {
		err := snmpService.StopSnmp()
		if err != nil {
			return err
		}
		err = snmpService.StartSnmp(snmpConfig)
		if err != nil {
			return err
		}
	}
	return SuccessWithOperate(c, "SNMP配置-修改: 系统配置,外发配置修改", snmpConfig)
}

// 外发配置-SYSLOG

type SyslogConfig struct {
	SyslogState  string `json:"syslogState"`
	SyslogServer string `json:"syslogServer"`
	SyslogPort   int    `json:"syslogPort"`
}

func OutSendSyslogGetEndpoint(c echo.Context) error {
	item := propertyRepository.FindAuMap("syslog")
	var syslogConfig SyslogConfig
	syslogConfig.SyslogState = item["syslog-state"]
	syslogConfig.SyslogServer = item["syslog-server"]
	iSyslogPort, err := strconv.Atoi(item["syslog-port"])
	if nil != err {
		iSyslogPort = 514
		log.Errorf("Atoi Error: %v", err)
	}
	syslogConfig.SyslogPort = iSyslogPort

	return SuccessWithOperate(c, "", syslogConfig)
}

func OutSendSyslogUpdateEndpoint(c echo.Context) error {
	var syslogConfig SyslogConfig
	if err := c.Bind(&syslogConfig); err != nil {
		log.Errorf("Bind Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if err := propertyRepository.UpdateByName(&model.Property{Name: "syslog-state", Value: syslogConfig.SyslogState}, "syslog-state"); nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	var state string
	if "true" == syslogConfig.SyslogState {
		state = "开启"

		if err := propertyRepository.UpdateByName(&model.Property{Name: "syslog-server", Value: syslogConfig.SyslogServer}, "syslog-server"); nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
		if err := propertyRepository.UpdateByName(&model.Property{Name: "syslog-port", Value: strconv.Itoa(syslogConfig.SyslogPort)}, "syslog-port"); nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
	} else {
		state = "关闭"
	}

	return SuccessWithOperate(c, "SYSLOG配置-修改: "+state+"SYSLOG配置", nil)
}

// 安全配置-登录会话
type LoginSessionConfig struct {
	Overtime    int `json:"overtime"`
	OnlineCount int `json:"onlineCount"`
}

func SecurityLoginSessionGetEndpoint(c echo.Context) error {
	item := propertyRepository.FindAuMap("login-session")
	var loginSessionConfig LoginSessionConfig
	iOvertime, err := strconv.Atoi(item["login-session-overtime"])
	if nil != err {
		iOvertime = 40
		log.Errorf("Atoi Error: %v", err)
	}
	iOnlineCount, err := strconv.Atoi(item["login-session-online-count"])
	if nil != err {
		iOvertime = 10
		log.Errorf("Atoi Error: %v", err)
	}

	loginSessionConfig.Overtime = iOvertime
	loginSessionConfig.OnlineCount = iOnlineCount

	return SuccessWithOperate(c, "", loginSessionConfig)
}

func SecurityLoginSessionUpdateEndpoint(c echo.Context) error {
	var loginSessionConfig LoginSessionConfig
	if err := c.Bind(&loginSessionConfig); err != nil {
		log.Errorf("Bind Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if err := propertyRepository.UpdateByName(&model.Property{Name: "login-session-overtime", Value: strconv.Itoa(loginSessionConfig.Overtime)}, "login-session-overtime"); nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if err := propertyRepository.UpdateByName(&model.Property{Name: "login-session-online-count", Value: strconv.Itoa(loginSessionConfig.OnlineCount)}, "login-session-online-count"); nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	return SuccessWithOperate(c, "登录会话配置-修改: 登录会话超时["+strconv.Itoa(loginSessionConfig.Overtime)+"], 同一用户在线会话数["+strconv.Itoa(loginSessionConfig.OnlineCount)+"]", nil)
}

type DisplayConfigForPaging struct {
	LoginBackground string `json:"loginBackground"`
	LogoIconImage   string `json:"logoIconImage"`
	SysTitle        string `json:"sysTitle"`
}

// UiConfigGetEndpoint 界面配置-获取
func UiConfigGetEndpoint(c echo.Context) error {
	dir, _ := os.Getwd()
	var displayConfig DisplayConfigForPaging
	propertyMap, err := propertyRepository.FindMapByNames([]string{constant.SystemTitle, constant.LogoIconImage, constant.LoginBackground})
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}
	displayConfig.SysTitle = propertyMap[constant.SystemTitle]
	displayConfig.LoginBackground = "/" + propertyMap[constant.LoginBackground]
	if config.GlobalCfg.Debug == false {
		if ok := utils.FileExists(dir + "/tkbastion/web/dist/" + propertyMap[constant.LogoIconImage]); ok {
			displayConfig.LoginBackground = "/" + propertyMap[constant.LogoIconImage]
		} else {
			displayConfig.LoginBackground = "/" + constant.LogoIconImage
		}
	} else {
		if ok := utils.FileExists(dir + "/web/dist/" + propertyMap[constant.LogoIconImage]); ok {
			displayConfig.LogoIconImage = "/" + propertyMap[constant.LogoIconImage]
		} else {
			displayConfig.LogoIconImage = "/" + constant.LogoIconImage
		}
	}
	return Success(c, displayConfig)
}

// UiConfigUpdateEndpoint 界面配置-修改
func UiConfigUpdateEndpoint(c echo.Context) error {
	propertyMap, err := propertyRepository.FindMapByNames([]string{constant.SystemTitle, constant.LogoIconImage, constant.LoginBackground})
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	// 获取当前文件目录
	dir, err := os.Getwd()
	if err != nil {
		log.Errorf("Get Pwd Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	loginBackground, err := c.FormFile("loginBackground")
	if err != nil {
		log.Errorf("FormFile Error: %v", err)
	} else {
		if err := utils.UploadSaveFiles(loginBackground, dir+"/web/dist", constant.LoginBackground); err != nil {
			log.Errorf("UploadSaveFiles Error: %v", err)
			return FailWithDataOperate(c, 500, "上传失败", "", err)
		}
		// 将资源文件加载到路由中
		c.Echo().File("/"+constant.LoginBackground, dir+"/web/dist/"+constant.LoginBackground)
	}
	logoIconImage, err := c.FormFile("logoIconImage")
	if err != nil {
		log.Errorf("FormFile Error: %v", err)
	} else {
		if propertyMap[constant.LogoIconImage] != constant.LogoIconImage {
			// 删除原有文件
			if err := os.Remove(dir + "/web/dist/" + propertyMap[constant.LogoIconImage]); nil != err {
				log.Errorf("Delete File Error: %v", err)
			}
		}
		if err := utils.UploadSaveFiles(logoIconImage, dir+"/web/dist", logoIconImage.Filename); err != nil {
			log.Errorf("UploadSaveFiles Error: %v", err)
			return FailWithDataOperate(c, 500, "上传失败", "", err)
		}
		if err := propertyRepository.UpdateByName(&model.Property{Name: constant.LogoIconImage, Value: logoIconImage.Filename}, constant.LogoIconImage); nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "上传失败", "", err)
		}
		// 将资源文件加载到路由中
		c.Echo().File("/"+logoIconImage.Filename, dir+"/web/dist/"+logoIconImage.Filename)
	}
	sysTitle := c.QueryParam("sysTitle")
	if sysTitle != "" {
		err := propertyRepository.UpdateByName(&model.Property{Name: constant.SystemTitle, Value: sysTitle}, constant.SystemTitle)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
	}
	return SuccessWithOperate(c, "界面配置-修改: 修改界面配置", nil)
}

// SystemPerformanceFormGetEndpoint 告警配置-获取系统性能表单形式
func SystemPerformanceFormGetEndpoint(c echo.Context) error {
	alarmSysPerformanceConfigMap, err := propertyRepository.FindMapByNames(constant.AlarmConfigPerformance)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		log.Errorf("系统配置-告警配置: 获取系统性能告警配置失败")
	}
	getNum := func(key string) int {
		num, err := strconv.Atoi(key)
		if nil != err {
			log.Errorf("Atoi Error: %v", err)
			return 0
		}
		return num
	}
	alarmConfig := make([]model.AlarmConfig, 4)
	alarmConfig[0] = model.AlarmConfig{
		Event:          "CPU使用率",
		ThresholdValue: getNum(alarmSysPerformanceConfigMap["cpu-max"]),
		IsMail:         utils.StrToBoolPtr(alarmSysPerformanceConfigMap["cpu-mail"]),
		IsMessage:      utils.StrToBoolPtr(alarmSysPerformanceConfigMap["cpu-msg"]),
		IsSyslog:       utils.StrToBoolPtr(alarmSysPerformanceConfigMap["cpu-syslog"]),
		AlarmLevel:     alarmSysPerformanceConfigMap["cpu-level"],
	}
	alarmConfig[1] = model.AlarmConfig{
		Event:          "内存使用率",
		ThresholdValue: getNum(alarmSysPerformanceConfigMap["mem-max"]),
		IsMail:         utils.StrToBoolPtr(alarmSysPerformanceConfigMap["mem-mail"]),
		IsMessage:      utils.StrToBoolPtr(alarmSysPerformanceConfigMap["mem-msg"]),
		IsSyslog:       utils.StrToBoolPtr(alarmSysPerformanceConfigMap["mem-syslog"]),
		AlarmLevel:     alarmSysPerformanceConfigMap["mem-level"],
	}
	alarmConfig[2] = model.AlarmConfig{
		Event:          "磁盘使用率",
		ThresholdValue: getNum(alarmSysPerformanceConfigMap["disk-max"]),
		IsMail:         utils.StrToBoolPtr(alarmSysPerformanceConfigMap["disk-mail"]),
		IsMessage:      utils.StrToBoolPtr(alarmSysPerformanceConfigMap["disk-msg"]),
		IsSyslog:       utils.StrToBoolPtr(alarmSysPerformanceConfigMap["disk-syslog"]),
		AlarmLevel:     alarmSysPerformanceConfigMap["disk-level"],
	}
	alarmConfig[3] = model.AlarmConfig{
		Event:          "数据盘使用率",
		ThresholdValue: getNum(alarmSysPerformanceConfigMap["data-max"]),
		IsMail:         utils.StrToBoolPtr(alarmSysPerformanceConfigMap["data-mail"]),
		IsMessage:      utils.StrToBoolPtr(alarmSysPerformanceConfigMap["data-msg"]),
		IsSyslog:       utils.StrToBoolPtr(alarmSysPerformanceConfigMap["data-syslog"]),
		AlarmLevel:     alarmSysPerformanceConfigMap["data-level"],
	}
	return Success(c, alarmConfig)
}

// AlarmSysPerformanceGetEndpoint 获取系统性能告警
func AlarmSysPerformanceGetEndpoint(c echo.Context) error {
	alarmSysPerformanceConfigMap, err := propertyRepository.FindMapByNames(constant.AlarmConfigPerformance)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		log.Errorf("系统配置-告警配置: 获取系统性能告警配置失败")
	}
	var alarmSysPerformanceConfig = model.AlarmPerformanceConfig{
		CpuMax:    alarmSysPerformanceConfigMap["cpu-max"],
		CpuMsg:    utils.StrToBoolPtr(alarmSysPerformanceConfigMap["cpu-msg"]),
		CpuMail:   utils.StrToBoolPtr(alarmSysPerformanceConfigMap["cpu-mail"]),
		CpuSyslog: utils.StrToBoolPtr(alarmSysPerformanceConfigMap["cpu-syslog"]),
		CpuLevel:  alarmSysPerformanceConfigMap["cpu-level"],

		MemMax:    alarmSysPerformanceConfigMap["mem-max"],
		MemMsg:    utils.StrToBoolPtr(alarmSysPerformanceConfigMap["mem-msg"]),
		MemMail:   utils.StrToBoolPtr(alarmSysPerformanceConfigMap["mem-mail"]),
		MemSyslog: utils.StrToBoolPtr(alarmSysPerformanceConfigMap["mem-syslog"]),
		MemLevel:  alarmSysPerformanceConfigMap["mem-level"],

		DiskMax:    alarmSysPerformanceConfigMap["disk-max"],
		DiskMsg:    utils.StrToBoolPtr(alarmSysPerformanceConfigMap["disk-msg"]),
		DiskMail:   utils.StrToBoolPtr(alarmSysPerformanceConfigMap["disk-mail"]),
		DiskSyslog: utils.StrToBoolPtr(alarmSysPerformanceConfigMap["disk-syslog"]),
		DiskLevel:  alarmSysPerformanceConfigMap["disk-level"],

		VisualMemMax:    alarmSysPerformanceConfigMap["data-max"],
		VisualMemMsg:    utils.StrToBoolPtr(alarmSysPerformanceConfigMap["data-msg"]),
		VisualMemMail:   utils.StrToBoolPtr(alarmSysPerformanceConfigMap["data-mail"]),
		VisualMemSyslog: utils.StrToBoolPtr(alarmSysPerformanceConfigMap["data-syslog"]),
		VisualMemLevel:  alarmSysPerformanceConfigMap["data-level"],
	}
	return Success(c, alarmSysPerformanceConfig)
}

// AlarmSysPerformanceUpdateEndpoint 编辑系统性能告警
func AlarmSysPerformanceUpdateEndpoint(c echo.Context) error {
	var alarmPerformanceConfig model.AlarmPerformanceConfig
	if err := c.Bind(&alarmPerformanceConfig); err != nil {
		log.Errorf("Bind Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	properties := make([]model.Property, 20)
	properties[0] = model.Property{Name: "cpu-max", Value: alarmPerformanceConfig.CpuMax}
	properties[1] = model.Property{Name: "cpu-msg", Value: strconv.FormatBool(*alarmPerformanceConfig.CpuMsg)}
	properties[2] = model.Property{Name: "cpu-mail", Value: strconv.FormatBool(*alarmPerformanceConfig.CpuMail)}
	properties[3] = model.Property{Name: "cpu-syslog", Value: strconv.FormatBool(*alarmPerformanceConfig.CpuSyslog)}
	properties[4] = model.Property{Name: "cpu-level", Value: alarmPerformanceConfig.CpuLevel}

	properties[5] = model.Property{Name: "mem-max", Value: alarmPerformanceConfig.MemMax}
	properties[6] = model.Property{Name: "mem-msg", Value: strconv.FormatBool(*alarmPerformanceConfig.MemMsg)}
	properties[7] = model.Property{Name: "mem-mail", Value: strconv.FormatBool(*alarmPerformanceConfig.MemMail)}
	properties[8] = model.Property{Name: "mem-syslog", Value: strconv.FormatBool(*alarmPerformanceConfig.MemSyslog)}
	properties[9] = model.Property{Name: "mem-level", Value: alarmPerformanceConfig.MemLevel}

	properties[10] = model.Property{Name: "disk-max", Value: alarmPerformanceConfig.DiskMax}
	properties[11] = model.Property{Name: "disk-msg", Value: strconv.FormatBool(*alarmPerformanceConfig.DiskMsg)}
	properties[12] = model.Property{Name: "disk-mail", Value: strconv.FormatBool(*alarmPerformanceConfig.DiskMail)}
	properties[13] = model.Property{Name: "disk-syslog", Value: strconv.FormatBool(*alarmPerformanceConfig.DiskSyslog)}
	properties[14] = model.Property{Name: "disk-level", Value: alarmPerformanceConfig.DiskLevel}

	properties[15] = model.Property{Name: "data-max", Value: alarmPerformanceConfig.VisualMemMax}
	properties[16] = model.Property{Name: "data-msg", Value: strconv.FormatBool(*alarmPerformanceConfig.VisualMemMsg)}
	properties[17] = model.Property{Name: "data-mail", Value: strconv.FormatBool(*alarmPerformanceConfig.VisualMemMail)}
	properties[18] = model.Property{Name: "data-syslog", Value: strconv.FormatBool(*alarmPerformanceConfig.VisualMemSyslog)}
	properties[19] = model.Property{Name: "data-level", Value: alarmPerformanceConfig.VisualMemLevel}

	for _, v := range properties {
		if err := propertyRepository.Update(&v); nil != err {
			log.Errorf("DB Error: %v", err)
			log.Errorf("系统配置-告警配置: 修改系统性能告警配置失败")
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
	}
	err := newJobService.AlarmConfigurationPerMonitoring()
	if err != nil {
		log.Errorf("系统配置-告警配置: 执行告警检测任务失败")
	}
	return SuccessWithOperate(c, "告警配置-修改: [系统配置中修改系统性能告警配置]", nil)
}

// SystemAccessFormGetEndpoint 获取系统性能表单形式
func SystemAccessFormGetEndpoint(c echo.Context) error {
	alarmUserAccessConfigMap, err := propertyRepository.FindMapByNames(constant.AlarmConfigAccess)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		log.Errorf("系统配置-告警配置: 获取系统访问量配置失败")
	}
	getNum := func(key string) int {
		num, err := strconv.Atoi(key)
		if nil != err {
			log.Errorf("Atoi Error: %v", err)
			return 0
		}
		return num
	}
	alarmConfig := make([]model.AlarmConfig, 6)
	alarmConfig[0] = model.AlarmConfig{
		Event:          "用户在线数",
		ThresholdValue: getNum(alarmUserAccessConfigMap["user-max"]),
		IsMail:         utils.StrToBoolPtr(alarmUserAccessConfigMap["user-mail"]),
		IsMessage:      utils.StrToBoolPtr(alarmUserAccessConfigMap["user-msg"]),
		IsSyslog:       utils.StrToBoolPtr(alarmUserAccessConfigMap["user-syslog"]),
		AlarmLevel:     alarmUserAccessConfigMap["user-level"],
	}
	alarmConfig[1] = model.AlarmConfig{
		Event:          "SSH协议在线数",
		ThresholdValue: getNum(alarmUserAccessConfigMap["ssh-max"]),
		IsMail:         utils.StrToBoolPtr(alarmUserAccessConfigMap["ssh-mail"]),
		IsMessage:      utils.StrToBoolPtr(alarmUserAccessConfigMap["ssh-msg"]),
		IsSyslog:       utils.StrToBoolPtr(alarmUserAccessConfigMap["ssh-syslog"]),
		AlarmLevel:     alarmUserAccessConfigMap["ssh-level"],
	}
	alarmConfig[2] = model.AlarmConfig{
		Event:          "RDP协议在线数",
		ThresholdValue: getNum(alarmUserAccessConfigMap["rdp-max"]),
		IsMail:         utils.StrToBoolPtr(alarmUserAccessConfigMap["rdp-mail"]),
		IsMessage:      utils.StrToBoolPtr(alarmUserAccessConfigMap["rdp-msg"]),
		IsSyslog:       utils.StrToBoolPtr(alarmUserAccessConfigMap["rdp-syslog"]),
		AlarmLevel:     alarmUserAccessConfigMap["rdp-level"],
	}
	alarmConfig[3] = model.AlarmConfig{
		Event:          "VNC协议在线数",
		ThresholdValue: getNum(alarmUserAccessConfigMap["vnc-max"]),
		IsMail:         utils.StrToBoolPtr(alarmUserAccessConfigMap["vnc-mail"]),
		IsMessage:      utils.StrToBoolPtr(alarmUserAccessConfigMap["vnc-msg"]),
		IsSyslog:       utils.StrToBoolPtr(alarmUserAccessConfigMap["vnc-syslog"]),
		AlarmLevel:     alarmUserAccessConfigMap["vnc-level"],
	}
	alarmConfig[4] = model.AlarmConfig{
		Event:          "TELNET协议在线数",
		ThresholdValue: getNum(alarmUserAccessConfigMap["telnet-max"]),
		IsMail:         utils.StrToBoolPtr(alarmUserAccessConfigMap["telnet-mail"]),
		IsMessage:      utils.StrToBoolPtr(alarmUserAccessConfigMap["telnet-msg"]),
		IsSyslog:       utils.StrToBoolPtr(alarmUserAccessConfigMap["telnet-syslog"]),
		AlarmLevel:     alarmUserAccessConfigMap["telnet-level"],
	}
	alarmConfig[5] = model.AlarmConfig{
		Event:          "应用在线数",
		ThresholdValue: getNum(alarmUserAccessConfigMap["app-max"]),
		IsMail:         utils.StrToBoolPtr(alarmUserAccessConfigMap["app-mail"]),
		IsMessage:      utils.StrToBoolPtr(alarmUserAccessConfigMap["app-msg"]),
		IsSyslog:       utils.StrToBoolPtr(alarmUserAccessConfigMap["app-syslog"]),
		AlarmLevel:     alarmUserAccessConfigMap["app-level"],
	}
	return Success(c, alarmConfig)
}

// AlarmSysAccessGetEndpoint 获取系统访问量告警
func AlarmSysAccessGetEndpoint(c echo.Context) error {
	alarmUserAccessConfigMap, err := propertyRepository.FindMapByNames(constant.AlarmConfigAccess)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		log.Errorf("系统配置-告警配置: 获取系统访问量配置失败")
	}
	var AlarmUserAccessConfig = model.AlarmUserAccessConfig{
		UserMax:      alarmUserAccessConfigMap["user-max"],
		UserMsg:      utils.StrToBoolPtr(alarmUserAccessConfigMap["user-msg"]),
		UserMail:     utils.StrToBoolPtr(alarmUserAccessConfigMap["user-mail"]),
		UserSyslog:   utils.StrToBoolPtr(alarmUserAccessConfigMap["user-syslog"]),
		UserLevel:    alarmUserAccessConfigMap["user-level"],
		SshMax:       alarmUserAccessConfigMap["ssh-max"],
		SshMsg:       utils.StrToBoolPtr(alarmUserAccessConfigMap["ssh-msg"]),
		SshMail:      utils.StrToBoolPtr(alarmUserAccessConfigMap["ssh-mail"]),
		SshSyslog:    utils.StrToBoolPtr(alarmUserAccessConfigMap["ssh-syslog"]),
		SshLevel:     alarmUserAccessConfigMap["ssh-level"],
		RdpMax:       alarmUserAccessConfigMap["rdp-max"],
		RdpMsg:       utils.StrToBoolPtr(alarmUserAccessConfigMap["rdp-msg"]),
		RdpMail:      utils.StrToBoolPtr(alarmUserAccessConfigMap["rdp-mail"]),
		RdpSyslog:    utils.StrToBoolPtr(alarmUserAccessConfigMap["rdp-syslog"]),
		RdpLevel:     alarmUserAccessConfigMap["rdp-level"],
		TelnetMax:    alarmUserAccessConfigMap["telnet-max"],
		TelnetMsg:    utils.StrToBoolPtr(alarmUserAccessConfigMap["telnet-msg"]),
		TelnetMail:   utils.StrToBoolPtr(alarmUserAccessConfigMap["telnet-mail"]),
		TelnetSyslog: utils.StrToBoolPtr(alarmUserAccessConfigMap["telnet-syslog"]),
		TelnetLevel:  alarmUserAccessConfigMap["telnet-level"],
		VncMax:       alarmUserAccessConfigMap["vnc-max"],
		VncMsg:       utils.StrToBoolPtr(alarmUserAccessConfigMap["vnc-msg"]),
		VncMail:      utils.StrToBoolPtr(alarmUserAccessConfigMap["vnc-mail"]),
		VncSyslog:    utils.StrToBoolPtr(alarmUserAccessConfigMap["vnc-syslog"]),
		VncLevel:     alarmUserAccessConfigMap["vnc-level"],
		AppMax:       alarmUserAccessConfigMap["app-max"],
		AppMsg:       utils.StrToBoolPtr(alarmUserAccessConfigMap["app-msg"]),
		AppMail:      utils.StrToBoolPtr(alarmUserAccessConfigMap["app-mail"]),
		AppSyslog:    utils.StrToBoolPtr(alarmUserAccessConfigMap["app-syslog"]),
		AppLevel:     alarmUserAccessConfigMap["app-level"],
	}
	return Success(c, AlarmUserAccessConfig)
}

// AlarmSysAccessUpdateEndpoint 编辑系统访问量告警
func AlarmSysAccessUpdateEndpoint(c echo.Context) error {
	var alarmUserAccessConfig model.AlarmUserAccessConfig
	if err := c.Bind(&alarmUserAccessConfig); err != nil {
		log.Errorf("Bind Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	properties := make([]model.Property, 30)
	properties[0] = model.Property{Name: "user-max", Value: alarmUserAccessConfig.UserMax}
	properties[1] = model.Property{Name: "user-msg", Value: strconv.FormatBool(*alarmUserAccessConfig.UserMsg)}
	properties[2] = model.Property{Name: "user-mail", Value: strconv.FormatBool(*alarmUserAccessConfig.UserMail)}
	properties[3] = model.Property{Name: "user-syslog", Value: strconv.FormatBool(*alarmUserAccessConfig.UserSyslog)}
	properties[4] = model.Property{Name: "user-level", Value: alarmUserAccessConfig.UserLevel}
	properties[5] = model.Property{Name: "ssh-max", Value: alarmUserAccessConfig.SshMax}
	properties[6] = model.Property{Name: "ssh-msg", Value: strconv.FormatBool(*alarmUserAccessConfig.SshMsg)}
	properties[7] = model.Property{Name: "ssh-mail", Value: strconv.FormatBool(*alarmUserAccessConfig.SshMail)}
	properties[8] = model.Property{Name: "ssh-syslog", Value: strconv.FormatBool(*alarmUserAccessConfig.SshSyslog)}
	properties[9] = model.Property{Name: "ssh-level", Value: alarmUserAccessConfig.SshLevel}
	properties[10] = model.Property{Name: "rdp-max", Value: alarmUserAccessConfig.RdpMax}
	properties[11] = model.Property{Name: "rdp-msg", Value: strconv.FormatBool(*alarmUserAccessConfig.RdpMsg)}
	properties[12] = model.Property{Name: "rdp-mail", Value: strconv.FormatBool(*alarmUserAccessConfig.RdpMail)}
	properties[13] = model.Property{Name: "rdp-syslog", Value: strconv.FormatBool(*alarmUserAccessConfig.RdpSyslog)}
	properties[14] = model.Property{Name: "rdp-level", Value: alarmUserAccessConfig.RdpLevel}
	properties[15] = model.Property{Name: "telnet-max", Value: alarmUserAccessConfig.TelnetMax}
	properties[16] = model.Property{Name: "telnet-msg", Value: strconv.FormatBool(*alarmUserAccessConfig.TelnetMsg)}
	properties[17] = model.Property{Name: "telnet-mail", Value: strconv.FormatBool(*alarmUserAccessConfig.TelnetMail)}
	properties[18] = model.Property{Name: "telnet-syslog", Value: strconv.FormatBool(*alarmUserAccessConfig.TelnetSyslog)}
	properties[19] = model.Property{Name: "telnet-level", Value: alarmUserAccessConfig.TelnetLevel}
	properties[20] = model.Property{Name: "vnc-max", Value: alarmUserAccessConfig.VncMax}
	properties[21] = model.Property{Name: "vnc-msg", Value: strconv.FormatBool(*alarmUserAccessConfig.VncMsg)}
	properties[22] = model.Property{Name: "vnc-mail", Value: strconv.FormatBool(*alarmUserAccessConfig.VncMail)}
	properties[23] = model.Property{Name: "vnc-syslog", Value: strconv.FormatBool(*alarmUserAccessConfig.VncSyslog)}
	properties[24] = model.Property{Name: "vnc-level", Value: alarmUserAccessConfig.VncLevel}
	properties[25] = model.Property{Name: "app-max", Value: alarmUserAccessConfig.AppMax}
	properties[26] = model.Property{Name: "app-msg", Value: strconv.FormatBool(*alarmUserAccessConfig.AppMsg)}
	properties[27] = model.Property{Name: "app-mail", Value: strconv.FormatBool(*alarmUserAccessConfig.AppMail)}
	properties[28] = model.Property{Name: "app-syslog", Value: strconv.FormatBool(*alarmUserAccessConfig.AppSyslog)}
	properties[29] = model.Property{Name: "app-level", Value: alarmUserAccessConfig.AppLevel}
	for _, property := range properties {
		if err := propertyRepository.Update(&property); err != nil {
			log.Errorf("DB Error: %v", err)
			log.Errorf("系统配置-告警配置: 修改系统访问量配置失败")
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
	}
	err := newJobService.AlarmConfigurationAccMonitoring()
	if err != nil {
		log.Errorf("系统配置-告警配置: 执行系统访问量告警检测任务失败")
	}
	return SuccessWithOperate(c, "告警配置-修改: [系统配置中修改系统访问量告警配置]", nil)
}

// ExtendConfigGetEndpoint 扩展配置-获取
func ExtendConfigGetEndpoint(c echo.Context) error {
	var extendConfig []model.ExtendConfig
	err := identityConfigRepository.DB.Table("extend_config").Order("priority asc").Find(&extendConfig).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", err)
	}
	var extendConfigList = make([]model.ExtendConfigDTO, len(extendConfig))
	for i, v := range extendConfig {
		extendConfigList[i] = v.ExtendConfigToDTO()
	}
	return Success(c, extendConfigList)
}

// ExtendConfigCreateEndpoint 扩展配置-新建
func ExtendConfigCreateEndpoint(c echo.Context) error {
	var extend model.ExtendConfigDTO
	if err := c.Bind(&extend); err != nil {
		log.Errorf("Bind Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "新建失败", "", err)
	}

	if err := c.Validate(&extend); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	// 检查是否存在
	var extendConfigExists []model.ExtendConfig
	err := identityConfigRepository.DB.Table("extend_config").Where("name = ?", extend.Name).Find(&extendConfigExists).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}
	if len(extendConfigExists) > 0 {
		return FailWithDataOperate(c, 500, "扩展配置-新建: [扩展配置"+extend.Name+"已存在]", "", nil)
	}
	extendConfig := extend.DTOtoExtendConfig()
	if extendConfig.Priority <= 0 || extendConfig.Priority > 10 {
		return FailWithDataOperate(c, 500, "排序范围为1-10", "", nil)
	}
	var count int64

	if err = identityConfigRepository.DB.Table("extend_config").Count(&count).Error; nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新建失败", "", err)
	}
	if count >= 10 {
		log.Errorf("扩展配置-新建: [扩展配置数量已达上限: 10]")
		return FailWithDataOperate(c, 500, "扩展配置数量已达上限: 10", "", nil)
	}
	if extendConfig.Priority >= int(count) {
		extendConfig.Priority = int(count) + 1
	} else {
		err := identityConfigRepository.DB.Table("extend_config").Where("priority >= ?", extendConfig.Priority).Update("priority", gorm.Expr("priority + ?", 1)).Error
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "新建失败", "", err)
		}
	}
	extendConfig.ID = utils.UUID()
	if err := identityConfigRepository.DB.Table("extend_config").Create(&extendConfig).Error; err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新建失败", "", err)
	}
	return SuccessWithOperate(c, "系统配置-扩展配置-新建: ["+extendConfig.Name+"]", nil)
}

// ExtendConfigUpdateEndpoint 扩展配置-修改
func ExtendConfigUpdateEndpoint(c echo.Context) error {
	var extend model.ExtendConfigDTO
	var extendConfigOld model.ExtendConfig
	if err := c.Bind(&extend); err != nil {
		log.Errorf("Bind Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if err := c.Validate(&extend); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	// 检查是否存在
	var extendConfigExists []model.ExtendConfig
	err := identityConfigRepository.DB.Table("extend_config").Where("name = ? and id != ? ", extend.Name, extend.ID).Find(&extendConfigExists).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}
	if len(extendConfigExists) > 0 {
		return FailWithDataOperate(c, 500, "扩展配置-新建: [扩展配置"+extend.Name+"已存在]", "", nil)
	}
	extendConfig := extend.DTOtoExtendConfig()
	if extendConfig.Priority <= 0 || extendConfig.Priority > 10 {
		return FailWithDataOperate(c, 500, "排序范围为1-10", "", nil)
	}
	var count int64

	if err = identityConfigRepository.DB.Table("extend_config").Count(&count).Error; nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if err := identityConfigRepository.DB.Table("extend_config").Where("id = ?", extend.ID).First(&extendConfigOld).Error; nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if extendConfig.Priority > extendConfigOld.Priority {
		err := identityConfigRepository.DB.Table("extend_config").Where("priority > ? and priority <= ?", extendConfigOld.Priority, extendConfig.Priority).Update("priority", gorm.Expr("priority - ?", 1)).Error
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
	} else if extendConfig.Priority < extendConfigOld.Priority {
		err := identityConfigRepository.DB.Table("extend_config").Where("priority >= ? and priority < ?", extendConfig.Priority, extendConfigOld.Priority).Update("priority", gorm.Expr("priority + ?", 1)).Error
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
	}
	if extendConfig.Priority > int(count) {
		extendConfig.Priority = int(count)
	}
	if err := identityConfigRepository.DB.Table("extend_config").Where("id = ?", extendConfig.ID).Updates(&extendConfig).Error; nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	return SuccessWithOperate(c, "扩展配置-修改: 修改系统配置-扩展配置名称["+extendConfig.Name+"]", nil)
}

// ExtendConfigDeleteEndpoint 扩展配置-删除
func ExtendConfigDeleteEndpoint(c echo.Context) error {
	id := c.Param("id")
	split := strings.Split(id, ",")
	successName := ""
	successCount := 0
	for _, v := range split {
		var extendConfig model.ExtendConfig
		if err := identityConfigRepository.DB.Table("extend_config").Where("id = ?", v).Find(&extendConfig).Error; err != nil {
			log.Errorf("DB Error: %v", err)
			continue
		}
		err := identityConfigRepository.DB.Table("extend_config").Where("id = ?", v).Delete(model.ExtendConfig{}).Error
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "删除失败", "", err)
		}
		successName += extendConfig.Name + ","
		successCount++
	}
	if len(successName) > 0 {
		successName = successName[:len(successName)-1]
	}
	var extendConfig []model.ExtendConfig
	err := identityConfigRepository.DB.Table("extend_config").Order("priority asc").Find(&extendConfig).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}
	for i, v := range extendConfig {
		err := identityConfigRepository.DB.Table("extend_config").Where("id = ?", v.ID).Update("priority", i+1).Error
		if nil != err {
			log.Errorf("DB Error: %v", err)
		}
	}
	return SuccessWithOperate(c, "扩展配置-删除: 删除系统配置-扩展配置[删除成功名称:"+successName+",删除成功数: "+strconv.Itoa(successCount)+"]", nil)
}
