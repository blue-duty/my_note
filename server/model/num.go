package model

type Num struct {
	I string `gorm:"type:varchar(8);primary_key;not null;comment:辅助会话统计功能" json:"i"`
}

func (r *Num) TableName() string {
	return "nums"
}
