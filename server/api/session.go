package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
	"tkbastion/pkg/config"
	"tkbastion/pkg/constant"
	"tkbastion/pkg/global"
	"tkbastion/pkg/global/file_session"
	"tkbastion/pkg/global/session"
	"tkbastion/pkg/guacd"
	"tkbastion/pkg/log"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/repository"
	"tkbastion/server/service"
	"tkbastion/server/utils"

	"github.com/jlaffaye/ftp"

	"github.com/pkg/sftp"

	"gorm.io/gorm"

	"github.com/gorilla/websocket"

	"github.com/labstack/echo/v4"
)

// NewSessionCreateEndpoint 1. 连接设备->创建会话
func NewSessionCreateEndpoint(c echo.Context) error {
	assetId := c.QueryParam("assetId")
	mode := c.QueryParam("mode")
	download := c.QueryParam("download")
	upload := c.QueryParam("upload")
	watermark := c.QueryParam("watermark")
	user, _ := GetCurrentAccountNew(c)

	if mode != "app" {
		err, isAllow := sysMaintainService.IsAllowOperate()
		if nil != err {
			log.Error("主机运维-设备接入: 获取系统授权类型失败")
			return FailWithDataOperate(c, 500, "登录失败, 请联系技术人员查看系统日志以确定失败原因", "", nil)
		}
		if !isAllow {
			return FailWithDataOperate(c, 400, "系统被授权后才可登录运维主机, 请您先联系厂商获取授权文件", "主机运维-设备接入: 接入失败, 失败原因[系统未获取授权]", nil)
		}
		s, err := createNewSession(c.RealIP(), assetId, &user, false, download, upload, watermark)
		if err != nil {
			return err
		}
		return Success(c, s)
	} else {
		//err, isAllow := sysMaintainService.IsAllowOperate()
		//if nil != err {
		//	log.Error("主机运维-设备接入: 获取系统授权类型失败")
		//	return FailWithDataOperate(c, 500, "登录失败, 请联系技术人员查看系统日志以确定失败原因", "", nil)
		//}
		//if !isAllow {
		//	return FailWithDataOperate(c, 400, "系统被授权后才可登录运维主机, 请您先联系厂商获取授权文件", "主机运维-设备接入: 接入失败, 失败原因[系统未获取授权]", nil)
		//}
		s, err := createNewSession(c.RealIP(), assetId, &user, true)
		if err != nil {
			return err
		}
		return Success(c, s)
	}
}

