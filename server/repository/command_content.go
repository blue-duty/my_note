package repository

import (
	"gorm.io/gorm"
	"tkbastion/server/model"
)

type CommandContentRepository struct {
	DB *gorm.DB
}

func NewCommandContentRepository(db *gorm.DB) *CommandContentRepository {
	commandContentRepository = &CommandContentRepository{DB: db}
	return commandContentRepository
}

func (r CommandContentRepository) Create(o *model.CommandContent) (err error) {
	if err = r.DB.Create(o).Error; err != nil {
		return err
	}
	return nil
}

func (r CommandContentRepository) FindByContentId(id string) (o []model.CommandContent, err error) {
	err = r.DB.Where("content_id = ?", id).Find(&o).Error
	return
}
func (r CommandContentRepository) FindByStrategyId(id string) (o model.CommandContent, err error) {
	err = r.DB.Where("content_id = ?", id).Find(&o).Error
	return
}
func (r CommandContentRepository) FindById(id string) (o model.CommandContent, err error) {
	err = r.DB.Where("id = ?", id).Find(&o).Error
	return
}
func (r CommandContentRepository) FindByLimitingConditions(contentId, auto, content, isRegular string) (o []model.CommandContent, total int64, err error) {
	db := r.DB.Table("command_contents").Where("content_id = ?", contentId)
	if len(auto) > 0 {
		if auto == "否" {
			db = db.Where("content like ? or description like ?  or is_regular = ?", "%"+auto+"%", "%"+auto+"%", false)
		} else if auto == "是" {
			db = db.Where("content like ? or description like ?  or is_regular = ?", "%"+auto+"%", "%"+auto+"%", true)
		} else {
			db = db.Where("content like ? or description like ?", "%"+auto+"%", "%"+auto+"%")
		}
	}
	if len(content) > 0 {
		db = db.Where("content like ? ", "%"+content+"%")
	}
	if len(isRegular) > 0 {
		if isRegular == "否" {
			db = db.Where("is_regular = ?", false)
		}
		if isRegular == "是" {
			db = db.Where("is_regular = ?", true)
		}
	}
	err = db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = db.Find(&o).Error
	return
}

func (r CommandContentRepository) UpdateByContendId(o *model.CommandContent, id string) error {
	return r.DB.Where("content_id = ?", id).Updates(o).Error
}
func (r CommandContentRepository) UpdateById(o *model.CommandContent, id string) error {
	return r.DB.Where("id = ? ", id).Updates(o).Error
}

func (r CommandContentRepository) DeleteById(id string) error {
	return r.DB.Where("id = ?", id).Delete(&model.CommandContent{}).Error
}

func (r CommandContentRepository) DeleteByContentId(contentId string) error {
	return r.DB.Where("content_id = ?", contentId).Delete(&model.CommandContent{}).Error
}
