package repository

import (
	"gorm.io/gorm"
	"strings"
	"time"
	"tkbastion/server/model"
	"tkbastion/server/utils"
)

type IdentityConfigRepository struct {
	DB *gorm.DB
}

func NewIdentityConfigRepository(db *gorm.DB) *IdentityConfigRepository {
	identityConfigRepository := &IdentityConfigRepository{DB: db}
	return identityConfigRepository
}

func (r IdentityConfigRepository) IsEmpty() (result bool) {
	var count int64
	r.DB.Model(&model.IdentityConfig{}).Count(&count)
	if count == 0 {
		result = true
	} else {
		result = false
	}
	return
}

//	func (r IdentityConfigRepository) Create(o *model.IdentityConfig) error {
//		return r.DB.Create(o).Error
//	}
//
//	func (r IdentityConfigRepository) Update(o *model.IdentityConfig) error {
//		return r.DB.Select(`id`, `login_fail_times`, `expiration_time`, `password_length`, `password_num`, `password_lower`, `password_upper`, `password_special`, `expire_time`).UpdateColumns(o).Error
//	}
//
//	func (r IdentityConfigRepository) FindByID(id string) (o model.IdentityConfigDTO, err error) {
//		err = r.DB.Where("id = 1", id).First(&o).Error
//		return
//	}
//
//	func (r IdentityConfigRepository) FindById(id string) (o *model.IdentityConfig, err error) {
//		err = r.DB.Where("id = ?", id).First(&o).Error
//		return
//	}
//
//	func (r IdentityConfigRepository) FindConfig() (o *model.IdentityConfig, err error) {
//		err = r.DB.Where("id = ?", "1").First(&o).Error
//		return
//	}
//
//	func (r IdentityConfigRepository) Truncate() error {
//		return r.DB.Exec("TRUNCATE TABLE identity_configs").Error
//	}
func (r IdentityConfigRepository) FindLonginConfig() (*model.LoginConfig, error) {
	var o model.IdentityConfig
	err := r.DB.Where("id = ?", "1").First(&o).Error
	if err != nil {
		return nil, err
	}
	return &model.LoginConfig{
		ID:             o.ID,
		LoginLockWay:   o.LoginLockWay,
		AttemptTimes:   o.AttemptTimes,
		ContinuousTime: o.ContinuousTime,
		LockTime:       o.LockTime,
		LockIp:         o.LockIp,
	}, nil
}

func (r IdentityConfigRepository) FindPasswordConfig() (*model.PasswordConfig, error) {
	var o model.IdentityConfig
	err := r.DB.Where("id = ?", "1").First(&o).Error
	if err != nil {
		return nil, err
	}
	return &model.PasswordConfig{
		ID:                  o.ID,
		ForceChangePassword: o.ForceChangePassword,
		PasswordLength:      o.PasswordLength,
		PasswordCheck:       o.PasswordCheck,
		PasswordSameTimes:   o.PasswordSameTimes,
		PasswordCycle:       o.PasswordCycle,
		PasswordRemind:      o.PasswordRemind,
	}, nil
}

func (r IdentityConfigRepository) UpdatePasswordConfig(o *model.PasswordConfig) error {
	var identityConfig = model.IdentityConfig{
		ForceChangePassword: o.ForceChangePassword,
		PasswordLength:      o.PasswordLength,
		PasswordCheck:       o.PasswordCheck,
		PasswordSameTimes:   o.PasswordSameTimes,
		PasswordCycle:       o.PasswordCycle,
		PasswordRemind:      o.PasswordRemind,
	}
	identityConfigMap := utils.Struct2MapByStructTag(identityConfig)
	err := r.DB.Table("identity_configs").Where("id = ?", "1").Updates(identityConfigMap).Error
	return err
}
func (r IdentityConfigRepository) UpdateLoginConfig(o *model.LoginConfig) error {
	var identityConfig = model.IdentityConfig{
		LoginLockWay:   o.LoginLockWay,
		AttemptTimes:   o.AttemptTimes,
		ContinuousTime: o.ContinuousTime,
		LockTime:       o.LockTime,
	}
	err := r.DB.Where("id = ?", "1").Updates(identityConfig).Error
	return err
}

func (r IdentityConfigRepository) RecordIp(ip string) (err error) {
	var identityConfig model.IdentityConfig
	err = r.DB.Where("id = ?", "1").Find(&identityConfig).Error
	if err != nil {
		return
	}
	split := strings.Split(identityConfig.LockIp, ";")
	if strings.Contains(split[0], ip) {
		return
	}
	var lockIp []string
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	lockIp = append(lockIp, timeStr+" : "+ip)
	lockIp = append(lockIp, split...)
	lockIpStr := strings.Join(lockIp, " ; \n")
	if len(lockIpStr) > 2048 {
		lockIpStr = strings.Join(lockIp[:len(lockIp)-2], " ; \n")
	}
	identityConfig.LockIp = lockIpStr
	err = r.DB.Where("id = ?", "1").Updates(identityConfig).Error
	return
}
