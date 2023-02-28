package api

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strings"
	"tkbastion/pkg/constant"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/pkg/validator"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func NewJopPagingEndpoint(c echo.Context) error {
	var req dto.NewJobForSearch
	req.Name = c.QueryParam("name")
	req.Department = c.QueryParam("department")
	req.Content = c.QueryParam("content")
	req.RunTimeType = c.QueryParam("runTimeType")
	req.Auto = c.QueryParam("auto")

	u, _ := GetCurrentAccountNew(c)
	var departmentIds []int64
	err := GetChildDepIds(u.DepartmentId, &departmentIds)
	req.DepartmentIds = departmentIds

	resp, err := newjobRepository.FindAll(context.TODO(), &req)
	if err != nil {
		return FailWithDataOperate(c, 500, "查询失败", "任务列表-查询: 查询计划任务, 失败原因["+err.Error()+"]", nil)
	}

	if resp == nil {
		resp = []dto.NewJobForPage{}
	}

	return c.JSON(200, H{
		"code":    1,
		"success": true,
		"message": "成功",
		"data":    resp,
	})
}

func NewJopCreateEndpoint(c echo.Context) error {
	var reqj dto.NewJobForJson
	if err := c.Bind(&reqj); err != nil {
		log.Errorf("NewJopCreateEndpoint Bind error: %v", err)
		return FailWithData(c, 400, "新增失败", nil)
	}

	if err := c.Validate(&reqj); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	req := reqj.ToNewJobForCreate()

	u, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "未登录")
	}
	req.DepartmentID = u.DepartmentId
	req.Department = u.DepartmentName

	// 查询名称是否重复
	_, err := newjobRepository.FindByName(context.TODO(), req.Name)
	if err == nil {
		return FailWithDataOperate(c, 500, "任务名称已存在", "任务列表-新增: 新增计划任务, 失败原因[任务名称"+req.Name+"已存在]", nil)
	} else if err != gorm.ErrRecordNotFound {
		log.Errorf("NewJopCreateEndpoint FindByNameId error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "任务列表-新增: 新增计划任务, 失败原因["+err.Error()+"]", nil)
	}

	if req.RunTimeType == "Periodic" {
		// 查询周期是否正确
		if req.Periodic < 1 {
			return FailWithDataOperate(c, 500, "周期必须大于0", "任务列表-新增: 新增计划任务, 失败原因[周期必须大于0]", nil)
		}
	}

	req.ID = utils.UUID()

	// 保存shell脚本
	if req.RunType == "shell" {
		shellScript, err := c.FormFile("shellScript")
		if err != nil {
			log.Errorf("NewJopCreateEndpoint FormFile error: %v", err)
			return FailWithDataOperate(c, 500, "新增失败", "任务列表-新增: 新增计划任务, 失败原因["+err.Error()+"]", nil)
		}
		if shellScript == nil {
			return FailWithDataOperate(c, 500, "新增失败", "任务列表-新增: 新增计划任务, 失败原因[shell脚本不能为空]", nil)
		}
		// 保存shell脚本
		src, err := shellScript.Open()
		if nil != err {
			log.Error("任务列表-新增: 新增计划任务, 失败原因[打开文件失败]")
			return FailWithDataOperate(c, 500, "新增失败", "任务列表-新增: 新增计划任务, 失败原因[shell脚本打开失败]", nil)
		}
		defer func(src multipart.File) {
			err := src.Close()
			if err != nil {
				log.Errorf("Close Error: %v", err)
			}
		}(src)

		if !utils.FileExists(constant.ShellPath) {
			err := os.MkdirAll(constant.ShellPath, 0755)
			if err != nil {
				log.Errorf("MkdirAll Error: %v", err)
				return FailWithDataOperate(c, 500, "新增失败", "任务列表-新增: 新增计划任务, 失败原因[创建目录失败]", nil)
			}
		}

		dst, err := os.Create(path.Join(constant.ShellPath, shellScript.Filename+"."+req.ID))
		if nil != err {
			log.Error("新增计划任务, 失败原因[创建文件失败]")
			return FailWithDataOperate(c, 500, "新增失败", "任务列表-新增: 新增计划任务, 失败原因[shell脚本创建失败]", nil)
		}
		defer func(dst *os.File) {
			err := dst.Close()
			if err != nil {
				log.Errorf("Close Error: %v", err)
			}
		}(dst)
		if _, err = io.Copy(dst, src); err != nil {
			log.Error("新增计划任务, 失败原因[写入文件失败]")
			return FailWithDataOperate(c, 500, "写入文件失败", "", nil)
		}

		req.ShellName = shellScript.Filename + "." + req.ID
	} else {
		req.ShellName = ""
	}

	if req.RunTimeType != "Manual" {
		err = newJobService.NewPlanJob(req)
		if err != nil {
			log.Errorf("NewJopCreateEndpoint NewPlanJob error: %v", err)
			return FailWithDataOperate(c, 500, "新增失败", "任务列表-新增: 新增计划任务, 失败原因["+err.Error()+"]", nil)
		}
	}
	err = newjobRepository.Create(context.TODO(), &req)
	if err != nil {
		log.Errorf("NewJopCreateEndpoint Create error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "任务列表-新增: 新增计划任务, 失败原因["+err.Error()+"]", nil)
	}
	return SuccessWithOperate(c, "任务列表-新增: 新增计划任务, 名称["+req.Name+"]", nil)
}

func NewJopUpdateEndpoint(c echo.Context) error {
	var reqj dto.NewJobForJsonUpdate
	if err := c.Bind(&reqj); err != nil {
		log.Error("任务列表-修改: 修改计划任务, 失败原因[参数绑定失败]", err)
		return FailWithData(c, 400, "修改失败", nil)
	}

	req := reqj.ToNewJobForUpdate()

	// 数据校验
	if err := c.Validate(&req); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	// 查询名称是否重复
	_, err := newjobRepository.FindByIdName(context.TODO(), req.Name, req.ID)
	if err == nil {
		return FailWithDataOperate(c, 500, "任务名称已存在", "任务列表-修改: 修改计划任务, 失败原因[任务名称"+req.Name+"已存在]", nil)
	} else if err != gorm.ErrRecordNotFound {
		log.Error("任务列表-修改: 修改计划任务, 失败原因[查询任务名称失败]")
		return FailWithDataOperate(c, 500, "修改失败", "任务列表-修改: 修改计划任务, 失败原因["+err.Error()+"]", nil)
	}

	if req.RunTimeType == "Periodic" {
		// 查询周期是否正确
		if req.Periodic < 1 {
			return FailWithDataOperate(c, 500, "周期必须大于0", "任务列表-修改: 修改计划任务, 失败原因[周期必须大于0]", nil)
		}
	}

	job, err := newjobRepository.FindById(context.TODO(), req.ID)
	if err != nil {
		log.Error("任务列表-修改: 修改计划任务, 失败原因[任务不存在]")
		return FailWithDataOperate(c, 500, "修改失败", "任务列表-修改: 修改计划任务, 失败原因["+err.Error()+"]", nil)
	}

	if req.RunType == "shell" {
		shellScript, err := c.FormFile("shellScript")
		if err == nil {
			if shellScript != nil {
				// 删除原来的shell脚本
				if job.ShellName != "" {
					err = os.Remove(path.Join(constant.ShellPath, job.ShellName))
					if err != nil {
						log.Errorf("任务列表-修改: 修改计划任务, 失败原因[删除原来的shell脚本失败]")
						return FailWithDataOperate(c, 500, "修改失败", "任务列表-修改: 修改计划任务, 失败原因["+err.Error()+"]", nil)
					}
				}
				// 创建新的shell脚本
				// 保存shell脚本
				src, err := shellScript.Open()
				if nil != err {
					log.Error("任务列表-新增: 新增计划任务, 失败原因[打开文件失败]")
					return FailWithDataOperate(c, 500, "新增失败", "任务列表-新增: 新增计划任务, 失败原因[shell脚本打开失败]", nil)
				}
				defer func(src multipart.File) {
					err := src.Close()
					if err != nil {
						log.Errorf("Close Error: %v", err)
					}
				}(src)

				dst, err := os.Create(path.Join(constant.ShellPath, shellScript.Filename+"."+req.ID))
				if nil != err {
					log.Error("新增计划任务, 失败原因[创建文件失败]")
					return FailWithDataOperate(c, 500, "新增失败", "任务列表-新增: 新增计划任务, 失败原因[shell脚本创建失败]", nil)
				}
				defer func(dst *os.File) {
					err := dst.Close()
					if err != nil {
						log.Errorf("Close Error: %v", err)
					}
				}(dst)
				if _, err = io.Copy(dst, src); err != nil {
					log.Error("新增计划任务, 失败原因[写入文件失败]")
					return FailWithDataOperate(c, 500, "写入文件失败", "", nil)
				}

				req.ShellName = shellScript.Filename + "." + req.ID
			}
		} else if err.Error() != "http: no such file" {
			log.Error("任务列表-修改: 修改计划任务, 失败原因[shell脚本获取失败]")
			return FailWithDataOperate(c, 500, "修改失败", "任务列表-修改: 修改计划任务, 失败原因["+err.Error()+"]", nil)
		}
	} else {
		req.ShellName = ""
	}
	r := dto.NewJobForCreate{
		ID:           job.ID,
		Name:         req.Name,
		RunType:      req.RunType,
		RunTime:      req.RunTime,
		RunTimeType:  req.RunTimeType,
		Command:      req.Command,
		ShellName:    req.ShellName,
		StartAt:      req.StartAt,
		EndAt:        req.EndAt,
		Periodic:     req.Periodic,
		PeriodicType: req.PeriodicType,
	}

	if req.RunTimeType != "Manual" {
		err = newJobService.NewPlanJob(r)
		if err != nil {
			log.Error("任务列表-修改: 修改计划任务, 失败原因[修改计划任务失败]")
			return FailWithDataOperate(c, 500, "修改失败", "任务列表-修改: 修改计划任务, 失败原因["+err.Error()+"]", nil)
		}
	}

	err = newjobRepository.Update(context.TODO(), &req)
	if err != nil {
		log.Error("任务列表-修改: 修改计划任务, 失败原因[修改计划任务失败]")
		return FailWithDataOperate(c, 500, "修改失败", "任务列表-修改: 修改计划任务, 失败原因["+err.Error()+"]", nil)
	}

	return SuccessWithOperate(c, "任务列表-修改: 修改计划任务, 名称["+job.Name+"->"+req.Name+"]", nil)
}

func NewJopDeleteEndpoint(c echo.Context) error {
	id := c.Param("id")

	job, err := newjobRepository.FindById(context.TODO(), id)
	if err != nil {
		log.Errorf("NewJopDeleteEndpoint FindById error: %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "任务列表-删除: 删除计划任务, 失败原因["+err.Error()+"]", nil)
	}

	err = global.SCHEDULER.RemoveByTag(job.ID)
	if err != nil {
		log.Errorf("删除计划任务, 失败原因[%v]", err)
	}

	err = newjobRepository.Delete(context.TODO(), id)
	if err != nil {
		log.Errorf("NewJopDeleteEndpoint Delete error: %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "任务列表-删除: 删除计划任务, 失败原因["+err.Error()+"]", nil)
	}

	return SuccessWithOperate(c, "任务列表-删除: 删除计划任务, 名称["+job.Name+"]", nil)
}

// NewJopDeleteBatchEndpoint 批量删除
func NewJopDeleteBatchEndpoint(c echo.Context) error {
	id := c.Param("id")
	ids := strings.Split(id, ",")

	_, err := global.SCHEDULER.FindJobsByTag(ids...)
	if err != nil {
		log.Errorf("未找到相关自动任务, error: %v", err)
	}
	err = global.SCHEDULER.RemoveByTags(ids...)
	if err != nil {
		log.Errorf("删除自动任务失败, error: %v", err)
	}

	var name string
	for _, id := range ids {
		job, err := newjobRepository.FindById(context.TODO(), id)
		if err != nil {
			log.Errorf("NewJopDeleteBatchEndpoint FindById error: %v", err)
			return FailWithDataOperate(c, 500, "删除失败", "任务列表-批量删除: 批量删除计划任务, 失败原因["+err.Error()+"]", nil)
		}
		name += job.Name + ","
	}
	err = newjobRepository.DeleteAll(context.TODO(), ids)
	if err != nil {
		log.Errorf("NewJopDeleteBatchEndpoint DeleteAll error: %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "任务列表-批量删除: 批量删除计划任务, 失败原因["+err.Error()+"]", nil)
	}

	return SuccessWithOperate(c, "任务列表-批量删除: 批量删除计划任务, 名称["+name+"]", nil)
}

func NewJopStartEndpoint(c echo.Context) error {
	id := c.Param("id")

	job, err := newjobRepository.FindById(context.TODO(), id)
	if err != nil {
		log.Errorf("NewJopStartEndpoint FindById error: %v", err)
		return FailWithDataOperate(c, 500, "执行失败", "任务列表-执行: 执行计划任务, 失败原因[任务未找到]", nil)
	}
	err = newJobService.RunJobNow(id)
	if err != nil {
		return FailWithDataOperate(c, 300, "执行失败", "任务列表-执行: 执行计划任务, 失败原因["+err.Error()+"]", nil)
	}

	return SuccessWithOperate(c, "任务列表-执行: 执行计划任务, 名称["+job.Name+"]", nil)
}

// NewJopDeviceEndpoint 关联设备
func NewJopDeviceEndpoint(c echo.Context) error {
	id := c.Param("id")
	aids := c.QueryParam("aids")
	ids := utils.IdHandle(aids)

	job, err := newjobRepository.FindById(context.TODO(), id)
	if err != nil {
		log.Errorf("NewJopDeviceEndpoint FindById error: %v", err)
		return FailWithDataOperate(c, 500, "关联设备失败", "任务列表-关联: 任务["+job.Name+"|"+job.Department+"]"+"关联设备, 失败原因["+err.Error()+"]", nil)
	}

	var name string
	for _, id := range ids {
		passport, err := newAssetRepository.GetPassportById(context.TODO(), id)
		if err != nil {
			log.Errorf("NewJopDeviceEndpoint GetPassportById error: %v", err)
			return FailWithDataOperate(c, 500, "关联设备失败", "任务列表-关联: 任务["+job.Name+"|"+job.Department+"]"+"关联设备, 失败原因["+err.Error()+"]", nil)
		}
		name += passport.AssetName + "|" + passport.Ip + "|" + passport.Passport + "、"
	}

	if len(name) > 0 {
		name = name[:len(name)-1]
	}

	err = newjobRepository.JobBindAssets(context.TODO(), id, ids)
	if err != nil {
		log.Errorf("NewJopDeviceEndpoint JobBindAssets error: %v", err)
		return FailWithDataOperate(c, 500, "关联设备失败", "任务列表-关联: 任务["+job.Name+"|"+job.Department+"]"+"关联设备, 失败原因["+err.Error()+"]", nil)
	}
	return SuccessWithOperate(c, "任务列表-关联: 任务["+job.Name+"|"+job.Department+"]"+"关联设备, 设备["+name+"]", nil)
}

// NewJopDeviceGroupEndpoint 关联设备组
func NewJopDeviceGroupEndpoint(c echo.Context) error {
	id := c.Param("id")
	aids := c.QueryParam("aids")
	ids := utils.IdHandle(aids)

	job, err := newjobRepository.FindById(context.TODO(), id)
	if err != nil {
		log.Errorf("NewJopDeviceGroupEndpoint FindById error: %v", err)
		return FailWithDataOperate(c, 500, "关联设备组失败", "任务列表-关联: 任务["+job.Name+"|"+job.Department+"]"+"关联设备组, 失败原因["+err.Error()+"]", nil)
	}

	var name string
	for _, id := range ids {
		group, err := newAssetGroupRepository.GetAssetGroupById(context.TODO(), id)
		if err != nil {
			log.Errorf("NewJopDeviceGroupEndpoint FindById error: %v", err)
			return FailWithDataOperate(c, 500, "关联设备组失败", "任务列表-关联: 任务["+job.Name+"|"+job.Department+"]"+"关联设备组, 失败原因["+err.Error()+"]", nil)
		}
		name += group.Name + "|" + group.Department + "、"
	}

	if len(name) > 0 {
		name = name[:len(name)-1]
	}

	err = newjobRepository.JobBindAssetGroups(context.TODO(), id, ids)
	if err != nil {
		log.Errorf("NewJopDeviceGroupEndpoint JobBindAssetGroups error: %v", err)
		return FailWithDataOperate(c, 500, "关联设备组失败", "任务列表-关联: 任务["+job.Name+"|"+job.Department+"]"+"关联设备组, 失败原因["+err.Error()+"]", nil)
	}
	return SuccessWithOperate(c, "任务列表-关联: 任务["+job.Name+"|"+job.Department+"]"+"关联设备组, 设备组["+name+"]", nil)
}

func NewJopGetEndpoint(c echo.Context) error {
	id := c.Param("id")

	job, err := newjobRepository.GetJobForEdit(context.TODO(), id)
	if err != nil {
		log.Errorf("NewJopGetEndpoint FindById error: %v", err)
		return FailWithDataOperate(c, 500, "获取任务失败", "任务列表-获取: 获取任务失败, 失败原因["+err.Error()+"]", nil)
	}

	return Success(c, job)
}

func NewJopDevicePagingEndpoint(c echo.Context) error {
	id := c.Param("id")

	job, err := newjobRepository.FindById(context.TODO(), id)
	if err != nil {
		log.Errorf("获取任务记录失败: %v", err)
		return FailWithDataOperate(c, 500, "获取任务设备失败", "任务列表-获取: 任务["+job.Name+"|"+job.Department+"]"+"获取任务设备, 失败原因["+err.Error()+"]", nil)
	}
	did := job.DepartmentID

	var dids []int64
	err = GetChildDepIds(did, &dids)
	if err != nil {
		log.Errorf("获取任务部门机构以下所有部门失败: %v", err)
		return FailWithDataOperate(c, 500, "获取任务设备失败", "任务列表-获取: 任务["+job.Name+"|"+job.Department+"]"+"获取任务设备, 失败原因["+err.Error()+"]", nil)
	}
	// 策略所属部门及下级部门包含的所有设备账号(包括已被选择设备账号)
	assetAllArr, err := newAssetRepository.GetPassportByDepartmentIds(context.TODO(), dids)
	if nil != err {
		log.Errorf("获取任务可关联设备失败: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	var assetAll = make([]dto.ForRelate, 0)
	for i := range assetAllArr {
		if assetAllArr[i].Protocol == constant.SSH {
			var pp dto.ForRelate
			pp.ID = assetAllArr[i].ID
			depChinaName, err := DepChainName(assetAllArr[i].DepartmentId)
			if nil != err {
				log.Errorf("DepChainName Error: %v", err)
				return FailWithDataOperate(c, 500, "查询失败", "", nil)
			}
			pp.Name = assetAllArr[i].AssetName + "[" + assetAllArr[i].Ip + "]" + "[" + assetAllArr[i].Passport + "]" + "[" + assetAllArr[i].Protocol + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
			assetAll = append(assetAll, pp)
		}
	}
	return SuccessWithOperate(c, "", assetAll)
}

func NewJopDeviceListEndpoint(c echo.Context) error {
	id := c.Param("id")

	assetids, err := newjobRepository.FindAssetIdsByJobId(context.TODO(), id)
	if err != nil {
		log.Errorf("NewJopDeviceListEndpoint FindById error: %v", err)
		return FailWithDataOperate(c, 500, "获取任务设备失败", "任务列表-获取: 任务["+id+"]"+"获取任务设备, 失败原因["+err.Error()+"]", nil)
	}

	assetAllArr, err := newAssetRepository.GetPassportByIds(context.TODO(), assetids)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	var assetAll = make([]dto.ForRelate, 0)
	for i := range assetAllArr {
		var pp dto.ForRelate
		pp.ID = assetAllArr[i].ID
		depChinaName, err := DepChainName(assetAllArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		pp.Name = assetAllArr[i].AssetName + "[" + assetAllArr[i].Ip + "]" + "[" + assetAllArr[i].Passport + "]" + "[" + assetAllArr[i].Protocol + "]" + "[" + depChinaName[:len(depChinaName)-1] + "]"
		assetAll = append(assetAll, pp)
	}
	return SuccessWithOperate(c, "", assetAll)
}

func NewJopDeviceGroupPagingEndpoint(c echo.Context) error {
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
	var resp = make([]dto.AssetGroupWithId, 0)
	for i := range assetGroupAllArr {
		var pp dto.AssetGroupWithId
		pp.ID = assetGroupAllArr[i].Id
		depChinaName, err := DepChainName(assetGroupAllArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		pp.Name = assetGroupAllArr[i].Name + "[" + depChinaName[:len(depChinaName)-1] + "]"
		resp = append(resp, pp)
	}

	return SuccessWithOperate(c, "", resp)
}

func NewJopDeviceGroupListEndpoint(c echo.Context) error {
	id := c.Param("id")

	assetGroupids, err := newjobRepository.FindAssetGroupIdsByJobId(context.TODO(), id)
	if err != nil {
		log.Errorf("NewJopDeviceGroupListEndpoint FindById error: %v", err)
		return FailWithDataOperate(c, 500, "获取任务设备组失败", "任务列表-获取: 任务["+id+"]"+"获取任务设备组, 失败原因["+err.Error()+"]", nil)
	}

	assetGroupAllArr, err := newAssetGroupRepository.GetAssetGroupByIds(context.TODO(), assetGroupids)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	var resp = make([]dto.AssetGroupWithId, 0)
	for i := range assetGroupAllArr {
		var pp dto.AssetGroupWithId
		pp.ID = assetGroupAllArr[i].Id
		depChinaName, err := DepChainName(assetGroupAllArr[i].DepartmentId)
		if nil != err {
			log.Errorf("DepChainName Error: %v", err)
			return FailWithDataOperate(c, 500, "查询失败", "", nil)
		}
		pp.Name = assetGroupAllArr[i].Name + "[" + depChinaName[:len(depChinaName)-1] + "]"
		resp = append(resp, pp)
	}

	return Success(c, resp)
}

// NewJopScriptEndpoint 获取已上传的脚本文件
func NewJopScriptEndpoint(c echo.Context) error {
	id := c.Param("id")

	u, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}

	job, err := newjobRepository.FindById(context.TODO(), id)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "下载失败", "", nil)
	}

	scriptPath := path.Join(constant.ShellPath, job.ShellName)
	if !utils.FileExists(scriptPath) {
		return FailWithDataOperate(c, 500, "下载失败", "", nil)
	}

	// 生成文件名
	fileName := job.ShellName[:strings.LastIndex(job.ShellName, ".")]

	// 记录操作日志
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		Result:          "成功",
		LogTypes:        "用户日志",
		Created:         utils.NowJsonTime(),
		LogContents:     "任务列表-编辑: 任务[" + job.Name + "]下载脚本文件",
		Users:           u.Username,
	}
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}
	// 读取文件转为*bytes.Buffer
	file, err := os.Open(scriptPath)
	if err != nil {
		return FailWithDataOperate(c, 500, "下载失败", "", nil)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Error("file close error: ", err)
		}
	}(file)

	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, file)
}
