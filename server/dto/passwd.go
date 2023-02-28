package dto

type PasswdChange struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	RunType      string `json:"run_type"`
	GenerateRule string `json:"generate_rule"`
}

// PasswdView 账号密码查看
type PasswdView struct {
	ID         string `json:"id"`
	Ip         string `json:"ip"`
	Department string `json:"department"`
	AssetName  string `json:"asset_name"`
	SystemType string `json:"system_type"`
	Protocol   string `json:"protocol"`
	Passport   string `json:"passport"`
}

// PasswdViewDetail 账号密码查看详情
type PasswdViewDetail struct {
	AssetIp    string `json:"asset_ip"`
	AssetName  string `json:"asset_name"`
	Passport   string `json:"passport"`
	Protocol   string `json:"protocol"`
	SystemType string `json:"system_type"`
	OldPasswd  string `json:"old_passwd"`
	NewPasswd  string `json:"new_passwd"`
	// 上次修改时间
	LastChangeTime string `json:"last_change_time"`
}

type PasswdChangeResult struct {
	AssetIp    string `json:"asset_ip"`
	AssetName  string `json:"asset_name"`
	Passport   string `json:"passport"`
	ChangeTime string `json:"change_time"`
	Name       string `json:"name"`
	Result     string `json:"result"`
}

// PasswdChangeResultStatistical 账号密码改密统计
type PasswdChangeResultStatistical struct {
	AssetIp   string `json:"asset_ip"`
	AssetName string `json:"asset_name"`
	Passport  string `json:"passport"`
	Success   int    `json:"success"`
	Failure   int    `json:"failure"`
}
