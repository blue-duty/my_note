package model

import "tkbastion/server/utils"

// 协议访问
type ProtocolCountByDay struct {
	Daytime string `json:"dayTime"`
	Ssh     int    `json:"ssh"`
	Rdp     int    `json:"rdp"`
	Telnet  int    `json:"telnet"`
	Vnc     int    `json:"vnc"`
	App     int    `json:"app"`
	Tcp     int    `json:"tcp"`
	Ftp     int    `json:"ftp"`
	Sftp    int    `json:"sftp"`
	Total   int    `json:"total"`
}

// 协议访问
type ProtocolCountExport struct {
	Daytime string `json:"dayTime"`
	Ssh     int    `json:"ssh"`
	Rdp     int    `json:"rdp"`
	Telnet  int    `json:"telnet"`
	Vnc     int    `json:"vnc"`
	App     int    `json:"app"`
	Tcp     int    `json:"tcp"`
	Total   int    `json:"total"`
}

// 详细数据
type LoginDetails struct {
	DayTime     string `json:"dayTime"`
	Username    string `json:"username"`
	Nickname    string `json:"nickname"`
	SourceIp    string `json:"sourceIp"`
	Protocol    string `json:"protocol"`
	Result      string `json:"result"`
	Description string `json:"description"`
}

type SessionDetailsInfo struct {
	AssetIP          string         `json:"assetIP"`
	ConnectedTime    utils.JsonTime `json:"connectedTime"`
	DisconnectedTime string         `json:"disconnectedTime"`
	CreateName       string         `json:"createName"`
	CreateNick       string         `json:"createNick"`
	Protocol         string         `json:"protocol"`
	Recording        string         `json:"recording"`
	Result           string         `json:"result"`
	Description      string         `json:"description"`
}

func (r *SessionDetailsInfo) ToLoginDetailsDto() *LoginDetails {
	if r.DisconnectedTime == "" {
		r.Result = "失败"
	} else {
		r.Result = "成功"
	}
	return &LoginDetails{
		DayTime:     r.ConnectedTime.Format("2006-01-02 15:04:05"),
		SourceIp:    r.AssetIP,
		Username:    r.CreateName,
		Nickname:    r.CreateNick,
		Protocol:    r.Protocol,
		Result:      r.Result,
		Description: r.Description,
	}
}

type LoginDetailsInfo struct {
	Username    string         `json:"username"`
	Nickname    string         `json:"nickname"`
	ClientIP    string         `json:"clientIP"`
	LoginTime   utils.JsonTime `json:"loginTime"`
	LoginResult string         `json:"loginResult"`
	Protocol    string         `json:"protocol"`
	LoginType   string         `json:"loginType"`
	Description string         `json:"description"`
}

func (r *LoginDetailsInfo) ToLoginDetailsDto() *LoginDetails {
	return &LoginDetails{
		DayTime:     r.LoginTime.Format("2006-01-02 15:04:05"),
		SourceIp:    r.ClientIP,
		Username:    r.Username,
		Nickname:    r.Nickname,
		Protocol:    r.Protocol,
		Result:      r.LoginResult,
		Description: r.Description,
	}
}
