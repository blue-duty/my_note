package model

import "tkbastion/server/utils"

type NewJob struct {
	ID   string `gorm:"type:varchar(128);primary_key;not null;comment:任务id" map:"key:id"`
	Name string `gorm:"type:varchar(64);unique;not null;comment:任务名称" map:"key:name"`
	// 运行方式
	RunType   string `gorm:"type:varchar(16);not null;comment:运行方式" map:"key:run_type"` //1. command 2. shell
	ShellName string `gorm:"type:varchar(128);not null;comment:shell名称" map:"key:shell_name;empty"`
	Command   string `gorm:"type:varchar(128);not null;comment:命令" map:"key:command;empty"`
	// 运行时间
	RunTimeType  string         `gorm:"type:varchar(16);not null;comment:运行时间类型" map:"key:run_time_type"` // 1. Manual 2. Scheduled 3. Periodic
	Department   string         `gorm:"type:varchar(64);not null;comment:部门" map:"key:department"`
	DepartmentID int64          `gorm:"type:bigint;not null;comment:部门id" map:"key:department_id"`
	RunTime      utils.JsonTime `gorm:"type:datetime(3);comment:运行时间" map:"key:run_time;empty"`
	StartAt      utils.JsonTime `gorm:"type:datetime(3);comment:开始时间" map:"key:start_at;empty"`
	EndAt        utils.JsonTime `gorm:"type:datetime(3);comment:结束时间" map:"key:end_at;empty"`
	PeriodicType string         `gorm:"type:varchar(16);not null;comment:周期类型" map:"key:periodic_type;empty"` // 1. Day 2. Week 3. Month 4. Minute 5. Hour
	Periodic     int            `gorm:"type:integer;comment:周期" map:"key:periodic;empty"`
	Info         string         `gorm:"type:varchar(128);not null;comment:任务描述" map:"key:info;empty"`
}

func (NewJob) TableName() string {
	return "new_job"
}

type NewJobWithAssets struct {
	ID      string `gorm:"type:varchar(128);primary_key;not null;comment:id"`
	Jid     string `gorm:"type:varchar(128);not null;comment:任务id"`
	AssetId string `gorm:"type:varchar(128);not null;comment:设备id"`
}

func (NewJobWithAssets) TableName() string {
	return "new_job_with_assets"
}

type NewJobWithAssetGroups struct {
	ID           string `gorm:"type:varchar(128);primary_key;not null;comment:id"`
	Jid          string `gorm:"type:varchar(128);not null;comment:任务id"`
	AssetGroupId string `gorm:"type:varchar(128);not null;comment:设备组id"`
}

func (NewJobWithAssetGroups) TableName() string {
	return "new_job_with_asset_groups"
}

type NewJobLog struct {
	ID           int64          `gorm:"type:bigint(20) AUTO_INCREMENT;not null;comment:id"`
	Name         string         `gorm:"type:varchar(64);not null;comment:任务名称"`
	Department   string         `gorm:"type:varchar(64);not null;comment:部门"`
	DepartmentID int64          `gorm:"type:bigint(20);not null;comment:部门id"`
	Command      string         `gorm:"type:varchar(128);not null;comment:执行内容"`
	Type         string         `gorm:"type:varchar(16);not null;comment:执行方式"`
	StartAt      utils.JsonTime `gorm:"type:datetime(3);not null;comment:开始时间"`
	EndAt        utils.JsonTime `gorm:"type:datetime(3);not null;comment:结束时间"`
	Result       string         `gorm:"type:varchar(128);not null;comment:结果"`
	AssetName    string         `gorm:"type:varchar(128);not null;comment:设备名称"`
	AssetIp      string         `gorm:"type:varchar(128);not null;comment:设备ip"`
	Passport     string         `gorm:"type:varchar(128);not null;comment:帐号"`
	Port         int            `gorm:"type:int(5);not null;comment:端口"`
	ResultInfo   string         `gorm:"type:varchar(4096);not null;comment:结果信息"`
}

func (NewJobLog) TableName() string {
	return "new_job_logs"
}
