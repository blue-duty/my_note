package repository

import (
	"tkbastion/server/model"

	"gorm.io/gorm"
)

type UserCollecteRepository struct {
	DB *gorm.DB
}

func NewUserCollecteRepository(db *gorm.DB) *UserCollecteRepository {
	userCollecteRepository = &UserCollecteRepository{DB: db}
	return userCollecteRepository
}

func (r *UserCollecteRepository) Create(o *model.UserCollecte) (err error) {
	return r.DB.Create(o).Error
}

func (r *UserCollecteRepository) CreateAppCollecte(o *model.UserCollectApp) (err error) {
	return r.DB.Create(o).Error
}

func (r *UserCollecteRepository) RemoveAppCollecte(o *model.UserCollectApp) (err error) {
	return r.DB.Where("user_id = ? AND application_id = ?", o.UserId, o.ApplicationId).Delete(&model.UserCollectApp{}).Error
}

func (r *UserCollecteRepository) FindByUserId(userId string) (o []model.UserCollecte, err error) {
	err = r.DB.Where("user_id = ?", userId).Find(&o).Error
	return
}

func (r *UserCollecteRepository) FindCollectAppByUserId(userId string) (o []model.UserCollectApp, err error) {
	err = r.DB.Where("user_id = ?", userId).Find(&o).Error
	return
}

func (r *UserCollecteRepository) FindByUserIdStrArr(userId string) (assetAccountStrArr []string, err error) {
	err = r.DB.Model(model.UserCollecte{}).Where("user_id = ?", userId).Pluck("asset_account_id", &assetAccountStrArr).Error
	return
}

func (r *UserCollecteRepository) DeleteById(id int64) error {
	return r.DB.Where("id = ?", id).Delete(&model.UserCollecte{}).Error
}

func (r *UserCollecteRepository) DeleteByUserIdAssetAccountId(userId, assetAccountId string) error {
	return r.DB.Model(model.UserCollecte{}).Where("user_id = ? AND asset_account_id = ?", userId, assetAccountId).Delete(&model.UserCollecte{}).Error
}

func (r *UserCollecteRepository) DeleteAssetAccountByUser(userId string) error {
	return r.DB.Model(model.UserCollecte{}).Where("user_id = ?", userId).Delete(&model.UserCollecte{}).Error
}
