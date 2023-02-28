package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"path"
	"strconv"
	"tkbastion/pkg/global/session"
	"tkbastion/server/repository"
	"tkbastion/server/utils"

	"tkbastion/pkg/config"
	"tkbastion/pkg/constant"
	"tkbastion/pkg/global"
	"tkbastion/pkg/guacd"
	"tkbastion/pkg/log"
	"tkbastion/server/model"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

const (
	TunnelClosed             int = -1
	Normal                   int = 0
	NotFoundSession          int = 800
	NewTunnelError           int = 801
	ForcedDisconnect         int = 802
	AccessGatewayUnAvailable int = 803
	AccessGatewayCreateError int = 804
	AssetNotActive           int = 805
	NewSshClientError        int = 806
)

// NewTunEndpoint 4. 通过session id新建一个tunnel
func NewTunEndpoint(c echo.Context) error {
	ws, err := UpGrader.Upgrade(c.Response().Writer, c.Request(), nil)
	if err != nil {
		log.Errorf("升级为WebSocket协议失败:%v", err.Error())
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}

	width := c.QueryParam("width")
	height := c.QueryParam("height")
	dpi := c.QueryParam("dpi")
	sessionId := c.Param("id") //	接入资产

	intWidth, _ := strconv.Atoi(width)
	intHeight, _ := strconv.Atoi(height)

	configuration := guacd.NewConfiguration()

	user, _ := GetCurrentAccountNew(c)
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "运维日志",
		Created:         utils.NowJsonTime(),
		Users:           user.Username,
		Names:           user.Nickname,
	}

	var s model.Session

	configuration.SetParameter("width", width)
	configuration.SetParameter("height", height)
	configuration.SetParameter("dpi", dpi)
	s, err = newSessionRepository.GetById(context.TODO(), sessionId)
	if err != nil {
		log.Warnf("会话不存在")
		operateLog.Result = "失败"
		operateLog.LogContents = "应用运维-登录: 登录主机, 失败原因[会话不存在]"
		err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
		if nil != err {
			log.Errorf("记录操作日志失败: %v", err.Error())
		}
		return FailWithDataOperate(c, 400, "会话不存在", "应用运维-登录: 登录主机, 失败原因[会话不存在]", err)
	}

	setNewSessionConfig(s, configuration)

	configuration.SetParameter("hostname", s.AssetIP)
	configuration.SetParameter("port", strconv.Itoa(s.AssetPort))

	for name := range configuration.Parameters {
		// 替换数据库空格字符串占位符为真正的空格
		if configuration.Parameters[name] == "-" {
			configuration.Parameters[name] = ""
		}
	}

	if v, ok := global.PasswdStore[sessionId]; ok {
		configuration.SetParameter("password", v.Password)
		configuration.SetParameter("username", v.Passport)
		delete(global.PasswdStore, sessionId)
	}

	addr := global.Config.Guacd.Hostname + ":" + strconv.Itoa(global.Config.Guacd.Port)

	log.Infof("%#v", configuration)

	guacdTunnel, err := guacd.NewTunnel(addr, configuration)
	if err != nil {
		disConnectNewSession(ws, NewTunnelError, err.Error())
		return FailWithDataOperate(c, 400, "建立连接失败", "应用运维-登录: 登录主机["+s.AssetIP+":"+strconv.Itoa(s.AssetPort)+"], 失败原因: ["+err.Error()+"]", err)
	}

	sessionTerminal := &session.Session{
		ID:          sessionId,
		Protocol:    s.Protocol,
		WebSocket:   ws,
		Mode:        constant.Guacd,
		GuacdTunnel: guacdTunnel,
	}

	sessionTerminal.Observer = session.NewObserver(sessionId)
	session.GlobalSessionManager.Add(sessionTerminal)
	sess := model.Session{
		ConnectionId: guacdTunnel.UUID,
		Width:        intWidth,
		Height:       intHeight,
		Status:       constant.Connecting,
		Recording:    configuration.GetParameter(guacd.RecordingPath),
	}
	// 创建新会话
	log.Debugf("[%v] 创建新会话: %v", sessionId, sess.ConnectionId)
	if err := newSessionRepository.Update(context.TODO(), &sess, sessionId); err != nil {
		return FailWithDataOperate(c, 400, "登录失败", "应用运维-登录: 登录主机["+s.AssetIP+":"+strconv.Itoa(s.AssetPort)+"], 失败原因: ["+err.Error()+"]", err)
	}

	operateLog.Result = "成功"
	operateLog.LogContents = "应用运维-登录: 登录主机[" + s.AssetIP + ":" + strconv.Itoa(s.AssetPort) + "]"
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("记录操作日志失败: %v", err.Error())
	}

	guacamoleHandler := NewTunnelHandler(ws, guacdTunnel)
	guacamoleHandler.Start()
	defer guacamoleHandler.Stop()

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Debugf("[%v] WebSocket已关闭, %v", sessionId, err.Error())
			// guacdTunnel.Read() 会阻塞，所以要先把guacdTunnel客户端关闭，才能退出Guacd循环
			_ = guacdTunnel.Close()

			NewCloseSessionById(sessionId, Normal, "用户正常退出", false)
			return nil
		}
		_, err = guacdTunnel.WriteAndFlush(message)
		if err != nil {
			NewCloseSessionById(sessionId, TunnelClosed, "远程连接已关闭", false)
			return nil
		}
	}
}

