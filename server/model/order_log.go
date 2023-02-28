package model

import "tkbastion/server/utils"

type OrderLog struct {
	ID              string         `gorm:"type:varchar(128);primary_key;not null;comment:日志id" json:"id"`
	Created         utils.JsonTime `gorm:"type:datetime(3);not null;comment:时间" json:"created"`
	ApproveTime     utils.JsonTime `gorm:"type:datetime(3);comment:'审批时间'" json:"approveTime"`
	IP              string         `gorm:"type:varchar(64);not null;comment:申请ip" json:"ip"`
	Asset           string         `gorm:"type:varchar(64);not null;comment:资产名称" json:"asset"`
	Applicant       string         `gorm:"type:varchar(64);not null;comment:申请人" json:"applicant"`
	Status          string         `gorm:"type:varchar(64);not null;comment:状态" json:"status"`
	Approved        string         `gorm:"type:varchar(64);not null;comment:审批人" json:"approved"`
	Information     string         `gorm:"type:varchar(64);not null;comment:操作命令" json:"information"`
	ApplicationType string         `gorm:"type:varchar(64);not null;comment:申请类型" json:"applicationType"`
}

func (r *OrderLog) TableName() string {
	return "order_logs"
}

type OrderLogForPage struct {
	ID        string         `json:"id"`
	Created   utils.JsonTime `json:"created"`
	IP        string         `json:"ip"`
	Asset     string         `json:"asset"`
	Applicant string         `json:"applicant"`
	Status    string         `json:"status"`
	Approved  string         `json:"approved"`
	Command   string         `json:"command"`
}
