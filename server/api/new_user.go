package api

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"tkbastion/pkg/constant"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/global"
	"tkbastion/pkg/totp"
	"tkbastion/pkg/validator"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/xuri/excelize/v2"
)

// UserNewCreateEndpoint 创建用户
func UserNewCreateEndpoint(c echo.Context) error {
	var userNew model.UserNew
	if err := c.Bind(&userNew); err != nil {
		log.Errorf("Bind Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}
	role, err := roleRepository.FindById(userNew.RoleId)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}

	if err := c.Validate(&userNew); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	if role.Name == constant.SystemAdmin && userNew.DepartmentId != 0 {
		return FailWithDataOperate(c, 500, "系统管理员只能在根部门", "", nil)
	}
	// 认证服务器用户校验
	if strings.Contains(userNew.AuthenticationWay, constant.LDAPOrAD) {
		if userNew.AuthenticationServerId == 0 {
			return FailWithDataOperate(c, 500, "认证服务器不能为空", "", nil)
		}
	}
	if strings.Contains(userNew.AuthenticationWay, constant.RADIUS) || strings.Contains(userNew.AuthenticationWay, constant.Email) {
		if userNew.AuthenticationServer == "" {
			return FailWithDataOperate(c, 500, "认证服务器不能为空", "", nil)
		}
	}
	// TOTP校验
	if strings.Contains(userNew.AuthenticationWay, constant.TOTP) {
		// 验证TOTP
		if !totp.Validate(userNew.TOTP, userNew.TOTPSecret) {
			return FailWithDataOperate(c, 400, "TOTP 验证失败, 请重试", "用户列表-新增: 失败原因[TOTP验证失败]", nil)
		}
	}
	// 校验密码
	if strings.Contains(userNew.AuthenticationWay, constant.StaticPassword) {
		if userNew.Password != userNew.VerifyPassword {
			return FailWithDataOperate(c, 500, "两次密码不一致", "", nil)
		}
		result, code, msg, err := CheckPasswordComplexity(userNew.Password)
		if !result {
			return FailWithDataOperate(c, code, msg, "", err)
		}
	}

	// 用户名不可重复
	var itemExists []model.UserNew
	if err := userNewRepository.DB.Where("username = ? ", userNew.Username).Find(&itemExists).Error; nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if 0 != len(itemExists) {
		return FailWithDataOperate(c, 403, "用户名或昵称已存在", "用户列表-新增: 用户名["+userNew.Username+"],失败原因["+userNew.Username+"已存在]", nil)
	}

	// 密码加密存储
	pwd := userNew.Password
	pass, err := utils.Encoder.Encode([]byte(pwd))
	if err != nil {
		log.Errorf("Encode Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}
	userNew.Password = string(pass)
	userNew.VerifyPassword = string(pass)
	userNew.SamePwdJudge = utils.StrJoin("", string(pass))
	userNew.ID = utils.UUID()
	userNew.Created = utils.NowJsonTime()
	userNew.PasswordUpdated = utils.NowJsonTime()
	userNew.Status = constant.Enable

	// 通过角色id找到角色名称
	userNew.RoleName, err = roleRepository.FindRoleNameById(userNew.RoleId)
	if err != nil {
		log.Errorf("FindRoleNameById Error: %v", err)
	}
	// 通过部门id找到部门名称
	Department, err := departmentRepository.FindById(userNew.DepartmentId)
	if err != nil {
		log.Errorf("FindById Error: %v", err)
	}
	userNew.DepartmentName = Department.Name
	// 判断邮件认证是否启用
	var TRUE = true
	var FALSE = false
	if strings.Contains(userNew.AuthenticationWay, constant.Email) {
		userNew.VerifyMailState = &TRUE
	} else {
		userNew.VerifyMailState = &FALSE
	}
	if userNew.IsRandomPassword && userNew.SendWay == constant.Email {
		item := propertyRepository.FindAuMap("mail")
		if "false" == item["mail-state"] {
			return FailWithDataOperate(c, 500, "邮件服务未启用", "", nil)
		}
		err := sysConfigService.SendMail([]string{userNew.Mail}, "随机密码", "登录用户名:"+userNew.Username+"\n   "+"随机密码"+pwd)
		if err != nil {
			log.Errorf("SendMail Error: %v", err)
			log.Errorf("发送邮件密码信息失败, 用户名: %v", userNew.Username)
			return FailWithDataOperate(c, 500, "密码邮件发送失败", "用户列表-新增: 用户名["+userNew.Username+"]", err)
		}
	}
	// 创建用户
	if err := userNewRepository.Create(&userNew); nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}
	UpdateUserAssetAppAppserverDep(constant.USER, constant.ADD, userNew.DepartmentId, int64(-1))

	return SuccessWithOperate(c, "用户列表-新增: 用户名["+userNew.Username+"]", userNew)
}

// UserNewUpdateEndpoint 修改用户
func UserNewUpdateEndpoint(c echo.Context) error {
	id := c.Param("id")
	oldUserNew, err := userNewRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	var userNew model.UserNew
	if err := c.Bind(&userNew); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	role, err := roleRepository.FindById(userNew.RoleId)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}

	if err := c.Validate(&userNew); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	if role.Name == constant.SystemAdmin && userNew.DepartmentId != 0 {
		return FailWithDataOperate(c, 500, "系统管理员只能在根部门", "", nil)
	}
	// 认证服务器用户校验
	if strings.Contains(userNew.AuthenticationWay, constant.LDAPOrAD) {
		if userNew.AuthenticationServerId == 0 {
			return FailWithDataOperate(c, 500, "认证服务器不能为空", "", nil)
		}
	}
	if strings.Contains(userNew.AuthenticationWay, constant.RADIUS) || strings.Contains(userNew.AuthenticationWay, constant.Email) {
		if userNew.AuthenticationServer == "" {
			return FailWithDataOperate(c, 500, "认证服务器不能为空", "", nil)
		}
	}
	// 校验TOTP----------
	// 1.原有用户是TOTP用户--->TOTP用户
	if strings.Contains(oldUserNew.AuthenticationWay, constant.TOTP) && strings.Contains(userNew.AuthenticationWay, constant.TOTP) {
		// 验证TOTP
		if totp.Validate(userNew.TOTP, oldUserNew.TOTPSecret) {
			userNew.TOTPSecret = oldUserNew.TOTPSecret
		} else if totp.Validate(userNew.TOTP, userNew.TOTPSecret) {
			oldUserNew.TOTPSecret = userNew.TOTPSecret
		} else {
			return FailWithDataOperate(c, 400, "TOTP 验证失败, 请重试", "用户列表-修改: 失败原因[TOTP验证失败]", nil)
		}
	}
	// 2.原有用户非TOTP用户--->TOTP用户
	if !strings.Contains(oldUserNew.AuthenticationWay, constant.TOTP) && strings.Contains(userNew.AuthenticationWay, constant.TOTP) {
		// 验证TOTP
		if !totp.Validate(userNew.TOTP, userNew.TOTPSecret) {
			return FailWithDataOperate(c, 400, "TOTP 验证失败, 请重试", "用户列表-修改: 失败原因[TOTP验证失败]", nil)
		}
	}

	// 校验密码
	var pwd string
	if userNew.Password != "" && strings.Contains(userNew.AuthenticationWay, constant.StaticPassword) {
		if userNew.Password == "" {
			return FailWithDataOperate(c, 500, "密码不能为空", "", nil)
		}
		if userNew.Password != userNew.VerifyPassword {
			return FailWithDataOperate(c, 500, "两次密码不一致", "", nil)
		}
		result, code, msg, err := CheckPasswordComplexity(userNew.Password)
		if !result {
			return FailWithDataOperate(c, code, msg, "", err)
		}
		// 密码加密存储
		pwd = userNew.Password
		var pass []byte
		if pass, err = utils.Encoder.Encode([]byte(pwd)); err != nil {
			log.Errorf("Encode Error: %v", err)
			return FailWithDataOperate(c, 500, "新增失败", "", err)
		}
		userNew.Password = string(pass)
		userNew.VerifyPassword = string(pass)
		userNew.SamePwdJudge = utils.StrJoin(oldUserNew.SamePwdJudge, string(pass))
		if ok := CheckPasswordRepeatTimes(oldUserNew.SamePwdJudge, pwd); !ok {
			return FailWithDataOperate(c, 500, "新密码不可与最近使用密码相同", "", nil)
		}
		userNew.PasswordUpdated = utils.NowJsonTime()
	}

	// 用户名不可重复
	var itemExists []model.UserNew
	if err := userNewRepository.DB.Where("username = ? and id != ?", userNew.Username, userNew.ID).Find(&itemExists).Error; nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if 0 != len(itemExists) {
		return FailWithDataOperate(c, 500, "用户名或昵称已存在", "用户列表-修改: 用户名["+userNew.Username+"],失败原因["+userNew.Username+"已存在]", nil)
	}

	// 描述信息更改
	if userNew.Description == "" {
		userNew.Description = " "
	}
	// 切换角色
	if oldUserNew.RoleId != userNew.RoleId {
		name, err := roleRepository.FindRoleNameById(userNew.RoleId)
		if err != nil {
			log.Errorf("FindRoleNameById Error: %v", err)
		}
		userNew.RoleName = name
	}
	// 切换部门
	if oldUserNew.DepartmentId != userNew.DepartmentId {
		Department, err := departmentRepository.FindById(userNew.DepartmentId)
		if err != nil {
			log.Errorf("FindRoleNameById Error: %v", err)
		}
		userNew.DepartmentName = Department.Name
	}
	//由已过期修改为可用时间内则修改状态为已禁用
	if oldUserNew.Status == constant.Expiration {
		if *userNew.IsPermanent || (time.Now().After(userNew.BeginValidTime.Time) && time.Now().Before(userNew.EndValidTime.Time)) {
			userNew.Status = constant.Disable
		}
	}
	if oldUserNew.Status == constant.Expiration {
		if (*userNew.IsPermanent || (time.Now().After(userNew.BeginValidTime.Time) && time.Now().Before(userNew.EndValidTime.Time))) && userNew.Username == "admin" {
			userNew.Status = constant.Enable
		}
	}

	// 判断邮件认证是否启用
	var TRUE = true
	var FALSE = false
	if strings.Contains(userNew.AuthenticationWay, constant.Email) {
		userNew.VerifyMailState = &TRUE
	} else {
		userNew.VerifyMailState = &FALSE
	}

	// 更新用户信息
	if err := userNewRepository.UpdateMapById(userNew, id); nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	// 发送随机密码邮件
	if userNew.IsRandomPassword && userNew.SendWay == constant.Email {
		err := sysConfigService.SendMail([]string{userNew.Mail}, "随机密码", "登录用户名: "+userNew.Username+"\n"+"随机密码: "+pwd)
		if err != nil {
			log.Errorf("SendMail Error: %v", err)
			log.Errorf("发送邮件密码信息失败, 用户名: %v", userNew.Username)
		}
	}
	if oldUserNew.DepartmentId != userNew.DepartmentId {
		UpdateUserAssetAppAppserverDep(constant.USER, constant.UPDATE, oldUserNew.DepartmentId, userNew.DepartmentId)
	}
	return SuccessWithOperate(c, "用户列表-修改: 用户姓名["+oldUserNew.Nickname+"->"+userNew.Nickname+"]", userNew)
}

// UserNewIsDisableEndpoint 启用/禁用用户
func UserNewIsDisableEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	isDisable := c.QueryParam("status")
	userNew, err := userNewRepository.FindById(id)
	if userNew.Status == constant.Expiration {
		return FailWithDataOperate(c, 500, "用户已过期", "", err)
	}
	if isDisable == constant.Disable {
		userNew.Status = constant.Disable
	} else {
		userNew.Status = constant.Enable
	}
	if err != nil {
		log.Errorf("DB Error: %v", err)
		if isDisable == constant.Disable {
			return FailWithDataOperate(c, 500, "禁用失败", "", err)
		} else {
			return FailWithDataOperate(c, 500, "启用失败", "", err)
		}
	}
	if err := userNewRepository.UpdateStructById(userNew, id); nil != err {
		log.Errorf("DB Error: %v", err)
		if isDisable == constant.Disable {
			return FailWithDataOperate(c, 500, "禁用失败", "", err)
		} else {
			return FailWithDataOperate(c, 500, "启用失败", "", err)
		}
	}
	if isDisable == constant.Disable {
		return SuccessWithOperate(c, "用户列表-禁用: [用户"+userNew.Nickname+":"+constant.Disable+"]", nil)
	} else {
		return SuccessWithOperate(c, "用户列表-启用: [用户"+userNew.Nickname+":"+constant.Enable+"]", nil)
	}
}

