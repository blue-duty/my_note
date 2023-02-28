package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/pkg/terminal"
	"tkbastion/server/model"
	"tkbastion/server/repository"
	"tkbastion/server/utils"

	"github.com/sirupsen/logrus"
	"github.com/ziutek/telnet"

	"golang.org/x/crypto/ssh"
)

// CreateAutoChangePasswdTask 创建自动改密任务
func CreateAutoChangePasswdTask(id string) error {
	doFunc := func() {
		var err error
		pc, err := repository.PasswdChangeRepo.FindByID(context.TODO(), id)
		if err != nil {
			log.Error("获取改密策略失败，错误信息:%s", err)
			return
		}

		if pc.RunType == "Periodic" {
			if (pc.StartAt.Before(time.Now().Add(time.Minute*1)) || pc.StartAt.Equal(time.Now().Add(time.Minute*1))) && pc.EndAt.After(time.Now()) {
				goto job
			}
			if pc.EndAt.Before(time.Now()) {
				err := global.SCHEDULER.RemoveByTag(pc.ID)
				if err != nil {
					return
				}
			}
			return
		} else {
			if pc.RunTime.Before(time.Now()) {
				err := global.SCHEDULER.RemoveByTag(pc.ID)
				if err != nil {
					return
				}
			} else {
				goto job
			}
		}

	job:
		{
			log.Infof("开始执行自动改密任务:%s", pc.Name)
			pids, err := repository.PasswdChangeRepo.FindPasswdChangeAccountIds(context.TODO(), id)
			if err != nil {
				log.Error("获取改密策略关联的账号id失败，错误信息:%s", err)
				return
			}
			passports, err := repository.AssetNewDao.GetPassportListByIds(context.TODO(), pids)
			if err != nil {
				log.Error("获取改密策略关联的账号失败，错误信息:%s", err)
				return
			}
			systemTypes, err := repository.SystemTypeDto.GetSystemTypeIDs(context.TODO())
			if err != nil {
				log.Error("获取系统类型失败，错误信息", err)
				return
			}
			encryption := NewEncryption(&pc, passports, systemTypes)
			if err := encryption.ChangePasswd(); err != nil {
				log.Error("自动改密失败，错误信息:%s", err)
				return
			}
			log.Infof("自动改密任务:%s 执行完成", id)
			return
		}
	}
	pc, err := repository.PasswdChangeRepo.FindByID(context.TODO(), id)
	if err != nil {
		log.Error("获取改密策略失败，错误信息:%s", err)
		return err
	}
	if pc.RunType == "Scheduled" {
		// 定时任务
		var err error
		_, err = global.SCHEDULER.Tag(pc.ID).CronWithSeconds(pc.RunTime.Format("05 04 15 02 01 *")).Do(doFunc)
		if err != nil {
			log.Errorf("创建自动改密任务失败，改密策略名称:%s，错误信息:%s", pc.Name, err)
			return err
		}
		log.Infof("开启改密任务 [%s], 运行中计划任务数量: [%d]", pc.Name, len(global.SCHEDULER.Jobs()))
	}
	if pc.RunType == "Periodic" {
		var err error
		switch pc.PeriodicType {
		case "Day":
			_, err = global.SCHEDULER.Every(pc.Periodic).Tag(id).Day().Do(doFunc)
		case "Week":
			_, err = global.SCHEDULER.Every(pc.Periodic).Tag(id).Week().Do(doFunc)
		case "Month":
			_, err = global.SCHEDULER.Every(pc.Periodic).Tag(id).Month().Do(doFunc)
		case "Hour":
			_, err = global.SCHEDULER.Every(pc.Periodic).Tag(id).Hour().Do(doFunc)
		case "Minute":
			_, err = global.SCHEDULER.Every(pc.Periodic).Tag(id).Minute().Do(doFunc)
		}
		if err != nil {
			log.Errorf("创建自动改密任务失败，改密策略名称:%s，错误信息:%s", pc.Name, err)
			return err
		}
		log.Infof("开启改密任务 [%s], 运行中计划任务数量: [%d]", pc.Name, len(global.SCHEDULER.Jobs()))
	}
	global.SCHEDULER.StartAsync()
	return nil
}

