package api

import (
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
	"strconv"
	"strings"
	"time"
	"tkbastion/pkg/constant"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/log"
	"tkbastion/pkg/validator"
	"tkbastion/server/model"
	"tkbastion/server/utils"
)

// UserStrategyCreateEndpoint 新建策略
func UserStrategyCreateEndpoint(c echo.Context) error {
	var userStrategyDTO model.UserStrategyDTO
	if err := c.Bind(&userStrategyDTO); err != nil {
		log.Errorf("Bind Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}

	if err := c.Validate(&userStrategyDTO); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	item, err := userStrategyDTO.ToUserStrategy()
	if err != nil {
		log.Errorf("UserDto conv User Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}

	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 500, "用户信息已失效，请重新登录")
	}

	priority, err := userStrategyRepository.CountByDepartmentId(account.DepartmentId)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}
	item.Priority = priority + 1

	// 用户名或昵称不可重复
	var itemExists []model.UserStrategy

	if err := userNewRepository.DB.Where("name = ?", item.Name).Find(&itemExists).Error; nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}
	if 0 != len(itemExists) {
		return FailWithDataOperate(c, 403, "策略名称已存在", "用户策略-新增: 策略名称["+item.Name+"],失败原因[策略名称已存在]", nil)
	}

	// 根据部门id获取部门深度
	depth, _ := DepLevel(account.DepartmentId)
	item.ID = utils.UUID()
	item.Created = utils.NowJsonTime()
	item.DepartmentId = account.DepartmentId
	item.DepartmentDepth = depth
	item.DepartmentName = account.DepartmentName
	item.Status = constant.Disable

	if err := userStrategyRepository.Creat(&item); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}
	// 添加策略关联的用户组
	if userStrategyDTO.UserGroupId != "" {
		split := strings.Split(userStrategyDTO.UserGroupId, ",")
		for i := range split {
			// 关联用户组
			if err := userStrategyRepository.DB.Table("user_strategy_user_group").Create(&model.UserStrategyUserGroup{
				ID:             utils.UUID(),
				UserGroupId:    split[i],
				UserStrategyId: item.ID,
			}).Error; nil != err {
				log.Errorf("DB Error: %v", err)
				return FailWithDataOperate(c, 500, "新增失败", "", err)
			}
		}
	}

	// 添加策略关联的用户
	if userStrategyDTO.UserId != "" {
		split := strings.Split(userStrategyDTO.UserId, ",")
		for i := range split {
			// 关联用户
			if err := userStrategyRepository.DB.Table("user_strategy_users").Create(&model.UserStrategyUsers{
				ID:             utils.UUID(),
				UserId:         split[i],
				UserStrategyId: item.ID,
			}).Error; nil != err {
				log.Errorf("DB Error: %v", err)
				return FailWithDataOperate(c, 500, "新增失败", "", err)
			}
		}
	}
	return SuccessWithOperate(c, "用户策略-新增: 策略名称["+item.Name+"]", nil)
}

