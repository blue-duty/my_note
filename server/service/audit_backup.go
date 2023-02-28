package service

import (
	"context"
	"encoding/base64"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"tkbastion/pkg/constant"
	"tkbastion/pkg/global"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/repository"
	"tkbastion/server/utils"

	"github.com/labstack/gommon/log"
)

type auditBackupService struct {
	orderLogRepository    *repository.WorkOrderNewRepository
	operateLogRepository  *repository.OperateLogRepository
	loginLogRepository    *repository.LoginLogRepository
	hostOperateRepository *repository.HostOperateRepository
	propertyRepository    *repository.PropertyRepository
}

func NewAuditBackupService(orderLogRepository *repository.WorkOrderNewRepository, operateLogRepository *repository.OperateLogRepository, loginLogRepository *repository.LoginLogRepository, hostOperateRepository *repository.HostOperateRepository, propertyRepository *repository.PropertyRepository) *auditBackupService {
	return &auditBackupService{orderLogRepository: orderLogRepository, operateLogRepository: operateLogRepository, loginLogRepository: loginLogRepository, hostOperateRepository: hostOperateRepository, propertyRepository: propertyRepository}
}

func (s *auditBackupService) BackupAuditLog() (string, error) {
	now := time.Now().Format("20060102150405")
	var backupPath string
	backupPath = path.Join(constant.BackupPath, "tkbastion_审计日志备份_"+now)
	// 判断文件夹不存在时自动创建
	if !utils.FileExists(backupPath) {
		if err := os.MkdirAll(backupPath, os.ModePerm); err != nil {
			log.Errorf("MkdirAll Error: %v", err)
			return "", err
		}
	}
	// 登陆日志
	{
		exportFileName := "tk_登录日志.xlsx"
		items, err := s.loginLogRepository.Find("", "", "", "", "", "")
		if nil != err {
			log.Errorf("审计备份-获取登录日志失败: %v", err)
		}

		header := []string{"登录时间", "来源地址", "用户名", "姓名", "协议", "登录方式", "登录结果", "描述"}
		var forExport = make([][]string, 0, len(items))
		for i := 0; i < len(items); i++ {
			forExport = append(forExport, []string{
				items[i].LoginTime.Format("2006-01-02 15:04:05"),
				items[i].ClientIP,
				items[i].Username,
				items[i].Nickname,
				items[i].Protocol,
				items[i].LoginType,
				items[i].LoginResult,
				items[i].Description,
			})
		}

		file, err := utils.CreateExcelFile(exportFileName, header, forExport)
		if err != nil {
			log.Errorf("审计备份-创建登录日志Excel文件失败: %v", err)
		}
		err = file.SaveAs(path.Join(backupPath, "tk_登录日志.xlsx"))
		if err != nil {
			log.Errorf("审计备份-保存登录日志失败: %v", err)
		}
	}
	// 操作日志
	{
		items, err := s.operateLogRepository.Find("", "", "", "", "", "")
		if nil != err {
			log.Errorf("操作日志备份失败: %v", err)
		}
		var operateLogArr []model.OperateForPageNew
		var operateLog model.OperateForPageNew
		for i := range items {
			operateLog.Created = items[i].Created
			operateLog.IP = items[i].IP
			operateLog.Users = items[i].Users
			operateLog.Names = items[i].Names
			operateLog.Result = items[i].Result

			separator1Index := strings.Index(items[i].LogContents, "-")
			operateLog.FunctionalModule = items[i].LogContents[:separator1Index]
			separator2Index := strings.Index(items[i].LogContents, ":")
			if -1 == separator1Index {
				log.Errorf("操作日志内容字段格式错误: %v", items[i].LogContents)
				continue
			}

			if -1 == separator2Index {
				operateLog.Action = items[i].LogContents[separator1Index+1:]
				log.Errorf("操作日志内容字段格式错误, 缺少\":\": %v", items[i].LogContents)
				operateLog.LogContents = "操作成功"
			} else {
				operateLog.Action = items[i].LogContents[separator1Index+1 : separator2Index]
				operateLog.LogContents = items[i].LogContents[separator2Index+2:]
			}

			operateLogArr = append(operateLogArr, operateLog)
		}

		var forExport = make([][]string, 0, len(items))
		for i := 0; i < len(operateLogArr); i++ {
			forE := []string{
				items[i].Created.Format("2006-01-02 15:04:05"),
				items[i].IP,
				items[i].Users,
				items[i].Names,
			}
			separator1Index := strings.Index(items[i].LogContents, "-")
			forE = append(forE, items[i].LogContents[:separator1Index])
			separator2Index := strings.Index(items[i].LogContents, ":")
			if -1 == separator2Index {
				forE = append(forE, items[i].LogContents[separator1Index+1:])
				forE = append(forE, "操作成功")
			} else {
				forE = append(forE, items[i].LogContents[separator1Index+1:separator2Index])
				forE = append(forE, items[i].LogContents[separator2Index+2:])
			}
			forE = append(forE, items[i].Result)
			forExport = append(forExport, forE)
		}

		exportFileName := "操作日志.xlsx"
		header := []string{"操作时间", "来源地址", "用户名", "姓名", "功能模块", "动作", "详细内容", "结果"}
		file, err := utils.CreateExcelFile(exportFileName, header, forExport)
		if err != nil {
			log.Errorf("审计备份-创建操作日志Excel文件失败: %v", err)
		}
		err = file.SaveAs(path.Join(backupPath, "tk_操作日志.xlsx"))
		if err != nil {
			log.Errorf("审计备份-操作日志-保存文件失败: %v", err)
		}
	}
	// 工单日志
	{
		newWorkOrderLogList, err := s.orderLogRepository.FindAll()
		if err != nil {
			log.Errorf("WorkOrder FindWorkOrderLogByDepIds err: %v", err)
		}
		workOrderLogArr := make([]dto.WorkOrderLogForExport, len(newWorkOrderLogList))
		for i, v := range newWorkOrderLogList {
			workOrderLogArr[i] = dto.WorkOrderLogForExport{
				Title:           v.Title,
				ApplyTime:       v.ApplyTime.Format("2006-01-02 15:04:05"),
				ApproveTime:     v.ApproveTime.Format("2006-01-02 15:04:05"),
				ApproveUsername: v.ApproveUsername,
				ApproveNickname: v.ApproveNickname,
				Department:      v.Department,
				Result:          v.Result,
				Info:            v.ApproveInfo,
			}
		}
		workOrderLogStringsForExport := make([][]string, len(workOrderLogArr))
		for i, v := range workOrderLogArr {
			user := utils.Struct2StrArr(v)
			workOrderLogStringsForExport[i] = make([]string, len(user))
			workOrderLogStringsForExport[i] = user
		}
		userHeaderForExport := []string{"标题", "提交时间", "审批时间", "审批人", "姓名", "部门机构", "审批结果", "审批备注"}
		userFileNameForExport := "审批日志"
		file, err := utils.CreateExcelFile(userFileNameForExport, userHeaderForExport, workOrderLogStringsForExport)
		if err != nil {
			log.Errorf("审计备份-创建审批日志Excel文件失败: %v", err)
		}
		err = file.SaveAs(path.Join(backupPath, "tk_审批日志.xlsx"))
		if err != nil {
			log.Errorf("审计备份-审批日志-保存文件失败: %v", err)
		}
	}
	// 运维日志
	{
		opl, err := s.hostOperateRepository.GetOperateLogsForExport()
		if nil != err {
			log.Errorf("GetOperateLogList Error: %v", err)
		}

		forExport := make([][]string, len(opl))
		for i, v := range opl {
			asset := utils.Struct2StrArr(v)
			forExport[i] = make([]string, len(asset))
			forExport[i] = asset
		}
		headerForExport := []string{"登录时间", "来源IP", "运维人员账号", "运维人员名称", "类型", "运维对象名称", "运维对象地址", "运维对象账号", "登出时间"}
		fileNameForExport := "运维日志"
		file, err := utils.CreateExcelFile(fileNameForExport, headerForExport, forExport)
		if err != nil {
			log.Errorf("审计备份-创建运维日志Excel文件失败: %v", err)
		}
		err = file.SaveAs(path.Join(backupPath, "tk_运维日志.xlsx"))
		if err != nil {
			log.Errorf("审计备份-运维日志-保存文件失败: %v", err)
		}
	}
	// 系统告警
	{
		systemAlarmLog, err := repository.OperateAlarmLogRepo.FindBySystemAlarmLogForSearch(context.TODO(), "", "", "")
		if err != nil {
			log.Error("获取系统告警日志失败")
		}
		data := make([][]string, len(systemAlarmLog))
		for i, v := range systemAlarmLog {
			var level = ""
			if v.Level == "high" {
				level = "高"
			} else if v.Level == "middle" {
				level = "中"
			} else {
				level = "低"
			}
			var systemAlarmLogForExport = dto.SystemAlarmLogForExport{
				AlarmTime: v.AlarmTime.Format("2006-01-02 15:04:05"),
				Content:   v.Content,
				Strategy:  v.Strategy,
				Level:     level,
				Result:    v.Result,
			}
			data[i] = utils.Struct2StrArr(systemAlarmLogForExport)
		}

		header := []string{"告警时间", "告警内容", "触发策略", "事件级别", "发送结果"}

		file, err := utils.CreateExcelFile("系统告警日志", header, data)
		if err != nil {
			log.Errorf("审计备份-创建系统告警日志Excel文件失败: %v", err)
		}
		err = file.SaveAs(path.Join(backupPath, "tk_系统告警日志.xlsx"))
		if err != nil {
			log.Errorf("审计备份-系统告警-保存文件失败: %v", err)
		}
	}
	// 操作告警
	{
		operateAlarmLog, err := repository.OperateAlarmLogRepo.FindByAlarmLogForSearch(context.TODO(), dto.OperateAlarmLogForSearch{})
		if err != nil {
			log.Error("获取操作告警日志失败")
		}

		operateAlarmLogStringsForExport := make([][]string, len(operateAlarmLog))
		for i, v := range operateAlarmLog {
			operateAlarmLogStringsForExport[i] = utils.Struct2StrArr(v)
		}

		header := []string{"告警时间", "来源地址", "用户名", "姓名", "设备地址", "设备账号", "协议", "告警内容", "触发策略", "事件级别", "发送结果"}

		file, err := utils.CreateExcelFile("告警报表", header, operateAlarmLogStringsForExport)
		if err != nil {
			log.Errorf("审计备份-创建操作告警日志Excel文件失败: %v", err)
		}
		err = file.SaveAs(path.Join(backupPath, "tk_操作告警日志.xlsx"))
		if err != nil {
			log.Errorf("审计备份-操作告警-保存文件失败: %v", err)
		}
	}

	return "tkbastion_审计日志备份_" + now, nil
}

