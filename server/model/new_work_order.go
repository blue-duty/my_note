package model

import "tkbastion/server/utils"

type NewWorkOrder struct {
	ID      string `gorm:"type:varchar(128);primary_key;comment:'申请id'" json:"id"`
	OrderId string `gorm:"index;type:varchar(128);comment:'工单id'" json:"orderId"`

	Title         string `gorm:"index;type:varchar(128);comment:'工单标题'" json:"title"`
	Department    string `gorm:"type:varchar(128);comment:'部门机构'" json:"department"`
	DepartmentId  int64  `gorm:"index;type:bigint;comment:'部门机构id'" json:"departmentId"`
	Role          string `gorm:"index;type:varchar(128);comment:'角色'" json:"role"`
	RoleId        string `gorm:"index;type:varchar(128);comment:'角色Id'" json:"roleId"`
	WorkOrderType string `gorm:"type:varchar(128);comment:'工单类型'" json:"workOrderType"`

	// 工单状态 (未提交 已提交 已过期 已关闭 审批通过 审批拒绝)
	Status      string `gorm:"type:varchar(128);comment:'申请状态'" json:"status"`
	Description string `gorm:"type:varchar(128);comment:'描述'" json:"description" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`

	// 申请人信息
	ApplyUser string         `gorm:"type:varchar(128);comment:'申请人'" json:"applyUser"`
	ApplyId   string         `gorm:"type:varchar(128);comment:'申请人Id'" json:"applyId"`
	ApplyTime utils.JsonTime `gorm:"type:datetime(3);comment:'申请时间'" json:"applyTime"`

	// 审批人信息
	ApproveUser string `gorm:"type:varchar(128);comment:'审批人'" json:"approveUser"`
	ApproveId   string `gorm:"type:varchar(128);comment:'审批人Id'" json:"approveId"`

	// 是否永久有效
	IsPermanent *bool          `gorm:"type:tinyint(1);comment:'是否永久有效'" json:"isPermanent"`
	BeginTime   utils.JsonTime `gorm:"type:datetime(3);comment:'开始时间'" json:"beginTime"`
	EndTime     utils.JsonTime `gorm:"type:datetime(3);comment:'结束时间'" json:"endTime"`

	// 是否上传、下载、水印
	IsUpload    *bool `gorm:"type:tinyint(1);comment:'是否上传'" json:"isUpload"`
	IsDownload  *bool `gorm:"type:tinyint(1);comment:'是否下载'" json:"isDownload"`
	IsWatermark *bool `gorm:"type:tinyint(1);comment:'是否水印'" json:"isWatermark"`
}

func (r *NewWorkOrder) TableName() string {
	return "new_work_order"
}

// 审批日志表

type WorkOrderApprovalLog struct {
	ID               string         `gorm:"type:varchar(128);primary_key;comment:'申请id'" json:"id"`
	WorkOrderId      string         `gorm:"index;type:varchar(128);comment:'工单id'" json:"workOrderId"`
	Number           int            `gorm:"index;type:int;comment:'序号'" json:"number"`
	ApprovalUsername string         `gorm:"type:varchar(128);comment:'审批人'" json:"approvalUsername"`
	ApprovalNickname string         `gorm:"type:varchar(128);comment:'审批人姓名'" json:"approvalNickname"`
	ApprovalId       string         `gorm:"index;type:varchar(128);comment:'审批人Id'" json:"approvalId"`
	Department       string         `gorm:"type:varchar(128);comment:'部门机构'" json:"department"`
	DepartmentId     int64          `gorm:"index;type:bigint;comment:'部门机构id'" json:"departmentId"`
	ApprovalDate     utils.JsonTime `gorm:"type:datetime(3);default:null;comment:'审批时间'" json:"approvalDate"`
	Result           string         `gorm:"type:varchar(128);comment:'审批结果'" json:"result"`
	ApproveWay       string         `gorm:"type:varchar(128);comment:'审批方式'" json:"approveWay"`
	IsFinalApprove   bool           `gorm:"type:tinyint(1);comment:'是否最终审批'" json:"isFinalApprove"`
	ApprovalInfo     string         `gorm:"type:varchar(128);comment:'审批意见'" json:"approvalInfo"`
}

func (r *WorkOrderApprovalLog) TableName() string {
	return "work_order_approval_log"
}

type WorkOrderAsset struct {
	ID          string `gorm:"type:varchar(128);primary_key;comment:'工单id'" json:"id"`
	WorkOrderId string `gorm:"type:varchar(128);comment:'工单id'" json:"workOrderId"`
	AssetId     string `gorm:"type:varchar(128);comment:'资产id'" json:"assetId"`
}

func (r *WorkOrderAsset) TableName() string {
	return "work_order_asset"
}

type NewWorkOrderLog struct {
	ID              string         `gorm:"type:varchar(128);primary_key;comment:'申请id'" json:"id"`
	OrderId         string         `gorm:"index;type:varchar(128);comment:'工单id'" json:"orderId"`
	Title           string         `gorm:"type:varchar(128);comment:'标题/指令'" json:"title"`
	WorkOrderType   string         `gorm:"type:varchar(128);comment:'工单类型'" json:"workOrderType"`
	ApplyTime       utils.JsonTime `gorm:"type:datetime(3);comment:'提交时间'" json:"applyTime"`
	ApproveTime     utils.JsonTime `gorm:"type:datetime(3);comment:'审批时间'" json:"approveTime"`
	ApproveUsername string         `gorm:"type:varchar(128);comment:'审批人'" json:"approveUsername"`
	ApproveNickname string         `gorm:"type:varchar(128);comment:'审批人姓名'" json:"approveNickname"`
	Department      string         `gorm:"type:varchar(128);comment:'部门机构'" json:"department"`
	DepartmentId    int64          `gorm:"index;type:bigint;comment:'部门机构id'" json:"departmentId"`
	Result          string         `gorm:"type:varchar(128);comment:'审批结果'" json:"result"`
	ApproveInfo     string         `gorm:"type:varchar(128);comment:'审批意见'" json:"approveInfo"`
}

func (r *NewWorkOrderLog) TableName() string {
	return "new_work_order_log"
}

type WorkOrderPolicyConfig struct {
	OrderRange     string `json:"orderRange"`
	ApproveWay     string `json:"approveWay"`
	ApproveLevel   int    `json:"approveLevel"`
	IsFinalApprove bool   `json:"isFinalApprove"`
}
