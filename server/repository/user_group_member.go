package repository

import (
	"gorm.io/gorm"
	"tkbastion/server/model"
)

type UserGroupMemberRepository struct {
	DB *gorm.DB
}

func NewUserGroupMemberRepository(db *gorm.DB) *UserGroupMemberRepository {
	userGroupMemberRepository = &UserGroupMemberRepository{DB: db}
	return userGroupMemberRepository
}

func (r UserGroupMemberRepository) FindUserIdsByUserGroupId(userGroupId string) (o []string, err error) {
	err = r.DB.Table("user_group_members").Select("user_id").Where("user_group_id = ?", userGroupId).Find(&o).Error
	return
}

func (r UserGroupMemberRepository) FindUserGroupIdsByUserId(userId string) (o []string, err error) {
	// 先查询用户所在的用户
	err = r.DB.Table("user_group_members").Select("user_group_id").Where("user_id = ?", userId).Find(&o).Error
	return
}

func (r UserGroupMemberRepository) Create(o *model.UserGroupMember) error {
	return r.DB.Create(o).Error
}

func (r UserGroupMemberRepository) DeleteByUserId(userId string) error {
	return r.DB.Where("user_id = ?", userId).Delete(&model.UserGroupMember{}).Error
}

func (r UserGroupMemberRepository) DeleteByUserGroupId(userGroupId string) error {
	return r.DB.Where("user_group_id = ?", userGroupId).Delete(&model.UserGroupMember{}).Error
}

func (r UserGroupMemberRepository) CountByUserGroupId(userGroupId string) (count int64, err error) {
	err = r.DB.Table("user_group_members").Where("user_group_id = ?", userGroupId).Count(&count).Error
	return
}

func (r UserGroupMemberRepository) Truncate() error {
	return r.DB.Exec("TRUNCATE TABLE user_group_members").Error
}
