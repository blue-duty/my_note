package repository

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"
	"tkbastion/pkg/constant"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/dustin/go-humanize"
)

type SessionRepositoryNew struct {
	baseRepository
}

var SessionRepo = new(SessionRepositoryNew)

// Create 新建会话
func (s *SessionRepositoryNew) Create(c context.Context, session *model.Session) error {
	return s.GetDB(c).Create(session).Error
}

// Update 更新会话
func (s *SessionRepositoryNew) Update(c context.Context, session *model.Session, id string) error {
	return s.GetDB(c).Model(&model.Session{}).Where("id = ?", id).Updates(session).Error
}

// Delete 删除会话
func (s *SessionRepositoryNew) Delete(c context.Context, id string) error {
	return s.GetDB(c).Delete(&model.Session{}, "id = ?", id).Error
}

// GetAllSession 获取所有会话
func (s *SessionRepositoryNew) GetAllSession(c context.Context) (sessions []model.Session, err error) {
	err = s.GetDB(c).Find(&sessions).Error
	return
}

// GetById 根据id获取会话
func (s *SessionRepositoryNew) GetById(c context.Context, id string) (model.Session, error) {
	var session model.Session
	err := s.GetDB(c).Where("id = ?", id).First(&session).Error
	return session, err
}

func (s *SessionRepositoryNew) UpdateWindowSizeById(todo context.Context, width int, height int, id string) error {
	return s.GetDB(todo).Model(&model.Session{}).Where("id = ?", id).Update("width", width).Update("height", height).Error
}

// GetHistorySession 查询历史会话
func (s *SessionRepositoryNew) GetHistorySession(c context.Context, ss dto.SessionForSearch) (sp []dto.SessionForPage, err error) {
	var sessions []model.Session
	if len(ss.Command) != 0 {
		var sids []string
		err = s.GetDB(c).Model(&model.CommandRecord{}).Where("content like ?", "%"+ss.Command+"%").Pluck("session_id", &sids).Error
		if err != nil {
			return
		}
		if len(sids) == 0 {
			return
		}
		db := s.GetDB(c).Model(&model.Session{}).Where("department_id in (?) and id in (?) and disconnected_time is not null and status = ?", ss.DepartmentIds, sids, constant.Disconnected)
		err = db.Order("create_time desc").Find(&sessions).Error
		if err != nil {
			return
		}
	} else {
		db := s.GetDB(c).Model(&model.Session{}).Where("department_id in (?) and disconnected_time is not null and status = ?", ss.DepartmentIds, constant.Disconnected)
		if ss.Auto != "" {
			db = db.Where("asset_name like ?", "%"+ss.Auto+"%").Or("asset_ip like ?", "%"+ss.Auto+"%").Or("pass_port like ?", "%"+ss.Auto+"%").Or("protocol like ?", "%"+ss.Auto+"%").Or("client_ip like ?", "%"+ss.Auto+"%").Or("create_name like ?", "%"+ss.Auto+"%").Or("create_nick like ?", "%"+ss.Auto+"%")
		} else if ss.AssetName != "" {
			db = db.Where("asset_name like ?", "%"+ss.AssetName+"%")
		} else if ss.AssetIP != "" {
			db = db.Where("asset_ip like ?", "%"+ss.AssetIP+"%")
		} else if ss.Protocol != "" {
			db = db.Where("protocol like ?", "%"+ss.Protocol+"%")
		} else if ss.Passport != "" {
			db = db.Where("pass_port like ?", "%"+ss.Passport+"%")
		} else if ss.IP != "" {
			db = db.Where("client_ip like ?", "%"+ss.IP+"%")
		} else if ss.UserName != "" {
			db = db.Where("create_name like ?", "%"+ss.UserName+"%")
		} else if len(ss.OperateTime) != 0 {
			// operateTime在建立会话和结束会话之间
			db = db.Where("? between DATE_FORMAT(connected_time,'%Y-%m-%d') and DATE_FORMAT(disconnected_time,'%Y-%m-%d')", ss.OperateTime)
		}
		err = db.Order("create_time desc").Find(&sessions).Error
		if err != nil {
			return
		}
	}

	sp = make([]dto.SessionForPage, 0, len(sessions))
	for _, session := range sessions {
		sp = append(sp, dto.SessionForPage{
			ID:           session.ID,
			AssetName:    session.AssetName,
			AssetIP:      session.AssetIP,
			Protocol:     session.Protocol,
			Passport:     session.PassPort,
			IP:           session.ClientIP,
			UserName:     session.CreateName,
			UserNickname: session.CreateNick,
			Height:       session.Height,
			Width:        session.Width,
			IsRecordings: utils.FileExists(path.Join(session.Recording)),
			IsReply:      session.IsRead == 1,
			StartTime:    session.ConnectedTime.Format("2006-01-02 15:04:05"),
			EndTime:      session.DisconnectedTime.Format("2006-01-02 15:04:05"),
		})
	}
	return
}