// 获取sftp/ftp连接信息
func (s *auditBackupService) getFtpOrSftpInfo() (FtpClientConfig, SftpClientConfig, error) {
	property := s.propertyRepository.FindAllMap()
	if property["enable_remote_automatic_backup"] == "false" {
		return FtpClientConfig{}, SftpClientConfig{}, nil
	}
	if property["remote_backup_protocol"] == "FTP" {
		ftpConfig := FtpClientConfig{}
		ftpConfig.Host = property["remote_backup_host"]
		ftpConfig.Port, _ = strconv.ParseInt(property["remote_backup_port"], 10, 64)
		ftpConfig.Username = property["remote_backup_account"]
		ftpConfig.Password = property["remote_backup_password"]
		ftpConfig.SavePath = property["remote_backup_path"]
		if property["remote_backup_compress"] == "1" {
			ftpConfig.IsPress = true
		} else {
			ftpConfig.IsPress = false
		}
		origData, err := base64.StdEncoding.DecodeString(ftpConfig.Password)
		if err != nil {
			return FtpClientConfig{}, SftpClientConfig{}, err
		}
		decryptedCBC, err := utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
		if err != nil {
			return FtpClientConfig{}, SftpClientConfig{}, err
		}
		ftpConfig.Password = string(decryptedCBC)
		return ftpConfig, SftpClientConfig{}, nil
	} else {
		sftpConfig := SftpClientConfig{}
		sftpConfig.Host = property["remote_backup_host"]
		sftpConfig.Port, _ = strconv.ParseInt(property["remote_backup_port"], 10, 64)
		sftpConfig.Username = property["remote_backup_account"]
		sftpConfig.Password = property["remote_backup_password"]
		sftpConfig.SavePath = property["remote_backup_path"]
		if property["remote_backup_compress"] == "1" {
			sftpConfig.IsPress = true
		} else {
			sftpConfig.IsPress = false
		}
		origData, err := base64.StdEncoding.DecodeString(sftpConfig.Password)
		if err != nil {
			return FtpClientConfig{}, SftpClientConfig{}, err
		}
		decryptedCBC, err := utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
		if err != nil {
			return FtpClientConfig{}, SftpClientConfig{}, err
		}
		sftpConfig.Password = string(decryptedCBC)
		return FtpClientConfig{}, sftpConfig, nil
	}
}

