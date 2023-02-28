package service

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"tkbastion/pkg/constant"
	"tkbastion/pkg/global"
	"tkbastion/pkg/guacd"
	"tkbastion/pkg/log"
	"tkbastion/pkg/terminal"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/repository"
	"tkbastion/server/utils"

	"golang.org/x/crypto/ssh"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

type NewJobService struct {
	newJobRepository               *repository.JobRepositoryNew
	newAssetRepository             *repository.AssetRepositoryNew
	propertyRepository             *repository.PropertyRepository
	regularReportRepository        *repository.RegularReportRepository
	newSessionRepository           *repository.SessionRepositoryNew
	loginLogRepository             *repository.LoginLogRepository
	userAccessStatisticsRepository *repository.UserAccessStatisticsRepository
	operateReportRepository        *repository.OperateReportRepository
	operateAlarmLogRepository      *repository.OperateAlarmLogRepository
	messageRepository              *repository.MessageRepository
	userNewRepository              *repository.UserNewRepository
	appSessionRepository           *repository.AppSessionRepository
}

func NewNewJobService(newJobRepository *repository.JobRepositoryNew, newAssetRepository *repository.AssetRepositoryNew,
	propertyRepository *repository.PropertyRepository, regularReportRepository *repository.RegularReportRepository,
	newSessionRepository *repository.SessionRepositoryNew, loginLogRepository *repository.LoginLogRepository,
	userAccessStatisticsRepository *repository.UserAccessStatisticsRepository, operateReportRepository *repository.OperateReportRepository,
	operateAlarmLogRepository *repository.OperateAlarmLogRepository, messageRepository *repository.MessageRepository,
	userNewRepository *repository.UserNewRepository, appSessionRepository *repository.AppSessionRepository) *NewJobService {
	return &NewJobService{
		newJobRepository:               newJobRepository,
		newAssetRepository:             newAssetRepository,
		propertyRepository:             propertyRepository,
		regularReportRepository:        regularReportRepository,
		newSessionRepository:           newSessionRepository,
		loginLogRepository:             loginLogRepository,
		userAccessStatisticsRepository: userAccessStatisticsRepository,
		operateReportRepository:        operateReportRepository,
		operateAlarmLogRepository:      operateAlarmLogRepository,
		messageRepository:              messageRepository,
		userNewRepository:              userNewRepository,
		appSessionRepository:           appSessionRepository,
	}
}

// NewPlanJob 新建任务
func (s NewJobService) NewPlanJob(job dto.NewJobForCreate) (err error) {
	doFunc := func() {
		if job.RunTimeType == "Periodic" {
			if (job.StartAt.Before(time.Now().Add(time.Minute*1)) || job.StartAt.Equal(time.Now().Add(time.Minute*1))) && job.EndAt.After(time.Now()) {
				goto job
			}
			if job.EndAt.Before(time.Now()) {
				err := global.SCHEDULER.RemoveByTag(job.ID)
				log.Infof("删除任务: %v", job.ID)
				if err != nil {
					return
				}
			}
			return
		} else {
			// 一次性任务
			// 	误差1分钟
			if job.RunTime.Before(time.Now().Add(time.Minute*1)) || job.RunTime.Equal(time.Now().Add(time.Minute*1)) || job.RunTime.After(time.Now()) {
				err := global.SCHEDULER.RemoveByTag(job.ID)
				log.Infof("删除任务: %v", job.ID)
				if err != nil {
					return
				}
			} else {
				goto job
			}
		}
	job:
		{
			pid, nj, err := s.newJobRepository.FindPassportIdsAndJobByJobId(context.TODO(), job.ID)
			if err != nil {
				return
			}
			passports, err := s.newAssetRepository.GetPassportListByIds(context.TODO(), pid)
			if err != nil {
				return
			}
			//msgChan := make(chan Result)
			var wg sync.WaitGroup
			for _, v := range passports {
				var (
					v = v
				)
				err = s.runJobByJobAndPassport(nj, v, &wg)
				if err != nil {
					continue
				}
			}
			wg.Wait()
			return
		}
	}
	j, _ := global.SCHEDULER.FindJobsByTag(job.ID)
	if len(j) > 0 {
		err := global.SCHEDULER.RemoveByTag(job.ID)
		if err != nil {
			log.Errorf("删除计划任务失败: %v", err)
			return err
		}
	}

	switch job.RunTimeType {
	case "Scheduled":
		// 定时任务
		if job.RunTime.Before(time.Now()) {
			return nil
		}
		fmt.Println(job.RunTime.Format("05 04 15 02 01 *"))
		fmt.Println(job.RunTime.Format("2006-01-02 15:04:05"))
		_, err = global.SCHEDULER.Tag(job.ID).CronWithSeconds(job.RunTime.Format("05 04 15 02 01 *")).Do(doFunc)
		if err != nil {
			return err
		}
		log.Info("添加定时任务成功")
	case "Periodic":
		// 周期任务
		switch job.PeriodicType {
		case "Day":
			_, err = global.SCHEDULER.Every(job.Periodic).Day().Tag(job.ID).Do(doFunc)
			if err != nil {
				log.Errorf("添加周期任务失败: %v", err)
				return err
			}
		case "Week":
			_, err = global.SCHEDULER.Every(job.Periodic).Week().Tag(job.ID).Do(doFunc)
		case "Month":
			_, err = global.SCHEDULER.Every(job.Periodic).Month().Tag(job.ID).Do(doFunc)
		case "Minute":
			_, err = global.SCHEDULER.Every(job.Periodic).Minute().Tag(job.ID).Do(doFunc)
		case "Hour":
			_, err = global.SCHEDULER.Every(job.Periodic).Hour().Tag(job.ID).Do(doFunc)
		}
		if err != nil {
			return err
		}
		log.Info("添加周期任务成功")
	default:
		return
	}
	global.SCHEDULER.StartAsync()
	return
}

// RunJobNow 立即执行
func (s NewJobService) RunJobNow(id string) (err error) {
	job, err := s.newJobRepository.FindById(context.TODO(), id)
	if err != nil {
		log.Errorf("查询任务失败: %v", err)
		return
	}
	pids, nj, err := s.newJobRepository.FindPassportIdsAndJobByJobId(context.TODO(), job.ID)
	if err != nil {
		log.Errorf("查询任务资产失败: %v", err)
		return
	}
	passports, err := s.newAssetRepository.GetPassportListByIds(context.TODO(), pids)
	if err != nil {
		log.Errorf("查询资产失败: %v", err)
		return
	}
	//msgChan := make(chan Result)
	wg := sync.WaitGroup{}
	for _, v := range passports {
		var (
			v = v
		)
		err = s.runJobByJobAndPassport(nj, v, &wg)
		if err != nil {
			log.Error("执行任务失败: %v", err)
			continue
		}
	}
	wg.Wait()
	return
}

