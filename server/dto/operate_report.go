package dto

// AssetAccess 资产运维
type AssetAccess struct {
	Time     string `json:"time"`
	AssetNum int    `json:"assetNum"`
	UserNum  int    `json:"userNum"`
}

// AssetSession 设备会话时长
type AssetSession struct {
	AssetName string `json:"assetName"`
	AssetIP   string `json:"assetIP"`
	Time      int64  `json:"time"`
	Name      string `json:"name"`
}

//// AssetSessionDetail 资产运维详情
//type AssetSessionDetail struct {
//	StartTime string `json:"startTime"`
//	EndTime   string `json:"endTime"`
//	AssetIp   string `json:"assetIp"`
//	AssetName string `json:"assetName"`
//	Passport  string `json:"passport"`
//	Protocol  string `json:"protocol"`
//	ClientIp  string `json:"clientIp"`
//}

// UserSession 用户会话时长
type UserSession struct {
	UserName string `json:"userName"`
	NickName string `json:"nickName"`
	Name     string `json:"name"`
	Time     int64  `json:"time"`
}

// UserSessionDetail 用户会话时长详情
type UserSessionDetail struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	ClientIp  string `json:"clientIp"`
}

type SessionForExport struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	ClientIp  string `json:"clientIp"`
	AssetName string `json:"assetName"`
	AssetIp   string `json:"assetIp"`
	Passport  string `json:"passport"`
	Protocol  string `json:"protocol"`
}

// AssetAccessAssetDetail 资产运维设备详情
type AssetAccessAssetDetail struct {
	StartAt  string `json:"startAt"`
	EndAt    string `json:"endAt"`
	Asset    string `json:"asset"`
	AssetIP  string `json:"assetIP"`
	Passport string `json:"passport"`
	Protocol string `json:"protocol"`
	ClientIP string `json:"clientIP"`
}

// AssetAccessUserDetail 资产运维用户详情
type AssetAccessUserDetail struct {
	StartAt  string `json:"startAt"`
	EndAt    string `json:"endAt"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	ClientIP string `json:"clientIP"`
}

// AssetAccessExport 运维报表导出
type AssetAccessExport struct {
	StartAt  string `json:"startAt"`
	EndAt    string `json:"endAt"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	ClientIP string `json:"clientIP"`
	Asset    string `json:"asset"`
	AssetIP  string `json:"assetIP"`
	Passport string `json:"passport"`
	Protocol string `json:"protocol"`
}

// CommandStatistics 命令统计
type CommandStatistics struct {
	Command string `json:"command"`
	Cnt     int    `json:"cnt"`
}

// CommandStatisticsDetail 命令统计导出/详情
type CommandStatisticsDetail struct {
	Created   string `json:"created"`
	Content   string `json:"content"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	ClientIp  string `json:"clientIp"`
	AssetName string `json:"assetName"`
	AssetIp   string `json:"assetIp"`
	Passport  string `json:"passport"`
	Protocol  string `json:"protocol"`
}

// 告警报表
type AlarmReport struct {
	AlarmTime   string `json:"alarmTime"`
	HighAlert   int    `json:"highAlert"`
	MiddleAlert int    `json:"middleAlert"`
	LowAlert    int    `json:"lowAlert"`
}

// 告警报表详情
type AlarmReportDetail struct {
	AlarmTime  string `json:"alarmTime"`
	Username   string `json:"username"`
	Nickname   string `json:"nickname"`
	ClientIp   string `json:"clientIp"`
	AssetIp    string `json:"assetIp"`
	Passport   string `json:"passport"`
	Protocol   string `json:"protocol"`
	AlarmRule  string `json:"alarmRule"`
	AlarmLevel string `json:"alarmLevel"`
}
