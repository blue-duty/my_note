package repository

import (
	"gorm.io/gorm"
	"tkbastion/server/model"
)

type PolicyConfigRepository struct {
	DB *gorm.DB
}

func NewPolicyConfigRepository(db *gorm.DB) *PolicyConfigRepository {
	policyConfigRepository := &PolicyConfigRepository{DB: db}
	return policyConfigRepository
}

func (r PolicyConfigRepository) FindAll() (o []model.PolicyConfig, err error) {
	err = r.DB.Find(&o).Error
	return
}

func (r PolicyConfigRepository) Create(o *model.PolicyConfig) error {
	return r.DB.Create(o).Error
}

func (r PolicyConfigRepository) Update(o *model.PolicyConfig) error {
	return r.DB.Select(`id`, `status_all`, `status_system_disk`, `status_data_disk`, `status_memory`, `status_cpu`, `continued_system_disk`, `continued_data_disk`, `continued_memory`, `continued_cpu`, `threshold_system_disk`, `threshold_data_disk`, `threshold_memory`, `threshold_cpu`, `path_system_disk`, `path_data_disk`, `frequency`, `frequency_time_type`).UpdateColumns(o).Error
}
func (r PolicyConfigRepository) FindByID(id string) (o model.PolicyConfigDTO, err error) {
	err = r.DB.Where("id = 1", id).First(&o).Error
	return
}

func (r PolicyConfigRepository) FindById(id string) (o *model.PolicyConfig, err error) {
	err = r.DB.Where("id = ?", id).First(&o).Error
	return
}

func (r PolicyConfigRepository) FindConfig() (o *model.PolicyConfig, err error) {
	err = r.DB.Where("id = ?", "1").First(&o).Error
	return
}
func (r PolicyConfigRepository) Truncate() error {
	return r.DB.Exec("TRUNCATE TABLE policy_configs").Error
}
