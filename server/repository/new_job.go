package repository

import (
	"context"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"tkbastion/pkg/constant"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/labstack/gommon/log"

	"gorm.io/gorm"

	"github.com/pkg/errors"
)

type JobRepositoryNew struct {
	baseRepository
}

func (j *JobRepositoryNew) FindAll(ctx context.Context, jfs *dto.NewJobForSearch) (jobs []dto.NewJobForPage, err error) {
	var job []model.NewJob
	if len(jfs.DepartmentIds) == 0 {
		return
	}
	db := j.GetDB(ctx).Where("department_id in (?)", jfs.DepartmentIds)
	if jfs.Auto != "" {
		var rtt []string
		if strings.Contains("手动执行", jfs.Auto) {
			rtt = append(rtt, "Manual")
		}
		if strings.Contains("定时执行", jfs.Auto) {
			rtt = append(rtt, "Scheduled")
		}
		if strings.Contains("周期执行", jfs.Auto) {
			rtt = append(rtt, "Periodic")
		}
		db = db.Where("name LIKE ? or department like ? or run_type in (?) or run_time_type like ? or periodic_type like ? or info like ? or shell_path like ? or command like ? or periodic like ?", "%"+jfs.Auto+"%", "%"+jfs.Auto+"%", rtt, "%"+jfs.Auto+"%", "%"+jfs.Auto+"%", "%"+jfs.Auto+"%", "%"+jfs.Auto+"%", "%"+jfs.Auto+"%", "%"+jfs.Auto+"%")
	} else if jfs.Name != "" {
		db = db.Where("name LIKE ?", "%"+jfs.Name+"%")
	} else if jfs.Department != "" {
		db = db.Where("department like ?", "%"+jfs.Department+"%")
	} else if jfs.Content != "" {
		db = db.Where("command LIKE ?", "%"+jfs.Content+"%").Or("shell_path LIKE ?", "%"+jfs.Content+"%")
	} else if jfs.RunTimeType != "" {
		var runTimeType []string
		if strings.Contains("手动执行", jfs.RunTimeType) {
			runTimeType = append(runTimeType, "Manual")
		}
		if strings.Contains("定时执行", jfs.RunTimeType) {
			runTimeType = append(runTimeType, "Scheduled")
		}
		if strings.Contains("周期执行", jfs.RunTimeType) {
			runTimeType = append(runTimeType, "Periodic")
		}
		db = db.Where("run_time_type in (?)", runTimeType)
	}
	err = db.Find(&job).Error

	jobs = make([]dto.NewJobForPage, len(job))
	for i, v := range job {
		jobs[i] = dto.NewJobForPage{
			ID:         v.ID,
			Name:       v.Name,
			Info:       v.Info,
			Department: v.Department,
		}
		if v.RunType == "command" {
			jobs[i].Content = "命令: " + v.Command
		} else {
			jobs[i].Content = "脚本: " + v.ShellName[:strings.LastIndex(v.ShellName, ".")]
		}
		if v.RunTimeType == "Manual" {
			jobs[i].RunTimeType = "手动执行"
		} else if v.RunTimeType == "Scheduled" {
			jobs[i].RunTimeType = "定时执行,时间:" + v.RunTime.Format("2006-01-02 15:04:05")
		} else {
			var s string
			switch v.PeriodicType {
			case "Day":
				s = strconv.Itoa(v.Periodic) + "天"
			case "Week":
				s = strconv.Itoa(v.Periodic) + "周"
			case "Month":
				s = strconv.Itoa(v.Periodic) + "月"
			case "Minute":
				s = strconv.Itoa(v.Periodic) + "分钟"
			case "Hour":
				s = strconv.Itoa(v.Periodic) + "小时"
			}
			jobs[i].RunTimeType = "周期执行,周期:" + s + "时间:" + v.StartAt.Format("2006-01-02 15:04:05") + "到" + v.EndAt.Format("2006-01-02 15:04:05")
		}
	}

	return
}

func (j *JobRepositoryNew) GetAll(ctx context.Context) (jobs []model.NewJob, err error) {
	err = j.GetDB(ctx).Find(&jobs).Error
	return
}

// GetJobForEdit 通过id获取job编辑信息
func (j *JobRepositoryNew) GetJobForEdit(ctx context.Context, id string) (dto.NewJobForUpdate, error) {
	var job dto.NewJobForUpdate
	var nj model.NewJob
	err := j.GetDB(ctx).Where("id = ?", id).First(&nj).Error
	if err != nil {
		return job, err
	}
	if nj.RunType == "command" {
		job.ShellName = ""
		job.Command = nj.Command
	} else {
		job.Command = ""
		job.ShellName = nj.ShellName[:strings.LastIndex(nj.ShellName, ".")]
	}
	job.ID = nj.ID
	job.Name = nj.Name
	job.RunType = nj.RunType
	job.RunTimeType = nj.RunTimeType
	job.RunTime = nj.RunTime
	job.StartAt = nj.StartAt
	job.EndAt = nj.EndAt
	job.PeriodicType = nj.PeriodicType
	job.Periodic = nj.Periodic
	job.Info = nj.Info
	return job, nil
}

