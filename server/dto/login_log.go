package dto

import "tkbastion/server/utils"

type LoginLogForPage struct {
	ID              string         `json:"id"`
	Username        string         `json:"username"`
	Nickname        string         `json:"nickname"`
	ClientIP        string         `json:"clientIp"`
	ClientUserAgent string         `json:"clientUserAgent"`
	LoginTime       utils.JsonTime `json:"loginTime"`
	LogoutTime      utils.JsonTime `json:"logoutTime"`
	LoginResult     string         `json:"loginResult"`
	LoginType       string         `json:"loginType"`
	Protocol        string         `json:"protocol"`
	Description     string         `json:"description"`
	Source          string         `json:"source"`
}
