package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/service"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
)

func AppOperatePagingEndpoint(c echo.Context) error {
	var aofs dto.AppOperateForSearch
	aofs.AppServer = c.QueryParam("server")
	aofs.Name = c.QueryParam("name")
	aofs.Program = c.QueryParam("app")
	aofs.Auto = c.QueryParam("auto")
	account, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 401, "请登陆")
	}

	err := GetChildDepIds(account.DepartmentId, &aofs.Departments)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}

	// 获取可运维的app账号
	appAuthOperateArr, err := findAppForAppAuthOperate(c, account.ID, aofs)
	if nil != err {
		log.Errorf("findAppForAppAuthOperate Error: %v", err)
		return Success(c, nil)
	}

	// 至此, 我们通过策略有效期、授权时段、IP限制已经过滤出了一份去重地可运维设备主号集合
	// 添加资产申请中该用户申请过的还未过期的设备账号
	userCollecteArr, err := userCollecteRepository.FindCollectAppByUserId(account.ID)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	// 根据用户收藏表去设置hostAuthOperateArr切片中设备账号的收藏字段
	// 同时反过来根据hostAuthOperateArr切片中设备账号 去清除用户收藏表中该用户以前收藏的主机，但现在该用户已没有权限运维的主机收藏记录或现在已被删除的设备账号
	isExist := false
	for i := range userCollecteArr {
		isExist = false
		for j := range appAuthOperateArr {
			if appAuthOperateArr[j].Id == userCollecteArr[i].ApplicationId {
				appAuthOperateArr[j].Collection = "true"
				isExist = true
			}
			if !isExist {
				err = userCollecteRepository.DeleteById(userCollecteArr[i].Id)
				if nil != err {
					log.Errorf("DB Error: %v", err)
					return FailWithDataOperate(c, 500, "查询失败", "", nil)
				}
			}
		}
	}

	return SuccessWithOperate(c, "", appAuthOperateArr)
}

// 获取可运维的app账号
func findAppForAppAuthOperate(c echo.Context, id string, aofs dto.AppOperateForSearch) (apps []dto.AppOperate, err error) {
	// 当前用户所属的所有用户组
	userGroupIds, err := userGroupMemberRepository.FindUserGroupIdsByUserId(id)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return nil, err
	}

	sql := "SELECT id, name, department_id, department_name, state, button_state, description, relate_user, relate_user_group, relate_asset, relate_asset_group, relate_app, dep_level, priority, auth_expiration_date, strategy_begin_time, strategy_end_time, strategy_time_flag, ip_limit_type, ip_limit_list FROM operate_auth WHERE ("
	whereCondition := " relate_user LIKE '%" + id + "%'"
	for i := range userGroupIds {
		whereCondition += " OR relate_user_group LIKE '%" + userGroupIds[i] + "%'"
	}
	whereCondition += ") AND state != 'overdue' AND button_state != 'off'"
	orderBy := "ORDER BY dep_level ASC, department_name, priority ASC"
	sql += whereCondition
	sql += orderBy

	var operateAuthArr []model.OperateAuth
	err = operateAuthRepository.DB.Raw(sql).Find(&operateAuthArr).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return nil, err
	}

	var notAuthAppIdArr []string
	for i := range operateAuthArr {
		fmt.Println(operateAuthArr[i].RelateApp)
		ok, err := service.ExpDateService.JudgeExpOperateAuth(&operateAuthArr[i])
		if nil != err {
			log.Errorf("JudgeExpOperateAuth Error: %v", err)
			return nil, err
		}
		if !ok {
			fmt.Println("过期了")
			relateAppArr := strings.Split(operateAuthArr[i].RelateApp, ",")
			notAuthAppIdArr = append(notAuthAppIdArr, relateAppArr...)
		} else {
			// 当前时间在此策略的授权时间段内
			if "blackList" == operateAuthArr[i].IpLimitType {
				// 黑名单
				if isIpsContainIp(c.RealIP(), operateAuthArr[i].IpLimitList) {
					// 当前登录IP属于黑名单列表
					relateAppArr := strings.Split(operateAuthArr[i].RelateApp, ",")
					notAuthAppIdArr = append(notAuthAppIdArr, relateAppArr...)
				} else {
					// 当前登录IP不属于黑名单列表
					// 策略关联的所有不在notAuthAssetIdArr内的账号加入hostAuthOperateArr
					err = findUsableAppForAppAuthOperateArr(operateAuthArr[i], notAuthAppIdArr, &apps, aofs)
					if nil != err {
						log.Errorf("strategyRelAsIdNotBelongArrJoinOthArr Error: %v", err)
						return nil, err
					}
				}
			} else {
				// 白名单
				if isIpsContainIp(c.RealIP(), operateAuthArr[i].IpLimitList) {
					// 当前登录IP属于白名单列表
					// 策略关联的所有不在notAuthAssetIdArr内的账号加入hostAuthOperateArr
					err = findUsableAppForAppAuthOperateArr(operateAuthArr[i], notAuthAppIdArr, &apps, aofs)
					if nil != err {
						log.Errorf("strategyRelAsIdNotBelongArrJoinOthArr Error: %v", err)
						return nil, err
					}
				} else {
					// 当前登录IP不属于白名单列表
					relateAppArr := strings.Split(operateAuthArr[i].RelateApp, ",")
					notAuthAppIdArr = append(notAuthAppIdArr, relateAppArr...)
				}
			}
		}
	}
	apps = appOperateArrRemoveDuplicates(apps)
	return
}

