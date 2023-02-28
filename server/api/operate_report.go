package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
	"tkbastion/pkg/log"
	"tkbastion/server/dto"
	"tkbastion/server/utils"

	"baliance.com/gooxml/document"
	"github.com/labstack/echo/v4"
	"github.com/signintech/gopdf"
)

// GetAssetAccessReport 通过时间段获取资产运维报表
func GetAssetAccessReport(c echo.Context) error {
	// 获取参数
	startTime := c.QueryParam("startTime")
	endTime := c.QueryParam("endTime")
	// 判断查询的时间段是否合法
	if startTime == "" || endTime == "" {
		return Success(c, nil)
	}
	// 判断查询的间隔类型
	// 1. 间隔类型为天
	// 2. 间隔类型为周
	// 3. 间隔类型为月
	// 4. 间隔类型为小时
	t := utils.GetQueryType(startTime, endTime)
	switch t {
	case 1:
		resp, err := operateReportRepository.GetOperateLoginLogByDay(context.TODO(), startTime, endTime)
		if err != nil {
			return Success(c, resp)
		}
		return Success(c, resp)
	case 4:
		resp, err := operateReportRepository.GetOperateLoginLogByHour(context.TODO(), startTime, endTime)
		if err != nil {
			return Success(c, resp)
		}
		return Success(c, resp)
	case 2:
		resp, err := operateReportRepository.GetOperateLoginLogByWeek(context.TODO(), startTime, endTime)
		if err != nil {
			return Success(c, resp)
		}
		return Success(c, resp)
	case 3:
		resp, err := operateReportRepository.GetOperateLoginLogByMonth(context.TODO(), startTime, endTime)
		if err != nil {
			return Success(c, resp)
		}
		return Success(c, resp)
	default:
		return Success(c, "参数错误")
	}
}

// GetAssetAccessReportAssetDetailByTime 通过时间获取资产运维设备详情表
func GetAssetAccessReportAssetDetailByTime(c echo.Context) error {
	// 判断时间的格式
	// 1. 间隔类型为天
	// 2. 间隔类型为周
	// 3. 间隔类型为月
	// 4. 间隔类型为小时
	time := c.QueryParam("time")
	if time == "" {
		return Success(c, nil)
	}
	if len(time) == 10 {
		resp, err := operateReportRepository.GetOperateLoginLogAssetDetailByDay(context.TODO(), time)
		if err != nil {
			return Success(c, resp)
		}
		return Success(c, resp)
	}
	if len(time) == 13 {
		resp, err := operateReportRepository.GetOperateLoginLogAssetDetailByHour(context.TODO(), time)
		if err != nil {
			return Success(c, resp)
		}
		return Success(c, resp)
	}
	if len(time) == 7 {
		resp, err := operateReportRepository.GetOperateLoginLogAssetDetailByMonth(context.TODO(), time)
		if err != nil {
			return Success(c, resp)
		}
		return Success(c, resp)
	}
	// TODO 有问题
	if len(time) == 8 {
		resp, err := operateReportRepository.GetOperateLoginLogAssetDetailByWeek(context.TODO(), time)
		if err != nil {
			return Success(c, resp)
		}
		return Success(c, resp)
	}
	return Success(c, nil)
}

// GetAssetAccessReportUserDetailByTime 通过时间获取资产运维用户详情表
func GetAssetAccessReportUserDetailByTime(c echo.Context) error {
	// 判断时间的格式
	// 1. 间隔类型为天
	// 2. 间隔类型为周
	// 3. 间隔类型为月
	// 4. 间隔类型为小时
	time := c.QueryParam("time")
	if time == "" {
		return Success(c, nil)
	}
	if len(time) == 10 {
		resp, err := operateReportRepository.GetOperateLoginLogUserDetailByDay(context.TODO(), time)
		if err != nil {
			return Success(c, resp)
		}
		return Success(c, resp)
	}
	if len(time) == 13 {
		resp, err := operateReportRepository.GetOperateLoginLogUserDetailByHour(context.TODO(), time)
		if err != nil {
			return Success(c, resp)
		}
		return Success(c, resp)
	}
	if len(time) == 7 {
		resp, err := operateReportRepository.GetOperateLoginLogUserDetailByMonth(context.TODO(), time)
		if err != nil {
			return Success(c, resp)
		}
		return Success(c, resp)
	}
	if len(time) == 8 {
		resp, err := operateReportRepository.GetOperateLoginLogUserDetailByWeek(context.TODO(), time)
		if err != nil {
			return Success(c, resp)
		}
		return Success(c, resp)
	}
	return Success(c, nil)
}

