package repository

import (
	"gorm.io/gorm"
	"tkbastion/server/model"
	"tkbastion/server/utils"
)

type RegularReportRepository struct {
	DB *gorm.DB
}

func NewRegularReportRepository(db *gorm.DB) *RegularReportRepository {
	regularReportRepository = &RegularReportRepository{DB: db}
	return regularReportRepository
}

func (r *RegularReportRepository) Create(regularReport *model.RegularReport) error {
	return r.DB.Create(regularReport).Error
}

func (r *RegularReportRepository) FindById(id string) (regularReport model.RegularReport, err error) {
	err = r.DB.Where("id = ?", id).First(&regularReport).Error
	return
}

func (r *RegularReportRepository) UpdateById(regularReport *model.RegularReport, id string) error {
	return r.DB.Model(&model.RegularReport{}).Where("id = ?", id).Updates(regularReport).Error
}

func (r *RegularReportRepository) DeleteById(id string) error {
	return r.DB.Where("id = ?", id).Delete(&model.RegularReport{}).Error
}

func (r *RegularReportRepository) FindByConditions(auto, name, description string) (regularReports []model.RegularReport, err error) {
	db := r.DB.Table("regular_reports")
	if auto != "" {
		var str string
		if auto == "天" {
			str = "day"
		} else if auto == "周" {
			str = "week"
		} else if auto == "月" {
			str = "month"
		}
		db = db.Where("name like ? or description like ? or periodic_type like ?", "%"+auto+"%", "%"+auto+"%", "%"+str+"%")
	} else {
		if name != "" {
			db = db.Where("name like ?", "%"+name+"%")
		}
		if description != "" {
			db = db.Where("description like ?", "%"+description+"%")
		}
	}
	err = db.Find(&regularReports).Error
	return
}

// CreateRegularReportLog 创建定期策略执行的日志
func (r *RegularReportRepository) CreateRegularReportLog(name, periodicType, reportType, filename string) error {
	var regularReportLog = model.RegularReportLog{
		ID:           utils.UUID(),
		Name:         name,
		ExecuteTime:  utils.NowJsonTime(),
		PeriodicType: periodicType,
		ReportType:   reportType,
		FileName:     filename,
	}
	err := r.DB.Table("regular_reports_log").Create(&regularReportLog).Error
	return err
}

// FindRegularReportLogByConditions 根据条件查询定期策略执行的日志
func (r *RegularReportRepository) FindRegularReportLogByConditions(auto, name, executeTime string) (regularReportLogs []model.RegularReportLog, err error) {
	db := r.DB.Table("regular_reports_log")
	if auto != "" {
		db = db.Where("name like ? or periodic_type like ? or report_type like ?", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%")
	} else {
		if name != "" {
			db = db.Where("name like ?", "%"+name+"%")
		}
		if executeTime != "" {
			db = db.Where("execute_time like ?", "%"+executeTime+"%")
		}
	}
	err = db.Find(&regularReportLogs).Error
	return
}
