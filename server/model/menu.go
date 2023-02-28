package model

type Menu struct {
	Id        int    `gorm:"type:int(3);primary_key;not null;comment:菜单结点id"  json:"id"`
	Title     string `gorm:"type:varchar(32);not null;comment:菜单结点名称"  json:"title"`
	Icon      string `gorm:"type:varchar(32);not null;comment:菜单结点react图标名称"  json:"icon"`
	Name      string `gorm:"type:varchar(32);not null;comment:菜单结点名称"  json:"name"`
	Path      string `gorm:"type:varchar(64);not null;comment:菜单结点路由地址"  json:"path"`
	Paths     string `gorm:"type:varchar(32);not null;comment:菜单结点路径"  json:"paths"`
	Component string `gorm:"type:varchar(128);not null;comment:菜单结点组件地址"  json:"component"`
	Type      string `gorm:"type:varchar(8);not null;comment:菜单结点类型"  json:"type"`
	ParentId  int    `gorm:"type:int(3);not null;comment:菜单结点父级结点id"  json:"parentId"`
	Children  []Menu `json:"children,omitempty" gorm:"-"`
}

func (r *Menu) TableName() string {
	return "menus"
}
