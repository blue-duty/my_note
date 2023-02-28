package repository

import (
	"context"
	"encoding/base64"
	"strings"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"gorm.io/gorm"
)

type AssetRepositoryNew struct {
	baseRepository
}

var AssetNewDao = new(AssetRepositoryNew)

func (a *AssetRepositoryNew) GetAssetList(ctx context.Context, FastSearch dto.AssetForSearch) (assets []model.NewAsset, err error) {
	if FastSearch.DepartmentIds == nil {
		return
	}
	db := a.GetDB(ctx).Where("department_id in (?)", FastSearch.DepartmentIds)
	if FastSearch.IP != "" {
		db = db.Where("ip like ?", "%"+FastSearch.IP+"%")
	} else if FastSearch.Name != "" {
		db = db.Where("name like ?", "%"+FastSearch.Name+"%")
	} else if FastSearch.AssetType != "" {
		var assetType []string
		err = a.GetDB(ctx).Model(&model.SystemType{}).Where("name like ?", "%"+FastSearch.AssetType+"%").Select("id").Find(&assetType).Error
		if err != nil {
			return nil, err
		}
		db = db.Where("asset_type in (?)", assetType)
	} else if FastSearch.Department != "" {
		var department []int64
		err := a.GetDB(ctx).Model(&model.Department{}).Where("name like ?", "%"+FastSearch.Department+"%").Select("id").Find(&department).Error
		if err != nil {
			return nil, err
		}
		db = db.Where("department_id in (?)", department)
	} else if FastSearch.Auto != "" {
		var department []int64
		err = a.GetDB(ctx).Model(&model.Department{}).Where("name like ?", "%"+FastSearch.Auto+"%").Select("id").Find(&department).Error
		var assetType []string
		err = a.GetDB(ctx).Model(&model.SystemType{}).Where("name like ?", "%"+FastSearch.Auto+"%").Select("id").Find(&assetType).Error
		if err != nil {
			return nil, err
		}
		db = db.Where("ip like ? or name like ? or asset_type in (?) or department_id in (?)",
			"%"+FastSearch.Auto+"%", "%"+FastSearch.Auto+"%", assetType, department)
	}
	err = db.Find(&assets).Error
	return
}

func (a *AssetRepositoryNew) GetPassport(ctx context.Context, id string) (asset []model.PassPort, err error) {
	err = a.GetDB(ctx).Where("asset_id = ?", id).Find(&asset).Error
	return
}

func (a *AssetRepositoryNew) GetPassportConfig(ctx context.Context, id string) (configmap map[string]string, err error) {
	var passport []model.PassportConfiguration
	err = a.GetDB(ctx).Where("passport_id = ?", id).Find(&passport).Error
	if err != nil {
		return
	}
	configmap = make(map[string]string)
	for _, v := range passport {
		configmap[v.Name] = v.Value
	}
	return
}

func (a *AssetRepositoryNew) CreateAsset(ctx context.Context, asset model.NewAsset) (err error) {
	err = a.GetDB(ctx).Create(&asset).Error
	return
}

func (a *AssetRepositoryNew) GetAssetTypeId(ctx context.Context, assetType string) (id []string, err error) {
	err = a.GetDB(ctx).Model(&model.SystemType{}).Where("name like ?", "%"+assetType+"%").Select("id").Find(&id).Error
	return
}

func (a *AssetRepositoryNew) GetAssetCountByDepIds(ctx context.Context, depIds []int64) (count int64, err error) {
	err = a.GetDB(ctx).Model(&model.NewAsset{}).Where("department_id in ?", depIds).Count(&count).Error
	return
}

// IncrAssetCount 设备资产数+
func (a *AssetRepositoryNew) IncrAssetCount(ctx context.Context, assetid string, count int) error {
	return a.GetDB(ctx).Model(&model.NewAsset{}).Where("id = ?", assetid).Update("pass_port_count", gorm.Expr("pass_port_count + ?", count)).Error
}

// DecrAssetCount 设备资产数-
func (a *AssetRepositoryNew) DecrAssetCount(ctx context.Context, assetid string, count int) error {
	return a.GetDB(ctx).Model(&model.NewAsset{}).Where("id = ?", assetid).Update("pass_port_count", gorm.Expr("pass_port_count - ?", count)).Error
}

// GetAssetListByDepartmentIds 根据部门id获取资产列表
func (a *AssetRepositoryNew) GetAssetListByDepartmentIds(ctx context.Context, departmentIds []int64) (assets []model.NewAsset, err error) {
	err = a.GetDB(ctx).Where("department_id in (?)", departmentIds).Find(&assets).Error
	return
}

func (a *AssetRepositoryNew) CreatePassport(ctx context.Context, passport model.PassPort) (err error) {
	err = a.GetDB(ctx).Create(&passport).Error
	return
}