func (j *JobRepositoryNew) Create(ctx context.Context, job *dto.NewJobForCreate) error {
	if job.RunType == "command" {
		job.ShellName = ""
	} else {
		job.Command = ""
	}
	if job.RunTimeType == "Manual" {
		job.RunTime = utils.JsonTime{}
		job.StartAt = utils.JsonTime{}
		job.EndAt = utils.JsonTime{}
		job.Periodic = 0
		job.PeriodicType = ""
	} else if job.RunTimeType == "Scheduled" {
		job.StartAt = utils.JsonTime{}
		job.EndAt = utils.JsonTime{}
		job.Periodic = 0
		job.PeriodicType = ""
	} else {
		job.RunTime = utils.JsonTime{}
	}

	db := j.GetDB(ctx).Begin()
	jb := &model.NewJob{
		ID:           job.ID,
		Name:         job.Name,
		RunType:      job.RunType,
		RunTimeType:  job.RunTimeType,
		RunTime:      job.RunTime,
		StartAt:      job.StartAt,
		EndAt:        job.EndAt,
		PeriodicType: job.PeriodicType,
		Periodic:     job.Periodic,
		Info:         job.Info,
		DepartmentID: job.DepartmentID,
		Department:   job.Department,
	}

	if job.RunType == "command" {
		jb.Command = job.Command
	} else {
		jb.ShellName = job.ShellName
	}
	if err := db.Create(jb).Error; err != nil {
		db.Rollback()
		return err
	}
	assetids := utils.IdHandle(job.AssetIds)
	assetGroupIds := utils.IdHandle(job.AssetGroupId)
	for _, v := range assetids {
		if err := db.Create(&model.NewJobWithAssets{
			ID:      utils.UUID(),
			Jid:     job.ID,
			AssetId: v,
		}).Error; err != nil {
			db.Rollback()
			return err
		}
	}
	for _, v := range assetGroupIds {
		if err := db.Create(&model.NewJobWithAssetGroups{
			ID:           utils.UUID(),
			Jid:          job.ID,
			AssetGroupId: v,
		}).Error; err != nil {
			db.Rollback()
			return err
		}
	}
	db.Commit()
	return nil
}

func (j *JobRepositoryNew) Update(ctx context.Context, job *dto.NewJobForUpdate) error {
	if job.RunType == "command" {
		job.ShellName = ""
	} else {
		job.Command = ""
	}
	if job.RunTimeType == "Manual" {
		job.RunTime = utils.JsonTime{}
		job.StartAt = utils.JsonTime{}
		job.EndAt = utils.JsonTime{}
		job.Periodic = 0
		job.PeriodicType = ""
	} else if job.RunTimeType == "Scheduled" {
		job.StartAt = utils.JsonTime{}
		job.EndAt = utils.JsonTime{}
		job.Periodic = 0
		job.PeriodicType = ""
	} else {
		job.RunTime = utils.JsonTime{}
	}

	return j.GetDB(ctx).Model(&model.NewJob{}).Where("id = ?", job.ID).Updates(utils.Struct2MapByStructTag(model.NewJob{
		ID:           job.ID,
		Name:         job.Name,
		RunType:      job.RunType,
		ShellName:    job.ShellName,
		Command:      job.Command,
		RunTimeType:  job.RunTimeType,
		RunTime:      job.RunTime,
		StartAt:      job.StartAt,
		EndAt:        job.EndAt,
		Periodic:     job.Periodic,
		PeriodicType: job.PeriodicType,
		Info:         job.Info,
	})).Error
}

func deleteJodByJobId(db *gorm.DB, jobid string) error {
	var shellPath string
	if err := db.Model(&model.NewJob{}).Where("id = ?", jobid).Pluck("shell_name", &shellPath).Error; err != nil {
		return err
	}
	if err := db.Where("jid = ?", jobid).Delete(&model.NewJobWithAssets{}).Error; err != nil {
		return err
	}
	if err := db.Where("jid = ?", jobid).Delete(&model.NewJobWithAssetGroups{}).Error; err != nil {
		return err
	}
	if err := db.Where("id = ?", jobid).Delete(&model.NewJob{}).Error; err != nil {
		return err
	}
	fmt.Println("shell", shellPath)
	if shellPath != "" {
		if err := os.Remove(path.Join(constant.ShellPath, shellPath)); err != nil {
			log.Error("删除任务脚本失败", err)
		}
	}
	return nil
}

