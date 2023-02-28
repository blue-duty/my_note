package api

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"tkbastion/pkg/constant"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/pkg/validator"
	"tkbastion/server/utils"

	"github.com/xuri/excelize/v2"

	"gorm.io/gorm"

	"tkbastion/server/model"

	"github.com/labstack/echo/v4"
)

// 事务  TODO
func DepartmentCreateEndpoint(c echo.Context) error {
	var item model.Department
	if err := c.Bind(&item); nil != err {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}

	// 数据校验
	if err := c.Validate(&item); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	// 上级部门信息
	fatDep, err := departmentRepository.FindById(item.FatherId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}
	// 不能与兄弟部门重名
	broDepArr, err := departmentRepository.BroDep(item.FatherId)
	for i := range broDepArr {
		if item.Name == broDepArr[i].Name {
			return FailWithDataOperate(c, 400, "部门名称重复", "部门机构-新增: 部门名称["+item.Name+"]"+", 上级部门["+fatDep.Name+"], 失败原因[部门名称重复]", err)
		}
	}

	if err := departmentRepository.Create(&item); nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", nil)
	}
	dep, err := departmentRepository.FindByNameFatherId(item.Name, item.FatherId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", nil)
	}
	err = addDepartmentUpdateNodeRelation(dep.ID, item.FatherId)
	if nil != err {
		return FailWithDataOperate(c, 500, "新增失败", "", nil)
	}

	return SuccessWithOperate(c, "部门机构-新增: 部门名称["+item.Name+"], 上级部门["+fatDep.Name+"]", item)
}

// 某部门下新增节点后，更新父节点及兄弟节点间关系
func addDepartmentUpdateNodeRelation(depId, fatDepId int64) error {
	fatNode, err := departmentRepository.FindById(fatDepId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return err
	}

	if -2 == fatNode.LeftChildId {
		// 父节点无左孩子，设置其为左孩子
		fatNode.LeftChildId = depId
		err = departmentRepository.UpdateById(fatDepId, &fatNode)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return err
		}
	} else {
		// 找到左孩子节点
		leftChildNode, err := departmentRepository.FindById(fatNode.LeftChildId)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return err
		}
		for {
			if -2 == leftChildNode.RightBroId {
				leftChildNode.RightBroId = depId
				err = departmentRepository.UpdateById(leftChildNode.ID, &leftChildNode)
				if nil != err {
					log.Errorf("DB Error: %v", err)
					return err
				}
				return nil
			} else {
				// 找左孩子的右兄弟节点，找到后右兄弟成为新的左孩子
				leftChildNode, err = departmentRepository.FindById(leftChildNode.RightBroId)
				if nil != err {
					log.Errorf("DB Error: %v", err)
					return err
				}
			}
		}
	}
	return nil
}

func DepartmentPagingEndpoint(c echo.Context) error {
	// 获取用户所属部门，只能看到自己所属部门与下级部门数据（此处的数据指的是"部门"这个数据）
	// 至于此用户是否能看到部门数据，这个受用户角色控制是否能看到此菜单，是否有此菜单权限，与数据权限不同，此处不管这点，默认走到这个api，即是拥有菜单和api权限的
	account, _ := GetCurrentAccountNew(c)
	auto := c.QueryParam("auto")
	departmentName := c.QueryParam("department_name")
	departmentId := c.QueryParam("department_id")
	if "" == auto && "" == departmentName && "" == departmentId {
		var departTreeArr model.DepartmentTree
		fatNode, err := departmentRepository.FindById(account.DepartmentId)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return SuccessWithOperate(c, "", []model.DepartmentTree{departTreeArr})
		}

		fatNodeArr := &departTreeArr
		err = FindDepartment(fatNode, fatNodeArr)
		if nil != err {
			return SuccessWithOperate(c, "", []model.DepartmentTree{departTreeArr})
		}

		if "" != departTreeArr.Name {
			departTreeArr.FatherId = -1
		}
		return SuccessWithOperate(c, "", []model.DepartmentTree{departTreeArr})
	}
	// TODO 各种查询失败建议不要以FailWithDataOperate形式返回, 这样页面会显示失败, 挫败感很强, 可以以SuccessWithOperate形式返回, 数据为空就行
	departTreeWithCondition, err := FindDepartmentWithCondition(auto, departmentName, departmentId, account.DepartmentId)
	if nil != err || "" == departTreeWithCondition.Name {
		// 可能没err, 但如果返回的结构体为空结构体, 需返回一个没有数据的数组而不是返回一个有一个数据的数组, 即使这一个数据是该类型数据空值, 前端展示也会有问题
		log.Errorf("FindDepartmentWithCondition Error: %v", err)
		return SuccessWithOperate(c, "", []model.DepartmentTree{})
	}
	if "" != departTreeWithCondition.Name {
		departTreeWithCondition.FatherId = -1
	}
	return SuccessWithOperate(c, "", []model.DepartmentTree{departTreeWithCondition})
}

func FindDepartmentWithCondition(auto, departmentName, departmentId string, genDepId int64) (departTreeWithCondition model.DepartmentTree, err error) {
	var depIds []int64
	if "" != auto {
		sql := "SELECT id FROM department WHERE "
		whereCondition := " id LIKE '%" + auto + "%' OR name LIKE '%" + auto + "%' OR asset_count LIKE '%" + auto + "%' OR user_count LIKE '%" + auto + "%' OR app_count LIKE '%" + auto + "%' OR app_server_count LIKE '%" + auto + "%' OR description LIKE '%" + auto + "%'"
		sql += whereCondition
		err = departmentRepository.DB.Raw(sql).Find(&depIds).Error
	} else if "" != departmentName {
		depIds, err = departmentRepository.FindIdsByVagueName(departmentName)
	} else if "" != departmentId {
		iDepartmentId, err := strconv.Atoi(departmentId)
		if nil != err {
			log.Errorf("Atoi Error: %v", err)
			return model.DepartmentTree{}, err
		}

		_, err = departmentRepository.FindById(int64(iDepartmentId))
		depIds = append(depIds, int64(iDepartmentId))
	}
	if nil != err {
		if err == gorm.ErrRecordNotFound {
			return model.DepartmentTree{}, err
		}
		log.Errorf("DB Error: %v", err)
		return model.DepartmentTree{}, err
	}

	var fatherIdsAll, fatIds []int64
	for i := range depIds {
		err = UpDepartment(depIds[i], &fatIds)
		if nil != err {
			log.Errorf("UpDepartment Error: %v", err)
			return model.DepartmentTree{}, err
		}
		fatherIdsAll = append(fatherIdsAll, fatIds...)
	}
	if 0 == len(fatherIdsAll) {
		return model.DepartmentTree{}, nil
	}
	// fatAllIds去重
	fatherIdsAll = RemoveDuplicates(fatherIdsAll)

	genNode, err := departmentRepository.FindById(genDepId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return model.DepartmentTree{}, err
	}
	err = FindDepartmentNotInIds(genNode, &departTreeWithCondition, fatherIdsAll)
	if nil != err {
		return model.DepartmentTree{}, err
	}

	return departTreeWithCondition, nil
}