// UserNewDeleteEndpoint 删除用户
func UserNewDeleteEndpoint(c echo.Context) error {
	id := c.Param("id")
	split := strings.Split(id, ",")
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, -1, "获取用户信息失效,请重新登录")
	}
	var deleteName string
	for i := range split {
		item, err := userNewRepository.FindById(split[i])
		if err != nil {
			log.Errorf("DB Error: %v", err)
			continue
		}
		if item.Username == account.Username {
			log.Errorf("DB Error: %v", "not delete self")
			return FailWithDataOperate(c, 500, "禁止删除当前登录用户", "", err)
		}
		if item.Username == "admin" {
			log.Errorf("DB Error: %v", "not delete admin")
			return FailWithDataOperate(c, 500, "禁止删除系统管理员", "", err)
		}
		deleteName += item.Nickname + ","
	}
	for i := range split {
		if err := DeleteUserById(split[i]); err != nil {
			log.Errorf("DB Error: %v", err)
		}
	}
	if len(deleteName) > 0 {
		deleteName = deleteName[:len(deleteName)-1]
	}
	return SuccessWithOperate(c, "用户列表-删除: 删除用户["+deleteName+"]", nil)
}

// UserNewPagingEndpoint 用户列表
func UserNewPagingEndpoint(c echo.Context) error {
	pageIndex, _ := strconv.Atoi(c.QueryParam("pageIndex"))
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))
	auto := c.QueryParam("auto")
	username := c.QueryParam("username")
	nickname := c.QueryParam("nickname")
	department := c.QueryParam("department")
	role := c.QueryParam("role")
	status := c.QueryParam("status")
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 401, "您的登录信息已失效,请重新登录后再试.")
	}
	var departmentId []int64
	if err := GetChildDepIds(account.DepartmentId, &departmentId); err != nil {
		log.Errorf("GetChildDepIds Error: %v", err)
	}
	userList, total, err := userNewRepository.FindByLimitingConditions(pageIndex, pageSize, auto, username, nickname, department, role, status, departmentId)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	for i := range userList {
		// 根据部门机构id获取部门机构名称
		dep, err := departmentRepository.FindById(userList[i].DepartmentId)
		if err != nil {
			log.Errorf("DB Error: %v", err)
		}
		userList[i].DepartmentName = dep.Name
		// 更新用户的部门机构名称
		if err := userNewRepository.UpdateStructById(model.UserNew{DepartmentName: dep.Name}, userList[i].ID); nil != err {
			log.Errorf("DB Error: %v", err)
		}
		if !userList[i].IsPermanent && time.Now().After(userList[i].EndValidTime.Time) {
			userList[i].Status = constant.Expiration
			if err := userNewRepository.UpdateStructById(model.UserNew{Status: userList[i].Status}, userList[i].ID); nil != err {
				log.Errorf("DB Error: %v", err)
			}
		}
		if userList[i].IsPermanent {
			userList[i].BeginValidTime = utils.NewJsonTime(time.Time{})
			userList[i].EndValidTime = utils.NewJsonTime(time.Time{})
		}
		if strings.Contains(userList[i].AuthenticationWay, constant.LDAPOrAD) {
			userList[i].AuthenticationServer, err = ldapAdAuthRepository.FindLdapAdServerAddressById(userList[i].AuthenticationServerId)
			if err != nil {
				log.Errorf("Update AuthenticationServer Error: %v", err)
			}
		}
	}
	return Success(c, H{
		"total": total,
		"items": userList,
	})
}

