package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"baliance.com/gooxml/color"
	"baliance.com/gooxml/document"
	"baliance.com/gooxml/measurement"
	"baliance.com/gooxml/schema/soo/wml"
	"github.com/klarkxy/gohtml"
	"github.com/signintech/gopdf"
)

// PdfExport 导出pdf文件
func PdfExport(pdf *gopdf.GoPdf, title string, tableTitle []string, size []int, data [][]string, xs, ys int) (err error, xe, ye int) {
	if len(tableTitle) != len(size) {
		err = fmt.Errorf("tableTitle size not equal size")
		return
	}
	// 获取数据的宽度
	var width = 0
	for _, v := range size {
		width += v
	}
	if width > 580 {
		err = fmt.Errorf("size too large")
		return
	}
	// 获取页边距
	var marginC = (595 - width) / 2
	var marginH = 30
	// 对齐方式
	alignCenter := gopdf.CellOption{Align: gopdf.Center | gopdf.Middle,
		Border: gopdf.AllBorders, Float: gopdf.Right}
	center := 595 / 2

	//设置font TODO 字体文件需要更换，先记录
	err = pdf.AddTTFFont("simhei", "./config/microsoft.ttf")
	if err != nil {
		return
	}
	err = pdf.SetFont("simhei", "", 20)
	if err != nil {
		return
	}

	// 设置初始值
	var x = marginC
	var y = marginH
	// 设置表头
	if ys != 0 {
		y = ys + 40
	}

	// 设置标题
	if title != "" {
		pdf.SetX(float64(center - len(title)*8/2))
		pdf.SetY(float64(y))
		pdf.Cell(nil, title)
	}

	// 设置表头
	// 设置表头字体
	err = pdf.SetFont("simhei", "", 10)
	//pdf.SetTextColor(0xa8, 0xa8, 0xa8)
	pdf.SetTextColor(0x00, 0x00, 0x00)
	if err != nil {
		return
	}

	for i := 0; i < len(tableTitle); i++ {
		pdf.SetX(float64(x))
		pdf.SetY(float64(y + 40))
		pdf.CellWithOption(&gopdf.Rect{
			W: float64(size[i]),
			H: 15,
		}, tableTitle[i], alignCenter)
		x += size[i]
	}

	y += 55
	ye = y
	xe = center

	// 写入数据
	var i = 0
	for ; i < len(data); i++ {
		x = marginC
		if len(data[i]) != len(size) {
			continue
		}
		for j := 0; j < len(data[i]); j++ {
			pdf.SetX(float64(x))
			pdf.SetY(float64(y + i*15))
			if float64(y+i*15) > 790 {
				pdf.AddPage()
				data = data[i:]
				i = 0
				y = marginH
				break
			}
			pdf.CellWithOption(&gopdf.Rect{
				W: float64(size[j]),
				H: 15,
			}, data[i][j], alignCenter)
			x += size[j]
			ye = y + i*15
		}
	}
	return
}

