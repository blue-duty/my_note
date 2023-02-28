package repository

import (
	"context"
	"tkbastion/server/dto"
	"tkbastion/server/model"
)

type CommandRepositoryNew struct {
	baseRepository
}

// GetCommandNew 获取指令
func (r *CommandRepositoryNew) GetCommandNew(c context.Context, commandID string) (dto.CommandForGet, error) {
	var command dto.CommandForGet
	err := r.GetDB(c).Model(&model.NewCommand{}).Where("id = ?", commandID).First(&command).Error
	return command, err
}

// GetCommandNewList 获取指令列表
func (r *CommandRepositoryNew) GetCommandNewList(c context.Context, cfs dto.CommandForSearch) ([]dto.CommandForPage, int64, error) {
	var commands []dto.CommandForPage
	var total int64
	db := r.GetDB(c).Model(&model.NewCommand{}).Where("user_id = ?", cfs.Uid)
	if cfs.Auto != "" {
		db = db.Where("name like ? or content like ?", "%"+cfs.Auto+"%", "%"+cfs.Auto+"%")
	} else if cfs.Name != "" {
		db = db.Where("name like ?", "%"+cfs.Name+"%")
	} else if cfs.Content != "" {
		db = db.Where("content like ?", "%"+cfs.Content+"%")
	}

	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = db.Order("created desc").Find(&commands).Error
	return commands, total, err
}

// GetCommandNewByName 获取指令
func (r *CommandRepositoryNew) GetCommandNewByName(c context.Context, name string) (dto.CommandForGet, error) {
	var command dto.CommandForGet
	err := r.GetDB(c).Model(&model.NewCommand{}).Where("name = ?", name).First(&command).Error
	return command, err
}

// GetCommandNewByIdAndName 查询名称是否存在
func (r *CommandRepositoryNew) GetCommandNewByIdAndName(c context.Context, commandID string, name string) (dto.CommandForGet, error) {
	var command dto.CommandForGet
	err := r.GetDB(c).Model(&model.NewCommand{}).Where("id != ? and name = ?", commandID, name).First(&command).Error
	return command, err
}

// CreateCommandNew 创建指令
func (r *CommandRepositoryNew) CreateCommandNew(c context.Context, command model.NewCommand) error {
	return r.GetDB(c).Create(&command).Error
}

// UpdateCommandNew 更新指令
func (r *CommandRepositoryNew) UpdateCommandNew(c context.Context, command dto.CommandForUpdate) error {
	return r.GetDB(c).Model(&model.NewCommand{}).Where("id = ?", command.ID).Updates(&model.NewCommand{
		Name:    command.Name,
		Content: command.Content,
		Info:    command.Info,
	}).Error
}

// DeleteCommandNew 删除指令
func (r *CommandRepositoryNew) DeleteCommandNew(c context.Context, commandID []string) error {
	return r.GetDB(c).Where("id in (?)", commandID).Delete(&model.NewCommand{}).Error
}

// GetCommandNewById 获取指令
func (r *CommandRepositoryNew) GetCommandNewById(c context.Context, commandID string) (dto.CommandForGet, error) {
	var command dto.CommandForGet
	err := r.GetDB(c).Model(&model.NewCommand{}).Where("id = ?", commandID).First(&command).Error
	return command, err
}
