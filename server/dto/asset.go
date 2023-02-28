package dto

type AssetForPage struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	IP            string `json:"ip"`
	AssetType     string `json:"assetType"`
	PassportCount int    `json:"passportCount"`
	Department    string `json:"department"`
	DepartmentId  int64  `json:"departmentId"`
	Info          string `json:"info"`
}

type AssetForSearch struct {
	Auto       string `json:"auto"`
	IP         string `json:"ip"`
	Name       string `json:"name"`
	AssetType  string `json:"assetType"`
	Department string `json:"department"`
	//PageSize      int      `json:"pageSize"`
	//PageLimit     int      `json:"pageLimit"`
	DepartmentIds []int64 `json:"departmentIds"`
}

type AssetForCreate struct {
	Name           string `json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	IP             string `json:"ip" validate:"required,ip"`
	AssetType      string `json:"assetType"`
	Department     int64  `json:"department"`
	Info           string `json:"info" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
	LoginType      string `json:"loginType"`
	Protocol       string `json:"protocol"`
	Passport       string `json:"passport"`
	Port           int    `json:"port" validate:"required,min=1,max=65535"`
	PassportType   string `json:"passportType"`
	Password       string `json:"password"`
	Passphrase     string `json:"passphrase"`
	SftpPath       string `json:"sftpPath"`
	RdpDomain      string `json:"rdpDomain"`
	RdpEnableDrive string `json:"rdpEnableDrive"`
	RdpDriveId     string `json:"rdpDriveId"`
	AppSerId       string `json:"appSerId"`
	TermProgram    string `json:"termProgram"`
	DisplayProgram string `json:"displayProgram"`
}

type AssetForUpdate struct {
	ID           string `json:"id"`
	Name         string `json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	IP           string `json:"ip" validate:"required,ip"`
	AssetType    string `json:"assetType"`
	DepartmentId int64  `json:"departmentId"`
	Info         string `json:"info" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
}

type AssetForDetail struct {
	Name        string `json:"name"`
	IP          string `json:"ip"`
	AssetType   string `json:"assetType"`
	Department  string `json:"department"`
	Info        string `json:"info"`
	CreatedTime string `json:"createdTime"`
}

type PassportForAsset struct {
	Passport string `json:"passport"`
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
	Status   string `json:"status"`
	Created  string `json:"created"`
}

// AssetForPolicy 运维策略
type AssetForPolicy struct {
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Protocol string `json:"protocol"`
	Passport string `json:"passport"`
	Policy   string `json:"policy"`
}

// AssetForCommandPolicy 指令策略
type AssetForCommandPolicy struct {
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Protocol string `json:"protocol"`
	Passport string `json:"passport"`
	Level    string `json:"level"`
	Action   string `json:"action"`
	Policy   string `json:"policy"`
}

// AssetForBatchUpdate 批量编辑
type AssetForBatchUpdate struct {
	AssetIds   []string `json:"assetIds"`
	AssetType  string   `json:"assetType"`
	Department string   `json:"department"`
	LoginType  string   `json:"loginType"`
	Protocol   string   `json:"protocol"`
	Port       string   `json:"port"`
	Password   string   `json:"password"`
}

type PassPortForPage struct {
	ID             string `json:"id"`
	LoginType      string `json:"loginType"`
	Protocol       string `json:"protocol"`
	Port           int    `json:"port"`
	PassportType   string `json:"passportType"`
	Passport       string `json:"passport"`
	Password       string `json:"password"`
	KeyFile        string `json:"keyFile"`
	Passphrase     string `json:"passphrase"`
	Status         string `json:"status"`
	SftpPath       string `json:"sftpPath"`
	IsSshKey       string `json:"isSshKey"`
	RdpDomain      string `json:"rdpDomain"`
	RdpEnableDrive string `json:"rdpEnableDrive"`
	RdpDriveId     string `json:"rdpDriveId"`
	AppSerId       string `json:"appSerId"`
	TermProgram    string `json:"termProgram"`
	DisplayProgram string `json:"displayProgram"`
}

type PassPortForCreate struct {
	LoginType      string `json:"loginType"`
	AssetId        string `json:"assetId"`
	Protocol       string `json:"protocol"`
	Port           int    `json:"port"`
	PassportType   string `json:"passportType"`
	Password       string `json:"password"`
	Passphrase     string `json:"passphrase"`
	Passport       string `json:"passport"`
	SftpPath       string `json:"sftpPath"`
	RdpDomain      string `json:"rdpDomain"`
	RdpEnableDrive string `json:"rdpEnableDrive"`
	RdpDriveId     string `json:"rdpDriveId"`
	AppSerId       string `json:"appSerId"`
	TermProgram    string `json:"termProgram"`
	DisplayProgram string `json:"displayProgram"`
}

//type PassPortForUpdate struct {
//	ID             string `json:"id"`
//	LoginType      string `json:"loginType"`
//	Protocol       string `json:"protocol"`
//	Port           int    `json:"port"`
//	PassportType   string `json:"passportType"`
//	Password       string `json:"password"`
//	Passphrase     string `json:"passphrase"`
//	Passport       string `json:"passport"`
//	RdpDomain      string `json:"rdpDomain"`
//	RdpEnableDrive string `json:"rdpEnableDrive"`
//	RdpDriveId     string `json:"rdpDriveId"`
//}

// AssetForExport 资产导出
type AssetForExport struct {
	Name         string `json:"name"`
	IP           string `json:"ip"`
	Department   string `json:"department"`
	AssetType    string `json:"assetType"`
	Info         string `json:"info"`
	Passport     string `json:"passport"`
	Port         int    `json:"port"`
	SshKey       string `json:"sshKey"`
	Protocol     string `json:"protocol"`
	LoginType    string `json:"loginType"`
	SftpPath     string `json:"sftpPath"`
	PassportType string `json:"passportType"`
	Status       string `json:"status"`
}

// PassportWithPasswordForExport 包含密码的账号导出
type PassportWithPasswordForExport struct {
	Name         string `json:"name"`
	IP           string `json:"ip"`
	Department   string `json:"department"`
	AssetType    string `json:"assetType"`
	Info         string `json:"info"`
	Passport     string `json:"passport"`
	Password     string `json:"password"`
	Port         int    `json:"port"`
	SshKey       string `json:"sshKey"`
	Protocol     string `json:"protocol"`
	LoginType    string `json:"loginType"`
	SftpPath     string `json:"sftpPath"`
	PassportType string `json:"passportType"`
	Status       string `json:"status"`
}