// NewSessionHistoryEndpoint 查询历史会话
func NewSessionHistoryEndpoint(c echo.Context) error {
	var ss dto.SessionForSearch
	ss.AssetIP = c.QueryParam("assetIp")
	ss.Auto = c.QueryParam("auto")
	ss.UserName = c.QueryParam("userName")
	ss.IP = c.QueryParam("ip")
	ss.Protocol = c.QueryParam("protocol")
	ss.AssetName = c.QueryParam("assetName")
	ss.Passport = c.QueryParam("passport")
	ss.OperateTime = c.QueryParam("operateTime")
	ss.Command = c.QueryParam("command")

	u, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "获取用户信息失败")
	}
	var departmentIds []int64
	err := GetChildDepIds(u.DepartmentId, &departmentIds)
	if err != nil {
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	ss.DepartmentIds = departmentIds

	resp, err := newSessionRepository.GetHistorySession(context.TODO(), ss)
	if err != nil {
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	return c.JSON(200, H{
		"code":    1,
		"success": true,
		"message": "成功",
		"data":    resp,
	})
}

// NewSessionExportEndpoint 导出历史会话
func NewSessionExportEndpoint(c echo.Context) error {
	u, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	var departmentIds []int64
	err := GetChildDepIds(u.DepartmentId, &departmentIds)
	if err != nil {
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}

	assetForExport, err := repository.SessionRepo.GetExportSession(context.TODO(), departmentIds)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	assetStringsForExport := make([][]string, len(assetForExport))
	for i, v := range assetForExport {
		asset := utils.Struct2StrArr(v)
		assetStringsForExport[i] = make([]string, len(asset))
		assetStringsForExport[i] = asset
	}
	assetHeaderForExport := []string{"开始时间", "结束时间", "用户名", "姓名", "来源地址", "设备名称", "设备地址", "设备账号", "协议"}
	assetFileNameForExport := "主机历史会话"
	file, err := utils.CreateExcelFile(assetFileNameForExport, assetHeaderForExport, assetStringsForExport)
	if err != nil {
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	fileName := "主机历史会话.xlsx"

	// 记录操作日志
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		Result:          "成功",
		LogTypes:        "用户日志",
		Names:           u.Nickname,
		Created:         utils.NowJsonTime(),
		LogContents:     "主机审计-导出: 导出主机历史会话成功",
		Users:           u.Username,
	}
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}

	var buff bytes.Buffer
	if err = file.Write(&buff); err != nil {
		log.Errorf("Write Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "", err)
	}

	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
}

// NewSessionOnlineEndpoint 查询在线会话
func NewSessionOnlineEndpoint(c echo.Context) error {
	var ss dto.SessionForSearch
	ss.AssetIP = c.QueryParam("assetIp")
	ss.Auto = c.QueryParam("auto")
	ss.Protocol = c.QueryParam("assetName")
	ss.UserName = c.QueryParam("userName")
	ss.IP = c.QueryParam("ip")

	u, _ := GetCurrentAccountNew(c)
	var departmentIds []int64
	err := GetChildDepIds(u.DepartmentId, &departmentIds)
	if err != nil {
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	ss.DepartmentIds = departmentIds

	resp, err := newSessionRepository.GetOnlineSession(context.TODO(), ss)
	if err != nil {
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	return c.JSON(200, H{
		"code":    1,
		"success": true,
		"message": "成功",
		"data":    resp,
	})
}

func SessionClipboardCreateEndpoint(c echo.Context) error {
	sessionId := c.Param("id")
	clips := c.FormValue("clipboard")
	t := utils.NowJsonTime()

	if err := newSessionRepository.CreateClipboardRecord(context.TODO(), &model.ClipboardRecord{
		SessionId: sessionId,
		Content:   clips,
		ClipTime:  t,
	}); err != nil {
		return FailWithDataOperate(c, 500, "保存失败", "", nil)
	}
	return Success(c, nil)
}

func SessionClipboardGetEndpoint(c echo.Context) error {
	sessionId := c.Param("id")
	clips, err := newSessionRepository.GetClipboardRecord(context.TODO(), sessionId)
	if err != nil {
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	return c.JSON(200, H{
		"code":    1,
		"success": true,
		"message": "成功",
		"data":    clips,
	})
}

// NewSessionDetailEndpoint 会话详情
func NewSessionDetailEndpoint(c echo.Context) error {
	sessionId := c.Param("id")
	s, err := newSessionRepository.GetSessionDetailById(context.TODO(), sessionId)
	if err != nil {
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	return c.JSON(200, H{
		"code":    1,
		"success": true,
		"message": "成功",
		"data":    s,
	})
}

func createNewSession(clientIp, assetId string, user *model.UserNew, app bool, auth ...string) (string, error) {
	if app {
		app, err := newApplicationRepository.GetApplicationById(context.TODO(), assetId)
		if err != nil {
			log.Errorf("主机运维-应用接入: 获取应用信息失败, 失败原因: %s", err)
			return "", err
		}
		s := model.AppSession{
			ID:           utils.UUID(),
			AppId:        app.ID,
			AppName:      app.Name,
			AppIP:        app.IP,
			AppPort:      app.Port,
			ProgramName:  app.ProgramName,
			ProgramId:    app.ProgramID,
			PassPort:     app.Passport,
			Creator:      user.ID,
			CreateName:   user.Username,
			DepartmentId: app.DepartmentID,
			Department:   app.Department,
			CreateNick:   user.Nickname,
			ClientIP:     clientIp,
			CreateTime:   utils.NowJsonTime(),
			Status:       constant.NoConnect,
		}
		if err := appSessionRepository.Create(context.TODO(), &s); err != nil {
			return "", err
		}
		// 创建用户协议访问记录
		err = userAccessStatisticsRepository.AddUserAccessStatisticsByProtocol(context.TODO(), user, constant.APPLICATIION, s.ID, clientIp, "失败", "登陆失败")
		if err != nil {
			log.Errorf("创建用户协议访问记录失败: %v", err)
		}
		return s.ID, nil
	} else {
		passport, err := newAssetRepository.GetPassportById(context.TODO(), assetId)
		if err != nil {
			return "", err
		}
		ppconfig, err := repository.AssetNewDao.GetPassportConfig(context.TODO(), passport.ID)
		if err != nil {
			return "", err
		}
		s := &model.Session{
			ID:           utils.UUID(),
			Protocol:     passport.Protocol,
			PassportId:   passport.ID,
			AssetName:    passport.AssetName,
			AssetIP:      passport.Ip,
			PassPort:     passport.Passport,
			AssetPort:    passport.Port,
			Creator:      user.ID,
			CreateName:   user.Username,
			DepartmentId: passport.DepartmentId,
			CreateNick:   user.Nickname,
			ClientIP:     clientIp,
			CreateTime:   utils.NowJsonTime(),
			Status:       constant.NoConnect,
		}
		if ppconfig["rdp_enable_drive"] == "true" {
			s.StorageId = ppconfig["rdp_drive_path"]
		} else {
			s.StorageId = "nil"
		}
		if s.Protocol == constant.SSH {
			s.Mode = constant.Naive
		} else {
			s.Mode = constant.Guacd
		}
		toolAuth := func(auth string) int {
			if auth == "on" {
				return 1
			}
			return 0
		}
		if len(auth) > 0 {
			s.Download = toolAuth(auth[0])
			s.Upload = toolAuth(auth[1])
			s.Watermark = toolAuth(auth[2])
		}
		if err := newSessionRepository.Create(context.TODO(), s); err != nil {
			return "", err
		}

		// 创建用户协议访问记录
		err = userAccessStatisticsRepository.AddUserAccessStatisticsByProtocol(context.TODO(), user, passport.Protocol, s.ID, clientIp, "失败", "登陆失败")
		if err != nil {
			log.Errorf("创建用户协议访问记录失败: %v", err)
		}

		return s.ID, nil
	}
}

// NewSessionResizeEndpoint 2. 会话窗口大小调整
func NewSessionResizeEndpoint(c echo.Context) error {
	width := c.QueryParam("width")
	height := c.QueryParam("height")
	sessionId := c.Param("id")
	mode := c.QueryParam("mode")

	if len(width) == 0 || len(height) == 0 {
		panic("参数异常")
	}
	intWidth, _ := strconv.Atoi(width)
	intHeight, _ := strconv.Atoi(height)

	if mode != "app" {
		if err := appSessionRepository.UpdateWindowSizeById(context.TODO(), intWidth, intHeight, sessionId); err != nil {
			return FailWithDataOperate(c, 500, "更新失败", "", nil)
		}
		return Success(c, nil)
	} else {
		if err := newSessionRepository.UpdateWindowSizeById(context.TODO(), intWidth, intHeight, sessionId); err != nil {
			return err
		}
		return Success(c, "")
	}
}

// NewSessionConnectEndpoint 3. 更改会话状态为已连接
func NewSessionConnectEndpoint(c echo.Context) error {
	sessionId := c.Param("id")
	mode := c.QueryParam("mode")

	if mode == "app" {
		s := model.AppSession{
			Status:        constant.Connected,
			ConnectedTime: utils.NowJsonTime(),
		}

		if err := appSessionRepository.Update(context.TODO(), &s, sessionId); err != nil {
			return err
		}

		// 更新用户协议访问记录
		err := userAccessStatisticsRepository.UpdateUserAccessUpdateSuccessInfo(context.TODO(), sessionId, "登陆成功")
		if err != nil {
			log.Errorf("更新用户协议访问记录失败: %v", err)
		}

		return Success(c, nil)
	} else {
		s := model.Session{}
		s.Status = constant.Connected
		s.ConnectedTime = utils.NowJsonTime()

		if err := newSessionRepository.Update(context.TODO(), &s, sessionId); err != nil {
			return err
		}

		// 更新用户协议访问记录
		err := userAccessStatisticsRepository.UpdateUserAccessUpdateSuccessInfo(context.TODO(), sessionId, "登陆成功")
		if err != nil {
			log.Errorf("更新用户协议访问记录失败: %v", err)
		}

		return Success(c, nil)
	}
}

// NewSessionDisconnectEndpoint 断开会话
func NewSessionDisconnectEndpoint(c echo.Context) error {
	sessionIds := c.Param("id")
	var sessionInfo string
	split := strings.Split(sessionIds, ",")
	for i := range split {
		s, err := newSessionRepository.GetById(context.TODO(), split[i])
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "断开在线会话失败", "", err)
		}
		sessionInfo += s.AssetIP + ":" + strconv.Itoa(s.AssetPort) + "(" + s.AssetName + "),"
		NewCloseSessionById(split[i], ForcedDisconnect, "管理员强制关闭了此会话", false)
	}
	return SuccessWithOperate(c, "实时会话-断开: 断开会话["+sessionInfo+"]成功", nil)
}

var newMutex sync.Mutex

func NewCloseSessionById(sessionId string, code int, reason string, app bool) {
	newMutex.Lock()
	defer newMutex.Unlock()
	if app {
		tkSession := session.GlobalSessionManager.GetById(sessionId)
		if tkSession != nil {
			log.Debugf("[%v] 会话关闭，原因:%v", sessionId, reason)
			writeCloseMessage(tkSession.WebSocket, tkSession.Mode, code, reason)
			if tkSession.Observer != nil {
				tkSession.Observer.Range(func(key string, ob *session.Session) {
					writeCloseMessage(ob.WebSocket, ob.Mode, code, reason)
					log.Debugf("[%v] 强制踢出会话的观察者: %v", sessionId, ob.ID)
				})
			}
		}
		session.GlobalSessionManager.Del(sessionId)
	} else {
		s, err := newSessionRepository.GetById(context.TODO(), sessionId)
		if nil != err {
			log.Errorf("获取会话失败: %v", err)
			return
		}
		if s.Protocol != constant.FTP && s.Protocol != constant.SFTP {
			tkSession := session.GlobalSessionManager.GetById(sessionId)
			if tkSession != nil {
				log.Debugf("[%v] 会话关闭，原因:%v", sessionId, reason)
				writeCloseMessage(tkSession.WebSocket, tkSession.Mode, code, reason)
				if tkSession.Observer != nil {
					tkSession.Observer.Range(func(key string, ob *session.Session) {
						writeCloseMessage(ob.WebSocket, ob.Mode, code, reason)
						log.Debugf("[%v] 强制踢出会话的观察者: %v", sessionId, ob.ID)
					})
				}
			}
			session.GlobalSessionManager.Del(sessionId)
		} else {
			filesession := file_session.FileSessions.Get(sessionId)
			if filesession != nil {
				if filesession.Protocol == constant.SFTP {
					sftpClient := filesession.Client.(*sftp.Client)
					err := sftpClient.Close()
					if err != nil {
						log.Error("关闭SFTP连接失败", err)
					}
				} else if filesession.Protocol == constant.FTP {
					ftpClient := filesession.Client.(*ftp.ServerConn)
					err := ftpClient.Quit()
					if err != nil {
						log.Error("关闭FTP连接失败", err)
					}
				}
			}
		}
	}

	disDBSess(sessionId, app)
}

func writeCloseMessage(ws *websocket.Conn, mode string, code int, reason string) {
	switch mode {
	case constant.Guacd:
		if ws != nil {
			err := guacd.NewInstruction("error", "", strconv.Itoa(code))
			_ = ws.WriteMessage(websocket.TextMessage, []byte(err.String()))
			disconnect := guacd.NewInstruction("disconnect")
			_ = ws.WriteMessage(websocket.TextMessage, []byte(disconnect.String()))
		}
	case constant.Naive:
		if ws != nil {
			msg := `0` + reason
			_ = ws.WriteMessage(websocket.TextMessage, []byte(msg))
		}
	case constant.Terminal:
		// 这里是关闭观察者的ssh会话
		if ws != nil {
			msg := `0` + reason
			_ = ws.WriteMessage(websocket.TextMessage, []byte(msg))
		}
	}
}

// 文件浏览器-----------------------------------------------------------------------------------------------------------------------------

// NewLsEndpoint 1. 获取文件列表
func NewLsEndpoint(c echo.Context) error {
	sessionId := c.Param("id")
	s, err := newSessionRepository.GetById(context.TODO(), sessionId)
	if err != nil {
		return err
	}

	remoteDir := c.FormValue("dir")
	if "ssh" == s.Protocol {
		tkSession := session.GlobalSessionManager.GetById(sessionId)
		if tkSession == nil {
			return errors.New("获取会话失败")
		}

		if tkSession.Terminal.SftpClient == nil {
			sftpClient, err := sftp.NewClient(tkSession.Terminal.SshClient)
			if err != nil {
				return err
			}
			tkSession.Terminal.SftpClient = sftpClient
		}

		fileInfos, err := tkSession.Terminal.SftpClient.ReadDir(remoteDir)
		if err != nil {
			return err
		}

		var files = make([]service.File, 0)
		for i := range fileInfos {
			file := service.File{
				Name:    fileInfos[i].Name(),
				Path:    path.Join(remoteDir, fileInfos[i].Name()),
				IsDir:   fileInfos[i].IsDir(),
				Mode:    fileInfos[i].Mode().String(),
				IsLink:  fileInfos[i].Mode()&os.ModeSymlink == os.ModeSymlink,
				ModTime: utils.NewJsonTime(fileInfos[i].ModTime()),
				Size:    fileInfos[i].Size(),
			}

			files = append(files, file)
		}

		return Success(c, files)
	} else if "rdp" == s.Protocol {
		if s.StorageId != "nil" {
			storageId := s.StorageId
			files, err := storageNewService.StorageLs(remoteDir, storageId)
			if err != nil {
				return err
			}
			return Success(c, files)
		}
		return Success(c, []service.File{})
	}

	return errors.New("当前协议不支持此操作")
}

// NewDownloadEndpoint 2. 下载文件
func NewDownloadEndpoint(c echo.Context) error {
	sessionId := c.Param("id")
	s, err := newSessionRepository.GetById(context.TODO(), sessionId)
	if err != nil {
		return FailWithDataOperate(c, 500, "下载文件失败", "", err)
	}
	if s.Download != 1 {
		return FailWithDataOperate(c, 403, "下载文件失败", "", errors.New("当前会话不支持下载文件"))
	}
	file := c.QueryParam("file")

	// 获取带后缀的文件名称
	filenameWithSuffix := path.Base(file)
	if "ssh" == s.Protocol {
		tkSession := session.GlobalSessionManager.GetById(sessionId)
		if tkSession == nil {
			return errors.New("获取会话失败")
		}

		if f, err := tkSession.Terminal.SftpClient.Stat(file); err == nil {
			size := f.Size()
			if size > 1024*1024*1024 {
				return FailWithDataOperate(c, 403, "下载文件失败", "", errors.New("文件过大，不支持下载"))
			}
		} else {
			return err
		}

		dstFileForLocal, err := tkSession.Terminal.SftpClient.Open(file)
		if err != nil {
			return err
		}
		defer func(dstFile *sftp.File) {
			err := dstFile.Close()
			if err != nil {
				log.Error("关闭文件失败")
			}
		}(dstFileForLocal)
		// 保存一份到本地
		if _, err := os.Stat(path.Join(config.GlobalCfg.Guacd.Recording, sessionId)); os.IsNotExist(err) {
			err := os.MkdirAll(path.Join(config.GlobalCfg.Guacd.Recording, sessionId), 0777)
			if err != nil {
				log.Error("创建本地文件夹失败", err)
			}
		}
		localFilePath := path.Join(config.GlobalCfg.Guacd.Recording, sessionId, utils.UUID()+"."+filenameWithSuffix)
		localFile, err := os.Create(localFilePath)
		if err != nil {
			log.Error("创建本地文件失败", err)
		}
		_, err = io.Copy(localFile, dstFileForLocal)
		if err != nil {
			log.Error("复制文件失败", err)
		}

		// 记录日志
		fileRecord := model.FileRecord{
			SessionId: sessionId,
			// 路径上取文件名
			FileName:   path.Base(file),
			CreateTime: utils.NowJsonTime(),
			Action:     constant.DOWNLOAD,
			FilePath:   localFilePath,
			FileSize:   0,
		}
		err = repository.SessionRepo.CreateFileRecord(context.Background(), &fileRecord)
		if err != nil {
			log.Errorf("记录下载文件日志失败: %v", err)
		}

		// 发送
		dstFile, err := tkSession.Terminal.SftpClient.Open(file)
		if err != nil {
			return err
		}
		defer func(dstFile *sftp.File) {
			err := dstFile.Close()
			if err != nil {
				log.Error("关闭文件失败")
			}
		}(dstFile)
		c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filenameWithSuffix))

		var buff bytes.Buffer
		if _, err := dstFile.WriteTo(&buff); err != nil {
			return err
		}
		return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
	} else if "rdp" == s.Protocol {
		storageId := s.StorageId
		// 保存一份到本地
		drivePath := config.GlobalCfg.Guacd.Drive
		if strings.Contains(file, "../") {
			return errors.New("非法请求")
		}
		p := path.Join(path.Join(drivePath, storageId), file)

		// 判断文件的大小
		if fileSize, err := utils.FileSize(p); err != nil {
			return err
		} else {
			if fileSize > 1024*1024*1024 {
				return errors.New("文件过大")
			}
		}
		dstFileForLocal, err := os.Open(p)
		if err != nil {
			return err
		}

		if _, err := os.Stat(path.Join(config.GlobalCfg.Guacd.Recording, sessionId)); os.IsNotExist(err) {
			err := os.MkdirAll(path.Join(config.GlobalCfg.Guacd.Recording, sessionId), 0777)
			if err != nil {
				log.Error("创建本地文件夹失败", err)
			}
		}
		localFilePath := path.Join(config.GlobalCfg.Guacd.Recording, sessionId, utils.UUID()+"."+filenameWithSuffix)
		localFile, err := os.Create(localFilePath)
		if err != nil {
			log.Error("创建本地文件失败", err)
		}
		_, err = io.Copy(localFile, dstFileForLocal)
		if err != nil {
			log.Error("复制文件失败", err)
		}

		// 记录日志
		fileRecord := model.FileRecord{
			SessionId: sessionId,
			// 路径上取文件名
			FileName:   path.Base(file),
			CreateTime: utils.NowJsonTime(),
			Action:     constant.DOWNLOAD,
			FilePath:   localFilePath,
			FileSize:   0,
		}
		err = repository.SessionRepo.CreateFileRecord(context.Background(), &fileRecord)
		if err != nil {
			log.Errorf("记录下载文件日志失败: %v", err)
		}

		return storageNewService.StorageDownload(c, file, storageId)
	}

	return err
}

// NewUploadEndpoint 3. 上传文件
func NewUploadEndpoint(c echo.Context) error {
	sessionId := c.Param("id")
	s, err := newSessionRepository.GetById(context.TODO(), sessionId)
	if err != nil {
		return FailWithDataOperate(c, 500, "上传文件失败", "", err)
	}
	if s.Upload != 1 {
		return FailWithDataOperate(c, 403, "上传文件失败, 无权限上传", "主机运维-上传文件: 无权限上传文件", nil)
	}

	file, err := c.FormFile("file")
	if err != nil {
		return FailWithDataOperate(c, 500, "上传文件失败", "", err)
	}
	filename := file.Filename

	// 保存一份到本地
	fileCopeForLocal, err := file.Open()
	if err != nil {
		log.Error("打开文件失败", err)
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}
	if _, err := os.Stat(path.Join(config.GlobalCfg.Guacd.Recording, sessionId)); os.IsNotExist(err) {
		err := os.MkdirAll(path.Join(config.GlobalCfg.Guacd.Recording, sessionId), 0777)
		if err != nil {
			log.Error("创建本地文件夹失败", err)
		}
	}
	localFilePath := path.Join(config.GlobalCfg.Guacd.Recording, sessionId, utils.UUID()+"."+filename)
	if _, err := os.Stat(path.Join(config.GlobalCfg.Guacd.Recording, sessionId)); !os.IsNotExist(err) {
		localFile, err := os.Create(localFilePath)
		if err != nil {
			log.Error("创建本地文件失败", err)
		}
		defer func(localFile *os.File) {
			err := localFile.Close()
			if err != nil {
				log.Error("关闭本地文件失败", err)
			}
		}(localFile)
		_, err = io.Copy(localFile, fileCopeForLocal)
		if err != nil {
			log.Error("复制文件失败", err)
		}
	}
	log.Infof("上传文件到本地成功，文件路径：%s", localFilePath)

	src, err := file.Open()
	if err != nil {
		return FailWithDataOperate(c, 500, "上传文件失败", "", err)
	}

	remoteDir := c.QueryParam("dir")
	remoteFile := path.Join(remoteDir, filename)

	// 记录日志
	err = repository.SessionRepo.CreateFileRecord(context.Background(), &model.FileRecord{
		SessionId:  sessionId,
		FileName:   filename,
		FileSize:   0,
		CreateTime: utils.NowJsonTime(),
		Action:     constant.UPLOAD,
		FilePath:   localFilePath,
	})
	if err != nil {
		log.Errorf("记录上传文件日志失败: %v", err)
	}

	if "ssh" == s.Protocol {
		tkSession := session.GlobalSessionManager.GetById(sessionId)
		if tkSession == nil {
			return errors.New("获取会话失败")
		}

		sftpClient := tkSession.Terminal.SftpClient
		// 文件夹不存在时自动创建文件夹
		if _, err := sftpClient.Stat(remoteDir); os.IsNotExist(err) {
			if err := sftpClient.MkdirAll(remoteDir); err != nil {
				return err
			}
		}

		dstFile, err := sftpClient.Create(remoteFile)
		if err != nil {
			return err
		}
		defer func(dstFile *sftp.File) {
			err := dstFile.Close()
			if err != nil {
				log.Error("关闭文件失败", err)
			}
		}(dstFile)

		if _, err = io.Copy(dstFile, src); err != nil {
			return err
		}
		return Success(c, nil)
	} else if "rdp" == s.Protocol {
		if err := storageNewService.StorageUpload(c, file, s.StorageId); err != nil {
			return err
		}
		return Success(c, nil)
	}

	return err
}

// -------------------------------------------------------------------------------------------------------------------------------------

// NewSessionReplayEndpoint 会话回放
func NewSessionReplayEndpoint(c echo.Context) error {
	sessionId := c.Param("id")
	var recording string
	s, err := newSessionRepository.GetById(context.TODO(), sessionId)
	if err == gorm.ErrRecordNotFound {
		return FailWithDataOperate(c, 404, "会话不存在", "主机审计-回放: 会话不存在", err)
	} else if err == nil {
		if constant.SSH == s.Protocol {
			recording = s.Recording
		} else {
			recording = s.Recording + "/recording"
		}
	} else {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "回放失败", "", err)
	}

	log.Debugf("读取录屏文件: %v, 是否存在: %v, 是否为文件: %v", recording, utils.FileExists(recording), utils.IsFile(recording))
	user, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithDataOperate(c, 401, "回放失败", "主机审计-回放: 用户未登录", err)
	}

	if !utils.FileExists(recording) || !utils.IsFile(recording) {
		return FailWithDataOperate(c, 404, "回放失败", "主机审计-回放: 录屏文件不存在", err)
	}

	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "审计日志",
		Created:         utils.NowJsonTime(),
		Users:           user.Username,
		Names:           user.Nickname,
		Result:          "成功",
		LogContents:     "主机审计-回放: 资产" + s.AssetIP + ":" + strconv.Itoa(s.AssetPort) + "(" + s.AssetName + ")" + ", 连接协议" + s.Protocol + ", 来源IP: " + s.ClientIP + ", 接入时间: " + s.ConnectedTime.String(),
	}
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}

	http.ServeFile(c.Response(), c.Request(), recording)
	return nil
}

