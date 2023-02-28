package api

import (
	"bytes"
	"net/http"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
)

func OperateAlertPagingEndpoint(c echo.Context) error {
	var operateAlarmLogForSearch dto.OperateAlarmLogForSearch
	operateAlarmLogForSearch.Auto = c.QueryParam("auto")
	operateAlarmLogForSearch.ClientIP = c.QueryParam("client_ip")
	operateAlarmLogForSearch.Username = c.QueryParam("username")
	operateAlarmLogForSearch.AssetIp = c.QueryParam("asset_ip")
	operateAlarmLogForSearch.Passport = c.QueryParam("passport")
	operateAlarmLogForSearch.Content = c.QueryParam("content")
	operateAlarmLogForSearch.Level = c.QueryParam("level")

	_, f := GetCurrentAccountNew(c)
	if f == false {
		return FailWithData(c, 401, "请登录", nil)
	}

	operateAlarmLog, err := operateAlarmLogRepository.FindByAlarmLogForSearch(c.Request().Context(), operateAlarmLogForSearch)
	if err != nil {
		log.Error("获取操作告警日志失败")
		return Success(c, []dto.OperateAlarmLog{})
	}

	return Success(c, operateAlarmLog)
}

func OperateAlertExportEndpoint(c echo.Context) error {

	u, f := GetCurrentAccountNew(c)
	if f == false {
		return FailWithData(c, 401, "请登录", nil)
	}

	operateAlarmLog, err := operateAlarmLogRepository.FindByAlarmLogForSearch(c.Request().Context(), dto.OperateAlarmLogForSearch{})
	if err != nil {
		log.Error("获取操作告警日志失败")
	}

	operateAlarmLogStringsForExport := make([][]string, len(operateAlarmLog))
	for i, v := range operateAlarmLog {
		operateAlarmLogStringsForExport[i] = utils.Struct2StrArr(v)
	}

	header := []string{"告警时间", "来源地址", "用户名", "姓名", "设备地址", "设备账号", "协议", "告警内容", "触发策略", "事件级别", "发送结果"}

	file, err := utils.CreateExcelFile("告警报表", header, operateAlarmLogStringsForExport)
	if err != nil {
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	fileName := "操作告警报表.xlsx"

	// 记录操作日志
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		Result:          "成功",
		LogTypes:        "操作日志",
		Created:         utils.NowJsonTime(),
		Users:           u.Username,
		Names:           u.Nickname,
		LogContents:     "操作告警-导出, [导出文件名]:" + fileName,
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

// SystemAlertPagingEndpoint 系统告警获取
func SystemAlertPagingEndpoint(c echo.Context) error {
	auto := c.QueryParam("auto")
	content := c.QueryParam("content")
	level := c.QueryParam("level")
	systemAlarmLog, err := operateAlarmLogRepository.FindBySystemAlarmLogForSearch(c.Request().Context(), auto, content, level)
	if err != nil {
		log.Error("获取系统告警日志失败")
		return Success(c, []dto.OperateAlarmLog{})
	}
	for i := range systemAlarmLog {
		var lev = ""
		if systemAlarmLog[i].Level == "high" {
			lev = "高"
		} else if systemAlarmLog[i].Level == "middle" {
			lev = "中"
		} else {
			lev = "低"
		}
		systemAlarmLog[i].Level = lev
	}
	return Success(c, systemAlarmLog)
}

// SystemAlertExportEndpoint 系统告警导出
func SystemAlertExportEndpoint(c echo.Context) error {
	account, found := GetCurrentAccountNew(c)
	if !found {
		return FailWithData(c, 401, "用户信息已过期，请重新登录", nil)
	}
	systemAlarmLog, err := operateAlarmLogRepository.FindBySystemAlarmLogForSearch(c.Request().Context(), "", "", "")
	if err != nil {
		log.Error("获取系统告警日志失败")
	}
	data := make([][]string, len(systemAlarmLog))
	for i, v := range systemAlarmLog {
		var level = ""
		if v.Level == "high" {
			level = "高"
		} else if v.Level == "middle" {
			level = "中"
		} else {
			level = "低"
		}
		var systemAlarmLogForExport = dto.SystemAlarmLogForExport{
			AlarmTime: v.AlarmTime.Format("2006-01-02 15:04:05"),
			Content:   v.Content,
			Strategy:  v.Strategy,
			Level:     level,
			Result:    v.Result,
		}
		data[i] = utils.Struct2StrArr(systemAlarmLogForExport)
	}

	header := []string{"告警时间", "告警内容", "触发策略", "事件级别", "发送结果"}

	file, err := utils.CreateExcelFile("系统告警日志", header, data)
	if err != nil {
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	fileName := "系统告警日志.xlsx"

	// 记录操作日志
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		Result:          "成功",
		LogTypes:        "用户日志",
		LogContents:     "系统告警-导出: 导出文件名:" + fileName,
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