// PasswdChangeNow 立即通过策略改密
func PasswdChangeNow(id string) error {
	pc, err := repository.PasswdChangeRepo.FindByID(context.TODO(), id)
	if err != nil {
		log.Error("获取改密策略失败，错误信息:%s", err)
		return err
	}
	pids, err := repository.PasswdChangeRepo.FindPasswdChangeAccountIds(context.TODO(), id)
	if err != nil {
		log.Error("获取改密策略关联的账号id失败，错误信息:%s", err)
		return err
	}
	passports, err := repository.AssetNewDao.GetPassportListByIds(context.TODO(), pids)
	if err != nil {
		log.Error("获取改密策略关联的账号失败，错误信息:%s", err)
		return err
	}
	systemTypes, err := repository.SystemTypeDto.GetSystemTypeIDs(context.TODO())
	if err != nil {
		log.Error("获取系统类型失败，错误信息", err)
		return err
	}
	encryption := NewEncryption(&pc, passports, systemTypes)
	if err := encryption.ChangePasswd(); err != nil {
		log.Error("自动改密失败，错误信息:%s", err)
		return err
	}
	return nil
}

type Encryption struct {
	encrypt  *model.PasswdChange
	passport []model.PassPort
	isSame   bool
	password string
	IsRoot   bool
	Windows  string
	Linux    string
	sync.Mutex
}

// NewEncryption 创建一个改密结构
func NewEncryption(encrypt *model.PasswdChange, passport []model.PassPort, systemType map[string]string) *Encryption {
	return &Encryption{
		encrypt:  encrypt,
		passport: passport,
		isSame:   encrypt.GenerateRule == 1 || encrypt.GenerateRule == 3,
		IsRoot:   *encrypt.IsPrivilege,
		Windows:  systemType["WINDOWS"],
		Linux:    systemType["LINUX"],
	}
}

// GeneratePassword 生成密码
func (e *Encryption) GeneratePassword() string {
	var pwlen int
	if *e.encrypt.IsComplexity {
		pwlen = e.encrypt.MinLength
	}
	genPasswd := func() string {
		var l int
		if *e.encrypt.IsComplexity {
			rand.Seed(time.Now().UnixNano())
			l = rand.Intn(pwlen-8) + 8
		} else {
			l = 8
		}
		var password string
		for len(password) < l {
			// 随机调用
			// 初始化随机数种子
			rand.NewSource(time.Now().UnixNano())
			// 定义一个包含四个函数指针的切片
			funcs := []func() string{utils.GetRandomLowerChar, utils.GetRandomUpperChar, utils.GetRandomSpecialChar, utils.GetRandomNumber}
			// 随机调用一个函数
			password += funcs[rand.Intn(len(funcs))]()
		}
		return password
	}

	if e.encrypt.GenerateRule == 1 {
		e.password = genPasswd()
		return e.password
	}
	if e.encrypt.GenerateRule == 3 {
		origData, err := base64.StdEncoding.DecodeString(e.encrypt.Password)
		if err != nil {
			log.Error("base64 decode error: %s", err)
			e.isSame = false
		}
		decryptedCBC, err := utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
		if err != nil {
			log.Error("aes decrypt error: %s", err)
			e.isSame = false
		}
		e.password = string(decryptedCBC)
		return e.password
	} else {
		return genPasswd()
	}
}

// ChangePasswd 为账号改密
func (e *Encryption) ChangePasswd() error {
	var wg sync.WaitGroup
	for _, p := range e.passport {
		wg.Add(1)
		go func(p model.PassPort) {
			defer wg.Done()
			if e.IsRoot || p.AssetType == e.Windows {
				// 通过root改密
				if err := e.changePasswdByRoot(p); err != nil {
					WriteLog(e.encrypt, p, "失败", "通过root权限改密失败，错误信息:"+err.Error())
					log.Error("change password with root error: %s", err)
				} else {
					WriteLog(e.encrypt, p, "成功", "")
				}
			} else {
				// 不通过root改密
				if err := e.changePasswd(p); err != nil {
					WriteLog(e.encrypt, p, "失败", "通过普通权限改密失败，错误信息:"+err.Error())
					log.Error("change password without root error: %s", err)
				} else {
					WriteLog(e.encrypt, p, "成功", "")
				}
			}
		}(p)
	}
	wg.Wait()
	return nil
}

