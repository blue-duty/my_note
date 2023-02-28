package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/repository"
	"tkbastion/server/service"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
)

// PasswordAuthEndpoint 密码认证
func PasswordAuthEndpoint(c echo.Context) error {
	var loginAccount LoginAccount
	if err := c.Bind(&loginAccount); err != nil {
		log.Errorf("Bind Error: %v", err)
		return err
	}

	// 获取系统管理员角色
	role, err := roleRepository.FindByRoleName("系统管理员")
	if err != nil {
		return FailWithDataOperate(c, 401, "密码错误", "密码查看-认证: 获取系统管理员角色失败", nil)
	}

	user, err := userNewRepository.FindByName(loginAccount.Username)
	if err != nil {
		return NotFound(c, "用户名未找到")
	}

	// 判断用户是否有系统管理员角色
	if user.RoleId != role.ID {
		return FailWithDataOperate(c, 401, "密码错误", "密码查看-认证: 用户不是系统管理员", nil)
	}

	if err := utils.Encoder.Match([]byte(user.Password), []byte(loginAccount.Password)); err != nil {
		return FailWithDataOperate(c, 401, "密码错误", "密码查看-认证: 系统管理员密码认证失败", nil)
	}
	return SuccessWithOperate(c, "密码查看-认证: 系统管理员密码认证成功", nil)
}

// HasExportPasswordEndpoint 是否存在导出密码验证
func HasExportPasswordEndpoint(c echo.Context) error {
	p, err := propertyRepository.FindByName("export_password")
	if err != nil {
		return Success(c, false)
	}

	if p.Value == "" {
		return Success(c, false)
	}
	return Success(c, true)
}

// PasswordExportAuthEndpoint 导出密码认证
func PasswordExportAuthEndpoint(c echo.Context) error {
	type Password struct {
		Password string `json:"password"`
	}

	var pw Password
	if err := c.Bind(&pw); err != nil {
		log.Errorf("Bind Error: %v", err)
		return err
	}

	p, err := propertyRepository.FindByName("export_password")
	if err != nil {
		return FailWithDataOperate(c, 500, "密码验证失败", "", nil)
	}
	fmt.Println("fmt", p.Value)
	fmt.Println(pw.Password)
	if err := utils.Encoder.Match([]byte(p.Value), []byte(pw.Password)); err != nil {
		return FailWithDataOperate(c, 401, "密码错误", "密码导出-认证: 导出密码认证失败", nil)
	}

	return SuccessWithOperate(c, "密码导出-认证: 导出密码认证成功", nil)
}

// PasswordViewPagingEndpoint 密码查看页面
func PasswordViewPagingEndpoint(c echo.Context) error {
	auto := c.QueryParam("auto")
	passport := c.QueryParam("passport")
	assetName := c.QueryParam("asset_name")
	assetIp := c.QueryParam("asset_ip")
	systemType := c.QueryParam("system_type")
	protocol := c.QueryParam("protocol")

	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}

	resp, err := newAssetRepository.GetPassportRecord(context.TODO(), auto, passport, assetIp, assetName, systemType, protocol)
	if err != nil {
		log.Error("获取密码记录失败", err)
	}

	return Success(c, resp)
}

// PasswordViewGetEndpoint 查看单条密码记录
func PasswordViewGetEndpoint(c echo.Context) error {
	id := c.Param("id")
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}

	resp, err := newAssetRepository.GetPassportRecordById(context.TODO(), id)
	if err != nil {
		log.Error("获取密码记录失败", err)
	}

	return Success(c, resp)
}

// PasswordViewExportEndpoint 导出密码记录
func PasswordViewExportEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	u, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}

	var passportForExport []dto.PassportWithPasswordForExport
	var err error
	if id == "all" {
		passportForExport, err = repository.AssetNewDao.GetPassportWithPasswdForExport(context.TODO(), "")
	} else {
		passportForExport, err = repository.AssetNewDao.GetPassportWithPasswdForExport(context.TODO(), id)
	}
	if err != nil {
		log.Error("获取密码记录失败", err)
	}

	forExport := make([][]string, len(passportForExport))
	for i, v := range passportForExport {
		pp := utils.Struct2StrArr(v)
		forExport[i] = make([]string, len(pp))
		forExport[i] = pp
	}

	headerForExport := []string{"设备名称", "设备地址", "部门名称", "系统类型", "描述", "设备账号名称", "设备账号密码", "端口", "ssh_key", "设备账号协议类型", "登录方式", "SFTP访问路径", "设备账号角色", "状态"}
	nameForExport := "设备列表"
	file, err := utils.CreateExcelFile(nameForExport, headerForExport, forExport)
	if err != nil {
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	fileName := "设备列表.xlsx"

	// 记录操作日志
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		Result:          "成功",
		LogTypes:        "用户日志",
		Created:         utils.NowJsonTime(),
		LogContents:     "密码查看-导出: 导出设备密码信息, 文件[" + fileName + "]",
		Users:           u.Username,
		Names:           u.Nickname,
	}
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}

	var buff bytes.Buffer
	if err = file.Write(&buff); err != nil {
		log.Errorf("Write Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "", err)
	}

	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
}

