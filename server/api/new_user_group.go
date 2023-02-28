package api

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
	"strings"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/global"
	"tkbastion/pkg/validator"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/xuri/excelize/v2"
)

// UserGroupNewCreateEndpoint 创建用户组
func UserGroupNewCreateEndpoint(c echo.Context) error {
	var userGroupNewDTO dto.UserGroupNewCreate
	if err := c.Bind(&userGroupNewDTO); err != nil {
		log.Errorf("Bind Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}

	if err := c.Validate(&userGroupNewDTO); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	if userGroupNewDTO.Name == "" {
		log.Errorf("Data Error: %v", "用户组名称不能为空")
		return FailWithDataOperate(c, 403, "用户组名称不能为空", "用户分组-新增: 用户组名称["+userGroupNewDTO.Name+"],失败原因[用户组名称不能为空]", nil)
	}

	// 用户组名称不可重复
	var itemExists []model.UserGroupNew
	if err := userGroupNewRepository.DB.Where("name = ?", userGroupNewDTO.Name).Find(&itemExists).Error; nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if 0 != len(itemExists) {
		return FailWithDataOperate(c, 403, "用户组名称已存在", "用户分组-新增: 用户组名称["+userGroupNewDTO.Name+"],失败原因["+userGroupNewDTO.Name+"已存在]", nil)
	}
	// 获取当前用户信息添加部门id与名称
	account, found := GetCurrentAccountNew(c)
	if !found {
		log.Errorf("GetCurrentAccountNew Error: %v", found)
		return FailWithDataOperate(c, 500, "用户信息已失效", "", found)
	}
	var split []string
	if userGroupNewDTO.MemberIds != "" {
		split = strings.Split(userGroupNewDTO.MemberIds, ",")
	}
	item := model.UserGroupNew{
		ID:             utils.UUID(),
		Name:           userGroupNewDTO.Name,
		Created:        utils.NowJsonTime(),
		DepartmentName: account.DepartmentName,
		DepartmentId:   account.DepartmentId,
		Description:    userGroupNewDTO.Info,
		Total:          len(split),
	}

	// 将用户组写入数据库
	if err := userGroupNewRepository.Create(&item); nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}

	// 处理关联的用户组成员id
	if len(split) > 0 {
		for i := range split {
			// 查询用户id是否存在
			if _, err := userNewRepository.FindById(split[i]); err != nil {
				log.Errorf("DB Error: %v", err)
				continue
			}
			if err := userGroupMemberRepository.Create(&model.UserGroupMember{
				ID:          utils.UUID(),
				UserGroupId: item.ID,
				UserId:      split[i],
			}); nil != err {
				log.Errorf("DB Error: %v", err)
				return FailWithDataOperate(c, 500, "新增失败", "", err)
			}
		}
	}
	return SuccessWithOperate(c, "用户分组-新增: 用户组名称["+item.Name+"]", nil)
}

// UserGroupNewUpdateEndpoint 修改用户组
func UserGroupNewUpdateEndpoint(c echo.Context) error {
	id := c.Param("id")
	userGroup, err := userGroupNewRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	var userGroupNew dto.UserGroupNewUpdate
	if err := c.Bind(&userGroupNew); err != nil {
		log.Errorf("Bind Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if err := c.Validate(&userGroupNew); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	if userGroupNew.Name == "" {
		log.Errorf("Data Error: %v", "用户组名称不能为空")
		return FailWithDataOperate(c, 403, "用户组名称不能为空", "用户分组-修改: 用户组名称["+userGroupNew.Name+"],失败原因[用户组名称不能为空]", nil)
	}

	// 用户组名称不可重复
	var itemExists []model.UserGroupNew
	if err := userGroupNewRepository.DB.Where("name = ? and id != ?", userGroupNew.Name, userGroupNew.ID).Find(&itemExists).Error; nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if 0 != len(itemExists) {
		return FailWithDataOperate(c, 500, "用户组名称已存在", "用户分组-修改: 用户组名称["+userGroupNew.Name+"],失败原因["+userGroupNew.Name+"已存在]", nil)
	}
	// 描述信息修改
	if userGroupNew.Description == "" {
		userGroupNew.Description = " "
	}
	// 更新用户组信息
	if err := userGroupNewRepository.UpdateById(id, &model.UserGroupNew{
		Name:        userGroupNew.Name,
		Description: userGroupNew.Description,
	}); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	return SuccessWithOperate(c, "用户分组-修改: 用户组名称["+userGroup.Name+"->"+userGroupNew.Name+"]", nil)
}

// UserGroupNewDeleteEndpoint 删除用户组
func UserGroupNewDeleteEndpoint(c echo.Context) error {
	id := c.Param("id")
	split := strings.Split(id, ",")
	var nameDelete string
	for i := range split {
		userGroup, err := userGroupNewRepository.FindById(split[i])
		if err != nil {
			log.Errorf("DB Error: %v", err)
			continue
		}
		if err = DeleteUserGroupById(split[i]); err != nil {
			log.Errorf("DB Error: %v", err)
		}
		nameDelete += userGroup.Name + ","
	}
	if len(nameDelete) > 0 {
		nameDelete = nameDelete[:len(nameDelete)-1]
	}
	return SuccessWithOperate(c, "用户分组-删除: 用户组名称["+nameDelete+"]", nil)
}

// UserGroupNewPagingEndpoint 用户组列表
func UserGroupNewPagingEndpoint(c echo.Context) error {
	pageIndex, _ := strconv.Atoi(c.QueryParam("pageIndex"))
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))
	auto := c.QueryParam("auto")
	name := c.QueryParam("name")
	department := c.QueryParam("department")
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 401, "您的登录信息已失效,请重新登录后再试.")
	}
	var departmentId []int64
	if err := GetChildDepIds(account.DepartmentId, &departmentId); err != nil {
		log.Errorf("GetChildDepIds Error: %v", err)
	}
	userGroupList, _, err := userGroupNewRepository.FindByLimitingConditions(pageIndex, pageSize, auto, name, department, departmentId)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	for i := range userGroupList {
		// 根据部门机构id获取部门机构名称
		dep, err := departmentRepository.FindById(userGroupList[i].DepartmentId)
		if err != nil {
			log.Errorf("DB Error: %v", err)
		}
		userGroupList[i].DepartmentName = dep.Name
		// 更新用户的部门机构名称
		if err := userGroupNewRepository.UpdateById(userGroupList[i].ID, &model.UserGroupNew{DepartmentName: dep.Name}); nil != err {
			log.Errorf("DB Error: %v", err)
		}
		total, err := userGroupMemberRepository.CountByUserGroupId(userGroupList[i].ID)
		if err != nil {
			log.Errorf("DB Error: %v", err)
		}
		if userGroupList[i].Total != int(total) {
			if err := userGroupNewRepository.UpdateById(userGroupList[i].ID, &model.UserGroupNew{
				Total: int(total),
			}); err != nil {
				log.Errorf("DB Error: %v", err)
			}
		}
		userGroupList[i].Total = int(total)
	}
	return Success(c, userGroupList)
}

