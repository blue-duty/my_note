package api

import (
	"context"
	"strings"
	"time"
	"tkbastion/pkg/constant"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/model"
	"tkbastion/server/service"
	"tkbastion/server/utils"

	"gorm.io/gorm"

	"github.com/labstack/echo/v4"
)

func HostOperatePagingEndpoint(c echo.Context) error {
	account, _ := GetCurrentAccountNew(c)
	// 当前用户所属的所有用户组
	userGroupIds, err := userGroupMemberRepository.FindUserGroupIdsByUserId(account.ID)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	sql := "SELECT id, name, department_id, department_name, state, button_state, description, relate_user, relate_user_group,download,upload,watermark, relate_asset, relate_asset_group, relate_app, dep_level, priority, auth_expiration_date, strategy_begin_time, strategy_end_time, strategy_time_flag, ip_limit_type, ip_limit_list FROM operate_auth WHERE ("
	whereCondition := " relate_user LIKE '%" + account.ID + "%'"
	for i := range userGroupIds {
		whereCondition += " OR relate_user_group LIKE '%" + userGroupIds[i] + "%'"
	}
	whereCondition += " ) AND state != 'overdue' AND button_state != 'off'"
	orderBy := " ORDER BY dep_level ASC, department_name, priority ASC"
	sql += whereCondition
	sql += orderBy

	var operateAuthArr []model.OperateAuth
	err = operateAuthRepository.DB.Raw(sql).Find(&operateAuthArr).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	var hostAuthOperateArr []model.HostOperate
	var notAuthAssetIdArr []string
	for i := range operateAuthArr {
		ok, err := service.ExpDateService.JudgeExpOperateAuth(&operateAuthArr[i])
		if nil != err {
			log.Errorf("JudgeExpOperateAuth Error: %v", err)
			return SuccessWithOperate(c, "", nil)
		}
		if !ok {
			// 当前时间未在此策略的授权时间段内
			err = strategyRelateAssetIdJoinNotAuthArr(operateAuthArr[i], &notAuthAssetIdArr)
			if nil != err {
				log.Errorf("strategyRelateAssetIdJoinNotAuthArr Error: %v", err)
				return SuccessWithOperate(c, "", nil)
			}
		} else {
			// 当前时间在此策略的授权时间段内
			if "blackList" == operateAuthArr[i].IpLimitType {
				// 黑名单
				if isIpsContainIp(c.RealIP(), operateAuthArr[i].IpLimitList) {
					// 当前登录IP属于黑名单列表
					err = strategyRelateAssetIdJoinNotAuthArr(operateAuthArr[i], &notAuthAssetIdArr)
					if nil != err {
						log.Errorf("strategyRelateAssetIdJoinArr Error: %v", err)
						return SuccessWithOperate(c, "", nil)
					}
				} else {
					// 当前登录IP不属于黑名单列表
					// 策略关联的所有不在notAuthAssetIdArr内的账号加入hostAuthOperateArr
					err = strategyRelAsIdNotBelongArrJoinOthArr(operateAuthArr[i], notAuthAssetIdArr, &hostAuthOperateArr)
					if nil != err {
						log.Errorf("strategyRelAsIdNotBelongArrJoinOthArr Error: %v", err)
						return SuccessWithOperate(c, "", nil)
					}
				}
			} else {
				// 白名单
				if isIpsContainIp(c.RealIP(), operateAuthArr[i].IpLimitList) {
					// 当前登录IP属于白名单列表
					// 策略关联的所有不在notAuthAssetIdArr内的账号加入hostAuthOperateArr
					err = strategyRelAsIdNotBelongArrJoinOthArr(operateAuthArr[i], notAuthAssetIdArr, &hostAuthOperateArr)
					if nil != err {
						log.Errorf("strategyRelAsIdNotBelongArrJoinOthArr Error: %v", err)
						return SuccessWithOperate(c, "", nil)
					}
				} else {
					// 当前登录IP不属于白名单列表
					err = strategyRelateAssetIdJoinNotAuthArr(operateAuthArr[i], &notAuthAssetIdArr)
					if nil != err {
						log.Errorf("strategyRelateAssetIdJoinArr Error: %v", err)
						return SuccessWithOperate(c, "", nil)
					}
				}
			}
		}
	}

	// 注意, 此处需在去重前添加, 避免出现重复主机(如果申请工单的逻辑是可以申请全部主机, 那在去重后添加就有可能出现重复的主机账号)
	// 1. 获取该用户所有的有效工单（审批通过/且在有效期之内）
	workOrderList, err := workOrderNewRepository.FindValidOrderByUserId(account.ID)
	if nil != err {
		log.Errorf("FindValidOrderByUserId Error: %v", err)
	}
	boolToStr := func(b bool) string {
		if b {
			return "on"
		}
		return "off"
	}
	for _, workOrder := range workOrderList {
		if !*workOrder.IsPermanent && (time.Now().Before(workOrder.BeginTime.Time) || time.Now().After(workOrder.EndTime.Time)) {
			continue
		}
		// 2. 获取工单关联的所有设备账号
		workOrderAsset, err := workOrderNewRepository.FindByWorkOrderId(workOrder.ID)
		if nil != err {
			log.Errorf("FindByWorkOrderId Error: %v", err)
			continue
		}
		for _, v := range workOrderAsset {
			asset, err := newAssetRepository.GetPassportById(context.TODO(), v.AssetId)
			if nil != err {
				log.Errorf("FindAssetByWorkOrderId Error: %v", err)
				continue
			}
			// 3. 将工单关联的设备账号加入hostAuthOperateArr
			hostAuthOperateArr = append(hostAuthOperateArr, model.HostOperate{
				Id:         asset.ID,
				AssetName:  asset.AssetName,
				Ip:         asset.Ip,
				Name:       asset.Passport,
				Status:     asset.Status,
				Protocol:   asset.Protocol,
				Collection: "false",
				Download:   boolToStr(*workOrder.IsDownload),
				Upload:     boolToStr(*workOrder.IsUpload),
				Watermark:  boolToStr(*workOrder.IsWatermark),
				LoginType:  asset.LoginType,
			})
		}
	}
	// 且如果工单添加的主机是被上述策略排除掉的主机也属于正常情况, 因为工单功能本来就是让用户去使用本来正常无权限使用的主机
	// 这其中就包括被策略禁止的主机
	hostAuthOperateArr = HostOperateArrRemoveDuplicates(hostAuthOperateArr)
	// 至此, 我们通过策略有效期、授权时段、IP限制、工单申请过滤出了一份去重后的可运维设备账号集合

	userCollecteArr, err := userCollecteRepository.FindByUserId(account.ID)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	// 根据用户收藏表去设置hostAuthOperateArr切片中设备账号的收藏字段
	// 同时反过来根据hostAuthOperateArr切片中设备账号 去清除用户收藏表中该用户以前收藏的主机，但现在该用户已没有权限运维的主机收藏记录或现在已被删除的设备账号或现在已被删除的设备账号
	isExist := false
	for i := range userCollecteArr {
		isExist = false
		for j := range hostAuthOperateArr {
			if hostAuthOperateArr[j].Id == userCollecteArr[i].AssetAccountId {
				hostAuthOperateArr[j].Collection = "true"
				isExist = true
				break
			}
		}
		if !isExist {
			err = userCollecteRepository.DeleteById(userCollecteArr[i].Id)
			if nil != err {
				log.Errorf("DB Error: %v", err)
				return SuccessWithOperate(c, "", nil)
			}
		}
	}

	// 搜索条件逻辑处理, 应在更新用户收藏表后进行
	var searchWhereCondition string
	auto := c.QueryParam("auto")
	assetName := c.QueryParam("asset_name")
	assetAddress := c.QueryParam("asset_address")
	protocol := c.QueryParam("protocol")
	assetAccount := c.QueryParam("asset_account")
	if "" != auto {
		protocol = strings.ToLower(auto)
		searchWhereCondition = ") AND (asset_name LIKE '%" + auto + "%' OR ip LIKE '%" + auto + "%' OR protocol LIKE '%" + protocol + "%' OR passport LIKE '%" + auto + "%')"
	} else if "" != assetName {
		searchWhereCondition = ") AND asset_name LIKE '%" + assetName + "%'"
	} else if "" != assetAddress {
		searchWhereCondition = ") AND ip LIKE '%" + assetAddress + "%'"
	} else if "" != protocol {
		protocol = strings.ToLower(protocol)
		searchWhereCondition = ") AND protocol LIKE '%" + protocol + "%'"
	} else if "" != assetAccount {
		searchWhereCondition = ") AND passport LIKE '%" + assetAccount + "%'"
	}
	// 如果是搜索功能, 则在此if语句返回
	if 0 != len(searchWhereCondition) {
		var assetAccountArrStr string
		for i := range hostAuthOperateArr {
			assetAccountArrStr = assetAccountArrStr + "'" + hostAuthOperateArr[i].Id + "', "
		}
		assetAccountArrStr = assetAccountArrStr[:len(assetAccountArrStr)-2]

		searchSql := "SELECT * FROM pass_ports WHERE id IN (" + assetAccountArrStr
		searchSql += searchWhereCondition
		var assetAccountArr []model.PassPort
		err = global.DBConn.Raw(searchSql).Find(&assetAccountArr).Error
		if nil != err {
			if gorm.ErrRecordNotFound == err {
				return SuccessWithOperate(c, "", nil)
			}
			log.Errorf("DB Error: %v", err)
			return SuccessWithOperate(c, "", nil)
		}

		// 此处不能直接查账号表然后返回, 因为账号表中没有权限控制字段, 因此需排除原hostAuthOperateArr中除assetAccountArr以外的账号
		hostAuthOperateArr = searchWithConditionAccountRemoveDuplicates(hostAuthOperateArr, assetAccountArr)
		return SuccessWithOperate(c, "", hostAuthOperateArr)
	}

	return SuccessWithOperate(c, "", hostAuthOperateArr)
}