// UpdatePassportForPasswd 修改账号密码
func (a *AssetRepositoryNew) UpdatePassportForPasswd(ctx context.Context, id string, passwd, lastPasswd string, lastChange utils.JsonTime) error {
	// 查询账号信息
	db := a.GetDB(ctx).Begin()
	var passport model.PassPort
	var err error
	err = db.Where("id = ?", id).First(&passport).Error
	if err != nil {
		db.Rollback()
		return err
	}
	// 获取账号对应设备的所有用户名相同的账号id
	var passportIds []string
	// 查询时忽略大小写
	err = db.Model(&model.PassPort{}).Where("asset_id = ? and passport = ?", passport.AssetId, passport.Passport).Select("id").Find(&passportIds).Error
	if err != nil {
		db.Rollback()
		return err
	}

	// 更新账号密码
	err = db.Model(&model.PassPort{}).Where("id in (?)", passportIds).Updates(map[string]interface{}{
		"password":                  passwd,
		"last_password":             lastPasswd,
		"last_change_password_time": lastChange,
	}).Error
	if err != nil {
		db.Rollback()
		return err
	}

	return db.Commit().Error
}

func (a *AssetRepositoryNew) GetAssetByIP(ctx context.Context, ip string) (asset model.NewAsset, err error) {
	err = a.GetDB(ctx).Where("ip = ?", ip).First(&asset).Error
	return
}

func (a *AssetRepositoryNew) GetAssetByID(ctx context.Context, id string) (asset model.NewAsset, err error) {
	err = a.GetDB(ctx).Where("id = ?", id).First(&asset).Error
	return
}

func (a *AssetRepositoryNew) UpdateAsset(ctx context.Context, asset model.NewAsset) (err error) {
	db := a.GetDB(ctx).Begin()
	assetMap := utils.Struct2MapByStructTag(asset)
	err = db.Model(&model.NewAsset{}).Where("id = ?", asset.ID).Updates(assetMap).Error
	if err != nil {
		db.Rollback()
	}

	err = db.Model(&model.PassPort{}).Where("asset_id = ?", asset.ID).Updates(map[string]interface{}{
		"asset_type":    asset.AssetType,
		"asset_name":    asset.Name,
		"ip":            asset.IP,
		"department_id": asset.DepartmentId,
	}).Error
	if err != nil {
		log.Errorf("更新部门时出错: %v", err)
		db.Rollback()
	}
	db.Commit()
	return
}

func deleteByIds(db *gorm.DB, id string) error {
	err := db.Where("asset_id = ?", id).Delete(model.CommandRelevance{}).Error
	if err != nil {
		return err
	}
	// 用户授权的删除
	//err = db.Where("asset_id = ?", asset.ID).Delete(model.OperateAuth{}).Error
	err = db.Where("asset_id in (select id from pass_ports where asset_id = ?)", id).Delete(model.AssetGroupWithAsset{}).Error
	if err != nil {
		return err
	}
	// 删除设备账号之前删除设备账号权限报表
	err = db.Where("asset_account_id in (select id from pass_ports where asset_id = ?)", id).Delete(model.AssetAuthReportForm{}).Error
	if err != nil {
		return err
	}
	err = db.Where("asset_id = ?", id).Delete(model.PassPort{}).Error
	if err != nil {
		return err
	}
	err = db.Where("id = ?", id).Delete(model.NewAsset{}).Error
	if err != nil {
		return err
	}
	if err := db.Where("passport_id in (select id from pass_ports where asset_id = ?)", id).Delete(model.PassportConfiguration{}).Error; err != nil {
		return err
	}
	return nil
}

func (a *AssetRepositoryNew) DeleteAsset(ctx context.Context, id string) error {
	db := a.GetDB(ctx).Begin()
	err := deleteByIds(db, id)
	if err != nil {
		db.Rollback()
		return err
	}
	db.Commit()
	return nil
}

func (a *AssetRepositoryNew) GetPassPortByID(ctx context.Context, id string) (passport model.PassPort, err error) {
	err = a.GetDB(ctx).Where("id = ?", id).First(&passport).Error
	return
}

func (a *AssetRepositoryNew) UpdatePassport(todo context.Context, port model.PassPort) error {
	passMap := utils.Struct2MapByStructTag(port)
	db := a.GetDB(todo).Begin()
	if err := db.Model(&model.PassPort{
		ID: port.ID,
	}).Updates(&passMap).Error; err != nil {
		db.Rollback()
	}
	if port.IsSshKey == 1 {
		if err := db.Model(&model.PassPort{}).Where("id = ?", port.ID).Updates(map[string]interface{}{
			"password":                  "",
			"last_password":             "",
			"last_change_password_time": nil,
		}).Error; err != nil {
			db.Rollback()
		}
	} else {
		if err := db.Model(&model.PassPort{}).Where("id = ?", port.ID).Updates(map[string]interface{}{
			"private_key": "",
			"passphrase":  "",
		}).Error; err != nil {
			db.Rollback()
		}
	}

	return db.Commit().Error
}

