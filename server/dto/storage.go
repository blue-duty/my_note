package dto

type StorageForCreate struct {
	Name       string `json:"name" validate:"required" validate:"required,min=1,max=32,alphanumunicode"`
	Department int64  `json:"departmentId"`
	Info       string `json:"info" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
}

type StorageForSearch struct {
	Name        string `json:"name"`
	Department  string `json:"department"`
	Auto        string `json:"auto"`
	Departments []int64
}

type StorageForPage struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	DepartmentId int64  `json:"departmentId"`
	Department   string `json:"department"`
	LimitSize    int64  `json:"limitSize"`
	UseSize      int64  `json:"useSize"`
	Info         string `json:"info"`
}
