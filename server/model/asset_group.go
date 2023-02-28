package model

import "tkbastion/server/utils"

type AssetGroup struct {
	Id      string         `gorm:"type:varchar(128);not null;primary_key;comment:'资产组ID'" json:"id"`
	Name    string         `gorm:"type:varchar(64);not null;comment:'资产组名称'" json:"name" validate:"required,max=64"`
	Count   int            `gorm:"type:int(5);not null;comment:'资产个数'" json:"count" validate:"gte=0,lte=65535"`
	Owner   string         `gorm:"type:varchar(128);not null;comment:'资产组所有者'" json:"owner" validate:"max=128"`
	Info    string         `gorm:"type:text;comment:'资产组信息'" json:"info"`
	Created utils.JsonTime `gorm:"type:datetime(3);not null;comment:'创建时间'" json:"created"`
	Assets  []Asset        `gorm:"many2many:asset_group_asset"`
}

func (AssetGroup) TableName() string {
	return "asset_group"
}

type AssetGroupForPage struct {
	Id      string         `json:"id"`
	Name    string         `json:"name"`
	Count   int            `json:"count"`
	Owner   string         `json:"owner"`
	OwnerId string         `json:"ownerId"`
	Info    string         `json:"info"`
	Created utils.JsonTime `json:"created"`
}

type AssetGroupUser struct {
	Id  string `gorm:"type:varchar(128);not null;primary_key;comment:'资产组用户ID'" json:"id"`
	Aid string `gorm:"type:varchar(128);not null;comment:'资产组ID'" json:"aid"`
	Uid string `gorm:"type:varchar(128);not null;comment:'用户ID'" json:"uid"`
}

func (AssetGroupUser) TableName() string {
	return "asset_group_user"
}

type AssetGroupUserGroup struct {
	Id   string `gorm:"type:varchar(128);not null;primary_key;comment:'资产组用户组ID'" json:"id"`
	Aid  string `gorm:"type:varchar(128);not null;comment:'资产组ID'" json:"aid"`
	Ugid string `gorm:"type:varchar(128);not null;comment:'用户组ID'" json:"ugid"`
}

func (AssetGroupUserGroup) TableName() string {
	return "asset_group_user_group"
}

type AsseyWithMode struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	//AccountType  string         `gorm:"type:varchar(16);not null;comment:登录方式" json:"accountType" label:"登录方式"`
	//Username     string         `gorm:"type:varchar(64);default:'';comment:用户名" json:"username" label:"[授权账户输入框]" validate:"max=64"`
	//Password     string         `gorm:"type:varchar(64);default:'';comment:密码" json:"password" label:"[授权密码输入框]" validate:"max=64"`
	//CredentialId string         `gorm:"type:varchar(128);index;default:'';comment:授权凭证id" json:"credentialId" label:"授权凭证id"`
	//PrivateKey   string         `gorm:"type:varchar(4096);default:'';comment:私钥" json:"privateKey" label:"私钥" validate:"max=4096"`
	//Passphrase   string         `gorm:"type:varchar(512);default:'';comment:私钥密码" json:"passphrase" label:"私钥密码" validate:"max=512"`
	//Description string         `gorm:"type:varchar(4096);default:'';comment:资产描述" json:"description" label:"[资产描述输入框]" validate:"max=4096"`
	Active  bool           `json:"active"`
	Created utils.JsonTime `json:"created"`
	Tags    string         `json:"tags"`
	//Owner   string         `gorm:"type:varchar(128);index;not null;comment:资产所有者id"  json:"owner" label:"资产所有者id"`
	SshMode string `json:"sshMode"`
}
