package api

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
	"tkbastion/pkg/constant"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/global"
	"tkbastion/pkg/global/work_order"
	"tkbastion/pkg/log"
	"tkbastion/pkg/validator"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// ---------------------------------配置部分---------------------------------

// WorkOrderSettingPagingEndPoint 查看工单配置
func WorkOrderSettingPagingEndPoint(c echo.Context) (err error) {
	var workOrderPolicyConfig model.WorkOrderPolicyConfig
	orderSettingMap := make(map[string]string, 4)
	orderSettingMap, err = propertyRepository.FindMapByNames([]string{"order-range", "approve-way", "approve-level", "is-final-approve"})
	if err != nil {
		log.Errorf("WorkOrderSetting FindById err: %v", err)
		return FailWithDataOperate(c, 500, "查找失败", "", nil)
	}
	workOrderPolicyConfig.OrderRange = orderSettingMap["order-range"]
	workOrderPolicyConfig.ApproveWay = orderSettingMap["approve-way"]
	workOrderPolicyConfig.ApproveLevel, err = strconv.Atoi(orderSettingMap["approve-level"])
	if err != nil {
		log.Errorf("WorkOrderSetting FindById err: %v", err)
		workOrderPolicyConfig.ApproveLevel = 1
	}
	workOrderPolicyConfig.IsFinalApprove, err = strconv.ParseBool(orderSettingMap["is-final-approve"])
	if err != nil {
		log.Errorf("WorkOrderSetting FindById err: %v", err)
		workOrderPolicyConfig.IsFinalApprove = false
	}
	return SuccessWithOperate(c, "", workOrderPolicyConfig)
}

// WorkOrderSettingUpdateEndPoint 更新工单配置
func WorkOrderSettingUpdateEndPoint(c echo.Context) error {
	var workOrderPolicyConfig model.WorkOrderPolicyConfig
	if err := c.Bind(&workOrderPolicyConfig); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if err := propertyRepository.UpdateByName(&model.Property{
		Value: workOrderPolicyConfig.OrderRange,
	}, "order-range"); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if err := propertyRepository.UpdateByName(&model.Property{
		Value: workOrderPolicyConfig.ApproveWay,
	}, "approve-way"); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if err := propertyRepository.UpdateByName(&model.Property{
		Value: strconv.Itoa(workOrderPolicyConfig.ApproveLevel),
	}, "approve-level"); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if err := propertyRepository.UpdateByName(&model.Property{
		Value: strconv.FormatBool(workOrderPolicyConfig.IsFinalApprove),
	}, "is-final-approve"); err != nil {
		log.Errorf("DB Error: %v", err)
	}
	return SuccessWithOperate(c, "访问工单策略-修改: [系统配置-策略配置,修改系统访问工单策略]", workOrderPolicyConfig)
}

// ---------------------------------申请部分---------------------------------

// NewWorkOrderCreateEndPoint 新建工单申请
func NewWorkOrderCreateEndPoint(c echo.Context) (err error) {
	var item dto.WorkOrderForCreate
	if err := c.Bind(&item); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "新建失败", "", err)
	}
	if item.Title == "" {
		return FailWithDataOperate(c, 500, "工单标题不能为空", "", err)
	}

	if err := c.Validate(&item); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}
	// 获取当前的时间字符串与当前微秒时间戳拼接
	// 生成一个四位数的随机数
	num := func(min, max int) int {
		if min >= max || min == 0 || max == 0 {
			return max
		}
		return rand.Intn(max-min+1) + min
	}(1000, 9999)

	orderId := utils.NowJsonTime().Format("20060102150405") + strconv.FormatInt(int64(num), 10)
	// 获取当前用户
	account, _ := GetCurrentAccountNew(c)
	workOrder := model.NewWorkOrder{
		ID:            utils.UUID(),
		OrderId:       orderId,
		Title:         item.Title,
		Department:    account.DepartmentName,
		DepartmentId:  account.DepartmentId,
		Role:          account.RoleName,
		RoleId:        account.RoleId,
		WorkOrderType: constant.AccessWorkOrder,
		Status:        constant.NotSubmitted,
		Description:   item.Description,
		ApplyUser:     account.Username,
		ApplyId:       account.ID,
		ApplyTime:     utils.NowJsonTime(),
		IsPermanent:   &item.IsPermanent,
		BeginTime:     item.BeginTime,
		EndTime:       item.EndTime,
		IsUpload:      &item.IsUpload,
		IsDownload:    &item.IsDownload,
		IsWatermark:   &item.IsWatermark,
	}
	// 检查工单是否关联设备
	if len(item.AssetId) == 0 {
		return FailWithDataOperate(c, 500, "请关联至少一个设备", "", err)
	}
	// 创建工单
	if err := workOrderNewRepository.Create(&workOrder); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新建失败", "", err)
	}

	// 处理关联的设备
	if len(item.AssetId) > 0 {
		splits := strings.Split(item.AssetId, ",")
		for i := range splits {
			workOrderAsset := model.WorkOrderAsset{
				ID:          utils.UUID(),
				WorkOrderId: workOrder.ID,
				AssetId:     splits[i],
			}
			if err := workOrderAssetRepository.Create(&workOrderAsset); err != nil {
				log.Errorf("DB Error: %v", err)
				return FailWithDataOperate(c, 500, "新建失败", "", err)
			}
		}
	}
	// 创建资产审批的日志
	// 获取当前审批配置的审批配置
	orderSettingMap := make(map[string]string, 4)
	orderSettingMap, err = propertyRepository.FindMapByNames([]string{"order-range", "approve-way", "approve-level", "is-final-approve"})
	if err != nil {
		log.Errorf("WorkOrderSetting FindById err: %v", err)
		return FailWithDataOperate(c, 500, "获取审批配置失败", "", nil)
	}

	isFinalApprove, _ := strconv.ParseBool(orderSettingMap["is-final-approve"])
	level, err := strconv.ParseInt(orderSettingMap["approve-level"], 10, 64)
	if err != nil {
		log.Errorf("WorkOrderSetting FindById err: %v", err)
	}

	// 获取当前用户的部门深度（由0开始）所以获取部门深度与级数比较时应+1
	// 将当前用户的部门深度与审批配置中的审批级数（由1开始）比较
	// 审批级数大于当前用户部门深度时，审批级数为当前用户部门深度
	depth, _ := DepLevel(account.DepartmentId)
	if depth+1 <= int(level) {
		level = int64(depth + 1)
	}
	departmentId := account.DepartmentId
	for i := 1; i <= int(level); i++ {
		// 获取当前部门详细信息
		department, _ := departmentRepository.FindById(departmentId)
		// 获取有效部门部门管理员信息
		departmentAdmin, err := userNewRepository.FindDepartmentAdminByDepartmentId(department.ID)
		if err != nil {
			log.Errorf("WorkOrderSetting FindById err: %v", err)
			return FailWithDataOperate(c, 500, "获取部门管理员失败", "", nil)
		}
		// 拼接部门管理员所有名称信息
		var username, nickname string
		var ApprovalLog model.WorkOrderApprovalLog
		if len(departmentAdmin) > 0 {
			for _, v := range departmentAdmin {
				username += v.Username + "\n"
				nickname += v.Nickname + "\n"
			}
			ApprovalLog = model.WorkOrderApprovalLog{
				ID:               utils.UUID(),
				WorkOrderId:      workOrder.ID,
				Number:           i,
				ApprovalUsername: username,
				ApprovalNickname: nickname,
				Department:       department.Name,
				DepartmentId:     department.ID,
				Result:           constant.NotApproved,
				ApproveWay:       orderSettingMap["approve-way"],
				IsFinalApprove:   isFinalApprove,
				ApprovalInfo:     "",
			}
		} else { /* 此处说明该部门没有部门管理员存在 */
			//var result string
			//if orderSettingMap["approve-way"] == constant.SignModeMany { /* 会签模式忽略该层意见直接同意 */ //  TODO: 无部门管理员-应该需要特殊处理
			//	result = constant.Approved
			//} else {
			//	result = constant.NotApproved /* 多人模式下忽略该层意见正常待审批 */
			//}
			ApprovalLog = model.WorkOrderApprovalLog{
				ID:               utils.UUID(),
				WorkOrderId:      workOrder.ID,
				ApprovalUsername: "无",
				ApprovalNickname: "无",
				Number:           i,
				Department:       department.Name,
				DepartmentId:     department.ID,
				Result:           constant.NotApproved,
				ApproveWay:       orderSettingMap["approve-way"],
				IsFinalApprove:   isFinalApprove,
				ApprovalInfo:     "", //会签模式下暂时忽略该层意见,多人模式下忽略该层意见正常待审批
			}
		}
		if err := workOrderApprovalLogRepository.Create(&ApprovalLog); err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "新建审批记录表失败", "", err)
		}
		// 部门id去到上级部门
		if department.FatherId != -1 {
			departmentId = department.FatherId
		} else {
			break
		}
	}
	// 如果开启终审，添加终审记录
	// 查询系统管理员信息
	systemUser, err := userNewRepository.FindByName("admin")
	if err != nil {
		log.Errorf("WorkOrderSetting FindById err: %v", err)
	}
	if isFinalApprove {
		ApprovalLog := model.WorkOrderApprovalLog{
			ID:               utils.UUID(),
			WorkOrderId:      workOrder.ID,
			ApprovalUsername: systemUser.Username,
			ApprovalNickname: systemUser.Nickname,
			Number:           int(level + 1),
			Department:       systemUser.DepartmentName,
			DepartmentId:     systemUser.DepartmentId,
			Result:           constant.NotApproved,
			ApproveWay:       orderSettingMap["approve-way"],
			IsFinalApprove:   isFinalApprove,
			ApprovalInfo:     "",
		}
		if err := workOrderApprovalLogRepository.Create(&ApprovalLog); err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "新建审批记录表失败", "", err)
		}
	}
	return SuccessWithOperate(c, "工单申请-新建: 新建工单名称["+item.Title+"]", nil)
}

