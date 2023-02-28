package repository

import (
	"strconv"
	"strings"
	"time"
	"tkbastion/server/dto"
	"tkbastion/server/model"

	"gorm.io/gorm"
)

type LoginLogRepository struct {
	DB *gorm.DB
}

func NewLoginLogRepository(db *gorm.DB) *LoginLogRepository {
	loginLogRepository = &LoginLogRepository{DB: db}
	return loginLogRepository
}

func (r LoginLogRepository) Find(auto, ipAddress, user, name, loginType, loginResult string) (o []model.LoginLogForPageNew, err error) {
	db := r.DB.Table("login_logs").Select("login_logs.id, login_logs.user_id, login_logs.client_ip, login_logs.client_user_agent, login_logs.login_time, login_logs.logout_time, login_logs.login_result, login_logs.username, login_logs.nickname, login_logs.protocol, login_logs.login_type, login_logs.description")

	if ipAddress != "" {
		db = db.Where("login_logs.client_ip like ?", "%"+ipAddress+"%")
	}

	if user != "" {
		db = db.Where("login_logs.username like ?", "%"+user+"%")
	}

	if name != "" {
		db = db.Where("login_logs.nickname like ?", "%"+name+"%")
	}

	if "" != loginType {
		loginType = strings.ToUpper(loginType)
		db = db.Where("login_logs.login_type like ?", "%"+loginType+"%")
	}

	if "" != loginResult {
		if strings.Contains(loginResult, "成") || strings.Contains(loginResult, "功") {
			loginResult = "成功"
		} else if strings.Contains(loginResult, "失") || strings.Contains(loginResult, "败") {
			loginResult = "失败"
		}

		db = db.Where("login_logs.login_result like ?", "%"+loginResult+"%")
	}

	if "" != auto {
		loginResult = auto
		if strings.Contains(auto, "成") || strings.Contains(auto, "功") {
			loginResult = "成功"
		} else if strings.Contains(auto, "失") || strings.Contains(auto, "败") {
			loginResult = "失败"
		}

		protocol := strings.ToLower(auto)
		if strings.Contains(protocol, "https") {
			protocol = "https"
		} else if strings.Contains(protocol, "http") {
			protocol = "http"
		}

		db = db.Where("login_logs.client_ip like ? OR login_logs.username like ? OR login_logs.nickname like ? OR login_logs.login_type like ? OR login_logs.login_result like ? OR login_logs.login_time like ? OR login_logs.description like ? OR login_logs.protocol like ?", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+loginResult+"%", "%"+auto+"%", "%"+auto+"%", "%"+protocol+"%")
	}

	err = db.Order("login_logs.login_time desc").Find(&o).Error
	if o == nil {
		o = make([]model.LoginLogForPageNew, 0)
	}
	return
}

func (r LoginLogRepository) FindAliveLoginLogsByUserId(userId string) (o []model.LoginLog, err error) {
	err = r.DB.Where("logout_time is null and user_id = ?", userId).Find(&o).Error
	return
}

func (r LoginLogRepository) Create(o *model.LoginLog) (err error) {
	return r.DB.Create(o).Error
}

func (r LoginLogRepository) DeleteByIdIn(ids []string) (err error) {
	return r.DB.Where("id in ?", ids).Delete(&model.LoginLog{}).Error
}

func (r LoginLogRepository) FindById(id string) (o model.LoginLog, err error) {
	err = r.DB.Where("id = ?", id).First(&o).Error
	return
}

func (r LoginLogRepository) Update(o *model.LoginLog) error {
	return r.DB.Updates(o).Error
}

// FindOutTimeLoginLogs 查询超出保存期限的登录日志
func (r LoginLogRepository) FindOutTimeLoginLogs(dayLimit int) (o []model.LoginLog, err error) {
	limitTime := time.Now().Add(time.Duration(-dayLimit*24) * time.Hour)
	err = r.DB.Where("logout_time < ?", limitTime).Find(&o).Error
	return
}
func (r LoginLogRepository) DeleteExceptOnline() error {
	return r.DB.Transaction(func(tx *gorm.DB) (err error) {
		// 非在线用户的登录日志
		err = tx.Where("logout_time is not null").Delete(&model.LoginLog{}).Error
		if err != nil {
			return err
		}
		return nil
	})
}

