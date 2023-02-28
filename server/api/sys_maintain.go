package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"tkbastion/pkg/constant"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/model"
	tkservice "tkbastion/server/service"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
)

// 许可管理
type LicenseManagement struct {
	CustomerName      string `json:"customerName"`
	AuthType          string `json:"authType"`
	ProductId         string `json:"productId"`
	AuthResourceCount string `json:"authResourceCount"`
	OverdueTime       string `json:"overdueTime"`
}

func LicenseManagementGetEndpoint(c echo.Context) error {
	item := propertyRepository.FindAuMap("tkbastion")

	origData, err := base64.StdEncoding.DecodeString(item["tkbastion-value1"])
	if nil != err {
		log.Errorf("DecodeString Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	decryptedCBC, err := utils.AesDecryptCBC(origData, []byte("trunkeyTkbastion"))
	if err != nil {
		log.Errorf("AesDecryptCBC Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	customerName := string(decryptedCBC)

	origData, err = base64.StdEncoding.DecodeString(item["tkbastion-value2"])
	if nil != err {
		log.Errorf("DecodeString Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	decryptedCBC, err = utils.AesDecryptCBC(origData, []byte("trunkeyTkbastion"))
	if err != nil {
		log.Errorf("AesDecryptCBC Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	authType := string(decryptedCBC)

	origData, err = base64.StdEncoding.DecodeString(item["tkbastion-value4"])
	if nil != err {
		log.Errorf("DecodeString Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	decryptedCBC, err = utils.AesDecryptCBC(origData, []byte("trunkeyTkbastion"))
	if err != nil {
		log.Errorf("AesDecryptCBC Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	authResourceCount := string(decryptedCBC)

	origData, err = base64.StdEncoding.DecodeString(item["tkbastion-value5"])
	if nil != err {
		log.Errorf("DecodeString Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	decryptedCBC, err = utils.AesDecryptCBC(origData, []byte("trunkeyTkbastion"))
	if err != nil {
		log.Errorf("AesDecryptCBC Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}
	overdueTime := string(decryptedCBC)

	var licenseManagement LicenseManagement
	licenseManagement.CustomerName = customerName
	if "true" == authType {
		authType = "已授权"
	} else {
		authType = "未授权"
	}
	licenseManagement.AuthType = authType
	licenseManagement.ProductId = item["tkbastion-value3"][:24]
	licenseManagement.AuthResourceCount = authResourceCount
	licenseManagement.OverdueTime = overdueTime

	return SuccessWithOperate(c, "", licenseManagement)
}

func LicenseManagementDownLicenseEndpoint(c echo.Context) error {
	productId, err := sysMaintainService.GetProductId()
	if nil != err {
		log.Error("许可管理-下载申请许可文件失败")
		return FailWithDataOperate(c, 500, "下载失败", "", nil)
	}

	// 生成License文件
	var buff bytes.Buffer
	_, err = buff.Write([]byte(productId))
	if err != nil {
		log.Errorf("Write Error: %v", err.Error())
		return FailWithDataOperate(c, 500, "下载失败", "", nil)
	}

	user, _ := GetCurrentAccountNew(c)
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "",
		Created:         utils.NowJsonTime(),
		Users:           user.Username,
		Names:           user.Nickname,
		LogContents:     "许可管理-下载: 许可申请文件",
		Result:          "成功",
	}
	err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}

	//设置请求头  使用浏览器下载
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename=tkbastion_license.rc")
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
}

// 许可管理控制的是是否可运维主机(不包括登录测试)以及添加设备
// 考虑到实际客户使用场景, 此功能不需要做太多安全检测防许可滥用, 同时太多安全检测如果出Bug影响很严重
// 因此这里暂时搁置一些安全检测, 如根据value6解码后值判断, ID是否匹配系统ID, 剩下4个值是否和value6解码后值相等
// 上述情况下, 直接杀死程序吗  -->  目前不判断
// 目前只做一种安全检测, 通过value6解码值重置value1-5, 但前提是我们默认, value6值是未经"篡改"的
// 同时, 我们添加设备时 及 运维资产时  也假设value1-5经过了"正确的"value6修复后, 也是"正确的", 不需再进行ID是否是机器ID或 Value1-5(不包含3即ID)组合后与签名是否相等判断
func LicenseManagementImportLicenseEndpoint(c echo.Context) error {
	licenseFile, err := c.FormFile("licenseFile")
	if nil != err {
		log.Errorf("FormFile Error: %v", err)
		log.Error("许可管理-导入许可文件失败")
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}

	if "tkbastion_license.rc" != licenseFile.Filename {
		return FailWithDataOperate(c, 400, "许可文件格式不正确", "许可管理-导入: 文件名称["+licenseFile.Filename+"], 失败原因[文件格式不正确]", nil)
	}

	src, err := licenseFile.Open()
	if nil != err {
		log.Errorf("Open Error: %v", err)
		log.Error("许可管理-导入许可文件失败")
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}
	defer func(src multipart.File) {
		err := src.Close()
		if nil != err {
			log.Errorf("Close Error: %v", err)
			log.Error("许可管理-关闭打开文件连接失败")
		}
	}(src)

	license, err := ioutil.ReadAll(src)
	if nil != err {
		log.Errorf("ReadAll Error: %v", err)
		log.Error("许可管理-导入许可文件失败")
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}
	licenseStr := string(license)

	licenseStrArr, err := sysMaintainService.DecryptLicenseContent(licenseStr)
	if nil != err {
		log.Error("许可管理-解密许可文件内容失败")
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}

	productId, err := sysMaintainService.GetProductId()
	if nil != err {
		log.Error("许可管理-获取产品PID失败")
		return FailWithDataOperate(c, 500, "导入失败", "", nil)
	}

	if productId != licenseStrArr[2] {
		return FailWithDataOperate(c, 400, "许可文件不合法", "许可管理-导入: 文件名称["+licenseFile.Filename+"], 失败原因[许可文件不合法]", nil)
	}

	err = sysMaintainService.UpdateProductLicenseInfo(licenseStrArr, licenseStr)
	if nil != err {
		log.Error("许可管理-更新产品授权许可信息失败")
		return FailWithDataOperate(c, 500, "授权许可信息更新失败", "", nil)
	}

	origData, err := base64.StdEncoding.DecodeString(licenseStrArr[3])
	if nil != err {
		log.Errorf("DecodeString Error: %v", err)
		log.Error("许可管理-根据新许可文件授权资源数调整旧的设备数失败")
		return FailWithDataOperate(c, 500, "根据许可文件授权资源数调整旧的设备数失败", "", nil)
	}
	decryptedCBC, err := utils.AesDecryptCBC(origData, []byte("trunkeyTkbastion"))
	if err != nil {
		log.Errorf("AesDecryptCBC Error: %v", err)
		log.Error("许可管理-根据新许可文件授权资源数调整旧的设备数失败")
		return FailWithDataOperate(c, 500, "根据许可文件授权资源数调整旧的设备数失败", "", nil)
	}
	authResourceCount, err := strconv.Atoi(string(decryptedCBC))
	if nil != err {
		log.Errorf("Atoi Error: %v", err)
		log.Error("许可管理-根据新许可文件授权资源数调整旧的设备数失败")
		return FailWithDataOperate(c, 500, "根据许可文件授权资源数调整旧的设备数失败", "", nil)
	}

	// 根据此authResourceCount变量值, 调整旧的设备个数, 若旧设备个数大于此值, 删除后添加的设备
	err = newAssetRepository.UpdateAssetByAuthResourceCount(context.TODO(), authResourceCount)
	if nil != err {
		log.Error("许可管理-根据新许可文件授权资源数调整旧的设备数失败")
		return FailWithDataOperate(c, 500, "根据许可文件授权资源数调整旧的设备数失败", "", nil)
	}

	return SuccessWithOperate(c, "许可管理-导入: 文件名称["+licenseFile.Filename+"]", nil)
}

// SysVersionGetEndpoint 系统版本
func SysVersionGetEndpoint(c echo.Context) error {
	return Success(c, global.Version)
}

// SysUpgradeGetEndpoint 系统版本
func SysUpgradeGetEndpoint(c echo.Context) error {
	file, err := c.FormFile("upgradeFile")
	if nil != err {
		log.Error("系统升级-获取文件失败")
		return FailWithDataOperate(c, 500, "获取文件失败", "", nil)
	}
	if file.Filename != constant.UpgradeFileName {
		return FailWithDataOperate(c, 400, "文件不合法", "系统升级-上传升级包: 文件名称["+file.Filename+"], 失败原因[文件不合法]", nil)
	}
	// 将升级包保存/data/tkbastion/upgrade目录下
	if _, err := os.ReadDir(constant.UpgradePath); os.IsNotExist(err) {
		if err := os.MkdirAll(constant.UpgradePath, 0755); nil != err {
			log.Error("系统升级-创建升级目录失败")
			return FailWithDataOperate(c, 500, "创建升级目录失败", "", nil)
		}
	}
	upgradeFilePath := path.Join(constant.UpgradePath, file.Filename)
	src, err := file.Open()
	if nil != err {
		log.Error("系统升级-打开文件失败")
		return FailWithDataOperate(c, 500, "打开文件失败", "", nil)
	}
	defer func(src multipart.File) {
		err := src.Close()
		if err != nil {
			log.Errorf("Close Error: %v", err)
		}
	}(src)

	dst, err := os.Create(upgradeFilePath)
	if nil != err {
		log.Error("系统升级-创建文件失败")
		return FailWithDataOperate(c, 500, "创建文件失败", "", nil)
	}
	defer func(dst *os.File) {
		err := dst.Close()
		if err != nil {
			log.Errorf("Close Error: %v", err)
		}
	}(dst)

	if _, err = io.Copy(dst, src); err != nil {
		log.Error("系统升级-写入文件失败")
		return FailWithDataOperate(c, 500, "写入文件失败", "", nil)
	}

	return SuccessWithOperate(c, "系统升级-上传升级包: 文件名称["+file.Filename+"]", nil)
}

// SysRebootEndpoint 重启
func SysRebootEndpoint(c echo.Context) error {
	user, _ := GetCurrentAccountNew(c)
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "",
		Created:         utils.NowJsonTime(),
		Users:           user.Username,
		Names:           user.Nickname,
		LogContents:     "系统工具-重启: [重启操作]",
		Result:          "成功",
	}

	cmd := exec.Command("bash", "-c", "reboot")
	err := global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}
	if err := cmd.Run(); err != nil {
		log.Errorf("Run Error:%V", err)
		return FailWithDataOperate(c, 422, "重启失败", "系统维护-系统工具: 重启", err)
	}
	return SuccessWithOperate(c, "系统工具-重启: 重启", nil)
}

// SysShutdownEndpoint 关机
func SysShutdownEndpoint(c echo.Context) error {
	user, _ := GetCurrentAccountNew(c)
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "",
		Created:         utils.NowJsonTime(),
		Users:           user.Username,
		Names:           user.Nickname,
		LogContents:     "系统工具-关机: [关机操作]",
		Result:          "成功",
	}
	cmd := exec.Command("bash", "-c", "poweroff")
	err := global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}
	if err := cmd.Run(); err != nil {
		log.Errorf("Run Error:%V", err)
		return FailWithDataOperate(c, 422, "关机失败", "系统维护-系统工具: 关机", err)
	}
	return SuccessWithOperate(c, "系统工具-关机: 关机", nil)
}

// SysRestoreEndpoint 恢复出厂设置
func SysRestoreEndpoint(c echo.Context) error {
	mysqlAddress := global.Config.Mysql.Hostname
	mysqlPort := strconv.Itoa(global.Config.Mysql.Port)
	mysqlUser := global.Config.Mysql.Username
	mysqlPassword := global.Config.Mysql.Password
	mysqlDatabase := global.Config.Mysql.Database
	err := backupService.ClearMySqlDb(mysqlAddress, mysqlPort, mysqlUser, mysqlPassword, mysqlDatabase)
	if err != nil {
		log.Errorf("Run Error:%V", err)
		return FailWithDataOperate(c, 422, "恢复出厂设置失败", "系统维护-系统工具: 恢复出厂设置", err)
	}
	// 删除/data/tkbastion/drive目录下的所有文件
	if _, err := utils.ExecShell("rm -rf /data/tkbastion/drive/*"); err != nil {
		log.Errorf("删除/data/tkbastion/drive目录下的所有文件失败: %v", err)
	}
	// 删除/data/tkbastion/recording目录下的所有文件
	if _, err := utils.ExecShell("rm -rf /data/tkbastion/recording/*"); err != nil {
		log.Errorf("删除/data/tkbastion/recording目录下的所有文件失败: %v", err)
	}
	// 删除/data/tkbastion/regular-report目录下的所有文件
	if _, err := utils.ExecShell("rm -rf /data/tkbastion/regular-report/*"); err != nil {
		log.Errorf("删除/data/tkbastion/regular-report目录下的所有文件失败: %v", err)
	}
	tkservice.SetupService()
	InitService()
	if err := InitDBData(); nil != err {
		log.WithError(err).Errorf("初始化数据异常,异常信息: %v", err.Error())
		os.Exit(0)
	}
	if err := newJobService.ReloadJob(); err != nil {
		log.Errorf("初始化定时任务异常,异常信息: %v", err.Error())
	}
	InitCasbin()
	//InitVideo()
	//InitContainerId()
	InitSession()
	return SuccessWithOperate(c, "系统工具-恢复出厂设置: 恢复出厂设置", nil)
}

// SysUsageGetEndpoint 获取系统利用率
func SysUsageGetEndpoint(c echo.Context) error {
	typeUsage := c.QueryParam("typeUsage")
	interval := c.QueryParam("interval")
	startTime := c.QueryParam("startTime")
	endTime := c.QueryParam("endTime")
	usage, err := propertyRepository.GetSysUsage(startTime, endTime, interval, typeUsage)
	if err != nil {
		log.Errorf("DB Error: %v", err)
		log.Errorf("GetSysUsage Error: %v", "获取cpu利用率失败")
		return FailWithDataOperate(c, 500, "获取系统利用率失败", "", err)
	}
	return Success(c, usage)
}
