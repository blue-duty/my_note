package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strconv"
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
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

func NewAssetPagingEndpoint(c echo.Context) error {
	u, _ := GetCurrentAccountNew(c)
	var req dto.AssetForSearch
	req.IP = c.QueryParam("ip")
	req.Name = c.QueryParam("name")
	req.Auto = c.QueryParam("auto")
	req.AssetType = c.QueryParam("assetType")
	req.Department = c.QueryParam("department")

	var departmentIds []int64
	err := GetChildDepIds(u.DepartmentId, &departmentIds)
	if err != nil {
		return Success(c, []dto.AssetForPage{})
	}
	req.DepartmentIds = departmentIds
	// 获取用户所属部门及子部门
	assetsList, err := newAssetRepository.GetAssetList(context.TODO(), req)
	if err != nil {
		return Success(c, []dto.AssetForPage{})
	}
	var resp []dto.AssetForPage
	for _, asset := range assetsList {
		department, err := departmentRepository.FindById(asset.DepartmentId)
		if err != nil {
			continue
		}
		systemType, err := systemTypeRepository.GetSystemTypeByID(context.TODO(), asset.AssetType)
		if err != nil {
			continue
		}
		var assetForPage dto.AssetForPage
		assetForPage.ID = asset.ID
		assetForPage.Name = asset.Name
		assetForPage.IP = asset.IP
		assetForPage.AssetType = systemType.Name
		assetForPage.PassportCount = asset.PassPortCount
		assetForPage.Department = department.Name
		assetForPage.DepartmentId = asset.DepartmentId
		assetForPage.Info = asset.Info
		resp = append(resp, assetForPage)
	}

	return c.JSON(200, H{
		"code":    1,
		"success": true,
		"message": "成功",
		"data":    resp,
	})
}

func GetX11ProgramIdEndpoint(c echo.Context) error {
	pid := c.Param("id")
	programId, err := newAssetRepository.GetPassportConfig(context.TODO(), pid)
	if err != nil {
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}

	var rep H = make(map[string]interface{})
	if v, ok := programId["x11_term_program"]; ok {
		rep["termProgram"] = v
	} else {
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}

	if v, ok := programId["x11_display_program"]; ok {
		rep["displayProgram"] = v
	} else {
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}

	return Success(c, rep)
}

