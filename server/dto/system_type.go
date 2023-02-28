package dto

type SystemType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Info string `json:"info"`
	//是否默认
	Default bool `json:"default"`
}

type SystemTypeForCreate struct {
	Name string `json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	Type string `json:"type"`
	Info string `json:"info" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
}

type SystemTypeForUpdate struct {
	//ID   string `json:"id"`
	Name string `json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	//Type string `json:"type"`
	Info string `json:"info" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
}
