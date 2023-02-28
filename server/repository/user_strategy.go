package repository

import (
	"tkbastion/server/model"

	"gorm.io/gorm"
)

type UserStrategyRepository struct {
	DB *gorm.DB
}

func NewUserStrategyRepository(db *gorm.DB) *UserStrategyRepository {
	userStrategyRepository = &UserStrategyRepository{DB: db}
	return userStrategyRepository
}

func (r *UserStrategyRepository) FindAll() (userStrategy []model.UserStrategy, err error) {
	// 按优先级升序排列
	err = r.DB.Table("user_strategy").Order("priority asc").Find(&userStrategy).Error
	return
}

func (r *UserStrategyRepository) FindByDepartmentId(id int64) (userStrategy []model.UserStrategy, err error) {
	// 按优先级升序排列
	err = r.DB.Table("user_strategy").Where("department_id = ?", id).Order("priority asc").Find(&userStrategy).Error
	return
}

func (r *UserStrategyRepository) CountByDepartmentId(depId int64) (total int64, err error) {
	err = r.DB.Table("user_strategy").Where("department_id = ?", depId).Count(&total).Error
	return
}

func (r *UserStrategyRepository) Creat(o *model.UserStrategy) (err error) {
	err = r.DB.Create(o).Error
	return
}

func (r *UserStrategyRepository) FindById(id string) (o model.UserStrategy, err error) {
	err = r.DB.Where("id=?", id).Find(&o).Error
	return
}

func (r *UserStrategyRepository) FindByLimitingConditions(pageIndex, pageSize int, auto, department, name, description, status string, departmentId []int64) (o []model.UserStrategy, total int64, err error) {
	db := r.DB.Table("user_strategy").Where("department_id in (?)", departmentId)
	if len(auto) > 0 {
		db = db.Where("department_name like ? or name like ? or description like ? or status like ?", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%")
	} else {
		if len(department) > 0 {
			db = db.Where("department_name like ?", "%"+department+"%")
		}
		if len(name) > 0 {
			db = db.Where("name like ?", "%"+name+"%")
		}
		if len(description) > 0 {
			db = db.Where("description like ?", "%"+description+"%")
		}
		if len(status) > 0 {
			db = db.Where("status like ?", "%"+status+"%")
		}
	}
	err = db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	// 按照部门深度和优先级升序排列
	err = db.Order("department_depth asc, priority asc").Find(&o).Error
	return
}

func (r *UserStrategyRepository) UpdateById(o *model.UserStrategy, id string) (err error) {
	err = r.DB.Model(&model.UserStrategy{}).Where("id=?", id).Updates(o).Error
	return
}

func (r *UserStrategyRepository) DeleteById(id string) (err error) {
	err = r.DB.Where("id=?", id).Delete(&model.UserStrategy{}).Error
	return
}

func (r *UserStrategyRepository) DeleteByUserId(userId string) (err error) {
	err = r.DB.Table("user_strategy_users").Where("user_id = ?", userId).Delete(&model.UserStrategy{}).Error
	return
}
func (r *UserStrategyRepository) DeleteByUserGroupId(userGroupId string) (err error) {
	err = r.DB.Table("user_strategy_user_group").Where("user_group_id = ?", userGroupId).Delete(&model.UserStrategy{}).Error
	return
}

func (r *UserStrategyRepository) FindStrategyIdByUserGroupId(userGroupId string) (userStrategyId []string, err error) {
	err = r.DB.Table("user_strategy_user_group").Select("user_strategy_id").Where("user_group_id = ?", userGroupId).Find(&userStrategyId).Error
	return
}

func (r *UserStrategyRepository) FindUserStrategyByDepartmentId(departmentId []int64) (userStrategy []model.UserStrategy, err error) {
	err = r.DB.Table("user_strategy").Where("department_id in (?)", departmentId).Find(&userStrategy).Error
	return
}

func (r *UserStrategyRepository) DeleteUserStrategyByDepartmentId(departmentId []int64) (err error) {
	err = r.DB.Table("user_strategy").Where("department_id in (?)", departmentId).Delete(&model.UserStrategy{}).Error
	return
}

func (r *UserStrategyRepository) DeleteByDepNotGen() (err error) {
	err = r.DB.Table("user_strategy").Where("department_id != 0").Delete(&model.UserStrategy{}).Error
	return
}