// NewAssetGetEndpoint 获取设备基本信息
func NewAssetGetEndpoint(c echo.Context) error {
	id := c.Param("id")
	asset, err := newAssetRepository.GetDetailAssetById(context.TODO(), id)
	if err != nil {
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	return Success(c, asset)
}

func NewAssetCreateEndpoint(c echo.Context) error {
	var req dto.AssetForCreate

	u, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithDataOperate(c, 401, "创建失败", "创建设备: 用户未登录", nil)
	}

	if err := c.Bind(&req); err != nil {
		return FailWithDataOperate(c, 500, "新增失败, 绑定失败, "+err.Error(), "", nil)
	}

	// 数据校验
	if err := c.Validate(&req); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	// 查询数据库中的所有设备数量
	var count int64
	err := global.DBConn.Model(&model.NewAsset{}).Count(&count).Error
	if err != nil {
		return FailWithDataOperate(c, 500, "新增失败", "设备列表-新增: 查询数据库中的所有设备数量失败", nil)
	}

	err, isover := sysMaintainService.IsOverAuthResourceCountLimit(int(count))
	if nil != err {
		log.Error("设备列表-新增设备: 获取授权许可最大资源数失败")
		return FailWithDataOperate(c, 500, "新增失败, 请联系技术人员查看系统日志以确定失败原因", "", nil)
	}
	if isover {
		return FailWithDataOperate(c, 400, "系统当前设备数已达到授权许可的最大资源数", "设备列表-新增设备: 新增失败, 失败原因[系统当前设备数已达到授权许可的最大资源数]", nil)
	}

	_, err = newAssetRepository.GetAssetByIP(context.TODO(), req.IP)
	if err == nil {
		return FailWithDataOperate(c, 400, "设备地址已存在", "设备列表-新增: 新增设备, 失败原因[设备地址"+req.IP+"已存在]", nil)
	} else if err != gorm.ErrRecordNotFound {
		return FailWithDataOperate(c, 400, "新增失败", "设备列表-新增: 新增设备, 失败原因["+err.Error()+"]", nil)
	}
	_, err = newAssetRepository.GetAssetByName(context.TODO(), req.Name)
	if err == nil {
		return FailWithDataOperate(c, 400, "设备名称已存在", "设备列表-新增: 新增设备, 失败原因[设备名称"+req.Name+"已存在]", nil)
	} else if err != gorm.ErrRecordNotFound {
		return FailWithDataOperate(c, 400, "新增失败", "设备列表-新增: 新增设备, 失败原因["+err.Error()+"]", nil)
	}
	encryptedCBC, err := utils.AesEncryptCBC([]byte(req.Password), global.Config.EncryptionPassword)
	if err != nil {
		return err
	}
	req.Password = base64.StdEncoding.EncodeToString(encryptedCBC)
	asset := model.NewAsset{
		ID:            utils.UUID(),
		Name:          req.Name,
		IP:            req.IP,
		AssetType:     req.AssetType,
		DepartmentId:  u.DepartmentId,
		Info:          req.Info,
		PassPortCount: 1,
		Created:       utils.NowJsonTime(),
	}
	err = newAssetRepository.CreateAsset(context.TODO(), asset)
	if err != nil {
		return err
	}

	var passPortForCreate model.PassPort
	passPortForCreate.ID = utils.UUID()
	passPortForCreate.LoginType = req.LoginType
	passPortForCreate.Protocol = req.Protocol
	passPortForCreate.Passport = req.Passport
	passPortForCreate.Port = req.Port
	passPortForCreate.PassportType = req.PassportType
	passPortForCreate.Password = req.Password
	passPortForCreate.AssetId = asset.ID
	passPortForCreate.AssetName = asset.Name
	passPortForCreate.AssetType = asset.AssetType
	passPortForCreate.DepartmentId = u.DepartmentId
	passPortForCreate.Status = "enable"
	passPortForCreate.SftpPath = req.SftpPath
	passPortForCreate.IsSshKey = 0
	passPortForCreate.Ip = asset.IP
	passPortForCreate.Created = utils.NowJsonTime()
	passPortForCreate.Name = asset.Name + "[" + asset.IP + "]" + "[" + req.Passport + "]" + "[" + req.Protocol + "]" // 设备名称[IP][帐号][协议]

	isSshKey := c.FormValue("isSshKey")
	log.Info(isSshKey)
	// 将私钥文件req.Privatekey保存在本地constant.PrivateKeyPath目录中
	if isSshKey == "是" {
		privateKey, _ := c.FormFile("privateKey")
		if privateKey == nil {
			src, err := privateKey.Open()
			if nil != err {
				log.Error("设备列表-新增设备: 读取私钥文件失败")
				return FailWithDataOperate(c, 500, "新增失败, 请联系技术人员查看系统日志以确定失败原因", "", nil)
			}
			defer func(src multipart.File) {
				err := src.Close()
				if err != nil {
					log.Errorf("Close Error: %v", err)
				}
			}(src)

			dst, err := os.Create(path.Join(constant.PrivateKeyPath, privateKey.Filename+"."+passPortForCreate.ID))
			if nil != err {
				log.Error("设备列表-新增设备: 保存私钥文件失败")
				return FailWithDataOperate(c, 500, "新增失败, 请联系技术人员查看系统日志以确定失败原因", "", nil)
			}
			defer func(dst *os.File) {
				err := dst.Close()
				if err != nil {
					log.Errorf("Close Error: %v", err)
				}
			}(dst)
			if _, err = io.Copy(dst, src); err != nil {
				log.Error("设备列表-新增设备: 保存私钥文件失败")
				return FailWithDataOperate(c, 500, "写入文件失败", "", nil)
			}

			if req.Passphrase != "" {
				encryptedPass, err := utils.AesEncryptCBC([]byte(req.Passphrase), global.Config.EncryptionPassword)
				if err != nil {
					return err
				}
				passPortForCreate.Passphrase = base64.StdEncoding.EncodeToString(encryptedPass)
			}
			passPortForCreate.PrivateKey = privateKey.Filename + "." + passPortForCreate.ID
			passPortForCreate.IsSshKey = 1
		} else {
			return FailWithDataOperate(c, 400, "新增失败, 请上传私钥文件", "", nil)
		}
	}

	err = newAssetRepository.CreatePassport(context.TODO(), passPortForCreate)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithData(c, 500, "新增失败", nil)
	}

	// 设备账号创建成功后，保存高级设置
	var assetAdvancedSetting []model.PassportConfiguration
	if req.Protocol == "rdp" {
		if req.RdpDomain != "" {
			assetAdvancedSetting = append(assetAdvancedSetting, model.PassportConfiguration{
				PassportId: passPortForCreate.ID,
				Name:       "rdp_domain",
				Value:      req.RdpDomain,
			})
		}
		if req.RdpEnableDrive == "是" {
			assetAdvancedSetting = append(assetAdvancedSetting, model.PassportConfiguration{
				PassportId: passPortForCreate.ID,
				Name:       "rdp_enable_drive",
				Value:      "true",
			})
			if req.RdpDriveId != "" {
				assetAdvancedSetting = append(assetAdvancedSetting, model.PassportConfiguration{
					PassportId: passPortForCreate.ID,
					Name:       "rdp_drive_path",
					Value:      path.Join(storageNewService.GetBaseDrivePath(), req.RdpDriveId),
				})
			}
		}
	}
	// X11相关参数
	if req.Protocol == "x11" {
		if req.AppSerId != "" && req.TermProgram != "" && req.DisplayProgram != "" {
			assetAdvancedSetting = append(assetAdvancedSetting, model.PassportConfiguration{
				PassportId: passPortForCreate.ID,
				Name:       "x11_appserver_id",
				Value:      req.AppSerId,
			})
			assetAdvancedSetting = append(assetAdvancedSetting, model.PassportConfiguration{
				PassportId: passPortForCreate.ID,
				Name:       "x11_term_program",
				Value:      req.TermProgram,
			})
			assetAdvancedSetting = append(assetAdvancedSetting, model.PassportConfiguration{
				PassportId: passPortForCreate.ID,
				Name:       "x11_display_program",
				Value:      req.DisplayProgram,
			})
		} else {
			return FailWithDataOperate(c, 400, "新增失败, X11协议下，应用服务器、终端程序、显示程序不能为空", "设备列表-新增: 新增设备, 失败原因[X11协议下，应用服务器、终端程序、显示程序不能为空]", nil)
		}
	}

	if len(assetAdvancedSetting) > 0 {
		err = global.DBConn.Model(&model.PassportConfiguration{}).Create(&assetAdvancedSetting).Error
		if err != nil {
			log.Errorf("设备账号创建成功后，保存高级设置失败: %v", err)
		}
	}

	UpdateUserAssetAppAppserverDep(constant.ASSET, constant.ADD, asset.DepartmentId, int64(-1))
	return SuccessWithOperate(c, "设备列表-新增: 新增设备, 用户["+u.Nickname+"], 设备名称["+req.Name+"]", nil)
}

func NewAssetUpdateEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}
	var req dto.AssetForUpdate
	if err := c.Bind(&req); err != nil {
		log.Errorf("Bind error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", nil)
	}

	// 数据校验
	if err := c.Validate(&req); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	asset, err := newAssetRepository.GetAssetByID(context.TODO(), req.ID)
	if err != nil {
		return FailWithDataOperate(c, 500, "修改失败", "修改设备信息: 设备未找到", nil)
	}
	oldDepartment := asset.DepartmentId
	oldName := asset.Name
	asset.Name = req.Name
	asset.IP = req.IP
	asset.AssetType = req.AssetType
	asset.DepartmentId = req.DepartmentId
	asset.Info = req.Info
	err = newAssetRepository.UpdateAsset(context.TODO(), asset)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", nil)
	}

	if oldDepartment != req.DepartmentId {
		UpdateUserAssetAppAppserverDep(constant.ASSET, constant.UPDATE, oldDepartment, req.DepartmentId)
	}

	return SuccessWithOperate(c, "设备列表-修改: 原设备: ["+oldName+"], 新设备: ["+asset.Name+"]", nil)
}

func NewAssetDeleteEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}
	id := c.Param("id")
	asset, err := newAssetRepository.GetAssetByID(context.TODO(), id)
	if err != nil {
		return FailWithDataOperate(c, 500, "删除失败", "设备列表-删除: 删除设备, 失败原因[设备未找到]", nil)
	}
	err = newAssetRepository.DeleteAsset(context.TODO(), id)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "", nil)
	}

	UpdateUserAssetAppAppserverDep(constant.ASSET, constant.DELETE, asset.DepartmentId, int64(-1))

	return SuccessWithOperate(c, "设备列表-删除: 删除设备, 设备名称["+asset.Name+"], 设备地址["+asset.IP+"]", nil)
}

// NewAssetBatchDeleteEndpoint 批量删除
func NewAssetBatchDeleteEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}
	id := c.QueryParam("id")
	ids := utils.IdHandle(id)
	var name string
	for _, v := range ids {
		asset, err := newAssetRepository.GetAssetByID(context.TODO(), v)
		if err != nil {
			return FailWithDataOperate(c, 500, "删除失败", "设备列表-删除: 批量删除设备, 失败原因[设备未找到]", nil)
		}
		UpdateUserAssetAppAppserverDep(constant.ASSET, constant.DELETE, asset.DepartmentId, int64(-1))
		name = name + asset.Name + ","
	}
	err := newAssetRepository.BatchDeleteAsset(context.TODO(), ids)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "", nil)
	}
	return SuccessWithOperate(c, "设备列表-删除: 批量删除设备, 设备名称["+name+"]", nil)
}

func PassPortCreateEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}
	var req dto.PassPortForCreate
	if err := c.Bind(&req); err != nil {
		log.Errorf("Bind error: %v", err)
		return FailWithDataOperate(c, 500, "新增设备帐号失败", "", nil)
	}
	asset, err := newAssetRepository.GetAssetByID(context.TODO(), req.AssetId)
	if err != nil {
		return FailWithDataOperate(c, http.StatusNotFound, "新增设备帐号失败", "设备账号-新增: 失败原因[设备未找到]", nil)
	}

	// 查询用户名是否存在
	_, err = newAssetRepository.GetPassPortByAssetIDAndUsername(context.TODO(), req.AssetId, req.Passport, req.Protocol)
	if err == nil {
		return FailWithDataOperate(c, http.StatusConflict, "新增设备帐号失败", "设备账号-新增: 失败原因[设备帐号已存在]", nil)
	} else if err != nil && err != gorm.ErrRecordNotFound {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "新增设备帐号失败", "", nil)
	}

	encryptedCBC, err := utils.AesEncryptCBC([]byte(req.Password), global.Config.EncryptionPassword)
	if err != nil {
		return err
	}
	req.Password = base64.StdEncoding.EncodeToString(encryptedCBC)
	passPort := model.PassPort{
		ID:           utils.UUID(),
		Name:         asset.Name + "[" + asset.IP + "]" + "[" + req.Passport + "]" + "[" + req.Protocol + "]", // 设备名称[IP][帐号][协议]
		LoginType:    req.LoginType,
		AssetName:    asset.Name,
		AssetType:    asset.AssetType,
		DepartmentId: asset.DepartmentId,
		Ip:           asset.IP,
		Protocol:     req.Protocol,
		Passport:     req.Passport,
		Port:         req.Port,
		PassportType: req.PassportType,
		Password:     req.Password,
		AssetId:      req.AssetId,
		Status:       "enable",
		SftpPath:     req.SftpPath,
		IsSshKey:     0,
		Created:      utils.NowJsonTime(),
	}

	// 将私钥文件req.Privatekey保存在本地constant.PrivateKeyPath目录中
	isSshKey := c.FormValue("isSshKey")
	log.Info(isSshKey)
	// 将私钥文件req.Privatekey保存在本地constant.PrivateKeyPath目录中
	if isSshKey == "是" {
		privateKey, _ := c.FormFile("privateKey")
		log.Info(privateKey.Filename)
		src, err := privateKey.Open()
		if nil != err {
			log.Error("设备账号-新增: 失败原因[打开私钥文件失败]")
			return FailWithDataOperate(c, 500, "新增失败, 请联系技术人员查看系统日志以确定失败原因", "", nil)
		}
		defer func(src multipart.File) {
			err := src.Close()
			if err != nil {
				log.Errorf("Close Error: %v", err)
			}
		}(src)

		if !utils.FileExists(constant.PrivateKeyPath) {
			err = os.MkdirAll(constant.PrivateKeyPath, 0777)
			log.Info("创建私钥文件夹成功")
			if err != nil {
				log.Errorf("MkdirAll Error: %v", err)
				return FailWithDataOperate(c, 500, "新增失败, 请联系技术人员查看系统日志以确定失败原因", "", nil)
			}
		}

		dst, err := os.Create(path.Join(constant.PrivateKeyPath, privateKey.Filename+"."+passPort.ID))
		if nil != err {
			log.Error("设备账号-新增: 失败原因[创建私钥文件失败]", err)
			return FailWithDataOperate(c, 500, "新增失败, 请联系技术人员查看系统日志以确定失败原因", "", nil)
		}
		defer func(dst *os.File) {
			err := dst.Close()
			if err != nil {
				log.Errorf("Close Error: %v", err)
			}
		}(dst)
		if _, err = io.Copy(dst, src); err != nil {
			log.Error("设备账号-新增: 失败原因[复制私钥文件失败]")
			return FailWithDataOperate(c, 500, "写入文件失败", "", nil)
		}

		passPort.PrivateKey = privateKey.Filename + "." + passPort.ID
		passPort.IsSshKey = 1
		if req.Passphrase != "" {
			encryptedP, err := utils.AesEncryptCBC([]byte(req.Passphrase), global.Config.EncryptionPassword)
			if err != nil {
				return err
			}
			passPort.Passphrase = base64.StdEncoding.EncodeToString(encryptedP)
		}
	}
	err = newAssetRepository.CreatePassport(context.TODO(), passPort)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", nil)
	}

	err = newAssetRepository.IncrAssetCount(context.TODO(), req.AssetId, 1)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", nil)
	}

	// 设备账号创建成功后，保存设备账户高级配置
	err = newAssetRepository.PassportAdvanced(context.TODO(), req, passPort.ID)
	if err != nil {
		log.Errorf("设备账号创建成功后，保存高级设置失败: %v", err)
	}

	return SuccessWithOperate(c, "设备账号-新增: 设备"+asset.Name+"新增帐号["+passPort.Passport+"]成功", nil)
}

func PassPortUpdateEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithDataOperate(c, 401, "更新设备帐号失败", "设备账号-修改: 失败原因[用户未登录]", nil)
	}
	id := c.Param("id")
	var req dto.PassPortForCreate
	if err := c.Bind(&req); err != nil {
		log.Errorf("Bind error: %v", err)
		return FailWithDataOperate(c, 500, "修改设备帐号失败", "", nil)
	}
	passPort, err := newAssetRepository.GetPassPortByID(context.TODO(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return FailWithDataOperate(c, http.StatusNotFound, "修改设备帐号失败", "设备账号-修改: 失败原因[设备账号不存在]", nil)
		}
		return FailWithDataOperate(c, 500, "修改设备帐号失败", "", nil)
	}
	a, err := newAssetRepository.GetAssetByID(context.TODO(), passPort.AssetId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return FailWithDataOperate(c, http.StatusNotFound, "修改设备帐号失败", "设备账号-修改: 失败原因[设备不存在]", nil)
		}
		return FailWithDataOperate(c, 500, "修改设备帐号失败", "", nil)
	}
	oldPassport := passPort.Passport

	isSshKey := c.FormValue("isSshKey")
	// 将私钥文件req.Privatekey保存在本地constant.PrivateKeyPath目录中
	if isSshKey == "是" {
		privateKey, _ := c.FormFile("privateKey")
		// 删除旧的私钥文件
		if privateKey != nil {
			if passPort.PrivateKey != "" {
				err = os.Remove(path.Join(constant.PrivateKeyPath, passPort.PrivateKey))
				if err != nil {
					log.Errorf("删除旧的私钥文件失败: %v", err)
				}
			}

			src, err := privateKey.Open()
			if nil != err {
				log.Error("设备账号-修改: 失败原因[打开私钥文件失败]")
				return FailWithDataOperate(c, 500, "修改失败, 请联系技术人员查看系统日志以确定失败原因", "", nil)
			}
			defer func(src multipart.File) {
				err := src.Close()
				if err != nil {
					log.Errorf("Close Error: %v", err)
				}
			}(src)

			dst, err := os.Create(path.Join(constant.PrivateKeyPath, privateKey.Filename+"."+passPort.ID))
			if nil != err {
				log.Error("设备账号-修改: 失败原因[创建私钥文件失败]")
				return FailWithDataOperate(c, 500, "修改失败, 请联系技术人员查看系统日志以确定失败原因", "", nil)
			}
			defer func(dst *os.File) {
				err := dst.Close()
				if err != nil {
					log.Errorf("Close Error: %v", err)
				}
			}(dst)
			if _, err = io.Copy(dst, src); err != nil {
				log.Error("设备账号-修改设备帐号: 修改设备帐号, 失败原因[复制私钥文件失败]")
				return FailWithDataOperate(c, 500, "修改失败, 请联系技术人员查看系统日志以确定失败原因", "", nil)
			}
			passPort.PrivateKey = privateKey.Filename + "." + passPort.ID
			passPort.IsSshKey = 1
		}
		if req.Passphrase != "" {
			encryptedCBC, err := utils.AesEncryptCBC([]byte(req.Passphrase), global.Config.EncryptionPassword)
			if err != nil {
				return err
			}
			passPort.Passphrase = base64.StdEncoding.EncodeToString(encryptedCBC)
		}
	} else {
		if passPort.PrivateKey != "" {
			err = os.Remove(path.Join(constant.PrivateKeyPath, passPort.PrivateKey))
			if err != nil {
				log.Errorf("删除旧的私钥文件失败: %v", err)
			}
		}
		passPort.PrivateKey = ""
		passPort.IsSshKey = 0
		passPort.Passphrase = ""
		if req.Password != "" {
			encryptedCBC, err := utils.AesEncryptCBC([]byte(req.Password), global.Config.EncryptionPassword)
			if err != nil {
				return FailWithDataOperate(c, 500, "修改设备帐号失败", "", nil)
			}
			passPort.Password = base64.StdEncoding.EncodeToString(encryptedCBC)
		}
	}

	if req.SftpPath != "" {
		passPort.SftpPath = req.SftpPath
	}
	passPort.LoginType = req.LoginType
	passPort.Protocol = req.Protocol
	passPort.Passport = req.Passport
	passPort.Port = req.Port
	passPort.PassportType = req.PassportType
	passPort.ID = id

	err = newAssetRepository.UpdatePassport(context.TODO(), passPort)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", nil)
	}

	// 设备账号修改成功后，保存设备账户高级配置
	err = newAssetRepository.PassportAdvanced(context.TODO(), req, passPort.ID)
	if err != nil {
		log.Errorf("设备账号修改成功后，保存高级设置失败: %v", err)
	}

	return SuccessWithOperate(c, "设备账号-修改: 设备"+a.Name+"修改账号["+oldPassport+"]->["+passPort.Passport+"]成功", nil)
}

func PassPortDeleteEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}
	id := c.Param("id")
	passPort, err := newAssetRepository.GetPassPortByID(context.TODO(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return FailWithDataOperate(c, http.StatusNotFound, "设备账号-删除: 设备账号不存在", "", nil)
		}
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "", nil)
	}
	a, err := newAssetRepository.GetAssetByID(context.TODO(), passPort.AssetId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return FailWithDataOperate(c, http.StatusNotFound, "设备账号-删除: 设备不存在", "", nil)
		}
		log.Errorf("删除设备账号失败: %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "", nil)
	}
	if passPort.PrivateKey != "" {
		err = os.Remove(path.Join(constant.PrivateKeyPath, passPort.PrivateKey))
		if os.IsNotExist(err) {
			log.Errorf("删除设备账号失败: 设备账号私钥文件不存在")
		} else if err != nil {
			log.Errorf("删除设备账号失败: %v", err)
		}
	}
	err = newAssetRepository.DeletePassport(context.TODO(), id)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "", nil)
	}

	err = newAssetRepository.DecrAssetCount(context.TODO(), passPort.AssetId, 1)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "", nil)
	}

	return SuccessWithOperate(c, "设备账号-删除: 设备"+a.Name+"删除帐号["+passPort.Passport+"]成功", nil)
}

// PassPortBatchDeleteEndpoint 批量删除设备帐号
func PassPortBatchDeleteEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}
	aid := c.Param("aid")
	id := c.QueryParam("id")
	ids := utils.IdHandle(id)
	asset, err := newAssetRepository.GetAssetByID(context.TODO(), aid)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return FailWithDataOperate(c, http.StatusNotFound, "设备账号-删除: 设备不存在", "", nil)
		}
		return FailWithDataOperate(c, 500, "删除失败", "", nil)
	}
	var passport string
	for _, v := range ids {
		pp, err := newAssetRepository.GetPassPortByID(context.TODO(), v)
		if err != nil {
			continue
		}
		if pp.PrivateKey != "" {
			err := os.Remove(path.Join(constant.PrivateKeyPath, pp.PrivateKey))
			if os.IsNotExist(err) {
				log.Error("删除私钥失败: 文件不存在")
			} else if err != nil {
				log.Error("删除私钥失败: ", err)
			}
		}
		passport = passport + pp.Passport + ","
	}
	err = newAssetRepository.BatchDeletePassport(context.TODO(), ids)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "", nil)
	}

	err = newAssetRepository.DecrAssetCount(context.TODO(), aid, len(ids))
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "", nil)
	}

	return SuccessWithOperate(c, "设备账号-删除: 设备"+asset.Name+"删除账号["+passport+"]成功", nil)
}

// 将账号高级配置保存到结构体中
func savePassportAdvanced(req *dto.PassPortForPage, id string) error {
	// 保存账号高级配置
	advanced, err := newAssetRepository.GetPassportConfig(context.TODO(), id)
	if err != nil {
		return err
	}
	//fmt.Println("advanced", advanced)
	if v, ok := advanced["rdp_domain"]; ok {
		req.RdpDomain = v
	}
	if _, ok := advanced["rdp_enable_drive"]; ok {
		req.RdpEnableDrive = "是"
		if vv, ok := advanced["rdp_drive_path"]; ok {
			_, id := path.Split(vv)
			req.RdpDriveId = id
		} else {
			req.RdpEnableDrive = "否"
		}
	} else {
		req.RdpEnableDrive = "否"
	}

	if v, ok := advanced["x11_appserver_id"]; ok {
		req.AppSerId = v
	}
	if v, ok := advanced["x11_term_program"]; ok {
		req.TermProgram = v
	}
	if v, ok := advanced["x11_display_program"]; ok {
		req.DisplayProgram = v
	}

	return nil
}

// PassPortListEndpoint 获取当前设备的所有帐号
func PassPortListEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}
	id := c.Param("id")
	passports, err := newAssetRepository.GetPassport(context.TODO(), id)
	var data []dto.PassPortForPage
	for _, v := range passports {
		var p string
		if v.Passport == "" {
			p = "空用户"
		} else {
			p = v.Passport
		}
		ppp := dto.PassPortForPage{
			ID:           v.ID,
			LoginType:    v.LoginType,
			Protocol:     v.Protocol,
			Passport:     p,
			Port:         v.Port,
			PassportType: v.PassportType,
			Password:     "",
			Passphrase:   "",
			SftpPath:     v.SftpPath,
			Status:       v.Status,
		}
		if v.IsSshKey == 1 {
			ppp.IsSshKey = "是"
			ppp.KeyFile = v.PrivateKey[:strings.LastIndex(v.PrivateKey, ".")]
		} else {
			ppp.IsSshKey = "否"
			ppp.KeyFile = ""
		}

		// 获取高级设置
		err = savePassportAdvanced(&ppp, v.ID)
		if err != nil {
			log.Errorf("DB error: %v", err)
			return FailWithDataOperate(c, 500, "获取设备帐号失败", "", nil)
		}
		data = append(data, ppp)
	}
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	return SuccessWithOperate(c, " ", data)
}