// UserEditBatchUserEndpoint  批量编辑用户
func UserEditBatchUserEndpoint(c echo.Context) error {
	id := c.Param("id")
	var userForEditBatch dto.UserForEditBatch
	if err := c.Bind(&userForEditBatch); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "批量编辑失败", "", err)
	}
	// 校验密码复杂度
	if userForEditBatch.Password != "" {
		if userForEditBatch.Password != userForEditBatch.RePassword {
			return Fail(c, 500, "两次密码输入不一致")
		}
		result, code, msg, err := CheckPasswordComplexity(userForEditBatch.Password)
		if !result {
			return FailWithDataOperate(c, code, msg, "", err)
		}
		// 密码加密存储
		pwd := userForEditBatch.Password
		pass, err := utils.Encoder.Encode([]byte(pwd))
		if err != nil {
			log.Errorf("Encode Error: %v", err)
			return FailWithDataOperate(c, 500, "新增失败", "", err)
		}
		userForEditBatch.Password = string(pass)
	}
	var departmentId int64
	var err error
	if userForEditBatch.DepartmentId != "" {
		departmentId, err = strconv.ParseInt(userForEditBatch.DepartmentId, 10, 64)
		if err != nil {
			log.Errorf("ParseInt Error: %v", err)
		}
	}
	// 通过部门id查部门名称
	department, err := departmentRepository.FindById(departmentId)
	if err != nil {
		log.Errorf("DB Error: %v", err)
	}

	// 将edit转化为userNew
	var user model.UserNew
	if userForEditBatch.RoleId != "" {
		// 通过角色id查角色名称
		role, err := roleRepository.FindById(userForEditBatch.RoleId)
		if err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "新增失败", "", err)
		}
		user.RoleId = userForEditBatch.RoleId
		user.RoleName = role.Name
	}

	if userForEditBatch.Password != "" {
		user.PasswordUpdated = utils.NowJsonTime()
		user.Password = userForEditBatch.Password
		user.VerifyPassword = userForEditBatch.Password
	}
	var name string
	split := strings.Split(id, ",")
	for i := range split {
		userInfo, err := userNewRepository.FindById(split[i])
		if err != nil {
			log.Errorf("FindById Error: %v", err)
			continue
		}
		if userInfo.Username == "admin" {
			log.Errorf("admin can not be edited")
			continue
		}
		if userForEditBatch.Password != "" {
			user.SamePwdJudge = utils.StrJoin(userInfo.SamePwdJudge, user.Password)
			if ok := CheckPasswordRepeatTimes(userInfo.SamePwdJudge, user.Password); !ok {
				log.Errorf("CheckPasswordRepeatTimes Error: %v", userInfo.Username+": 新密码不可与最近使用密码相同")
				continue
			}
		}
		if userForEditBatch.IsPermanent == false && (userForEditBatch.BeginValidTime.IsZero() || userForEditBatch.EndValidTime.IsZero()) {
			user.IsPermanent = userInfo.IsPermanent
			user.BeginValidTime = userInfo.BeginValidTime
			user.EndValidTime = userInfo.EndValidTime
		} else if userForEditBatch.IsPermanent == false && !userForEditBatch.BeginValidTime.IsZero() && !userForEditBatch.EndValidTime.IsZero() {
			user.IsPermanent = &userForEditBatch.IsPermanent
			user.BeginValidTime = userForEditBatch.BeginValidTime
			user.EndValidTime = userForEditBatch.EndValidTime
		} else if userForEditBatch.IsPermanent == true {
			user.IsPermanent = &userForEditBatch.IsPermanent
			if userInfo.Status == constant.Expiration && userForEditBatch.IsPermanent == true {
				user.Status = constant.Disable
			}
		}

		if userForEditBatch.DepartmentId != "" {
			user.DepartmentId = departmentId
			user.DepartmentName = department.Name
		} else {
			user.DepartmentId = userInfo.DepartmentId
			user.DepartmentName = userInfo.DepartmentName
		}
		user.Wechat = userInfo.Wechat
		user.Mail = userInfo.Mail
		user.Phone = userInfo.Phone
		user.Description = userInfo.Description
		user.QQ = userInfo.QQ
		if err = userNewRepository.UpdateMapById(user, split[i]); err != nil {
			log.Errorf("UpdateById Error: %v", err)
			return FailWithDataOperate(c, 500, "批量编辑失败", "", err)
		}
		if userInfo.DepartmentId != departmentId {
			UpdateUserAssetAppAppserverDep(constant.USER, constant.UPDATE, userInfo.DepartmentId, departmentId)
		}
		name += userInfo.Username + ","
	}
	if len(name) > 0 {
		name = name[:len(name)-1]
	}
	return SuccessWithOperate(c, "用户列表-批量编辑: 用户名["+name+"]", nil)
}