func appOperateArrRemoveDuplicates(arr []dto.AppOperate) []dto.AppOperate {
	result := make([]dto.AppOperate, 0, len(arr))
	temp := map[string]struct{}{}
	for _, item := range arr {
		l := len(temp)
		temp[item.Id] = struct{}{}
		if len(temp) != l {
			result = append(result, item)
		}
	}
	return result
}

func findUsableAppForAppAuthOperateArr(auth model.OperateAuth, notAuthAppIdArr []string, appAuthOperateArr *[]dto.AppOperate, aofs dto.AppOperateForSearch) error {
	// 策略关联的所有不在notAuthAssetIdArr内的账号加入hostAuthOperateArr
	relateAppArr := strings.Split(auth.RelateApp, ",")
	fmt.Println("relateAppArr: ", relateAppArr)

	apps, err := newApplicationRepository.GetApplicationIdsAndAofs(context.TODO(), relateAppArr, aofs)
	if nil != err {
		log.Errorf("GetApplicationIdsAndAofs Error: %v", err)
		return err
	}
	for j := range apps {
		//if !isAssetIdBelongArr(relateAppArr[j], notAuthAppIdArr) {
		//	asset, err := newApplicationRepository.GetApplicationById(context.TODO(), relateAppArr[j])
		//	if nil != err {
		//		log.Errorf("DB Error: %v", err)
		//		return err
		//	}
		if !isAssetIdBelongArr(apps[j].ID, notAuthAppIdArr) {
			// 加入三种权限控制策略 TODO
			*appAuthOperateArr = append(*appAuthOperateArr, dto.AppOperate{Id: apps[j].ID, Name: apps[j].Name, Program: apps[j].ProgramName, AppServer: apps[j].AppSerName, Collection: "false"})
		}
		//}
	}
	return nil
}

