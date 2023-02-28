package repository

import (
	"gorm.io/gorm"
	"tkbastion/server/model"
)

type WorkOrderApprovalLogRepository struct {
	DB *gorm.DB
}

func NewWorkOrderApprovalLogRepository(db *gorm.DB) *WorkOrderApprovalLogRepository {
	workOrderApprovalLogRepository = &WorkOrderApprovalLogRepository{DB: db}
	return workOrderApprovalLogRepository
}

func (r *WorkOrderApprovalLogRepository) Create(workOrderApprovalLog *model.WorkOrderApprovalLog) error {
	return r.DB.Create(workOrderApprovalLog).Error
}

func (r *WorkOrderApprovalLogRepository) FindByWorkOrderId(id string) (o []model.WorkOrderApprovalLog, err error) {
	err = r.DB.Order("number asc").Where("work_order_id = ?", id).Find(&o).Error
	return
}

func (r *WorkOrderApprovalLogRepository) UpdateByWorkOrderIdAndDepartmentId(workOrderId string, number int, departmentId int64, o *model.WorkOrderApprovalLog) (err error) {
	if number == 0 {
		err = r.DB.Model(&model.WorkOrderApprovalLog{}).Where("work_order_id = ? and department_id = ?", workOrderId, departmentId).Updates(o).Error
	} else {
		err = r.DB.Model(&model.WorkOrderApprovalLog{}).Where("work_order_id = ? and department_id = ? and number = ?", workOrderId, departmentId, number).Updates(o).Error
	}
	return
}
