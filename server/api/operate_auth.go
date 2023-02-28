package api

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/log"
	"tkbastion/pkg/validator"
	"tkbastion/server/utils"

	"gorm.io/gorm"

	"tkbastion/server/model"

	"github.com/labstack/echo/v4"
)

func OperateAuthCreateEndpoint(c echo.Context) error {
	var operateAuthDto model.OperateAuthDTO
	if err := c.Bind(&operateAuthDto); nil != err {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}

	// 数据校验
	if err := c.Validate(&operateAuthDto); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	item, err := operateAuthDto.ToOperateAuth()
	if err != nil {
		log.Errorf("ToOperateAuth Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}

	account, isSuccess := GetCurrentAccountNew(c)
	if !isSuccess {
		log.Error("GetCurrentAccountNew Error")
		return FailWithDataOperate(c, 500, "新增失败", "", nil)
	}
	// 策略名称不能重复
	_, err = operateAuthRepository.FindByNameDepId(item.Name, account.DepartmentId)
	if nil != err && gorm.ErrRecordNotFound != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", nil)
	}
	if gorm.ErrRecordNotFound != err {
		return FailWithDataOperate(c, 400, "策略名称重复", "运维授权-新增: 策略名称["+item.Name+"], 失败原因[策略名称重复]", nil)
	}

	item.DepartmentId = account.DepartmentId
	item.DepartmentName = account.DepartmentName
	depLevel, err := DepLevel(account.DepartmentId)
	if nil != err {
		log.Errorf("DepLevel Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", nil)
	}
	item.DepLevel = depLevel

	depStrategyCount, err := operateAuthRepository.StrategyCountByDepId(account.DepartmentId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", nil)
	}
	item.Priority = int(depStrategyCount) + 1

	err = operateAuthRepository.Create(&item)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", nil)
	}

	// 新增运维授权策略时同步新增资产权限报表数据
	err = createAssetAuthReportFrom(account, item)
	if nil != err {
		log.Errorf("新增运维授权策略时, 同步新增资产权限报表数据失败: %v, 策略名称: %v, 所在部门: %v", err.Error(), item.Name, account.DepartmentName)
	}

	// 新增运维授权策略时同步新增应用权限报表数据
	err = createAppAuthReportFrom(account, item)
	if nil != err {
		log.Errorf("新增运维授权策略时, 同步新增应用权限报表数据失败: %v, 策略名称: %v, 所在部门: %v", err.Error(), item.Name, account.DepartmentName)
	}

	return SuccessWithOperate(c, "运维授权-新增: 策略名称["+item.Name+"]", item)
}

// 资产授权报表 ---->

func createAssetAuthReportFrom(account model.UserNew, item model.OperateAuth) error {
	operateAuth, err := operateAuthRepository.FindByNameDepId(item.Name, account.DepartmentId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return err
	}
	var userIdArr []string
	var assetAccountIdArr []string

	userIdStrArr := strings.Split(operateAuth.RelateUser, ",")
	for i := range userIdStrArr {
		if "" == userIdStrArr[i] {
			continue
		}

		userIdArr = append(userIdArr, userIdStrArr[i])
	}

	userGroupIdStrArr := strings.Split(operateAuth.RelateUserGroup, ",")
	for j := range userGroupIdStrArr {
		if "" == userGroupIdStrArr[j] {
			continue
		}

		userArr, err := GetUserByUserGroupId(userGroupIdStrArr[j])
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return err
		}
		for k := range userArr {
			userIdArr = append(userIdArr, userArr[k].ID)
		}
	}

	assetAccountStrArr := strings.Split(operateAuth.RelateAsset, ",")
	for i := range assetAccountStrArr {
		if "" == assetAccountStrArr[i] {
			continue
		}

		assetAccountIdArr = append(assetAccountIdArr, assetAccountStrArr[i])
	}

	assetAccountGroupIdStrArr := strings.Split(operateAuth.RelateAssetGroup, ",")
	for j := range assetAccountGroupIdStrArr {
		if "" == assetAccountGroupIdStrArr[j] {
			continue
		}

		assetAccountArr, err := newAssetGroupRepository.GetPassportByAssetGroupId(context.TODO(), assetAccountGroupIdStrArr[j])
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return err
		}
		for k := range assetAccountArr {
			assetAccountIdArr = append(assetAccountIdArr, assetAccountArr[k].ID)
		}
	}

	userIdArr = StrRemoveDuplicates(userIdArr)
	assetAccountIdArr = StrRemoveDuplicates(assetAccountIdArr)

	for i := range userIdArr {
		for j := range assetAccountIdArr {
			var assetAuthReportForm model.AssetAuthReportForm
			assetAuthReportForm.UserId = userIdArr[i]
			assetAuthReportForm.AssetAccountId = assetAccountIdArr[j]
			assetAuthReportForm.OperateAuthId = operateAuth.ID

			err = assetAuthReportFormRepository.Create(&assetAuthReportForm)
			if nil != err {
				log.Errorf("DB Error: %v", err.Error())
			}
		}
	}

	return nil
}

