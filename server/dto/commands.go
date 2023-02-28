package dto

type CommandForCreate struct {
	Name    string `json:"name" validate:"required,max=64"`
	Content string `json:"content" validate:"required,max=191"`
	Info    string `json:"info"`
}

type CommandForUpdate struct {
	ID      string `json:"id" `
	Name    string `json:"name"`
	Content string `json:"content"`
	Info    string `json:"info"`
}

type CommandForPage struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Created string `json:"created"`
	Info    string `json:"info"`
}

type CommandForGet struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

type CommandForSearch struct {
	Auto    string `json:"auto"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Uid     string `json:"uid"`
}
