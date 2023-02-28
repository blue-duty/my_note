package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strings"
	"tkbastion/pkg/config"
	"tkbastion/pkg/constant"
	"tkbastion/pkg/file_connect"
	"tkbastion/pkg/global"
	"tkbastion/pkg/global/file_session"
	"tkbastion/server/model"
	"tkbastion/server/service"
	"tkbastion/server/utils"

	"github.com/jlaffaye/ftp"

	"github.com/pkg/sftp"

	"github.com/labstack/gommon/log"

	"tkbastion/server/repository"

	"github.com/labstack/echo/v4"
)

func NewSftpEndpoint(c echo.Context) error {
	id := c.QueryParam("sessionId")

	session, err := repository.SessionRepo.GetById(context.Background(), id)
	if err != nil {
		return FailWithDataOperate(c, 500, "连接失败", "", nil)
	}

	user, _ := GetCurrentAccountNew(c)
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "运维日志",
		Created:         utils.NowJsonTime(),
		Users:           user.Username,
		Names:           user.Nickname,
	}

	sftpClient, dir, err := file_connect.NewSftpClientBySession(session)
	if err != nil {
		operateLog.Result = "失败"
		operateLog.LogContents = "主机运维-登陆: 失败原因[" + err.Error() + "]"
		err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
		if nil != err {
			log.Errorf("记录操作日志失败: %v", err.Error())
		}
		return FailWithDataOperate(c, 500, "连接失败", "", nil)
	}

	// 创建目录保存录像
	if err := os.MkdirAll(path.Join(config.GlobalCfg.Guacd.Recording, id), 0755); err != nil {
		log.Error("创建录像目录失败", err)
	}

	sess := model.Session{
		ConnectionId: session.ConnectionId,
		Width:        0,
		Height:       0,
		Status:       constant.Connecting,
		Recording:    path.Join(config.GlobalCfg.Guacd.Recording, id),
	}
	if err := newSessionRepository.Update(context.TODO(), &sess, id); err != nil {
		log.Error("更新会话状态失败", err)
	}

	file_session.FileSessions.Set(id, &file_session.FileSession{
		ID:       id,
		Protocol: constant.SFTP,
		Dir:      dir,
		Client:   sftpClient,
	})

	operateLog.Result = "成功"
	operateLog.LogContents = "主机运维-登陆: 主机[" + session.AssetName + "], 账号[" + session.PassPort + "]"
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("记录操作日志失败: %v", err.Error())
	}

	return Success(c, nil)
}

func NewFtpEndpoint(c echo.Context) error {
	id := c.QueryParam("sessionId")

	session, err := repository.SessionRepo.GetById(context.Background(), id)
	if err != nil {
		return FailWithDataOperate(c, 500, "连接失败", "", nil)
	}

	user, _ := GetCurrentAccountNew(c)
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "运维日志",
		Created:         utils.NowJsonTime(),
		Users:           user.Username,
		Names:           user.Nickname,
	}

	ftpClient, err := file_connect.NewFtpClientBySession(session)
	if err != nil {
		operateLog.Result = "失败"
		operateLog.LogContents = "主机运维-登陆: 失败原因[" + err.Error() + "]"
		err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
		if nil != err {
			log.Errorf("记录操作日志失败: %v", err.Error())
		}
		return FailWithDataOperate(c, 500, "连接失败", "", nil)
	}

	sess := model.Session{
		ConnectionId: session.ConnectionId,
		Width:        0,
		Height:       0,
		Status:       constant.Connecting,
		Recording:    path.Join(config.GlobalCfg.Guacd.Recording, id),
	}
	if err := newSessionRepository.Update(context.TODO(), &sess, id); err != nil {
		log.Error("更新会话状态失败", err)
	}

	file_session.FileSessions.Set(id, &file_session.FileSession{
		ID:       id,
		Protocol: constant.FTP,
		Dir:      "/",
		Client:   ftpClient,
	})

	operateLog.Result = "成功"
	operateLog.LogContents = "主机运维-登陆: 主机[" + session.AssetName + "], 账号[" + session.PassPort + "]"
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("记录操作日志失败: %v", err.Error())
	}

	return Success(c, nil)
}

