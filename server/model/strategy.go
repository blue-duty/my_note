package model

//import "tkbastion/server/utils"
//
//type Strategy struct {
//	ID        string         `gorm:"primary_key,type:varchar(36)" json:"id"`
//	Name      string         `gorm:"type:varchar(500)" json:"name"`
//	Upload    string         `gorm:"type:varchar(1);default:0" json:"upload"` // 1 = true, 0 = false
//	Download  string         `gorm:"type:varchar(1);default:0" json:"download"`
//	Delete    string         `gorm:"type:varchar(1);default:0" json:"delete"`
//	Rename    string         `gorm:"type:varchar(1);default:0" json:"rename"`
//	Edit      string         `gorm:"type:varchar(1);default:0" json:"edit"`
//	CreateDir string         `gorm:"type:varchar(1);default:0" json:"createDir"`
//	Copy      string         `gorm:"type:varchar(1);default:0" json:"copy"`
//	Paste     string         `gorm:"type:varchar(1);default:0" json:"paste"`
//	Created   utils.JsonTime `json:"created"`
//}
//
//func (r *Strategy) TableName() string {
//	return "strategies"
//}