func AppOperateCollectEndpoint(c echo.Context) error {
	assetAccountId := c.Param("id")
	collect := c.QueryParam("collect")
	account, isSuccess := GetCurrentAccountNew(c)
	if !isSuccess {
		log.Error("GetCurrentAccountNew Error")
		return Fail(c, 401, "请先登录")
	}
	assetAccount, err := newApplicationRepository.GetApplicationById(context.TODO(), assetAccountId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}

	if "true" == collect {
		userCollect := model.UserCollectApp{UserId: account.ID, ApplicationId: assetAccountId}
		err = userCollecteRepository.CreateAppCollecte(&userCollect)
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "收藏失败", "", nil)
		}
		return SuccessWithOperate(c, "应用运维-收藏应用: 应用名称["+assetAccount.Name+"], 应用程序["+assetAccount.ProgramName+"] 应用服务器["+assetAccount.AppSerName+"] 设备账号["+assetAccount.Passport+"]", nil)
	} else {
		err = userCollecteRepository.RemoveAppCollecte(&model.UserCollectApp{
			UserId:        account.ID,
			ApplicationId: assetAccountId,
		})
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "取消收藏失败", "", nil)
		}
		return SuccessWithOperate(c, "应用运维-取消收藏应用: 应用名称["+assetAccount.Name+"], 应用程序["+assetAccount.ProgramName+"] 应用服务器["+assetAccount.AppSerName+"] 设备账号["+assetAccount.Passport+"]", nil)
	}
}

func GetAppOperateRecentUsedEndpoint(c echo.Context) error {
	var aofs dto.AppOperateForSearch
	aofs.AppServer = c.QueryParam("server")
	aofs.Name = c.QueryParam("name")
	aofs.Program = c.QueryParam("app")
	aofs.Auto = c.QueryParam("auto")

	account, isSuccess := GetCurrentAccountNew(c)
	if !isSuccess {
		log.Error("GetCurrentAccountNew Error")
		return Fail(c, 401, "请先登录")
	}
	err := GetChildDepIds(account.DepartmentId, &aofs.Departments)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}
	assetAccounts, err := appSessionRepository.GetRecentAppIdByUserId(context.TODO(), account.ID)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	operate, err := findAppForAppAuthOperate(c, account.ID, aofs)
	if err != nil {
		log.Errorf("findAppForAppAuthOperate Error: %v", err)
		return FailWithDataOperate(c, 500, "查询失败", "", nil)
	}

	userCollecteArr, err := userCollecteRepository.FindCollectAppByUserId(account.ID)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return Success(c, nil)
	}

	var appOperateArr []dto.AppOperate
	// 判断是否在assetAccounts中
	for i := range operate {
		for j := range assetAccounts {
			if operate[i].Id == assetAccounts[j] {
				appOperateArr = append(appOperateArr, operate[i])
			}
		}
	}

	// 判断是否在userCollecteArr中
	for i := range operate {
		for j := range userCollecteArr {
			if operate[i].Id == userCollecteArr[j].ApplicationId {
				appOperateArr[i].Collection = "true"
			}
		}
	}

	return SuccessWithOperate(c, "查询成功", appOperateArr)
}

func AppOperateConnectTestEndpoint(c echo.Context) error {
	assetAccountId := c.Param("asset_account_id")

	assetAccount, err := newApplicationRepository.FindById(context.TODO(), assetAccountId)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "连接测试失败", "", nil)
	}

	active := utils.Tcping(assetAccount.IP, assetAccount.Port)
	if !active {
		return FailWithDataOperate(c, 200, "连接失败, 设备不在线", "应用运维-连接测试: 应用名称["+assetAccount.Name+"], 应用程序["+assetAccount.ProgramName+"] 应用服务器["+assetAccount.AppSerName+"] 设备账号["+assetAccount.Passport+"]", nil)
	}

	return SuccessWithOperate(c, "应用运维-连接测试: 应用名称["+assetAccount.Name+"], 应用程序["+assetAccount.ProgramName+"] 应用服务器["+assetAccount.AppSerName+"] 设备账号["+assetAccount.Passport+"], 测试结果[成功]", nil)
}

