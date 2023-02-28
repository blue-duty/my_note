package service

import (
	"bytes"
	"errors"
	"fmt"
	"time"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/log"
	"tkbastion/server/model"
	"tkbastion/server/repository"
	"tkbastion/server/utils"

	"github.com/robfig/cron/v3"
)

//策略配置的相关服务

type policyConfigSrv struct {
	baseService
	flag    bool //间隔时间标志
	job     *cron.Cron
	MailBuf chan *model.MailSend //发送邮件chan  邮箱
	Cache   *model.MailTiming
}

func NewPolicyConfigSrv() *policyConfigSrv {
	return &policyConfigSrv{
		job:     cron.New(),
		MailBuf: make(chan *model.MailSend, 10),
		Cache:   new(model.MailTiming),
	}
}

// InitPolicyConfigData 初始化策略配置数据表
func (p policyConfigSrv) InitPolicyConfigData() error {
	//初始化策略配置数据
	item, err := repository.PolicyConfigDao.FindById("1")
	if err != nil {
		item = &model.PolicyConfig{
			ID:             "1",
			PathSystemDisk: "/",
			PathDataDisk:   "/data",
			Created:        utils.NowJsonTime(),
			Updated:        utils.NowJsonTime(),
		}
		if err := repository.PolicyConfigDao.Create(item); err != nil {
			return err
		}
		return nil
	}
	return nil
}

//定时任务:每秒检测系统盘占用、数据盘占用、内存占用、Cpu占用.
//1、数据库启动插入数据 ok
//2、检测服务
//3、定时服务
//逻辑:
// 获取数据库数据   检查邮件是否可用   校测邮件是否配置
// 检测系统盘占用、数据盘占用、内存占用、Cpu占用.
// 1-多处登录  2-持续时间

// Run  系统检测邮箱报警启动
func (p policyConfigSrv) Run(c *model.PolicyConfig) error {
	if c.StatusAll == 1 {
		_, err := p.policyConfigCheckRun()
		if err != nil {
			return nil
		}
	} else if c.StatusAll == 0 {
		err := p.policyConfigCheckStop()
		if err != nil {
			return nil
		}
	}
	return nil
}

// PolicyConfigCheckRun 策略检测运行
func (p policyConfigSrv) policyConfigCheckRun() (bool, error) {
	if p.job != nil && len(p.job.Entries()) <= 0 {
		//执行定时任务
		log.Infof("开启系统状态监控")
		p.job.AddFunc("@every 1s", p.controller)
		p.job.Start()
		return true, nil
	}

	return false, nil
}

// PolicyConfigCheckStop 策略检测停止
func (p policyConfigSrv) policyConfigCheckStop() error {
	//关闭定时任务
	if p.job != nil {
		log.Infof("系统状态监控: 关闭")
		ens := p.job.Entries()
		for _, val := range ens {
			log.Infof("关闭系统状态监控")
			p.job.Remove(val.ID)
		}
	}

	return nil
}

// StartCheck 开始检测,1次检测的逻辑控制
func (p policyConfigSrv) controller() {
	var err error
	//检查邮箱配置状态
	mail, err := p.checkStatus()
	if err != nil {
		log.Errorf("邮箱配置异常,异常信息:%v", err)
	}
	//处理邮件信息
	if err = p.putCheckMail(mail); err != nil {
		log.Errorf("发送邮箱异常,异常信息:%v", err)
	}
}

// CheckStatus 检查邮件配置状态
func (p policyConfigSrv) checkStatus() (string, error) {
	var err error
	//检测邮件是否可用
	if err = MailSrv.CheckMail(); err != nil {
		return "", err
	}
	//检测邮件是否配置
	mail, err := repository.PropertyDao.FindByName("mail-receiver")
	if err != nil || mail.Value == "" {
		return "", errors.New(errs.MailRecipientIsNull)
	}
	return mail.Value, nil
}

