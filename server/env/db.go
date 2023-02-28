package env

import (
	"fmt"
	"tkbastion/pkg/config"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func SetupDB() *gorm.DB {
	global.Config = config.GlobalCfg
	var logMode logger.Interface
	// TODO gorm日志级别, 即数据表查询输出的日志级别, 删除(注释)debug
	if global.Config.Debug {
		logMode = logger.Default.LogMode(logger.Info)
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

	return db
}
