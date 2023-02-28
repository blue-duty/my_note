package api

import (
	"bytes"
	"net/http"
	"strings"

	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
)

func OperateLogPagingEndpoint(c echo.Context) error {
	auto := c.QueryParam("auto")
	ipAddress := c.QueryParam("ipAddress")
	user := c.QueryParam("user")
	name := c.QueryParam("name")
	functionalModule := c.QueryParam("functionalModule")
	action := c.QueryParam("action")

	items, err := operateLogRepository.Find(auto, ipAddress, user, name, functionalModule, action)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	var operateLogArr []model.OperateForPageNew
	var operateLog model.OperateForPageNew
	for i := range items {
		operateLog.Created = items[i].Created
		operateLog.IP = items[i].IP
		operateLog.Users = items[i].Users
		operateLog.Names = items[i].Names
		operateLog.Result = items[i].Result

		separator1Index := strings.Index(items[i].LogContents, "-")
		operateLog.FunctionalModule = items[i].LogContents[:separator1Index]
		separator2Index := strings.Index(items[i].LogContents, ":")
		if -1 == separator1Index {
			log.Errorf("操作日志内容字段格式错误: %v", items[i].LogContents)
			continue
		}

		if -1 == separator2Index {
			operateLog.Action = items[i].LogContents[separator1Index+1:]
			log.Errorf("操作日志内容字段格式错误, 缺少\":\": %v", items[i].LogContents)
			operateLog.LogContents = "操作成功"
		} else {
			operateLog.Action = items[i].LogContents[separator1Index+1 : separator2Index]
			operateLog.LogContents = items[i].LogContents[separator2Index+2:]
		}

		operateLogArr = append(operateLogArr, operateLog)
	}

	return SuccessWithOperate(c, "", operateLogArr)
}

func OperateLogExportEndpoint(c echo.Context) error {
	auto := c.QueryParam("auto")
	ipAddress := c.QueryParam("ipAddress")
	user := c.QueryParam("user")
	name := c.QueryParam("name")
	functionalModule := c.QueryParam("functionalModule")
	action := c.QueryParam("action")

	account, isSuccess := GetCurrentAccountNew(c)
	if !isSuccess {
		log.Error("GetCurrentAccountNew Error")
		return FailWithDataOperate(c, 500, "导出失败", "", nil)
	}

	items, err := operateLogRepository.Find(auto, ipAddress, user, name, functionalModule, action)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "", err)
	}

	var operateLogArr []model.OperateForPageNew
	var operateLog model.OperateForPageNew
	for i := range items {
		operateLog.Created = items[i].Created
		operateLog.IP = items[i].IP
		operateLog.Users = items[i].Users
		operateLog.Names = items[i].Names
		operateLog.Result = items[i].Result

		separator1Index := strings.Index(items[i].LogContents, "-")
		operateLog.FunctionalModule = items[i].LogContents[:separator1Index]
		separator2Index := strings.Index(items[i].LogContents, ":")
		if -1 == separator1Index {
			log.Errorf("操作日志内容字段格式错误: %v", items[i].LogContents)
			continue
		}

		if -1 == separator2Index {
			operateLog.Action = items[i].LogContents[separator1Index+1:]
			log.Errorf("操作日志内容字段格式错误, 缺少\":\": %v", items[i].LogContents)
			operateLog.LogContents = "操作成功"
		} else {
			operateLog.Action = items[i].LogContents[separator1Index+1 : separator2Index]
			operateLog.LogContents = items[i].LogContents[separator2Index+2:]
		}

		operateLogArr = append(operateLogArr, operateLog)
	}

	var forExport = make([][]string, 0, len(items))
	for i := 0; i < len(operateLogArr); i++ {
		forE := []string{
			items[i].Created.Format("2006-01-02 15:04:05"),
			items[i].IP,
			items[i].Users,
			items[i].Names,
		}
		separator1Index := strings.Index(items[i].LogContents, "-")
		forE = append(forE, items[i].LogContents[:separator1Index])
		separator2Index := strings.Index(items[i].LogContents, ":")
		if -1 == separator2Index {
			forE = append(forE, items[i].LogContents[separator1Index+1:])
			forE = append(forE, "操作成功")
		} else {
			forE = append(forE, items[i].LogContents[separator1Index+1:separator2Index])
			forE = append(forE, items[i].LogContents[separator2Index+2:])
		}
		forE = append(forE, items[i].Result)
		forExport = append(forExport, forE)
	}

	exportFileName := "操作日志.xlsx"
	header := []string{"操作时间", "来源地址", "用户名", "姓名", "功能模块", "动作", "详细内容", "结果"}
	file, err := utils.CreateExcelFile(exportFileName, header, forExport)
	if err != nil {
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}

	opl := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "",
		Created:         utils.NowJsonTime(),
		Users:           account.Username,
		Names:           account.Nickname,
		LogContents:     "操作日志-导出: 导出操作日志数据文件成功",
		Result:          "成功",
	}
	err = global.DBConn.Model(model.OperateLog{}).Create(&opl).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}

	//将数据存入buffer
	var buff bytes.Buffer
	if err = file.Write(&buff); err != nil {
		log.Errorf("Write Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "", err)
	}

	//设置请求头  使用浏览器下载
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+exportFileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
}