func searchWithConditionAccountRemoveDuplicates(hostAuthOperateArr []model.HostOperate, assetAccountArr []model.PassPort) (removeDuplicatesHostOperateArr []model.HostOperate) {
	for i := range assetAccountArr {
		for j := range hostAuthOperateArr {
			if assetAccountArr[i].ID == hostAuthOperateArr[j].Id {
				removeDuplicatesHostOperateArr = append(removeDuplicatesHostOperateArr, hostAuthOperateArr[j])
				break
			}
		}
	}
	return
}

func searchWithConditionAccountRemoveDuplicatesStr(hostAuthOperateArr []model.HostOperate, assetAccountArr []string) (removeDuplicatesHostOperateArr []model.HostOperate) {
	for i := range assetAccountArr {
		for j := range hostAuthOperateArr {
			if assetAccountArr[i] == hostAuthOperateArr[j].Id {
				removeDuplicatesHostOperateArr = append(removeDuplicatesHostOperateArr, hostAuthOperateArr[j])
				break
			}
		}
	}
	return
}

func HostOperateGraphicalPagingEndpoint(c echo.Context) error {
	account, _ := GetCurrentAccountNew(c)
	// 当前用户所属的所有用户组
	userGroupIds, err := userGroupMemberRepository.FindUserGroupIdsByUserId(account.ID)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	sql := "SELECT id, name, department_id, department_name, state, button_state, description, relate_user, relate_user_group, relate_asset, relate_asset_group, relate_app, dep_level,download,upload,watermark, priority, auth_expiration_date, strategy_begin_time, strategy_end_time, strategy_time_flag, ip_limit_type, ip_limit_list FROM operate_auth WHERE ("
	whereCondition := " relate_user LIKE '%" + account.ID + "%'"
	for i := range userGroupIds {
		whereCondition += " OR relate_user_group LIKE '%" + userGroupIds[i] + "%'"
	}
	whereCondition += " ) AND state != 'overdue' AND button_state != 'off'"
	orderBy := " ORDER BY dep_level ASC, department_name, priority ASC"
	sql += whereCondition
	sql += orderBy

	var operateAuthArr []model.OperateAuth
	err = operateAuthRepository.DB.Raw(sql).Find(&operateAuthArr).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	var hostAuthOperateArr []model.HostOperate
	var notAuthAssetIdArr []string
	for i := range operateAuthArr {
		ok, err := service.ExpDateService.JudgeExpOperateAuth(&operateAuthArr[i])
		if nil != err {
			log.Errorf("JudgeExpOperateAuth Error: %v", err)
			return SuccessWithOperate(c, "", nil)
		}
		if !ok {
			// 当前时间未在此策略的授权时间段内
			err = strategyRelateAssetIdJoinNotAuthArr(operateAuthArr[i], &notAuthAssetIdArr)
			if nil != err {
				log.Errorf("strategyRelateAssetIdJoinNotAuthArr Error: %v", err)
				return SuccessWithOperate(c, "", nil)
			}
		} else {
			// 当前时间在此策略的授权时间段内
			if "blackList" == operateAuthArr[i].IpLimitType {
				// 黑名单
				if isIpsContainIp(c.RealIP(), operateAuthArr[i].IpLimitList) {
					// 当前登录IP属于黑名单列表
					err = strategyRelateAssetIdJoinNotAuthArr(operateAuthArr[i], &notAuthAssetIdArr)
					if nil != err {
						log.Errorf("strategyRelateAssetIdJoinArr Error: %v", err)
						return SuccessWithOperate(c, "", nil)
					}
				} else {
					// 当前登录IP不属于黑名单列表
					// 策略关联的所有不在notAuthAssetIdArr内的账号加入hostAuthOperateArr
					err = strategyRelAsIdNotBelongArrJoinOthArr(operateAuthArr[i], notAuthAssetIdArr, &hostAuthOperateArr)
					if nil != err {
						log.Errorf("strategyRelAsIdNotBelongArrJoinOthArr Error: %v", err)
						return SuccessWithOperate(c, "", nil)
					}
				}
			} else {
				// 白名单
				if isIpsContainIp(c.RealIP(), operateAuthArr[i].IpLimitList) {
					// 当前登录IP属于白名单列表
					// 策略关联的所有不在notAuthAssetIdArr内的账号加入hostAuthOperateArr
					err = strategyRelAsIdNotBelongArrJoinOthArr(operateAuthArr[i], notAuthAssetIdArr, &hostAuthOperateArr)
					if nil != err {
						log.Errorf("strategyRelAsIdNotBelongArrJoinOthArr Error: %v", err)
						return SuccessWithOperate(c, "", nil)
					}
				} else {
					// 当前登录IP不属于白名单列表
					err = strategyRelateAssetIdJoinNotAuthArr(operateAuthArr[i], &notAuthAssetIdArr)
					if nil != err {
						log.Errorf("strategyRelateAssetIdJoinArr Error: %v", err)
						return SuccessWithOperate(c, "", nil)
					}
				}
			}
		}
	}

	// 添加该用户工单申请通过且未过有效期的设备账号 TODO
	// 1. 获取该用户所有的有效工单（审批通过/且在有效期之内）
	workOrderList, err := workOrderNewRepository.FindValidOrderByUserId(account.ID)
	if nil != err {
		log.Errorf("FindValidOrderByUserId Error: %v", err)
	}
	boolToStr := func(b bool) string {
		if b {
			return "on"
		}
		return "off"
	}
	for _, workOrder := range workOrderList {
		if !*workOrder.IsPermanent && (time.Now().Before(workOrder.BeginTime.Time) || time.Now().After(workOrder.EndTime.Time)) {
			continue
		}
		// 2. 获取工单关联的所有设备账号
		workOrderAsset, err := workOrderNewRepository.FindByWorkOrderId(workOrder.ID)
		if nil != err {
			log.Errorf("FindByWorkOrderId Error: %v", err)
			continue
		}
		for _, v := range workOrderAsset {
			asset, err := newAssetRepository.GetPassportById(context.TODO(), v.AssetId)
			if nil != err {
				log.Errorf("FindAssetByWorkOrderId Error: %v", err)
				continue
			}
			// 3. 将图形设备加入hostAuthOperateArr
			if asset.Protocol == constant.VNC || asset.Protocol == constant.RDP {
				hostAuthOperateArr = append(hostAuthOperateArr, model.HostOperate{
					Id:         asset.ID,
					AssetName:  asset.AssetName,
					Ip:         asset.Ip,
					Name:       asset.Passport,
					Status:     asset.Status,
					Protocol:   asset.Protocol,
					Collection: "false",
					Download:   boolToStr(*workOrder.IsDownload),
					Upload:     boolToStr(*workOrder.IsUpload),
					Watermark:  boolToStr(*workOrder.IsWatermark),
				})
			}
		}
	}
	// 注意, 此处需在去重前添加, 避免出现重复主机(如果申请工单的逻辑是可以申请全部主机, 那在去重后添加就有可能出现重复的主机账号)
	// 且如果工单添加的主机是被上述策略排除掉的主机也属于正常情况, 因为工单功能本来就是让用户去使用本来正常无权限使用的主机
	// 这其中就包括被策略禁止的主机
	hostAuthOperateArr = HostOperateArrRemoveDuplicates(hostAuthOperateArr)
	// 至此, 我们通过策略有效期、授权时段、IP限制、工单申请过滤出了一份去重后的可运维设备账号集合

	userCollecteArr, err := userCollecteRepository.FindByUserId(account.ID)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	// 根据用户收藏表去设置hostAuthOperateArr切片中设备账号的收藏字段
	// 同时反过来根据hostAuthOperateArr切片中设备账号 去清除用户收藏表中该用户以前收藏的主机，但现在该用户已没有权限运维的主机收藏记录或现在已被删除的设备账号
	isExist := false
	for i := range userCollecteArr {
		isExist = false
		for j := range hostAuthOperateArr {
			if hostAuthOperateArr[j].Id == userCollecteArr[i].AssetAccountId {
				hostAuthOperateArr[j].Collection = "true"
				isExist = true
				break
			}
		}
		if !isExist {
			err = userCollecteRepository.DeleteById(userCollecteArr[i].Id)
			if nil != err {
				log.Errorf("DB Error: %v", err)
				return SuccessWithOperate(c, "", nil)
			}
		}
	}

	// 图形协议及搜索条件逻辑处理, 应在更新用户收藏表后进行
	var searchWhereCondition string
	auto := c.QueryParam("auto")
	assetName := c.QueryParam("asset_name")
	assetAddress := c.QueryParam("asset_address")
	protocol := c.QueryParam("protocol")
	assetAccount := c.QueryParam("asset_account")
	if "" != auto {
		protocol = strings.ToLower(auto)
		searchWhereCondition = ") AND (protocol = 'rdp' OR protocol = 'vnc') AND (asset_name LIKE '%" + auto + "%' OR ip LIKE '%" + auto + "%' OR protocol LIKE '%" + protocol + "%' OR passport LIKE '%" + auto + "%')"
	} else if "" != assetName {
		searchWhereCondition = ") AND (protocol = ''rdp'' OR protocol = 'vnc') AND asset_name LIKE '%" + assetName + "%'"
	} else if "" != assetAddress {
		searchWhereCondition = ") AND (protocol = 'rdp' OR protocol = 'vnc') AND ip LIKE '%" + assetAddress + "%'"
	} else if "" != protocol {
		protocol = strings.ToLower(protocol)
		searchWhereCondition = ") AND (protocol = 'rdp' OR protocol = 'vnc') AND protocol LIKE '%" + protocol + "%'"
	} else if "" != assetAccount {
		searchWhereCondition = ") AND (protocol = 'rdp' OR protocol = 'vnc') AND passport LIKE '%" + assetAccount + "%'"
	} else {
		searchWhereCondition = ") AND (protocol = 'rdp' OR protocol = 'vnc')"
	}

	var assetAccountArrStr string
	for i := range hostAuthOperateArr {
		assetAccountArrStr = assetAccountArrStr + "'" + hostAuthOperateArr[i].Id + "', "
	}
	// 无可运维设备账号
	if "" == assetAccountArrStr {
		return SuccessWithOperate(c, "", hostAuthOperateArr)
	}
	assetAccountArrStr = assetAccountArrStr[:len(assetAccountArrStr)-2]

	searchSql := "SELECT * FROM pass_ports WHERE id IN (" + assetAccountArrStr
	searchSql += searchWhereCondition
	var assetAccountArr []model.PassPort
	err = global.DBConn.Raw(searchSql).Find(&assetAccountArr).Error
	if nil != err {
		if gorm.ErrRecordNotFound == err {
			return SuccessWithOperate(c, "", nil)
		}
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	// 此处不能直接查账号表然后返回, 因为账号表中没有权限控制字段, 因此需排除原hostAuthOperateArr中除assetAccountArr以外的账号
	hostAuthOperateArr = searchWithConditionAccountRemoveDuplicates(hostAuthOperateArr, assetAccountArr)
	return SuccessWithOperate(c, "", hostAuthOperateArr)
}

func HostOperateCharacterPagingEndpoint(c echo.Context) error {
	account, _ := GetCurrentAccountNew(c)
	// 当前用户所属的所有用户组
	userGroupIds, err := userGroupMemberRepository.FindUserGroupIdsByUserId(account.ID)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	sql := "SELECT id, name, department_id, department_name, state, button_state, description, relate_user, relate_user_group,download,upload,watermark, relate_asset, relate_asset_group, relate_app, dep_level, priority, auth_expiration_date, strategy_begin_time, strategy_end_time, strategy_time_flag, ip_limit_type, ip_limit_list FROM operate_auth WHERE ("
	whereCondition := " relate_user LIKE '%" + account.ID + "%'"
	for i := range userGroupIds {
		whereCondition += " OR relate_user_group LIKE '%" + userGroupIds[i] + "%'"
	}
	whereCondition += " ) AND state != 'overdue' AND button_state != 'off'"
	orderBy := " ORDER BY dep_level ASC, department_name, priority ASC"
	sql += whereCondition
	sql += orderBy

	var operateAuthArr []model.OperateAuth
	err = operateAuthRepository.DB.Raw(sql).Find(&operateAuthArr).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	var hostAuthOperateArr []model.HostOperate
	var notAuthAssetIdArr []string
	for i := range operateAuthArr {
		ok, err := service.ExpDateService.JudgeExpOperateAuth(&operateAuthArr[i])
		if nil != err {
			log.Errorf("JudgeExpOperateAuth Error: %v", err)
			return SuccessWithOperate(c, "", nil)
		}
		if !ok {
			// 当前时间未在此策略的授权时间段内
			err = strategyRelateAssetIdJoinNotAuthArr(operateAuthArr[i], &notAuthAssetIdArr)
			if nil != err {
				log.Errorf("strategyRelateAssetIdJoinNotAuthArr Error: %v", err)
				return SuccessWithOperate(c, "", nil)
			}
		} else {
			// 当前时间在此策略的授权时间段内
			if "blackList" == operateAuthArr[i].IpLimitType {
				// 黑名单
				if isIpsContainIp(c.RealIP(), operateAuthArr[i].IpLimitList) {
					// 当前登录IP属于黑名单列表
					err = strategyRelateAssetIdJoinNotAuthArr(operateAuthArr[i], &notAuthAssetIdArr)
					if nil != err {
						log.Errorf("strategyRelateAssetIdJoinArr Error: %v", err)
						return SuccessWithOperate(c, "", nil)
					}
				} else {
					// 当前登录IP不属于黑名单列表
					// 策略关联的所有不在notAuthAssetIdArr内的账号加入hostAuthOperateArr
					err = strategyRelAsIdNotBelongArrJoinOthArr(operateAuthArr[i], notAuthAssetIdArr, &hostAuthOperateArr)
					if nil != err {
						log.Errorf("strategyRelAsIdNotBelongArrJoinOthArr Error: %v", err)
						return SuccessWithOperate(c, "", nil)
					}
				}
			} else {
				// 白名单
				if isIpsContainIp(c.RealIP(), operateAuthArr[i].IpLimitList) {
					// 当前登录IP属于白名单列表
					// 策略关联的所有不在notAuthAssetIdArr内的账号加入hostAuthOperateArr
					err = strategyRelAsIdNotBelongArrJoinOthArr(operateAuthArr[i], notAuthAssetIdArr, &hostAuthOperateArr)
					if nil != err {
						log.Errorf("strategyRelAsIdNotBelongArrJoinOthArr Error: %v", err)
						return SuccessWithOperate(c, "", nil)
					}
				} else {
					// 当前登录IP不属于白名单列表
					err = strategyRelateAssetIdJoinNotAuthArr(operateAuthArr[i], &notAuthAssetIdArr)
					if nil != err {
						log.Errorf("strategyRelateAssetIdJoinArr Error: %v", err)
						return SuccessWithOperate(c, "", nil)
					}
				}
			}
		}
	}

	// 添加该用户工单申请通过且未过有效期的设备账号 TODO
	// 1. 获取该用户所有的有效工单（审批通过/且在有效期之内）
	workOrderList, err := workOrderNewRepository.FindValidOrderByUserId(account.ID)
	if nil != err {
		log.Errorf("FindValidOrderByUserId Error: %v", err)
	}
	boolToStr := func(b bool) string {
		if b {
			return "on"
		}
		return "off"
	}
	for _, workOrder := range workOrderList {
		if !*workOrder.IsPermanent && (time.Now().Before(workOrder.BeginTime.Time) || time.Now().After(workOrder.EndTime.Time)) {
			continue
		}
		// 2. 获取工单关联的所有设备账号
		workOrderAsset, err := workOrderNewRepository.FindByWorkOrderId(workOrder.ID)
		if nil != err {
			log.Errorf("FindByWorkOrderId Error: %v", err)
			continue
		}
		for _, v := range workOrderAsset {
			asset, err := newAssetRepository.GetPassportById(context.TODO(), v.AssetId)
			if nil != err {
				log.Errorf("FindAssetByWorkOrderId Error: %v", err)
				continue
			}
			// 3. 将字符设备加入hostAuthOperateArr
			if asset.Protocol == constant.SSH || asset.Protocol == constant.TELNET {
				hostAuthOperateArr = append(hostAuthOperateArr, model.HostOperate{
					Id:         asset.ID,
					AssetName:  asset.AssetName,
					Ip:         asset.Ip,
					Name:       asset.Passport,
					Status:     asset.Status,
					Protocol:   asset.Protocol,
					Collection: "false",
					Download:   boolToStr(*workOrder.IsDownload),
					Upload:     boolToStr(*workOrder.IsUpload),
					Watermark:  boolToStr(*workOrder.IsWatermark),
				})
			}
		}
	}
	// 注意, 此处需在去重前添加, 避免出现重复主机(如果申请工单的逻辑是可以申请全部主机, 那在去重后添加就有可能出现重复的主机账号)
	// 且如果工单添加的主机是被上述策略排除掉的主机也属于正常情况, 因为工单功能本来就是让用户去使用本来正常无权限使用的主机
	// 这其中就包括被策略禁止的主机
	hostAuthOperateArr = HostOperateArrRemoveDuplicates(hostAuthOperateArr)
	// 至此, 我们通过策略有效期、授权时段、IP限制、工单申请过滤出了一份去重后的可运维设备账号集合

	userCollecteArr, err := userCollecteRepository.FindByUserId(account.ID)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	// 根据用户收藏表去设置hostAuthOperateArr切片中设备账号的收藏字段
	// 同时反过来根据hostAuthOperateArr切片中设备账号 去清除用户收藏表中该用户以前收藏的主机，但现在该用户已没有权限运维的主机收藏记录或现在已被删除的设备账号
	isExist := false
	for i := range userCollecteArr {
		isExist = false
		for j := range hostAuthOperateArr {
			if hostAuthOperateArr[j].Id == userCollecteArr[i].AssetAccountId {
				hostAuthOperateArr[j].Collection = "true"
				isExist = true
				break
			}
		}
		if !isExist {
			err = userCollecteRepository.DeleteById(userCollecteArr[i].Id)
			if nil != err {
				log.Errorf("DB Error: %v", err)
				return SuccessWithOperate(c, "", nil)
			}
		}
	}

	// 字符协议及搜索条件逻辑处理, 应在更新用户收藏表后进行
	var searchWhereCondition string
	auto := c.QueryParam("auto")
	assetName := c.QueryParam("asset_name")
	assetAddress := c.QueryParam("asset_address")
	protocol := c.QueryParam("protocol")
	assetAccount := c.QueryParam("asset_account")
	if "" != auto {
		protocol = strings.ToLower(auto)
		searchWhereCondition = ") AND (protocol = 'ssh' OR protocol = 'telnet') AND (asset_name LIKE '%" + auto + "%' OR ip LIKE '%" + auto + "%' OR protocol LIKE '%" + protocol + "%' OR passport LIKE '%" + auto + "%')"
	} else if "" != assetName {
		searchWhereCondition = ") AND (protocol = 'ssh' OR protocol = 'telnet') AND asset_name LIKE '%" + assetName + "%'"
	} else if "" != assetAddress {
		searchWhereCondition = ") AND (protocol = 'ssh' OR protocol = 'telnet') AND ip LIKE '%" + assetAddress + "%'"
	} else if "" != protocol {
		protocol = strings.ToLower(protocol)
		searchWhereCondition = ") AND (protocol = 'ssh' OR protocol = 'telnet') AND protocol LIKE '%" + protocol + "%'"
	} else if "" != assetAccount {
		searchWhereCondition = ") AND (protocol = 'ssh' OR protocol = 'telnet') AND passport LIKE '%" + assetAccount + "%'"
	} else {
		searchWhereCondition = ") AND (protocol = 'ssh' OR protocol = 'telnet')"
	}

	var assetAccountArrStr string
	for i := range hostAuthOperateArr {
		assetAccountArrStr = assetAccountArrStr + "'" + hostAuthOperateArr[i].Id + "', "
	}
	// 无可运维设备账号
	if "" == assetAccountArrStr {
		return SuccessWithOperate(c, "", hostAuthOperateArr)
	}
	assetAccountArrStr = assetAccountArrStr[:len(assetAccountArrStr)-2]

	searchSql := "SELECT * FROM pass_ports WHERE id IN (" + assetAccountArrStr
	searchSql += searchWhereCondition
	var assetAccountArr []model.PassPort
	err = global.DBConn.Raw(searchSql).Find(&assetAccountArr).Error
	if nil != err {
		if gorm.ErrRecordNotFound == err {
			return SuccessWithOperate(c, "", nil)
		}
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	// 此处不能直接查账号表然后返回, 因为账号表中没有权限控制字段, 因此需排除原hostAuthOperateArr中除assetAccountArr以外的账号
	hostAuthOperateArr = searchWithConditionAccountRemoveDuplicates(hostAuthOperateArr, assetAccountArr)
	return SuccessWithOperate(c, "", hostAuthOperateArr)
}

// 这里不能直接从用户收藏表中查, 因为用户收藏表中不包含权限控制字段, 并不清楚这些设备账号受哪个策略的权限进行控制
func HostOperateCollectPagingEndpoint(c echo.Context) error {
	account, _ := GetCurrentAccountNew(c)
	// 当前用户所属的所有用户组
	userGroupIds, err := userGroupMemberRepository.FindUserGroupIdsByUserId(account.ID)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	sql := "SELECT id, name, department_id, department_name, state, button_state, description, relate_user, relate_user_group,download,upload,watermark, relate_asset, relate_asset_group, relate_app, dep_level, priority, auth_expiration_date, strategy_begin_time, strategy_end_time, strategy_time_flag, ip_limit_type, ip_limit_list FROM operate_auth WHERE ("
	whereCondition := " relate_user LIKE '%" + account.ID + "%'"
	for i := range userGroupIds {
		whereCondition += " OR relate_user_group LIKE '%" + userGroupIds[i] + "%'"
	}
	whereCondition += " ) AND state != 'overdue' AND button_state != 'off'"
	orderBy := " ORDER BY dep_level ASC, department_name, priority ASC"
	sql += whereCondition
	sql += orderBy

	var operateAuthArr []model.OperateAuth
	err = operateAuthRepository.DB.Raw(sql).Find(&operateAuthArr).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	var hostAuthOperateArr []model.HostOperate
	var notAuthAssetIdArr []string
	for i := range operateAuthArr {
		ok, err := service.ExpDateService.JudgeExpOperateAuth(&operateAuthArr[i])
		if nil != err {
			log.Errorf("JudgeExpOperateAuth Error: %v", err)
			return SuccessWithOperate(c, "", nil)
		}
		if !ok {
			// 当前时间未在此策略的授权时间段内
			err = strategyRelateAssetIdJoinNotAuthArr(operateAuthArr[i], &notAuthAssetIdArr)
			if nil != err {
				log.Errorf("strategyRelateAssetIdJoinNotAuthArr Error: %v", err)
				return SuccessWithOperate(c, "", nil)
			}
		} else {
			// 当前时间在此策略的授权时间段内
			if "blackList" == operateAuthArr[i].IpLimitType {
				// 黑名单
				if isIpsContainIp(c.RealIP(), operateAuthArr[i].IpLimitList) {
					// 当前登录IP属于黑名单列表
					err = strategyRelateAssetIdJoinNotAuthArr(operateAuthArr[i], &notAuthAssetIdArr)
					if nil != err {
						log.Errorf("strategyRelateAssetIdJoinArr Error: %v", err)
						return SuccessWithOperate(c, "", nil)
					}
				} else {
					// 当前登录IP不属于黑名单列表
					// 策略关联的所有不在notAuthAssetIdArr内的账号加入hostAuthOperateArr
					err = strategyRelAsIdNotBelongArrJoinOthArr(operateAuthArr[i], notAuthAssetIdArr, &hostAuthOperateArr)
					if nil != err {
						log.Errorf("strategyRelAsIdNotBelongArrJoinOthArr Error: %v", err)
						return SuccessWithOperate(c, "", nil)
					}
				}
			} else {
				// 白名单
				if isIpsContainIp(c.RealIP(), operateAuthArr[i].IpLimitList) {
					// 当前登录IP属于白名单列表
					// 策略关联的所有不在notAuthAssetIdArr内的账号加入hostAuthOperateArr
					err = strategyRelAsIdNotBelongArrJoinOthArr(operateAuthArr[i], notAuthAssetIdArr, &hostAuthOperateArr)
					if nil != err {
						log.Errorf("strategyRelAsIdNotBelongArrJoinOthArr Error: %v", err)
						return SuccessWithOperate(c, "", nil)
					}
				} else {
					// 当前登录IP不属于白名单列表
					err = strategyRelateAssetIdJoinNotAuthArr(operateAuthArr[i], &notAuthAssetIdArr)
					if nil != err {
						log.Errorf("strategyRelateAssetIdJoinArr Error: %v", err)
						return SuccessWithOperate(c, "", nil)
					}
				}
			}
		}
	}

	// 添加该用户工单申请通过且未过有效期的设备账号 TODO
	// 1. 获取该用户所有的有效工单（审批通过/且在有效期之内）
	workOrderList, err := workOrderNewRepository.FindValidOrderByUserId(account.ID)
	if nil != err {
		log.Errorf("FindValidOrderByUserId Error: %v", err)
	}
	boolToStr := func(b bool) string {
		if b {
			return "on"
		}
		return "off"
	}
	for _, workOrder := range workOrderList {
		if !*workOrder.IsPermanent && (time.Now().Before(workOrder.BeginTime.Time) || time.Now().After(workOrder.EndTime.Time)) {
			continue
		}
		// 2. 获取工单关联的所有设备账号
		workOrderAsset, err := workOrderNewRepository.FindByWorkOrderId(workOrder.ID)
		if nil != err {
			log.Errorf("FindByWorkOrderId Error: %v", err)
			continue
		}
		for _, v := range workOrderAsset {
			asset, err := newAssetRepository.GetPassportById(context.TODO(), v.AssetId)
			if nil != err {
				log.Errorf("FindAssetByWorkOrderId Error: %v", err)
				continue
			}
			// 3. 设备加入hostAuthOperateArr
			hostAuthOperateArr = append(hostAuthOperateArr, model.HostOperate{
				Id:         asset.ID,
				AssetName:  asset.AssetName,
				Ip:         asset.Ip,
				Name:       asset.Passport,
				Status:     asset.Status,
				Protocol:   asset.Protocol,
				Collection: "false",
				Download:   boolToStr(*workOrder.IsDownload),
				Upload:     boolToStr(*workOrder.IsUpload),
				Watermark:  boolToStr(*workOrder.IsWatermark),
			})

		}
	}
	// 注意, 此处需在去重前添加, 避免出现重复主机(如果申请工单的逻辑是可以申请全部主机, 那在去重后添加就有可能出现重复的主机账号)
	// 且如果工单添加的主机是被上述策略排除掉的主机也属于正常情况, 因为工单功能本来就是让用户去使用本来正常无权限使用的主机
	// 这其中就包括被策略禁止的主机
	hostAuthOperateArr = HostOperateArrRemoveDuplicates(hostAuthOperateArr)
	// 至此, 我们通过策略有效期、授权时段、IP限制、工单申请过滤出了一份去重后的可运维设备账号集合

	userCollecteArr, err := userCollecteRepository.FindByUserId(account.ID)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	// 根据用户收藏表去设置hostAuthOperateArr切片中设备账号的收藏字段
	// 同时反过来根据hostAuthOperateArr切片中设备账号 去清除用户收藏表中该用户以前收藏的主机，但现在该用户已没有权限运维的主机收藏记录或现在已被删除的设备账号
	isExist := false
	for i := range userCollecteArr {
		isExist = false
		for j := range hostAuthOperateArr {
			if hostAuthOperateArr[j].Id == userCollecteArr[i].AssetAccountId {
				hostAuthOperateArr[j].Collection = "true"
				isExist = true
				break
			}
		}
		if !isExist {
			err = userCollecteRepository.DeleteById(userCollecteArr[i].Id)
			if nil != err {
				log.Errorf("DB Error: %v", err)
				return SuccessWithOperate(c, "", nil)
			}
		}
	}

	// 更新用户收藏表后, 重新获取用户收藏的设备账号列表
	newCollectAccountStrArr, err := userCollecteRepository.FindByUserIdStrArr(account.ID)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	// 查找收藏的设备及搜索条件逻辑处理, 应在更新用户收藏表后进行
	var searchWhereCondition string
	auto := c.QueryParam("auto")
	assetName := c.QueryParam("asset_name")
	assetAddress := c.QueryParam("asset_address")
	protocol := c.QueryParam("protocol")
	assetAccount := c.QueryParam("asset_account")
	if "" != auto {
		protocol = strings.ToLower(auto)
		searchWhereCondition = ") AND (asset_name LIKE '%" + auto + "%' OR ip LIKE '%" + auto + "%' OR protocol LIKE '%" + protocol + "%' OR passport LIKE '%" + auto + "%')"
	} else if "" != assetName {
		searchWhereCondition = ") AND asset_name LIKE '%" + assetName + "%'"
	} else if "" != assetAddress {
		searchWhereCondition = ") AND ip LIKE '%" + assetAddress + "%'"
	} else if "" != protocol {
		protocol = strings.ToLower(protocol)
		searchWhereCondition = ") AND protocol LIKE '%" + protocol + "%'"
	} else if "" != assetAccount {
		searchWhereCondition = ") AND passport LIKE '%" + assetAccount + "%'"
	}

	if 0 != len(searchWhereCondition) {
		var assetAccountArrStr string
		for i := range hostAuthOperateArr {
			assetAccountArrStr = assetAccountArrStr + "'" + hostAuthOperateArr[i].Id + "', "
		}
		assetAccountArrStr = assetAccountArrStr[:len(assetAccountArrStr)-2]

		searchSql := "SELECT * FROM pass_ports WHERE id IN (" + assetAccountArrStr
		searchSql += searchWhereCondition
		var assetAccountArr []model.PassPort
		err = global.DBConn.Raw(searchSql).Find(&assetAccountArr).Error
		if nil != err {
			if gorm.ErrRecordNotFound == err {
				return SuccessWithOperate(c, "", nil)
			}
			log.Errorf("DB Error: %v", err)
			return SuccessWithOperate(c, "", nil)
		}

		// 此处不能直接查账号表然后返回, 因为账号表中没有权限控制字段, 因此需排除原hostAuthOperateArr中除assetAccountArr以外的账号
		hostAuthOperateArr = searchWithConditionAccountRemoveDuplicates(hostAuthOperateArr, assetAccountArr)
		// 再次排除hostAuthOperateArr中未被收藏的账号
		hostAuthOperateArr = searchWithConditionAccountRemoveDuplicatesStr(hostAuthOperateArr, newCollectAccountStrArr)
		return SuccessWithOperate(c, "", hostAuthOperateArr)
	}
	// 排除hostAuthOperateArr中未被收藏的账号
	hostAuthOperateArr = searchWithConditionAccountRemoveDuplicatesStr(hostAuthOperateArr, newCollectAccountStrArr)
	return SuccessWithOperate(c, "", hostAuthOperateArr)
}

func HostOperateArrRemoveDuplicates(hostOperateArr []model.HostOperate) (removeDuplicatesHostOperateArr []model.HostOperate) {
	for i := range hostOperateArr {
		isDuplicates := false
		for j := range removeDuplicatesHostOperateArr {
			if hostOperateArr[i].Id == removeDuplicatesHostOperateArr[j].Id {
				isDuplicates = true
				break
			}
		}
		if !isDuplicates {
			removeDuplicatesHostOperateArr = append(removeDuplicatesHostOperateArr, hostOperateArr[i])
		}
	}
	return
}

func strategyRelateAssetIdJoinNotAuthArr(operateAuth model.OperateAuth, notAuthAssetIdArr *[]string) error {
	relateAssetArr := strings.Split(operateAuth.RelateAsset, ",")
	// 关联设备账号加入 不可运维设备账号数组中
	*notAuthAssetIdArr = append(*notAuthAssetIdArr, relateAssetArr...)

	// 关联设备组所包含的所有账号加入 不可运维设备账号数组中
	relateAssetGroupArr := strings.Split(operateAuth.RelateAssetGroup, ",")
	assetIdArr, err := newAssetGroupRepository.GetPassportIdsByAssetGroupIds(context.TODO(), relateAssetGroupArr)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return err
	}
	*notAuthAssetIdArr = append(*notAuthAssetIdArr, assetIdArr...)
	return nil
}

func HostOperateCollectEndpoint(c echo.Context) error {
	assetAccountId := c.Param("id")
	collect := c.QueryParam("collect")
	account, isSuccess := GetCurrentAccountNew(c)
	if !isSuccess {
		log.Error("GetCurrentAccountNew Error")
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}
	assetAccount, err := newAssetRepository.GetPassportById(context.TODO(), assetAccountId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}

	if "true" == collect {
		userCollect := model.UserCollecte{UserId: account.ID, AssetAccountId: assetAccountId}
		err = userCollecteRepository.Create(&userCollect)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "收藏失败", "", nil)
		}
		return SuccessWithOperate(c, "主机运维-收藏: 设备名称["+assetAccount.AssetName+"], 设备地址["+assetAccount.Ip+"] 协议["+assetAccount.Protocol+"] 设备账号["+assetAccount.Name+"]", nil)
	} else {
		err = userCollecteRepository.DeleteByUserIdAssetAccountId(account.ID, assetAccountId)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "取消收藏失败", "", nil)
		}
	}

	return SuccessWithOperate(c, "主机运维-取消收藏: 设备名称["+assetAccount.AssetName+"], 设备地址["+assetAccount.Ip+"] 协议["+assetAccount.Protocol+"] 设备账号["+assetAccount.Name+"]", nil)
}