// ExportAssetAccessReport 通过时间间隔和导出格式导出资产运维报表
func ExportAssetAccessReport(c echo.Context) error {
	// 获取参数
	startTime := c.QueryParam("startTime")
	endTime := c.QueryParam("endTime")
	exportType := c.QueryParam("exportType")
	if startTime == "" || endTime == "" || exportType == "" {
		return Success(c, nil)
	}
	// 判断查询的间隔类型
	// 1. 间隔类型为天
	// 2. 间隔类型为周
	// 3. 间隔类型为月
	// 4. 间隔类型为小时
	t := utils.GetQueryType(startTime, endTime)
	var content1 []dto.AssetAccess
	var content2 []dto.AssetAccessExport
	fmt.Println(t)
	if t == 1 {
		c1, c2, err := operateReportRepository.GetOperateLoginLogExportByDay(context.TODO(), startTime, endTime)
		if err != nil {
			fmt.Println(err)
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		content1 = c1
		content2 = c2
	} else if t == 4 {
		c1, c2, err := operateReportRepository.GetOperateLoginLogExportByHour(context.TODO(), startTime, endTime)
		if err != nil {
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		content1 = c1
		content2 = c2
	} else if t == 3 {
		c1, c2, err := operateReportRepository.GetOperateLoginLogExportByWeek(context.TODO(), startTime, endTime)
		if err != nil {
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		content1 = c1
		content2 = c2
	} else if t == 2 {
		c1, c2, err := operateReportRepository.GetOperateLoginLogExportByMonth(context.TODO(), startTime, endTime)
		if err != nil {
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		content1 = c1
		content2 = c2
	} else {
		return FailWithDataOperate(c, 500, "导出失败", "", nil)
	}
	// 导出
	// 数据转化为[]string
	var data1 = make([][]string, len(content1))
	var data2 = make([][]string, len(content2))
	for i, v := range content1 {
		data1[i] = utils.Struct2StrArr(v)
	}
	for i, v := range content2 {
		data2[i] = utils.Struct2StrArr(v)
	}
	//fmt.Println(data1)
	//fmt.Println(data2)
	// 表头
	var header1 = []string{"时间", "设备数", "用户数"}
	var header2 = []string{"开始时间", "结束时间", "用户名", "姓名", "来源地址", "设备名称", "设备地址", "设备账号", "协议"}
	var fileName = "运维报表"
	var fileReader *bytes.Reader
	switch exportType {
	case "pdf":
		// 导出pdf
		size1 := []int{100, 50, 50}
		size2 := []int{100, 100, 50, 50, 50, 50, 50, 50, 50}
		pdf := gopdf.GoPdf{}
		pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
		pdf.AddPage()
		err, xs, ys := utils.PdfExport(&pdf, "统计数据", header1, size1, data1, 0, 0)
		if err != nil {
			log.Errorf("导出pdf失败: %v", err)
		}
		err, xs, ys = utils.PdfExport(&pdf, "详细数据", header2, size2, data2, xs, ys)
		if err != nil {
			log.Errorf("导出pdf失败: %v", err)
		}

		var buff *bytes.Reader
		buff, err = utils.PdfToReader(&pdf)
		if err != nil {
			return Fail(c, 500, "导出失败")
		}
		fileReader = buff
		fileName = fileName + "-" + time.Now().Format("20060102150405") + ".pdf"
	case "word":
		// 导出word
		fmt.Println("word")
		d := document.New()
		err := utils.CreateWord(d, "统计数据", header1, data1)
		if err != nil {
			return FailWithDataOperate(c, 500, err.Error(), "", err)
		}
		fmt.Println("word1")
		err = utils.CreateWord(d, "详细数据", header2, data2)
		if err != nil {
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		fmt.Println("word2")
		file, err := utils.DocumentToReader(d)
		if err != nil {
			fmt.Println(err)
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		fmt.Println("word3")
		fileReader = file
		fileName = fileName + "-" + time.Now().Format("20060102150405") + ".docx"
	case "csv":
		// 导出csv
		file, err := utils.ExportCsv(header1, header2, data1, data2)
		if err != nil {
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		fileReader = file
		fileName = fileName + "-" + time.Now().Format("20060102150405") + ".csv"
	case "html":
		// 导出html
		file, err := utils.ExportHtml(header1, header2, data1, data2)
		if err != nil {
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		fileReader = file
		fileName = fileName + "-" + time.Now().Format("20060102150405") + ".html"
	}
	// 返回文件流
	fmt.Println("返回文件流")
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, fileReader)
}

// GetSessionAssetsReport 获取设备会话时长报表
func GetSessionAssetsReport(c echo.Context) error {
	// 获取参数
	startTime := c.QueryParam("startTime")
	endTime := c.QueryParam("endTime")
	// 查询数据
	content, err := operateReportRepository.GetOperateSessionLogAsset(context.TODO(), startTime, endTime)
	if err != nil {
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	// 返回数据
	return SuccessWithOperate(c, "", content)
}

// GetSessionUsersReport 获取用户会话时长报表
func GetSessionUsersReport(c echo.Context) error {
	// 获取参数
	startTime := c.QueryParam("startTime")
	endTime := c.QueryParam("endTime")
	// 查询数据
	content, err := operateReportRepository.GetOperateSessionLogUser(context.TODO(), startTime, endTime)
	if err != nil {
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	// 返回数据
	return SuccessWithOperate(c, "", content)
}

// ExportSessionReport 导出会话时长报表
func ExportSessionReport(c echo.Context) error {
	// 获取参数
	startTime := c.QueryParam("startTime")
	endTime := c.QueryParam("endTime")
	exportType := c.QueryParam("exportType")
	searchType := c.QueryParam("searchType")
	// 查询数据

	var (
		header1 []string
		header2 = []string{"开始时间", "结束时间", "用户名", "姓名", "来源地址", "设备名称", "设备地址", "设备账号", "协议"}
		data1   [][]string
		data2   [][]string
	)
	if searchType == "用户" {
		header1 = []string{"用户名", "姓名", "时长(秒)"}
		content, err := operateReportRepository.GetOperateSessionLogUser(context.TODO(), startTime, endTime)
		if err != nil {
			log.Errorf("查询失败: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", err)
		}
		data1 = make([][]string, len(content))
		for i, v := range content {
			data1[i] = []string{v.UserName, v.NickName, strconv.Itoa(int(v.Time))}
		}
	} else {
		header1 = []string{"设备名称", "设备地址", "时长(秒)"}
		content, err := operateReportRepository.GetOperateSessionLogAsset(context.TODO(), startTime, endTime)
		if err != nil {
			log.Errorf("查询失败: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", err)
		}
		data1 = make([][]string, len(content))
		for i, v := range content {
			data1[i] = []string{v.AssetName, v.AssetIP, strconv.Itoa(int(v.Time))}
		}
	}

	content1, err := operateReportRepository.GetOperateSessionLogExport(context.TODO(), startTime, endTime)
	if err != nil {
		log.Errorf("查询失败: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	data2 = make([][]string, len(content1))
	for i, v := range content1 {
		data2[i] = utils.Struct2StrArr(v)
	}

	// 导出文件
	var fileName = "会话时长报表"
	var fileReader *bytes.Reader
	switch exportType {
	case "pdf":
		// 导出pdf
		size1 := []int{100, 50, 50}
		size2 := []int{100, 100, 50, 50, 50, 50, 50, 50, 50}
		pdf := gopdf.GoPdf{}
		pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
		pdf.AddPage()
		err, xs, ys := utils.PdfExport(&pdf, "统计数据", header1, size1, data1, 0, 0)
		if err != nil {
			log.Errorf("导出pdf失败: %v", err)
		}
		err, xs, ys = utils.PdfExport(&pdf, "详细数据", header2, size2, data2, xs, ys)
		if err != nil {
			log.Errorf("导出pdf失败: %v", err)
			return FailWithDataOperate(c, 500, "导出pdf失败", "", err)
		}
		fileReader = bytes.NewReader(pdf.GetBytesPdf())
		fileName = fileName + "-" + time.Now().Format("20060102150405") + ".pdf"
	case "csv":
		// 导出csv
		fileReader, err = utils.ExportCsv(header1, header2, data1, data2)
		if err != nil {
			log.Errorf("导出csv失败: %v", err)
			return FailWithDataOperate(c, 500, "导出csv失败", "", err)
		}
		fileName = fileName + "-" + time.Now().Format("20060102150405") + ".csv"
	case "word":
		// 导出word
		d := document.New()
		err = utils.CreateWord(d, "统计数据", header1, data1)
		if err != nil {
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		err = utils.CreateWord(d, "详细数据", header2, data2)
		if err != nil {
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		fileReader, err = utils.DocumentToReader(d)
		if err != nil {
			fmt.Println(err)
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		fileName = fileName + "-" + time.Now().Format("20060102150405") + ".docx"
	case "html":
		// 导出html
		fileReader, err = utils.ExportHtml(header1, header2, data1, data2)
		if err != nil {
			log.Errorf("导出html失败: %v", err)
			return FailWithDataOperate(c, 500, "导出html失败", "", err)
		}
		fileName = fileName + "-" + time.Now().Format("20060102150405") + ".html"
	default:
		return FailWithDataOperate(c, 500, "导出类型错误", "", nil)
	}

	// 返回文件流
	fmt.Println("返回文件流")
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, fileReader)
}

// GetSessionReportAssetDetailByTime 获取设备会话时长报表详情
func GetSessionReportAssetDetailByTime(c echo.Context) error {
	// 获取参数
	startTime := c.QueryParam("startTime")
	endTime := c.QueryParam("endTime")
	assetName := c.QueryParam("assetName")
	// 查询数据
	content, err := operateReportRepository.GetOperateSessionLogAssetDetail(context.TODO(), assetName, startTime, endTime)
	if err != nil {
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	// 返回数据
	return SuccessWithOperate(c, "", content)
}

// GetSessionReportUserDetailByTime 获取用户会话时长报表详情
func GetSessionReportUserDetailByTime(c echo.Context) error {
	// 获取参数
	startTime := c.QueryParam("startTime")
	endTime := c.QueryParam("endTime")
	username := c.QueryParam("username")
	// 查询数据
	content, err := operateReportRepository.GetOperateSessionLogUserDetail(context.TODO(), username, startTime, endTime)
	if err != nil {
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	// 返回数据
	return SuccessWithOperate(c, "", content)
}

// GetCommandRecordByTime 通过时间段获取命令统计
func GetCommandRecordByTime(c echo.Context) error {
	startTime := c.QueryParam("startTime")
	endTime := c.QueryParam("endTime")
	commandRecord, err := operateReportRepository.GetOperateCommandCount(context.TODO(), startTime, endTime)
	if err != nil {
		log.Errorf("获取命令统计失败: %v", err)
		return Fail(c, 500, "获取命令统计失败")
	}
	return Success(c, commandRecord)
}

func GetCommandRecordDetails(c echo.Context) error {
	startTime := c.QueryParam("startTime")
	endTime := c.QueryParam("endTime")
	command := c.QueryParam("command")
	commandRecordDetails, err := operateReportRepository.GetOperateCommandDetails(context.TODO(), startTime, endTime, command)
	if err != nil {
		log.Errorf("获取命令统计详情失败: %v", err)
		return Fail(c, 500, "获取命令统计详情失败")
	}
	return Success(c, commandRecordDetails)
}

func ExportCommandRecordReport(c echo.Context) error {
	startTime := c.QueryParam("startTime")
	endTime := c.QueryParam("endTime")
	exportType := c.QueryParam("exportType")
	commandRecord, err := operateReportRepository.GetOperateCommandCount(context.TODO(), startTime, endTime)
	if err != nil {
		log.Errorf("获取命令统计失败: %v", err)
		return Fail(c, 500, "获取命令统计失败")
	}
	commandRecordDetails, err := operateReportRepository.GetOperateCommandDetails(context.TODO(), startTime, endTime, "")
	if err != nil {
		log.Errorf("获取命令统计详情失败: %v", err)
		return Fail(c, 500, "获取命令统计详情失败")
	}
	var commandRecordDetailsExport = make([]dto.CommandStatisticsDetail, len(commandRecordDetails))
	for i, v := range commandRecordDetails {
		commandRecordDetailsExport[i].Created = v.Created.Format("2006-01-02 15:04:05")
		commandRecordDetailsExport[i].Content = v.Content
		commandRecordDetailsExport[i].Username = v.Username
		commandRecordDetailsExport[i].Nickname = v.Nickname
		commandRecordDetailsExport[i].ClientIp = v.ClientIp
		commandRecordDetailsExport[i].AssetName = v.AssetName
		commandRecordDetailsExport[i].AssetIp = v.AssetIp
		commandRecordDetailsExport[i].Passport = v.Passport
		commandRecordDetailsExport[i].Protocol = v.Protocol
	}
	fileName := "命令统计-" + time.Now().Format("20060102150405")
	var data1, data2 [][]string
	for _, v := range commandRecord {
		data := utils.Struct2StrArr(v)
		data1 = append(data1, data)
	}
	data1Title := []string{"命令", "次数"}

	for _, v := range commandRecordDetailsExport {
		data := utils.Struct2StrArr(v)
		data2 = append(data2, data)
	}
	data2Title := []string{"执行时间", "命令", "用户名", "姓名", "来源地址", "设备名称", "设备地址", "设备账号", "协议"}
	var fileReader *bytes.Reader

	switch exportType {
	case "pdf":
		size1 := []int{80, 80}
		size2 := []int{100, 60, 40, 60, 80, 80, 80, 40, 35}
		//创建一个pdf文档
		pdf := gopdf.GoPdf{}
		pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
		pdf.AddPage()
		err, xs, ys := utils.PdfExport(&pdf, "统计数据", data1Title, size1, data1, 0, 0)
		if err != nil {
			fmt.Println(err)
			return Fail(c, 500, "导出失败")
		}
		err, _, _ = utils.PdfExport(&pdf, "详细数据", data2Title, size2, data2, xs, ys)
		if err != nil {
			return Fail(c, 500, "导出失败")
		}
		// 将pdf文件转换成字节流
		fileReader, err = utils.PdfToReader(&pdf)
		if err != nil {
			log.Errorf("PdfToReader error: %v", err)
		}
		fileName = fileName + ".pdf"
	case "word":
		// 创建word文件
		d := document.New()
		err := utils.CreateWord(d, "统计数据", data1Title, data1)
		if err != nil {
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		err = utils.CreateWord(d, "详细数据", data2Title, data2)
		if err != nil {
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		file, err := utils.DocumentToReader(d)
		if err != nil {
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		fileReader = file
		fileName = fileName + ".docx"
	case "html":
		// 创建html文件
		file, err := utils.ExportHtml(data1Title, data2Title, data1, data2)
		if err != nil {
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		fileReader = file
		fileName = fileName + ".html"
	case "csv":
		// 创建csv文件
		file, err := utils.ExportCsv(data1Title, data2Title, data1, data2)
		if err != nil {
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		fileReader = file
		fileName = fileName + ".csv"
	}

	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, fileReader)
}

func GetAlarmReport(c echo.Context) error {
	start := c.QueryParam("startTime")
	end := c.QueryParam("endTime")
	// 如果时间相差一一天以内则小时统计否则按天统计
	var searchType, newStart, newEnd = utils.JudgeTimeGapType(start, end)
	fmt.Println(newStart, newEnd)
	protocolCountByDay, err := operateAlarmLogRepository.FindByAlarmLogByDate(context.TODO(), newStart, newEnd, searchType)
	if err != nil {
		log.Errorf("GetProtocolCountByDayEndpoint error: %v", err)
		return FailWithDataOperate(c, 500, "查找失败", "", err)
	}
	return Success(c, protocolCountByDay)
}
func GetAlarmReportDetails(c echo.Context) error {
	date := c.QueryParam("dateString")
	level := c.QueryParam("level")
	// 将date转换为 日期格式
	if date == "" {
		return Success(c, []dto.AlarmReportDetail{})
	}
	start, end := utils.JudgeTimeType(date)
	//endDate, _ := time.Parse("2006-01-02", date)
	//end := endDate.AddDate(0, 0, 1).Format("2006-01-02")
	alarmReportDetail, err := operateAlarmLogRepository.GetAlarmLogDetailsStatist(context.TODO(), start, end, level)
	if err != nil {
		log.Errorf("GetLoginDetailsEndpoint error: %v", err)
	}
	return Success(c, alarmReportDetail)
}
func ExportAlarmReport(c echo.Context) error {
	startTime := c.QueryParam("startTime")
	endTime := c.QueryParam("endTime")
	exportType := c.QueryParam("exportType")
	var searchType, newStart, newEnd = utils.JudgeTimeGapType(startTime, endTime)
	var (
		header1      = []string{"时间", "高", "中", "低"}
		header2      = []string{"告警时间", "用户名", "姓名", "来源地址", "设备地址", "设备账号", "协议", "触发策略", "告警级别"}
		data1, data2 [][]string
	)
	content1, err := operateAlarmLogRepository.FindByAlarmLogByDate(context.TODO(), newStart, newEnd, searchType)
	if err != nil {
		log.Errorf("GetProtocolCountByDayEndpoint error: %v", err)
		log.Errorf("运维报表-告警统计: 导出[获取数据文件失败]")
	}
	content2, err := operateAlarmLogRepository.GetAlarmLogDetailsStatist(context.TODO(), startTime, endTime, "")
	if err != nil {
		log.Errorf("GetLoginDetailsEndpoint error: %v", err)
		log.Errorf("运维报表-告警统计: 导出[获取数据文件失败]")
	}
	data1 = make([][]string, len(content1))
	for i, v := range content1 {
		data1[i] = utils.Struct2StrArr(v)
	}
	data2 = make([][]string, len(content2))
	for i, v := range content2 {
		data2[i] = utils.Struct2StrArr(v)
	}

	var (
		fileReader io.Reader
		fileName   = "告警报表"
	)

	switch exportType {
	case "pdf":
		// 创建pdf文件
		size1 := []int{100, 80, 80, 80}
		size2 := []int{110, 60, 60, 70, 70, 60, 40, 40, 40}
		pdf := gopdf.GoPdf{}
		pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
		pdf.AddPage()
		err, xs, ys := utils.PdfExport(&pdf, "统计数据", header1, size1, data1, 0, 0)
		if err != nil {
			log.Errorf("导出pdf失败: %v", err)
		}
		err, xs, ys = utils.PdfExport(&pdf, "详细数据", header2, size2, data2, xs, ys)
		if err != nil {
			log.Errorf("导出pdf失败: %v", err)
			return FailWithDataOperate(c, 500, "导出pdf失败", "", err)
		}
		fileReader = bytes.NewReader(pdf.GetBytesPdf())
		fileName = fileName + "-" + time.Now().Format("20060102150405") + ".pdf"
	case "csv":
		// 导出csv
		fileReader, err = utils.ExportCsv(header1, header2, data1, data2)
		if err != nil {
			log.Errorf("导出csv失败: %v", err)
			return FailWithDataOperate(c, 500, "导出csv失败", "", err)
		}
		fileName = fileName + "-" + time.Now().Format("20060102150405") + ".csv"
	case "word":
		// 导出word
		d := document.New()
		err = utils.CreateWord(d, "统计数据", header1, data1)
		if err != nil {
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		err = utils.CreateWord(d, "详细数据", header2, data2)
		if err != nil {
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		fileReader, err = utils.DocumentToReader(d)
		if err != nil {
			fmt.Println(err)
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		fileName = fileName + "-" + time.Now().Format("20060102150405") + ".docx"
	case "html":
		// 导出html
		fileReader, err = utils.ExportHtml(header1, header2, data1, data2)
		if err != nil {
			log.Errorf("导出html失败: %v", err)
			return FailWithDataOperate(c, 500, "导出html失败", "", err)
		}
		fileName = fileName + "-" + time.Now().Format("20060102150405") + ".html"
	default:
		return FailWithDataOperate(c, 500, "导出类型错误", "", nil)
	}

	// 返回文件流
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, fileReader)
}
