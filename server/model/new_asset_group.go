package model

import "tkbastion/server/utils"

type NewAssetGroup struct {
	Id           string         `gorm:"type:varchar(128);not null;primary_key;comment:'资产组ID'" json:"id" map:"key:id"`
	Name         string         `gorm:"type:varchar(64);not null;comment:'资产组名称'" json:"name" validate:"required,max=64" map:"key:name"`
	Count        int            `gorm:"type:int(5);not null;comment:'设备数'" json:"count" validate:"gte=0,lte=65535" map:"key:count"`
	Department   string         `gorm:"type:varchar(128);not null;comment:'资产组所属部门'" json:"department" validate:"max=128" map:"key:department"`
	DepartmentId int64          `gorm:"type:bigint(20);not null;comment:'资产组所属部门ID'" json:"departmentId" validate:"gte=0" map:"key:department_id"`
	Info         string         `gorm:"type:text;comment:'资产组信息'" json:"info" validate:"max=128" map:"key:info;empty"`
	Created      utils.JsonTime `gorm:"type:datetime(3);not null;comment:'创建时间'" json:"created" map:"key:created"`
}

func (NewAssetGroup) TableName() string {
	return "new_asset_group"
}

type AssetGroupWithAsset struct {
	ID           string `gorm:"type:varchar(128);not null;primary_key;comment:'id'" json:"id"`
	AssetId      string `gorm:"type:varchar(128);not null;comment:'资产ID'" json:"assetId"`
	AssetGroupId string `gorm:"type:varchar(128);not null;comment:'资产组ID'" json:"assetGroupId"`
}

func (AssetGroupWithAsset) TableName() string {
	return "asset_group_with_asset"
}