func FindDepartmentNotInIds(fatNode model.Department, fatNodeArr *model.DepartmentTree, notDepIds []int64) (err error) {
	fatNodeArr.ID = fatNode.ID
	fatNodeArr.FatherId = fatNode.FatherId
	fatNodeArr.Name = fatNode.Name
	fatNodeArr.Description = fatNode.Description

	var childIds []int64
	err = GetChildDepIds(fatNode.ID, &childIds)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return err
	}

	userCount, err := userNewRepository.FindUserCountByDepIds(childIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		userCount = 0
	}
	appCount, err := newApplicationRepository.FindAppCountByDepartmentIds(context.TODO(), childIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		appCount = 0
	}
	assetCount, err := newAssetRepository.GetAssetCountByDepIds(context.TODO(), childIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		assetCount = 0
	}
	appServerCount, err := newApplicationServerRepository.FindAppSerCountByDepIds(context.TODO(), childIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		appServerCount = 0
	}
	fatNodeArr.AppCount = int(appCount)
	fatNodeArr.AssetCount = int(assetCount)
	fatNodeArr.UserCount = int(userCount)
	fatNodeArr.AppServerCount = int(appServerCount)

	fatNodeArr.ChildArr = nil

	if -2 == fatNode.LeftChildId {
		return nil
	} else {
		// 找到第一个孩子
		childNode, err := departmentRepository.FindById(fatNode.LeftChildId)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return err
		}

		for true {
			// 找到所有孩子
			if idIsExistIds(childNode.ID, notDepIds) {
				fatNodeArr.ChildArr = append(fatNodeArr.ChildArr, model.DepartmentTree{ID: childNode.ID})
			}
			if -2 == childNode.RightBroId {
				break
			} else {
				childNode, err = departmentRepository.FindById(childNode.RightBroId)
				if nil != err {
					log.Errorf("DB Error: %v", err)
					return err
				}
			}
		}
		for i := range fatNodeArr.ChildArr {
			fatNode, err = departmentRepository.FindById(fatNodeArr.ChildArr[i].ID)
			if nil != err {
				log.Errorf("DB Error: %v", err)
				return err
			}
			FindDepartmentNotInIds(fatNode, &fatNodeArr.ChildArr[i], notDepIds)
		}
	}

	return nil
}

// 部门id切片去重
func RemoveDuplicates(depIds []int64) []int64 {
	// 只需要检查key，因此value使用空结构体，不占内存
	processed := make(map[int64]struct{})

	uniqDepIds := make([]int64, 0)
	for _, depId := range depIds {
		// 如果部门id已经处理过，就跳过
		if _, ok := processed[depId]; ok {
			continue
		}

		// 将唯一的部门id加到切片中
		uniqDepIds = append(uniqDepIds, depId)

		// 将部门depId标记为已存在
		processed[depId] = struct{}{}
	}

	return uniqDepIds
}

func idIsExistIds(depId int64, depIds []int64) bool {
	isExist := false
	for i := range depIds {
		if depIds[i] == depId {
			isExist = true
			break
		}
	}

	return isExist
}

func FindDepartment(fatNode model.Department, fatNodeArr *model.DepartmentTree) error {
	// 外层函数的departTreeArr变量为给前端返回数据，  外层函数的fatNode变量为当前用户所属部门的Node信息
	// 先把自身fatNode信息补全，然后找到所有孩子，遍历孩子，再把孩子当成新的fatNode递归获取所有部门信息
	fatNodeArr.ID = fatNode.ID
	fatNodeArr.FatherId = fatNode.FatherId
	fatNodeArr.Name = fatNode.Name
	fatNodeArr.Description = fatNode.Description

	var childIds []int64
	err := GetChildDepIds(fatNode.ID, &childIds)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return err
	}

	userCount, err := userNewRepository.FindUserCountByDepIds(childIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		userCount = 0
	}
	appCount, err := newApplicationRepository.FindAppCountByDepartmentIds(context.TODO(), childIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		appCount = 0
	}
	assetCount, err := newAssetRepository.GetAssetCountByDepIds(context.TODO(), childIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		assetCount = 0
	}
	appServerCount, err := newApplicationServerRepository.FindAppSerCountByDepIds(context.TODO(), childIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		appServerCount = 0
	}
	fatNodeArr.AppCount = int(appCount)
	fatNodeArr.AssetCount = int(assetCount)
	fatNodeArr.UserCount = int(userCount)
	fatNodeArr.AppServerCount = int(appServerCount)
	fatNodeArr.ChildArr = nil

	if -2 == fatNode.LeftChildId {
		return nil
	} else {
		// 找到第一个孩子
		childNode, err := departmentRepository.FindById(fatNode.LeftChildId)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return err
		}

		for true {
			// 找到所有孩子
			fatNodeArr.ChildArr = append(fatNodeArr.ChildArr, model.DepartmentTree{ID: childNode.ID})
			if -2 == childNode.RightBroId {
				break
			} else {
				childNode, err = departmentRepository.FindById(childNode.RightBroId)
				if nil != err {
					log.Errorf("DB Error: %v", err)
					return err
				}
			}
		}
		for i := range fatNodeArr.ChildArr {
			fatNode, err = departmentRepository.FindById(fatNodeArr.ChildArr[i].ID)
			if nil != err {
				log.Errorf("DB Error: %v", err)
				return err
			}
			FindDepartment(fatNode, &fatNodeArr.ChildArr[i])
		}
	}

	return nil
}

