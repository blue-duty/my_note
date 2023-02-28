package api

import (
	"context"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/log"
	"tkbastion/pkg/validator"
	"tkbastion/server/dto"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// NewAssetGroupPagingEndpoint 资产组列表
func NewAssetGroupPagingEndpoint(c echo.Context) error {
	u, _ := GetCurrentAccountNew(c)
	auto := c.QueryParam("auto")
	agn := c.QueryParam("asset_group_name")
	d := c.QueryParam("department")
	var departmentIds []int64
	err := GetChildDepIds(u.DepartmentId, &departmentIds)
	if err != nil {
		return FailWithData(c, 500, "获取失败", nil)
	}

	assetGroupList, err := newAssetGroupRepository.GetAssetGroupListBydsadd(context.TODO(), departmentIds, auto, agn, d)
	if err != nil {
		log.Errorf("GetAssetGroupList error: %v", err)
		return Fail(c, 500, "获取设备组列表失败")
	}

	return Success(c, assetGroupList)
}

// NewAssetGroupCreateEndpoint 新增设备组
func NewAssetGroupCreateEndpoint(c echo.Context) error {
	u, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithData(c, 401, "请重新登陆", nil)
	}
	var req dto.AssetGroupCreateRequest
	if err := c.Bind(&req); err != nil {
		log.Errorf("Bind error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", nil)
	}

	// 数据校验
	if err := c.Validate(&req); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	req.DepartmentId = u.DepartmentId
	req.Department = u.DepartmentName

	_, err := newAssetGroupRepository.GetAssetGroupByName(context.TODO(), req.Name)
	if err == nil {
		return FailWithDataOperate(c, 500, "新增失败", "设备组列表-新增: 新增设备组, 失败原因[设备组名称"+req.Name+"已存在]", nil)
	} else if err != nil && err != gorm.ErrRecordNotFound {
		log.Errorf("DB Error: %v", err)
		return FailWithData(c, 500, "新增失败", nil)
	}
	err = newAssetGroupRepository.CreateAssetGroup(context.TODO(), &req)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithData(c, 500, "新增失败", nil)
	}
	return SuccessWithOperate(c, "设备组列表-新增: 新增设备组, 设备组名称["+req.Name+"]", nil)
}

// NewAssetGroupUpdateEndpoint 更新设备组
func NewAssetGroupUpdateEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithData(c, 401, "请重新登陆", nil)
	}
	id := c.Param("id")
	var req dto.AssetGroupUpdateRequest
	if err := c.Bind(&req); err != nil {
		log.Errorf("Bind error: %v", err)
		return FailWithData(c, 500, "更新失败", nil)
	}

	// 数据校验
	if err := c.Validate(&req); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	assetGroup, err := newAssetGroupRepository.GetAssetGroupById(context.TODO(), id)
	if err != nil {
		log.Errorf("GetAssetGroupById error: %v", err)
		return FailWithData(c, 500, "更新失败", nil)
	}
	_, err = newAssetGroupRepository.GetAssetGroupByNameAndId(context.TODO(), req.Name, id)
	if err == nil {
		return FailWithDataOperate(c, 500, "更新失败", "设备组列表-更新: 更新设备组, 失败原因[设备组名称"+req.Name+"已存在]", nil)
	} else if err != nil && err != gorm.ErrRecordNotFound {
		log.Errorf("DB Error: %v", err)
		return FailWithData(c, 500, "更新失败", nil)
	}
	err = newAssetGroupRepository.UpdateAssetGroup(context.TODO(), &req)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithData(c, 500, "更新失败", nil)
	}
	return SuccessWithOperate(c, "设备组列表-更新: 更新设备组, 设备组名称["+assetGroup.Name+"->"+req.Name+"]", nil)
}

// NewAssetGroupDeleteEndpoint 删除设备组
func NewAssetGroupDeleteEndpoint(c echo.Context) error {
	//u, _ := GetCurrentAccount(c)
	id := c.Param("id")
	assetGroup, err := newAssetGroupRepository.GetAssetGroupById(context.TODO(), id)
	if err != nil {
		log.Errorf("GetAssetGroupById error: %v", err)
		return FailWithData(c, 500, "删除失败", nil)
	}
	err = newAssetGroupRepository.DeleteAssetGroup(context.TODO(), id)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithData(c, 500, "删除失败", nil)
	}
	return SuccessWithOperate(c, "设备组列表-删除: 删除设备组, 设备组名称["+assetGroup.Name+"]", nil)
}

// NewAssetGroupBatchDeleteEndpoint 批量删除设备组
func NewAssetGroupBatchDeleteEndpoint(c echo.Context) error {
	//u, _ := GetCurrentAccount(c)
	id := c.QueryParam("id")
	ids := utils.IdHandle(id)
	ags, err := newAssetGroupRepository.GetAssetGroupByIds(context.TODO(), ids)
	if err != nil {
		log.Errorf("GetAssetGroupByIds error: %v", err)
		return FailWithData(c, 500, "批量删除失败", nil)
	}
	agns := ""
	for _, ag := range ags {
		agns += ag.Name + ","
	}
	err = newAssetGroupRepository.DeleteAssetGroupByIds(context.TODO(), ids)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithData(c, 500, "删除失败", nil)
	}
	return SuccessWithOperate(c, "设备分组-批量删除: 批量删除设备组, 设备组名称["+agns+"]", nil)
}

