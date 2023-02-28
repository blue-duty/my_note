package model

//type CommandRelevance struct {
//	ID              string `gorm:"type:varchar(128);primary_key;not null;comment:id" json:"id"`
//	CommandPolicyId string `gorm:"type:varchar(128);index;not null;comment:指令策略id"          json:"commandPolicyId" `
//	UserId          string `gorm:"type:varchar(128);index;not null;comment:用户id"             json:"userId" `
//	UserGroupId     string `gorm:"type:varchar(128);index;not null;comment:用户组id"            json:"userGroupId" `
//	CommandSetId    string `gorm:"type:varchar(128);index;not null;comment:指令集id"          json:"commandSetId" `
//	PassportId         string `gorm:"type:varchar(128);index;not null;comment:主机id"             json:"assetId" `
//	AssetGroupId    string `gorm:"type:varchar(128);index;not null;comment:主机组id"             json:"assetGroupId" `
//}
//
//func (r *CommandRelevance) TableName() string {
//	return "command_relevances"
//}

//type CommandPolicy struct {
//	ID         string         `gorm:"type:varchar(128);primary_key;not null;comment:指令策略id" json:"id" label:"指令策略id"`
//	Name       string         `gorm:"type:varchar(64);not null;default:'';comment:策略名称" json:"name"  `
//	Created    utils.JsonTime `gorm:"type:datetime(3);index;not null;comment:时间" json:"created"`
//	Priority   int            `gorm:"type:int;not null;default:99;comment:优先级" json:"priority"  validate:"min=1,max=99"` //优先级，越小越优先
//	PolicyType string         `gorm:"type:varchar(64);not null;default:'指令阻断';comment:执行动作类别" json:"policyType"  `
//	Describe   string         `gorm:"type:varchar(4096);not null;default:'';comment:策略表描述" json:"describe"  `
//	Status     string         `gorm:"type;type:tinyint(1);default:'0';comment:开启状态"        json:"status"`
//}
//
//func (r *CommandPolicy) TableName() string {
//	return "command_policies"
//}

//type CommandSet struct {
//	ID       string         `gorm:"type:varchar(128);primary_key;not null;comment:指令集id" json:"id" label:"指令集id"`
//	Name     string         `gorm:"type:varchar(64);not null;default:'';comment:指令集名称" json:"name"  `
//	Created  utils.JsonTime `gorm:"type:datetime(3);not null;comment:时间" json:"created"`
//	Describe string         `gorm:"type:varchar(4096);not null;default:'';comment:指令集描述" json:"describe"  `
//	Content  string         `gorm:"type:varchar(4096);not null;default:'';comment:指令集内容" json:"content"  `
//}
//
//func (r *CommandSet) TableName() string {
//	return "command_sets"
//}
//
//type CommandContent struct {
//	ID        string `gorm:"type:varchar(128);primary_key;not null;comment:指令id" json:"id" `
//	ContentId string `gorm:"type:varchar(128);index;not null" json:"contentId" `
//	Content   string `gorm:"type:longtext " json:"content" label:"[指令内容输入框]" `
//	IsRegular *bool  `gorm:"type:tinyint(1);default:0;comment:是否正则"        json:"regular"`
//	Describe  string `gorm:"type:varchar(4096);not null;default:'';comment:单条指令描述" json:"describe"  `
//}
//
//func (r *CommandContent) TableName() string {
//	return "command_contents"
//}