// DeletePassword 删除设备账号的密码
func (a *AssetRepositoryNew) DeletePassword(ctx context.Context, id string) error {
	return a.GetDB(ctx).Model(&model.PassPort{}).Where("id = ?", id).Updates(map[string]interface{}{
		"password":                  "",
		"last_password":             "",
		"last_change_password_time": nil,
	}).Error

}

// BatchDeleteAsset 批量删除资产
func (a *AssetRepositoryNew) BatchDeleteAsset(ctx context.Context, ids []string) error {
	db := a.GetDB(ctx).Begin()
	for _, id := range ids {
		err := deleteByIds(db, id)
		if err != nil {
			db.Rollback()
			return err
		}
	}
	db.Commit()
	return nil
}

func (a *AssetRepositoryNew) DeletePassport(todo context.Context, id string) error {
	db := a.GetDB(todo).Begin()
	if err := db.Where("passport_id = ?", id).Delete(&model.PassportConfiguration{}).Error; err != nil {
		log.Errorf("delete passport configuration error: %v", err)
	}
	if err := db.Where("id = ?", id).Delete(&model.PassPort{}).Error; err != nil {
		db.Rollback()
		return err
	}
	if err := db.Where("asset_id = ?", id).Delete(&model.AssetGroupWithAsset{}).Error; err != nil {
		db.Rollback()
		return err
	}
	db.Commit()
	return nil
}

// BatchDeletePassport 批量删除帐号
func (a *AssetRepositoryNew) BatchDeletePassport(ctx context.Context, ids []string) error {
	db := a.GetDB(ctx).Begin()
	if err := db.Where("id in (?)", ids).Delete(&model.PassPort{}).Error; err != nil {
		db.Rollback()
		return err
	}
	if err := db.Where("passport_id in (?)", ids).Delete(&model.PassportConfiguration{}).Error; err != nil {
		log.Errorf("delete passport configuration error: %v", err)
	}
	if err := db.Where("asset_id in (?)", ids).Delete(&model.AssetGroupWithAsset{}).Error; err != nil {
		db.Rollback()
		return err
	}
	db.Commit()
	return nil
}

func (a *AssetRepositoryNew) DisablePassport(todo context.Context, id string) error {
	return a.GetDB(todo).Model(&model.PassPort{}).Where("id = ?", id).Update("status", "disable").Error
}

func (a *AssetRepositoryNew) EnablePassport(todo context.Context, id string) error {
	return a.GetDB(todo).Model(&model.PassPort{}).Where("id = ?", id).Update("status", "enable").Error
}

// GetPassportListByIds 通过ids获取ssh帐号列表
func (a *AssetRepositoryNew) GetPassportListByIds(ctx context.Context, ids []string) (passport []model.PassPort, err error) {
	err = a.GetDB(ctx).Where("id in (?)", ids).Find(&passport).Error
	for i := range passport {
		// 解密密码
		if passport[i].IsSshKey != 1 {
			origData, err := base64.StdEncoding.DecodeString(passport[i].Password)
			if err != nil {
				continue
			}
			decryptedCBC, err := utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
			if err != nil {
				continue
			}
			passport[i].Password = string(decryptedCBC)
		}

		if passport[i].Passphrase != "" {
			// 解密私钥密钥
			origData, err := base64.StdEncoding.DecodeString(passport[i].Passphrase)
			if err != nil {
				continue
			}
			decryptedCBC, err := utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
			if err != nil {
				continue
			}
			passport[i].Passphrase = string(decryptedCBC)
		}
	}
	return
}

// GetAssetWithPassport 查询设备帐号信息
func (a *AssetRepositoryNew) GetAssetWithPassport(ctx context.Context, id string) (asset []dto.ForRelate, err error) {
	var passport []model.PassPort
	err = a.GetDB(ctx).Where("asset_id = ?", id).Find(&passport).Error
	if err != nil {
		return
	}
	asset = make([]dto.ForRelate, len(passport))
	for i, v := range passport {
		asset[i].ID = v.ID
		asset[i].Name = v.Name
	}
	return
}

func (a *AssetRepositoryNew) GetAssetByIds(todo context.Context, ds []string) (assets []model.NewAsset, err error) {
	err = a.GetDB(todo).Where("id in (?)", ds).Find(&assets).Error
	return
}

func (a *AssetRepositoryNew) GetAssetByName(todo context.Context, name string) (asset model.NewAsset, err error) {
	err = a.GetDB(todo).Where("name = ?", name).First(&asset).Error
	return
}