// GetCurrentDepartmentUserChildEndpoint 获取当前部门所能看到的所有用户
func GetCurrentDepartmentUserChildEndpoint(c echo.Context) error {
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 401, "您的登录信息已失效,请重新登录后再试.")
	}
	userList, err := GetCurrentDepartmentUserChildren(account.DepartmentId)
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
	return SuccessWithOperate(c, "查询成功", userList)
}

// GetCurrentUserGroupDepartmentUserChildrenEndpoint 获取当前用户组所在部门所能看到的用户
func GetCurrentUserGroupDepartmentUserChildrenEndpoint(c echo.Context) error {
	userGroupId := c.Param("id")
	userGroup, err := userGroupNewRepository.FindById(userGroupId)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	userList, err := GetCurrentDepartmentUserChildren(userGroup.DepartmentId)
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

// UserGroupNewMemberEndpoint 用户组已关联的用户
func UserGroupNewMemberEndpoint(c echo.Context) error {
	id := c.Param("id")
	var userGroupMemberList []model.UserNew
	// 根据id找到当前用户组
	userGroup, err := userGroupNewRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	// 根据当前用户组id找到现有的用户组成员
	userGroupMember, err := userGroupMemberRepository.FindUserIdsByUserGroupId(userGroup.ID)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	// 根据用户组成员id找到用户组成员信息
	for i := range userGroupMember {
		user, err := userNewRepository.FindById(userGroupMember[i])
		if err != nil {
			log.Errorf("DB Error: %v", err)
			continue
		}
		depChinaName, err := DepChainName(user.DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
		}
		user.Username = user.Username + "[" + user.Nickname + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
		userGroupMemberList = append(userGroupMemberList, user)
	}
	return Success(c, userGroupMemberList)
}

// UserGroupNewMemberAddEndpoint 用户组关联用户
func UserGroupNewMemberAddEndpoint(c echo.Context) error {
	id := c.Param("id")
	userId := c.QueryParam("aid")
	// 根据id找到当前用户组
	userGroup, err := userGroupNewRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	// 删除原有用户组的成员
	if err := userGroupMemberRepository.DeleteByUserGroupId(userGroup.ID); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "", err)
	}
	if len(userId) != 0 {
		split := strings.Split(userId, ",")
		for i := range split {
			if err := userGroupMemberRepository.Create(&model.UserGroupMember{
				ID:          utils.UUID(),
				UserGroupId: userGroup.ID,
				UserId:      split[i],
			}); err != nil {
				log.Errorf("DB Error: %v", err)
				return FailWithDataOperate(c, 500, "添加失败", "", err)
			}
		}
		userGroup.Total = len(split)
	}
	return SuccessWithOperate(c, "用户分组-关联用户: 用户组名称["+userGroup.Name+"]", nil)
}

