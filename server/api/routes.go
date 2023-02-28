package api

import (
	"net/http"
	"os"
	"tkbastion/pkg/log"

	"github.com/labstack/echo/v4"

	"tkbastion/pkg/config"
	"tkbastion/pkg/validator"
	"tkbastion/server/repository"
	tkservice "tkbastion/server/service"

	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

// Token TODO token名字,后期再看是否需要修改
// 已添加至const中
const Token = "X-Auth-Token"

// SetupRoutes api包下,某业务模块若要执行DB操作,可以使用这里变量(xxxRepository)(repository包为封装的DB方法)
// 非api包,则调用全局DB接口:global.DBConn
// service包为封装的某些模块的初始化数据功能
func SetupRoutes(db *gorm.DB) *echo.Echo {

	InitRepository(db)
	repository.SetupRepository(db)
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
	InitLoadMiddlewarePath()

	e := echo.New()
	//TODO 注册参数校验器
	e.Validator = new(validator.CustomValidator)
	e.HideBanner = true

	e.Use(log.Hook()) // 包含记录操作日志逻辑
	if config.GlobalCfg.Debug == false {
		e.File("/tkbastion", "/tkbastion/dist/index.html")
		fileList, err := os.ReadDir("/tkbastion/web/dist")
		if err != nil {
			log.WithError(err).Errorf("读取目录下的所有文件异常,异常信息: %v", err.Error())
		}
		for _, file := range fileList {
			if file.IsDir() {
				if file.Name() == "static" {
					e.Static("/"+file.Name(), "/tkbastion/web/dist/"+file.Name())
				}
			} else {
				e.File("/"+file.Name(), "/tkbastion/web/dist/"+file.Name())
			}
		}
	} else {
		// 读取目录下的所有文件
		e.File("/", "web/dist/index.html")
		fileList, err := os.ReadDir("web/dist")
		if err != nil {
			log.WithError(err).Errorf("读取目录下的所有文件异常,异常信息: %v", err.Error())
		}
		for _, file := range fileList {
			if file.IsDir() {
				if file.Name() == "static" {
					e.Static("/"+file.Name(), "web/dist/"+file.Name())
				}
			} else {
				e.File("/"+file.Name(), "web/dist/"+file.Name())
			}
		}
	}

	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		Skipper:      middleware.DefaultSkipper,
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
	}))
	e.Use(ErrorHandler)
	e.Use(Auth)
	e.Use(ExpDataCheck)
	e.Use(AuthCheckRole)

	e.POST("/login", LoginEndpointNew)
	e.POST("/loginWithAuth", LoginWithAuthEndpointNew)
	e.GET("/userLoginType", UserLoginTypeEndpoint)
	e.GET("/keepalived", KeepalivedEndpoint)

	e.POST("/loginWithTotp", loginWithTotpEndpoint)
	e.POST("/loginWithMail", loginWithMailEndpoint)
	e.POST("/logout", LogoutEndpoint)
	e.POST("/login-change-password", FirstLoginForceChangePasswordEndpoint)

	e.GET("/reload-totp", ReloadTOTPEndpoint)
	e.POST("/confirm-totp", ConfirmTOTPEndpoint)
	e.GET("/info", InfoEndpointNew)
	e.GET("/user-role-menus", RoleMenuEndpoint)
	e.GET("/tree-role-select", RoleButtonMenuTreeSelectEndpoint)

	overview := e.Group("/overview")
	{
		// 概览头行数量获取
		overview.GET("/counter", OverviewCounterNewEndPoint)
		// 最近用户访问量、最近资产访问量
		overview.GET("/recent-user-asset", OverviewRecentUserAndAssetCountEndPoint)
		// 累计用户、资产访问量TOP5
		overview.GET("/top-user-asset", OverviewRecentAssetAccessCountEndPoint)
		// 待审批工单
		overview.GET("/un-approval", OverviewUnapprovedListEndpoint)
		// 访问数量统计
		overview.GET("/access-count", OverviewAccessCountEndpoint)
		// 获取系统信息
		overview.GET("/system-info", OverviewSystemInfoEndpoint)
		// 许可信息
		overview.GET("/license-info", LicenseManagementGetEndpoint)
		// 实时在线用户
		overview.GET("/online-user", OnlineUsersPagingEndpoint)
	}

	departments := e.Group("/departments")
	{
		departments.GET("", DepartmentPagingEndpoint)
		departments.POST("", DepartmentCreateEndpoint)
		departments.PUT("/:id", DepartmentUpdateEndpoint)
		departments.DELETE("/:id", DepartmentDeleteEndpoint)

		// TODO 暂不考虑导出的文件各列宽度并未根据数据长度适配引起的用户体验感不佳问题
		departments.GET("/download-template", DepartmentDownloadTemplateEndpoint)
		departments.POST("/import", DepartmentImportEndpoint)
		departments.GET("/export", DepartmentExportEndpoint)
	}

	users := e.Group("/users")
	{
		users.POST("", UserNewCreateEndpoint)
		users.GET("/paging", UserNewPagingEndpoint)
		users.PUT("/:id", UserNewUpdateEndpoint)
		users.DELETE("/:id", UserNewDeleteEndpoint)
		// 获取角色信息
		users.GET("/roles", RolePagingForUserCreatEndpoint)

		// 下载模板
		users.GET("/download-template", UserDownloadTemplateEndpoint)
		users.POST("/import", UserNewImportEndpoint)
		users.GET("/export", UserNewExportEndpoint)

		// 启用/禁用
		users.POST("/enable", UserNewIsDisableEndpoint)
		// 批量编辑用户
		users.POST("/batch-edit/:id", UserEditBatchUserEndpoint)
		// 获取用户详情
		users.GET("/detail/:id", GetDetailsUserInfo)
		// 获取用户组详情
		users.GET("/user-group/:id", GetUserGroupInfo)
		// 获取用户策略详情
		users.GET("/strategy/:id", GetUserStrategyInfo)
		// 获取设备资产详情
		users.GET("/assets/:id", GetDeviceAssetInfo)
		// 获取应用资产详情
		users.GET("/app/:id", GetAppAssetInfo)
		// 获取指令策略详情
		users.GET("/command-strategy/:id", GetCommandStrategyInfo)

	}

	userGroups := e.Group("/user-groups")
	{
		userGroups.POST("", UserGroupNewCreateEndpoint)
		userGroups.GET("/paging", UserGroupNewPagingEndpoint)
		userGroups.PUT("/:id", UserGroupNewUpdateEndpoint)
		userGroups.DELETE("/:id", UserGroupNewDeleteEndpoint)

		// 下载模板
		userGroups.GET("/download-template", UserGroupDownloadTemplateEndpoint)
		userGroups.POST("/import", UserGroupNewImportEndpoint)
		userGroups.GET("/export", UserGroupNewExportEndpoint)

		// 查看当前用户所能看到的用户
		userGroups.GET("/user-paging", GetCurrentDepartmentUserChildEndpoint)
		// 查看当前用户组所属部门所能看到的所有用户
		userGroups.GET("/user-paging/:id", GetCurrentUserGroupDepartmentUserChildrenEndpoint)
		// 查看已关联的用户
		userGroups.GET("/user-relate/:id", UserGroupNewMemberEndpoint)
		// 关联用户
		userGroups.POST("/user-relate/:id", UserGroupNewMemberAddEndpoint)
	}

	userStrategy := e.Group("/user-strategy")
	{
		// 策略列表
		userStrategy.GET("/paging", UserStrategyPagingEndpoint)
		// 新建策略
		userStrategy.POST("", UserStrategyCreateEndpoint)
		// 更新策略
		userStrategy.PUT("/:id", UserStrategyUpdateEndpoint)
		// 删除策略
		userStrategy.DELETE("/:id", UserStrategyDeleteEndpoint)
		// 启用或禁用策略
		userStrategy.POST("/enable", UserStrategyEnableEndpoint)

		// 新建时查看策略可关联用户
		userStrategy.GET("/users", GetCurrentDepartmentUserChildEndpoint)
		// 直接关联时查看可关联用户
		userStrategy.GET("/users/:id", UserStrategyGetUserEndpoint)
		// 查看已关联用户
		userStrategy.GET("/related-users/:id", UserStrategyRelatedUsersEndpoint)
		// 关联用户
		userStrategy.POST("/related-users", UserStrategyAddUserEndpoint)

		// 新建时查看策略可关联用户组
		userStrategy.GET("/user-groups", GetCurrentDepartmentUserGroupChildEndpoint)
		// 直接关联时查看可关联的用户组
		userStrategy.GET("/user-groups/:id", UserStrategyGetUserGroupEndpoint)
		// 查看已关联用户组
		userStrategy.GET("/related-groups/:id", UserStrategyRelatedGroupsEndpoint)
		// 关联用户组
		userStrategy.POST("/related-groups", UserStrategyAddUserGroupEndpoint)
	}

	role := e.Group("/role")
	{
		// 角色列表
		role.GET("/paging", RolePagingEndpoint)
		// 角色创建
		role.POST("", RoleCreateEndpoint)
		// 角色更新
		role.PUT("/:id", RoleUpdateEndpoint)
		// 角色删除
		role.DELETE("/:id", RoleDeleteEndpoint)
	}

	newAsset := e.Group("/new-assets")
	{
		// 通过id获取X11对应的两个程序id
		newAsset.GET("/x11/:id", GetX11ProgramIdEndpoint)
		newAsset.GET("/paging", NewAssetPagingEndpoint)
		newAsset.POST("", NewAssetCreateEndpoint)
		newAsset.PUT("/:id", NewAssetUpdateEndpoint)
		// 下载私钥文件
		newAsset.GET("/download-private-key/:id", NewAssetDownloadPrivateKeyEndpoint)
		newAsset.DELETE("/:id", NewAssetDeleteEndpoint)
		newAsset.DELETE("/all", NewAssetBatchDeleteEndpoint)
		// 导出
		newAsset.GET("/export", NewAssetExportEndpoint)
		// 导入
		newAsset.POST("/import", NewAssetImportEndpoint)
		// 下载模板
		newAsset.GET("/download-template", NewAssetDownloadTemplateEndpoint)

		// 批量编辑
		newAsset.PUT("/batch-edit/:id", AssetBatchEditEndpoint)

		// 获取设备详情
		newAsset.GET("/detail/:id", NewAssetGetEndpoint)
		// 获取设备账号详情
		newAsset.GET("/account/:id", PassPortListForAssetEndpoint)
		// 获取设备分组详情
		newAsset.GET("/group/:id", GetAssetGroupEndpoint)
		// 获取设备运维策略详情
		newAsset.GET("/strategy/:id", GetAssetPolicyEndpoint)
		// 获取设备运维指令详情
		newAsset.GET("/command/:id", GetAssetCommandPolicyEndpoint)
	}

	// 设备账号管理
	passport := e.Group("/passport")
	{
		//passport.GET("/paging",)
		passport.POST("", PassPortCreateEndpoint)
		passport.PUT("/:id", PassPortUpdateEndpoint)
		passport.DELETE("/:id", PassPortDeleteEndpoint)
		//passport.GET("/:id", PassPortGetEndpoint)
		// 批量删除设备帐号
		passport.DELETE("/:aid/all", PassPortBatchDeleteEndpoint)
		// 禁用账号
		passport.PUT("/:id/disable", PassPortDisableEndpoint)
		// 启用账号
		passport.PUT("/:id/enable", PassPortEnableEndpoint)
		// 查询设备所有账号
		passport.GET("/asset/:id", PassPortListEndpoint)
	}

	newAssetGroup := e.Group("/new-asset-group")
	{
		newAssetGroup.GET("/paging", NewAssetGroupPagingEndpoint)
		newAssetGroup.POST("", NewAssetGroupCreateEndpoint)
		newAssetGroup.PUT("/:id", NewAssetGroupUpdateEndpoint)
		newAssetGroup.DELETE("/:id", NewAssetGroupDeleteEndpoint)
		newAssetGroup.DELETE("/all", NewAssetGroupBatchDeleteEndpoint)
		// 查询创建时可关联设备
		newAssetGroup.GET("/asset", NewAssetGroupAssetPagingEndpoint)
		// 查询编辑时可关联设备
		newAssetGroup.GET("/asset-for-edit/:id", NewAssetGroupAssetsEndpoint)
		// 查询已关联设备
		newAssetGroup.GET("/asset/:id", NewAssetGroupAssetListEndpoint)
		// 关联设备
		newAssetGroup.PUT("/:id/assets", NewAssetGroupAssetEndpoint)
	}

	// 系统类型
	systemType := e.Group("/system-type")
	{
		systemType.GET("/paging", SystemTypePagingEndpoint)
		systemType.POST("", SystemTypeCreateEndpoint)
		systemType.PUT("/:id", SystemTypeUpdateEndpoint)
		systemType.DELETE("/:id", SystemTypeDeleteEndpoint)
		// 批量删除
		systemType.DELETE("", SystemTypeBatchDeleteEndpoint)
	}

	// 新应用服务器
	newAppServer := e.Group("/new-app-ser")
	{
		newAppServer.GET("/paging", GetApplicationServerList)
		newAppServer.POST("", AddApplicationServer)
		newAppServer.PUT("/:id", UpdateApplicationServer)
		newAppServer.GET("/:id", GetApplicationServer)
		newAppServer.DELETE("/:id", DeleteApplicationServer)
		// 批量删除
		newAppServer.DELETE("/all", DeleteApplicationServers)

		// 新建程序
		newAppServer.POST("/program", AddProgram)
		// 更新程序
		newAppServer.PUT("/program/:id", UpdateProgram)
		// 删除程序
		newAppServer.DELETE("/program/:id", DeleteProgram)
		// 批量删除程序
		newAppServer.DELETE("/program/all", DeleteMultiProgram)
	}

	// 新应用
	newApp := e.Group("/new-app")
	{
		newApp.GET("/paging", GetApplicationList)
		newApp.POST("", AddApplication)
		newApp.PUT("/:id", UpdateApplication)
		newApp.DELETE("/:id", DeleteApplication)
		// 批量删除
		newApp.DELETE("/all", DeleteApplications)
		// 获取应用详情
		newApp.GET("/:id", GetApplicationById)
		// 获取策略
		newApp.GET("/policy/:id", GetApplicationOpsPolicy)
		// 导出
		newApp.GET("/export", ExportApplication)
	}

	// 新计划任务
	newJob := e.Group("/new-job")
	{
		newJob.GET("/paging", NewJopPagingEndpoint)
		newJob.POST("", NewJopCreateEndpoint)
		newJob.PUT("/:id", NewJopUpdateEndpoint)
		// 获取已上传的脚本
		newJob.GET("/:id/script", NewJopScriptEndpoint)
		newJob.DELETE("/:id", NewJopDeleteEndpoint)
		newJob.DELETE("/:id/all", NewJopDeleteBatchEndpoint)
		newJob.GET("/:id", NewJopGetEndpoint)
		// 执行计划任务
		newJob.POST("/:id/run", NewJopStartEndpoint)
		// 关联设备
		newJob.POST("/:id/asset", NewJopDeviceEndpoint)
		// 关联设备组
		newJob.POST("/:id/asset-group", NewJopDeviceGroupEndpoint)
		// 获取可关联的设备
		newJob.GET("/:id/asset", NewJopDevicePagingEndpoint)
		// 获取已关联的设备
		newJob.GET("/:id/asset/relate", NewJopDeviceListEndpoint)
		// 获取可关联的设备组
		newJob.GET("/:id/asset-group", NewJopDeviceGroupPagingEndpoint)
		// 获取已关联的设备组
		newJob.GET("/:id/asset-group/relate", NewJopDeviceGroupListEndpoint)
	}

	// 新计划任务日志
	newJobLog := e.Group("/new-job-log")
	{
		newJobLog.GET("/paging", NewJopLogPagingEndpoint)
		newJobLog.GET("/:id", NewJopLogGetEndpoint)
		// 导出
		newJobLog.GET("/:id/export", NewJopLogExportEndpoint)
	}

	// session
	newSession := e.Group("/new-session")
	{
		newSession.GET("/paging", NewSessionHistoryEndpoint) //查看历史会话
		// 导出
		newSession.GET("/export", NewSessionExportEndpoint)
		// 获取会话详情
		newSession.GET("/:id", NewSessionDetailEndpoint)
		// 应用和设备通用-----------
		newSession.POST("", NewSessionCreateEndpoint) // 归属设备连接
		// 更改会话状态
		newSession.PUT("/:id", NewSessionConnectEndpoint)
		// 调整会话窗口大小
		newSession.PUT("/:id/size", NewSessionResizeEndpoint)
		// ---------------------

		// 会话回放
		newSession.GET("/:id/replay", NewSessionReplayEndpoint)
		// 在线会话
		newSession.GET("/online", NewSessionOnlineEndpoint)
		// 创建剪贴板记录
		newSession.POST("/:id/clipboard", SessionClipboardCreateEndpoint)
		// 获取剪贴板记录
		newSession.GET("/:id/clipboard", SessionClipboardGetEndpoint)

		// 设备接入
		newSession.GET("/:id/tunnel", NewTunEndpoint)                  // 归属主机接入、监控在线会话菜单api
		newSession.GET("/:id/tunnel-monitor", NewSessionTunnelMonitor) // 归属监控在线会话菜单api
		newSession.PUT("/:id/disconnect", NewSessionDisconnectEndpoint)
		newSession.GET("/command", GetSshCommand)

		// ssh连接
		newSession.GET("/ssh", NewSSHEndpoint)        // 归属主机接入、监控在线会话菜单api
		newSession.GET("/ssh-monitor", NewSshMonitor) // 归属监控在线会话菜单api

		// ssh登陆测试
		newSession.GET("/ssh-test", SshTest)
		// 手动登陆
		newSession.POST("/login-manual", LoginManual)

		// file-session
		{ // sftp连接
			newSession.GET("/sftp", NewSftpEndpoint)
			// ftp连接
			newSession.GET("/ftp", NewFtpEndpoint)
			// 获取文件操作记录
			newSession.GET("/file-record", FileRecordEndpoint)
			// 下载记录中的文件
			newSession.GET("/file-download", FileDownloadEndpoint)
		}

		// 文件浏览器操作
		{
			newSession.POST("/:id/ls", NewLsEndpoint)
			newSession.GET("/:id/download", NewDownloadEndpoint)
			newSession.POST("/:id/upload", NewUploadEndpoint)
		}

		// 在线用户
		newSession.GET("/online-users", OnlineUsersPagingEndpoint)
		newSession.POST("/online-users/disconnect", OnlineUsersDisconnect)

		// 标为已阅
		newSession.PUT("/read", MarkAsRead)
		// 标为未阅
		newSession.PUT("/unread", MarkAsUnread)
		// 下载会话录像
		newSession.GET("/download", NewSessionVideoDownloadEndpoint)
	}

	// file_session
	fileSession := e.Group("/file-session")
	{
		fileSession.POST("/:id/ls", FileSessionLsEndpoint)
		fileSession.GET("/:id/download", FileSessionDownloadEndpoint)
		fileSession.POST("/:id/upload", FileSessionUploadEndpoint)
		fileSession.PUT("/:id/disconnect", FileSessionDisconnectEndpoint)
		fileSession.POST("/:id/mkdir", FileSessionMkdirEndpoint)
		fileSession.POST("/:id/rm", FileSessionRmEndpoint)
		fileSession.POST("/:id/rename", FileSessionRenameEndpoint)
	}

	// app_session
	appSession := e.Group("/app-session")
	{
		appSession.GET("/paging", AppSessionHistoryEndpoint) //查看历史会话
		// 导出历史会话
		appSession.GET("/export", AppSessionExportEndpoint)
		// 获取会话详情
		appSession.GET("/:id", AppSessionDetailEndpoint)

		// 应用接入
		appSession.GET("/:id/tunnel", AppTunEndpoint)
		//appSession.GET("/:id/tunnel-monitor", AppSessionTunnelMonitor)

		// 已阅
		appSession.PUT("/read", MarkAppAsRead)
		// 未阅
		appSession.PUT("/unread", MarkAppAsUnread)
		// 下载会话录像
		appSession.GET("/download", AppSessionVideoDownloadEndpoint)
		// 回放
		appSession.GET("/:id/replay", AppSessionReplayEndpoint)
	}

	newStorage := e.Group("/new-storage")
	{
		newStorage.GET("/paging", NewStoragePagingEndpoint)
		newStorage.POST("", NewStorageCreateEndpoint)
		newStorage.PUT("/:id", NewStorageUpdateEndpoint)
		newStorage.DELETE("/:id", NewStorageDeleteEndpoint)

		newStorage.POST("/:id/ls", NewStorageLsEndpoint)
		newStorage.GET("/:id/download", NewStorageDownloadEndpoint)
		newStorage.POST("/:id/upload", NewStorageUploadEndpoint)
		newStorage.POST("/:id/rm", NewStorageDeleteFileEndpoint)
		newStorage.POST("/:id/mkdir", NewStorageMkdirEndpoint)
		newStorage.POST("/:id/rename", NewStorageRenameEndpoint)
		newStorage.POST("/:id/edit", NewStorageEditEndpoint)
	}

	// 登录报表
	loginReport := e.Group("/login-report")
	{
		// 协议访问统计
		loginReport.GET("/protocol", GetProtocolCountStatistEndpoint)
		loginReport.GET("/protocol-details", GetLoginDetailsEndpoint)
		loginReport.GET("/protocol-export", GetProtocolCountStatistExportEndpoint)
		// 用户访问统计
		loginReport.GET("/user/chart", GetUserCountStatistEndpoint)
		loginReport.GET("/user/surface", GetUserCountStatistSurfaceEndpoint)
		loginReport.GET("/user/details", GetUserLoginDetailsEndpoint)
		loginReport.GET("/user/export", GetUserCountStatistExportEndpoint)
		// 登录尝试统计
		loginReport.GET("/attempt", GetAttemptCountStatistEndpoint)
		loginReport.GET("/attempt/details", GetAttemptDetailsEndpoint)
		loginReport.GET("/attempt-export", GetAttemptCountStatistExportEndpoint)
	}

	// 运维报表
	operaReport := e.Group("/opera-report")
	{
		// 资产运维
		operaReport.GET("/asset", GetAssetAccessReport)
		operaReport.GET("/asset/export", ExportAssetAccessReport)
		operaReport.GET("/asset/asset", GetAssetAccessReportAssetDetailByTime)
		operaReport.GET("/asset/user", GetAssetAccessReportUserDetailByTime)
		// 会话时长
		operaReport.GET("/session-assets", GetSessionAssetsReport)
		operaReport.GET("/session-users", GetSessionUsersReport)
		operaReport.GET("/session/export", ExportSessionReport)
		operaReport.GET("/session/asset", GetSessionReportAssetDetailByTime)
		operaReport.GET("/session/user", GetSessionReportUserDetailByTime)
		// 命令统计
		operaReport.GET("/command", GetCommandRecordByTime)
		operaReport.GET("/command-details", GetCommandRecordDetails)
		operaReport.GET("/command/export", ExportCommandRecordReport)
		// 告警报表
		operaReport.GET("/alarm", GetAlarmReport)
		operaReport.GET("/alarm-details", GetAlarmReportDetails)
		operaReport.GET("/alarm/export", ExportAlarmReport)
	}
	// 定期报表
	regularReport := e.Group("/regular-report")
	{
		// 定期策略
		regularReport.GET("/policy", FindRegularReportEndpoint)
		regularReport.POST("/policy", CreateRegularReportEndpoint)
		regularReport.PUT("/policy/:id", UpdateRegularReportEndpoint)
		regularReport.DELETE("/policy/:id", DeleteRegularReportEndpoint)

		// 查看报表
		regularReport.GET("/report", FindRegularReportLogEndpoint)
		regularReport.GET("/report/download", DownloadRegularReportLogEndpoint)
	}

	// 指令控制
	commandControl := e.Group("/command-control")
	{
		// 指令策略
		commandControl.GET("/strategy/paging", CommandStrategyPagingEndpoint)
		commandControl.POST("/strategy", CommandStrategyCreatEndpoint)
		commandControl.DELETE("/strategy/:id", CommandStrategyDeleteEndpoint)
		commandControl.PUT("/strategy/:id", CommandStrategyUpdateEndpoint)
		commandControl.POST("/strategy-status", CommandStrategyStatusEndpoint)

		// 指令集
		commandControl.POST("/set", CommandSetCreateEndpoint)
		commandControl.PUT("/set/:id", CommandSetUpdateEndpoint)
		commandControl.DELETE("/set/:id", CommandSetDeleteEndpoint)
		commandControl.GET("/set/paging", CommandSetPagingEndpoint)

		// 指令内容
		commandControl.POST("/content", CommandContentCreateEndpoint)
		commandControl.PUT("/content", CommandContentUpdateEndpoint)
		commandControl.DELETE("/content", CommandContentDeleteEndpoint)
		commandControl.GET("/content", CommandContentPagingEndpoint)

		// 指令策略关联指令内容
		commandControl.GET("/strategy-content/:id", CommandStrategyContentPagingEndpoint)
		commandControl.POST("/strategy-content", CommandStrategyContentUpdateEndpoint)

		// 指令策略关联指令集
		commandControl.GET("/strategy-set", CommandStrategySetPagingEndpoint)
		commandControl.GET("/strategy-relate-set/:id", CommandStrategyRelateSetEndpoint)
		commandControl.POST("/strategy-set", CommandStrategyRelateSetUpdateEndpoint)

		// 新建时获取所有关联的主机
		commandControl.GET("/strategy-all-asset", CommandStrategyAllAssetEndpoint)
		commandControl.GET("/strategy-asset/:id", CommandStrategyAssetPagingEndpoint)
		commandControl.GET("/strategy-relate-asset/:id", CommandStrategyRelateAssetEndpoint)
		commandControl.POST("/strategy-asset", CommandStrategyRelateAssetUpdateEndpoint)

		// 新建时获取所有可关联的主机组
		commandControl.GET("/strategy-all-asset-group", CommandStrategyAllAssetGroupEndpoint)
		commandControl.GET("/strategy-asset-group/:id", CommandStrategyAssetGroupPagingEndpoint)
		commandControl.GET("/strategy-relate-asset-group/:id", CommandStrategyRelateAssetGroupEndpoint)
		commandControl.POST("/strategy-asset-group", CommandStrategyRelateAssetGroupUpdateEndpoint)

		// 新建时获取所有可关联的用户
		commandControl.GET("/strategy-all-user", CommandStrategyAllUserEndpoint)
		commandControl.GET("/strategy-user/:id", CommandStrategyUserPagingEndpoint)
		commandControl.GET("/strategy-relate-user/:id", CommandStrategyRelateUserEndpoint)
		commandControl.POST("/strategy-user", CommandStrategyRelateUserUpdateEndpoint)

		// 新建时获取所有可关联的用户组
		commandControl.GET("/strategy-all-user-group", CommandStrategyAllUserGroupEndpoint)
		commandControl.GET("/strategy-user-group/:id", CommandStrategyUserGroupPagingEndpoint)
		commandControl.GET("/strategy-relate-user-group/:id", CommandStrategyRelateUserGroupEndpoint)
		commandControl.POST("/strategy-user-group", CommandStrategyRelateUserGroupUpdateEndpoint)

	}

	credentials := e.Group("/credentials")
	{
		credentials.GET("", CredentialAllEndpoint) //	主机查询、新增,权限也用到此api
		credentials.GET("/paging", CredentialPagingEndpoint)
		credentials.POST("", CredentialCreateEndpoint)
		credentials.PUT("/:id", CredentialUpdateEndpoint)
		credentials.DELETE("/:id", CredentialDeleteEndpoint)
		credentials.GET("/:id", CredentialGetEndpoint)
		credentials.POST("/:id/change-owner", CredentialChangeOwnerEndpoint)
	}

	// TODO 权限控制
	// 每次根据存储的策略id、设备账号id、用户id去实时查信息进行展示
	operateAuths := e.Group("/operate-auths")
	{
		operateAuths.GET("", OperateAuthPagingEndpoint)
		operateAuths.POST("", OperateAuthCreateEndpoint)
		operateAuths.PUT("/:id", OperateAuthUpdateEndpoint)
		operateAuths.DELETE("/:id", OperateAuthDeleteEndpoint)
		operateAuths.POST("/change-button-state", OperateAuthChangeButtonStateEndpoint)

		// 新建时获取关联数据
		operateAuths.GET("/create-relate-user", OperateAuthCreateRelateUserEndpoint)
		operateAuths.GET("/create-relate-user-group", OperateAuthCreateRelateUserGroupEndpoint)
		operateAuths.GET("/create-relate-asset", OperateAuthCreateRelateAssetEndpoint)
		operateAuths.GET("/create-relate-asset-group", OperateAuthCreateRelateAssetGroupEndpoint)
		operateAuths.GET("/create-relate-application", OperateAuthCreateRelateApplicantEndpoint)

		// 点击关联时获取所有资源数据与已关联的资源数据
		operateAuths.GET("/update-relate-user-all", OperateAuthUpdateRelateUserAllEndpoint)
		operateAuths.GET("/update-relate-user", OperateAuthUpdateRelateUserEndpoint)
		operateAuths.GET("/update-relate-user-group-all", OperateAuthUpdateRelateUserGroupAllEndpoint)
		operateAuths.GET("/update-relate-user-group", OperateAuthUpdateRelateUserGroupEndpoint)
		operateAuths.GET("/update-relate-asset-all", OperateAuthUpdateRelateAssetAllEndpoint)
		operateAuths.GET("/update-relate-asset", OperateAuthUpdateRelateAssetEndpoint)
		operateAuths.GET("/update-relate-asset-group-all", OperateAuthUpdateRelateAssetGroupAllEndpoint)
		operateAuths.GET("/update-relate-asset-group", OperateAuthUpdateRelateAssetGroupEndpoint)
		operateAuths.GET("/update-relate-application", OperateAuthUpdateRelateApplicationEndpoint)
		operateAuths.GET("/update-relate-application-all", OperateAuthUpdateRelateApplicantAllEndpoint)

		// 点击关联时，更新关联数据
		operateAuths.POST("/update-resource-relate", OperateAuthUpdateResourceRelateEndpoint)
	}

	hostOperate := e.Group("/host-operate")
	{
		hostOperate.GET("", HostOperatePagingEndpoint)
		hostOperate.GET("/graphical", HostOperateGraphicalPagingEndpoint)
		hostOperate.GET("/character", HostOperateCharacterPagingEndpoint)
		hostOperate.GET("/collect", HostOperateCollectPagingEndpoint)
		hostOperate.GET("/recent-use", GetRecentUsedPassport)
		hostOperate.POST("/:id/collect", HostOperateCollectEndpoint)
		hostOperate.POST("/:id/connect-test", HostOperateConnectTestEndpoint)

		newCommand := hostOperate.Group("/command")
		{
			newCommand.GET("/asset", CommandAssetEndpoint)
			newCommand.GET("/paging", NewCommandPagingEndpoint)
			newCommand.POST("", NewCommandCreateEndpoint)
			newCommand.PUT("/:id", NewCommandUpdateEndpoint)
			newCommand.DELETE("/:id", NewCommandDeleteEndpoint)
			newCommand.GET("/:id", NewCommandGetEndpoint)
		}
	}

	// 密码管理
	password := e.Group("/passwd")
	{
		// 身份验证
		password.POST("/auth", PasswordAuthEndpoint)
		// 是否存在导出密码验证
		password.GET("/export-auth", HasExportPasswordEndpoint)
		// 导出密码验证
		password.POST("/export-auth", PasswordExportAuthEndpoint)
		// 密码查看
		view := password.Group("/view")
		{
			view.GET("/paging", PasswordViewPagingEndpoint)
			view.GET("/:id", PasswordViewGetEndpoint)
			view.GET("/export", PasswordViewExportEndpoint)
		}

		//改密策略
		policy := password.Group("/policy")
		{
			policy.GET("/paging", PasswordPolicyPagingEndpoint)
			policy.POST("", PasswordPolicyCreateEndpoint)
			policy.PUT("/:id", PasswordPolicyUpdateEndpoint)
			// 获取编辑信息
			policy.GET("/:id", PasswordPolicyGetEndpoint)
			policy.DELETE("/:id", PasswordPolicyDeleteEndpoint)
			policy.POST("/:id", PasswordPolicyRunNow) // 立即执行
			// 获取所有ssh设备
			policy.GET("/create-relate-asset", PasswordPolicyGetAllSshDeviceEndpoint)
			// 获取所有设备组
			policy.GET("/create-relate-asset-group", PasswordPolicyGetAllDeviceGroupEndpoint)
			// 关联设备
			policy.GET("/:id/relate-asset", PasswordPolicyRelateAssetEndpoint)
			policy.PUT("/:id/relate-asset", PasswordPolicyUpdateRelateAssetEndpoint)
			// 关联设备组
			policy.GET("/:id/relate-asset-group", PasswordPolicyRelateAssetGroupEndpoint)
			policy.PUT("/:id/relate-asset-group", PasswordPolicyUpdateRelateAssetGroupEndpoint)
		}

		// 改密记录
		record := password.Group("/record")
		{
			record.GET("/paging", PasswordRecordPagingEndpoint)
			// 改密统计
			record.GET("/statistics", PasswordRecordStatisticsEndpoint)
			// 改密统计详情
			record.GET("/statistics-detail", PasswordRecordStatisticsDetailEndpoint)
		}
	}

	authReportForms := e.Group("/auth-report-form")
	{
		authReportForms.GET("/asset-auth/paging", AssetAuthPagingEndpoint)
		authReportForms.GET("/asset-auth/export", AssetAuthExportEndpoint)
		authReportForms.GET("/app-auth/paging", AppAuthPagingEndpoint)
		authReportForms.GET("/app-auth/export", AppAuthExportEndpoint)
	}

	appOperate := e.Group("/app-operate")
	{
		appOperate.GET("", AppOperatePagingEndpoint)
		appOperate.GET("/collect", AppOperateCollectListEndpoint)
		appOperate.POST("/:id/collect", AppOperateCollectEndpoint)
		// 最近常用
		appOperate.GET("/recent-use", GetAppOperateRecentUsedEndpoint)
	}

	resourceSharers := e.Group("/resource-sharers")
	{
		resourceSharers.GET("", RSGetSharersEndPoint)
		resourceSharers.POST("/remove-resources", ResourceRemoveByUserIdAssignEndPoint)
		resourceSharers.POST("/add-resources", ResourceAddByUserIdAssignEndPoint)
	}

	policyConfigs := e.Group("/policy-configs")
	{
		policyConfigs.GET("", PolicyGetEndpoint)
		policyConfigs.PUT("", PolicyconfigUpdateEndpoint)
		policyConfigs.POST("/send-test-mail", SendTestMail)
	}

	// 登录日志
	loginLogs := e.Group("/login-logs")
	{
		loginLogs.GET("/paging", LoginLogPagingEndpoint)
		loginLogs.GET("/export", LoginLogExportEndpoint)
	}

	// 运维日志
	devopsLogs := e.Group("/devops-logs")
	{
		devopsLogs.GET("/paging", DevopsLogEndpoint)
		devopsLogs.GET("/export", DevopsLogExportEndpoint)
	}

	// 操作日志
	operateLogs := e.Group("/operate-logs")
	{
		operateLogs.GET("/paging", OperateLogPagingEndpoint)
		operateLogs.GET("/export", OperateLogExportEndpoint)
	}
	// 系统告警
	systemAlerts := e.Group("/system-alerts")
	{
		systemAlerts.GET("/paging", SystemAlertPagingEndpoint)
		systemAlerts.GET("/export", SystemAlertExportEndpoint)
	}

	// 操作告警
	operateAlerts := e.Group("/operate-alerts")
	{
		operateAlerts.GET("/paging", OperateAlertPagingEndpoint)
		operateAlerts.GET("/export", OperateAlertExportEndpoint)
	}

	// 审批日志
	workOrderLogs := e.Group("/work-order-logs")
	{
		workOrderLogs.GET("/paging", GetWorkOrderLogEndpoint)
		workOrderLogs.GET("/:id", GetWorkOrderLogDetailEndpoint)
		workOrderLogs.GET("/asset/:id", NewWorkOrderLogAssetInfoEndPoint)
		workOrderLogs.GET("/log/:id", NewWorkOrderLogApproveInfoEndPoint)
		workOrderLogs.GET("/export", ExportWorkOrderLogEndpoint)
	}

	// 审计备份 :已更新
	auditBackup := e.Group("/audit-backup")
	{
		// 查看已存在备份
		auditBackup.GET("/paging", AuditBackupFilePagingEndpoint)
		// 导出备份
		auditBackup.GET("/export", AuditBackupExportEndpoint)
		// 删除备份
		auditBackup.DELETE("/:file", AuditBackupDeleteEndpoint)
		// 立即备份
		auditBackup.POST("/now", AuditBackupEndpoint)
	}

	// 系统配置
	sysConfigs := e.Group("/sys-configs")
	{
		// 系统时间
		sysConfigs.GET("/sys-time", SysTimeGetEndpoint)
		sysConfigs.PUT("/sys-time/set-time", SysTimeSetTimePutEndpoint)
		sysConfigs.PUT("/sys-time/sync-time", SysTimeSyncTimePutEndpoint)
		sysConfigs.PUT("/sys-time/auto-sync-time", SysTimeAutoSyncTimePutEndpoint)

		// 认证配置
		//			RADIUS
		sysConfigs.GET("/auth-config/radius", AuthConfigRadiusGetEndpoint)
		sysConfigs.PUT("/auth-config/radius", AuthConfigRadiusUpdateEndpoint)
		//			LDAP/AD
		sysConfigs.GET("/auth-config/ldap-ad", AuthConfigLdapAdPagingEndpoint)
		sysConfigs.GET("/auth-config/ldap-ad/:id", AuthConfigLdapAdGetEndpoint)
		sysConfigs.POST("/auth-config/ldap-ad", AuthConfigLdapAdCreateEndpoint)
		sysConfigs.PUT("/auth-config/ldap-ad/:id", AuthConfigLdapAdUpdateEndpoint)
		sysConfigs.DELETE("/auth-config/ldap-ad/:id", AuthConfigLdapAdDeleteEndpoint)
		sysConfigs.POST("/auth-config/ldap-ad/sync-account", AuthConfigLdapAdSyncAccountEndpoint)
		// 指纹认证
		sysConfigs.GET("/auth-config/fingerprint", AuthConfigFingerprintGetEndpoint)
		sysConfigs.PUT("/auth-config/fingerprint", AuthConfigFingerprintUpdateEndpoint)

		// 外发配置
		//			邮件配置
		sysConfigs.GET("/out-send/mail", OutSendMailGetEndpoint)
		sysConfigs.PUT("/out-send/mail", OutSendMailUpdateEndpoint)
		sysConfigs.POST("/out-send/mail/send-test-mail", OutSendMailSendTestMailEndpoint)
		// 			短信配置
		sysConfigs.GET("/out-send/sms", OutSendSmsGetEndpoint)
		sysConfigs.PUT("/out-send/sms", OutSendSmsUpdateEndpoint)
		sysConfigs.POST("/out-send/sms/send-test-sms", OutSendSmsSendTestSmsEndpoint)
		// 			SNMP配置
		sysConfigs.GET("/out-send/snmp", OutSendSnmpGetEndpoint)
		sysConfigs.PUT("/out-send/snmp", OutSendSnmpUpdateEndpoint)
		//			SYSLOG配置
		sysConfigs.GET("/out-send/syslog", OutSendSyslogGetEndpoint)
		sysConfigs.PUT("/out-send/syslog", OutSendSyslogUpdateEndpoint)
		// 安全配置
		// 			登录锁定配置
		sysConfigs.GET("/security/login-lock", LoginLockConfigGetEndpoint)
		sysConfigs.PUT("/security/login-lock", LoginLockConfigUpdateEndpoint)
		//			密码策略配置
		sysConfigs.GET("/security/password-policy", PasswordConfigGetEndpoint)
		sysConfigs.PUT("/security/password-policy", PasswordConfigUpdateEndpoint)
		// 			登录会话配置
		sysConfigs.GET("/security/login-session", SecurityLoginSessionGetEndpoint)
		sysConfigs.PUT("/security/login-session", SecurityLoginSessionUpdateEndpoint)
		// 			远程管理主机
		sysConfigs.GET("/security/remote-manage-host", RemoteManageHostGetEndpoint)
		sysConfigs.PUT("/security/remote-manage-host", RemoteManageHostUpdateEndpoint)
		// 			导出密码配置
		sysConfigs.PUT("/security/export-password", ExportPasswordConfigEndpoint)
		// HA集群配置
		sysConfigs.GET("/ha-cluster", HaClusterGetEndpoint)
		sysConfigs.PUT("/ha-cluster", HaClusterUpdateEndpoint)

		// 界面配置
		sysConfigs.GET("/ui-config", UiConfigGetEndpoint)
		sysConfigs.PUT("/ui-config", UiConfigUpdateEndpoint)
		// 告警配置
		// 			系统性能
		sysConfigs.GET("/alarm/system-performance-form", SystemPerformanceFormGetEndpoint)
		sysConfigs.GET("/alarm/sys-performance", AlarmSysPerformanceGetEndpoint)
		sysConfigs.PUT("/alarm/sys-performance", AlarmSysPerformanceUpdateEndpoint)
		//			系统访问量
		sysConfigs.GET("/alarm/system-access-form", SystemAccessFormGetEndpoint)
		sysConfigs.GET("/alarm/sys-access", AlarmSysAccessGetEndpoint)
		sysConfigs.PUT("/alarm/sys-access", AlarmSysAccessUpdateEndpoint)
		// 策略配置
		//		指令审批策略
		sysConfigs.GET("/command-policy/config", CommandStrategyConfigEndpoint)
		sysConfigs.PUT("/command-policy/config", CommandStrategyConfigUpdateEndpoint)
		// 		访问工单策略
		sysConfigs.GET("/work-order", WorkOrderSettingPagingEndPoint)
		sysConfigs.PUT("/work-order", WorkOrderSettingUpdateEndPoint)
		// 		策略优先级
		sysConfigs.GET("/policy-priority", CommandStrategyPriorityEndpoint)
		sysConfigs.PUT("/policy-priority", CommandStrategyPriorityUpdateEndpoint)
		// 审计备份
		sysConfigs.GET("/audit-backup", AuditBackupGetEndpoint)    // 系统设置-审计备份
		sysConfigs.PUT("/audit-backup", AuditBackupUpdateEndpoint) // 系统设置-审计备份
		// 存储空间，时间限制
		sysConfigs.GET("/storage", CapacityConfigGetEndpoint)  // 系统设置-存储空间
		sysConfigs.PUT("/storage", CapacityConfigEditEndpoint) // 系统设置-存储空间
		// 默认磁盘空间配置
		sysConfigs.GET("/default-disk", DefaultDiskConfigGetEndpoint)  // 系统设置-默认磁盘空间
		sysConfigs.PUT("/default-disk", DefaultDiskConfigEditEndpoint) // 系统设置-默认磁盘空间
		// 扩展配置
		sysConfigs.GET("/extend", ExtendConfigGetEndpoint)           // 系统设置-扩展配置
		sysConfigs.POST("/extend", ExtendConfigCreateEndpoint)       // 系统设置-扩展配置
		sysConfigs.PUT("/extend/:id", ExtendConfigUpdateEndpoint)    // 系统设置-扩展配置
		sysConfigs.DELETE("/extend/:id", ExtendConfigDeleteEndpoint) // 系统设置-扩展配置
	}

	// 网络配置
	networkConfigs := e.Group("/network-configs")
	{
		// 网络配置
		networkConfigs.GET("/interface-config", NetworkConfigGetEndpoint)            //系统设置-网络配置菜单-api权限表
		networkConfigs.PUT("/interface-config/:name", NetworkConfigUpdateEndpoint)   //系统设置-网络配置菜单-api权限表
		networkConfigs.POST("/interface-config/:name", NetworkConfigRestartEndpoint) //系统设置-网络配置菜单-api权限表
		networkConfigs.GET("/dns-config", DnsConfigGetEndpoint)                      //系统设置-dns配置菜单-api权限表
		networkConfigs.PUT("/dns-config", DnsConfigUpdateEndpoint)                   //系统设置-dns配置菜单-api权限表

		//静态路由
		networkConfigs.GET("/static-route", GetStaticRoute)                           // 系统设置-静态路由菜单-api权限表
		networkConfigs.POST("/static-route", CreateStaticRoute)                       // 系统设置-静态路由菜单-api权限表
		networkConfigs.DELETE("/static-route/:ip", DeleteStaticRoute)                 // 系统设置-静态路由菜单-api权限表
		networkConfigs.PUT("/static-route-edit/:destinationAddress", EditStaticRoute) // 系统设置-静态路由菜单-api权限表

		// 网络诊断
		networkConfigs.POST("/network/detection", NetworkDetectionPutEndpoint) // 系统设置-网络诊断菜单-api权限表
	}

	// 系统维护
	sysMaintains := e.Group("/sys-maintains")
	{
		// 系统版本信息
		sysMaintains.GET("/version", SysVersionGetEndpoint) // 系统维护-系统版本信息
		// 系统升级
		sysMaintains.POST("/upgrade", SysUpgradeGetEndpoint) // 系统维护-系统升级

		// 系统利用率
		sysMaintains.GET("/usage", SysUsageGetEndpoint) // 系统维护-系统利用率

		// 许可管理
		sysMaintains.GET("/license-management", LicenseManagementGetEndpoint)
		sysMaintains.GET("/license-management/down-license", LicenseManagementDownLicenseEndpoint)
		sysMaintains.POST("/license-management/import-license", LicenseManagementImportLicenseEndpoint)

		// 配置备份
		sysMaintains.GET("/backup-paging", BackupFilePagingEndpoint)    // 系统维护-查询列表
		sysMaintains.POST("/backup-create", BackupCreateEndpoint)       // 系统维护-创建备份
		sysMaintains.POST("/backup-restore", BackupRestoreEndpoint)     // 系统维护-还原备份
		sysMaintains.DELETE("/backup/:name", BackupDeleteEndpoint)      // 系统维护-删除备份
		sysMaintains.GET("/backup-export", BackupDownloadEndpoint)      // 系统维护-导出备份文件
		sysMaintains.POST("/backup-import", BackupRestoreLocalEndpoint) // 系统维护-导入备份文件

		// 系统工具
		sysMaintains.GET("/reboot", SysRebootEndpoint)     // 系统维护-重启
		sysMaintains.GET("/shutdown", SysShutdownEndpoint) // 系统设置-关机
		sysMaintains.POST("/reset", SysRestoreEndpoint)    // 系统设置-恢复出厂设置
	}

	e.GET("/properties", PropertyGetEndpoint)            // 系统设置菜单-api权限表
	e.PUT("/properties", PropertyUpdateEndpoint)         // 系统设置菜单-api权限表
	e.GET("/sys-logs-level", SysLogsLevelGetEndpoint)    // 系统设置-系统日志配置菜单-api权限表
	e.PUT("/sys-logs-level", SysLogsLevelUpdateEndpoint) // 系统设置-系统日志配置菜单-api权限表
	e.POST("/webconfig", WebConfigUpdateEndpoint)        // 系统设置-web配置-api权限表
	e.POST("/httpsstatus/:status", HttpsUpdateEndpoint)  // 系统设置-web配置-api权限表
	e.GET("/httpsstatus", HttpsGetEndpoint)              // 系统设置-web配置-api权限表
	e.GET("/webconfig", WebConfigGetEndpoint)            // 系统设置-web配置-api权限表
	e.PUT("/operation-mode", ModifyOperationMode)        // 系统设置-修改系统运行模式

	// 新的工单审批路由
	workOrderNew := e.Group("/work-order")
	{
		// 申请人工单
		workOrderNew.POST("", NewWorkOrderCreateEndPoint)
		workOrderNew.PUT("/apply/:id", NewWorkOrderUpdateEndPoint)
		workOrderNew.GET("/apply-paging", WorkOrderApplyListEndPoint)               // 申请人工单列表
		workOrderNew.GET("/apply-relate", NewWorkOrderGetRelateDeviceEndPoint)      // 获取工单可关联的设备
		workOrderNew.GET("/apply-related/:id", NewWorkOrderHadRelateDeviceEndPoint) // 获取工单已关联的设备
		workOrderNew.POST("/apply-relate", NewWorkOrderRelateDeviceEndPoint)        // 关联设备
		workOrderNew.POST("/apply-submit/:id", NewWorkOrderSubmitEndPoint)          // 提交工单
		workOrderNew.POST("/apply-cancel/:id", NewWorkOrderCancelEndPoint)          // 撤销工单

		// 审批人工单
		workOrderNew.GET("/approve-paging", WorkOrderApproveListEndPoint) // 审批人工单列表
		workOrderNew.POST("/approve", WorkOrderApproveEndPoint)           // 审批
		workOrderNew.POST("/close", WorkOrderCloseOrCancelEndpoint)       // 关闭工单

		// 工单详情
		workOrderNew.GET("/details/:id", NewWorkOrderDetailEndPoint)   // 工单详情
		workOrderNew.GET("/assets/:id", NewWorkOrderAssetInfoEndPoint) // 工单关联的设备详情
		workOrderNew.GET("/log/:id", NewWorkOrderApproveInfoEndPoint)  // 工单审批日志

		// 取消工单
		workOrderNew.GET("/cancel", CancelWorkOrder) // 取消工单
	}

	logFile := e.Group("/log")
	{
		// 日志列表
		logFile.GET("/syslog", SysLogPagingEndpoint)
		// 导出系统日志
		logFile.GET("/syslog/export", SysLogExportEndpoint)
		// 日志列表
		logFile.GET("/guacd", GuacdLogPagingEndpoint)
		// 导出guacd日志
		logFile.GET("/guacd/export", GuacdLogExportEndpoint)
	}

	// 这4个路由归属于个人中心菜单api权限下
	personal := e.Group("/personal")
	{
		personal.GET("/valid-time-password", ValidTimePasswordEndpoint)
		personal.PUT("/change-password", ChangePasswordEndpoint) // 已修改,个人中心修改密码
		personal.GET("/self-info", GetSelfInfoEndpoint)          // 已修改,个人中心获取个人信息
		personal.PUT("/change-info", ChangePersonalInformation)  // 已修改,个人中心修改个人信息
	}
	// 消息中心
	message := e.Group("/message")
	{
		message.GET("/count", MessageCountEndpoint)
		message.GET("/unread", MassageUnreadEndpoint)
		message.GET("/pending-approval", MessagePendingApprovalEndPoint)
		message.GET("/paging", MessagePagingEndpoint)
		message.POST("/batch-mark/:id", MessageMarkEndpoint)
		message.POST("/all-mark", MessageAllMarkEndpoint)
		message.DELETE("/clear", MessageClearEndpoint)
		message.DELETE("/:id", MessageDeleteEndpoint)
	}
	return e
}