func HostOperateConnectTestEndpoint(c echo.Context) error {
	assetAccountId := c.Param("id")
	assetAccount, err := newAssetRepository.GetPassportById(context.TODO(), assetAccountId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "连接测试失败", "", nil)
	}

	active := utils.Tcping(assetAccount.Ip, assetAccount.Port)
	if !active {
		return FailWithDataOperate(c, 200, "连接失败, 设备不在线", "主机运维-连接测试: 设备名称["+assetAccount.AssetName+"], 设备地址["+assetAccount.Ip+"], 协议["+assetAccount.Protocol+"], 设备账号["+assetAccount.Name+"], 测试结果[失败]", nil)
	}

	return SuccessWithOperate(c, "主机运维-连接测试: 设备名称["+assetAccount.AssetName+"], 设备地址["+assetAccount.Ip+"], 协议["+assetAccount.Protocol+"], 设备账号["+assetAccount.Name+"], 测试结果[成功]", nil)
}

func isIpsContainIp(ip, ipList string) bool {
	ipListArr := strings.Split(ipList, "\n")
	for i := range ipListArr {
		if strings.Contains(ipListArr[i], "-") {
			// 范围段
			split := strings.Split(ipListArr[i], "-")
			if len(split) < 2 {
				continue
			}
			start := split[0]
			end := split[1]
			intLoginIp := utils.IpToInt(ip)
			if intLoginIp < utils.IpToInt(start) || intLoginIp > utils.IpToInt(end) {
				continue
			}
		} else {
			// IP
			if ip != ipListArr[i] {
				continue
			}
		}

		return true
	}

	return false
}