// UserDownloadTemplateEndpoint 下载模板
func UserDownloadTemplateEndpoint(c echo.Context) error {
	userHeaderForExport := []string{"用户名(必填)", "姓名", "部门名称(不填默认为根部门)(如果部门名称重复请在名称后加{{id}} 例如:测试部{{22}})", "角色名(必填)", "密码(必填)", "邮箱", "QQ", "微信", "手机号码"}
	userFileNameForExport := "用户列表"
	file, err := utils.CreateTemplateFile(userFileNameForExport, userHeaderForExport)
	if err != nil {
		log.Errorf("CreateExcelFile Error: %v", err)
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	fileName := "用户列表导入模板.xlsx"
	var buff bytes.Buffer
	if err = file.Write(&buff); err != nil {
		log.Errorf("Write Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "", err)
	}
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
}

// UserNewImportEndpoint 导入用户
func UserNewImportEndpoint(c echo.Context) error {
	// 获取文件
	//isCover := c.FormValue("is_cover")
	file, err := c.FormFile("file")
	if err != nil {
		log.Errorf("FormFile Error: %v", err)
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}
	src, err := file.Open()
	if nil != err {
		log.Errorf("Open Error: %v", err)
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}
	defer func(src multipart.File) {
		err := src.Close()
		if nil != err {
			log.Errorf("Close Error: %v", err)
		}
	}(src)

	// 读excel流
	xlsx, err := excelize.OpenReader(src)
	if nil != err {
		log.Errorf("OpenReader Error: %v", err)
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}

	// 根据名字获取cells的内容，返回的是一个[][]string
	records, err := xlsx.GetRows(xlsx.GetSheetName(xlsx.GetActiveSheetIndex()))
	if nil != err {
		log.Errorf("GetRows Error: %v", err)
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}
	// 获取当前账户
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 500, "获取当前账户失败")
	}
	var departmentId = account.DepartmentId
	var departmentName = account.DepartmentName
	var departmentIds []int64
	err = GetChildDepIds(departmentId, &departmentIds)
	if err != nil {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}

	// 导入文件有几行, 0行文件为空, 1行代表只有标题行也是"空"
	total := len(records)
	if total <= 1 {
		return FailWithDataOperate(c, 400, "导入文件数据为空", "用户列表-导入: 文件名["+file.Filename+"], 失败原因[导入文件数据为空]", nil)
	}
	if len(records[0]) != 9 {
		return FailWithDataOperate(c, 400, "导入文件格式错误", "用户列表-导入: 文件名["+file.Filename+"], 失败原因[导入文件格式错误]", nil)
	}
	var nameSuccess, nameFiled string
	var successNum, filedNum int
	// 覆盖导入
	//if isCover == "true" {
	//	// 查找要删除的用户
	//	var userNewList []model.UserNew
	//	if err := userNewRepository.DB.Where("department_id in (?) and username != ? and username != ?", departmentIds, "admin", account.Username).Find(&userNewList).Error; nil != err {
	//		log.Errorf("DB Error: %v", err)
	//		return Fail(c, 500, "导入失败")
	//
	//	}
	//	for _, userNew := range userNewList {
	//		err := DeleteUserById(userNew.ID)
	//		if err != nil {
	//			log.Errorf("DeleteUserById Error: %v", err)
	//		}
	//	}
	//}
	for i := range records {
		if i == 0 {
			continue
		}
		if len(records[i][0]) == 0 || len(records[i][1]) == 0 || len(records[i][3]) == 0 || len(records[i][4]) == 0 {
			nameFiled += records[i][0] + "[" + "必填项为空" + "]" + ","
			filedNum++
			continue
		}
		// 用户名和昵称不能重复
		var itemExists []model.UserNew
		if err := userNewRepository.DB.Where("username = ? ", records[i][0]).Find(&itemExists).Error; nil != err {
			log.Errorf("DB Error: %v", err)
			continue
		}
		if 0 != len(itemExists) {
			nameFiled += records[i][0] + "[" + "名称重复" + "]" + ","
			filedNum++
			continue
		}

		// 密码校验
		result, _, msg, err := CheckPasswordComplexity(records[i][4])
		if !result {
			log.Errorf("DB Error: %v", err)
			nameFiled += records[i][0] + "[" + msg + "]" + ","
			filedNum++
			continue
		}

		// 密码加密存储
		pass, err := utils.Encoder.Encode([]byte(records[i][4]))
		if err != nil {
			log.Errorf(records[i][0]+"Encode Error: %v", err)
			nameFiled += records[i][0] + "[" + "密码加密失败" + "]" + ","
			filedNum++
			continue
		}

		//
		if len(records[i][2]) != 0 {
			index := strings.IndexByte(records[i][2], '{')
			if index != -1 {
				Id, err := strconv.ParseInt(records[i][2][index+1:len(records[i][2])-1], 10, 64)
				if err != nil {
					log.Errorf("ParseInt Error: %v", err)
					nameFiled += records[i][0] + "[" + "部门ID错误" + "]" + ","
					filedNum++
					continue
				}
				departmentId = Id
				departmentName = records[i][2][0:index]
			} else {
				department, err := departmentRepository.FindByName(records[i][2])
				if err != nil {
					log.Errorf("FindByNameId Error: %v", err)
					nameFiled += records[i][0] + "[" + "部门不存在" + "]" + ","
					filedNum++
					continue
				}
				departmentName = records[i][2]
				departmentId = department.ID
			}
		} else {
			departmentId = account.DepartmentId
			departmentName = account.DepartmentName
		}

		// 判断此用户部门的等级是不是符合要求
		res := IsDepIdBelongDepIds(departmentId, departmentIds)
		if !res {
			nameFiled += records[i][0] + "[" + "无权限导入此用户" + "]" + ","
			filedNum++
			continue
		}

		role, err := roleRepository.FindByRoleName(records[i][3])
		fmt.Println(role, role, role, role, role, role)
		if err != nil {
			log.Errorf("FindByRoleName Error: %v", err)
			nameFiled += records[i][0] + "[" + "角色不存在" + "]" + ","
			filedNum++
			continue
		}
		var mail, qq, wechat, phone string
		var temp = true
		if len(records[i]) == 6 {
			mail = records[i][5]
		}
		if len(records[i]) == 7 {
			qq = records[i][6]
		}
		if len(records[i]) == 8 {
			wechat = records[i][7]
		}
		if len(records[i]) == 9 {
			phone = records[i][8]
		}
		userNew := model.UserNew{
			ID:                utils.UUID(),
			Username:          records[i][0],
			Nickname:          records[i][1],
			DepartmentName:    departmentName,
			DepartmentId:      departmentId,
			RoleName:          role.Name,
			RoleId:            role.ID,
			Password:          string(pass),
			VerifyPassword:    string(pass),
			PasswordUpdated:   utils.NowJsonTime(),
			Created:           utils.NowJsonTime(),
			Mail:              mail,
			QQ:                qq,
			Wechat:            wechat,
			Phone:             phone,
			IsPermanent:       &temp,
			Status:            constant.Enable,
			AuthenticationWay: constant.StaticPassword,
			SamePwdJudge:      utils.StrJoin("", string(pass)),
		}

		if userNew.RoleName == constant.SystemAdmin && userNew.DepartmentId != 0 {
			nameFiled += records[i][0] + "[" + "系统管理员只能在根部门" + "]" + ","
			filedNum++
			continue
		}

		// 创建用户
		if err = userNewRepository.Create(&userNew); err != nil {
			log.Errorf(records[i][0]+"Create Error: %v", err)
			nameFiled += records[i][0] + "[" + "创建失败" + "]" + ","
			filedNum++
			continue
		}
		// 更新部门用户数
		UpdateUserAssetAppAppserverDep(constant.USER, constant.ADD, userNew.DepartmentId, int64(-1))
		nameSuccess += records[i][0] + ","
		successNum++
	}
	if len(nameSuccess) != 0 {
		nameSuccess = nameSuccess[0 : len(nameSuccess)-1]
	}
	if len(nameFiled) != 0 {
		nameFiled = nameFiled[0 : len(nameFiled)-1]
	}
	return SuccessWithOperate(c, "用户列表-导入: [导入成功用户:{"+nameSuccess+"},成功数:"+strconv.Itoa(successNum)+"]"+"[导入失败用户:{"+nameFiled+"},失败数:"+strconv.Itoa(filedNum)+"]", nil)
}

