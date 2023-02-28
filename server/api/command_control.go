package api

import (
	"context"
	"strconv"
	"strings"
	"time"
	"tkbastion/pkg/constant"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/log"
	"tkbastion/pkg/validator"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// CommandStrategyCreatEndpoint 指令策略创建
func CommandStrategyCreatEndpoint(c echo.Context) error {
	var item dto.CommandStrategyForCreat
	// 绑定数据
	if err := c.Bind(&item); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}

	if err := c.Validate(&item); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}
	// 策略名称不能重复
	var itemExists []model.CommandStrategy
	if err := commandStrategyRepository.DB.Where("name = ?", item.Name).Find(&itemExists).Error; nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}
	if 0 != len(itemExists) {
		return FailWithDataOperate(c, 403, "策略名称已存在", "指令控制-指令策略-新增: 策略名称["+item.Name+"],失败原因[策略名称已存在]", nil)
	}
	// 获取当前用户信息添加部门id与名称
	account, found := GetCurrentAccountNew(c)
	if !found {
		log.Errorf("GetCurrentAccount Error: %v", found)
		return FailWithDataOperate(c, 500, "用户信息已失效", "", found)
	}
	departmentDepth, err := DepLevel(account.DepartmentId)
	if err != nil {
		log.Errorf("DepLevel Error: %v", err)
		return FailWithDataOperate(c, 500, "获取部门层级失败", "", err)
	}
	// 获取当前部门id下有多少人以确定优先级
	count, err := commandStrategyRepository.CountByDepartmentId(account.DepartmentId)
	if err != nil {
		log.Errorf("Count Error: %v", err)
	}
	itemCreat := model.CommandStrategy{
		ID:              utils.UUID(),
		Name:            item.Name,
		DepartmentId:    account.DepartmentId,
		DepartmentName:  account.DepartmentName,
		DepartmentDepth: departmentDepth,
		Level:           item.Level,
		Action:          item.Action,
		Status:          constant.Disable,
		Description:     item.Description,
		Priority:        count + 1,
		AlarmByMessage:  &item.AlarmByMessage,
		AlarmByEmail:    &item.AlarmByEmail,
		AlarmByPhone:    &item.AlarmByPhone,
		IsPermanent:     &item.IsPermanent,
		BeginValidTime:  item.BeginValidTime,
		EndValidTime:    item.EndValidTime,
		CreateTime:      utils.NowJsonTime(),
	}
	if err := commandStrategyRepository.Create(&itemCreat); err != nil {
		log.Errorf("Create Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}

	// 处理直接关联的内容
	t := true
	if err := commandContentRepository.Create(&model.CommandContent{
		ID:          utils.UUID(),
		ContentId:   itemCreat.ID,
		Content:     item.Cmd,
		IsRegular:   &t,
		Description: "指令策略直接关联的指令内容",
	}); err != nil {
		log.Errorf("Create CommandContent Error: %v", err)
	}

	// 处理关联的指令集
	if item.CommandSetId != "" {
		splits := strings.Split(item.CommandSetId, ",")
		for _, v := range splits {
			if err := commandRelevanceRepository.Create(&model.CommandRelevance{
				ID:                utils.UUID(),
				CommandStrategyId: itemCreat.ID,
				CommandSetId:      v,
			}); err != nil {
				log.Errorf("Create CommandSet CommandRelevance Error: %v", err)
			}
		}
	}

	// 处理关联用户
	if item.UserId != "" {
		splits := strings.Split(item.UserId, ",")
		for _, v := range splits {
			if err := commandRelevanceRepository.Create(&model.CommandRelevance{
				ID:                utils.UUID(),
				CommandStrategyId: itemCreat.ID,
				UserId:            v,
			}); err != nil {
				log.Errorf("Create User CommandRelevance Error: %v", err)
			}
		}

	}

	// 处理关联的用户组
	if item.UserGroupId != "" {
		splits := strings.Split(item.UserGroupId, ",")
		for _, v := range splits {
			if err := commandRelevanceRepository.Create(&model.CommandRelevance{
				ID:                utils.UUID(),
				CommandStrategyId: itemCreat.ID,
				UserGroupId:       v,
			}); err != nil {
				log.Errorf("Create UserGroup CommandRelevance Error: %v", err)
			}
		}
	}

	// 处理关联的主机
	if item.AssetId != "" {
		splits := strings.Split(item.AssetId, ",")
		for _, v := range splits {
			if err := commandRelevanceRepository.Create(&model.CommandRelevance{
				ID:                utils.UUID(),
				CommandStrategyId: itemCreat.ID,
				AssetId:           v,
			}); err != nil {
				log.Errorf("Create Asset CommandRelevance Error: %v", err)
			}
		}
	}

	// 处理关联的主机组
	if item.AssetGroupId != "" {
		splits := strings.Split(item.AssetGroupId, ",")
		for _, v := range splits {
			if err := commandRelevanceRepository.Create(&model.CommandRelevance{
				ID:                utils.UUID(),
				CommandStrategyId: itemCreat.ID,
				AssetGroupId:      v,
			}); err != nil {
				log.Errorf("Create AssetGroup CommandRelevance Error: %v", err)
			}
		}
	}
	return SuccessWithOperate(c, "指令策略-新增: 策略名称["+item.Name+"]", item)
}

// CommandStrategyUpdateEndpoint 指令策略修改
func CommandStrategyUpdateEndpoint(c echo.Context) error {
	id := c.Param("id")
	oldStrategy, err := commandStrategyRepository.FindById(id)
	if err != nil {
		log.Errorf("FindById Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	var item dto.CommandStrategyForUpdate
	// 绑定数据
	if err := c.Bind(&item); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if err := c.Validate(&item); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}
	// 策略名称不能重复
	var itemExists []model.CommandStrategy
	if err := commandStrategyRepository.DB.Where("name = ? and id != ? ", item.Name, id).Find(&itemExists).Error; nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if 0 != len(itemExists) {
		return FailWithDataOperate(c, 403, "策略名称已存在", "指令策略-修改: 策略名称["+item.Name+"],失败原因[策略名称已存在]", nil)
	}

	// 更新空描述信息
	if item.Description == "" {
		item.Description = " "
	}

	if item.Priority != oldStrategy.Priority {
		// 新优先级大于原优先级，此范围内数据优先级-1
		if item.Priority > oldStrategy.Priority {
			if err := commandStrategyRepository.DB.Table("command_strategy").Where("priority > ? and priority <= ? and department_id = ? ", oldStrategy.Priority, item.Priority, oldStrategy.DepartmentId).Update("priority", gorm.Expr("priority - 1")).Error; nil != err {
				log.Errorf("DB Error: %v", err)
				return FailWithDataOperate(c, 500, "更新优先级失败", "", err)
			}
		}
		// 新优先级小玉于原优先级，此范围内数据优先级+1
		if item.Priority < oldStrategy.Priority {
			if err := commandStrategyRepository.DB.Table("command_strategy").Where("priority >= ? and priority < ? and department_id = ? ", item.Priority, oldStrategy.Priority, oldStrategy.DepartmentId).Update("priority", gorm.Expr("priority + 1")).Error; nil != err {
				log.Errorf("DB Error: %v", err)
				return FailWithDataOperate(c, 500, "更新优先级失败", "", err)
			}

		}

	}
	// 获取当前部门的策略数量
	count, _ := commandStrategyRepository.CountByDepartmentId(oldStrategy.DepartmentId)
	if item.Priority >= count {
		item.Priority = count
	}
	//由已过期修改为可用时间内则修改状态为已禁用
	var status string
	if oldStrategy.Status == constant.Expiration {
		if item.IsPermanent || (time.Now().After(item.BeginValidTime.Time) && time.Now().Before(item.EndValidTime.Time)) {
			status = constant.Disable
		}
	}
	if err := commandStrategyRepository.UpdateById(&model.CommandStrategy{
		ID:             item.ID,
		Name:           item.Name,
		Priority:       item.Priority,
		Level:          item.Level,
		Action:         item.Action,
		Description:    item.Description,
		AlarmByMessage: &item.AlarmByMessage,
		AlarmByEmail:   &item.AlarmByEmail,
		AlarmByPhone:   &item.AlarmByPhone,
		IsPermanent:    &item.IsPermanent,
		Status:         status,
		BeginValidTime: item.BeginValidTime,
		EndValidTime:   item.EndValidTime,
	}, id); err != nil {
		log.Errorf("Update Error: %v", err)
		return FailWithDataOperate(c, 500, "更新失败", "", err)
	}
	return SuccessWithOperate(c, "指令策略-修改: 策略名称["+item.Name+"]", item)
}

// CommandStrategyDeleteEndpoint 指令策略删除
func CommandStrategyDeleteEndpoint(c echo.Context) error {
	id := c.Param("id")
	splits := strings.Split(id, ",")
	var deleteName string
	var deleteCount int
	for i := range splits {
		if splits[i] == "" {
			continue
		}
		//查找id是否有效
		commandStrategy, err := commandStrategyRepository.FindById(splits[i])
		if err != nil {
			log.Errorf("FindById Error: %v", err)
			continue
		}
		if err := DeleteCommandStrategyById(splits[i]); err != nil {
			log.Errorf("DeleteByDepartmentId Error: %v", err)
			continue
		}
		deleteName += commandStrategy.Name + ","
		deleteCount++
	}
	return SuccessWithOperate(c, "指令策略-删除: 删除策略名称["+deleteName+"]删除成功数["+strconv.Itoa(deleteCount)+"]", nil)
}

// CommandStrategyPagingEndpoint  指令策略列表获取
func CommandStrategyPagingEndpoint(c echo.Context) error {
	pageIndex, _ := strconv.Atoi(c.QueryParam("pageIndex"))
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))
	auto := c.QueryParam("auto")
	name := c.QueryParam("name")
	department := c.QueryParam("department")
	level := c.QueryParam("level")
	action := c.QueryParam("action")
	status := c.QueryParam("status")
	description := c.QueryParam("description")
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 401, "您的登录信息已失效,请重新登录后再试.")
	}
	var departmentId []int64
	if err := GetChildDepIds(account.DepartmentId, &departmentId); err != nil {
		log.Errorf("GetChildDepIds Error: %v", err)
	}
	commandStrategy, _, err := commandStrategyRepository.FindByLimitingConditions(pageIndex, pageSize, auto, name, department, level, action, status, description, departmentId)
	if err != nil {
		log.Errorf("FindByLimitingConditions Error: %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", err)
	}
	for i := range commandStrategy {
		if *commandStrategy[i].IsPermanent == false {
			if time.Now().After(commandStrategy[i].EndValidTime.Time) {
				commandStrategy[i].Status = constant.Expiration
				_ = commandStrategyRepository.UpdateById(&commandStrategy[i], commandStrategy[i].ID)
			}
		}
		dep, err := departmentRepository.FindById(commandStrategy[i].DepartmentId)
		if err != nil {
			log.Errorf("FindById Error: %v", err)
		}
		commandStrategy[i].DepartmentName = dep.Name
		// 更新策略的部门机构名称
		_ = commandStrategyRepository.UpdateById(&model.CommandStrategy{DepartmentName: dep.Name}, commandStrategy[i].ID)
	}
	return Success(c, commandStrategy)
}

