package repository

import (
	"gorm.io/gorm"
	"tkbastion/server/model"
)

type CommandSetRepository struct {
	DB *gorm.DB
}

func NewCommandSetRepository(db *gorm.DB) *CommandSetRepository {
	commandSetRepository = &CommandSetRepository{DB: db}
	return commandSetRepository
}

func (r CommandSetRepository) FindAll() (o []model.CommandSet, err error) {
	err = r.DB.Find(&o).Error
	return
}

func (r CommandSetRepository) Create(o *model.CommandSet) (err error) {
	if err = r.DB.Create(o).Error; err != nil {
		return err
	}
	return nil
}

func (r CommandSetRepository) FindById(id string) (o model.CommandSet, err error) {
	err = r.DB.Where("id = ?", id).First(&o).Error
	return
}

func (r CommandSetRepository) FindByLimitingConditions(pageIndex, pageSize int, auto, name, level, description string) (o []model.CommandSet, total int64, err error) {
	db := r.DB.Table("command_sets")
	if len(auto) > 0 {
		db = db.Where("name like ? or level like ? or description like ? or content like ? ", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%")
	} else {
		if len(name) > 0 {
			db = db.Where("name like ? ", "%"+name+"%")
		}
		if len(level) > 0 {
			db = db.Where("level like ? ", "%"+level+"%")
		}
		if len(description) > 0 {
			db = db.Where("description like ? ", "%"+description+"%")
		}
	}
	err = db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = db.Order("created desc").Find(&o).Error
	return
}

func (r CommandSetRepository) UpdateById(o *model.CommandSet, id string) error {
	o.ID = id
	return r.DB.Updates(o).Error
}

func (r CommandSetRepository) DeleteById(id string) error {
	return r.DB.Where("id = ?", id).Delete(&model.CommandSet{}).Error
}

func (r CommandSetRepository) Truncate() error {
	return r.DB.Exec("TRUNCATE TABLE command_sets").Error
}
