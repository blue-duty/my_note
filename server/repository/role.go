package repository

import (
	"gorm.io/gorm"
	"tkbastion/server/model"
)

type RoleRepository struct {
	DB *gorm.DB
}

func NewRoleRepository(db *gorm.DB) *RoleRepository {
	roleRepository = &RoleRepository{DB: db}
	return roleRepository
}

func (r *RoleRepository) Find() (o []model.Role, err error) {
	err = r.DB.Order("is_edit asc").Find(&o).Error
	return
}

func (r *RoleRepository) FindForUserCreate() (o []model.RoleForUserCreate, err error) {
	err = r.DB.Table("roles").Select("id, name").Order("is_edit asc").Find(&o).Error
	return
}

func (r *RoleRepository) FindById(id string) (o model.Role, err error) {
	err = r.DB.Where("id = ?", id).Find(&o).Error
	return
}
func (r *RoleRepository) FindRoleNameById(id string) (name string, err error) {
	err = r.DB.Table("roles").Select("name").Where("id = ?", id).Find(&name).Error
	return
}

func (r *RoleRepository) DeleteById(id string) (err error) {
	err = r.DB.Where("id = ?", id).Delete(model.Role{}).Error
	return
}

func (r *RoleRepository) FindByRoleName(roleName string) (o *model.Role, err error) {
	err = r.DB.Where("name = ?", roleName).First(&o).Error
	return
}

func (r *RoleRepository) Creat(o *model.Role) (err error) {
	err = r.DB.Create(o).Error
	return
}

func (r *RoleRepository) UpdateById(o *model.Role, id string) (err error) {
	err = r.DB.Where("id = ?", id).Updates(o).Error
	return
}

func (r *RoleRepository) FindMenuIdsByName(roleName string) (menuId []int, err error) {
	err = r.DB.Table("role_menus").Select("menu_id").Where("name = ?", roleName).Find(&menuId).Error
	return
}

func (r *RoleRepository) DeleteRoleMenuByName(roleName string) (err error) {
	err = r.DB.Table("role_menus").Where("name = ?", roleName).Delete(model.RoleMenu{}).Error
	return
}

func (r *RoleRepository) FindThreeLevelMenuIdsById(roleName string, secondLevel []int) (menuId []int, err error) {
	err = r.DB.Table("role_menus").Select("menu_id").Where("name = ? and menu_id not in ? ", roleName, secondLevel).Find(&menuId).Error
	return
}

func (r *RoleRepository) Count() (total int64, err error) {
	err = r.DB.Model(&model.Role{}).Count(&total).Error
	return
}
