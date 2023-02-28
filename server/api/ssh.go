package api

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"tkbastion/pkg/config"
	"tkbastion/pkg/constant"
	"tkbastion/pkg/global"
	"tkbastion/pkg/global/session"
	"tkbastion/pkg/global/work_order"
	"tkbastion/pkg/guacd"
	"tkbastion/pkg/log"
	"tkbastion/pkg/terminal"
	"tkbastion/server/model"
	"tkbastion/server/utils"
	"unicode/utf8"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var UpGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	Subprotocols: []string{"guacamole"},
}

const (
	Closed          = 0
	Connected       = 1
	Data            = 2
	Resize          = 3
	Ping            = 4
	CommandBreak    = 5  //命令阻断
	CommandOrdering = 6  //命令审批中
	CommodOk        = 7  //审批通过
	OutTime         = 8  //申请超时
	Cancel          = 9  //取消申请
	NeedApproval    = 10 //需要审批
	Approval        = 11 //审批
)

type Message struct {
	Type    int    `json:"type"`
	Content string `json:"content"`
}

func NewMessage(_type int, content string) Message {
	return Message{Content: content, Type: _type}
}

func (r Message) ToString() string {
	if r.Content != "" {
		return strconv.Itoa(r.Type) + r.Content
	} else {
		return strconv.Itoa(r.Type) + "-"
	}
}

func ParseMessage(value string) (message Message, err error) {
	if value == "" {
		return
	}

	_type, err := strconv.Atoi(value[:1])
	if err != nil {
		return
	}
	var content = value[1:]
	message = NewMessage(_type, content)
	return
}

type WindowSize struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

type policyNameRule struct {
	Name string
	Rule string
}

func WriteMessage(ws *websocket.Conn, msg Message) error {
	message := []byte(msg.ToString())
	return ws.WriteMessage(websocket.TextMessage, message)
}

//func ReadMessage(ws *websocket.Conn) (code int, msg Message, err error) {
//	_, message, err := ws.ReadMessage()
//	if err != nil {
//		return
//	}
//	msg, err = ParseMessage(string(message))
//	if err != nil {
//		return
//	}
//	code = msg.Type
//	return
//}

