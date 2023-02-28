package repository

import (
	"gorm.io/gorm"
	"tkbastion/server/model"
)

type MessageRepository struct {
	DB *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	messageRepository = &MessageRepository{DB: db}
	return messageRepository
}

func (r MessageRepository) Create(o *model.Message) (err error) {
	if err = r.DB.Create(o).Error; err != nil {
		return err
	}
	return nil
}

func (r MessageRepository) DeleteById(id string) (err error) {
	return r.DB.Where("id = ?", id).Delete(&model.Message{}).Error
}
func (r MessageRepository) UpdateById(o *model.Message, id string) (err error) {
	err = r.DB.Where("id = ?", id).Updates(&o).Error
	return
}
func (r MessageRepository) FindById(id string) (o model.Message, err error) {
	err = r.DB.Where("id = ?", id).Find(&o).Error
	return
}
func (r MessageRepository) FindByRecId(recId string) (o []model.Message, err error) {
	err = r.DB.Where("receive_id = ?", recId).Order("created desc").Find(&o).Error
	return
}

func (r MessageRepository) FindByRecIdAndTheme(recId, theme string) (o []model.Message, err error) {
	err = r.DB.Table("message").Where("receive_id = ? and type like ?", recId, "%"+theme+"%").Find(&o).Error
	return
}

func (r MessageRepository) FindAll() (o []model.Message, err error) {
	err = r.DB.Find(&o).Error
	return
}

func (r MessageRepository) Find(recId, auto, theme, level string) (o []model.Message, err error) {
	var searchLevel string
	if level == "高" || auto == "高" {
		searchLevel = "high"
	} else if level == "中" || auto == "中" {
		searchLevel = "middle"
	} else if level == "低" || auto == "低" {
		searchLevel = "low"
	}
	db := r.DB.Table("message").Where("receive_id = ?", recId)
	if auto != "" {
		db = db.Where("theme like ? or content like ? or level like ?", "%"+auto+"%", "%"+auto+"%", "%"+searchLevel+"%")
	} else {
		if theme != "" {
			db = db.Where("theme like ? ", "%"+theme+"%")
		}
		if level != "" {
			db = db.Where("level = ?", searchLevel)
		}
	}
	err = db.Order("created desc").Find(&o).Error
	return
}
