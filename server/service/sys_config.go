package service

import (
	"errors"
	"log/syslog"
	"strconv"
	"tkbastion/pkg/log"
	"tkbastion/server/repository"

	"gopkg.in/gomail.v2"
)

type SysConfigService struct {
	propertyRepository *repository.PropertyRepository
}

func NewSysConfigService(propertyRepository *repository.PropertyRepository) *SysConfigService {
	return &SysConfigService{propertyRepository: propertyRepository}
}

func (r SysConfigService) SendTestMail(host, username, password string, port int, ssl bool, to []string, subject, text string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", "Tkbastion <"+username+">")
	m.SetHeader("To", to...)        //  收件人
	m.SetHeader("Subject", subject) //  主题
	m.SetBody("text/html", text)    //  正文

	d := gomail.NewDialer(host, port, username, password)
	if ssl {
		(*d).SSL = true
	}

	err := d.DialAndSend(m)
	return err
}

func (r SysConfigService) SendMail(to []string, subject, text string) error {
	item := r.propertyRepository.FindAuMap("mail")
	if "false" == item["mail-state"] {
		return errors.New("未开启邮件配置, 发送邮件失败")
	}
	iMailPort, err := strconv.Atoi(item["mail-port"])
	if nil != err {
		log.Errorf("Atoi Error: %v", err.Error())
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", "Tkbastion <"+item["mail-account"]+">")
	m.SetHeader("To", to...)        //  收件人
	m.SetHeader("Subject", subject) //  主题
	m.SetBody("text/html", text)    //  正文

	d := gomail.NewDialer(item["mail-send-mail-server"], iMailPort, item["mail-account"], item["mail-password"])
	if "ssl" == item["mail-secret-type"] {
		(*d).SSL = true
	}

	err = d.DialAndSend(m)
	return err
}

func (r SysConfigService) SendSyslog(info string) error {
	item := r.propertyRepository.FindAuMap("syslog")
	if "false" == item["syslog-state"] {
		return errors.New("未开启SYSLOG配置, 发送日志失败")
	}
	_, err := strconv.Atoi(item["syslog-port"])
	if nil != err {
		log.Errorf("Atoi Error: %v", err.Error())
		return err
	}

	sysLog, err := syslog.Dial("tcp", item["syslog-server"]+":"+item["syslog-port"], syslog.LOG_ERR, "Tkbastion")
	if nil != err {
		log.Errorf("TCP Dial Error: %v", err.Error())

		sysLog, err = syslog.Dial("udp", item["syslog-server"]+":"+item["syslog-port"], syslog.LOG_ERR, "Tkbastion")
		if nil != err {
			log.Errorf("UDP Dial Error: %v", err.Error())
			return err
		}
	}

	sysLog.Emerg("[WARN] " + info)
	return nil
}

func (r SysConfigService) SendMailToAdmin(subject, text string) error {
	// 获取管理员id
	admin, err := repository.UserNewDao.FindAdmin()
	if err != nil {
		return err
	}
	item := r.propertyRepository.FindAuMap("mail")
	if "false" == item["mail-state"] {
		return errors.New("未开启邮件配置, 发送邮件失败")
	}
	iMailPort, err := strconv.Atoi(item["mail-port"])
	if nil != err {
		log.Errorf("Atoi Error: %v", err.Error())
		return err
	}
	m := gomail.NewMessage()
	m.SetHeader("From", "Tkbastion <"+item["mail-account"]+">")
	m.SetHeader("To", admin.Mail)   //  收件人
	m.SetHeader("Subject", subject) //  主题
	m.SetBody("text/html", text)    //  正文

	d := gomail.NewDialer(item["mail-send-mail-server"], iMailPort, item["mail-account"], item["mail-password"])
	if "ssl" == item["mail-secret-type"] {
		(*d).SSL = true
	}
	err = d.DialAndSend(m)
	return err
}
