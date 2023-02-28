package model

import (
	"tkbastion/server/utils"
)

type LoginLog struct {
	ID              string         `gorm:"type:varchar(156);primary_key;not null;comment:登录用户token" json:"id"`
	UserId          string         `gorm:"type:varchar(128);index;not null;comment:登录用户id" json:"userId"`
	Username        string         `gorm:"type:varchar(128);comment:登录用户名" json:"username"`
	Nickname        string         `gorm:"type:varchar(128);comment:登录用户昵称" json:"nickname"`
	DepartmentName  string         `gorm:"type:varchar(128);comment:登录用户部门" json:"departmentName"`
	DepartmentId    int64          `gorm:"type:bigint(20);comment:登录用户部门id" json:"departmentId"`
	ClientIP        string         `gorm:"type:varchar(64);default:'';comment:用户ip" json:"clientIp"`
	ClientUserAgent string         `gorm:"type:varchar(512);default:'';comment:用户客户端信息" json:"clientUserAgent"`
	LoginTime       utils.JsonTime `gorm:"type:datetime(3);not null;comment:登录时间" json:"loginTime"`
	LogoutTime      utils.JsonTime `gorm:"type:datetime(3);default:null;comment:登出时间" json:"logoutTime"`
	LoginResult     string         `gorm:"type:varchar(8);default:null;comment:登录结果" json:"loginResult"`
	Remember        bool           `gorm:"type:tinyint(1);not null;comment:是否记住用户" json:"remember"`
	Protocol        string         `gorm:"type:varchar(8);default:null;comment:协议" json:"protocol"`
	LoginType       string         `gorm:"type:varchar(32);default:null;comment:登录方式" json:"loginType"`
	Description     string         `gorm:"type:varchar(512);default:null;comment:描述" json:"description"`
	Source          string         `gorm:"type:varchar(32);default:null;comment:来源" json:"source"`
}

type LoginLogForPage struct {
	ID              string         `json:"id"`
	UserId          string         `json:"userId"`
	UserName        string         `json:"userName"`
	ClientIP        string         `json:"clientIp"`
	ClientUserAgent string         `json:"clientUserAgent"`
	LoginTime       utils.JsonTime `json:"loginTime"`
	LogoutTime      utils.JsonTime `json:"logoutTime"`
	LoginResult     string         `json:"loginResult"`
	Remember        bool           `json:"remember"`
	Source          string         ` json:"source"`
}

type LoginLogForPageNew struct {
	ID              string         `json:"id"`
	UserId          string         `json:"userId"`
	Username        string         `json:"username"`
	Nickname        string         `json:"nickname"`
	ClientIP        string         `json:"clientIp"`
	ClientUserAgent string         `json:"clientUserAgent"`
	LoginTime       utils.JsonTime `json:"loginTime"`
	LogoutTime      utils.JsonTime `json:"logoutTime"`
	LoginResult     string         `json:"loginResult"`
	Remember        bool           `json:"remember"`
	Source          string         ` json:"source"`
	Protocol        string         `json:"protocol"`
	LoginType       string         `json:"loginType"`
	Description     string         `json:"description"`
}

func (r *LoginLog) TableName() string {
	return "login_logs"
}
