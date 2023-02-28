package repository

import (
	"tkbastion/server/model"

	"gorm.io/gorm"
)

type LdapAdAuthRepository struct {
	DB *gorm.DB
}

func NewLdapAdAuthRepository(db *gorm.DB) *LdapAdAuthRepository {
	ldapAdAuthRepository = &LdapAdAuthRepository{DB: db}
	return ldapAdAuthRepository
}

func (r LdapAdAuthRepository) Create(o *model.LdapAdAuth) (err error) {
	return r.DB.Create(o).Error
}

func (r LdapAdAuthRepository) Find() (ldapAdAuthArr []model.LdapAdAuth, err error) {
	err = r.DB.Model(model.LdapAdAuth{}).Find(&ldapAdAuthArr).Error
	return
}

func (r LdapAdAuthRepository) UpdateById(id int64, o *model.LdapAdAuth) (err error) {
	o.ID = id
	return r.DB.Where("id = ?", id).Updates(o).Error
}

func (r LdapAdAuthRepository) FindById(id int64) (o model.LdapAdAuth, err error) {
	err = r.DB.Where("id = ?", id).First(&o).Error
	return
}

func (r LdapAdAuthRepository) FindByNameDepId(name string, depId int64) (o model.LdapAdAuth, err error) {
	err = r.DB.Where("name = ? AND department_id = ?", name, depId).First(&o).Error
	return
}

func (r LdapAdAuthRepository) FindByDepIdsAndSort(depIds []int64) (o []model.LdapAdAuth, err error) {
	sql := "SELECT id, name, department_id, department_name, state, button_state, description, relate_user, relate_user_group, relate_asset, relate_asset_group, relate_app, dep_level, priority, auth_expiration_date, strategy_begin_time, strategy_end_time, strategy_time_flag, ip_limit_type, ip_limit_list FROM operate_auth where department_id in ? ORDER BY dep_level ASC, department_name, priority ASC"
	err = r.DB.Raw(sql, depIds).Find(&o).Error
	return
}

func (r LdapAdAuthRepository) DeleteById(id int64) (err error) {
	return r.DB.Where("id = ?", id).Delete(model.LdapAdAuth{}).Error
}

func (r LdapAdAuthRepository) DeleteInDepIds(depIds []int64) (err error) {
	return r.DB.Where("department_id in ?", depIds).Delete(model.LdapAdAuth{}).Error
}

func (r LdapAdAuthRepository) DeleteByDepNotGen() (err error) {
	return r.DB.Where("department_id != 0").Delete(model.LdapAdAuth{}).Error
}

func (r LdapAdAuthRepository) StrategyCountByDepId(depId int64) (total int64, err error) {
	err = r.DB.Model(model.LdapAdAuth{}).Where("department_id = ?", depId).Count(&total).Error
	return
}

func (r LdapAdAuthRepository) BigStrategyPriority(depId int64, priority int) (o []model.LdapAdAuth, err error) {
	err = r.DB.Model(model.LdapAdAuth{}).Where("department_id = ? AND priority > ?", depId, priority).Find(&o).Error
	return
}

func (r LdapAdAuthRepository) UpdateDepLevelByDepId(depId int64, depLevel int) (err error) {
	return r.DB.Model(model.LdapAdAuth{}).Where("department_id = ?", depId).Update("dep_level", depLevel).Error
}

func (r LdapAdAuthRepository) FindPriorityRangeByDepId(smallPriority, bigPriority int, depId int64) (o []model.LdapAdAuth, err error) {
	err = r.DB.Where("department_id = ? AND (priority >= ? AND priority <= ?)", depId, smallPriority, bigPriority).Find(&o).Error
	return
}

func (r LdapAdAuthRepository) UpdateColById(id int64, col string, val interface{}) (err error) {
	err = r.DB.Model(model.LdapAdAuth{}).Where("id = ?", id).Update(col, val).Error
	return
}

// FindLdapAdServerAddressById 通过认证服务器id找到对应的名字
func (r LdapAdAuthRepository) FindLdapAdServerAddressById(id int64) (name string, err error) {
	err = r.DB.Model(model.LdapAdAuth{}).Where("id = ?", id).Select("ldap_ad_server_address").Find(&name).Error
	return
}