// ReloadJob 重新加载任务
func (s NewJobService) ReloadJob() (err error) {
	{
		// 计划任务
		var jobs []model.NewJob
		jobs, err = s.newJobRepository.GetAll(context.TODO())
		if err != nil {
			log.Errorf("查询计划任务列表失败: %v", err)
		}

		for _, v := range jobs {
			if v.RunTimeType != "Manual" {
				job := dto.NewJobForCreate{
					ID:           v.ID,
					Name:         v.Name,
					Command:      v.Command,
					Periodic:     v.Periodic,
					PeriodicType: v.PeriodicType,
					Department:   v.Department,
					DepartmentID: v.DepartmentID,
					RunTimeType:  v.RunTimeType,
					RunTime:      v.RunTime,
					RunType:      v.RunType,
					ShellName:    v.ShellName,
					Info:         v.Info,
					StartAt:      v.StartAt,
					EndAt:        v.EndAt,
				}
				err = s.NewPlanJob(job)
				if err != nil {
					log.Errorf("添加计划任务[%s]失败: %v", job.Name, err)
				}
				log.Info("开启计划任务 [", v.Name, "], 运行中计划任务数量: [", len(global.SCHEDULER.Jobs()), "]")
			}
		}
	}

	{
		// 每隔一小时删除一次未使用的会话信息
		// 删除数据库中连接失败与连接中但尚未建立连接的记录
		_, err = global.SCHEDULER.Every(1).Hours().Do(func() {
			sessions, _ := repository.SessionRepo.GetSessionByStatus(context.TODO(), []string{constant.NoConnect, constant.Connecting})
			if len(sessions) > 0 {
				now := time.Now()
				for i := range sessions {
					if now.Sub(sessions[i].ConnectedTime.Time) > time.Hour*1 {
						_ = repository.SessionRepo.Delete(context.TODO(), sessions[i].ID)
						s := sessions[i].CreateName + "@" + sessions[i].AssetIP + ":" + strconv.Itoa(sessions[i].AssetPort)
						log.Infof("会话「%v」ID「%v」超过1小时未打开,已删除.", s, sessions[i].ID)
					}
				}
			}
		})
		if err != nil {
			log.Error("每隔一小时删除一次未使用的会话信息失败: ", err)
			return
		}
		log.Infof("开启计划任务 [每隔一小时删除一次未使用的会话信息], 运行中计划任务数量: [%d]", len(global.SCHEDULER.Jobs()))
	}

	{
		// 系统配置中设置的任务
		pro := s.propertyRepository.FindAllMap()
		{
			// 本地审计备份和远程审计备份
			if pro["enable_remote_automatic_backup"] == "1" {
				if rbi, ok := pro["remote_backup_interval"]; ok {
					err := s.NewBackupJob(rbi, constant.RemoteBackup)
					if err != nil {
						log.Errorf("开启远程审计备份失败: %v", err)
					}
					//log.Infof("开启计划任务 [远程审计备份], 运行中计划任务数量: [%d]", len(global.SCHEDULER.Jobs()))
				} else {
					log.Info("未配置远程审计备份时间,远程审计备份任务重新加载失败")
				}
			}
			if pro["enable_local_automatic_backup"] == "1" {
				if lbi, ok := pro["local_backup_interval"]; ok {
					err := s.NewBackupJob(lbi, constant.LocalBackup)
					if err != nil {
						log.Errorf("开启本地审计备份失败: %v", err)
					}
					//log.Infof("开启计划任务 [本地审计备份], 运行中计划任务数量: [%d]", len(global.SCHEDULER.Jobs()))
				} else {
					log.Info("未配置本地审计备份时间,本地审计备份任务重新加载失败")
				}
			}
		}
		{
			// 存储空间监控
			if pro["enable_storage_space"] == "1" {
				err := s.AddStorageLimitJob()
				if err != nil {
					log.Errorf("开启存储空间监控失败: %v", err)
				}
				log.Infof("开启计划任务 [存储空间限制], 运行中计划任务数量: [%d]", len(global.SCHEDULER.Jobs()))
			}
			// 日志存储时间监控
			if pro["enable_log_storage_limit"] == "1" {
				err := s.DeleteTimeoutLog()
				if err != nil {
					log.Errorf("开启日志存储时间监控失败: %v", err)
				}
				log.Infof("开启计划任务 [日志存储时间限制], 运行中计划任务数量: [%d]", len(global.SCHEDULER.Jobs()))
			}
		}
	}

	{
		// 重新加载改密策略
		pc, err := repository.PasswdChangeRepo.FindAll(context.TODO())
		if err != nil {
			log.Error("重新加载改密策略失败: ", err)
		}
		for i := range pc {
			if pc[i].RunType != "Manual" {
				err = CreateAutoChangePasswdTask(pc[i].ID)
				if err != nil {
					log.Errorf("重新加载改密策略[%s]失败: %v", pc[i].Name, err)
				}
			}
		}
	}

	{
		// 重新加载系统利用率监控
		if err := s.WriteSysUsage(); err != nil {
			log.Errorf("重新加载系统利用率监控任务[%s]失败: %v", "系统利用率", err)
		}
	}

	{
		// 重新加载系统利用率告警
		if err = s.AlarmConfigurationPerMonitoring(); err != nil {
			log.Errorf("重新加载系统利用率告警任务[%s]失败: %v", "系统利用率", err)
		}
		// 重新加载系统访问量告警
		if err = s.AlarmConfigurationAccMonitoring(); err != nil {
			log.Errorf("重新加载系统访问量告警任务[%s]失败: %v", "系统访问量", err)
		}
	}

	// 根据有效期更新策略状态定时任务
	//_, err = global.SCHEDULER.Every(1).Hour().Do(func() {
	//	var operateAuthArr []model.OperateAuth
	//	err := global.DBConn.Model(model.OperateAuth{}).Find(&operateAuthArr).Error
	//	if nil != err {
	//		log.Errorf("DB Error: %v", err)
	//		log.Error("定时任务[根据有效期更新策略状态]执行失败")
	//		return
	//	}
	//	for i := range operateAuthArr {
	//		if operateAuthArr[i].StrategyTimeFlag {
	//			operateAuthArr[i].State = operateAuthArr[i].ButtonState
	//		} else {
	//			now := utils.NowJsonTime()
	//			if now.After(operateAuthArr[i].StrategyEndTime.Time) || now.Before(operateAuthArr[i].StrategyBeginTime.Time) {
	//				operateAuthArr[i].State = "overdue"
	//			} else {
	//				operateAuthArr[i].State = operateAuthArr[i].ButtonState
	//			}
	//		}
	//		err = global.DBConn.Model(model.OperateAuth{}).Where("id = ?", operateAuthArr[i].ID).Updates(&operateAuthArr[i]).Error
	//		if nil != err {
	//			log.Errorf("DB Error: %v", err)
	//			log.Error("定时任务[根据有效期更新策略状态]执行失败")
	//			return
	//		}
	//	}
	//	log.Info("定时任务[根据有效期更新策略状态]执行成功")
	//})
	//if err != nil {
	//	log.Errorf("AddJob Error: %v", err)
	//	return
	//}
	//log.Debugf("开启计划任务 [根据有效期更新策略状态], 运行中计划任务数量[%v]", len(global.Cron.Entries()))
	//
	//err = s.AutoSyncTime()
	//if nil != err {
	//	log.Errorf("AddJob Error: %v", err)
	//	return err
	//}
	//log.Debugf("开启计划任务[自动同步时间], 运行中计划任务数量[%v]", len(global.Cron.Entries()))
	return
}

// NewBackupJob 创建备份任务
func (s NewJobService) NewBackupJob(t, jt string) (err error) {
	if jt == constant.LocalBackup {
		// 查询是否存在本地备份任务
		j, _ := global.SCHEDULER.FindJobsByTag(constant.LocalBackup)
		if len(j) > 0 {
			err := global.SCHEDULER.RemoveByTag(constant.LocalBackup)
			if err != nil {
				log.Errorf("删除计划任务失败: %v", err)
				return err
			}
		}
		fmt.Println("创建本地备份任务", t)
		_, err = global.SCHEDULER.Every(1).Day().Tag(constant.LocalBackup).At(t).Do(func() {
			name, err := AuditBackupSrv.BackupAuditLog()
			if err != nil {
				log.Errorf("Backup Error: %v", err)
				return
			}
			command := "zip -rj " + path.Join(constant.BackupPath, name+".zip") + " " + path.Join(constant.BackupPath, name)
			//err = ExecutiveCommand("tar -Pczf " + "tkbastion_审计日志_" + now + ".tar -C " + backupPath + " tkbastion_审计日志_" + now)
			_, err = utils.ExecShell(command)
			fmt.Println(err)
			if err != nil {
				log.Errorf("zip Error: %v", err)
			}

			//移动文件至备份文件夹
			_, _ = utils.ExecShell("mv " + name + ".zip " + constant.BackupPath)
			//计划任务日志
			fmt.Println(name)
		})
		if err != nil {
			log.Errorf("AddJob Error: %v", err)
			return
		}
		global.SCHEDULER.StartAsync()
		log.Infof("开启计划任务[本地审计日志备份], 运行中计划任务数量[%v]", global.SCHEDULER.Len())
	} else {
		// 查询是否存在远程备份任务
		j, _ := global.SCHEDULER.FindJobsByTag(constant.RemoteBackup)
		if len(j) > 0 {
			err := global.SCHEDULER.RemoveByTag(constant.RemoteBackup)
			if err != nil {
				log.Errorf("删除计划任务失败: %v", err)
				return err
			}
		}
		_, err = global.SCHEDULER.Every(1).Day().At(t).Tag(constant.RemoteBackup).Do(func() {
			err = AuditBackupSrv.RemoteBackup()
			if err != nil {
				log.Errorf("Backup Error: %v", err)
				return
			}
		})
		if err != nil {
			log.Errorf("开启计划任务[远程审计日志备份]失败, 错误[%s]", err.Error())
			return
		}
		global.SCHEDULER.StartAsync()
		log.Infof("开启计划任务[远程审计日志备份], 运行中计划任务数量[%v]", global.SCHEDULER.Len())
	}
	return
}

// AutoSyncTime 自动同步时间
//func (s NewJobService)AutoSyncTime() error {
//	autoSyncTime, err := s.propertyRepository.FindByNameId("auto-sync-time")
//	if nil != err {
//		log.Errorf("DB Error: %v", err)
//		return err
//	} else {
//		if "true" == autoSyncTime.Value {
//			_,err := global.SCHEDULER.Every(1).Hour().Do(func() {
//				ntpServer, err := s.propertyRepository.FindByNameId("ntp-server")
//				if nil != err {
//					log.Errorf("DB Error: %v", err)
//					log.Error("定时任务[自动同步时间]执行失败")
//					return
//				}
//				err = exec.Command("bash", "-c", `ntpdate `+ntpServer.Value).Run()
//				if nil != err {
//					log.Errorf("Run Error: %v", err)
//					log.Error("定时任务[自动同步时间]执行失败")
//					return
//				}
//				log.Info("定时任务[自动同步时间]执行成功")
//				return
//			})
//			if err != nil {
//				return err
//			}
//		}
//	}
//	return nil
//}

// AutoSyncUserByLdapAd LDAP/AD认证服务器自动同步用户定时任务
//func (s NewJobService) AutoSyncUserByLdapAd() {
//	var ldapAdArr []model.LdapAdAuth
//	err := s.propertyRepository.DB.Model(model.LdapAdAuth{}).Where("ldap_ad_sync_type = 'auto'").Find(&ldapAdArr).Error
//	if nil != err {
//		log.Errorf("DB Error: %v", err)
//		return
//	}
//
//	for i := range ldapAdArr {
//		hourIndex := strings.Index(ldapAdArr[i].LdapAdSyncTime, "点")
//		hour := ldapAdArr[i].LdapAdSyncTime[:hourIndex]
//		minuteIndex := strings.Index(ldapAdArr[i].LdapAdSyncTime, "分")
//		minute := ldapAdArr[i].LdapAdSyncTime[minuteIndex-2 : minuteIndex]
//		//ldapAdSyncUserId, err := global.Cron.AddJob(cronStr, LdapAdSyncUserJob{jobService: &r, ldapAdAuthId: ldapAdArr[i].ID})
//		_,err = global.SCHEDULER.Every(1).Week().Do(func() {
//			err := s.authenticationService.LdapAdSyncUser(ldapAdArr[i].ID)
//			if err != nil {
//				log.Info("定时任务[LDAP/AD用户同步" + strconv.Itoa(int(ldapAdArr[i].ID)) + "]执行失败")
//				return
//			}
//			log.Info("定时任务[LDAP/AD用户同步" + strconv.Itoa(int(ldapAdArr[i].ID)) + "]执行成功")
//		})
//		if nil != err {
//			log.Errorf("AddJob Error: %v", err)
//			return
//		}
//		log.Debugf("开启计划任务 [LDAP/AD用户同步"+strconv.Itoa(int(ldapAdArr[i].ID))+"], 运行中计划任务数量[%v]", len(global.Cron.Entries()))
//	}
//	return
//}

func ExecCommandBySSH(cmd, ip string, port int, username, password, privateKey, passphrase string) (result string, err error) {
	sshClient, err := terminal.NewSshClient(ip, port, username, password, privateKey, passphrase)
	if err != nil {
		log.Errorf("连接ssh服务器失败: %v", err)
		return "", err
	}

	session, err := sshClient.NewSession()
	if err != nil {
		log.Errorf("创建ssh会话失败: %v", err)
		return "", err
	}
	defer func(session *ssh.Session) {
		_ = session.Close()
	}(session)
	//执行远程命令
	combo, err := session.CombinedOutput(cmd)
	if err != nil {
		log.Errorf("执行远程命令失败: %v", err)
		return "", err
	}
	return string(combo), nil
}

