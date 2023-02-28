package model

import (
	"tkbastion/server/utils"
)

//1、新建一个DTO结构体增加字段校验
//2、

// PolicyConfig
// 策略配置表   和数据库映射
type PolicyConfig struct {
	ID               string `gorm:"type:varchar(128);primary_key;comment:策略配置id"       json:"id"                         `
	StatusAll        int    `gorm:"type;type:tinyint(1);default:0;comment:总状态"           json:"statusAll"                       `
	StatusSystemDisk bool   `gorm:"type;type:tinyint(1);default:0;comment:系统盘状态"        json:"statusSystemDisk"                       `
	StatusDataDisk   bool   `gorm:"type;type:tinyint(1);default:0;comment:数据盘状态"        json:"statusDataDisk"                       `
	StatusMemory     bool   `gorm:"type;type:tinyint(1);default:0;comment:内存占用状态"       json:"statusMemory"                       `
	StatusCpu        bool   `gorm:"type;type:tinyint(1);default:0;comment: Cpu占用状态"      json:"statusCpu"                            `

	ContinuedSystemDisk int64 `gorm:"type:bigint;default:60;comment:系统盘持续时间"             json:"continuedSystemDisk"                      `
	ContinuedDataDisk   int64 `gorm:"type:bigint;default:60;comment:数据盘持续时间"           json:"continuedDataDisk"                      `
	ContinuedMemory     int64 `gorm:"type:bigint;default:60;comment:内存持续时间"           json:"continuedMemory"                      `
	ContinuedCpu        int64 `gorm:"type:bigint;default:60;comment:Cpu持续时间"           json:"continuedCpu"                      `

	ThresholdSystemDisk int `gorm:"type:int;default:80;comment:系统盘超过比"                    json:"thresholdSystemDisk"                    `
	ThresholdDataDisk   int `gorm:"type:int;default:80;comment:  数据盘超过比"                    json:"thresholdDataDisk"                    `
	ThresholdMemory     int `gorm:"type:int;default:80;comment:  数据盘超过比"                    json:"thresholdMemory"                    `
	ThresholdCpu        int `gorm:"type:int;default:80;comment:  数据盘超过比"                    json:"thresholdCpu"                    `

	PathSystemDisk string `gorm:"type:varchar(128);comment:系统盘路径"               json:"pathSystemDisk"                    `
	PathDataDisk   string `gorm:"type:varchar(128);comment:数据盘路径"                 json:"pathDataDisk" `

	Frequency         int64          `gorm:"type:bigint;default:120;comment:发送频率"              json:"frequency"                     `
	FrequencyTimeType string         `gorm:"type:varchar(7);default:second;comment:发送时间单位"   json:"frequencyTimeType"                      `
	Created           utils.JsonTime `gorm:"type:datetime(3);not null;comment:创建日期"            json:"created"                       `
	Updated           utils.JsonTime `gorm:"type:datetime(3);not null;comment:修改日期"            json:"updated"                       `
}

func (p *PolicyConfig) ConvPolicyConfigDTO() *PolicyConfigDTO {
	return &PolicyConfigDTO{
		ID:                  p.ID,
		StatusAll:           p.StatusAll,
		StatusSystemDisk:    p.StatusSystemDisk,
		StatusDataDisk:      p.StatusDataDisk,
		StatusMemory:        p.StatusMemory,
		StatusCpu:           p.StatusCpu,
		ContinuedSystemDisk: p.ContinuedSystemDisk,
		ContinuedDataDisk:   p.ContinuedDataDisk,
		ContinuedMemory:     p.ContinuedMemory,
		ContinuedCpu:        p.ContinuedCpu,

		ThresholdSystemDisk: p.ThresholdSystemDisk,
		ThresholdDataDisk:   p.ThresholdDataDisk,
		ThresholdMemory:     p.ThresholdMemory,
		ThresholdCpu:        p.ThresholdCpu,

		PathSystemDisk: p.PathSystemDisk,
		PathDataDisk:   p.PathDataDisk,

		Frequency:         p.convertTimeByType(p.Frequency, p.FrequencyTimeType),
		FrequencyTimeType: p.FrequencyTimeType,
		Created:           p.Created,
		Updated:           p.Updated,
	}
}

// convertTimeByType   数据库存储的秒，转换为前端的时间单位
func (p *PolicyConfig) convertTimeByType(time int64, timeType string) int64 {
	switch timeType {
	case "second":
		return time
	case "minute":
		return time / 60
	case "hour":
		return time / 60 / 60
	case "day":
		return time / 60 / 60 / 24
	case "month":
		return time / 60 / 60 / 24 / 30
	}
	return time
}

