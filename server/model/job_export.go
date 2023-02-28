package model

import "tkbastion/server/utils"

type JobExport struct {
	ID        string         `gorm:"type:varchar(128);primary_key;not null;comment:计划任务id" label:"计划任务" json:"id" `
	Name      string         `gorm:"type:varchar(64);unique;not null;comment:计划任务名" json:"name"  validate:"required,max=64"`
	Cycle     string         `gorm:"type:varchar(32);not null;comment:周期" json:"cycle" label:"周期" `
	FirstTime int            `gorm:"type:int;comment:具体时间" json:"first_time" label:"具体时间" `
	Week      int            `gorm:"ype:varchar(32);comment:周" json:"week" label:"周" `
	Created   utils.JsonTime `gorm:"type:datetime(3);not null;comment:任务创建时间" json:"created" label:"任务创建时间"`
	FormName  string         `gorm:"type:varchar(32);not null;comment:报表名称" json:"form_name" label:"报表名称"`
	Corn      string         `gorm:"type:varchar(32);not null;comment:corn表达式" json:"corn" label:"corn表达式"`
	Describe  string         `gorm:"type:varchar(32);not null;comment:描述" json:"describe" label:"描述"`
	Cyclet    string         `gorm:"type:varchar(32);not null;comment:周期显示" json:"cyclet" label:"周期显示" `
}

func (r *JobExport) TableName() string {
	return "job_export"
}