func (a *AssetRepositoryNew) BatchEditAsset(todo context.Context, d *dto.AssetForBatchUpdate) error {
	db := a.GetDB(todo).Begin()
	if d.Department != "" {
		if err := db.Model(&model.NewAsset{}).Where("id in (?)", d.AssetIds).Update("department_id", utils.String2Int(d.Department)).Error; err != nil {
			db.Rollback()
			return err
		}
		if err := db.Model(&model.PassPort{}).Where("asset_id in (?)", d.AssetIds).Update("department_id", utils.String2Int(d.Department)).Error; err != nil {
			db.Rollback()
			return err
		}
	}
	if d.AssetType != "" {
		if err := db.Model(&model.NewAsset{}).Where("id in (?)", d.AssetIds).Update("asset_type", d.AssetType).Error; err != nil {
			db.Rollback()
			return err
		}
	}
	if d.LoginType != "" {
		if err := db.Model(&model.PassPort{}).Where("asset_id in (?)", d.AssetIds).Update("login_type", d.LoginType).Error; err != nil {
			db.Rollback()
			return err
		}
	}
	if d.Protocol != "" && d.Port != "" {
		if err := db.Model(&model.PassPort{}).Where("asset_id in (?)", d.AssetIds).Update("protocol", d.Protocol).Error; err != nil {
			db.Rollback()
			return err
		}
		if err := db.Model(&model.PassPort{}).Where("asset_id in (?)", d.AssetIds).Update("port", utils.String2Int(d.Port)).Error; err != nil {
			db.Rollback()
			return err
		}
	}
	if d.Password != "" {
		pass, err := utils.Encoder.Encode([]byte(d.Password))
		if err != nil {
			log.Errorf("Encode Error: %v", err)
		}
		d.Password = string(pass)
		if err := db.Model(&model.PassPort{}).Where("asset_id in (?)", d.AssetIds).Update("password", d.Password).Error; err != nil {
			db.Rollback()
			return err
		}
	}
	db.Commit()
	return nil
}

func (a *AssetRepositoryNew) GetPassportByDepartmentIds(todo context.Context, ds []int64) (passport []model.PassPort, err error) {
	err = a.GetDB(todo).Where("department_id in (?) and status = ?", ds, "enable").Find(&passport).Error
	return
}

func (a *AssetRepositoryNew) GetDepartmentByDepartmentId(todo context.Context, id int64) (department model.Department, err error) {
	err = a.GetDB(todo).Where("id = ?", id).First(&department).Error
	return
}

func (a *AssetRepositoryNew) GetAssetCountByDepId(todo context.Context, depId int64) (count int64, err error) {
	err = a.GetDB(todo).Model(model.NewAsset{}).Where("department_id = ?", depId).Count(&count).Error
	return
}

func (a *AssetRepositoryNew) GetPassportById(todo context.Context, id string) (passport model.PassPort, err error) {
	err = a.GetDB(todo).Where("id = ?", id).First(&passport).Error
	return
}

func (a *AssetRepositoryNew) GetPassportWithPasswordById(todo context.Context, id string) (passport model.PassPort, err error) {
	err = a.GetDB(todo).Where("id = ?", id).First(&passport).Error
	if err != nil {
		log.Error("获取设备帐号信息失败", err)
		return
	}
	if passport.Password != "" {
		origData, err := base64.StdEncoding.DecodeString(passport.Password)
		if err != nil {
			log.Error("解密设备帐号密码失败", err)
			return passport, err
		}
		decryptedCBC, err := utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
		if err != nil {
			log.Error("解密设备帐号密码失败", err)
			return passport, err
		}
		passport.Password = string(decryptedCBC)
	}
	if passport.Protocol == "ssh" {
		if passport.Passphrase != "" {
			origData, err := base64.StdEncoding.DecodeString(passport.Passphrase)
			if err != nil {
				log.Error("解密设备帐号密码失败", err)
				return passport, err
			}
			decryptedCBC, err := utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
			if err != nil {
				log.Error("解密设备帐号密码失败", err)
				return passport, err
			}
			passport.Passphrase = string(decryptedCBC)
		}
	}
	//log.Infof("获取设备帐号信息成功,%#v", passport)
	return
}

func (a *AssetRepositoryNew) GetPassportToExport(todo context.Context, ids []int64) ([]dto.AssetForExport, error) {
	var DepChainName func(id int64) (string, error)
	DepChainName = func(depId int64) (string, error) {
		var depChainName string
		var err error
		delDepNode, err := departmentRepository.FindById(depId)
		if nil != err {
			return "", err
		}

		if delDepNode.FatherId == -1 {
			depChainName += "根部门."
		} else {
			depChainName, err = DepChainName(delDepNode.FatherId)
			if nil != err {
				return "", err
			}
			depChainName += delDepNode.Name
			depChainName += "."
		}
		return depChainName, nil
	}

	var passport []dto.AssetForExport
	var assets []model.PassPort
	err := a.GetDB(todo).Where("department_id in (?)", ids).Find(&assets).Error
	if err != nil {
		return nil, err
	}
	passport = make([]dto.AssetForExport, len(assets))
	for i, v := range assets {
		dpn, err := DepChainName(v.DepartmentId)
		if err != nil {
			continue
		}
		var systemType model.SystemType
		err = a.GetDB(todo).Where("id = ?", v.AssetType).First(&systemType).Error
		if err != nil {
			continue
		}
		passport[i].Name = v.AssetName
		passport[i].IP = v.Ip
		passport[i].Protocol = v.Protocol
		passport[i].Port = v.Port
		passport[i].AssetType = systemType.Name
		passport[i].Department = dpn[:len(dpn)-1]
		if v.LoginType == "auto" {
			passport[i].LoginType = "自动登录"
		} else {
			passport[i].LoginType = "手动登录"
		}
		passport[i].Passport = v.Passport
		if v.PrivateKey != "" {
			passport[i].SshKey = "是"
		} else {
			passport[i].SshKey = "否"
		}
		passport[i].SftpPath = v.SftpPath
		if v.PassportType == "Administrators" {
			passport[i].PassportType = "管理员"
		} else {
			passport[i].PassportType = "普通用户"
		}
		if v.Status == "enable" {
			passport[i].Status = "已启用"
		} else {
			passport[i].Status = "已禁用"
		}
	}
	return passport, nil
}

