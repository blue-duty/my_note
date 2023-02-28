package repository

import (
	"context"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"
)

type AssetGroupRepositoryNew struct {
	baseRepository
}

func (a *AssetGroupRepositoryNew) GetAssetGroupListBydsadd(ctx context.Context, departmentIds []int64, auto, agn, d string) ([]dto.AssetGroupForPage, error) {
	var assetGroups []dto.AssetGroupForPage
	db := a.GetDB(ctx).Table("new_asset_group").Select("new_asset_group.id,new_asset_group.name,new_asset_group.info,department.name as department,new_asset_group.created,count(asset_group_with_asset.asset_id) as count").Where("department_id in (?)", departmentIds)
	if auto != "" {
		db = db.Where("new_asset_group.name like ? or department like ? ", "%"+auto+"%", "%"+auto+"%")
	} else if agn != "" {
		db = db.Where("new_asset_group.name like ?", "%"+agn+"%")
	} else if d != "" {
		db = db.Where("department like ?", "%"+d+"%")
	}
	// 计算设备组关联设备的数量
	err := db.Joins("left join department on department.id = new_asset_group.department_id").Joins("left join asset_group_with_asset on asset_group_with_asset.asset_group_id = new_asset_group.id").Group("new_asset_group.id").Find(&assetGroups).Error
	return assetGroups, err
}

func (a *AssetGroupRepositoryNew) GetAssetGroupListByDepartmentIds(ctx context.Context, departmentIds []int64) ([]model.NewAssetGroup, error) {
	var assetGroups []model.NewAssetGroup
	err := a.GetDB(ctx).Where("department_id in (?)", departmentIds).Find(&assetGroups).Error
	return assetGroups, err
}

// CreateAssetGroup 新增资产组
func (a *AssetGroupRepositoryNew) CreateAssetGroup(ctx context.Context, assetGroup *dto.AssetGroupCreateRequest) error {
	aids := utils.IdHandle(assetGroup.Assets)
	assets := make([]model.AssetGroupWithAsset, len(aids))
	at := model.NewAssetGroup{
		Id:           utils.UUID(),
		Name:         assetGroup.Name,
		Info:         assetGroup.Info,
		Department:   assetGroup.Department,
		DepartmentId: assetGroup.DepartmentId,
		Created:      utils.NowJsonTime(),
	}
	for k, v := range aids {
		assets[k] = model.AssetGroupWithAsset{
			ID:           utils.UUID(),
			AssetGroupId: at.Id,
			AssetId:      v,
		}
	}
	db := a.GetDB(ctx).Begin()
	err := db.Create(&at).Error
	if err != nil {
		db.Rollback()
		return err
	}
	if len(assets) > 0 {
		err = db.Create(&assets).Error
		if err != nil {
			db.Rollback()
			return err
		}
	}
	db.Commit()
	return nil
}

// GetAssetGroupByName 通过名称查询资产组
func (a *AssetGroupRepositoryNew) GetAssetGroupByName(ctx context.Context, name string) (assetGroup model.NewAssetGroup, err error) {
	err = a.GetDB(ctx).Where("name = ?", name).First(&assetGroup).Error
	return
}

// GetAssetGroupByAssetId 通过资产id获取资产组
func (a *AssetGroupRepositoryNew) GetAssetGroupByAssetId(todo context.Context, assetId string) (assetGroupForAsset []dto.AssetGroupForAsset, err error) {
	var passports []string
	err = a.GetDB(todo).Model(model.PassPort{}).Where("asset_id = ?", assetId).Pluck("id", &passports).Error
	if err != nil {
		return
	}
	var assetGroups []model.NewAssetGroup
	err = a.GetDB(todo).Table("new_asset_group").Select(" new_asset_group.name, new_asset_group.department,new_asset_group.created").Joins("left join asset_group_with_asset on new_asset_group.id = asset_group_with_asset.asset_group_id").Where("asset_group_with_asset.asset_id in (?)", passports).Scan(&assetGroups).Error
	if err != nil {
		return
	}
	var mp = make(map[string]dto.AssetGroupForAsset)
	for _, v := range assetGroups {
		if _, ok := mp[v.Name]; !ok {
			mp[v.Name] = dto.AssetGroupForAsset{
				Name:        v.Name,
				Department:  v.Department,
				CreatedTime: v.Created.Time.Format("2006-01-02 15:04:05"),
			}
		}
	}

	for _, v := range mp {
		assetGroupForAsset = append(assetGroupForAsset, v)
	}

	return
}

// GetAssetGroupByNameAndId 通过名称和id查询资产组
func (a *AssetGroupRepositoryNew) GetAssetGroupByNameAndId(ctx context.Context, name string, id string) (assetGroup model.NewAssetGroup, err error) {
	err = a.GetDB(ctx).Where("name = ? and id != ?", name, id).First(&assetGroup).Error
	return
}