func (r LoginLogRepository) FindForPaging(pageIndex, pageSize int, auto, sourceIp, username, nickname, loginType, loginResult string) (o []dto.LoginLogForPage, total int64, err error) {
	db := r.DB.Table("login_logs").Select("id,login_time,client_ip,source,login_type,login_result,username,nickname,description")
	if auto != "" {
		db = db.Where("sourceIp like ? or username like ? or nickname like ? or login_type like ? or login_result like ?", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%")
	} else {
		if sourceIp != "" {
			db = db.Where("sourceIp like ?", "%"+sourceIp+"%")
		}
		if username != "" {
			db = db.Where("username like ?", "%"+username+"%")
		}
		if nickname != "" {
			db = db.Where("nickname like ?", "%"+nickname+"%")
		}
		if loginType != "" {
			db = db.Where("login_type like ?", "%"+loginType+"%")
		}
		if loginResult != "" {
			db = db.Where("login_result like ?", "%"+loginResult+"%")
		}
	}
	err = db.Count(&total).Error
	if err != nil {
		return
	}
	err = db.Order("login_time desc").Offset((pageIndex - 1) * pageSize).Limit(pageSize).Find(&o).Error
	return
}

// 获取最近三十天的登录日志

func (r LoginLogRepository) GetLastDaysLoginLog(day int) (o []dto.LoginLogForPage, err error) {
	err = r.DB.Table("login_logs").Select("id,login_time,client_ip,source,login_type,login_result,username,nickname,description").Order("login_time desc").Limit(day).Find(&o).Error
	return
}

func (r LoginLogRepository) GetLoginDetailsStatist(start, end string) (o []model.LoginDetailsInfo, err error) {
	db := r.DB.Table("login_logs")
	err = db.Where("login_time BETWEEN ? AND ?", start, end).Find(&o).Error
	return
}

func (r LoginLogRepository) GetUserCountThisWeek() (o []dto.DayCount, err error) {
	sql := "select date_format(login_time,'%w') as day,count(username) as cnt from login_logs where YEARWEEK(date_format(login_time,'%Y-%m-%d'),1) = YEARWEEK(now(),1)  group by day order by day"
	var temp []dto.DayCountTemp
	err = r.DB.Raw(sql).Find(&temp).Error
	if len(temp) > 0 {
		if temp[0].Day == 0 {
			temp[0].Day = 7
			var dayCount = temp[0]
			for i := 0; i < len(temp)-1; i++ {
				temp[i] = temp[i+1]
			}
			temp[len(temp)-1] = dayCount
		}
	}
	for _, v := range temp {
		o = append(o, dto.DayCount{Day: strconv.Itoa(v.Day), Cnt: v.Cnt})
	}
	return
}

func (r LoginLogRepository) GetUserCountThisMonth() (o []dto.DayCount, err error) {
	sql := "select date_format(login_time,'%d') as day,count(username) as cnt from login_logs where DATE_FORMAT(login_time,'%Y%m') = DATE_FORMAT(CURDATE(),'%Y%m')  group by day order by day"
	err = r.DB.Raw(sql).Find(&o).Error
	return
}

func (r LoginLogRepository) GetUserCountThisYear() (o []dto.DayCount, err error) {
	sql := "select date_format(login_time,'%m') as day,count(username) as cnt from login_logs where DATE_FORMAT(login_time,'%y')=DATE_FORMAT(CURDATE( ),'%y') group by day order by day"
	err = r.DB.Raw(sql).Find(&o).Error
	return
}

func (r LoginLogRepository) GetDeviceCountThisWeek() (o []dto.DayCount, err error) {
	sql := "select date_format(login_time,'%w') as day,count(asset_name) as cnt from operation_and_maintenance_log where YEARWEEK(date_format(login_time,'%Y-%m-%d'),1) = YEARWEEK(now(),1)  group by day order by day"
	var temp []dto.DayCountTemp
	err = r.DB.Raw(sql).Find(&temp).Error
	if len(temp) > 0 {
		if temp[0].Day == 0 {
			temp[0].Day = 7
			var dayCount = temp[0]
			for i := 0; i < len(temp)-1; i++ {
				temp[i] = temp[i+1]
			}
			temp[len(temp)-1] = dayCount
		}
	}
	for _, v := range temp {
		o = append(o, dto.DayCount{Day: strconv.Itoa(v.Day), Cnt: v.Cnt})
	}
	return
}

func (r LoginLogRepository) GetDeviceCountThisMonth() (o []dto.DayCount, err error) {
	sql := "select date_format(login_time,'%d') as day,count(asset_name) as cnt from operation_and_maintenance_log where DATE_FORMAT(login_time,'%Y%m') = DATE_FORMAT(CURDATE(),'%Y%m')  group by day order by day"
	err = r.DB.Raw(sql).Find(&o).Error
	return
}

func (r LoginLogRepository) GetDeviceCountThisYear() (o []dto.DayCount, err error) {
	sql := "select date_format(login_time,'%m') as day,count(asset_name) as cnt from operation_and_maintenance_log where DATE_FORMAT(login_time,'%y')=DATE_FORMAT(CURDATE( ),'%y') group by day order by day"
	err = r.DB.Raw(sql).Find(&o).Error
	return
}

func (r LoginLogRepository) GetUserTop5CountThisWeek() (o []dto.VisitsUserTop, err error) {
	sql := "select distinct count(*) as cnt,username,nickname from login_logs where YEARWEEK(date_format(login_time,'%Y-%m-%d'),1) = YEARWEEK(now(),1)  group by username,nickname order by cnt desc limit 5"
	err = r.DB.Raw(sql).Find(&o).Error
	return
}

func (r LoginLogRepository) GetUserTop5CountThisMonth() (o []dto.VisitsUserTop, err error) {
	sql := "select distinct count(*) as cnt,username,nickname from login_logs where DATE_FORMAT(login_time,'%Y%m') = DATE_FORMAT(CURDATE(),'%Y%m')  group by username,nickname order by cnt desc limit 5"
	err = r.DB.Raw(sql).Find(&o).Error
	return
}

func (r LoginLogRepository) GetUserTop5CountThisYear() (o []dto.VisitsUserTop, err error) {
	sql := "select distinct count(*) as cnt,username,nickname from login_logs where DATE_FORMAT(login_time,'%y')=DATE_FORMAT(CURDATE( ),'%y')  group by username,nickname order by cnt desc limit 5"
	err = r.DB.Raw(sql).Find(&o).Error
	return
}

func (r LoginLogRepository) GetDeviceTop5CountThisWeek() (o []dto.VisitsDeviceTop, err error) {
	sql := "select distinct count(*) as cnt,asset_name as device_name,asset_ip as device_ip from operation_and_maintenance_log where YEARWEEK(date_format(login_time,'%Y-%m-%d'),1) = YEARWEEK(now(),1)  group by asset_name,asset_ip order by cnt desc limit 5"
	err = r.DB.Raw(sql).Find(&o).Error
	return
}

func (r LoginLogRepository) GetDeviceTop5CountThisMonth() (o []dto.VisitsDeviceTop, err error) {
	sql := "select distinct count(*) as cnt,asset_name as device_name,asset_ip as device_ip from operation_and_maintenance_log where DATE_FORMAT(login_time,'%Y%m') = DATE_FORMAT(CURDATE(),'%Y%m')  group by asset_name,asset_ip order by cnt desc limit 5"
	err = r.DB.Raw(sql).Find(&o).Error
	return
}

func (r LoginLogRepository) GetDeviceTop5CountThisYear() (o []dto.VisitsDeviceTop, err error) {
	sql := "select distinct count(*) as cnt,asset_name as device_name,asset_ip as device_ip from operation_and_maintenance_log where DATE_FORMAT(login_time,'%y')=DATE_FORMAT(CURDATE( ),'%y')  group by asset_name,asset_ip order by cnt desc limit 5"
	err = r.DB.Raw(sql).Find(&o).Error
	return
}