// UserStrategyUpdateEndpoint 修改策略
func UserStrategyUpdateEndpoint(c echo.Context) error {
	id := c.Param("id")
	// 未修改前用户组信息
	oldUserStrategy, err := userStrategyRepository.FindById(id)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	var userStrategyDTO model.UserStrategyDTO
	if err := c.Bind(&userStrategyDTO); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if err := c.Validate(&userStrategyDTO); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	item, err := userStrategyDTO.ToUserStrategy()
	if err != nil {
		log.Errorf("UserDto conv User Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	// 用户名或昵称不可重复
	var itemExists []model.UserStrategy

	if err := userStrategyRepository.DB.Where("name = ? and id != ?", item.Name, item.ID).Find(&itemExists).Error; nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if 0 != len(itemExists) {
		return FailWithDataOperate(c, 403, "策略名称已存在", "用户策略-修改: 策略名称["+oldUserStrategy.Name+"->"+item.Name+"],失败原因["+item.Name+"已存在]", nil)
	}

	// 描述信息为空时更新描述信息
	if item.Description == "" {
		item.Description = " "
	}
	if item.IpLimitList == "" {
		item.IpLimitList = " "
	} else {
		item.IpLimitList = strings.TrimSpace(item.IpLimitList)
	}
	if oldUserStrategy.Priority != item.Priority {
		// 新优先级大于原优先级，此范围内数据优先级-1
		if item.Priority > oldUserStrategy.Priority {
			if err := userStrategyRepository.DB.Table("user_strategy").Where("priority > ? and priority <= ? and department_id = ? ", oldUserStrategy.Priority, item.Priority, oldUserStrategy.DepartmentId).Update("priority", gorm.Expr("priority - 1")).Error; nil != err {
				log.Errorf("DB Error: %v", err)
				return FailWithDataOperate(c, 500, "更新优先级失败", "", err)
			}
		}
		// 新优先级小玉于原优先级，此范围内数据优先级+1
		if item.Priority < oldUserStrategy.Priority {
			if err := userStrategyRepository.DB.Table("user_strategy").Where("priority >= ? and priority < ? and department_id = ? ", item.Priority, oldUserStrategy.Priority, oldUserStrategy.DepartmentId).Update("priority", gorm.Expr("priority + 1")).Error; nil != err {
				log.Errorf("DB Error: %v", err)
				return FailWithDataOperate(c, 500, "更新优先级失败", "", err)
			}
		}
	}
	priority, err := userStrategyRepository.CountByDepartmentId(oldUserStrategy.DepartmentId)
	if err != nil {
		log.Errorf("DB Error: %v", err)
	}
	if item.Priority >= priority {
		item.Priority = priority
	}
	//由已过期修改为可用时间内则修改状态为已禁用
	if oldUserStrategy.Status == constant.Expiration {
		if *item.IsPermanent || (time.Now().After(item.BeginValidTime.Time) && time.Now().Before(item.EndValidTime.Time)) {
			item.Status = constant.Disable
		}
	}
	// 更新策略
	if err := userStrategyRepository.UpdateById(&item, id); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	return SuccessWithOperate(c, "用户策略-修改: 策略名称["+oldUserStrategy.Name+"->"+item.Name+"]", item)
}

// UserStrategyDeleteEndpoint 删除策略
func UserStrategyDeleteEndpoint(c echo.Context) error {
	id := c.Param("id")
	split := strings.Split(id, ",")
	var departmentIds []int64
	var nameDelete string
	for i := range split {
		userStrategy, err := userStrategyRepository.FindById(split[i])
		if err != nil {
			log.Errorf("DB Error %v", err)
			continue
		}
		if err := DeleteUserStrategyById(split[i]); err != nil {
			continue
		}
		nameDelete += userStrategy.Name + ","
		departmentIds = append(departmentIds, userStrategy.DepartmentId)
	}
	return SuccessWithOperate(c, "用户策略-删除: 策略名称["+nameDelete+"]", nil)
}

// UserStrategyPagingEndpoint 查看所有策略
func UserStrategyPagingEndpoint(c echo.Context) error {
	pageIndex, _ := strconv.Atoi(c.QueryParam("pageIndex"))
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))
	auto := c.QueryParam("auto")
	department := c.QueryParam("department")
	name := c.QueryParam("name")
	description := c.QueryParam("description")
	status := c.QueryParam("status")
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 401, "您的登录信息已失效,请重新登录后再试.")
	}
	var departmentId []int64
	if err := GetChildDepIds(account.DepartmentId, &departmentId); err != nil {
		log.Errorf("GetChildDepIds Error: %v", err)
	}
	userStrategy, _, err := userStrategyRepository.FindByLimitingConditions(pageIndex, pageSize, auto, department, name, description, status, departmentId)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	userStrategyList := make([]model.UserStrategyDTO, 0)
	for i := range userStrategy {
		if *userStrategy[i].IsPermanent == false && (time.Now().After(userStrategy[i].EndValidTime.Time) || time.Now().Before(userStrategy[i].BeginValidTime.Time)) {
			userStrategy[i].Status = constant.Expiration
			_ = userStrategyRepository.UpdateById(&model.UserStrategy{Status: constant.Expiration}, userStrategy[i].ID)
		}
		dep, err := departmentRepository.FindById(userStrategy[i].DepartmentId)
		if err != nil {
			log.Errorf("FindById Error: %v", err)
			continue
		}
		userStrategy[i].DepartmentName = dep.Name
		// 更新策略的部门机构名称
		_ = userStrategyRepository.UpdateById(&model.UserStrategy{DepartmentName: dep.Name}, userStrategy[i].ID)
		userStrategyTemp, err := userStrategy[i].ToUserStrategyDTO()
		if err != nil {
			log.Errorf("ToUserStrategyDTO Error: %v", err)
			continue
		}
		userStrategyList = append(userStrategyList, *userStrategyTemp)
	}
	return Success(c, userStrategyList)
}