// UserNewExportEndpoint 导出用户
func UserNewExportEndpoint(c echo.Context) error {
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 401, "您的登录信息已失效,请重新登录后再试.")
	}
	var userForExport []dto.UserForExport
	var departmentId []int64
	err := GetChildDepIds(account.DepartmentId, &departmentId)
	if err != nil {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "", err)
	}
	// 获取数据
	userForExport, err = userNewRepository.UserExport(departmentId)
	if err != nil {
		log.Errorf("UserExport Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "", err)
	}

	userStringsForExport := make([][]string, len(userForExport))
	for i, v := range userForExport {
		v.Id = strconv.Itoa(i + 1)
		user := utils.Struct2StrArr(v)
		userStringsForExport[i] = make([]string, len(user))
		userStringsForExport[i] = user
	}
	userHeaderForExport := []string{"序号", "用户名", "姓名", "部门名称", "角色", "状态", "邮箱", "QQ", "微信", "手机号码", "描述"}
	userFileNameForExport := "用户列表"
	file, err := utils.CreateExcelFile(userFileNameForExport, userHeaderForExport, userStringsForExport)
	if err != nil {
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	fileName := "用户列表.xlsx"

	// 记录操作日志
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		Result:          "成功",
		LogTypes:        "用户日志",
		LogContents:     "用户列表-导出: 部门[" + account.DepartmentName + "下所有用户]",
		Created:         utils.NowJsonTime(),
		Users:           account.Username,
		Names:           account.Nickname,
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

// GetDetailsUserInfo   获取用户详情
func GetDetailsUserInfo(c echo.Context) error {
	id := c.Param("id")
	userNew, err := userNewRepository.FindById(id)
	if err != nil {
		log.Errorf("FindById Error: %v", err)
		return FailWithDataOperate(c, 500, "获取用户详情失败", "", err)
	}
	var validTime string
	if *userNew.IsPermanent == true {
		validTime = "永久有效"
	} else {
		if !userNew.BeginValidTime.IsZero() {
			validTime += userNew.BeginValidTime.Format("2006-01-02 15:04:05")
		}
		validTime += " ~ "
		if !userNew.EndValidTime.IsZero() {
			validTime += userNew.EndValidTime.Format("2006-01-02 15:04:05")
		}
	}
	depChinaName, err := DepChainName(userNew.DepartmentId)
	if err != nil {
		log.Errorf("DepChainName Error: %v", err)
		depChinaName = userNew.DepartmentName
	}
	depChinaName = strings.TrimRight(depChinaName, ".")
	userForDetail := dto.UserDetailForBasis{
		ID:                userNew.ID,
		Username:          userNew.Username,
		Nickname:          userNew.Nickname,
		Department:        depChinaName,
		RoleName:          userNew.RoleName,
		AuthenticationWay: userNew.AuthenticationWay,
		Status:            userNew.Status,
		Mail:              userNew.Mail,
		QQ:                userNew.QQ,
		Wechat:            userNew.Wechat,
		Phone:             userNew.Phone,
		Description:       userNew.Description,
		ValidTime:         validTime,
	}

	return SuccessWithOperate(c, "获取用户详情成功", userForDetail)
}

// GetUserGroupInfo 获取用户分组信息
func GetUserGroupInfo(c echo.Context) error {
	id := c.Param("id")
	_, err := userNewRepository.FindById(id)
	if err != nil {
		log.Errorf("FindById Error: %v", err)
		return Fail(c, 500, "用户不存在")
	}
	// 通过用户id找到与之关联的用户组
	userGroupId, err := userGroupMemberRepository.FindUserGroupIdsByUserId(id)
	if err != nil {
		log.Errorf("FindUserGroupIdsByUserId Error: %v", err)
	}
	var userDetailForUserGroup []dto.UserDetailForUserGroup
	for _, v := range userGroupId {
		userGroup, err := userGroupNewRepository.FindById(v)
		if err != nil {
			log.Errorf("FindById Error: %v", err)
			continue
		}
		depChinaName, err := DepChainName(userGroup.DepartmentId)
		if len(depChinaName) != 0 {
			if depChinaName[len(depChinaName)-1] == '.' {
				depChinaName = depChinaName[:len(depChinaName)-1]
			}
		}
		userDetailForUserGroup = append(userDetailForUserGroup, dto.UserDetailForUserGroup{
			Name:       userGroup.Name,
			Department: depChinaName,
			Create:     userGroup.Created,
		})
	}
	return Success(c, userDetailForUserGroup)
}

// GetUserStrategyInfo 获取用户策略信息
func GetUserStrategyInfo(c echo.Context) error {
	id := c.Param("id")
	_, err := userNewRepository.FindById(id)
	if err != nil {
		log.Errorf("FindById Error: %v", err)
		return Fail(c, 500, "用户不存在")
	}
	var userDetailForStrategy []dto.UserDetailForUserStrategy
	// 通过用户id找到与之关联的策略
	var strategyIds []string
	err = userStrategyRepository.DB.Table("user_strategy_users").Select("user_strategy_id").Where("user_id = ?", id).Find(&strategyIds).Error
	if err != nil {
		return SuccessWithOperate(c, "", userDetailForStrategy)
	}
	fmt.Println("strategyIds111", strategyIds)

	// 通过用户id找到与之关联的用户组
	userGroupId, err := userGroupMemberRepository.FindUserGroupIdsByUserId(id)
	if err != nil {
		log.Errorf("FindUserGroupIdsByUserId Error: %v", err)
	}
	fmt.Println("userGroupId", userGroupId)
	// 通过用户组id找到与之关联的策略
	for _, v := range userGroupId {
		strategyId, err := userStrategyRepository.FindStrategyIdByUserGroupId(v)
		fmt.Println("strategyId", strategyId)
		if err != nil {
			log.Errorf("FindStrategyIdByUserGroupId Error: %v", err)
		}
		strategyIds = append(strategyIds, strategyId...)
	}
	fmt.Println("strategyIds", strategyIds)
	// 去重
	strategyIds = utils.RemoveDuplicatesAndEmpty(strategyIds)
	fmt.Println("strategyIds", strategyIds)
	for _, v := range strategyIds {
		strategy, err := userStrategyRepository.FindById(v)
		if err != nil {
			log.Errorf("FindById Error: %v", err)
			continue
		}
		depChinaName, err := DepChainName(strategy.DepartmentId)
		if len(depChinaName) != 0 {
			if depChinaName[len(depChinaName)-1] == '.' {
				depChinaName = depChinaName[:len(depChinaName)-1]
			}
		}
		userDetailForStrategy = append(userDetailForStrategy, dto.UserDetailForUserStrategy{
			Name:       strategy.Name,
			Department: depChinaName,
			Status:     strategy.Status,
		})
	}
	return SuccessWithOperate(c, "", userDetailForStrategy)
}

// GetDeviceAssetInfo TODO  获取设备资产信息
func GetDeviceAssetInfo(c echo.Context) error {
	userId := c.Param("id")
	var deviceAssetList []dto.UserDetailForAsset
	// 通过id获取用户
	user, err := userNewRepository.FindById(userId)
	if err != nil {
		log.Errorf("FindById Error: %v", err)
		return Fail(c, 500, "用户不存在")
	}
	// 通过用户id找到与之关联的用户组id
	userGroupId, err := userGroupMemberRepository.FindUserGroupIdsByUserId(user.ID)
	if err != nil {
		log.Errorf("FindUserGroupIdsByUserId Error: %v", err)
	}
	// 通过用户/用户组查找与用户有关的运维策略
	var operationArr []model.OperateAuth
	for _, v := range userGroupId {
		operationAuth, err := operateAuthRepository.FindByRateUserGroupId(v)
		if err != nil {
			log.Errorf("FindByRateUserId Error: %v", err)
		}
		for _, v1 := range operationAuth {
			if v1.State == "on" {
				operationArr = append(operationArr, v1)
			}
		}
	}
	operationAuth, err := operateAuthRepository.FindByRateUserId(user.ID)
	if err != nil {
		log.Errorf("FindByRateUserId Error: %v", err)
	}
	for _, v := range operationAuth {
		if v.State == "on" {
			operationArr = append(operationArr, v)
		}
	}
	// 通过运维策略id找到与之关联的设备
	for _, operation := range operationArr {
		if operation.RelateAsset != "" {
			assetIds := strings.Split(operation.RelateAsset, ",")
			for _, v := range assetIds {
				asset, err := newAssetRepository.GetPassPortByID(context.TODO(), v)
				if err != nil {
					log.Errorf("FindById Error: %v", err)
					continue
				}
				// 通过asset.assetType获取资产类型
				assetType, err := systemTypeRepository.GetSystemTypeByID(context.TODO(), asset.AssetType)
				if err != nil {
					log.Errorf("FindById Error: %v", err)
					continue
				}
				depName, err := DepChainName(asset.DepartmentId)
				if err != nil {
					log.Errorf("DepChainName Error: %v", err)
					continue
				}
				index := strings.Index(asset.Name, "[")
				if index != -1 {
					asset.Name = asset.Name[:index]
				}
				depName = strings.TrimSuffix(depName, ".")
				deviceAssetList = append(deviceAssetList, dto.UserDetailForAsset{
					ID:              asset.ID,
					Name:            asset.Name,
					Address:         asset.Ip,
					SystemType:      assetType.Name,
					Protocol:        asset.Protocol,
					Passport:        asset.Passport,
					PolicyOperation: operation.Name + "[" + depName + "]",
				})
			}
		}
		if operation.RelateAssetGroup != "" {
			assetGroupIds := strings.Split(operation.RelateAssetGroup, ",")
			for _, v2 := range assetGroupIds {
				// 通过资产组id找到与之关联的资产
				assetGroupAsset, err := newAssetGroupRepository.GetAssetGroupAsset(context.TODO(), v2)
				if err != nil {
					log.Errorf("GetAssetGroupAsset Error: %v", err)
					continue
				}
				for _, v3 := range assetGroupAsset {
					asset, err := newAssetRepository.GetPassPortByID(context.TODO(), v3.AssetId)
					if err != nil {
						log.Errorf("GetPassPortByID Error: %v", err)
						continue
					}
					// 通过asset.assetType获取资产类型
					assetType, err := systemTypeRepository.GetSystemTypeByID(context.TODO(), asset.AssetType)
					if err != nil {
						log.Errorf("FindById Error: %v", err)
						continue
					}
					depName, err := DepChainName(asset.DepartmentId)
					if err != nil {
						log.Errorf("DepChainName Error: %v", err)
						continue
					}
					depName = strings.TrimSuffix(depName, ".")
					deviceAssetList = append(deviceAssetList, dto.UserDetailForAsset{
						ID:              asset.ID,
						Name:            asset.Name,
						Address:         asset.Ip,
						SystemType:      assetType.Name,
						Protocol:        asset.Protocol,
						Passport:        asset.Passport,
						PolicyOperation: operation.Name + "[" + depName + "]",
					})
				}
			}
		}
	}
	// 查询该用户的工单信息
	workOrderList, err := workOrderNewRepository.FindValidOrderByUserId(user.ID)
	if nil != err {
		log.Errorf("FindValidOrderByUserId Error: %v", err)
	}
	for _, workOrder := range workOrderList {
		if !*workOrder.IsPermanent && (time.Now().Before(workOrder.BeginTime.Time) || time.Now().After(workOrder.EndTime.Time)) {
			continue
		}
		// 2. 获取工单关联的所有设备账号
		workOrderAsset, err := workOrderNewRepository.FindByWorkOrderId(workOrder.ID)
		if nil != err {
			log.Errorf("FindByWorkOrderId Error: %v", err)
			continue
		}
		for _, v3 := range workOrderAsset {
			asset, err := newAssetRepository.GetPassPortByID(context.TODO(), v3.AssetId)
			if err != nil {
				log.Errorf("GetPassPortByID Error: %v", err)
				continue
			}
			// 通过asset.assetType获取资产类型
			assetType, err := systemTypeRepository.GetSystemTypeByID(context.TODO(), asset.AssetType)
			if err != nil {
				log.Errorf("FindById Error: %v", err)
				continue
			}
			depName, err := DepChainName(asset.DepartmentId)
			if err != nil {
				log.Errorf("DepChainName Error: %v", err)
				continue
			}
			depName = strings.TrimSuffix(depName, ".")
			deviceAssetList = append(deviceAssetList, dto.UserDetailForAsset{
				ID:              asset.ID,
				Name:            asset.Name,
				Address:         asset.Ip,
				SystemType:      assetType.Name,
				Protocol:        asset.Protocol,
				Passport:        asset.Passport,
				PolicyOperation: constant.AccessWorkOrder + "[" + depName + "]",
			})
		}
	}
	// 去重 deviceAssetList
	assetMap := make(map[string]int, 0)
	var deviceAssetList2 []dto.UserDetailForAsset
	for _, v := range deviceAssetList {
		if _, ok := assetMap[v.ID]; !ok {
			assetMap[v.ID] = 1
			deviceAssetList2 = append(deviceAssetList2, v)
		}
	}
	return Success(c, deviceAssetList2)
}

// GetAppAssetInfo todo  获取应用资产信息
func GetAppAssetInfo(c echo.Context) error {
	userId := c.Param("id")
	var appList []dto.UserDetailForApp
	// 通过id获取用户
	user, err := userNewRepository.FindById(userId)
	if err != nil {
		log.Errorf("FindById Error: %v", err)
		return Fail(c, 500, "用户不存在")
	}
	// 通过用户id找到与之关联的用户组
	userGroupId, err := userGroupMemberRepository.FindUserGroupIdsByUserId(user.ID)
	if err != nil {
		log.Errorf("FindUserGroupIdsByUserId Error: %v", err)
	}
	// 通过用户/用户组查找与用户有关的运维策略
	var operationArr []model.OperateAuth
	for _, v := range userGroupId {
		operationAuth, err := operateAuthRepository.FindByRateUserGroupId(v)
		if err != nil {
			log.Errorf("FindByRateUserId Error: %v", err)
		}
		operationArr = append(operationArr, operationAuth...)
	}
	operationAuth, err := operateAuthRepository.FindByRateUserId(user.ID)
	if err != nil {
		log.Errorf("FindByRateUserId Error: %v", err)
	}
	operationArr = append(operationArr, operationAuth...)
	// 通过运维策略id找到与之关联的应用
	for _, operation := range operationArr {
		if operation.RelateApp != "" {
			appIds := strings.Split(operation.RelateApp, ",")
			for _, v := range appIds {
				app, err := newApplicationRepository.FindById(context.TODO(), v)
				if err != nil {
					log.Errorf("GetAppByID Error: %v", err)
					continue
				}
				depName, err := DepChainName(app.DepartmentID)
				if err != nil {
					log.Errorf("DepChainName Error: %v", err)
					continue
				}
				depName = strings.TrimSuffix(depName, ".")
				appList = append(appList, dto.UserDetailForApp{
					ID:        app.ID,
					Name:      app.Name,
					Program:   app.ProgramName,
					Username:  "",
					Server:    app.AppSerName,
					PolicyApp: operation.Name + "[" + depName + "]",
				})
			}
		}
	}
	// 去重 appList
	appMap := make(map[string]int, 0)
	var appList2 []dto.UserDetailForApp
	for _, v := range appList {
		if _, ok := appMap[v.ID]; !ok {
			appMap[v.ID] = 1
			appList2 = append(appList2, v)
		}
	}
	return Success(c, appList2)
}

// GetCommandStrategyInfo 获取指令策略信息
func GetCommandStrategyInfo(c echo.Context) error {
	id := c.Param("id")
	var commandDetailForCommandStrategy []dto.UserDetailForCommandStrategy
	user, err := userNewRepository.FindById(id)
	if err != nil {
		log.Errorf("FindById Error: %v", err)
		return Fail(c, 500, "用户不存在")
	}
	// 通过用户id找到与之关联的用户组
	userGroupId, err := userGroupMemberRepository.FindUserGroupIdsByUserId(user.ID)
	if err != nil {
		log.Errorf("FindUserGroupIdsByUserId Error: %v", err)
	}
	userGroupId = append(userGroupId, user.ID)
	// 通过用户/用户组查找与用户有关的指令策略
	var strategyIds []string
	strategyIds, err = commandRelevanceRepository.FindByUserIdOrUserGroupId(userGroupId)
	if err != nil {
		log.Errorf("FindByRateUserId Error: %v", err)
	}
	//strategyIds 去重
	strategyIds = utils.RemoveDuplicatesAndEmpty(strategyIds)
	// 根据指令策略id找到与关联模型
	var commandRelevance []model.CommandRelevance
	for _, v := range strategyIds {
		commandRate, err := commandRelevanceRepository.FindByCommandStrategyId(v)
		if err != nil {
			log.Errorf("FindByCommandStrategyId Error: %v", err)
			continue
		}
		commandRelevance = append(commandRelevance, commandRate...)
	}
	for _, v := range commandRelevance {
		if v.AssetId != "-" {
			// 找到这条策略
			commandStrategy, err := commandStrategyRepository.FindById(v.CommandStrategyId)
			if err != nil {
				log.Errorf("FindById Error: %v", err)
				continue
			}
			asset, err := newAssetRepository.GetPassPortByID(context.TODO(), v.AssetId)
			if err != nil {
				log.Errorf("FindById Error: %v", err)
				continue
			}
			depName, err := DepChainName(asset.DepartmentId)
			if err != nil {
				log.Errorf("DepChainName Error: %v", err)
				continue
			}
			depName = strings.TrimSuffix(depName, ".")
			commandDetailForCommandStrategy = append(commandDetailForCommandStrategy, dto.UserDetailForCommandStrategy{
				Name:     asset.Name,
				Address:  asset.Ip,
				Protocol: asset.Protocol,
				Passport: asset.Passport,
				Action:   commandStrategy.Action,
				Strategy: commandStrategy.Name + "[" + depName + "]",
			})
		}
		if v.AssetGroupId != "-" {
			// 找到这条策略
			commandStrategy, err := commandStrategyRepository.FindById(v.CommandStrategyId)
			if err != nil {
				log.Errorf("FindById Error: %v", err)
				continue
			}
			assetGroupAsset, err := newAssetGroupRepository.GetAssetGroupAsset(context.TODO(), v.AssetGroupId)
			if err != nil {
				log.Errorf("GetAssetGroupAsset Error: %v", err)
				continue
			}
			for _, v3 := range assetGroupAsset {
				asset, err := newAssetRepository.GetPassPortByID(context.TODO(), v3.AssetId)
				if err != nil {
					log.Errorf("GetPassPortByID Error: %v", err)
					continue
				}
				depName, err := DepChainName(asset.DepartmentId)
				if err != nil {
					log.Errorf("DepChainName Error: %v", err)
					continue
				}
				depName = strings.TrimSuffix(depName, ".")
				commandDetailForCommandStrategy = append(commandDetailForCommandStrategy, dto.UserDetailForCommandStrategy{
					Name:     asset.Name,
					Address:  asset.Ip,
					Protocol: asset.Protocol,
					Passport: asset.Passport,
					Action:   commandStrategy.Action,
					Strategy: commandStrategy.Name + "[" + depName + "]",
				})
			}
		}
	}
	return SuccessWithOperate(c, "", commandDetailForCommandStrategy)
}

// RolePagingForUserCreatEndpoint 展示角色信息
func RolePagingForUserCreatEndpoint(c echo.Context) error {
	roleForUserCreate, err := roleRepository.FindForUserCreate()
	if err != nil {
		log.Errorf("DB Error %v", err)
	}
	for i, v := range roleForUserCreate {
		if v.Name == "运维用户" {
			roleForUserCreate[0], roleForUserCreate[i] = roleForUserCreate[i], roleForUserCreate[0]
		}
	}
	return Success(c, H{
		"total": len(roleForUserCreate),
		"items": roleForUserCreate,
	})
}

// ZipFiles 打包文件到zip
func ZipFiles(zipName string, files []*os.File) error {
	newFile, err := os.Create(zipName)
	if err != nil {
		return err
	}
	defer func(newFile *os.File) {
		err := newFile.Close()
		if err != nil {
			log.Errorf("Close Error: %v", err)
		}
	}(newFile)

	writer := zip.NewWriter(newFile)
	for _, file := range files {
		err := compress(file, writer)
		if err != nil {
			return err
		}
	}
	return nil
}

// 递归打包文件
func compress(file *os.File, zw *zip.Writer) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		fileInfos, err := file.Readdir(-1)
		if err != nil {
			return err
		}
		for _, fi := range fileInfos {
			f, err := os.Open(file.Name() + "/" + fi.Name())
			if err != nil {
				return err
			}
			err = compress(f, zw)
			if err != nil {
				return err
			}
		}
	} else {
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err = io.Copy(writer, file); err != nil {
			return err
		}
	}
	return nil
}