// CommandStrategyStatusEndpoint 启用/禁用指令策略
func CommandStrategyStatusEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	status := c.QueryParam("status")
	strategy, err := commandStrategyRepository.FindById(id)
	if err != nil {
		log.Errorf("FindById Error: %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", err)
	}
	if strategy.Status == constant.Expiration {
		return Fail(c, 500, "策略已过期")
	}
	if status == constant.Disable {
		strategy.Status = constant.Disable
	} else {
		strategy.Status = constant.Enable
	}
	if err := commandStrategyRepository.UpdateById(&strategy, id); err != nil {
		log.Errorf("UpdateById Error: %v", err)
		return FailWithDataOperate(c, 500, "更新失败", "", err)
	}
	if status == constant.Disable {
		return SuccessWithOperate(c, "指令策略-禁用: 策略名称["+strategy.Name+"]", nil)
	} else {
		return SuccessWithOperate(c, "指令策略-启用: 策略名称["+strategy.Name+"]", nil)
	}
}

// CommandSetCreateEndpoint 指令集创建
func CommandSetCreateEndpoint(c echo.Context) (err error) {
	var item model.CommandSet
	if err := c.Bind(&item); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}

	if err := c.Validate(&item); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	//指令集名称不可重复
	var itemExists []model.CommandSet
	err = commandSetRepository.DB.Where("name = ?", item.Name).Find(&itemExists).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}
	if 0 != len(itemExists) {
		return FailWithDataOperate(c, 422, "指令集已存在", "指令集-新增: "+item.Name+"已存在", nil)
	}
	//新增数据
	item.ID = utils.UUID()
	item.Created = utils.NowJsonTime()
	if err := commandSetRepository.Create(&item); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}
	return SuccessWithOperate(c, "指令集-新增: 新增指令集名称["+item.Name+"]", nil)
}