func AppTunEndpoint(c echo.Context) error {
	ws, err := UpGrader.Upgrade(c.Response().Writer, c.Request(), nil)
	if err != nil {
		log.Errorf("升级为WebSocket协议失败:%v", err.Error())
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}

	width := c.QueryParam("width")
	height := c.QueryParam("height")
	dpi := c.QueryParam("dpi")
	sessionId := c.Param("id") //	接入资产
	x11id := c.QueryParam("x11id")

	intWidth, _ := strconv.Atoi(width)
	intHeight, _ := strconv.Atoi(height)

	configuration := guacd.NewConfiguration()

	user, f := GetCurrentAccountNew(c)
	if !f {
		disConnectNewSession(ws, NewTunnelError, "用户未登录")
		return FailWithDataOperate(c, 400, "用户未登录", "应用运维-登录: 登录主机, 失败原因[用户未登录]", err)
	}
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "运维日志",
		Created:         utils.NowJsonTime(),
		Users:           user.Username,
		Names:           user.Nickname,
	}

	configuration.SetParameter("width", width)
	configuration.SetParameter("height", height)
	configuration.SetParameter("dpi", dpi)
	if x11id != "" {
		s, err := repository.SessionRepo.GetById(context.TODO(), sessionId)
		if err != nil {
			log.Warnf("会话不存在")
			operateLog.Result = "失败"
			operateLog.LogContents = "应用运维-登录: 登录主机, 失败原因[会话不存在]"
			err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
			if nil != err {
				log.Errorf("编写操作日志失败: %v", err)
			}
			return FailWithDataOperate(c, 400, "会话不存在", "应用运维-登录: 登录主机, 失败原因[会话不存在]", err)
		}
		setX11SessionConfig(s, x11id, configuration)

		for name := range configuration.Parameters {
			// 替换数据库空格字符串占位符为真正的空格
			if configuration.Parameters[name] == "-" {
				configuration.Parameters[name] = ""
			}
		}

		addr := global.Config.Guacd.Hostname + ":" + strconv.Itoa(global.Config.Guacd.Port)

		guacdTunnel, err := guacd.NewTunnel(addr, configuration)
		if err != nil {
			disConnectNewSession(ws, NewTunnelError, err.Error())
			return FailWithDataOperate(c, 400, "建立连接失败", "主机运维-登录: 登录主机["+s.AssetIP+":"+strconv.Itoa(s.AssetPort)+"], 失败原因: ["+err.Error()+"]", err)
		}

		sessionTerminal := &session.Session{
			ID:          sessionId,
			Protocol:    constant.RDP,
			WebSocket:   ws,
			GuacdTunnel: guacdTunnel,
			Mode:        constant.Guacd,
		}

		sessionTerminal.Observer = session.NewObserver(sessionId)
		session.GlobalSessionManager.Add(sessionTerminal)
		sess := model.Session{
			ConnectionId: guacdTunnel.UUID,
			Width:        intWidth,
			Height:       intHeight,
			Status:       constant.Connecting,
			Recording:    configuration.GetParameter(guacd.RecordingPath),
		}
		// 创建新会话
		log.Debugf("[%v] 创建新会话: %v", sessionId, sess.ConnectionId)
		if err := newSessionRepository.Update(context.TODO(), &sess, sessionId); err != nil {
			return FailWithDataOperate(c, 400, "登录失败", "主机运维-登录: 登录主机["+s.AssetIP+":"+strconv.Itoa(s.AssetPort)+"], 失败原因: ["+err.Error()+"]", err)
		}

		operateLog.Result = "成功"
		operateLog.LogContents = "应用运维-登录: 登录主机[" + s.AssetIP + ":" + strconv.Itoa(s.AssetPort) + "]"

		err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
		if nil != err {
			log.Errorf("编写操作日志失败: %v", err)
		}

		guacamoleHandler := NewTunnelHandler(ws, guacdTunnel)
		guacamoleHandler.Start()
		defer guacamoleHandler.Stop()

		for {
			_, message, err := ws.ReadMessage()
			//fmt.Println("message:", message)
			if err != nil {
				log.Infof("[%v] WebSocket已关闭, %v", sessionId, err.Error())
				// guacdTunnel.Read() 会阻塞，所以要先把guacdTunnel客户端关闭，才能退出Guacd循环
				_ = guacdTunnel.Close()

				NewCloseSessionById(sessionId, Normal, "用户正常退出", false)
				return nil
			}
			_, err = guacdTunnel.WriteAndFlush(message)
			if err != nil {
				NewCloseSessionById(sessionId, TunnelClosed, "远程连接已关闭", false)
				return nil
			}
		}
	} else {
		s, err := appSessionRepository.GetById(context.TODO(), sessionId)
		if err != nil {
			log.Warnf("会话不存在")
			operateLog.Result = "失败"
			operateLog.LogContents = "应用运维-登录: 登录主机, 失败原因[会话不存在]"
			err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
			if nil != err {
				log.Errorf("编写操作日志失败: %v", err)
			}
			return FailWithDataOperate(c, 400, "会话不存在", "应用运维-登录: 登录主机, 失败原因[会话不存在]", err)
		}

		setAppSessionConfig(s, configuration)

		configuration.SetParameter("hostname", s.AppIP)
		configuration.SetParameter("port", strconv.Itoa(s.AppPort))

		for name := range configuration.Parameters {
			// 替换数据库空格字符串占位符为真正的空格
			if configuration.Parameters[name] == "-" {
				configuration.Parameters[name] = ""
			}
		}

		addr := global.Config.Guacd.Hostname + ":" + strconv.Itoa(global.Config.Guacd.Port)

		guacdTunnel, err := guacd.NewTunnel(addr, configuration)
		if err != nil {
			disConnectNewSession(ws, NewTunnelError, err.Error())
			return FailWithDataOperate(c, 400, "建立连接失败", "应用运维-登录: 登录应用["+s.AppIP+":"+strconv.Itoa(s.AppPort)+"], 失败原因: ["+err.Error()+"]", err)
		}

		sessionTerminal := &session.Session{
			ID:          sessionId,
			Protocol:    constant.RDP,
			WebSocket:   ws,
			GuacdTunnel: guacdTunnel,
			Mode:        constant.Guacd,
		}

		sessionTerminal.Observer = session.NewObserver(sessionId)
		session.GlobalSessionManager.Add(sessionTerminal)
		sess := model.AppSession{
			ConnectionId: guacdTunnel.UUID,
			Width:        intWidth,
			Height:       intHeight,
			Status:       constant.Connecting,
			Recording:    configuration.GetParameter(guacd.RecordingPath),
		}
		// 创建新会话
		log.Debugf("[%v] 创建新会话: %v", sessionId, sess.ConnectionId)
		if err := appSessionRepository.Update(context.TODO(), &sess, sessionId); err != nil {
			return FailWithDataOperate(c, 400, "登录失败", "应用运维-登录: 登录应用["+s.AppIP+":"+strconv.Itoa(s.AppPort)+"], 失败原因: ["+err.Error()+"]", err)
		}

		operateLog.Result = "成功"
		operateLog.LogContents = "应用运维-登录: 登录应用[" + s.AppIP + ":" + strconv.Itoa(s.AppPort) + "]"

		err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
		if nil != err {
			log.Errorf("编写操作日志失败: %v", err)
		}

		guacamoleHandler := NewTunnelHandler(ws, guacdTunnel)
		guacamoleHandler.Start()
		defer guacamoleHandler.Stop()

		for {
			_, message, err := ws.ReadMessage()
			//fmt.Println("message:", message)
			if err != nil {
				log.Infof("[%v] WebSocket已关闭, %v", sessionId, err.Error())
				// guacdTunnel.Read() 会阻塞，所以要先把guacdTunnel客户端关闭，才能退出Guacd循环
				_ = guacdTunnel.Close()

				NewCloseSessionById(sessionId, Normal, "用户正常退出", true)
				return nil
			}
			_, err = guacdTunnel.WriteAndFlush(message)
			if err != nil {
				NewCloseSessionById(sessionId, TunnelClosed, "远程连接已关闭", true)
				return nil
			}
		}
	}
}