// GetExportSession 获取导出会话的数据
func (s *SessionRepositoryNew) GetExportSession(c context.Context, department []int64) (se []dto.SessionForExport, err error) {
	err = s.GetDB(c).Table("session").Select("DATE_FORMAT(connected_time,'%Y-%m-%d %H:%i:%s') as start_time, DATE_FORMAT(disconnected_time,'%Y-%m-%d %H:%i:%s') as end_time, asset_name, asset_ip, pass_port as passport, protocol, client_ip, create_name as username, create_nick as nickname").Where("department_id in (?) and disconnected_time is not null and status = ?", department, constant.Disconnected).Order("create_time desc").Scan(&se).Error
	return
}

// GetSessionDetailById 获取会话详情
func (s *SessionRepositoryNew) GetSessionDetailById(c context.Context, id string) (sd dto.SessionDetail, err error) {
	var session model.Session
	err = s.GetDB(c).Where("id = ?", id).Find(&session).Error
	if err != nil {
		return
	}

	fmt.Println("session", session)

	user, err := userNewRepository.FindById(session.Creator)
	if err != nil {
		return
	}

	var size string
	if session.Protocol != constant.SSH {
		stat, err := os.Stat(path.Join(session.Recording, "recording"))
		if err != nil {
			size = "未查出"
		} else {
			size = humanize.Bytes(uint64(stat.Size()))
		}
	} else {
		stst, err := os.Stat(session.Recording)
		if err != nil {
			size = "未查出"
		} else {
			size = humanize.Bytes(uint64(stst.Size()))
		}
	}

	// 获取会话时长
	duration := session.DisconnectedTime.Time.Sub(session.ConnectedTime.Time)

	sd = dto.SessionDetail{
		AssetName:   session.AssetName,
		AssetIP:     session.AssetIP,
		Protocol:    session.Protocol,
		Passport:    session.PassPort,
		CilentIP:    session.ClientIP,
		UserName:    session.CreateName,
		NickName:    session.CreateNick,
		UserId:      user.ID,
		StartAndEnd: session.ConnectedTime.Format("2006-01-02 15:04:05") + " - " + session.DisconnectedTime.Format("2006-01-02 15:04:05"),
		// 会话时长
		SessionTime:       duration.String(),
		SessionSzie:       size,
		AuthenticationWay: user.AuthenticationWay,
	}
	return
}

func (s *SessionRepositoryNew) GetOnlineSession(todo context.Context, ss dto.SessionForSearch) (sp []dto.SessionOnline, err error) {
	db := s.GetDB(todo).Model(&model.Session{}).Where("department_id in (?) and status = ?", ss.DepartmentIds, constant.Connected)
	if ss.Auto != "" {
		db = db.Where("asset_name like ? or asset_ip like ? or client_ip like ? or create_name like ? or create_nick like ? or pass_port like ? or protocol like ? or department like ? or create_time like ? or connected_time like ? or disconnected_time like ?", "%"+ss.Auto+"%", "%"+ss.Auto+"%", "%"+ss.Auto+"%", "%"+ss.Auto+"%", "%"+ss.Auto+"%", "%"+ss.Auto+"%", "%"+ss.Auto+"%", "%"+ss.Auto+"%", "%"+ss.Auto+"%", "%"+ss.Auto+"%", "%"+ss.Auto+"%")
	} else if ss.AssetName != "" {
		db = db.Where("asset_name like ?", "%"+ss.AssetName+"%")
	} else if ss.AssetIP != "" {
		db = db.Where("asset_ip like ?", "%"+ss.AssetIP+"%")
	} else if ss.IP != "" {
		db = db.Where("client_ip like ?", "%"+ss.IP+"%")
	} else if ss.UserName != "" {
		db = db.Where("create_name like ?", "%"+ss.UserName+"%")
	}
	var sessions []model.Session
	err = db.Order("create_time desc").Find(&sessions).Error
	if err != nil {
		return
	}

	sp = make([]dto.SessionOnline, len(sessions))
	for i, session := range sessions {
		sp[i] = dto.SessionOnline{
			ID:           session.ID,
			AssetName:    session.AssetName,
			AssetIP:      session.AssetIP,
			Protocol:     session.Protocol,
			Passport:     session.PassPort,
			IP:           session.ClientIP,
			UserName:     session.CreateName,
			UserNickname: session.CreateNick,
			StartTIme:    session.ConnectedTime.Format("2006-01-02 15:04:05"),
			Width:        session.Width,
			Height:       session.Height,
		}
	}
	return
}