// GetCurrentDepartmentUserChildren 获取某部门下所有用户，参数传当前的部门id
func GetCurrentDepartmentUserChildren(departmentId int64) (userChild []model.UserNew, err error) {
	var departmentIds []int64
	if err := GetChildDepIds(departmentId, &departmentIds); err != nil {
		log.Errorf("GetChildDepIds Error: %v", err)
	}
	if userChild, err = userNewRepository.FindUserByDepartmentId(departmentIds); err != nil {
		return nil, err
	}
	return userChild, nil
}

// DeleteUserChildByDepartmentIds 删除部门下所有用户，参数传部门id
func DeleteUserChildByDepartmentIds(departmentId []int64) error {
	for _, v := range departmentId {
		userChild, err := GetCurrentDepartmentUserChildren(v)
		if err != nil {
			log.Errorf("GetCurrentDepartmentUserChildren Error :%v", err)
		}
		for i := range userChild {
			if err := DeleteUserById(userChild[i].ID); err != nil {
				log.Errorf("DeleteUserById Error :%v", err)
			}
		}
	}
	return nil
}

// DeleteUserByAuthServerId 通过认证服务器Id删除用户
func DeleteUserByAuthServerId(authServerId int64) error {
	// 通过认证服务器Id获取用户
	users, err := userNewRepository.FindByAuthServerId(authServerId)
	if err != nil {
		log.Errorf("FindByAuthServerId Error: %v", err)
		return err
	}
	for _, user := range users {
		// 删除用户
		if user.Username == "admin" {
			continue
		}
		err = DeleteUserById(user.ID)
		if err != nil {
			log.Errorf("DeleteUserById Error: %v", err)
			return err
		}
	}
	return nil
}