// CommandSetUpdateEndpoint 指令集更新
func CommandSetUpdateEndpoint(c echo.Context) (err error) {
	id := c.Param("id")
	commandSetInfo, err := commandSetRepository.FindById(id)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	//绑定数据
	var item model.CommandSet
	if err := c.Bind(&item); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if err := c.Validate(&item); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	//指令集名称不可重复
	var itemExists []model.CommandSet
	err = commandSetRepository.DB.Where("name = ? and id != ?", item.Name, id).Find(&itemExists).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if 0 != len(itemExists) {
		log.Errorf("Duplicate data names Error: %v", err)
		return FailWithDataOperate(c, 403, "指令集已存在", "修改指令集: 指令集名称["+item.Name+"]已存在", nil)
	}

	// 更新空描述信息
	if item.Description == "" {
		item.Description = " "
	}

	//更新数据
	if err := commandSetRepository.UpdateById(&item, item.ID); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	return SuccessWithOperate(c, "指令集-修改: 指令集名称["+commandSetInfo.Name+"->"+item.Name+"]", nil)
}

// CommandSetDeleteEndpoint 指令集删除
func CommandSetDeleteEndpoint(c echo.Context) (err error) {
	id := c.Param("id")
	var successDeleteCommandSet string
	var successCount int
	splits := strings.Split(id, ",")
	for i := range splits {
		if "" == splits[i] {
			continue
		}
		// 查看id对应的指令集是否存在
		commandSetInfo, err := commandSetRepository.FindById(splits[i])
		if nil != err {
			log.Errorf("DB Error: %v", err)
			continue
		}
		if commandSetInfo.Name == "非法指令集" || commandSetInfo.Name == "高危指令集" || commandSetInfo.Name == "敏感指令集" {
			return FailWithDataOperate(c, 403, "不允许删除默认指令集", "已删除指令集["+successDeleteCommandSet+"]"+"删除成功数["+strconv.Itoa(successCount)+"]"+"失败原因:不允许删除默认指令集["+commandSetInfo.Name+"]", err)
		}

		// 删除该指令集
		if err := commandSetRepository.DeleteById(splits[i]); err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "删除失败", "", err)
		}
		// 删除该指令集下的指令、
		if err := commandContentRepository.DeleteByContentId(splits[i]); err != nil {
			log.Errorf("DeleteByContentId Error: %v", err)
		}
		// 与该指令集有关的关联
		if err := commandRelevanceRepository.DeleteBySetId(splits[i]); err != nil {
			log.Errorf("DeleteBySetId Error: %v", err)
		}
		successDeleteCommandSet += commandSetInfo.Name + ","
		successCount++
	}
	if len(successDeleteCommandSet) > 0 {
		successDeleteCommandSet = successDeleteCommandSet[:len(successDeleteCommandSet)-1]
	}
	return SuccessWithOperate(c, "指令集-删除: 指令集名称["+successDeleteCommandSet+"]"+" 删除成功数["+strconv.Itoa(successCount)+"]", nil)
}