// GetRootAccount 查询设备中是否存在root账号
func (e *Encryption) GetRootAccount(deviceId string, isTelnet bool) (model.PassPort, bool /* 是否存在root账号 */) {
	// 查询设备中是否存在root账号
	passports, err := repository.AssetNewDao.GetPassport(context.TODO(), deviceId)
	if err != nil {
		log.Error("find passport by device id error: %s", err)
		return model.PassPort{}, false
	}

	for _, p := range passports {
		if strings.ToLower(p.Passport) == "root" {
			if isTelnet && p.Protocol == "telnet" && p.AssetType == e.Linux {
				origData, err := base64.StdEncoding.DecodeString(p.Password)
				if err != nil {
					log.Error("base64 decode error: %s", err)
					return model.PassPort{}, false
				}
				decryptedCBC, err := utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
				if err != nil {
					log.Error("aes decrypt error: %s", err)
					return model.PassPort{}, false
				}
				p.Password = string(decryptedCBC)
				return p, true
			}
			if !isTelnet && p.Protocol == "ssh" && p.AssetType == e.Linux {
				origData, err := base64.StdEncoding.DecodeString(p.Password)
				if err != nil {
					log.Error("base64 decode error: %s", err)
					return model.PassPort{}, false
				}
				decryptedCBC, err := utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
				if err != nil {
					log.Error("aes decrypt error: %s", err)
					return model.PassPort{}, false
				}
				p.Password = string(decryptedCBC)
				return p, true
			}
		}
		if strings.ToLower(p.Passport) == "administrator" {
			if isTelnet && p.Protocol == "telnet" && p.AssetType == e.Windows {
				origData, err := base64.StdEncoding.DecodeString(p.Password)
				if err != nil {
					log.Error("base64 decode error: %s", err)
					return model.PassPort{}, false
				}
				decryptedCBC, err := utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
				if err != nil {
					log.Error("aes decrypt error: %s", err)
					return model.PassPort{}, false
				}
				p.Password = string(decryptedCBC)
				return p, true
			}
			if !isTelnet && p.Protocol == "ssh" && p.AssetType == e.Windows {
				origData, err := base64.StdEncoding.DecodeString(p.Password)
				if err != nil {
					log.Error("base64 decode error: %s", err)
					return model.PassPort{}, false
				}
				decryptedCBC, err := utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
				if err != nil {
					log.Error("aes decrypt error: %s", err)
					return model.PassPort{}, false
				}
				p.Password = string(decryptedCBC)
				return p, true
			}
		}
	}
	return model.PassPort{}, false
}