// PasswordPolicyPagingEndpoint 查看密码策略
func PasswordPolicyPagingEndpoint(c echo.Context) error {
	auto := c.QueryParam("auto")
	name := c.QueryParam("name")
	runType := c.QueryParam("run_type")
	genRule := c.QueryParam("gen_rule")

	resp, err := repository.PasswdChangeRepo.Find(context.TODO(), name, runType, genRule, auto)
	if err != nil {
		log.Error("获取密码策略失败", err)
	}

	return Success(c, resp)
}

// PasswordPolicyCreateEndpoint 新增密码策略
func PasswordPolicyCreateEndpoint(c echo.Context) error {
	var req model.PasswdChange
	if err := c.Bind(&req); err != nil {
		log.Error("绑定参数失败", err)
		return Fail(c, 400, "参数错误")
	}

	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}

	f, err := repository.PasswdChangeRepo.NameIsDuplicate(context.TODO(), req.Name, "")
	if err != nil {
		return FailWithDataOperate(c, 500, "新建失败", "", nil)
	}
	if f {
		return FailWithDataOperate(c, 403, "策略名称已存在", "密码策略-新增: 策略名称["+req.Name+"]已存在", nil)
	}

	req.ID = utils.UUID()
	switch req.RunType {
	case "Manual":
		req.RunTypeName = "手动执行"
	case "Scheduled":
		req.RunTypeName = "定时执行"
	case "Periodic":
		req.RunTypeName = "周期执行"
	default:
		return Fail(c, 400, "请选择执行方式")
	}

	switch req.GenerateRule {
	case 1:
		req.GenerateRuleName = "生成不同密码"
	case 2:
		req.GenerateRuleName = "生成相同密码"
	case 3:
		req.GenerateRuleName = "指定相同密码"
	default:
		return Fail(c, 400, "请选择生成规则")
	}

	encryptedCBC, err := utils.AesEncryptCBC([]byte(req.Password), global.Config.EncryptionPassword)
	if err != nil {
		log.Error("加密失败", err)
		return FailWithDataOperate(c, 402, "新建失败", "密码策略-新增: 加密失败", err)
	}
	req.Password = base64.StdEncoding.EncodeToString(encryptedCBC)

	if err := repository.PasswdChangeRepo.Create(context.TODO(), &req, utils.IdHandle(req.DriveIds), utils.IdHandle(req.DriveGroups)); err != nil {
		log.Error("新增密码策略失败", err)
		return Fail(c, 500, "新增密码策略失败")
	}

	if req.RunType != "Manual" {
		err := service.CreateAutoChangePasswdTask(req.ID)
		if err != nil {
			log.Error("创建自动修改密码任务失败", err)
			return Fail(c, 500, "创建自动修改密码任务失败")
		}
	}

	return SuccessWithOperate(c, "密码策略-新增: 新增改密策略["+req.Name+"]成功", nil)
}

