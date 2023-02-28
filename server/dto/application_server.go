package dto

type ApplicationServerForPage struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	IP           string `json:"ip"`
	Type         string `json:"type"`
	Info         string `json:"info"`
	Department   string `json:"department"`
	Passport     string `json:"passport"`
	DepartmentID int64  `json:"departmentId"`
}

type ApplicationServerForSearch struct {
	IP          string `json:"ip"`
	Name        string `json:"name"`
	Auto        string `json:"auto"`
	Department  string `json:"department"`
	Departments []int64
}

type ApplicationServerForInsert struct {
	Name         string `json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	IP           string `json:"ip" validate:"required,ipv4"`
	Port         int    `json:"port" validate:"required,min=1,max=65535"`
	Type         string `json:"type"`
	DepartmentID int64
	Department   string
	Passport     string `json:"passport"`
	Password     string `json:"password"`
	Info         string `json:"info" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
}

type ApplicationServerForUpdate struct {
	ID           string `json:"id"`
	Name         string `json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	IP           string `json:"ip" validate:"required,ipv4"`
	Port         int    `json:"port" validate:"required,min=1,max=65535"`
	Type         string `json:"type"`
	DepartmentID int64  `json:"departmentId"`
	Department   string
	Passport     string `json:"passport"`
	Password     string `json:"password"`
	Info         string `json:"info" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
}