// GetPassportWithPasswdForExport 包含密码的导出
func (a *AssetRepositoryNew) GetPassportWithPasswdForExport(todo context.Context, id string) (passport []dto.PassportWithPasswordForExport, err error) {
	var DepChainName func(id int64) (string, error)
	DepChainName = func(depId int64) (string, error) {
		var depChainName string
		var err error
		delDepNode, err := departmentRepository.FindById(depId)
		if nil != err {
			return "", err
		}

		if delDepNode.FatherId == -1 {
			depChainName += "根部门."
		} else {
			depChainName, err = DepChainName(delDepNode.FatherId)
			if nil != err {
				return "", err
			}
			depChainName += delDepNode.Name
			depChainName += "."
		}
		return depChainName, nil
	}

	var pp []model.PassPort
	if id != "" {
		err = a.GetDB(todo).Where("id = ?", id).Find(&pp).Error
	} else {
		err = a.GetDB(todo).Find(&pp).Error
	}
	if err != nil {
		return nil, err
	}
	passport = make([]dto.PassportWithPasswordForExport, len(pp))
	for i, v := range pp {
		dpn, err := DepChainName(v.DepartmentId)
		if err != nil {
			continue
		}
		var systemType model.SystemType
		err = a.GetDB(todo).Where("id = ?", v.AssetType).First(&systemType).Error
		if err != nil {
			continue
		}
		passport[i].Name = v.AssetName
		passport[i].IP = v.Ip
		passport[i].Protocol = v.Protocol
		passport[i].Port = v.Port
		passport[i].AssetType = systemType.Name
		passport[i].Department = dpn[:len(dpn)-1]
		if v.LoginType == "auto" {
			passport[i].LoginType = "自动登录"
		} else {
			passport[i].LoginType = "手动登录"
		}
		passport[i].Passport = v.Passport
		if v.PrivateKey != "" {
			passport[i].SshKey = "是"
		} else {
			passport[i].SshKey = "否"
		}
		passport[i].SftpPath = v.SftpPath
		if v.PassportType == "Administrators" {
			passport[i].PassportType = "管理员"
		} else {
			passport[i].PassportType = "普通用户"
		}
		if v.Status == "enable" {
			passport[i].Status = "已启用"
		} else {
			passport[i].Status = "已禁用"
		}

		if v.Password != "" {
			origData, err := base64.StdEncoding.DecodeString(v.Password)
			if err != nil {
				log.Error("解密密码失败", err)
			}
			decryptedCBC, err := utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
			if err != nil {
				log.Error("解密密码失败", err)
			}
			passport[i].Password = string(decryptedCBC)
		} else {
			passport[i].Password = "未设置"
		}
	}
	return passport, nil
}

func (a *AssetRepositoryNew) GetPassportByIds(todo context.Context, ids []string) (passport []model.PassPort, err error) {
	err = a.GetDB(todo).Where("id in (?)", ids).Find(&passport).Error
	return
}

func (a *AssetRepositoryNew) CreateByAssetAndPassport(todo context.Context, asset model.NewAsset, passport model.PassPort) error {
	db := a.GetDB(todo).Begin()
	if err := db.Create(&asset).Error; err != nil {
		db.Rollback()
		return err
	}
	passport.AssetId = asset.ID
	if err := db.Create(&passport).Error; err != nil {
		db.Rollback()
		return err
	}
	db.Commit()
	return nil
}

func DepChainName(depId int64) (depChainName string, err error) {
	delDepNode, err := departmentRepository.FindById(depId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return "", err
	}

	if -1 == delDepNode.FatherId {
		depChainName += "根部门."
	} else {
		depChainName, err = DepChainName(delDepNode.FatherId)
		if nil != err {
			return "", err
		}
		depChainName += delDepNode.Name
		depChainName += "."
	}
	return
}

