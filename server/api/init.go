package api

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	casbinlog "github.com/casbin/casbin/v2/log"
	"os"
	"os/exec"
	"strings"
	"time"
	"tkbastion/pkg/constant"
	"tkbastion/server/repository"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"tkbastion/pkg/config"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/pkg/service"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	xormadapter "github.com/casbin/xorm-adapter/v2"
	"github.com/patrickmn/go-cache"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitVideo() {
	// 初始化视频
	all, err := newSessionRepository.GetAllSession(context.TODO())
	if err != nil {
		log.Errorf("初始化处理设备录像文件失败: %s", err)
		return
	}
	c := 0
	for _, s := range all {
		if s.IsDownloadRecord == 1 || s.IsDownloadRecord == 2 {
			if err := newSessionRepository.UpdateVideoDownload(context.TODO(), s.ID, 0); err != nil {
				log.Errorf("初始化处理设备录像文件失败: %s", err)
				return
			}
		}
		if s.Recording != "" {
			if s.Mode != constant.Guacd {
				f := utils.FileExists(s.Recording[:len(s.Recording)-4] + "gif")
				if f {
					err = os.Remove(s.Recording[:len(s.Recording)-4] + "gif")
					if err != nil {
						c++
					}
				}
			} else {
				f := utils.FileExists(s.Recording + "/recording.m4v")
				if f {
					err = os.Remove(s.Recording + "/recording.m4v")
					if err != nil {
						c++
					}
				}
			}
		}
	}
	ac := 0
	allApp, err := appSessionRepository.GetAllAppSession(context.TODO())
	if err != nil {
		log.Errorf("初始化处理应用录像文件失败: %s", err)
		return
	}
	for _, sp := range allApp {
		if sp.Recording != "" {
			if sp.Download == 1 || sp.Download == 2 {
				if err := appSessionRepository.UpdateVideoDownload(context.TODO(), sp.ID, 0); err != nil {
					log.Errorf("初始化处理应用录像文件失败: %s", err)
					return
				}
			}
			f := utils.FileExists(sp.Recording + "/recording.m4v")
			if f {
				err = os.Remove(sp.Recording + "/recording.m4v")
				if err != nil {
					ac++
				}
			}
		}
	}
	log.Infof("初始化处理录像文件完成，删除失败 %v 个设备录像文件，删除失败 %v 个应用录像文件", c, ac)
}

// InitSession 将所有未结束的会话状态设置为已结束且结束时间为当前时间
func InitSession() {
	if err := repository.SessionRepo.UpdateAllSessionStatus(context.TODO()); err != nil {
		log.Errorf("初始化会话状态失败: %s", err)
		return
	}
	log.Info("初始化会话状态完成")
}

func InitDBData() (err error) {
	if err := menuService.InitMenus(casbinRuleRepository); err != nil {
		return err
	}
	if err := propertyService.InitProperties(); err != nil {
		return err
	}
	if err := numService.InitNums(); err != nil {
		return err
	}
	if err := roleService.InitRole(); err != nil {
		return err
	}
	if err := userService.InitUserNew(); err != nil {
		return err
	}
	if err := identityService.IdentityConfigData(); err != nil {
		return err
	}
	if err := jobService.InitJob(); err != nil {
		return err
	}
	//if err := userService.FixUserOnlineState(); err != nil {
	//	return err
	//}
	//if err := storageService.InitStorages(); err != nil {
	//	return err
	//}
	//if err := messageService.InitAlarmConfig(); err != nil {
	//	return err
	//}
	if err := snmpService.InitSnmp(); err != nil {
		return err
	}
	if err := commandControlService.InitCommandControl(); err != nil {
		return err
	}
	if err := departmentService.InitDepartment(); err != nil {
		return err
	}
	if err := systemTypeService.InitSystemType(); err != nil {
		return err
	}
	return nil
}