// GetCurrentDepartmentUserGroupChild 获取某部门下所有用户组，参数传当前的部门id
func GetCurrentDepartmentUserGroupChild(departmentId int64) (userChild []model.UserGroupNew, err error) {
	var departmentIds []int64
	if err := GetChildDepIds(departmentId, &departmentIds); err != nil {
		log.Errorf("GetChildDepIds Error: %v", err)
	}
	if userChild, err = userGroupNewRepository.FindUserGroupByDepartmentId(departmentIds); err != nil {
		return nil, err
	}
	return userChild, nil
}

// DeleteUserGroupChildByDepartmentIds 删除某部门下所有用户组，参数传部门id
func DeleteUserGroupChildByDepartmentIds(departmentId []int64) (err error) {
	for _, v := range departmentId {
		userGroupChild, err := GetCurrentDepartmentUserGroupChild(v)
		if err != nil {
			log.Errorf("GetCurrentDepartmentUserChildren Error :%v", err)
		}
		for i := range userGroupChild {
			if err := DeleteUserGroupById(userGroupChild[i].ID); err != nil {
				log.Errorf("DeleteUserById Error :%v", err)
			}
		}
	}
	return nil
}

// DeleteUserGroupById 通过用户组id删除用户组
func DeleteUserGroupById(userGroupId string) error {
	userGroup, err := userGroupNewRepository.FindById(userGroupId)
	if err != nil {
		return err
	}

	// 删除用户组
	err = userGroupNewRepository.DeleteById(userGroupId)
	if err != nil {
		return err
	}

	// 删除用户组关联
	if err := userGroupMemberRepository.DeleteByUserGroupId(userGroupId); err != nil {
		log.Errorf("DB Error: %v", err)
	}

	// 删除用户策略关联
	if err := userStrategyRepository.DB.Table("user_strategy_user_group").Where("user_group_id = ?", userGroup.ID).Delete(&model.UserStrategyUsers{}).Error; nil != err {
		log.Errorf("DB Error: %v", err)
	}

	// 更新授权策略的关联的用户组
	//operateAuth, err := operateAuthRepository.FindByRateUserId(userGroupId)
	//if err != nil {
	//	log.Errorf("FindByRateUserId Error: %v", err)
	//}
	//for _, v := range operateAuth {
	//	// 处理用户id
	//	userGroupIds := strings.Split(v.RelateUser, ",")
	//	relateUserGroup := ""
	//	for _, v2 := range userGroupIds {
	//		if v2 != userGroupId {
	//			relateUserGroup += v2
	//		}
	//	}
	//	// 更新授权策略的关联的用户组
	//	if err := operateAuthRepository.UpdateById(v.ID, &model.OperateAuth{RelateUserGroup: relateUserGroup}); err != nil {
	//		log.Errorf("UpdateById Error: %v", err)
	//	}
	//}

	// 删除指令策略关联
	if err := commandRelevanceRepository.DeleteByUserGroupId(userGroupId); err != nil {
		log.Errorf("DB Error: %v", err)
	}

	return nil
}

// GetUserByUserGroupId 输入用户组id，返回用户组成员结构体数组
func GetUserByUserGroupId(userGroupId string) (user []model.UserNew, err error) {
	var userId []string
	if userId, err = userGroupMemberRepository.FindUserIdsByUserGroupId(userGroupId); err != nil {
		return nil, err
	}
	for i := range userId {
		userTemp, err := userNewRepository.FindById(userId[i])
		if err != nil {
			continue
		}
		user = append(user, userTemp)
	}
	return user, nil
}