// changeSSHPasswdWithRoot 使用root权限的改密
func (e *Encryption) changePasswdByRoot(p model.PassPort) error {
	// 有root权限
	// 1. 查询设备中是否存在root账号
	var (
		root model.PassPort
		ok   bool
	)
	if p.Protocol == "telnet" {
		root, ok = e.GetRootAccount(p.AssetId, true)
	} else {
		root, ok = e.GetRootAccount(p.AssetId, false)
	}

	password := e.GeneratePassword()
	if !ok {
		// 不存在root账号
		// 1. 生成随机密码
		if p.AssetType == e.Windows && p.Protocol == "telnet" {
			return errors.New("windows设备telnet协议不支持非管理员权限改密")
		}

		if p.Protocol == "telnet" {
			err := e.changeTelnetPasswdForLinux(p, password)
			if err != nil {
				return err
			}
		} else if p.Protocol == "ssh" {
			// 2. 建立连接
			sshDial, err := terminal.NewSshClient(p.Ip, p.Port, p.Passport, p.Password, "-", "-")
			if err != nil {
				log.Error("new ssh client error: %s", err)
				return err
			}
			defer func(sshDial *ssh.Client) {
				err := sshDial.Close()
				if err != nil {
					log.Error("ssh dial close error: %s", err)
				}
			}(sshDial)
			if p.AssetType == e.Linux {
				// 3.1 修改Linux设备密码
				err = e.changeSSHPasswdWithoutRoot(sshDial, p.Password, password)
				if err != nil {
					log.Error("change password without root error: %s", err)
					return err
				}
			} else if p.AssetType == e.Windows {
				session, err := sshDial.NewSession()
				if err != nil {
					log.Error("new ssh session error: %s", err)
					return err
				}
				defer func(session *ssh.Session) {
					err := session.Close()
					if err != nil && err != io.EOF {
						log.Error("ssh session close error: %s", err)
					}
				}(session)

				// 执行改密命令
				_, err = session.CombinedOutput("net user " + p.Passport + " " + password)
				if err != nil {
					log.Error("change password error: %s", err)
					return err
				}
			}
		} else {
			return errors.New("不支持的协议")
		}
	} else {
		if p.AssetType == e.Windows {
			if p.Protocol == "telnet" {
				err := e.changePasswdWithTelnetForWindows(root, p.Passport, password)
				if err != nil {
					log.Error("change password with telnet error: %s", err)
					return err
				}
			}
			if p.Protocol == "ssh" {
				sshDial, err := terminal.NewSshClient(root.Ip, root.Port, root.Passport, root.Password, "-", "-")
				if err != nil {
					log.Error("new ssh client error: %s", err)
					return err
				}
				defer func(sshDial *ssh.Client) {
					err := sshDial.Close()
					if err != nil {
						log.Error("ssh dial close error: %s", err)
					}
				}(sshDial)
				session, err := sshDial.NewSession()
				if err != nil {
					log.Error("new ssh session error: %s", err)
					return err
				}
				defer func(session *ssh.Session) {
					err := session.Close()
					if err != nil && err != io.EOF {
						log.Error("ssh session close error: %s", err)
					}
				}(session)

				// 执行改密命令
				_, err = session.CombinedOutput("net user " + p.Passport + " " + password)
				if err != nil {
					log.Error("change password error: %s", err)
					return err
				}
			}
		} else if p.AssetType == e.Linux {
			// 存在root账号
			// 1. 生成随机密码
			// 2. 建立连接
			if p.Protocol == "telnet" {
				err := e.changeTelnetPasswdWithRootForLinux(root, p.Passport, password)
				if err != nil {
					log.Error("change password with telnet error: %s", err)
					return err
				}
			} else if p.Protocol == "ssh" {
				sshDial, err := terminal.NewSshClient(root.Ip, root.Port, root.Passport, root.Password, "-", "-")
				if err != nil {
					log.Error("new ssh client error: %s", err)
					return err
				}
				defer func(sshDial *ssh.Client) {
					err := sshDial.Close()
					if err != nil {
						log.Error("ssh dial close error: %s", err)
					}
				}(sshDial)
				// 3. 修改密码
				// 3.1 修改Linux设备密码
				err = e.changeSSHPasswdWithoutRoot(sshDial, p.Password, password)
				if err != nil {
					log.Error("change password without root error: %s", err)
					return err
				}
			}
		} else {
			return errors.New("不支持的设备类型")
		}
	}
	// 4. 修改数据库中的信息
	// 加密新密码
	encryptedCBC, err := utils.AesEncryptCBC([]byte(password), global.Config.EncryptionPassword)
	if err != nil {
		log.Error("aes encrypt error: %s", err)
		return err
	}
	pp := base64.StdEncoding.EncodeToString(encryptedCBC)
	// 加密旧密码
	encryptedCBC, err = utils.AesEncryptCBC([]byte(p.Password), global.Config.EncryptionPassword)
	if err != nil {
		log.Error("aes encrypt error: %s", err)
		return err
	}
	oldPassword := base64.StdEncoding.EncodeToString(encryptedCBC)
	err = repository.AssetNewDao.UpdatePassportForPasswd(context.TODO(), p.ID, pp, oldPassword, utils.NowJsonTime())
	if err != nil {
		log.Error("update passport password error: %s", err)
		return err
	}
	return nil
}