// CommandSetPagingEndpoint 指令集查询
func CommandSetPagingEndpoint(c echo.Context) error {
	pageIndex, _ := strconv.Atoi(c.QueryParam("pageIndex"))
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))
	auto := c.QueryParam("auto")
	name := c.QueryParam("name")
	level := c.QueryParam("level")
	description := c.QueryParam("description")

	items, _, err := commandSetRepository.FindByLimitingConditions(pageIndex, pageSize, auto, name, level, description)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	for i := range items {
		var str string
		content, err := commandContentRepository.FindByContentId(items[i].ID)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", err)
		}
		for j := range content {
			str += content[j].Content + ";\n"
		}
		if "" != str {
			str = str[:len(str)-1]
		}
		items[i].Content = str
	}
	return Success(c, items)
}

// CommandContentPagingEndpoint 指令集内容查询
func CommandContentPagingEndpoint(c echo.Context) error {
	contentId := c.QueryParam("contentId")
	auto := c.QueryParam("auto")
	content := c.QueryParam("content")
	isRegular := c.QueryParam("isRegular")
	items, _, err := commandContentRepository.FindByLimitingConditions(contentId, auto, content, isRegular)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	return Success(c, items)
}

// CommandContentCreateEndpoint 指令集内容添加
func CommandContentCreateEndpoint(c echo.Context) error {
	contentId := c.QueryParam("contentId")
	//绑定数据
	var item model.CommandContent
	if err := c.Bind(&item); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "绑定失败", "", err)
	}

	if err := c.Validate(&item); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	item.ID = utils.UUID()
	item.ContentId = contentId
	// 指令集内容不可重复
	var itemExists []model.CommandContent
	err := commandContentRepository.DB.Where("content_id = ? and content = ?", item.ContentId, item.Content).Find(&itemExists).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "添加失败", "", err)
	}
	if 0 != len(itemExists) {
		return FailWithDataOperate(c, 403, "命令已存在", "指令集管理-添加: 添加指令内容["+item.Content+"],失败原因: 内容已存在", err)
	}
	//添加数据
	if err := commandContentRepository.Create(&item); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "添加失败", "", err)
	}
	return SuccessWithOperate(c, "指令集管理-添加: 指令内容["+item.Content+"]", nil)
}

// CommandContentUpdateEndpoint 指令集内容修改
func CommandContentUpdateEndpoint(c echo.Context) error {
	contentId := c.QueryParam("contentId")
	editId := c.QueryParam("id")

	//绑定数据
	var item model.CommandContent
	if err := c.Bind(&item); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "绑定失败", "", err)
	}

	if err := c.Validate(&item); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	//指令内容不可重复
	var itemExists []model.CommandContent
	if err := commandContentRepository.DB.Where("content_id = ?", contentId).Find(&itemExists).Error; err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	for i := range itemExists {
		if itemExists[i].Content == item.Content && itemExists[i].ID != editId {
			return FailWithDataOperate(c, 403, "命令已存在", "指令集管理-修改: 指令控制修改指令集内容["+item.Content+"],失败原因: 内容已存在", nil)
		}
	}

	// 更新空描述信息
	if item.Description == "" {
		item.Description = " "
	}

	//更新数据
	if err := commandContentRepository.UpdateById(&item, editId); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	return SuccessWithOperate(c, "指令集管理-修改: 指令控制修改指令集内容["+item.Content+"]", nil)
}

// CommandContentDeleteEndpoint 指令集内容删除
func CommandContentDeleteEndpoint(c echo.Context) error {
	deleteId := c.QueryParam("id")
	var successDelete string
	var successCount int
	splits := strings.Split(deleteId, ",")
	for i := range splits {
		if "" == splits[i] {
			continue
		}
		// 查找该数据是否存在
		item, err := commandContentRepository.FindById(splits[i])
		if nil != err {
			log.Errorf("DB Error: %v", err)
			continue
		}

		if err := commandContentRepository.DeleteById(splits[i]); err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 403, "删除失败", "指令集管理-删除: 删除指令内容["+successDelete+"],删除成功数["+strconv.Itoa(successCount)+"]"+"删除失败内容:["+item.Content+"]", err)
		}
		successDelete += item.Content + ","
		successCount++
	}
	if len(successDelete) > 0 {
		successDelete = successDelete[:len(successDelete)-1]
	}
	return SuccessWithOperate(c, "指令集管理-删除: 删除指令内容["+successDelete+"],删除成功数["+strconv.Itoa(successCount)+"]", nil)
}

// CommandStrategyContentPagingEndpoint 直接关联的指令内容查询
func CommandStrategyContentPagingEndpoint(c echo.Context) error {
	id := c.Param("id")
	content, err := commandContentRepository.FindByStrategyId(id)
	if err != nil {
		log.Errorf("FindByContentId Error: %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", err)
	}
	return SuccessWithOperate(c, "", content)
}

// CommandStrategyContentUpdateEndpoint 直接关联的指令内容添加和修改
func CommandStrategyContentUpdateEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	cmd := c.QueryParam("cmd")
	oldContent, err := commandContentRepository.FindByStrategyId(id)
	if err != nil {
		log.Errorf("FindByContentId Error: %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", err)
	}
	// 更新
	if err := commandContentRepository.UpdateByContendId(&model.CommandContent{
		Content: cmd,
	}, id); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	return SuccessWithOperate(c, "指令策略-关联指令: ["+oldContent.Content+"->"+cmd+"]", nil)
}

// CommandStrategySetPagingEndpoint 指令策略关联指令集查询指令集
func CommandStrategySetPagingEndpoint(c echo.Context) error {
	commandSet, err := commandSetRepository.FindAll()
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	for i := range commandSet {
		commandSet[i].Name = commandSet[i].Name + "[" + commandSet[i].Level + "集]"
	}
	return SuccessWithOperate(c, "", commandSet)
}

