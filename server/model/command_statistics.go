package model

import (
	"time"
)

type CommandStatist struct {
	Name        string    `gorm:"type:varchar(64);not null;comment:指令名称" json:"name"  validate:"required,max=64"`
	Created     time.Time `gorm:"type:datetime(3);not null;comment:创建日期" json:"created"  label:"创建日期"`
	Ip          string    `gorm:"type:varchar(64);comment:ip" json:"ip"  validate:"required,max=64"`
	Assetname   string    `gorm:"type:varchar(64);comment:资产名称"  validate:"required,max=64" label:"资产名称"`
	Accountname string    `gorm:"type:varchar(64);comment:执行人"  validate:"required,max=64" label:"执行人"`
	Sourceip    string    `gorm:"type:varchar(64);comment:来源ip"  validate:"required,max=64" label:"来源ip"`
}

func (r *CommandStatist) TableName() string {
	return "command_statist"
}