// PassPortListForAssetEndpoint 获取当前设备的所有帐号
func PassPortListForAssetEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}
	id := c.Param("id")
	data, err := newAssetRepository.GetPassportByAssetId(context.TODO(), id)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	return SuccessWithOperate(c, " ", data)
}

// GetAssetGroupEndpoint 获取当前设备的所在的所有组
func GetAssetGroupEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}
	id := c.Param("id")
	data, err := newAssetGroupRepository.GetAssetGroupByAssetId(context.TODO(), id)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	return SuccessWithOperate(c, " ", data)
}

// GetAssetPolicyEndpoint 获取设备的所有运维策略
func GetAssetPolicyEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}
	id := c.Param("id")
	data, err := newAssetRepository.GetOpsPolicyByAssetId(context.TODO(), id)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	return Success(c, data)
}

// GetAssetCommandPolicyEndpoint 获取设备的所有指令策略
func GetAssetCommandPolicyEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}
	id := c.Param("id")
	data, err := newAssetRepository.GetCmdPolicyByAssetId(context.TODO(), id)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	return Success(c, data)
}

// PassPortDisableEndpoint 禁用设备账号
func PassPortDisableEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}
	id := c.Param("id")
	passPort, err := newAssetRepository.GetPassPortByID(context.TODO(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return FailWithDataOperate(c, 404, "设备帐号不存在", "设备账号-禁用: 原因[设备帐号不存在]", nil)
		}
		log.Error("获取设备帐号失败: %v", err)
		return FailWithDataOperate(c, 500, "获取设备帐号失败", "设备账号-禁用: 原因[获取设备帐号失败]", nil)
	}
	a, err := newAssetRepository.GetAssetByID(context.TODO(), passPort.AssetId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return FailWithDataOperate(c, 404, "设备不存在", "设备账号-禁用: 原因[设备不存在]", nil)
		}
		log.Error("获取设备失败: %v", err)
		return FailWithDataOperate(c, 500, "获取设备失败", "设备账号-禁用: 原因[获取设备失败]", nil)
	}
	err = newAssetRepository.DisablePassport(context.TODO(), id)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "禁用失败", "", nil)
	}
	return SuccessWithOperate(c, "设备账号-禁用: 设备名称["+a.Name+"], 设备账号["+passPort.Passport+"]", nil)
}

// PassPortEnableEndpoint 启用设备帐号
func PassPortEnableEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}
	id := c.Param("id")
	passPort, err := newAssetRepository.GetPassPortByID(context.TODO(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return FailWithDataOperate(c, 404, "设备帐号不存在", "设备账号-启用: 原因[设备帐号不存在]", nil)
		}
		return FailWithDataOperate(c, 500, "获取设备帐号失败", "", nil)
	}
	a, err := newAssetRepository.GetAssetByID(context.TODO(), passPort.AssetId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return FailWithDataOperate(c, 404, "设备不存在", "设备账号-启用: 原因[设备不存在]", nil)
		}
		return FailWithDataOperate(c, 500, "获取设备失败", "", nil)
	}
	err = newAssetRepository.EnablePassport(context.TODO(), id)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "启用失败", "", nil)
	}
	return SuccessWithOperate(c, "设备账号-启用: 设备名称["+a.Name+"], 设备账号["+passPort.Passport+"]", nil)
}

// AssetBatchEditEndpoint 批量编辑设备
func AssetBatchEditEndpoint(c echo.Context) error {
	_, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请先登录")
	}
	id := c.Param("id")
	ids := utils.IdHandle(id)
	var ae dto.AssetForBatchUpdate
	if err := c.Bind(&ae); err != nil {
		log.Error("参数错误: %v", err)
		return FailWithDataOperate(c, 500, "批量编辑失败", "", nil)
	}
	ae.AssetIds = ids
	var name string
	for _, v := range ids {
		a, err := newAssetRepository.GetAssetByID(context.TODO(), v)
		if err != nil {
			return FailWithDataOperate(c, 500, "批量编辑失败", "设备列表-编辑: 批量编辑设备: 失败原因[设备未找到]", nil)
		}
		name = name + a.Name + ","
	}
	err := newAssetRepository.BatchEditAsset(context.TODO(), &ae)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "批量编辑失败", "", nil)
	}
	return SuccessWithOperate(c, "设备列表-编辑: 批量编辑设备, 设备名称["+name+"]", nil)
}

// NewAssetExportEndpoint 导出设备列表
func NewAssetExportEndpoint(c echo.Context) error {
	u, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	var departmentIds []int64
	err := GetChildDepIds(u.DepartmentId, &departmentIds)
	if err != nil {
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}

	assetForExport, err := newAssetRepository.GetPassportToExport(context.TODO(), departmentIds)
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", nil)
	}
	assetStringsForExport := make([][]string, len(assetForExport))
	for i, v := range assetForExport {
		asset := utils.Struct2StrArr(v)
		assetStringsForExport[i] = make([]string, len(asset))
		assetStringsForExport[i] = asset
	}
	assetHeaderForExport := []string{"设备名称", "设备地址", "部门名称", "系统类型", "描述", "设备账号名称", "端口", "ssh_key", "设备账号协议类型", "登录方式", "SFTP访问路径", "设备账号角色", "状态"}
	assetFileNameForExport := "设备列表"
	file, err := utils.CreateExcelFile(assetFileNameForExport, assetHeaderForExport, assetStringsForExport)
	if err != nil {
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	fileName := "设备列表.xlsx"

	// 记录操作日志
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		Result:          "成功",
		LogTypes:        "用户日志",
		Names:           u.Nickname,
		Created:         utils.NowJsonTime(),
		LogContents:     "设备列表-导出: 导出文件[" + fileName + "]",
		Users:           u.Username,
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
	//
	//return WriteFileToClient(c, file, fileName)

	//设置请求头  使用浏览器下载
	//c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	//return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
	//将文件写入到response并转为xlsx格式
	//c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	//c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	//c.Response().WriteHeader(http.StatusOK)
	//c.Response().Write(buff.Bytes())
	// 将文件以数据流方式传入客户端并命名为fileName
	//c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	//c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	//c.Response().WriteHeader(http.StatusOK)
	//c.Response().Write(buff.Bytes())
	//return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))

	// 将文件以数据流方式传入客户端并命名为fileName
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	//c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	//c.Response().WriteHeader(http.StatusOK)
	//c.Response().Write(buff.Bytes())
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
}

