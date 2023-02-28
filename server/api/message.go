package api

import (
	"strings"
	"tkbastion/pkg/log"
	"tkbastion/server/dto"
	"tkbastion/server/model"

	"github.com/labstack/echo/v4"
)

// MassageUnreadEndpoint 当前用户的未读信息
func MassageUnreadEndpoint(c echo.Context) error {
	user, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 500, "用户登录信息已失效，请重新登录")
	}
	messageInfo, err := messageRepository.FindByRecId(user.ID)
	if err != nil {
		log.Errorf("DB Error: %v", err)
	}
	var message []model.Message
	for i := range messageInfo {
		if messageInfo[i].Status == false {
			message = append(message, messageInfo[i])
		}
	}
	return Success(c, message)
}

// 获取当前用户待审批的工单
func MessagePendingApprovalEndPoint(c echo.Context) error {
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 500, "当前用户信息已过期,请重新登录")
	}
	// 获取当前部门以下所有部门id
	var depIds []int64
	err := GetChildDepIds(account.DepartmentId, &depIds)
	if err != nil {
		return err
	}
	// 通过部门id查询工单列表
	// 查看自己所有未审批的工单
	unApproval, err := workOrderNewRepository.GetUnApprovalWorkOrder(account.RoleName, depIds)
	if err != nil {
		return err
	}
	var workOrderList []dto.WorkOrderPendingApproval
	for _, v := range unApproval {
		workOrderApprovalLog, err := workOrderApprovalLogRepository.FindByWorkOrderId(v.ID)
		if err != nil {
			log.Errorf("WorkOrderApprovalLog FindByWorkOrderId err: %v", err)
			continue
		}
		// 判断当前用户是否能看见
		for _, v1 := range workOrderApprovalLog {
			if account.DepartmentId == v1.DepartmentId {
				workOrderList = append(workOrderList, dto.WorkOrderPendingApproval{
					OrderType:   v.WorkOrderType,
					WorkOrderId: v.OrderId,
					Applicant:   v.ApplyUser,
					Department:  v.Department,
					ApplyTime:   v.ApplyTime.Format("2006-01-02 15:04:05"),
				})
			}
		}
	}
	return Success(c, workOrderList)
}

// MessageCountEndpoint 返回未读消息和待审批的消息总数
func MessageCountEndpoint(c echo.Context) error {
	user, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 500, "用户登录信息已失效，请重新登录")
	}
	messageInfo, err := messageRepository.FindByRecId(user.ID)
	if err != nil {
		log.Errorf("DB Error: %v", err)
	}
	var count = 0
	for i := range messageInfo {
		if messageInfo[i].Status == false {
			count++
		}
	}
	var depIds []int64
	err = GetChildDepIds(user.DepartmentId, &depIds)
	if err != nil {
		return err
	}
	// 通过部门id查询工单列表
	// 查看自己所有未审批的工单
	unApproval, err := workOrderNewRepository.GetUnApprovalWorkOrder(user.RoleName, depIds)
	if err != nil {
		return err
	}
	for _, v := range unApproval {
		workOrderApprovalLog, err := workOrderApprovalLogRepository.FindByWorkOrderId(v.ID)
		if err != nil {
			log.Errorf("WorkOrderApprovalLog FindByWorkOrderId err: %v", err)
			continue
		}
		// 判断当前用户是否能看见
		for _, v1 := range workOrderApprovalLog {
			if user.DepartmentId == v1.DepartmentId {
				count++
			}
		}
	}
	return Success(c, count)
}

// MessagePagingEndpoint 查询当前用户的所有信息
func MessagePagingEndpoint(c echo.Context) error {
	auto := c.QueryParam("auto")
	theme := c.QueryParam("theme")
	level := c.QueryParam("level")
	user, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 500, "用户登录信息已失效，请重新登录")
	}
	messageInfo, err := messageRepository.Find(user.ID, auto, theme, level)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "读取失败", "", nil)
	}
	return Success(c, messageInfo)
}

// MessageDeleteEndpoint 批量删除或单独删除
func MessageDeleteEndpoint(c echo.Context) error {
	id := c.Param("id")
	split := strings.Split(id, ",")
	var theme string
	for i := range split {
		messageInfo, err := messageRepository.FindById(split[i])
		if err != nil {
			log.Errorf("DB Error: %v", err)
			continue
		}
		if err := messageRepository.DeleteById(split[i]); err != nil {
			log.Errorf("DB Error: %v", err)
		}
		theme += messageInfo.Theme + ","
	}
	if len(theme) > 1 {
		theme = theme[:len(theme)-1]
	}
	return SuccessWithOperate(c, "消息中心-删除: ["+theme+"]", nil)
}

// MessageClearEndpoint 全部删除
func MessageClearEndpoint(c echo.Context) error {
	user, _ := GetCurrentAccountNew(c)
	messageInfo, err := messageRepository.FindByRecId(user.ID)
	if err != nil {
		log.Errorf("DB FindByRecId Error: %v", err)
		return FailWithDataOperate(c, 500, "读取失败", "", nil)
	}
	for i := range messageInfo {
		if err := messageRepository.DeleteById(messageInfo[i].ID); err != nil {
			log.Errorf("DB DeleteByDepartmentId Error: %v", err)
			return FailWithDataOperate(c, 500, "删除失败", "", nil)
		}
	}
	return SuccessWithOperate(c, "消息中心-清空: [删除所有消息]", nil)
}

// MessageMarkEndpoint 批量选择消息标记已读
func MessageMarkEndpoint(c echo.Context) error {
	id := c.Param("id")
	split := strings.Split(id, ",")
	theme := ""
	for i := range split {
		message, err := messageRepository.FindById(split[i])
		if err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "读取失败", "", nil)
		}
		message.Status = true
		if err := messageRepository.UpdateById(&message, split[i]); err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "更新失败", "", nil)
		}
		theme += message.Theme + ","
	}
	if len(theme) > 1 {
		theme = theme[:len(theme)-1]
	}
	return SuccessWithOperate(c, "消息中心-已读: [标记"+theme+"]", nil)
}

// MessageAllMarkEndpoint 全部消息标记已读
func MessageAllMarkEndpoint(c echo.Context) error {
	user, _ := GetCurrentAccountNew(c)
	items, err := messageRepository.FindByRecId(user.ID)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "读取失败", "", nil)
	}
	for i := range items {
		if items[i].Status == false {
			items[i].Status = true
			if err := messageRepository.UpdateById(&items[i], items[i].ID); err != nil {
				log.Errorf("DB Error: %v", err)
				return FailWithDataOperate(c, 500, "更新失败", "", nil)
			}
		}
	}
	return SuccessWithOperate(c, "消息中心-全部已读: [标记全部]", nil)
}
