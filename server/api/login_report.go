package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
	"tkbastion/pkg/log"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"baliance.com/gooxml/document"
	"github.com/labstack/echo/v4"
	"github.com/signintech/gopdf"
)

//协议访问统计

// GetProtocolCountStatistEndpoint 协议访问统计按每天进行统计数量
func GetProtocolCountStatistEndpoint(c echo.Context) error {
	start := c.QueryParam("start")
	end := c.QueryParam("end")
	// 如果时间相差一一天以内则小时统计否则按天统计
	var searchType, newStart, newEnd = utils.JudgeTimeGapType(start, end)
	protocolCountByDay, err := newSessionRepository.GetProtocolCountByDay(context.TODO(), newStart, newEnd, searchType)
	if err != nil {
		log.Errorf("GetProtocolCountByDayEndpoint error: %v", err)
		return FailWithDataOperate(c, 500, "查找失败", "", err)
	}
	for i := range protocolCountByDay {
		if protocolCountByDay[i].Tcp != 0 {
			protocolCountByDay[i].Total += protocolCountByDay[i].Tcp
		}
	}
	return Success(c, protocolCountByDay)
}

// GetLoginDetailsEndpoint 获取详细数据
func GetLoginDetailsEndpoint(c echo.Context) (err error) {
	date := c.QueryParam("dateString")
	protocol := c.QueryParam("protocol")
	// 将date转换为 日期格式
	start, end := utils.JudgeTimeType(date)
	if start == "" || end == "" {
		return Fail(c, 500, "时间格式错误")
	}
	fmt.Println(start, end)
	var sessionDetails, loginLogDetails []model.LoginDetails
	if protocol == "all" {
		loginLogDetailTemp, err := loginLogRepository.GetLoginDetailsStatist(start, end)
		if err != nil {
			log.Errorf("GetLoginDetailsEndpoint error: %v", err)
		}
		for _, v := range loginLogDetailTemp {
			loginLogDetails = append(loginLogDetails, *v.ToLoginDetailsDto())
		}
		for _, v := range loginLogDetails {
			v.Protocol = "tcp"
		}
		sessionDetailTemp, err := newSessionRepository.GetLoginDetails(context.TODO(), start, end, "")
		if err != nil {
			log.Errorf("GetLoginDetailsEndpoint error: %v", err)
		}
		for _, v := range sessionDetailTemp {
			sessionDetails = append(sessionDetails, *v.ToLoginDetailsDto())
		}
		sessionDetails = append(sessionDetails, loginLogDetails...)
		return Success(c, sessionDetails)
	} else if protocol == "tcp" {
		loginLogDetailTemp, err := loginLogRepository.GetLoginDetailsStatist(start, end)
		if err != nil {
			log.Errorf("GetLoginDetailsEndpoint error: %v", err)
		}
		for _, v := range loginLogDetailTemp {
			loginLogDetails = append(loginLogDetails, *v.ToLoginDetailsDto())
		}
		for _, v := range loginLogDetails {
			v.Protocol = "tcp"
		}
		return Success(c, loginLogDetails)
	} else {
		sessionDetailTemp, err := newSessionRepository.GetLoginDetails(context.TODO(), start, end, protocol)
		if err != nil {
			log.Errorf("GetLoginDetailsEndpoint error: %v", err)
		}
		for _, v := range sessionDetailTemp {
			sessionDetails = append(sessionDetails, *v.ToLoginDetailsDto())
		}
		return Success(c, sessionDetails)
	}
}