func NewSessionTunnelMonitor(c echo.Context) error {
	ws, err := UpGrader.Upgrade(c.Response().Writer, c.Request(), nil)
	if err != nil {
		log.Errorf("升级为WebSocket协议失败:%v", err.Error())
		return err
	}
	sessionId := c.Param("id")

	user, _ := GetCurrentAccountNew(c)
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "运维日志",
		Created:         utils.NowJsonTime(),
		Users:           user.Username,
	}

	s, err := newSessionRepository.GetById(context.TODO(), sessionId)
	if err != nil {
		log.Errorf("获取会话失败: %v", err)
		return FailWithDataOperate(c, 400, "会话不存在", "主机审计-监控: 监控主机, 失败原因[会话不存在]", err)
	}
	if s.Status != constant.Connected {
		log.Errorf("会话状态不正确: %v", s.Status)
		disConnectNewSession(ws, AssetNotActive, "会话离线")
		return FailWithDataOperate(c, 400, "会话离线", "主机审计-监控: 监控主机, 失败原因[会话离线]", err)
	}
	connectionId := s.ConnectionId
	configuration := guacd.NewConfiguration()
	configuration.ConnectionID = connectionId
	sessionId = s.ID
	configuration.SetParameter("width", strconv.Itoa(s.Width))
	configuration.SetParameter("height", strconv.Itoa(s.Height))
	configuration.SetParameter("dpi", "96")

	addr := config.GlobalCfg.Guacd.Hostname + ":" + strconv.Itoa(config.GlobalCfg.Guacd.Port)
	asset := fmt.Sprintf("%s:%s", configuration.GetParameter("hostname"), configuration.GetParameter("port"))
	log.Debugf("[%v] 新建 guacd 会话, guacd=%v, asset=%v", sessionId, addr, asset)

	guacdTunnel, err := guacd.NewTunnel(addr, configuration)
	if err != nil {
		operateLog.Result = "失败"
		operateLog.LogContents = "主机审计-监控: 监控主机, 失败原因[新建 guacd 会话失败]"
		err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
		if nil != err {
			log.Errorf("编写操作日志失败: %v", err)
		}
		disConnectNewSession(ws, NewTunnelError, err.Error())
		log.Printf("[%v] 建立连接失败: %v", sessionId, err.Error())
		return err
	}

	nextSession := &session.Session{
		ID:          sessionId,
		Protocol:    s.Protocol,
		WebSocket:   ws,
		GuacdTunnel: guacdTunnel,
	}

	if s.Protocol == "x11" {
		nextSession.Protocol = constant.RDP
	}

	// 要监控会话
	forObsSession := session.GlobalSessionManager.GetById(sessionId)
	if forObsSession == nil {
		operateLog.Result = "失败"
		operateLog.LogContents = "主机审计-监控: 监控主机, 失败原因[会话不存在]"
		err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
		if nil != err {
			log.Errorf("编写操作日志失败: %v", err)
		}
		disConnectNewSession(ws, NotFoundSession, "获取会话失败")
		return nil
	}
	nextSession.ID = utils.UUID()
	forObsSession.Observer.Add(nextSession)
	log.Debugf("[%v:%v] 观察者[%v]加入会话[%v]", sessionId, connectionId, nextSession.ID, s.ConnectionId)

	guacamoleHandler := NewTunnelHandler(ws, guacdTunnel)
	guacamoleHandler.Start()
	defer guacamoleHandler.Stop()

	operateLog.Result = "成功"
	operateLog.LogContents = "主机审计-监控: 监控主机, 成功"
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("编写操作日志失败: %v", err)
	}

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Debugf("[%v:%v] WebSocket已关闭, %v", sessionId, connectionId, err.Error())
			// guacdTunnel.Read() 会阻塞，所以要先把guacdTunnel客户端关闭，才能退出Guacd循环
			_ = guacdTunnel.Close()

			observerId := nextSession.ID
			forObsSession.Observer.Del(observerId)
			log.Debugf("[%v:%v] 观察者[%v]退出会话", sessionId, connectionId, observerId)
			return nil
		}
		_, err = guacdTunnel.WriteAndFlush(message)
		if err != nil {
			NewCloseSessionById(sessionId, TunnelClosed, "远程连接已关闭", false)
			return nil
		}
	}
}