// NewWorkOrderUpdateEndPoint 修改工单申请
func NewWorkOrderUpdateEndPoint(c echo.Context) error {
	id := c.Param("id")
	oldWorkOrder, err := workOrderNewRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "获取工单该失败", "", err)
	}
	var item dto.WorkOrderForUpdate
	if err := c.Bind(&item); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "绑定数据失败", "", err)
	}

	if err := c.Validate(&item); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	if item.Description == "" {
		item.Description = " "
	}
	if err := workOrderNewRepository.UpdateById(id, &model.NewWorkOrder{
		Title:       item.Title,
		BeginTime:   item.BeginTime,
		EndTime:     item.EndTime,
		IsPermanent: &item.IsPermanent,
		IsWatermark: &item.IsWatermark,
		IsDownload:  &item.IsDownload,
		IsUpload:    &item.IsUpload,
		Description: item.Description,
	}); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改工单申请失败", "", err)
	}
	return SuccessWithOperate(c, "工单申请-修改: 工单["+oldWorkOrder.Title+"]", item)
}

// NewWorkOrderGetRelateDeviceEndPoint 获取工单关联时可关联的设备
func NewWorkOrderGetRelateDeviceEndPoint(c echo.Context) (err error) {
	// 获取工单审批的配置
	orderSettingMap := make(map[string]string)
	orderSettingMap, err = propertyRepository.FindMapByNames([]string{"order-range"})
	if err != nil {
		log.Errorf("WorkOrderSetting FindById err: %v", err)
		return FailWithDataOperate(c, 500, "获取审批配置失败", "", nil)
	}
	var deviceList []model.PassPort
	// 获取当前用户信息
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, -1, "当前用户信息已过期，请重新登录")
	}
	if orderSettingMap["order-range"] == "department" {
		// 此处获取本部门及以下部门的设备
		// 获取当前账户下的所有主机
		var depIds []int64
		err := GetChildDepIds(account.DepartmentId, &depIds)
		if nil != err {
			log.Errorf("GetChildDepIds Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		// 策略所属部门及下级部门包含的所有设备账号(包括已被选择设备账号)
		passportArr, err := newAssetRepository.GetPassportByDepartmentIds(context.TODO(), depIds)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		for i := range passportArr {
			depChinaName, err := DepChainName(passportArr[i].DepartmentId)
			if nil != err {
				log.Errorf("DepChainName Error: %v", err)
				return FailWithDataOperate(c, 500, "查询失败", "", nil)
			}
			passportArr[i].Name = passportArr[i].AssetName + "[" + passportArr[i].Ip + "]" + "[" + passportArr[i].Passport + "]" + "[" + passportArr[i].Protocol + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
		}
		deviceList = passportArr
	} else {
		// 此处获取所有的设备
		passportArr, err := newAssetRepository.GetAllPassport(context.TODO())
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		for i := range passportArr {
			depChinaName, err := DepChainName(passportArr[i].DepartmentId)
			if nil != err {
				log.Errorf("DepChainName Error: %v", err)
				return FailWithDataOperate(c, 500, "查询失败", "", nil)
			}
			passportArr[i].Name = passportArr[i].AssetName + "[" + passportArr[i].Ip + "]" + "[" + passportArr[i].Passport + "]" + "[" + passportArr[i].Protocol + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
		}
		deviceList = passportArr
	}
	passportList, err := GetUserPassportByUserId(c.RealIP(), account.ID)
	if err != nil {
		log.Errorf("GetUserPassportByUserId Error: %v", err)
	}
	deviceMap := make(map[string]bool, len(passportList))
	for _, v := range passportList {
		deviceMap[v.Id] = true
	}
	deviceArr := make([]model.PassPort, 0)
	for _, v := range deviceList {
		if _, ok := deviceMap[v.ID]; !ok {
			deviceArr = append(deviceArr, v)
		}
	}
	return Success(c, deviceArr)
}

// NewWorkOrderHadRelateDeviceEndPoint 获取工单关联时已关联的设备
func NewWorkOrderHadRelateDeviceEndPoint(c echo.Context) error {
	id := c.Param("id")
	// 通过工单id获取工单关联的设备
	workOrderAsset, err := workOrderAssetRepository.FindByWorkOrderId(id)
	if err != nil {
		log.Errorf("WorkOrderSetting FindById err: %v", err)
		return FailWithDataOperate(c, 500, "获取工单关联的设备失败", "", nil)
	}
	var deviceList []model.PassPort
	for _, v := range workOrderAsset {
		passportArr, err := newAssetRepository.GetPassportById(context.TODO(), v.AssetId)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			// 如果错误是not found，那么就不返回错误，并删除这个id的关联
			if err == gorm.ErrRecordNotFound {
				// 删除关联
				_ = workOrderAssetRepository.DeleteByAssetId(v.AssetId)
				continue
			}
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		depChinaName, err := DepChainName(passportArr.DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		passportArr.Name = passportArr.AssetName + "[" + passportArr.Ip + "]" + "[" + passportArr.Name + "]" + "[" + passportArr.Protocol + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
		deviceList = append(deviceList, passportArr)
	}
	return Success(c, deviceList)
}

// NewWorkOrderRelateDeviceEndPoint 工单关联设备
func NewWorkOrderRelateDeviceEndPoint(c echo.Context) error {
	workOrderId := c.QueryParam("id")
	assetId := c.QueryParam("assetId")
	// 检测该工单id是否存在
	workOrder, err := workOrderNewRepository.FindById(workOrderId)
	if err != nil {
		log.Errorf("WorkOrder FindById err: %v", err)
		return FailWithDataOperate(c, 500, "工单不存在", "", nil)
	}
	// 通过工单id删除与该工单关联的设备
	err = workOrderAssetRepository.DeleteByWorkOrderId(workOrderId)
	if err != nil {
		log.Errorf("WorkOrderAsset DeleteByWorkOrderId err: %v", err)
		return FailWithDataOperate(c, 500, "删除工单关联的设备失败", "", nil)
	}
	if assetId != "" {
		assetIdArr := strings.Split(assetId, ",")
		for _, v := range assetIdArr {
			workOrderAsset := model.WorkOrderAsset{
				ID:          utils.UUID(),
				WorkOrderId: workOrderId,
				AssetId:     v,
			}
			// 保存工单与设备关联
			err = workOrderAssetRepository.Create(&workOrderAsset)
			if err != nil {
				log.Errorf("WorkOrderAsset Save err: %v", err)
				return FailWithDataOperate(c, 500, "工单关联设备失败", "", nil)
			}
		}
	}
	return SuccessWithOperate(c, "工单申请-关联设备: 工单["+workOrder.Title+"]", nil)
}

// NewWorkOrderSubmitEndPoint 提交工单
func NewWorkOrderSubmitEndPoint(c echo.Context) error {
	id := c.Param("id")
	// 通过工单id获取工单
	workOrder, err := workOrderNewRepository.FindById(id)
	if err != nil {
		log.Errorf("WorkOrder FindById err: %v", err)
		return FailWithDataOperate(c, 500, "工单不存在", "", nil)
	}
	// 检测工单是否可提交
	if total, _ := workOrderNewRepository.CountByWorkOrderId(id); total == 0 {
		return FailWithDataOperate(c, 500, "工单未关联设备", "", nil)
	}
	if workOrder.Status != constant.NotSubmitted {
		return FailWithDataOperate(c, 500, "工单不可提交", "", nil)
	} else {
		// 提交工单
		workOrder.Status = constant.Submitted
		workOrder.ApplyTime = utils.NowJsonTime()
		err = workOrderNewRepository.UpdateById(id, &workOrder)
		if err != nil {
			log.Errorf("WorkOrder Update err: %v", err)
			return FailWithDataOperate(c, 500, "提交工单失败", "", nil)
		}
	}
	return SuccessWithOperate(c, "工单申请-提交: 工单["+workOrder.Title+": 已提交]", nil)
}

// NewWorkOrderCancelEndPoint 撤销工单
func NewWorkOrderCancelEndPoint(c echo.Context) error {
	id := c.Param("id")
	// 通过工单id获取工单
	workOrder, err := workOrderNewRepository.FindById(id)
	if err != nil {
		log.Errorf("WorkOrder FindById err: %v", err)
		return FailWithDataOperate(c, 500, "工单不存在", "", nil)
	}
	// 检测工单是否可取消
	if workOrder.Status != constant.Submitted {
		return FailWithDataOperate(c, 500, "工单不可取消", "", nil)
	} else {
		// 取消工单
		workOrder.Status = constant.NotSubmitted
		err = workOrderNewRepository.UpdateById(id, &workOrder)
		if err != nil {
			log.Errorf("WorkOrder Update err: %v", err)
			return FailWithDataOperate(c, 500, "取消工单失败", "", nil)
		}
	}
	return SuccessWithOperate(c, "工单申请-取消: 工单["+workOrder.Title+": 已取消]", nil)
}

// WorkOrderApplyListEndPoint 查看工单申请列表
func WorkOrderApplyListEndPoint(c echo.Context) error {
	auto := c.QueryParam("auto")
	OrderId := c.QueryParam("orderId")
	applyTime := c.QueryParam("applyTime")
	title := c.QueryParam("title")
	orderType := c.QueryParam("orderType")
	status := c.QueryParam("status")
	// 获取当前用户
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, -1, "当前用户信息已过期，请重新登录")
	}
	// 通过申请人id查询工单列表
	workOrderList, err := workOrderNewRepository.FindByApplyIdAndConditions(account.ID, auto, OrderId, applyTime, title, orderType, status)
	if err != nil {
		log.Errorf("WorkOrder FindByApplyId err: %v", err)
		return FailWithDataOperate(c, 500, "查询工单列表失败", "", nil)
	}
	for _, v := range workOrderList {
		f := false
		if v.IsPermanent == &f && time.Now().After(v.EndTime.Time) {
			v.Status = constant.Expiration
			if err := workOrderNewRepository.UpdateById(v.ID, &v); err != nil {
				log.Errorf("WorkOrder Update err: %v", err)
			}
		}
	}
	return SuccessWithOperate(c, "", workOrderList)
}

