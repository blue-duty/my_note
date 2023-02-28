package model

import "tkbastion/server/utils"

type Role struct {
	ID      string         `json:"id"`
	Name    string         `gorm:"type:varchar(64);not null;comment:角色名称"     validate:"required,max=64" `
	Created utils.JsonTime `gorm:"type:datetime(3);not null;comment:创建日期" json:"created" label:"创建日期"`
	IsEdit  bool           `gorm:"type:tinyint(1);not null;comment:是否可编辑" json:"isEdit" label:"是否可编辑"`
	Menu    *[]Menu        `gorm:"many2many:role_menus;foreignKey:name;joinForeignKey:name;references:id;joinReferences:menu_id;"`
}

type RoleForUserCreate struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (r *UserRole) TableName() string {
	return "roles"
}

type UserRole struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"      validate:"required,max=64" `
	IsEdit  bool           `gorm:"type:tinyint(1);not null;comment:是否可编辑" json:"isEdit" label:"是否可编辑"`
	Created utils.JsonTime `gorm:"type:datetime(3);not null;comment:创建日期" json:"created" label:"创建日期"`
	MenuIds []int          `json:"menuIds"`
}

type UserRoleTest struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"      validate:"required,max=64" `
	IsEdit  bool           `gorm:"type:tinyint(1);not null;comment:是否可编辑" json:"isEdit" label:"是否可编辑"`
	Created utils.JsonTime `gorm:"type:datetime(3);not null;comment:创建日期" json:"created" label:"创建日期"`
	MenuIds [][]int        `json:"menuIds"`
}

type RoleMenu struct {
	Name   string `gorm:"type:varchar(64);primary_key;not null;comment:菜单名称"  json:"name"`
	MenuId int    `gorm:"type:int(3);primary_key;not null;comment:菜单结点id" json:"menuId"`
}

func (r *RoleMenu) TableName() string {
	return "role_menus"
}
