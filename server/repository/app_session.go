package repository

import (
	"context"
	"os"
	"path"
	"time"
	"tkbastion/pkg/constant"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/dustin/go-humanize"
)

type AppSessionRepository struct {
	baseRepository
}

// Create 新建会话
func (s *AppSessionRepository) Create(c context.Context, session *model.AppSession) error {
	return s.GetDB(c).Create(session).Error
}

// Update 更新会话
func (s *AppSessionRepository) Update(c context.Context, session *model.AppSession, id string) error {
	return s.GetDB(c).Model(&model.AppSession{}).Where("id = ?", id).Updates(session).Error
}

// Delete 删除
func (s *AppSessionRepository) Delete(c context.Context, id string) error {
	return s.GetDB(c).Where("id = ?", id).Delete(&model.AppSession{}).Error
}

// GetById 根据id获取会话
func (s *AppSessionRepository) GetById(c context.Context, id string) (model.AppSession, error) {
	var session model.AppSession
	err := s.GetDB(c).Where("id = ?", id).First(&session).Error
	return session, err
}

// UpdateVideoDownload 更新视频下载状态
func (s *AppSessionRepository) UpdateVideoDownload(c context.Context, id string, status int) error {
	return s.GetDB(c).Model(&model.AppSession{}).Where("id = ?", id).Update("download_status", status).Error
}

// GetRecentAppIdByUserId 通过用户id获取用户7天内最多使用的前15个应用id
func (s *AppSessionRepository) GetRecentAppIdByUserId(c context.Context, userId string) (appIds []string, err error) {
	var session []model.AppSession
	err = s.GetDB(c).Model(&model.AppSession{}).Select("app_id,count(app_id) as count").Where("creator = ?", userId).Where("disconnected_time > ?", time.Now().AddDate(0, 0, -7)).Group("app_id").Order("count desc").Limit(15).Find(&session).Error
	if err != nil {
		return
	}
	appIds = make([]string, 0, len(session))
	for _, v := range session {
		appIds = append(appIds, v.AppId)
	}
	return
}

func (s *AppSessionRepository) UpdateWindowSizeById(todo context.Context, width int, height int, id string) error {
	return s.GetDB(todo).Model(&model.AppSession{}).Where("id = ?", id).Update("width", width).Update("height", height).Error
}

// GetHistorySession 查询历史会话
func (s *AppSessionRepository) GetHistorySession(c context.Context, ss dto.AppSessionForSearch) (sp []dto.AppSessionForPage, err error) {
	db := s.GetDB(c).Model(&model.AppSession{}).Where("department_id in (?) and disconnected_time is not null and status = ?", ss.DepartmentIds, constant.Disconnected)
	if ss.Auto != "" {
		db = db.Where("app_name like ?", "%"+ss.Auto+"%").Or("app_ip like ?", "%"+ss.Auto+"%").Or("program_name like ?", "%"+ss.Auto+"%").Or("pass_port like ?", "%"+ss.Auto+"%").Or("client_ip like ?", "%"+ss.Auto+"%").Or("create_name like ?", "%"+ss.Auto+"%").Or("create_nick like ?", "%"+ss.Auto+"%")
	} else if ss.AppName != "" {
		db = db.Where("app_name like ?", "%"+ss.AppName+"%")
	} else if ss.IP != "" {
		db = db.Where("app_ip like ?", "%"+ss.IP+"%")
	} else if ss.Program != "" {
		db = db.Where("program_name like ?", "%"+ss.Program+"%")
	} else if ss.Passport != "" {
		db = db.Where("pass_port like ?", "%"+ss.Passport+"%")
	} else if ss.UserName != "" {
		db = db.Where("create_name like ?", "%"+ss.UserName+"%")
	} else if ss.NickName != "" {
		db = db.Where("create_nick like ?", "%"+ss.NickName+"%")
	}

	var session []model.AppSession
	err = db.Order("disconnected_time desc").Find(&session).Error
	if err != nil {
		return
	}
	sp = make([]dto.AppSessionForPage, 0, len(session))
	for _, v := range session {
		var downloadStatus string
		if v.DownloadStatus == 0 {
			downloadStatus = "未解码"
		} else if v.DownloadStatus == 1 {
			downloadStatus = "解码中"
		} else {
			downloadStatus = "已解码"
		}

		sp = append(sp, dto.AppSessionForPage{
			ID:             v.ID,
			AppName:        v.AppName,
			Program:        v.ProgramName,
			IP:             v.ClientIP,
			Passport:       v.PassPort,
			UserName:       v.CreateName,
			NickName:       v.CreateNick,
			IsRead:         v.IsRead == 1,
			IsRecord:       utils.FileExists(path.Join(v.Recording)),
			DownloadStatus: downloadStatus,
			StartTime:      v.ConnectedTime.Format("2006-01-02 15:04:05"),
			EndTime:        v.DisconnectedTime.Format("2006-01-02 15:04:05"),
		})
	}
	return
}

