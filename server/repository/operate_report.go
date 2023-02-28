package repository

import (
	"context"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"
)

type OperateReportRepository struct {
	baseRepository
}

// GetOperateLoginLogByHour 查询运维日志并按小时分组
func (r *OperateReportRepository) GetOperateLoginLogByHour(c context.Context, startTime, endTime string) ([]dto.AssetAccess, error) {
	var assetAccess []dto.AssetAccess
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("count(distinct asset_name) as asset_num, count(distinct username) as user_num,DATE_FORMAT(login_time, '%Y-%m-%d %H') as time").Where("DATE_FORMAT(login_time, '%Y-%m-%d') between ? and ?", startTime, endTime).Group("time").Scan(&assetAccess).Error
	return assetAccess, err
}

// GetOperateLoginLogAssetDetailByHour 按小时查询设备的运维日志
func (r *OperateReportRepository) GetOperateLoginLogAssetDetailByHour(c context.Context, t string) ([]dto.AssetAccessAssetDetail, error) {
	var assetAccessAssetDetail []dto.AssetAccessAssetDetail
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("DATE_FORMAT(login_time,'%Y-%m-%d %H:%i:%s') as start_at, DATE_FORMAT(logout_time,'%Y-%m-%d %H:%i:%s') as end_at, asset_name as asset, asset_ip as asset_ip, passport as passport, protocol as protocol, ip as client_ip").Where("DATE_FORMAT(login_time, '%Y-%m-%d %H') between ? and ?", t, utils.AddHourTime(t)).Scan(&assetAccessAssetDetail).Error
	return assetAccessAssetDetail, err
}

// GetOperateLoginLogUserDetailByHour 按小时查询用户的运维日志
func (r *OperateReportRepository) GetOperateLoginLogUserDetailByHour(c context.Context, t string) ([]dto.AssetAccessUserDetail, error) {
	var assetAccessUserDetail []dto.AssetAccessUserDetail
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("DATE_FORMAT(login_time,'%Y-%m-%d %H:%i:%s') as start_at, DATE_FORMAT(logout_time,'%Y-%m-%d %H:%i:%s') as end_at, username as username, nickname as nickname, ip as client_ip").Where("DATE_FORMAT(login_time, '%Y-%m-%d %H') between ? and ?", t, utils.AddHourTime(t)).Scan(&assetAccessUserDetail).Error
	return assetAccessUserDetail, err
}

// GetOperateLoginLogExportByHour 查询运维日志并按小时分组用于导出
func (r *OperateReportRepository) GetOperateLoginLogExportByHour(c context.Context, startTime, endTime string) ([]dto.AssetAccess, []dto.AssetAccessExport, error) {
	var assetAccess []dto.AssetAccess
	var assetAccessExport []dto.AssetAccessExport
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("count(distinct asset_name) as asset_num, count(distinct username) as user_num,DATE_FORMAT(login_time, '%Y-%m-%d %H') as time").Where("DATE_FORMAT(login_time, '%Y-%m-%d') between ? and ?", startTime, endTime).Group("time").Scan(&assetAccess).Error
	err = r.GetDB(c).Table("operation_and_maintenance_log").Select("DATE_FORMAT(login_time,'%Y-%m-%d %H:%i:%s') as start_at, DATE_FORMAT(logout_time,'%Y-%m-%d %H:%i:%s') as end_at, asset_name as asset, asset_ip as asset_ip, passport as passport, protocol as protocol, ip as client_ip, username as username, nickname as nickname").Where("DATE_FORMAT(login_time, '%Y-%m-%d') between ? and ?", startTime, endTime).Scan(&assetAccessExport).Error

	//for i := 0; i < len(assetAccess); i++ {
	//	assetAccess[i].Time = utils.StrToTime(assetAccess[i].Time)
	//}
	//for i := 0; i < len(assetAccessExport); i++ {
	//	assetAccessExport[i].StartAt = utils.StrToTime(assetAccessExport[i].StartAt)
	//	assetAccessExport[i].EndAt = utils.StrToTime(assetAccessExport[i].EndAt)
	//}
	return assetAccess, assetAccessExport, err
}

// GetOperateLoginLogByDay 查询运维日志并按天分组
func (r *OperateReportRepository) GetOperateLoginLogByDay(c context.Context, startTime, endTime string) ([]dto.AssetAccess, error) {
	var assetAccess []dto.AssetAccess
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("count(distinct asset_name) as asset_num, count(distinct username) as user_num,DATE_FORMAT(login_time, '%Y-%m-%d') as time").Where("DATE_FORMAT(login_time, '%Y-%m-%d') between ? and ?", startTime, endTime).Group("time").Scan(&assetAccess).Error
	return assetAccess, err
}

