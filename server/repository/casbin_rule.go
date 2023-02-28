package repository

import (
	"tkbastion/server/model"

	"gorm.io/gorm"
)

type CasbinRuleRepository struct {
	DB *gorm.DB
}

func NewCasbinRuleRepository(db *gorm.DB) *CasbinRuleRepository {
	casbinRuleRepository = &CasbinRuleRepository{DB: db}
	return casbinRuleRepository
}

func (r CasbinRuleRepository) Create(roleName string, menuId []int) error {
	var menuApiArr []model.CasbinApi
	err := r.DB.Table("casbin_api").Where("menu_id in ?", menuId).Find(&menuApiArr).Error
	if nil != err {
		return err
	}
	tx := r.DB.Begin()
	for _, menuApi := range menuApiArr {
		err = tx.Create(&model.CasbinRule{PType: "p", V0: roleName, V1: menuApi.Path, V2: menuApi.Action, V3: "", V4: "", V5: ""}).Error
		if nil != err {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return nil
}

// Update TODO 前期方便这样写,后期需改成事务commit的形式,保证若更新用户组菜单的api权限,需要做到旧api权限全删除,新api权限全添加,防止出现删除了一部分权限和增加了一部分权限的bug
func (r CasbinRuleRepository) Update(oldUserGroupName, newUserGroupName string, menuId []int) error {
	err := r.DB.Where("v0 = ?", oldUserGroupName).Delete(&model.CasbinRule{}).Error
	if nil != err {
		return err
	}

	var menuApiArr []model.CasbinApi
	err = r.DB.Where("menu_id in ?", menuId).Find(&menuApiArr).Error
	if nil != err {
		return err
	}

	tx := r.DB.Begin()
	for _, menuApi := range menuApiArr {
		err = tx.Create(&model.CasbinRule{PType: "p", V0: newUserGroupName, V1: menuApi.Path, V2: menuApi.Action, V3: "", V4: "", V5: ""}).Error
		if nil != err {
			tx.Rollback()
			return err
		}
	}

	tx.Commit()
	return nil
}

// TODO 事务
func (r CasbinRuleRepository) Delete(userGroupName string) error {
	err := r.DB.Where("v0 = ?", userGroupName).Delete(&model.CasbinRule{}).Error

	return err
}

func (r CasbinRuleRepository) DeleteByRoleName(roleName string) error {
	err := r.DB.Where("v0 = ?", roleName).Delete(&model.CasbinRule{}).Error
	return err
}
