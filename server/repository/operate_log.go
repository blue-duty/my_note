package repository

import (
	"strings"
	"time"
	"tkbastion/server/model"

	"gorm.io/gorm"
)

type OperateLogRepository struct {
	DB *gorm.DB
}

func NewOperateLogRepository(db *gorm.DB) *OperateLogRepository {
	operateLogRepository = &OperateLogRepository{DB: db}
	return operateLogRepository
}

func (r OperateLogRepository) Find(auto, ipAddress, user, name, functionalModule, action string) (o []model.OperateForPage, err error) {
	db := r.DB.Table("operate_logs").Select("operate_logs.id,operate_logs.created,operate_logs.log_types,operate_logs.client_user_agent,operate_logs.log_contents, operate_logs.users, operate_logs.names, operate_logs.ip, operate_logs.result")

	if ipAddress != "" {
		db = db.Where("operate_logs.ip like ?", "%"+ipAddress+"%")
	}

	if user != "" {
		db = db.Where("operate_logs.users like ?", "%"+user+"%")
	}

	if name != "" {
		db = db.Where("operate_logs.names like ?", "%"+name+"%")
	}

	if "" != functionalModule {
		db = db.Where("operate_logs.log_contents like ?", "%"+functionalModule+"%")
	}

	if "" != action {
		db = db.Where("operate_logs.log_contents like ?", "%"+action+"%")
	}

	if "" != auto {
		result := auto
		if strings.Contains(auto, "成") || strings.Contains(auto, "功") {
			result = "成功"
		} else if strings.Contains(auto, "失") || strings.Contains(auto, "败") {
			result = "失败"
		}

		db = db.Where("operate_logs.ip like ? OR operate_logs.users like ? OR operate_logs.names like ? OR operate_logs.log_contents like ? OR operate_logs.result like ? OR operate_logs.created like ?", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+result+"%", "%"+auto+"%")
	}

	err = db.Order("operate_logs.created desc").Find(&o).Error
	if o == nil {
		o = make([]model.OperateForPage, 0)
	}
	return
}

func (r OperateLogRepository) Create(o *model.OperateLog) (err error) {
	return r.DB.Create(o).Error
}

func (r OperateLogRepository) DeleteByIdIn(ids []int) (err error) {
	return r.DB.Where("id in ?", ids).Delete(&model.OperateLog{}).Error
}

// FindOutTimeOperationLogs 查询超时的操作日志
func (r OperateLogRepository) FindOutTimeOperationLogs(dayLimit int) (o []model.OperateLog, err error) {
	limitTime := time.Now().Add(time.Duration(-dayLimit*24) * time.Hour)
	err = r.DB.Where("created < ?", limitTime).Find(&o).Error
	if o == nil {
		o = make([]model.OperateLog, 0)
	}
	return
}

func (r OperateLogRepository) Truncate() error {
	return r.DB.Exec("TRUNCATE TABLE operate_logs").Error
}