// NewWorkOrderAssetInfoEndPoint 查看工单的的资产信息
func NewWorkOrderAssetInfoEndPoint(c echo.Context) error {
	workOrderId := c.Param("id")
	// 通过工单id获取工单
	assetList, err := FindPassportListByWorkOrderId(workOrderId)
	if err != nil {
		log.Errorf("WorkOrder FindById err: %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	var workOrderAsset []dto.WorkOrderForAsset
	for _, v := range assetList {
		// 通过部门id获取部门名称
		department, err := departmentRepository.FindById(v.DepartmentId)
		if err != nil {
			log.Errorf("Department FindById err: %v", err)
		}
		workOrderAsset = append(workOrderAsset, dto.WorkOrderForAsset{
			ID:         v.ID,
			Name:       v.AssetName,
			Ip:         v.Ip,
			Protocol:   v.Protocol,
			Passport:   v.Passport,
			Department: department.Name,
			Port:       v.Port,
		})
	}
	return Success(c, workOrderAsset)
}

// NewWorkOrderApproveInfoEndPoint 查看工单的审批信息
func NewWorkOrderApproveInfoEndPoint(c echo.Context) error {
	workOrderId := c.Param("id")
	// 通过工单id获取工单
	workOrder, err := workOrderNewRepository.FindById(workOrderId)
	if err != nil {
		log.Errorf("WorkOrder FindById err: %v", err)
		return FailWithDataOperate(c, 500, "工单不存在", "", nil)
	}
	// 通过工单id获取工单关联的审批信息
	workOrderApproveList, err := workOrderApprovalLogRepository.FindByWorkOrderId(workOrder.ID)
	if err != nil {
		log.Errorf("WorkOrderApprovalLog FindByWorkOrderId err: %v", err)
		return FailWithDataOperate(c, 500, "获取工单关联的审批信息失败", "", nil)
	}
	var workOrderApprovePaging = make([]dto.WorkOrderApprovalLogForPaging, len(workOrderApproveList))
	for i, v := range workOrderApproveList {
		workOrderApprovePaging[i].ID = v.ID
		workOrderApprovePaging[i].WorkOrderId = v.WorkOrderId
		workOrderApprovePaging[i].Number = v.Number
		workOrderApprovePaging[i].ApprovalUsername = v.ApprovalUsername
		workOrderApprovePaging[i].ApprovalNickname = v.ApprovalNickname
		workOrderApprovePaging[i].Department = v.Department
		workOrderApprovePaging[i].DepartmentId = v.DepartmentId
		workOrderApprovePaging[i].ApprovalId = v.ApprovalId
		workOrderApprovePaging[i].Result = v.Result
		workOrderApprovePaging[i].ApproveWay = v.ApproveWay
		workOrderApprovePaging[i].IsFinalApprove = v.IsFinalApprove
		workOrderApprovePaging[i].ApprovalInfo = v.ApprovalInfo
		if v.ApprovalDate.IsZero() {
			workOrderApprovePaging[i].ApprovalDate = ""
		} else {
			workOrderApprovePaging[i].ApprovalDate = v.ApprovalDate.Time.Format("2006-01-02 15:04:05")
		}
	}
	return Success(c, workOrderApprovePaging)
}

// NewWorkOrderDetailEndPoint 查看工单的详情
func NewWorkOrderDetailEndPoint(c echo.Context) error {
	id := c.Param("id")
	// 通过工单id获取工单
	workOrder, err := workOrderNewRepository.FindById(id)
	if err != nil {
		log.Errorf("WorkOrder FindById err: %v", err)
		return FailWithDataOperate(c, 500, "工单不存在", "", nil)
	}
	// 查询申请人
	applyAccount, err := userNewRepository.FindById(workOrder.ApplyId)
	if err != nil {
		log.Errorf("User FindById err: %v", err)
		return FailWithDataOperate(c, 500, "查询申请人失败", "", nil)
	}
	var validTime string
	if workOrder.IsPermanent != nil {
		if *workOrder.IsPermanent {
			validTime = "永久有效"
		} else {
			if !workOrder.BeginTime.IsZero() {
				validTime += workOrder.BeginTime.Format("2006-01-02 15:04:05")
			}
			validTime += "~"
			if !workOrder.EndTime.IsZero() {
				validTime += workOrder.EndTime.Format("2006-01-02 15:04:05")
			}
		}
	} else {
		if !workOrder.BeginTime.IsZero() {
			validTime += workOrder.BeginTime.Format("2006-01-02 15:04:05") + "~"
		}
		validTime += "~"
		if !workOrder.EndTime.IsZero() {
			validTime += workOrder.EndTime.Format("2006-01-02 15:04:05")
		}
	}
	workOrderDetail := dto.WorkOrderForDetail{
		ID:            workOrder.ID,
		OrderId:       workOrder.OrderId,
		Title:         workOrder.Title,
		WorkOrderType: workOrder.WorkOrderType,
		Status:        workOrder.Status,
		Description:   workOrder.Description,
		ApplyUser:     workOrder.ApplyUser,
		ValidTime:     validTime,
		ApplyUserName: applyAccount.Nickname,
		ApplyId:       workOrder.ApplyId,
		ApplyTime:     workOrder.ApplyTime.Format("2006-01-02 15:04:05"),
	}
	return Success(c, workOrderDetail)
}

// ---------------------------------审批部分---------------------------------

// WorkOrderApproveListEndPoint 查看工单审批列表
func WorkOrderApproveListEndPoint(c echo.Context) error {
	auto := c.QueryParam("auto")
	OrderId := c.QueryParam("orderId")
	applyTime := c.QueryParam("applyTime")
	title := c.QueryParam("title")
	workOrderType := c.QueryParam("workOrderType")
	status := c.QueryParam("status")
	// 获取当前用户
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, -1, "当前用户信息已过期，请重新登录")
	}
	// 获取当前部门以下所有部门id
	var depIds []int64
	err := GetChildDepIds(account.DepartmentId, &depIds)
	if err != nil {
		return err
	}
	// 通过部门id查询工单列表
	workOrderList, err := workOrderNewRepository.FindByApproveDepIds(depIds, auto, OrderId, applyTime, title, workOrderType, status)
	if err != nil {
		log.Errorf("WorkOrder FindByApproveDepIds err: %v", err)
		return FailWithDataOperate(c, 500, "查询工单列表失败", "", nil)
	}
	var workOrderPaging = make([]model.NewWorkOrder, 0, len(workOrderList))
	for i, v := range workOrderList {
		// 根据工单id查询审批记录表
		workOrderApprovalLog, err := workOrderApprovalLogRepository.FindByWorkOrderId(v.ID)
		if err != nil {
			log.Errorf("WorkOrderApprovalLog FindByWorkOrderId err: %v", err)
			continue
		}
		// 判断当前用户是否能看见
		for _, v1 := range workOrderApprovalLog {
			if account.DepartmentId == v1.DepartmentId {
				workOrderPaging = append(workOrderPaging, workOrderList[i])
				break
			}
		}
	}
	// 通过部门id查询工单列表
	return Success(c, workOrderPaging)
}

