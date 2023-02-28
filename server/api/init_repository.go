package api

import (
	"tkbastion/pkg/service"
	"tkbastion/server/repository"

	"gorm.io/gorm"

	s "tkbastion/server/service"
)

var (
	userNewRepository              *repository.UserNewRepository
	userGroupNewRepository         *repository.UserGroupNewRepository
	roleRepository                 *repository.RoleRepository
	userStrategyRepository         *repository.UserStrategyRepository
	userGroupMemberRepository      *repository.UserGroupMemberRepository
	resourceSharerRepository       *repository.ResourceSharerRepository
	regularReportRepository        *repository.RegularReportRepository
	newAssetRepository             *repository.AssetRepositoryNew
	newAssetGroupRepository        *repository.AssetGroupRepositoryNew
	newApplicationRepository       *repository.ApplicationRepositoryNew
	newApplicationServerRepository *repository.ApplicationServerRepositoryNew
	newjobRepository               *repository.JobRepositoryNew
	newSessionRepository           *repository.SessionRepositoryNew
	systemTypeRepository           *repository.SystemTypeRepository
	credentialRepository           *repository.CredentialRepository
	propertyRepository             *repository.PropertyRepository
	numRepository                  *repository.NumRepository
	jobRepository                  *repository.JobRepository
	jobLogRepository               *repository.JobLogRepository
	loginLogRepository             *repository.LoginLogRepository
	menuRepository                 *repository.MenuRepository
	casbinRuleRepository           *repository.CasbinRuleRepository
	operateLogRepository           *repository.OperateLogRepository
	identityConfigRepository       *repository.IdentityConfigRepository
	operateAlarmLogRepository      *repository.OperateAlarmLogRepository
	//strategyRepository         *repository.StrategyRepository
	policyConfigRepository *repository.PolicyConfigRepository
	orderLogRepository     *repository.OrderLogRepository
	messageRepository      *repository.MessageRepository
	recipientsRepository   *repository.RecipientsRepository

	appSessionRepository  *repository.AppSessionRepository
	commandsNewRepository *repository.CommandRepositoryNew

	commandStrategyRepository      *repository.CommandStrategyRepository
	commandSetRepository           *repository.CommandSetRepository
	commandContentRepository       *repository.CommandContentRepository
	commandRelevanceRepository     *repository.CommandRelevanceRepository
	userAccessStatisticsRepository *repository.UserAccessStatisticsRepository
	workOrderNewRepository         *repository.WorkOrderNewRepository
	workOrderApprovalLogRepository *repository.WorkOrderApprovalLogRepository
	workOrderAssetRepository       *repository.WorkOrderAssetRepository
	jobExportRepository            *repository.JobExportRepository
	storageRepositoryNew           *repository.StorageRepositoryNew

	departmentRepository          *repository.DepartmentRepository
	operateAuthRepository         *repository.OperateAuthRepository
	hostOperateRepository         *repository.HostOperateRepository
	userCollecteRepository        *repository.UserCollecteRepository
	ldapAdAuthRepository          *repository.LdapAdAuthRepository
	assetAuthReportFormRepository *repository.AssetAuthReportFormRepository
	appAuthReportFormRepository   *repository.AppAuthReportFormRepository
	operateReportRepository       *repository.OperateReportRepository

	jobService            *service.JobService
	snmpService           *service.SnmpService
	propertyService       *service.PropertyService
	userService           *service.UserService
	identityService       *service.IdentityService
	storageNewService     *s.StorageServiceNew
	mailService           *service.MailService
	numService            *service.NumService
	credentialService     *service.CredentialService
	menuService           *service.MenuService
	backupService         *service.BackupService
	commandControlService *service.CommandControlService
	authenticationService *service.AuthenticationService
	messageService        *s.MessageService

	departmentService  *service.DepartmentService
	roleService        *service.RoleService
	operateAuthService *service.OperateAuthService
	newJobService      *s.NewJobService
	systemTypeService  *service.SystemTypeService
	sysConfigService   *s.SysConfigService
	sysMaintainService *s.SysMaintainService
)

