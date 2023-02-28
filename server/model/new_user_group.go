package model

import (
	"tkbastion/server/utils"
)

type UserGroupNew struct {
	ID             string         `gorm:"type:varchar(128);primary_key;not null;comment:用户组id" json:"id" label:"用户组id"`
	Name           string         `gorm:"type:varchar(64);not null;comment:用户组名" json:"name" label:"用户组名" validate:"required,max = 64"`
	Created        utils.JsonTime `gorm:"type:datetime(3);not null;comment:创建日期" json:"created" label:"创建日期"`
	DepartmentName string         `gorm:"type:varchar(64);not null;comment:部门机构" json:"departmentName"`
	DepartmentId   int64          `gorm:"type:bigint(20);not null;comment:部门机构id" json:"departmentId"`
	Description    string         `gorm:"type:varchar(64);not null;comment:描述" json:"description"`
	Total          int            `gorm:"type:int(3);not null;comment:用户成员数" json:"total" label:"用户成员数" `
}

func (r *UserGroupNew) TableName() string {
	return "user_group_new"
}

type UserGroupMember struct {
	ID          string `gorm:"type:varchar(128);not null;comment:用户组成员id"  json:"name"`
	UserId      string `gorm:"type:varchar(128);index;not null;comment:用户id" json:"userId"`
	UserGroupId string `gorm:"type:varchar(128);index;not null;comment:用户组id" json:"userGroupId"`
}

func (r *UserGroupMember) TableName() string {
	return "user_group_members"
}
