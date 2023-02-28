package model

import "strconv"

type ChangePassword struct {
	OldPassword     string `json:"oldPassword"     label:"原有密码"`
	NewPassword     string `json:"newPassword"     label:"新密码" `
	ConfirmPassword string `json:"confirmPassword" label:"确认密码"`
}

type ChangePersonalInfo struct {
	Nickname string `json:"nickname"     label:"昵称"        `
	Email    string `json:"email"     label:"邮箱"  `
	Phone    string `json:"phone"     label:"手机号"  `
	QQ       string `json:"qq"     label:"QQ"    `
}

type IdentityConfig struct {
	ID                  int    `gorm:"primary_key;AUTO_INCREMENT;comment:鉴别策略配置id" json:"id" map:"key:id"`
	LoginLockWay        string `gorm:"type:varchar(128);comment:登录锁定方式" json:"loginLockWay" map:"key:login_lock_way"`
	AttemptTimes        int    `gorm:"type:int;default:5;comment:登录失败次数" json:"attemptTimes" map:"key:attempt_times"`
	ContinuousTime      int    `gorm:"type:int;default:2;comment:连续登录失败次数" json:"continuousTime" map:"key:continuous_time"`
	LockTime            int    `gorm:"type:int;default:30;comment:锁定时间" json:"lockTime" map:"key:lock_time"`
	LockIp              string `gorm:"type:varchar(1024);comment:锁定IP" json:"lockIp" map:"key:lock_ip"`
	ForceChangePassword *bool  `gorm:"type:tinyint(1);default:0;comment:强制修改密码" json:"forceChangePassword" map:"key:force_change_password"`
	PasswordLength      int    `gorm:"type:int;default:8;comment:密码长度" json:"passwordLength" map:"key:password_length"`
	PasswordCheck       *bool  `gorm:"type:tinyint(1);default:0;comment:是否校验密码" json:"passwordCheck" map:"key:password_check"`
	PasswordSameTimes   int    `gorm:"type:int;default:3;comment:密码相同次数" json:"passwordSameTimes" map:"key:password_same_times"`
	PasswordCycle       int    `gorm:"type:int;default:0;comment:密码循环周期" json:"passwordCycle" map:"key:password_cycle;empty"`
	PasswordRemind      int    `gorm:"type:int;default:0;comment:密码提醒周期" json:"passwordRemind" map:"key:password_remind;empty"`
}

type LoginConfig struct {
	ID             int    `gorm:"primary_key;AUTO_INCREMENT;comment:鉴别策略配置id" json:"id" `
	LoginLockWay   string `gorm:"type:varchar(128);comment:登录锁定方式" json:"loginLockWay" `
	AttemptTimes   int    `gorm:"type:int;default:5;comment:登录失败次数" json:"attemptTimes" `
	ContinuousTime int    `gorm:"type:int;default:2;comment:连续登录失败时间" json:"continuousTime" `
	LockTime       int    `gorm:"type:int;default:30;comment:锁定时间" json:"lockTime" `
	LockIp         string `gorm:"type:varchar(1024);comment:锁定IP" json:"lockIp" `
}

type PasswordConfig struct {
	ID                  int   `gorm:"primary_key;AUTO_INCREMENT;comment:鉴别策略配置id" json:"id" `
	ForceChangePassword *bool `gorm:"type:tinyint(1);default:0;comment:强制修改密码" json:"forceChangePassword" `
	PasswordLength      int   `gorm:"type:int;default:8;comment:密码长度" json:"passwordLength" `
	PasswordCheck       *bool `gorm:"type:tinyint(1);default:0;comment:是否校验密码" json:"passwordCheck" `
	PasswordSameTimes   int   `gorm:"type:int;default:3;comment:密码相同次数" json:"passwordSameTimes" `
	PasswordCycle       int   `gorm:"type:int;default:0;comment:密码循环周期" json:"passwordCycle" `
	PasswordRemind      int   `gorm:"type:int;default:0;comment:密码提醒周期" json:"passwordRemind" `
}

// ExtendConfig 扩展配置
type ExtendConfig struct {
	ID          string `gorm:"primary_key;type:varchar(128);comment:扩展配置id" json:"id" `
	Name        string `gorm:"type:varchar(128);comment:扩展配置名称" json:"name" validate:"required"`
	LinkAddress string `gorm:"type:varchar(128);comment:扩展配置链接地址" json:"linkAddress" validate:"required" `
	Priority    int    `gorm:"type:int;default:0;comment:优先级" json:"priority" validate:"required,max=20,min=0" `
}

type ExtendConfigDTO struct {
	ID          string `gorm:"primary_key;type:varchar(128);comment:扩展配置id" json:"id" `
	Name        string `gorm:"type:varchar(128);comment:扩展配置名称" json:"name" validate:"required" validate:"required,max=10,regexp=^[\\p{Han}\\p{L}]+$"`
	LinkAddress string `gorm:"type:varchar(128);comment:扩展配置链接地址" json:"linkAddress" validate:"required" validate:"required,max=128,regexp=^http(s)?://([\\w-]+\\.)+[\\w-]+(/[\\w- ./?%&=]*)?$"`
	Priority    string `gorm:"type:int;default:0;comment:优先级" json:"priority" validate:"required" `
}

func (r *ExtendConfigDTO) DTOtoExtendConfig() ExtendConfig {
	priority, _ := strconv.Atoi(r.Priority)
	return ExtendConfig{
		ID:          r.ID,
		Name:        r.Name,
		LinkAddress: r.LinkAddress,
		Priority:    priority,
	}
}
func (r *ExtendConfig) ExtendConfigToDTO() ExtendConfigDTO {
	return ExtendConfigDTO{
		ID:          r.ID,
		Name:        r.Name,
		LinkAddress: r.LinkAddress,
		Priority:    strconv.Itoa(r.Priority),
	}
}

func (p *ExtendConfig) TableName() string {
	return "extend_config"
}