func FileSessionLsEndpoint(c echo.Context) error {
	id := c.Param("id")
	p := c.FormValue("dir")
	fileSession := file_session.FileSessions.Get(id)
	if fileSession == nil {
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}

	if fileSession.Client == nil {
		session, err := repository.SessionRepo.GetById(context.Background(), id)
		if err != nil {
			return FailWithDataOperate(c, 500, "操作失败", "", nil)
		}
		if fileSession.Protocol == constant.SFTP {
			sftpClient, dir, err := file_connect.NewSftpClientBySession(session)
			if err != nil {
				return FailWithDataOperate(c, 500, "操作失败", "", nil)
			}
			fileSession.Client = sftpClient
			fileSession.Dir = dir
		} else if fileSession.Protocol == constant.FTP {
			ftpClient, err := file_connect.NewFtpClientBySession(session)
			if err != nil {
				return FailWithDataOperate(c, 500, "操作失败", "", nil)
			}
			fileSession.Client = ftpClient
			fileSession.Dir = "/"
		}
	}

	if fileSession.Protocol == constant.SFTP {
		if p == "/" {
			p = fileSession.Dir
		}
		// 如果目录不是规定的目录，就不允许访问
		if !strings.HasPrefix(p, fileSession.Dir) {
			return FailWithDataOperate(c, 500, "当前目录无权限查看", "", nil)
		}
		fileInfos, err := fileSession.Client.(*sftp.Client).ReadDir(p)
		if err != nil {
			return err
		}

		var files = make([]service.File, 0, len(fileInfos))
		for i := range fileInfos {
			file := service.File{
				Name:    fileInfos[i].Name(),
				Path:    path.Join(p, fileInfos[i].Name()),
				IsDir:   fileInfos[i].IsDir(),
				Mode:    fileInfos[i].Mode().String(),
				IsLink:  fileInfos[i].Mode()&os.ModeSymlink == os.ModeSymlink,
				ModTime: utils.NewJsonTime(fileInfos[i].ModTime()),
				Size:    fileInfos[i].Size(),
			}
			files = append(files, file)
		}
		return Success(c, H{
			"files": files,
			"dir":   p,
		})
	} else if fileSession.Protocol == constant.FTP {
		if p == "" {
			p = fileSession.Dir
		}
		fileInfos, err := fileSession.Client.(*ftp.ServerConn).List(p)
		if err != nil {
			return err
		}

		var files = make([]service.File, 0, len(fileInfos))
		for i := range fileInfos {
			file := service.File{
				Name:    fileInfos[i].Name,
				Path:    path.Join(p, fileInfos[i].Name),
				IsDir:   fileInfos[i].Type == ftp.EntryTypeFolder,
				Mode:    fileInfos[i].Type.String(),
				IsLink:  fileInfos[i].Type == ftp.EntryTypeLink,
				ModTime: utils.NewJsonTime(fileInfos[i].Time),
				Size:    int64(fileInfos[i].Size),
			}
			files = append(files, file)
		}
		return Success(c, H{
			"files": files,
			"dir":   p,
		})
	} else {
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}
}