func CheckEncryptionKey() error {
	global.Config = config.GlobalCfg
	if "" == global.Config.EncryptionKey && "" == global.Config.NewEncryptionKey {
		fmt.Print("未配置加密密钥,采用系统默认加密密钥\n")

		global.Config.EncryptionKey = "tig-fortress-machine"
		md5Sum := fmt.Sprintf("%x", md5.Sum([]byte(global.Config.EncryptionKey)))
		global.Config.EncryptionPassword = []byte(md5Sum)
	} else if "" != global.Config.EncryptionKey && "" == global.Config.NewEncryptionKey {
		fmt.Print("采用用户配置加密密钥\n")

		md5Sum := fmt.Sprintf("%x", md5.Sum([]byte(global.Config.EncryptionKey)))
		global.Config.EncryptionPassword = []byte(md5Sum)
	} else if "" != global.Config.EncryptionKey && "" != global.Config.NewEncryptionKey {
		fmt.Print("更换加密密钥\n")

		md5Sum := fmt.Sprintf("%x", md5.Sum([]byte(global.Config.NewEncryptionKey)))
		global.Config.EncryptionPassword = []byte(md5Sum)

		err := ChangeEncryptionKey(global.Config.EncryptionKey, global.Config.NewEncryptionKey)
		if nil != err {
			return err
		}
	} else {
		// TODO
		// 产品手册需写
		// 1.不输入密钥时使用系统默认密钥进行加密,若系统已用默认密钥加密过一些主机,则修改密钥时旧密钥需填写默认密钥:tig-fortress-machine.若用户此前已配置过密钥,则旧密钥需填写用户之前配置密钥.
		// 2.若用户此前使用默认密钥加密过一些资源,之后想更换密钥,不能直接使用EncryptionKey参数,因为已使用过默认密钥加密过数据,需使用旧密钥(系统默认密钥),新密钥格式修改密钥
		// 3.即系统初次启动时可自行配置密钥,或不配置密钥使用默认密钥加密方式.   在系统已加密过一些资源后,若要更换密钥,需使用更换密钥格式参数,旧密钥为默认密钥或用户此前配置的密钥

		return errors.New("更换密钥时需同时配置旧密钥与新密钥")
	}
	return nil
}

func ResetPassword(username string) error {
	user, err := userNewRepository.FindByName(username)
	if err != nil {
		return err
	}
	password := "admin"
	passwd, err := utils.Encoder.Encode([]byte(password))
	if err != nil {
		return err
	}
	if err := userNewRepository.UpdateStructById(model.UserNew{
		Password:       string(passwd),
		VerifyPassword: string(passwd)}, user.ID); err != nil {
		return err
	}
	log.Debugf("用户「%v」密码初始化为: %v", user.Username, password)
	return nil
}

func ResetTotp(username string) error {
	user, err := userNewRepository.FindByName(username)
	if err != nil {
		return err
	}
	if err := userNewRepository.UpdateStructById(model.UserNew{TOTPSecret: "-"}, user.ID); err != nil {
		return err
	}
	log.Debugf("用户「%v」已重置TOTP", user.Username)
	return nil
}

func ChangeEncryptionKey(oldEncryptionKey, newEncryptionKey string) error {
	oldPassword := []byte(fmt.Sprintf("%x", md5.Sum([]byte(oldEncryptionKey))))
	newPassword := []byte(fmt.Sprintf("%x", md5.Sum([]byte(newEncryptionKey))))

	credentials, err := credentialRepository.FindAll()
	if err != nil {
		return err
	}
	for i := range credentials {
		credential := credentials[i]
		if err := credentialRepository.Decrypt(&credential, oldPassword); err != nil {
			return err
		}
		if err := credentialRepository.Encrypt(&credential, newPassword); err != nil {
			return err
		}
		if err := credentialRepository.UpdateById(&credential, credential.ID); err != nil {
			return err
		}
	}

	//assets, err := assetRepository.FindAll()
	//if err != nil {
	//	return err
	//}
	//for i := range assets {
	//	asset := assets[i]
	//	if err := assetRepository.Decrypt(&asset, oldPassword); err != nil {
	//		return err
	//	}
	//	if err := assetRepository.Encrypt(&asset, newPassword); err != nil {
	//		return err
	//	}
	//	if err := assetRepository.UpdateById(&asset, asset.ID); err != nil {
	//		return err
	//	}
	//}

	log.Infof("encryption key has being changed.")
	return nil
}