func InitRepository(db *gorm.DB) {
	userNewRepository = repository.NewUserNewRepository(db)
	userGroupNewRepository = repository.NewUserGroupNewRepository(db)
	roleRepository = repository.NewRoleRepository(db)
	userStrategyRepository = repository.NewUserStrategyRepository(db)
	resourceSharerRepository = repository.NewResourceSharerRepository(db)
	regularReportRepository = repository.NewRegularReportRepository(db)
	newAssetRepository = new(repository.AssetRepositoryNew)
	newAssetGroupRepository = new(repository.AssetGroupRepositoryNew)
	systemTypeRepository = new(repository.SystemTypeRepository)
	newApplicationRepository = new(repository.ApplicationRepositoryNew)
	newApplicationServerRepository = new(repository.ApplicationServerRepositoryNew)
	newjobRepository = new(repository.JobRepositoryNew)
	newSessionRepository = new(repository.SessionRepositoryNew)
	appSessionRepository = new(repository.AppSessionRepository)
	operateReportRepository = new(repository.OperateReportRepository)
	commandsNewRepository = new(repository.CommandRepositoryNew)
	storageRepositoryNew = new(repository.StorageRepositoryNew)
	userAccessStatisticsRepository = new(repository.UserAccessStatisticsRepository)
	appAuthReportFormRepository = new(repository.AppAuthReportFormRepository)
	operateAlarmLogRepository = new(repository.OperateAlarmLogRepository)
	credentialRepository = repository.NewCredentialRepository(db)
	propertyRepository = repository.NewPropertyRepository(db)
	numRepository = repository.NewNumRepository(db)
	identityConfigRepository = repository.NewIdentityConfigRepository(db)
	jobRepository = repository.NewJobRepository(db)
	jobLogRepository = repository.NewJobLogRepository(db)
	loginLogRepository = repository.NewLoginLogRepository(db)
	menuRepository = repository.NewMenuRepository(db)
	casbinRuleRepository = repository.NewCasbinRuleRepository(db)
	operateLogRepository = repository.NewOperateLogRepository(db)
	//strategyRepository = repository.NewStrategiesRepository(db)
	userGroupMemberRepository = repository.NewUserGroupMemberRepository(db)
	workOrderNewRepository = repository.NewWorkOrderNewRepository(db)
	workOrderApprovalLogRepository = repository.NewWorkOrderApprovalLogRepository(db)
	workOrderAssetRepository = repository.NewWorkOrderAssetRepository(db)
	jobExportRepository = repository.NewJobExportRepository(db)

	commandRelevanceRepository = repository.NewCommandRelevanceRepository(db)
	commandContentRepository = repository.NewCommandContentRepository(db)
	commandSetRepository = repository.NewCommandSetRepository(db)
	commandStrategyRepository = repository.NewCommandStrategyRepository(db)
	orderLogRepository = repository.NewOrderLogRepository(db)
	messageRepository = repository.NewMessageRepository(db)
	recipientsRepository = repository.NewRecipientsRepository(db)

	departmentRepository = repository.NewDepartmentRepository(db)
	operateAuthRepository = repository.NewOperateAuthRepository(db)
	hostOperateRepository = repository.NewHostOperateRepository(db)
	userCollecteRepository = repository.NewUserCollecteRepository(db)
	ldapAdAuthRepository = repository.NewLdapAdAuthRepository(db)
	assetAuthReportFormRepository = repository.NewAssetAuthReportFormRepository(db)
}

func InitService() {
	newJobService = s.NewNewJobService(newjobRepository, newAssetRepository, propertyRepository, regularReportRepository, newSessionRepository,
		loginLogRepository, userAccessStatisticsRepository, operateReportRepository, operateAlarmLogRepository,
		messageRepository, userNewRepository, appSessionRepository)
	departmentService = service.NewDepartmentService(departmentRepository)
	roleService = service.NewRoleService(roleRepository)
	propertyService = service.NewPropertyService(propertyRepository)
	snmpService = service.NewSnmpService(propertyRepository)
	userService = service.NewUserService(userNewRepository, loginLogRepository)
	mailService = service.NewMailService(propertyRepository)
	numService = service.NewNumService(numRepository)
	s.AuditBackupSrv = s.NewAuditBackupService(workOrderNewRepository, operateLogRepository, loginLogRepository, hostOperateRepository, propertyRepository)
	identityService = service.NewIdentityConfigService(identityConfigRepository)
	credentialService = service.NewCredentialService(credentialRepository)
	menuService = service.NewMenuService(menuRepository)
	backupService = service.NewBackupService()
	commandControlService = service.NewCommandControlService(commandRelevanceRepository, commandSetRepository,
		commandContentRepository, userGroupMemberRepository, commandStrategyRepository, newAssetGroupRepository, newAssetRepository)
	authenticationService = service.NewAuthenticationService(userNewRepository, roleRepository)
	jobService = service.NewJobService(jobRepository, jobLogRepository, credentialRepository, propertyRepository, authenticationService, jobExportRepository, operateLogRepository)

	departmentService = service.NewDepartmentService(departmentRepository)
	operateAuthService = service.NewOperateAuthService(operateAuthRepository)

	systemTypeService = service.NewSystemTypeService(systemTypeRepository)
	messageService = s.NewMessageService()
	sysConfigService = s.NewSysConfigService(propertyRepository)
	sysMaintainService = s.NewSysMaintainService(propertyRepository)
	storageNewService = s.NewStorageServiceNew(storageRepositoryNew, propertyRepository)
}