func disDBSess(sessionId string, app bool) {
	if app {
		s, err := appSessionRepository.GetById(context.TODO(), sessionId)
		if err != nil {
			return
		}
		if s.Status == constant.Disconnected {
			return
		}

		if s.Status == constant.Connecting {
			// 会话还未建立成功，无需保留数据
			_ = appSessionRepository.Delete(context.TODO(), sessionId)
			return
		}

		ss := model.AppSession{}
		ss.ID = sessionId
		ss.Status = constant.Disconnected
		ss.DisconnectedTime = utils.NowJsonTime()

		_ = appSessionRepository.Update(context.TODO(), &ss, sessionId)

		ol := model.OperationAndMaintenanceLog{
			LoginTime:    s.ConnectedTime,
			Username:     s.CreateName,
			Nickname:     s.CreateNick,
			Ip:           s.ClientIP,
			AssetName:    s.AppName,
			AssetIp:      s.AppIP,
			Passport:     s.PassPort,
			Protocol:     "应用",
			DepartmentID: s.DepartmentId,
			LogoutTime:   ss.DisconnectedTime,
		}
		err = global.DBConn.Model(model.OperationAndMaintenanceLog{}).Create(&ol).Error
		if nil != err {
			log.Errorf("DB Error: %v", err)
		}
	} else {
		s, err := newSessionRepository.GetById(context.TODO(), sessionId)
		if err != nil {
			return
		}
		if s.Status == constant.Disconnected {
			return
		}

		if s.Status == constant.Connecting {
			// 会话还未建立成功，无需保留数据
			_ = newSessionRepository.Delete(context.TODO(), sessionId)
			return
		}

		ss := model.Session{}
		ss.ID = sessionId
		ss.Status = constant.Disconnected
		ss.DisconnectedTime = utils.NowJsonTime()

		_ = newSessionRepository.Update(context.TODO(), &ss, sessionId)

		ol := model.OperationAndMaintenanceLog{
			LoginTime:    s.ConnectedTime,
			Username:     s.CreateName,
			Nickname:     s.CreateNick,
			Ip:           s.ClientIP,
			AssetName:    s.AssetName,
			AssetIp:      s.AssetIP,
			Passport:     s.PassPort,
			Protocol:     s.Protocol,
			DepartmentID: s.DepartmentId,
			LogoutTime:   ss.DisconnectedTime,
		}

		err = global.DBConn.Model(model.OperationAndMaintenanceLog{}).Create(&ol).Error
		if nil != err {
			log.Errorf("DB Error: %v", err)
		}
	}
}