// TODO 事务
func DepartmentDeleteEndpoint(c echo.Context) error {
	id := c.Param("id")
	iId, err := strconv.Atoi(id)
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "", nil)
	}

	// 查询部门是否存在设备、用户、应用
	if d, err := departmentRepository.FindById(int64(iId)); nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "", nil)
	} else {
		if d.AssetCount > 0 || d.AppCount > 0 || d.UserCount > 0 || d.AppServerCount > 0 {
			return FailWithDataOperate(c, 400, "该部门下存在资产、用户、应用或应用服务器，无法删除", "", nil)
		}
	}

	// 不允许删除根部门
	if 0 == iId {
		return FailWithDataOperate(c, 400, "不允许删除根部门", "部门机构-删除: 部门名称[根部门], 失败原因[不允许删除根部门]", err)
	}
	// 不允许删除当前用户所属部门
	account, _ := GetCurrentAccountNew(c)
	if int64(iId) == account.DepartmentId {
		return FailWithDataOperate(c, 400, "不允许删除当前用户所属部门", "部门机构-删除: 部门名称["+account.DepartmentName+"], 失败原因[不允许删除当前用户所属部门]", err)
	}

	var delDepIds []int64
	delDepNode, err := departmentRepository.FindById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "", nil)
	}

	err = GetChildDepIds(int64(iId), &delDepIds)
	if nil != err {
		return FailWithDataOperate(c, 500, "删除失败", "", nil)
	}
	err = curNodeUpdateFatherBroRelation(delDepNode)
	if nil != err {
		return FailWithDataOperate(c, 500, "删除失败", "", nil)
	}

	deleteDepsData(delDepIds)

	return SuccessWithOperate(c, "部门机构-删除: 部门名称["+delDepNode.Name+"]", nil)
}

// 输入部门id参数，返回以"."为连接符的从根部门至参数depId所在部门的 链式部门名称串
func DepChainName(depId int64) (depChainName string, err error) {
	delDepNode, err := departmentRepository.FindById(depId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return "", err
	}

	if -1 == delDepNode.FatherId {
		depChainName += "根部门."
	} else {
		depChainName, err = DepChainName(delDepNode.FatherId)
		if nil != err {
			return "", err
		}
		depChainName += delDepNode.Name
		depChainName += "."
	}
	return
}

// 返回此部门下级所有部门id(包含此部门id)
// delDepIds 为输出型参数
func GetChildDepIds(depId int64, delDepIds *[]int64) error {
	// 因为不允许删除本部门, 因此这里传的部门id肯定可以删除
	delDepNode, err := departmentRepository.FindById(depId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return err
	}

	var childIdArr []int64
	*delDepIds = append(*delDepIds, delDepNode.ID)
	if -2 == delDepNode.LeftChildId {
		return nil
	} else {
		// 找到第一个孩子
		childNode, err := departmentRepository.FindById(delDepNode.LeftChildId)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return err
		}

		for true {
			// 找到所有孩子
			childIdArr = append(childIdArr, childNode.ID)
			if -2 == childNode.RightBroId {
				break
			} else {
				childNode, err = departmentRepository.FindById(childNode.RightBroId)
				if nil != err {
					log.Errorf("DB Error: %v", err)
					return err
				}
			}
		}

		for i := range childIdArr {
			GetChildDepIds(childIdArr[i], delDepIds)
		}
	}
	return nil
}

func deleteDepsData(depIds []int64) {
	// 删除某一数据失败时记log, 不返回, 继续删除后续的数据
	// 因删除部门资源后可能导致部分后续删除的资源调用接口失败, 因此部门资源最后一个删除

	// 用户
	err := DeleteUserChildByDepartmentIds(depIds)
	if nil != err {
		log.Error("删除部门时同步删除用户失败: ", err)
	}
	// 用户分组
	err = DeleteUserGroupChildByDepartmentIds(depIds)
	if nil != err {
		log.Error("删除部门时同步删除用户分组失败: ", err)
	}
	// 用户策略
	err = DeleteUserStrategyChildByDepartmentIds(depIds)
	if nil != err {
		log.Error("删除部门时同步删除用户策略失败: ", err)
	}
	// 设备列表
	asset, err := newAssetRepository.DeleteByDepartmentId(context.TODO(), depIds)
	if nil != err {
		log.Error("删除部门时同步删除设备列表/权限报表失败: ", err)
	}
	for _, v := range asset {
		UpdateUserAssetAppAppserverDep(constant.ASSET, constant.DELETE, v.DepartmentId, int64(-1))
	}
	// 设备分组
	err = newAssetGroupRepository.DeleteAssetGroupByDepartmentId(context.TODO(), depIds)
	if nil != err {
		log.Error("删除部门时同步删除设备分组失败: ", err)
	}
	// 应用管理
	apps, err := newApplicationRepository.DeleteByDepartmentId(context.TODO(), depIds)
	if nil != err {
		log.Error("删除部门时同步删除应用管理失败: ", err)
	}
	for _, v := range apps {
		UpdateUserAssetAppAppserverDep(constant.APP, constant.DELETE, v.DepartmentID, int64(-1))
	}
	// 应用服务器
	appServers, err := newApplicationServerRepository.DeleteByDepartmentId(context.TODO(), depIds)
	if nil != err {
		log.Errorf("DeleteApplicationServerByDepartmentId Error: %v", err)
	}
	for _, v := range appServers {
		UpdateUserAssetAppAppserverDep(constant.APPSERVER, constant.DELETE, v.DepartmentID, int64(-1))
	}
	// 磁盘空间
	storages, err := storageRepositoryNew.FindByDepartmentId(context.TODO(), depIds)
	if nil != err {
		log.Error("删除部门时同步删除磁盘空间失败: ", err)
	}
	for _, v := range storages {
		err := storageNewService.DeleteStorageById(v.ID)
		if err != nil {
			log.Error("删除部门时同步删除磁盘空间失败: ", err)
		}
	}
	// 运维授权
	err = DelOperateAuthByDepIds(depIds)
	if nil != err {
		log.Error("删除部门时同步删除此部门下运维授权策略/权限报表失败")
	}
	// 指令控制
	if err := DeleteCommandStrategyByDepartmentIds(depIds); err != nil {
		log.Errorf("删除部门时同步删除指令控制失败: %v", err)
	}
	// 任务列表
	if err := newjobRepository.DeleteByDepartmentId(context.TODO(), depIds); err != nil {
		log.Errorf("删除部门时同步删除任务列表失败: %v", err)
	}
	// 执行日志
	if err := newjobRepository.DeleteJobLogByDepartmentId(context.TODO(), depIds); err != nil {
		log.Errorf("删除部门时同步删除执行日志失败: %v", err)
	}
	//部门机构
	err = departmentRepository.DeleteInDepIds(depIds)
	if nil != err {
		log.Error("删除部门时同步删除当前部门及下属部门失败: ", err)
	}
}