// AddStorageLimitJob 添加存储空间限制任务
func (s NewJobService) AddStorageLimitJob() error {
	_, err := global.SCHEDULER.Every(1).Hour().Tag(constant.DiskDetection).Do(func() {
		// 录像
		flag, err := s.propertyRepository.FindByName("enable_storage_space")
		if err != nil {
			log.Errorf("获取磁盘占用率配置失败 %v", err)
			return
		}
		if flag.Value == "1" {
			{
				size, err := s.propertyRepository.FindByName("storage_space_threshold")
				if err != nil {
					log.Errorf("获取磁盘占用率配置失败 %v", err)
					return
				}
				sizeInt, err := strconv.Atoi(size.Value)
				if err != nil {
					log.Errorf("获取磁盘占用设置转数字失败 %v", err)
					return
				}
				si, err := utils.DiskOccupation("/data/tkbastion")
				if err != nil {
					log.Errorf("获取系统磁盘占用率失败 %v", err)
					return
				}
				if si > sizeInt {
					p, err := s.propertyRepository.FindByName("storage_space_threshold_action")
					if err != nil {
						log.Errorf("获取磁盘占用率配置失败 %v", err)
						return
					}
					if p.Value == "1" {
						err := s.propertyRepository.UpdateByName(&model.Property{
							Name:  guacd.EnableRecording,
							Value: "false",
						}, guacd.EnableRecording)
						if err != nil {
							log.Errorf("修改录像配置失败 %v", err)
							return
						}
						log.Info("磁盘空间占用率超过阈值，停止录像")
					} else { // 覆盖最早文件
						file, err := utils.SortFile("/data/tkbastion/recording/")
						if err != nil {
							log.Errorf("获取系统文件信息失败 %v", err)
							return
						}
						for si > sizeInt {
							fmt.Println(file[0].Name())
							err := os.RemoveAll(filepath.Join("/data/tkbastion/recording/", file[0].Name()))
							if err != nil {
								log.Errorf("删除文件失败 %v", err)
								return
							}
							si, err = utils.DiskOccupation("/data/tkbastion")
							if err != nil {
								log.Errorf("获取系统磁盘占用率失败 %v", err)
								return
							}
							file = file[1:]
						}
					}
				}
			}
			{
				// 日志
				flag, err := s.propertyRepository.FindByName("log_storage_space_threshold")
				if err != nil {
					log.Errorf("获取删除日志设置失败 %v", err)
					return
				}
				var c int64
				s.propertyRepository.DB.Model(&model.JobLog{}).Count(&c)
				l := int64(utils.StringToInt(flag.Value)) * 10000
				if c > l {
					log.Infof("任务日志数量超过阈值，开始删除日志")
					s.propertyRepository.DB.Delete(&model.JobLog{}, "id in (select id from job_logs order by time_stamp id limit ?)", l)
				}
				s.propertyRepository.DB.Model(&model.LoginLog{}).Count(&c)
				if c > l {
					log.Infof("登陆日志数量超过阈值，开始删除日志")
					s.propertyRepository.DB.Delete(&model.LoginLog{}, "id in (select id from login_logs order by logout_time limit ?)", l)
				}
				s.propertyRepository.DB.Model(&model.OperateLog{}).Count(&c)
				if c > l {
					s.propertyRepository.DB.Delete(&model.OperateLog{}, "id in (select id from operate_logs order by created limit ?)", l)
				}
				s.propertyRepository.DB.Model(&model.OrderLog{}).Count(&c)
				if c > l {
					s.propertyRepository.DB.Delete(&model.OrderLog{}, "id in (select id from order_logs order by created limit ?)", l)
				}
				s.propertyRepository.DB.Model(&model.NewJobLog{}).Count(&c)
				if c > l {
					s.propertyRepository.DB.Delete(&model.NewJobLog{}, "id in (select id from new_job_logs order by end_at limit ?)", l)
				}
				s.propertyRepository.DB.Model(&model.PasswdChangeResult{}).Count(&c)
				if c > l {
					s.propertyRepository.DB.Delete(&model.PasswdChangeResult{}, "id in (select id from encryption_logs order by timestamp limit ?)", l)
				}
				s.propertyRepository.DB.Model(&model.SystemAlarmLog{}).Count(&c)
				if c > l {
					s.propertyRepository.DB.Delete(&model.SystemAlarmLog{}, "id in (select id from system_alarm_log order by alarm_time limit ?)", l)
				}
				s.propertyRepository.DB.Model(&model.OperateAlarmLog{}).Count(&c)
				if c > l {
					s.propertyRepository.DB.Delete(&model.OperateAlarmLog{}, "id in (select id from operate_alarm_log order by alarm_time limit ?)", l)
				}
			}
		}
	})
	if err != nil {
		log.Errorf("AddJob Error: %v", err)
		return err
	}
	global.SCHEDULER.StartAsync()
	//log.Debugf("开启计划任务[存储空间限制], 运行中计划任务数量[%v]", global.SCHEDULER.Len())
	return nil
}

// DeleteTimeoutLog 查询超时日志并删除
func (s NewJobService) DeleteTimeoutLog() error {
	_, err := global.SCHEDULER.Every(1).Day().Tag(constant.LogTimeDetection).Do(func() {
		// 查询超时日志并删除
		{
			// 操作日志
			flag, err := s.propertyRepository.FindByName("operation_log_save_limit")
			if err != nil {
				log.Errorf("获取操作日志保存限制失败 %v", err)
				return
			}
			// 查询超时日志并删除
			var ids []int
			s.propertyRepository.DB.Model(&model.OperateLog{}).Where("created < ?", time.Now().AddDate(0, 0, -utils.StringToInt(flag.Value))).Pluck("id", &ids)
			if len(ids) > 0 {
				s.propertyRepository.DB.Delete(&model.OperateLog{}, "id in (?)", ids)
			}
		}
		{
			// 登录日志
			flag, err := s.propertyRepository.FindByName("login_log_save_limit")
			if err != nil {
				log.Errorf("获取登录日志保存限制失败 %v", err)
				return
			}
			// 查询超时日志并删除
			var ids []string
			s.propertyRepository.DB.Model(&model.LoginLog{}).Where("logout_time < ?", time.Now().AddDate(0, 0, -utils.StringToInt(flag.Value))).Pluck("id", &ids)
			if len(ids) > 0 {
				s.propertyRepository.DB.Delete(&model.LoginLog{}, "id in (?)", ids)
			}
		}
		{
			// 审批日志
			flag, err := s.propertyRepository.FindByName("audit_log_save_limit")
			if err != nil {
				log.Errorf("获取审批日志保存限制失败 %v", err)
				return
			}
			// 查询超时日志并删除
			var ids []string
			s.propertyRepository.DB.Model(&model.NewWorkOrderLog{}).Where("apply_time < ?", time.Now().AddDate(0, 0, -utils.StringToInt(flag.Value))).Pluck("id", &ids)
			if len(ids) > 0 {
				s.propertyRepository.DB.Delete(&model.NewWorkOrderLog{}, "id in (?)", ids)
			}
		}
		{
			// 运维日志
			flag, err := s.propertyRepository.FindByName("operation_and_maintenance_log_save_limit")
			if err != nil {
				log.Errorf("获取运维日志保存限制失败 %v", err)
				return
			}
			// 查询超时日志并删除
			var ids []int64
			s.propertyRepository.DB.Model(&model.OperationAndMaintenanceLog{}).Where("logout_time < ?", time.Now().AddDate(0, 0, -utils.StringToInt(flag.Value))).Pluck("id", &ids)
			if len(ids) > 0 {
				s.propertyRepository.DB.Delete(&model.OperationAndMaintenanceLog{}, "id in (?)", ids)
			}
		}
		{
			// 自动任务日志
			flag, err := s.propertyRepository.FindByName("auto_task_log_save_limit")
			if err != nil {
				log.Errorf("获取自动任务日志保存限制失败 %v", err)
				return
			}
			// 查询超时日志并删除
			var ids []int64
			s.propertyRepository.DB.Model(&model.NewJobLog{}).Where("end_at < ?", time.Now().AddDate(0, 0, -utils.StringToInt(flag.Value))).Pluck("id", &ids)
			if len(ids) > 0 {
				s.propertyRepository.DB.Delete(&model.NewJobLog{}, "id in (?)", ids)
			}
		}
		{
			// 改密日志
			flag, err := s.propertyRepository.FindByName("encrypt_log_save_limit")
			if err != nil {
				log.Errorf("获取改密日志保存限制失败 %v", err)
				return
			}
			// 查询超时日志并删除
			var ids []int
			s.propertyRepository.DB.Model(&model.PasswdChangeResult{}).Where("change_time < ?", time.Now().AddDate(0, 0, -utils.StringToInt(flag.Value))).Pluck("id", &ids)
			if len(ids) > 0 {
				s.propertyRepository.DB.Delete(&model.PasswdChangeResult{}, "id in (?)", ids)
			}
		}
		{
			// 系统告警日志
			flag, err := s.propertyRepository.FindByName("system_alert_log_save_limit")
			if err != nil {
				log.Errorf("获取系统告警日志保存限制失败 %v", err)
				return
			}
			// 查询超时日志并删除
			var ids []int64
			s.propertyRepository.DB.Model(&model.SystemAlarmLog{}).Where("alarm_time < ?", time.Now().AddDate(0, 0, -utils.StringToInt(flag.Value))).Pluck("id", &ids)
			if len(ids) > 0 {
				s.propertyRepository.DB.Delete(&model.SystemAlarmLog{}, "id in (?)", ids)
			}
		}
		{
			// 操作告警日志
			flag, err := s.propertyRepository.FindByName("operae_alert_log_save_limit")
			if err != nil {
				log.Errorf("获取操作告警日志保存限制失败 %v", err)
				return
			}
			// 查询超时日志并删除
			var ids []int64
			s.propertyRepository.DB.Model(&model.OperateAlarmLog{}).Where("alarm_time < ?", time.Now().AddDate(0, 0, -utils.StringToInt(flag.Value))).Pluck("id", &ids)
			if len(ids) > 0 {
				s.propertyRepository.DB.Delete(&model.OperateAlarmLog{}, "id in (?)", ids)
			}
		}
	})
	if err != nil {
		log.Errorf("AddJob Error: %v", err)
		return err
	}
	global.SCHEDULER.StartAsync()
	//log.Debugf("开启计划任务[删除超时日志], 运行中计划任务数量[%v]", global.SCHEDULER.Len())
	return nil
}

