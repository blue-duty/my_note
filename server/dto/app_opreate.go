package dto

type AppOperate struct {
	Id         string `json:"id" `
	Name       string `json:"name" `
	Program    string `json:"program" `
	AppServer  string `json:"appServer" `
	Collection string `json:"collection" `
}

type AppOperateForSearch struct {
	Auto        string `json:"auto" `
	Name        string `json:"name" `
	Program     string `json:"program" `
	AppServer   string `json:"appServer" `
	Departments []int64
}

type OperateLog struct {
	LoginTime  string `json:"loginTime" `
	Username   string `json:"username" `
	Nickname   string `json:"nickname" `
	Protocol   string `json:"protocol" `
	AssetName  string `json:"assetName" `
	AssetIp    string `json:"assetIp" `
	Ip         string `json:"ip" `
	Passport   string `json:"passport" `
	LogoutTime string `json:"logoutTime" `
}

type OperateLogForSearch struct {
	Auto        string `json:"auto" `
	LogoutTime  string `json:"logoutTime" `
	LoginTime   string `json:"loginTime" `
	Username    string `json:"username" `
	Nickname    string `json:"nickname" `
	Ip          string `json:"ip" `
	AssetName   string `json:"assetName" `
	AssetIp     string `json:"assetIp" `
	Passport    string `json:"passport" `
	Departments []int64
}

type OperateLogForExport struct {
	LoginTime  string `json:"loginTime" `
	Ip         string `json:"ip" `
	Username   string `json:"username" `
	Nickname   string `json:"nickname" `
	Type       string `json:"type" `
	AssetName  string `json:"assetName" `
	AssetIp    string `json:"assetIp" `
	Passport   string `json:"passport" `
	LogoutTime string `json:"logoutTime" `
}