func SetupCache() *cache.Cache {
	// 配置缓存器
	mCache := cache.New(5*time.Minute, 1*time.Minute) //5min为过期时间，1min为清理缓存的时间
	mCache.OnEvicted(func(key string, value interface{}) {
		if strings.HasPrefix(key, constant.Token) {
			token := GetTokenFormCacheKey(key)
			log.Debugf("用户Token「%v」过期", token)
			err := userService.Logout(token)
			if err != nil {
				log.Errorf("退出登录失败 %v", err)
			}
		}
	})
	return mCache
}

func SetupDB() *gorm.DB {
	var logMode logger.Interface
	// TODO gorm日志级别, 即数据表查询输出的日志级别, 删除(注释)debug
	if global.Config.Debug {
		logMode = logger.Default.LogMode(logger.Error)
	} else {
		logMode = logger.Default.LogMode(logger.Silent)
	}

	fmt.Printf("当前数据库模式为:%v\n", global.Config.DB)
	var err error
	var db *gorm.DB

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		global.Config.Mysql.Username,
		global.Config.Mysql.Password,
		global.Config.Mysql.Hostname,
		global.Config.Mysql.Port,
		global.Config.Mysql.Database,
	)
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logMode,
	})
	if err != nil {
		log.WithError(err).Panic("连接数据库异常 Error: ", err)
	}

	if err := db.AutoMigrate(&model.JobExport{}, &model.UserGroupMember{}, &model.LoginLog{}, &model.Num{}, &model.JobLog{}, &model.Menu{},
		&model.CasbinRule{}, &model.CasbinApi{}, &model.OperateLog{}, &model.PolicyConfig{}, &model.Job{}, &model.UserCollectApp{}, &model.NewCommand{},
		&model.IdentityConfig{}, &model.UserStrategy{}, &model.UserStrategyUsers{}, &model.UserStrategyUserGroup{}, &model.NewStorage{}, model.CommandRecord{},
		&model.CommandRelevance{}, &model.CommandSet{}, &model.PassportConfiguration{}, &model.CommandStrategy{}, &model.CommandContent{}, &model.OrderLog{},
		&model.NewWorkOrder{}, &model.WorkOrderApprovalLog{}, &model.NewWorkOrderLog{}, &model.WorkOrderAsset{}, &model.NewApplicationServer{}, &model.NewAsset{},
		&model.NewApplication{}, &model.NewProgram{}, &model.NewAssetGroup{}, &model.Department{}, &model.OperateAuth{}, &model.PassPort{}, &model.UserCollecte{},
		&model.Role{}, &model.RoleMenu{}, &model.AssetGroupWithAsset{}, &model.SystemType{}, &model.UserGroupNew{}, &model.UserNew{}, &model.Role{}, &model.RoleMenu{},
		&model.LdapAdAuth{}, &model.Session{}, &model.AppSession{}, &model.NewJob{}, &model.NewJobLog{}, &model.NewJobWithAssets{}, &model.NewJobWithAssetGroups{},
		&model.Property{}, &model.AssetAuthReportForm{}, &model.OperationAndMaintenanceLog{}, &model.ExtendConfig{}, &model.ProtocolAccessLog{}, &model.OperateAlarmLog{},
		&model.SystemAlarmLog{}, &model.UsageCpu{}, &model.UsageMem{}, &model.UsageDisk{}, &model.RegularReport{}, &model.RegularReportLog{}, &model.Message{},
		model.PasswdChange{}, model.PasswdChangeDevice{}, model.PasswdChangeDeviceGroup{}, &model.PasswdChangeResult{}, &model.ApplicationAuthReportForm{},
		&model.FileRecord{}, &model.ClipboardRecord{},
	); err != nil {
		log.WithError(err).Panic("初始化数据库表结构异常")
	}
	return db
}