// GetSessionByStatus 通过状态获取会话
func (s *SessionRepositoryNew) GetSessionByStatus(c context.Context, status []string) (sessions []model.Session, err error) {
	err = s.GetDB(c).Where("status in (?)", status).Find(&sessions).Error
	return
}

func (s *SessionRepositoryNew) GetLastMonthDaysConnect(todo context.Context, day int) (o []model.Session, err error) {
	db := s.GetDB(todo).Table("session")
	err = db.Order("connected_time desc").Limit(day).Find(&o).Error
	return
}

func (s *SessionRepositoryNew) GetAccessMonthsCount(todo context.Context, protocol []string, count *dto.AccessCount) (err error) {
	nowTime := time.Now()
	lastMonthTime := nowTime.AddDate(0, 0, -30)
	err = s.GetDB(todo).Table("operation_and_maintenance_log").Where("protocol in (?)", protocol).Count(&count.TotalCount).Error
	if err != nil {
		return
	}
	err = s.GetDB(todo).Table("operation_and_maintenance_log").Where("protocol in (?) and login_time BETWEEN ? AND ?", protocol, lastMonthTime, nowTime).Count(&count.RecentMonthCount).Error
	if err != nil {
		return
	}
	return
}

// GetProtocolCountByDay 获取登录协议的数量
func (s *SessionRepositoryNew) GetProtocolCountByDay(todo context.Context, start, end, searchType string) (o []model.ProtocolCountByDay, err error) {
	sql := `SELECT distinct dateTemp.time AS daytime,IFNULL(rdp,0) AS rdp,IFNULL(ssh,0) AS ssh,IFNULL(vnc,0) AS vnc,IFNULL(telnet,0) AS telnet,IFNULL(ftp,0) AS ftp,IFNULL(sftp,0) AS sftp,IfNULL(application,0) AS app,IfNULL(tcp,0) AS tcp,couall.total AS total FROM ((SELECT DATE_FORMAT(login_time,'` + searchType + `') as time FROM login_logs ` +
		`WHERE DATE_FORMAT(login_time,'` + searchType + `') BETWEEN ? AND ?) UNION ALL (SELECT DATE_FORMAT(connected_time,'` + searchType + `') as time FROM session WHERE DATE_FORMAT(connected_time,'` + searchType + `') BETWEEN ? AND ?)) AS dateTemp` +
		` LEFT JOIN (SELECT time,count(rdp) AS rdp FROM (SELECT DATE_FORMAT(connected_time,'` + searchType + `') time,protocol AS rdp FROM session WHERE protocol='rdp' GROUP BY connected_time,protocol) AS a GROUP BY time) AS rdp ON rdp.time=dateTemp.time` +
		` LEFT JOIN (SELECT time,count(ssh) AS ssh FROM (SELECT DATE_FORMAT(connected_time,'` + searchType + `') time,protocol AS ssh FROM session WHERE protocol='ssh' GROUP BY connected_time,protocol) AS a GROUP BY time) AS ssh ON dateTemp.time=ssh.time ` +
		` LEFT JOIN (SELECT time,count(vnc) AS vnc FROM (SELECT DATE_FORMAT(connected_time,'` + searchType + `') time,protocol AS vnc FROM session WHERE protocol='vnc' GROUP BY connected_time,protocol) AS a GROUP BY time) AS vnc ON vnc.time=dateTemp.time` +
		` LEFT JOIN (SELECT time,count(telnet) AS telnet FROM (SELECT DATE_FORMAT(connected_time,'` + searchType + `') time,protocol AS telnet FROM session WHERE protocol='telnet' GROUP BY connected_time,protocol) AS a GROUP BY time) AS telnet ON telnet.time =dateTemp.time` +
		` LEFT JOIN (SELECT time,count(ftp) AS ftp FROM (SELECT DATE_FORMAT(connected_time,'` + searchType + `') time,protocol AS ftp FROM session WHERE protocol='ftp' GROUP BY connected_time,protocol) AS a GROUP BY time) AS ftp ON ftp.time =dateTemp.time` +
		` LEFT JOIN (SELECT time,count(sftp) AS sftp FROM (SELECT DATE_FORMAT(connected_time,'` + searchType + `') time,protocol AS sftp FROM session WHERE protocol='sftp' GROUP BY connected_time,protocol) AS a GROUP BY time) AS sftp ON sftp.time =dateTemp.time` +
		` LEFT JOIN (SELECT time,count(application) AS application FROM (SELECT DATE_FORMAT(connected_time,'` + searchType + `') time,protocol AS application FROM session WHERE protocol='应用' GROUP BY connected_time,protocol) AS a GROUP BY time) AS application ON application.time =dateTemp.time` +
		` LEFT JOIN (SELECT DATE_FORMAT(login_time,'` + searchType + `') time,count(*) AS tcp FROM login_logs GROUP BY time) as tcpp ON tcpp.time =dateTemp.time` +
		` LEFT JOIN (SELECT time,sum(cou) total FROM (SELECT DATE_FORMAT(connected_time,'` + searchType + `') time,count(connected_time) AS cou FROM session GROUP BY connected_time) AS a GROUP BY time) AS couall ON dateTemp.time=couall.time order by dateTemp.time desc`
	err = s.GetDB(todo).Raw(sql, start, end, start, end).Find(&o).Error
	return
}