func updateAssetAuthReportFrom(account model.UserNew, operateAuthId int64) error {
	err := deleteAssetAuthReportFrom(operateAuthId)
	if nil != err {
		log.Errorf("DB Error: %v", err.Error())
		return err
	}

	operateAuth, err := operateAuthRepository.FindById(operateAuthId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return err
	}

	err = createAssetAuthReportFrom(account, operateAuth)
	if nil != err {
		log.Errorf("DB Error: %v", err.Error())
		return err
	}

	return nil
}

func deleteAssetAuthReportFrom(operateAuthId int64) error {
	err := assetAuthReportFormRepository.DeleteByOperateAuthId(operateAuthId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return err
	}
	return nil
}

// 资产授权报表 <------

// 应用授权报表 ---->

func createAppAuthReportFrom(account model.UserNew, item model.OperateAuth) error {
	operateAuth, err := operateAuthRepository.FindByNameDepId(item.Name, account.DepartmentId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return err
	}
	var userIdArr []string
	var appIdArr []string

	userIdStrArr := strings.Split(operateAuth.RelateUser, ",")
	for i := range userIdStrArr {
		if "" == userIdStrArr[i] {
			continue
		}

		userIdArr = append(userIdArr, userIdStrArr[i])
	}

	userGroupIdStrArr := strings.Split(operateAuth.RelateUserGroup, ",")
	for j := range userGroupIdStrArr {
		if "" == userGroupIdStrArr[j] {
			continue
		}

		userArr, err := GetUserByUserGroupId(userGroupIdStrArr[j])
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return err
		}
		for k := range userArr {
			userIdArr = append(userIdArr, userArr[k].ID)
		}
	}

	assetAccountStrArr := strings.Split(operateAuth.RelateApp, ",")
	for i := range assetAccountStrArr {
		if "" == assetAccountStrArr[i] {
			continue
		}

		appIdArr = append(appIdArr, assetAccountStrArr[i])
	}

	userIdArr = StrRemoveDuplicates(userIdArr)
	appIdArr = StrRemoveDuplicates(appIdArr)

	for i := range userIdArr {
		for j := range appIdArr {
			var applicationAuthReportForm model.ApplicationAuthReportForm
			applicationAuthReportForm.UserId = userIdArr[i]
			applicationAuthReportForm.ApplicationId = appIdArr[j]
			applicationAuthReportForm.OperateAuthId = operateAuth.ID

			err = appAuthReportFormRepository.Create(context.TODO(), &applicationAuthReportForm)
			if nil != err {
				log.Errorf("DB Error: %v", err.Error())
			}
		}
	}

	return nil
}

func updateAppAuthReportFrom(account model.UserNew, operateAuthId int64) error {
	err := deleteAppAuthReportFrom(operateAuthId)
	if nil != err {
		log.Errorf("DB Error: %v", err.Error())
		return err
	}

	operateAuth, err := operateAuthRepository.FindById(operateAuthId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return err
	}

	err = createAppAuthReportFrom(account, operateAuth)
	if nil != err {
		log.Errorf("DB Error: %v", err.Error())
		return err
	}

	return nil
}

func deleteAppAuthReportFrom(operateAuthId int64) error {
	err := appAuthReportFormRepository.DeleteByOperateAuthId(context.TODO(), operateAuthId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return err
	}
	return nil
}

// 应用授权报表 <------