type OnlineUser struct {
	Token          string         `json:"token"`
	UserName       string         `json:"userName"`
	NickName       string         `json:"nickName"`
	LoginTime      utils.JsonTime `json:"loginTime"`
	LoginAddress   string         `json:"loginAddress"`
	LastActiveTime utils.JsonTime `json:"lastActiveTime"`
	IsOwn          string         `json:"isOwn"`
}

func OnlineUsersPagingEndpoint(c echo.Context) error {
	ownToken := GetToken(c)
	var OnlineUserArr []OnlineUser

	cacheM := global.Cache.Items()
	for k, v := range cacheM {
		if strings.Contains(k, constant.Token) {
			var onlineUser OnlineUser
			onlineUser.Token = v.Object.(global.AuthorizationNew).Token
			user := v.Object.(global.AuthorizationNew).UserNew
			onlineUser.UserName = user.Username
			onlineUser.NickName = user.Nickname
			onlineUser.LoginTime = v.Object.(global.AuthorizationNew).LoginTime
			onlineUser.LoginAddress = v.Object.(global.AuthorizationNew).LoginAddress
			onlineUser.LastActiveTime = v.Object.(global.AuthorizationNew).LastActiveTime

			if strings.EqualFold(ownToken, v.Object.(global.AuthorizationNew).Token) {
				onlineUser.IsOwn = "true"
			} else {
				onlineUser.IsOwn = "false"
			}

			OnlineUserArr = append(OnlineUserArr, onlineUser)
		}
	}

	return SuccessWithOperate(c, "", OnlineUserArr)
}