// WriteSysUsage 写入系统利用率日志
func (s NewJobService) WriteSysUsage() error {
	_ = global.SCHEDULER.RemoveByTag(constant.SystemUsagePercentTag)
	_, err := global.SCHEDULER.Every(5).Minute().Tag(constant.SystemUsagePercentTag).Do(func() {
		{
			twoPoint := func(num float64) float64 {
				num, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", num), 64)
				return num
			}
			cpuPercent, err := cpu.Percent(time.Second, false)
			if err != nil {
				log.Errorf("cpu.Percent Error: %v", err)
			}
			// 内存占用
			memPercent, err := mem.VirtualMemory()
			if err != nil {
				log.Errorf("mem.VirtualMemory Error: %v", err)
			}
			// 磁盘占用
			diskPercent, err := disk.Usage("/")
			if err != nil {
				log.Errorf("disk.Usage Error: %v", err)
			}
			var cpuUsage = model.UsageCpu{
				Percent:  twoPoint(cpuPercent[0]),
				Used:     uint64(cpuPercent[0]),
				Free:     100 - uint64(cpuPercent[0]),
				Total:    100,
				Datetime: utils.NowJsonTime(),
			}
			var memUsage = model.UsageMem{
				Percent:  twoPoint(memPercent.UsedPercent),
				Total:    memPercent.Total,
				Used:     memPercent.Used,
				Free:     memPercent.Free,
				Datetime: utils.NowJsonTime(),
			}
			var diskUsage = model.UsageDisk{
				Percent:  twoPoint(diskPercent.UsedPercent),
				Total:    diskPercent.Total,
				Used:     diskPercent.Used,
				Free:     diskPercent.Free,
				Datetime: utils.NowJsonTime(),
			}
			err = s.propertyRepository.CreatCpuUsage(&cpuUsage)
			if err != nil {
				log.Errorf("CreatCpuUsage Error: %v", err)
			}
			err = s.propertyRepository.CreatMemUsage(&memUsage)
			if err != nil {
				log.Errorf("CreatMemUsage Error: %v", err)
			}
			err = s.propertyRepository.CreatDiskUsage(&diskUsage)
			if err != nil {
				log.Errorf("CreatDiskUsage Error: %v", err)
			}
		}
	})
	if err != nil {
		log.Errorf("AddJob Error: %v", err)
		return err
	}
	log.Infof("开启计划任务 [系统利用率写入], 运行中计划任务数量: [%d]", len(global.SCHEDULER.Jobs()))
	return nil
}

// AlarmConfigurationPerMonitoring 执行监控系统告警配置-系统利用率-并执行相关动作
func (s NewJobService) AlarmConfigurationPerMonitoring() error {
	// 查询所有的系统性能配置
	alarmSysConfigMap, err := s.propertyRepository.FindMapByNames(constant.AlarmConfigPerformance)
	if nil != err {
		log.Errorf("系统配置-告警配置: 获取系统性能告警配置失败")
	}
	_ = global.SCHEDULER.RemoveByTag(constant.AlarmConfigPerformanceTag)
	_, err = global.SCHEDULER.Every(5).Minute().Tag(constant.AlarmConfigPerformanceTag).Do(func() {
		{
			cpuPercent, err := cpu.Percent(time.Second, false)
			if err != nil {
				log.Errorf("cpu.Percent Error: %v", err)
			}
			if cpuPercent[0] >= utils.Str2Float64(alarmSysConfigMap["cpu-max"]) {
				text := "CPU:" + fmt.Sprintf("%.2f%%", cpuPercent[0]) + ",阈值:" + alarmSysConfigMap["cpu-max"] + "%"
				var result string
				if *utils.StrToBoolPtr(alarmSysConfigMap["cpu-msg"]) {
					if err := MessageService.SendAdminMessage(MessageService{}, "CPU超过阈值", alarmSysConfigMap["cpu-level"], text, constant.AlertMessage); err != nil {
						log.Errorf("系统CPU利用率发送告警信息失败: %v", err)
						result += "消息告警失败 "
					} else {
						result += "消息告警成功 "
					}
				}
				if *utils.StrToBoolPtr(alarmSysConfigMap["cpu-mail"]) {
					if err := SysConfigService.SendMailToAdmin(SysConfigService{propertyRepository: s.propertyRepository}, "CPU超过阈值", text); err != nil {
						log.Errorf("系统CPU利用率发送告警邮件失败: %v", err)
						result += "邮件告警失败 "
					} else {
						result += "邮件告警成功 "
					}
				}
				if *utils.StrToBoolPtr(alarmSysConfigMap["cpu-syslog"]) {
					if err := SysConfigService.SendSyslog(SysConfigService{propertyRepository: s.propertyRepository}, "CPU超过阈值:["+text+"]等级: "+alarmSysConfigMap["cpu-level"]); err != nil {
						log.Errorf("系统CPU利用率发送SYSLOG日志失败: %v", err)
						result += "SYSLOG告警失败 "
					} else {
						result += "SYSLOG告警成功 "
					}
				}
				sysAlarmLog := model.SystemAlarmLog{
					AlarmTime: utils.NowJsonTime(),
					Content:   text,
					Level:     alarmSysConfigMap["cpu-level"],
					Result:    result,
					Strategy:  "CPU使用率",
				}
				if err := s.operateAlarmLogRepository.CreateSystemAlarmLog(context.TODO(), &sysAlarmLog); err != nil {
					log.Errorf("系统告警日志记录失败: %v", err)
				}
			}
		}
		{
			// 内存占用
			memPercent, err := mem.VirtualMemory()
			if err != nil {
				log.Errorf("mem.VirtualMemory Error: %v", err)
			}
			if memPercent.UsedPercent >= utils.Str2Float64(alarmSysConfigMap["mem-max"]) {
				var result string
				text := "内存:" + fmt.Sprintf("%.2f%%", memPercent.UsedPercent) + ",阈值:" + alarmSysConfigMap["mem-max"] + "%"
				if *utils.StrToBoolPtr(alarmSysConfigMap["cpu-msg"]) {
					if err := MessageService.SendAdminMessage(MessageService{}, "内存超过阈值", alarmSysConfigMap["mem-level"], text, constant.AlertMessage); err != nil {
						log.Errorf("系统内存利用率发送告警信息失败: %v", err)
						result += "消息告警失败 "
					} else {
						result += "消息告警成功 "
					}
				}
				if *utils.StrToBoolPtr(alarmSysConfigMap["mem-mail"]) {
					if err := SysConfigService.SendMailToAdmin(SysConfigService{propertyRepository: s.propertyRepository}, "内存超过阈值", text); err != nil {
						log.Errorf("系统内存利用率发送告警邮件失败: %v", err)
						result += "邮件告警失败 "
					} else {
						result += "邮件告警成功 "
					}
				}
				if *utils.StrToBoolPtr(alarmSysConfigMap["mem-syslog"]) {
					if err := SysConfigService.SendSyslog(SysConfigService{propertyRepository: s.propertyRepository}, "内存超过阈值:["+text+"]等级: "+alarmSysConfigMap["mem-level"]); err != nil {
						log.Errorf("系统内存利用率发送SYSLOG日志失败: %v", err)
						result += "SYSLOG告警失败 "
					} else {
						result += "SYSLOG告警成功 "
					}
				}
				sysAlarmLog := model.SystemAlarmLog{
					AlarmTime: utils.NowJsonTime(),
					Content:   text,
					Level:     alarmSysConfigMap["mem-level"],
					Result:    result,
					Strategy:  "内存使用率",
				}
				if err := s.operateAlarmLogRepository.CreateSystemAlarmLog(context.TODO(), &sysAlarmLog); err != nil {
					log.Errorf("系统告警日志记录失败: %v", err)
				}
			}
		}
		{
			// 磁盘占用
			diskPercent, err := disk.Usage("/")
			if err != nil {
				log.Errorf("disk.Usage Error: %v", err)
			}
			if diskPercent.UsedPercent >= utils.Str2Float64(alarmSysConfigMap["disk-max"]) {
				text := "磁盘:" + fmt.Sprintf("%.2f%%", diskPercent.UsedPercent) + ",阈值:" + alarmSysConfigMap["disk-max"] + "%"
				var result string
				if *utils.StrToBoolPtr(alarmSysConfigMap["cpu-msg"]) {
					if err := MessageService.SendAdminMessage(MessageService{}, "磁盘超过阈值", alarmSysConfigMap["disk-level"], text, constant.AlertMessage); err != nil {
						log.Errorf("系统磁盘利用率发送告警信息失败: %v", err)
						result += "消息告警失败 "
					} else {
						result += "消息告警成功 "
					}
				}
				if *utils.StrToBoolPtr(alarmSysConfigMap["disk-mail"]) {
					if err := SysConfigService.SendMailToAdmin(SysConfigService{propertyRepository: s.propertyRepository}, "磁盘超过阈值", text); err != nil {
						log.Errorf("系统磁盘利用率发送告警邮件失败: %v", err)
						result += "邮件告警失败 "
					} else {
						result += "邮件告警成功 "
					}
				}
				if *utils.StrToBoolPtr(alarmSysConfigMap["disk-syslog"]) {
					if err := SysConfigService.SendSyslog(SysConfigService{propertyRepository: s.propertyRepository}, "磁盘超过阈值:["+text+"]等级: "+alarmSysConfigMap["disk-level"]); err != nil {
						log.Errorf("磁盘利用率发送SYSLOG日志失败: %v", err)
						result += "SYSLOG告警失败 "
					} else {
						result += "SYSLOG告警成功 "
					}
				}
				sysAlarmLog := model.SystemAlarmLog{
					AlarmTime: utils.NowJsonTime(),
					Content:   text,
					Level:     alarmSysConfigMap["disk-level"],
					Result:    result,
					Strategy:  "磁盘使用率",
				}
				if err := s.operateAlarmLogRepository.CreateSystemAlarmLog(context.TODO(), &sysAlarmLog); err != nil {
					log.Errorf("系统告警日志记录失败: %v", err)
				}
			}
		}
		{
			// 数据盘
			diskDataPercent, err := disk.Usage("/data")
			if err != nil {
				log.Errorf("disk.Usage Error: %v", err)
			}
			if diskDataPercent.UsedPercent >= utils.Str2Float64(alarmSysConfigMap["data-max"]) {
				text := "数据盘:" + fmt.Sprintf("%.2f%%", diskDataPercent.UsedPercent) + ",阈值:" + alarmSysConfigMap["data-max"] + "%"
				var result string
				if *utils.StrToBoolPtr(alarmSysConfigMap["data-msg"]) {
					if err := MessageService.SendAdminMessage(MessageService{}, "数据盘超过阈值", alarmSysConfigMap["data-level"], text, constant.AlertMessage); err != nil {
						log.Errorf("系统数据盘利用率发送告警信息失败: %v", err)
						result += "消息告警失败 "
					} else {
						result += "消息告警成功 "
					}
				}
				if *utils.StrToBoolPtr(alarmSysConfigMap["data-mail"]) {
					if err := SysConfigService.SendMailToAdmin(SysConfigService{propertyRepository: s.propertyRepository}, "数据盘超过阈值", text); err != nil {
						log.Errorf("系统数据盘利用率发送告警邮件失败: %v", err)
						result += "邮件告警失败 "
					} else {
						result += "邮件告警成功 "
					}
				}
				if *utils.StrToBoolPtr(alarmSysConfigMap["data-syslog"]) {
					if err := SysConfigService.SendSyslog(SysConfigService{propertyRepository: s.propertyRepository}, "数据盘超过阈值:["+text+"]等级: "+alarmSysConfigMap["data-level"]); err != nil {
						log.Errorf("系统数据盘利用率发送SYSLOG日志失败: %v", err)
						result += "SYSLOG告警失败 "
					} else {
						result += "SYSLOG告警成功 "
					}
				}
				sysAlarmLog := model.SystemAlarmLog{
					AlarmTime: utils.NowJsonTime(),
					Content:   text,
					Level:     alarmSysConfigMap["data-level"],
					Result:    result,
					Strategy:  "数据盘使用率",
				}
				if err := s.operateAlarmLogRepository.CreateSystemAlarmLog(context.TODO(), &sysAlarmLog); err != nil {
					log.Errorf("系统告警日志记录失败: %v", err)
				}
			}
		}
	})
	if err != nil {
		log.Errorf("AddJob Error: %v", err)
		return err
	}
	global.SCHEDULER.StartAsync()
	log.Infof("开启计划任务 [系统利用率监控], 运行中计划任务数量: [%d]", len(global.SCHEDULER.Jobs()))
	return nil
}