func (s *SessionRepositoryNew) GetLoginDetails(todo context.Context, start, end, protocol string) (o []model.SessionDetailsInfo, err error) {
	db := s.GetDB(todo).Table("session")
	if protocol != "" {
		db = db.Where("protocol = ?", protocol)
	}
	err = db.Where("DATE_FORMAT(connected_time,'%Y-%m-%d %H:%i:%s') BETWEEN ? AND ?", start, end).Find(&o).Order("connected_time desc").Error
	return
}

// UpdateRead 标为已读
func (s *SessionRepositoryNew) UpdateRead(todo context.Context, id []string) (err error) {
	err = s.GetDB(todo).Model(&model.Session{}).Where("id in (?)", id).Update("is_read", 1).Error
	return
}

// UpdateUnRead 标为未阅
func (s *SessionRepositoryNew) UpdateUnRead(todo context.Context, id []string) (err error) {
	err = s.GetDB(todo).Model(&model.Session{}).Where("id in (?)", id).Update("is_read", 0).Error
	return
}

// UpdateAllRead 全部标为已阅
func (s *SessionRepositoryNew) UpdateAllRead(todo context.Context) (err error) {
	err = s.GetDB(todo).Model(&model.Session{}).Where("is_read = ?", 0).Update("is_read", 1).Error
	return
}

// UpdateVideoDownload 标为已下载
func (s *SessionRepositoryNew) UpdateVideoDownload(todo context.Context, id string, d int) (err error) {
	err = s.GetDB(todo).Model(&model.Session{}).Where("id = ?", id).Update("is_download_record", d).Error
	return
}

// GetSSHOnlineCount 查询SSH协议在线数
func (s *SessionRepositoryNew) GetSSHOnlineCount(c context.Context) (online int64, err error) {
	err = s.GetDB(c).Model(&model.Session{}).Where("protocol = ? and status = ?", constant.SSH, constant.Connected).Count(&online).Error
	return
}

// GetRDPOnlineCount 查询RDP协议在线数
func (s *SessionRepositoryNew) GetRDPOnlineCount(c context.Context) (online int64, err error) {
	err = s.GetDB(c).Model(&model.Session{}).Where("protocol = ? and status = ?", constant.RDP, constant.Connected).Count(&online).Error
	return
}

// GetVNCOnlineCount 查询VNC协议在线数
func (s *SessionRepositoryNew) GetVNCOnlineCount(c context.Context) (online int64, err error) {
	err = s.GetDB(c).Model(&model.Session{}).Where("protocol = ? and status = ?", constant.VNC, constant.Connected).Count(&online).Error
	return
}

// GetTelnetOnlineCount 查询Telnet协议在线数
func (s *SessionRepositoryNew) GetTelnetOnlineCount(c context.Context) (online int64, err error) {
	err = s.GetDB(c).Model(&model.Session{}).Where("protocol = ? and status = ?", constant.TELNET, constant.Connected).Count(&online).Error
	return
}