// changeSSHPasswdWithoutRoot 不使用root权限的改密
func (e *Encryption) changePasswd(p model.PassPort) error {
	// 没有root权限
	// 1. 生成随机密码
	password := e.GeneratePassword()
	if p.Protocol == "telnet" {
		if p.AssetType == e.Windows {
			return errors.New("windows设备不支持非管理员telnet协议改密")
		}
		err := e.changeTelnetPasswdForLinux(p, password)
		if err != nil {
			log.Error("change password with telnet error: %s", err)
			return err
		}
	} else if p.Protocol == "ssh" {
		// 2. 建立连接
		sshDial, err := terminal.NewSshClient(p.Ip, p.Port, p.Passport, p.Password, p.PrivateKey, "-")
		if err != nil {
			log.Error("new ssh client error: %s", err)
			return err
		}
		defer func(sshDial *ssh.Client) {
			err := sshDial.Close()
			if err != nil {
				log.Error("ssh dial close error: %s", err)
			}
		}(sshDial)
		// 3. 修改密码
		if p.AssetType == e.Linux {
			if strings.ToLower(p.Passport) != "root" {
				err = e.changeSSHPasswdWithoutRoot(sshDial, p.Password, password)
				if err != nil {
					log.Error("change password without root error: %s", err)
					return err
				}
			} else {
				err = e.changeSSHPasswdWithRoot(sshDial, p.Passport, password)
				if err != nil {
					log.Error("change password with root error: %s", err)
					return err
				}
			}
		}
		if p.AssetType == e.Windows {
			var session *ssh.Session
			if session, err = sshDial.NewSession(); err != nil {
				log.Error("new ssh session error: %s", err)
				return err
			}
			defer func(session *ssh.Session) {
				if err := session.Close(); err != nil && err != io.EOF {
					log.Error("ssh session close error: %s", err)
				}
			}(session)

			// 执行改密命令
			if _, err := session.CombinedOutput("net user " + p.Passport + " " + password); err != nil {
				log.Error("change password error: %s", err)
				return err
			}
		}
	} else {
		return errors.New("不支持的设备类型")
	}
	// 4. 修改数据库中的信息
	// 加密新密码
	encryptedCBC, err := utils.AesEncryptCBC([]byte(password), global.Config.EncryptionPassword)
	if err != nil {
		log.Error("aes encrypt error: %s", err)
		return err
	}
	pp := base64.StdEncoding.EncodeToString(encryptedCBC)
	// 加密旧密码
	encryptedCBC, err = utils.AesEncryptCBC([]byte(p.Password), global.Config.EncryptionPassword)
	if err != nil {
		log.Error("aes encrypt error: %s", err)
		return err
	}
	oldPassword := base64.StdEncoding.EncodeToString(encryptedCBC)
	err = repository.AssetNewDao.UpdatePassportForPasswd(context.TODO(), p.ID, pp, oldPassword, utils.NowJsonTime())
	if err != nil {
		log.Error("update passport password error: %s", err)
		return err
	}
	return nil
}

// WriteLog 编写改密日志
func WriteLog(encrypt *model.PasswdChange, passport model.PassPort, result, info string) {
	pcr := model.PasswdChangeResult{
		PasswdChangeID: encrypt.ID,
		AccountID:      passport.ID,
		DeviceID:       passport.AssetId,
		Result:         result,
		ChangeTime:     utils.NowJsonTime(),
		ResultDesc:     info,
	}

	err := repository.PasswdChangeRepo.WritePasswdChangeLog(context.TODO(), pcr)
	if err != nil {
		log.Error("write passwd change log error: %s", err)
	}
}

