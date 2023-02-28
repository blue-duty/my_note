package repository

import (
	"tkbastion/pkg/constant"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"gorm.io/gorm"
)

type WorkOrderNewRepository struct {
	DB *gorm.DB
}

func NewWorkOrderNewRepository(db *gorm.DB) *WorkOrderNewRepository {
	workOrderNewRepository = &WorkOrderNewRepository{DB: db}
	return workOrderNewRepository
}

func (r *WorkOrderNewRepository) Create(newWorkOrder *model.NewWorkOrder) error {
	return r.DB.Create(newWorkOrder).Error
}

func (r *WorkOrderNewRepository) FindById(id string) (newWorkOrder model.NewWorkOrder, err error) {
	err = r.DB.Where("id = ?", id).Find(&newWorkOrder).Error
	return
}

func (r *WorkOrderNewRepository) FindByApplyId(id string) (newWorkOrder []model.NewWorkOrder, err error) {
	err = r.DB.Where("apply_id = ?", id).Find(&newWorkOrder).Error
	return
}

func (r *WorkOrderNewRepository) DeleteByApplyId(id string) (err error) {
	err = r.DB.Where("apply_id = ?", id).Delete(&model.NewWorkOrder{}).Error
	return
}

func (r *WorkOrderNewRepository) FindByOrderId(id string) (newWorkOrder model.NewWorkOrder, err error) {
	err = r.DB.Where("order_id = ?", id).Find(&newWorkOrder).Error
	return
}