func (a *AssetRepositoryNew) GetDetailAssetById(todo context.Context, id string) (asset dto.AssetForDetail, err error) {
	var assetModel model.NewAsset
	err = a.GetDB(todo).Where("id = ?", id).First(&assetModel).Error
	if err != nil {
		return
	}
	asset.Name = assetModel.Name
	asset.IP = assetModel.IP
	var sytemType string
	err = a.GetDB(todo).Model(&model.SystemType{}).Where("id = ?", assetModel.AssetType).Pluck("name", &sytemType).Error
	if err != nil {
		return
	}
	asset.AssetType = sytemType
	asset.Department, err = DepChainName(assetModel.DepartmentId)
	if err != nil {
		return
	}
	asset.Department = asset.Department[:len(asset.Department)-1]
	asset.CreatedTime = assetModel.Created.Format("2006-01-02 15:04:05")
	asset.Info = assetModel.Info
	return
}

func (a *AssetRepositoryNew) GetPassportByAssetId(todo context.Context, id string) (passport []dto.PassportForAsset, err error) {
	var passportModel []model.PassPort
	err = a.GetDB(todo).Where("asset_id = ?", id).Find(&passportModel).Error
	if err != nil {
		return
	}
	passport = make([]dto.PassportForAsset, len(passportModel))
	for i, v := range passportModel {
		if v.Passport == "" {
			passport[i].Passport = "空用户"
		} else {
			passport[i].Passport = v.Passport
		}
		passport[i].Protocol = v.Protocol
		passport[i].Port = v.Port
		passport[i].Created = v.Created.Format("2006-01-02 15:04:05")
		passport[i].Status = v.Status
	}
	return
}

// GetOpsPolicyByAssetId 获取设备对应的所有运维策略
func (a *AssetRepositoryNew) GetOpsPolicyByAssetId(todo context.Context, id string) (opsPolicy []dto.AssetForPolicy, err error) {
	var pp []model.PassPort
	err = a.GetDB(todo).Table("pass_ports").Where("asset_id = ?", id).Find(&pp).Error
	if err != nil {
		log.Errorf("获取设备的账号失败: %v", err)
		return
	}
	var mp = make(map[dto.AssetForPolicy]bool)
	for _, p := range pp {
		var operateAuth []model.OperateAuth
		err = a.GetDB(todo).Where("relate_asset like ?", "%"+p.ID+"%").Find(&operateAuth).Error
		if err != nil {
			log.Errorf("获取运维策略失败: %v", err)
			return
		}

		// 获取设备组
		var assetGroup []model.AssetGroupWithAsset
		err = a.GetDB(todo).Where("asset_id = ?", p.ID).Find(&assetGroup).Error
		if err != nil {
			log.Errorf("获取设备组失败: %v", err)
			return
		}
		var operateAuth2 []model.OperateAuth
		for _, ag := range assetGroup {
			err = a.GetDB(todo).Where("relate_asset_group like ?", "%"+ag.AssetGroupId+"%").Find(&operateAuth2).Error
			if err != nil {
				log.Errorf("获取运维策略失败: %v", err)
				return
			}
			operateAuth = append(operateAuth, operateAuth2...)
		}

		for _, v := range operateAuth {
			uids := utils.IdHandle(v.RelateUser)
			for _, uid := range uids {
				var user model.UserNew
				err = a.GetDB(todo).Where("id = ?", uid).First(&user).Error
				if err != nil {
					log.Errorf("获取用户信息失败: %v", err)
					continue
				}
				var afp dto.AssetForPolicy
				afp.Passport = p.Passport
				afp.Protocol = p.Protocol
				afp.Username = user.Username
				afp.Nickname = user.Nickname
				dp, err := DepChainName(v.DepartmentId)
				if err != nil {
					log.Errorf("获取部门信息失败: %v", err)
					return []dto.AssetForPolicy{}, err
				}
				afp.Policy = v.Name + "[" + dp[:len(dp)-1] + "]"

				if _, ok := mp[afp]; !ok {
					mp[afp] = true
					opsPolicy = append(opsPolicy, afp)
				}
			}
		}
	}
	return
}