func (s *auditBackupService) RemoteBackup() error {
	ftpc, sftpc, err := s.getFtpOrSftpInfo()
	if err != nil {
		log.Errorf("GetFtpOrSftpInfo Error: %v", err)
	}
	backpath, err := s.BackupAuditLog()
	if err != nil {
		log.Errorf("Local Backup Error: %v", err)
		return err
	}
	if ftpc.Host != "" {
		err = ftpc.uploadFtp(backpath)
		if err != nil {
			log.Errorf("Ftp Upload Error: %v", err)
			return err
		}
	} else {
		err := sftpc.connect()
		if err != nil {
			log.Errorf("Sftp Connect Error: %v", err)
			return err
		}
		defer func(sftp *SftpClientConfig) {
			err := sftp.close()
			if err != nil {
				log.Errorf("Sftp Close Error: %v", err)
			}
		}(&sftpc)
		err = sftpc.upload(backpath)
		if err != nil {
			log.Errorf("Sftp Upload Error: %v", err)
			return err
		}
	}

	// 删除本地备份文件夹
	err = os.RemoveAll(path.Join(constant.BackupPath, backpath))
	if err != nil {
		log.Errorf("Delete Local Backup Error: %v", err)
		return err
	}
	return err
}

func (s *auditBackupService) Find(auto, name, time string) (fileitem []File, total int64) {
	// 判断文件夹不存在时自动创建
	fileitem = make([]File, 0)
	if !utils.FileExists(constant.BackupPath) {
		if err := os.MkdirAll(constant.BackupPath, os.ModePerm); err != nil {
			return
		}
	}
	fileInfos, err := os.ReadDir(constant.BackupPath)
	if err != nil {
		return
	}
	for i := range fileInfos {
		if !strings.HasSuffix(fileInfos[i].Name(), ".zip") {
			continue
		}
		//if !fileInfos[i].IsDir() {
		//	continue
		//}
		fileInfo, _ := fileInfos[i].Info()
		file := File{
			Name:    fileInfos[i].Name()[0 : len(fileInfos[i].Name())-4],
			IsDir:   fileInfos[i].IsDir(),
			Mode:    fileInfo.Mode().String(),
			IsLink:  fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink,
			ModTime: utils.NewJsonTime(fileInfo.ModTime()),
			Size:    fileInfo.Size(),
		}
		// fmt.Println(file)
		if len(name) > 0 && !strings.Contains(file.Name, name) {
			continue
		} else {
			if len(time) > 0 {
				if strings.Contains(file.Name, time) {
					fileitem = append(fileitem, file)
					total++
				}
			} else {
				fileitem = append(fileitem, file)
				total++
			}
		}
	}
	//if order == "ascend" {
	//	for i := 0; i < len(fileitem)-1; i++ {
	//		for j := 0; j < len(fileitem)-1-i; j++ {
	//			//根据时间升序
	//			if fileitem[j].ModTime.Unix() > fileitem[j+1].ModTime.Unix() {
	//				fileitem[j], fileitem[j+1] = fileitem[j+1], fileitem[j]
	//			}
	//		}
	//	}
	//}
	//if order == "descend" {
	for i := 0; i < len(fileitem)-1; i++ {
		for j := 0; j < len(fileitem)-1-i; j++ {
			//根据时间降序
			if fileitem[j].ModTime.Unix() < fileitem[j+1].ModTime.Unix() {
				fileitem[j], fileitem[j+1] = fileitem[j+1], fileitem[j]
			}
		}
	}
	//}
	return
}