// GetRecentUsedDevice 查询用户的最近设备使用数
func (s *SessionRepositoryNew) GetRecentUsedDevice(c context.Context, username, auto, name, ip, protocol, passport string) (hostOperateArr []model.HostOperate, err error) {
	var sql0 string
	var sql1 string
	var sql2 string
	sql0 = "select a.passport_id as id,a.asset_name,a.asset_ip as ip,a.protocol,a.pass_port as name,b.status,count(a.pass_port) as cnt from session as a  left join pass_ports as b on a.passport_id = b.id where a.create_name =  '" + username + "'"
	sql2 = " group by a.passport_id, a.asset_name, a.asset_ip, a.protocol, a.pass_port, b.status order by cnt desc"
	if auto != "" {
		sql1 += " and ( a.asset_name like '%" + auto + "%'" + " or a.asset_ip like '%" + auto + "%'" + " or a.pass_port like '%" + auto + "%'" + " or a.protocol like '%" + auto + "%') "
	}
	if name != "" {
		sql1 += " and asset_name like '%" + name + "%'"
	}
	if ip != "" {
		sql1 += " and asset_ip like '%" + ip + "%'"
	}
	if protocol != "" {
		sql1 += " and protocol like '%" + protocol + "%'"
	}
	if passport != "" {
		sql1 += " and pass_port like '%" + passport + "%'"
	}
	fmt.Println(sql0+sql1+sql2, 111)
	err = s.GetDB(c).Raw(sql0 + sql1 + sql2).Scan(&hostOperateArr).Error
	return hostOperateArr, err
}

// CreateFileRecord 创建文件操作历史记录
func (s *SessionRepositoryNew) CreateFileRecord(c context.Context, record *model.FileRecord) (err error) {
	record.ID = utils.UUID()
	err = s.GetDB(c).Create(record).Error
	return
}

// GetFileRecord 获取文件操作历史记录
func (s *SessionRepositoryNew) GetFileRecord(c context.Context, id string) (record model.FileRecord, err error) {
	err = s.GetDB(c).Where("id = ?", id).First(&record).Error
	return
}

// GetFileRecordBySessionId 获取sessionId的文件操作历史记录
func (s *SessionRepositoryNew) GetFileRecordBySessionId(c context.Context, param ...string) (record []model.FileRecord, err error) {
	if len(param) > 0 {
		db := s.GetDB(c).Where("session_id = ?", param[0])
		if len(param) > 1 {
			if param[1] == "上传" || param[1] == "上" || param[1] == "传" {
				db = db.Where("action = ?", constant.UPLOAD)
			} else if param[1] == "下载" || param[1] == "下" || param[1] == "载" {
				db = db.Where("action = ?", constant.DOWNLOAD)
			} else {
				db = db.Where("file_name like ? or create_time like ?", "%"+param[1]+"%", "%"+param[1]+"%")
			}
		}
		err = db.Order("create_time asc").Find(&record).Error
	}
	return
}

// UpdateRDPStorageId 更新RDP会话的StorageId
func (s *SessionRepositoryNew) UpdateRDPStorageId(c context.Context, id, storageId string) (err error) {
	err = s.GetDB(c).Model(&model.Session{}).Where("id = ?", id).Update("storage_id", storageId).Error
	return
}

// UpdateAllSessionStatus 将所有的会话状态改为已断开，断开时间为当前时间
func (s *SessionRepositoryNew) UpdateAllSessionStatus(c context.Context) (err error) {
	err = s.GetDB(c).Model(&model.Session{}).Where("status = ?", constant.Connected).Updates(map[string]interface{}{"status": constant.Disconnected, "disconnected_time": time.Now()}).Error
	return
}

// CreateClipboardRecord 新建剪切板记录
func (s *SessionRepositoryNew) CreateClipboardRecord(c context.Context, record *model.ClipboardRecord) (err error) {
	record.ID = utils.UUID()
	err = s.GetDB(c).Create(record).Error
	return
}

// GetClipboardRecord 获取session的剪切板记录
func (s *SessionRepositoryNew) GetClipboardRecord(c context.Context, id string) (record []dto.ClipboardRecord, err error) {
	var clipboardRecord []model.ClipboardRecord
	err = s.GetDB(c).Where("session_id = ?", id).Order("clip_time desc").Find(&clipboardRecord).Error
	if err != nil {
		return
	}
	if len(clipboardRecord) > 0 {
		for _, v := range clipboardRecord {
			record = append(record, dto.ClipboardRecord{
				Content:  v.Content,
				ClipTime: v.ClipTime.Format("2006-01-02 15:04:05"),
			})
		}
	}
	return
}
