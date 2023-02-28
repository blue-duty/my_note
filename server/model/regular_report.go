package model

import "tkbastion/server/utils"

type RegularReport struct {
	ID   string `gorm:"type:varchar(128);primary_key;not null;comment:定期策略id" json:"id"`
	Name string `gorm:"type:varchar(64);unique;not null;comment:策略名称" json:"name"`
	//ReportType     string `gorm:"type:varchar(64);not null;comment:报表类型" json:"report_type"` // 1.Protocol 2.User 3.Login 4.Asset 5.Session 6.Command 7.Alarm
	IsProtocol   *bool  `gorm:"type:tinyint(1);default:0;comment:是否统计协议访问" json:"isProtocol"`
	IsUser       *bool  `gorm:"type:tinyint(1);default:0;comment:是否统计用户访问" json:"isUser"`
	IsLogin      *bool  `gorm:"type:tinyint(1);default:0;comment:是否统计登录尝试" json:"isLogin"`
	IsAsset      *bool  `gorm:"type:tinyint(1);default:0;comment:是否统计资产访问" json:"isAsset"`
	IsSession    *bool  `gorm:"type:tinyint(1);default:0;comment:是否统计会话访问" json:"isSession"`
	IsCommand    *bool  `gorm:"type:tinyint(1);default:0;comment:是否统计指令访问" json:"isCommand"`
	IsAlarm      *bool  `gorm:"type:tinyint(1);default:0;comment:是否统计告警访问" json:"isAlarm"`
	PeriodicType string `gorm:"type:varchar(16);not null;comment:周期类型" json:"periodicType"` // 1. Day 2. Week 3. Month
	Periodic     uint   `gorm:"type:varchar(16);not null;comment:周期" json:"periodic"`       // 1. Week: 1-7 2. Month: 1-12
	Description  string `gorm:"type:varchar(128);not null;comment:策略描述" json:"description"`
}

func (r *RegularReport) TableName() string {
	return "regular_reports"
}

type RegularReportLog struct {
	ID           string         `gorm:"type:varchar(128);primary_key;not null;comment:定期策略id" json:"id"`
	Name         string         `gorm:"type:varchar(64);not null;comment:策略名称" json:"name"`
	ExecuteTime  utils.JsonTime `gorm:"type:datetime(3);not null;comment:执行时间" json:"executeTime"`
	ReportType   string         `gorm:"type:varchar(64);not null;comment:报表类型" json:"reportType"`
	PeriodicType string         `gorm:"type:varchar(16);not null;comment:周期类型" json:"periodicType"` // 1. Day 2. Week 3. Month
	FileName     string         `gorm:"type:varchar(64);not null;comment:文件名称" json:"fileName"`
}

func (r *RegularReportLog) TableName() string {
	return "regular_reports_log"
}

type RegularReportForPaging struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ReportType   string `json:"reportType"`
	IsProtocol   bool   `json:"isProtocol"`
	IsUser       bool   `json:"isUser"`
	IsLogin      bool   `json:"isLogin"`
	IsAsset      bool   `json:"isAsset"`
	IsSession    bool   `json:"isSession"`
	IsCommand    bool   `json:"isCommand"`
	IsAlarm      bool   `json:"isAlarm"`
	PeriodicType string `json:"periodicType"` // 1. Day 2. Week 3. Month
	Periodic     uint   `json:"periodic"`     // 1. Week: 1-7 2. Month: 1-12
	Description  string `json:"description"`
}

type RegularReportLogForPaging struct {
	ID           string `json:"id"`
	Name         string ` json:"name"`
	ExecuteTime  string ` json:"executeTime"`
	ReportType   string ` json:"reportType"`
	PeriodicType string ` json:"periodicType"` // 1. Day 2. Week 3. Month
	FileName     string ` json:"fileName"`
}
