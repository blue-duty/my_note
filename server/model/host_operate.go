package model

import (
	"tkbastion/server/utils"
)

// HostOperate 需加上三个权限控制字段
type HostOperate struct {
	Id         string `json:"id" `
	AssetName  string `json:"assetName" `
	Ip         string `json:"ip" `
	Protocol   string `json:"protocol" `
	Name       string `json:"name" `
	Status     string `json:"status" `
	Download   string `json:"download" `
	Upload     string `json:"upload" `
	Watermark  string `json:"watermark" `
	LoginType  string `json:"loginType" `
	Collection string `json:"collection" `
}

type UserCollecte struct {
	Id             int64  `gorm:"primary_key;type:bigint(20) AUTO_INCREMENT;not null;comment:用户收藏表ID" json:"id"`
	UserId         string `gorm:"type:varchar(128);not null;comment:用户id" json:"userId"`
	AssetAccountId string `gorm:"type:varchar(128);not null;comment:设备帐号id" json:"assetAccountId" `
}

func (UserCollecte) TableName() string {
	return "user_collecte"
}

type OperationAndMaintenanceLog struct {
	Id           int64          `gorm:"primary_key;type:bigint(20) AUTO_INCREMENT;not null;comment:运维日志表ID" json:"id"`
	LoginTime    utils.JsonTime `gorm:"type:datetime(3);not null;comment:登录时间" json:"loginTime"`
	Username     string         `gorm:"type:varchar(128);not null;comment:用户名" json:"username"`
	Nickname     string         `gorm:"type:varchar(128);not null;comment:用户昵称" json:"nickname"`
	DepartmentID int64          `gorm:"type:bigint(20);not null;comment:部门ID" json:"departmentId"`
	Protocol     string         `gorm:"type:varchar(128);not null;comment:协议" json:"protocol"`
	AssetName    string         `gorm:"type:varchar(128);not null;comment:资产名称" json:"assetName"`
	AssetIp      string         `gorm:"type:varchar(128);not null;comment:资产IP" json:"assetIp"`
	Ip           string         `gorm:"type:varchar(128);not null;comment:IP" json:"ip"`
	Passport     string         `gorm:"type:varchar(128);not null;comment:帐号" json:"passport"`
	LogoutTime   utils.JsonTime `gorm:"type:datetime(3);not null;comment:登出时间" json:"logoutTime"`
	Result       string         `gorm:"type:char(10);not null;comment:结果" json:"result"`
	Info         string         `gorm:"type:varchar(128);not null;comment:信息" json:"info"`
}

func (OperationAndMaintenanceLog) TableName() string {
	return "operation_and_maintenance_log"
}

type UserCollectApp struct {
	Id            int64  `gorm:"primary_key;type:bigint(20) AUTO_INCREMENT;not null;comment:用户收藏表ID" json:"id"`
	UserId        string `gorm:"type:varchar(128);not null;comment:用户id" json:"userId"`
	ApplicationId string `gorm:"type:varchar(128);not null;comment:应用id" json:"applicationId" `
}

func (UserCollectApp) TableName() string {
	return "user_collect_app"
}