// 删除某部门及下级部门后，更新此部门的上级部门及兄弟部门间节点关系
// 也可用于修改某部门所属上级部门后，更新原父节点，兄弟节点间关系
// 找到此节点对应的父节点，判断此节点是不是父的左孩子节点，如果是，将其右兄弟节点id变为父节点的左孩子节点
// 若不是，查找其为谁的右兄弟节点， 将此节点的右兄弟id更新为被删除节点的右兄弟节点id
func curNodeUpdateFatherBroRelation(delDepNode model.Department) error {
	fatherNode, err := departmentRepository.FindById(delDepNode.FatherId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return err
	}
	leftNode, err := departmentRepository.FindById(fatherNode.LeftChildId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return err
	}
	if leftNode.ID == delDepNode.ID {
		fatherNode.LeftChildId = delDepNode.RightBroId
		err = departmentRepository.UpdateById(fatherNode.ID, &fatherNode)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return err
		}
	} else {
		for leftNode.RightBroId != delDepNode.ID {
			leftNode, err = departmentRepository.FindById(leftNode.RightBroId)
			if nil != err {
				log.Errorf("DB Error: %v", err)
				return err
			}
		}
		leftNode.RightBroId = delDepNode.RightBroId
		err = departmentRepository.UpdateById(leftNode.ID, &leftNode)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return err
		}

		// 修改部门所属上级部门后，该部门已是此上级部门孩子节点的最后一个右兄弟，因此右兄弟节点置-2
		delDepNode.RightBroId = -2
		err = departmentRepository.DB.Table("department").Where("id = ?", delDepNode.ID).Update("right_bro_id", -2).Error
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return err
		}
	}
	return nil
}

