package model

// CasbinRule TODO 这里多个字段定义为一个唯一性索引代码有bug,目前会把每个字段定义为一个唯一性索引
/*type CasbinRule struct {
	PType string `gorm:"type:varchar(100);unique_index:casbin_rule;not null;comment:策略"`
	V0    string `gorm:"type:varchar(100);unique_index:casbin_rule;not null;comment:角色名"`
	V1    string `gorm:"type:varchar(100);unique_index:casbin_rule;not null;comment:路由"`
	V2    string `gorm:"type:varchar(100);unique_index:casbin_rule;not null;comment:HTTP方法"`
	V3    string `gorm:"type:varchar(100);unique_index:casbin_rule;default:null;comment:备选字段,若现有V0、V1、V2字段组合不满足新的访问控制规则,可启用此字段生成更多权限控制规则组合,实现更细致的权限控制"`
	V4    string `gorm:"type:varchar(100);unique_index:casbin_rule;default:null;comment:备选字段,同V3"`
	V5    string `gorm:"type:varchar(100);unique_index:casbin_rule;default:null;comment:备选字段,同V3"`
}*/
type CasbinRule struct {
	PType string `gorm:"type:varchar(100);unique_index:casbin_rule;not null;default:'';comment:策略"`
	V0    string `gorm:"type:varchar(100);unique_index:casbin_rule;not null;default:'';comment:角色名"`
	V1    string `gorm:"type:varchar(100);unique_index:casbin_rule;not null;default:'';comment:路由"`
	V2    string `gorm:"type:varchar(100);unique_index:casbin_rule;not null;default:'';comment:HTTP方法"`
	V3    string `gorm:"type:varchar(100);unique_index:casbin_rule;not null;default:'';comment:备选字段,若现有V0、V1、V2字段组合不满足新的访问控制规则,可启用此字段生成更多权限控制规则组合,实现更细致的权限控制"`
	V4    string `gorm:"type:varchar(100);unique_index:casbin_rule;not null;default:'';comment:备选字段,同V3"`
	V5    string `gorm:"type:varchar(100);unique_index:casbin_rule;not null;default:'';comment:备选字段,同V3"`
}

func (r *CasbinRule) TableName() string {
	return "casbin_rule"
}

// TODO 这里多个字段定义为一个唯一性索引代码有bug,目前会把每个字段定义为一个唯一性索引
type CasbinApi struct {
	MenuId   int    `gorm:"type:int(3);not null;comment:路由所在菜单结点id"`
	Path     string `gorm:"type:varchar(128);unique_index:casbin_api;not null;comment:路由"`
	Action   string `gorm:"type:varchar(64);unique_index:casbin_api;not null;comment:HTTP方法"`
	LogTypes string `gorm:"type:varchar(16);not null;comment:api所属日志类型"`
}

func (r *CasbinApi) TableName() string {
	return "casbin_api"
}