// AppSessionTunnelMonitor 应用监控
func AppSessionTunnelMonitor(c echo.Context) error {
	ws, err := UpGrader.Upgrade(c.Response().Writer, c.Request(), nil)
	if err != nil {
		log.Errorf("升级为WebSocket协议失败:%v", err.Error())
		return err
	}
	sessionId := c.Param("id")

	s, err := appSessionRepository.GetById(context.TODO(), sessionId)
	if err != nil {
		return err
	}
	if s.Status != constant.Connected {
		disConnectNewSession(ws, AssetNotActive, "会话离线")
		return nil
	}
	connectionId := s.ConnectionId
	configuration := guacd.NewConfiguration()
	configuration.ConnectionID = connectionId
	sessionId = s.ID
	configuration.SetParameter("width", strconv.Itoa(s.Width))
	configuration.SetParameter("height", strconv.Itoa(s.Height))
	configuration.SetParameter("dpi", "96")

	addr := config.GlobalCfg.Guacd.Hostname + ":" + strconv.Itoa(config.GlobalCfg.Guacd.Port)
	asset := fmt.Sprintf("%s:%s", configuration.GetParameter("hostname"), configuration.GetParameter("port"))
	log.Debugf("[%v] 新建 guacd 会话, guacd=%v, asset=%v", sessionId, addr, asset)

	user, _ := GetCurrentAccountNew(c)
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "运维日志",
		Created:         utils.NowJsonTime(),
		Users:           user.Username,
		Names:           user.Nickname,
	}

	guacdTunnel, err := guacd.NewTunnel(addr, configuration)
	if err != nil {
		operateLog.Result = "失败"
		operateLog.LogContents = "应用审计-监控: 监控应用, 失败原因[新建 guacd 会话失败]"
		disConnectNewSession(ws, NewTunnelError, err.Error())
		log.Printf("[%v] 建立连接失败: %v", sessionId, err.Error())
		return err
	}

	nextSession := &session.Session{
		ID:          sessionId,
		Protocol:    constant.RDP,
		WebSocket:   ws,
		GuacdTunnel: guacdTunnel,
	}

	// 要监控会话
	forObsSession := session.GlobalSessionManager.GetById(sessionId)
	if forObsSession == nil {
		disConnectNewSession(ws, NotFoundSession, "获取会话失败")
		return nil
	}
	nextSession.ID = utils.UUID()
	forObsSession.Observer.Add(nextSession)
	log.Debugf("[%v:%v] 观察者[%v]加入会话[%v]", sessionId, connectionId, nextSession.ID, s.ConnectionId)

	guacamoleHandler := NewTunnelHandler(ws, guacdTunnel)
	guacamoleHandler.Start()
	defer guacamoleHandler.Stop()

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Debugf("[%v:%v] WebSocket已关闭, %v", sessionId, connectionId, err.Error())
			// guacdTunnel.Read() 会阻塞，所以要先把guacdTunnel客户端关闭，才能退出Guacd循环
			_ = guacdTunnel.Close()

			observerId := nextSession.ID
			forObsSession.Observer.Del(observerId)
			log.Debugf("[%v:%v] 观察者[%v]退出会话", sessionId, connectionId, observerId)
			return nil
		}
		_, err = guacdTunnel.WriteAndFlush(message)
		if err != nil {
			NewCloseSessionById(sessionId, TunnelClosed, "远程连接已关闭", true)
			return nil
		}
	}
}