// PdfToReader 传入pdf文件，返回文件流
func PdfToReader(pdf *gopdf.GoPdf) (*bytes.Reader, error) {
	// 保存文件为临时文件
	name := strconv.FormatInt(time.Now().Unix(), 10)
	err := pdf.WritePdf(name + ".pdf")
	if err != nil {
		return nil, err
	}
	// 读取临时文件
	f, err := os.Open(name + ".pdf")
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Println(err)
		}
	}(f)
	// 删除临时文件
	defer func() {
		err := os.Remove(name + ".pdf")
		if err != nil {
			log.Println(err)
		}
	}()
	// 读取文件内容
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// CreateWord 通过标题、表头和内容生成word文档
func CreateWord(d *document.Document, title string, header []string, content [][]string) error {
	// 创建文档
	para := d.AddParagraph()
	para.Properties().SetAlignment(wml.ST_JcCenter)
	run := para.AddRun()
	run.Properties().SetSize(15)
	run.Properties().SetBold(true)
	//run.Properties().SetItalic(true)
	// 设置标题
	if title != "" {
		run.AddText(title)
	}
	d.AddParagraph()

	// 设定表格样式
	ts := d.Styles.AddStyle("MyTableStyle", wml.ST_StyleTypeTable, false)
	tp := ts.TableProperties()
	tp.SetRowBandSize(1)
	tp.SetColumnBandSize(1)
	tp.SetTableIndent(measurement.Zero)

	// first row bold
	s := ts.TableConditionalFormatting(wml.ST_TblStyleOverrideTypeFirstRow)
	s.RunProperties().SetBold(true)

	// 设置表头
	table := d.AddTable()
	table.Properties().SetLayout(wml.ST_TblLayoutTypeAutofit)
	table.Properties().SetWidthAuto()
	table.Properties().SetAlignment(wml.ST_JcTableCenter)
	table.Properties().SetStyle("MyTableStyle")
	table.Properties().SetCellSpacingAuto()
	//look := table.Properties().TableLook()
	//look.SetFirstColumn(true)
	//look.SetFirstRow(true)
	//look.SetHorizontalBanding(true)
	borders := table.Properties().Borders()
	borders.SetInsideHorizontal(wml.ST_BorderSingle, color.Black, measurement.Zero)
	row := table.AddRow()
	for _, v := range header {
		cell := row.AddCell()
		cellPara := cell.AddParagraph()
		cell.Properties().SetWidthAuto()
		cell.Properties().SetShading(wml.ST_ShdSolid, color.LightGray, color.Auto)
		cellPara.Properties().SetAlignment(wml.ST_JcLeft)
		cellrun := cellPara.AddRun()
		cellrun.Properties().SetSize(10)
		cellrun.AddText(v)
	}

	// 设置内容
	for _, v := range content {
		row := table.AddRow()
		for _, v2 := range v {
			cell := row.AddCell()
			// 通过文本的长度设置单元格宽度
			cell.Properties().SetWidthAuto()
			cellPara := cell.AddParagraph()
			cellPara.Properties().SetAlignment(wml.ST_JcLeft)
			cellrun := cellPara.AddRun()
			cellrun.Properties().SetSize(8)
			cellrun.AddText(v2)
		}
	}
	// 添加两个空行
	d.AddParagraph()
	d.AddParagraph()

	err := d.Validate()
	if err != nil {
		return err
	}

	return nil
}

// DocumentToReader document.Document 转为bytes.Reader
func DocumentToReader(d *document.Document) (*bytes.Reader, error) {
	// 保存文件为临时文件
	name := strconv.FormatInt(time.Now().Unix(), 10)
	err := d.SaveToFile(name + ".docx")
	if err != nil {
		return nil, err
	}
	// 读取临时文件
	f, err := os.Open(name + ".docx")
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Println(err)
		}
	}(f)
	// 删除临时文件
	defer func() {
		err := os.Remove(name + ".docx")
		if err != nil {
			log.Println(err)
		}
	}()
	// 读取文件内容
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// ExportCsv 通过标题、表头和内容生成csv bytes.Reader
func ExportCsv(header1, header2 []string, content1, content2 [][]string) (*bytes.Reader, error) {
	// 创建文档
	buf := new(bytes.Buffer)
	w := csv.NewWriter(buf)

	// 设置文字格式utf-8
	w.UseCRLF = true
	buf.WriteString("\xEF\xBB\xBF") // 写入UTF-8 BOM
	// 设置标题
	err := w.Write([]string{"统计数据"})
	if err != nil {
		return nil, err
	}
	// 设置表头
	err = w.Write(header1)
	if err != nil {
		return nil, err
	}
	// 设置内容
	for _, v := range content1 {
		err := w.Write(v)
		if err != nil {
			return nil, err
		}
	}

	// 设置标题
	err = w.Write([]string{"详细数据"})
	if err != nil {
		return nil, err
	}
	// 设置表头
	err = w.Write(header2)
	if err != nil {
		return nil, err
	}
	// 中文乱码
	w.UseCRLF = true
	buf.WriteString("\xEF\xBB\xBF") // 写入UTF-8 BOM
	// 设置内容
	for _, v := range content2 {
		err := w.Write(v)
		if err != nil {
			return nil, err
		}
	}

	w.Flush()
	return bytes.NewReader(buf.Bytes()), nil
}

