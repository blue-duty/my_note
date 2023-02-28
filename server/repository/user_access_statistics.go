package repository

import (
	"context"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"
)

type UserAccessStatisticsRepository struct {
	baseRepository
}

func (u *UserAccessStatisticsRepository) GetUserAccessStatisticsByDay(c context.Context, start, end string) (userAccessStatistics []dto.UserProtocolAccessCountStatistics, err error) {
	// 查询用户每天不同用户的不同协议的访问次数
	err = u.GetDB(c).Table("protocol_access_log").Select("DATE_FORMAT(daytime,'%Y-%m-%d') as daytime,protocol_access_log.username as username,protocol_access_log.nickname as nickname,sum(protocol_access_log.protocol = 'ssh') as ssh,sum(protocol_access_log.protocol = 'rdp') as rdp,sum(protocol_access_log.protocol = 'telnet') as telnet,sum(protocol_access_log.protocol = 'vnc') as vnc,sum(protocol_access_log.protocol = 'app') as app,sum(protocol_access_log.protocol = 'tcp') as tcp,sum(protocol_access_log.protocol = 'tcp') as tcp,DATE_FORMAT(protocol_access_log.daytime,'%Y-%m-%d') as daytime,sum(protocol_access_log.protocol = 'ssh' or protocol_access_log.protocol = 'rdp' or protocol_access_log.protocol = 'telnet' or protocol_access_log.protocol = 'vnc' or protocol_access_log.protocol = 'app' or protocol_access_log.protocol = 'tcp') as total").Where("DATE_FORMAT(daytime,'%Y-%m-%d') BETWEEN ? AND ?", start, end).Group("DATE_FORMAT(daytime,'%Y-%m-%d'),username,nickname").Order("daytime desc").Scan(&userAccessStatistics).Error
	return
}

// GetUserAccessStatistics 获取用户访问统计
func (u *UserAccessStatisticsRepository) GetUserAccessStatistics(c context.Context, start, end string) (userAccessStatistics []dto.UserAccessStatistics, err error) {
	err = u.GetDB(c).Table("protocol_access_log").Select("protocol_access_log.username,protocol_access_log.nickname,protocol_access_log.protocol,DATE_FORMAT(protocol_access_log.daytime,'%Y-%m-%d %H:%i:%s') as daytime,protocol_access_log.client_ip,protocol_access_log.result,protocol_access_log.info").Where("DATE_FORMAT(daytime,'%Y-%m-%d') BETWEEN ? AND ?", start, end).Order("daytime desc").Scan(&userAccessStatistics).Error
	return
}

func (u *UserAccessStatisticsRepository) GetUserAccessStatisticsTotal(c context.Context, start, end string) (userAccessStatisticsTotal []dto.UserAccessTotal, err error) {
	// 获取用户在一段时间内的访问总数
	err = u.GetDB(c).Table("protocol_access_log").Select("protocol_access_log.username as username,sum(protocol_access_log.protocol = 'ssh' or protocol_access_log.protocol = 'rdp' or protocol_access_log.protocol = 'telnet' or protocol_access_log.protocol = 'vnc' or protocol_access_log.protocol = 'app' or protocol_access_log.protocol = 'tcp') as totals").Where("DATE_FORMAT(daytime,'%Y-%m-%d') BETWEEN ? AND ?", start, end).Group("username").Order("totals desc").Scan(&userAccessStatisticsTotal).Error
	return
}