func StrRemoveDuplicates(needRemoveDuplicatesArr []string) (removeDuplicatesArr []string) {
	for i := range needRemoveDuplicatesArr {
		if !IsBelong(needRemoveDuplicatesArr[i], removeDuplicatesArr) {
			removeDuplicatesArr = append(removeDuplicatesArr, needRemoveDuplicatesArr[i])
		}
	}
	return
}

func IsBelong(str string, strArr []string) bool {
	isBelong := false
	for i := range strArr {
		if str == strArr[i] {
			isBelong = true
			break
		}
	}
	return isBelong
}

func OperateAuthPagingEndpoint(c echo.Context) error {
	account, _ := GetCurrentAccountNew(c)
	var depIdArr []int64
	err := GetChildDepIds(account.DepartmentId, &depIdArr)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	// 因各部门的架构调整，导致原有数据表中存储的部门深度数据、部门名称数据已经不准确，需先进行一次更新
	err = UpdateDepInfoByDepIds(depIdArr)
	if nil != err {
		return SuccessWithOperate(c, "", nil)
	}

	var operateAuthArr []model.OperateAuth
	auto := c.QueryParam("auto")
	policyName := c.QueryParam("policy_name")
	departmentName := c.QueryParam("department_name")
	describe := c.QueryParam("describe")
	state := c.QueryParam("state")

	if "" == auto && "" == policyName && "" == departmentName && "" == describe && "" == state {
		operateAuthArr, err = operateAuthRepository.FindByDepIdsAndSort(depIdArr)
		if nil != err {
			if gorm.ErrRecordNotFound == err {
				return SuccessWithOperate(c, "", nil)
			}
			log.Errorf("DB Error: %v", err)
			return SuccessWithOperate(c, "", nil)
		}
	} else {
		// 这里存在一个问题
		// 例如搜索的是"启", 我们搜索出来的数据是那一刻数据库中已启用的, 但在后面的逻辑里会根据有效期去更新状态, 也即意味着, 返回页面时, 某条策略的状态已经变为了已过期
		// 一般是进到运维授权页面再搜索. 在进到页面时已经实时更新了一次策略有效期, 因此所以这种情况只有极其特殊情况会发生
		// 如果在进入运维授权页面后过了一段时间再搜索, 也即意外着在这段时间内, 有的策略的有效期状态实际已经发生变化, 但数据库未改变
		// 此时就会产生上述问题, 但这种情况只要再点击一次搜索(上一次搜索时已经更新了库内策略状态值), 这次搜索就会显示正确的数据
		operateAuthArr, err = findOperateAuthWithCondition(depIdArr, auto, policyName, departmentName, describe, state)
		if nil != err {
			if gorm.ErrRecordNotFound == err {
				return SuccessWithOperate(c, "", nil)
			}
			log.Errorf("findOperateAuthWithCondition Error: %v", err)
			return SuccessWithOperate(c, "", nil)
		}
	}

	var operateAuthDtoArr []model.OperateAuthDTO
	for i := range operateAuthArr {
		if operateAuthArr[i].StrategyTimeFlag {
			operateAuthArr[i].State = operateAuthArr[i].ButtonState
		} else {
			now := utils.NowJsonTime()
			if now.After(operateAuthArr[i].StrategyEndTime.Time) || now.Before(operateAuthArr[i].StrategyBeginTime.Time) {
				operateAuthArr[i].State = "overdue"
			} else {
				operateAuthArr[i].State = operateAuthArr[i].ButtonState
			}
		}
		err = operateAuthRepository.UpdateById(operateAuthArr[i].ID, &operateAuthArr[i])
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return SuccessWithOperate(c, "", nil)
		}

		operateAuthDto, err := operateAuthArr[i].ToOperateAuthDto()
		if nil != err {
			log.Errorf("ToOperateAuthDto Error: %v", err)
			return SuccessWithOperate(c, "", nil)
		}
		operateAuthDtoArr = append(operateAuthDtoArr, *operateAuthDto)
	}

	return SuccessWithOperate(c, "", operateAuthDtoArr)
}