// PasswordPolicyUpdateEndpoint 修改密码策略
func PasswordPolicyUpdateEndpoint(c echo.Context) error {
	var req model.PasswdChange
	if err := c.Bind(&req); err != nil {
		log.Error("绑定参数失败", err)
		return Fail(c, 400, "参数错误")
	}

	id := c.Param("id")
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}

	f, err := repository.PasswdChangeRepo.NameIsDuplicate(context.TODO(), req.Name, id)
	if err != nil {
		return FailWithDataOperate(c, 500, "编辑失败", "", nil)
	}
	if f {
		return FailWithDataOperate(c, 403, "策略名称已存在", "密码策略-编辑: 策略名称["+req.Name+"]已存在", nil)
	}

	pw, err := repository.PasswdChangeRepo.FindByID(context.TODO(), id)
	if err != nil {
		log.Error("获取密码策略失败", err)
		return Fail(c, 500, "获取密码策略失败")
	}

	req.ID = id

	switch req.RunType {
	case "Manual":
		req.RunTypeName = "手动执行"
	case "Scheduled":
		req.RunTypeName = "定时执行"
	case "Periodic":
		req.RunTypeName = "周期执行"
	default:
		return Fail(c, 400, "请选择执行方式")
	}

	switch req.GenerateRule {
	case 2:
		req.GenerateRuleName = "生成相同密码"
	case 1:
		req.GenerateRuleName = "生成不同密码"
	case 3:
		req.GenerateRuleName = "指定相同密码"
	default:
		return Fail(c, 400, "请选择生成规则")
	}

	if err := repository.PasswdChangeRepo.Update(context.TODO(), &req); err != nil {
		log.Error("修改密码策略失败", err)
		return Fail(c, 500, "修改密码策略失败")
	}

	if pw.RunType != "Manual" {
		err = global.SCHEDULER.RemoveByTag(pw.ID)
		if err != nil {
			log.Error("删除定时任务失败", err)
		}
	}

	if req.RunType != "Manual" {
		err = service.CreateAutoChangePasswdTask(pw.ID)
		if err != nil {
			log.Error("创建自动修改密码任务失败", err)
			return FailWithDataOperate(c, 500, "修改密码策略失败", "密码策略-修改: 创建自动修改密码任务失败", err)
		}
	}

	return SuccessWithOperate(c, "改密策略-修改: 修改改密策略["+pw.Name+"->"+req.Name+"]成功", nil)
}

// PasswordPolicyDeleteEndpoint 删除密码策略
func PasswordPolicyDeleteEndpoint(c echo.Context) error {
	id := c.Param("id")

	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}

	ids := strings.Split(id, ",")
	name := ""
	for _, id := range ids {
		pw, err := repository.PasswdChangeRepo.FindByID(context.TODO(), id)
		if err != nil {
			log.Error("获取密码策略失败", err)
			return Fail(c, 500, "获取密码策略失败")
		}

		if err := repository.PasswdChangeRepo.Delete(context.TODO(), id); err != nil {
			log.Error("删除密码策略失败", err)
			return Fail(c, 400, "删除密码策略失败")
		}

		if pw.RunType != "Manual" {
			_, err = global.SCHEDULER.FindJobsByTag(pw.ID)
			if err != nil {
				log.Error("未找到定时任务", err)
			} else {
				err = global.SCHEDULER.RemoveByTag(pw.ID)
				if err != nil {
					log.Error("删除定时任务失败", err)
					return Fail(c, 500, "删除定时任务失败")
				}
			}
		}
		name += pw.Name + ","
	}

	return SuccessWithOperate(c, "改密策略-删除: 删除改密策略["+name+"]成功", nil)
}

// PasswordPolicyRunNow 执行密码策略
func PasswordPolicyRunNow(c echo.Context) error {
	id := c.Param("id")

	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}

	pc, err := repository.PasswdChangeRepo.FindByID(context.TODO(), id)
	if err != nil {
		log.Error("获取密码策略失败", err)
		return Fail(c, 500, "获取密码策略失败")
	}

	err = service.PasswdChangeNow(id)
	if err != nil {
		log.Error("立即执行密码策略失败", err)
		return FailWithDataOperate(c, 201, "执行失败", "改密策略-立即执行: 立即执行改密策略["+pc.Name+"]失败", nil)
	}

	return SuccessWithOperate(c, "改密策略-立即执行: 立即执行改密策略["+pc.Name+"]成功", nil)
}

// PasswordPolicyGetEndpoint 获取密码策略
func PasswordPolicyGetEndpoint(c echo.Context) error {
	id := c.Param("id")
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}

	resp, err := repository.PasswdChangeRepo.GetPasswordPolcyByIdForEdit(context.TODO(), id)
	if err != nil {
		log.Error("获取密码策略失败", err)
		return Fail(c, 500, "获取密码策略失败")
	}

	return Success(c, resp)
}

func PasswordPolicyGetAllSshDeviceEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}

	resp, err := repository.PasswdChangeRepo.GetAllSshDevice(context.TODO())
	if err != nil {
		log.Error("获取所有SSH设备失败", err)
		return Fail(c, 500, "获取所有SSH设备失败")
	}

	return Success(c, resp)
}

func PasswordPolicyGetAllDeviceGroupEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}

	resp, err := repository.PasswdChangeRepo.GetAllhDeviceGroup(context.TODO())
	if err != nil {
		log.Error("获取所有设备组失败", err)
		return Fail(c, 500, "获取所有设备组失败")
	}

	return Success(c, resp)
}

