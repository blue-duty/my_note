package api

import (
	"github.com/labstack/echo/v4"
	"path"
	"strconv"
	"strings"
	"tkbastion/pkg/constant"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/model"
	"tkbastion/server/utils"
)

// 查询定期策略

func FindRegularReportEndpoint(c echo.Context) (err error) {
	auto := c.QueryParam("auto")
	name := c.QueryParam("name")
	description := c.QueryParam("description")
	var regularReport []model.RegularReport
	regularReport, err = regularReportRepository.FindByConditions(auto, name, description)
	if err != nil {
		log.Errorf("Find regular report failed: %v", err)
		log.Errorf("定期报表-定期策略: 查询失败")
	}
	regularReportForPaging := make([]model.RegularReportForPaging, len(regularReport))
	for i, v := range regularReport {
		reportType := ""
		periodicType := ""
		if *v.IsProtocol {
			reportType += "协议访问统计 \n"
		}
		if *v.IsUser {
			reportType += "用户访问统计 \n"
		}
		if *v.IsLogin {
			reportType += "尝试登录统计 \n"
		}
		if *v.IsAsset {
			reportType += "资产运维 \n"
		}
		if *v.IsSession {
			reportType += "会话时长 \n"
		}
		if *v.IsCommand {
			reportType += "命令统计 \n"
		}
		if *v.IsAlarm {
			reportType += "告警报表 \n"
		}
		if v.PeriodicType == "week" {
			periodicType = "按周 每周" + strconv.Itoa(int(v.Periodic))
		} else if v.PeriodicType == "month" {
			periodicType = "按月 每月" + strconv.Itoa(int(v.Periodic)) + "号"
		} else {
			periodicType = "按天"
		}
		regularReportForPaging[i] = model.RegularReportForPaging{
			ID:           v.ID,
			Name:         v.Name,
			Description:  v.Description,
			ReportType:   reportType,
			PeriodicType: periodicType,
			Periodic:     v.Periodic,
			IsProtocol:   *v.IsProtocol,
			IsUser:       *v.IsUser,
			IsLogin:      *v.IsLogin,
			IsAsset:      *v.IsAsset,
			IsSession:    *v.IsSession,
			IsCommand:    *v.IsCommand,
			IsAlarm:      *v.IsAlarm,
		}
	}
	return Success(c, regularReportForPaging)
}

// 创建定期策略

func CreateRegularReportEndpoint(c echo.Context) (err error) {
	var regularReport model.RegularReport
	err = c.Bind(&regularReport)
	if err != nil {
		log.Errorf("Create regular report failed: %v", err)
		log.Errorf("定期报表-定期策略: 数据绑定失败")
		return FailWithDataOperate(c, 500, "创建失败", "", nil)
	}
	regularReport.ID = utils.UUID()
	err = regularReportRepository.Create(&regularReport)
	if err != nil {
		log.Errorf("Create regular report failed: %v", err)
		log.Errorf("定期报表-定期策略: 创建失败")
	}
	err = newJobService.RunRegularReport(regularReport.ID)
	if err != nil {
		log.Errorf("Run regular report failed: %v", err)
		log.Errorf("定期报表-定期策略: 定期执行失败")
	}
	return SuccessWithOperate(c, "定期策略-新建: 策略名称["+regularReport.Name+"]", nil)
}

// 更新定期策略

func UpdateRegularReportEndpoint(c echo.Context) (err error) {
	id := c.Param("id")
	var regularReport model.RegularReport
	err = c.Bind(&regularReport)
	if err != nil {
		log.Errorf("Update regular report failed: %v", err)
		log.Errorf("定期报表-定期策略: 数据绑定失败")
		return FailWithDataOperate(c, 500, "更新失败", "", nil)
	}

	// 更新空描述信息
	if regularReport.Description == "" {
		regularReport.Description = " "
	}

	regularReportOld, err := regularReportRepository.FindById(id)
	if err != nil {
		log.Errorf("Update regular report failed: %v", err)
		log.Errorf("定期报表-定期策略: 更新失败:id=%v", id)
		return FailWithDataOperate(c, 500, "更新失败", "", nil)
	}
	err = regularReportRepository.UpdateById(&regularReport, regularReport.ID)
	if err != nil {
		log.Errorf("Update regular report failed: %v", err)
		log.Errorf("定期报表-定期策略: 更新失败")
	}

	err = global.SCHEDULER.RemoveByTag(regularReport.ID)
	if err != nil {
		log.Errorf("Remove: %v Error: %v", regularReport.ID, err.Error())
	}
	err = newJobService.RunRegularReport(regularReport.ID)
	if err != nil {
		log.Errorf("Run regular report failed: %v", err)
		log.Errorf("定期报表-定期策略: 定期执行失败")
	}
	return SuccessWithOperate(c, "定期策略-修改: 策略名称["+regularReportOld.Name+"]", nil)
}

// 删除定期策略

