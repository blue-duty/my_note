package api

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"tkbastion/pkg/constant"
	"tkbastion/pkg/global"
	"tkbastion/pkg/guacd"
	"tkbastion/pkg/log"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
)

// 审计文件备份配置
// start =========> routes.go 742,743

func AuditBackupGetEndpoint(c echo.Context) error {
	sysLogsConfig, err := propertyRepository.FindMapByNames(constant.AuditBackup)
	if err != nil {
		log.Errorf("获取审计备份配置失败:%v", err)
		return FailWithDataOperate(c, 500, "获取审计备份配置失败", "", err)
	}
	delete(sysLogsConfig, "remote_backup_password")

	return Success(c, sysLogsConfig)
}

func AuditBackupUpdateEndpoint(c echo.Context) error {
	var item map[string]interface{}
	_, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithData(c, 401, "获取当前用户失败", "")
	}
	if err := c.Bind(&item); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if _, ok := item["enable_remote_automatic_backup"]; ok {
		if item["enable_remote_automatic_backup"] == "1" {
			// 验证地址格式，端口，路径
			if ip, ok := item["remote_backup_host"]; !ok {
				return FailWithData(c, 500, "远程备份地址不能为空", "")
			} else {
				if !utils.IsIP(ip.(string)) {
					return FailWithData(c, 500, "远程备份地址格式错误", "")
				}
			}
			if port, ok := item["remote_backup_port"]; !ok {
				return FailWithData(c, 500, "远程备份端口不能为空", "")
			} else {
				if port.(int) < 1 || port.(int) > 65535 {
					return FailWithData(c, 500, "远程备份端口格式错误", "")
				}
			}
			if path, ok := item["remote_backup_path"]; !ok {
				return FailWithData(c, 500, "远程备份路径不能为空", "")
			} else {
				if strings.Contains(path.(string), "/") {
					return FailWithData(c, 500, "远程备份路径格式错误", "")
				}
			}
		}
	}
	p1, err := propertyRepository.FindByName("enable_remote_automatic_backup")
	if err != nil {
		log.Errorf("获取审计备份配置失败:%v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	p2, err := propertyRepository.FindByName("enable_local_automatic_backup")
	if err != nil {
		log.Errorf("获取审计备份配置失败:%v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	op := "系统配置-存储配置: 修改远程审计备份["
	if _, ok := item["enable_remote_automatic_backup"]; ok {
		if item["enable_remote_automatic_backup"] == "0" {
			if p1.Value == "1" {
				//关闭远程备份
				err := global.SCHEDULER.RemoveByTag(constant.RemoteBackup)
				if err != nil {
					log.Errorf("关闭远程备份自动任务失败:%v", err)
				}
				err = propertyRepository.DeleteByNames(constant.RemoteAuditBackup)
				if err != nil {
					log.Errorf("删除远程审计备份配置失败:%v", err)
					return FailWithDataOperate(c, 500, "修改失败", "", err)
				}
				err = propertyRepository.UpdateByName(&model.Property{Name: "enable_remote_automatic_backup", Value: "0"}, "enable_remote_automatic_backup")
				if err != nil {
					log.Errorf("关闭远程备份失败:%v", err)
					return FailWithDataOperate(c, 500, "修改失败", "", err)
				}
				op = op + "状态: 开启->关闭]"
			} else {
				op = op + "状态: 未修改]"
			}
		} else {
			if p1.Value == "0" {
				err := newJobService.NewBackupJob(item["remote_backup_interval"].(string), constant.RemoteBackup)
				if err != nil {
					log.Errorf("开启远程备份失败:%v", err)
					return FailWithDataOperate(c, 500, "修改失败", "", err)
				}
				err = propertyRepository.DeleteByNames(constant.RemoteAuditBackup)
				if err != nil {
					log.Errorf("删除远程审计备份配置失败:%v", err)
					//return FailWithDataOperate(c, 500, "修改失败", "", err)
				}
				for _, v := range constant.RemoteAuditBackup {
					if v == "remote_backup_password" {
						continue
					}
					if _, ok := item[v]; ok {
						err := propertyRepository.Create(&model.Property{
							Name:  v,
							Value: fmt.Sprintf("%v", item[v]),
						})
						if err != nil {
							log.Errorf("修改远程审计备份配置失败:%v", err)
							return FailWithDataOperate(c, 500, "修改失败", "", err)
						}
					}
				}
				var passport string
				if item["remote_backup_password"] != nil {
					encryptedCBC, err := utils.AesEncryptCBC([]byte(item["remote_backup_password"].(string)), global.Config.EncryptionPassword)
					if err != nil {
						return err
					}
					passport = base64.StdEncoding.EncodeToString(encryptedCBC)
				}
				err = propertyRepository.Update(&model.Property{
					Name:  "remote_backup_password",
					Value: passport,
				})
				if err != nil {
					log.Errorf("修改远程审计备份配置-加密远程备份账号密码失败:%v", err)
					return FailWithDataOperate(c, 500, "修改失败", "", err)
				}
				err = propertyRepository.Update(&model.Property{
					Name:  "enable_remote_automatic_backup",
					Value: "1",
				})
				if err != nil {
					log.Errorf("修改远程审计备份配置失败:%v", err)
					return FailWithDataOperate(c, 500, "修改失败", "", err)
				}
				op = op + "状态: 关闭->开启, 远程备份时间: " + item["remote_backup_interval"].(string) + "]"
			} else {
				//关闭旧的定时任务
				err := global.SCHEDULER.RemoveByTag(constant.RemoteBackup)
				if err != nil {
					log.Errorf("关闭远程备份自动任务失败:%v", err)
				}
				err = newJobService.NewBackupJob(item["remote_backup_interval"].(string), constant.RemoteBackup)
				if err != nil {
					log.Errorf("修改远程备份失败:%v", err)
					return FailWithDataOperate(c, 500, "修改失败", "", err)
				}
				err = propertyRepository.DeleteByNames(constant.RemoteAuditBackup)
				if err != nil {
					log.Errorf("删除远程审计备份配置失败:%v", err)
					return FailWithDataOperate(c, 500, "修改失败", "", err)
				}
				for _, v := range constant.RemoteAuditBackup {
					if _, ok := item[v]; ok {
						if v == "remote_backup_password" {
							continue
						}
						err := propertyRepository.Create(&model.Property{
							Name:  v,
							Value: fmt.Sprintf("%v", item[v]),
						})
						if err != nil {
							log.Errorf("修改远程审计备份配置失败:%v", err)
							return FailWithDataOperate(c, 500, "修改失败", "", err)
						}
					}
				}
				var passport string
				if item["remote_backup_password"] != nil {
					encryptedCBC, err := utils.AesEncryptCBC([]byte(item["remote_backup_password"].(string)), global.Config.EncryptionPassword)
					if err != nil {
						return err
					}
					passport = base64.StdEncoding.EncodeToString(encryptedCBC)
				}
				err = propertyRepository.Create(&model.Property{
					Name:  "remote_backup_password",
					Value: passport,
				})
				if err != nil {
					log.Errorf("修改远程审计备份配置-加密远程备份账号密码失败:%v", err)
					return FailWithDataOperate(c, 500, "修改失败", "", err)
				}
				op = op + "状态: 开启->开启, 远程备份时间: " + item["remote_backup_interval"].(string) + "]"
			}
		}
	} else {
		op = op + "状态: 未修改]"
	}
	op = op + ", 修改本地审计备份["
	if _, ok := item["enable_local_automatic_backup"]; ok {
		if item["enable_local_automatic_backup"].(string) == "0" {
			if p2.Value == "1" {
				//关闭本地备份
				err := global.SCHEDULER.RemoveByTag(constant.LocalBackup)
				if err != nil {
					log.Errorf("关闭本地备份任务失败:%v", err)
				}
				err = propertyRepository.Update(&model.Property{
					Name:  "enable_local_automatic_backup",
					Value: "0",
				})
				if err != nil {
					log.Errorf("修改本地审计备份配置失败:%v", err)
					return FailWithDataOperate(c, 500, "修改失败", "", err)
				}
				err = propertyRepository.DeleteByNames(constant.LocalAuditBackup)
				if err != nil {
					log.Errorf("删除本地审计备份配置失败:%v", err)
					return FailWithDataOperate(c, 500, "修改失败", "", err)
				}
				op = op + "状态: 开启->关闭]"
			} else {
				op = op + "状态: 关闭->关闭]"
			}
		} else {
			if p2.Value == "0" {
				err := newJobService.NewBackupJob(item["local_backup_interval"].(string), constant.LocalBackup)
				if err != nil {
					log.Errorf("开启本地备份失败:%v", err)
					return FailWithDataOperate(c, 500, "修改失败", "", err)
				}
				for _, v := range constant.LocalAuditBackup {
					if _, ok := item[v]; ok {
						err := propertyRepository.Create(&model.Property{
							Name:  v,
							Value: fmt.Sprintf("%v", item[v]),
						})
						if err != nil {
							log.Errorf("修改本地审计备份配置失败:%v", err)
							return FailWithDataOperate(c, 500, "修改失败", "", err)
						}
					}
				}
				err = propertyRepository.Update(&model.Property{
					Name:  "enable_local_automatic_backup",
					Value: "1",
				})
				if err != nil {
					log.Errorf("修改本地审计备份配置失败:%v", err)
					return FailWithDataOperate(c, 500, "修改失败", "", err)
				}
				op = op + "状态: 关闭->开启, 本地备份时间: " + item["local_backup_interval"].(string) + "]"
			} else {
				// 删除旧的定时任务
				err := global.SCHEDULER.RemoveByTag(constant.LocalBackup)
				if err != nil {
					log.Errorf("关闭本地备份任务失败:%v", err)
				}
				err = newJobService.NewBackupJob(item["local_backup_interval"].(string), constant.LocalBackup)
				if err != nil {
					log.Errorf("修改本地备份失败:%v", err)
					return FailWithDataOperate(c, 500, "修改失败", "", err)
				}
				for _, v := range constant.LocalAuditBackup {
					if _, ok := item[v]; ok {
						err := propertyRepository.Update(&model.Property{
							Name:  v,
							Value: fmt.Sprintf("%v", item[v]),
						})
						if err != nil {
							log.Errorf("修改本地审计备份配置失败:%v", err)
							return FailWithDataOperate(c, 500, "修改失败", "", err)
						}
					}
				}
				op = op + "状态: 开启->开启, 本地备份时间: " + item["local_backup_interval"].(string) + "]"
			}
		}
	} else {
		op = op + "状态: 未修改]"
	}
	return SuccessWithOperate(c, op, nil)
}

// end <=========

// 存储空间配置
// start ==========> routes.go 745, 746

func CapacityConfigGetEndpoint(c echo.Context) error {
	sysLogsConfig, err := propertyRepository.FindMapByNames(constant.StorageSpaceConfig)
	if err != nil {
		log.Errorf("获取审计备份配置失败:%v", err)
		return FailWithDataOperate(c, 500, "获取审计备份配置失败", "", err)
	}

	return Success(c, sysLogsConfig)
}

func CapacityConfigEditEndpoint(c echo.Context) error {
	var item map[string]string
	_, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithData(c, 401, "获取当前用户失败", "")
	}
	if err := c.Bind(&item); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	fmt.Println(item)
	p1, err := propertyRepository.FindByName("enable_storage_space")
	if err != nil {
		log.Errorf("获取存储空间配置失败:%v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	p2, err := propertyRepository.FindByName("enable_log_storage_limit")
	if err != nil {
		log.Errorf("获取存储时间配置失败:%v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	op := "系统配置-存储配置: 存储空间限制, 修改存储空间限制["
	// 存储空间限制
	{
		if _, ok := item["enable_storage_space"]; ok {
			if item["enable_storage_space"] == "0" {
				if p1.Value == "1" {
					err := propertyRepository.Update(&model.Property{
						Name:  "enable_storage_space",
						Value: "0",
					})
					if err != nil {
						log.Errorf("修改存储空间配置失败:%v", err)
						return FailWithDataOperate(c, 500, "修改失败", "", err)
					}
					err = global.SCHEDULER.RemoveByTag(constant.DiskDetection)
					if err != nil {
						log.Errorf("删除磁盘检测任务失败:%v", err)
					}
					err = propertyRepository.DeleteByNames(constant.StorageSpace)
					if err != nil {
						log.Errorf("修改存储空间配置失败:%v", err)
						return FailWithDataOperate(c, 500, "修改失败", "", err)
					}
					err = propertyRepository.Update(&model.Property{
						Name:  guacd.EnableRecording,
						Value: "true",
					})
					if err != nil {
						log.Errorf("修改录像配置失败:%v", err)
					}
					op = op + "状态: 开启->关闭] "
				} else {
					op = op + "状态: 关闭->关闭] "
				}
			} else {
				if item[constant.StorageSpace[2]] == "" {
					return FailWithData(c, 500, "日志条数限制不能为空", "")
				} else {
					ll := utils.StringToInt(item[constant.StorageSpace[2]])
					if ll < 0 || ll > 100 {
						return FailWithData(c, 500, "日志条数限制有效值为0-100", "")
					}
				}
				if p1.Value == "0" {
					err := newJobService.AddStorageLimitJob()
					if err != nil {
						log.Errorf("添加磁盘检测任务失败:%v", err)
						return FailWithDataOperate(c, 500, "修改失败", "", err)
					}
					err = propertyRepository.Update(&model.Property{
						Name:  "enable_storage_space",
						Value: "1",
					})
					if err != nil {
						log.Errorf("修改存储空间配置失败:%v", err)
						return FailWithDataOperate(c, 500, "修改失败", "", err)
					}
					op = op + "状态: 关闭->开启] "
				} else {
					op = op + "状态: 开启->开启] "
				}
				err = propertyRepository.DeleteByNames(constant.StorageSpace)
				if err != nil {
					log.Errorf("修改存储空间配置失败:%v", err)
					return FailWithDataOperate(c, 500, "修改失败", "", err)
				}
				for i := range constant.StorageSpace {
					if v, ok := item[constant.StorageSpace[i]]; ok {
						err := propertyRepository.Create(&model.Property{
							Name:  constant.StorageSpace[i],
							Value: v,
						})
						if err != nil {
							log.Errorf("修改存储空间配置失败:%v", err)
							return FailWithDataOperate(c, 500, "修改失败", "", err)
						}
					}
				}
			}
		}
	}

	// 日志存储限制
	op = op + ",修改日志存储限制["
	{
		if _, ok := item["enable_log_storage_limit"]; ok {
			if item["enable_log_storage_limit"] == "0" {
				if p2.Value == "1" {
					err := propertyRepository.Update(&model.Property{
						Name:  "enable_log_storage_limit",
						Value: "0",
					})
					if err != nil {
						log.Error("修改存储空间配置失败")
						return FailWithDataOperate(c, 500, "修改失败", "", err)
					}
					err = global.SCHEDULER.RemoveByTag(constant.LogTimeDetection)
					if err != nil {
						log.Error("删除日志存储限制任务失败")
						return FailWithDataOperate(c, 500, "修改失败", "", err)
					}
					err = propertyRepository.DeleteByNames(constant.LogStorageTimeName)
					if err != nil {
						log.Error("修改存储空间配置失败")
						return FailWithDataOperate(c, 500, "修改失败", "", err)
					}
					op = op + "日志存储限制: 开启->关闭] "
				} else {
					op = op + "日志存储限制: 关闭->关闭] "
				}
			} else {
				if p2.Value == "0" {
					err := newJobService.DeleteTimeoutLog()
					if err != nil {
						log.Error("添加日志存储限制任务失败")
						return FailWithDataOperate(c, 500, "修改失败", "", err)
					}
					err = propertyRepository.Update(&model.Property{
						Name:  "enable_log_storage_limit",
						Value: "1",
					})
					if err != nil {
						log.Error("修改存储空间配置失败")
						return FailWithDataOperate(c, 500, "修改失败", "", err)
					}
					op = op + "日志存储限制: 关闭->开启] "
				} else {
					op = op + "日志存储限制: 开启->开启] "
				}
				err = propertyRepository.DeleteByNames(constant.LogStorageTimeName)
				if err != nil {
					log.Error("修改存储空间配置失败")
					return FailWithDataOperate(c, 500, "修改失败", "", err)
				}
				for i := range constant.LogStorageTimeName {
					if v, ok := item[constant.LogStorageTimeName[i]]; ok {
						err := propertyRepository.Create(&model.Property{
							Name:  constant.LogStorageTimeName[i],
							Value: v,
						})
						if err != nil {
							log.Errorf("修改存储空间配置失败:%v", err)
							return FailWithDataOperate(c, 500, "修改失败", "", err)
						}
					}
				}
			}
		}
	}

	return SuccessWithOperate(c, op, "修改成功")
}

// end <=========

// 默认磁盘空间配置 <=========

func DefaultDiskConfigGetEndpoint(c echo.Context) error {
	properties := propertyRepository.FindAuMap("storage_size")
	//if err != nil {
	//	log.Error("查询默认磁盘空间配置失败")
	//	return FailWithDataOperate(c, 500, "查询失败", "", err)
	//}
	return Success(c, properties)
}

func DefaultDiskConfigEditEndpoint(c echo.Context) error {
	item := make(map[string]interface{})
	if err := c.Bind(&item); err != nil {
		log.Error("修改默认磁盘空间配置失败")
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	p, err := propertyRepository.FindByName("storage_size")
	if err != nil {
		log.Error("查询默认磁盘空间配置失败")
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if _, ok := item["storage_size"]; ok {
		se := item["storage_size"].(int)
		if se < 1 || se > 10000 {
			return Fail(c, 400, "磁盘空间大小范围为1-10000")
		}
		err := propertyRepository.Update(&model.Property{
			Name:  "storage_size",
			Value: strconv.Itoa(se),
		})
		if err != nil {
			log.Error("修改默认磁盘空间配置失败")
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		}
	}
	return SuccessWithOperate(c, "系统配置-存储配置: 修改磁盘大小配置["+p.Value+"->"+strconv.Itoa(item["storage_size"].(int))+"]", "修改成功")
}
