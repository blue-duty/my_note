package dto

type ApplicationForPage struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	AppSerName   string `json:"appSerName"`
	ProgramName  string `json:"programName"`
	Department   string `json:"department"`
	AppSerId     string `json:"appSerId"`
	ProgramId    string `json:"programId"`
	DepartmentID int64  `json:"departmentId"`
	Param        string `json:"param"`
	Info         string `json:"info"`
}

type ApplicationForSearch struct {
	Auto        string `json:"auto"`
	Name        string `json:"name"`
	AppSerName  string `json:"appSerName"`
	ProgramName string `json:"programName"`
	Department  string `json:"department"`
	PageSize    int    `json:"pageSize"`
	PageIndex   int    `json:"pageIndex"`
	Departments []int64
}

type ApplicationForInsert struct {
	Name         string `json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	AppSerId     string `json:"appSerId"`
	ProgramId    string `json:"programId"`
	DepartmentID int64  `json:"departmentId"`
	Param        string `json:"param"`
	Info         string `json:"info" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
}

type ApplicationForUpdate struct {
	ID           string `json:"id"`
	Name         string `json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	AppSerId     string `json:"appSerId"`
	ProgramId    string `json:"programId"`
	DepartmentID int64  `json:"departmentId"`
	Param        string `json:"param"`
	Info         string `json:"info" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
}

type ApplicationForOperate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AppSerName  string `json:"appSerName"`
	ProgramName string `json:"programName"`
	Department  string `json:"department"`
}

type ApplicationForSession struct {
	Name     string
	IP       string
	Port     int
	Passport string
	Password string
	Path     string
	Param    string
}

type ApplicationForExport struct {
	AppSerName  string `json:"appSerName"`
	Department  string `json:"department"`
	Name        string `json:"name"`
	ProgramName string `json:"programName"`
	IP          string `json:"ip"`
	Port        int    `json:"port"`
	Passport    string `json:"passport"`
	Password    string `json:"password"`
	Path        string `json:"path"`
	Param       string `json:"param"`
	Info        string `json:"info"`
}

type ApplicationForDetail struct {
	Name        string `json:"name"`
	AppSerName  string `json:"appSerName"`
	ProgramName string `json:"programName"`
	Department  string `json:"department"`
	Param       string `json:"param"`
	Info        string `json:"info"`
	Created     string `json:"created"`
}

type ApplicationForPolicy struct {
	Username   string `json:"username"`
	Nickname   string `json:"nickname"`
	Department string `json:"department"`
	Policy     string `json:"policy"`
}
