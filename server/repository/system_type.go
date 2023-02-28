package repository

import (
	"context"
	"strings"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"gorm.io/gorm"
)

type SystemTypeRepository struct {
	baseRepository
}

var SystemTypeDto = new(SystemTypeRepository)

func (s *SystemTypeRepository) GetSystemTypeList(ctx context.Context) ([]dto.SystemType, error) {
	var systemTypes []model.SystemType
	err := s.GetDB(ctx).Find(&systemTypes).Error
	if err != nil {
		return make([]dto.SystemType, 0), err
	}
	var systemTypeList []dto.SystemType
	systemTypeList = make([]dto.SystemType, len(systemTypes))
	for i, systemType := range systemTypes {
		systemTypeList[i] = dto.SystemType{
			ID:      systemType.ID,
			Name:    systemType.Name,
			Type:    systemType.Type,
			Info:    systemType.Info,
			Default: systemType.Default,
		}
	}
	return systemTypeList, nil
}

// GetSystemTypeIDs 获取Windows和Linux系统类型的ID
func (s *SystemTypeRepository) GetSystemTypeIDs(ctx context.Context) (map[string]string, error) {
	systemTypeIDs := make(map[string]string)
	var systemTypes []model.SystemType
	err := s.GetDB(ctx).Where("name in (?)", []string{"LINUX", "WINDOWS"}).Find(&systemTypes).Error
	if err != nil {
		return systemTypeIDs, err
	}
	for _, systemType := range systemTypes {
		systemTypeIDs[systemType.Name] = systemType.ID
	}
	return systemTypeIDs, nil
}

// GetSystemTypeByNameOrAuto 根据名称或者自动获取
func (s *SystemTypeRepository) GetSystemTypeByNameOrAuto(ctx context.Context, name, auto string) ([]dto.SystemType, error) {
	var systemType []model.SystemType
	if name != "" {
		err := s.GetDB(ctx).Where("name like ?", "%"+name+"%").Find(&systemType).Error
		if err != nil {
			return []dto.SystemType{}, err
		}

	} else if auto != "" {
		var t string
		if strings.Contains("主机", auto) {
			t = "host"
		}
		if strings.Contains("网络", auto) {
			t = "network"
		}
		err := s.GetDB(ctx).Where("name like ? or type = ? or info like ?", "%"+auto+"%", t, "%"+auto+"%").Find(&systemType).Error
		if err != nil {
			return []dto.SystemType{}, err
		}
	} else {
		err := s.GetDB(ctx).Find(&systemType).Error
		if err != nil {
			return []dto.SystemType{}, err
		}
	}
	var systemTypeList []dto.SystemType
	systemTypeList = make([]dto.SystemType, len(systemType))
	for i, systemType := range systemType {
		systemTypeList[i] = dto.SystemType{
			ID:      systemType.ID,
			Name:    systemType.Name,
			Type:    systemType.Type,
			Info:    systemType.Info,
			Default: systemType.Default,
		}
	}
	return systemTypeList, nil
}

// CreateSystemType 新建
func (s *SystemTypeRepository) CreateSystemType(ctx context.Context, systemType *dto.SystemTypeForCreate) error {
	return s.GetDB(ctx).Create(&model.SystemType{
		ID:      utils.UUID(),
		Name:    systemType.Name,
		Type:    systemType.Type,
		Info:    systemType.Info,
		Default: false,
	}).Error
}

// CreateDefaultSystemType 创建默认系统类型
func (s *SystemTypeRepository) CreateDefaultSystemType(ctx context.Context) error {
	var systemType model.SystemType
	err := s.GetDB(ctx).Where(&model.SystemType{Default: true}).First(&systemType).Error
	if err == gorm.ErrRecordNotFound {
		err = s.GetDB(ctx).Create(&model.SystemType{
			ID:      utils.UUID(),
			Name:    "LINUX",
			Type:    "host",
			Info:    " ",
			Default: true,
		}).Error
		if err != nil {
			return err
		}
		err = s.GetDB(ctx).Create(&model.SystemType{
			ID:      utils.UUID(),
			Name:    "WINDOWS",
			Type:    "host",
			Info:    " ",
			Default: true,
		}).Error
		if err != nil {
			return err
		}
		err = s.GetDB(ctx).Create(&model.SystemType{
			ID:      utils.UUID(),
			Name:    "UNIX",
			Type:    "host",
			Info:    " ",
			Default: true,
		}).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateSystemType 更新
func (s *SystemTypeRepository) UpdateSystemType(ctx context.Context, systemType *dto.SystemTypeForUpdate, id string) error {
	return s.GetDB(ctx).Model(&model.SystemType{}).Where("id = ?", id).Updates(utils.Struct2MapByStructTag(model.SystemType{
		Name: systemType.Name,
		Info: systemType.Info,
	})).Error
}

// DeleteSystemType 删除
func (s *SystemTypeRepository) DeleteSystemType(ctx context.Context, id string) error {
	return s.GetDB(ctx).Delete(&model.SystemType{}, "id = ?", id).Error
}

// BatchDeleteSystemType 批量删除
func (s *SystemTypeRepository) BatchDeleteSystemType(ctx context.Context, ids []string) error {
	db := s.GetDB(ctx).Begin()
	for _, id := range ids {
		err := db.Delete(&model.SystemType{}, "id = ?", id).Error
		if err != nil {
			db.Rollback()
			return err
		}
	}
	db.Commit()
	return nil
}

// GetSystemTypeByName 根据名称获取
func (s *SystemTypeRepository) GetSystemTypeByName(ctx context.Context, name string) (model.SystemType, error) {
	var systemType model.SystemType
	err := s.GetDB(ctx).Where("name = ?", name).First(&systemType).Error
	return systemType, err
}

// GetSystemTypeNames 获取所有已存在的名称
func (s *SystemTypeRepository) GetSystemTypeNames(ctx context.Context) ([]string, error) {
	var systemType []string
	err := s.GetDB(ctx).Model(&model.SystemType{}).Pluck("name", &systemType).Error
	return systemType, err
}

// GetSystemTypeByNameID 根据名称,id获取
func (s *SystemTypeRepository) GetSystemTypeByNameID(ctx context.Context, id string) ([]string, error) {
	var systemType []string
	err := s.GetDB(ctx).Where("id != ?", id).Model(&model.SystemType{}).Pluck("name", &systemType).Error
	return systemType, err
}

func (s *SystemTypeRepository) GetSystemTypeByID(ctx context.Context, id string) (model.SystemType, error) {
	var systemType model.SystemType
	err := s.GetDB(ctx).Where("id = ?", id).First(&systemType).Error
	return systemType, err
}

// GetSystemTypeByIDs 根据ids获取
func (s *SystemTypeRepository) GetSystemTypeByIDs(ctx context.Context, ids []string) ([]model.SystemType, error) {
	var systemTypes []model.SystemType
	err := s.GetDB(ctx).Where("id in (?)", ids).Find(&systemTypes).Error
	return systemTypes, err
}