// NewAssetGroupAssetEndpoint 设备组关联设备
func NewAssetGroupAssetEndpoint(c echo.Context) error {
	//u, _ := GetCurrentAccount(c)
	//var req dto.AssetGroupAssetRequest
	//if err := c.Bind(&req); err != nil {
	//	log.Errorf("Bind error: %v", err)
	//	return FailWithData(c, 500, "关联失败", nil)
	//}
	gid := c.Param("id")
	id := c.QueryParam("aid")
	_, f := GetCurrentAccountNew(c)
	if f == false {
		return Fail(c, 401, "请登录")
	}

	assetGroup, err := newAssetGroupRepository.GetAssetGroupById(context.TODO(), gid)
	if err != nil {
		log.Errorf("GetAssetGroupById error: %v", err)
		return FailWithData(c, 500, "关联失败", nil)
	}

	if id == "" {
		err := newAssetGroupRepository.DeleteAllAssets(context.TODO(), gid)
		if err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithData(c, 500, "关联失败", nil)
		}
		return SuccessWithOperate(c, "设备组关联设备-删除: 删除所有设备组关联设备, 设备组["+assetGroup.Name+"]", nil)
	}
	ids := utils.IdHandle(id)

	assets, err := newAssetRepository.GetAssetByIds(context.TODO(), ids)
	if err != nil {
		log.Errorf("GetAssetByIds error: %v", err)
		return FailWithData(c, 500, "关联失败", nil)
	}
	assetNames := ""
	for _, asset := range assets {
		assetNames += asset.Name + ","
	}

	err = newAssetGroupRepository.UpdateAssetGroupAsset(context.TODO(), gid, ids)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithData(c, 500, "关联失败", nil)
	}
	return SuccessWithOperate(c, "设备组列表-关联设备: 关联设备, 设备组名称["+assetGroup.Name+"], 关联设备名称["+assetNames+"]", nil)
}

func NewAssetGroupAssetPagingEndpoint(c echo.Context) error {
	u, _ := GetCurrentAccountNew(c)
	var departmentIds []int64
	err := GetChildDepIds(u.DepartmentId, &departmentIds)
	if err != nil {
		return FailWithData(c, 500, "获取部门失败", nil)
	}

	assetAllArr, err := newAssetRepository.GetPassportByDepartmentIds(context.TODO(), departmentIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	var resp = make([]dto.ForRelate, len(assetAllArr))

	for i := range assetAllArr {
		depChinaName, err := DepChainName(assetAllArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		resp[i].ID = assetAllArr[i].ID
		resp[i].Name = assetAllArr[i].AssetName + "[" + assetAllArr[i].Ip + "]" + "[" + assetAllArr[i].Passport + "]" + "[" + assetAllArr[i].Protocol + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}
	return SuccessWithOperate(c, "", resp)
}

// NewAssetGroupAssetListEndpoint 设备组查询关联设备
func NewAssetGroupAssetListEndpoint(c echo.Context) error {
	//u, _ := GetCurrentAccount(c)
	id := c.Param("id")
	assetGroup, err := newAssetGroupRepository.GetAssetGroupAsset(context.TODO(), id)
	if err != nil {
		log.Errorf("GetAssetGroupAsset error: %v", err)
		return FailWithData(c, 500, "查询失败", nil)
	}

	var resp = make([]string, len(assetGroup))
	for i := range assetGroup {
		resp[i] = assetGroup[i].AssetId
	}
	return SuccessWithOperate(c, "", resp)
}

func NewAssetGroupAssetsEndpoint(c echo.Context) error {
	//u, _ := GetCurrentAccount(c)
	id := c.Param("id")
	assetGroup, err := newAssetGroupRepository.GetAssetGroupById(context.TODO(), id)
	if err != nil {
		log.Errorf("GetAssetGroupById error: %v", err)
		return FailWithData(c, 500, "查询失败", nil)
	}

	// 获取部门机构
	var departmentIds []int64
	err = GetChildDepIds(assetGroup.DepartmentId, &departmentIds)
	if err != nil {
		return FailWithData(c, 500, "获取部门失败", nil)
	}
	assetAllArr, err := newAssetRepository.GetPassportByDepartmentIds(context.TODO(), departmentIds)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	var resp = make([]dto.ForRelate, len(assetAllArr))

	for i := range assetAllArr {
		depChinaName, err := DepChainName(assetAllArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		resp[i].ID = assetAllArr[i].ID
		resp[i].Name = assetAllArr[i].AssetName + "[" + assetAllArr[i].Ip + "]" + "[" + assetAllArr[i].Passport + "]" + "[" + assetAllArr[i].Protocol + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
	}
	return SuccessWithOperate(c, "", resp)
}