// AlarmConfigurationAccMonitoring 执行监控系统告警配置-系统访问量-并执行相关动作
func (s NewJobService) AlarmConfigurationAccMonitoring() error {
	// 查询所有的访问量配置
	alarmSysConfigMap, err := s.propertyRepository.FindMapByNames(constant.AlarmConfigAccess)
	if nil != err {
		log.Errorf("系统配置-告警配置: 获取系统性能告警配置失败")
	}
	_ = global.SCHEDULER.RemoveByTag(constant.AlarmConfigAccessTag)
	_, err = global.SCHEDULER.Every(1).Minute().Tag(constant.AlarmConfigAccessTag).Do(func() {
		{
			// 获取同时在线用户数
			cacheM := global.Cache.Items()
			var count int
			for k := range cacheM {
				if strings.Contains(k, constant.Token) {
					count++
				}
				configCount, _ := strconv.Atoi(alarmSysConfigMap["user-max"])
				if count >= configCount {
					text := "用户访问量:" + alarmSysConfigMap["user-max"] + ",阈值:" + alarmSysConfigMap["user-max"] + "%"
					var result string
					if *utils.StrToBoolPtr(alarmSysConfigMap["user-msg"]) {
						if err := MessageService.SendAdminMessage(MessageService{}, "用户访问量超过阈值", alarmSysConfigMap["user-level"], text, constant.AlertMessage); err != nil {
							log.Errorf("用户访问量发送告警信息失败: %v", err)
							result += "邮件告警失败 "
						} else {
							result += "邮件告警成功 "
						}
					}
					if *utils.StrToBoolPtr(alarmSysConfigMap["user-mail"]) {
						if err := SysConfigService.SendMailToAdmin(SysConfigService{propertyRepository: s.propertyRepository}, "用户访问量超过阈值", text); err != nil {
							log.Errorf("用户访问量发送告警邮件失败: %v", err)
							result += "邮件告警失败 "
						} else {
							result += "邮件告警成功 "
						}
					}
					if *utils.StrToBoolPtr(alarmSysConfigMap["user-syslog"]) {
						if err := SysConfigService.SendSyslog(SysConfigService{propertyRepository: s.propertyRepository}, "用户访问量超过阈值:["+text+"]等级: "+alarmSysConfigMap["user-level"]); err != nil {
							log.Errorf("用户访问量发送SYSLOG日志失败: %v", err)
							result += "SYSLOG告警失败 "
						} else {
							result += "SYSLOG告警成功 "
						}
					}
					sysAlarmLog := model.SystemAlarmLog{
						AlarmTime: utils.NowJsonTime(),
						Content:   text,
						Level:     alarmSysConfigMap["user-level"],
						Result:    result,
						Strategy:  "用户访问量",
					}
					if err := s.operateAlarmLogRepository.CreateSystemAlarmLog(context.TODO(), &sysAlarmLog); err != nil {
						log.Errorf("系统告警日志记录失败: %v", err)
					}
				}
			}
		}
		{
			// 获取同时在线的SSH数
			count, err := s.newSessionRepository.GetSSHOnlineCount(context.TODO())
			if err != nil {
				log.Errorf("获取SSH在线数失败: %v", err)
			}
			configCount, _ := strconv.Atoi(alarmSysConfigMap["ssh-max"])
			if count >= int64(configCount) {
				text := "SSH访问量:" + alarmSysConfigMap["ssh-max"] + ",阈值:" + alarmSysConfigMap["ssh-max"]
				var result string
				if *utils.StrToBoolPtr(alarmSysConfigMap["ssh-msg"]) {
					if err := MessageService.SendAdminMessage(MessageService{}, "SSH访问量超过阈值", alarmSysConfigMap["ssh-level"], text, constant.AlertMessage); err != nil {
						log.Errorf("SSH访问量发送告警信息失败: %v", err)
						result += "邮件告警失败 "
					} else {
						result += "邮件告警成功 "
					}
				}
				if *utils.StrToBoolPtr(alarmSysConfigMap["ssh-mail"]) {
					if err := SysConfigService.SendMailToAdmin(SysConfigService{propertyRepository: s.propertyRepository}, "SSH访问量超过阈值", text); err != nil {
						log.Errorf("SSH访问量发送告警邮件失败: %v", err)
						result += "邮件告警失败 "
					} else {
						result += "邮件告警成功 "
					}
				}
				if *utils.StrToBoolPtr(alarmSysConfigMap["ssh-syslog"]) {
					if err := SysConfigService.SendSyslog(SysConfigService{propertyRepository: s.propertyRepository}, "SSH协议访问量超过阈值:["+text+"]等级: "+alarmSysConfigMap["ssh-level"]); err != nil {
						log.Errorf("SSH访问量发送SYSLOG日志失败: %v", err)
						result += "SYSLOG告警失败 "
					} else {
						result += "SYSLOG告警成功 "
					}
				}
				sysAlarmLog := model.SystemAlarmLog{
					AlarmTime: utils.NowJsonTime(),
					Content:   text,
					Level:     alarmSysConfigMap["ssh-level"],
					Result:    result,
					Strategy:  "SSH访问量",
				}
				if err := s.operateAlarmLogRepository.CreateSystemAlarmLog(context.TODO(), &sysAlarmLog); err != nil {
					log.Errorf("系统告警日志记录失败: %v", err)
				}
			}

		}
		{
			// 获取同时在线的RDP数
			count, err := s.newSessionRepository.GetRDPOnlineCount(context.TODO())
			if err != nil {
				log.Errorf("获取RDP在线数失败: %v", err)
			}
			configCount, _ := strconv.Atoi(alarmSysConfigMap["rdp-max"])
			if count >= int64(configCount) {
				text := "RDP访问量:" + alarmSysConfigMap["rdp-max"] + ",阈值:" + alarmSysConfigMap["rdp-max"]
				var result string
				if *utils.StrToBoolPtr(alarmSysConfigMap["rdp-msg"]) {
					if err := MessageService.SendAdminMessage(MessageService{}, "RDP访问量超过阈值", alarmSysConfigMap["rdp-level"], text, constant.AlertMessage); err != nil {
						log.Errorf("RDP访问量发送告警信息失败: %v", err)
						result += "邮件告警失败 "
					} else {
						result += "邮件告警成功 "
					}
				}
				if *utils.StrToBoolPtr(alarmSysConfigMap["rdp-mail"]) {
					if err := SysConfigService.SendMailToAdmin(SysConfigService{propertyRepository: s.propertyRepository}, "RDP访问量超过阈值", text); err != nil {
						log.Errorf("RDP访问量发送告警邮件失败: %v", err)
						result += "邮件告警失败 "
					} else {
						result += "邮件告警成功 "
					}
				}
				if *utils.StrToBoolPtr(alarmSysConfigMap["rdp-syslog"]) {
					if err := SysConfigService.SendSyslog(SysConfigService{propertyRepository: s.propertyRepository}, "RDP协议访问量超过阈值:["+text+"]等级: "+alarmSysConfigMap["rdp-level"]); err != nil {
						log.Errorf("RDP访问量发送SYSLOG日志失败: %v", err)
						result += "SYSLOG告警失败 "
					} else {
						result += "SYSLOG告警成功 "
					}
				}
				sysAlarmLog := model.SystemAlarmLog{
					AlarmTime: utils.NowJsonTime(),
					Content:   text,
					Level:     alarmSysConfigMap["rdp-level"],
					Result:    result,
					Strategy:  "RDP访问量",
				}
				if err := s.operateAlarmLogRepository.CreateSystemAlarmLog(context.TODO(), &sysAlarmLog); err != nil {
					log.Errorf("系统告警日志记录失败: %v", err)
				}
			}
		}
		{
			// 获取同时在线的VNC数
			count, err := s.newSessionRepository.GetVNCOnlineCount(context.TODO())
			if err != nil {
				log.Errorf("获取VNC在线数失败: %v", err)
			}
			configCount, _ := strconv.Atoi(alarmSysConfigMap["vnc-max"])
			if count >= int64(configCount) {
				text := "VNC访问量:" + alarmSysConfigMap["vnc-max"] + ",阈值:" + alarmSysConfigMap["vnc-max"]
				var result string
				if *utils.StrToBoolPtr(alarmSysConfigMap["vnc-msg"]) {
					if err := MessageService.SendAdminMessage(MessageService{}, "VNC访问量超过阈值", alarmSysConfigMap["vnc-level"], text, constant.AlertMessage); err != nil {
						log.Errorf("VNC访问量发送告警信息失败: %v", err)
						result += "邮件告警失败 "
					} else {
						result += "邮件告警成功 "
					}
				}
				if *utils.StrToBoolPtr(alarmSysConfigMap["vnc-mail"]) {
					if err := SysConfigService.SendMailToAdmin(SysConfigService{propertyRepository: s.propertyRepository}, "VNC访问量超过阈值", text); err != nil {
						log.Errorf("VNC访问量发送告警邮件失败: %v", err)
						result += "邮件告警失败 "
					} else {
						result += "邮件告警成功 "
					}
				}
				if *utils.StrToBoolPtr(alarmSysConfigMap["vnc-syslog"]) {
					if err := SysConfigService.SendSyslog(SysConfigService{propertyRepository: s.propertyRepository}, "VNC访问量超过阈值:["+text+"]等级: "+alarmSysConfigMap["vnc-level"]); err != nil {
						log.Errorf("VNC访问量发送SYSLOG日志失败: %v", err)
						result += "SYSLOG告警失败 "
					} else {
						result += "SYSLOG告警成功 "
					}
				}
				sysAlarmLog := model.SystemAlarmLog{
					AlarmTime: utils.NowJsonTime(),
					Content:   text,
					Level:     alarmSysConfigMap["vnc-level"],
					Result:    result,
					Strategy:  "VNC访问量",
				}
				if err := s.operateAlarmLogRepository.CreateSystemAlarmLog(context.TODO(), &sysAlarmLog); err != nil {
					log.Errorf("系统告警日志记录失败: %v", err)
				}
			}
		}
		{
			// 获取同时在线的TELNET数
			count, err := s.newSessionRepository.GetTelnetOnlineCount(context.TODO())
			if err != nil {
				log.Errorf("获取TELNET在线数失败: %v", err)
			}
			configCount, _ := strconv.Atoi(alarmSysConfigMap["telnet-max"])
			if count >= int64(configCount) {
				text := "TELNET访问量:" + alarmSysConfigMap["telnet-max"] + ",阈值:" + alarmSysConfigMap["telnet-max"]
				var result string
				if *utils.StrToBoolPtr(alarmSysConfigMap["telnet-msg"]) {
					if err := MessageService.SendAdminMessage(MessageService{}, "TELNET访问量超过阈值", alarmSysConfigMap["telnet-level"], text, constant.AlertMessage); err != nil {
						log.Errorf("TELNET访问量发送告警信息失败: %v", err)
						result += "邮件告警失败 "
					} else {
						result += "邮件告警成功 "
					}
				}
				if *utils.StrToBoolPtr(alarmSysConfigMap["telnet-mail"]) {
					if err := SysConfigService.SendMailToAdmin(SysConfigService{propertyRepository: s.propertyRepository}, "TELNET访问量超过阈值", text); err != nil {
						log.Errorf("TELNET访问量发送告警邮件失败: %v", err)
						result += "邮件告警失败 "
					} else {
						result += "邮件告警成功 "
					}
				}
				if *utils.StrToBoolPtr(alarmSysConfigMap["telnet-syslog"]) {
					if err := SysConfigService.SendSyslog(SysConfigService{propertyRepository: s.propertyRepository}, "TELNET访问量超过阈值:["+text+"]等级: "+alarmSysConfigMap["telnet-level"]); err != nil {
						log.Errorf("TELNET访问量发送SYSLOG日志失败: %v", err)
						result += "SYSLOG告警失败 "
					} else {
						result += "SYSLOG告警成功 "
					}
				}
				sysAlarmLog := model.SystemAlarmLog{
					AlarmTime: utils.NowJsonTime(),
					Content:   text,
					Level:     alarmSysConfigMap["telnet-level"],
					Result:    result,
					Strategy:  "TELNET访问量",
				}
				if err := s.operateAlarmLogRepository.CreateSystemAlarmLog(context.TODO(), &sysAlarmLog); err != nil {
					log.Errorf("系统告警日志记录失败: %v", err)
				}
			}
			{
				// 获取同时在线的应用数
				count, err := s.appSessionRepository.GetAppOnlineCount(context.TODO())
				if err != nil {
					log.Errorf("获取应用在线数失败: %v", err)
				}
				configCount, _ := strconv.Atoi(alarmSysConfigMap["app-max"])
				if count >= int64(configCount) {
					text := "应用访问量:" + alarmSysConfigMap["app-max"] + ",阈值:" + alarmSysConfigMap["app-max"]
					var result string
					if *utils.StrToBoolPtr(alarmSysConfigMap["app-msg"]) {
						if err := MessageService.SendAdminMessage(MessageService{}, "应用访问量超过阈值", alarmSysConfigMap["app-level"], text, constant.AlertMessage); err != nil {
							log.Errorf("应用访问量发送告警信息失败: %v", err)
							result += "邮件告警失败 "
						} else {
							result += "邮件告警成功 "
						}
					}
					if *utils.StrToBoolPtr(alarmSysConfigMap["app-mail"]) {
						if err := SysConfigService.SendMailToAdmin(SysConfigService{propertyRepository: s.propertyRepository}, "应用访问量超过阈值", text); err != nil {
							log.Errorf("应用访问量发送告警邮件失败: %v", err)
							result += "邮件告警失败 "
						} else {
							result += "邮件告警成功 "
						}
					}
					if *utils.StrToBoolPtr(alarmSysConfigMap["app-syslog"]) {
						if err := SysConfigService.SendSyslog(SysConfigService{propertyRepository: s.propertyRepository}, "应用数超过阈值:["+text+"]等级: "+alarmSysConfigMap["app-level"]); err != nil {
							log.Errorf("应用访问量发送SYSLOG日志失败: %v", err)
							result += "SYSLOG告警失败 "
						} else {
							result += "SYSLOG告警成功 "
						}
					}
					sysAlarmLog := model.SystemAlarmLog{
						AlarmTime: utils.NowJsonTime(),
						Content:   text,
						Level:     alarmSysConfigMap["app-level"],
						Result:    result,
						Strategy:  "应用访问量",
					}
					if err := s.operateAlarmLogRepository.CreateSystemAlarmLog(context.TODO(), &sysAlarmLog); err != nil {
						log.Errorf("系统告警日志记录失败: %v", err)
					}
				}
			}
		}
	})
	if err != nil {
		log.Errorf("AddJob Error: %v", err)
		return err
	}
	global.SCHEDULER.StartAsync()
	log.Infof("开启计划任务 [系统访问量监控], 运行中计划任务数量: [%d]", len(global.SCHEDULER.Jobs()))
	return nil
}

