package model

type ApplicationAuthReportForm struct {
	ID              int64  `gorm:"primary_key;type:bigint(20) AUTO_INCREMENT;not null;comment:应用权限报表数据ID" json:"id"`
	OperateAuthId   int64  `gorm:"type:bigint(20);not null;comment:运维授权策略ID" json:"operateAuthId"`
	OperateAuthName string `gorm:"type:varchar(32);not null;default:'';comment:运维授权策略名称" json:"operateAuthName"`
	ApplicationId   string `gorm:"type:varchar(128);primary_key;not null;comment:应用id" json:"applicationId"`
	AppSerName      string `gorm:"type:varchar(64);not null;default:'';comment:应用服务名称" json:"appSerName"`
	ProgramName     string `gorm:"type:varchar(64);not null;default:'';comment:程序名称" json:"programName"`
	AppName         string `gorm:"type:varchar(64);not null;default:'';comment:应用名称" json:"appName"`
	UserId          string `gorm:"type:varchar(128);not null;comment:用户id" json:"userId"`
	Username        string `gorm:"type:varchar(64);default:'';not null;comment:用户名" json:"username"`
	Nickname        string `gorm:"type:varchar(32);default:'';not null;comment:昵称" json:"nickname"`
}

func (r *ApplicationAuthReportForm) TableName() string {
	return "application_auth_report_form"
}