// GetCmdPolicyByAssetId 获取设备对应的所有指令策略
func (a *AssetRepositoryNew) GetCmdPolicyByAssetId(todo context.Context, id string) (cmdPolicy []dto.AssetForCommandPolicy, err error) {
	var pp []model.PassPort
	err = a.GetDB(todo).Table("pass_ports").Where("asset_id = ?", id).Find(&pp).Error
	if err != nil {
		return
	}

	var mp = make(map[dto.AssetForCommandPolicy]bool)

	for _, p := range pp {
		var pids []string
		err = a.GetDB(todo).Table("command_relevances").Where("asset_id = ?", p.ID).Pluck("command_strategy_id", &pids).Error
		if err != nil {
			return
		}
		var assetGroup []model.AssetGroupWithAsset
		err = a.GetDB(todo).Where("asset_id = ?", p.ID).Find(&assetGroup).Error
		if err != nil {
			log.Errorf("获取设备组失败: %v", err)
			return
		}
		var pids2 []string
		for _, ag := range assetGroup {
			err = a.GetDB(todo).Table("command_relevances").Where("asset_group_id = ?", ag.AssetGroupId).Pluck("command_strategy_id", &pids2).Error
			if err != nil {
				return
			}
			pids = append(pids, pids2...)
		}
		for _, pid := range pids {
			var commandStrategy model.CommandStrategy
			err = a.GetDB(todo).Where("id = ?", pid).First(&commandStrategy).Error
			var uids []string
			err = a.GetDB(todo).Table("command_relevances").Where("command_strategy_id = ? and user_id != '-'", pid).Pluck("user_id", &uids).Error
			if err != nil {
				continue
			}
			for _, uid := range uids {
				var user model.UserNew
				err = a.GetDB(todo).Where("id = ?", uid).First(&user).Error
				if err != nil {
					continue
				}
				var afp dto.AssetForCommandPolicy
				afp.Passport = p.Passport
				afp.Protocol = p.Protocol
				afp.Username = user.Username
				afp.Nickname = user.Nickname
				dp, err := DepChainName(commandStrategy.DepartmentId)
				if err != nil {
					continue
				}
				afp.Policy = commandStrategy.Name + "[" + dp[:len(dp)-1] + "]"
				afp.Level = commandStrategy.Level
				afp.Action = commandStrategy.Action

				if _, ok := mp[afp]; !ok {
					mp[afp] = true
					cmdPolicy = append(cmdPolicy, afp)
				}
			}
		}
	}
	return
}

func (a *AssetRepositoryNew) Truncate() error {
	return a.GetDB(context.TODO()).Exec("truncate table new_assets").Error
}

func (a *AssetRepositoryNew) DeleteByDepartmentId(ctx context.Context, id []int64) ([]model.NewAsset, error) {
	db := a.GetDB(ctx).Begin()
	var assets []model.NewAsset
	err := db.Where("department_id in (?)", id).Find(&assets).Error
	if err != nil {
		db.Rollback()
		return nil, err
	}
	for _, asset := range assets {
		err = deleteByIds(db, asset.ID)
		if err != nil {
			db.Rollback()
			return nil, err
		}
	}
	db.Commit()
	return assets, nil
}

func (a *AssetRepositoryNew) CreateByPassport(todo context.Context, pp model.PassPort) error {
	err := a.GetDB(todo).Model(&model.NewAsset{}).Where("id = ?", pp.AssetId).Update("pass_port_count", gorm.Expr("pass_port_count + ?", 1)).Error
	if err != nil {
		return err
	}
	return a.GetDB(todo).Create(&pp).Error
}

func (a *AssetRepositoryNew) GetPassportByAssetIds(todo context.Context, ids []string) (passport []model.PassPort, err error) {
	err = a.GetDB(todo).Where("asset_id in (?)", ids).Find(&passport).Error
	return
}

func (a *AssetRepositoryNew) GetAllPassport(todo context.Context) (passport []model.PassPort, err error) {
	err = a.GetDB(todo).Find(&passport).Error
	return
}

// UpdatePassportStatus 更新账户状态
func (a *AssetRepositoryNew) UpdatePassportStatus(todo context.Context, id string, status int) error {
	return a.GetDB(todo).Model(&model.PassPort{}).Where("id = ?", id).Update("active", status).Error
}

// PassportAdvanced 设备账户高级配置保存
func (a *AssetRepositoryNew) PassportAdvanced(todo context.Context, req dto.PassPortForCreate, id string) error {
	db := a.GetDB(todo).Begin()
	err := db.Model(&model.PassportConfiguration{}).Where("passport_id = ?", id).Delete(model.PassportConfiguration{}).Error
	if err != nil {
		db.Rollback()
		return err
	}
	var assetAdvancedSetting []model.PassportConfiguration
	if req.RdpDomain != "" {
		assetAdvancedSetting = append(assetAdvancedSetting, model.PassportConfiguration{
			PassportId: id,
			Name:       "rdp_domain",
			Value:      req.RdpDomain,
		})
	}
	if req.RdpEnableDrive == "是" {
		assetAdvancedSetting = append(assetAdvancedSetting, model.PassportConfiguration{
			PassportId: id,
			Name:       "rdp_enable_drive",
			Value:      "true",
		})
		if req.RdpDriveId != "" {
			assetAdvancedSetting = append(assetAdvancedSetting, model.PassportConfiguration{
				PassportId: id,
				Name:       "rdp_drive_path",
				Value:      req.RdpDriveId,
			})
		}
	}
	if req.Protocol == "x11" {
		if req.AppSerId != "" && req.TermProgram != "" && req.DisplayProgram != "" {
			assetAdvancedSetting = append(assetAdvancedSetting, model.PassportConfiguration{
				PassportId: id,
				Name:       "x11_appserver_id",
				Value:      req.AppSerId,
			})
			assetAdvancedSetting = append(assetAdvancedSetting, model.PassportConfiguration{
				PassportId: id,
				Name:       "x11_term_program",
				Value:      req.TermProgram,
			})
			assetAdvancedSetting = append(assetAdvancedSetting, model.PassportConfiguration{
				PassportId: id,
				Name:       "x11_display_program",
				Value:      req.DisplayProgram,
			})
		}
	}

	if len(assetAdvancedSetting) > 0 {
		err = db.Model(&model.PassportConfiguration{}).Create(&assetAdvancedSetting).Error
		if err != nil {
			db.Rollback()
			return err
		}
	}

	db.Commit()
	return nil
}

