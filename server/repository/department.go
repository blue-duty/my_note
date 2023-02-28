package repository

import (
	"tkbastion/server/model"

	"gorm.io/gorm"
)

type DepartmentRepository struct {
	DB *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) *DepartmentRepository {
	departmentRepository = &DepartmentRepository{DB: db}
	return departmentRepository
}

func (r DepartmentRepository) Create(o *model.Department) (err error) {
	return r.DB.Create(o).Error
}

// BroDep 此接口返回的该上级部门下所有子部门
func (r DepartmentRepository) BroDep(fatId int64) (broDepArr []model.Department, err error) {
	err = r.DB.Where("father_id = ?", fatId).Find(&broDepArr).Error
	return
}

func (r DepartmentRepository) UpdateById(id int64, o *model.Department) (err error) {
	o.ID = id
	return r.DB.Where("id = ?", id).Updates(o).Error
}

func (r DepartmentRepository) FindById(id int64) (o model.Department, err error) {
	err = r.DB.Where("id = ?", id).First(&o).Error
	return
}
func (r DepartmentRepository) FindNameById(id int64) (o string, err error) {
	err = r.DB.Model(model.Department{}).Select("name").Where("id = ?", id).First(&o).Error
	return
}

func (r DepartmentRepository) FindByNameFatherId(name string, fatherDepId int64) (o model.Department, err error) {
	err = r.DB.Where("name = ? AND father_id = ?", name, fatherDepId).First(&o).Error
	return
}

// FindByName 因不同部门的下级部门名称可以重复，此处可能会查出多个部门，但只取第一个，使用时需注意
func (r DepartmentRepository) FindByName(name string) (o model.Department, err error) {
	err = r.DB.Where("name = ?", name).First(&o).Error
	return
}

func (r DepartmentRepository) DeleteInDepIds(ids []int64) (err error) {
	return r.DB.Where("id in ?", ids).Delete(model.Department{}).Error
}

func (r DepartmentRepository) Count() (total int64, err error) {
	err = r.DB.Model(&model.Department{}).Count(&total).Error
	return
}

// 通过输入的参数模糊匹配name字段，返回ids
func (r DepartmentRepository) FindIdsByVagueName(name string) (depIds []int64, err error) {
	sql := "SELECT id FROM department "
	whereCondition := "WHERE name LIKE '%" + name + "%'"
	sql += whereCondition
	err = departmentRepository.DB.Raw(sql).Find(&depIds).Error

	return
}