// GetOperateLoginLogAssetDetailByDay 按天查询设备的运维日志
func (r *OperateReportRepository) GetOperateLoginLogAssetDetailByDay(c context.Context, t string) ([]dto.AssetAccessAssetDetail, error) {
	var assetAccessAssetDetail []dto.AssetAccessAssetDetail
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("DATE_FORMAT(login_time,'%Y-%m-%d %H:%i:%s') as start_at, DATE_FORMAT(logout_time,'%Y-%m-%d %H:%i:%s') as end_at, asset_name as asset, asset_ip as asset_ip, passport as passport, protocol as protocol, ip as client_ip").Where("DATE_FORMAT(login_time, '%Y-%m-%d') between ? and ?", t, utils.AddDayTime(t)).Scan(&assetAccessAssetDetail).Error
	return assetAccessAssetDetail, err
}

// GetOperateLoginLogUserDetailByDay 按天查询用户的运维日志
func (r *OperateReportRepository) GetOperateLoginLogUserDetailByDay(c context.Context, t string) ([]dto.AssetAccessUserDetail, error) {
	var assetAccessUserDetail []dto.AssetAccessUserDetail
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("DATE_FORMAT(login_time,'%Y-%m-%d %H:%i:%s') as start_at, DATE_FORMAT(logout_time,'%Y-%m-%d %H:%i:%s') as end_at, username as username, nickname as nickname, ip as client_ip").Where("DATE_FORMAT(login_time, '%Y-%m-%d') between ? and ?", t, utils.AddDayTime(t)).Scan(&assetAccessUserDetail).Error
	return assetAccessUserDetail, err
}

// GetOperateLoginLogExportByDay 查询运维日志并按天分组用于导出
func (r *OperateReportRepository) GetOperateLoginLogExportByDay(c context.Context, startTime, endTime string) ([]dto.AssetAccess, []dto.AssetAccessExport, error) {
	var assetAccess []dto.AssetAccess
	var assetAccessExport []dto.AssetAccessExport
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("count(distinct asset_name) as asset_num, count(distinct username) as user_num,DATE_FORMAT(login_time, '%Y-%m-%d') as time").Where("DATE_FORMAT(login_time, '%Y-%m-%d') between ? and ?", startTime, endTime).Group("time").Scan(&assetAccess).Error
	err = r.GetDB(c).Table("operation_and_maintenance_log").Select("DATE_FORMAT(login_time,'%Y-%m-%d %H:%i:%s') as start_at, DATE_FORMAT(logout_time,'%Y-%m-%d %H:%i:%s') as end_at, asset_name as asset, asset_ip as asset_ip, passport as passport, protocol as protocol, ip as client_ip, username as username, nickname as nickname").Where("DATE_FORMAT(login_time, '%Y-%m-%d') between ? and ?", startTime, endTime).Scan(&assetAccessExport).Error
	return assetAccess, assetAccessExport, err
}

// GetOperateLoginLogByMonth 查询运维日志并按月分组
func (r *OperateReportRepository) GetOperateLoginLogByMonth(c context.Context, startTime, endTime string) ([]dto.AssetAccess, error) {
	var assetAccess []dto.AssetAccess
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("count(distinct asset_name) as asset_num, count(distinct username) as user_num,DATE_FORMAT(login_time, '%Y-%m') as time").Where("DATE_FORMAT(login_time, '%Y-%m-%d') between ? and ?", startTime, endTime).Group("time").Scan(&assetAccess).Error

	return assetAccess, err
}

// GetOperateLoginLogAssetDetailByMonth 按月查询设备的运维日志
func (r *OperateReportRepository) GetOperateLoginLogAssetDetailByMonth(c context.Context, t string) ([]dto.AssetAccessAssetDetail, error) {
	var assetAccessAssetDetail []dto.AssetAccessAssetDetail
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("DATE_FORMAT(login_time,'%Y-%m-%d %H:%i:%s') as start_at, DATE_FORMAT(logout_time,'%Y-%m-%d %H:%i:%s') as end_at, asset_name as asset, asset_ip as asset_ip, passport as passport, protocol as protocol, ip as client_ip").Where("DATE_FORMAT(login_time, '%Y-%m') between ? and ?", t, utils.AddMonthTime(t)).Scan(&assetAccessAssetDetail).Error
	return assetAccessAssetDetail, err
}

// GetOperateLoginLogUserDetailByMonth 按月查询用户的运维日志
func (r *OperateReportRepository) GetOperateLoginLogUserDetailByMonth(c context.Context, t string) ([]dto.AssetAccessUserDetail, error) {
	var assetAccessUserDetail []dto.AssetAccessUserDetail
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("DATE_FORMAT(login_time,'%Y-%m-%d %H:%i:%s') as start_at, DATE_FORMAT(logout_time,'%Y-%m-%d %H:%i:%s') as end_at, username as username, nickname as nickname, ip as client_ip").Where("DATE_FORMAT(login_time, '%Y-%m') between ? and ?", t, utils.AddMonthTime(t)).Scan(&assetAccessUserDetail).Error
	return assetAccessUserDetail, err
}