// UserStrategyEnableEndpoint 启用/禁用策略
func UserStrategyEnableEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	status := c.QueryParam("status")
	strategyInfo, err := userStrategyRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "启用/禁用失败", "", err)
	}
	if strategyInfo.Status == constant.Expiration {
		return Fail(c, 500, "策略已过期")
	}
	if status == constant.Enable {
		strategyInfo.Status = constant.Enable
	} else {
		strategyInfo.Status = constant.Disable
	}
	if err := userStrategyRepository.UpdateById(&strategyInfo, id); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "启用/禁用失败", "", err)
	}
	return SuccessWithOperate(c, "用户策略-启用/禁用: 策略["+strategyInfo.Name+strategyInfo.Status+"]", nil)
}

// UserStrategyGetUserEndpoint 直接关联时查看可关联用户
func UserStrategyGetUserEndpoint(c echo.Context) error {
	id := c.Param("id")
	userStrategy, err := userStrategyRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	// 根据策略的部门id
	userList, err := GetCurrentDepartmentUserChildren(userStrategy.DepartmentId)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	for i := range userList {
		depChinaName, err := DepChainName(userList[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		userList[i].Username = userList[i].Username + "[" + userList[i].Nickname + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}
	return Success(c, userList)
}

// UserStrategyRelatedUsersEndpoint 获取策略已关联的用户
func UserStrategyRelatedUsersEndpoint(c echo.Context) error {
	id := c.Param("id")
	userStrategy, err := userStrategyRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	var userStrategyUser []model.UserStrategyUsers
	if err := userStrategyRepository.DB.Table("user_strategy_users").Where("user_strategy_id = ?", userStrategy.ID).Find(&userStrategyUser).Error; nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	var userList []model.UserNew
	for i := range userStrategyUser {
		userNew, err := userNewRepository.FindById(userStrategyUser[i].UserId)
		if err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", err)
		}
		depChinaName, _ := DepChainName(userNew.DepartmentId)
		userNew.Username = userNew.Username + "[" + userNew.Nickname + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
		userList = append(userList, userNew)
	}
	return Success(c, userList)
}

// UserStrategyAddUserEndpoint 策略关联用户
func UserStrategyAddUserEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	userId := c.QueryParam("userId")
	// 根据id找到当前策略
	userStrategy, err := userStrategyRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	// 删除原有策略的用户
	if err := userStrategyRepository.DB.Table("user_strategy_users").Where("user_strategy_id = ?", userStrategy.ID).Delete(&model.UserStrategyUsers{}).Error; err != nil {
		log.Errorf("DB Error: %v", err)
	}
	// 添加新的策略用户
	if userId != "" {
		split := strings.Split(userId, ",")
		for i := range split {
			userStrategyUser := model.UserStrategyUsers{
				ID:             utils.UUID(),
				UserStrategyId: userStrategy.ID,
				UserId:         split[i],
			}
			if err := userStrategyRepository.DB.Table("user_strategy_users").Create(&userStrategyUser).Error; err != nil {
				log.Errorf("DB Error: %v", err)
				return FailWithDataOperate(c, 500, "添加失败", "", err)
			}
		}
	}
	return SuccessWithOperate(c, "用户策略-关联用户: 策略名称["+userStrategy.Name+"]", nil)
}

// GetCurrentDepartmentUserGroupChildEndpoint 获取所有用户组
func GetCurrentDepartmentUserGroupChildEndpoint(c echo.Context) error {
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 401, "您的登录信息已失效,请重新登录后再试.")
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

// UserStrategyGetUserGroupEndpoint 直接关联时查看可关联用户组
func UserStrategyGetUserGroupEndpoint(c echo.Context) error {
	id := c.Param("id")
	userStrategy, err := userStrategyRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	var depIds []int64
	err = GetChildDepIds(userStrategy.DepartmentId, &depIds)
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

	return Success(c, userGroupAllArr)

}

// UserStrategyRelatedGroupsEndpoint 获取已关联的用户组
func UserStrategyRelatedGroupsEndpoint(c echo.Context) error {
	id := c.Param("id")
	userStrategy, err := userStrategyRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	var userStrategyUserGroup []model.UserStrategyUserGroup
	if err := userStrategyRepository.DB.Table("user_strategy_user_group").Where("user_strategy_id = ?", userStrategy.ID).Find(&userStrategyUserGroup).Error; nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	var userGroupList []model.UserGroupNew
	for i := range userStrategyUserGroup {
		userGroup, err := userGroupNewRepository.FindById(userStrategyUserGroup[i].UserGroupId)
		if err != nil {
			log.Errorf("DB Error: %v", err)
			continue
		}
		depChinaName, err := DepChainName(userGroup.DepartmentId)
		userGroup.Name = userGroup.Name + "[" + depChinaName[:len(depChinaName)-1] + "]"
		userGroupList = append(userGroupList, userGroup)
	}
	return Success(c, userGroupList)
}