func setAppSessionConfig(s model.AppSession, configuration *guacd.Configuration) {
	enableRecord, err := propertyRepository.FindByName(guacd.EnableRecording)
	if err != nil {
		log.Errorf("获取启用录像配置失败: %v", err.Error())
	}
	if enableRecord.Value == "true" {
		configuration.SetParameter(guacd.RecordingPath, path.Join(config.GlobalCfg.Guacd.Recording, s.ID))
		configuration.SetParameter(guacd.CreateRecordingPath, "true")
	}

	pp, err := newApplicationRepository.GetApplicationForSessionByProgramId(context.TODO(), s.ProgramId)
	if err != nil {
		log.Errorf("获取应用[%v]的凭证失败: %v", s.AppName, err.Error())
		return
	}
	configuration.Protocol = constant.RDP
	configuration.SetParameter("username", s.PassPort)
	configuration.SetParameter("password", pp.Password)
	configuration.SetParameter(guacd.RemoteApp, "||"+pp.Name)
	if pp.Param != "" && pp.Param != "-" {
		configuration.SetParameter(guacd.RemoteAppArgs, pp.Param)
	}
	configuration.SetParameter(guacd.RemoteAppDir, pp.Path)
	configuration.SetParameter("security", "any")
	configuration.SetParameter("ignore-cert", "true")
	configuration.SetParameter("create-drive-path", "true")
	configuration.SetParameter("resize-method", "display-update")
}

