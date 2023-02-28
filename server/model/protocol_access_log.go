package model

import "tkbastion/server/utils"

type ProtocolAccessLog struct {
	Id        int64          `gorm:"primary_key;type:bigint(20) AUTO_INCREMENT;not null;comment:协议访问日志表ID" json:"id"`
	Daytime   utils.JsonTime `gorm:"type:datetime(3);not null;comment:日期" json:"daytime"`
	Protocol  string         `gorm:"type:varchar(128);not null;comment:协议" json:"protocol"`
	ClientIp  string         `gorm:"type:varchar(128);not null;comment:来源IP" json:"clientIp"`
	UserId    string         `gorm:"type:varchar(128);not null;comment:用户ID" json:"userId"`
	Username  string         `gorm:"type:varchar(128);not null;comment:用户名" json:"username"`
	Nickname  string         `gorm:"type:varchar(128);not null;comment:用户昵称" json:"nickname"`
	SessionId string         `gorm:"type:varchar(128);not null;comment:会话ID" json:"sessionId"`
	Result    string         `gorm:"type:char(10);not null;comment:结果" json:"result"`
	Info      string         `gorm:"type:varchar(128);comment:信息" json:"info"`
}

func (ProtocolAccessLog) TableName() string {
	return "protocol_access_log"
}
