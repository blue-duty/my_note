package repository

import (
	"time"
	"tkbastion/server/model"

	"gorm.io/gorm"
)

type OrderLogRepository struct {
	DB *gorm.DB
}

func NewOrderLogRepository(db *gorm.DB) *OrderLogRepository {
	orderLogRepository = &OrderLogRepository{DB: db}
	return orderLogRepository
}

func (r OrderLogRepository) Find(pageIndex, pageSize int, asset, applicant, approved, ip, result, beginTime, endTime, applicationType string) (o []model.OrderLog, total int64, err error) {

	db := r.DB.Table("order_logs")
	dbCounter := r.DB.Table("order_logs").Select("DISTINCT order_logs.id")

	if asset != "" {
		db = db.Where("order_logs.asset = ?", asset)
		dbCounter = dbCounter.Where("order_logs.asset = ?", asset)
	}

	if applicant != "" {
		db = db.Where("order_logs.applicant = ?", applicant)
		dbCounter = dbCounter.Where("order_logs.applicant = ?", applicant)
	}
	if approved != "" {
		db = db.Where("order_logs.approved = ?", approved)
		dbCounter = dbCounter.Where("order_logs.approved = ?", approved)
	}
	if ip != "" {
		db = db.Where("order_logs.ip like ?", "%"+ip+"%")
		dbCounter = dbCounter.Where("order_logs.ip like ?", "%"+ip+"%")
	}
	if result != "" {
		db = db.Where("order_logs.status = ?", result)
		dbCounter = dbCounter.Where("order_logs.status = ?", result)
	}
	if "" != beginTime && "" != endTime {
		db = db.Where("unix_timestamp(order_logs.created) >= unix_timestamp(?) AND unix_timestamp(order_logs.created) <= unix_timestamp(?)", beginTime, endTime)
		dbCounter = dbCounter.Where("unix_timestamp(order_logs.created) >= unix_timestamp(?) AND unix_timestamp(order_logs.created) <= unix_timestamp(?)", beginTime, endTime)
	} else if "" != beginTime && "" == endTime {
		db = db.Where("unix_timestamp(order_logs.created) >= unix_timestamp(?)", beginTime)
		dbCounter = dbCounter.Where("unix_timestamp(order_logs.created) >= unix_timestamp(?)", beginTime)
	} else if "" == beginTime && "" != endTime {
		db = db.Where("unix_timestamp(order_logs.created) <= unix_timestamp(?)", endTime)
		dbCounter = dbCounter.Where("unix_timestamp(order_logs.created) <= unix_timestamp(?)", endTime)
	}

	if applicationType != "" {
		db = db.Where("order_logs.application_type = ?", applicationType)
		dbCounter = dbCounter.Where("order_logs.application_type = ?", applicationType)
	}

	db = db.Order("order_logs.created desc").Offset((pageIndex - 1) * pageSize).Limit(pageSize)
	err = dbCounter.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = db.Find(&o).Error
	if o == nil {
		o = make([]model.OrderLog, 0)
	}
	return
}

func (r OrderLogRepository) Export(asset, applicant, approved, ip, result, atype, beginTime, endTime string) (o []model.OrderLog, total int64, err error) {
	db := r.DB.Table("order_logs")
	dbCounter := r.DB.Table("order_logs").Select("DISTINCT order_logs.id")

	if asset != "" {
		db = db.Where("order_logs.asset = ?", asset)
		dbCounter = dbCounter.Where("order_logs.asset = ?", asset)
	}

	if applicant != "" {
		db = db.Where("order_logs.applicant = ?", applicant)
		dbCounter = dbCounter.Where("order_logs.applicant = ?", applicant)
	}
	if approved != "" {
		db = db.Where("order_logs.approved = ?", approved)
		dbCounter = dbCounter.Where("order_logs.approved = ?", approved)
	}
	if ip != "" {
		db = db.Where("order_logs.ip like ?", "%"+ip+"%")
		dbCounter = dbCounter.Where("order_logs.ip like ?", "%"+ip+"%")
	}
	if result != "" {
		db = db.Where("order_logs.status = ?", result)
		dbCounter = dbCounter.Where("order_logs.status = ?", result)
	}

	if atype != "" {
		db = db.Where("order_logs.application_type = ?", atype)
		dbCounter = dbCounter.Where("order_logs.application_type = ?", atype)
	}
	if "" != beginTime && "" != endTime {
		db = db.Where("unix_timestamp(order_logs.created) >= unix_timestamp(?) AND unix_timestamp(order_logs.created) <= unix_timestamp(?)", beginTime, endTime)
		dbCounter = dbCounter.Where("unix_timestamp(order_logs.created) >= unix_timestamp(?) AND unix_timestamp(order_logs.created) <= unix_timestamp(?)", beginTime, endTime)
	} else if "" != beginTime && "" == endTime {
		db = db.Where("unix_timestamp(order_logs.created) >= unix_timestamp(?)", beginTime)
		dbCounter = dbCounter.Where("unix_timestamp(order_logs.created) >= unix_timestamp(?)", beginTime)
	} else if "" == beginTime && "" != endTime {
		db = db.Where("unix_timestamp(order_logs.created) <= unix_timestamp(?)", endTime)
		dbCounter = dbCounter.Where("unix_timestamp(order_logs.created) <= unix_timestamp(?)", endTime)
	}
	db = db.Order("order_logs.created desc")
	err = dbCounter.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = db.Find(&o).Error
	if o == nil {
		o = make([]model.OrderLog, 0)
	}
	return
}

func (r OrderLogRepository) Create(o *model.OrderLog) error {
	return r.DB.Create(o).Error
}

func (r OrderLogRepository) DeleteLogById(id string) error {
	return r.DB.Where("id = ?", id).Delete(&model.OrderLog{}).Error
}

// GetExpiredOrderLog 查询超出时间的审批日志
func (r OrderLogRepository) GetExpiredOrderLog(dayLimit int) (o []model.OrderLog, err error) {
	limitTime := time.Now().Add(time.Duration(-dayLimit*24) * time.Hour)
	err = r.DB.Where("order_logs.created <= ?", limitTime).Find(&o).Error
	if o == nil {
		o = make([]model.OrderLog, 0)
	}
	return
}

func (r OrderLogRepository) Truncate() error {
	return r.DB.Exec("TRUNCATE TABLE order_logs").Error
}
