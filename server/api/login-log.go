package api

import (
	"bytes"
	"net/http"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
)

func LoginLogPagingEndpoint(c echo.Context) error {
	auto := c.QueryParam("auto")
	ipAddress := c.QueryParam("ipAddress")
	user := c.QueryParam("user")
	name := c.QueryParam("name")
	loginType := c.QueryParam("loginType")
	loginResult := c.QueryParam("loginResult")

	items, err := loginLogRepository.Find(auto, ipAddress, user, name, loginType, loginResult)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	return SuccessWithOperate(c, "", items)
}

func LoginLogExportEndpoint(c echo.Context) error {
	auto := c.QueryParam("auto")
	ipAddress := c.QueryParam("ipAddress")
	user := c.QueryParam("user")
	name := c.QueryParam("name")
	loginType := c.QueryParam("loginType")
	loginResult := c.QueryParam("loginResult")

	account, isSuccess := GetCurrentAccountNew(c)
	if !isSuccess {
		log.Error("GetCurrentAccountNew Error")
		return FailWithDataOperate(c, 500, "导出失败", "", nil)
	}

	items, err := loginLogRepository.Find(auto, ipAddress, user, name, loginType, loginResult)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "", err)
	}

	var forExport = make([][]string, 0, len(items))
	for i := 0; i < len(items); i++ {
		forExport = append(forExport, []string{
			items[i].LoginTime.Format("2006-01-02 15:04:05"),
			items[i].ClientIP,
			items[i].Username,
			items[i].Nickname,
			items[i].Protocol,
			items[i].LoginType,
			items[i].LoginResult,
			items[i].Description,
		})
	}

	exportFileName := "登录日志.xlsx"
	header := []string{"登录时间", "来源地址", "用户名", "姓名", "协议", "登录方式", "登录结果", "描述"}

	file, err := utils.CreateExcelFile(exportFileName, header, forExport)
	if err != nil {
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}

	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "",
		Created:         utils.NowJsonTime(),
		Users:           account.Username,
		Names:           account.Nickname,
		LogContents:     "登录日志-导出: 导出登录日志数据文件成功",
		Result:          "成功",
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

	//设置请求头  使用浏览器下载
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+exportFileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
}
