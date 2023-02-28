package model

type NewApplicationServer struct {
	ID           string `gorm:"type:varchar(128);not null;primary_key;comment:'应用服务器ID'" json:"id" map:"key:id"`
	Name         string `gorm:"type:varchar(64);not null;comment:'应用服务器名称'" json:"name" validate:"required,max=64" map:"key:name"`
	IP           string `gorm:"type:varchar(64);not null;comment:'应用服务器IP'" json:"ip" validate:"required,hostname|ip" map:"key:ip"`
	Port         int    `gorm:"type:int(5);not null;comment:'应用服务器端口'" json:"port" validate:"required,gte=0,lte=65535" map:"key:port"`
	Type         string `gorm:"type:varchar(64);not null;comment:'应用服务器类型'" json:"type" validate:"required,max=64" map:"key:type"`
	DepartmentID int64  `gorm:"type:bigint(20);not null;comment:'所属部门ID'" json:"department_id" validate:"required,gte=0" map:"key:department_id;empty"`
	Department   string `gorm:"type:varchar(64);not null;comment:'所属部门名称'" json:"department" validate:"required,max=64" map:"key:department"`
	//Created  utils.JsonTime `gorm:"type:datetime(3);not null;comment:'创建时间'" json:"created" map:"created"`
	Passport string `gorm:"type:varchar(64);comment:'应用服务器账号'" json:"passport" validate:"required,max=64" map:"key:passport"`
	Password string `gorm:"type:varchar(64);comment:'应用服务器密码'" json:"password" validate:"required,max=64" map:"key:password"`
	Info     string `gorm:"type:varchar(128);comment:'应用服务器信息'" json:"info" validate:"max=128" map:"key:info;empty"`
}

func (NewApplicationServer) TableName() string {
	return "new_application_server"
}
