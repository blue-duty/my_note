package repository

import (
	"gorm.io/gorm"
	"tkbastion/server/model"
)

type WorkOrderAssetRepository struct {
	DB *gorm.DB
}

func NewWorkOrderAssetRepository(db *gorm.DB) *WorkOrderAssetRepository {
	workOrderAssetRepository = &WorkOrderAssetRepository{DB: db}
	return workOrderAssetRepository
}

func (r *WorkOrderAssetRepository) Create(workOrderAsset *model.WorkOrderAsset) error {
	return r.DB.Create(workOrderAsset).Error
}

func (r *WorkOrderAssetRepository) DeleteByWorkOrderIdAndAssetId(workOrderId, assetId string) error {
	return r.DB.Where("work_order_id = ? and asset_id = ?", workOrderId, assetId).Delete(&model.WorkOrderAsset{}).Error
}

func (r *WorkOrderAssetRepository) DeleteByAssetId(id string) error {
	return r.DB.Where("asset_id = ? ", id).Delete(model.WorkOrderAsset{}).Error
}

func (r *WorkOrderAssetRepository) DeleteByWorkOrderId(id string) error {
	return r.DB.Where("work_order_id = ?", id).Delete(model.WorkOrderAsset{}).Error
}

func (r *WorkOrderAssetRepository) FindByWorkOrderIdAndAssetId(workOrderId, assetId string) (o []model.WorkOrderAsset, err error) {
	err = r.DB.Where("work_order_id = ? and asset_id = ?", workOrderId, assetId).Find(&o).Error
	return
}

func (r *WorkOrderAssetRepository) FindByWorkOrderId(id string) (o []model.WorkOrderAsset, err error) {
	err = r.DB.Where("Work_order_id = ?", id).Find(&o).Error
	return
}