func findOperateAuthWithCondition(depIdArr []int64, auto, policyName, departmentName, describe, state string) (operateAuthArr []model.OperateAuth, err error) {
	var stateWhereCondition, whereCondition string
	depIdArrStr := "("
	for i := range depIdArr {
		depIdArrStr = depIdArrStr + strconv.Itoa(int(depIdArr[i])) + ", "
	}
	depIdArrStr = depIdArrStr[:len(depIdArrStr)-2]
	depIdArrStr += ")"
	sql := "SELECT id, name, department_id, department_name, state, button_state,download,upload,watermark,description, relate_user, relate_user_group, relate_asset, relate_asset_group, relate_app, dep_level, priority, auth_expiration_date, strategy_begin_time, strategy_end_time, strategy_time_flag, ip_limit_type, ip_limit_list FROM operate_auth WHERE"
	orderBy := "ORDER BY dep_level ASC, department_name, priority ASC"
	if "" != auto {
		// 如果搜索状态是"禁启"、"启禁"等无关顺序, 只要有"启"就以启为准
		if strings.Contains(auto, "启") {
			stateWhereCondition = " OR state = 'on') "
		} else if strings.Contains(auto, "禁") {
			stateWhereCondition = " OR state = 'off') "
		} else if strings.Contains(auto, "过") || strings.Contains(auto, "期") {
			stateWhereCondition = " OR state = 'overdue') "
		} else {
			stateWhereCondition = ") "
		}

		whereCondition = " department_id in " + depIdArrStr + " AND (name LIKE '%" + auto + "%' OR department_name LIKE '%" + auto + "%' OR description LIKE '%" + auto + "%' OR priority LIKE '%" + auto + "%'"
		sql += whereCondition
		sql += stateWhereCondition
	} else if "" != policyName {
		whereCondition = " department_id in " + depIdArrStr + " AND name LIKE '%" + policyName + "%' "
		sql += whereCondition
	} else if "" != departmentName {
		whereCondition = " department_id in " + depIdArrStr + " AND department_name LIKE '%" + departmentName + "%' "
		sql += whereCondition
	} else if "" != describe {
		whereCondition = " department_id in " + depIdArrStr + " AND description LIKE '%" + describe + "%' "
		sql += whereCondition
	} else {
		// 如果搜索状态是"禁启"、"启禁"等无关顺序, 只要有"启"就以启为准
		if strings.Contains(state, "启") {
			whereCondition = " department_id in " + depIdArrStr + " AND state = 'on' "
		} else if strings.Contains(state, "禁") {
			whereCondition = " department_id in " + depIdArrStr + " AND state = 'off' "
		} else if strings.Contains(state, "过") || strings.Contains(state, "期") {
			whereCondition = " department_id in " + depIdArrStr + " AND state = 'overdue' "
		} else {
			return nil, nil
		}
		sql += whereCondition
	}
	sql += orderBy
	fmt.Println("sql语句：", sql)
	err = operateAuthRepository.DB.Raw(sql).Find(&operateAuthArr).Error
	if nil != err {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		log.Errorf("DB Error: %v", err)
		return nil, err
	}
	return
}

func UpdateDepInfoByDepIds(depIds []int64) error {
	var err error
	var depLevel int
	for i := range depIds {
		depLevel, err = DepLevel(depIds[i])
		if nil != err {
			log.Errorf("DepLevel Error: %v", err)
			return err
		}

		depName, err := departmentRepository.FindNameById(depIds[i])
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return err
		}

		err = operateAuthRepository.UpdateDepLevelByDepId(depIds[i], depLevel)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return err
		}

		err = operateAuthRepository.UpdateDepNameByDepId(depIds[i], depName)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return err
		}
	}
	return nil
}

