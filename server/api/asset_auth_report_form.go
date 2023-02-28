package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"
	"tkbastion/pkg/log"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"baliance.com/gooxml/document"
	"github.com/signintech/gopdf"

	"gorm.io/gorm"

	"github.com/labstack/echo/v4"
)

func AssetAuthPagingEndpoint(c echo.Context) error {
	auto := c.QueryParam("auto")
	assetName := c.QueryParam("assetName")
	assetAddress := c.QueryParam("assetAddress")
	assetAccount := c.QueryParam("assetAccount")
	userName := c.QueryParam("userName")

	items, err := assetAuthReportFormRepository.Find()
	if nil != err {
		if gorm.ErrRecordNotFound != err {
			log.Errorf("DB Error: %v", err.Error())
		}
		return SuccessWithOperate(c, "", nil)
	}

	for i := range items {
		var assetAuthReportFrom model.AssetAuthReportForm

		assetAuthReportFrom.ID = items[i].ID

		assetAuthReportFrom.AssetAccountId = items[i].AssetAccountId
		assetAccount, err := newAssetRepository.GetPassPortByID(context.TODO(), items[i].AssetAccountId)
		if nil != err {
			log.Errorf("DB Error: %v", err.Error())
			continue
		}
		assetAuthReportFrom.AssetName = assetAccount.AssetName
		assetAuthReportFrom.AssetAddress = assetAccount.Ip
		assetAuthReportFrom.AssetAccount = assetAccount.Passport

		assetAuthReportFrom.UserId = items[i].UserId
		user, err := userNewRepository.FindById(items[i].UserId)
		if nil != err {
			log.Errorf("DB Error: %v", err.Error())
			continue
		}
		assetAuthReportFrom.Username = user.Username
		assetAuthReportFrom.Nickname = user.Nickname

		assetAuthReportFrom.OperateAuthId = items[i].OperateAuthId
		operateAuth, err := operateAuthRepository.FindById(items[i].OperateAuthId)
		if nil != err {
			log.Errorf("DB Error: %v", err.Error())
			continue
		}
		assetAuthReportFrom.OperateAuthName = operateAuth.Name

		err = assetAuthReportFormRepository.UpdateById(items[i].ID, &assetAuthReportFrom)
		if nil != err {
			log.Errorf("DB Error: %v", err.Error())
			continue
		}
	}

	assetAuthReportFormArr, err := assetAuthReportFormRepository.FindWithCondition(auto, assetName, assetAddress, assetAccount, userName)
	if nil != err && gorm.ErrRecordNotFound != err {
		log.Errorf("DB Error: %v", err.Error())
		return SuccessWithOperate(c, "", nil)
	}

	return SuccessWithOperate(c, "", assetAuthReportFormArr)
}

// AssetAuthExportEndpoint 导出
func AssetAuthExportEndpoint(c echo.Context) error {
	items, err := assetAuthReportFormRepository.Find()
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

	var reportForExport = make([]dto.AssetAuthReportForExport, len(items))
	for i := range items {
		reportForExport[i].AssetName = items[i].AssetName
		reportForExport[i].AssetIP = items[i].AssetAddress
		reportForExport[i].Username = items[i].Username
		reportForExport[i].Nickname = items[i].Nickname
		reportForExport[i].Passport = items[i].AssetAccount
		reportForExport[i].AuthName = items[i].OperateAuthName
	}

	var data = make([][]string, len(reportForExport))
	for i := range reportForExport {
		data[i] = utils.Struct2StrArr(reportForExport[i])
	}

	var handler = []string{"设备名称", "设备地址", "设备账号", "用户名", "姓名", "策略名称"}
	var fileName = "资产权限报表"
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
