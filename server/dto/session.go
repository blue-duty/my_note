package dto

type SessionForSearch struct {
	Auto          string  `json:"auto"`
	AssetName     string  `json:"assetName"`   // 设备名称
	AssetIP       string  `json:"assetIp"`     // 设备地址
	Protocol      string  `json:"protocol"`    // 协议
	Passport      string  `json:"passport"`    // 账号
	IP            string  `json:"ip"`          // 来源地址
	UserName      string  `json:"userName"`    // 用户名
	OperateTime   string  `json:"operateTime"` // 操作时间
	Command       string  `json:"command"`     // 命令
	DepartmentIds []int64 // 部门id
}

type AppSessionForSearch struct {
	Auto          string `json:"auto"`
	AppName       string `json:"appName"`  // 应用名称
	Program       string `json:"program"`  // 程序名称
	Passport      string `json:"passport"` // 账号
	IP            string `json:"ip"`       // 来源地址
	UserName      string `json:"userName"` // 用户名
	NickName      string `json:"nickName"` // 昵称
	DepartmentIds []int64
}

type AppSessionForExport struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	ClientIp  string `json:"clientIp"`
	AppName   string `json:"appName"`
	Program   string `json:"program"`
	Passport  string `json:"passport"`
}

type AppSessionForPage struct {
	ID             string `json:"id"`
	AppName        string `json:"appName"`
	Program        string `json:"program"`
	Passport       string `json:"passport"`
	IP             string `json:"ip"`
	UserName       string `json:"userName"`
	NickName       string `json:"nickName"`
	StartTime      string `json:"startTime"`
	EndTime        string `json:"endTime"`
	DownloadStatus string `json:"downloadStatus"`
	IsRecord       bool   `json:"isRecord"`
	IsRead         bool   `json:"isRead"`
}

type AppSessionForDetail struct {
	AppName           string `json:"appName"`
	IP                string `json:"ip"`
	Passport          string `json:"passport"`
	Program           string `json:"program"`
	StartAndEnd       string `json:"startandend"`   // 开始时间和结束时间
	SessionTime       string `json:"sessionTime"`   // 会话时长
	SessionSzie       string `json:"sessionSize"`   // 会话大小
	UserId            string `json:"userId"`        // 用户id
	UserName          string `json:"userName"`      // 用户名
	NickName          string `json:"nickName"`      // 昵称
	CilentIP          string `json:"cilentIp"`      // 来源地址
	AuthenticationWay string `json:"authentionWay"` //认证类型
}

type SessionForPage struct {
	ID           string `json:"id"`           // 会话id
	AssetName    string `json:"assetName"`    // 设备名称
	AssetIP      string `json:"assetIp"`      // 设备地址
	Protocol     string `json:"protocol"`     // 协议
	Passport     string `json:"passport"`     // 账号
	IP           string `json:"ip"`           // 来源地址
	UserName     string `json:"userName"`     // 用户名
	UserNickname string `json:"userNickname"` // 用户昵称
	Height       int    `json:"height"`       // 高度
	IsRecordings bool   `json:"isRecording"`  // 是否录像
	IsReply      bool   `json:"isReply"`      // 是否回放
	Width        int    `json:"width"`        // 宽度
	StartTime    string `json:"startTime"`    // 开始时间
	EndTime      string `json:"endTime"`      // 结束时间
}

type SessionDetail struct {
	AssetName         string `json:"assetName"`     // 设备名称
	Protocol          string `json:"protocol"`      // 协议
	AssetIP           string `json:"assetIp"`       // 设备地址
	Passport          string `json:"passport"`      // 账号
	StartAndEnd       string `json:"startandend"`   // 开始时间和结束时间
	SessionTime       string `json:"sessionTime"`   // 会话时长
	SessionSzie       string `json:"sessionSize"`   // 会话大小
	UserId            string `json:"userId"`        // 用户id
	UserName          string `json:"userName"`      // 用户名
	NickName          string `json:"nickName"`      // 昵称
	CilentIP          string `json:"cilentIp"`      // 来源地址
	AuthenticationWay string `json:"authentionWay"` //认证类型
}

// SessionOnline 在线会话
type SessionOnline struct {
	ID           string `json:"id"`           // 会话id
	AssetName    string `json:"assetName"`    // 设备名称
	AssetIP      string `json:"assetIp"`      // 设备地址
	Protocol     string `json:"protocol"`     // 协议
	Passport     string `json:"passport"`     // 账号
	IP           string `json:"ip"`           // 来源地址
	UserName     string `json:"userName"`     // 用户名
	UserNickname string `json:"userNickname"` // 用户昵称
	Height       int    `json:"height"`       // 高度
	Width        int    `json:"width"`        // 宽度
	StartTIme    string `json:"startTime"`    // 开始时间
}

type ClipboardRecord struct {
	Content  string `json:"content"`  // 剪切板内容
	ClipTime string `json:"clipTime"` // 剪切板时间
}