// CommandStrategyRelateSetEndpoint 指令策略关联指令集查询已关联指令集
func CommandStrategyRelateSetEndpoint(c echo.Context) error {
	id := c.Param("id")
	_, err := commandStrategyRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	// 根据策略id找到所有的关联关系
	relevance, err := commandRelevanceRepository.FindByCommandStrategyId(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	// 声明一个指令集数组存放已关联的指令集
	var commandSet []model.CommandSet
	for i := range relevance {
		if relevance[i].CommandSetId == "-" {
			continue
		}
		// 根据指令集id找到指令集
		set, err := commandSetRepository.FindById(relevance[i].CommandSetId)
		if err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", err)
		}
		set.Name = set.Name + "[" + set.Level + "集]"
		commandSet = append(commandSet, set)
	}
	return SuccessWithOperate(c, "", commandSet)
}

// CommandStrategyRelateSetUpdateEndpoint 指令策略关联指令集的添加或删除
func CommandStrategyRelateSetUpdateEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	commandSetId := c.QueryParam("commandSetId")
	// 根据id找到策略
	strategyInfo, err := commandStrategyRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	// 删除所有关联的指令集
	if err := commandRelevanceRepository.DeleteByCommandStrategyIdAndCommandSetIdIsNotNull(id); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	splits := strings.Split(commandSetId, ",")
	for i := range splits {
		if splits[i] == "-" {
			continue
		}
		if err := commandRelevanceRepository.Create(&model.CommandRelevance{
			ID:                utils.UUID(),
			CommandStrategyId: id,
			CommandSetId:      splits[i],
		}); err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
	}
	return SuccessWithOperate(c, "指令策略-关联指令集: 名称["+strategyInfo.Name+"]", nil)

}

// CommandStrategyAllAssetEndpoint 新建时获取所有可关联的主机
func CommandStrategyAllAssetEndpoint(c echo.Context) error {
	// 获取当前账户
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, -1, "当前用户信息已过期，请重新登录")
	}
	// 获取当前账户下的所有主机
	var depIds []int64
	err := GetChildDepIds(account.DepartmentId, &depIds)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	// 策略所属部门及下级部门包含的所有设备账号(包括已被选择设备账号)
	assetAllArr, err := newAssetRepository.GetPassportByDepartmentIds(context.TODO(), depIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	assetList := make([]model.PassPort, 0)
	for i := range assetAllArr {
		if assetAllArr[i].Protocol == constant.SSH {
			depChinaName, err := DepChainName(assetAllArr[i].DepartmentId)
			if nil != err {
				log.Errorf("DepChainName Error: %v", err)
				return FailWithDataOperate(c, 500, "查询失败", "", nil)
			}
			assetAllArr[i].Name = assetAllArr[i].AssetName + "[" + assetAllArr[i].Ip + "]" + "[" + assetAllArr[i].Passport + "]" + "[" + assetAllArr[i].Protocol + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
			assetList = append(assetList, assetAllArr[i])
		}
	}
	return SuccessWithOperate(c, "", assetList)
}

