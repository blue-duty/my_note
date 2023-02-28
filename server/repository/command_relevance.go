package repository

import (
	"tkbastion/server/model"
	_ "tkbastion/server/model"

	"gorm.io/gorm"
)

type CommandRelevanceRepository struct {
	DB *gorm.DB
}

func NewCommandRelevanceRepository(db *gorm.DB) *CommandRelevanceRepository {
	commandRelevanceRepository = &CommandRelevanceRepository{DB: db}
	return commandRelevanceRepository
}

func (r CommandRelevanceRepository) Create(o *model.CommandRelevance) (err error) {
	if err = r.DB.Create(o).Error; err != nil {
		return err
	}
	return nil
}

func (r CommandRelevanceRepository) DeleteBySetId(setId string) (err error) {
	return r.DB.Where("command_set_id = ?", setId).Delete(&model.CommandRelevance{}).Error
}

func (r CommandRelevanceRepository) FindByCommandStrategyId(strategyId string) (o []model.CommandRelevance, err error) {
	err = r.DB.Where("command_strategy_id = ?", strategyId).Find(&o).Error
	return
}

func (r CommandRelevanceRepository) FindCommandSetByStrategyId(id string) (o []model.CommandRelevance, err error) {
	err = r.DB.Where("command_set_id != ? and command_strategy_id = ?", "-", id).Find(&o).Error
	return
}

func (r CommandRelevanceRepository) DeleteByCommandStrategyIdAndCommandSetIdIsNotNull(policyId string) (err error) {
	err = r.DB.Where("command_strategy_id = ? and command_set_id != ?", policyId, "-").Delete(&model.CommandRelevance{}).Error
	return
}
func (r CommandRelevanceRepository) DeleteByCommandStrategyIdAndUserIdIsNotNull(policyId string) (err error) {
	return r.DB.Where("command_strategy_id = ? and user_id != ? ", policyId, "-").Delete(&model.CommandRelevance{}).Error
}
func (r CommandRelevanceRepository) DeleteByCommandStrategyIdAndAssetIdIsNotNull(policyId string) (err error) {
	return r.DB.Where("command_strategy_id = ? and asset_id != ? ", policyId, "-").Delete(&model.CommandRelevance{}).Error
}
func (r CommandRelevanceRepository) DeleteByCommandStrategyIdAndAssetGroupIdIsNotNull(policyId string) (err error) {
	return r.DB.Where("command_strategy_id = ? and asset_group_id != ? ", policyId, "-").Delete(&model.CommandRelevance{}).Error
}
func (r CommandRelevanceRepository) DeleteByCommandStrategyIdAndUserGroupIdIsNotNull(policyId string) (err error) {
	return r.DB.Where("command_strategy_id = ? and user_group_id != ? ", policyId, "-").Delete(&model.CommandRelevance{}).Error
}

func (r CommandRelevanceRepository) FindStrategyIdByOtherId(assetId, userId, groupId string) (o []string, err error) {
	err = r.DB.Table("command_relevances").Select("command_strategy_id").Where("asset_id = ? or user_id = ? or user_group_id = ?", assetId, userId, groupId).Find(&o).Error
	return
}

func (r CommandRelevanceRepository) FindStrategyIdByOthersId(assetId, userId, groupId, assetGroupId string) (o []string, err error) {
	err = r.DB.Table("command_relevances").Select("command_strategy_id").Where("asset_id = ? or user_id = ? or user_group_id = ? or asset_group_id = ?", assetId, userId, groupId, assetGroupId).Find(&o).Error
	return
}

func (r CommandRelevanceRepository) FindByUserIdOrUserGroupId(id []string) (o []string, err error) {
	err = r.DB.Table("command_relevances").Select("command_strategy_id").Where("user_id in ? or user_group_id in ?", id, id).Find(&o).Error
	return
}

// DeleteByStrategyId 通过策略id删除关联
func (r CommandRelevanceRepository) DeleteByStrategyId(strategy string) (err error) {
	return r.DB.Where("command_strategy_id = ?", strategy).Delete(&model.CommandRelevance{}).Error
}

// DeleteByCommandSetId 通过指令集id删除关联
func (r CommandRelevanceRepository) DeleteByCommandSetId(commandSetId string) (err error) {
	return r.DB.Where("command_set_id = ?", commandSetId).Delete(&model.CommandRelevance{}).Error
}

// DeleteByUserId 通过用户id删除关联
func (r CommandRelevanceRepository) DeleteByUserId(userId string) (err error) {
	return r.DB.Where("user_id = ?", userId).Delete(&model.CommandRelevance{}).Error
}

// DeleteByUserGroupId 通过用户组id删除关联
func (r CommandRelevanceRepository) DeleteByUserGroupId(userGroupId string) (err error) {
	return r.DB.Where("user_group_id = ?", userGroupId).Delete(&model.CommandRelevance{}).Error
}

// DeleteByAssetId 通过资产id删除关联
func (r CommandRelevanceRepository) DeleteByAssetId(assetId string) (err error) {
	return r.DB.Where("asset_id = ?", assetId).Delete(&model.CommandRelevance{}).Error
}

// DeleteByAssetGroupId 通过资产组id删除关联
func (r CommandRelevanceRepository) DeleteByAssetGroupId(assetGroupId string) (err error) {
	return r.DB.Where("asset_group_id = ?", assetGroupId).Delete(&model.CommandRelevance{}).Error
}

// DeleteByStrategyIdAndAssetId 通过策略id和设备id删除关联
func (r CommandRelevanceRepository) DeleteByStrategyIdAndAssetId(strategyId, assetId string) (err error) {
	return r.DB.Where("command_strategy_id = ? and asset_id = ?", strategyId, assetId).Delete(&model.CommandRelevance{}).Error
}
