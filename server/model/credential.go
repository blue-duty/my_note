package model

import (
	"tkbastion/server/utils"
)

type Credential struct {
	ID         string         `gorm:"type:varchar(128);primary_key;not null;comment:授权凭证id"  json:"id" label:"授权凭证id"`
	Name       string         `gorm:"type:varchar(64);unique;not null;comment:授权凭证名称"  json:"name"  label:"[凭证名称输入框]" validate:"required,max=64"`
	Type       string         `gorm:"type:varchar(16);not null;comment:登录资源方式"  json:"type"  label:"登录资源方式" validate:"required"`
	Username   string         `gorm:"type:varchar(64);default:'';comment:用户名"  json:"username" label:"[授权账户输入框]" validate:"max=64"`
	Password   string         `gorm:"type:varchar(64);default:'';comment:密码"  json:"password" label:"[授权密码输入框]" validate:"max=64"`
	PrivateKey string         `gorm:"type:varchar(4096);default:'';comment:私钥"  json:"privateKey" label:"[私钥输入框]" validate:"max=4096"`
	Passphrase string         `gorm:"type:varchar(512);default:'';comment:私钥密码"  json:"passphrase" label:"[私钥密码输入框]" validate:"max=512"`
	Created    utils.JsonTime `gorm:"type:datetime(3);not null;comment:凭证创建日期"  json:"created" label:"凭证创建日期" validate:"required"`
	Owner      string         `gorm:"type:varchar(128);index;not null;comment:凭证所有者" json:"owner" label:"凭证所有者"`
}

func (r *Credential) TableName() string {
	return "credentials"
}

type CredentialForPage struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Username    string         `json:"username"`
	Created     utils.JsonTime `json:"created"`
	Owner       string         `json:"owner"`
	OwnerName   string         `json:"ownerName"`
	SharerCount int64          `json:"sharerCount"`
}

type CredentialSimpleVo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
