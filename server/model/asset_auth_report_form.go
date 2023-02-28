package model

type AssetAuthReportForm struct {
	ID              int64  `gorm:"primary_key;type:bigint(20) AUTO_INCREMENT;not null;comment:资产权限报表数据ID" json:"id"`
	AssetAccountId  string `gorm:"type:varchar(128);primary_key;not null;comment:设备帐号id" json:"assetAccountId"`
	AssetName       string `gorm:"type:varchar(64);not null;default:'';comment:设备名称" json:"assetName"`
	AssetAddress    string `gorm:"type:varchar(64);not null;default:'';comment:设备地址" json:"assetAddress"`
	AssetAccount    string `gorm:"type:varchar(64);not null;default:'';comment:设备帐号" json:"assetAccount"`
	UserId          string `gorm:"type:varchar(128);not null;comment:用户id" json:"userId"`
	Username        string `gorm:"type:varchar(64);default:'';not null;comment:用户名" json:"username"`
	Nickname        string `gorm:"type:varchar(32);default:'';not null;comment:昵称" json:"nickname"`
	OperateAuthId   int64  `gorm:"type:bigint(20);not null;comment:运维授权策略ID" json:"operateAuthId"`
	OperateAuthName string `gorm:"type:varchar(32);not null;default:'';comment:运维授权策略名称" json:"operateAuthName"`
}

func (r *AssetAuthReportForm) TableName() string {
	return "asset_auth_report_form"
}