// GetExportSession 获取会话导出数据
func (s *AppSessionRepository) GetExportSession(c context.Context, departments []int64) (se []dto.AppSessionForExport, err error) {
	var session []model.AppSession
	err = s.GetDB(c).Model(&model.AppSession{}).Where("department_id in (?) and disconnected_time is not null and status = ?", departments, constant.Disconnected).Order("disconnected_time desc").Find(&session).Error
	if err != nil {
		return
	}
	se = make([]dto.AppSessionForExport, 0, len(session))
	for _, v := range session {
		se = append(se, dto.AppSessionForExport{
			AppName:   v.AppName,
			Program:   v.ProgramName,
			ClientIp:  v.ClientIP,
			Passport:  v.PassPort,
			Username:  v.CreateName,
			Nickname:  v.CreateNick,
			StartTime: v.ConnectedTime.Format("2006-01-02 15:04:05"),
			EndTime:   v.DisconnectedTime.Format("2006-01-02 15:04:05"),
		})
	}
	return
}

// GetDetailSession 查询会话详情
func (s *AppSessionRepository) GetDetailSession(c context.Context, id string) (sp dto.AppSessionForDetail, err error) {
	var session model.AppSession
	err = s.GetDB(c).Where("id = ?", id).First(&session).Error
	if err != nil {
		return
	}

	user, err := userNewRepository.FindById(session.Creator)
	if err != nil {
		return
	}

	var size string
	stat, err := os.Stat(path.Join(session.Recording, "recording"))
	if err != nil {
		size = "未查出"
	} else {
		size = humanize.Bytes(uint64(stat.Size()))
	}

	// 获取会话时长
	duration := session.DisconnectedTime.Time.Sub(session.ConnectedTime.Time)
	sp = dto.AppSessionForDetail{
		AppName:           session.AppName,
		IP:                session.AppIP,
		Passport:          session.PassPort,
		Program:           session.ProgramName,
		UserName:          session.CreateName,
		NickName:          session.CreateNick,
		CilentIP:          session.ClientIP,
		UserId:            user.ID,
		AuthenticationWay: user.AuthenticationWay,
		StartAndEnd:       session.ConnectedTime.Format("2006-01-02 15:04:05") + " - " + session.DisconnectedTime.Format("2006-01-02 15:04:05"),
		// 会话时长
		SessionTime: duration.String(),
		SessionSzie: size,
	}
	return
}

func (s *AppSessionRepository) MarkAsRead(c context.Context, id []string) error {
	return s.GetDB(c).Model(&model.AppSession{}).Where("id in (?)", id).Update("is_read", 1).Error
}

func (s *AppSessionRepository) MarkAsUnRead(c context.Context, id []string) error {
	return s.GetDB(c).Model(&model.AppSession{}).Where("id in (?)", id).Update("is_read", 0).Error
}

func (s *AppSessionRepository) UpdateDownloadStatus(c context.Context, id string, status int) error {
	return s.GetDB(c).Model(&model.AppSession{}).Where("id = ?", id).Update("download_status", status).Error
}

// MarkAllAsRead 全部标为已读
func (s *AppSessionRepository) MarkAllAsRead(c context.Context) error {
	return s.GetDB(c).Model(&model.AppSession{}).Where("is_read = ?", 0).Update("is_read", 1).Error
}

// GetAppOnlineCount 查询应用在线数
func (s *AppSessionRepository) GetAppOnlineCount(c context.Context) (count int64, err error) {
	err = s.GetDB(c).Model(&model.AppSession{}).Where("status = ?", constant.Connected).Count(&count).Error
	return
}

// GetAllAppSession 获取所有应用会话
func (s *AppSessionRepository) GetAllAppSession(c context.Context) (session []model.AppSession, err error) {
	err = s.GetDB(c).Find(&session).Error
	return
}
