package model

import "tkbastion/server/utils"

type NewApplication struct {
	ID           string         `gorm:"type:varchar(128);primary_key;comment:'应用ID'" json:"id" map:"key:id"`
	Name         string         `gorm:"type:varchar(64);comment:'应用名称'" json:"name" validate:"required,max=64" map:"key:name"`
	Param        string         `gorm:"type:varchar(128);comment:'应用参数'" json:"param" validate:"max=128" map:"key:param;empty"`
	Info         string         `gorm:"type:varchar(128);comment:'应用信息'" json:"info" validate:"max=128" map:"key:info;empty"`
	ProgramName  string         `gorm:"type:varchar(64);comment:'程序名称'" json:"programName" validate:"required,max=64" map:"key:program_name"`
	AppSerName   string         `gorm:"type:varchar(64);comment:'应用服务器名称'" json:"appSerName" validate:"required,max=64" map:"key:app_ser_name"`
	IP           string         `gorm:"type:varchar(64);not null;comment:'应用服务器IP'" json:"ip" validate:"required,hostname|ip" map:"key:ip"`
	Port         int            `gorm:"type:int(5);not null;comment:'应用服务器端口'" json:"port" validate:"required,gte=0,lte=65535" map:"key:port"`
	Passport     string         `gorm:"type:varchar(64);comment:'应用服务器账号'" json:"passport" validate:"required,max=64" map:"key:passport"`
	Password     string         `gorm:"type:varchar(64);comment:'应用服务器密码'" json:"password" validate:"required,max=64" map:"key:password"`
	DepartmentID int64          `gorm:"type:bigint(20);comment:'部门ID'" json:"departmentId" validate:"required,gte=0" map:"key:department_id;empty"`
	Department   string         `gorm:"type:varchar(64);comment:'部门名称'" json:"department" validate:"required,max=64" map:"key:department"`
	ProgramID    string         `gorm:"type:varchar(128);comment:'程序ID'" json:"programId" validate:"required,max=128" map:"key:program_id"`
	AppSerId     string         `gorm:"type:varchar(128);comment:'应用服务器ID'" json:"appSerId" validate:"required,max=128" map:"key:app_ser_id"`
	Created      utils.JsonTime `gorm:"type:datetime;comment:'创建时间'" json:"created" map:"key:created"`
	Path         string         `gorm:"type:varchar(128);comment:'应用路径'" json:"path" validate:"required,max=128" map:"key:path"`
}

func (NewApplication) TableName() string {
	return "new_application"
}