// 通过root改密
func (e *Encryption) changeSSHPasswdWithRoot(sshd *ssh.Client, passport, newPassword string) error {
	session, err := sshd.NewSession()
	if err != nil {
		//fmt.Println("dialect.Session error:", err)
		logrus.Error("dialect.Session error:", err)
		return err
	}
	defer func(session *ssh.Session) {
		err := session.Close()
		if err != nil && err != io.EOF {
			logrus.Error("session.Close error:", err)
		}
	}(session)
	// 运行命令
	// 读取命令的输出
	stdin, err := session.StdinPipe()
	if err != nil {
		log.Fatal("输入错误", err)
		return err
	}
	//设置session的标准输出和错误输出分别是os.stdout,os,stderr.就是输出到后台
	stdout, err := session.StdoutPipe()
	//session.Stderr = os.Stderr
	//session.Stdout = os.Stdout
	// 写入命令
	err = session.Start("passwd " + passport)
	if err != nil {
		log.Error("session.Start error:", err)
		return err
	}
	fmt.Println("请输入新密码", newPassword)
	_, err = fmt.Fprintf(stdin, "%s\n", newPassword)
	time.Sleep(time.Second * 5)
	if err != nil && err != io.EOF {
		log.Error("fmt.Fprintf error:", err)
		return err
	}
	fmt.Println("请再次输入新密码", newPassword)
	_, err = fmt.Fprintf(stdin, "%s\n", newPassword)
	time.Sleep(time.Second * 5)
	if err != nil && err != io.EOF {
		log.Error("fmt.Fprintf error:", err)
		return err
	}

	err = session.Wait()
	if err != nil {
		log.Error("session.Wait error:", err)
		return err
	}

	// 判断是否成功
	out, err := io.ReadAll(stdout)
	if err != nil {
		log.Error("ioutil.ReadAll error:", err)
		return err
	}
	fmt.Println(string(out))
	if strings.Contains(string(out), "successfully") {
		return nil
	}
	log.Info("Password changed successfully")
	return nil
}

// 通过普通用户改密
func (e *Encryption) changeSSHPasswdWithoutRoot(sshd *ssh.Client, oldPassword, newPassword string) error {
	session, err := sshd.NewSession()
	if err != nil {
		//fmt.Println("dialect.Session error:", err)
		log.Error("dialect.Session error:", err)
		return err
	}
	defer func(session *ssh.Session) {
		err := session.Close()
		if err != nil && err != io.EOF {
			log.Error("session.Close error:", err)
		}
	}(session)
	// 运行命令
	// 读取命令的输出
	stdin, err := session.StdinPipe()
	if err != nil {
		log.Fatal("输入错误", err)
		return err
	}
	//设置session的标准输出和错误输出分别是os.stdout,os,stderr.就是输出到后台
	//stdout, err := session.StdoutPipe()
	//session.Stderr = os.Stderr
	//session.Stdout = os.Stdout
	// 写入命令
	err = session.Start("passwd")
	if err != nil {
		log.Error("session.Start error:", err)
		return err
	}
	_, err = fmt.Fprintf(stdin, oldPassword+"\n")
	time.Sleep(time.Second * 1)
	if err != nil {
		log.Fatal("fmt.Fprintf error:", err)
		return err
	}
	_, err = fmt.Fprintf(stdin, "%s\n", newPassword)
	time.Sleep(time.Second * 1)
	if err != nil {
		log.Error("fmt.Fprintf error:", err)
		return err
	}
	_, err = fmt.Fprintf(stdin, "%s\n", newPassword)
	time.Sleep(time.Second * 1)
	if err != nil {
		log.Error("fmt.Fprintf error:", err)
		return err
	}

	return nil
}

const timeout = 10 * time.Second

// 通过telnet root改密linux
func (e *Encryption) changeTelnetPasswdWithRootForLinux(p model.PassPort, passport, newPassword string) error {
	t, err := telnet.DialTimeout("tcp", p.Ip+":"+strconv.Itoa(p.Port), timeout)
	if err != nil {
		fmt.Println("Error connecting:", err.Error())
		os.Exit(1)
	}
	defer func(t *telnet.Conn) {
		err := t.Close()
		if err != nil {
			log.Info("t.Close error:", err)
		}
	}(t)

	err = expect(t, "login: ")
	if err != nil {
		return err
	}
	err = sendLine(t, p.Passport)
	if err != nil {
		return err
	}

	err = expect(t, "ssword: ")
	if err != nil {
		return err
	}
	err = sendLine(t, p.Password)
	if err != nil {
		return err
	}

	err = expect(t, "#")
	if err != nil {
		return err
	}
	err = sendLine(t, "passwd "+passport)
	if err != nil {
		return err
	}

	err = expect(t, "ew password:")
	if err != nil {
		return err
	}
	err = sendLine(t, newPassword)
	if err != nil {
		return err
	}

	err = expect(t, "ew password:")
	if err != nil {
		return err
	}
	err = sendLine(t, newPassword)
	if err != nil {
		return err
	}

	err = expect(t, "successfully")
	return err
}

