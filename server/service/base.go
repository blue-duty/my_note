package service

import (
	"context"
	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
)

type baseService struct {
}

func (service baseService) Context(db *gorm.DB) context.Context {
	return context.WithValue(context.TODO(), "db", db)
}

// SetupService 服务对象初始化
func SetupService() {
	ExpDateService = new(expDateService) //时间限制服务  : 限制登录时间
	PolicyConfigSrv = NewPolicyConfigSrv()
	MailSrv = new(mailSrv)

	if err := setupInit(); err != nil {
		log.Errorf("服务对象初始化出现异常,异常信息:%v", err)
	}
}

// setupInit 服务对象的一些初始化操作,服务的一些初始化不放入init中，因为init破坏程序的可读性。只有三方库配置时可放入init中
func setupInit() error {
	//策略配置服务的初始化
	if err := PolicyConfigSrv.InitPolicyConfigData(); err != nil {
		return err
	}
	PolicyConfigSrv.SetupRun()

	//
	return nil
}
