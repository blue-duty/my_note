package repository

import (
	"tkbastion/server/model"

	"gorm.io/gorm"
)

type AssetAuthReportFormRepository struct {
	DB *gorm.DB
}

func NewAssetAuthReportFormRepository(db *gorm.DB) *AssetAuthReportFormRepository {
	assetAuthReportFormRepository = &AssetAuthReportFormRepository{DB: db}
	return assetAuthReportFormRepository
}

func (r AssetAuthReportFormRepository) Create(o *model.AssetAuthReportForm) (err error) {
	return r.DB.Create(o).Error
}

func (r AssetAuthReportFormRepository) Find() (o []model.AssetAuthReportForm, err error) {
	err = r.DB.Find(&o).Error
	return
}

func (r AssetAuthReportFormRepository) FindWithCondition(auto, assetName, assetAddress, assetAccount, userName string) (o []model.AssetAuthReportForm, err error) {
	db := r.DB.Table("asset_auth_report_form").Select("asset_auth_report_form.id, asset_auth_report_form.asset_account_id, asset_auth_report_form.asset_name, asset_auth_report_form.asset_address, asset_auth_report_form.asset_account, asset_auth_report_form.user_id, asset_auth_report_form.username, asset_auth_report_form.nickname, asset_auth_report_form.operate_auth_id, asset_auth_report_form.operate_auth_name")

	if assetName != "" {
		db = db.Where("asset_auth_report_form.asset_name like ?", "%"+assetName+"%")
	}

	if assetAddress != "" {
		db = db.Where("asset_auth_report_form.asset_address like ?", "%"+assetAddress+"%")
	}

	if assetAccount != "" {
		db = db.Where("asset_auth_report_form.asset_account like ?", "%"+assetAccount+"%")
	}

	if "" != userName {
		db = db.Where("asset_auth_report_form.username like ?", "%"+userName+"%")
	}

	if "" != auto {
		db = db.Where("asset_auth_report_form.asset_name like ? OR asset_auth_report_form.asset_address like ? OR asset_auth_report_form.asset_account like ? OR asset_auth_report_form.username like ? OR asset_auth_report_form.nickname like ? OR asset_auth_report_form.operate_auth_name like ?", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%")
	}

	err = db.Order("asset_auth_report_form.id desc").Find(&o).Error
	if o == nil {
		o = make([]model.AssetAuthReportForm, 0)
	}
	return
}

func (r AssetAuthReportFormRepository) UpdateById(id int64, o *model.AssetAuthReportForm) (err error) {
	o.ID = id
	return r.DB.Where("id = ?", id).Updates(o).Error
}

func (r AssetAuthReportFormRepository) DeleteByOperateAuthId(id int64) (err error) {
	return r.DB.Where("operate_auth_id = ?", id).Delete(model.AssetAuthReportForm{}).Error
}

func (r AssetAuthReportFormRepository) DeleteByUserId(id string) (err error) {
	return r.DB.Where("user_id = ?", id).Delete(model.AssetAuthReportForm{}).Error
}