func strategyRelAsIdNotBelongArrJoinOthArr(operateAuth model.OperateAuth, notAuthAssetIdArr []string, hostAuthOperateArr *[]model.HostOperate) error {
	relateAssetIdArr := strings.Split(operateAuth.RelateAsset, ",")
	relateAssetArr, err := newAssetRepository.GetPassportByIds(context.TODO(), relateAssetIdArr)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return err
	}

	// 关联的设备账号
	for i := range relateAssetArr {
		if !isAssetIdBelongArr(relateAssetArr[i].ID, notAuthAssetIdArr) {
			*hostAuthOperateArr = append(*hostAuthOperateArr, model.HostOperate{
				Id:         relateAssetArr[i].ID,
				AssetName:  relateAssetArr[i].AssetName,
				Ip:         relateAssetArr[i].Ip,
				Protocol:   relateAssetArr[i].Protocol,
				Name:       relateAssetArr[i].Passport,
				Status:     relateAssetArr[i].Status,
				Download:   operateAuth.Download,
				Upload:     operateAuth.Upload,
				Watermark:  operateAuth.Watermark,
				Collection: "false",
				LoginType:  relateAssetArr[i].LoginType,
			})
		}
	}

	// 关联的设备组
	relateAssetGroupArr := strings.Split(operateAuth.RelateAssetGroup, ",")
	assetIdArr, err := newAssetGroupRepository.GetPassportIdsByAssetGroupIds(context.TODO(), relateAssetGroupArr)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return err
	}

	for j := range assetIdArr {
		if !isAssetIdBelongArr(assetIdArr[j], notAuthAssetIdArr) {
			asset, err := newAssetRepository.GetPassportById(context.TODO(), assetIdArr[j])
			if nil != err {
				log.Errorf("DB Error: %v", err)
				return err
			}

			*hostAuthOperateArr = append(*hostAuthOperateArr, model.HostOperate{
				Id:         asset.ID,
				AssetName:  asset.AssetName,
				Ip:         asset.Ip,
				Protocol:   asset.Protocol,
				Name:       asset.Passport,
				Status:     asset.Status,
				Download:   operateAuth.Download,
				Upload:     operateAuth.Upload,
				Watermark:  operateAuth.Watermark,
				Collection: "false",
				LoginType:  asset.LoginType,
			})
		}
	}
	return nil
}