// RunRegularReport 执行定期策略
func (s NewJobService) RunRegularReport(id string) error {
	regularReport, err := s.regularReportRepository.FindById(id)
	if err != nil {
		return err
	}
	switch regularReport.PeriodicType {
	case "day":
		_, err := global.SCHEDULER.Every(1).Day().Tag(id).Do(func() {
			{
				end := time.Now().Format("2006-01-02")
				start := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
				s.regularReportExec(regularReport, start, end)
			}
		})
		if err != nil {
			log.Errorf("AddJob Error: %v", err)
			return err
		}
	case "week":
		_, err := global.SCHEDULER.Every(1).Weekday(time.Weekday(regularReport.Periodic)).Do(func() {
			{
				end := time.Now().Format("2006-01-02")
				start := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
				s.regularReportExec(regularReport, start, end)
			}
		})
		if err != nil {
			log.Errorf("AddJob Error: %v", err)
			return err
		}
	case "month":
		_, err := global.SCHEDULER.Every(1).Month(int(regularReport.Periodic)).Do(func() {
			{
				end := time.Now().Format("2006-01-02")
				start := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
				s.regularReportExec(regularReport, start, end)
			}
		})
		if err != nil {
			log.Errorf("AddJob Error: %v", err)
			return err
		}
	default:
		log.Errorf("定期策略执行失败, 未知的周期类型[%v]", regularReport.PeriodicType)
		return errors.New("未知的周期类型")
	}
	global.SCHEDULER.StartAsync()
	return nil
}

