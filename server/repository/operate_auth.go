package repository

import (
	"tkbastion/server/model"

	"gorm.io/gorm"
)

type OperateAuthRepository struct {
	DB *gorm.DB
}

func NewOperateAuthRepository(db *gorm.DB) *OperateAuthRepository {
	operateAuthRepository = &OperateAuthRepository{DB: db}
	return operateAuthRepository
}

func (r OperateAuthRepository) Create(o *model.OperateAuth) (err error) {
	return r.DB.Create(o).Error
}

func (r OperateAuthRepository) FindByDepIds(depIds []int64) (operateAuthArr []model.OperateAuth, err error) {
	err = r.DB.Find(&operateAuthArr).Where("department_id in ?", depIds).Error
	return
}

func (r OperateAuthRepository) FindByRateUserId(userId string) (operateAuth []model.OperateAuth, err error) {
	err = r.DB.Where("relate_user like ?", "%"+userId+"%").First(&operateAuth).Error
	return
}

func (r OperateAuthRepository) FindByRateUserGroupId(userGroupId string) (operateAuth []model.OperateAuth, err error) {
	err = r.DB.Where("relate_user_group like ?", "%"+userGroupId+"%").First(&operateAuth).Error
	return
}

func (r OperateAuthRepository) UpdateById(id int64, o *model.OperateAuth) (err error) {
	o.ID = id
	return r.DB.Where("id = ?", id).Updates(o).Error
}

func (r OperateAuthRepository) FindById(id int64) (o model.OperateAuth, err error) {
	err = r.DB.Where("id = ?", id).First(&o).Error
	return
}

func (r OperateAuthRepository) FindByNameDepId(name string, depId int64) (o model.OperateAuth, err error) {
	err = r.DB.Where("name = ? AND department_id = ?", name, depId).First(&o).Error
	return
}

func (r OperateAuthRepository) FindByDepIdsAndSort(depIds []int64) (o []model.OperateAuth, err error) {
	sql := "SELECT id, name, department_id, department_name, state, button_state, description, relate_user,download,upload,watermark, relate_user_group, relate_asset, relate_asset_group, relate_app, dep_level, priority, auth_expiration_date, strategy_begin_time, strategy_end_time, strategy_time_flag, ip_limit_type, ip_limit_list FROM operate_auth where department_id in ? ORDER BY dep_level ASC, department_name, priority ASC"
	err = r.DB.Raw(sql, depIds).Find(&o).Error
	return
}

func (r OperateAuthRepository) DeleteById(id int64) (err error) {
	return r.DB.Where("id = ?", id).Delete(model.OperateAuth{}).Error
}

func (r OperateAuthRepository) DeleteInDepIds(depIds []int64) (err error) {
	return r.DB.Where("department_id in ?", depIds).Delete(model.OperateAuth{}).Error
}

func (r OperateAuthRepository) DeleteByDepNotGen() (err error) {
	return r.DB.Where("department_id != 0").Delete(model.OperateAuth{}).Error
}

func (r OperateAuthRepository) StrategyCountByDepId(depId int64) (total int64, err error) {
	err = r.DB.Model(model.OperateAuth{}).Where("department_id = ?", depId).Count(&total).Error
	return
}

func (r OperateAuthRepository) BigStrategyPriority(depId int64, priority int) (o []model.OperateAuth, err error) {
	err = r.DB.Model(model.OperateAuth{}).Where("department_id = ? AND priority > ?", depId, priority).Find(&o).Error
	return
}

func (r OperateAuthRepository) UpdateDepLevelByDepId(depId int64, depLevel int) (err error) {
	return r.DB.Model(model.OperateAuth{}).Where("department_id = ?", depId).Update("dep_level", depLevel).Error
}

func (r OperateAuthRepository) UpdateDepNameByDepId(depId int64, depName string) (err error) {
	return r.DB.Model(model.OperateAuth{}).Where("department_id = ?", depId).Update("department_name", depName).Error
}

func (r OperateAuthRepository) FindPriorityRangeByDepId(smallPriority, bigPriority int, depId int64) (o []model.OperateAuth, err error) {
	err = r.DB.Where("department_id = ? AND (priority >= ? AND priority <= ?)", depId, smallPriority, bigPriority).Find(&o).Error
	return
}

func (r OperateAuthRepository) UpdateColById(id int64, col string, val interface{}) (err error) {
	err = r.DB.Model(model.OperateAuth{}).Where("id = ?", id).Update(col, val).Error
	return
}