// PolicyConfigDTO 数据交换实体  和前端交换数据对应
// validate:"len=36|len=0"   validate:"max=1,min=4"   validate:"path"
type PolicyConfigDTO struct {
	ID               string `label:"[策略配置id]"      json:"id"                                            `
	StatusAll        int    `label:"[总状态]"          json:"statusAll"             validate:"-" `
	StatusSystemDisk bool   `label:"[系统盘状态]"       json:"statusSystemDisk"  validate:"-"      `
	StatusDataDisk   bool   `label:"[数据盘状态]"       json:"statusDataDisk"     validate:"-"   `
	StatusMemory     bool   `label:"[内存占用状态]"        json:"statusMemory"      validate:"-"               `
	StatusCpu        bool   `label:"[Cpu占用状态]"          json:"statusCpu"        validate:"-"               `

	ContinuedSystemDisk int64 `label:"[系统盘持续时间]"       json:"continuedSystemDisk"    validate:"required,min=1"                  `
	ContinuedDataDisk   int64 `label:"[数据盘持续时间]"         json:"continuedDataDisk"     validate:"required,min=1"                 `
	ContinuedMemory     int64 `label:"[内存持续时间]"           json:"continuedMemory"        validate:"required,min=1"             `
	ContinuedCpu        int64 `label:"[Cpu持续时间]"              json:"continuedCpu"         validate:"required,min=1"            `

	ThresholdSystemDisk int `label:"[系统盘超过比]"      json:"thresholdSystemDisk"          validate:"required,min=1,max=100"        `
	ThresholdDataDisk   int `label:"[数据盘超过比]"        json:"thresholdDataDisk"         validate:"required,min=1,max=100"           `
	ThresholdMemory     int `label:"[内存超过比]"        json:"thresholdMemory"              validate:"required,min=1,max=100"     `
	ThresholdCpu        int `label:"[Cpu超过比]"        json:"thresholdCpu"                 validate:"required,min=1,max=100"  `

	PathSystemDisk string `label:"[系统盘路径]"  json:"pathSystemDisk"           validate:"-"                        `
	PathDataDisk   string `label:"[数据盘路径]"  json:"pathDataDisk"              validate:"-"                      `

	Frequency         int64          `label:"[发送频率]"    json:"frequency"            validate:"required,min=1"                             `
	FrequencyTimeType string         `label:"[发送时间单位]"    json:"frequencyTimeType"                          `
	Created           utils.JsonTime `label:"[创建日期]"   json:"created"                       `
	Updated           utils.JsonTime `label:"[修改日期]"   json:"updated"         validate:"-"                     `
}

func (p *PolicyConfigDTO) ConvPolicyConfig() *PolicyConfig {
	return &PolicyConfig{
		ID:               p.ID,
		StatusAll:        p.StatusAll,
		StatusSystemDisk: p.StatusSystemDisk,
		StatusDataDisk:   p.StatusDataDisk,
		StatusMemory:     p.StatusMemory,
		StatusCpu:        p.StatusCpu,

		ContinuedSystemDisk: p.ContinuedSystemDisk,
		ContinuedDataDisk:   p.ContinuedDataDisk,
		ContinuedMemory:     p.ContinuedMemory,
		ContinuedCpu:        p.ContinuedCpu,

		ThresholdSystemDisk: p.ThresholdSystemDisk,
		ThresholdDataDisk:   p.ThresholdDataDisk,
		ThresholdMemory:     p.ThresholdMemory,
		ThresholdCpu:        p.ThresholdCpu,

		PathSystemDisk: p.PathSystemDisk,
		PathDataDisk:   p.PathDataDisk,

		Frequency:         p.convertSecond(p.Frequency, p.FrequencyTimeType),
		FrequencyTimeType: p.FrequencyTimeType,
		Created:           p.Created,
		Updated:           p.Updated,
	}
}

// convertSecond  前端参数中的时间+时间单位统一转换为秒
func (p *PolicyConfigDTO) convertSecond(time int64, timeType string) int64 {
	switch timeType {
	case "second":
		return time
	case "minute":
		return time * 60
	case "hour":
		return time * 60 * 60
	case "day":
		return time * 60 * 60 * 24
	case "month":
		return time * 60 * 60 * 24 * 30
	}
	return 60
}

// MailSend 邮件发送消息实体
type MailSend struct {
	Recipient string
	Body      string
	Time      int64
	OutOfTime int64
}

func (m MailSend) Equals(item *MailSend) bool {
	//fmt.Printf("当前:%+v  比较:%+v",m,item)
	return m.Body == item.Body
}

type MailTiming struct {
	ContinuedSystemDiskOld int64 `gorm:"type:bigint;default:0;comment:系统盘已持续时间"             json:"continued_system_disk_old"                      `
	ContinuedDataDiskOld   int64 `gorm:"type:bigint;default:0;comment:数据盘已持续时间"           json:"continued_data_disk_old"                      `
	ContinuedMemoryOld     int64 `gorm:"type:bigint;default:0;comment:内存已持续时间"           json:"continued_memory_old"                      `
	ContinuedCpuOld        int64 `gorm:"type:bigint;default:0;comment:Cpu已持续时间"           json:"continued_cpu_old"                      `
	FrequencyOld           int64 `gorm:"type:bigint;default:0;comment:发送频率已过秒"       json:"frequency_old"                     `
}
