package model

import (
	"tkbastion/server/utils"
)

type AssetRelation struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

type Command struct {
	ID      string         `gorm:"type:varchar(128);primary_key;not null;comment:指令id" label:"指令id" json:"id"  `
	Name    string         `gorm:"type:varchar(64);unique;not null;comment:指令名称" json:"name" label:"[指令名称输入框]" validate:"required,max=64"`
	Content string         `gorm:"type:longtext;default:null;comment:指令内容" json:"content" label:"[指令内容输入框]" validate:"required,max=191" `
	Created utils.JsonTime `gorm:"type:datetime(3);not null;comment:创建日期" json:"created"  label:"创建日期"`
	Owner   string         `gorm:"type:varchar(128);index;not null;comment:指令所有者"  json:"owner"  label:"指令所有者"`
}

type CommandForPage struct {
	ID          string         `gorm:"primary_key" json:"id"`
	Name        string         `json:"name"`
	Content     string         `json:"content"`
	Created     utils.JsonTime `json:"created"`
	Owner       string         `json:"owner"`
	OwnerName   string         `json:"ownerName"`
	SharerCount int64          `json:"sharerCount"`
}

func (r *Command) TableName() string {
	return "commands"
}