func DeleteRegularReportEndpoint(c echo.Context) (err error) {
	id := c.Param("id")
	split := strings.Split(id, ",")
	nameSuccess := ""
	count := 0
	for _, v := range split {
		item, err := regularReportRepository.FindById(v)
		if err != nil {
			log.Errorf("Delete regular report failed: %v", err)
			log.Errorf("定期报表-定期策略: 删除失败:id=%v", v)
			continue
		}
		err = regularReportRepository.DeleteById(v)
		if err != nil {
			log.Errorf("Delete regular report failed: %v", err)
			log.Errorf("定期报表-定期策略: 删除失败:id=%v", v)
			continue
		}
		err = global.SCHEDULER.RemoveByTag(v)
		if err != nil {
			log.Errorf("Remove: %v Error: %v", v, err.Error())
			log.Errorf("定期报表-定期策略: 删除自动任务:id=%v", v)
			continue
		}
		nameSuccess += item.Name + ","
		count++
	}
	if len(nameSuccess) > 0 {
		nameSuccess = nameSuccess[:len(nameSuccess)-1]
	}
	return SuccessWithOperate(c, "定期策略-删除: [名称: "+nameSuccess+",数量: "+strconv.Itoa(count)+"]", nil)
}

// 查询定期策略

func FindRegularReportLogEndpoint(c echo.Context) (err error) {
	auto := c.QueryParam("auto")
	name := c.QueryParam("name")
	executeTime := c.QueryParam("executeTime")
	var regularReportLog []model.RegularReportLog
	regularReportLog, err = regularReportRepository.FindRegularReportLogByConditions(auto, name, executeTime)
	if err != nil {
		log.Errorf("Find regular report log failed: %v", err)
		log.Errorf("定期报表-定期策略日志: 查询失败")
	}
	regularReportLogForPaging := make([]model.RegularReportLogForPaging, len(regularReportLog))
	for i, v := range regularReportLog {
		regularReportLogForPaging[i] = model.RegularReportLogForPaging{
			ID:           v.ID,
			Name:         v.Name,
			ExecuteTime:  v.ExecuteTime.Time.Format("2006-01-02"),
			PeriodicType: v.PeriodicType,
			ReportType:   v.ReportType,
			FileName:     v.FileName,
		}
	}
	return Success(c, regularReportLogForPaging)
}

// 下载定期策略日志

func DownloadRegularReportLogEndpoint(c echo.Context) error {
	fileName := c.QueryParam("fileName")
	reportType := c.QueryParam("reportType")
	// 1.Protocol 2.User 3.Login 4.Asset 5.Session 6.Command 7.Alarm
	switch reportType {
	case "protocol":
		if ok := utils.FileExists(path.Join(constant.RegularReportProtocolPath, fileName)); !ok {
			log.Errorf("Download regular report log failed: %v", "文件不存在")
			return FailWithDataOperate(c, 500, "下载失败", "", nil)
		}
		return c.Attachment(path.Join(constant.RegularReportProtocolPath, fileName), fileName)
	case "user":
		if ok := utils.FileExists(path.Join(constant.RegularReportUserPath, fileName)); !ok {
			log.Errorf("Download regular report log failed: %v", "文件不存在")
			return FailWithDataOperate(c, 500, "下载失败", "", nil)
		}
		return c.Attachment(path.Join(constant.RegularReportUserPath, fileName), fileName)
	case "login":
		if ok := utils.FileExists(path.Join(constant.RegularReportLoginPath, fileName)); !ok {
			log.Errorf("Download regular report log failed: %v", "文件不存在")
			return FailWithDataOperate(c, 500, "下载失败", "", nil)
		}
		return c.Attachment(path.Join(constant.RegularReportLoginPath, fileName), fileName)
	case "asset":
		if ok := utils.FileExists(path.Join(constant.RegularReportAssetPath, fileName)); !ok {
			log.Errorf("Download regular report log failed: %v", "文件不存在")
			return FailWithDataOperate(c, 500, "下载失败", "", nil)
		}
		return c.Attachment(path.Join(constant.RegularReportAssetPath, fileName), fileName)
	case "session":
		if ok := utils.FileExists(path.Join(constant.RegularReportSessionPath, fileName)); !ok {
			log.Errorf("Download regular report log failed: %v", "文件不存在")
			return FailWithDataOperate(c, 500, "下载失败", "", nil)
		}
		return c.Attachment(path.Join(constant.RegularReportSessionPath, fileName), fileName)
	case "command":
		if ok := utils.FileExists(path.Join(constant.RegularReportCommandPath, fileName)); !ok {
			log.Errorf("Download regular report log failed: %v", "文件不存在")
			return FailWithDataOperate(c, 500, "下载失败", "", nil)
		}
		return c.Attachment(path.Join(constant.RegularReportCommandPath, fileName), fileName)
	case "alarm":
		if ok := utils.FileExists(path.Join(constant.RegularReportAlarmPath, fileName)); !ok {
			log.Errorf("Download regular report log failed: %v", "文件不存在")
			return FailWithDataOperate(c, 500, "下载失败", "", nil)
		}
		return c.Attachment(path.Join(constant.RegularReportAlarmPath, fileName), fileName)
	default:
		log.Errorf("Download regular report log failed: %v", "报表类型错误")
		return FailWithDataOperate(c, 500, "下载失败", "", nil)
	}
}