func (s NewJobService) regularReportExec(regularReport model.RegularReport, start, end string) {
	// 1.Protocol 2.User 3.LoginAttempt 4.Asset 5.Session 6.Command 7.Alarm
	// 1.执行协议访问统计 2. 执行用户访问统计 3. 执行登录尝试统计 4. 执行资产访问统计 5. 执行会话统计 6. 执行命令统计 7. 执行告警统计
	if *regularReport.IsProtocol {
		// 协议访问统计
		fileName := time.Now().Format("2006-01-02")
		protocolCountByDay, err := s.newSessionRepository.GetProtocolCountByDay(context.TODO(), start, end, "%Y-%m-%d")
		if err != nil {
			log.Errorf("GetProtocolCountByDayEndpoint error: %v", err)
			log.Errorf("定期报表-报表查看: 下载失败")
		}
		var protocolCountExport []model.ProtocolCountExport
		for i := range protocolCountByDay {
			if protocolCountByDay[i].Tcp != 0 {
				protocolCountByDay[i].Total += protocolCountByDay[i].Tcp
			}
			protocolCountExport = append(protocolCountExport, model.ProtocolCountExport{
				Daytime: protocolCountByDay[i].Daytime,
				Ssh:     protocolCountByDay[i].Ssh,
				Rdp:     protocolCountByDay[i].Rdp,
				Telnet:  protocolCountByDay[i].Telnet,
				Vnc:     protocolCountByDay[i].Vnc,
				App:     protocolCountByDay[i].App,
				Tcp:     protocolCountByDay[i].Tcp,
				Total:   protocolCountByDay[i].Total,
			})
		}
		// 查询详细数据
		var sessionDetails, loginLogDetails []model.LoginDetails
		sessionDetailTemp, err := s.newSessionRepository.GetLoginDetails(context.TODO(), start+" 00:00:00", end+" 23:59:59", "")
		if err != nil {
			log.Errorf("GetLoginDetailsEndpoint error: %v", err)
		}
		for _, v := range sessionDetailTemp {
			sessionDetails = append(sessionDetails, *v.ToLoginDetailsDto())
		}
		loginLogDetailTemp, err := s.loginLogRepository.GetLoginDetailsStatist(start+" 00:00:00", end+" 23:59:59")
		if err != nil {
			log.Errorf("GetLoginDetailsEndpoint error: %v", err)
		}
		for _, v := range loginLogDetailTemp {
			loginLogDetails = append(loginLogDetails, *v.ToLoginDetailsDto())
		}
		sessionDetails = append(sessionDetails, loginLogDetails...)
		// 将结构体数组转换为字符串数组
		var data1, data2 [][]string
		for _, v := range protocolCountExport {
			data := utils.Struct2StrArr(v)
			data1 = append(data1, data)
		}
		data1Title := []string{"日期", "SSH", "RDP", "TELNET", "VNC", "应用发布", "前台", "合计"}

		for _, v := range sessionDetails {
			data := utils.Struct2StrArr(v)
			data2 = append(data2, data)
		}
		data2Title := []string{"登录时间", "用户名", "姓名", "来源地址", "协议", "结果", "描述"}
		file, err := utils.ExportCsv(data1Title, data2Title, data1, data2)
		if err != nil {
			log.Errorf("ExportCsv error: %v", err)
			log.Errorf("定期报表-报表查看: 下载失败")
		}
		// 保存文件
		fileName += "_协议访问统计.csv"
		err = utils.SaveFile(constant.RegularReportProtocolPath, fileName, file)
		if err != nil {
			log.Errorf("SaveFile error: %v", err)
			log.Errorf("定期报表-报表查看: 下载失败")
		}

		err = s.regularReportRepository.CreateRegularReportLog(regularReport.Name, regularReport.PeriodicType, "protocol", fileName)
		if err != nil {
			return
		}
	}
	if *regularReport.IsUser {
		fileName := time.Now().Format("2006-01-02")
		var (
			header1      = []string{"时间", "用户名", "真实姓名", "SSH", "RDP", "TELNET", "VNC", "应用发布", "前台", "合计"}
			header2      = []string{"登陆时间", "用户名", "姓名", "来源地址", "协议", "结果", "描述"}
			data1, data2 [][]string
		)
		content1, err := s.userAccessStatisticsRepository.GetUserAccessStatisticsByDay(context.TODO(), start, end)
		if err != nil {
			log.Errorf("查询用户访问统计数据失败: %v", err)
		}
		data1 = make([][]string, len(content1))
		for i, v := range content1 {
			data1[i] = utils.Struct2StrArr(v)
		}
		content2, err := s.userAccessStatisticsRepository.GetUserAccessStatistics(context.TODO(), start, end)
		if err != nil {
			log.Errorf("查询用户访问详细数据失败: %v", err)
		}
		data2 = make([][]string, len(content2))
		for i, v := range content2 {
			data2[i] = utils.Struct2StrArr(v)
		}

		// 导出csv
		file, err := utils.ExportCsv(header1, header2, data1, data2)
		if err != nil {
			log.Errorf("ExportCsv error: %v", err)
			log.Errorf("定期报表-报表查看: 下载失败")
		}
		// 保存文件
		fileName += "_用户访问统计.csv"
		err = utils.SaveFile(constant.RegularReportUserPath, fileName, file)
		if err != nil {
			log.Errorf("SaveFile error: %v", err)
			log.Errorf("定期报表-报表查看: 下载失败")
		}
		err = s.regularReportRepository.CreateRegularReportLog(regularReport.Name, regularReport.PeriodicType, "user", fileName)
		if err != nil {
			return
		}
	}
	if *regularReport.IsLogin {
		fileName := time.Now().Format("2006-01-02")
		// 查询数据
		content1, err := s.userAccessStatisticsRepository.GetLoginStatistics(context.TODO(), start, end)
		if err != nil {
			log.Errorf("获取登陆尝试统计数据失败: %v", err)
		}
		content2, err := s.userAccessStatisticsRepository.GetLoginStatisticsDetail(context.TODO(), start, end)
		if err != nil {
			log.Errorf("获取登陆尝试详细数据失败: %v", err)
		}
		// 处理数据
		header1 := []string{"时间", "用户数", "成功次数", "失败次数", "来源IP数", "总次数"}
		header2 := []string{"登陆时间", "用户名", "姓名", "来源地址", "结果", "描述"}
		var data1 = make([][]string, len(content1))
		var data2 = make([][]string, len(content2))
		for i, v := range content1 {
			data1[i] = utils.Struct2StrArr(v)
		}
		for i, v := range content2 {
			data2[i] = utils.Struct2StrArr(v)
		}
		file, err := utils.ExportCsv(header1, header2, data1, data2)
		if err != nil {
			log.Errorf("导出csv失败: %v", err)
		}
		// 保存文件
		fileName += "_尝试登录统计.csv"
		err = utils.SaveFile(constant.RegularReportLoginPath, fileName, file)
		if err != nil {
			log.Errorf("SaveFile error: %v", err)
			log.Errorf("定期报表-报表查看: 下载失败")
		}
		err = s.regularReportRepository.CreateRegularReportLog(regularReport.Name, regularReport.PeriodicType, "login", fileName)
		if err != nil {
			return
		}
	}
	if *regularReport.IsAsset {
		fileName := time.Now().Format("2006-01-02")
		t := utils.GetQueryType(start, end)
		var content1 []dto.AssetAccess
		var content2 []dto.AssetAccessExport
		fmt.Println(t)
		if t == 1 {
			c1, c2, err := s.operateReportRepository.GetOperateLoginLogExportByDay(context.TODO(), start, end)
			if err != nil {
				log.Errorf("获取资产访问统计数据失败: %v", err)
			}
			content1 = c1
			content2 = c2
		} else if t == 4 {
			c1, c2, err := s.operateReportRepository.GetOperateLoginLogExportByHour(context.TODO(), start, end)
			if err != nil {
				log.Errorf("获取资产访问统计数据失败: %v", err)
			}
			content1 = c1
			content2 = c2
		} else if t == 3 {
			c1, c2, err := s.operateReportRepository.GetOperateLoginLogExportByWeek(context.TODO(), start, end)
			if err != nil {
				log.Errorf("获取资产访问统计数据失败: %v", err)
			}
			content1 = c1
			content2 = c2
		} else if t == 2 {
			c1, c2, err := s.operateReportRepository.GetOperateLoginLogExportByMonth(context.TODO(), start, end)
			if err != nil {
				log.Errorf("获取资产访问统计数据失败: %v", err)
			}
			content1 = c1
			content2 = c2
		} else {
			log.Errorf("获取资产访问统计数据失败")
		}
		// 导出
		// 数据转化为[]string
		var data1 = make([][]string, len(content1))
		var data2 = make([][]string, len(content2))
		for i, v := range content1 {
			data1[i] = utils.Struct2StrArr(v)
		}
		for i, v := range content2 {
			data2[i] = utils.Struct2StrArr(v)
		}
		//fmt.Println(data1)
		//fmt.Println(data2)
		// 表头
		var header1 = []string{"时间", "设备数", "用户数"}
		var header2 = []string{"开始时间", "结束时间", "用户名", "姓名", "来源地址", "设备名称", "设备地址", "设备账号", "协议"}
		file, err := utils.ExportCsv(header1, header2, data1, data2)
		if err != nil {
			log.Errorf("导出csv失败: %v", err)
		}
		// 保存文件
		fileName += "_主机运维.csv"
		err = utils.SaveFile(constant.RegularReportAssetPath, fileName, file)
		if err != nil {
			log.Errorf("SaveFile error: %v", err)
			log.Errorf("定期报表-报表查看: 下载失败")
		}
		err = s.regularReportRepository.CreateRegularReportLog(regularReport.Name, regularReport.PeriodicType, "asset", fileName)
		if err != nil {
			return
		}
	}
	if *regularReport.IsSession {
		fileName := time.Now().Format("2006-01-02")
		var (
			header1 = []string{"设备名称", "设备地址", "时长(秒)"}
			header2 = []string{"开始时间", "结束时间", "用户名", "姓名", "来源地址", "设备名称", "设备地址", "设备账号", "协议"}
			data1   [][]string
			data2   [][]string
		)
		content, err := s.operateReportRepository.GetOperateSessionLogAsset(context.TODO(), start, end)
		if err != nil {
			log.Errorf("查询会话时长失败: %v", err)
		}
		data1 = make([][]string, len(content))
		for i, v := range content {
			data1[i] = []string{v.AssetName, v.AssetIP, strconv.Itoa(int(v.Time))}
		}
		content1, err := s.operateReportRepository.GetOperateSessionLogExport(context.TODO(), start, end)
		if err != nil {
			log.Errorf("查询会话时长失败: %v", err)
		}
		data2 = make([][]string, len(content1))
		for i, v := range content1 {
			data2[i] = utils.Struct2StrArr(v)
		}
		// 导出csv
		file, err := utils.ExportCsv(header1, header2, data1, data2)
		if err != nil {
			log.Errorf("下载csv失败: %v", err)
		}
		// 保存文件
		fileName += "_会话时长.csv"
		err = utils.SaveFile(constant.RegularReportSessionPath, fileName, file)
		if err != nil {
			log.Errorf("SaveFile error: %v", err)
			log.Errorf("定期报表-报表查看: 下载失败")
		}
		err = s.regularReportRepository.CreateRegularReportLog(regularReport.Name, regularReport.PeriodicType, "session", fileName)
		if err != nil {
			return
		}
	}
	if *regularReport.IsCommand {
		fileName := time.Now().Format("2006-01-02")
		commandRecord, err := s.operateReportRepository.GetOperateCommandCount(context.TODO(), start, end)
		if err != nil {
			log.Errorf("获取命令统计失败: %v", err)
		}
		commandRecordDetails, err := s.operateReportRepository.GetOperateCommandDetails(context.TODO(), start, end, "")
		if err != nil {
			log.Errorf("获取命令统计详情失败: %v", err)
		}
		var commandRecordDetailsExport = make([]dto.CommandStatisticsDetail, len(commandRecordDetails))
		for i, v := range commandRecordDetails {
			commandRecordDetailsExport[i].Created = v.Created.Format("2006-01-02 15:04:05")
			commandRecordDetailsExport[i].Content = v.Content
			commandRecordDetailsExport[i].Username = v.Username
			commandRecordDetailsExport[i].Nickname = v.Nickname
			commandRecordDetailsExport[i].ClientIp = v.ClientIp
			commandRecordDetailsExport[i].AssetName = v.AssetName
			commandRecordDetailsExport[i].AssetIp = v.AssetIp
			commandRecordDetailsExport[i].Passport = v.Passport
			commandRecordDetailsExport[i].Protocol = v.Protocol
		}
		var data1, data2 [][]string
		for _, v := range commandRecord {
			data := utils.Struct2StrArr(v)
			data1 = append(data1, data)
		}
		data1Title := []string{"命令", "次数"}

		for _, v := range commandRecordDetailsExport {
			data := utils.Struct2StrArr(v)
			data2 = append(data2, data)
		}
		data2Title := []string{"执行时间", "命令", "用户名", "姓名", "来源地址", "设备名称", "设备地址", "设备账号", "协议"}
		file, err := utils.ExportCsv(data1Title, data2Title, data1, data2)
		if err != nil {
			log.Errorf("导出命令统计详情失败: %v", err)
		}
		// 保存文件
		fileName += "_命令统计.csv"
		err = utils.SaveFile(constant.RegularReportCommandPath, fileName, file)
		if err != nil {
			log.Errorf("SaveFile error: %v", err)
			log.Errorf("定期报表-报表查看: 下载失败")
		}
		err = s.regularReportRepository.CreateRegularReportLog(regularReport.Name, regularReport.PeriodicType, "command", fileName)
		if err != nil {
			return
		}
	}
	if *regularReport.IsAlarm {
		fileName := time.Now().Format("2006-01-02")
		var (
			header1      = []string{"时间", "高", "中", "低"}
			header2      = []string{"告警时间", "用户名", "姓名", "来源地址", "设备地址", "设备账号", "协议", "触发策略", "告警级别"}
			data1, data2 [][]string
		)
		content1, err := s.operateAlarmLogRepository.FindByAlarmLogByDate(context.TODO(), start, end, "")
		if err != nil {
			log.Errorf("GetProtocolCountByDayEndpoint error: %v", err)
			log.Errorf("运维报表-告警统计: 导出[获取数据文件失败]")
		}
		content2, err := s.operateAlarmLogRepository.GetAlarmLogDetailsStatist(context.TODO(), start, end, "")
		if err != nil {
			log.Errorf("GetLoginDetailsEndpoint error: %v", err)
			log.Errorf("运维报表-告警统计: 导出[获取数据文件失败]")
		}
		data1 = make([][]string, len(content1))
		for i, v := range content1 {
			data1[i] = utils.Struct2StrArr(v)
		}
		data2 = make([][]string, len(content2))
		for i, v := range content2 {
			data2[i] = utils.Struct2StrArr(v)
		}
		file, err := utils.ExportCsv(header1, header2, data1, data2)
		if err != nil {
			log.Errorf("导出csv失败: %v", err)
		}
		// 保存文件
		fileName += "_告警统计.csv"
		err = utils.SaveFile(constant.RegularReportAlarmPath, fileName, file)
		if err != nil {
			log.Errorf("SaveFile error: %v", err)
			log.Errorf("定期报表-报表查看: 下载失败")
		}
		err = s.regularReportRepository.CreateRegularReportLog(regularReport.Name, regularReport.PeriodicType, "alarm", fileName)
		if err != nil {
			return
		}
	}
}

