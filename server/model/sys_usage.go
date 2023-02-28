package model

import "tkbastion/server/utils"

type Usage struct {
	ID       uint64         `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	Datetime utils.JsonTime `json:"datetime" `
	Percent  float64        `json:"percent" `
	Total    uint64         `json:"total" `
	Used     uint64         `json:"used" `
	Free     uint64         `json:"free" `
}

type GetSysUsage struct {
	TypeUsage string `json:"typeUsage" `
	Interval  string `json:"interval" `
	StartTime string `json:"startTime" `
	EndTime   string `json:"endTime" `
}

type UsageCpu struct {
	ID       uint64         `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	Datetime utils.JsonTime `json:"datetime" `
	Percent  float64        `json:"percent" `
	Total    uint64         `json:"total" `
	Used     uint64         `json:"used" `
	Free     uint64         `json:"free" `
}

func (UsageCpu) TableName() string {
	return "usage_cpu"
}

type UsageMem struct {
	ID       uint64         `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	Datetime utils.JsonTime `json:"datetime" `
	Percent  float64        `json:"percent" `
	Total    uint64         `json:"total" `
	Used     uint64         `json:"used" `
	Free     uint64         `json:"free" `
}

func (UsageMem) TableName() string {
	return "usage_mem"
}

type UsageDisk struct {
	ID       uint64         `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	Datetime utils.JsonTime `json:"datetime" `
	Percent  float64        `json:"percent" `
	Total    uint64         `json:"total" `
	Used     uint64         `json:"used" `
	Free     uint64         `json:"free" `
}

func (UsageDisk) TableName() string {
	return "usage_disk"
}
