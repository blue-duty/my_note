package repository

import (
	"gorm.io/gorm"
	"tkbastion/server/model"
)

type JobExportRepository struct {
	DB *gorm.DB
}

func NewJobExportRepository(db *gorm.DB) *JobExportRepository {
	jobexportRepository = &JobExportRepository{DB: db}
	return jobexportRepository
}

func (r JobExportRepository) Create(o *model.JobExport) (err error) {
	return r.DB.Create(o).Error
}

func (r JobExportRepository) UpdateById(id string, o *model.JobExport) (err error) {
	o.ID = id
	return r.DB.Updates(o).Error
}

func (r JobExportRepository) FindById(id string) (o model.JobExport, err error) {
	err = r.DB.Where("id = ?", id).First(&o).Error
	return
}

func (r JobExportRepository) FindByName(name string) (o []model.JobExport, err error) {
	err = r.DB.Where("name like ?", "%"+name+"%").Find(&o).Error
	return
}

func (r JobExportRepository) DeleteById(id string) (err error) {
	return r.DB.Where("id =?", id).Delete(model.JobExport{}).Error
}

func (r JobExportRepository) FindAll() (o []model.JobExport, err error) {
	err = r.DB.Find(&o).Error
	return
}
func (r JobExportRepository) UpdateWrite(id string, write string) error {
	return r.DB.Table("job_export").Where("id = ? ", id).Update("job_export.write", write).Error
}
func (r JobExportRepository) FindBYWrite() (o model.JobExport, err error) {
	err = r.DB.Where("job_export.write = ?", "1").Find(&o).Error
	return
}

func (r JobExportRepository) Truncate() error {
	return r.DB.Exec("TRUNCATE TABLE job_export").Error
}