// PasswordPolicyRelateAssetEndpoint 密码策略关联资产
func PasswordPolicyRelateAssetEndpoint(c echo.Context) error {
	id := c.Param("id")

	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}

	resp, err := repository.PasswdChangeRepo.FindPasswdChangeDevice(context.TODO(), id)
	if err != nil {
		log.Error("获取密码策略关联资产失败", err)
	}

	return Success(c, resp)
}

// PasswordPolicyUpdateRelateAssetEndpoint 修改密码策略关联资产
func PasswordPolicyUpdateRelateAssetEndpoint(c echo.Context) error {
	id := c.Param("id")
	assetIds := c.QueryParam("ids")

	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}
	assets := strings.Split(assetIds, ",")

	pw, err := repository.PasswdChangeRepo.FindByID(context.TODO(), id)
	if err != nil {
		log.Error("获取密码策略失败", err)
		return Fail(c, 500, "获取密码策略失败")
	}

	if err := repository.PasswdChangeRepo.UpdatePasswdChangeDevice(context.TODO(), id, assets); err != nil {
		log.Error("修改密码策略关联资产失败", err)
		return Fail(c, 500, "关联设备失败")
	}

	return SuccessWithOperate(c, "改密策略-关联资产: 策略["+pw.Name+"]修改关联资产成功", nil)
}

// PasswordPolicyRelateAssetGroupEndpoint 密码策略关联资产组
func PasswordPolicyRelateAssetGroupEndpoint(c echo.Context) error {
	id := c.Param("id")

	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}

	resp, err := repository.PasswdChangeRepo.FindPasswdChangeDeviceGroup(context.TODO(), id)
	if err != nil {
		log.Error("获取密码策略关联资产组失败", err)
	}

	return Success(c, resp)
}

// PasswordPolicyUpdateRelateAssetGroupEndpoint 修改密码策略关联资产组
func PasswordPolicyUpdateRelateAssetGroupEndpoint(c echo.Context) error {
	id := c.Param("id")
	assetGroupIds := c.QueryParam("ids")

	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}
	assetGroups := strings.Split(assetGroupIds, ",")

	pw, err := repository.PasswdChangeRepo.FindByID(context.TODO(), id)
	if err != nil {
		log.Error("获取密码策略失败", err)
		return Fail(c, 500, "获取密码策略失败")
	}

	if err := repository.PasswdChangeRepo.UpdatePasswdChangeDeviceGroup(context.TODO(), id, assetGroups); err != nil {
		log.Error("修改密码策略关联资产组失败", err)
		return Fail(c, 500, "关联资产组失败")
	}

	return SuccessWithOperate(c, "改密策略-关联资产组: 策略["+pw.Name+"]修改关联资产组成功", nil)
}

// PasswordRecordPagingEndpoint 密码记录
func PasswordRecordPagingEndpoint(c echo.Context) error {
	auto := c.QueryParam("auto")
	assetIp := c.QueryParam("asset_ip")
	assetName := c.QueryParam("asset_name")
	passport := c.QueryParam("passport")
	name := c.QueryParam("name")
	result := c.QueryParam("result")

	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}

	resp, err := repository.PasswdChangeRepo.FindPasswdChangeLog(context.TODO(), auto, name, assetName, assetIp, passport, result)
	if err != nil {
		log.Error("获取修改密码记录失败", err)
	}

	return Success(c, resp)
}

// PasswordRecordStatisticsEndpoint 密码记录统计
func PasswordRecordStatisticsEndpoint(c echo.Context) error {
	auto := c.QueryParam("auto")
	assetIp := c.QueryParam("asset_ip")
	assetName := c.QueryParam("asset_name")
	passport := c.QueryParam("passport")

	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}

	resp, err := repository.PasswdChangeRepo.FindPasswdChangeLogStatistical(context.TODO(), auto, assetName, assetIp, passport)
	if err != nil {
		log.Error("获取密码记录统计失败", err)
	}

	return Success(c, resp)
}

// PasswordRecordStatisticsDetailEndpoint 密码记录统计详情
func PasswordRecordStatisticsDetailEndpoint(c echo.Context) error {
	assetIp := c.QueryParam("asset_ip")
	result := c.QueryParam("result")

	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}

	var err error
	var resp []dto.PasswdChangeResult
	if result == "success" {
		resp, err = repository.PasswdChangeRepo.FindPasswdChangeLogStatisticalDetail(context.TODO(), assetIp, true)
	} else {
		resp, err = repository.PasswdChangeRepo.FindPasswdChangeLogStatisticalDetail(context.TODO(), assetIp, false)
	}
	if err != nil {
		log.Error("获取密码记录统计详情失败", err)
	}

	return Success(c, resp)
}
