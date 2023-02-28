package api

import (
	"context"
	"strings"
	"time"
	"tkbastion/pkg/constant"
	"tkbastion/pkg/global"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/service"
	"tkbastion/server/utils"

	"gorm.io/gorm"

	"github.com/labstack/gommon/log"

	"github.com/labstack/echo/v4"
)

func NewCommandPagingEndpoint(c echo.Context) error {
	var cfs dto.CommandForSearch
	cfs.Auto = c.QueryParam("auto")
	cfs.Name = c.QueryParam("name")
	cfs.Content = c.QueryParam("content")

	u, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 403, "无权限")
	}

	cfs.Uid = u.ID

	commands, _, err := commandsNewRepository.GetCommandNewList(context.TODO(), cfs)
	if err != nil {
		log.Error("GetCommandNewList: ", err)
		return Fail(c, 500, err.Error())
	}

	return Success(c, commands)
}

func NewCommandCreateEndpoint(c echo.Context) error {
	var command dto.CommandForCreate
	if err := c.Bind(&command); err != nil {
		return Fail(c, 400, err.Error())
	}

	var commandModel model.NewCommand
	u, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 403, "无权限")
	}
	commandModel.Name = command.Name
	commandModel.Content = command.Content
	commandModel.Info = command.Info
	commandModel.UserId = u.ID
	commandModel.ID = utils.UUID()
	commandModel.Created = utils.NowJsonTime()

	_, err := commandsNewRepository.GetCommandNewByName(context.TODO(), command.Name)
	if err == nil {
		return FailWithDataOperate(c, 400, "命令已存在", "动态指令-创建: 指令名称["+command.Name+"], 失败原因[命令已存在]", nil)
	} else if err != gorm.ErrRecordNotFound {
		log.Error("GetCommandNewByName: ", err)
		return Fail(c, 500, err.Error())
	}

	err = commandsNewRepository.CreateCommandNew(context.TODO(), commandModel)
	if err != nil {
		log.Error("CreateCommandNew: ", err)
		return Fail(c, 500, err.Error())
	}

	return SuccessWithOperate(c, "动态指令-创建: 指令名称["+command.Name+"], 创建成功", nil)
}

func NewCommandUpdateEndpoint(c echo.Context) error {
	id := c.Param("id")
	var command dto.CommandForUpdate
	if err := c.Bind(&command); err != nil {
		log.Errorf("Bind: %v", err)
		return Fail(c, 400, err.Error())
	}

	_, err := commandsNewRepository.GetCommandNewByIdAndName(context.TODO(), id, command.Name)
	if err == nil {
		return FailWithDataOperate(c, 400, "命令已存在", "动态指令-修改: 指令名称["+command.Name+"], 失败原因[命令已存在]", nil)
	} else if err != gorm.ErrRecordNotFound {
		log.Error("GetCommandNewByName: ", err)
		return Fail(c, 500, err.Error())
	}

	commandModel, err := commandsNewRepository.GetCommandNewById(context.TODO(), id)
	if err != nil {
		log.Error("GetCommandNewById: ", err)
		return Fail(c, 500, err.Error())
	}

	command.ID = commandModel.ID

	err = commandsNewRepository.UpdateCommandNew(context.TODO(), command)
	if err != nil {
		log.Error("UpdateCommandNew: ", err)
		return Fail(c, 500, err.Error())
	}

	return SuccessWithOperate(c, "动态指令-修改: 指令名称["+command.Name+"]->["+commandModel.Name+"],指令内容["+command.Content+"]->["+commandModel.Content+"]", nil)
}

func NewCommandDeleteEndpoint(c echo.Context) error {
	id := c.Param("id")
	ids := strings.Split(id, ",")

	var names string
	for _, id := range ids {
		command, err := commandsNewRepository.GetCommandNewById(context.TODO(), id)
		if err != nil {
			log.Error("GetCommandNewById: ", err)
			return Fail(c, 500, err.Error())
		}
		names += command.Name + ","
	}
	err := commandsNewRepository.DeleteCommandNew(context.TODO(), ids)
	if err != nil {
		log.Error("DeleteCommandNew: ", err)
		return Fail(c, 500, err.Error())
	}

	return SuccessWithOperate(c, "动态指令-删除: 指令名称["+names+"], 删除成功", nil)
}

func NewCommandGetEndpoint(c echo.Context) error {
	id := c.Param("id")

	err, isAllow := sysMaintainService.IsAllowOperate()
	if nil != err {
		log.Error("动态指令-执行: 获取系统授权类型失败")
		return FailWithDataOperate(c, 500, "执行失败, 请联系技术人员查看系统日志以确定失败原因", "", nil)
	}
	if !isAllow {
		return FailWithDataOperate(c, 400, "系统被授权后才可使用动态指令, 请您先联系厂商获取授权文件", "动态指令-执行: 执行失败, 失败原因[系统未获取授权]", nil)
	}
	command, err := commandsNewRepository.GetCommandNewById(context.TODO(), id)
	if err != nil {
		log.Error("GetCommandNewById: ", err)
		return Fail(c, 500, err.Error())
	}

	return Success(c, command)
}

func CommandAssetEndpoint(c echo.Context) error {
	account, _ := GetCurrentAccountNew(c)
	// 当前用户所属的所有用户组
	userGroupIds, err := userGroupMemberRepository.FindUserGroupIdsByUserId(account.ID)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return SuccessWithOperate(c, "", nil)
	}

	var sql = "SELECT id, name, department_id, department_name, state, button_state, description, relate_user, relate_user_group,download,upload,watermark, relate_asset, relate_asset_group, relate_app, dep_level, priority, auth_expiration_date, strategy_begin_time, strategy_end_time, strategy_time_flag, ip_limit_type, ip_limit_list FROM operate_auth WHERE ("
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
			if asset.Protocol == constant.SSH {
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
	searchWhereCondition := ") AND protocol = 'ssh'"

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