func DepartmentUpdateEndpoint(c echo.Context) error {
	id := c.Param("id")
	iId, err := strconv.Atoi(id)
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", nil)
	}
	var item model.Department
	if err := c.Bind(&item); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	// 数据校验
	if err := c.Validate(&item); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	// 不允许修改根部门的名称与上级部门id
	if 0 == iId {
		item.FatherId = -1
		item.Name = "根部门"
	}

	depNode, err := departmentRepository.FindById(int64(iId))
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	// 不允许修改当前用户所属部门的父部门
	account, _ := GetCurrentAccountNew(c)
	if (account.DepartmentName == depNode.Name) && (item.FatherId != depNode.FatherId) {
		return FailWithDataOperate(c, 400, "不允许修改当前用户所属部门的上级部门", "部门机构-修改: 部门名称["+depNode.Name+"], 失败原因[不允许修改当前用户所属部门的上级部门]", err)
	}
	// 部门名称不能与兄弟部门重复
	broDepArr, err := departmentRepository.BroDep(item.FatherId)
	for i := range broDepArr {
		if item.Name == broDepArr[i].Name && int64(iId) != broDepArr[i].ID {
			return FailWithDataOperate(c, 400, "部门名称重复", "部门机构-修改: 部门名称["+depNode.Name+"->"+item.Name+"], 失败原因[部门名称重复]", err)
		}
	}
	// 不允许修改其父部门为 自身或下属任一部门
	var childDepIds []int64
	err = GetChildDepIds(int64(iId), &childDepIds)
	if nil != err {
		log.Errorf("部门机构-修改失败, 获取下级部门失败-GetChildDepIds:%v, 输入参数id: %v", err.Error(), iId)
		return FailWithDataOperate(c, 500, "修改失败", "", nil)
	}
	if IsDepIdBelongDepIds(item.FatherId, childDepIds) {
		return FailWithDataOperate(c, 400, "上级部门设置错误", "部门机构-修改: 部门名称["+depNode.Name+"->"+item.Name+"], 失败原因[修改上级部门为自身或下级所属部门]", err)
	}

	var isUpdateFatherDep bool
	// 如果未修改部门的上级所属部门
	if item.FatherId == depNode.FatherId {
		isUpdateFatherDep = false
	} else {
		isUpdateFatherDep = true

		// 更新原所属父节点及兄弟节点间关系
		err = curNodeUpdateFatherBroRelation(depNode)
		if nil != err {
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
		// 更新其新所属父节点及兄弟节点间关系
		err = addDepartmentUpdateNodeRelation(int64(iId), item.FatherId)
		if nil != err {
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
	}

	departmentM := map[string]interface{}{}
	departmentM["father_id"] = item.FatherId
	departmentM["description"] = item.Description
	departmentM["name"] = item.Name
	err = departmentRepository.DB.Table("department").Where("id = ?", iId).Updates(departmentM).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	oldFatherNode, err := departmentRepository.FindById(depNode.FatherId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}
	if isUpdateFatherDep {
		newFatherNode, err := departmentRepository.FindById(item.FatherId)
		if nil != err {
			log.Errorf("DB Error: %v", err)
		}

		return SuccessWithOperate(c, "部门机构-修改: 部门名称["+depNode.Name+"->"+item.Name+"], 上级部门["+oldFatherNode.Name+"->"+newFatherNode.Name+"]", nil)
	} else {
		return SuccessWithOperate(c, "部门机构-修改: 部门名称["+depNode.Name+"->"+item.Name+"], 上级部门["+oldFatherNode.Name+"]", nil)
	}
}

func UpdateUserAssetAppAppserverDep(dataType, op string, oldDepId, newDepId int64) {
	var upDepIds, oldUpDepIds, newUpDepIds []int64
	var err error
	var count int
	var depNode model.Department

	if constant.UPDATE != op {
		// 非更改部门，而是简单的新增或删除数据
		// 获取数据所属部门及上级部门
		err = UpDepartment(oldDepId, &upDepIds)
		if nil != err {
			log.Error(op + dataType + "后, 更新" + dataType + "所属部门及上级部门的" + dataType + "数失败. 获取上级部门数据错误")
			return
		}

		if constant.ADD == op {
			count = constant.ADDDATA
		} else {
			count = constant.DELETEDATA
		}

		for i := range upDepIds {
			updateMap := map[string]interface{}{}

			depNode, err = departmentRepository.FindById(upDepIds[i])
			if nil != err {
				log.Errorf("DB Error: %v", err)
				log.Error(op + dataType + "后, 更新" + dataType + "所属部门及上级部门的" + dataType + "数失败. 查询部门数据失败")
				return
			}

			switch dataType {
			case constant.USER:
				depNode.UserCount += count
			case constant.ASSET:
				depNode.AssetCount += count
			case constant.APP:
				depNode.AppCount += count
			case constant.APPSERVER:
				depNode.AppServerCount += count
			}
			updateMap["id"] = depNode.ID
			updateMap["name"] = depNode.Name
			updateMap["father_id"] = depNode.FatherId
			updateMap["left_child_id"] = depNode.LeftChildId
			updateMap["right_bro_id"] = depNode.RightBroId
			updateMap["description"] = depNode.Description
			updateMap["asset_count"] = depNode.AssetCount
			updateMap["user_count"] = depNode.UserCount
			updateMap["app_count"] = depNode.AppCount
			updateMap["app_server_count"] = depNode.AppServerCount
			err = departmentRepository.DB.Table("department").Where("id = ?", depNode.ID).Updates(updateMap).Error
			if nil != err {
				log.Errorf("DB Error: %v", err)
				log.Error(op + dataType + "后, 更新" + dataType + "所属部门及上级部门的" + dataType + "数失败. 更新部门数据失败")
				return
			}
		}
	} else {
		// 更改资源所属部门
		// 获取数据所属原部门及原上级部门
		err = UpDepartment(oldDepId, &oldUpDepIds)
		if nil != err {
			log.Error("修改" + dataType + "所属部门后, 更新原部门及原上级部门" + dataType + "数失败. 获取部门数据失败")
			return
		}
		// 数据所属原部门及原上级部门, 数据-1
		for i := range oldUpDepIds {
			updateMap := map[string]interface{}{}

			depNode, err = departmentRepository.FindById(oldUpDepIds[i])
			if nil != err {
				log.Errorf("DB Error: %v", err)
				log.Error("修改" + dataType + "所属部门后, 更新原部门及原上级部门" + dataType + "数失败. 查询部门数据失败")
				return
			}
			switch dataType {
			case constant.USER:
				depNode.UserCount += constant.DELETEDATA
			case constant.ASSET:
				depNode.AssetCount += constant.DELETEDATA
			case constant.APP:
				depNode.AppCount += constant.DELETEDATA
			case constant.APPSERVER:
				depNode.AppServerCount += constant.DELETEDATA
			}
			updateMap["id"] = depNode.ID
			updateMap["name"] = depNode.Name
			updateMap["father_id"] = depNode.FatherId
			updateMap["left_child_id"] = depNode.LeftChildId
			updateMap["right_bro_id"] = depNode.RightBroId
			updateMap["description"] = depNode.Description
			updateMap["asset_count"] = depNode.AssetCount
			updateMap["user_count"] = depNode.UserCount
			updateMap["app_count"] = depNode.AppCount
			updateMap["app_server_count"] = depNode.AppServerCount
			err = departmentRepository.DB.Table("department").Where("id = ?", depNode.ID).Updates(updateMap).Error
			if nil != err {
				log.Errorf("DB Error: %v", err)
				log.Error("修改" + dataType + "所属部门后, 更新原部门及原上级部门" + dataType + "数失败. 更新部门数据失败")
				return
			}
		}

		// 更改资源所属部门
		// 获取数据所属新部门及新的上级部门
		err = UpDepartment(newDepId, &newUpDepIds)
		if nil != err {
			log.Error("修改" + dataType + "所属部门后, 更新新部门及新上级部门" + dataType + "数失败. 获取部门数据失败")
			return
		}
		// 数据所属新部门及新的上级部门, 数据+1
		for i := range newUpDepIds {
			updateMap := map[string]interface{}{}

			depNode, err = departmentRepository.FindById(newUpDepIds[i])
			if nil != err {
				log.Errorf("DB Error: %v", err)
				log.Error("修改" + dataType + "所属部门后, 更新新部门及新上级部门" + dataType + "数失败. 查询部门数据失败")
				return
			}
			switch dataType {
			case constant.USER:
				depNode.UserCount += constant.ADDDATA
			case constant.ASSET:
				depNode.AssetCount += constant.ADDDATA
			case constant.APP:
				depNode.AppCount += constant.ADDDATA
			case constant.APPSERVER:
				depNode.AppServerCount += constant.ADDDATA
			}
			updateMap["id"] = depNode.ID
			updateMap["name"] = depNode.Name
			updateMap["father_id"] = depNode.FatherId
			updateMap["left_child_id"] = depNode.LeftChildId
			updateMap["right_bro_id"] = depNode.RightBroId
			updateMap["description"] = depNode.Description
			updateMap["asset_count"] = depNode.AssetCount
			updateMap["user_count"] = depNode.UserCount
			updateMap["app_count"] = depNode.AppCount
			updateMap["app_server_count"] = depNode.AppServerCount
			err = departmentRepository.DB.Table("department").Where("id = ?", depNode.ID).Updates(updateMap).Error
			if nil != err {
				log.Errorf("DB Error: %v", err)
				log.Error("修改" + dataType + "所属部门后, 更新新部门及新上级部门" + dataType + "数失败. 更新部门数据失败")
				return
			}
		}
	}
}

// 获取输入depId所属部门的上级部门一直至根部门的部门节点，包含depId所属部门与根部门
// upDepIds 为输出型参数
func UpDepartment(depId int64, upDepIds *[]int64) error {
	depNode, err := departmentRepository.FindById(depId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return err
	}
	*upDepIds = append(*upDepIds, depId)

	for -1 != depNode.FatherId {
		depNode, err = departmentRepository.FindById(depNode.FatherId)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return err
		}
		*upDepIds = append(*upDepIds, depNode.ID)
	}
	return nil
}