func FileSessionDownloadEndpoint(c echo.Context) error {
	id := c.Param("id")
	f := c.QueryParam("file")

	fs, err := repository.SessionRepo.GetById(context.Background(), id)
	if err != nil {
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}
	if fs.Download == 0 {
		return FailWithDataOperate(c, 403, "无权限下载", "", nil)
	}

	fileSession := file_session.FileSessions.Get(id)
	if fileSession == nil {
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}

	filenameWithSuffix := path.Base(f)
	if fileSession.Protocol == constant.SFTP {
		// 如果目标文件不在规定的目录下，不允许下载
		if !strings.HasPrefix(f, fileSession.Dir) {
			return FailWithDataOperate(c, 403, "无权限下载", "", nil)
		}
		// 保存一份到本地
		sftpFileForLocal, err := fileSession.Client.(*sftp.Client).Open(f)
		if err != nil {
			log.Errorf("打开文件失败: %v", err.Error())
			return FailWithDataOperate(c, 500, "操作失败", "", nil)
		}
		if _, err := os.Stat(path.Join(config.GlobalCfg.Guacd.Recording, id)); os.IsNotExist(err) {
			err := os.MkdirAll(path.Join(config.GlobalCfg.Guacd.Recording, id), 0777)
			if err != nil {
				log.Error("创建本地文件夹失败", err)
			}
		}

		localFilePath := path.Join(config.GlobalCfg.Guacd.Recording, id, utils.UUID()+"."+filenameWithSuffix)
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
		_, err = io.Copy(localFile, sftpFileForLocal)
		if err != nil {
			log.Error("复制文件失败", err)
		}
		log.Infof("文件保存本地成功: %v", localFilePath)

		// 创建文件操作记录
		stat, err := fileSession.Client.(*sftp.Client).Stat(f)
		if err != nil {
			log.Error("获取文件信息失败", err)
		}
		var fileRecord = &model.FileRecord{
			SessionId:  id,
			FileName:   filenameWithSuffix,
			FilePath:   localFilePath,
			FileSize:   stat.Size(),
			Action:     constant.DOWNLOAD,
			CreateTime: utils.NowJsonTime(),
		}
		err = repository.SessionRepo.CreateFileRecord(context.Background(), fileRecord)
		if err != nil {
			log.Error("创建文件操作记录失败", err)
		}

		sftpFile, err := fileSession.Client.(*sftp.Client).Open(f)
		if err != nil {
			log.Errorf("打开文件失败: %v", err.Error())
			return FailWithDataOperate(c, 500, "操作失败", "", nil)
		}

		defer func(file *sftp.File) {
			err := file.Close()
			if err != nil {
				log.Error("关闭文件失败", err)
			}
		}(sftpFile)
		c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filenameWithSuffix))

		var buff bytes.Buffer
		if _, err := sftpFile.WriteTo(&buff); err != nil {
			return err
		}

		return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
	} else if fileSession.Protocol == constant.FTP {
		//// 保存一份到本地
		ftpFileForLocal, err := fileSession.Client.(*ftp.ServerConn).Retr(f)
		if err != nil {
			log.Errorf("打开文件失败: %v", err.Error())
			return FailWithDataOperate(c, 500, "操作失败", "", nil)
		}
		if _, err := os.Stat(path.Join(config.GlobalCfg.Guacd.Recording, id)); os.IsNotExist(err) {
			err := os.MkdirAll(path.Join(config.GlobalCfg.Guacd.Recording, id), 0777)
			if err != nil {
				log.Error("创建本地文件夹失败", err)
			}
		}

		localFilePath := path.Join(config.GlobalCfg.Guacd.Recording, id, utils.UUID()+"."+filenameWithSuffix)
		localFile, err := os.Create(localFilePath)
		if err != nil {
			log.Error("创建本地文件失败", err)
		}
		_, err = io.Copy(localFile, ftpFileForLocal)
		if err != nil {
			log.Error("复制文件失败", err)
		}
		func(localFile *os.File) {
			err := localFile.Close()
			if err != nil {
				log.Error("关闭本地文件失败", err)
			}
		}(localFile)
		log.Infof("文件保存本地成功: %v", localFilePath)
		// 关闭打开的ftpFileForLocal文件
		err = ftpFileForLocal.Close()
		if err != nil {
			log.Error("关闭ftpFileForLocal文件失败", err)
		}

		// 创建文件操作记录
		stat, err := fileSession.Client.(*ftp.ServerConn).FileSize(f)
		fmt.Println("stat", stat)

		var fileRecord = &model.FileRecord{
			SessionId:  id,
			FileName:   filenameWithSuffix,
			FilePath:   localFilePath,
			FileSize:   stat,
			Action:     constant.DOWNLOAD,
			CreateTime: utils.NowJsonTime(),
		}
		err = repository.SessionRepo.CreateFileRecord(context.Background(), fileRecord)
		if err != nil {
			log.Error("创建文件操作记录失败", err)
		}

		ftpFile, err := os.Open(localFilePath)
		if err != nil {
			log.Errorf("打开文件失败: %v", err.Error())
			return FailWithDataOperate(c, 500, "操作失败", "", nil)
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				log.Error("关闭文件失败", err)
			}
		}(ftpFile)

		c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filenameWithSuffix))

		var buff bytes.Buffer
		if _, err := io.Copy(&buff, ftpFile); err != nil {
			log.Error("复制文件失败", err)
			return FailWithDataOperate(c, 500, "操作失败", "", nil)
		}

		return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
	} else {
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}
}