func NewSSHEndpoint(c echo.Context) (err error) {
	ws, err := UpGrader.Upgrade(c.Response().Writer, c.Request(), nil)
	if err != nil {
		log.Errorf("升级为WebSocket协议失败: %v", err.Error())
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}
	defer func(ws *websocket.Conn) {
		err := ws.Close()
		if err != nil {
			log.Errorf("关闭WebSocket连接失败: %v", err.Error())
		}
	}(ws)

	sessionId := c.QueryParam("sessionId")
	cols, _ := strconv.Atoi(c.QueryParam("cols"))
	rows, _ := strconv.Atoi(c.QueryParam("rows"))
	s, err := newSessionRepository.GetById(context.TODO(), sessionId)
	if nil != err {
		wsErr := WriteMessage(ws, NewMessage(Closed, "获取会话失败"))
		if nil != wsErr {
			log.Errorf("WriteMessage Error: %v", wsErr)
		} else {
			log.Errorf("获取会话失败: %v", err)
		}
		return FailWithDataOperate(c, 500, "会话不存在", "", err)
	}

	var (
		CommandPermission  []utils.CommandPolicyContent
		CommandApplication []utils.CommandPolicyContent
		CommandDeny        []utils.CommandPolicyContent
		SessionDeny        []utils.CommandPolicyContent
	)

	user, _ := GetCurrentAccountNew(c)
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "运维日志",
		Created:         utils.NowJsonTime(),
		Users:           user.Username,
		Names:           user.Nickname,
	}
	passportInfo, err := newAssetRepository.GetPassPortByID(context.TODO(), s.PassportId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}

	//获取命令审批规则
	CommandPermission, CommandApplication, CommandDeny, SessionDeny, err = commandControlService.FindCommandContentByPolicy(passportInfo.ID, user.ID)
	if nil != err {
		log.Errorf("获取命令阻断规则失败: %v", err)
	}

	fmt.Println(CommandPermission, CommandApplication, CommandDeny, SessionDeny)

	pw, err := newAssetRepository.GetPassportWithPasswordById(context.TODO(), s.PassportId)
	if nil != err {
		log.Errorf("获取账号密码失败: %v", err)
	}

	approveTimeout, err := propertyRepository.FindByName("approval-timeout")
	if err != nil {
		log.Errorf("获取审批超时时间失败: %v", err)
	}

	var (
		username   = s.PassPort
		pass       = pw.Password
		ip         = s.AssetIP
		port       = s.AssetPort
		privateKey = path.Join(constant.PrivateKeyPath, pw.PrivateKey)
		passphrase = pw.Passphrase
	)

	if v, ok := global.PasswdStore[sessionId]; ok {
		pass = v.Password
		username = v.Passport
	}
	delete(global.PasswdStore, sessionId)

	if pw.IsSshKey == 0 {
		privateKey = ""
		passphrase = ""
	} else {
		pass = ""
	}

	recording := ""
	var isRecording = false
	property, err := propertyRepository.FindByName(guacd.EnableRecording)
	if err == nil && property.Value == "true" {
		isRecording = true
	}
	if isRecording {
		// 存储录像地址及录像名称
		recording = path.Join(config.GlobalCfg.Guacd.Recording, sessionId, "recording.cast")
	}

	var xterm = "xterm-256color"
	// 封装stdin、stdout
	Terminal, err := terminal.NewTerminal(
		ip, port, username, pass, privateKey, passphrase, rows, cols, recording, xterm, true,
	)
	if nil != err {
		log.Errorf("Error: %v", err)
		return WriteMessage(ws, NewMessage(Closed, "请检查登录主机的凭证或服务器配置是否正确，若问题仍存在，请联系技术人员查看系统日志以确定其他可能的原因"))
	}
	// 建立本地终端与远程主机连接
	if err := Terminal.RequestPty(xterm, rows, cols); err != nil {
		log.Errorf("RequestPty Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}
	// 远程主机开启一个登录shell
	if err := Terminal.Shell(); err != nil {
		log.Errorf("Shell Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}

	sess := model.Session{
		ConnectionId: sessionId,
		Width:        cols,
		Height:       rows,
		Status:       constant.Connecting,
		Recording:    recording,
	}
	// 创建新会话
	log.Debugf("创建新会话 %v", sess.ConnectionId)
	if err := newSessionRepository.Update(context.TODO(), &sess, sessionId); nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}
	// 操作日志
	operateLog.Result = "成功"
	operateLog.LogContents = "主机运维-登录: 登录资产: " + s.AssetIP + ":" + strconv.Itoa(s.AssetPort) + "(" + passportInfo.AssetName + ")"
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}

	if err := WriteMessage(ws, NewMessage(Connected, "")); err != nil {
		log.Errorf("WriteMessage Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}

	sessionTerminal := &session.Session{
		ID:          s.ID,
		Protocol:    s.Protocol,
		Mode:        constant.Naive,
		WebSocket:   ws,
		GuacdTunnel: nil,
		Terminal:    Terminal,
		Observer:    session.NewObserver(s.ID),
	}
	// 建立会话后,此会话属于全局会话中的一员
	session.GlobalSessionManager.Add(sessionTerminal)

	ctx, cancel := context.WithCancel(context.Background())
	tick := time.NewTicker(time.Millisecond * time.Duration(60))
	defer tick.Stop()

	// 创建记录执行命令及时间的文件
	// 之所以放在此处执行创建文件操作是因为上边代码中的terminal.NewTerminal部分已经帮我们做好了目录是否存在检查，权限等工作，此处可成功创建文件
	var commandRecordFile *os.File
	commandRecordFile, err = os.Create(path.Join(config.GlobalCfg.Guacd.Recording, sessionId, "command.txt"))
	if err != nil {
		log.Errorf("Create Error: %v", err)
	}
	defer func(sessionId string) {
		err := commandStrategyRepository.CreateCommandRecord(sessionId)
		if err != nil {
			log.Errorf("CreateCommandRecord Error: %v", err)
		}
	}(sessionId)
	defer func(commandRecordFile *os.File) {
		err := commandRecordFile.Close()
		if err != nil {
			log.Errorf("Close commandRecordFile Error: %v", err)
		}
	}(commandRecordFile)

	// 上下键有问题，录屏文件中记录的是这些字符，  我们目前过滤时string中不是这些字符，需研究，录屏是怎样以这种方式记录的。byte->变成了成这样。
	// 而且这种实现方式还是有问题，比如这次命令是cd /sh，按下上键后变成 cd /share但第二次服务器只返回are，我们用are覆盖command后，command为are是错误的(正确的为cd /share)
	// 所以还是想办法解决掉第一种思路遇到的问题
	//upDownReg := regexp.MustCompile(`\\u0007|\\u0008|\\u001b].|\\u001b\[[A-Z]|\\u001b\[.{2}`)
	//if nil == upDownReg {
	//	log.Errorf("MustCompile Error. 命令阻断策略失效!")
	//}

	var buf []byte
	var command string
	var clientString string            // 保存从js客户端的ws读取的字符串,与服务器回显字符串比较,若相同,则表示此字符串为可显示字符串
	var tabFromServerSyncString string // 按下tab键进行命令补全时，保存服务器返回的字符串，补全至原有command中
	//var upDownFromServerSyncString string // 按下"上"、"下"按键时，根据服务器回显的信息覆盖command(无论有无历史命令，无的话如果按键后是空则覆盖( TODO 此处存在问题，如果历史命令按键后为空，但原有命令还在，command被覆盖为空了，不准确))
	var historyArr []string
	isNeedGetHistory := true // 首次接入时，获取历史命令输入记录，配合按下上、下键时特殊情况处理、以及切换用户后也许获取一次history
	getHistoryChan := make(chan bool)
	clientWsBegin := false
	var recordCommand string
	//upDownKey := false
	//isDisplayableChan := make(chan bool)
	//isNeedCompareDisplayFromServer := false
	isNeedSyncChan := make(chan bool)
	isNeedSyncFromServer := false
	mouseIndex := 0 // 下一个要插入字符的位置索引(鼠标所在位置索引)
	dataChan := make(chan rune)
	historyArrIndex := 0 // 历史命令记录索引
	// 从远程服务器读,读取到内容发送至dataChan
	go func() {
	SshLoop:
		for {
			select {
			case <-ctx.Done():
				log.Debugf("WebSocket已关闭，即将关闭SSH连接...")
				break SshLoop
			default:
				// 从远程服务器读,读取为UTF-8编码的Unicode字符
				r, size, err := Terminal.StdoutReader.ReadRune()
				if err != nil {
					log.Debugf("SSH 读取失败，即将退出循环...")
					_ = WriteMessage(ws, NewMessage(Closed, ""))
					break SshLoop
				}
				if size > 0 {
					dataChan <- r
				}
			}
		}
		log.Debugf("SSH 连接已关闭，退出循环。")
	}()

	// 定时将服务器内容写入客户端、监控者ws、录屏文件
	go func() {
	tickLoop:
		for {
			select {
			case <-ctx.Done():
				break tickLoop
			case data := <-dataChan:
				if data != utf8.RuneError {
					p := make([]byte, utf8.RuneLen(data))
					utf8.EncodeRune(p, data)
					buf = append(buf, p...)
				} else {
					buf = append(buf, []byte("@")...)
				}
			case <-tick.C:
				if len(buf) > 0 {
					s := string(buf)
					//fmt.Println("\nserverString:", strings.TrimSpace(strings.Replace(s, "\u0008", "", -1)))
					//fmt.Println("\nLen:", len(strings.TrimSpace(strings.Replace(s, "\u0008", "", -1))))
					if isNeedGetHistory && clientWsBegin {
						tmpArr := strings.Split(s, "\n")
						if len(tmpArr) > 1 {
							tmpArr = tmpArr[1 : len(tmpArr)-1]
						}
						for _, v := range tmpArr {
							if len(historyArr) < 7 {
								continue
							}
							historyArr = append(historyArr, v[7:])
						}
						getHistoryChan <- true
						isNeedGetHistory = false
						buf = []byte{}
						goto tickLoop
					}
					if isNeedSyncFromServer && clientWsBegin {
						// 按下tab有换行符肯定没补全，如果没有换行存在两种情况:1.命令已补全 2.没有可补全命令
						if !strings.Contains(s, "\n") {
							// 有时补全命令时，服务器回显的可见字符个数与s个数不相等，因此需过滤掉不可见字符
							var newTabFromServerSyncString string
							for i := range s {
								if strconv.IsPrint(rune(s[i])) {
									newTabFromServerSyncString += string(s[i])
								}
							}

							tabFromServerSyncString = newTabFromServerSyncString
							isNeedSyncChan <- true
						} else {
							isNeedSyncChan <- false
						}

						isNeedSyncFromServer = false
					}
					// 写入录屏记录文件
					if isRecording {
						_ = Terminal.Recorder.WriteData(s)
					}
					// 向所有监视会话的WS写数据
					sessionTerminal.Observer.Range(
						func(key string, ob *session.Session) {
							_ = WriteMessage(ob.WebSocket, NewMessage(Data, s))
							log.Debugf("[%v] 强制踢出会话的观察者: %v", sessionId, ob.ID)
						},
					)
					// 写入ws,流向此次连接js客户端
					if err := WriteMessage(ws, NewMessage(Data, s)); err != nil {
						log.Debugf("WebSocket写入失败，即将退出循环...")
						cancel()
					}
					buf = []byte{}
					//fmt.Println("here6", s, "\n")
				}
			}
		}
		log.Debugf("SSH 连接已关闭，退出定时器循环。")
	}()

	// 从js客户端的ws读，并根据数据类型分别探测远程连接是否存活，发送数据至远程服务器，调整窗口大小
	for {
		// 从websocket读，此处为server代码，因此是通过ws从js客户端读
		_, message, err := ws.ReadMessage()
		if err != nil {
			// web socket会话关闭后主动关闭ssh会话
			log.Debugf("WebSocket已关闭")
			NewCloseSessionById(sessionId, Normal, "用户正常退出", false)
			cancel()
			break
		}

		msg, err := ParseMessage(string(message))
		if err != nil {
			log.Warnf("消息解码失败: %v, 原始字符串:%v", err, string(message))
			continue
		}

		switch msg.Type {
		case Resize:
			decodeString, err := base64.StdEncoding.DecodeString(msg.Content)
			if err != nil {
				log.Warnf("Base64解码失败: %v，原始字符串:%v", err, msg.Content)
				continue
			}
			var winSize WindowSize
			err = json.Unmarshal(decodeString, &winSize)
			if err != nil {
				log.Warnf("解析SSH会话窗口大小失败: %v，原始字符串:%v", err, msg.Content)
				continue
			}
			if err := Terminal.WindowChange(winSize.Rows, winSize.Cols); err != nil {
				log.Warnf("更改SSH会话窗口大小失败: %v", err)
			}
			_ = newSessionRepository.UpdateWindowSizeById(context.TODO(), winSize.Rows, winSize.Cols, sessionId)
		case Data:
			clientWsBegin = true
			//获取历史命令，处理"上"、"下"特殊按键
			if isNeedGetHistory {
				err := getHistory(Terminal)
				if nil != err {
					NewCloseSessionById(sessionId, TunnelClosed, "远程连接已关闭", false)
				}
				<-getHistoryChan
			}
			//fmt.Println("\n\n\n\nhistorylen:", len(historyArr))
			input := []byte(msg.Content)
			clientString = msg.Content
			hexInput := hex.EncodeToString(input)
			//fmt.Println("\n\n16:", hexInput)
			switch hexInput {
			case "0d", "0a": // 回车ctrl+j / 回车ctrl+m
				// 审批的话，在此处，如果触发了命令阻断策略，则考虑如何加入审批功能
				// 审批（等待）、不审批（自动阻断）
				var result bool
				if f, r := utils.Matching(SessionDeny, command); f {
					fmt.Println("触发会话拦截策略")
					input, _ = hex.DecodeString("03")

					var alarmLogResult string
					var text string
					text += "用户名: " + user.Username + ";\n"
					text += "来源地址: " + c.RealIP() + ";\n"
					text += "设备名称: " + passportInfo.AssetName + ";\n"
					text += "设备地址: " + passportInfo.Ip + ";\n"
					text += "登录账号: " + passportInfo.Passport + ";\n"
					text += "登录协议: " + passportInfo.Protocol + ";\n"
					text += "事件描述: " + "使用命令: " + command + " 时,触发会话阻断;\n"
					text += "触发策略: " + r.CommandStrategyName + ";\n"
					text += "操作时间: " + utils.NowJsonTime().Format("2006-01-02 15:04:05") + ";\n"
					if r.IsMessage {
						err := messageService.SendAdminMessage("操作告警", r.Level, text, constant.AlertMessage)
						if err != nil {
							alarmLogResult += "消息告警失败 "
							log.Errorf("发送告警消息失败: %v", err)
						}
						alarmLogResult += "消息告警成功 "
					} else {
						alarmLogResult += "未配置发送告警消息 "
					}
					mailText := `<br>用户:` + user.Username + `在` + `使用命令: ` + command + ` 时,触发会话阻断,详细信息如下：`
					mailText += `<br>用户名: ` + user.Username +
						`<br>来源地址: ` + c.RealIP() +
						`<br>设备名称: ` + passportInfo.AssetName +
						`<br>设备地址: ` + passportInfo.Ip +
						`<br>登录账号: ` + passportInfo.Passport +
						`<br>登录协议: ` + passportInfo.Protocol +
						`<br>事件描述: ` + `使用命令: ` + command + ` 时,触发会话阻断` +
						`<br>触发策略: ` + r.CommandStrategyName +
						`<br>操作时间: ` + utils.NowJsonTime().Format("2006-01-02 15:04:05")
					if r.IsEmail {
						err := sysConfigService.SendMail([]string{user.Mail}, "操作告警", mailText)
						if err != nil {
							alarmLogResult += "邮件告警失败 "
							log.Errorf("发送告警邮件失败: %v", err)
						}
						alarmLogResult += "邮件告警成功 "
					} else {
						alarmLogResult += "未配置发送告警邮件 "
					}

					err := operateAlarmLogRepository.Create(context.TODO(), &model.OperateAlarmLog{
						AlarmTime:         utils.NowJsonTime(),
						ClientIP:          c.RealIP(),
						UserId:            user.ID,
						PassportId:        passportInfo.ID,
						AssetId:           passportInfo.AssetId,
						Content:           "危险命令: " + command + ", 会话已断开",
						Level:             r.Level,
						CommandStrategyId: r.CommandStrategyId,
						Result:            alarmLogResult,
					})
					if err != nil {
						log.Errorf("创建告警日志失败: %v", err)
					}

					session.GlobalSessionManager.Del(sessionId)
				}
				if f, r := utils.Matching(CommandDeny, command); f {
					fmt.Println("触发命令拦截策略")
					input, _ = hex.DecodeString("03")
					if err := WriteMessage(
						ws, NewMessage(CommandBreak, "危险命令: "+command+",   触发阻断!"),
					); err != nil {
						log.Debugf("WebSocket写入失败，即将退出循环...")
						cancel()
					}
					var alarmLogResult string
					var text string
					text += "用户名: " + user.Username + ";\n"
					text += "来源地址: " + c.RealIP() + ";\n"
					text += "设备名称: " + passportInfo.AssetName + ";\n"
					text += "设备地址: " + passportInfo.Ip + ";\n"
					text += "登录账号: " + passportInfo.Passport + ";\n"
					text += "登录协议: " + passportInfo.Protocol + ";\n"
					text += "事件描述: " + "使用命令: " + command + " 时,触发命令阻断;\n"
					text += "触发策略: " + r.CommandStrategyName + ";\n"
					text += "操作时间: " + utils.NowJsonTime().Format("2006-01-02 15:04:05") + ";\n"
					if r.IsMessage {
						err := messageService.SendAdminMessage("操作告警", r.Level, text, constant.AlertMessage)
						if err != nil {
							alarmLogResult += "消息告警失败"
							log.Errorf("发送告警消息失败: %v", err)
						}
						alarmLogResult += "消息告警成功 "
					} else {
						alarmLogResult += "未配置发送告警消息 "
					}
					mailText := `<br>用户:` + user.Username + `在` + `使用命令: ` + command + ` 时,触发命令阻断,详细信息如下：`
					mailText += `<br>用户名: ` + user.Username +
						`<br>来源地址: ` + c.RealIP() +
						`<br>设备名称: ` + passportInfo.AssetName +
						`<br>设备地址: ` + passportInfo.Ip +
						`<br>登录账号: ` + passportInfo.Passport +
						`<br>登录协议: ` + passportInfo.Protocol +
						`<br>事件描述: ` + `使用命令: ` + command + ` 时,触发命令阻断` +
						`<br>触发策略: ` + r.CommandStrategyName +
						`<br>操作时间: ` + utils.NowJsonTime().Format("2006-01-02 15:04:05")
					if r.IsEmail {
						err := sysConfigService.SendMail([]string{user.Mail}, "操作告警", mailText)
						if err != nil {
							alarmLogResult += "邮件告警失败 "
							log.Errorf("发送告警邮件失败: %v", err)
						}
						alarmLogResult += "邮件告警成功 "
					} else {
						alarmLogResult += "未配置发送告警邮件 "
					}

					err := operateAlarmLogRepository.Create(context.TODO(), &model.OperateAlarmLog{
						AlarmTime:         utils.NowJsonTime(),
						ClientIP:          c.RealIP(),
						UserId:            user.ID,
						PassportId:        passportInfo.ID,
						AssetId:           passportInfo.AssetId,
						Content:           "危险命令: " + command + ",   审批超时!",
						Level:             r.Level,
						CommandStrategyId: r.CommandStrategyId,
						Result:            alarmLogResult,
					})
					if err != nil {
						log.Errorf("创建告警日志失败: %v", err)
					}

					result = true
					goto label
				}
				//如果白名单为空则忽略
				if len(CommandPermission) > 0 {
					f, r := utils.Mismatch(CommandPermission, command)
					if f {
						fmt.Println("触发命令权限策略")
						input, _ = hex.DecodeString("03")
						if err := WriteMessage(
							ws, NewMessage(CommandBreak, "危险命令: "+command+",   触发阻断!"),
						); err != nil {
							log.Debugf("WebSocket写入失败，即将退出循环...")
							cancel()
						}

						var alarmLogResult string
						var text string
						text += "用户名: " + user.Username + ";\n"
						text += "来源地址: " + c.RealIP() + ";\n"
						text += "设备名称: " + passportInfo.AssetName + ";\n"
						text += "设备地址: " + passportInfo.Ip + ";\n"
						text += "登录账号: " + passportInfo.Passport + ";\n"
						text += "登录协议: " + passportInfo.Protocol + ";\n"
						text += "事件描述: " + "使用命令:" + command + " 时,触发命令阻断;\n"
						text += "触发策略: " + r.CommandStrategyName + ";\n"
						text += "操作时间: " + utils.NowJsonTime().Format("2006-01-02 15:04:05") + ";\n"
						if r.IsMessage {
							err := messageService.SendAdminMessage("操作告警", r.Level, text, constant.AlertMessage)
							if err != nil {
								alarmLogResult += "消息告警失败 "
								log.Errorf("发送告警消息失败: %v", err)
							}
							alarmLogResult += "消息告警成功 "
						} else {
							alarmLogResult += "未配置发送告警消息 "
						}
						mailText := `<br>用户:` + user.Username + `在` + `使用命令: ` + command + ` 时,触发命令阻断,详细信息如下：`
						mailText += `<br>用户名: ` + user.Username +
							`<br>来源地址: ` + c.RealIP() +
							`<br>设备名称: ` + passportInfo.AssetName +
							`<br>设备地址: ` + passportInfo.Ip +
							`<br>登录账号: ` + passportInfo.Passport +
							`<br>登录协议: ` + passportInfo.Protocol +
							`<br>事件描述: ` + `使用命令: ` + command + ` 时,触发命令阻断;` +
							`<br>触发策略: ` + r.CommandStrategyName +
							`<br>操作时间: ` + utils.NowJsonTime().Format("2006-01-02 15:04:05")
						if r.IsEmail {
							err := sysConfigService.SendMail([]string{user.Mail}, "操作告警", mailText)
							if err != nil {
								alarmLogResult += "邮件告警失败 "
								log.Errorf("发送告警邮件失败: %v", err)
							}
							alarmLogResult += "邮件告警成功 "
						} else {
							alarmLogResult += "未配置发送告警邮件 "
						}

						err := operateAlarmLogRepository.Create(context.TODO(), &model.OperateAlarmLog{
							AlarmTime:         utils.NowJsonTime(),
							ClientIP:          c.RealIP(),
							UserId:            user.ID,
							PassportId:        passportInfo.ID,
							AssetId:           passportInfo.AssetId,
							Content:           "危险命令: " + command + ",   触发阻断!",
							Level:             r.Level,
							CommandStrategyId: r.CommandStrategyId,
							Result:            alarmLogResult,
						})
						if err != nil {
							log.Errorf("创建告警日志失败: %v", err)
						}

						result = true
						goto label
					}
				}
				if f, r := utils.Matching(CommandApplication, command); f {
					// 告诉前端，需要审批
					//fmt.Println("触发命令审批策略")
					//if err := WriteMessage(
					//	ws, NewMessage(NeedApproval, "危险命令: "+command+",   需要审批!"),
					//); err != nil {
					//	log.Debugf("WebSocket写入失败，即将退出循环...")
					//	cancel()
					//}
					//// 等待前端用户是否进行审批
					//code, _, err := ReadMessage(ws)
					//if err != nil {
					//	log.Infof("WebSocket读取失败，即将退出循环...")
					//	cancel()
					//}
					//fmt.Println("审批结果:", code)
					// 等待审批结果
					// 1. 审批通过，继续执行
					// 2. 审批不通过，阻断
					// 3. 审批超时，阻断
					//if code == Approval {
					if err := WriteMessage(
						ws, NewMessage(CommandOrdering, "危险命令: "+command+",   审核中!"),
					); err != nil {
						log.Infof("WebSocket写入失败，即将退出循环...")
						cancel()
					}
					// 发起命令审批工单
					ip := c.RealIP()
					order := work_order.NewOrder(&passportInfo, &user, ip, command)

					var text string
					text += "用户名: " + user.Username + ";\n"
					text += "来源地址: " + c.RealIP() + ";\n"
					text += "设备名称: " + passportInfo.AssetName + ";\n"
					text += "设备地址: " + passportInfo.Ip + ";\n"
					text += "登录账号: " + passportInfo.Passport + ";\n"
					text += "登录协议: " + passportInfo.Protocol + ";\n"
					text += "事件描述: " + "使用命令: " + command + " 时,触发命令审批;\n"
					text += "触发策略: " + r.CommandStrategyName + ";\n"
					text += "操作时间: " + utils.NowJsonTime().Format("2006-01-02 15:04:05") + ";\n"
					if r.IsMessage {
						err := messageService.SendAdminMessage("命令审批", r.Level, text, constant.WorkOrderMessage)
						if err != nil {
							log.Errorf("发送告警消息失败: %v", err)
						}
					}
					mailText := `<br>用户:` + user.Username + `在` + `使用命令: ` + command + ` 时,触发命令审批,详细信息如下：`
					mailText += `<br>用户名: ` + user.Username +
						`<br>来源地址: ` + c.RealIP() +
						`<br>设备名称: ` + passportInfo.AssetName +
						`<br>设备地址: ` + passportInfo.Ip +
						`<br>登录账号: ` + passportInfo.Passport +
						`<br>登录协议: ` + passportInfo.Protocol +
						`<br>事件描述: ` + `使用命令: ` + command + ` 时,触发命令审批` +
						`<br>触发策略: ` + r.CommandStrategyName +
						`<br>操作时间: ` + utils.NowJsonTime().Format("2006-01-02 15:04:05")

					if r.IsEmail {
						err := sysConfigService.SendMail([]string{user.Mail}, "命令审批", mailText)
						if err != nil {
							log.Errorf("发送告警邮件失败: %v", err)
						}
					}

					work_order.WorkOrderChannel <- order
					tl := utils.String2Int(approveTimeout.Value)
					t := time.NewTicker(time.Minute * time.Duration(tl))

					var status string
					select {
					case <-t.C:
						pro, err := propertyRepository.FindByName("expired-action")
						if err != nil {
							log.Errorf("获取超时处理方式失败: %v", err)
							input, _ = hex.DecodeString("03")
						} else {
							if err := workOrderApprovalLogRepository.UpdateByWorkOrderIdAndDepartmentId(order.ID, 1, user.DepartmentId, &model.WorkOrderApprovalLog{
								ApprovalUsername: user.Username,
								ApprovalNickname: user.Nickname,
								ApprovalId:       user.ID,
								ApprovalDate:     utils.NowJsonTime(),
								Result:           "已超时",
								ApprovalInfo:     "审批超时",
							}); err != nil {
								log.Errorf("更新审批日志失败: %v", err)
							}
							// 更新工单状态
							if err := workOrderNewRepository.UpdateByOrderId(order.ID, &model.NewWorkOrder{
								Status:      "已超时",
								ApproveUser: user.Username,
								ApproveId:   user.ID,
							}); err != nil {
								log.Errorf("工单状态更新失败: %v", err)
							}
							olg, err := workOrderNewRepository.FindByOrderId(order.ID)
							if err != nil {
								log.Errorf("查询工单失败: %v", err)
							}
							// 更新工单审批记录
							if err := workOrderNewRepository.CreateWorkOrderLog(&olg, &user, "超时", "指令工单", ""); err != nil {
								log.Errorf("工单审批记录更新失败: %v", err)
							}

							input, _ = hex.DecodeString("03")
							status = "已超时"
							log.Errorf("命令审批工单超时: %v", order.ID)
							if err := WriteMessage(
								ws, NewMessage(OutTime, "危险命令: "+command+",   审批超时!"),
							); err != nil {
								log.Debugf("WebSocket写入失败，即将退出循环...")
								cancel()
							}
							if pro.Value == "session-deny" {
								session.GlobalSessionManager.Del(sessionId)
							}
						}
						result = true
					case orderStatus := <-order.Status:
						if orderStatus == constant.Approved {
							log.Errorf("命令审批工单通过: %v", order.ID)
							status = "已通过"
							if err := WriteMessage(
								ws, NewMessage(CommodOk, "危险命令: "+command+",   审批通过!"),
							); err != nil {
								log.Debugf("WebSocket写入失败，即将退出循环...")
								cancel()
							}
							result = false
						} else if orderStatus == constant.Rejected {
							log.Errorf("命令审批工单拒绝: %v", order.ID)
							input, _ = hex.DecodeString("03")
							if err := WriteMessage(
								ws, NewMessage(CommandBreak, "危险命令: "+command+",   触发阻断!"),
							); err != nil {
								log.Debugf("WebSocket写入失败，即将退出循环...")
								cancel()
							}
							status = "已驳回"
							result = true
						} else {
							log.Errorf("命令审批工单取消: %v", order.ID)
							input, _ = hex.DecodeString("03")
							if err := WriteMessage(
								ws, NewMessage(Cancel, "危险命令: "+command+",   取消申请!"),
							); err != nil {
								log.Debugf("WebSocket写入失败，即将退出循环...")
								cancel()
							}
							status = "已取消"
							result = true
						}

						//更新order结构体，并将审核日志写入数据库
						order, _ = work_order.GetOrder(order.ID)
						orderLog := model.OrderLog{
							ID:              utils.UUID(),
							Asset:           order.Asset.AssetName,
							IP:              order.IP,
							Approved:        order.Approved,
							Applicant:       order.Applicant.Username,
							Information:     "命令审批: " + order.Command,
							Status:          status,
							Created:         order.Time,
							ApproveTime:     utils.NowJsonTime(),
							ApplicationType: "命令审批",
						}

						work_order.RemoveOrder(order.ID)
						if err := orderLogRepository.Create(&orderLog); err != nil {
							return FailWithDataOperate(c, 500, "审批日志写入数据库失败", "", err)
						}
					}
					// 发送告警信息
					//sendCommandBreakMail(policyNameRuleArr[i].Name, policyNameRuleArr[i].Rule, command, c.RealIP(), passportInfo.Name, passportInfo.IP, user.Username)
					//} else {
					//	// 不申请则直接阻断
					//	input, _ = hex.DecodeString("03")
					//}
				}
				// 这里记录时，之后可备注此命令被阻断、审批通过等
			label:
				if command != "" {
					now := int(time.Now().Unix())
					sessionRecord, err := newSessionRepository.GetById(context.TODO(), sessionId)
					if err != nil {
						log.Errorf("获取session记录失败: %v", err)
					}
					delta := now - int(sessionRecord.ConnectedTime.Unix())
					hour := delta / 3600
					min := (delta - hour*3600) / 60
					second := delta - hour*3600 - min*60

					hourStr := strconv.Itoa(hour)
					minStr := strconv.Itoa(min)
					secondStr := strconv.Itoa(second)
					if hour < 10 {
						hourStr = "0" + hourStr
					}
					if min < 10 {
						minStr = "0" + minStr
					}
					if second < 10 {
						secondStr = "0" + secondStr
					}
					//fmt.Println("delta:", delta, "hour:", hour, "min:", min, "second:", second, "hourStr:", hourStr, "minStr:", minStr, "SecondStr:", secondStr)
					_, err = commandRecordFile.WriteString(hourStr + ":" + minStr + ":" + secondStr + "\n")
					if nil != err {
						log.Warnf("命令写入文件错误 WriteString Error: %v", err)
					}
					_, err = commandRecordFile.WriteString(command + "\n")
					if nil != err {
						log.Warnf("命令写入文件错误 WriteString Error: %v", err)
					}
				}

				//命令没有被阻断才被计入数组中
				if !result && command != "" {
					// !数字 指令特殊处理，数字有效则不计入历史命令数组，若是! 11 中途有空格，则计入历史命令数组
					// 单纯！需要计入历史命令数组
					if len(command) > 1 && command[0] == '!' {
						//特殊处理 ! 后是数字的情况
						if command[1] >= '0' && command[1] <= '9' || command[1] == '-' {
							tempNum, _ := strconv.ParseInt(command[1:], 10, 0)
							if tempNum > 0 && tempNum <= int64(len(historyArr)) { //数值为正，且有效
								command = historyArr[tempNum-1]
							} else if tempNum < 0 && tempNum >= -int64(len(historyArr)) { //数值为负，且有效
								command = historyArr[int64(len(historyArr))+tempNum]
							} else {
								command = historyArr[len(historyArr)-1]
							}
						}

						//特殊处理 ! 后是 ? + 指令 的情况 (!? + string,指执行上一条包含string的指令)
						if command[1] == '?' {
							for i := len(historyArr) - 1; i >= 0; i-- {
								if strings.Contains(historyArr[i], command[2:]) {
									command = historyArr[i]
									break
								}
							}
						}
						//特殊处理 ! 后是 字符指令 的情况 (! + string,指执行上一条以string开头的指令)
						if command[1] >= 'a' && command[1] <= 'z' || command[1] >= 'A' && command[1] <= 'Z' {
							for i := len(historyArr) - 1; i >= 0; i-- {
								if strings.HasPrefix(historyArr[i], command[1:]) {
									command = historyArr[i]
									break
								}
							}
						}
						//特殊处理 ！！情况
						if command[1] == '!' {
							command = historyArr[len(historyArr)-1]
						}

					}

					if len(historyArr) > 0 {
						//fmt.Println("=======",historyArr[len(historyArr)-1],"=======")
						//fmt.Println("=======",command,"=======")
						var tempStr1 string
						for i := range historyArr[len(historyArr)-1] {
							if strconv.IsPrint(rune(historyArr[len(historyArr)-1][i])) {
								tempStr1 += string(historyArr[len(historyArr)-1][i])
							}
						}
						var tempStr2 string
						for i := range command {
							if strconv.IsPrint(rune(command[i])) {
								tempStr2 += string(command[i])
							}
						}
						//fmt.Printf("******%s******\n", tempStr)
						//fmt.Printf("++++++%s++++++\n",command)
						if strings.Compare(tempStr1, tempStr2) != 0 {
							historyArr = append(historyArr, command)
						}
					} else {
						historyArr = append(historyArr, command)
					}
					// history -c 指令特殊处理，清空历史数组
					if strings.Contains(command, "history ") && strings.Contains(command, "-c") {
						historyArr = []string{}
					}
					// history -d  数字 指令特殊处理，删除特定指令更新数组，多个数字时只删除第一个（有效数字）
					if strings.Contains(command, "history ") && strings.Contains(command, "-d") {
						var index int
						command = strings.Trim(command, " ")
						tempArr := strings.Split(command, " ")
						for i := range tempArr {
							//fmt.Printf("tempArr[%d]:%s\n",i,tempArr[i])
							index_, _ := strconv.ParseInt(tempArr[i], 10, 0)
							index = int(index_)
							//fmt.Printf("---index:%d\n",index)
							if index != 0 {
								break
							}
						}
						//fmt.Println("-----index:",index)
						if index > 0 && index < len(historyArr) {
							tmpArr := historyArr[index:]
							historyArr = historyArr[:index-1] //删除指定index的命令
							historyArr = append(historyArr, tmpArr...)
						}
					}
				}
				//fmt.Println("\nhistoryLen:", len(historyArr))
				//for _, v := range historyArr {
				//	fmt.Println(v)
				//}
				mouseIndex = 0
				command = ""
				historyArrIndex = len(historyArr)
				//isNeedCompareDisplayFromServer = false
				//fmt.Println("enter here")
			case "7f", "08": // backspace  || ctrl + h
				if 0 != mouseIndex {
					mouseIndex--
					command = command[:mouseIndex] + command[mouseIndex+1:] // 前包后不包
				}
				//isNeedCompareDisplayFromServer = false
			case "0c": // ctrl+l
				//isNeedCompareDisplayFromServer = false
			case "1b5b43": // ->
				if len(command) != mouseIndex {
					mouseIndex++
				}
				//isNeedCompareDisplayFromServer = false
			case "1b5b44": // <-
				if 0 != mouseIndex {
					mouseIndex--
				}
				//isNeedCompareDisplayFromServer = false
			case "15": // ctrl+u
				command = command[mouseIndex:]
				mouseIndex = 0
				//isNeedCompareDisplayFromServer = false
			case "03", "1b72": // ctrl+c  || Alt + r 清空历史命令
				mouseIndex = 0
				command = ""
				//isNeedCompareDisplayFromServer = false
				//
			case "1b5b41", "10": //上 || ctrl+p
				if historyArrIndex > 0 {
					historyArrIndex--
					command = historyArr[historyArrIndex]
				} else {
					command = recordCommand
				}

				mouseIndex = len(command)
			//isNeedCompareDisplayFromServer = false
			//upDownKey = true
			//isNeedSyncFromServer = true
			case "1b5b42", "0e": //下  || ctrl + n
				if historyArrIndex < len(historyArr)-1 {
					historyArrIndex++
					command = historyArr[historyArrIndex]
				} else {
					command = recordCommand
				}
				mouseIndex = len(command)
			//isNeedCompareDisplayFromServer = false
			//upDownKey = true
			//isNeedSyncFromServer = true
			case "09": // tab  || ctrl + i
				//isNeedCompareDisplayFromServer = false
				isNeedSyncFromServer = true
			case "12": // ctrl + r 执行后需更新历史数组值，暂未实现

			default:
				//isNeedCompareDisplayFromServer = true
				for i := range clientString {
					if strconv.IsPrint(rune(clientString[i])) {
						recordCommand += string(clientString[i])
					}
				}
				if "" != recordCommand {
					command = command[:mouseIndex] + recordCommand + command[mouseIndex:]
					mouseIndex += len(recordCommand)
					recordCommand = ""
				}
			}
			_, err := Terminal.Write(input)
			if err != nil {
				NewCloseSessionById(sessionId, TunnelClosed, "远程连接已关闭", false)
			}
			//之所以需要判断是否显示是因为，有时client获取到的字符(Fn+F3等)不会在服务器显示，有时服务器显示的又从client获取不到(Fn+F7)因此需综合判断
			//if isNeedCompareDisplayFromServer && <-isDisplayableChan {
			//	fmt.Println("here4")
			//	command = command[:mouseIndex] + clientString + command[mouseIndex:]
			//	mouseIndex += len(clientString)
			//} else if isNeedSyncFromServer && <-isNeedSyncChan {
			if isNeedSyncFromServer && <-isNeedSyncChan {
				//fmt.Println("!!!!!!!!!")
				//if upDownKey {
				//	fmt.Println("here")
				//	upDownKey = false
				//	command = upDownFromServerSyncString
				//	mouseIndex = len(upDownFromServerSyncString)
				//	fmt.Println("mouseIndex:", mouseIndex)
				//} else {
				command = command[:mouseIndex] + tabFromServerSyncString + command[mouseIndex:]
				mouseIndex += len(tabFromServerSyncString)
				//}
			}
			//fmt.Println("command:  ", command, "\n")
		case Ping:
			_, _, err := Terminal.SshClient.Conn.SendRequest("helloworld1024@foxmail.com", true, nil)
			if err != nil {
				NewCloseSessionById(sessionId, TunnelClosed, "远程连接已关闭", false)
			} else {
				_ = WriteMessage(ws, NewMessage(Ping, ""))
			}

		}
	}
	return err
}

func CreateTerminalBySession(session model.Session) (*terminal.Terminal, error) {
	pp, err := newAssetRepository.GetPassportWithPasswordById(context.TODO(), session.PassportId)
	if err != nil {
		return nil, err
	}
	var (
		username   = pp.Passport
		pw         = pp.Password
		ip         = session.AssetIP
		port       = session.AssetPort
		privatekey = path.Join(constant.PrivateKeyPath, pp.PrivateKey)
		passphrase = pp.Passphrase
	)
	return terminal.NewTerminal(ip, port, username, pw, privatekey, passphrase, 10, 10, "", "", false)
}

func NewSshMonitor(c echo.Context) error {
	ws, err := UpGrader.Upgrade(c.Response().Writer, c.Request(), nil)
	if err != nil {
		log.Errorf("升级为WebSocket协议失败:%v", err.Error())
		return FailWithDataOperate(c, 500, "监控失败", "", err)
	}
	defer func(ws *websocket.Conn) {
		err := ws.Close()
		if err != nil {
			log.Errorf("关闭WebSocket连接失败:%v", err.Error())
		}
	}(ws)

	user, _ := GetCurrentAccountNew(c)
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "运维日志",
		Created:         utils.NowJsonTime(),
		Users:           user.Username,
		Names:           user.Nickname,
	}

	sessionId := c.QueryParam("sessionId")
	s, err := newSessionRepository.GetById(context.TODO(), sessionId)
	if err != nil {
		log.Errorf("DB Error: %v", err.Error())
		return WriteMessage(ws, NewMessage(Closed, "获取会话失败"))
	}

	sessionTerminal := session.GlobalSessionManager.GetById(sessionId)
	if sessionTerminal == nil {
		operateLog.Result = "失败"
		operateLog.LogContents = "主机审计-监控: " + s.AssetIP + " 监控失败, 会话不存在"
		err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
		if nil != err {
			log.Errorf("DB Error: %v", err)
		}
		log.Warnf("会话已离线")
		return WriteMessage(ws, NewMessage(Closed, "会话已离线"))
	}

	obId := utils.UUID()
	obSession := &session.Session{
		ID:        obId,
		Protocol:  s.Protocol,
		WebSocket: ws,
		Mode:      constant.Naive,
	}

	sessionTerminal.Observer.Add(obSession)
	log.Debugf("会话 %v 观察者 %v 进入", sessionId, obId)

	// 记录操作日志
	operateLog.Result = "成功"
	operateLog.LogContents = "主机审计-监控: " + "监控会话: " + s.AssetIP + ":" + strconv.Itoa(s.AssetPort)
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Debugf("会话 %v 观察者 %v 退出", sessionId, obId)
			sessionTerminal.Observer.Del(obId)
			break
		}
	}
	return nil
}

