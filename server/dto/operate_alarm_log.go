package dto

type OperateAlarmLog struct {
	AlarmTime string `json:"alarm_time"`
	ClientIP  string `json:"client_ip"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AssetIp   string `json:"asset_ip"`
	Passport  string `json:"passport"`
	Protocol  string `json:"protocol"`
	Content   string `json:"content"`
	Strategy  string `json:"strategy"`
	Level     string `json:"level"`
	Result    string `json:"result"`
}

type OperateAlarmLogForSearch struct {
	Auto     string `json:"auto"`
	ClientIP string `json:"client_ip"`
	Username string `json:"username"`
	AssetIp  string `json:"asset_ip"`
	Passport string `json:"passport"`
	Content  string `json:"content"`
	Level    string `json:"level"`
}

type SystemAlarmLogForExport struct {
	AlarmTime string `json:"alarmTime" `
	Content   string `json:"content" `
	Strategy  string `json:"strategy" `
	Level     string `json:"level" `
	Result    string `json:"result" `
}