func FileSessionUploadEndpoint(c echo.Context) error {
	id := c.Param("id")

	fs, err := repository.SessionRepo.GetById(context.Background(), id)
	if err != nil {
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}
	if fs.Download == 0 {
		return FailWithDataOperate(c, 403, "无权限上传", "", nil)
	}

	fileSession := file_session.FileSessions.Get(id)
	if fileSession == nil {
		log.Error("文件会话不存在")
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}
	file, err := c.FormFile("file")
	if err != nil {
		log.Error("获取文件失败", err)
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}

	filename := file.Filename
	remoteDir := c.QueryParam("dir")

	// 保存一份到本地
	fileCopeForLocal, err := file.Open()
	if err != nil {
		log.Error("打开文件失败", err)
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}
	if _, err := os.Stat(path.Join(config.GlobalCfg.Guacd.Recording, id)); os.IsNotExist(err) {
		err := os.MkdirAll(path.Join(config.GlobalCfg.Guacd.Recording, id), 0777)
		if err != nil {
			log.Error("创建本地文件夹失败", err)
		}
	}
	localFilePath := path.Join(config.GlobalCfg.Guacd.Recording, id, utils.UUID()+"."+filename)
	if _, err := os.Stat(path.Join(config.GlobalCfg.Guacd.Recording, id)); !os.IsNotExist(err) {
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

	// 创建文件操作记录
	var fileRecord = &model.FileRecord{
		SessionId:  id,
		FileName:   filename,
		FilePath:   localFilePath,
		FileSize:   file.Size,
		Action:     constant.UPLOAD,
		CreateTime: utils.NowJsonTime(),
	}
	err = repository.SessionRepo.CreateFileRecord(context.Background(), fileRecord)
	if err != nil {
		log.Error("创建文件操作记录失败", err)
	}

	src, err := file.Open()
	if err != nil {
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}
	defer func(src multipart.File) {
		err := src.Close()
		if err != nil {
			log.Error("关闭文件失败", err)
		}
	}(src)
	if fileSession.Protocol == constant.SFTP {
		remoteFile := path.Join(remoteDir, filename)
		sftpClient := fileSession.Client.(*sftp.Client)
		// 文件夹不存在时自动创建文件夹
		if _, err := sftpClient.Stat(remoteDir); os.IsNotExist(err) {
			if err := sftpClient.MkdirAll(remoteDir); err != nil {
				log.Error("创建文件夹失败", err)
				return FailWithDataOperate(c, 500, "操作失败", "", nil)
			}
		}

		dstFile, err := sftpClient.Create(remoteFile)
		if err != nil {
			log.Error("创建文件失败", err)
			return FailWithDataOperate(c, 500, "操作失败", "", nil)
		}
		defer func(dstFile *sftp.File) {
			err := dstFile.Close()
			if err != nil {
				log.Error("关闭文件失败", err)
			}
		}(dstFile)

		if _, err = io.Copy(dstFile, src); err != nil {
			return FailWithDataOperate(c, 500, "操作失败", "", nil)
		}
		return SuccessWithOperate(c, "主机接入-文件上传: 文件["+remoteFile+"]", nil)
	} else if fileSession.Protocol == constant.FTP {
		remoteFile := path.Join(fileSession.Dir, remoteDir, filename)
		ftpClient := fileSession.Client.(*ftp.ServerConn)
		// 文件夹不存在时自动创建文件夹
		if _, err := ftpClient.GetEntry(remoteDir); os.IsNotExist(err) {
			if err := ftpClient.MakeDir(remoteDir); err != nil {
				return FailWithDataOperate(c, 500, "操作失败", "", nil)
			}
		}

		err := ftpClient.Stor(remoteFile, src)
		if err != nil {
			return FailWithDataOperate(c, 500, "操作失败", "", nil)
		}
		return SuccessWithOperate(c, "主机接入-文件上传: 文件["+remoteFile+"]", nil)

	} else {
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}
}

func FileSessionDisconnectEndpoint(c echo.Context) error {
	id := c.Param("id")
	NewCloseSessionById(id, Normal, "用户正常退出", false)
	return Success(c, nil)
}

func FileSessionMkdirEndpoint(c echo.Context) error {
	id := c.Param("id")
	filesession := file_session.FileSessions.Get(id)
	if filesession == nil {
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}

	dir := c.QueryParam("dir")
	if filesession.Protocol == constant.SFTP {
		sftpClient := filesession.Client.(*sftp.Client)
		if _, err := sftpClient.Stat(dir); os.IsNotExist(err) {
			if err := sftpClient.Mkdir(dir); err != nil {
				return FailWithDataOperate(c, 500, "操作失败", "", nil)
			}
		}
		return SuccessWithOperate(c, "主机接入-创建文件夹: 文件夹["+dir+"]", nil)
	} else if filesession.Protocol == constant.FTP {
		ftpClient := filesession.Client.(*ftp.ServerConn)
		if _, err := ftpClient.GetEntry(dir); os.IsNotExist(err) {
			if err := ftpClient.MakeDir(dir); err != nil {
				return FailWithDataOperate(c, 500, "操作失败", "", nil)
			}
		}
		return SuccessWithOperate(c, "主机接入-创建文件夹: 文件夹["+dir+"]", nil)
	} else {
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}
}

func FileSessionRmEndpoint(c echo.Context) error {
	id := c.Param("id")
	filesession := file_session.FileSessions.Get(id)
	if filesession == nil {
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}

	f := c.FormValue("file")
	if filesession.Protocol == constant.SFTP {
		sftpClient := filesession.Client.(*sftp.Client)
		stat, err := sftpClient.Stat(f)
		if err != nil {
			return FailWithDataOperate(c, 500, "操作失败", "", nil)
		}
		if stat.IsDir() {
			if err := sftpClient.RemoveDirectory(f); err != nil {
				return FailWithDataOperate(c, 500, "操作失败", "", nil)
			}
		} else {
			if err := sftpClient.Remove(f); err != nil {
				return FailWithDataOperate(c, 500, "操作失败", "", nil)
			}
		}
		return SuccessWithOperate(c, "主机接入-删除文件: 文件/文件夹["+f+"]", nil)
	} else if filesession.Protocol == constant.FTP {
		ftpClient := filesession.Client.(*ftp.ServerConn)
		stat, err := ftpClient.GetEntry(f)
		if err != nil {
			return FailWithDataOperate(c, 500, "操作失败", "", nil)
		}
		if stat.Type == ftp.EntryTypeFolder {
			if err := ftpClient.RemoveDir(f); err != nil {
				return FailWithDataOperate(c, 500, "操作失败", "", nil)
			}
		} else {
			if err := ftpClient.Delete(f); err != nil {
				return FailWithDataOperate(c, 500, "操作失败", "", nil)
			}
		}
		return SuccessWithOperate(c, "主机接入-删除文件: 文件/文件夹["+f+"]", nil)
	} else {
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}
}

// FileSessionRenameEndpoint 文件重命名
func FileSessionRenameEndpoint(c echo.Context) error {
	return nil
}

// FileRecordEndpoint 获取文件操作记录
func FileRecordEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "未登录")
	}

	id := c.QueryParam("sessionId")
	auto := c.QueryParam("auto")
	fileRecords, err := repository.SessionRepo.GetFileRecordBySessionId(context.Background(), id, auto)
	if err != nil {
		return Fail(c, 500, "获取文件操作记录失败")
	}
	return Success(c, fileRecords)
}

// FileDownloadEndpoint 文件下载
func FileDownloadEndpoint(c echo.Context) error {
	id := c.QueryParam("recordId")
	u, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "未登录")
	}

	fileRecord, err := repository.SessionRepo.GetFileRecord(context.Background(), id)
	if err != nil {
		log.Error("获取文件操作记录失败", err)
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}

	if !utils.FileExists(fileRecord.FilePath) {
		return Fail(c, 500, "文件不存在")
	}

	// 记录操作日志
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		Result:          "成功",
		LogTypes:        "用户日志",
		Created:         utils.NowJsonTime(),
		Names:           u.Nickname,
		LogContents:     "主机审计-记录: 下载文件[" + fileRecord.FileName + "]",
		Users:           u.Username,
	}
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}
	// 读取文件转为*bytes.Buffer
	file, err := os.Open(fileRecord.FilePath)
	if err != nil {
		return FailWithDataOperate(c, 500, "下载失败", "", nil)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Error("file close error: ", err)
		}
	}(file)

	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileRecord.FileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, file)
}
