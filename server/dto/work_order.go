package dto

import "tkbastion/server/utils"

type WorkOrderForCreate struct {
	Title       string         `json:"title" `
	IsPermanent bool           `json:"isPermanent" `
	BeginTime   utils.JsonTime `json:"beginTime" `
	EndTime     utils.JsonTime `json:"endTime" `
	// 是否上传、下载、水印
	IsUpload    bool   `json:"isUpload" `
	IsDownload  bool   `json:"isDownload" `
	IsWatermark bool   `json:"isWatermark" `
	Description string `json:"description" `
	// 关联设备id
	AssetId string `json:"assetId" `
}

type WorkOrderForUpdate struct {
	Title       string         `json:"title" `
	BeginTime   utils.JsonTime `json:"beginTime" `
	EndTime     utils.JsonTime `json:"endTime" `
	IsPermanent bool           `json:"isPermanent" `
	// 是否上传、下载、水印
	IsUpload    bool   `json:"isUpload" `
	IsDownload  bool   `json:"isDownload" `
	IsWatermark bool   `json:"isWatermark" `
	Description string `json:"description" `
}

type WorkOrderForDetail struct {
	ID            string `json:"id" `
	OrderId       string `json:"orderId" `
	Title         string `json:"title" `
	WorkOrderType string `json:"workOrderType" `
	Status        string `json:"status" `
	Description   string `json:"description" `
	ApplyUser     string `json:"applyUser" `
	ApplyUserName string `json:"applyUserName" `
	ApplyId       string `json:"applyId" `
	ValidTime     string `json:"validTime" `
	ApplyTime     string `json:"applyTime" `
}

type WorkOrderForAsset struct {
	ID         string `json:"id" `
	Name       string `json:"name" `
	Ip         string `json:"ip" `
	Protocol   string `json:"protocol" `
	Passport   string `json:"passport" `
	Department string `json:"department" `
	Port       int    `json:"port" `
}

type WorkOrderLogForExport struct {
	Title           string `json:"title" `
	ApplyTime       string `json:"applyTime" `
	ApproveTime     string `json:"approveTime" `
	ApproveUsername string `json:"approveUsername" `
	ApproveNickname string `json:"approveNickname" `
	Department      string `json:"department" `
	Result          string `json:"result" `
	Info            string `json:"info" `
}

type WorkOrderApprovalLogForPaging struct {
	ID               string `json:"id" `
	WorkOrderId      string `json:"workOrderId" `
	Number           int    `json:"number" `
	ApprovalUsername string `json:"approvalUsername" `
	ApprovalNickname string `json:"approvalNickname" `
	ApprovalId       string `json:"approvalId" `
	Department       string `json:"department" `
	DepartmentId     int64  `json:"departmentId" `
	ApprovalDate     string `json:"approvalDate" `
	Result           string `json:"result" `
	ApproveWay       string `json:"approveWay" `
	IsFinalApprove   bool   `json:"isFinalApprove" `
	ApprovalInfo     string `json:"approvalInfo" `
}
