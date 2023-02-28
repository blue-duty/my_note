package api

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"strconv"
	"tkbastion/pkg/constant"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/validator"
	"tkbastion/server/model"
	"tkbastion/server/repository"
	"tkbastion/server/service"
)

func PolicyconfigGreatEndpoint(c echo.Context) error {
	log.Debugf("--------进入方法")
	var policyConfigDTO model.PolicyConfigDTO
	var err error
	if err = c.Bind(&policyConfigDTO); err != nil {
		log.Errorf("Bind Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}
	//数据校验
	if err := c.Validate(policyConfigDTO); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetErrLogMsg(err, errs.PolicyConfigCreateLog)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}
	return SuccessWithOperate(c, "策略配置:"+policyConfigDTO.ID, policyConfigDTO)
}
func PolicyconfigUpdateEndpoint(c echo.Context) error {
	log.Debugf("--------进入方法")
	//绑定数据
	var err error
	var policyConfigDTO model.PolicyConfigDTO
	if err := c.Bind(&policyConfigDTO); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	//数据校验
	if err := c.Validate(policyConfigDTO); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetErrLogMsg(err, errs.PolicyConfigUpdateLog)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}
	policyConfig := policyConfigDTO.ConvPolicyConfig()
	if err = repository.PolicyConfigDao.Update(policyConfig); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	service.PolicyConfigSrv.Run(policyConfig)
	StatusAll := policyConfigDTO.StatusAll
	StatusSystemDisk := policyConfigDTO.StatusSystemDisk
	StatusDataDisk := policyConfigDTO.StatusDataDisk
	StatusMemory := policyConfigDTO.StatusMemory
	StatusCpu := policyConfigDTO.StatusCpu
	Frequency := strconv.FormatInt(policyConfigDTO.Frequency, 10)
	FrequencyTimeType := policyConfigDTO.FrequencyTimeType
	var S string

	switch FrequencyTimeType {
	case "second":
		S = "秒"
	case "minute":
		S = "分"
	case "hour":
		S = "小时"
	case "day":
		S = "天"
	case "month":
		S = "月"
	}
	var AllSwitch string
	var SystemDisk string
	var DataDisk string
	var Memory string
	var Cpu string
	if StatusAll == 0 {
		AllSwitch = "关闭"
	} else {
		AllSwitch = "开启"
	}
	if StatusSystemDisk == false {
		SystemDisk = "关"
	} else {
		SystemDisk = "开"
	}
	if StatusDataDisk == false {
		DataDisk = "关"
	} else {
		DataDisk = "开"
	}
	if StatusMemory == false {
		Memory = "关"
	} else {
		Memory = "开"
	}
	if StatusCpu == false {
		Cpu = "关"
	} else {
		Cpu = "开"
	}

	//返回数据
	return SuccessWithOperate(c, "安全策略-系统状态:修改配置"+AllSwitch+",系统盘监控:"+SystemDisk+","+"数据盘监控:"+DataDisk+","+
		""+"内存占用监控:"+Memory+","+"CPU负荷监控:"+Cpu+","+"告警邮件发送频率:"+Frequency+S, nil)
}

func PolicyGetEndpoint(c echo.Context) error {
	log.Debugf("--------进入方法")

	item, err := repository.PolicyConfigDao.FindConfig()

	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}

	dto := item.ConvPolicyConfigDTO()
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}

	return SuccessWithOperate(c, "系统状态:", dto)
}
func SendTestMail(c echo.Context) error {
	propertiesMap := propertyRepository.FindAllMap()
	host := propertiesMap[constant.MailHost]
	port := propertiesMap[constant.MailPort]
	username := propertiesMap[constant.MailUsername]
	password := propertiesMap[constant.MailPassword]
	receiver := propertiesMap[constant.MailReceiver]
	if host == "" || host == "-" || port == "" || port == "-" || username == "" || username == "-" || password == "" || password == "-" || receiver == "" || receiver == "-" {
		return FailWithDataOperate(c, 400, "邮箱信息不完整,请先在系统设置页面进行邮箱配置", "安全策略-系统状态:发送测试邮件: "+receiver+"邮箱信息不完整", nil)
	}

	err := mailService.NewSendMail(host, port, username, password, []string{receiver}, "[Tkbastion] 系统状态", "测试邮件")
	if nil != err {
		log.Errorf("NewSendMail Error: %v", err)
		return FailWithDataOperate(c, 500, "发送测试邮件失败", "", err)
	}

	return SuccessWithOperate(c, "安全策略-系统状态:发送测试邮件:"+receiver, nil)
}
