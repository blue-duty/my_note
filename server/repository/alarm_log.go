package repository

import (
	"context"
	"fmt"
	"tkbastion/pkg/log"
	"tkbastion/server/dto"
	"tkbastion/server/model"
)

type OperateAlarmLogRepository struct {
	baseRepository
}

var OperateAlarmLogRepo = new(OperateAlarmLogRepository)

func (r OperateAlarmLogRepository) Create(c context.Context, o *model.OperateAlarmLog) (err error) {
	return r.GetDB(c).Create(o).Error
}

func (r OperateAlarmLogRepository) FindByAlarmLogForSearch(c context.Context, o dto.OperateAlarmLogForSearch) (o2 []dto.OperateAlarmLog, err error) {
	db := r.GetDB(c).Table("operate_alarm_log").Select("DATE_FORMAT(operate_alarm_log.alarm_time,'%Y-%m-%d %H:%i:%s') as alarm_time,operate_alarm_log.client_ip as client_ip,user_new.username as username,user_new.nickname as nickname,new_assets.ip as asset_ip ,pass_ports.protocol as protocol,pass_ports.passport as passport,command_strategy.name as strategy,operate_alarm_log.content as content,operate_alarm_log.level as level,operate_alarm_log.result as result").Joins("left join user_new on operate_alarm_log.user_id = user_new.id").Joins("left join new_assets on operate_alarm_log.asset_id = new_assets.id").Joins("left join pass_ports on operate_alarm_log.passport_id = pass_ports.id").Joins("left join command_strategy on operate_alarm_log.command_strategy_id = command_strategy.id")
	if o.Auto != "" {
		db = db.Where("alarm_time like ? or client_ip like ? or username like ? or asset_ip like ? or passport like ? or content like ? or level like ?", "%"+o.Auto+"%", "%"+o.Auto+"%", "%"+o.Auto+"%", "%"+o.Auto+"%", "%"+o.Auto+"%", "%"+o.Auto+"%", "%"+o.Auto+"%")
	} else if o.ClientIP != "" {
		db = db.Where("client_ip like ?", "%"+o.ClientIP+"%")
	} else if o.Username != "" {
		db = db.Where("username like ?", "%"+o.Username+"%")
	} else if o.AssetIp != "" {
		db = db.Where("asset_ip like ?", "%"+o.AssetIp+"%")
	} else if o.Passport != "" {
		db = db.Where("passport like ?", "%"+o.Passport+"%")
	} else if o.Content != "" {
		db = db.Where("content like ?", "%"+o.Content+"%")
	} else if o.Level != "" {
		db = db.Where("level like ?", "%"+o.Level+"%")
	} else {
		db = db.Where("1=1")
	}
	err = db.Order("alarm_time desc").Scan(&o2).Error
	return
}

func (r OperateAlarmLogRepository) CreateSystemAlarmLog(c context.Context, o *model.SystemAlarmLog) (err error) {
	return r.GetDB(c).Table("system_alarm_log").Create(o).Error
}

func (r OperateAlarmLogRepository) GetSystemAlarmCount(c context.Context) (o int64, err error) {
	err = r.GetDB(c).Table("system_alarm_log").Count(&o).Error
	return
}

func (r OperateAlarmLogRepository) FindBySystemAlarmLogForSearch(c context.Context, auto, content, level string) (o []model.SystemAlarmLog, err error) {
	db := r.GetDB(c).Table("system_alarm_log")
	if auto != "" {
		db = db.Where("strategy like ? or content like ? or level like ? or result like ?", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%")
	} else if content != "" {
		db = db.Where("content like ?", "%"+content+"%")
	} else if level != "" {
		db = db.Where("level like ?", "%"+level+"%")
	} else {
		db = db.Where("1=1")
	}
	err = db.Order("alarm_time desc").Scan(&o).Error
	return
}

// 查询告警日志

func (r OperateAlarmLogRepository) FindByAlarmLogByDate(c context.Context, start, end, searchType string) (o []dto.AlarmReport, err error) {
	// 查询告警日志
	if searchType == "" {
		searchType = "%Y-%m-%d"
	}
	fmt.Println(start, end, searchType)
	sql := `select DATE_FORMAT( operate_alarm_log.alarm_time,'` + searchType + `') as alarm_time,sum(operate_alarm_log.level = 'high') as high_alert,sum( operate_alarm_log.level = 'middle') as middle_alert,sum( operate_alarm_log.level = 'low') as low_alert from operate_alarm_log WHERE DATE_FORMAT( operate_alarm_log.alarm_time,'` + searchType + `') BETWEEN ? AND ? group by DATE_FORMAT(operate_alarm_log.alarm_time,'` + searchType + `') order by alarm_time desc;`
	err = r.GetDB(c).Raw(sql, start, end).Find(&o).Error
	fmt.Println(o)
	return
}

// 查询详细告警日志
func (r OperateAlarmLogRepository) GetAlarmLogDetailsStatist(c context.Context, startTime, endTime, level string) ([]dto.AlarmReportDetail, error) {
	var o []model.OperateAlarmLog
	db := r.GetDB(c).Table("operate_alarm_log").Where("alarm_time between ? and ? ", startTime, endTime)
	if level != "" {
		db = db.Where("level = ?", level)
	}
	err := db.Find(&o).Error
	details := make([]dto.AlarmReportDetail, len(o))
	for i, v := range o {
		user, err := userNewRepository.FindById(v.UserId)
		if err != nil {
			log.Errorf("GetAlarmLogDetailsStatist FindById err: %v", err)
		}
		strategy, err := commandStrategyRepository.FindById(v.CommandStrategyId)
		if err != nil {
			log.Errorf("GetAlarmLogDetailsStatist FindById err: %v", err)
		}
		// 查询newAssetRepository表
		var passport model.PassPort
		err = r.GetDB(c).Table("pass_ports").Where("id = ?", v.PassportId).Find(&passport).Error
		if err != nil {
			log.Errorf("GetAlarmLogDetailsStatist FindById err: %v", err)
		}
		var levelTemp string
		if v.Level == "high" {
			levelTemp = "高"
		} else if v.Level == "middle" {
			levelTemp = "中"
		} else {
			levelTemp = "低"
		}
		details[i] = dto.AlarmReportDetail{
			AlarmTime:  v.AlarmTime.Format("2006-01-02 15:04:05"),
			Username:   user.Username,
			Nickname:   user.Nickname,
			ClientIp:   v.ClientIP,
			AssetIp:    passport.Ip,
			Passport:   passport.Passport,
			Protocol:   passport.Protocol,
			AlarmRule:  strategy.Name,
			AlarmLevel: levelTemp,
		}
	}
	return details, err
}