func OnlineUsersDisconnect(c echo.Context) error {
	disconnectToken := c.QueryParam("token")
	ownToken := GetToken(c)
	if strings.EqualFold(ownToken, disconnectToken) {
		return FailWithDataOperate(c, 400, "不允许断开当前登录用户连接", "在线用户-断开: 断开操作者自身连接, 失败原因[不允许断开当前登录用户的连接]", nil)
	}

	cacheKey := BuildCacheKeyByToken(disconnectToken)
	auth, isExist := global.Cache.Get(cacheKey)
	if !isExist {
		// 在执行断开前该用户token已被删除
		return SuccessWithOperate(c, "", nil)
	}
	global.Cache.Delete(cacheKey)

	return SuccessWithOperate(c, "在线用户-断开: 用户名["+auth.(global.AuthorizationNew).UserNew.Username+"], 登录地址["+auth.(global.AuthorizationNew).LoginAddress+"]", nil)
}

// MarkAsRead 标为已阅
func MarkAsRead(c echo.Context) error {
	user, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithDataOperate(c, 400, "标为已阅失败", "主机审计-标为已阅: 失败原因[获取当前用户失败]", nil)
	}
	ids := c.QueryParam("ids")
	if ids == "" {
		return FailWithDataOperate(c, 400, "标记已读失败", "主机审计-标为已阅: 失败原因[参数错误]", nil)
	}

	if ids == "all" {
		err := newSessionRepository.UpdateAllRead(context.TODO())
		if err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "标记已读失败", "主机审计-标为已阅: 失败原因[数据库错误]", nil)
		}
		return SuccessWithOperate(c, "主机审计-标为已阅: 用户["+user.Username+"]"+", 标记所有主机记录为已阅", nil)
	}

	idsArr := strings.Split(ids, ",")
	err := newSessionRepository.UpdateRead(context.TODO(), idsArr)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "标记已读失败", "主机审计-标为已阅: 失败原因[数据库错误]", nil)
	}
	return SuccessWithOperate(c, "主机审计-标为已阅: 用户["+user.Username+"]"+", 标记主机记录["+ids+"]为已阅", nil)
}

