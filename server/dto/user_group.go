package dto

type UserGroupNewCreate struct {
	ID        string `json:"id"`
	Name      string `json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	MemberIds string `json:"memberIds"`
	Info      string `json:"info" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
}

type UserGroupNewUpdate struct {
	ID          string `json:"id"`
	Name        string `json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	Description string `json:"description" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
}

type UserGroupForExport struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	DepartmentName string `json:"departmentName"`
	Description    string `json:"description"`
	Total          int    `json:"total"`
	Members        string `json:"members"`
}