// WorkOrderApproveEndPoint 审批工单/通过或驳回·通过传agree 驳回传disagree
func WorkOrderApproveEndPoint(c echo.Context) error {
	id := c.QueryParam("id")
	approvalInfo := c.QueryParam("approvalInfo")
	result := c.QueryParam("result")
	// 查询该工单是否存在
	workOrder, err := workOrderNewRepository.FindById(id)
	if err != nil {
		log.Errorf("WorkOrder FindById err: %v", err)
		return FailWithDataOperate(c, 500, "工单不存在", "", nil)
	}
	// 查询该工单的审批记录表
	workOrderApprovalLog, err := workOrderApprovalLogRepository.FindByWorkOrderId(id)
	if err != nil {
		log.Errorf("WorkOrderApprovalLog FindByWorkOrderId err: %v", err)
		return FailWithDataOperate(c, 500, "查询工单审批记录失败", "", nil)
	}
	if len(workOrderApprovalLog) == 0 {
		return Fail(c, 500, "该工单未找到审批记录")
	}
	var status string
	if result == "agree" {
		result = constant.Agree
		status = constant.Approved
	} else {
		result = constant.Disagree
		status = constant.Rejected
	}
	// 获取当前用户
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, -1, "当前用户信息已过期，请重新登录")
	}
	if account.RoleName != constant.SystemAdmin && account.RoleName != constant.DepartmentAdmin { // 只有系统管理员和部门管理员才能审批工单
		return Fail(c, 500, "您没有权限审批该工单")
	}
	// 检查该工单的状态 已通过，已驳回，已关闭的工单不能再审批
	if workOrder.Status == constant.Approved || workOrder.Status == constant.Rejected || workOrder.Status == constant.Closed {
		return Fail(c, 500, "该工单流程已结束")
	}

	// -----------------指令工单审批-----------------

	if workOrder.WorkOrderType == "指令工单" {
		fmt.Println("指令工单审批")
		order, _ := work_order.GetOrder(workOrder.OrderId)
		if nil == order {
			log.Error("workOrderApproval order is nil")
			// 未找到此工单，更新该工单的状态
			if err := workOrderNewRepository.UpdateById(workOrder.ID, &model.NewWorkOrder{
				Status: constant.Closed,
			}); err != nil {
				log.Errorf("WorkOrder UpdateById err: %v", err)
				return FailWithDataOperate(c, 500, "更新工单状态失败", "", nil)
			}
			return FailWithDataOperate(c, 500, "此工单已关闭", "", nil)
		}
		//向审批人发送审批结果
		fmt.Println("-----------------指令工单审批-----------------")
		fmt.Println("status", status)
		order.Status <- status
		order.Approved = account.Nickname
		defer close(order.Status)
		if err := workOrderApprovalLogRepository.UpdateByWorkOrderIdAndDepartmentId(id, 1, account.DepartmentId, &model.WorkOrderApprovalLog{
			ApprovalUsername: account.Username,
			ApprovalNickname: account.Nickname,
			ApprovalId:       account.ID,
			ApprovalDate:     utils.NowJsonTime(),
			Result:           result,
			ApprovalInfo:     approvalInfo,
		}); err != nil {
			log.Errorf("WorkOrderApprovalLog UpdateByWorkOrderId err: %v", err)
			return FailWithDataOperate(c, 500, "更新工单审批记录失败", "", nil)
		}
		// 创建审批日志
		if err := workOrderNewRepository.CreateWorkOrderLog(&workOrder, &account, result, "指令工单", approvalInfo); err != nil {
			log.Errorf("CreateWorkOrderLog err: %v", err)
		}
		// 更新工单状态
		if err := workOrderNewRepository.UpdateById(id, &model.NewWorkOrder{
			Status:      status,
			ApproveUser: account.Username,
			ApproveId:   account.ID,
		}); err != nil {
			log.Errorf("WorkOrder UpdateById err: %v", err)
			return FailWithDataOperate(c, 500, "审核失败", "", nil)
		}
		return Success(c, "审批成功")
	}

	// -----------------访问工单审批-----------------

	depthCurrent, err := DepLevel(account.DepartmentId)
	if err != nil {
		log.Errorf("DepLevel err: %v", err)
		return FailWithDataOperate(c, 500, "查询部门层级失败", "", nil)
	}
	depthApply, err := DepLevel(workOrder.DepartmentId)
	if err != nil {
		log.Errorf("DepLevel err: %v", err)
		return FailWithDataOperate(c, 500, "查询部门层级失败", "", nil)
	}
	if depthCurrent > depthApply { // 当前用户的部门层级小于申请人的部门层级，不能审批
		return Fail(c, 500, "您没有权限审批该工单")
	}

	/*
		下列审批的具体逻辑，根据以上的筛选条件，得到需要审批的有两类:1.审批中 2.已提交


			是否终审
			1.终审开启
				1.1.会签模式
					1.1.1.当前用户是终审用户
					1.1.2.当前用户不是终审用户，但和终审用户是同一部门
					1.1.3.当前用户不是终审用户，但和终审用户不是同一部门
				1.2.多人模式
					1.2.1.当前用户是终审用户
					1.2.2.当前用户不是终审用户，但和终审用户是同一部门
					1.2.3.当前用户不是终审用户，但和终审用户不是同一部门
			2.终审关闭
				2.1.会签模式
					2.1.1.当前用户是最后一个审批人
					2.1.2.当前用户不是最后一个审批人
				2.2.多人模式
					2.2.1.当前用户是最后一个审批人
					2.2.2.当前用户不是最后一个审批人
	*/
	num := len(workOrderApprovalLog)
	// 终审开启
	if workOrderApprovalLog[0].IsFinalApprove {
		// 会签模式
		if workOrderApprovalLog[0].ApproveWay == constant.SignModeMany {
			//-------检查是否轮到自己审批
			for _, v := range workOrderApprovalLog {
				depth, err := DepLevel(v.DepartmentId)
				if err != nil {
					log.Errorf("DepLevel err: %v", err)
					return FailWithDataOperate(c, 500, "查询部门层级失败", "", nil)
				}
				if depthCurrent < depth && v.Result == constant.NotApproved {
					return Fail(c, 500, "该工单还有下级部门未审批")
				}
			}
			// 当前用户是终审用户
			if account.DepartmentId == 0 && account.RoleName == constant.SystemAdmin {
				// 终审人的决定工单会直接更新工单最终状态
				if err := workOrderApprovalLogRepository.UpdateByWorkOrderIdAndDepartmentId(id, num, account.DepartmentId, &model.WorkOrderApprovalLog{
					ApprovalUsername: account.Username,
					ApprovalNickname: account.Nickname,
					ApprovalId:       account.ID,
					ApprovalDate:     utils.NowJsonTime(),
					Result:           result,
					ApprovalInfo:     approvalInfo,
				}); err != nil {
					log.Errorf("WorkOrderApprovalLog UpdateByWorkOrderId err: %v", err)
					return FailWithDataOperate(c, 500, "更新工单审批记录失败", "", nil)
				}
				// 创建审批日志
				if err := workOrderNewRepository.CreateWorkOrderLog(&workOrder, &account, result, constant.AccessWorkOrder, approvalInfo); err != nil {
					log.Errorf("CreateWorkOrderLog err: %v", err)
				}
				// 更新工单状态
				if err := workOrderNewRepository.UpdateById(id, &model.NewWorkOrder{
					Status:      status,
					ApproveUser: account.Username,
					ApproveId:   account.ID,
				}); err != nil {
					log.Errorf("WorkOrder UpdateById err: %v", err)
					return FailWithDataOperate(c, 500, "审核失败", "", nil)
				}
			} else if account.DepartmentId == 0 && account.RoleName == constant.DepartmentAdmin {
				// 当前用户不是终审用户,但和终审用户是同一部门
				// 更新工单审批记录
				if err := workOrderApprovalLogRepository.UpdateByWorkOrderIdAndDepartmentId(id, num-1, account.DepartmentId, &model.WorkOrderApprovalLog{
					ApprovalUsername: account.Username,
					ApprovalNickname: account.Nickname,
					ApprovalId:       account.ID,
					ApprovalDate:     utils.NowJsonTime(),
					Result:           result,
					ApprovalInfo:     approvalInfo,
				}); err != nil {
					log.Errorf("WorkOrderApprovalLog UpdateByWorkOrderId err: %v", err)
					return FailWithDataOperate(c, 500, "更新工单审批记录失败", "", nil)
				}
				// 创建审批日志
				if err := workOrderNewRepository.CreateWorkOrderLog(&workOrder, &account, result, constant.AccessWorkOrder, approvalInfo); err != nil {
					log.Errorf("CreateWorkOrderLog err: %v", err)
				}
				// 拒绝的话直接更新工单状态为审批拒绝
				if result == constant.Disagree {
					if err := workOrderNewRepository.UpdateById(id, &model.NewWorkOrder{
						Status:      status,
						ApproveUser: account.Username,
						ApproveId:   account.ID,
					}); err != nil {
						log.Errorf("WorkOrder UpdateById err: %v", err)
						return FailWithDataOperate(c, 500, "审核失败", "", nil)
					}
				} else {
					if err := workOrderNewRepository.UpdateById(id, &model.NewWorkOrder{
						Status: constant.Approving,
					}); err != nil {
						log.Errorf("WorkOrder UpdateById err: %v", err)
						return FailWithDataOperate(c, 500, "审核失败", "", nil)
					}
				}
			} else {
				// 不是终审用户，也和终审用户不是同一部门
				// 更新工单审批记录
				if err := workOrderApprovalLogRepository.UpdateByWorkOrderIdAndDepartmentId(id, 0, account.DepartmentId, &model.WorkOrderApprovalLog{
					ApprovalUsername: account.Username,
					ApprovalNickname: account.Nickname,
					ApprovalId:       account.ID,
					ApprovalDate:     utils.NowJsonTime(),
					Result:           result,
					ApprovalInfo:     approvalInfo,
				}); err != nil {
					log.Errorf("WorkOrderApprovalLog UpdateByWorkOrderId err: %v", err)
					return FailWithDataOperate(c, 500, "更新工单审批记录失败", "", nil)
				}
				// 创建审批日志
				if err := workOrderNewRepository.CreateWorkOrderLog(&workOrder, &account, result, constant.AccessWorkOrder, approvalInfo); err != nil {
					log.Errorf("CreateWorkOrderLog err: %v", err)
				}
				// 拒绝的话直接更新工单状态为审批拒绝
				if result == constant.Disagree {
					if err := workOrderNewRepository.UpdateById(id, &model.NewWorkOrder{
						Status:      status,
						ApproveUser: account.Username,
						ApproveId:   account.ID,
					}); err != nil {
						log.Errorf("WorkOrder UpdateById err: %v", err)
						return FailWithDataOperate(c, 500, "审核失败", "", nil)
					}
				} else {
					if err := workOrderNewRepository.UpdateById(id, &model.NewWorkOrder{
						Status: constant.Approving,
					}); err != nil {
						log.Errorf("WorkOrder UpdateById err: %v", err)
						return FailWithDataOperate(c, 500, "审核失败", "", nil)
					}
				}
			}
		}
		// 多人模式
		if workOrderApprovalLog[0].ApproveWay == constant.SignModeOne {
			// 当前用户是终审用户
			if account.DepartmentId == 0 && account.RoleName == constant.SystemAdmin {
				// 检查是否到自己审批
				var flag = false
				for _, v := range workOrderApprovalLog {
					if v.Result == constant.Approved {
						flag = true
					}
				}
				if !flag {
					return FailWithDataOperate(c, 500, "待下级部门审批", "", nil)
				}
				// 更新工单审批记录
				if err := workOrderApprovalLogRepository.UpdateByWorkOrderIdAndDepartmentId(id, num, account.DepartmentId, &model.WorkOrderApprovalLog{
					ApprovalUsername: account.Username,
					ApprovalNickname: account.Nickname,
					ApprovalId:       account.ID,
					ApprovalDate:     utils.NowJsonTime(),
					Result:           result,
					ApprovalInfo:     approvalInfo,
				}); err != nil {
					log.Errorf("WorkOrderApprovalLog UpdateByWorkOrderId err: %v", err)
					return FailWithDataOperate(c, 500, "更新工单审批记录失败", "", nil)
				}
				// 创建审批日志
				if err := workOrderNewRepository.CreateWorkOrderLog(&workOrder, &account, result, constant.AccessWorkOrder, approvalInfo); err != nil {
					log.Errorf("CreateWorkOrderLog err: %v", err)
				}
				// 更新工单状态
				if err := workOrderNewRepository.UpdateById(id, &model.NewWorkOrder{
					Status:      status,
					ApproveUser: account.Username,
					ApproveId:   account.ID,
				}); err != nil {
					log.Errorf("WorkOrder UpdateById err: %v", err)
					return FailWithDataOperate(c, 500, "审核失败", "", nil)
				}
			} else if account.DepartmentId == 0 && account.RoleName == constant.SystemAdmin {
				// 当前用户不是终审用户,但和终审用户是同一部门
				// 更新工单审批记录
				if err := workOrderApprovalLogRepository.UpdateByWorkOrderIdAndDepartmentId(id, num-1, account.DepartmentId, &model.WorkOrderApprovalLog{
					ApprovalUsername: account.Username,
					ApprovalNickname: account.Nickname,
					ApprovalId:       account.ID,
					ApprovalDate:     utils.NowJsonTime(),
					Result:           result,
					ApprovalInfo:     approvalInfo,
				}); err != nil {
					log.Errorf("WorkOrderApprovalLog UpdateByWorkOrderId err: %v", err)
					return FailWithDataOperate(c, 500, "更新工单审批记录失败", "", nil)
				}
				// 创建审批日志
				if err := workOrderNewRepository.CreateWorkOrderLog(&workOrder, &account, result, constant.AccessWorkOrder, approvalInfo); err != nil {
					log.Errorf("CreateWorkOrderLog err: %v", err)
				}
				// 拒绝的话直接更新工单状态为审批拒绝
				if result == constant.Disagree {
					if err := workOrderNewRepository.UpdateById(id, &model.NewWorkOrder{
						Status:      status,
						ApproveUser: account.Username,
						ApproveId:   account.ID,
					}); err != nil {
						log.Errorf("WorkOrder UpdateById err: %v", err)
						return FailWithDataOperate(c, 500, "审核失败", "", nil)
					}
				} else {
					if err := workOrderNewRepository.UpdateById(id, &model.NewWorkOrder{
						Status: constant.Approving,
					}); err != nil {
						log.Errorf("WorkOrder UpdateById err: %v", err)
						return FailWithDataOperate(c, 500, "审核失败", "", nil)
					}
				}
			} else {
				// 不是终审用户，也和终审用户不是同一部门
				// 更新工单审批记录
				if err := workOrderApprovalLogRepository.UpdateByWorkOrderIdAndDepartmentId(id, 0, account.DepartmentId, &model.WorkOrderApprovalLog{
					ApprovalUsername: account.Username,
					ApprovalNickname: account.Nickname,
					ApprovalId:       account.ID,
					ApprovalDate:     utils.NowJsonTime(),
					Result:           result,
					ApprovalInfo:     approvalInfo,
				}); err != nil {
					log.Errorf("WorkOrderApprovalLog UpdateByWorkOrderId err: %v", err)
					return FailWithDataOperate(c, 500, "更新工单审批记录失败", "", nil)
				}
				// 创建审批日志
				if err := workOrderNewRepository.CreateWorkOrderLog(&workOrder, &account, result, constant.AccessWorkOrder, approvalInfo); err != nil {
					log.Errorf("CreateWorkOrderLog err: %v", err)
				}
				// 拒绝的话直接更新工单状态为审批拒绝
				if result == constant.Disagree {
					if err := workOrderNewRepository.UpdateById(id, &model.NewWorkOrder{
						Status:      status,
						ApproveUser: account.Username,
						ApproveId:   account.ID,
					}); err != nil {
						log.Errorf("WorkOrder UpdateById err: %v", err)
						return FailWithDataOperate(c, 500, "审核失败", "", nil)
					}
				} else {
					if err := workOrderNewRepository.UpdateById(id, &model.NewWorkOrder{
						Status: constant.Approving,
					}); err != nil {
						log.Errorf("WorkOrder UpdateById err: %v", err)
						return FailWithDataOperate(c, 500, "审核失败", "", nil)
					}
				}
			}
		}
	}
	// 终审关闭
	if !workOrderApprovalLog[0].IsFinalApprove {
		// 会签模式
		if workOrderApprovalLog[0].ApproveWay == constant.SignModeMany {
			//-------检查是否轮到自己审批
			for _, v := range workOrderApprovalLog {
				depth, err := DepLevel(v.DepartmentId)
				if err != nil {
					log.Errorf("DepLevel err: %v", err)
					return FailWithDataOperate(c, 500, "查询部门层级失败", "", nil)
				}
				if depthCurrent < depth && v.Result == constant.NotApproved {
					return Fail(c, 500, "该工单还有下级部门未审批")
				}
			}
			// 当前用户是最后一个审批人
			if workOrderApprovalLog[len(workOrderApprovalLog)-1].DepartmentId == account.DepartmentId {
				// 更新日志审批结果
				if err := workOrderApprovalLogRepository.UpdateByWorkOrderIdAndDepartmentId(id, num, account.DepartmentId, &model.WorkOrderApprovalLog{
					ApprovalUsername: account.Username,
					ApprovalNickname: account.Nickname,
					ApprovalId:       account.ID,
					ApprovalDate:     utils.NowJsonTime(),
					Result:           result,
					ApprovalInfo:     approvalInfo,
				}); err != nil {
					log.Errorf("WorkOrderApprovalLog UpdateByWorkOrderId err: %v", err)
					return FailWithDataOperate(c, 500, "更新工单审批记录失败", "", nil)
				}
				// 创建审批日志
				if err := workOrderNewRepository.CreateWorkOrderLog(&workOrder, &account, result, constant.AccessWorkOrder, approvalInfo); err != nil {
					log.Errorf("CreateWorkOrderLog err: %v", err)
				}
				// 更新工单状态
				if err := workOrderNewRepository.UpdateById(id, &model.NewWorkOrder{
					Status:      status,
					ApproveUser: account.Username,
					ApproveId:   account.ID,
				}); err != nil {
					log.Errorf("WorkOrder UpdateById err: %v", err)
					return FailWithDataOperate(c, 500, "审核失败", "", nil)
				}
			}
			// 当前用户不是最后一个审批人
			if workOrderApprovalLog[len(workOrderApprovalLog)-1].DepartmentId != account.DepartmentId {
				// 更新日志审批结果
				if err := workOrderApprovalLogRepository.UpdateByWorkOrderIdAndDepartmentId(id, 0, account.DepartmentId, &model.WorkOrderApprovalLog{
					ApprovalUsername: account.Username,
					ApprovalNickname: account.Nickname,
					ApprovalId:       account.ID,
					ApprovalDate:     utils.NowJsonTime(),
					Result:           result,
					ApprovalInfo:     approvalInfo,
				}); err != nil {
					log.Errorf("WorkOrderApprovalLog UpdateByWorkOrderId err: %v", err)
					return FailWithDataOperate(c, 500, "更新工单审批记录失败", "", nil)
				}
				// 创建审批日志
				if err := workOrderNewRepository.CreateWorkOrderLog(&workOrder, &account, result, constant.AccessWorkOrder, approvalInfo); err != nil {
					log.Errorf("CreateWorkOrderLog err: %v", err)
				}
				// 按条件更新工单状态
				if result == constant.Disagree {
					// 若为拒绝直接更新工单状态为审批拒绝
					if err := workOrderNewRepository.UpdateById(id, &model.NewWorkOrder{
						Status:      status,
						ApproveUser: account.Username,
						ApproveId:   account.ID,
					}); err != nil {
						log.Errorf("WorkOrder UpdateById err: %v", err)
						return FailWithDataOperate(c, 500, "审核失败", "", nil)
					}
				} else {
					// 通过的话直接更新工单状态为审批中
					if err := workOrderNewRepository.UpdateById(id, &model.NewWorkOrder{
						Status:      constant.Approving,
						ApproveUser: account.Username,
						ApproveId:   account.ID,
					}); err != nil {
						log.Errorf("WorkOrder UpdateById err: %v", err)
						return FailWithDataOperate(c, 500, "审核失败", "", nil)
					}
				}
			}
		}
		// 多人模式直接审批
		if workOrderApprovalLog[0].ApproveWay == constant.SignModeOne {
			// 检查是否有权限审批资产
			assetList, err := FindPassportListByWorkOrderId(workOrder.ID)
			if err != nil {
				log.Errorf("WorkOrder FindById err: %v", err)
				return FailWithDataOperate(c, 500, "获取失败", "", nil)
			}
			for _, asset := range assetList {
				depth := GetUserDepth(asset.ID, "asset")
				if depth == -1 {
					return Fail(c, 500, "获取资产部门信息失败")
				}
				if depth < depthCurrent {
					return Fail(c, 500, "该工单所申请的设备部分从属于更高级部门,故您没有权限审批该工单")
				}
			}
			// 此条件下可以直接审批
			if err := workOrderApprovalLogRepository.UpdateByWorkOrderIdAndDepartmentId(id, 0, account.DepartmentId, &model.WorkOrderApprovalLog{
				ApprovalUsername: account.Username,
				ApprovalNickname: account.Nickname,
				ApprovalId:       account.ID,
				ApprovalDate:     utils.NowJsonTime(),
				Result:           result,
				ApprovalInfo:     approvalInfo,
			}); err != nil {
				log.Errorf("WorkOrderApprovalLog UpdateByWorkOrderId err: %v", err)
				return FailWithDataOperate(c, 500, "更新工单审批记录失败", "", nil)
			}
			// 创建审批日志
			if err := workOrderNewRepository.CreateWorkOrderLog(&workOrder, &account, result, constant.AccessWorkOrder, approvalInfo); err != nil {
				log.Errorf("CreateWorkOrderLog err: %v", err)
			}
			// 更新工单状态
			if err := workOrderNewRepository.UpdateById(id, &model.NewWorkOrder{
				Status:      status,
				ApproveUser: account.Username,
				ApproveId:   account.ID,
			}); err != nil {
				log.Errorf("WorkOrder UpdateById err: %v", err)
				return FailWithDataOperate(c, 500, "审核失败", "", nil)
			}
		}
	}
	return SuccessWithOperate(c, "工单审批-审批: 工单["+workOrder.Title+"->"+status+"]", nil)

}