// 封装定时任务执行函数
func (s NewJobService) runJobByJobAndPassport(job model.NewJob, v model.PassPort, wg *sync.WaitGroup) (err error) {
	type Result struct {
		result     string
		resultInfo string
		runType    string
		startAt    utils.JsonTime
		endAt      utils.JsonTime
	}
	var msg Result
	switch job.RunTimeType {
	case "Scheduled":
		// 定时任务
		msg.runType = "定时执行"
	case "Periodic":
		// 周期任务
		msg.runType = "周期执行"
	case "Manual":
		// 手动任务
		msg.runType = "手动执行"
	default:
		return
	}
	if job.RunType == "command" {
		go func(err error) {
			wg.Add(1)
			defer wg.Done()
			t1 := utils.NowJsonTime()
			var password, privateKey, passphrase string
			if v.IsSshKey == 0 {
				password = v.Password
				privateKey = ""
				passphrase = ""
			} else {
				password = ""
				privateKey = path.Join(constant.PrivateKeyPath, v.PrivateKey)
				passphrase = v.Passphrase
			}
			result, err := ExecCommandBySSH(job.Command, v.Ip, v.Port, v.Passport, password, privateKey, passphrase)
			t2 := utils.NowJsonTime()
			//elapsed := time.Since(t1)
			if err != nil {
				msg.result = "失败"
				msg.resultInfo = err.Error()
				msg.startAt = t1
				msg.endAt = t2
				//log.Infof(msg)
			} else {
				//msg.resultInfo = fmt.Sprintf("资产「%v」Shell执行成功,返回值「%v」,耗时「%v」", v.AssetName, result, elapsed)
				//log.Infof(msg)
				msg.result = "成功"
				msg.resultInfo = result
				msg.startAt = t1
				msg.endAt = t2
			}

			department, err := s.newAssetRepository.GetDepartmentByDepartmentId(context.TODO(), job.DepartmentID)
			joblog := model.NewJobLog{
				Department:   department.Name,
				Name:         job.Name,
				DepartmentID: job.DepartmentID,
				Command:      "命令: " + job.Command,
				StartAt:      msg.startAt,
				EndAt:        msg.endAt,
				Result:       msg.result,
				Type:         msg.runType,
				AssetName:    v.AssetName,
				AssetIp:      v.Ip,
				Passport:     v.Passport,
				Port:         v.Port,
				ResultInfo:   msg.resultInfo,
			}
			err = s.newJobRepository.WriteRunLog(context.TODO(), joblog)
			if err != nil {
				return
			}
			//msgChan <- msg
		}(err)
	} else if job.RunType == constant.FuncCheckAssetStatusJob {
		go func(err error) {
			wg.Add(1)
			defer wg.Done()
			t1 := time.Now()
			active := utils.Tcping(v.Ip, v.Port)
			t2 := time.Now()
			elapsed := t2.Sub(t1)
			m := fmt.Sprintf("资产「%v」存活状态检测完成,存活「%v」,耗时「%v」", v.Name, active, elapsed)
			if active {
				err = s.newAssetRepository.UpdatePassportStatus(context.TODO(), v.ID, 1)
				if err != nil {
					log.Errorf("更新资产「%v」状态失败,错误信息「%v」", v.Name, err)
					return
				}
			} else {
				err = s.newAssetRepository.UpdatePassportStatus(context.TODO(), v.ID, 0)
				if err != nil {
					log.Errorf("更新资产「%v」状态失败,错误信息「%v」", v.Name, err)
					return
				}
			}
			log.Infof(m)
			department, err := s.newAssetRepository.GetDepartmentByDepartmentId(context.TODO(), job.DepartmentID)
			joblog := model.NewJobLog{
				Department:   department.Name,
				Name:         job.Name,
				DepartmentID: job.DepartmentID,
				Command:      "存活状态检测",
				Type:         msg.runType,
				StartAt:      utils.JsonTime{Time: t1},
				EndAt:        utils.JsonTime{Time: t2},
				Result:       "成功",
				AssetName:    v.AssetName,
				AssetIp:      v.Ip,
				Passport:     v.Passport,
				Port:         v.Port,
				ResultInfo:   m,
			}
			err = s.newJobRepository.WriteRunLog(context.TODO(), joblog)
			if err != nil {
				return
			}
			//msgChan <- m
		}(err)
	} else if job.RunType == "shell" { // 脚本
		go func(err error) {
			wg.Add(1)
			defer wg.Done()
			t1 := utils.NowJsonTime()
			var password, privateKey, passphrase string
			if v.IsSshKey != 1 {
				password = v.Password
				privateKey = ""
				passphrase = ""
			} else {
				password = ""
				privateKey = path.Join(constant.PrivateKeyPath, v.PrivateKey)
				passphrase = v.Passphrase
			}
			sshClient, err := terminal.NewSshClient(v.Ip, v.Port, v.Passport, password, privateKey, passphrase)
			if err != nil {
				msg.result = "失败"
				msg.resultInfo = err.Error()
				msg.startAt = t1
				msg.endAt = utils.NowJsonTime()
				log.Infof("自动任务「%v」执行失败,资产「%v」连接失败,错误信息「%v」", v.Name, err)
				return
			}
			defer func(sshClient *ssh.Client) {
				err := sshClient.Close()
				if err != nil {
					log.Errorf("关闭ssh连接失败,错误信息「%v」", err)
				}
			}(sshClient)
			result, err := utils.RunScriptOnRemoteServer(sshClient, path.Join(constant.ShellPath, job.ShellName))
			t2 := utils.NowJsonTime()
			//elapsed := time.Since(t1)
			if err != nil {
				msg.result = "失败"
				msg.resultInfo = err.Error()
				msg.startAt = t1
				msg.endAt = t2
				//log.Infof(msg)
			} else {
				//msg.resultInfo = fmt.Sprintf("资产「%v」Shell执行成功,返回值「%v」,耗时「%v」", v.AssetName, result, elapsed)
				//log.Infof(msg)
				msg.result = "成功"
				msg.resultInfo = result
				msg.startAt = t1
				msg.endAt = t2
			}

			department, err := s.newAssetRepository.GetDepartmentByDepartmentId(context.TODO(), job.DepartmentID)
			joblog := model.NewJobLog{
				Department:   department.Name,
				Name:         job.Name,
				DepartmentID: job.DepartmentID,
				Command:      "脚本: " + job.ShellName[:strings.LastIndex(job.ShellName, ".")],
				Type:         msg.runType,
				StartAt:      msg.startAt,
				EndAt:        msg.endAt,
				Result:       msg.result,
				AssetName:    v.AssetName,
				AssetIp:      v.Ip,
				Passport:     v.Passport,
				Port:         v.Port,
				ResultInfo:   msg.resultInfo,
			}
			err = s.newJobRepository.WriteRunLog(context.TODO(), joblog)
			if err != nil {
				return
			}
			//msgChan <- msg
		}(err)
	}
	return
}