// GetSshCommand 获取命令信息
func GetSshCommand(c echo.Context) error {
	// 创建一个返回格式
	type Response struct {
		// 命令
		Command string `json:"command"`
		// 时间
		Time string `json:"time"`
	}

	var rsp []Response

	_, l := GetCurrentAccountNew(c)
	if !l {
		return Fail(c, 403, "用户未登录")
	}

	id := c.QueryParam("id")
	auto := c.QueryParam("auto")
	_, err := newSessionRepository.GetById(context.TODO(), id)
	if err != nil {
		return Success(c, rsp)
	}
	stat, err := os.Stat(path.Join(config.GlobalCfg.Guacd.Recording, id, "command.txt"))
	if err != nil {
		return Success(c, rsp)
	}
	// 读取文件
	f, err := os.Open(path.Join(config.GlobalCfg.Guacd.Recording, id, "command.txt"))
	if err != nil {
		return Success(c, rsp)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Errorf("关闭文件失败:%v", err.Error())
		}
	}(f)
	// 读取文件内容
	content, err := io.ReadAll(f)
	if err != nil {
		return Success(c, rsp)
	}
	if string(content) == "" {
		return Success(c, rsp)
	}
	// 通过换行符分割
	commands := strings.Split(string(content), "\n")
	fmt.Println(commands)

	rsp = make([]Response, len(commands)/2)
	// 遍历命令
	//var command string
	//var commandTime string
	for i := 0; i < len(commands); i++ {
		if i%2 == 1 {
			rsp[i/2].Command = commands[i]
		} else {
			if i == len(commands)-1 {
				break
			}
			// 通过:分割
			times := strings.Split(commands[i], ":")
			if len(times) != 3 {
				return Success(c, rsp)
			}
			// 获取时间
			hour, err := strconv.Atoi(times[0])
			if err != nil {
				return Success(c, rsp)
			}
			minute, err := strconv.Atoi(times[1])
			if err != nil {
				return Success(c, rsp)
			}
			second, err := strconv.Atoi(times[2])
			if err != nil {
				return Success(c, rsp)
			}
			rsp[i/2].Time = stat.ModTime().Add(time.Hour*time.Duration(hour) + time.Minute*time.Duration(minute) + time.Second*time.Duration(second)).Format("2006-01-02 15:04:05")
		}
	}

	for i := 0; i < len(rsp); i++ {
		if strings.Contains(rsp[i].Command, auto) || strings.Contains(rsp[i].Time, auto) {
			continue
		} else {
			rsp = append(rsp[:i], rsp[i+1:]...)
			i--
		}
	}

	// 记录操作日志
	//user, _ := GetCurrentAccountNew(c)
	//operateLog := model.OperateLog{
	//	Ip:              c.RealIP(),
	//	ClientUserAgent: c.Request().UserAgent(),
	//	LogTypes:        "运维日志",
	//	Created:         utils.NowJsonTime(),
	//	Users:           user.Username,
	//	Result:          "成功",
	//	LogContents:     "主机审计-命令: 获取会话命令: " + session.AssetName + "[" + session.AssetIP + ":" + strconv.Itoa(session.AssetPort) + "]",
	//}
	//err = env.GetDB().Model(model.OperateLog{}).Create(&operateLog).Error
	//if nil != err {
	//	log.Errorf("DB Error: %v", err)
	//}

	return Success(c, rsp)
}

