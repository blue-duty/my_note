package repository

import (
	"tkbastion/server/dto"
	"tkbastion/server/model"

	"gorm.io/gorm"
)

type HostOperateRepository struct {
	DB *gorm.DB
}

func NewHostOperateRepository(db *gorm.DB) *HostOperateRepository {
	hostOperateRepository = &HostOperateRepository{DB: db}
	return hostOperateRepository
}

// GetOperateLogList 运维日志
func (h *HostOperateRepository) GetOperateLogList(oplfs dto.OperateLogForSearch) ([]dto.OperateLog, error) {
	var operateLog []dto.OperateLog
	db := h.DB.Model(&model.OperationAndMaintenanceLog{}).Select("DATE_FORMAT(login_time,'%Y-%m-%d %H:%i:%s') as login_time,DATE_FORMAT(logout_time,'%Y-%m-%d %H:%i:%s') as logout_time,ip,username,nickname,protocol,asset_name,asset_ip,passport").Where("department_id in (?)", oplfs.Departments)
	if oplfs.Auto != "" {
		db = db.Where("username like ? or nickname like ? or ip like ? or asset_ip like ? or asset_name like ? or passport like ? or login_time like ? or logout_time like ?", "%"+oplfs.Auto+"%", "%"+oplfs.Auto+"%", "%"+oplfs.Auto+"%", "%"+oplfs.Auto+"%", "%"+oplfs.Auto+"%", "%"+oplfs.Auto+"%", "%"+oplfs.Auto+"%", "%"+oplfs.Auto+"%")
	} else if oplfs.Username != "" {
		db = db.Where("username like ?", "%"+oplfs.Username+"%")
	} else if oplfs.Nickname != "" {
		db = db.Where("nickname like ?", "%"+oplfs.Nickname+"%")
	} else if oplfs.Ip != "" {
		db = db.Where("ip like ?", "%"+oplfs.Ip+"%")
	} else if oplfs.AssetIp != "" {
		db = db.Where("asset_ip like ?", "%"+oplfs.AssetIp+"%")
	} else if oplfs.AssetName != "" {
		db = db.Where("asset_name like ?", "%"+oplfs.AssetName+"%")
	} else if oplfs.Passport != "" {
		db = db.Where("passport like ?", "%"+oplfs.Passport+"%")
	} else if oplfs.LoginTime != "" {
		db = db.Where("login_time like ?", "%"+oplfs.LoginTime+"%")
	} else if oplfs.LogoutTime != "" {
		db = db.Where("logout_time like ?", "%"+oplfs.LogoutTime+"%")
	}
	err := db.Order("logout_time desc").Find(&operateLog).Error
	if err != nil {
		return nil, err
	}
	return operateLog, nil
}

func (h *HostOperateRepository) GetOperateLogForExport(dp []int64) ([]dto.OperateLogForExport, error) {
	var operateLog []model.OperationAndMaintenanceLog
	db := h.DB.Model(&model.OperationAndMaintenanceLog{}).Where("department_id in (?)", dp)
	err := db.Find(&operateLog).Error
	if err != nil {
		return nil, err
	}
	var operateLogList []dto.OperateLogForExport
	for _, v := range operateLog {
		var logForExport dto.OperateLogForExport
		logForExport.LoginTime = v.LoginTime.Format("2006-01-02 15:04:05")
		logForExport.Username = v.Username
		logForExport.Nickname = v.Nickname
		logForExport.AssetIp = v.AssetIp
		logForExport.AssetName = v.AssetName
		logForExport.Ip = v.Ip
		logForExport.Passport = v.Passport
		logForExport.LogoutTime = v.LogoutTime.Format("2006-01-02 15:04:05")
		if v.Protocol != "应用" {
			logForExport.Type = "应用"
		} else {
			logForExport.Type = "设备"
		}
		operateLogList = append(operateLogList, logForExport)
	}
	return operateLogList, nil
}

// GetOperateLogsForExport GetOperateLogListByUser 运维日志
func (h *HostOperateRepository) GetOperateLogsForExport() ([]dto.OperateLogForExport, error) {
	var operateLog []model.OperationAndMaintenanceLog
	err := h.DB.Model(&model.OperationAndMaintenanceLog{}).Find(&operateLog).Error
	if err != nil {
		return nil, err
	}
	var operateLogList []dto.OperateLogForExport
	for _, v := range operateLog {
		var logForExport dto.OperateLogForExport
		logForExport.LoginTime = v.LoginTime.Format("2006-01-02 15:04:05")
		logForExport.Username = v.Username
		logForExport.Nickname = v.Nickname
		logForExport.AssetIp = v.AssetIp
		logForExport.AssetName = v.AssetName
		logForExport.Ip = v.Ip
		logForExport.Passport = v.Passport
		logForExport.LogoutTime = v.LogoutTime.Format("2006-01-02 15:04:05")
		if v.Protocol != "应用" {
			logForExport.Type = "应用"
		} else {
			logForExport.Type = "设备"
		}
		operateLogList = append(operateLogList, logForExport)
	}
	return operateLogList, nil
}