// GetOperateLoginLogExportByMonth 查询运维日志并按月分组用于导出
func (r *OperateReportRepository) GetOperateLoginLogExportByMonth(c context.Context, startTime, endTime string) ([]dto.AssetAccess, []dto.AssetAccessExport, error) {
	var assetAccess []dto.AssetAccess
	var assetAccessExport []dto.AssetAccessExport
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("count(distinct asset_name) as asset_num, count(distinct username) as user_num,DATE_FORMAT(login_time, '%Y-%m') as time").Where("DATE_FORMAT(login_time, '%Y-%m-%d') between ? and ?", startTime, endTime).Group("time").Scan(&assetAccess).Error
	err = r.GetDB(c).Table("operation_and_maintenance_log").Select("DATE_FORMAT(login_time,'%Y-%m-%d %H:%i:%s') as start_at, DATE_FORMAT(logout_time,'%Y-%m-%d %H:%i:%s') as end_at, asset_ip as asset_ip, passport as passport, protocol as protocol, ip as client_ip, username as username, nickname as nickname").Where("DATE_FORMAT(login_time, '%Y-%m-%d') between ? and ?", startTime, endTime).Scan(&assetAccessExport).Error
	return assetAccess, assetAccessExport, err
}

// GetOperateLoginLogByWeek 查询运维日志并按周分组
func (r *OperateReportRepository) GetOperateLoginLogByWeek(c context.Context, startTime, endTime string) ([]dto.AssetAccess, error) {
	var assetAccess []dto.AssetAccess
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("count(distinct asset_name) as asset_num, count(distinct username) as user_num,DATE_FORMAT(login_time, '%Y-w%u') as time").Where("login_time >= ? and login_time <= ?", startTime, endTime).Group("time").Scan(&assetAccess).Error

	return assetAccess, err
}

// GetOperateLoginLogAssetDetailByWeek 按周查询设备的运维日志
func (r *OperateReportRepository) GetOperateLoginLogAssetDetailByWeek(c context.Context, t string) ([]dto.AssetAccessAssetDetail, error) {
	var assetAccessAssetDetail []dto.AssetAccessAssetDetail
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("DATE_FORMAT(login_time,'%Y-%m-%d %H:%i:%s') as start_at, DATE_FORMAT(logout_time,'%Y-%m-%d %H:%i:%s') as end_at, asset_name as asset, asset_ip as asset_ip, passport as passport, protocol as protocol, ip as client_ip").Where("DATE_FORMAT(login_time, '%Y-w%u') = ?", t).Scan(&assetAccessAssetDetail).Error
	return assetAccessAssetDetail, err
}

// GetOperateLoginLogUserDetailByWeek 按周查询用户的运维日志
func (r *OperateReportRepository) GetOperateLoginLogUserDetailByWeek(c context.Context, t string) ([]dto.AssetAccessUserDetail, error) {
	var assetAccessUserDetail []dto.AssetAccessUserDetail
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("DATE_FORMAT(login_time,'%Y-%m-%d %H:%i:%s') as start_at, DATE_FORMAT(logout_time,'%Y-%m-%d %H:%i:%s') as end_at, username as username, nickname as nickname, ip as client_ip").Where("DATE_FORMAT(login_time, '%Y-w%u') = ?", t).Scan(&assetAccessUserDetail).Error
	return assetAccessUserDetail, err
}

// GetOperateLoginLogExportByWeek 查询运维日志并按周分组用于导出
func (r *OperateReportRepository) GetOperateLoginLogExportByWeek(c context.Context, startTime, endTime string) ([]dto.AssetAccess, []dto.AssetAccessExport, error) {
	var assetAccess []dto.AssetAccess
	var assetAccessExport []dto.AssetAccessExport
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("count(distinct asset_name) as asset_num, count(distinct username) as user_num,DATE_FORMAT(login_time, '%Y-w%u') as time").Where("DATE_FORMAT(login_time, '%Y-%m-%d') between ? and ?", startTime, endTime).Group("time").Scan(&assetAccess).Error
	err = r.GetDB(c).Table("operation_and_maintenance_log").Select("DATE_FORMAT(login_time,'%Y-%m-%d %H:%i:%s') as start_at, DATE_FORMAT(logout_time,'%Y-%m-%d %H:%i:%s') as end_at, asset_name as asset, asset_ip as asset_ip, passport as passport, protocol as protocol, ip as client_ip, username as username, nickname as nickname").Where("DATE_FORMAT(login_time, '%Y-%m-%d') between ? and ?", startTime, endTime).Scan(&assetAccessExport).Error
	return assetAccess, assetAccessExport, err
}