// DeleteRadiusUsers 删除radius用户
func DeleteRadiusUsers() error {
	// 获取所有radius用户
	radiusUsers, err := userNewRepository.FindByAuthType(constant.RADIUS)
	if err != nil {
		log.Errorf("Find Radius Users Error: %v", err)
		return err
	}
	for _, radiusUser := range radiusUsers {
		// 删除radius用户
		if radiusUser.Username == "admin" {
			continue
		}
		err = DeleteUserById(radiusUser.ID)
		if err != nil {
			log.Errorf("DeleteById Error: %v", err)
			return err
		}
	}
	return nil
}

// DeleteMailUsers 删除邮件用户
func DeleteMailUsers() error {
	// 获取所有邮件认证用户
	mailUsers, err := userNewRepository.FindByAuthType(constant.Email)
	if err != nil {
		log.Errorf("Find Mail User Error: %v", err)
		return err
	}
	for _, radiusUser := range mailUsers {
		// 删除radius用户
		if radiusUser.Username == "admin" {
			continue
		}
		err = DeleteUserById(radiusUser.ID)
		if err != nil {
			log.Errorf("DeleteUserById Error: %v", err)
			return err
		}
		// 注意保留此函数
		DelUserCollecteAssetAccount("careful")
		return nil
	}
	return nil
}