// Export2Csv 通过表头和内容生成csv bytes.Reader
func Export2Csv(header []string, content [][]string) (*bytes.Reader, error) {
	// 创建文档
	buf := new(bytes.Buffer)
	w := csv.NewWriter(buf)
	// 设置表头
	err := w.Write(header)
	if err != nil {
		return nil, err
	}
	// 中文乱码
	// 设置文字格式utf-8
	w.UseCRLF = true
	buf.WriteString("\xEF\xBB\xBF") // 写入UTF-8 BOM

	// 设置内容
	for _, v := range content {
		err := w.Write(v)
		if err != nil {
			return nil, err
		}
	}

	w.Flush()
	return bytes.NewReader(buf.Bytes()), nil
}

// ExportHtml 通过标题、表头和内容生成html bytes.Reader
func ExportHtml(header1, header2 []string, content1, content2 [][]string) (data *bytes.Reader, err error) {
	htm := gohtml.NewHtml()
	htm.Html().Lang("zh-CN")
	htm.Meta().Charset("utf-8")
	htm.Meta().Http_equiv("X-UA-Compatible").Content("IE=edge")
	htm.Meta().Content("width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no").Name("viewport")
	htm.Head().Title().Text("报表")
	htmlDiv := htm.Body().Tag("div")
	//统计数据
	htmlDiv.Tag("div").Attr("style", "text-align: center").Text("统计数据")
	htmlTableView := htmlDiv.Tag("table").Attr("style", "width: 35%;transform: translateX(93%);clear: both;background-color: transparent;margin-top: 6px !important;margin-bottom: 6px !important;border-collapse: separate !important;")
	htmlTrView := htmlTableView.Thead().Attr("style", "background: #FAFAFA; display: table-header-group;border-radius: 4px 4px 0px 0px;font-family: NotoSansHans-Medium;font-size: 14px; color: rgba(0, 0, 0, 0.85);line-height: 22px;border-color: inherit;vertical-align: middle;text-align: left;").Tr()
	for _, v := range header1 {
		htmlTrView.Th().Text(v)
	}
	htmlTbodyView := htmlTableView.Tag("tbody").Attr("style", "font-family: NotoSansHans-Regular;font-size: 14px;color: rgba(0, 0, 0, 0.65);line-height: 22px;")
	for _, v := range content1 {
		droip := htmlTbodyView.Tr()
		for i := 0; i < len(v); i++ {
			droip.Td().Text(v[i])
		}
	}
	//详细数据
	htmlDiv.Tag("div").Attr("style", "text-align: center;margin-top: 35px;").Text("详细数据")
	htmlTable := htmlDiv.Tag("table").Attr("style", "width: 65%;transform: translateX(30%);clear: both;background-color: transparent;margin-top: 6px !important;margin-bottom: 6px !important;border-collapse: separate !important;")
	htmlTr := htmlTable.Thead().Attr("style", "background: #FAFAFA; display: table-header-group;border-radius: 4px 4px 0px 0px;font-family: NotoSansHans-Medium;font-size: 14px; color: rgba(0, 0, 0, 0.85);line-height: 22px;border-color: inherit;vertical-align: middle;text-align: left;").Tag("tr").Tag("tr")
	for _, v := range header2 {
		htmlTr.Th().Text(v)
	}
	htmlTbody := htmlTable.Tag("tbody").Attr("style", "font-family: NotoSansHans-Regular;font-size: 14px;color: rgba(0, 0, 0, 0.65);line-height: 22px;")
	for _, v := range content2 {
		htmlB := htmlTbody.Tr()
		for i := 0; i < len(v); i++ {
			htmlB.Td().Text(v[i])
		}
	}
	//// 保存为html文件
	//err = os.WriteFile("temp.html", []byte(htm.String()), 0666)

	buf := new(bytes.Buffer)
	htmldata := []byte(htm.String())
	err = binary.Write(buf, binary.BigEndian, &htmldata)
	data = bytes.NewReader(buf.Bytes())
	return
}