// AppOperateCollectListEndpoint 查询用户收藏的应用
func AppOperateCollectListEndpoint(c echo.Context) error {
	var aofs dto.AppOperateForSearch
	aofs.AppServer = c.QueryParam("server")
	aofs.Name = c.QueryParam("name")
	aofs.Program = c.QueryParam("app")
	aofs.Auto = c.QueryParam("auto")
	account, isSuccess := GetCurrentAccountNew(c)
	if !isSuccess {
		return Fail(c, 402, "获取当前登录用户失败")
	}

	err := GetChildDepIds(account.DepartmentId, &aofs.Departments)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return FailWithDataOperate(c, 500, "操作失败", "", nil)
	}

	// 获取可运维的app账号
	appAuthOperateArr, err := findAppForAppAuthOperate(c, account.ID, aofs)
	if nil != err {
		log.Errorf("findAppForAppAuthOperate Error: %v", err)
		return Success(c, nil)
	}

	userCollecteArr, err := userCollecteRepository.FindCollectAppByUserId(account.ID)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return Success(c, nil)
	}

	resp := make([]dto.AppOperate, 0)
	for i := range appAuthOperateArr {
		for j := range userCollecteArr {
			if appAuthOperateArr[i].Id == userCollecteArr[j].ApplicationId {
				appAuthOperateArr[i].Collection = "true"
				resp = append(resp, appAuthOperateArr[i])
			}
		}
	}
	return Success(c, resp)
}

// DevopsLogEndpoint 获取运维日志
func DevopsLogEndpoint(c echo.Context) error {
	u, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 403, "获取用户信息失败")
	}
	var oplfs dto.OperateLogForSearch
	oplfs.Auto = c.QueryParam("auto")
	oplfs.LogoutTime = c.QueryParam("logoutTime")
	oplfs.LoginTime = c.QueryParam("loginTime")
	oplfs.Username = c.QueryParam("username")
	oplfs.Ip = c.QueryParam("ip")
	oplfs.Nickname = c.QueryParam("nickname")
	oplfs.AssetName = c.QueryParam("assetName")
	oplfs.AssetIp = c.QueryParam("assetIp")
	oplfs.Passport = c.QueryParam("passport")

	err := GetChildDepIds(u.DepartmentId, &oplfs.Departments)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return Fail(c, 500, "获取部门信息失败")
	}
	opl, err := hostOperateRepository.GetOperateLogList(oplfs)
	if nil != err {
		log.Errorf("GetOperateLogList Error: %v", err)
		return Fail(c, 500, "获取运维日志失败")
	}
	return Success(c, opl)
}

func DevopsLogExportEndpoint(c echo.Context) error {
	u, f := GetCurrentAccountNew(c)
	if !f {
		return Fail(c, 403, "获取用户信息失败")
	}

	var dp []int64
	err := GetChildDepIds(u.DepartmentId, &dp)
	if nil != err {
		log.Errorf("GetChildDepIds Error: %v", err)
		return Fail(c, 500, "获取部门信息失败")
	}
	opl, err := hostOperateRepository.GetOperateLogForExport(dp)
	if nil != err {
		log.Errorf("GetOperateLogList Error: %v", err)
		return Fail(c, 500, "获取运维日志失败")
	}

	forExport := make([][]string, len(opl))
	for i, v := range opl {
		asset := utils.Struct2StrArr(v)
		forExport[i] = make([]string, len(asset))
		forExport[i] = asset
	}
	headerForExport := []string{"登录时间", "来源IP", "运维人员账号", "运维人员名称", "类型", "运维对象名称", "运维对象地址", "运维对象账号", "登出时间"}
	fileNameForExport := "运维日志"
	file, err := utils.CreateExcelFile(fileNameForExport, headerForExport, forExport)
	if err != nil {
		return FailWithDataOperate(c, 403, "导出失败", "", nil)
	}
	fileName := "运维日志.xlsx"

	// 记录操作日志
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		Result:          "成功",
		LogTypes:        "操作日志",
		Created:         utils.NowJsonTime(),
		LogContents:     "运维日志-导出: 导出文件[" + fileName + "]",
		Users:           u.Username,
		Names:           u.Nickname,
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

	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+fileName)
	return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(buff.Bytes()))
}