// GetOperateCommandCount 查询指令统计表
func (r *OperateReportRepository) GetOperateCommandCount(c context.Context, startTime, endTime string) ([]dto.CommandStatistics, error) {
	var command []dto.CommandStatistics
	db := r.GetDB(c).Table("command_records").Where("DATE_FORMAT(created, '%Y-%m-%d') between ? and ?", startTime, endTime)
	err := db.Select("count(*) as cnt, content as command").Group("command").Order("cnt desc").Scan(&command).Error
	return command, err
}

// GetOperateCommandDetails 查询指令统计详情
func (r *OperateReportRepository) GetOperateCommandDetails(c context.Context, startTime, endTime, command string) (commandRecord []model.CommandRecord, err error) {
	db := r.GetDB(c).Table("command_records").Where("DATE_FORMAT(created, '%Y-%m-%d') between ? and ? ", startTime, endTime)
	if command != "" {
		err = db.Where("content = ?", command).Find(&commandRecord).Order("created desc").Error
	} else {
		err = db.Find(&commandRecord).Order("created desc").Error
	}
	return commandRecord, err
}

// GetOperateSessionLogAsset 按设备查询会话时长
func (r *OperateReportRepository) GetOperateSessionLogAsset(c context.Context, start, end string) ([]dto.AssetSession, error) {
	var assetSession []dto.AssetSession
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("asset_name,asset_ip,CONCAT(asset_name,'(',asset_ip,')') as name, SUM(TIMESTAMPDIFF(SECOND, login_time, logout_time)) as time").Where("DATE_FORMAT(login_time, '%Y-%m-%d') between ? and ?", start, end).Group("asset_name, asset_ip, CONCAT(asset_name,'(',asset_ip,')')").Scan(&assetSession).Error
	return assetSession, err
}

// GetOperateSessionLogUser 按用户查询会话时长
func (r *OperateReportRepository) GetOperateSessionLogUser(c context.Context, start, end string) ([]dto.UserSession, error) {
	var userSession []dto.UserSession
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("username as user_name,nickname as nick_name,CONCAT(username,'(',nickname,')') as name, SUM(TIMESTAMPDIFF(SECOND, login_time, logout_time)) as time").Where("DATE_FORMAT(login_time, '%Y-%m-%d') between ? and ?", start, end).Group("username, nickname, CONCAT(username,'(',nickname,')')").Scan(&userSession).Error
	return userSession, err
}

// GetOperateSessionLogAssetDetail 通过设备及开始结束时间查询设备会话记录详情
func (r *OperateReportRepository) GetOperateSessionLogAssetDetail(c context.Context, asset, start, end string) ([]dto.AssetAccessAssetDetail, error) {
	var assetSessionDetail []dto.AssetAccessAssetDetail
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("DATE_FORMAT(login_time,'%Y-%m-%d %H:%i:%s') as start_at, DATE_FORMAT(logout_time,'%Y-%m-%d %H:%i:%s') as end_at, asset_name as asset, asset_ip as asset_ip, passport as passport, protocol as protocol, ip as client_ip").Where("DATE_FORMAT(login_time, '%Y-%m-%d') between ? and ? and asset_name = ?", start, end, asset).Scan(&assetSessionDetail).Error
	return assetSessionDetail, err
}

// GetOperateSessionLogUserDetail 通过用户及开始结束时间查询用户会话记录详情
func (r *OperateReportRepository) GetOperateSessionLogUserDetail(c context.Context, user, start, end string) ([]dto.AssetAccessUserDetail, error) {
	var userSessionDetail []dto.AssetAccessUserDetail
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("DATE_FORMAT(login_time,'%Y-%m-%d %H:%i:%s') as start_at, DATE_FORMAT(logout_time,'%Y-%m-%d %H:%i:%s') as end_at, ip as client_ip,username,nickname").Where("DATE_FORMAT(login_time, '%Y-%m-%d') between ? and ? and username = ?", start, end, user).Scan(&userSessionDetail).Error
	return userSessionDetail, err
}

// GetOperateSessionLogExport 获取运维时长报表导出数据
func (r *OperateReportRepository) GetOperateSessionLogExport(c context.Context, startTime, endTime string) ([]dto.SessionForExport, error) {
	var sessionForExport []dto.SessionForExport
	err := r.GetDB(c).Table("operation_and_maintenance_log").Select("DATE_FORMAT(login_time,'%Y-%m-%d %H:%i:%s') as start_time, DATE_FORMAT(logout_time,'%Y-%m-%d %H:%i:%s') as end_time, asset_name, asset_ip, passport, protocol, ip as client_ip, username, nickname").Where("DATE_FORMAT(login_time, '%Y-%m-%d') between ? and ?", startTime, endTime).Scan(&sessionForExport).Error
	return sessionForExport, err
}