// GetAssetCount 获取资产的总数量
func (a *AssetRepositoryNew) GetAssetCount(todo context.Context) (int, error) {
	var assrtCount int64
	err := a.GetDB(todo).Model(&model.NewAsset{}).Count(&assrtCount).Error
	if err != nil {
		return 0, err
	}
	return int(assrtCount), nil
}

func (a *AssetRepositoryNew) UpdateAssetByAuthResourceCount(todo context.Context, count int) error {
	// 查询所有的资产的数量
	var assrtCount int64
	err := a.GetDB(todo).Model(&model.NewAsset{}).Count(&assrtCount).Error
	if err != nil {
		return err
	}
	if int(assrtCount) < count {
		return nil
	}

	// 如果资产的数量大于授权的数量，那么就需要删除多余的资产
	var assets []string
	err = a.GetDB(todo).Model(&model.NewAsset{}).Order("created desc").Limit(int(assrtCount)-count).Pluck("id", &assets).Error
	if err != nil {
		return err
	}

	// 删除资产
	if err = a.BatchDeleteAsset(todo, assets); err != nil {
		return err
	}
	return nil
}

// GetPassportRecord 密码查看页面密码记录
func (a *AssetRepositoryNew) GetPassportRecord(todo context.Context, auto, passport, assetIp, assetName, systemType, protocol string) (records []dto.PasswdView, err error) {
	db := a.GetDB(todo).Table("pass_ports p").Select("p.id,p.passport,a.ip,s.name as system_type,a.name as asset_name,p.protocol,d.name as department").Joins("left join new_assets a on p.asset_id = a.id").Joins("left join system_type s on s.id = a.asset_type").Joins("left join department d on a.department_id = d.id")
	if auto != "" {
		db = db.Where("p.passport like ? OR a.ip like ? OR a.name like ? OR a.asset_type like ? OR p.protocol like ?", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%")
	} else if passport != "" {
		db = db.Where("p.passport like ?", "%"+passport+"%")
	} else if assetIp != "" {
		db = db.Where("a.ip like ?", "%"+assetIp+"%")
	} else if assetName != "" {
		db = db.Where("a.name like ?", "%"+assetName+"%")
	} else if systemType != "" {
		db = db.Where("s.name like ?", "%"+systemType+"%")
	} else if protocol != "" {
		db = db.Where("p.protocol like ?", "%"+protocol+"%")
	}
	err = db.Find(&records).Error

	return
}

// GetPassportRecordById 获取单条账号密码记录
func (a *AssetRepositoryNew) GetPassportRecordById(todo context.Context, id string) (record dto.PasswdViewDetail, err error) {
	err = a.GetDB(todo).Table("pass_ports p").Select("a.name as asset_name,a.ip as asset_ip,s.name as system_type,p.passport,p.protocol,p.password as new_passwd,p.last_password as old_passwd,if(p.last_change_password_time is null,'尚未修改',DATE_FORMAT(p.last_change_password_time,'%Y-%m-%d %H:%i:%s')) as last_change_time").Joins("left join new_assets a on p.asset_id = a.id").Joins("left join system_type s on s.id = a.asset_type").Joins("left join department d on a.department_id = d.id").Where("p.id = ?", id).Find(&record).Error
	if record.OldPasswd != "" {
		origData, err := base64.StdEncoding.DecodeString(record.OldPasswd)
		if err != nil {
			log.Error("解密密码失败", err)
			return record, err
		}
		decryptedCBC, err := utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
		if err != nil {
			log.Error("解密密码失败", err)
			return record, err
		}
		record.OldPasswd = string(decryptedCBC)
	}

	if record.NewPasswd != "" {
		origData, err := base64.StdEncoding.DecodeString(record.NewPasswd)
		if err != nil {
			log.Error("解密密码失败", err)
			return record, err
		}
		decryptedCBC, err := utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
		if err != nil {
			log.Error("解密密码失败", err)
			return record, err
		}
		record.NewPasswd = string(decryptedCBC)
	}
	record.Protocol = strings.ToUpper(record.Protocol)

	return record, err
}

func (a *AssetRepositoryNew) GetPassPortByAssetIDAndUsername(todo context.Context, id string, username string, protocol string) (passport model.PassPort, err error) {
	err = a.GetDB(todo).Where("asset_id = ? and passport = ? and protocol = ?", id, username, protocol).First(&passport).Error
	return
}
