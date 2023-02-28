package model

import (
	"tkbastion/server/utils"
)

type Message struct {
	ID        string         `json:"id" gorm:"column:id;type:varchar(128);primary_key;not null"`
	ReceiveId string         `json:"receiveId" gorm:"column:receive_id;type:varchar(128);not null"`
	Theme     string         `json:"theme" gorm:"column:theme;type:varchar(128);not null"`
	Level     string         `json:"level" gorm:"column:level;type:varchar(8);not null"`
	Content   string         `json:"content"  gorm:"column:content;type:varchar(2048);not null"`
	Status    bool           `json:"status"  gorm:"column:status;type:tinyint(1);not null"`
	Type      string         `json:"type" gorm:"column:type;type:varchar(8);not null"`
	Created   utils.JsonTime `json:"created" gorm:"column:created;type:datetime(3);not null"`
}

func (r *Message) TableName() string {
	return "message"
}

type Recipients struct {
	ID          string `json:"id"`
	Theme       string `json:"theme"`
	UserId      string `json:"user_id"`
	UserGroupId string `json:"user_group_id"`
}

func (r *Recipients) TableName() string {
	return "recipients"
}

type AlarmConfig struct {
	Event          string `json:"event"`
	ThresholdValue int    `json:"thresholdValue"`
	IsMail         *bool  `json:"isMail"`
	IsMessage      *bool  `json:"isMessage"`
	IsSyslog       *bool  `json:"isSyslog"`
	AlarmLevel     string `json:"alarmLevel"`
}

type AlarmPerformanceConfig struct {
	CpuMax    string `json:"cpuMax"`
	CpuMsg    *bool  `json:"cpuMsg"`
	CpuMail   *bool  `json:"cpuMail"`
	CpuSyslog *bool  `json:"cpuSyslog"`
	CpuLevel  string `json:"cpuLevel"`

	MemMax    string `json:"memMax"`
	MemMsg    *bool  `json:"memMsg"`
	MemMail   *bool  `json:"memMail"`
	MemSyslog *bool  `json:"memSyslog"`
	MemLevel  string `json:"memLevel"`

	DiskMax    string `json:"diskMax"`
	DiskMsg    *bool  `json:"diskMsg"`
	DiskMail   *bool  `json:"diskMail"`
	DiskSyslog *bool  `json:"diskSyslog"`
	DiskLevel  string `json:"diskLevel"`

	VisualMemMax    string `json:"visualMemMax"`
	VisualMemMsg    *bool  `json:"visualMemMsg"`
	VisualMemMail   *bool  `json:"visualMemMail"`
	VisualMemSyslog *bool  `json:"visualMemSyslog"`
	VisualMemLevel  string `json:"visualMemLevel"`
}

type AlarmUserAccessConfig struct {
	UserMax    string `json:"userMax"`
	UserMsg    *bool  `json:"userMsg"`
	UserMail   *bool  `json:"userMail"`
	UserSyslog *bool  `json:"userSyslog"`
	UserLevel  string `json:"userLevel"`

	SshMax    string `json:"sshMax"`
	SshMsg    *bool  `json:"sshMsg"`
	SshMail   *bool  `json:"sshMail"`
	SshSyslog *bool  `json:"sshSyslog"`
	SshLevel  string `json:"sshLevel"`

	RdpMax    string `json:"rdpMax"`
	RdpMsg    *bool  `json:"rdpMsg"`
	RdpMail   *bool  `json:"rdpMail"`
	RdpSyslog *bool  `json:"rdpSyslog"`
	RdpLevel  string `json:"rdpLevel"`

	TelnetMax    string `json:"telnetMax"`
	TelnetMsg    *bool  `json:"telnetMsg"`
	TelnetMail   *bool  `json:"telnetMail"`
	TelnetSyslog *bool  `json:"telnetSyslog"`
	TelnetLevel  string `json:"telnetLevel"`

	VncMax    string `json:"vncMax"`
	VncMsg    *bool  `json:"vncMsg"`
	VncMail   *bool  `json:"vncMail"`
	VncSyslog *bool  `json:"vncSyslog"`
	VncLevel  string `json:"vncLevel"`

	AppMax    string `json:"appMax"`
	AppMsg    *bool  `json:"appMsg"`
	AppMail   *bool  `json:"appMail"`
	AppSyslog *bool  `json:"appSyslog"`
	AppLevel  string `json:"appLevel"`
}