// 通过telnet普通用户改密linux
func (e *Encryption) changeTelnetPasswdForLinux(p model.PassPort, newPassword string) error {
	t, err := telnet.DialTimeout("tcp", p.Ip+":"+strconv.Itoa(p.Port), timeout)
	if err != nil {
		fmt.Println("Error connecting:", err.Error())
		os.Exit(1)
	}
	defer func(t *telnet.Conn) {
		err := t.Close()
		if err != nil {
			log.Info("t.Close error:", err)
		}
	}(t)

	err = expect(t, "login: ")
	if err != nil {
		return err
	}
	err = sendLine(t, p.Passport)
	if err != nil {
		return err
	}

	err = expect(t, "ssword: ")
	if err != nil {
		return err
	}
	err = sendLine(t, p.Password)
	if err != nil {
		return err
	}

	err = expect(t, "$")
	if err != nil {
		return err
	}
	err = sendLine(t, "passwd")
	if err != nil {
		return err
	}

	err = expect(t, "ssword:")
	if err != nil {
		return err
	}
	err = sendLine(t, newPassword)
	if err != nil {
		return err
	}

	err = expect(t, "ew password:")
	if err != nil {
		return err
	}
	err = sendLine(t, newPassword)
	if err != nil {
		return err
	}

	err = expect(t, "ew password:")
	if err != nil {
		return err
	}
	err = sendLine(t, newPassword)
	if err != nil {
		return err
	}

	err = expect(t, "successfully")
	return err
}

func expect(t *telnet.Conn, d ...string) error {
	err := t.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		return err
	}
	err = t.SkipUntil(d...)

	return err
}

func sendLine(t *telnet.Conn, s string) error {
	err := t.SetWriteDeadline(time.Now().Add(timeout))
	if err != nil {
		return err
	}
	buf := make([]byte, len(s)+1)
	copy(buf, s)
	buf[len(s)] = '\n'
	_, err = t.Write(buf)

	return err
}

// 通过telnet改密windows
func (e *Encryption) changePasswdWithTelnetForWindows(p model.PassPort, user, passwd string) error {
	tn, err := telnet.Dial("tcp", p.Ip+":"+strconv.Itoa(p.Port))
	if err != nil {
		log.Error("telnet.Dial error:", err)
		return err
	}
	defer func(tn *telnet.Conn) {
		err := tn.Close()
		if err != nil {
			log.Error("telnet.Close error:", err)
		}
	}(tn)

	// 读取欢迎信息
	buf := make([]byte, 1024)
	_, _ = tn.Read(buf)

	// 发送用户名
	_, err = tn.Write([]byte(p.Passport + "\r\n"))
	if err != nil {
		return err
	}
	time.Sleep(time.Millisecond * 500)

	buf = make([]byte, 1024)
	_, _ = tn.Read(buf)

	// 清空buf
	buf = make([]byte, 1024)

	// 发送密码
	_, err = tn.Write([]byte(p.Password + "\r\n"))
	if err != nil {
		return err
	}
	time.Sleep(time.Millisecond * 500)

	// 判断是否登录成功
	n, err := tn.Read(buf)
	if err != nil {
		return err
	}

	if strings.Contains(strings.ToLower(string(buf[:n])), "administrator") {
		// 改密
		_, err = tn.Write([]byte("net user " + user + " " + passwd + "\r\n"))
		if err != nil {
			return err
		}
		time.Sleep(time.Millisecond * 500)

		buf = make([]byte, 1024)
		_, _ = tn.Read(buf)
		return nil
	} else {
		return errors.New("telnet登录失败")
	}
}