func setX11SessionConfig(s model.Session, x11Id string, configuration *guacd.Configuration) {
	enableRecord, err := propertyRepository.FindByName(guacd.EnableRecording)
	if err != nil {
		log.Errorf("获取启用录像配置失败: %v", err.Error())
	}
	if enableRecord.Value == "true" {
		configuration.SetParameter(guacd.RecordingPath, path.Join(config.GlobalCfg.Guacd.Recording, s.ID))
		configuration.SetParameter(guacd.CreateRecordingPath, "true")
	}

	ap, err := newAssetRepository.GetPassportWithPasswordById(context.TODO(), s.PassportId)
	if err != nil {
		log.Errorf("获取主机[%v]的凭证失败: %v", s.AssetName, err.Error())
		return
	}
	pp, err := newApplicationRepository.GetApplicationForSessionByProgram(context.TODO(), x11Id)
	if err != nil {
		log.Errorf("获取应用凭证失败: %v", err.Error())
		return
	}

	configuration.Protocol = constant.RDP
	configuration.SetParameter("hostname", pp.IP)
	configuration.SetParameter("port", strconv.Itoa(pp.Port))
	configuration.SetParameter("username", pp.Passport)
	configuration.SetParameter("password", pp.Password)
	configuration.SetParameter(guacd.RemoteApp, "||"+pp.Name)
	configuration.SetParameter("security", "any")
	configuration.SetParameter("ignore-cert", "true")
	configuration.SetParameter("create-drive-path", "true")
	configuration.SetParameter("resize-method", "display-update")

	cpu, err := repository.AssetNewDao.GetPassportConfig(context.TODO(), s.PassportId)
	if err != nil {
		log.Errorf("获取主机[%v]的凭证失败: %v", s.AssetName, err.Error())
		return
	}

	for k, v := range cpu {
		if v == x11Id {
			if k == "x11_term_program" {
				configuration.SetParameter(guacd.RemoteAppArgs, "-ssh -X -l "+ap.Passport+" -pw "+ap.Password+" -P "+strconv.Itoa(ap.Port)+" "+ap.Ip)
			}
			break
		}
	}

}