// UserStrategyAddUserGroupEndpoint 策略关联用户组
func UserStrategyAddUserGroupEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	userGroupId := c.QueryParam("userGroupId")
	// 根据id找到当前策略
	userStrategy, err := userStrategyRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	// 删除原有策略的用户组
	if err := userStrategyRepository.DB.Table("user_strategy_user_group").Where("user_strategy_id = ?", userStrategy.ID).Delete(&model.UserStrategyUserGroup{}).Error; err != nil {
		log.Errorf("DB Error: %v", err)
	}
	// 添加新的策略用户组
	split := strings.Split(userGroupId, ",")
	for i := range split {
		userStrategyUserGroup := model.UserStrategyUserGroup{
			ID:             utils.UUID(),
			UserStrategyId: userStrategy.ID,
			UserGroupId:    split[i],
		}
		if err := userStrategyRepository.DB.Table("user_strategy_user_group").Create(&userStrategyUserGroup).Error; err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "添加失败", "", err)
		}
	}
	return SuccessWithOperate(c, "用户策略-关联用户组: 策略名称["+userStrategy.Name+"]", nil)
}

// GetCurrentDepartmentUserStrategyChild 获取某部门下所有策略，参数传当前的部门id
func GetCurrentDepartmentUserStrategyChild(departmentId int64) (userStrategy []model.UserStrategy, err error) {
	var departmentIds []int64
	if err := GetChildDepIds(departmentId, &departmentIds); err != nil {
		log.Errorf("GetChildDepIds Error: %v", err)
	}
	for i := range departmentIds {
		userStrategyTemp, err := userStrategyRepository.FindByDepartmentId(departmentIds[i])
		if err != nil {
			log.Errorf("FindByDepartmentId Error : %v", err)
		}
		userStrategy = append(userStrategy, userStrategyTemp...)
	}
	return userStrategy, nil
}

// DeleteUserStrategyChildByDepartmentIds 删除某部门下所有用户策略，参数传部门id
func DeleteUserStrategyChildByDepartmentIds(departmentId []int64) (err error) {
	for _, v := range departmentId {
		userStrategy, err := GetCurrentDepartmentUserStrategyChild(v)
		if err != nil {
			log.Errorf("GetCurrentDepartmentUserChildren Error :%v", err)
		}
		for i := range userStrategy {
			if err := DeleteUserStrategyById(userStrategy[i].ID); err != nil {
				log.Errorf("DeleteUserById Error :%v", err)
			}
		}
	}
	return nil
}

// DeleteUserStrategyById 通过用户策略id删除用户策略
func DeleteUserStrategyById(userStrategyId string) error {
	userStrategy, err := userStrategyRepository.FindById(userStrategyId)
	if err != nil {
		log.Errorf("DB Error %v", err)
		return nil
	}
	if err := userStrategyRepository.DeleteById(userStrategyId); err != nil {
		log.Errorf("DB Error: %v", err)
		return err
	}
	// 删除与该策略有关的关联关系
	if err := userStrategyRepository.DB.Table("user_strategy_users").Where("user_strategy_id = ?", userStrategyId).Delete(&model.UserStrategyUsers{}).Error; nil != err {
		log.Errorf("DB Error: %v", err)
	}
	if err := userStrategyRepository.DB.Table("user_strategy_user_group").Where("user_strategy_id = ?", userStrategyId).Delete(&model.UserStrategyUserGroup{}).Error; nil != err {
		log.Errorf("DB Error: %v", err)
	}

	strategyArr, err := userStrategyRepository.FindByDepartmentId(userStrategy.DepartmentId)
	if err != nil {
		log.Errorf("DB Error %v", err)
	}
	for j := range strategyArr {
		strategyArr[j].Priority = int64(j + 1)
		if err := userStrategyRepository.UpdateById(&strategyArr[j], strategyArr[j].ID); err != nil {
			log.Errorf("DB Error: %v", err)
		}
	}
	return nil
}