func (r *WorkOrderNewRepository) FindByApplyIdAndConditions(applyId, auto, OrderId, time, title, orderType, status string) (newWorkOrder []model.NewWorkOrder, err error) {
	db := r.DB.Table("new_work_order").Where("apply_id = ? and work_order_type = ?", applyId, "访问工单")
	if auto != "" {
		db = db.Where("order_id like ? or apply_time like ? or title like ? or work_order_type like ? or status like ?", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%")
	} else {
		if OrderId != "" {
			db = db.Where("order_id like ?", "%"+OrderId+"%")
		}
		if time != "" {
			db = db.Where("apply_time like ?", "%"+time+"%")
		}
		if title != "" {
			db = db.Where("title like ?", "%"+title+"%")
		}
		if orderType != "" {
			db = db.Where("work_order_type like ?", "%"+orderType+"%")
		}
		if status != "" {
			db = db.Where("status like ?", "%"+status+"%")
		}
	}
	err = db.Order("apply_time desc").Find(&newWorkOrder).Error
	return
}

func (r *WorkOrderNewRepository) FindByApproveDepIds(depIds []int64, auto, OrderId, applyTime, title, orderType, status string) (newWorkOrder []model.NewWorkOrder, err error) {
	db := r.DB.Where("department_id in (?) and status != ?", depIds, constant.NotSubmitted)
	if auto != "" {
		db = db.Where("order_id like ? or apply_time like ? or title like ? or work_order_type like ? or status like ?", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%")
	} else {
		if OrderId != "" {
			db = db.Where("order_id like ?", "%"+OrderId+"%")
		}
		if applyTime != "" {
			db = db.Where("apply_time like ?", "%"+applyTime+"%")
		}
		if title != "" {
			db = db.Where("title like ?", "%"+title+"%")
		}
		if orderType != "" {
			db = db.Where("work_order_type like ?", "%"+orderType+"%")
		}
		if status != "" {
			db = db.Where("status like ?", "%"+status+"%")
		}
	}
	err = db.Order("apply_time desc").Find(&newWorkOrder).Error
	return
}

func (r *WorkOrderNewRepository) UpdateById(id string, newWorkOrder *model.NewWorkOrder) (err error) {
	err = r.DB.Where("id = ?", id).Updates(newWorkOrder).Error
	return
}

func (r *WorkOrderNewRepository) UpdateByOrderId(orderId string, newWorkOrder *model.NewWorkOrder) (err error) {
	err = r.DB.Where("order_id = ?", orderId).Updates(newWorkOrder).Error
	return
}

// 查看某资产有无关联

func (r *WorkOrderNewRepository) CountByWorkOrderId(id string) (total int64, err error) {
	err = r.DB.Table("work_order_asset").Where("work_order_id = ?", id).Count(&total).Error
	return
}

func (r *WorkOrderNewRepository) FindByWorkOrderId(id string) (o []model.WorkOrderAsset, err error) {
	err = r.DB.Where("work_order_id = ?", id).Find(&o).Error
	return
}

// FindValidOrderByUserId 查看某用户的有效申请工单（审批通过，且在运维有效期内）
func (r *WorkOrderNewRepository) FindValidOrderByUserId(applyId string) (o []model.NewWorkOrder, err error) {
	err = r.DB.Where("apply_id = ? and status = ?", applyId, constant.Approved).Find(&o).Error
	return
}

// GetUnApprovalWorkOrder 查询用户所能看到的工单数量
func (r *WorkOrderNewRepository) GetUnApprovalWorkOrder(role string, depIds []int64) (o []model.NewWorkOrder, err error) {
	if role != constant.SystemAdmin && role != constant.DepartmentAdmin {
		return []model.NewWorkOrder{}, nil
	}
	err = r.DB.Where("department_id in (?) and status = ? ", depIds, constant.Submitted).Order("apply_time desc").Find(&o).Error
	return
}

// FindWorkOrderLogByDepIds 查询工单审批日志
func (r *WorkOrderNewRepository) FindWorkOrderLogByDepIds(ids []int64, auto, title, submitTime, approveTime, approveUsername, approveNickname, approveResult string) (o []model.NewWorkOrderLog, err error) {
	db := r.DB.Table("new_work_order_log").Where("department_id in (?)", ids)
	if auto != "" {
		db = db.Where("order_id like ? or title like ? or apply_time like ? or approve_time like ? or approve_username like ? or approve_nickname like ? or result like ? or department like ? or work_order_type like ?", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%")
	} else {
		if title != "" {
			db = db.Where("title like ?", "%"+title+"%")
		}
		if submitTime != "" {
			db = db.Where("submit_time like ?", "%"+submitTime+"%")
		}
		if approveTime != "" {
			db = db.Where("approve_time like ?", "%"+approveTime+"%")
		}
		if approveUsername != "" {
			db = db.Where("approve_username like ?", "%"+approveUsername+"%")
		}
		if approveNickname != "" {
			db = db.Where("approve_nickname like ?", "%"+approveNickname+"%")
		}
		if approveResult != "" {
			db = db.Where("approve_result like ?", "%"+approveResult+"%")
		}
	}
	err = db.Order("approve_time desc").Find(&o).Error
	return
}

func (r *WorkOrderNewRepository) FindAll() (o []model.NewWorkOrderLog, err error) {
	err = r.DB.Table("new_work_order_log").Order("approve_time desc").Find(&o).Error
	return
}

func (r *WorkOrderNewRepository) DeleteByOrderId(orderId string) (err error) {
	err = r.DB.Table("new_work_order_log").Where("order_id = ?", orderId).Delete(&model.NewWorkOrderLog{}).Error
	return
}

// 创建审批日志

func (r *WorkOrderNewRepository) CreateWorkOrderLog(workOrder *model.NewWorkOrder, account *model.UserNew, status, orderType, info string) error {
	var newWorkOrderLog = model.NewWorkOrderLog{
		ID:              utils.UUID(),
		OrderId:         workOrder.OrderId,
		Title:           workOrder.Title,
		WorkOrderType:   orderType,
		ApplyTime:       workOrder.ApplyTime,
		ApproveTime:     utils.NowJsonTime(),
		ApproveUsername: account.Username,
		ApproveNickname: account.Nickname,
		DepartmentId:    account.DepartmentId,
		Department:      account.DepartmentName,
		Result:          status,
		ApproveInfo:     info,
	}
	return r.DB.Table("new_work_order_log").Create(&newWorkOrderLog).Error
}
