package dto

type AssetGroupForPage struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Count      int    `json:"count"`
	Info       string `json:"info"`
	Department string `json:"department"`
}

type ForRelate struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AssetGroupWithId struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AssetGroupForAsset struct {
	Name        string `json:"name"`
	Department  string `json:"department"`
	CreatedTime string `json:"createdTime"`
}

type AssetGroupCreateRequest struct {
	DepartmentId int64
	Department   string
	Name         string `json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	Info         string `json:"info" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
	Assets       string `json:"assets"`
}

type AssetGroupUpdateRequest struct {
	ID   string `json:"id"`
	Name string `json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	Info string `json:"info" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
}

type AssetGroupAssetRequest struct {
	ID  string   `json:"id"`
	IDs []string `json:"ids"`
}