// 返回输入部门id的部门深度
// 根部门的部门深度为0
func DepLevel(depId int64) (depLevel int, err error) {
	depLevel = 0
	depNode, err := departmentRepository.FindById(depId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return -1, err
	}
	for 0 != depNode.ID {
		depNode, err = departmentRepository.FindById(depNode.FatherId)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return -1, err
		}
		depLevel += 1
	}
	return depLevel, nil
}

func DepartmentDownloadTemplateEndpoint(c echo.Context) error {
	departmentForExport := []string{"部门名称(必填)", "上级部门(必填)(如果部门名称重复请在名称后加{{id}} 例如:测试部{{380}})", "描述"}
	departmentFileNameForExport := "部门机构"
	file, err := utils.CreateTemplateFile(departmentFileNameForExport, departmentForExport)
	if err != nil {
		log.Errorf("CreateExcelFile Error: %v", err)
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	fileName := "部门机构导入模板.xlsx"
	var buff bytes.Buffer
	if err = file.Write(&buff); err != nil {
		log.Errorf("Write Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "", err)
	}

	user, _ := GetCurrentAccountNew(c)
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "",
		Created:         utils.NowJsonTime(),
		Users:           user.Username,
		Names:           user.Nickname,
		LogContents:     "部门机构-下载: 模板文件",
		Result:          "成功",
	}
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
}

func DepartmentImportEndpoint(c echo.Context) error {
	account, isSuccess := GetCurrentAccountNew(c)
	if !isSuccess {
		log.Errorf("GetCurrentAccountNew Error")
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}

	file, err := c.FormFile("file")
	if err != nil {
		log.Errorf("FormFile Error: %v", err)
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
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

	// 导入文件有几行, 0行文件为空, 1行代表只有标题行也是"空"
	total := len(records)
	if total <= 1 {
		return FailWithDataOperate(c, 400, "导入文件数据为空", "部门机构-导入: 文件名["+file.Filename+"], 失败原因[导入文件数据为空]", nil)
	}

	isCover := c.FormValue("is_cover")
	if "true" == isCover {
		// 删除当前部门下属所有部门及部门关联资源数据
		var delDepIdsIncludeOwn []int64
		err = GetChildDepIds(account.DepartmentId, &delDepIdsIncludeOwn)
		if nil != err {
			log.Errorf("GetRows Error: %v", err)
			return FailWithDataOperate(c, 500, "导入失败", "", nil)
		}
		delDepIdsExcludeOwn := DepRemoveDuplicates(account.DepartmentId, delDepIdsIncludeOwn)
		deleteDepsData(delDepIdsExcludeOwn)

		// 重新获取当前部门资源数并更新
		userCount, err := userNewRepository.FindUserCountByDepId(account.DepartmentId)
		if err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "导入失败", "", nil)
		}
		assetCount, err := newAssetRepository.GetAssetCountByDepId(context.TODO(), account.DepartmentId)
		if err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "导入失败", "", nil)
		}
		appCount, err := newApplicationRepository.GetAppCountByDepartmentId(context.TODO(), account.DepartmentId)
		if err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "导入失败", "", nil)
		}
		appServerCount, err := newApplicationServerRepository.GetAppServerByDepId(context.TODO(), account.DepartmentId)
		if err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "导入失败", "", nil)
		}

		currentDepM := map[string]interface{}{
			"left_child_id":    -2,
			"user_count":       userCount,
			"asset_count":      assetCount,
			"app_count":        appCount,
			"app_server_count": appServerCount,
		}
		err = departmentRepository.DB.Model(model.Department{}).Where("id = ?", account.DepartmentId).Updates(currentDepM).Error
		if err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "导入失败", "", nil)
		}
	}

	var success, fail string
	for i := 1; i < total; i++ {
		// 这里的record不会等于0, 如果等于0也即意味着这行没任何内容也即意外着它不会"存在"于total中
		// 这里需注意, 导入的表格文件如果有1列或2列, 则没有如下switch语句的话, record[2]会直接索引越界, 也即record[2]不会是""
		record := records[i]
		switch len(record) {
		case 1:
			// 此行数据必填项不完整, 跳过
			continue
		case 2:
			record = append(record, "")
		}

		var depIdsIncludeOwn []int64
		err = GetChildDepIds(account.DepartmentId, &depIdsIncludeOwn)
		if nil != err {
			log.Errorf("GetRows Error: %v", err)
			return FailWithDataOperate(c, 500, "导入失败", "", nil)
		}

		department := model.Department{
			Name:        record[0],
			Description: record[2],
		}

		if strings.Contains(record[1], "{{") && strings.Contains(record[1], "}}") {
			indexBegin := strings.LastIndex(record[1], "{")
			indexEnd := strings.Index(record[1], "}")
			fatherId, err := strconv.Atoi(record[1][indexBegin+1 : indexEnd])
			if nil != err {
				log.Errorf("Atoi Error: %v", err)
				fail += record[0] + "id转换失败, "
				continue
			}
			department.FatherId = int64(fatherId)
		} else {
			dep, err := departmentRepository.FindByName(record[1])
			if nil != err {
				if gorm.ErrRecordNotFound == err {
					fail += record[0] + "父部门不存在, "
					continue
				}
				log.Errorf("DB Error: %v", err)
				fail += record[0] + "父部门查询失败, "
				continue
			}
			department.FatherId = dep.ID
		}

		if !IsDepIdBelongDepIds(department.FatherId, depIdsIncludeOwn) {
			fail += record[0] + "父部门不属于当前部门, "
			continue
		}

		isExist := false
		// 不能与兄弟部门重名
		broDepArr, err := departmentRepository.BroDep(department.FatherId)
		for i := range broDepArr {
			if department.Name == broDepArr[i].Name {
				isExist = true
				break
			}
		}

		if isExist {
			fail += record[0] + "部门已存在, "
			continue
		}

		if err := departmentRepository.Create(&department); nil != err {
			log.Errorf("DB Error: %v", err)
			fail += record[0] + "新建失败, "
			continue
		}
		dep, err := departmentRepository.FindByNameFatherId(department.Name, department.FatherId)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			fail += record[0] + "查询父部门失败, "
			return FailWithDataOperate(c, 500, "导入失败", "", nil)
		}
		err = addDepartmentUpdateNodeRelation(dep.ID, department.FatherId)
		if nil != err {
			log.Errorf("addDepartmentUpdateNodeRelation Error: %v", err)
			fail += record[0] + "新建失败, "
			return FailWithDataOperate(c, 500, "导入失败", "", nil)
		}
		success += record[0] + "导入成功, "
	}

	return SuccessWithOperate(c, "部门机构-导入: 文件名称["+file.Filename+"], 成功["+success+"], 失败["+fail+"]", nil)
}

