package api

import (
	"context"
	"encoding/base64"
	"strings"
	"tkbastion/pkg/constant"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/pkg/validator"
	"tkbastion/server/dto"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// GetApplicationServerList 获取应用服务器列表
func GetApplicationServerList(c echo.Context) error {
	var req dto.ApplicationServerForSearch
	req.IP = c.QueryParam("ip")
	req.Name = c.QueryParam("name")
	req.Department = c.QueryParam("department")
	req.Auto = c.QueryParam("auto")

	u, _ := GetCurrentAccountNew(c)
	var departmentIds []int64
	err := GetChildDepIds(u.DepartmentId, &departmentIds)
	if err != nil {
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	req.Departments = departmentIds

	resp, err := newApplicationServerRepository.Find(context.TODO(), &req)
	if err != nil {
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	return c.JSON(200, H{
		"code":    1,
		"success": true,
		"message": "成功",
		"data":    resp,
	})
}

// AddApplicationServer 新增应用服务器
func AddApplicationServer(c echo.Context) error {
	u, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请重新登陆")
	}
	var req dto.ApplicationServerForInsert
	if err := c.Bind(&req); err != nil {
		return FailWithDataOperate(c, 500, "新增失败", "", nil)
	}
	if err := c.Validate(&req); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetErrLogMsg(err, errs.AssetCreateLog)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}
	encryptedCBC, err := utils.AesEncryptCBC([]byte(req.Password), global.Config.EncryptionPassword)
	if err != nil {
		return err
	}
	req.Password = base64.StdEncoding.EncodeToString(encryptedCBC)
	_, err = newApplicationServerRepository.FindByName(context.TODO(), req.Name)
	if err == nil {
		return FailWithDataOperate(c, 500, "服务器地址已存在", "应用管理-新增: 新增应用服务器, 失败原因[服务器名称"+req.Name+"已存在]", nil)
	} else if err != gorm.ErrRecordNotFound {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "新增失败", "应用管理-新增: 新增应用服务器, 失败原因["+err.Error()+"]", err)
	}
	_, err = newApplicationServerRepository.FindByIp(context.TODO(), req.IP)
	if err == nil {
		return FailWithDataOperate(c, 500, "服务器地址已存在", "应用管理-新增: 新增应用服务器, 失败原因[服务器地址"+req.IP+"已存在]", nil)
	} else if err != gorm.ErrRecordNotFound {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "新增失败", "应用管理-新增: 新增应用服务器, 失败原因["+err.Error()+"]", err)
	}

	req.Department = u.DepartmentName
	req.DepartmentID = u.DepartmentId

	err = newApplicationServerRepository.Insert(context.TODO(), &req)
	if err != nil {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "新增失败", "应用管理-新增: 新增应用服务器, 失败原因["+err.Error()+"]", err)
	}
	UpdateUserAssetAppAppserverDep(constant.APPSERVER, constant.ADD, req.DepartmentID, int64(-1))
	return SuccessWithOperate(c, "应用管理-新增: 新增应用服务器, 名称["+req.Name+"], IP["+req.IP+"]", nil)
}

