package model

import "tkbastion/server/utils"

type ResourceSharer struct {
	ID           string `gorm:"type:varchar(128);primary_key;not null;comment:资源、资源类型、用户id拼接的唯一id" json:"id"`
	ResourceId   string `gorm:"type:varchar(128);index;not null;comment:资源id" json:"resourceId"`
	ResourceType string `gorm:"type:varchar(16);index;not null;comment:资源类型" json:"resourceType"`
	//StrategyId   string         `gorm:"type:varchar(128);index;notnull;comment:策略配置id" json:"strategyId"`
	UserId      string         `gorm:"type:varchar(128);index;not null;comment:可共享此资源的用户id(被授权访问的用户id)" json:"userId"`
	UserGroupId string         `gorm:"type:varchar(128);index;default:'';comment:被授权访问的用户所属用户组id" json:"userGroupId"`
	StartTime   utils.JsonTime `gorm:"type:datetime(3);comment:'开始时间'" json:"startTime"`
	EndTime     utils.JsonTime `gorm:"type:datetime(3);comment:'结束时间'" json:"endTime"`
}

func (r *ResourceSharer) TableName() string {
	return "resource_sharers"
}