func IsDepIdBelongDepIds(depId int64, depIds []int64) bool {
	isBelong := false
	for i := range depIds {
		if depId == depIds[i] {
			isBelong = true
			break
		}
	}
	return isBelong
}

func DepRemoveDuplicates(depId int64, depIdArr []int64) (removeDuplicatesDepIdArr []int64) {
	for i := range depIdArr {
		if depId == depIdArr[i] {
			continue
		}
		removeDuplicatesDepIdArr = append(removeDuplicatesDepIdArr, depIdArr[i])
	}
	return
}

func DepartmentExportEndpoint(c echo.Context) error {
	auto := c.QueryParam("auto")
	departmentName := c.QueryParam("department_name")
	departmentId := c.QueryParam("department_id")

	name := "部门机构"
	header := []string{"部门名称", "部门ID", "用户数", "设备数", "应用数", "应用服务器数", "描述"}
	result := make([][]string, 0)

	account, isSuccess := GetCurrentAccountNew(c)
	if !isSuccess {
		log.Errorf("GetCurrentAccountNew Error")
		return FailWithDataOperate(c, 500, "导出失败", "", nil)
	}

	if "" == auto && "" == departmentName && "" == departmentId {
		var departTreeArr model.DepartmentTree
		fatNode, err := departmentRepository.FindById(account.DepartmentId)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "导出失败", "", nil)
		}

		fatNodeArr := &departTreeArr
		err = FindDepartmentExport(fatNode, fatNodeArr, &result)
		if nil != err {
			return FailWithDataOperate(c, 500, "导出失败", "", nil)
		}
	} else {
		_, err := FindDepartmentWithConditionExport(auto, departmentName, departmentId, account.DepartmentId, &result)
		if nil != err {
			log.Errorf("FindDepartmentWithCondition Error: %v", err)
			return FailWithDataOperate(c, 500, "导出失败", "", nil)
		}
	}

	file, err := utils.CreateExcelFile(name, header, result)
	if err != nil {
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	fileName := "部门机构.xlsx"

	//将数据存入buffer
	var buff bytes.Buffer
	if err = file.Write(&buff); err != nil {
		log.Errorf("Write Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "", err)
	}

	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "已弃用",
		Created:         utils.NowJsonTime(),
		Users:           account.Username,
		Names:           account.Nickname,
		LogContents:     "部门机构-导出: 导出部门机构数据文件成功",
		Result:          "成功",
	}
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}

	//设置请求头  使用浏览器下载
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
}

func FindDepartmentExport(fatNode model.Department, fatNodeArr *model.DepartmentTree, result *[][]string) error {
	// 外层函数的departTreeArr变量为给前端返回数据，  外层函数的fatNode变量为当前用户所属部门的Node信息
	// 先把自身fatNode信息补全，然后找到所有孩子，遍历孩子，再把孩子当成新的fatNode递归获取所有部门信息
	fatNodeArr.ID = fatNode.ID
	fatNodeArr.FatherId = fatNode.FatherId
	fatNodeArr.Name = fatNode.Name
	fatNodeArr.Description = fatNode.Description

	var childIds []int64
	err := GetChildDepIds(fatNode.ID, &childIds)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return err
	}

	userCount, err := userNewRepository.FindUserCountByDepIds(childIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		userCount = 0
	}
	appCount, err := newApplicationRepository.FindAppCountByDepartmentIds(context.TODO(), childIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		appCount = 0
	}
	assetCount, err := newAssetRepository.GetAssetCountByDepIds(context.TODO(), childIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		assetCount = 0
	}
	appServerCount, err := newApplicationServerRepository.FindAppSerCountByDepIds(context.TODO(), childIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		appServerCount = 0
	}
	fatNodeArr.AppCount = int(appCount)
	fatNodeArr.AssetCount = int(assetCount)
	fatNodeArr.UserCount = int(userCount)
	fatNodeArr.AppServerCount = int(appServerCount)

	fatNodeArr.ChildArr = nil

	depChainName, err := DepChainName(fatNode.ID)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return err
	}

	var resultTemp []string
	resultTemp = append(resultTemp, depChainName[:len(depChainName)-1])
	resultTemp = append(resultTemp, strconv.Itoa(int(fatNode.ID)))
	resultTemp = append(resultTemp, strconv.Itoa(fatNodeArr.UserCount))
	resultTemp = append(resultTemp, strconv.Itoa(fatNode.AssetCount))
	resultTemp = append(resultTemp, strconv.Itoa(fatNodeArr.AppCount))
	resultTemp = append(resultTemp, strconv.Itoa(fatNodeArr.AppServerCount))
	resultTemp = append(resultTemp, fatNode.Description)
	*result = append(*result, resultTemp)

	if -2 == fatNode.LeftChildId {
		return nil
	} else {
		// 找到第一个孩子
		childNode, err := departmentRepository.FindById(fatNode.LeftChildId)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return err
		}

		for true {
			// 找到所有孩子
			fatNodeArr.ChildArr = append(fatNodeArr.ChildArr, model.DepartmentTree{ID: childNode.ID})
			if -2 == childNode.RightBroId {
				break
			} else {
				childNode, err = departmentRepository.FindById(childNode.RightBroId)
				if nil != err {
					log.Errorf("DB Error: %v", err)
					return err
				}
			}
		}
		for i := range fatNodeArr.ChildArr {
			fatNode, err = departmentRepository.FindById(fatNodeArr.ChildArr[i].ID)
			if nil != err {
				log.Errorf("DB Error: %v", err)
				return err
			}
			FindDepartmentExport(fatNode, &fatNodeArr.ChildArr[i], result)
		}
	}

	return nil
}

