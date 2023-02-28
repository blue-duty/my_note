package dto

import "tkbastion/server/utils"

type CommandStrategyForCreat struct {
	ID             string         `json:"id" `
	Name           string         `json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	DepartmentId   int64          `json:"departmentId" `
	DepartmentName string         `json:"departmentName" `
	Level          string         `json:"level" `
	Action         string         `json:"action" `
	Status         string         `json:"status" `
	Description    string         `json:"description" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
	Priority       int64          `json:"priority" `
	AlarmByMessage bool           `json:"alarmByMessage" `
	AlarmByPhone   bool           `json:"alarmByPhone" `
	AlarmByEmail   bool           `json:"alarmByEmail" `
	IsPermanent    bool           `json:"isPermanent" `
	BeginValidTime utils.JsonTime `json:"beginValidTime" `
	EndValidTime   utils.JsonTime `json:"endValidTime" `
	Cmd            string         `json:"cmd" `
	UserId         string         `json:"userId" `
	UserGroupId    string         `json:"userGroupId" `
	AssetId        string         `json:"assetId" `
	AssetGroupId   string         `json:"assetGroupId" `
	CommandSetId   string         `json:"commandSetId" `
}
type CommandStrategyForUpdate struct {
	ID             string         `json:"id" `
	Name           string         `json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	Level          string         `json:"level" `
	Action         string         `json:"action" `
	Description    string         `json:"description" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
	Priority       int64          `json:"priority" `
	AlarmByMessage bool           `json:"alarmByMessage" `
	AlarmByEmail   bool           `json:"alarmByEmail" `
	AlarmByPhone   bool           `json:"alarmByPhone" `
	IsPermanent    bool           `json:"isPermanent" `
	BeginValidTime utils.JsonTime `json:"beginValidTime" `
	EndValidTime   utils.JsonTime `json:"endValidTime" `
}
