package dto

import (
	"tkbastion/server/utils"
)

type NewJobForSearch struct {
	Auto          string
	Name          string
	Department    string
	Content       string
	RunTimeType   string
	DepartmentIds []int64
}

type NewJobLogForSearch struct {
	Auto          string
	Name          string
	Department    string
	Content       string
	RunTimeType   string
	Result        string
	DepartmentIds []int64
}

type NewJobLog struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Department   string `json:"department"`
	Content      string `json:"content"`
	Type         string `json:"type"`
	Result       string `json:"result"`
	StartEndTime string `json:"startEndTime"`
}

type NewJobForPage struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	RunTimeType string `json:"runTimeType"`
	Department  string `json:"department"`
	Content     string `json:"content"`
	Info        string `json:"info"`
}

type NewJobLogForDetail struct {
	AssetName string `json:"assetName"`
	AssetIp   string `json:"assetIp"`
	Passport  string `json:"passport"`
	Port      string `json:"port"`
	Content   string `json:"content"`
	StartAt   string `json:"startAt"`
	EndAt     string `json:"endAt"`
	Result    string `json:"result"`
	ResultMsg string `json:"resultMsg"`
}

type NewJobLogForExport struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Ip         string `json:"ip"`
	Department string `json:"department"`
	Port       string `json:"port"`
	Passport   string `json:"passport"`
	Content    string `json:"content"`
	StartAt    string `json:"startAt"`
	EndAt      string `json:"endAt"`
	Result     string `json:"result"`
}

type NewJobForCreate struct {
	ID           string
	DepartmentID int64
	Department   string
	Name         string         `json:"name" binding:"required"`
	RunType      string         `json:"runType" binding:"required"`
	ShellName    string         `json:"shellName"`
	Command      string         `json:"command"`
	RunTimeType  string         `json:"runTimeType" binding:"required"`
	RunTime      utils.JsonTime `json:"runTime"`
	StartAt      utils.JsonTime `json:"startAt"`
	EndAt        utils.JsonTime `json:"endAt"`
	PeriodicType string         `json:"periodicType"`
	Periodic     int            `json:"periodic"`
	Info         string         `json:"info"`
	AssetIds     string         `json:"assetIds"`
	AssetGroupId string         `json:"assetGroupId"`
}

type NewJobForJson struct {
	ID           string
	DepartmentID int64
	Department   string
	Name         string `json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	RunType      string `json:"runType"`
	ShellName    string `json:"shellName"`
	Command      string `json:"command" validate:"required,max=128,alpha"`
	RunTimeType  string `json:"runTimeType" binding:"required"`
	RunTime      string `json:"runTime"`
	StartAt      string `json:"startAt"`
	EndAt        string `json:"endAt"`
	PeriodicType string `json:"periodicType"`
	Periodic     int    `json:"periodic"`
	Info         string `json:"info" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
	AssetIds     string `json:"assetIds"`
	AssetGroupId string `json:"assetGroupId"`
}

func (n NewJobForJson) ToNewJobForCreate() NewJobForCreate {
	return NewJobForCreate{
		ID:           n.ID,
		DepartmentID: n.DepartmentID,
		Department:   n.Department,
		Name:         n.Name,
		RunType:      n.RunType,
		ShellName:    n.ShellName,
		Command:      n.Command,
		RunTimeType:  n.RunTimeType,
		RunTime:      utils.StringToJSONTime(n.RunTime),
		StartAt:      utils.StringToJSONTime(n.StartAt),
		EndAt:        utils.StringToJSONTime(n.EndAt),
		PeriodicType: n.PeriodicType,
		Periodic:     n.Periodic,
		Info:         n.Info,
		AssetIds:     n.AssetIds,
		AssetGroupId: n.AssetGroupId,
	}
}

type NewJobForUpdate struct {
	ID           string         `json:"id" binding:"required"`
	Name         string         `json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	RunType      string         `json:"runType" binding:"required"`
	ShellName    string         `json:"shellName"`
	Command      string         `json:"command" validate:"required,max=128,alpha"`
	RunTimeType  string         `json:"runTimeType" binding:"required"`
	RunTime      utils.JsonTime `json:"runTime"`
	StartAt      utils.JsonTime `json:"startAt"`
	EndAt        utils.JsonTime `json:"endAt"`
	PeriodicType string         `json:"periodicType"`
	Periodic     int            `json:"periodic"`
	Info         string         `json:"info" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
}

type NewJobForJsonUpdate struct {
	ID           string `json:"id" binding:"required"`
	Name         string `json:"name" binding:"required"`
	RunType      string `json:"runType" binding:"required"`
	ShellName    string `json:"shellName"`
	Command      string `json:"command"`
	RunTimeType  string `json:"runTimeType" binding:"required"`
	RunTime      string `json:"runTime"`
	StartAt      string `json:"startAt"`
	EndAt        string `json:"endAt"`
	PeriodicType string `json:"periodicType"`
	Periodic     int    `json:"periodic"`
	Info         string `json:"info"`
}

func (n NewJobForJsonUpdate) ToNewJobForUpdate() NewJobForUpdate {
	return NewJobForUpdate{
		ID:           n.ID,
		Name:         n.Name,
		RunType:      n.RunType,
		ShellName:    n.ShellName,
		Command:      n.Command,
		RunTimeType:  n.RunTimeType,
		RunTime:      utils.StringToJSONTime(n.RunTime),
		StartAt:      utils.StringToJSONTime(n.StartAt),
		EndAt:        utils.StringToJSONTime(n.EndAt),
		PeriodicType: n.PeriodicType,
		Periodic:     n.Periodic,
		Info:         n.Info,
	}
}