// CommandStrategyAssetPagingEndpoint 指令策略关联资产查询所有资产
func CommandStrategyAssetPagingEndpoint(c echo.Context) error {
	id := c.Param("id")
	strategyInfo, err := commandStrategyRepository.FindById(id)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	var depIds []int64
	err = GetChildDepIds(strategyInfo.DepartmentId, &depIds)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	// 策略所属部门及下级部门包含的所有设备账号(包括已被选择设备账号)
	assetAllArr, err := newAssetRepository.GetPassportByDepartmentIds(context.TODO(), depIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	var assetList []model.PassPort
	for i := range assetAllArr {
		if assetAllArr[i].Protocol != constant.SSH {
			continue
		}
		depChinaName, err := DepChainName(assetAllArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		assetAllArr[i].Name = assetAllArr[i].AssetName + "[" + assetAllArr[i].Ip + "]" + "[" + assetAllArr[i].Passport + "]" + "[" + assetAllArr[i].Protocol + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
		assetList = append(assetList, assetAllArr[i])
	}
	return SuccessWithOperate(c, "", assetList)
}

// CommandStrategyRelateAssetEndpoint 指令策略关联资产查询已关联资产
func CommandStrategyRelateAssetEndpoint(c echo.Context) error {
	id := c.Param("id")
	// 根据id找到策略
	assetGroupInfo, err := commandStrategyRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	// 根据策略id找到所有的关联关系
	relevance, err := commandRelevanceRepository.FindByCommandStrategyId(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}

	// 声明一个设备数组存放已关联的设备
	var assetArr []model.PassPort
	for i := range relevance {
		if relevance[i].AssetId == "-" {
			continue
		}
		// 根据关联关系中的设备账号id找到设备账号
		passport, err := newAssetRepository.GetPassPortByID(context.TODO(), relevance[i].AssetId)
		if err != nil {
			log.Errorf("DB Error: %v", err)
			_ = commandRelevanceRepository.DeleteByStrategyIdAndAssetId(relevance[i].CommandStrategyId, relevance[i].AssetId)
		}

		depChinaName, err := DepChainName(assetGroupInfo.DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		passport.Name = passport.AssetName + "[" + passport.Ip + "]" + "[" + passport.Name + "]" + "[" + passport.Protocol + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
		assetArr = append(assetArr, passport)
	}
	return SuccessWithOperate(c, "", assetArr)
}

// CommandStrategyRelateAssetUpdateEndpoint 指令策略关联资产添加或删除
func CommandStrategyRelateAssetUpdateEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	assetId := c.QueryParam("assetId")
	// 根据id找到策略
	strategyInfo, err := commandStrategyRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	// 删除所有关联的资产账号
	err = commandRelevanceRepository.DeleteByCommandStrategyIdAndAssetIdIsNotNull(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	splits := strings.Split(assetId, ",")
	for i := range splits {
		if splits[i] == "-" {
			continue
		}
		// 保存关联关系
		if err = commandRelevanceRepository.Create(&model.CommandRelevance{
			ID:                utils.UUID(),
			CommandStrategyId: id,
			AssetId:           splits[i],
		}); err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", err)
		}
	}
	return SuccessWithOperate(c, "指令策略-关联设备: 名称["+strategyInfo.Name+"]", nil)
}

// CommandStrategyAllAssetGroupEndpoint 新建时获取所有可关联的主机组
func CommandStrategyAllAssetGroupEndpoint(c echo.Context) error {
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, -1, "当前用户信息已过期，请重新登录")
	}
	var depIds []int64
	err := GetChildDepIds(account.DepartmentId, &depIds)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	// 策略所属部门及下级部门包含的所有设备组(包括已被选择设备组)
	assetGroupAllArr, err := newAssetGroupRepository.GetAssetGroupListByDepartmentIds(context.TODO(), depIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	for i := range assetGroupAllArr {
		depChinaName, err := DepChainName(assetGroupAllArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		assetGroupAllArr[i].Name = assetGroupAllArr[i].Name + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}

	return SuccessWithOperate(c, "", assetGroupAllArr)

}

// CommandStrategyAssetGroupPagingEndpoint 指令策略关联资产组查询所有资产组
func CommandStrategyAssetGroupPagingEndpoint(c echo.Context) error {
	id := c.Param("id")

	strategyInfo, err := commandStrategyRepository.FindById(id)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	var depIds []int64
	err = GetChildDepIds(strategyInfo.DepartmentId, &depIds)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	// 策略所属部门及下级部门包含的所有设备组(包括已被选择设备组)
	assetGroupAllArr, err := newAssetGroupRepository.GetAssetGroupListByDepartmentIds(context.TODO(), depIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	for i := range assetGroupAllArr {
		depChinaName, err := DepChainName(assetGroupAllArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		assetGroupAllArr[i].Name = assetGroupAllArr[i].Name + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}

	return SuccessWithOperate(c, "", assetGroupAllArr)
}

// CommandStrategyRelateAssetGroupEndpoint 指令策略关联资产组查询已关联的资产组
func CommandStrategyRelateAssetGroupEndpoint(c echo.Context) error {
	id := c.Param("id")
	// 根据id找到策略
	_, err := commandStrategyRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	// 查询所有关联的资产组
	assetGroupArr, err := commandRelevanceRepository.FindByCommandStrategyId(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	var newAssetGroupArr []model.NewAssetGroup
	for i := range assetGroupArr {
		if assetGroupArr[i].AssetGroupId == "-" {
			continue
		}
		assetGroupInfo, err := newAssetGroupRepository.GetAssetGroupById(context.TODO(), assetGroupArr[i].AssetGroupId)
		if err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", err)
		}
		depChinaName, err := DepChainName(assetGroupInfo.DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		assetGroupInfo.Name = assetGroupInfo.Name + "[" + depChinaName[:len(depChinaName)-1] + "]"
		newAssetGroupArr = append(newAssetGroupArr, assetGroupInfo)
	}
	return SuccessWithOperate(c, "", newAssetGroupArr)
}

// CommandStrategyRelateAssetGroupUpdateEndpoint 指令策略关联资产组添加或删除
func CommandStrategyRelateAssetGroupUpdateEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	assetGroupId := c.QueryParam("assetGroupId")
	// 根据id找到策略
	strategyInfo, err := commandStrategyRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	// 删除所有关联资产组
	if err := commandRelevanceRepository.DeleteByCommandStrategyIdAndAssetGroupIdIsNotNull(id); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	splits := strings.Split(assetGroupId, ",")
	for i := range splits {
		if splits[i] == "-" {
			continue
		}
		// 保存关联关系
		if err = commandRelevanceRepository.Create(&model.CommandRelevance{
			ID:                utils.UUID(),
			CommandStrategyId: id,
			AssetGroupId:      splits[i],
		}); err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", err)
		}
	}
	return SuccessWithOperate(c, "指令策略-关联资产组: 名称["+strategyInfo.Name+"]", nil)
}

// CommandStrategyAllUserEndpoint 指令策略关联用户查询所有用户
func CommandStrategyAllUserEndpoint(c echo.Context) error {
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, -1, "当前用户信息已过期，请重新登录")
	}
	var depIds []int64
	err := GetChildDepIds(account.DepartmentId, &depIds)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	// 策略所属部门及下级部门包含的所有用户(包括已被选择用户)
	userAllArr, err := userNewRepository.FindUserByDepartmentIdEnable(depIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	for i := range userAllArr {
		depChinaName, err := DepChainName(userAllArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		userAllArr[i].Username = userAllArr[i].Username + "[" + userAllArr[i].Nickname + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}

	return SuccessWithOperate(c, "", userAllArr)
}

// CommandStrategyUserPagingEndpoint 指令策略关联用户查询所有用户
func CommandStrategyUserPagingEndpoint(c echo.Context) error {
	id := c.Param("id")
	// 根据id找到策略
	strategyInfo, err := commandStrategyRepository.FindById(id)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	var depIds []int64
	err = GetChildDepIds(strategyInfo.DepartmentId, &depIds)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	// 策略所属部门及下级部门包含的所有用户(包括已被选择用户)
	userAllArr, err := userNewRepository.FindUserByDepartmentIdEnable(depIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	for i := range userAllArr {
		depChinaName, err := DepChainName(userAllArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		userAllArr[i].Username = userAllArr[i].Username + "[" + userAllArr[i].Nickname + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}

	return SuccessWithOperate(c, "", userAllArr)
}

// CommandStrategyRelateUserEndpoint 指令策略关联用户查询已关联的用户
func CommandStrategyRelateUserEndpoint(c echo.Context) error {
	id := c.Param("id")
	strategyInfo, err := userStrategyRepository.FindById(id)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	// 查询关联关系
	userRelateArr, err := commandRelevanceRepository.FindByCommandStrategyId(id)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	var userArr []model.UserNew
	for i := range userRelateArr {
		if userRelateArr[i].UserId == "-" {
			continue
		}
		user, err := userNewRepository.FindById(userRelateArr[i].UserId)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		depChinaName, err := DepChainName(strategyInfo.DepartmentId)
		user.Username = user.Username + "[" + user.Nickname + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
		userArr = append(userArr, user)
	}
	return SuccessWithOperate(c, "", userArr)
}

// CommandStrategyRelateUserUpdateEndpoint 指令策略关联用户更新
func CommandStrategyRelateUserUpdateEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	userId := c.QueryParam("userId")
	// 查询策略
	strategyInfo, err := commandStrategyRepository.FindById(id)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	// 删除原有关联关系
	if err = commandRelevanceRepository.DeleteByCommandStrategyIdAndUserIdIsNotNull(id); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	// 新增关联关系
	splits := strings.Split(userId, ",")
	for i := range splits {
		if err = commandRelevanceRepository.Create(&model.CommandRelevance{
			ID:                utils.UUID(),
			CommandStrategyId: id,
			UserId:            splits[i],
		}); err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
	}
	return SuccessWithOperate(c, "指令策略-关联用户: 名称["+strategyInfo.Name+"]", nil)
}

// CommandStrategyAllUserGroupEndpoint 指令策略关联用户组查询所有用户组
func CommandStrategyAllUserGroupEndpoint(c echo.Context) error {
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, -1, "当前用户信息已过期，请重新登录")
	}
	var depIds []int64
	err := GetChildDepIds(account.DepartmentId, &depIds)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	// 策略所属部门及下级部门包含的所有用户组(包括已被选择用户组)
	userGroupAllArr, err := userGroupNewRepository.FindUserGroupByDepartmentId(depIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	for i := range userGroupAllArr {
		depChinaName, err := DepChainName(userGroupAllArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		userGroupAllArr[i].Name = userGroupAllArr[i].Name + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}
	return SuccessWithOperate(c, "", userGroupAllArr)
}

// CommandStrategyUserGroupPagingEndpoint 指令策略关联用户查询所有用户组
func CommandStrategyUserGroupPagingEndpoint(c echo.Context) error {
	id := c.Param("id")

	strategyInfo, err := commandStrategyRepository.FindById(id)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	var depIds []int64
	err = GetChildDepIds(strategyInfo.DepartmentId, &depIds)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	// 策略所属部门及下级部门包含的所有用户组(包括已被选择用户组)
	userGroupAllArr, err := userGroupNewRepository.FindUserGroupByDepartmentId(depIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	for i := range userGroupAllArr {
		depChinaName, err := DepChainName(userGroupAllArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		userGroupAllArr[i].Name = userGroupAllArr[i].Name + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}

	return SuccessWithOperate(c, "", userGroupAllArr)
}

// CommandStrategyRelateUserGroupEndpoint 指令策略关联用户组查询已关联的用户组
func CommandStrategyRelateUserGroupEndpoint(c echo.Context) error {
	id := c.Param("id")
	_, err := commandStrategyRepository.FindById(id)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	// 查询关联关系
	userGroupRelateArr, err := commandRelevanceRepository.FindByCommandStrategyId(id)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	var userGroupArr []model.UserGroupNew

	for i := range userGroupRelateArr {
		if userGroupRelateArr[i].UserGroupId == "-" {
			continue
		}
		userGroup, err := userGroupNewRepository.FindById(userGroupRelateArr[i].UserGroupId)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		depChinaName, err := DepChainName(userGroup.DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		userGroup.Name = userGroup.Name + "[" + depChinaName[:len(depChinaName)-1] + "]"
		userGroupArr = append(userGroupArr, userGroup)
	}
	return SuccessWithOperate(c, "", userGroupArr)
}

// CommandStrategyRelateUserGroupUpdateEndpoint 指令策略关联用户组更新
func CommandStrategyRelateUserGroupUpdateEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	userGroupId := c.QueryParam("userGroupId")
	strategyInfo, err := commandStrategyRepository.FindById(id)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	// 删除原有关联关系
	if err = commandRelevanceRepository.DeleteByCommandStrategyIdAndUserGroupIdIsNotNull(id); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	// 新增关联关系
	splits := strings.Split(userGroupId, ",")
	for i := range splits {
		if err = commandRelevanceRepository.Create(&model.CommandRelevance{
			ID:                utils.UUID(),
			CommandStrategyId: id,
			UserGroupId:       splits[i],
		}); err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
	}
	return SuccessWithOperate(c, "指令策略-关联用户组: 名称["+strategyInfo.Name+"]", nil)
}

// 指令策略配置
// 获取指令策略配置
func CommandStrategyConfigEndpoint(c echo.Context) error {
	var commandPolicyConfig model.CommandPolicyConfig
	commandSettingMap, err := propertyRepository.FindMapByNames([]string{"approval-timeout", "expired-action"})
	if err != nil {
		log.Errorf("WorkOrderSetting FindById err: %v", err)
		return FailWithDataOperate(c, 500, "查找失败", "", nil)
	}
	commandPolicyConfig.ApprovalTimeout, err = strconv.Atoi(commandSettingMap["approval-timeout"])
	if err != nil {
		log.Errorf("WorkOrderSetting FindById err: %v", err)
		commandPolicyConfig.ApprovalTimeout = 3
	}
	commandPolicyConfig.ExpiredAction = commandSettingMap["expired-action"]
	return SuccessWithOperate(c, "", commandPolicyConfig)
}

// 修改策略配置
func CommandStrategyConfigUpdateEndpoint(c echo.Context) error {
	var commandPolicyConfig model.CommandPolicyConfig
	if err := c.Bind(&commandPolicyConfig); err != nil {
		log.Errorf("CommandStrategyConfigUpdateEndpoint Bind err: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if err := propertyRepository.UpdateByName(&model.Property{
		Name:  "approval-timeout",
		Value: strconv.Itoa(commandPolicyConfig.ApprovalTimeout),
	}, "approval-timeout"); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	var expiredAction string
	if commandPolicyConfig.ExpiredAction != "session-deny" {
		expiredAction = "command-deny"
	} else {
		expiredAction = "session-deny"
	}
	if err := propertyRepository.UpdateByName(&model.Property{
		Name:  "expired-action",
		Value: expiredAction,
	}, "expired-action"); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	return SuccessWithOperate(c, "指令审批策略-修改: 系统配置策略配置改动,审批超时时间["+strconv.Itoa(commandPolicyConfig.ApprovalTimeout)+"], 过期执行动作["+commandPolicyConfig.ExpiredAction+"]", nil)
}

// 获取指令策略优先级
func CommandStrategyPriorityEndpoint(c echo.Context) error {
	var commandStrategyPriority model.CommandStrategyPriority
	commandSettingMap, err := propertyRepository.FindMapByNames([]string{"strategy-priority"})
	if err != nil {
		log.Errorf("WorkOrderSetting FindById err: %v", err)
		return FailWithDataOperate(c, 500, "查找失败", "", nil)
	}
	commandStrategyPriority.StrategyPriority = commandSettingMap["strategy-priority"]
	return SuccessWithOperate(c, "", commandStrategyPriority)
}

// 修改指令策略优先级
func CommandStrategyPriorityUpdateEndpoint(c echo.Context) error {
	var commandStrategyPriority model.CommandStrategyPriority
	if err := c.Bind(&commandStrategyPriority); err != nil {
		log.Errorf("CommandStrategyPriorityUpdateEndpoint Bind err: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if err := propertyRepository.UpdateByName(&model.Property{
		Name:  "strategy-priority",
		Value: commandStrategyPriority.StrategyPriority,
	}, "strategy-priority"); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	return SuccessWithOperate(c, "优先级策略配置-修改: 策略配置,优先级策略配置["+commandStrategyPriority.StrategyPriority+"]", nil)
}

// 通过指令策略id删除指令策略
func DeleteCommandStrategyById(commandStrategyId string) error {
	commandStrategy, err := commandStrategyRepository.FindById(commandStrategyId)
	if err != nil {
		log.Errorf("FindById Error: %v", err)
		return err
	}
	// 删除策略
	if err := commandStrategyRepository.DeleteById(commandStrategyId); err != nil {
		log.Errorf("DeleteByDepartmentId Error: %v", err)
	}
	// 删除与策略关联的表
	if err := commandRelevanceRepository.DeleteByStrategyId(commandStrategyId); err != nil {
		log.Errorf("DeleteByPolicyId Error: %v", err)
	}
	// 删除以策略直接关联的内容表
	if err := commandContentRepository.DeleteByContentId(commandStrategyId); err != nil {
		log.Errorf("DeleteByContentId Error: %v", err)
	}
	// 重新排序
	commandStrategyArr, err := commandStrategyRepository.FindByDepartmentId(commandStrategy.DepartmentId)
	if err != nil {
		log.Errorf("FindByDepartmentId Error: %v", err)

	}
	for j := range commandStrategyArr {
		commandStrategyArr[j].Priority = int64(j + 1)
		if err := commandStrategyRepository.UpdateById(&commandStrategyArr[j], commandStrategyArr[j].ID); err != nil {
			log.Errorf("UpdateById Error: %v", err)
		}
	}
	return nil
}

// 通过部门id删除指令策略
func DeleteCommandStrategyByDepartmentIds(departmentIds []int64) error {
	commandStrategyArr, err := commandStrategyRepository.FindByDepartmentIds(departmentIds)
	if err != nil {
		log.Errorf("FindByDepartmentIds Error: %v", err)
		return err
	}
	for i := range commandStrategyArr {
		if err := DeleteCommandStrategyById(commandStrategyArr[i].ID); err != nil {
			log.Errorf("DeleteCommandStrategyById Error: %v", err)
			return err
		}
	}
	return nil
}
