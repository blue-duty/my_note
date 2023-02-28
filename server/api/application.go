package api

import (
	"bytes"
	"context"
	"net/http"
	"strconv"
	"strings"
	"tkbastion/pkg/constant"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/pkg/validator"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// GetApplicationList 查询应用列表
func GetApplicationList(c echo.Context) error {
	var req dto.ApplicationForSearch
	req.Name = c.QueryParam("name")
	req.Department = c.QueryParam("department")
	req.Auto = c.QueryParam("auto")
	req.AppSerName = c.QueryParam("appSerName")
	req.ProgramName = c.QueryParam("programName")
	req.PageSize, _ = strconv.Atoi(c.QueryParam("pageSize"))
	req.PageIndex, _ = strconv.Atoi(c.QueryParam("pageIndex"))

	u, _ := GetCurrentAccountNew(c)

	var departmentIds []int64
	err := GetChildDepIds(u.DepartmentId, &departmentIds)
	req.Departments = departmentIds
	resp, err := newApplicationRepository.Find(context.TODO(), &req)
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

// AddApplication 新增应用
func AddApplication(c echo.Context) error {
	var req dto.ApplicationForInsert
	if err := c.Bind(&req); err != nil {
		return FailWithDataOperate(c, 500, "新增失败", "", nil)
	}
	if err := c.Validate(&req); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetErrLogMsg(err, errs.AssetCreateLog)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}
	_, err := newApplicationRepository.FindByName(context.TODO(), req.Name)
	if err == nil {
		return FailWithDataOperate(c, 500, "应用名称已存在", "应用管理-新增: 新增应用, 失败原因[应用名称"+req.Name+"已存在]", nil)
	} else if err != gorm.ErrRecordNotFound {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "新增失败", "应用管理-新增: 新增应用, 失败原因["+err.Error()+"]", err)
	}

	err = newApplicationRepository.Insert(context.TODO(), &req)
	if err != nil {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "新增失败", "应用管理-新增: 新增应用, 失败原因["+err.Error()+"]", err)
	}

	UpdateUserAssetAppAppserverDep(constant.APP, constant.ADD, req.DepartmentID, int64(-1))

	return SuccessWithOperate(c, "应用管理-新增: 新增应用, 名称["+req.Name+"]", nil)
}

// UpdateApplication 更新应用
func UpdateApplication(c echo.Context) error {
	var req dto.ApplicationForUpdate
	if err := c.Bind(&req); err != nil {
		return FailWithDataOperate(c, 500, "更新失败", "", nil)
	}
	if err := c.Validate(&req); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetErrLogMsg(err, errs.AssetCreateLog)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	app, err := newApplicationRepository.FindById(context.TODO(), req.ID)
	if err != nil {
		return FailWithDataOperate(c, 500, "应用不存在", "应用管理-更新: 更新应用, 失败原因[应用不存在]", nil)
	}

	if _, err := newApplicationRepository.FindByNameId(context.TODO(), req.Name, req.ID); err == nil {
		return FailWithDataOperate(c, 500, "应用名称已存在", "应用管理-更新: 更新应用, 失败原因[应用名称"+req.Name+"已存在]", nil)
	} else if err != gorm.ErrRecordNotFound {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "更新失败", "应用管理-更新: 更新应用, 失败原因["+err.Error()+"]", err)
	}

	err = newApplicationRepository.Update(context.TODO(), &req)
	if err != nil {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "更新失败", "应用管理-更新: 更新应用, 失败原因["+err.Error()+"]", err)
	}

	UpdateUserAssetAppAppserverDep(constant.APP, constant.UPDATE, app.DepartmentID, req.DepartmentID)

	return SuccessWithOperate(c, "应用管理-更新: 更新应用, 名称["+app.Name+"->"+req.Name+"]", nil)
}

// DeleteApplication 删除应用
func DeleteApplication(c echo.Context) error {
	id := c.Param("id")
	a, err := newApplicationRepository.FindById(context.TODO(), id)
	if err != nil {
		return FailWithDataOperate(c, 500, "删除失败", "应用管理-删除: 删除应用, 失败原因[应用不存在]", nil)
	}
	err = newApplicationRepository.Delete(context.TODO(), id)
	if err != nil {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "删除失败", "应用管理-删除: 删除应用, 失败原因["+err.Error()+"]", err)
	}

	UpdateUserAssetAppAppserverDep(constant.APP, constant.DELETE, a.DepartmentID, int64(-1))

	return SuccessWithOperate(c, "应用管理-删除: 删除应用, 名称["+a.Name+"]", nil)
}

// DeleteApplications 批量删除应用
func DeleteApplications(c echo.Context) error {
	id := c.QueryParam("id")
	ids := strings.Split(id, ",")
	var names string
	for _, id := range ids {
		a, err := newApplicationRepository.FindById(context.TODO(), id)
		if err != nil {
			return FailWithDataOperate(c, 500, "删除失败", "应用管理-批量删除: 删除应用, 失败原因[应用不存在]", nil)
		}
		names += a.Name + ","
		UpdateUserAssetAppAppserverDep(constant.APP, constant.DELETE, a.DepartmentID, int64(-1))
	}
	err := newApplicationRepository.DeleteMore(context.TODO(), ids)
	if err != nil {
		log.Errorf("DB error: %s", err)
		return FailWithDataOperate(c, 500, "删除失败", "应用管理-批量删除: 批量删除应用, 失败原因["+err.Error()+"]", err)
	}
	return SuccessWithOperate(c, "应用管理-批量删除: 批量删除应用, 名称["+names+"]", nil)
}

// GetApplicationById 根据id获取应用
func GetApplicationById(c echo.Context) error {
	id := c.Param("id")
	a, err := newApplicationRepository.FindDetailById(context.TODO(), id)
	if err != nil {
		return FailWithDataOperate(c, 500, "获取失败", "应用管理-获取: 获取应用, 失败原因[应用不存在]", nil)
	}
	return Success(c, a)
}

// GetApplicationOpsPolicy 查询应用的运维策略
func GetApplicationOpsPolicy(c echo.Context) error {
	id := c.Param("id")
	a, err := newApplicationRepository.FindPolicyById(context.TODO(), id)
	if err != nil {
		return FailWithDataOperate(c, 500, "获取失败", "应用管理-获取: 获取应用, 失败原因[应用不存在]", nil)
	}
	return Success(c, a)
}

// ExportApplication 导出应用
func ExportApplication(c echo.Context) error {
	u, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	var departmentIds []int64
	err := GetChildDepIds(u.DepartmentId, &departmentIds)
	if err != nil {
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}

	appForExport, err := newApplicationRepository.GetApplicationForExport(context.TODO(), departmentIds)
	if err != nil {
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}

	appStringForExport := make([][]string, len(appForExport))
	for i, app := range appForExport {
		apps := utils.Struct2StrArr(app)
		appStringForExport[i] = make([]string, len(apps))
		appStringForExport[i] = apps
	}
	appHandle := []string{"服务器名称", "部门机构", "应用名称", "应用程序", "服务器地址", "端口", "用户名", "密码", "应用路径", "应用参数", "描述"}

	file, err := utils.CreateExcelFile("应用列表", appHandle, appStringForExport)
	if err != nil {
		return FailWithDataOperate(c, 501, "导出失败", "", nil)
	}
	fileName := "应用列表.xlsx"

	// 记录操作日志
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		Result:          "成功",
		LogTypes:        "用户日志",
		Created:         utils.NowJsonTime(),
		LogContents:     "设备列表-导出[导出文件" + fileName + "]",
		Users:           u.Username,
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
