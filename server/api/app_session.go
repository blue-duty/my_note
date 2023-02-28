package api

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"tkbastion/pkg/constant"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"gorm.io/gorm"

	"github.com/labstack/echo/v4"
)

func AppSessionHistoryEndpoint(c echo.Context) error {
	var ss dto.AppSessionForSearch
	ss.AppName = c.QueryParam("appName")
	ss.Auto = c.QueryParam("auto")
	ss.Program = c.QueryParam("program")
	ss.Passport = c.QueryParam("passport")
	ss.UserName = c.QueryParam("userName")
	ss.IP = c.QueryParam("ip")
	ss.NickName = c.QueryParam("nickName")

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

	resp, err := appSessionRepository.GetHistorySession(context.TODO(), ss)
	if err != nil {
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	return Success(c, resp)
}

func AppSessionExportEndpoint(c echo.Context) error {
	u, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	var departmentIds []int64
	err := GetChildDepIds(u.DepartmentId, &departmentIds)
	if err != nil {
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}

	assetForExport, err := appSessionRepository.GetExportSession(context.TODO(), departmentIds)
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
	assetHeaderForExport := []string{"开始时间", "结束时间", "用户名", "姓名", "来源地址", "应用名称", "应用程序", "应用账号"}
	assetFileNameForExport := "应用历史会话"
	file, err := utils.CreateExcelFile(assetFileNameForExport, assetHeaderForExport, assetStringsForExport)
	if err != nil {
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	fileName := "应用历史会话.xlsx"

	// 记录操作日志
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		Result:          "成功",
		LogTypes:        "用户日志",
		Names:           u.Nickname,
		Created:         utils.NowJsonTime(),
		LogContents:     "应用审计-导出: 导出应用历史会话",
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

func AppSessionDetailEndpoint(c echo.Context) error {
	sessionId := c.Param("id")
	s, err := appSessionRepository.GetDetailSession(context.TODO(), sessionId)
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

func MarkAppAsRead(c echo.Context) error {
	id := c.QueryParam("ids")

	u, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithDataOperate(c, 401, "未登录", "", nil)
	}

	if id == "all" {
		err := appSessionRepository.MarkAllAsRead(context.TODO())
		if err != nil {
			return FailWithDataOperate(c, 500, "全部标为已阅失败", "", nil)
		}
		return SuccessWithOperate(c, "应用审计-标为已阅: 用户["+u.Username+"]全部标为已阅", nil)
	}

	ids := strings.Split(id, ",")
	err := appSessionRepository.MarkAsRead(context.TODO(), ids)
	if err != nil {
		return FailWithDataOperate(c, 500, "标为已阅失败", "", nil)
	}
	return SuccessWithOperate(c, "应用审计-标记已阅: 用户["+u.Username+"]标记已阅", nil)
}

func MarkAppAsUnread(c echo.Context) error {
	id := c.QueryParam("ids")
	ids := strings.Split(id, ",")

	u, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithDataOperate(c, 401, "未登录", "", nil)
	}
	err := appSessionRepository.MarkAsUnRead(context.TODO(), ids)
	if err != nil {
		return FailWithDataOperate(c, 500, "标记失败", "", nil)
	}
	return SuccessWithOperate(c, "应用审计-标记未阅: 用户["+u.Username+"]标记未阅", nil)
}

func AppSessionReplayEndpoint(c echo.Context) error {
	sessionId := c.Param("id")
	var recording string
	s, err := appSessionRepository.GetById(context.TODO(), sessionId)
	if err == gorm.ErrRecordNotFound {
		return FailWithDataOperate(c, 404, "会话不存在", "应用审计-回放: 会话不存在", err)
	} else if err == nil {
		recording = s.Recording + "/recording"
	} else {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "回放失败", "", err)
	}

	log.Debugf("读取录屏文件: %v, 是否存在: %v, 是否为文件: %v", recording, utils.FileExists(recording), utils.IsFile(recording))
	user, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithDataOperate(c, 401, "回放失败", "应用审计-回放: 用户未登录", err)
	}

	if !utils.FileExists(recording) || !utils.IsFile(recording) {
		return FailWithDataOperate(c, 404, "回放失败", "应用审计-回放: 用户["+user.Username+"], 回放应用["+s.AppName+"]会话, 会话录屏文件不存在", err)
	}

	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "审计日志",
		Created:         utils.NowJsonTime(),
		Users:           user.Username,
		Names:           user.Nickname,
		Result:          "成功",
		LogContents:     "应用审计-回放: 用户[" + user.Username + "], 回放应用[" + s.AppName + "]会话",
	}
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}

	return c.File(recording)
}

