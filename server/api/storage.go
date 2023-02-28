package api

import (
	"context"
	"os"
	"strconv"
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

func NewStoragePagingEndpoint(c echo.Context) error {
	var sfs dto.StorageForSearch
	sfs.Name = c.QueryParam("name")
	sfs.Department = c.QueryParam("department")
	sfs.Auto = c.QueryParam("auto")

	u, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithData(c, 401, "查询失败", nil)
	}

	err := GetChildDepIds(u.DepartmentId, &sfs.Departments)
	if err != nil {
		return FailWithData(c, 500, "查询失败", nil)
	}

	resp, err := storageRepositoryNew.FindBySearch(context.TODO(), sfs)
	if err != nil {
		return FailWithData(c, 500, "查询失败", nil)
	}

	return SuccessWithOperate(c, "", resp)
}

func NewStorageCreateEndpoint(c echo.Context) error {
	var ns dto.StorageForCreate
	u, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithData(c, 401, "创建失败", nil)
	}
	if err := c.Bind(&ns); err != nil {
		log.Errorf("NewStorageCreateEndpoint: %v", err)
		return FailWithData(c, 500, "创建失败", nil)
	}

	// 数据校验
	if err := c.Validate(&ns); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	_, err := storageRepositoryNew.FindByName(context.TODO(), ns.Name)
	if err == nil {
		return FailWithDataOperate(c, 500, "名称已存在", "磁盘空间-创建: 创建存储空间, 失败原因[存储空间名称已存在]", nil)
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return FailWithData(c, 500, "创建失败", nil)
	}

	id := utils.UUID()
	path, err := storageNewService.CreateStorage(id)
	if err != nil {
		return FailWithData(c, 500, "创建失败", nil)
	}

	var limitSize int64
	property, err := propertyRepository.FindByName("storage_size")
	if err != nil {
		return FailWithData(c, 500, "创建失败", nil)
	}
	limitSize, err = strconv.ParseInt(property.Value, 10, 64)
	if err != nil {
		return FailWithData(c, 500, "创建失败", nil)
	}

	err = storageRepositoryNew.Create(context.TODO(), &model.NewStorage{
		ID:             id,
		Name:           ns.Name,
		LimitSize:      limitSize * 1024 * 1024,
		Department:     u.DepartmentId,
		DepartmentName: u.DepartmentName,
		Info:           ns.Info,
		Created:        utils.NowJsonTime(),
	})
	if err != nil {
		_ = os.RemoveAll(path)
		return FailWithData(c, 500, "创建失败", nil)
	}

	return SuccessWithOperate(c, "磁盘空间-创建: 创建存储空间, 名称["+ns.Name+"]", nil)
}

func NewStorageUpdateEndpoint(c echo.Context) error {
	var ns dto.StorageForCreate
	_, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithData(c, 401, "更新失败", nil)
	}

	id := c.Param("id")
	if err := c.Bind(&ns); err != nil {
		log.Errorf("NewStorageUpdateEndpoint: %v", err)
		return FailWithData(c, 500, "更新失败", nil)
	}

	// 数据校验
	if err := c.Validate(&ns); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	d, err := departmentRepository.FindById(ns.Department)
	if err != nil {
		return FailWithData(c, 500, "更新失败", nil)
	}

	s, err := storageRepositoryNew.FindById(context.TODO(), id)
	if err != nil {
		return FailWithData(c, 500, "更新失败", nil)
	}

	_, err = storageRepositoryNew.FindByNameId(context.TODO(), ns.Name, id)
	if err == nil {
		return FailWithDataOperate(c, 500, "更新失败", "磁盘空间-更新: 更新存储空间, 失败原因[存储空间名称已存在]", nil)
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return FailWithData(c, 500, "更新失败", nil)
	}

	err = storageRepositoryNew.Update(context.TODO(), &model.NewStorage{
		ID:             id,
		Name:           ns.Name,
		Department:     d.ID,
		DepartmentName: d.Name,
		Info:           ns.Info,
	})
	if err != nil {
		log.Errorf("NewStorageUpdateEndpoint: %v", err)
		return FailWithData(c, 500, "更新失败", nil)
	}

	return SuccessWithOperate(c, "磁盘空间-更新: 更新存储空间, 名称["+s.Name+"->"+ns.Name+"]", nil)
}

func NewStorageDeleteEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithData(c, 401, "删除失败", nil)
	}

	id := c.Param("id")
	name := "磁盘空间-删除: 删除存储空间, 名称["
	ids := utils.IdHandle(id)
	for _, v := range ids {
		s, err := storageRepositoryNew.FindById(context.TODO(), v)
		if err != nil {
			return FailWithData(c, 500, "删除失败", nil)
		}
		name += s.Name + ","
		err = storageNewService.DeleteStorageById(v)
		if err != nil {
			return FailWithData(c, 500, "删除失败", nil)
		}
	}
	name = name[:len(name)-1] + "]"

	return SuccessWithOperate(c, name, nil)
}

func NewStorageLsEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithData(c, 401, "获取失败", nil)
	}

	id := c.Param("id")
	s, err := storageRepositoryNew.FindById(context.TODO(), id)
	if err != nil {
		return FailWithData(c, 500, "获取失败", nil)
	}

	remoteDir := c.FormValue("dir")
	resp, err := storageNewService.StorageLs(remoteDir, id)
	if err != nil {
		return FailWithData(c, 500, "获取失败", nil)
	}

	return SuccessWithOperate(c, "磁盘空间-查看: 查看存储空间, 存储空间["+s.Name+"], 目录["+remoteDir+"]", resp)
}

func NewStorageDownloadEndpoint(c echo.Context) error {
	account, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithDataOperate(c, 401, "操作失败", "", nil)
	}
	storageId := c.Param("id")
	diskSpaceInfo, err := storageRepositoryNew.FindById(context.TODO(), storageId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}

	file := c.QueryParam("file")
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "运维日志",
		Created:         utils.NowJsonTime(),
		Users:           account.Username,
		Names:           account.Nickname,
		Result:          "成功",
		LogContents:     "磁盘空间-下载: 下载文件, 磁盘空间[" + diskSpaceInfo.Name + "], 文件[" + file + "]",
	}
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}

	return storageNewService.StorageDownload(c, file, storageId)
}

func NewStorageUploadEndpoint(c echo.Context) error {
	storageId := c.Param("id")
	diskSpaceInfo, err := storageRepositoryNew.FindById(context.TODO(), storageId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}

	file, err := c.FormFile("file")
	if nil != err {
		log.Errorf("FormFile Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}

	if err := storageNewService.StorageUpload(c, file, storageId); err != nil {
		log.Errorf("StorageUpload Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}
	return SuccessWithOperate(c, "磁盘空间-上传: 上传文件, 存储空间["+diskSpaceInfo.Name+"]", nil)
}

func NewStorageDeleteFileEndpoint(c echo.Context) error {
	storageId := c.Param("id")
	diskSpaceInfo, err := storageRepositoryNew.FindById(context.TODO(), storageId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}

	// 文件夹或者文件
	file := c.FormValue("file")
	if err := storageNewService.StorageRm(file, storageId); err != nil {
		log.Errorf("StorageRm Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}
	return SuccessWithOperate(c, "磁盘空间-删除: 删除文件, 存储空间["+diskSpaceInfo.Name+"], 删除文件["+file+"]", nil)
}

func NewStorageMkdirEndpoint(c echo.Context) error {
	storageId := c.Param("id")
	diskSpaceInfo, err := storageRepositoryNew.FindById(context.TODO(), storageId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}

	remoteDir := c.QueryParam("dir")
	if err := storageNewService.StorageMkDir(remoteDir, storageId); err != nil {
		log.Errorf("StorageMkDir Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败,"+err.Error(), "", err)
	}
	return SuccessWithOperate(c, "磁盘空间-创建: 创建文件夹, 存储空间["+diskSpaceInfo.Name+"], 创建文件夹["+remoteDir+"]", nil)
}

func NewStorageRenameEndpoint(c echo.Context) error {
	storageId := c.Param("id")
	diskSpaceInfo, err := storageRepositoryNew.FindById(context.TODO(), storageId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}

	oldName := c.QueryParam("oldName")
	newName := c.QueryParam("newName")
	if err := storageNewService.StorageRename(oldName, newName, storageId); err != nil {
		log.Errorf("StorageRename Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败,"+err.Error(), "", err)
	}
	return SuccessWithOperate(c, "磁盘空间-重命名: 重命名文件, 存储空间["+diskSpaceInfo.Name+"], 重命名文件["+oldName+"->"+newName+"]", nil)
}

func NewStorageEditEndpoint(c echo.Context) error {
	storageId := c.Param("id")
	diskSpaceInfo, err := storageRepositoryNew.FindById(context.TODO(), storageId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}

	file := c.FormValue("file")
	fileContent := c.FormValue("fileContent")
	if err := storageNewService.StorageEdit(file, fileContent, storageId); err != nil {
		log.Errorf("StorageEdit Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", err)
	}
	return SuccessWithOperate(c, "磁盘空间-编辑: 编辑文件, 存储空间["+diskSpaceInfo.Name+"], 编辑文件["+file+"]", nil)
}