// WorkOrderCloseOrCancelEndpoint 关闭工单
func WorkOrderCloseOrCancelEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	pwd := c.QueryParam("password")
	// 获取当前用户
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, -1, "当前用户信息已过期，请重新登录")
	}
	// 验证密码是否正确
	if err := utils.Encoder.Match([]byte(account.Password), []byte(pwd)); err != nil {
		return Fail(c, -1, "您输入的密码不正确")
	}
	workOrder, err := workOrderNewRepository.FindById(id)
	if err != nil {
		log.Errorf("WorkOrder FindById err: %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	fmt.Println(workOrder, "workOrder", workOrder.Status)
	if workOrder.Status != constant.Approved {
		return Fail(c, 500, "该工单无法关闭")
	}

	// 检查有无撤销的权限
	// 仅多人模式下的审批通过才可关闭
	// 查询工单的审批记录表
	workOrderApprovalLog, err := workOrderApprovalLogRepository.FindByWorkOrderId(id)
	if err != nil {
		log.Errorf("WorkOrderApprovalLog FindByWorkOrderId err: %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	if len(workOrderApprovalLog) == 0 {
		return Fail(c, 500, "该工单没有审批记录，无法关闭")
	}
	// 查看当前工单的审批方式
	if workOrderApprovalLog[0].ApproveWay == constant.SignModeOne {
		return Fail(c, 500, "该工单为会签审批，无法关闭")
	}
	// 检查当前用户是否有权限关闭
	// 工单的审批人所在的部门深度
	approveAccountDepth := GetUserDepth(workOrder.ApproveId, "user")
	currentUserDepth := GetUserDepth(account.ID, "user")
	applyAccountDepth := GetUserDepth(workOrder.ApplyId, "user")
	if approveAccountDepth == -1 || currentUserDepth == -1 || applyAccountDepth == -1 {
		log.Errorf("GetUserDepth err: %v", err)
		return Fail(c, 500, "获取用户部门深度失败")
	}
	// 审批人级别比当前用户级别更高，无法关闭,当前用户是级别比申请人级别低，无法关闭
	if currentUserDepth > approveAccountDepth || currentUserDepth > applyAccountDepth {
		return Fail(c, 500, "您没有权限关闭该工单")
	} else {
		if err := workOrderApprovalLogRepository.UpdateByWorkOrderIdAndDepartmentId(id, 0, account.DepartmentId, &model.WorkOrderApprovalLog{
			ApprovalUsername: account.Username,
			ApprovalNickname: account.Nickname,
			ApprovalId:       account.ID,
			Result:           constant.ApprovalRevocation,
		}); err != nil {
			log.Errorf("WorkOrderApprovalLog UpdateByWorkOrderId err: %v", err)
			return FailWithDataOperate(c, 500, "更新工单审批记录失败", "", nil)
		}
		if err := workOrderNewRepository.UpdateById(id, &model.NewWorkOrder{
			Status:      constant.Closed,
			ApproveUser: account.Username,
			ApproveId:   account.ID,
		}); err != nil {
			log.Errorf("WorkOrder UpdateById err: %v", err)
			return FailWithDataOperate(c, 500, "关闭失败", "", nil)
		}
	}
	if err := workOrderNewRepository.CreateWorkOrderLog(&workOrder, &account, constant.Closed, constant.AccessWorkOrder, ""); err != nil {
		log.Errorf("WorkOrder CreateWorkOrderLog err: %v", err)
	}
	return SuccessWithOperate(c, "工单审批-关闭工单: 工单["+workOrder.Title+"->"+constant.Closed+"]", nil)
}

// ---------------------------------函数部分---------------------------------

// GetUserDepth 根据Id和id类型返回其部门所在深度
func GetUserDepth(id string, idType string) (depth int) {
	if idType == "user" {
		// 获取当前用户
		account, err := userNewRepository.FindById(id)
		if err != nil {
			log.Errorf("User FindById err: %v", err)
			return -1
		}
		// 获取当前用户的部门所在深度
		depth, err = DepLevel(account.DepartmentId)
		if err != nil {
			log.Errorf("DepLevel err: %v", err)
			return -1
		}
	}
	if idType == "asset" {
		// 获取当前资产
		passport, err := newAssetRepository.GetPassportById(context.TODO(), id)
		if err != nil {
			log.Errorf("GetPassportById err: %v", err)
			return -1
		}
		// 获取当前资产的部门所在深度
		depth, err = DepLevel(passport.DepartmentId)
		if err != nil {
			log.Errorf("DepLevel err: %v", err)
			return -1
		}
	}
	return depth
}

// FindPassportListByWorkOrderId 查找与该工单id关联的设备信息
func FindPassportListByWorkOrderId(workOrderId string) (assetList []model.PassPort, err error) {
	workOrder, err := workOrderNewRepository.FindById(workOrderId)
	if err != nil {
		return nil, err
	}
	// 通过工单id获取工单关联的设备
	workOrderAssetList, err := workOrderAssetRepository.FindByWorkOrderId(workOrder.ID)
	if err != nil {
		return nil, err
	}
	// 通过工单关联的设备id获取设备信息
	for _, v := range workOrderAssetList {
		passport, err := newAssetRepository.GetPassportById(context.TODO(), v.AssetId)
		if err != nil {
			_ = workOrderAssetRepository.DeleteByWorkOrderIdAndAssetId(workOrder.ID, v.AssetId)
			continue
		}
		assetList = append(assetList, passport)
	}
	return
}
func FindPassportListByOrderId(orderId string) (assetList []model.PassPort, err error) {
	workOrder, err := workOrderNewRepository.FindByOrderId(orderId)
	if err != nil {
		return nil, err
	}
	// 通过工单id获取工单关联的设备
	workOrderAssetList, err := workOrderAssetRepository.FindByWorkOrderId(workOrder.ID)
	if err != nil {
		return nil, err
	}
	// 通过工单关联的设备id获取设备信息
	for _, v := range workOrderAssetList {
		passport, err := newAssetRepository.GetPassportById(context.TODO(), v.AssetId)
		if err != nil {
			return nil, err
		}
		assetList = append(assetList, passport)
	}
	return
}

// 根据申请人id删除工单
func DeleteWorkOrderByApplyId(id string) (err error) {
	// 删除工单
	if err := workOrderNewRepository.DeleteByApplyId(id); err != nil {
		return err
	}
	// 删除工单关联的设备
	if err := workOrderAssetRepository.DeleteByWorkOrderId(id); err != nil {
		return err
	}
	// 查询该用户的工单记录
	workOrderLogList, err := workOrderNewRepository.FindByApplyId(id)
	if err != nil {
		return err
	}
	for _, v := range workOrderLogList {
		// 删除该申请人的工单审批记录
		if err := workOrderNewRepository.DeleteByOrderId(v.OrderId); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------分割线---------------------------------

// CancelWorkOrder 取消工单
func CancelWorkOrder(c echo.Context) error {
	// 获取当前用户
	account, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithDataOperate(c, 401, "取消工单失败", "", nil)
	}
	order, _ := work_order.GetOrderByUid(account.ID)
	if nil == order {
		fmt.Println("order", order)
		log.Error("workOrderApproval order is nil")
		return FailWithDataOperate(c, 500, "取消工单失败，未找到此工单", "命令工单审批:"+account.Nickname+"未找到此工单信息", nil)
	}
	//向审批人发送审批结果
	order.Status <- "已取消"
	order.Approved = account.Nickname
	defer close(order.Status)
	if err := workOrderApprovalLogRepository.UpdateByWorkOrderIdAndDepartmentId(order.ID, 1, account.DepartmentId, &model.WorkOrderApprovalLog{
		ApprovalUsername: account.Username,
		ApprovalNickname: account.Nickname,
		ApprovalId:       account.ID,
		ApprovalDate:     utils.NowJsonTime(),
		Result:           "已取消",
		ApprovalInfo:     "已取消",
	}); err != nil {
		log.Errorf("WorkOrderApprovalLog UpdateByWorkOrderId err: %v", err)
		return FailWithDataOperate(c, 500, "更新工单审批记录失败", "", nil)
	}
	// 更新工单状态
	if err := workOrderNewRepository.UpdateByOrderId(order.ID, &model.NewWorkOrder{
		Status:      "已取消",
		ApproveUser: account.Username,
		ApproveId:   account.ID,
	}); err != nil {
		log.Errorf("WorkOrder UpdateById err: %v", err)
		return FailWithDataOperate(c, 500, "取消工单失败", "", nil)
	}
	olg, err := workOrderNewRepository.FindByOrderId(order.ID)
	if err != nil {
		log.Errorf("workOrderLogRepository FindByWorkOrderId err: %v", err)
		return FailWithDataOperate(c, 500, "取消工单失败", "", nil)
	}
	// 更新工单审批记录
	if err := workOrderNewRepository.CreateWorkOrderLog(&olg, &account, "取消", "指令工单", ""); err != nil {
		log.Errorf("WorkOrder CreateWorkOrderLog err: %v", err)
		return FailWithDataOperate(c, 500, "取消工单失败", "", nil)
	}
	return Success(c, "取消工单成功")
}

// ---------------------------------日志部分---------------------------------

// GetWorkOrderLogEndpoint 获取审批日志
func GetWorkOrderLogEndpoint(c echo.Context) error {
	auto := c.QueryParam("auto")
	title := c.QueryParam("title")
	submitTime := c.QueryParam("submitTime")
	approveTime := c.QueryParam("approveTime")
	approveUsername := c.QueryParam("approveUsername")
	approveNickname := c.QueryParam("approveNickname")
	approveResult := c.QueryParam("approveResult")
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 500, "获取用户登录信息失效，请重新登录")
	}
	// 获取当前部门的下属所有部门id
	var depIds []int64
	err := GetChildDepIds(account.DepartmentId, &depIds)
	if err != nil {
		return Fail(c, 500, "获取部门信息失败")
	}
	newWorkOrderLogList, err := workOrderNewRepository.FindWorkOrderLogByDepIds(depIds, auto, title, submitTime, approveTime, approveUsername, approveNickname, approveResult)
	if err != nil {
		log.Errorf("FindWorkOrderLogByDepIds err: %v", err)
		return Fail(c, 500, "获取审批日志失败")
	}
	// 查询工单的审批记录
	return Success(c, newWorkOrderLogList)
}

// GetWorkOrderLogDetailEndpoint 获取审批日志详情
func GetWorkOrderLogDetailEndpoint(c echo.Context) error {
	orderId := c.Param("id")
	// 通过工单id获取工单
	workOrder, err := workOrderNewRepository.FindByOrderId(orderId)
	if err != nil {
		log.Errorf("WorkOrder FindById err: %v", err)
		return FailWithDataOperate(c, 500, "工单不存在", "", nil)
	}
	// 查询申请人
	applyAccount, err := userNewRepository.FindById(workOrder.ApplyId)
	if err != nil {
		log.Errorf("User FindById err: %v", err)
		return FailWithDataOperate(c, 500, "查询申请人失败", "", nil)
	}
	var validTime string
	if workOrder.IsPermanent != nil {
		if *workOrder.IsPermanent {
			validTime = "永久有效"
		} else {
			if !workOrder.BeginTime.IsZero() {
				validTime += workOrder.BeginTime.Format("2006-01-02 15:04:05")
			}
			validTime += "~"
			if !workOrder.EndTime.IsZero() {
				validTime += workOrder.EndTime.Format("2006-01-02 15:04:05")
			}
		}
	} else {
		if !workOrder.BeginTime.IsZero() {
			validTime += workOrder.BeginTime.Format("2006-01-02 15:04:05") + "~"
		}
		validTime += "~"
		if !workOrder.EndTime.IsZero() {
			validTime += workOrder.EndTime.Format("2006-01-02 15:04:05")
		}
	}
	workOrderDetail := dto.WorkOrderForDetail{
		ID:            workOrder.ID,
		OrderId:       workOrder.OrderId,
		Title:         workOrder.Title,
		WorkOrderType: workOrder.WorkOrderType,
		Status:        workOrder.Status,
		Description:   workOrder.Description,
		ApplyUser:     workOrder.ApplyUser,
		ValidTime:     validTime,
		ApplyUserName: applyAccount.Nickname,
		ApplyId:       workOrder.ApplyId,
		ApplyTime:     workOrder.ApplyTime.Format("2006-01-02 15:04:05"),
	}
	return Success(c, workOrderDetail)
}

// NewWorkOrderLogAssetInfoEndPoint 查看日志的的资产信息
func NewWorkOrderLogAssetInfoEndPoint(c echo.Context) error {
	orderId := c.Param("id")
	// 通过工单id获取工单
	assetList, err := FindPassportListByOrderId(orderId)
	if err != nil {
		log.Errorf("WorkOrder FindById err: %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	var workOrderAsset []dto.WorkOrderForAsset
	for _, v := range assetList {
		// 通过部门id获取部门名称
		department, err := departmentRepository.FindById(v.DepartmentId)
		if err != nil {
			log.Errorf("Department FindById err: %v", err)
		}
		workOrderAsset = append(workOrderAsset, dto.WorkOrderForAsset{
			ID:         v.ID,
			Name:       v.AssetName,
			Ip:         v.Ip,
			Protocol:   v.Protocol,
			Passport:   v.Passport,
			Department: department.Name,
			Port:       v.Port,
		})
	}
	return Success(c, workOrderAsset)
}

// NewWorkOrderLogApproveInfoEndPoint 查看日志的审批信息
func NewWorkOrderLogApproveInfoEndPoint(c echo.Context) error {
	orderId := c.Param("id")
	// 通过工单号获取工单
	workOrder, err := workOrderNewRepository.FindByOrderId(orderId)
	if err != nil {
		log.Errorf("WorkOrder FindById err: %v", err)
		return FailWithDataOperate(c, 500, "工单不存在", "", nil)
	}
	// 通过工单id获取工单关联的审批信息
	workOrderApproveList, err := workOrderApprovalLogRepository.FindByWorkOrderId(workOrder.ID)
	if err != nil {
		log.Errorf("WorkOrderApprovalLog FindByWorkOrderId err: %v", err)
		return FailWithDataOperate(c, 500, "获取工单关联的审批信息失败", "", nil)
	}
	var workOrderApprovePaging = make([]dto.WorkOrderApprovalLogForPaging, len(workOrderApproveList))
	for i, v := range workOrderApproveList {
		workOrderApprovePaging[i].ID = v.ID
		workOrderApprovePaging[i].WorkOrderId = v.WorkOrderId
		workOrderApprovePaging[i].Number = v.Number
		workOrderApprovePaging[i].ApprovalUsername = v.ApprovalUsername
		workOrderApprovePaging[i].ApprovalNickname = v.ApprovalNickname
		workOrderApprovePaging[i].Department = v.Department
		workOrderApprovePaging[i].DepartmentId = v.DepartmentId
		workOrderApprovePaging[i].ApprovalId = v.ApprovalId
		workOrderApprovePaging[i].Result = v.Result
		workOrderApprovePaging[i].ApproveWay = v.ApproveWay
		workOrderApprovePaging[i].IsFinalApprove = v.IsFinalApprove
		workOrderApprovePaging[i].ApprovalInfo = v.ApprovalInfo
		if v.ApprovalDate.IsZero() {
			workOrderApprovePaging[i].ApprovalDate = ""
		} else {
			workOrderApprovePaging[i].ApprovalDate = v.ApprovalDate.Time.Format("2006-01-02 15:04:05")
		}
	}
	return Success(c, workOrderApprovePaging)
}

// ExportWorkOrderLogEndpoint 导出审批日志
func ExportWorkOrderLogEndpoint(c echo.Context) error {
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 500, "获取用户登录信息失效，请重新登录")
	}
	// 获取当前部门的下属所有部门id
	var depIds []int64
	err := GetChildDepIds(account.DepartmentId, &depIds)
	if err != nil {
		return Fail(c, 500, "获取部门信息失败")
	}
	newWorkOrderLogList, err := workOrderNewRepository.FindWorkOrderLogByDepIds(depIds, "", "", "", "", "", "", "")
	if err != nil {
		log.Errorf("WorkOrder FindWorkOrderLogByDepIds err: %v", err)
		return Fail(c, 500, "获取审批日志失败")
	}
	workOrderLogArr := make([]dto.WorkOrderLogForExport, len(newWorkOrderLogList))
	for i, v := range newWorkOrderLogList {
		workOrderLogArr[i] = dto.WorkOrderLogForExport{
			Title:           v.Title,
			ApplyTime:       v.ApplyTime.Format("2006-01-02 15:04:05"),
			ApproveTime:     v.ApproveTime.Format("2006-01-02 15:04:05"),
			ApproveUsername: v.ApproveUsername,
			ApproveNickname: v.ApproveNickname,
			Department:      v.Department,
			Result:          v.Result,
			Info:            v.ApproveInfo,
		}
	}
	workOrderLogStringsForExport := make([][]string, len(workOrderLogArr))
	for i, v := range workOrderLogArr {
		user := utils.Struct2StrArr(v)
		workOrderLogStringsForExport[i] = make([]string, len(user))
		workOrderLogStringsForExport[i] = user
	}
	userHeaderForExport := []string{"标题", "提交时间", "审批时间", "审批人", "姓名", "部门机构", "审批结果", "审批备注"}
	userFileNameForExport := "审批日志"
	file, err := utils.CreateExcelFile(userFileNameForExport, userHeaderForExport, workOrderLogStringsForExport)
	if err != nil {
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	fileName := "审批日志.xlsx"

	// 记录操作日志
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		Result:          "成功",
		LogContents:     "审批日志-导出: [审批日志导出]",
		Created:         utils.NowJsonTime(),
		Users:           account.Username,
		Names:           account.Nickname,
	}
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}

	var buff bytes.Buffer
	if err = file.Write(&buff); err != nil {
		log.Errorf("Write Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "", err)
	}
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
}