func FindDepartmentNotInIdsExport(fatNode model.Department, fatNodeArr *model.DepartmentTree, notDepIds []int64, result *[][]string) (err error) {
	fatNodeArr.ID = fatNode.ID
	fatNodeArr.FatherId = fatNode.FatherId
	fatNodeArr.Name = fatNode.Name
	fatNodeArr.Description = fatNode.Description

	var childIds []int64
	err = GetChildDepIds(fatNode.ID, &childIds)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return err
	}

	userCount, err := userNewRepository.FindUserCountByDepIds(childIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		userCount = 0
	}
	appCount, err := newApplicationRepository.FindAppCountByDepartmentIds(context.TODO(), childIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		appCount = 0
	}
	assetCount, err := newAssetRepository.GetAssetCountByDepIds(context.TODO(), childIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		assetCount = 0
	}
	appServerCount, err := newApplicationServerRepository.FindAppSerCountByDepIds(context.TODO(), childIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		appServerCount = 0
	}
	fatNodeArr.AppCount = int(appCount)
	fatNodeArr.AssetCount = int(assetCount)
	fatNodeArr.UserCount = int(userCount)
	fatNodeArr.AppServerCount = int(appServerCount)

	fatNodeArr.ChildArr = nil

	depChainName, err := DepChainName(fatNode.ID)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return err
	}
	var resultTemp []string
	resultTemp = append(resultTemp, depChainName[:len(depChainName)-1])
	resultTemp = append(resultTemp, strconv.Itoa(int(fatNode.ID)))
	resultTemp = append(resultTemp, strconv.Itoa(fatNodeArr.UserCount))
	resultTemp = append(resultTemp, strconv.Itoa(fatNode.AssetCount))
	resultTemp = append(resultTemp, strconv.Itoa(fatNodeArr.AppCount))
	resultTemp = append(resultTemp, strconv.Itoa(fatNodeArr.AppServerCount))
	resultTemp = append(resultTemp, fatNode.Description)
	*result = append(*result, resultTemp)
	if -2 == fatNode.LeftChildId {
		return nil
	} else {
		// 找到第一个孩子
		childNode, err := departmentRepository.FindById(fatNode.LeftChildId)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return err
		}

		for true {
			// 找到所有孩子
			if idIsExistIds(childNode.ID, notDepIds) {
				fatNodeArr.ChildArr = append(fatNodeArr.ChildArr, model.DepartmentTree{ID: childNode.ID})
			}
			if -2 == childNode.RightBroId {
				break
			} else {
				childNode, err = departmentRepository.FindById(childNode.RightBroId)
				if nil != err {
					log.Errorf("DB Error: %v", err)
					return err
				}
			}
		}
		for i := range fatNodeArr.ChildArr {
			fatNode, err = departmentRepository.FindById(fatNodeArr.ChildArr[i].ID)
			if nil != err {
				log.Errorf("DB Error: %v", err)
				return err
			}
			FindDepartmentNotInIdsExport(fatNode, &fatNodeArr.ChildArr[i], notDepIds, result)
		}
	}

	return nil
}

func FindDepartmentWithConditionExport(auto, departmentName, departmentId string, genDepId int64, result *[][]string) (departTreeWithCondition model.DepartmentTree, err error) {
	var depIds []int64
	if "" != auto {
		sql := "SELECT id FROM department WHERE "
		whereCondition := " id LIKE '%" + auto + "%' OR name LIKE '%" + auto + "%' OR asset_count LIKE '%" + auto + "%' OR user_count LIKE '%" + auto + "%' OR app_count LIKE '%" + auto + "%' OR app_server_count LIKE '%" + auto + "%' OR description LIKE '%" + auto + "%'"
		sql += whereCondition
		err = departmentRepository.DB.Raw(sql).Find(&depIds).Error
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return model.DepartmentTree{}, err
		}
	} else if "" != departmentName {
		depIds, err = departmentRepository.FindIdsByVagueName(departmentName)
	} else if "" != departmentId {
		iDepartmentId, err := strconv.Atoi(departmentId)
		if nil != err {
			log.Errorf("Atoi Error: %v", err)
			return model.DepartmentTree{}, err
		}

		_, err = departmentRepository.FindById(int64(iDepartmentId))
		depIds = append(depIds, int64(iDepartmentId))
	}
	if nil != err {
		if err == gorm.ErrRecordNotFound {
			return model.DepartmentTree{}, err
		}
		log.Errorf("DB Error: %v", err)
		return model.DepartmentTree{}, err
	}

	var fatherIdsAll, fatIds []int64
	for i := range depIds {
		err = UpDepartment(depIds[i], &fatIds)
		if nil != err {
			log.Errorf("UpDepartment Error: %v", err)
			return model.DepartmentTree{}, err
		}
		fatherIdsAll = append(fatherIdsAll, fatIds...)
	}
	if 0 == len(fatherIdsAll) {
		return model.DepartmentTree{}, nil
	}
	// fatAllIds去重
	fatherIdsAll = RemoveDuplicates(fatherIdsAll)

	genNode, err := departmentRepository.FindById(genDepId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return model.DepartmentTree{}, err
	}
	err = FindDepartmentNotInIdsExport(genNode, &departTreeWithCondition, fatherIdsAll, result)
	if nil != err {
		return model.DepartmentTree{}, err
	}

	return departTreeWithCondition, nil
}
