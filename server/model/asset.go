package model

import (
	"tkbastion/server/utils"
)

type Asset struct {
	ID             string         `gorm:"type:varchar(128);primary_key;not null;comment:资产id" json:"id" label:"资产id"`
	Name           string         `gorm:"type:varchar(64);unique;not null;comment:资产名称" json:"name" label:"[资产名称输入框]" validate:"required,max=64"`
	Protocol       string         `gorm:"type:varchar(16);not null;comment:登录协议" json:"protocol" label:"登录协议" validate:"required"`
	IP             string         `gorm:"type:varchar(64);not null;comment:资产ip" json:"ip" label:"[资产的主机名或IP地址输入框]" validate:"required,hostname|ip"`
	Port           int            `gorm:"type:int(5);not null;comment:资产端口" json:"port" label:"[TCP端口输入框]" validate:"required,gte=0,lte=65535"`
	AccountType    string         `gorm:"type:varchar(16);not null;comment:登录方式" json:"accountType" label:"登录方式"`
	Username       string         `gorm:"type:varchar(64);default:'';comment:用户名" json:"username" label:"[授权账户输入框]" validate:"max=64"`
	Password       string         `gorm:"type:varchar(64);default:'';comment:密码" json:"password" label:"[授权密码输入框]" validate:"max=64"`
	CredentialId   string         `gorm:"type:varchar(128);index;default:'';comment:授权凭证id" json:"credentialId" label:"授权凭证id"`
	PrivateKey     string         `gorm:"type:varchar(4096);default:'';comment:私钥" json:"privateKey" label:"私钥" validate:"max=4096"`
	Passphrase     string         `gorm:"type:varchar(512);default:'';comment:私钥密码" json:"passphrase" label:"私钥密码" validate:"max=512"`
	Description    string         `gorm:"type:varchar(4096);default:'';comment:资产描述" json:"description" label:"[资产描述输入框]" validate:"max=4096"`
	Active         bool           `gorm:"type:tinyint(1);default:0;comment:资产是否在线" json:"active" label:"资产是否在线"`
	Created        utils.JsonTime `gorm:"type:datetime(3);not null;comment:资产添加时间"  json:"created" label:"资产添加时间"`
	Tags           string         `gorm:"type:varchar(64);default:'';comment:资产标签" json:"tags" label:"[资产标签输入框]" validate:"max=64"`
	Owner          string         `gorm:"type:varchar(128);index;not null;comment:资产所有者id"  json:"owner" label:"资产所有者id"`
	OperateNumbers int            `gorm:"type:int;not null;default:0;comment:运维次数" json:"operateNumbers" label:"运维次数"`
}

type AssetForPage struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	IP          string         `json:"ip"`
	Protocol    string         `json:"protocol"`
	Port        int            `json:"port"`
	Active      bool           `json:"active"`
	Created     utils.JsonTime `json:"created"`
	Tags        string         `json:"tags"`
	Owner       string         `json:"owner"`
	OwnerName   string         `json:"ownerName"`
	SharerCount int64          `json:"sharerCount"`
	SshMode     string         `json:"sshMode"`
}

func (r *Asset) TableName() string {
	return "assets"
}

type AssetAttribute struct {
	Id      string `gorm:"type:varchar(128);primary_key;not null;comment:资产属性id" json:"id"`
	AssetId string `gorm:"type:varchar(128);index;not null;comment:资产id" json:"assetId"`
	Name    string `gorm:"type:varchar(128);index;not null;comment:资产属性名" json:"name"`
	Value   string `gorm:"type:varchar(1024);not null;comment:资产属性值" json:"value"`
}

func (r *AssetAttribute) TableName() string {
	return "asset_attributes"
}