// MarkAsUnread 标为未阅
func MarkAsUnread(c echo.Context) error {
	u, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithDataOperate(c, 400, "标记未读失败", "主机审计-标为未阅: 失败原因[获取当前用户失败]", nil)
	}
	ids := c.QueryParam("ids")
	if ids == "" {
		return FailWithDataOperate(c, 400, "标记未读失败", "主机审计-标为未阅: 失败原因[参数错误]", nil)
	}

	idsArr := strings.Split(ids, ",")
	err := newSessionRepository.UpdateUnRead(context.TODO(), idsArr)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "标记未读失败", "主机审计-标为未阅: 失败原因[数据库错误]", nil)
	}
	return SuccessWithOperate(c, "主机审计-标为未阅: 用户["+u.Username+"]"+", 标记主机记录["+ids+"]为未阅", nil)
}

// NewSessionVideoDownloadEndpoint 下载会话录像
func NewSessionVideoDownloadEndpoint(c echo.Context) error {
	sessionId := c.QueryParam("id")
	s, err := newSessionRepository.GetById(context.TODO(), sessionId)
	if err != nil {
		return FailWithDataOperate(c, 500, "下载失败", "", err)
	}

	u, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithDataOperate(c, 500, "下载失败", "", err)
	}
	if s.IsDownloadRecord == 0 {
		NewDecodeVideo(s, u)
		return SuccessWithData(c, 201, "视频开始解码", nil)
	} else if s.IsDownloadRecord == 1 {
		return SuccessWithData(c, 202, "视频正在解码中，请稍后重新点击进行下载", nil)
	} else {
		if s.Protocol != constant.SSH {
			_, err := os.Stat(s.Recording + "/recording.m4v")
			if err != nil {
				NewDecodeVideo(s, u)
				operateLog := model.OperateLog{
					Ip:              c.RealIP(),
					ClientUserAgent: c.Request().UserAgent(),
					LogTypes:        "审计日志",
					Created:         utils.NowJsonTime(),
					Result:          "失败",
					Users:           u.Username,
					Names:           u.Nickname,
					LogContents:     "主机审计-下载录像: 用户[" + u.Username + "], 下载主机[" + s.AssetName + "]的录像失败, 原因[录像文件不存在]",
				}
				err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
				if nil != err {
					log.Errorf("DB Error: %v", err)
					_ = newSessionRepository.UpdateVideoDownload(context.TODO(), s.ID, 0)
					return FailWithDataOperate(c, 500, "下载失败", "", err)
				}
				return SuccessWithData(c, 203, "文件丢失，正在为您重新解码，解码成功后会在站内消息中通知您", nil)
			}
		} else {
			if _, err := os.Stat(s.Recording[:len(s.Recording)-4] + "gif"); err != nil {
				NewDecodeVideo(s, u)
				operateLog := model.OperateLog{
					Ip:              c.RealIP(),
					ClientUserAgent: c.Request().UserAgent(),
					LogTypes:        "审计日志",
					Created:         utils.NowJsonTime(),
					Users:           u.Username,
					Names:           u.Nickname,
					Result:          "失败",
					LogContents:     "主机审计-下载会话: 用户[" + u.Username + "], 下载主机[" + s.AssetName + "]的会话录像失败, 原因[录像文件不存在]",
				}
				err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
				if nil != err {
					log.Errorf("DB Error: %v", err)
					return FailWithDataOperate(c, 500, "下载失败", "", err)
				}
				return SuccessWithData(c, 203, "文件丢失，正在为您重新解码，解码成功后会在站内消息中通知您", nil)
			}
		}
	}
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "审计日志",
		Created:         utils.NowJsonTime(),
		Users:           u.Username,
		Names:           u.Nickname,
		Result:          "成功",
		LogContents:     "主机审计-下载会话: 用户[" + u.Username + "], 下载主机[" + s.AssetName + "]的会话录像",
	}
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "下载失败", "", err)
	}
	if s.Protocol != constant.SSH {
		return c.Attachment(s.Recording+"/recording.m4v", "recording.m4v")
	} else {
		return c.Attachment(s.Recording[:len(s.Recording)-4]+"gif", "recording.gif")
	}
}

