package model

type SystemType struct {
	ID   string `gorm:"type:varchar(128);not null;primary_key;comment:'系统类型id'" json:"id" map:"key:id"`
	Name string `gorm:"type:varchar(128);not null;comment:'系统类型名称'" json:"name" validate:"required,max=128" map:"key:name"` // 1. WINDOWS 2. LINUX 3. UNIX
	Type string `gorm:"type:char(10);not null;comment:'系统类型'" json:"type" validate:"required,max=10" map:"key:type"`        // 1. host 2. internet
	Info string `gorm:"type:varchar(128);not null;comment:'系统类型描述'" json:"info" validate:"max=128" map:"key:info;empty"`
	//是否默认
	Default bool `gorm:"type:boolean;not null;comment:'是否默认'" json:"default" map:"key:default"`
}

func (SystemType) TableName() string {
	return "system_type"
}