func OperateAuthDeleteEndpoint(c echo.Context) error {
	id := c.Param("id")
	var delStrategyName string
	idArr := strings.Split(id, ",")
	for i := range idArr {
		iId, err := strconv.Atoi(idArr[i])
		if nil != err {
			log.Errorf("Atoi Error: %v", err)
			return FailWithDataOperate(c, 500, "删除失败", "", nil)
		}

		operateAuth, err := operateAuthRepository.FindById(int64(iId))
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "删除失败", "", nil)
		}
		err = operateAuthRepository.DeleteById(int64(iId))
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "删除失败", "", nil)
		}

		// 该部门中策略优先级大于被删除策略优先级的, 优先级-1
		bigOperateAuthArr, err := operateAuthRepository.BigStrategyPriority(operateAuth.DepartmentId, operateAuth.Priority)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "删除失败", "", nil)
		}
		for i := range bigOperateAuthArr {
			bigOperateAuthArr[i].Priority = bigOperateAuthArr[i].Priority - 1
			err = operateAuthRepository.UpdateById(bigOperateAuthArr[i].ID, &bigOperateAuthArr[i])
			if nil != err {
				log.Errorf("DB Error: %v", err)
				return FailWithDataOperate(c, 500, "删除失败", "", nil)
			}
		}
		delStrategyName = delStrategyName + operateAuth.Name + ", "

		err = deleteAssetAuthReportFrom(int64(iId))
		if nil != err {
			log.Errorf("删除运维授权策略时, 同步删除资产权限报表数据失败: %v, 运维授权策略ID: %v", err.Error(), int64(iId))
		}

		// 删除运维授权策略时, 同步删除应用权限报表数据
		err = deleteAppAuthReportFrom(int64(iId))
		if nil != err {
			log.Errorf("删除运维授权策略时, 同步删除应用权限报表数据失败: %v, 运维授权策略ID: %v", err.Error(), int64(iId))
		}
	}

	return SuccessWithOperate(c, "运维授权-删除: 策略名称["+delStrategyName[:len(delStrategyName)-2]+"]", nil)
}

func OperateAuthUpdateEndpoint(c echo.Context) error {
	id := c.Param("id")
	iId, err := strconv.Atoi(id)
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", nil)
	}
	var operateAuthDto model.OperateAuthDTO
	if err := c.Bind(&operateAuthDto); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	// 数据校验
	if err := c.Validate(&operateAuthDto); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	item, err := operateAuthDto.ToOperateAuth()
	if err != nil {
		log.Errorf("ToOperateAuth Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}

	account, isSuccess := GetCurrentAccountNew(c)
	if !isSuccess {
		log.Error("GetCurrentAccountNew Error")
		return FailWithDataOperate(c, 500, "修改失败", "", nil)
	}

	// 策略名称不能重复
	operateAuth, err := operateAuthRepository.FindByNameDepId(item.Name, account.DepartmentId)
	if nil != err && gorm.ErrRecordNotFound != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", nil)
	}
	if (operateAuth.Name == item.Name) && (operateAuth.ID != int64(iId)) {
		return FailWithDataOperate(c, 400, "策略名称重复", "运维授权-修改: 策略名称["+operateAuth.Name+"->"+item.Name+"], 失败原因[策略名称重复]", nil)
	}

	ownOperateAuth, err := operateAuthRepository.FindById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", nil)
	}
	// 修改了优先级
	if item.Priority != ownOperateAuth.Priority {
		depStrategyCount, err := operateAuthRepository.StrategyCountByDepId(account.DepartmentId)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", nil)
		}

		if item.Priority > int(depStrategyCount) {
			item.Priority = int(depStrategyCount)
		}

		// 新优先级大于原优先级，此范围内数据优先级-1
		if item.Priority > ownOperateAuth.Priority {
			operateAuthArr, err := operateAuthRepository.FindPriorityRangeByDepId(ownOperateAuth.Priority, item.Priority, account.DepartmentId)
			if nil != err {
				log.Errorf("DB Error: %v", err)
				return FailWithDataOperate(c, 500, "修改失败", "", nil)
			}

			for i := range operateAuthArr {
				operateAuthArr[i].Priority += -1
				err = operateAuthRepository.UpdateById(operateAuthArr[i].ID, &operateAuthArr[i])
				if nil != err {
					log.Errorf("DB Error: %v", err)
					return FailWithDataOperate(c, 500, "修改失败", "", nil)
				}
			}
		}
		// 新优先级小玉于原优先级，此范围内数据优先级+1
		if item.Priority < ownOperateAuth.Priority {
			operateAuthArr, err := operateAuthRepository.FindPriorityRangeByDepId(item.Priority, ownOperateAuth.Priority, account.DepartmentId)
			if nil != err {
				log.Errorf("DB Error: %v", err)
				return FailWithDataOperate(c, 500, "修改失败", "", nil)
			}

			for i := range operateAuthArr {
				operateAuthArr[i].Priority += 1
				err = operateAuthRepository.UpdateById(operateAuthArr[i].ID, &operateAuthArr[i])
				if nil != err {
					log.Errorf("DB Error: %v", err)
					return FailWithDataOperate(c, 500, "修改失败", "", nil)
				}
			}
		}
	}

	err = operateAuthRepository.UpdateById(int64(iId), &item)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", nil)
	}

	// 处理策略有效期字段、描述字段、IP限制列表字段不更新零值问题
	zeroFieldM := map[string]interface{}{}
	zeroFieldM["strategy_time_flag"] = item.StrategyTimeFlag
	zeroFieldM["description"] = item.Description
	zeroFieldM["ip_limit_list"] = item.IpLimitList
	err = operateAuthRepository.DB.Table("operate_auth").Where("id = ?", int64(iId)).Updates(zeroFieldM).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", nil)
	}

	return SuccessWithOperate(c, "运维授权-修改: 策略名称["+ownOperateAuth.Name+"->"+item.Name+"]", nil)
}

func OperateAuthChangeButtonStateEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	iId, err := strconv.Atoi(id)
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}

	buttonState := c.QueryParam("buttonState")

	operateAuth, err := operateAuthRepository.FindById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}
	err = operateAuthRepository.UpdateColById(int64(iId), "button_state", buttonState)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}

	if "on" == buttonState {
		return SuccessWithOperate(c, "运维授权-启用: 策略名称["+operateAuth.Name+"]", nil)
	}

	return SuccessWithOperate(c, "运维授权-禁用: 策略名称["+operateAuth.Name+"]", nil)
}

func OperateAuthCreateRelateUserEndpoint(c echo.Context) error {
	account, isSuccess := GetCurrentAccountNew(c)
	if !isSuccess {
		log.Error("GetCurrentAccountNew Error")
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	var depIds []int64
	err := GetChildDepIds(account.DepartmentId, &depIds)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	userArr, err := userNewRepository.FindUserByDepartmentIdEnable(depIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	for i := range userArr {
		depChinaName, err := DepChainName(userArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		userArr[i].Username = userArr[i].Username + "[" + userArr[i].Nickname + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}

	return SuccessWithOperate(c, "", userArr)
}

func OperateAuthCreateRelateUserGroupEndpoint(c echo.Context) error {
	account, isSuccess := GetCurrentAccountNew(c)
	if !isSuccess {
		log.Error("GetCurrentAccountNew Error")
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	var depIds []int64
	err := GetChildDepIds(account.DepartmentId, &depIds)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	userGroupArr, err := userGroupNewRepository.FindUserGroupByDepartmentId(depIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	for i := range userGroupArr {
		depChinaName, err := DepChainName(userGroupArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		userGroupArr[i].Name = userGroupArr[i].Name + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}

	return SuccessWithOperate(c, "", userGroupArr)
}

func OperateAuthCreateRelateAssetEndpoint(c echo.Context) error {
	account, isSuccess := GetCurrentAccountNew(c)
	if !isSuccess {
		log.Error("GetCurrentAccountNew Error")
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	var depIds []int64
	err := GetChildDepIds(account.DepartmentId, &depIds)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	assetArr, err := newAssetRepository.GetPassportByDepartmentIds(context.TODO(), depIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	for i := range assetArr {
		depChinaName, err := DepChainName(assetArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		assetArr[i].Name = assetArr[i].AssetName + "[" + assetArr[i].Ip + "]" + "[" + assetArr[i].Passport + "]" + "[" + assetArr[i].Protocol + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}

	return SuccessWithOperate(c, "", assetArr)
}

func OperateAuthCreateRelateAssetGroupEndpoint(c echo.Context) error {
	account, isSuccess := GetCurrentAccountNew(c)
	if !isSuccess {
		log.Error("GetCurrentAccountNew Error")
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	var depIds []int64
	err := GetChildDepIds(account.DepartmentId, &depIds)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	assetGroupArr, err := newAssetGroupRepository.GetAssetGroupListByDepartmentIds(context.TODO(), depIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	for i := range assetGroupArr {
		depChinaName, err := DepChainName(assetGroupArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		assetGroupArr[i].Name = assetGroupArr[i].Name + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}

	return SuccessWithOperate(c, "", assetGroupArr)
}

func OperateAuthCreateRelateApplicantEndpoint(c echo.Context) error {
	account, isSuccess := GetCurrentAccountNew(c)
	if !isSuccess {
		log.Error("GetCurrentAccountNew Error")
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	var depIds []int64
	err := GetChildDepIds(account.DepartmentId, &depIds)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	applicantArr, err := newApplicationRepository.GetApplicantByDepartmentIds(context.TODO(), depIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	for i := range applicantArr {
		depChinaName, err := DepChainName(applicantArr[i].DepartmentID)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		applicantArr[i].Name = applicantArr[i].Name + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}

	return SuccessWithOperate(c, "", applicantArr)
}

func OperateAuthUpdateRelateUserEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	iId, err := strconv.Atoi(id)
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	operateAuth, err := operateAuthRepository.FindById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	userStrArr := strings.Split(operateAuth.RelateUser, ",")
	userArr, err := userNewRepository.FindByIds(userStrArr)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	for i := range userArr {
		depChinaName, err := DepChainName(userArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		userArr[i].Username = userArr[i].Username + "[" + userArr[i].Nickname + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}

	return SuccessWithOperate(c, "", userArr)
}

func OperateAuthUpdateRelateUserAllEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	iId, err := strconv.Atoi(id)
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	operateAuth, err := operateAuthRepository.FindById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	var depIds []int64
	err = GetChildDepIds(operateAuth.DepartmentId, &depIds)
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

func OperateAuthUpdateRelateUserGroupEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	iId, err := strconv.Atoi(id)
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	operateAuth, err := operateAuthRepository.FindById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	userGroupStrArr := strings.Split(operateAuth.RelateUserGroup, ",")
	userGroupArr, err := userGroupNewRepository.FindByIds(userGroupStrArr)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	for i := range userGroupArr {
		depChinaName, err := DepChainName(userGroupArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		userGroupArr[i].Name = userGroupArr[i].Name + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}

	return SuccessWithOperate(c, "", userGroupArr)
}

func OperateAuthUpdateRelateUserGroupAllEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	iId, err := strconv.Atoi(id)
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	operateAuth, err := operateAuthRepository.FindById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	var depIds []int64
	err = GetChildDepIds(operateAuth.DepartmentId, &depIds)
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

func OperateAuthUpdateRelateAssetEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	iId, err := strconv.Atoi(id)
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	operateAuth, err := operateAuthRepository.FindById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	assetStrArr := strings.Split(operateAuth.RelateAsset, ",")
	assetArr, err := newAssetRepository.GetPassportByIds(context.TODO(), assetStrArr)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	for i := range assetArr {
		depChinaName, err := DepChainName(assetArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		assetArr[i].Name = assetArr[i].AssetName + "[" + assetArr[i].Ip + "]" + "[" + assetArr[i].Passport + "]" + "[" + assetArr[i].Protocol + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}

	return SuccessWithOperate(c, "", assetArr)
}

func OperateAuthUpdateRelateAssetAllEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	iId, err := strconv.Atoi(id)
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	operateAuth, err := operateAuthRepository.FindById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	var depIds []int64
	err = GetChildDepIds(operateAuth.DepartmentId, &depIds)
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

	for i := range assetAllArr {
		depChinaName, err := DepChainName(assetAllArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		assetAllArr[i].Name = assetAllArr[i].AssetName + "[" + assetAllArr[i].Ip + "]" + "[" + assetAllArr[i].Passport + "]" + "[" + assetAllArr[i].Protocol + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}

	return SuccessWithOperate(c, "", assetAllArr)
}

func OperateAuthUpdateRelateAssetGroupEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	iId, err := strconv.Atoi(id)
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	operateAuth, err := operateAuthRepository.FindById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	assetGroupStrArr := strings.Split(operateAuth.RelateAssetGroup, ",")
	assetGroupArr, err := newAssetGroupRepository.GetAssetGroupByIds(context.TODO(), assetGroupStrArr)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	for i := range assetGroupArr {
		depChinaName, err := DepChainName(assetGroupArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		assetGroupArr[i].Name = assetGroupArr[i].Name + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}

	return SuccessWithOperate(c, "", assetGroupArr)
}

func OperateAuthUpdateRelateAssetGroupAllEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	iId, err := strconv.Atoi(id)
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	operateAuth, err := operateAuthRepository.FindById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	var depIds []int64
	err = GetChildDepIds(operateAuth.DepartmentId, &depIds)
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

func OperateAuthUpdateRelateApplicationEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	iId, err := strconv.Atoi(id)
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	operateAuth, err := operateAuthRepository.FindById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	applicationStrArr := strings.Split(operateAuth.RelateApp, ",")
	applicationArr, err := newApplicationRepository.GetApplicationByIds(context.TODO(), applicationStrArr)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	for i := range applicationArr {
		depChinaName, err := DepChainName(applicationArr[i].DepartmentID)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		applicationArr[i].Name = applicationArr[i].Name + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}

	return SuccessWithOperate(c, "", applicationArr)
}

func OperateAuthUpdateRelateApplicantAllEndpoint(c echo.Context) error {
	account, isSuccess := GetCurrentAccountNew(c)
	if !isSuccess {
		log.Error("GetCurrentAccountNew Error")
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	var depIds []int64
	err := GetChildDepIds(account.DepartmentId, &depIds)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	// 查询所有应用
	applicationAllArr, err := newApplicationRepository.GetApplicantByDepartmentIds(context.TODO(), depIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	for i := range applicationAllArr {
		depChinaName, err := DepChainName(applicationAllArr[i].DepartmentID)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		applicationAllArr[i].Name = applicationAllArr[i].Name + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}

	return SuccessWithOperate(c, "", applicationAllArr)
}

func OperateAuthUpdateResourceRelateEndpoint(c echo.Context) error {
	id := c.QueryParam("id")
	iId, err := strconv.Atoi(id)
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}

	resourceType := c.QueryParam("resource_type")
	resourceIds := c.QueryParam("resource_id")

	operateAuth, err := operateAuthRepository.FindById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "关联失败", "", nil)
	}

	var resource string
	switch resourceType {
	case "user":
		resource = "用户"
		err = operateAuthRepository.UpdateColById(int64(iId), "relate_user", resourceIds)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "关联失败", "", nil)
		}
	case "user_group":
		resource = "用户组"
		err = operateAuthRepository.UpdateColById(int64(iId), "relate_user_group", resourceIds)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "关联失败", "", nil)
		}
	case "asset":
		resource = "设备"
		err = operateAuthRepository.UpdateColById(int64(iId), "relate_asset", resourceIds)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "关联失败", "", nil)
		}
	case "asset_group":
		resource = "设备组"
		err = operateAuthRepository.UpdateColById(int64(iId), "relate_asset_group", resourceIds)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "关联失败", "", nil)
		}
	case "application":
		resource = "应用"
		err = operateAuthRepository.UpdateColById(int64(iId), "relate_app", resourceIds)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "关联失败", "", nil)
		}
	}

	account, isSuccess := GetCurrentAccountNew(c)
	if !isSuccess {
		log.Error("更新运维授权策略关联资源时, 更新资产权限报表数据失败: GetCurrentAccountNew Error, 策略名称: %v, 所属部门: %v", operateAuth.Name, operateAuth.DepartmentName)
		return SuccessWithOperate(c, "运维授权-关联: 策略名称["+operateAuth.Name+"], 资源["+resource+"]", nil)
	}

	err = updateAssetAuthReportFrom(account, int64(iId))
	if nil != err {
		log.Error("更新运维授权策略关联资源时, 更新资产权限报表数据失败, 策略名称: %v, 所属部门: %v", operateAuth.Name, operateAuth.DepartmentName)
	}

	err = updateAppAuthReportFrom(account, int64(iId))
	if nil != err {
		log.Error("更新运维授权策略关联资源时, 更新应用权限报表数据失败, 策略名称: %v, 所属部门: %v", operateAuth.Name, operateAuth.DepartmentName)
	}

	return SuccessWithOperate(c, "运维授权-关联: 策略名称["+operateAuth.Name+"], 资源["+resource+"]", nil)
}

func DelOperateAuthByDepIds(depIds []int64) error {
	operateAuthArr, err := operateAuthRepository.FindByDepIds(depIds)
	if nil != err {
		log.Errorf("DB Error: %v", err.Error())
		return err
	}
	err = operateAuthRepository.DeleteInDepIds(depIds)
	if nil != err {
		log.Errorf("DB Error: %v", err.Error())
		return err
	}

	for i := range operateAuthArr {
		err = assetAuthReportFormRepository.DeleteByOperateAuthId(operateAuthArr[i].ID)
		if nil != err {
			log.Errorf("DB Error: %v", err.Error())
			log.Error("删除部门时, 删除该部门下的运维授权策略, 继而删除该运维授权策略对应的资产权限报表失败")
		}
	}

	return nil
}