// UpdateApplicationServer 修改应用服务器
func UpdateApplicationServer(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "未登录")
	}

	id := c.Param("id")
	var req dto.ApplicationServerForUpdate
	if err := c.Bind(&req); err != nil {
		return FailWithDataOperate(c, 500, "修改失败", "", nil)
	}
	req.ID = id
	if err := c.Validate(&req); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetErrLogMsg(err, errs.AssetCreateLog)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}
	appSer, err := newApplicationServerRepository.FindById(context.TODO(), req.ID)
	if err != nil {
		return FailWithDataOperate(c, 500, "修改失败", "应用管理-修改: 修改应用服务器, 失败原因[服务器未找到]", err)
	}
	_, err = newApplicationServerRepository.FindByIdName(context.TODO(), req.Name, req.ID)
	if err == nil {
		return FailWithDataOperate(c, 500, "服务器名称已存在", "应用管理-修改: 修改应用服务器, 失败原因[服务器名称"+req.Name+"已存在]", nil)
	} else if err != gorm.ErrRecordNotFound {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "修改失败", "应用管理-修改: 修改应用服务器, 失败原因["+err.Error()+"]", err)
	}
	_, err = newApplicationServerRepository.FindByIpId(context.TODO(), req.IP, req.ID)
	if err == nil {
		return FailWithDataOperate(c, 500, "服务器地址已存在", "应用管理-修改: 修改应用服务器, 失败原因[服务器地址"+req.IP+"已存在]", nil)
	} else if err != gorm.ErrRecordNotFound {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "修改失败", "应用管理-修改: 修改应用服务器, 失败原因["+err.Error()+"]", err)
	}

	department, err := departmentRepository.FindById(req.DepartmentID)
	if err != nil {
		return FailWithDataOperate(c, 500, "修改失败", "应用管理-修改: 修改应用服务器, 失败原因[部门未找到]", err)
	}
	req.Department = department.Name

	err = newApplicationServerRepository.Update(context.TODO(), &req)
	if err != nil {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "修改失败", "应用管理-修改: 修改应用服务器, 失败原因["+err.Error()+"]", err)
	}

	if appSer.DepartmentID != req.DepartmentID {
		UpdateUserAssetAppAppserverDep(constant.APPSERVER, constant.UPDATE, appSer.DepartmentID, req.DepartmentID)
	}

	return SuccessWithOperate(c, "应用管理-修改: 修改应用服务器, 名称["+req.Name+"->"+appSer.Name+"], IP["+req.IP+"->"+appSer.IP+"]", nil)
}

// DeleteApplicationServer 删除应用服务器
func DeleteApplicationServer(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "未登录")
	}

	id := c.Param("id")

	appSer, err := newApplicationServerRepository.FindById(context.TODO(), id)
	if err != nil {
		return FailWithDataOperate(c, 500, "删除失败", "应用管理-删除: 删除应用服务器, 失败原因[服务器未找到]", err)
	}

	err = newApplicationServerRepository.Delete(context.TODO(), id)
	if err != nil {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "删除失败", "应用管理-删除: 删除应用服务器, 失败原因["+err.Error()+"]", err)
	}

	UpdateUserAssetAppAppserverDep(constant.APPSERVER, constant.DELETE, appSer.DepartmentID, int64(-1))

	return SuccessWithOperate(c, "应用管理-删除: 删除应用服务器, 名称["+appSer.Name+"], IP["+appSer.IP+"]", nil)
}

// DeleteApplicationServers 批量删除应用服务器
func DeleteApplicationServers(c echo.Context) error {
	id := c.QueryParam("id")
	ids := utils.IdHandle(id)
	var name string
	for _, v := range ids {
		applicationServer, err := newApplicationServerRepository.FindById(context.TODO(), v)
		if err != nil {
			log.Errorf("DB error: %s", err)
			return FailWithDataOperate(c, 500, "删除失败", "应用管理-删除: 删除应用服务器, 失败原因["+err.Error()+"]", err)
		}
		name = name + applicationServer.Name + ","
		UpdateUserAssetAppAppserverDep(constant.APPSERVER, constant.DELETE, applicationServer.DepartmentID, int64(-1))
	}
	err := newApplicationServerRepository.DeleteMultiApplicationServer(context.TODO(), ids)
	if err != nil {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "删除失败", "应用管理-删除: 批量删除应用服务器, 失败原因["+err.Error()+"]", err)
	}
	return SuccessWithOperate(c, "应用管理-删除: 批量删除应用服务器, 名称["+name+"]", nil)
}

