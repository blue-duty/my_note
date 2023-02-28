package dto

type AssetAuthReportForExport struct {
	AssetName string
	AssetIP   string
	Passport  string
	Username  string
	Nickname  string
	AuthName  string
}

type ApplicationAuthReportForExport struct {
	AppSerName  string
	AppName     string
	ProgramName string
	Username    string
	Nickname    string
	AuthName    string
}
