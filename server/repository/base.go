package repository

import (
	"context"
	"gorm.io/gorm"
	"tkbastion/pkg/constant"
	"tkbastion/server/env"
)

func SetupRepository(db *gorm.DB) {
	UserNewDao = NewUserNewRepository(db)
	UserStrategyDao = NewUserStrategyRepository(db)
	LoginLogDao = NewLoginLogRepository(db)
	PolicyConfigDao = NewPolicyConfigRepository(db)
	PropertyDao = NewPropertyRepository(db)
	IdentityConfigDao = NewIdentityConfigRepository(db)
	MessageDao = NewMessageRepository(db)
	RecipientsDao = NewRecipientsRepository(db)
	UserGroupMemberDao = NewUserGroupMemberRepository(db)
	RoleMenuDao = NewMenuRepository(db)
}

type baseRepository struct {
}

func (b *baseRepository) GetDB(c context.Context) *gorm.DB {
	db, ok := c.Value(constant.DB).(*gorm.DB)
	if !ok {
		return env.GetDB()
	}
	return db
}