// UserGroupDownloadTemplateEndpoint 下载模板
func UserGroupDownloadTemplateEndpoint(c echo.Context) error {
	userHeaderForExport := []string{"组名(必填)", "部门名称(不填默认为根部门)(如果部门名称重复请在名称后加{{id}} 例如:测试部{{22}})", "描述", "组员用户名（如果关联多个用户，请使用','隔开）"}
	userFileNameForExport := "用户分组"
	file, err := utils.CreateTemplateFile(userFileNameForExport, userHeaderForExport)
	if err != nil {
		log.Errorf("CreateExcelFile Error: %v", err)
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	fileName := "用户分组导入模板.xlsx"
	var buff bytes.Buffer
	if err = file.Write(&buff); err != nil {
		log.Errorf("Write Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "", err)
	}
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
}

// UserGroupNewImportEndpoint 导入用户组
func UserGroupNewImportEndpoint(c echo.Context) error {
	// 获取文件
	//isCover := c.FormValue("is_cover")
	file, err := c.FormFile("file")
	if err != nil {
		log.Errorf("FormFile Error: %v", err)
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}

	//验证文件类型
	fileSuffix := path.Ext(file.Filename) // 文件类型
	if fileSuffix != ".xlsx" {
		return FailWithDataOperate(c, 500, "文件类型错误", "", nil)
	}

	src, err := file.Open()
	if nil != err {
		log.Errorf("Open Error: %v", err)
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}
	defer func(src multipart.File) {
		err := src.Close()
		if nil != err {
			log.Errorf("Close Error: %v", err)
		}
	}(src)
	// 读excel流
	xlsx, err := excelize.OpenReader(src)
	if nil != err {
		log.Errorf("OpenReader Error: %v", err)
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}
	// 根据名字获取cells的内容，返回的是一个[][]string
	records, err := xlsx.GetRows(xlsx.GetSheetName(xlsx.GetActiveSheetIndex()))
	if nil != err {
		log.Errorf("GetRows Error: %v", err)
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}
	// 获取当前账户
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 500, "获取当前账户失败")
	}
	var departmentId = account.DepartmentId
	var departmentName = account.DepartmentName
	var departmentIds []int64
	err = GetChildDepIds(departmentId, &departmentIds)
	if err != nil {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}

	// 导入文件有几行, 0行文件为空, 1行代表只有标题行也是"空"
	if len(records) <= 1 {
		return FailWithDataOperate(c, 400, "导入文件数据为空", "用户分组-导入: 文件名["+file.Filename+"], 失败原因[导入文件数据为空]", nil)
	}
	if len(records[0]) != 4 {
		return FailWithDataOperate(c, 400, "导入文件格式错误", "用户分组-导入: 文件名["+file.Filename+"], 失败原因[导入文件格式错误]", nil)
	}

	var nameSuccess, nameFiled string
	var successNum, filedNum int
	//if isCover == "true" {
	//	// 查找要删除的用户分组
	//	var userGroupList []model.UserGroupNew
	//	err = userGroupNewRepository.DB.Where("department_id in (?) ", departmentIds).Find(&userGroupList).Error
	//	if err != nil {
	//		log.Errorf("Find Error: %v", err)
	//		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	//	}
	//	// 删除用户分组
	//	for _, userGroup := range userGroupList {
	//		err = DeleteUserGroupById(userGroup.ID)
	//		if err != nil {
	//			log.Errorf("User GroupName: %v , DeleteUserGroupById Error: %v", userGroup.Name, err)
	//		}
	//	}
	//}
	for i := range records {
		if i == 0 {
			continue
		}
		if len(records[i][0]) == 0 {
			nameFiled += records[i][0] + "[" + "必填项为空" + "]" + ","
			filedNum++
			continue
		}

		// 用户名和昵称不能重复
		var itemExists []model.UserNew
		if err := userGroupNewRepository.DB.Table("user_group_new").Where("name = ? ", records[i][0]).Find(&itemExists).Error; nil != err {
			log.Errorf("DB Error: %v", err)
			continue
		}
		if 0 != len(itemExists) {
			nameFiled += records[i][0] + "[" + "名称重复" + "]" + ","
			filedNum++
			continue
		}
		if len(records[i]) > 1 && len(records[i][1]) != 0 {
			index := strings.IndexByte(records[i][1], '{')
			if index != -1 {
				Id, err := strconv.ParseInt(records[i][1][index+1:len(records[i][1])-1], 10, 64)
				if err != nil {
					log.Errorf("ParseInt Error: %v", err)
					nameFiled += records[i][0] + "[" + "部门ID错误" + "]" + ","
					filedNum++
					continue
				}
				departmentId = Id
				departmentName = records[i][1][0:index]
			} else {
				department, err := departmentRepository.FindByName(records[i][1])
				if err != nil {
					log.Errorf("FindByNameId Error: %v", err)
					nameFiled += records[i][0] + "[" + "部门不存在" + "]" + ","
					filedNum++
					continue
				}
				departmentName = records[i][1]
				departmentId = department.ID
			}
		}
		// 判断此用户部门等级是不是符合要求
		res := IsDepIdBelongDepIds(departmentId, departmentIds)
		if !res {
			nameFiled += records[i][0] + "[" + "无权限导入此用户" + "]" + ","
			filedNum++
			continue
		}
		var userList []string
		var description string
		if len(records[i]) == 3 {
			description = records[i][2]
		}
		if len(records[i]) == 4 && len(records[i][3]) != 0 {
			description = records[i][2]
			userList = strings.Split(records[i][3], ",")
		}
		// 创建用户组
		userGroup := model.UserGroupNew{
			ID:             utils.UUID(),
			Name:           records[i][0],
			DepartmentId:   departmentId,
			DepartmentName: departmentName,
			Total:          len(userList),
			Description:    description,
			Created:        utils.NowJsonTime(),
		}
		if err = userGroupNewRepository.Create(&userGroup); err != nil {
			log.Errorf("Create Error: %v", err)
			nameFiled += records[i][0] + "[" + "创建用户组失败" + "]" + ","
			filedNum++
			continue
		}
		// 创建用户组成员
		for _, userL := range userList {
			user, err := userNewRepository.FindByName(userL)
			if err != nil {
				log.Errorf("FindByNameId Error: %v", err)
				continue
			}

			if err = userGroupMemberRepository.Create(&model.UserGroupMember{
				ID:          utils.UUID(),
				UserGroupId: userGroup.ID,
				UserId:      user.ID,
			}); err != nil {
				log.Errorf("Create Error: %v", err)
				nameFiled += records[i][0] + "[" + "创建用户组成员失败" + "]" + ","
				filedNum++
				continue
			}
			successNum++
			nameSuccess += records[i][0] + ","
		}
	}
	if len(nameSuccess) != 0 {
		nameSuccess = nameSuccess[0 : len(nameSuccess)-1]
	}
	if len(nameFiled) != 0 {
		nameFiled = nameFiled[0 : len(nameFiled)-1]
	}
	return SuccessWithOperate(c, "用户分组-导入: [导入成功分组:{"+nameSuccess+"},成功数:"+strconv.Itoa(successNum)+"]"+"[导入失败分组:{"+nameFiled+"},失败数:"+strconv.Itoa(filedNum)+"]", nil)
}