// GetUserAccessStatisticsByProtocol 根据协议获取用户访问数据
func (u *UserAccessStatisticsRepository) GetUserAccessStatisticsByProtocol(c context.Context, time, protocol, username string) (userAccessStatistics []dto.UserAccessStatistics, err error) {
	if protocol == "all" {
		err = u.GetDB(c).Table("protocol_access_log").Select("if(user_new.username is null,protocol_access_log.username,user_new.username) as username,if(user_new.nickname is null,protocol_access_log.nickname,user_new.nickname) as nickname,protocol_access_log.protocol,DATE_FORMAT(protocol_access_log.daytime,'%Y-%m-%d %H:%i:%s') as daytime,protocol_access_log.client_ip,protocol_access_log.result,protocol_access_log.info").Joins("left join user_new on protocol_access_log.user_id = user_new.id").Where("DATE_FORMAT(daytime,'%Y-%m-%d') = ? and protocol_access_log.protocol in (?)", time, []string{"ssh", "telnet", "rdp", "vnc", "app", "tcp"}).Order("daytime desc").Scan(&userAccessStatistics).Error
	} else {
		err = u.GetDB(c).Table("protocol_access_log").Select("if(user_new.username is null,protocol_access_log.username,user_new.username) as username,if(user_new.nickname is null,protocol_access_log.nickname,user_new.nickname) as nickname,protocol_access_log.protocol,DATE_FORMAT(protocol_access_log.daytime,'%Y-%m-%d %H:%i:%s') as daytime,protocol_access_log.client_ip,protocol_access_log.result,protocol_access_log.info").Joins("left join user_new on protocol_access_log.user_id = user_new.id").Where("DATE_FORMAT(daytime,'%Y-%m-%d') = ? AND protocol_access_log.protocol = ?", time, protocol).Order("daytime desc").Scan(&userAccessStatistics).Error
	}

	for i := 0; i < len(userAccessStatistics); i++ {
		if userAccessStatistics[i].Username != username {
			userAccessStatistics = append(userAccessStatistics[:i], userAccessStatistics[i+1:]...)
			i--
		}
	}

	return
}

// AddUserAccessStatisticsByProtocol 协议访问数据新建
func (u *UserAccessStatisticsRepository) AddUserAccessStatisticsByProtocol(c context.Context, user *model.UserNew, protocol, id, ip, result, info string) (err error) {
	// 判断今天是否有数据
	err = u.GetDB(c).Create(&model.ProtocolAccessLog{
		Daytime:   utils.NowJsonTime(),
		Protocol:  protocol,
		ClientIp:  ip,
		UserId:    user.ID,
		Username:  user.Username,
		Nickname:  user.Nickname,
		SessionId: id,
		Result:    result,
		Info:      info,
	}).Error
	return err
}

// UpdateUserAccessUpdateFailInfo 更新失败信息
func (u *UserAccessStatisticsRepository) UpdateUserAccessUpdateFailInfo(c context.Context, id, info string) (err error) {
	err = u.GetDB(c).Model(&model.ProtocolAccessLog{}).Where("session_id = ?", id).Update("info", info).Error
	return err
}

// UpdateUserAccessUpdateSuccessInfo 更新成功信息
func (u *UserAccessStatisticsRepository) UpdateUserAccessUpdateSuccessInfo(c context.Context, id, info string) (err error) {
	err = u.GetDB(c).Model(&model.ProtocolAccessLog{}).Where("session_id = ?", id).UpdateColumns(map[string]interface{}{"info": info, "result": "成功"}).Error
	return err
}

// GetLoginStatistics 获取登陆尝试统计
func (u *UserAccessStatisticsRepository) GetLoginStatistics(c context.Context, start, end string) (userAccessStatistics []dto.LoginAttemptStatistics, err error) {
	err = u.GetDB(c).Table("protocol_access_log").Select("DATE_FORMAT(daytime,'%Y-%m-%d') as daytime,SUM(IF(result = '成功', 1, 0)) AS success,SUM(IF(result = '失败', 1, 0)) AS failure,COUNT(distinct user_id) AS user_num,COUNT(distinct client_ip) AS ip_num,COUNT(result) AS totals").Where("DATE_FORMAT(daytime,'%Y-%m-%d') >= ? AND DATE_FORMAT(daytime,'%Y-%m-%d') <= ? AND protocol = 'tcp'", start, end).Group("DATE_FORMAT(daytime,'%Y-%m-%d')").Order("daytime desc").Scan(&userAccessStatistics).Error
	return
}