func Export2Html(header []string, content [][]string) (data *bytes.Reader, err error) {
	htm := gohtml.NewHtml()
	htm.Html().Lang("zh-CN")
	htm.Meta().Charset("utf-8")
	htm.Meta().Http_equiv("X-UA-Compatible").Content("IE=edge")
	htm.Meta().Content("width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no").Name("viewport")
	htm.Head().Title().Text("报表")
	htmlDiv := htm.Body().Tag("div")
	//统计数据
	htmlTableView := htmlDiv.Tag("table").Attr("style", "width: 35%;transform: translateX(93%);clear: both;background-color: transparent;margin-top: 6px !important;margin-bottom: 6px !important;border-collapse: separate !important;")
	htmlTrView := htmlTableView.Thead().Attr("style", "background: #FAFAFA; display: table-header-group;border-radius: 4px 4px 0px 0px;font-family: NotoSansHans-Medium;font-size: 14px; color: rgba(0, 0, 0, 0.85);line-height: 22px;border-color: inherit;vertical-align: middle;text-align: left;").Tr()
	for _, v := range header {
		htmlTrView.Th().Text(v)
	}
	htmlTbodyView := htmlTableView.Tag("tbody").Attr("style", "font-family: NotoSansHans-Regular;font-size: 14px;color: rgba(0, 0, 0, 0.65);line-height: 22px;")
	for _, v := range content {
		droip := htmlTbodyView.Tr()
		for i := 0; i < len(v); i++ {
			droip.Td().Text(v[i])
		}
	}
	buf := new(bytes.Buffer)
	htmldata := []byte(htm.String())
	err = binary.Write(buf, binary.BigEndian, &htmldata)
	data = bytes.NewReader(buf.Bytes())
	return
}

// AddWeekTime 为周时间段加一周
func AddWeekTime(t string) string {
	// 将周时间字符串转换为时间类型
	// 将时间字符串转换加一周
	// 截取星期字段
	week := t[5:]
	// 加一周
	weekInt, err := strconv.Atoi(week)
	if err != nil {
		return ""
	}
	weekInt = weekInt + 1
	weekStr := strconv.Itoa(weekInt)
	// 截取年份字段
	year := t[:4]
	// 计算下周时间是否跨年
	if weekInt > 52 {
		yearInt, err := strconv.Atoi(year)
		if err != nil {
			return ""
		}
		yearInt = yearInt + 1
		year = strconv.Itoa(yearInt)
		weekStr = "01"
	}
	// 拼接时间字符串
	return year + "-" + weekStr
}

// AddWeekTime2 2006-01-02加一周
func AddWeekTime2(t string) string {
	// 2006-01-02切出年月日
	year := t[:4]
	month := t[5:7]
	day := t[8:]
	// 加一周
	yearInt, err := strconv.Atoi(year)
	if err != nil {
		return ""
	}
	monthInt, err := strconv.Atoi(month)
	if err != nil {
		return ""
	}
	dayInt, err := strconv.Atoi(day)
	if err != nil {
		return ""
	}
	// 加一周
	dayInt = dayInt + 7
	// 判断是否跨月
	if dayInt > 31 {
		monthInt = monthInt + 1
		dayInt = dayInt - 31
	}
	// 判断是否跨年
	if monthInt > 12 {
		yearInt = yearInt + 1
		monthInt = monthInt - 12
	}
	// 拼接时间字符串
	return strconv.Itoa(yearInt) + "-" + strconv.Itoa(monthInt) + "-" + strconv.Itoa(dayInt)
	// 截取星期字段
}

