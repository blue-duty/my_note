package api

import (
	"bytes"
	"context"
	"net/http"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
)

func NewJopLogPagingEndpoint(c echo.Context) error {
	var njlfs dto.NewJobLogForSearch
	njlfs.Name = c.QueryParam("name")
	njlfs.Auto = c.QueryParam("auto")
	njlfs.Department = c.QueryParam("department")
	njlfs.RunTimeType = c.QueryParam("type")
	njlfs.Content = c.QueryParam("content")
	njlfs.Result = c.QueryParam("result")

	u, f := GetCurrentAccountNew(c)
	if f {
		err := GetChildDepIds(u.DepartmentId, &njlfs.DepartmentIds)
		if err != nil {
			return FailWithData(c, 500, "获取任务日志列表失败", nil)
		}
	} else {
		return FailWithData(c, 401, "身份验证失败", nil)
	}

	resp, err := newjobRepository.FindRunLog(context.TODO(), njlfs)
	if err != nil {
		return Success(c, njlfs)
	}
	return Success(c, resp)
}

func NewJopLogGetEndpoint(c echo.Context) error {
	id := c.Param("id")
	resp, err := newjobRepository.FindRunLogById(context.TODO(), id)
	if err != nil {
		return Fail(c, 500, "获取任务日志失败")
	}
	return Success(c, resp)
}

func NewJopLogExportEndpoint(c echo.Context) error {
	id := c.Param("id")

	u, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "身份验证失败")
	}

	logForExport, err := newjobRepository.FindRunLogForExport(context.TODO(), id)
	if err != nil {
		return Fail(c, 500, "获取任务日志失败")
	}

	logExport := make([][]string, 1)
	logExport[0] = utils.Struct2StrArr(logForExport)

	hander := []string{"日志ID", "任务名称", "设备地址", "部门机构", "设备端口", "设备账号", "命令/脚本", "开始时间", "结束时间", "执行结果"}
	fileName := "自动任务日志"

	file, err := utils.CreateExcelFile(fileName, hander, logExport)
	if err != nil {
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	name := logForExport.Name + "-自动任务日志.xlsx"

	// 记录操作日志
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		Result:          "成功",
		LogTypes:        "用户日志",
		LogContents:     "任务日志-导出: 任务名称[" + logForExport.Name + "]",
		Created:         utils.NowJsonTime(),
		Names:           u.Nickname,
		Users:           u.Username,
	}
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("记录操作日志失败: %v", err)
	}

	var buff bytes.Buffer
	if err = file.Write(&buff); err != nil {
		log.Errorf("自动任务日志ID%d导出失败: %v", logForExport.ID, err)
		return FailWithDataOperate(c, 500, "导出失败", "", err)
	}
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+name)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
}