// DeleteUserById 通过用户id删除用户
func DeleteUserById(userId string) error {
	user, err := userNewRepository.FindById(userId)
	if err != nil {
		return err
	}
	if user.Username == "admin" {
		return fmt.Errorf("admin用户不可删除")
	}
	// 删除用户
	err = userNewRepository.DeleteById(userId)
	if err != nil {
		return err
	}
	// 删除用户组关联
	if err := userGroupMemberRepository.DeleteByUserId(userId); err != nil {
		log.Errorf("DB Error: %v", err)
	}
	// 删除用户策略关联
	if err := userStrategyRepository.DB.Table("user_strategy_users").Where("user_id = ?", userId).Delete(&model.UserStrategyUsers{}).Error; nil != err {
		log.Errorf("DB Error: %v", err)
	}

	// 更新授权策略的关联的用户
	operateAuth, err := operateAuthRepository.FindByRateUserId(userId)
	if err != nil {
		log.Errorf("FindByRateUserId Error: %v", err)
	}
	for _, v := range operateAuth {
		// 处理用户id
		userIds := strings.Split(v.RelateUser, ",")
		relateUser := ""
		for _, v2 := range userIds {
			if v2 != userId {
				relateUser += v2
			}
		}
		// 更新授权策略的关联的用户
		if err := operateAuthRepository.UpdateById(v.ID, &model.OperateAuth{RelateUser: relateUser}); err != nil {
			log.Errorf("UpdateById Error: %v", err)
		}
	}

	// 删除指令策略关联
	if err := commandRelevanceRepository.DeleteByUserId(userId); err != nil {
		log.Errorf("DB Error: %v", err)
	}

	// 删除该用户的工单
	if err := DeleteWorkOrderByApplyId(userId); err != nil {
		log.Errorf("DeleteWorkOrderByApplyId Error: %v", err)
	}

	// 删除资产权限报表
	if err := assetAuthReportFormRepository.DeleteByUserId(userId); err != nil {
		log.Errorf("DeleteByUserId Error: %v", err)
	}

	// 更新部门用户数
	UpdateUserAssetAppAppserverDep(constant.USER, constant.DELETE, user.DepartmentId, int64(-1))
	// 删除用户收藏
	DelUserCollecteAssetAccount(userId)
	return nil
}