func InitCasbin() {
	// 初始化Casbin Handle
	// model
	var text = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = (r.sub == p.sub) && (keyMatch2(r.obj, p.obj) || keyMatch(r.obj, p.obj) || regexMatch(r.obj, p.obj)) && (r.act == p.act)
`
	mysqlConf := global.Config.Mysql
	mysqlSourceName := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", mysqlConf.Username, mysqlConf.Password, mysqlConf.Hostname, mysqlConf.Port, mysqlConf.Database)
	adapter, err := xormadapter.NewAdapter("mysql", mysqlSourceName, true)
	if nil != err {
		log.WithError(err).Panic("初始化Casbin权限控制异常—NewAdapter")
	}

	m, err := casbinmodel.NewModelFromString(text)
	if nil != err {
		log.WithError(err).Panic("初始化Casbin权限控制异常—NewModelFromString")
	}
	c, err := casbin.NewSyncedEnforcer(m, adapter)
	if nil != err {
		log.WithError(err).Panic("初始化Casbin权限控制异常—NewSyncedEnforcer")
	}
	err = c.LoadPolicy()
	if nil != err {
		log.WithError(err).Panic("初始化Casbin权限控制异常—LoadPolicy")
	}

	casbinlog.SetLogger(&casbinlog.DefaultLogger{})
	//c.EnableLog(false)

	global.CasbinEnforcer = c
}

func InitContainerId() {
	// 获取guacd容器ID
	_, err := service.GetCommandLinuxCon("service docker start")
	if err != nil {
		log.WithError(err).Panic("初始化guacd容器ID异常")
	}
	cl, err := client.NewClientWithOpts()
	if err != nil {
		log.WithError(err).Panic("获取docker-guacd容器id失败")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	list, err := cl.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		log.WithError(err).Panic("获取docker-guacd容器id失败")
	}
	for _, l := range list {
		if l.Image == "dushixiang/guacd" {
			global.CONTAINERID = l.ID
		}
	}

	err = propertyRepository.UpdateByName(&model.Property{
		Name:  "enable-debug",
		Value: "false",
	}, "enable-debug")
	if err != nil {
		log.WithError(err).Panic("初始化系统运行模式失败")
	}
	// 初始化guacd的连接模式
	err = service.DockerGuacd()
	if err != nil {
		log.WithError(err).Panic("配置guacd模式失败")
	}
	log.Info("初始化系统运行模式成功")
}

func WritePidFile() {
	cmd := exec.Command("bash", "-c", `ps -ef | grep tkbastion | grep -v "grep" |   awk '{print $2}'`)

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if nil != err {
		log.Errorf("Run Error: %v", err)
	}
	//fmt.Println("pid: ", out.String())

	file, err := os.Create("/tkbastion/tkbastion.pid")
	if nil != err {
		log.Errorf("创建pid文件失败: %v", err)
	}
	file.WriteString(out.String())
	file.Close()

	monitorTkbastionLogF, err := os.OpenFile("/monitor/monitorTkbastion.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if nil != err {
		log.Errorf("打开监控程序日志文件失败: %v", err)
	}
	monitorTkbastionLogF.WriteString(time.Now().Format("2006/01/02 15:04") + ": Tkbastion启动成功\n")
	log.Infof("Tkbastion启动成功")
	monitorTkbastionLogF.Close()
}

func InitLoadMiddlewarePath() {
	dir, _ := os.Getwd()
	fileList, err := os.ReadDir(dir + "/web/dist/")
	if err != nil {
		log.WithError(err).Errorf("读取目录下的所有文件异常,异常信息: %v", err.Error())
	}
	for _, file := range fileList {
		if !file.IsDir() {
			constant.LoadMiddlewarePath = append(constant.LoadMiddlewarePath, "/"+file.Name())
		}
	}
}