// putCheckMail 将邮件放入邮箱
func (p policyConfigSrv) putCheckMail(mail string) error {
	var err error

	//获取当前数据库数据
	config, err := repository.PolicyConfigDao.FindConfig()
	//_, err  = repository.PolicyConfigDao.FindConfig()
	if err != nil {
		return err
	}
	//总开关
	if !(config.StatusAll == 1) {
		return nil
	}

	var sysPercent float64
	var dataPercent float64
	var memPercent float64
	var cpuPercent float64
	/*//检测系统盘占用
	sysPercent := utils.GetDiskPercent("/")
	//检测数据盘占用
	dataPercent := utils.GetDiskPercent("/usr/")
	//检测内存占用
	memPercent := utils.GetMemPercent()
	//检测Cpu占用率
	cpuPercent := utils.GetCpuPercent()*/

	//计时和次数增加
	if config.StatusSystemDisk {
		p.Cache.ContinuedSystemDiskOld = p.Cache.ContinuedSystemDiskOld + 1
		//检测系统盘占用
		sysPercent = utils.GetDiskPercent("/")
		//log.Infof("时间:%v",p.Cache.ContinuedSystemDiskOld)
	}
	if config.StatusDataDisk {
		p.Cache.ContinuedDataDiskOld = p.Cache.ContinuedDataDiskOld + 1
		//检测数据盘占用
		dataPercent = utils.GetDiskPercent("/data/")
	}
	if config.StatusMemory {
		p.Cache.ContinuedMemoryOld = p.Cache.ContinuedMemoryOld + 1
		//检测内存占用
		memPercent = utils.GetMemPercent()
	}
	if config.StatusCpu {
		p.Cache.ContinuedCpuOld = p.Cache.ContinuedCpuOld + 1
		//检测Cpu占用率
		cpuPercent = utils.GetCpuPercent()
	}

	p.Cache.FrequencyOld = p.Cache.FrequencyOld + 1
	if p.Cache.FrequencyOld >= config.Frequency { //因为默认0不能更新，所以+1增大1
		p.flag = true
		p.Cache.FrequencyOld = 0 //重置
	}

	//拼接消息
	var buff bytes.Buffer
	if config.StatusSystemDisk && sysPercent >= float64(config.ThresholdSystemDisk) && p.Cache.ContinuedSystemDiskOld >= config.ContinuedSystemDisk {
		//log.Infof("%v||%v",sysPercent >= float64(config.ThresholdSystemDisk),p.Cache.ContinuedSystemDiskOld >= config.ContinuedSystemDisk)
		//log.Infof("sys:%v,thre:%v",sysPercent,float64(config.ThresholdSystemDisk))
		buff.WriteString(fmt.Sprintf("系统盘占用:%.2f%%,持续时间:%v秒,超过阈值:%v%%", sysPercent, p.Cache.ContinuedSystemDiskOld, config.ContinuedSystemDisk))
	} else if sysPercent < float64(config.ThresholdSystemDisk) {
		p.Cache.ContinuedSystemDiskOld = 0
	}

	if config.StatusDataDisk && dataPercent >= float64(config.ThresholdDataDisk) && p.Cache.ContinuedDataDiskOld >= config.ContinuedDataDisk {
		buff.WriteString(fmt.Sprintf("数据盘占用:%.2f%%,持续时间:%v秒,超过阈值:%v%%", dataPercent, p.Cache.ContinuedDataDiskOld, config.ContinuedDataDisk))
	} else if dataPercent < float64(config.ThresholdDataDisk) {
		p.Cache.ContinuedDataDiskOld = 0
	}

	if config.StatusMemory && memPercent >= float64(config.ThresholdMemory) && p.Cache.ContinuedMemoryOld >= config.ContinuedMemory {
		buff.WriteString(fmt.Sprintf("内存占用:%.2f%%,持续时间:%v秒,超过阈值:%v%%", memPercent, p.Cache.ContinuedMemoryOld, config.ContinuedMemory))
	} else if memPercent < float64(config.ThresholdMemory) {
		p.Cache.ContinuedMemoryOld = 0
	}

	if config.StatusCpu && cpuPercent >= float64(config.ThresholdCpu) && p.Cache.ContinuedCpuOld >= config.ContinuedCpu {
		buff.WriteString(fmt.Sprintf("CPU占用:%.2f%%,持续时间:%v秒,超过阈值:%v%%", cpuPercent, p.Cache.ContinuedCpuOld, config.ContinuedCpu))
	} else if cpuPercent < float64(config.ThresholdCpu) {
		p.Cache.ContinuedCpuOld = 0
	}

	/*if err = repository.PolicyConfigDao.Update(config); err != nil {
		return err
	}*/
	//log.Errorf("实体:%+v",config)
	//log.Errorf("系统盘占用:%v,数据盘占用:%v,内存占用:%v,Cpu占用%v", sysPercent, dataPercent, memPercent, cpuPercent)
	//buf不为空、评率超过上一次评率时间则发送   超过则发送电子邮件预警
	if buff.String() != "" && p.flag {
		p.flag = false
		m := &model.MailSend{
			Recipient: mail,
			Body:      buff.String(),
			Time:      time.Now().Unix(),
			OutOfTime: config.Frequency,
		}
		p.MailBuf <- m
		/*if err := MailSrv.SendMail(mail, buff.String()); err != nil {
			return err
		}*/
	}
	return nil
}