// GetQueryType 判断查询类型 1:日 2:周 3:月 4:小时
func GetQueryType(startTime, endTime string) int {
	// 判断查询类型 1:日 2:周 3:月 4:小时
	// 1:日 如果时间差等于1周，返回1
	// 2:周 如果时间差等于1月，返回2
	// 3:月 如果时间差大于1月，返回3
	// 4:小时 如果时间差为1天，返回4
	// 1:日
	fmt.Println("startTime:", startTime)
	fmt.Println("endTime:", endTime)
	if GetDays(startTime, endTime) > 0 && GetDays(startTime, endTime) <= 7 {
		return 1
	}
	// 2:周
	if GetDays(startTime, endTime) >= 7 && GetDays(startTime, endTime) <= 31 {
		return 2
	}
	// 3:月
	if GetDays(startTime, endTime) >= 30 {
		return 3
	}
	// 4:小时
	if GetDays(startTime, endTime) == 0 {
		return 4
	}
	return 0
}

// GetDays 求两个时间段相差天数
func GetDays(startTime, endTime string) int {
	year := startTime[:4]
	month := startTime[5:7]
	day := startTime[8:10]
	yearInt, err := strconv.Atoi(year)
	if err != nil {
		return 0
	}
	monthInt, err := strconv.Atoi(month)
	if err != nil {
		return 0
	}
	dayInt, err := strconv.Atoi(day)
	if err != nil {
		return 0
	}
	startTimeInt := time.Date(yearInt, time.Month(monthInt), dayInt, 0, 0, 0, 0, time.Local).Unix()
	year = endTime[:4]
	month = endTime[5:7]
	day = endTime[8:10]
	yearInt, err = strconv.Atoi(year)
	if err != nil {
		return 0
	}
	monthInt, err = strconv.Atoi(month)
	if err != nil {
		return 0
	}
	dayInt, err = strconv.Atoi(day)
	if err != nil {
		return 0
	}
	endTimeInt := time.Date(yearInt, time.Month(monthInt), dayInt, 0, 0, 0, 0, time.Local).Unix()
	return int((endTimeInt - startTimeInt) / 86400)
}

func AddDayTime(time string) string {
	// 将时间字符串转换为时间类型
	// 将时间字符串转换加一天
	// 截取年月日字段
	year := time[:4]
	month := time[5:7]
	day := time[8:10]
	// 加一天
	dayInt, err := strconv.Atoi(day)
	if err != nil {
		return ""
	}
	dayInt = dayInt + 1
	dayStr := strconv.Itoa(dayInt)
	// 计算下一天是否跨月
	if dayInt > 31 {
		monthInt, err := strconv.Atoi(month)
		if err != nil {
			return ""
		}
		monthInt = monthInt + 1
		month = strconv.Itoa(monthInt)
		dayStr = "01"
		// 计算下一天是否跨年
		if monthInt > 12 {
			yearInt, err := strconv.Atoi(year)
			if err != nil {
				return ""
			}
			yearInt = yearInt + 1
			year = strconv.Itoa(yearInt)
			month = "01"
		}
	}
	// 拼接时间字符串
	return year + "-" + month + "-" + dayStr
}

func AddMonthTime(time string) string {
	// 将月时间字符串转换为时间类型
	// 将时间字符串转换加一月
	// 截取月字段
	month := time[5:7]
	// 加一月
	monthInt, err := strconv.Atoi(month)
	if err != nil {
		return ""
	}
	monthInt = monthInt + 1
	monthStr := strconv.Itoa(monthInt)
	// 截取年份字段
	year := time[:4]
	// 计算下月时间是否跨年
	if monthInt > 12 {
		yearInt, err := strconv.Atoi(year)
		if err != nil {
			return ""
		}
		yearInt = yearInt + 1
		year = strconv.Itoa(yearInt)
		monthStr = "01"
	}
	// 拼接时间字符串
	return year + "-" + monthStr
}

