package repository

import (
	"context"
	"encoding/base64"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"gorm.io/gorm"
)

type PasswdChangeRepository struct {
	baseRepository
}

var PasswdChangeRepo = new(PasswdChangeRepository)

func (r PasswdChangeRepository) Find(c context.Context, name, runType, generateRule, auto string) ([]dto.PasswdChange, error) {
	var passwdChanges []dto.PasswdChange
	db := r.GetDB(c).Table("passwd_change").Select("id, name, run_type_name as run_type, generate_rule_name as generate_rule")
	if name != "" {
		db = db.Where("name LIKE ?", "%"+name+"%")
	} else if runType != "" {
		db = db.Where("run_type_name LIKE ?", "%"+runType+"%")
	} else if generateRule != "" {
		db = db.Where("generate_rule_name LIKE ?", "%"+generateRule+"%")
	} else if auto != "" {
		db = db.Where("name LIKE ? OR run_type_name LIKE ? OR generate_rule_name LIKE ?", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%")
	}

	err := db.Find(&passwdChanges).Error
	return passwdChanges, err
}

func (r PasswdChangeRepository) FindAll(c context.Context) ([]model.PasswdChange, error) {
	var passwdChanges []model.PasswdChange
	err := r.GetDB(c).Table("passwd_change").Find(&passwdChanges).Error
	return passwdChanges, err
}

func (r PasswdChangeRepository) FindByID(c context.Context, id string) (model.PasswdChange, error) {
	var passwdChange model.PasswdChange
	err := r.GetDB(c).Table("passwd_change").Where("id = ?", id).First(&passwdChange).Error
	return passwdChange, err
}

func (r PasswdChangeRepository) Create(c context.Context, passwdChange *model.PasswdChange, deviceIDs, deviceGroupIDs []string) error {
	tx := r.GetDB(c).Begin()
	err := tx.Table("passwd_change").Create(&passwdChange).Error
	if err != nil {
		tx.Rollback()
	}
	for _, deviceID := range deviceIDs {
		err = tx.Table("passwd_change_device").Create(&model.PasswdChangeDevice{
			PasswdChangeID: passwdChange.ID,
			DeviceID:       deviceID,
		}).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	for _, deviceGroupID := range deviceGroupIDs {
		err = tx.Table("passwd_change_device_group").Create(&model.PasswdChangeDeviceGroup{
			PasswdChangeID: passwdChange.ID,
			DeviceGroupID:  deviceGroupID,
		}).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func (r PasswdChangeRepository) NameIsDuplicate(c context.Context, name, id string) (bool, error) {
	db := r.GetDB(c).Model(&model.PasswdChange{}).Where("name = ?", name)
	if id != "" {
		db = db.Where("id != ?", id)
	}
	var pc model.PasswdChange
	err := db.First(&pc).Error
	if err == nil {
		return true, nil
	} else if err == gorm.ErrRecordNotFound {
		return false, nil
	} else {
		return false, err
	}
}

func (r PasswdChangeRepository) GetAllSshDevice(c context.Context) (forRelate []dto.ForRelate, err error) {
	var devices []model.PassPort
	err = r.GetDB(c).Model(&model.PassPort{}).Where("protocol = ? OR protocol = ?", "ssh", "telnet").Find(&devices).Error
	if err != nil {
		return
	}

	st, err := SystemTypeDto.GetSystemTypeIDs(c)
	if err != nil {
		return
	}

	forRelate = make([]dto.ForRelate, 0, len(devices))
	for _, device := range devices {
		if device.Protocol == "telnet" && device.AssetType != st["WINDOWS"] {
			continue
		}
		var pp dto.ForRelate
		pp.ID = device.ID
		depChinaName, err := DepChainName(device.DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
		}
		pp.Name = device.AssetName + "[" + device.Ip + "]" + "[" + device.Passport + "]" + "[" + device.Protocol + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
		forRelate = append(forRelate, pp)
	}
	return
}

func (r PasswdChangeRepository) GetAllhDeviceGroup(c context.Context) (forRelate []dto.ForRelate, err error) {
	var deviceGroups []model.NewAssetGroup
	err = r.GetDB(c).Model(&model.NewAssetGroup{}).Find(&deviceGroups).Error
	if err != nil {
		return
	}

	forRelate = make([]dto.ForRelate, 0, len(deviceGroups))
	for _, deviceGroup := range deviceGroups {
		var pp dto.ForRelate
		pp.ID = deviceGroup.Id
		depChinaName, err := DepChainName(deviceGroup.DepartmentId)
		if nil != err {
			log.Errorf("获取设备组部门链错误: %v", err)
		}
		pp.Name = deviceGroup.Name + "[" + depChinaName[:len(depChinaName)-1] + "]"
		forRelate = append(forRelate, pp)
	}
	return
}

func (r PasswdChangeRepository) Update(c context.Context, passwdChange *model.PasswdChange) error {
	if passwdChange.Password != "" {
		encryptedCBC, err := utils.AesEncryptCBC([]byte(passwdChange.Password), global.Config.EncryptionPassword)
		if err != nil {
			log.Error("加密失败", err)
		}
		passwdChange.Password = base64.StdEncoding.EncodeToString(encryptedCBC)
	}

	err := r.GetDB(c).Table("passwd_change").Where("id = ?", passwdChange.ID).Updates(&passwdChange).Error
	return err
}

func (r PasswdChangeRepository) Delete(c context.Context, id string) error {
	return r.GetDB(c).Table("passwd_change").Where("id = ?", id).Delete(&model.PasswdChange{}).Error
}

// UpdatePasswdChangeDevice 改密策略关联设备
func (r PasswdChangeRepository) UpdatePasswdChangeDevice(c context.Context, passwdChangeID string, deviceIDs []string) error {
	// 删除原有关联
	// 开启事务
	tx := r.GetDB(c).Begin()
	err := tx.Table("passwd_change_device").Where("passwd_change_id = ?", passwdChangeID).Delete(&model.PasswdChangeDevice{}).Error
	if err != nil {
		tx.Rollback()
		return err
	}
	// 新增关联
	for _, deviceID := range deviceIDs {
		err = tx.Table("passwd_change_device").Create(&model.PasswdChangeDevice{
			PasswdChangeID: passwdChangeID,
			DeviceID:       deviceID,
		}).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// UpdatePasswdChangeDeviceGroup 改密策略关联设备组
func (r PasswdChangeRepository) UpdatePasswdChangeDeviceGroup(c context.Context, passwdChangeID string, deviceGroupIDs []string) error {
	// 删除原有关联
	// 开启事务
	tx := r.GetDB(c).Begin()
	err := tx.Table("passwd_change_device_group").Where("passwd_change_id = ?", passwdChangeID).Delete(&model.PasswdChangeDeviceGroup{}).Error
	if err != nil {
		tx.Rollback()
		return err
	}
	// 新增关联
	for _, deviceGroupID := range deviceGroupIDs {
		err = tx.Table("passwd_change_device_group").Create(&model.PasswdChangeDeviceGroup{
			PasswdChangeID: passwdChangeID,
			DeviceGroupID:  deviceGroupID,
		}).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// FindPasswdChangeDevice 改密策略已关联设备
func (r PasswdChangeRepository) FindPasswdChangeDevice(c context.Context, passwdChangeID string) ([]model.PasswdChangeDevice, error) {
	var passwdChangeDevices []model.PasswdChangeDevice
	err := r.GetDB(c).Table("passwd_change_device").Where("passwd_change_id = ?", passwdChangeID).Find(&passwdChangeDevices).Error
	return passwdChangeDevices, err
}

// GetPasswordPolcyByIdForEdit 获取密码策略
func (r PasswdChangeRepository) GetPasswordPolcyByIdForEdit(c context.Context, id string) (model.PasswdChange, error) {
	var passwordPolicy model.PasswdChange
	err := r.GetDB(c).Where("id = ?", id).First(&passwordPolicy).Error
	passwordPolicy.Password = ""
	return passwordPolicy, err
}

// FindPasswdChangeDeviceGroup 改密策略已关联设备组
func (r PasswdChangeRepository) FindPasswdChangeDeviceGroup(c context.Context, passwdChangeID string) ([]model.PasswdChangeDeviceGroup, error) {
	var passwdChangeDeviceGroups []model.PasswdChangeDeviceGroup
	err := r.GetDB(c).Table("passwd_change_device_group").Where("passwd_change_id = ?", passwdChangeID).Find(&passwdChangeDeviceGroups).Error
	return passwdChangeDeviceGroups, err
}

// FindPasswdChangeLog 获取改密记录
func (r PasswdChangeRepository) FindPasswdChangeLog(c context.Context, auto, name, assetName, assetIp, passport, result string) ([]dto.PasswdChangeResult, error) {
	var passwdChangeResults []dto.PasswdChangeResult
	db := r.GetDB(c).Table("passwd_change_result").Select(" if(passwd_change.name is null,'策略已删除',passwd_change.name) as name,DATE_FORMAT(passwd_change_result.change_time, '%Y-%m-%d %H:%i:%s') as change_time,passwd_change_result.result,if(pass_ports.passport is not null,passport,'账号已删除') as passport,if(new_assets.name is null,'设备已删除',new_assets.name) as asset_name,if(new_assets.ip is null,'设备已删除',new_assets.ip) as asset_ip").Joins("left join passwd_change on passwd_change.id = passwd_change_result.passwd_change_id").Joins("left join new_assets on new_assets.id = passwd_change_result.device_id").Joins("left join pass_ports on pass_ports.id = passwd_change_result.account_id")
	if auto != "" {
		db = db.Where("pc.name LIKE ?  OR a.name LIKE ? OR a.ip LIKE ? OR p.passport LIKE ? OR pc.result LIKE ?", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%")
	} else if name != "" {
		db = db.Where("pc.name LIKE ?", "%"+name+"%")
	} else if assetName != "" {
		db = db.Where("a.name LIKE ?", "%"+assetName+"%")
	} else if assetIp != "" {
		db = db.Where("a.ip LIKE ?", "%"+assetIp+"%")
	} else if passport != "" {
		db = db.Where("p.passport LIKE ?", "%"+passport+"%")
	} else if result != "" {
		db = db.Where("pc.result LIKE ?", "%"+result+"%")
	}
	err := db.Order("passwd_change_result.change_time desc").Find(&passwdChangeResults).Error
	return passwdChangeResults, err
}

// FindPasswdChangeLogStatistical 获取改密结果统计
func (r PasswdChangeRepository) FindPasswdChangeLogStatistical(c context.Context, auto, assetName, assetIp, passport string) ([]dto.PasswdChangeResultStatistical, error) {
	var passwdChangeResults []dto.PasswdChangeResultStatistical
	// 成功 as success,失败 as failure
	db := r.GetDB(c).Table("passwd_change_result").Select("if(p.passport is not null,passport,'账号已删除') as passport,if(a.name is null,'设备已删除',a.name) as asset_name,if(a.ip is null,'设备已删除',a.ip) as asset_ip,sum(IF(passwd_change_result.result = '成功', 1, 0)) as success,SUM(IF(passwd_change_result.result = '失败', 1, 0)) as failure").Joins("left join new_assets a on a.id = passwd_change_result.device_id").Joins("left join pass_ports p on p.id = passwd_change_result.account_id").Group("a.name,a.ip,p.passport")
	if auto != "" {
		db = db.Where("a.name LIKE ? OR a.ip LIKE ? OR p.passport LIKE ?", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%")
	} else if assetName != "" {
		db = db.Where("a.name LIKE ?", "%"+assetName+"%")
	} else if assetIp != "" {
		db = db.Where("a.ip LIKE ?", "%"+assetIp+"%")
	} else if passport != "" {
		db = db.Where("p.passport LIKE ?", "%"+passport+"%")
	}
	err := db.Find(&passwdChangeResults).Error

	return passwdChangeResults, err
}

// FindPasswdChangeLogStatisticalDetail 改密结果统计详情
func (r PasswdChangeRepository) FindPasswdChangeLogStatisticalDetail(c context.Context, assetIp string, success bool) (result []dto.PasswdChangeResult, err error) {
	db := r.GetDB(c).Table("passwd_change_result").Select("if(pc.name is null,'策略已删除',pc.name) as name,DATE_FORMAT(passwd_change_result.change_time, '%Y-%m-%d %H:%i:%s') as change_time,passwd_change_result.result,if(p.passport is not null,passport,'账号已删除') as passport,if(a.name is null,'设备已删除',a.name) as asset_name,if(a.ip is null,'设备已删除',a.ip) as asset_ip").Joins("left join new_assets a on a.id = passwd_change_result.device_id").Joins("left join pass_ports p on p.id = passwd_change_result.account_id").Joins("left join passwd_change pc on pc.id = passwd_change_result.passwd_change_id")
	if success {
		db = db.Where("a.ip = ? AND passwd_change_result.result = '成功'", assetIp)
	} else {
		db = db.Where("a.ip = ? AND passwd_change_result.result = '失败'", assetIp)
	}
	err = db.Find(&result).Error
	return
}

// FindPasswdChangeAccountIds 获取改密策略所绑定所有账号id
func (r PasswdChangeRepository) FindPasswdChangeAccountIds(c context.Context, passwdChangeID string) ([]string, error) {
	var accountIds []string
	err := r.GetDB(c).Table("passwd_change_device").Where("passwd_change_id = ?", passwdChangeID).Pluck("device_id", &accountIds).Error
	// 获取设备组下所有设备id
	var deviceGroupIds []string
	err = r.GetDB(c).Table("passwd_change_device_group").Where("passwd_change_id = ?", passwdChangeID).Pluck("device_group_id", &deviceGroupIds).Error
	if err != nil {
		return nil, err
	}
	for _, deviceGroupId := range deviceGroupIds {
		var deviceIds []string
		err = r.GetDB(c).Table("asset_group_with_asset").Where("asset_group_id = ?", deviceGroupId).Pluck("asset_id", &deviceIds).Error
		accountIds = append(accountIds, deviceIds...)
	}

	// 去重
	var x []string
	m := make(map[string]bool)
	for _, v := range accountIds {
		if _, ok := m[v]; !ok {
			m[v] = true
			x = append(x, v)
		}
	}
	return x, err
}

// WritePasswdChangeLog 编写改密日志
func (r PasswdChangeRepository) WritePasswdChangeLog(c context.Context, log model.PasswdChangeResult) error {
	return r.GetDB(c).Create(&log).Error
}
