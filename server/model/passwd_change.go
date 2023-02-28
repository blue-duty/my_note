package model

import "tkbastion/server/utils"

// PasswdChange 改密策略
type PasswdChange struct {
	ID      string `gorm:"primary_key;column:id;type:varchar(128);not null;comment:策略id" json:"id" map:"key:id"`
	Name    string `gorm:"column:name;type:varchar(255);not null " json:"name" map:"key:name"`
	RunType string `gorm:"type:varchar(16);not null;comment:运行时间类型" json:"run_type" map:"key:run_type"` // 1. Manual 2. Scheduled 3. Periodic
	// 执行方式名称
	RunTypeName  string         `gorm:"type:varchar(255);not null;comment:执行方式名称" json:"run_type_name" map:"key:run_type_name"` // 1. Manual 2. Scheduled 3. Periodic
	RunTime      utils.JsonTime `gorm:"type:datetime(3);comment:运行时间" json:"run_time" map:"key:run_time;empty"`
	StartAt      utils.JsonTime `gorm:"type:datetime(3);comment:开始时间" json:"start_at" map:"key:start_at;empty"`
	EndAt        utils.JsonTime `gorm:"type:datetime(3);comment:结束时间" json:"end_at" map:"key:end_at;empty"`
	PeriodicType string         `gorm:"type:varchar(16);not null;comment:周期类型" json:"periodic_type" map:"key:periodic_type;empty"` // 1. Day 2. Week 3. Month 4. Minute 5. Hour
	Periodic     int            `gorm:"type:integer;comment:周期" json:"periodic" map:"key:periodic;empty"`
	// 是否开启密码复杂度策略
	IsComplexity *bool `gorm:"type:tinyint(1);not null;default:0;comment:是否开启密码复杂度策略" json:"is_complexity" map:"key:is_complexity;empty"`
	// 密码最小长度
	MinLength int `gorm:"column:min_length;type:int(11);not null;default:0" json:"min_length" map:"key:min_length;empty"`
	// 生成规则 1. 生成相同密码 2. 生成不同密码 3. 指定密码
	GenerateRule int `gorm:"column:generate_rule;type:tinyint(1);not null;default:0" json:"generate_rule" map:"key:generate_rule"`
	// 生成规则名称
	GenerateRuleName string `gorm:"column:generate_rule_name;type:varchar(255);not null;default:''" json:"generate_rule_name" map:"key:generate_rule_name"`
	// 指定密码
	Password string `gorm:"column:password;type:varchar(255);not null;default:''" json:"password" map:"key:password;empty"`
	// 优先使用特权账号
	IsPrivilege *bool  `gorm:"type:tinyint(1);not null;default:0;comment:优先使用特权账号" json:"is_privilege" map:"key:is_privilege;empty"`
	DriveIds    string `gorm:"-" json:"drive_ids"`
	// 设备组id
	DriveGroups string `gorm:"-" json:"drive_groups"`
}

// TableName 表名
func (PasswdChange) TableName() string {
	return "passwd_change"
}

// PasswdChangeDevice 改密策略关联设备
type PasswdChangeDevice struct {
	ID             int64  `gorm:"type:bigint(20) AUTO_INCREMENT;not null;comment:id"`
	PasswdChangeID string `gorm:"column:passwd_change_id;type:varchar(128);not null" json:"passwd_change_id"`
	DeviceID       string `gorm:"column:device_id;type:varchar(128);not null" json:"device_id"`
}

// TableName 表名
func (PasswdChangeDevice) TableName() string {
	return "passwd_change_device"
}

// PasswdChangeDeviceGroup 改密策略关联设备组
type PasswdChangeDeviceGroup struct {
	ID             int64  `gorm:"type:bigint(20) AUTO_INCREMENT;not null;comment:id"`
	PasswdChangeID string `gorm:"column:passwd_change_id;type:varchar(128);not null" json:"passwd_change_id"`
	DeviceGroupID  string `gorm:"column:device_group_id;type:varchar(128);not null" json:"device_group_id"`
}

// TableName 表名
func (PasswdChangeDeviceGroup) TableName() string {
	return "passwd_change_device_group"
}

// PasswdChangeResult 改密结果记录
type PasswdChangeResult struct {
	ID int64 `gorm:"type:bigint(20) AUTO_INCREMENT;not null;comment:id"`
	// 改密策略ID
	PasswdChangeID string `gorm:"column:passwd_change_id;type:varchar(128);not null" json:"passwd_change_id"`
	// 账号ID
	AccountID string `gorm:"column:account_id;type:varchar(255);not null;default:''" json:"account_id"`
	// 设备ID
	DeviceID string `gorm:"column:device_id;type:varchar(255);not null;default:''" json:"device_id"`
	// 修改结果 1. 成功 2. 失败
	Result string `gorm:"column:result;type:char(10);not null;default:''" json:"result"`
	// 结果描述
	ResultDesc string `gorm:"column:result_desc;type:varchar(255);not null;default:''" json:"result_desc"`
	// 修改时间
	ChangeTime utils.JsonTime `gorm:"column:change_time;type:datetime(3);not null;" json:"change_time"`
}

// TableName 表名
func (PasswdChangeResult) TableName() string {
	return "passwd_change_result"
}