func isAssetIdBelongArr(assetId string, assetIds []string) bool {
	isBelong := false
	for i := range assetIds {
		if assetId == assetIds[i] {
			isBelong = true
			break
		}
	}
	return isBelong
}

func DelUserCollecteAssetAccount(userId string) {
	err := userCollecteRepository.DeleteAssetAccountByUser(userId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}
}

// GetRecentUsedPassport 获取用户最近使用的设备账号
func GetRecentUsedPassport(c echo.Context) error {
	auto := c.QueryParam("auto")
	assetName := c.QueryParam("assetName")
	ip := c.QueryParam("ip")
	protocol := c.QueryParam("protocol")
	passport := c.QueryParam("passport")
	account, _ := GetCurrentAccountNew(c)
	recentUsedPassport := make([]model.HostOperate, 0)
	passportArr, err := newSessionRepository.GetRecentUsedDevice(context.TODO(), account.Username, auto, assetName, ip, protocol, passport)
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}
	ip = c.RealIP()
	allPassport, err := GetUserPassportByUserId(ip, account.ID)
	resMap := make(map[string]string, 0)
	for i := range allPassport {
		resMap[allPassport[i].Id] = "1"
	}
	for i := range passportArr {
		if _, ok := resMap[passportArr[i].Id]; ok {
			recentUsedPassport = append(recentUsedPassport, passportArr[i])
		}
		if len(recentUsedPassport) == 3 {
			break
		}
	}

	userCollecteArr, err := userCollecteRepository.FindByUserId(account.ID)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}
	// 根据用户收藏表去设置recentUsedPassport切片中设备账号的收藏字段
	for i := range userCollecteArr {
		for j := range recentUsedPassport {
			if recentUsedPassport[j].Id == userCollecteArr[i].AssetAccountId {
				recentUsedPassport[j].Collection = "true"
				break
			}
		}
	}

	return Success(c, recentUsedPassport)
}