func AddHourTime(time string) string {
	// 将月时间字符串转换为时间类型
	// 将时间字符串转换加一小时
	// 截取小时字段
	hour := time[11:13]
	// 加一小时
	hourInt, err := strconv.Atoi(hour)
	if err != nil {
		return ""
	}
	hourInt = hourInt + 1
	hourStr := strconv.Itoa(hourInt)
	// 截取年月日字段
	year := time[:4]
	month := time[5:7]
	day := time[8:10]
	// 计算下小时时间是否跨天
	if hourInt > 24 {
		dayInt, err := strconv.Atoi(day)
		if err != nil {
			return ""
		}
		dayInt = dayInt + 1
		day = strconv.Itoa(dayInt)
		hourStr = "00"
		// 计算下小时时间是否跨月
		if dayInt > 31 {
			monthInt, err := strconv.Atoi(month)
			if err != nil {
				return ""
			}
			monthInt = monthInt + 1
			month = strconv.Itoa(monthInt)
			day = "01"
			// 计算下小时时间是否跨年
			if monthInt > 12 {
				yearInt, err := strconv.Atoi(year)
				if err != nil {
					return ""
				}
				yearInt = yearInt + 1
				year = strconv.Itoa(yearInt)
				month = "01"
			}
		}
	}
	// 拼接时间字符串
	return year + "-" + month + "-" + day + " " + hourStr
}

// StrToTime 将时间字符串转换为2006-01-02 15:04:05格式
func StrToTime(ti string) string {
	// 将时间字符串转换为时间类型
	timeType := StringToJSONTime(ti)

	// 将时间类型转换为字符串
	return timeType.Format("2006-01-02 15:04:05")
}

// 判断导出数据时间间隔类型
func JudgeTimeGapType(startTime, endTime string) (searchType, newStart, newEnd string) {
	searchType = "%Y-%m-%d"
	newStart = startTime
	newEnd = endTime
	if startTime != "" && endTime != "" {
		//startT, err := time.Parse("2006-01-02 15:04:05", startTime)
		//if err != nil {
		//	return
		//}
		//endT, err := time.Parse("2006-01-02 15:04:05", endTime)
		//if err != nil {
		//	return
		//}
		//t1_ := startT.Add(time.Duration(endT.Sub(startT).Milliseconds()%86400000) * time.Millisecond)
		//day := int(endT.Sub(startT).Hours() / 24)
		//// 计算在t1+两个时间的余数之后天数是否有变化
		//if t1_.Day() != startT.Day() {
		//	day += 1
		//}

		startT, _ := time.Parse("2006-01-02", startTime)
		endT, _ := time.Parse("2006-01-02", endTime)
		day := startT.Sub(endT).Hours() / 24

		if day == 0 {
			searchType = "%Y-%m-%d %H"
			newStart = startT.Format("2006-01-02 15")
			newEnd = endT.AddDate(0, 0, 1).Format("2006-01-02 15")
		} else if day <= 31 {
			searchType = "%Y-%m-%d"
			newStart = startT.Format("2006-01-02")
			newEnd = endT.Format("2006-01-02")
		} else {
			searchType = "%Y-%m"
			newStart = startT.Format("2006-01")
			newEnd = endT.Format("2006-01")
		}
	}
	return
}

// 判断接入的时间类型,返回一个时间区间
func JudgeTimeType(Time string) (start string, end string) {
	// 时间精度到天 + 1天
	if len(Time) == 10 {
		start = Time + " 00:00:00"
		end = Time + " 23:59:59"
	}
	// 时间精度到小时 + 1小时
	if len(Time) == 13 {
		start = Time + ":00:00"
		end = Time + ":59:59"
	}
	// 时间精度到周 + 1周
	if len(Time) == 8 {
		year := Time[:4]
		week := Time[5:]
		weekInt, err := strconv.Atoi(week)
		if err != nil {
			return "", ""
		}
		weekInt = weekInt + 1
		weekStr := strconv.Itoa(weekInt)
		end = year + "-W" + weekStr
		start = Time
	}
	// 时间精度到月 + 1月
	if len(Time) == 7 {
		start = Time + "-01 00:00:00"
		end = Time + "-31 23:59:59"
	}
	return
}

func String2Time(str string) time.Time {
	if len(str) == 0 {
		return time.Time{}
	}
	if len(str) == 10 {
		str = str + " 00:00:00"
	}
	if len(str) == 13 {
		str = str + ":00:00"
	}
	if len(str) == 16 {
		str = str + ":00"
	}
	t, err := time.Parse("2006-01-02 15:04:05", str)
	if err != nil {
		return time.Time{}
	}
	return t
}
