package repository

var (
	userNewRepository         *UserNewRepository
	userGroupNewRepository    *UserGroupNewRepository
	roleRepository            *RoleRepository
	userStrategyRepository    *UserStrategyRepository
	userGroupMemberRepository *UserGroupMemberRepository
	resourceSharerRepository  *ResourceSharerRepository
	regularReportRepository   *RegularReportRepository
	credentialRepository      *CredentialRepository
	propertyRepository        *PropertyRepository
	numRepository             *NumRepository
	loginLogRepository        *LoginLogRepository
	menuRepository            *MenuRepository
	casbinRuleRepository      *CasbinRuleRepository
	operateLogRepository      *OperateLogRepository

	orderLogRepository  *OrderLogRepository
	jobexportRepository *JobExportRepository

	jobLogRepository *JobLogRepository
	jobRepository    *JobRepository

	commandStrategyRepository  *CommandStrategyRepository
	commandContentRepository   *CommandContentRepository
	commandSetRepository       *CommandSetRepository
	commandRelevanceRepository *CommandRelevanceRepository

	messageRepository              *MessageRepository
	recipientsRepository           *RecipientsRepository
	workOrderNewRepository         *WorkOrderNewRepository
	workOrderApprovalLogRepository *WorkOrderApprovalLogRepository
	workOrderAssetRepository       *WorkOrderAssetRepository

	departmentRepository           *DepartmentRepository
	operateAuthRepository          *OperateAuthRepository
	hostOperateRepository          *HostOperateRepository
	userCollecteRepository         *UserCollecteRepository
	ldapAdAuthRepository           *LdapAdAuthRepository
	newApplicationServerRepository = new(ApplicationServerRepositoryNew)
	assetAuthReportFormRepository  *AssetAuthReportFormRepository
)
