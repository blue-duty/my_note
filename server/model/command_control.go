package model

import "tkbastion/server/utils"

type CommandStrategy struct {
	ID              string         `json:"id" gorm:"column:id;type:varchar(36);primary_key;not null"`
	Name            string         `json:"name" gorm:"column:name;type:varchar(64);not null;default:'';comment:策略名称"`
	DepartmentId    int64          `json:"departmentId" gorm:"column:department_id;type:bigint(20);not null;default:0;comment:部门ID"`
	DepartmentName  string         `json:"departmentName" gorm:"column:department_name;type:varchar(64);not null;default:'';comment:部门名称"`
	DepartmentDepth int            `json:"departmentDepth" gorm:"column:department_depth;type:bigint(20);not null;default:0;comment:部门深度"`
	Level           string         `json:"level" gorm:"column:level;type:varchar(64);not null;default:'敏感指令';comment:策略等级"`
	Action          string         `json:"action" gorm:"column:action;type:varchar(64);not null;default:'指令申请';comment:策略动作"`
	Status          string         `json:"status" gorm:"column:status;type:varchar(64);not null;default:'';comment:策略状态"`
	Description     string         `json:"description" gorm:"column:description;type:varchar(255);not null;default:'';comment:策略描述"`
	Priority        int64          `json:"priority" gorm:"column:priority;type:bigint(20);not null;default:0;comment:策略优先级"`
	AlarmByEmail    *bool          `json:"alarmByEmail" gorm:"column:alarm_by_email;type:tinyint(1);not null;default:0;comment:邮件告警"`
	AlarmByMessage  *bool          `json:"alarmByMessage" gorm:"column:alarm_by_message;type:tinyint(1);not null;default:0;comment:消息告警"`
	AlarmByPhone    *bool          `json:"alarmByPhone" gorm:"column:alarm_by_phone;type:tinyint(1);not null;default:0;comment:电话告警"`
	IsPermanent     *bool          `json:"isPermanent" gorm:"column:is_permanent;type:tinyint(1);not null;default:0;comment:是否永久有效"`
	BeginValidTime  utils.JsonTime `gorm:"type:datetime(3);comment:'开始时间'" json:"beginValidTime"`
	EndValidTime    utils.JsonTime `gorm:"type:datetime(3);comment:'结束时间'" json:"endValidTime"`
	CreateTime      utils.JsonTime `gorm:"type:datetime(3);comment:'创建时间'" json:"createTime"`
}

func (r *CommandStrategy) TableName() string {
	return "command_strategy"
}

type CommandRelevance struct {
	ID                string `gorm:"type:varchar(128);primary_key;not null;comment:id" json:"id"`
	CommandStrategyId string `gorm:"type:varchar(128);not null;comment:策略id" json:"commandStrategyId"`
	UserId            string `gorm:"type:varchar(128);index;default:'-';comment:用户id"             json:"userId" `
	UserGroupId       string `gorm:"type:varchar(128);index;default:'-';comment:用户组id"            json:"userGroupId" `
	CommandSetId      string `gorm:"type:varchar(128);index;default:'-';comment:指令集id"          json:"commandSetId" `
	AssetId           string `gorm:"type:varchar(128);index;default:'-';comment:主机id"             json:"assetId" `
	AssetGroupId      string `gorm:"type:varchar(128);index;default:'-';comment:主机组id"             json:"assetGroupId" `
}

func (r *CommandRelevance) TableName() string {
	return "command_relevances"
}

type CommandSet struct {
	ID          string         `gorm:"type:varchar(128);primary_key;not null;comment:指令集id" json:"id" label:"指令集id"`
	Name        string         `gorm:"type:varchar(64);not null;default:'';comment:指令集名称" json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	Level       string         `gorm:"type:varchar(64);not null;default:'敏感指令';comment:指令集等级" json:"level"  `
	Created     utils.JsonTime `gorm:"type:datetime(3);not null;comment:时间" json:"created"`
	Description string         `gorm:"type:varchar(255);not null;default:'';comment:指令集描述" json:"description" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
	Content     string         `gorm:"type:varchar(4096);not null;default:'';comment:指令集内容" json:"content" `
}

func (r *CommandSet) TableName() string {
	return "command_sets"
}

type CommandContent struct {
	ID          string `gorm:"type:varchar(128);primary_key;not null;comment:指令id" json:"id" `
	ContentId   string `gorm:"type:varchar(128);index;not null" json:"contentId" `
	Content     string `gorm:"type:varchar(128) " json:"content" label:"[指令内容输入框]" `
	IsRegular   *bool  `gorm:"type:tinyint(1);default:0;comment:是否正则"        json:"isRegular"`
	Description string `gorm:"type:varchar(255);not null;default:'';comment:指令描述" json:"description" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
}

func (r *CommandContent) TableName() string {
	return "command_contents"
}

type CommandPolicyConfig struct {
	ApprovalTimeout int    `json:"approvalTimeout" `
	ExpiredAction   string `json:"expiredAction" `
}
type CommandStrategyPriority struct {
	StrategyPriority string `json:"strategyPriority" `
}

type CommandRecord struct {
	ID        string         `gorm:"type:varchar(128);primary_key;not null;comment:id" json:"id"`
	AssetId   string         `gorm:"type:varchar(128);not null;comment:主机id" json:"assetId"`
	SessionId string         `gorm:"type:varchar(128);not null;comment:会话id" json:"sessionId"`
	AssetName string         `gorm:"type:varchar(128);not null;comment:主机名称" json:"assetName"`
	AssetIp   string         `gorm:"type:varchar(128);not null;comment:主机ip" json:"assetIp"`
	ClientIp  string         `gorm:"type:varchar(128);not null;comment:客户端ip" json:"clientIp"`
	Passport  string         `gorm:"type:varchar(128);not null;comment:账号" json:"passport"`
	Username  string         `gorm:"type:varchar(128);not null;comment:用户名" json:"username"`
	Nickname  string         `gorm:"type:varchar(128);not null;comment:用户昵称" json:"nickname"`
	Created   utils.JsonTime `gorm:"type:datetime(3);not null;comment:时间" json:"created"`
	Protocol  string         `gorm:"type:varchar(128);not null;comment:协议" json:"protocol"`
	Content   string         `gorm:"type:varchar(1024);not null;comment:指令内容" json:"content"`
}

func (r *CommandRecord) TableName() string {
	return "command_records"
}

type RecordCommandAnalysis struct {
	Time    string `json:"time"`
	Command string `json:"command"`
	Count   int    `json:"count"`
}