// GetLoginStatisticsDetail 获取登陆尝试统计详细数据
func (u *UserAccessStatisticsRepository) GetLoginStatisticsDetail(c context.Context, start, end string) (userAccessStatistics []dto.LoginAttemptDetail, err error) {
	err = u.GetDB(c).Table("protocol_access_log").Select("DATE_FORMAT(daytime,'%Y-%m-%d') as daytime,client_ip,username,nickname,result,info").Where("DATE_FORMAT(daytime,'%Y-%m-%d') >= ? AND DATE_FORMAT(daytime,'%Y-%m-%d') <= ? AND protocol = 'tcp'", start, end).Order("daytime desc").Scan(&userAccessStatistics).Error
	return
}

// GetUserLoginStatisticsByTime 通过时间和查询类型获取用户登陆尝试数据
func (u *UserAccessStatisticsRepository) GetUserLoginStatisticsByTime(c context.Context, time, queryType string) (userAccessStatistics interface{}, err error) {
	switch queryType {
	case "user":
		var uid []string
		// 查询用户id并去重
		err = u.GetDB(c).Table("protocol_access_log").Select("distinct user_id").Where("DATE_FORMAT(daytime,'%Y-%m-%d') = ? AND protocol = 'tcp'", time).Scan(&uid).Error
		if err != nil {
			return nil, err
		}
		var user []dto.LoginAttemptUserDetail
		err = u.GetDB(c).Table("user_new").Select("username,nickname,role_name as role,department_name as department").Where("id in (?)", uid).Scan(&user).Error
		if err != nil {
			return nil, err
		}
		userAccessStatistics = user

	case "success":
		var success []dto.LoginAttemptDetail
		err = u.GetDB(c).Table("protocol_access_log").Select("user_new.username,user_new.nickname,DATE_FORMAT(protocol_access_log.daytime,'%Y-%m-%d %H:%i:%s') as daytime,protocol_access_log.client_ip,protocol_access_log.result,protocol_access_log.info").Joins("left join user_new on protocol_access_log.user_id = user_new.id").Where("DATE_FORMAT(daytime,'%Y-%m-%d') = ? AND protocol = 'tcp' AND result = '成功'", time).Order("daytime desc").Scan(&success).Error
		if err != nil {
			return nil, err
		}
		userAccessStatistics = success

	case "failure":
		var failure []dto.LoginAttemptDetail
		err = u.GetDB(c).Table("protocol_access_log").Select("user_new.username,user_new.nickname,DATE_FORMAT(protocol_access_log.daytime,'%Y-%m-%d %H:%i:%s') as daytime,protocol_access_log.client_ip,protocol_access_log.result,protocol_access_log.info").Joins("left join user_new on protocol_access_log.user_id = user_new.id").Where("DATE_FORMAT(daytime,'%Y-%m-%d') = ? AND protocol = 'tcp' AND result = '失败'", time).Order("daytime desc").Scan(&failure).Error
		if err != nil {
			return nil, err
		}
		userAccessStatistics = failure

	case "all":
		var all []dto.LoginAttemptDetail
		err = u.GetDB(c).Table("protocol_access_log").Select("user_new.username,user_new.nickname,DATE_FORMAT(protocol_access_log.daytime,'%Y-%m-%d %H:%i:%s') as daytime,protocol_access_log.client_ip,protocol_access_log.result,protocol_access_log.info").Joins("left join user_new on protocol_access_log.user_id = user_new.id").Where("DATE_FORMAT(daytime,'%Y-%m-%d') = ? AND protocol = 'tcp'", time).Order("daytime desc").Scan(&all).Error
		if err != nil {
			return nil, err
		}
		userAccessStatistics = all
	default:
		userAccessStatistics = []dto.LoginAttemptDetail{}
	}
	return
}
