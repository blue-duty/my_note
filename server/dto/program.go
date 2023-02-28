package dto

type NewProgramForPage struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
	Info string `json:"info"`
}

type NewProgramForInsert struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Info string `json:"info"`
	Aid  string `json:"aid"`
}

type NewProgramForUpdate struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
	Info string `json:"info"`
	Aid  string `json:"aid"`
}
