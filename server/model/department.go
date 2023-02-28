package model

type Department struct {
	ID             int64  `gorm:"primary_key;type:bigint(20) AUTO_INCREMENT;not null;comment:部门ID" json:"id"`
	Name           string `gorm:"type:varchar(32);not null;comment:部门名称" json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	FatherId       int64  `gorm:"type:bigint(20);not null;comment:父部门ID" json:"fatherId"`
	LeftChildId    int64  `gorm:"type:bigint(20);not null;default:-2;comment:左子部门ID" json:"leftChildId"`
	RightBroId     int64  `gorm:"type:bigint(20);not null;default:-2;右兄弟部门ID" json:"rightBroId"`
	Description    string `gorm:"type:varchar(128);not null;default:'';comment:部门描述" json:"description" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
	AssetCount     int    `gorm:"type:int(5);not null;default:0;comment:部门资产个数" json:"assetCount"`
	UserCount      int    `gorm:"type:int(5);not null;default:0;comment:部门用户个数" json:"userCount"`
	AppCount       int    `gorm:"type:int(5);not null;default:0;comment:部门应用个数" json:"appCount"`
	AppServerCount int    `gorm:"type:int(5);not null;default:0;comment:部门应用服务器个数" json:"appServerCount"`
}

type DepartmentTree struct {
	ID             int64            `json:"id"`
	Name           string           `json:"name"`
	FatherId       int64            `json:"fatherID"`
	ChildArr       []DepartmentTree `json:"children"`
	Description    string           `json:"description"`
	AssetCount     int              `json:"assetCount"`
	UserCount      int              `json:"userCount"`
	AppCount       int              `json:"appCount"`
	AppServerCount int              `json:"appServerCount"`
}

func (r *Department) TableName() string {
	return "department"
}