// GetProtocolCountStatistExportEndpoint 协议访问统计数据数据导出
func GetProtocolCountStatistExportEndpoint(c echo.Context) error {
	start := c.QueryParam("start")
	end := c.QueryParam("end")
	exportType := c.QueryParam("exportType")
	var searchType, newStart, newEnd = utils.JudgeTimeGapType(start, end)
	// 查询统计数据
	protocolCountByDay, err := newSessionRepository.GetProtocolCountByDay(context.TODO(), newStart, newEnd, searchType)
	if err != nil {
		log.Errorf("GetProtocolCountByDayEndpoint error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "", err)
	}
	var protocolCountExport []model.ProtocolCountExport
	for i := range protocolCountByDay {
		if protocolCountByDay[i].Tcp != 0 {
			protocolCountByDay[i].Total += protocolCountByDay[i].Tcp
		}
		protocolCountExport = append(protocolCountExport, model.ProtocolCountExport{
			Daytime: protocolCountByDay[i].Daytime,
			Ssh:     protocolCountByDay[i].Ssh,
			Rdp:     protocolCountByDay[i].Rdp,
			Telnet:  protocolCountByDay[i].Telnet,
			Vnc:     protocolCountByDay[i].Vnc,
			App:     protocolCountByDay[i].App,
			Tcp:     protocolCountByDay[i].Tcp,
			Total:   protocolCountByDay[i].Total,
		})
	}
	// 查询详细数据
	if start == end {
		start, end = utils.JudgeTimeType(start)
	}
	var sessionDetails, loginLogDetails []model.LoginDetails
	sessionDetailTemp, err := newSessionRepository.GetLoginDetails(context.TODO(), start, end, "")
	if err != nil {
		log.Errorf("GetLoginDetailsEndpoint error: %v", err)
	}
	for _, v := range sessionDetailTemp {
		sessionDetails = append(sessionDetails, *v.ToLoginDetailsDto())
	}
	loginLogDetailTemp, err := loginLogRepository.GetLoginDetailsStatist(start, end)
	if err != nil {
		log.Errorf("GetLoginDetailsEndpoint error: %v", err)
	}
	for _, v := range loginLogDetailTemp {
		loginLogDetails = append(loginLogDetails, *v.ToLoginDetailsDto())
	}
	sessionDetails = append(sessionDetails, loginLogDetails...)
	// 文件名
	fileName := "协议访问统计-" + time.Now().Format("20060102150405")
	// 将结构体数组转换为字符串数组
	var data1, data2 [][]string
	for _, v := range protocolCountExport {
		data := utils.Struct2StrArr(v)
		data1 = append(data1, data)
	}
	data1Title := []string{"日期", "SSH", "RDP", "TELNET", "VNC", "应用发布", "前台", "合计"}

	for _, v := range sessionDetails {
		data := utils.Struct2StrArr(v)
		data2 = append(data2, data)
	}
	data2Title := []string{"登录时间", "用户名", "姓名", "来源地址", "协议", "结果", "描述"}
	var fileReader *bytes.Reader

	switch exportType {
	case "pdf":
		size1 := []int{110, 50, 50, 50, 50, 100, 50, 50}
		size2 := []int{110, 60, 80, 100, 50, 50, 90}
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
	// 返回文件流
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, fileReader)
}

// 用户访问统计

// GetUserCountStatistEndpoint 用户访问统计柱状图
func GetUserCountStatistEndpoint(c echo.Context) error {
	// 获取参数
	start := c.QueryParam("start")
	end := c.QueryParam("end")
	// 查询数据
	userCount, err := userAccessStatisticsRepository.GetUserAccessStatisticsTotal(context.TODO(), start, end)
	if err != nil {
		log.Errorf("GetUserCountStatistEndpoint error: %v", err)
		return Fail(c, 500, "查询失败")
	}
	return Success(c, userCount)
}

// GetUserCountStatistSurfaceEndpoint 用户访问统计表格
func GetUserCountStatistSurfaceEndpoint(c echo.Context) error {
	// 获取参数
	start := c.QueryParam("start")
	end := c.QueryParam("end")
	// 查询数据
	userCount, err := userAccessStatisticsRepository.GetUserAccessStatisticsByDay(context.TODO(), start, end)
	if err != nil {
		log.Errorf("GetUserCountStatistSurfaceEndpoint error: %v", err)
		return Fail(c, 500, "查询失败")
	}
	return Success(c, userCount)
}

// GetUserLoginDetailsEndpoint 用户访问统计
func GetUserLoginDetailsEndpoint(c echo.Context) error {
	// 获取参数
	t := c.QueryParam("time")
	protocol := c.QueryParam("protocol")
	username := c.QueryParam("username")

	resp, err := userAccessStatisticsRepository.GetUserAccessStatisticsByProtocol(context.TODO(), t, protocol, username)
	if err != nil {
		log.Errorf("GetUserLoginDetailsEndpoint error: %v", err)
		return Success(c, []string{})
	}
	return Success(c, resp)
}

func GetUserCountStatistExportEndpoint(c echo.Context) error {
	// 获取参数
	start := c.QueryParam("start")
	end := c.QueryParam("end")
	exportType := c.QueryParam("exportType")
	// 查询数据

	var (
		header1      = []string{"时间", "用户名", "真实姓名", "SSH", "RDP", "TELNET", "VNC", "应用发布", "前台", "合计"}
		header2      = []string{"登陆时间", "用户名", "姓名", "来源地址", "协议", "结果", "描述"}
		data1, data2 [][]string
	)

	content1, err := userAccessStatisticsRepository.GetUserAccessStatisticsByDay(context.TODO(), start, end)
	if err != nil {
		log.Errorf("查询用户访问统计数据失败: %v", err)
		return Fail(c, 500, "查询失败")
	}
	data1 = make([][]string, len(content1))
	for i, v := range content1 {
		data1[i] = utils.Struct2StrArr(v)
	}

	content2, err := userAccessStatisticsRepository.GetUserAccessStatistics(context.TODO(), start, end)
	if err != nil {
		log.Errorf("查询用户访问详细数据失败: %v", err)
		return Fail(c, 500, "查询失败")
	}
	data2 = make([][]string, len(content2))
	for i, v := range content2 {
		data2[i] = utils.Struct2StrArr(v)
	}

	var (
		fileReader io.Reader
		fileName   = "用户访问统计"
	)

	switch exportType {
	case "pdf":
		// 创建pdf文件
		size1 := []int{80, 60, 70, 50, 50, 50, 50, 50, 50, 50}
		size2 := []int{110, 90, 90, 100, 50, 50, 50}
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

//登陆尝试统计

// GetAttemptCountStatistEndpoint 登陆尝试统计图
func GetAttemptCountStatistEndpoint(c echo.Context) error {
	// 获取参数
	start := c.QueryParam("start")
	end := c.QueryParam("end")
	// 查询数据
	userCount, err := userAccessStatisticsRepository.GetLoginStatistics(context.TODO(), start, end)
	if err != nil {
		log.Errorf("GetAttemptCountStatistEndpoint error: %v", err)
		return Fail(c, 500, "查询失败")
	}
	return Success(c, userCount)
}

// GetAttemptDetailsEndpoint 登陆尝试统计
func GetAttemptDetailsEndpoint(c echo.Context) error {
	// 获取参数
	t := c.QueryParam("time")
	ty := c.QueryParam("type")

	resp, err := userAccessStatisticsRepository.GetUserLoginStatisticsByTime(context.TODO(), t, ty)
	if err != nil {
		log.Errorf("GetAttemptDetailsEndpoint error: %v", err)
		return Success(c, []string{})
	}
	switch ty {
	case "user":
		return Success(c, resp.([]dto.LoginAttemptUserDetail))
	default:
		return Success(c, resp.([]dto.LoginAttemptDetail))
	}
}

// GetAttemptCountStatistExportEndpoint 登陆尝试统计导出
func GetAttemptCountStatistExportEndpoint(c echo.Context) error {
	// 获取参数
	start := c.QueryParam("start")
	end := c.QueryParam("end")
	exportType := c.QueryParam("exportType")

	// 查询数据
	content1, err := userAccessStatisticsRepository.GetLoginStatistics(context.TODO(), start, end)
	if err != nil {
		log.Errorf("获取登陆尝试统计数据失败: %v", err)
		return Fail(c, 500, "查询失败")
	}

	content2, err := userAccessStatisticsRepository.GetLoginStatisticsDetail(context.TODO(), start, end)
	if err != nil {
		log.Errorf("获取登陆尝试详细数据失败: %v", err)
		return Fail(c, 500, "查询失败")
	}

	// 处理数据
	header1 := []string{"时间", "用户数", "成功次数", "失败次数", "来源IP数", "总次数"}
	header2 := []string{"登陆时间", "用户名", "姓名", "来源地址", "结果", "描述"}
	var data1 = make([][]string, len(content1))
	var data2 = make([][]string, len(content2))
	for i, v := range content1 {
		data1[i] = utils.Struct2StrArr(v)
	}
	for i, v := range content2 {
		data2[i] = utils.Struct2StrArr(v)
	}

	var (
		fileReader io.Reader
		fileName   = "登陆尝试统计"
	)

	switch exportType {
	case "pdf":
		// 创建pdf文件
		size1 := []int{100, 50, 50, 50, 50, 50}
		size2 := []int{100, 100, 100, 100, 50, 100}
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