func (p policyConfigSrv) SendMail() {
	var ex model.MailSend
	ex.Time = -99
	for {
		select {
		case val := <-p.MailBuf:
			if !val.Equals(&ex) {
				ex = *val
				_ = MailSrv.SendMail(ex.Recipient, "系统报警", ex.Body)
				log.Infof("[发送邮件]:%+v", ex)
			}
		}
	}
}

// SetupRun 策略配置任务软件关闭后自启动
func (p policyConfigSrv) SetupRun() {
	go PolicyConfigSrv.SendMail()
	//获取当前数据库数据
	config, _ := repository.PolicyConfigDao.FindConfig()
	//_, err  = repository.PolicyConfigDao.FindConfig()
	if err := PolicyConfigSrv.Run(config); err != nil {
		log.Errorf("策略配置初始化出现异常,异常信息: %v", err)
	}
}

//putCheckMail 将邮件放入邮箱
/*func (p policyConfigSrv) putCheckMail(mail string) error {
	var err error

	//获取当前数据库数据
	config, err := repository.PolicyConfigDao.FindConfig()
	//_, err  = repository.PolicyConfigDao.FindConfig()
	if err != nil {
		return err
	}

	//检测系统盘占用
	sysPercent := utils.GetDiskPercent("/")
	//检测数据盘占用
	dataPercent := utils.GetDiskPercent("/usr/")
	//检测内存占用
	memPercent := utils.GetMemPercent()
	//检测Cpu占用率
	cpuPercent := utils.GetCpuPercent()

	//计时和次数增加
	if config.StatusSystemDisk {
		config.ContinuedSystemDiskOld = config.ContinuedSystemDiskOld + 1
	}
	if config.StatusDataDisk {
		config.ContinuedDataDiskOld = config.ContinuedDataDiskOld + 1
	}
	if config.StatusMemory {
		config.ContinuedMemoryOld = config.ContinuedMemoryOld + 1
	}
	if config.StatusCpu {
		config.ContinuedCpuOld = config.ContinuedCpuOld + 1
	}

	config.FrequencyOld = config.FrequencyOld + 1
	if config.FrequencyOld >= config.Frequency+1 { //因为默认0不能更新，所以+1增大1
		p.flag = true
		config.FrequencyOld = 1 //重置
	}

	//拼接消息
	var buff bytes.Buffer
	if config.StatusSystemDisk && sysPercent >= float64(config.ThresholdSystemDisk) && config.ContinuedSystemDiskOld >= config.ContinuedSystemDisk {
		buff.WriteString(fmt.Sprintf("系统盘占用:%v,持续时间:%v", sysPercent, config.ContinuedSystemDiskOld))
	} else if config.ContinuedSystemDiskOld >= config.ContinuedSystemDisk {
		config.ContinuedSystemDiskOld = 1
	}

	if config.StatusDataDisk && dataPercent >= float64(config.ThresholdDataDisk) && config.ContinuedDataDiskOld >= config.ContinuedDataDisk {
		buff.WriteString(fmt.Sprintf("数据盘占用:%v,持续时间:%v", dataPercent, config.ContinuedDataDiskOld))
	} else if config.ContinuedDataDiskOld >= config.ContinuedDataDisk {
		config.ContinuedDataDiskOld = 1
	}

	if config.StatusMemory && memPercent >= float64(config.ThresholdMemory) && config.ContinuedMemoryOld >= config.ContinuedMemory {
		buff.WriteString(fmt.Sprintf("内存占用:%v,持续时间:%v", memPercent, config.ContinuedMemoryOld))
	} else if config.ContinuedMemoryOld >= config.ContinuedMemory {
		config.ContinuedMemoryOld = 1
	}

	if config.StatusCpu && cpuPercent >= float64(config.ThresholdCpu) && config.ContinuedCpuOld >= config.ContinuedCpu {
		buff.WriteString(fmt.Sprintf("Cpu占用:%v,持续时间:%v", cpuPercent, config.ContinuedCpuOld))
	} else if config.ContinuedCpuOld >= config.ContinuedCpu {
		config.ContinuedCpuOld = 1
	}

	if err = repository.PolicyConfigDao.Update(config); err != nil {
		return err
	}
	//log.Errorf("实体:%+v",config)
	//log.Errorf("系统盘占用:%v,数据盘占用:%v,内存占用:%v,Cpu占用%v", sysPercent, dataPercent, memPercent, cpuPercent)
	//buf不为空、评率超过上一次评率时间则发送   超过则发送电子邮件预警
	if buff.String() != "" && p.flag {
		p.flag = false
		m := &model.MailSend{
			Recipient: mail,
			Body:      buff.String(),
			Time:      time.Now().Unix(),
			OutOfTime: config.Frequency,
		}
		p.MailBuf <- m
		//if err := MailSrv.SendMail(mail, buff.String()); err != nil {
		//	return err
		//}
	}
	return nil
}*/
