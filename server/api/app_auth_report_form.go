package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"
	"tkbastion/pkg/log"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"baliance.com/gooxml/document"
	"github.com/signintech/gopdf"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func AppAuthPagingEndpoint(c echo.Context) error {
	auto := c.QueryParam("auto")
	appName := c.QueryParam("appName")
	programName := c.QueryParam("programName")
	userName := c.QueryParam("userName")

	items, err := appAuthReportFormRepository.Find(context.TODO())
	if nil != err {
		if gorm.ErrRecordNotFound != err {
			log.Errorf("DB Error: %v", err.Error())
		}
		return SuccessWithOperate(c, "", nil)
	}

	for i := range items {
		var appAuthReportFrom model.ApplicationAuthReportForm

		appAuthReportFrom.ID = items[i].ID

		appAuthReportFrom.OperateAuthId = items[i].OperateAuthId
		operateAuth, err := operateAuthRepository.FindById(items[i].OperateAuthId)
		if nil != err {
			log.Errorf("DB Error: %v", err.Error())
			continue
		}
		appAuthReportFrom.OperateAuthName = operateAuth.Name

		appAuthReportFrom.ApplicationId = items[i].ApplicationId
		application, err := newApplicationRepository.FindById(context.TODO(), items[i].ApplicationId)
		if nil != err {
			log.Errorf("DB Error: %v", err.Error())
			continue
		}
		appAuthReportFrom.AppSerName = application.AppSerName
		appAuthReportFrom.ProgramName = application.ProgramName
		appAuthReportFrom.AppName = application.Name

		appAuthReportFrom.UserId = items[i].UserId
		user, err := userNewRepository.FindById(items[i].UserId)
		if nil != err {
			log.Errorf("DB Error: %v", err.Error())
			continue
		}
		appAuthReportFrom.Username = user.Username
		appAuthReportFrom.Nickname = user.Nickname

		err = appAuthReportFormRepository.UpdateById(context.TODO(), items[i].ID, &appAuthReportFrom)
		if nil != err {
			log.Errorf("DB Error: %v", err.Error())
			continue
		}
	}

	appAuthReportForms, err := appAuthReportFormRepository.FindWithCondition(context.TODO(), auto, appName, programName, userName)
	if nil != err {
		if gorm.ErrRecordNotFound != err {
			log.Errorf("DB Error: %v", err.Error())
		}
		return SuccessWithOperate(c, "", nil)
	}

	return SuccessWithOperate(c, "", appAuthReportForms)
}

func AppAuthExportEndpoint(c echo.Context) error {
	items, err := appAuthReportFormRepository.FindExportData(context.TODO())
	if nil != err {
		if gorm.ErrRecordNotFound != err {
			log.Errorf("DB Error: %v", err.Error())
		}
		return SuccessWithOperate(c, "", nil)
	}

	exportType := c.QueryParam("type")
	if exportType == "" {
		return Success(c, nil)
	}

	var data = make([][]string, len(items))
	for i := range items {
		data[i] = utils.Struct2StrArr(items[i])
	}

	var handler = []string{"应用发布服务器", "应用名称", "应用程序", "运维用户", "真实姓名", "策略名称"}
	var fileName = "应用授权报表"
	var fileReader *bytes.Reader

	switch exportType {
	case "pdf":
		pdf := gopdf.GoPdf{}
		pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
		pdf.AddPage()
		size := []int{110, 110, 60, 60, 60, 60}
		err, _, _ := utils.PdfExport(&pdf, "", handler, size, data, 0, 0)
		if err != nil {
			log.Errorf("导出pdf失败: %v", err)
		}

		fileReader, err = utils.PdfToReader(&pdf)
		if err != nil {
			return Fail(c, 500, "导出失败")
		}
		fileName = fileName + "-" + time.Now().Format("20060102150405") + ".pdf"
	case "csv":
		fileReader, err = utils.Export2Csv(handler, data)
		if err != nil {
			return Fail(c, 500, "导出失败")
		}
		fileName = fileName + "-" + time.Now().Format("20060102150405") + ".csv"
	case "word":
		d := document.New()
		err = utils.CreateWord(d, "", handler, data)
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
		file, err := utils.Export2Html(handler, data)
		if err != nil {
			return FailWithDataOperate(c, 500, "导出失败", "", err)
		}
		fileReader = file
		fileName = fileName + "-" + time.Now().Format("20060102150405") + ".html"
	default:
		return Fail(c, 500, "导出失败")
	}

	// 返回文件流
	fmt.Println("返回文件流")
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, fileReader)
}
