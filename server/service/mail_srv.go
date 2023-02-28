package service

import (
	"errors"
	"gopkg.in/gomail.v2"
	"strconv"
	"tkbastion/pkg/constant"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/log"
	"tkbastion/server/repository"
)

type mailSrv struct {
	baseService
}

// CheckMail 检查发送邮件的条件
func (r mailSrv) CheckMail() error {
	propertiesMap := repository.PropertyDao.FindAllMap()
	host := propertiesMap[constant.MailHost]
	port := propertiesMap[constant.MailPort]
	username := propertiesMap[constant.MailUsername]
	password := propertiesMap[constant.MailPassword]
	if host == "" || host == "-" || port == "" || port == "-" || username == "" || username == "-" || password == "" || password == "-" {
		return errors.New(errs.MailCheckFail)
	}
	return nil
}

// SendMail [封装类型A]发送邮件 参数1:收件人邮箱  参数2:发送内容
func (r mailSrv) SendMail(recipientMailAddr, subject, text string) error {
	propertiesMap := repository.PropertyDao.FindAllMap()
	host := propertiesMap[constant.MailHost]
	port := propertiesMap[constant.MailPort]
	username := propertiesMap[constant.MailUsername]
	password := propertiesMap[constant.MailPassword]
	//========================================================测试区域
	//========================================================
	if host == "" || host == "-" || port == "" || port == "-" || username == "" || username == "-" || password == "" || password == "-" {
		return errors.New(errs.MailCheckFail)

	}
	if err := r.NewSendMail(host, port, username, password, []string{recipientMailAddr}, "[Tkbastion] "+subject, text); err != nil {
		log.Errorf("发送邮件异常,异常信息:%v", err)
		return err
	}
	return nil
}

// NewSendMail [基本封装]新接口用于邮箱认证验证码的发送、测试邮件发送等出错后必须排查原因的功能, 未来新的功能需发送邮件时均调用此接口返回error进行错误判断
func (r mailSrv) NewSendMail(host, port, username, password string, to []string, subject, text string) error {
	iport, err := strconv.Atoi(port)
	if nil != err {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", "Tkbastion <"+username+">")
	m.SetHeader("To", to...)        //  收件人
	m.SetHeader("Subject", subject) //  主题
	m.SetBody("text/html", text)    //  正文

	d := gomail.NewDialer(host, iport, username, password)
	err = d.DialAndSend(m)
	return err
}
