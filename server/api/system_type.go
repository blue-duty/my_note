package api

import (
	"context"
	"strings"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/log"
	"tkbastion/pkg/validator"
	"tkbastion/server/dto"

	"github.com/labstack/echo/v4"
)

func SystemTypePagingEndpoint(c echo.Context) error {
	auto := c.QueryParam("auto")
	name := c.QueryParam("name")
	systemTypeList, err := systemTypeRepository.GetSystemTypeByNameOrAuto(context.TODO(), name, auto)
	if err != nil {
		return FailWithData(c, 500, "获取系统类型列表失败", nil)
	}
	return SuccessWithData(c, 200, "获取系统类型列表成功", systemTypeList)
}

func SystemTypeCreateEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithData(c, 401, "请重新登陆", nil)
	}

	var systemType dto.SystemTypeForCreate
	if err := c.Bind(&systemType); err != nil {
		log.Errorf("Bind Failed: %v", err)
		return FailWithData(c, 500, "新建系统类型失败", nil)
	}

	// 数据校验
	if err := c.Validate(&systemType); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	sts, err := systemTypeRepository.GetSystemTypeNames(context.TODO())
	if err != nil {
		log.Errorf("GetSystemTypeNames Failed: %v", err)
		return FailWithData(c, 500, "新建系统类型失败", nil)
	}

	for _, v := range sts {
		if strings.EqualFold(v, systemType.Name) {
			return FailWithDataOperate(c, 500, "新建系统类型失败", "系统类型-新建: 失败原因[系统类型名称"+systemType.Name+"已存在]", nil)
		}
	}

	if err := systemTypeRepository.CreateSystemType(context.TODO(), &systemType); err != nil {
		log.Errorf("CreateSystemType Failed: %v", err)
		return FailWithData(c, 500, "新建系统类型失败", nil)
	}
	return SuccessWithOperate(c, "系统类型-新建: 新建系统类型, 系统类型名称["+systemType.Name+"]", nil)
}

func SystemTypeUpdateEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithData(c, 401, "请重新登陆", nil)
	}
	id := c.Param("id")
	var systemType dto.SystemTypeForUpdate
	if err := c.Bind(&systemType); err != nil {
		log.Errorf("Bind Failed: %v", err)
		return FailWithData(c, 500, "更新系统类型失败", nil)
	}

	// 数据校验
	if err := c.Validate(&systemType); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	st, err := systemTypeRepository.GetSystemTypeByID(context.TODO(), id)
	if err != nil {
		log.Errorf("GetSystemTypeByID Failed: %v", err)
		return FailWithData(c, 500, "更新系统类型失败", nil)
	}
	sts, err := systemTypeRepository.GetSystemTypeByNameID(context.TODO(), id)
	if err != nil {
		log.Errorf("GetSystemTypeByNameID Failed: %v", err)
		return FailWithData(c, 500, "更新系统类型失败", nil)
	}

	for _, v := range sts {
		if strings.EqualFold(v, systemType.Name) {
			return FailWithDataOperate(c, 500, "更新系统类型失败", "系统类型-更新: 失败原因[系统类型名称"+systemType.Name+"已存在]", nil)
		}
	}

	if err := systemTypeRepository.UpdateSystemType(context.TODO(), &systemType, id); err != nil {
		log.Errorf("UpdateSystemType Failed: %v", err)
		return FailWithData(c, 500, "更新系统类型失败", nil)
	}
	return SuccessWithOperate(c, "系统类型-更新: 更新系统类型, 系统类型名称["+systemType.Name+"->"+st.Name+"]", nil)
}

func SystemTypeDeleteEndpoint(c echo.Context) error {
	id := c.Param("id")
	st, err := systemTypeRepository.GetSystemTypeByID(context.TODO(), id)
	if err != nil {
		log.Errorf("GetSystemTypeByID Failed: %v", err)
		return FailWithData(c, 500, "删除系统类型失败", nil)
	}
	if err := systemTypeRepository.DeleteSystemType(context.TODO(), id); err != nil {
		log.Errorf("DeleteSystemType Failed: %v", err)
		return FailWithData(c, 500, "删除系统类型失败", nil)
	}
	return SuccessWithOperate(c, "系统类型-删除: 删除系统类型, 系统类型名称["+st.Name+"]", nil)
}

func SystemTypeBatchDeleteEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	if id == "" {
		return FailWithData(c, 500, "删除系统类型失败", nil)
	}
	ids := strings.Split(id, ",")
	n := ""
	sts, err := systemTypeRepository.GetSystemTypeByIDs(context.TODO(), ids)
	if err != nil {
		log.Errorf("GetSystemTypeByIDs Failed: %v", err)
		return FailWithData(c, 500, "批量删除系统类型失败", nil)
	}
	for _, st := range sts {
		n += st.Name + ","
	}
	if err := systemTypeRepository.BatchDeleteSystemType(context.TODO(), ids); err != nil {
		log.Errorf("BatchDeleteSystemType Failed: %v", err)
		return FailWithData(c, 500, "批量删除系统类型失败", nil)
	}
	return SuccessWithOperate(c, "系统类型-批量删除: 系统类型名称["+n+"]", nil)
}