func AppSessionVideoDownloadEndpoint(c echo.Context) error {
	sessionId := c.QueryParam("id")
	s, err := appSessionRepository.GetById(context.TODO(), sessionId)
	if err != nil {
		return FailWithDataOperate(c, 500, "下载失败", "", err)
	}

	u, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithDataOperate(c, 500, "下载失败", "", err)
	}

	video := func() {
		err := appSessionRepository.UpdateVideoDownload(context.TODO(), s.ID, 1)
		if err != nil {
			log.Errorf("DB Error: %v", err)
		}
		commod := "guacenc  -s 1280x720 -r 20000000 " + s.Recording + "/recording"
		_, err = utils.ExecShell(commod)
		if err != nil {
			log.Errorf("执行命令失败: %v", err)
			_ = appSessionRepository.UpdateVideoDownload(context.TODO(), s.ID, 0)
			_ = messageService.SendUserMessage(u.ID, "会话下载", "", "录像[应用名称: "+s.AppName+", 来源地址: "+s.ClientIP+", 登陆用户: "+s.CreateName+", 录像时间:"+s.CreateTime.Format("2006-01-02 15:04:05")+"]解码失败，请联系管理员获取错误信息", constant.NoticeMessage)
		} else {
			_ = messageService.SendUserMessage(u.ID, "会话下载", "", "录像[应用名称: "+s.AppName+", 来源地址: "+s.ClientIP+", 登陆用户: "+s.CreateName+", 录像时间:"+s.CreateTime.Format("2006-01-02 15:04:05")+"]解码成功，可以下载", constant.NoticeMessage)
			err = appSessionRepository.UpdateVideoDownload(context.TODO(), s.ID, 2)
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
	}
	if s.DownloadStatus == 0 {
		go video()
		return SuccessWithData(c, 201, "录像开始解码", nil)
	} else if s.DownloadStatus == 1 {
		return SuccessWithData(c, 202, "录像正在解码中，请稍后重新点击进行下载", nil)
	} else {
		_, err := os.Stat(s.Recording + "/recording.m4v")
		if err != nil {
			go video()
			operateLog := model.OperateLog{
				Ip:              c.RealIP(),
				ClientUserAgent: c.Request().UserAgent(),
				LogTypes:        "审计日志",
				Created:         utils.NowJsonTime(),
				Result:          "失败",
				Users:           u.Username,
				Names:           u.Nickname,
				LogContents:     "应用审计-下载会话: 用户[" + u.Username + "], 下载应用[" + s.AppName + "]的会话录像失败, 原因[录像文件不存在]",
			}
			err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
			if nil != err {
				log.Errorf("DB Error: %v", err)
				_ = appSessionRepository.UpdateVideoDownload(context.TODO(), s.ID, 0)
				return FailWithDataOperate(c, 500, "下载失败", "", err)
			}
			return SuccessWithData(c, 203, "文件丢失，正在为您重新解码，解码成功后会在站内消息中通知您", nil)
		}
		operateLog := model.OperateLog{
			Ip:              c.RealIP(),
			ClientUserAgent: c.Request().UserAgent(),
			LogTypes:        "审计日志",
			Created:         utils.NowJsonTime(),
			Users:           u.Username,
			Names:           u.Nickname,
			Result:          "成功",
			LogContents:     "应用审计-下载会话: 用户[" + u.Username + "], 下载应用[" + s.AppName + "]的会话录像",
		}
		err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "下载失败", "", err)
		}
	}

	return c.Attachment(s.Recording+"/recording.m4v", "recording.m4v")
}
