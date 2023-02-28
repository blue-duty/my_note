package model

import "tkbastion/server/utils"

type NewCommand struct {
	ID      string         `gorm:"type:varchar(128);primary_key;not null;comment:指令id" label:"指令id" json:"id" map:"key:id"`
	Name    string         `gorm:"type:varchar(64);unique;not null;comment:指令名称" json:"name" label:"[指令名称输入框]" validate:"required,max=64" map:"key:name"`
	Content string         `gorm:"type:longtext;default:null;comment:指令内容" json:"content" label:"[指令内容输入框]" validate:"required" map:"key:content"`
	UserId  string         `gorm:"type:varchar(128);default:null;comment:创建人id" json:"userId" label:"[创建人id输入框]" validate:"required" map:"key:user_id"`
	Created utils.JsonTime `gorm:"type:datetime(3);not null;comment:创建日期" json:"created"  label:"创建日期" map:"key:created"`
	Info    string         `gorm:"type:varchar(128);default:null;comment:指令描述" json:"info" label:"[指令描述输入框]" validate:"max=128" map:"key:info;empty"`
}

func (NewCommand) TableName() string {
	return "new_commands"
}
