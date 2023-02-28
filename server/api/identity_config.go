package api

import (
	"fmt"
	"strings"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"gorm.io/gorm"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

// 获取登录锁定配置
func LoginLockConfigGetEndpoint(c echo.Context) error {
	loginLockConfig, err := identityConfigRepository.FindLonginConfig()
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	return Success(c, loginLockConfig)
}

// 修改登录锁定配置
func LoginLockConfigUpdateEndpoint(c echo.Context) error {
	var loginLockConfig model.LoginConfig
	if err := c.Bind(&loginLockConfig); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "数据错误", "", err)
	}
	fmt.Println(loginLockConfig)
	if err := identityConfigRepository.UpdateLoginConfig(&loginLockConfig); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	return SuccessWithOperate(c, "系统配置-安全配置: 修改[登录锁定配置]", nil)
}

// 获取密码策略配置
func PasswordConfigGetEndpoint(c echo.Context) error {
	passwordConfig, err := identityConfigRepository.FindPasswordConfig()
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", err)
	}
	return Success(c, passwordConfig)
}

// 修改密码策略配置
func PasswordConfigUpdateEndpoint(c echo.Context) error {
	var passwordConfig model.PasswordConfig
	if err := c.Bind(&passwordConfig); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "数据错误", "", err)
	}
	if err := identityConfigRepository.UpdatePasswordConfig(&passwordConfig); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	return SuccessWithOperate(c, "系统配置-安全配置: 修改[密码策略配置]", nil)
}

// RemoteManageHostGetEndpoint 获取远程管理主机
func RemoteManageHostGetEndpoint(c echo.Context) error {
	remoteManageHost, err := propertyRepository.FindByName("remote_manage_host")
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return Success(c, "")
	}
	return Success(c, H{
		"remote_login_ip": remoteManageHost.Value,
	})
}

// RemoteManageHostUpdateEndpoint 修改远程管理主机
func RemoteManageHostUpdateEndpoint(c echo.Context) error {
	var mp map[string]string
	if err := c.Bind(&mp); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "数据错误", "", err)
	}

	var remoteManageHost model.Property
	remoteManageHost.Name = "remote_manage_host"
	remoteManageHost.Value = mp["remote_login_ip"]

	// 验证是否为地址或地址段
	// 通过回车分割
	if len(remoteManageHost.Value) > 0 {
		if strings.Contains(remoteManageHost.Value, "\r\n") {
			ips := strings.Split(remoteManageHost.Value, "\r\n")
			for _, ip := range ips {
				if strings.Contains(ip, "-") {
					ipRange := strings.Split(ip, "-")
					if !utils.IsIP(ipRange[0]) || !utils.IsIP(ipRange[1]) {
						return FailWithDataOperate(c, 500, "IP格式错误", "", nil)
					}
				} else if !utils.IsIP(ip) {
					return FailWithDataOperate(c, 500, "IP格式错误", "", nil)
				}
			}
		}
		if strings.Contains(remoteManageHost.Value, "\n") {
			ips := strings.Split(remoteManageHost.Value, "\n")
			for _, ip := range ips {
				if strings.Contains(ip, "-") {
					ipRange := strings.Split(ip, "-")
					if !utils.IsIP(ipRange[0]) || !utils.IsIP(ipRange[1]) {
						return FailWithDataOperate(c, 500, "IP格式错误", "", nil)
					}
				} else if !utils.IsIP(ip) {
					return FailWithDataOperate(c, 500, "IP格式错误", "", nil)
				}
			}
		}
	}

	property, err := propertyRepository.FindByName("remote_manage_host")
	if err == gorm.ErrRecordNotFound {
		if err := propertyRepository.Create(&remoteManageHost); err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		} else {
			return SuccessWithOperate(c, "系统配置-安全配置: 添加远程管理主机地址["+remoteManageHost.Value+"]", nil)
		}
	}

	if err := propertyRepository.UpdateByName(&remoteManageHost, "remote_manage_host"); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	return SuccessWithOperate(c, "系统配置-安全配置: 修改远程管理主机地址["+property.Value+"->"+remoteManageHost.Value+"]", nil)
}

// ExportPasswordConfigEndpoint 导出密码配置
func ExportPasswordConfigEndpoint(c echo.Context) error {

	var req map[string]interface{}

	if err := c.Bind(&req); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "数据错误", "", err)
	}
	var exportPasswordConfig model.Property
	if v, ok := req["export_password_config"]; ok {
		exportPasswordConfig.Value = v.(string)
	}
	fmt.Println(exportPasswordConfig)

	exportPasswordConfig.Name = "export_password"
	fmt.Println(exportPasswordConfig)

	pwd := exportPasswordConfig.Value
	if pwd != "" {
		pass, err := utils.Encoder.Encode([]byte(pwd))
		if err != nil {
			log.Errorf("密码加密失败: %v", err)
			return FailWithDataOperate(c, 500, "编辑失败", "", err)
		}
		exportPasswordConfig.Value = string(pass)
	}

	_, err := propertyRepository.FindByName("export_password")
	if err == gorm.ErrRecordNotFound {
		if err := propertyRepository.Create(&exportPasswordConfig); err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		} else {
			return SuccessWithOperate(c, "系统配置-安全配置:添加导出密码配置", nil)
		}
	} else if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if err := propertyRepository.UpdateByName(&exportPasswordConfig, "export_password"); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	return SuccessWithOperate(c, "系统配置-安全配置: 修改导出密码配置", nil)
}