// NewDecodeVideo 录像解码
func NewDecodeVideo(s model.Session, u model.UserNew) {
	if s.Protocol != constant.SSH {
		go func() {
			err := newSessionRepository.UpdateVideoDownload(context.TODO(), s.ID, 1)
			if err != nil {
				log.Errorf("DB Error: %v", err)
			}
			commod := "guacenc  -s 1280x720 -r 20000000 " + s.Recording + "/recording"
			_, err = utils.ExecShell(commod)
			if err != nil {
				log.Errorf("执行命令失败: %v", err)
				_ = newSessionRepository.UpdateVideoDownload(context.TODO(), s.ID, 0)
				_ = messageService.SendUserMessage(u.ID, "会话下载", "", "录像[设备名称: "+s.AssetName+", 设备地址: "+s.AssetIP+", 协议: "+s.Protocol+"], 登陆用户: "+s.CreateName+", 登陆时间: "+s.CreateTime.Format("2006-01-02 15:04:05")+"]解码失败, 请联系管理员", constant.NoticeMessage)
			} else {
				_ = messageService.SendUserMessage(u.ID, "会话下载", "", "录像[设备名称: "+s.AssetName+", 设备地址: "+s.AssetIP+", 协议: "+s.Protocol+"], 登陆用户: "+s.CreateName+", 登陆时间: "+s.CreateTime.Format("2006-01-02 15:04:05")+"]解码成功，可以下载", constant.NoticeMessage)
				err = newSessionRepository.UpdateVideoDownload(context.TODO(), s.ID, 2)
				if err != nil {
					log.Errorf("更新会话下载状态失败: %v", err)
				}
			}

			p, _ := propertyRepository.FindByName("recording_save_time")
			if p.Value != "" {
				t, _ := strconv.Atoi(p.Value)
				// fmt.Println(time.Duration(t))
				time.Sleep(time.Duration(t) * time.Hour)
				err = os.Remove(s.Recording + "/recording.m4v")
				if err != nil {
					log.Errorf("删除文件失败: %v", err)
				}
				err = newSessionRepository.UpdateVideoDownload(context.TODO(), s.ID, 0)
				if err != nil {
					log.Errorf("更新会话解码状态失败: %v", err)
				}
				log.Infof("删除文件成功: %v", s.Recording+"/recording.m4v")
			}
		}()
	} else {
		go func() {
			err := newSessionRepository.UpdateVideoDownload(context.TODO(), s.ID, 1)
			if err != nil {
				log.Errorf("更新会话解码状态失败: %v", err)
			}
			var commod string
			if global.Config.Debug {
				dir, _ := os.Getwd()
				commod = dir + "/tools/agg2 " + s.Recording + " " + s.Recording[:len(s.Recording)-4] + "gif"
			} else {
				commod = "/tkbastion/tools/agg2 " + s.Recording + " " + s.Recording[:len(s.Recording)-4] + "gif"
			}
			fmt.Println(commod)
			_, err = utils.ExecShell(commod)
			if err != nil {
				log.Errorf("执行命令失败: %v", err)
				_ = newSessionRepository.UpdateVideoDownload(context.TODO(), s.ID, 0)
				_ = messageService.SendUserMessage(u.ID, "会话下载", "", "录像[设备名称: "+s.AssetName+", 设备地址: "+s.AssetIP+", 协议: "+s.Protocol+"], 登陆用户: "+s.CreateName+", 登陆时间: "+s.CreateTime.Format("2006-01-02 15:04:05")+"]解码失败, 请联系管理员", constant.NoticeMessage)
			} else {
				err = newSessionRepository.UpdateVideoDownload(context.TODO(), s.ID, 2)
				if err != nil {
					log.Errorf("更新会话解码状态失败: %v", err)
				}
				_ = messageService.SendUserMessage(u.ID, "会话下载", "", "录像[设备名称: "+s.AssetName+", 设备地址: "+s.AssetIP+", 协议: "+s.Protocol+"], 登陆用户: "+s.CreateName+", 登陆时间: "+s.CreateTime.Format("2006-01-02 15:04:05")+"]解码成功，可以下载", constant.NoticeMessage)
			}
			p, _ := propertyRepository.FindByName("recording_save_time")
			if p.Value != "" {
				t, _ := strconv.Atoi(p.Value)
				//fmt.Println(time.Duration(t))
				//time.Sleep(time.Minute * 1)
				time.Sleep(time.Duration(t) * time.Hour)
				err = os.Remove(s.Recording[:len(s.Recording)-4] + "gif")
				if err != nil {
					log.Errorf("删除文件失败: %v", err)
				}
				err = newSessionRepository.UpdateVideoDownload(context.TODO(), s.ID, 0)
				if err != nil {
					log.Errorf("更新会话解码状态失败: %v", err)
				}
				log.Infof("删除文件成功: %v", s.Recording[:len(s.Recording)-4]+"gif")
			}
		}()
	}
}

func LoginManual(c echo.Context) error {
	var passport map[string]string
	err := c.Bind(&passport)
	if err != nil {
		return Fail(c, 400, "参数错误")
	}
	if len(passport["password"]) == 0 {
		return Fail(c, 400, "参数错误")
	}

	passwd := new(global.PasswdStruct)
	if v, ok := passport["passport"]; ok {
		passwd.Passport = v
	} else {
		passwd.Passport = ""
	}
	passwd.Password = passport["password"]

	global.PasswdStore[passport["id"]] = *passwd
	return Success(c, nil)
}