func getHistory(Terminal *terminal.Terminal) error {
	input := []byte("history\n")
	_, err := Terminal.Write(input)
	return err
}

func sendCommandBreakMail(policyName, policyRule, excuteCommand, ip, assetName, assetIp, userName string) {
	propertiesMap := propertyRepository.FindAllMap()
	host := propertiesMap[constant.MailHost]
	port := propertiesMap[constant.MailPort]
	username := propertiesMap[constant.MailUsername]
	password := propertiesMap[constant.MailPassword]
	receiver := propertiesMap[constant.MailReceiver]
	if host == "" || host == "-" || port == "" || port == "-" || username == "" || username == "-" || password == "" || password == "-" || receiver == "" || receiver == "-" {
		log.Warningln("触发命令阻断策略: " + time.Now().Format("2006-01-02 15:04:05") + " 用户" + userName + ", 来源IP" + ip + ", 运维资产:" + assetName + "(" + assetIp + ")" + ", 触发的策略为" + policyName + ", 策略规则为" + policyRule + ", 执行的命令为" + excuteCommand + ". 因系统设置中邮箱配置信息不全, 发送告警邮件失败.")
		return
	}

	err := mailService.NewSendMail(
		host, port, username, password, []string{receiver}, "[Tkbastion] 命令阻断告警",
		"触发命令阻断策略: "+time.Now().Format("2006-01-02 15:04:05")+" 用户"+userName+", 来源IP"+ip+", 运维资产:"+assetName+"("+assetIp+")"+", 触发的策略为"+policyName+", 策略规则为"+policyRule+", 执行的命令为"+excuteCommand+".",
	)
	if nil != err {
		log.Warning("触发命令阻断策略: " + time.Now().Format("2006-01-02 15:04:05") + " 用户" + userName + ", 来源IP" + ip + ", 运维资产:" + assetName + "(" + assetIp + ")" + ", 触发的策略为" + policyName + ", 策略规则为" + policyRule + ", 执行的命令为" + excuteCommand + ". 发送告警邮件失败.")
		log.Errorf("NewSendMail Error: %v", err)
	}
}
