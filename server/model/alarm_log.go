package model

import "tkbastion/server/utils"

type OperateAlarmLog struct {
	ID                int64          `json:"id" gorm:"primary_key;auto_increment"`
	AlarmTime         utils.JsonTime `json:"alarm_time" gorm:"type:datetime(3);not null"`
	ClientIP          string         `json:"client_ip" gorm:"type:varchar(255);not null"`
	UserId            string         `json:"user_id" gorm:"type:varchar(128);not null"`
	PassportId        string         `json:"passport_id" gorm:"type:varchar(128);not null"`
	AssetId           string         `json:"asset_id" gorm:"type:varchar(128);not null"`
	Content           string         `json:"content" gorm:"type:varchar(255);not null"`             // 告警内容
	Level             string         `json:"level" gorm:"type:varchar(32);not null"`                // 告警级别
	CommandStrategyId string         `json:"command_strategy_id" gorm:"type:varchar(128);not null"` // 告警策略ID
	Result            string         `json:"result" gorm:"type:varchar(100);not null"`              // 告警结果
}

func (OperateAlarmLog) TableName() string {
	return "operate_alarm_log"
}

type SystemAlarmLog struct {
	ID        int64          `json:"id" gorm:"primary_key;auto_increment"`
	AlarmTime utils.JsonTime `json:"alarmTime" gorm:"type:datetime(3);not null"`
	Content   string         `json:"content" gorm:"type:varchar(255);not null"`  // 告警内容
	Level     string         `json:"level" gorm:"type:varchar(32);not null"`     // 告警级别
	Result    string         `json:"result" gorm:"type:varchar(255);not null"`   // 告警结果
	Strategy  string         `json:"strategy" gorm:"type:varchar(255);not null"` // 告警策略
}

func (SystemAlarmLog) TableName() string {
	return "system_alarm_log"
}
