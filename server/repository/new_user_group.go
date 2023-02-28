package repository

import (
	"tkbastion/server/dto"
	"tkbastion/server/model"

	"gorm.io/gorm"
)

type UserGroupNewRepository struct {
	DB *gorm.DB
}

func NewUserGroupNewRepository(db *gorm.DB) *UserGroupNewRepository {
	userGroupNewRepository = &UserGroupNewRepository{DB: db}
	return userGroupNewRepository
}

func (r *UserGroupNewRepository) Create(o *model.UserGroupNew) error {
	return r.DB.Create(o).Error
}

// 根据id删除用户组

func (r *UserGroupNewRepository) DeleteById(id string) error {
	return r.DB.Where("id = ?", id).Delete(&model.UserGroupNew{}).Error
}

// 根据id查询用户组

func (r *UserGroupNewRepository) FindById(id string) (o model.UserGroupNew, err error) {
	err = r.DB.Where("id = ?", id).Find(&o).Error
	return
}

// 根据id修改用户组

func (r *UserGroupNewRepository) UpdateById(id string, o *model.UserGroupNew) error {
	return r.DB.Model(&model.UserGroupNew{}).Where("id = ?", id).Updates(o).Error
}

// 查询用户组

func (r *UserGroupNewRepository) FindByLimitingConditions(pageIndex, pageSize int, auto, name, department string, departmentId []int64) (o []model.UserGroupNew, total int64, err error) {
	db := r.DB.Table("user_group_new").Where("department_id in (?)", departmentId)
	if len(auto) > 0 {
		db = db.Where("department_name like ? or name like ? ", "%"+auto+"%", "%"+auto+"%").Or("description like ?", "%"+auto+"%")
	} else {
		if len(department) > 0 {
			db = db.Where("department_name like ?", "%"+department+"%")
		}
		if len(name) > 0 {
			db = db.Where("name like ?", "%"+name+"%")
		}
	}
	err = db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = db.Order("created asc").Find(&o).Error
	return
}

func (r *UserGroupNewRepository) FindUserGroupByDepartmentId(departmentId []int64) (userGroupNew []model.UserGroupNew, err error) {
	err = r.DB.Table("user_group_new").Where("department_id in (?)", departmentId).Find(&userGroupNew).Error
	return
}

//func (r *UserGroupNewRepository) FindUserGroupByDepartmentIdNotUserGroupIds(departmentId []int64, userGroupIds []string) (userGroupNew []model.UserGroupNew, err error) {
//	err = r.DB.Table("user_group_new").Where("department_id in (?) AND id NOT IN ?", departmentId, userGroupIds).Find(&userGroupNew).Error
//	return
//}

func (r *UserGroupNewRepository) DeleteUserGroupByDepartmentId(departmentId []int64) (err error) {
	err = r.DB.Table("user_group_new").Where("department_id in (?)", departmentId).Delete(&model.UserGroupNew{}).Error
	return
}

func (r *UserGroupNewRepository) DeleteByDepNotGen() (err error) {
	err = r.DB.Table("user_group_new").Where("department_id != 0").Delete(&model.UserGroupNew{}).Error
	return
}

func (r *UserGroupNewRepository) UserGroupExport(departmentId []int64) (userGroupForExport []dto.UserGroupForExport, err error) {
	db := r.DB.Table("user_group_new").Select("id,name,department_name,description,total").Where("department_id in (?)", departmentId)
	err = db.Where("department_id in (?)", departmentId).Find(&userGroupForExport).Error
	return
}

func (r *UserGroupNewRepository) FindByIds(ids []string) (o []model.UserGroupNew, err error) {
	err = r.DB.Where("id in (?)", ids).Find(&o).Error
	return
}
