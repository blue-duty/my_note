package repository

import (
	"tkbastion/server/model"

	"gorm.io/gorm"
)

type NumRepository struct {
	DB *gorm.DB
}

func NewNumRepository(db *gorm.DB) *NumRepository {
	numRepository = &NumRepository{DB: db}
	return numRepository
}

func (r NumRepository) FindAll() (o []model.Num, err error) {
	err = r.DB.Find(&o).Error
	return
}

func (r NumRepository) Create(o *model.Num) (err error) {
	err = r.DB.Create(o).Error
	return
}
func (r NumRepository) Truncate() error {
	return r.DB.Exec("TRUNCATE TABLE nums").Error
}
