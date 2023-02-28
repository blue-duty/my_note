package model

import "tkbastion/server/utils"

type NewStorage struct {
	ID             string         `gorm:"primary_key;type:varchar(128);not null;comment:id" json:"id" map:"key:id"`
	Name           string         `gorm:"type:varchar(64);unique;not null;comment:存储名称" json:"name" validate:"required,max=64" map:"key:name"`
	LimitSize      int64          `gorm:"type:bigint" json:"limitSize" map:"key:limit_size"`                      // 大小限制，单位字节
	Department     int64          `gorm:"type:bigint(20)" json:"department" map:"key:department"`                 // 部门ID
	DepartmentName string         `gorm:"type:varchar(64);index" json:"departmentName" map:"key:department_name"` // 部门名称
	Info           string         `gorm:"type:text" json:"info" map:"key:info;empty"`                             // 存储信息
	Created        utils.JsonTime `gorm:"type:datetime(3)" json:"created" map:"key:created"`                      // 创建时间
}

func (r *NewStorage) TableName() string {
	return "new_storage"
}