func (j *JobRepositoryNew) Delete(ctx context.Context, id string) error {
	db := j.GetDB(ctx).Begin()
	if err := deleteJodByJobId(db, id); err != nil {
		db.Rollback()
		return err
	}
	db.Commit()
	return nil
}

func (j *JobRepositoryNew) DeleteAll(ctx context.Context, ids []string) error {
	db := j.GetDB(ctx).Begin()
	for _, v := range ids {
		if err := deleteJodByJobId(db, v); err != nil {
			db.Rollback()
			return err
		}
	}
	db.Commit()
	return nil
}

func (j *JobRepositoryNew) FindByName(todo context.Context, name string) (model.NewJob, error) {
	var job model.NewJob
	err := j.GetDB(todo).Where("name = ?", name).First(&job).Error
	return job, err
}

func (j *JobRepositoryNew) FindByIdName(todo context.Context, name string, id string) (model.NewJob, error) {
	var job model.NewJob
	err := j.GetDB(todo).Where("name = ? and id != ?", name, id).First(&job).Error
	return job, err
}

func (j *JobRepositoryNew) FindById(todo context.Context, id string) (model.NewJob, error) {
	var job model.NewJob
	err := j.GetDB(todo).Where("id = ?", id).First(&job).Error
	return job, err
}

func (j *JobRepositoryNew) JobBindAssets(todo context.Context, jid string, pIds []string) error {
	db := j.GetDB(todo).Begin()
	// 1. delete and insert
	if err := db.Where("jid = ?", jid).Delete(&model.NewJobWithAssets{}).Error; err != nil {
		db.Rollback()
		return err
	}
	for _, pId := range pIds {
		if err := db.Create(&model.NewJobWithAssets{
			ID:      utils.UUID(),
			Jid:     jid,
			AssetId: pId,
		}).Error; err != nil {
			db.Rollback()
			return err
		}
	}
	return db.Commit().Error
}

func (j *JobRepositoryNew) JobBindAssetGroups(todo context.Context, jid string, pIds []string) error {
	db := j.GetDB(todo).Begin()
	// 1. delete and insert
	if err := db.Where("jid = ?", jid).Delete(&model.NewJobWithAssetGroups{}).Error; err != nil {
		db.Rollback()
		return err
	}
	for _, pId := range pIds {
		if err := db.Create(&model.NewJobWithAssetGroups{
			ID:           utils.UUID(),
			Jid:          jid,
			AssetGroupId: pId,
		}).Error; err != nil {
			db.Rollback()
			return err
		}
	}
	return db.Commit().Error
}

func (j *JobRepositoryNew) FindPassportIdsAndJobByJobId(todo context.Context, jid string) ([]string, model.NewJob, error) {
	var pids []string
	var assetGroups []string
	err := j.GetDB(todo).Table("new_job_with_assets").Select("asset_id").Where("jid = ?", jid).Scan(&pids).Error
	if err != nil {
		return nil, model.NewJob{}, err
	}
	err = j.GetDB(todo).Table("new_job_with_asset_groups").Select("asset_group_id").Where("jid = ?", jid).Scan(&assetGroups).Error
	if err != nil {
		return nil, model.NewJob{}, err
	}
	// 1. 通过设备组id获取设备id
	var assetIdsByGroup []string
	err = j.GetDB(todo).Table("asset_group_with_asset").Select("asset_id").Where("asset_group_id in (?)", assetGroups).Scan(&assetIdsByGroup).Error
	if err != nil {
		return nil, model.NewJob{}, err
	}

	var job model.NewJob
	err = j.GetDB(todo).Where("id = ?", jid).First(&job).Error

	// 2. 合并设备id
	pids = append(pids, assetIdsByGroup...)
	// 3. 去重
	assetIdsByGroup = utils.RemoveDuplicatesAndEmpty(assetIdsByGroup)

	// 查询其中的ssh设备并返回其id
	var sshAssetIds []string
	err = j.GetDB(todo).Table("pass_ports").Select("id").Where("id in (?) and protocol = ?", assetIdsByGroup, constant.SSH).Scan(&sshAssetIds).Error

	return sshAssetIds, job, err
}

// FindAssetGroupIdsByJobId 通过任务id获取设备组id
func (j *JobRepositoryNew) FindAssetGroupIdsByJobId(todo context.Context, jid string) ([]string, error) {
	var assetGroups []string
	err := j.GetDB(todo).Model(&model.NewJobWithAssetGroups{}).Select("asset_group_id").Where("jid = ?", jid).Scan(&assetGroups).Error
	if err != nil {
		return nil, err
	}
	return assetGroups, nil
}

