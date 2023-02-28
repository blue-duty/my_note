package dto

type UserAccessTotal struct {
	Username string `json:"username"`
	Totals   int64  `json:"totals"`
}

// UserAccessStatistics 用户访问统计
type UserAccessStatistics struct {
	Daytime  string `json:"dayTime"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	ClientIP string `json:"sourceIp"`
	Protocol string `json:"protocol"`
	Result   string `json:"result"`
	Info     string `json:"description"`
}

// LoginAttemptStatistics 登陆尝试统计
type LoginAttemptStatistics struct {
	Daytime string `json:"daytime"`
	UserNum int64  `json:"userNum"`
	Success int64  `json:"success"`
	Failure int64  `json:"failure"`
	IpNum   int64  `json:"ipNum"`
	Totals  int64  `json:"totals"`
}

// LoginAttemptDetail 登陆尝试详细信息
type LoginAttemptDetail struct {
	Daytime  string `json:"daytime"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	ClientIP string `json:"clientIp"`
	Result   string `json:"result"`
	Info     string `json:"info"`
}

// LoginAttemptUserDetail 登陆尝试用户详细信息
type LoginAttemptUserDetail struct {
	Username   string `json:"username"`
	Nickname   string `json:"nickname"`
	Department string `json:"department"`
	Role       string `json:"role"`
}

// UserProtocolAccessCountStatistics 用户访问统计
type UserProtocolAccessCountStatistics struct {
	Daytime  string `json:"daytime"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Ssh      int64  `json:"ssh"`
	Rdp      int64  `json:"rdp"`
	Telnet   int64  `json:"telnet"`
	Vnc      int64  `json:"vnc"`
	App      int64  `json:"app"`
	Tcp      int64  `json:"tcp"`
	Total    int64  `json:"total"`
}