// NewAssetImportEndpoint 导入设备列表
func NewAssetImportEndpoint(c echo.Context) error {
	// 获取文件
	u, f := GetCurrentAccountNew(c)
	if f == false {
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}
	var departmentIds []int64
	file, err := c.FormFile("file")
	isCover, _ := strconv.ParseBool(c.FormValue("is_cover"))

	ac, err := newAssetRepository.GetAssetCount(context.TODO())
	if err != nil {
		log.Errorf("Get Asset Count err: %v", err)
		return FailWithDataOperate(c, 500, "导入失败", "设备列表-导入: 查询数据库中的所有设备数量失败", nil)
	}
	err, isover := sysMaintainService.IsOverAuthResourceCountLimit(ac)
	if nil != err {
		log.Error("设备列表-导入: 获取授权许可最大资源数失败")
		return FailWithDataOperate(c, 500, "导入失败, 请联系技术人员查看系统日志以确定失败原因", "", nil)
	}
	if isover {
		return FailWithDataOperate(c, 400, "系统当前设备数已达到授权许可的最大资源数", "设备列表-导入: 导入失败, 失败原因[系统当前设备数已达到授权许可的最大资源数]", nil)
	}

	err = GetChildDepIds(u.DepartmentId, &departmentIds)
	if err != nil {
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}
	if isCover == true {
		// 删除所有设备
		as, err := newAssetRepository.DeleteByDepartmentId(context.TODO(), departmentIds)
		if err != nil {
			return FailWithDataOperate(c, 500, "导入失败", "", nil)
		}
		for i := 0; i < len(as); i++ {
			UpdateUserAssetAppAppserverDep(constant.ASSET, constant.DELETE, as[i].DepartmentId, int64(-1))
		}
		ac = 0
	}
	if err != nil {
		log.Errorf("FormFile Error: %v", err)
		return FailWithDataOperate(c, 500, "导入失败", "", err)
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
	data, err := xlsx.GetRows(xlsx.GetSheetName(xlsx.GetActiveSheetIndex()))
	if nil != err {
		log.Errorf("GetRows Error: %v", err)
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}

	total := len(data)
	if total <= 1 {
		return FailWithDataOperate(c, 400, "导入文件数据为空", "用户列表-导入: 文件名["+file.Filename+"], 失败原因[导入文件数据为空]", nil)
	}
	if len(data[0]) != 12 {
		return FailWithDataOperate(c, 400, "导入文件格式错误", "用户列表-导入: 文件名["+file.Filename+"], 失败原因[导入文件格式错误]", nil)
	}
	var nameSuccess, nameFiled string
	var successNum, filedNum int

	for i := range data {
		if i == 0 {
			continue
		}
		if data[i][0] == "" {
			continue
		}
		var (
			p            int
			pt           string
			pw           = ""
			departmentId int64
			name         string
		)
		if len(data[i][1]) != 0 {
			index := strings.IndexByte(data[i][1], '{')
			if index != -1 {
				Id, err := strconv.ParseInt(data[i][1][index+1:len(data[i][2])-1], 10, 64)
				if err != nil {
					log.Errorf("ParseInt Error: %v", err)
				}
				departmentId = Id
				if _, err := departmentRepository.FindById(Id); err != nil {
					nameFiled += "第" + strconv.Itoa(i+1) + "行部门不存在;"
					filedNum++
					continue
				}
				if IsDepIdBelongDepIds(Id, departmentIds) == true {
					nameFiled += "第" + strconv.Itoa(i+1) + "行部门当前用户无权限操作;"
					filedNum++
					continue
				}
			} else {
				department, _ := departmentRepository.FindByName(data[i][1])
				departmentId = department.ID
				if IsDepIdBelongDepIds(departmentId, departmentIds) == false {
					nameFiled += "第" + strconv.Itoa(i+1) + "行部门当前用户无权限操作;"
					filedNum++
					continue
				}
			}
		} else {
			nameFiled += "第" + strconv.Itoa(i+1) + "行部门不能为空;"
			filedNum++
			continue
		}
		if data[i][7] != "" {
			p, _ = strconv.Atoi(data[i][8])
			if data[i][11] != "" {
				if data[i][11] == "1" {
					pt = "Administrator"
				} else {
					pt = "Ordinary"
				}
			} else {
				nameFiled += "第" + strconv.Itoa(i+1) + "行账号角色不能为空;"
				filedNum++
				continue
			}
			if data[i][6] != "" {
				encryptedCBC, err := utils.AesEncryptCBC([]byte(data[i][6]), global.Config.EncryptionPassword)
				if err != nil {
					log.Error("设备密码加密失败", err)
				}
				pw = base64.StdEncoding.EncodeToString(encryptedCBC)
			}
		}
		asset, err := newAssetRepository.GetAssetByName(context.TODO(), data[i][0])
		if err == gorm.ErrRecordNotFound {
			err, isover := sysMaintainService.IsOverAuthResourceCountLimit(ac + i)
			if nil != err {
				log.Error("设备列表-导入: 获取授权许可最大资源数失败")
				return FailWithDataOperate(c, 500, "导入失败, 请联系技术人员查看系统日志以确定失败原因", "", nil)
			}
			if isover {
				return FailWithDataOperate(c, 400, "系统当前设备数已达到授权许可的最大资源数", "用户列表-导入: 导入失败, 失败原因[系统当前设备数已达到授权许可的最大资源数]", nil)
			}
			systemType, err := systemTypeRepository.GetSystemTypeByName(context.TODO(), data[i][3])
			if err != nil {
				nameFiled += "第" + strconv.Itoa(i+1) + "行系统类型不存在;"
				filedNum++
				continue
			}
			a := model.NewAsset{
				ID:            utils.UUID(),
				Name:          data[i][0],
				DepartmentId:  departmentId,
				IP:            data[i][2],
				AssetType:     systemType.ID,
				Created:       utils.NowJsonTime(),
				PassPortCount: 1,
				Info:          data[i][4],
			}
			name = a.Name + "[" + a.IP + "]" + "[" + data[i][8] + "]" + "[" + data[i][7] + "]"
			pp := model.PassPort{
				ID:           utils.UUID(),
				Name:         name,
				AssetId:      a.ID,
				AssetType:    a.AssetType,
				AssetName:    a.Name,
				Ip:           a.IP,
				DepartmentId: a.DepartmentId,
				Passport:     data[i][5],
				Password:     pw,
				LoginType:    "自动登录",
				Protocol:     data[i][7],
				Port:         p,
				PassportType: pt,
				Status:       "enable",
				Created:      utils.NowJsonTime(),
				SftpPath:     data[i][10],
			}
			err = newAssetRepository.CreateByAssetAndPassport(context.TODO(), a, pp)
			if err != nil {
				log.Errorf("CreateByAssetAndPassport Error: %v", err)
			}
			UpdateUserAssetAppAppserverDep(constant.ASSET, constant.ADD, a.DepartmentId, int64(-1))
			nameSuccess += name + ","
			successNum++
		} else if err == nil {
			name = asset.Name + "[" + asset.IP + "]" + "[" + data[i][8] + "]" + "[" + data[i][7] + "]"
			pp := model.PassPort{
				ID:           utils.UUID(),
				Name:         name,
				AssetId:      asset.ID,
				AssetType:    asset.AssetType,
				AssetName:    asset.Name,
				Ip:           asset.IP,
				DepartmentId: asset.DepartmentId,
				Passport:     data[i][5],
				Password:     pw,
				LoginType:    "自动登录",
				Protocol:     strings.ToLower(data[i][7]),
				Port:         p,
				Created:      utils.NowJsonTime(),
				PassportType: pt,
				Status:       "enable",
				SftpPath:     data[i][10],
			}
			err := newAssetRepository.CreateByPassport(context.TODO(), pp)
			if err != nil {
				log.Errorf("CreateByPassport Error: %v", err)
			}
			nameSuccess += name + ","
			successNum++
		} else {
			log.Errorf("GetAssetByName Error: %v", err)
			return FailWithDataOperate(c, 500, "导入失败", "", err)
		}
	}

	return SuccessWithOperate(c, "设备列表-导入: "+strconv.Itoa(successNum)+"条成功,"+strconv.Itoa(filedNum)+"条失败;成功:"+nameSuccess+";失败:"+nameFiled, nil)
}

func NewAssetDownloadTemplateEndpoint(c echo.Context) error {
	userHeaderForExport := []string{"设备名称(必填)", "部门名称(如果部门名称重复请在名称后加{{id}} 例如:测试部{{380}})(必填)", "设备地址(必填)", "系统类型（Linux Windows....)（必填）", "描述", "账号", "密码", "协议类型(SSH RDP TELNET VNC FTP SFTP X11)（必填）", "端口(必填)", "运行参数", "访问目录", "1代表管理员0代表普通用户(必填)"}
	userFileNameForExport := "设备列表"
	file, err := utils.CreateTemplateFile(userFileNameForExport, userHeaderForExport)
	if err != nil {
		log.Errorf("CreateExcelFile Error: %v", err)
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	fileName := "设备列表导入模板.xlsx"
	var buff bytes.Buffer
	if err = file.Write(&buff); err != nil {
		log.Errorf("Write Error: %v", err)
		return FailWithDataOperate(c, 500, "导出失败", "", err)
	}
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
}

// NewAssetDownloadPrivateKeyEndpoint 下载私钥文件
func NewAssetDownloadPrivateKeyEndpoint(c echo.Context) error {
	id := c.Param("id")
	u, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 403, "请登录")
	}

	passport, err := newAssetRepository.GetPassportById(context.TODO(), id)
	if err != nil {
		log.Errorf("GetPassportById Error: %v", err)
		return Fail(c, 500, "下载失败")
	}

	filePath := path.Join(constant.PrivateKeyPath, passport.PrivateKey)
	fileName := passport.PrivateKey[:strings.LastIndex(passport.PrivateKey, ".")]

	if !utils.FileExists(filePath) {
		return Fail(c, 500, "下载失败")
	}

	// 记录操作日志
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		Result:          "成功",
		LogTypes:        "用户日志",
		Created:         utils.NowJsonTime(),
		LogContents:     "设备账号-编辑: 账号[" + passport.Name + "]下载私钥文件",
		Users:           u.Username,
	}
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}
	// 读取文件转为*bytes.Buffer
	file, err := os.Open(filePath)
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