// AddProgram 添加程序
func AddProgram(c echo.Context) error {
	var req dto.NewProgramForInsert
	if err := c.Bind(&req); err != nil {
		return FailWithDataOperate(c, 500, "添加失败", "", nil)
	}
	if err := c.Validate(&req); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetErrLogMsg(err, errs.AssetCreateLog)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}
	_, err := newApplicationServerRepository.FindProgramByName(context.TODO(), req.Aid, req.Name)
	if err == nil {
		return FailWithDataOperate(c, 500, "程序名称已存在", "应用管理-添加: 添加程序, 失败原因[程序名称"+req.Name+"已存在]", nil)
	} else if err != gorm.ErrRecordNotFound {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "添加失败", "应用管理-添加: 添加程序, 失败原因["+err.Error()+"]", err)
	}

	err = newApplicationServerRepository.InsertProgram(context.TODO(), &req)
	if err != nil {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "添加失败", "应用管理-添加: 添加程序, 失败原因["+err.Error()+"]", err)
	}
	return SuccessWithOperate(c, "应用管理-添加: 添加程序, 名称["+req.Name+"], 路径["+req.Path+"]", nil)
}

// UpdateProgram 修改程序
func UpdateProgram(c echo.Context) error {
	var req dto.NewProgramForUpdate
	if err := c.Bind(&req); err != nil {
		return FailWithDataOperate(c, 500, "修改失败", "", nil)
	}
	program, err := newApplicationServerRepository.FindProgramById(context.TODO(), req.ID)
	if err != nil {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "修改失败", "应用管理-修改: 修改程序, 失败原因[应用程序不存在]", err)
	}
	if err := c.Validate(&req); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetErrLogMsg(err, errs.AssetCreateLog)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}
	err = newApplicationServerRepository.UpdateProgram(context.TODO(), &req)
	if err != nil {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "修改失败", "应用管理-修改: 修改程序, 失败原因["+err.Error()+"]", err)
	}
	return SuccessWithOperate(c, "应用管理-修改: 修改程序, 名称["+program.Name+"->"+req.Name+"], 路径["+program.Path+"->"+req.Path+"]", nil)
}

// DeleteProgram 删除程序
func DeleteProgram(c echo.Context) error {
	id := c.Param("id")
	program, err := newApplicationServerRepository.FindProgramById(context.TODO(), id)
	if err != nil {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "删除失败", "应用管理-删除: 删除程序, 失败原因["+err.Error()+"]", err)
	}
	err = newApplicationServerRepository.DeleteProgram(context.TODO(), id)
	if err != nil {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "删除失败", "应用管理-删除: 删除程序, 失败原因["+err.Error()+"]", err)
	}
	return SuccessWithOperate(c, "应用管理-删除: 删除程序, ID["+program.Name+"]", nil)
}

// DeleteMultiProgram 批量删除程序
func DeleteMultiProgram(c echo.Context) error {
	id := c.QueryParam("id")
	ids := strings.Split(id, ",")
	var name string
	for _, id := range ids {
		program, err := newApplicationServerRepository.FindProgramById(context.TODO(), id)
		if err != nil {
			log.Errorf("DB error: %s", err)
			return FailWithDataOperate(c, 500, "删除失败", "应用管理-删除: 删除程序, 失败原因["+err.Error()+"]", err)
		}
		name = name + program.Name + ","
	}
	err := newApplicationServerRepository.DeleteMoreProgram(context.TODO(), ids)
	if err != nil {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "删除失败", "应用管理-删除: 批量删除程序, 失败原因["+err.Error()+"]", err)
	}
	return SuccessWithOperate(c, "应用管理-删除: 批量删除程序, 名称["+name+"]", nil)
}

func GetApplicationServer(c echo.Context) error {
	id := c.Param("id")
	applicationServer, err := newApplicationServerRepository.SearchProgramByAid(context.TODO(), id)
	if err != nil {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "获取失败", "应用管理-获取: 获取应用服务器, 失败原因["+err.Error()+"]", err)
	}
	return SuccessWithOperate(c, "", applicationServer)
}
