package model

type NewProgram struct {
	ID   string `gorm:"type:varchar(128);primary_key;comment:'程序ID'" json:"id" map:"key:id"`
	Name string `gorm:"type:varchar(64);comment:'程序名称'" json:"name" validate:"required,max=64" map:"key:name"`
	Path string `gorm:"type:varchar(128);comment:'程序路径'" json:"path" validate:"required,max=128" map:"key:path"`
	Info string `gorm:"type:varchar(128);comment:'程序信息'" json:"info" validate:"max=128" map:"key:info;empty"`
	Aid  string `gorm:"type:varchar(128);comment:'应用服务器ID'" json:"aid" validate:"required,max=128" map:"key:aid"`
}

func (NewProgram) TableName() string {
	return "new_program"
}