func setNewSessionConfig(s model.Session, configuration *guacd.Configuration) {
	enableRecord, err := propertyRepository.FindByName(guacd.EnableRecording)
	if err != nil {
		log.Errorf("获取启用录像配置失败: %v", err.Error())
	}
	if enableRecord.Value == "true" {
		configuration.SetParameter(guacd.RecordingPath, path.Join(config.GlobalCfg.Guacd.Recording, s.ID))
		configuration.SetParameter(guacd.CreateRecordingPath, "true")
	}

	pp, err := newAssetRepository.GetPassportWithPasswordById(context.TODO(), s.PassportId)
	if err != nil {
		log.Errorf("获取资产[%v]的凭证失败: %v", s.PassportId, err.Error())
		return
	}

	configuration.Protocol = pp.Protocol
	switch configuration.Protocol {
	case "rdp":
		configuration.SetParameter("username", pp.Passport)
		configuration.SetParameter("password", pp.Password)
		configuration.SetParameter("security", "any")
		configuration.SetParameter("ignore-cert", "true")
		configuration.SetParameter("create-drive-path", "true")
		configuration.SetParameter("resize-method", "display-update")
		advanced, err := newAssetRepository.GetPassportConfig(context.TODO(), s.PassportId)
		if err != nil {
			log.Errorf("获取资产[%v]的配置失败: %v", s.PassportId, err.Error())
		}
		if v, ok := advanced["rdp_domain"]; ok {
			configuration.SetParameter(guacd.Domain, v)
		}
		if _, ok := advanced["rdp_enable_drive"]; ok {
			if vv, ok := advanced["rdp_drive_path"]; ok {
				configuration.SetParameter(guacd.EnableDrive, "true")
				configuration.SetParameter(guacd.DriveName, "Tkbastion FileSystem")
				configuration.SetParameter(guacd.DrivePath, path.Join(config.GlobalCfg.Guacd.Drive, vv))
			}
		}
	case "ssh":
		//if len(s.KeyFile) > 0 && s.KeyFile != "-" {
		//	configuration.SetParameter("username", s.Username)
		//	configuration.SetParameter("private-key", s.KeyFile)
		//	configuration.SetParameter("passphrase", s.Passphrase)
		//} else {
		configuration.SetParameter("username", pp.Passport)
		configuration.SetParameter("password", pp.Password)
		//}
	case "vnc":
		configuration.SetParameter("username", pp.Passport)
		configuration.SetParameter("password", pp.Password)
		// 配置vnc的编码
		configuration.SetParameter("clipboard-encoding", "utf-8")
	case "telnet":
		configuration.SetParameter("username", pp.Passport)
		configuration.SetParameter("password", pp.Password)
	default:

	}
}

func disConnectNewSession(ws *websocket.Conn, code int, reason string) {
	// guacd 无法处理中文字符，所以进行了base64编码。
	encodeReason := base64.StdEncoding.EncodeToString([]byte(reason))
	err := guacd.NewInstruction("error", encodeReason, strconv.Itoa(code))
	_ = ws.WriteMessage(websocket.TextMessage, []byte(err.String()))
	disconnect := guacd.NewInstruction("disconnect")
	_ = ws.WriteMessage(websocket.TextMessage, []byte(disconnect.String()))
}