// GetUserPassportByUserId 获取用户拥有的所有设备账号
func GetUserPassportByUserId(ip, userId string) ([]model.HostOperate, error) {
	// 当前用户所属的所有用户组
	userGroupIds, err := userGroupMemberRepository.FindUserGroupIdsByUserId(userId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}

	sql := "SELECT id, name, department_id, department_name, state, button_state, description, relate_user, relate_user_group, relate_asset, relate_asset_group, relate_app, dep_level, priority, auth_expiration_date, strategy_begin_time, strategy_end_time, strategy_time_flag, ip_limit_type, ip_limit_list FROM operate_auth WHERE ("
	whereCondition := " relate_user LIKE '%" + userId + "%'"
	for i := range userGroupIds {
		whereCondition += " OR relate_user_group LIKE '%" + userGroupIds[i] + "%'"
	}
	whereCondition += " ) AND state != 'overdue' AND button_state != 'off'"
	orderBy := " ORDER BY dep_level ASC, department_name, priority ASC"
	sql += whereCondition
	sql += orderBy

	var operateAuthArr []model.OperateAuth
	err = operateAuthRepository.DB.Raw(sql).Find(&operateAuthArr).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return nil, err
	}

	var hostAuthOperateArr []model.HostOperate
	var notAuthAssetIdArr []string
	for i := range operateAuthArr {
		ok, err := service.ExpDateService.JudgeExpOperateAuth(&operateAuthArr[i])
		if nil != err {
			log.Errorf("JudgeExpOperateAuth Error: %v", err)
		}
		if !ok {
			// 当前时间未在此策略的授权时间段内
			err = strategyRelateAssetIdJoinNotAuthArr(operateAuthArr[i], &notAuthAssetIdArr)
			if nil != err {
				log.Errorf("strategyRelateAssetIdJoinNotAuthArr Error: %v", err)
			}
		} else {
			// 当前时间在此策略的授权时间段内
			if "blackList" == operateAuthArr[i].IpLimitType {
				// 黑名单
				if isIpsContainIp(ip, operateAuthArr[i].IpLimitList) {
					// 当前登录IP属于黑名单列表
					err = strategyRelateAssetIdJoinNotAuthArr(operateAuthArr[i], &notAuthAssetIdArr)
					if nil != err {
						log.Errorf("strategyRelateAssetIdJoinArr Error: %v", err)
					}
				} else {
					// 当前登录IP不属于黑名单列表
					// 策略关联的所有不在notAuthAssetIdArr内的账号加入hostAuthOperateArr
					err = strategyRelAsIdNotBelongArrJoinOthArr(operateAuthArr[i], notAuthAssetIdArr, &hostAuthOperateArr)
					if nil != err {
						log.Errorf("strategyRelAsIdNotBelongArrJoinOthArr Error: %v", err)
					}
				}
			} else {
				// 白名单
				if isIpsContainIp(ip, operateAuthArr[i].IpLimitList) {
					// 当前登录IP属于白名单列表
					// 策略关联的所有不在notAuthAssetIdArr内的账号加入hostAuthOperateArr
					err = strategyRelAsIdNotBelongArrJoinOthArr(operateAuthArr[i], notAuthAssetIdArr, &hostAuthOperateArr)
					if nil != err {
						log.Errorf("strategyRelAsIdNotBelongArrJoinOthArr Error: %v", err)
					}
				} else {
					// 当前登录IP不属于白名单列表
					err = strategyRelateAssetIdJoinNotAuthArr(operateAuthArr[i], &notAuthAssetIdArr)
					if nil != err {
						log.Errorf("strategyRelateAssetIdJoinArr Error: %v", err)
					}
				}
			}
		}
	}

	// 注意, 此处需在去重前添加, 避免出现重复主机(如果申请工单的逻辑是可以申请全部主机, 那在去重后添加就有可能出现重复的主机账号)
	// 1. 获取该用户所有的有效工单（审批通过/且在有效期之内）
	workOrderList, err := workOrderNewRepository.FindValidOrderByUserId(userId)
	if nil != err {
		log.Errorf("FindValidOrderByUserId Error: %v", err)
	}
	boolToStr := func(b bool) string {
		if b {
			return "on"
		}
		return "off"
	}
	for _, workOrder := range workOrderList {
		if !*workOrder.IsPermanent && (time.Now().Before(workOrder.BeginTime.Time) || time.Now().After(workOrder.EndTime.Time)) {
			continue
		}
		// 2. 获取工单关联的所有设备账号
		workOrderAsset, err := workOrderNewRepository.FindByWorkOrderId(workOrder.ID)
		if nil != err {
			log.Errorf("FindByWorkOrderId Error: %v", err)
			continue
		}
		for _, v := range workOrderAsset {
			asset, err := newAssetRepository.GetPassportById(context.TODO(), v.AssetId)
			if nil != err {
				log.Errorf("FindAssetByWorkOrderId Error: %v", err)
				continue
			}
			// 3. 将工单关联的设备账号加入hostAuthOperateArr
			hostAuthOperateArr = append(hostAuthOperateArr, model.HostOperate{
				Id:         asset.ID,
				AssetName:  asset.AssetName,
				Ip:         asset.Ip,
				Name:       asset.Name,
				Status:     asset.Status,
				Protocol:   asset.Protocol,
				Collection: "false",
				Download:   boolToStr(*workOrder.IsDownload),
				Upload:     boolToStr(*workOrder.IsUpload),
				Watermark:  boolToStr(*workOrder.IsWatermark),
			})
		}
	}
	// 2. 获取工单关联的所有设备账号
	// 且如果工单添加的主机是被上述策略排除掉的主机也属于正常情况, 因为工单功能本来就是让用户去使用本来正常无权限使用的主机
	// 这其中就包括被策略禁止的主机
	hostAuthOperateArr = HostOperateArrRemoveDuplicates(hostAuthOperateArr)
	return hostAuthOperateArr, nil
}