// UserGroupNewExportEndpoint 导出用户组
func UserGroupNewExportEndpoint(c echo.Context) error {
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 401, "您的登录信息已失效,请重新登录后再试.")
	}
	var userGroupForExport []dto.UserGroupForExport
	var departmentId []int64
	err := GetChildDepIds(account.DepartmentId, &departmentId)
	if err != nil {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "", err)
	}
	// 获取数据
	userGroupForExport, err = userGroupNewRepository.UserGroupExport(departmentId)
	if err != nil {
		log.Errorf("UserExport Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "", err)
	}
	// 根据组id查到组成员
	for i := range userGroupForExport {
		userId, err := userGroupMemberRepository.FindUserIdsByUserGroupId(userGroupForExport[i].Id)
		if err != nil {
			log.Errorf("FindUserIdsByUserGroupId Error: %v", err)
			continue
		}
		var name string
		var total int
		for j := range userId {
			userNew, err := userNewRepository.FindById(userId[j])
			if err != nil {
				log.Errorf("FindById Error: %v", err)
				continue
			}
			name += userNew.Username + ","
			total++
		}
		if len(name) != 0 {
			name = name[0 : len(name)-1]
		}
		userGroupForExport[i].Members = name
		userGroupForExport[i].Total = total
	}

	userGroupStringsForExport := make([][]string, len(userGroupForExport))
	for i, v := range userGroupForExport {
		v.Id = strconv.Itoa(i + 1)
		user := utils.Struct2StrArr(v)
		userGroupStringsForExport[i] = make([]string, len(user))
		userGroupStringsForExport[i] = user
	}
	userGroupHeaderForExport := []string{"序号", "组名", "部门机构", "描述", "成员数", "组员用户名"}
	userGroupFileNameForExport := "用户分组"
	file, err := utils.CreateExcelFile(userGroupFileNameForExport, userGroupHeaderForExport, userGroupStringsForExport)
	if err != nil {
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	fileName := "用户分组.xlsx"

	// 记录操作日志
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		Result:          "成功",
		LogTypes:        "用户日志",
		LogContents:     "用户分组-导出: 部门[" + account.DepartmentName + "下用户组]",
		Created:         utils.NowJsonTime(),
		Users:           account.Username,
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

//
