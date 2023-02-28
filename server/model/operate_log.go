package model

import (
	"tkbastion/server/utils"
)

type OperateLog struct {
	ID              int            `gorm:"type:int;primary_key;not null;AUTO_INCREMENT;comment:日志id" json:"id"`
	Created         utils.JsonTime `gorm:"type:datetime(3);index;not null;comment:时间" json:"created"`
	LogTypes        string         `gorm:"type:varchar(16);not null;default:'';comment:日志类型" json:"logTypes"`
	LogContents     string         `gorm:"type:varchar(2048);not null;comment:日志内容" json:"logContents"`
	Users           string         `gorm:"type:varchar(64);not null;comment:用户名" json:"users"`
	Names           string         `gorm:"type:varchar(64);not null;comment:姓名" json:"names"`
	Ip              string         `gorm:"type:varchar(64);not null;comment:来源ip" json:"ip"`
	ClientUserAgent string         `gorm:"type:varchar(512);not null;default:'';comment:用户客户端信息" json:"clientUserAgent"`
	Result          string         `gorm:"type:varchar(8);not null;comment:操作结果" json:"result"`
}

type OperateForPage struct {
	ID              int            `json:"id"`
	Users           string         `json:"users"`
	Names           string         `json:"names"`
	IP              string         `json:"ip"`
	ClientUserAgent string         `json:"clientUserAgent"`
	Created         utils.JsonTime `json:"created"`
	LogTypes        string         `json:"logTypes"`
	LogContents     string         `json:"logContents"`
	Result          string         `json:"result"`
}

type OperateForPageNew struct {
	ID               int            `json:"id"`
	Users            string         `json:"users"`
	Names            string         `json:"names"`
	IP               string         `json:"ip"`
	ClientUserAgent  string         `json:"clientUserAgent"`
	Created          utils.JsonTime `json:"created"`
	LogTypes         string         `json:"logTypes"`
	LogContents      string         `json:"logContents"`
	Result           string         `json:"result"`
	FunctionalModule string         `json:"functionalModule"`
	Action           string         `json:"action"`
}

func (r *OperateLog) TableName() string {
	return "operate_logs"
}
