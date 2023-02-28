package api

import (
	"bytes"
	"net/http"
	"strconv"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
	"github.com/tealeg/xlsx"
)

func OrderLogPagingEndpoint(c echo.Context) error {
	pageIndex, _ := strconv.Atoi(c.QueryParam("pageIndex"))
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))
	asset := c.QueryParam("asset")
	applicant := c.QueryParam("applicant")
	approved := c.QueryParam("approved")
	ip := c.QueryParam("ip")
	status := c.QueryParam("status")
	beginTime := c.QueryParam("beginTime")
	endTime := c.QueryParam("endTime")
	applicationType := c.QueryParam("applicationType")

	items, total, err := orderLogRepository.Find(pageIndex, pageSize, asset, applicant, approved, ip, status, beginTime, endTime, applicationType)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}

	return SuccessWithOperate(c, "", H{
		"total": total,
		"items": items,
	})
}

func OrderLogExportEndpoint(c echo.Context) error {
	asset := c.QueryParam("asset")
	applicant := c.QueryParam("applicant")
	approved := c.QueryParam("approved")
	ip := c.QueryParam("ip")
	status := c.QueryParam("status")
	beginTime := c.QueryParam("beginTime")
	endTime := c.QueryParam("endTime")
	atype := c.QueryParam("applicationType")

	items, _, err := orderLogRepository.Export(asset, applicant, approved, ip, status, atype, beginTime, endTime)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "导出工单审批日志失败", err)
	}

	name := "工单审批日志.xlsx"
	xFile := xlsx.NewFile()
	sheet, err := xFile.AddSheet("工单审批日志")
	if nil != err {
		log.Errorf("AddSheet Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "导出工单审批日志失败", err)
	}
	header := []string{"序号", "日志ID", "申请时间", "审批时间", "申请IP", "资产名称", "申请人", "申请状态", "审批人", "申请内容", "申请类型"}
	r := sheet.AddRow()
	var ce *xlsx.Cell
	for _, v := range header {
		ce = r.AddCell()
		ce.Value = v
	}
	for k, v := range items {
		r := sheet.AddRow()
		ce = r.AddCell()
		ce.Value = strconv.Itoa(k + 1)
		ce = r.AddCell()
		ce.Value = v.ID
		ce = r.AddCell()
		ce.Value = v.Created.Format("2006-01-02 15:04:05")
		ce = r.AddCell()
		ce.Value = v.ApproveTime.Format("2006-01-02 15:04:05")
		ce = r.AddCell()
		ce.Value = v.IP
		ce = r.AddCell()
		ce.Value = v.Asset
		ce = r.AddCell()
		ce.Value = v.Applicant
		ce := r.AddCell()
		ce.Value = v.Status
		ce = r.AddCell()
		//ce.Value = v.Command
		//ce = r.AddCell()
		ce.Value = v.Approved
		ce = r.AddCell()
		ce.Value = v.Information
		ce = r.AddCell()
		ce.Value = v.ApplicationType
	}

	err = xFile.Save(name)
	if nil != err {
		log.Errorf("Save Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "导出工单审批日志失败", err)
	}

	//将数据存入buffer
	var buff bytes.Buffer
	if err = xFile.Write(&buff); err != nil {
		log.Errorf("Write Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "导出工单审批日志失败", err)
	}

	user, _ := GetCurrentAccount(c)
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "审计日志",
		Created:         utils.NowJsonTime(),
		Users:           user.Username,
		LogContents:     "导出工单审批日志xlsx文件",
		Result:          "成功",
	}
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "导出工单审批日志失败", err)
	}

	//设置请求头  使用浏览器下载
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+name)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
}
