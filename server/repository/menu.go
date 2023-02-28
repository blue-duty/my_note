package repository

import (
	"tkbastion/server/model"

	"gorm.io/gorm"
)

type MenuRepository struct {
	DB *gorm.DB
}

func NewMenuRepository(db *gorm.DB) *MenuRepository {
	menuRepository = &MenuRepository{DB: db}
	return menuRepository
}

func (r MenuRepository) FindAll() (o []model.Menu, err error) {
	err = r.DB.Find(&o).Error
	return
}
