package repository

import (
	"gorm.io/gorm"
	"tkbastion/server/model"
)

type RecipientsRepository struct {
	DB *gorm.DB
}

func NewRecipientsRepository(db *gorm.DB) *RecipientsRepository {
	recipientsRepository = &RecipientsRepository{DB: db}
	return recipientsRepository
}

func (r RecipientsRepository) Create(o *model.Recipients) (err error) {
	if err = r.DB.Create(o).Error; err != nil {
		return err
	}
	return nil
}

// 根据theme 找 recipient

func (r RecipientsRepository) FindByTheme(theme string) (o []model.Recipients, err error) {
	err = r.DB.Where("theme = ?", theme).Find(&o).Error
	return
}

// 通过id和用户查找Recipients

func (r RecipientsRepository) FindByThemeAndUserId(theme, userId string) (o *model.Recipients, err error) {
	err = r.DB.Table("recipients").Where("theme = ? and user_id = ?", theme, userId).First(o).Error
	return
}

func (r RecipientsRepository) FindByThemeAndUserGroupId(theme, userGroupId string) (o *model.Recipients, err error) {
	err = r.DB.Table("recipients").Where("theme = ? and user_group_id = ?", theme, userGroupId).First(o).Error
	return
}

func (r RecipientsRepository) DeleteByThemeAndUserId(theme, userId string) error {
	return r.DB.Where("theme = ? and user_id = ?", theme, userId).Delete(&model.Recipients{}).Error
}

func (r RecipientsRepository) DeleteByThemeAndUserGroupId(theme, userGroupId string) error {
	return r.DB.Where("theme = ? and user_group_id = ?", theme, userGroupId).Delete(&model.Recipients{}).Error
}
func (r RecipientsRepository) Truncate() error {
	return r.DB.Exec("TRUNCATE TABLE recipients").Error
}