func (a *AssetGroupRepositoryNew) UpdateAssetGroup(todo context.Context, d *dto.AssetGroupUpdateRequest) error {
	return a.GetDB(todo).Model(&model.NewAssetGroup{}).Where("id = ?", d.ID).Updates(map[string]interface{}{
		"name": d.Name,
		"info": d.Info,
	}).Error
}

func (a *AssetGroupRepositoryNew) GetAssetGroupById(todo context.Context, id string) (assetGroup model.NewAssetGroup, err error) {
	err = a.GetDB(todo).Where("id = ?", id).First(&assetGroup).Error
	return
}

func (a *AssetGroupRepositoryNew) DeleteAllAssets(todo context.Context, id string) error {
	db := a.GetDB(todo).Begin()
	if err := db.Where("asset_group_id = ?", id).Delete(&model.AssetGroupWithAsset{}).Error; err != nil {
		db.Rollback()
		return err
	}
	err := db.Model(&model.NewAssetGroup{}).Where("id = ?", id).Update("count", 0).Error
	if err != nil {
		db.Rollback()
		return err
	}
	db.Commit()
	return nil
}

func (a *AssetGroupRepositoryNew) DeleteAssetGroup(todo context.Context, id string) error {
	return a.GetDB(todo).Where("id = ?", id).Delete(&model.NewAssetGroup{}).Error
}

// DeleteAssetGroupByIds 批量删除资产组
func (a *AssetGroupRepositoryNew) DeleteAssetGroupByIds(todo context.Context, ids []string) error {
	db := a.GetDB(todo).Begin()
	if err := db.Where("id in (?)", ids).Delete(&model.NewAssetGroup{}).Error; err != nil {
		db.Rollback()
		return err
	}
	db.Commit()
	return nil
}

// DeleteAssetGroupByDepartmentId 根据部门id删除资产组
func (a *AssetGroupRepositoryNew) DeleteAssetGroupByDepartmentId(todo context.Context, departmentId []int64) error {
	db := a.GetDB(todo).Begin()
	if err := db.Where("department_id in (?)", departmentId).Delete(&model.NewAssetGroup{}).Error; err != nil {
		db.Rollback()
		return err
	}
	db.Commit()
	return nil
}

// GetAssetGroupByIds 通过ids获取资产组
func (a *AssetGroupRepositoryNew) GetAssetGroupByIds(todo context.Context, ids []string) (assetGroups []model.NewAssetGroup, err error) {
	err = a.GetDB(todo).Where("id in (?)", ids).Find(&assetGroups).Error
	return
}

func (a *AssetGroupRepositoryNew) UpdateAssetGroupAsset(todo context.Context, id string, ids []string) error {
	db := a.GetDB(todo).Begin()
	if err := db.Model(&model.AssetGroupWithAsset{}).Where("asset_group_id = ?", id).Delete(&model.AssetGroupWithAsset{}).Error; err != nil {
		db.Rollback()
		return err
	}
	assetGroupWithAssets := make([]model.AssetGroupWithAsset, len(ids))
	for k, v := range ids {
		assetGroupWithAssets[k] = model.AssetGroupWithAsset{
			ID:           utils.UUID(),
			AssetGroupId: id,
			AssetId:      v,
		}
	}
	// 更改设备组设备计数
	if err := db.Model(&model.NewAssetGroup{}).Where("id = ?", id).Update("count", len(ids)).Error; err != nil {
		db.Rollback()
		return err
	}
	for _, v := range assetGroupWithAssets {
		if err := db.Create(&v).Error; err != nil {
			db.Rollback()
			return err
		}
	}
	db.Commit()
	return nil
}

func (a *AssetGroupRepositoryNew) GetAssetGroupAsset(todo context.Context, id string) (assetGroupWithAssets []model.AssetGroupWithAsset, err error) {
	err = a.GetDB(todo).Where("asset_group_id = ?", id).Find(&assetGroupWithAssets).Error
	return
}

func (a *AssetGroupRepositoryNew) GetPassportByAssetGroupId(todo context.Context, assetGroupId string) (passport []model.PassPort, err error) {
	var assets []string
	err = a.GetDB(todo).Model(&model.AssetGroupWithAsset{}).Where("asset_group_id in (?)", assetGroupId).Pluck("asset_id", &assets).Error
	if err != nil {
		return
	}
	err = a.GetDB(todo).Model(&model.PassPort{}).Where("asset_id in (?)", assets).Find(&passport).Error
	return
}

func (a *AssetGroupRepositoryNew) GetPassportIdsByAssetGroupIds(todo context.Context, assetGroupIds []string) (passportIds []string, err error) {
	err = a.GetDB(todo).Model(&model.AssetGroupWithAsset{}).Where("asset_group_id in ?", assetGroupIds).Pluck("asset_id", &passportIds).Error
	return
}
