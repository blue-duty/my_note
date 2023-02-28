package model

import (
	"tkbastion/server/utils"
)

type Job struct {
	ID          string         `gorm:"type:varchar(128);primary_key;not null;comment:计划任务id" label:"计划任务" json:"id" `
	CronJobId   int            `gorm:"type:int;default:null;comment:定时任务id" json:"cronJobId" label:"定时任务id"`
	Name        string         `gorm:"type:varchar(64);unique;not null;comment:计划任务名" json:"name" label:"[任务名称输入框]"  validate:"required,max=64"`
	Func        string         `gorm:"type:varchar(32);not null;comment:计划任务类型" json:"func" label:"计划任务类型"`
	Cron        string         `gorm:"type:varchar(32);not null;comment:cron表达式" json:"cron" label:"[cron表达式输入框]" validate:"required,max=32"`
	Mode        string         `gorm:"type:varchar(8);not null;comment:任务执行范围" json:"mode" label:"任务执行范围"`
	ResourceIds string         `gorm:"type:longtext;default:null;comment:资源id" json:"resourceIds" label:"资源id"`
	Status      string         `gorm:"type:varchar(16);default:null;comment:计划任务执行状态" json:"status" label:"计划任务执行状态"`
	Metadata    string         `gorm:"type:longtext;default:null;comment:计划任务shell命令" json:"metadata" label:"[shell脚本输入框]"`
	Created     utils.JsonTime `gorm:"type:datetime(3);not null;comment:任务创建时间" json:"created" label:"任务创建时间"`
	Updated     utils.JsonTime `gorm:"type:datetime(3);default:null;comment:任务更新时间" json:"updated" label:"任务更新时间"`
}

func (r *Job) TableName() string {
	return "jobs"
}

type JobLog struct {
	ID        string         `gorm:"type:varchar(128);primary_key;not null;comment:日志id" json:"id" `
	Timestamp utils.JsonTime `gorm:"type:datetime(3);not null;comment:记录时间" json:"timestamp"`
	JobId     string         `gorm:"type:varchar(128);not null;comment:计划任务id" json:"jobId"`
	Message   string         `gorm:"type:varchar(4096);default:'';comment:任务执行结果" json:"message"`
}

func (r *JobLog) TableName() string {
	return "job_logs"
}