func (j *JobRepositoryNew) FindAssetIdsByJobId(todo context.Context, jid string) ([]string, error) {
	var assetIds []string
	err := j.GetDB(todo).Model(&model.NewJobWithAssets{}).Select("asset_id").Where("jid = ?", jid).Scan(&assetIds).Error
	if err != nil {
		return nil, err
	}
	return assetIds, nil
}

// WriteRunLog 编写运行日志
func (j *JobRepositoryNew) WriteRunLog(todo context.Context, log model.NewJobLog) error {
	return j.GetDB(todo).Create(&log).Error
}

// FindRunLog 获取运行日志
func (j *JobRepositoryNew) FindRunLog(todo context.Context, njlfs dto.NewJobLogForSearch) ([]dto.NewJobLog, error) {
	var logs []model.NewJobLog
	if len(njlfs.DepartmentIds) == 0 {
		return nil, errors.New("部门id不能为空")
	}
	db := j.GetDB(todo).Where("department_id in (?)", njlfs.DepartmentIds)
	if njlfs.Name != "" {
		db = db.Where("name like ?", "%"+njlfs.Name+"%")
	} else if njlfs.Content != "" {
		db = db.Where("command LIKE ?", "%"+njlfs.Content+"%")
	} else if njlfs.Department != "" {
		db = db.Where("department like ?", "%"+njlfs.Department+"%")
	} else if njlfs.Result != "" {
		db = db.Where("result like ?", "%"+njlfs.Result+"%")
	} else if njlfs.RunTimeType != "" {
		db = db.Where("type like ?", "%"+njlfs.RunTimeType+"%")
	} else if njlfs.Auto != "" {
		db = db.Where("id like ? or name like ? or command like ? or department like ? or result like ? or type like ? or start_at like ? or end_at like ?", "%"+njlfs.Auto+"%", "%"+njlfs.Auto+"%", "%"+njlfs.Auto+"%", "%"+njlfs.Auto+"%", "%"+njlfs.Auto+"%", "%"+njlfs.Auto+"%", "%"+njlfs.Auto+"%", "%"+njlfs.Auto+"%")
	}
	err := db.Order("end_at desc").Find(&logs).Error
	if err != nil {
		return nil, err
	}
	var logDtos []dto.NewJobLog
	for _, jobLog := range logs {
		logDtos = append(logDtos, dto.NewJobLog{
			ID:           jobLog.ID,
			Name:         jobLog.Name,
			Content:      jobLog.Command,
			Department:   jobLog.Department,
			Result:       jobLog.Result,
			Type:         jobLog.Type,
			StartEndTime: jobLog.StartAt.Format("2006-01-02 15:04:05") + " -- " + jobLog.EndAt.Format("2006-01-02 15:04:05"),
		})
	}
	return logDtos, nil
}

// FindRunLogById 通过id获取运行日志
func (j *JobRepositoryNew) FindRunLogById(todo context.Context, id string) (*dto.NewJobLogForDetail, error) {
	var log model.NewJobLog
	err := j.GetDB(todo).First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &dto.NewJobLogForDetail{
		Content:   log.Command,
		AssetName: log.AssetName,
		AssetIp:   log.AssetIp,
		Port:      strconv.Itoa(log.Port),
		Passport:  log.Passport,
		Result:    log.Result,
		ResultMsg: log.ResultInfo,
		StartAt:   log.StartAt.Format("2006-01-02 15:04:05"),
		EndAt:     log.EndAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// FindRunLogForExport 获取日志导出数据
func (j *JobRepositoryNew) FindRunLogForExport(todo context.Context, id string) (dto.NewJobLogForExport, error) {
	var log model.NewJobLog
	err := j.GetDB(todo).First(&log, id).Error
	if err != nil {
		return dto.NewJobLogForExport{}, err
	}
	return dto.NewJobLogForExport{
		ID:         log.ID,
		Name:       log.Name,
		Ip:         log.AssetIp,
		Department: log.Department,
		Port:       strconv.Itoa(log.Port),
		Passport:   log.Passport,
		Content:    log.Command,
		StartAt:    log.StartAt.Format("2006-01-02 15:04:05"),
		EndAt:      log.EndAt.Format("2006-01-02 15:04:05"),
		Result:     log.Result,
	}, nil
}

func (j *JobRepositoryNew) DeleteByDepartmentId(todo context.Context, ids []int64) error {
	jids := make([]string, 0)
	err := j.GetDB(todo).Model(&model.NewJob{}).Select("id").Where("department_id in (?)", ids).Scan(&jids).Error
	if err != nil {
		return err
	}

	err = j.DeleteAll(todo, jids)
	if err != nil {
		return err
	}

	return nil
}

func (j *JobRepositoryNew) DeleteJobLogByDepartmentId(todo context.Context, ids []int64) error {
	return j.GetDB(todo).Where("department_id in (?)", ids).Delete(&model.NewJobLog{}).Error
}
