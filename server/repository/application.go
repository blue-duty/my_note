package repository

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"tkbastion/pkg/global"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"gorm.io/gorm"
)

type ApplicationRepositoryNew struct {
	baseRepository
}

func (r *ApplicationRepositoryNew) Find(ctx context.Context, s *dto.ApplicationForSearch) (o []dto.ApplicationForPage, err error) {
	var applications []model.NewApplication
	db := r.GetDB(ctx).Model(&model.NewApplication{}).Where("department_id in (?)", s.Departments)
	if s.Name != "" {
		db = db.Where("name like ?", "%"+s.Name+"%")
	}
	if s.AppSerName != "" {
		db = db.Where("app_ser_name like ?", "%"+s.AppSerName+"%")
	}
	if s.ProgramName != "" {
		db = db.Where("program_name like ?", "%"+s.ProgramName+"%")
	}
	if s.Department != "" {
		db = db.Where("department like ?", "%"+s.Department+"%")
	}
	if s.Auto != "" {
		db = db.Where("name like ? or app_ser_name like ? or program_name like ? or department like ? or info like ? or path like ?", "%"+s.Auto+"%", "%"+s.Auto+"%", "%"+s.Auto+"%", "%"+s.Auto+"%", "%"+s.Auto+"%", "%"+s.Auto+"%")
	}
	err = db.Find(&applications).Error
	o = make([]dto.ApplicationForPage, len(applications))
	for i, application := range applications {
		o[i] = dto.ApplicationForPage{
			ID:           application.ID,
			Name:         application.Name,
			Info:         application.Info,
			Department:   application.Department,
			ProgramName:  application.ProgramName,
			DepartmentID: application.DepartmentID,
			ProgramId:    application.ProgramID,
			AppSerId:     application.AppSerId,
			Param:        application.Param,
			AppSerName:   application.AppSerName,
		}
	}
	return
}

func updateApplictionById(db *gorm.DB, id string) (err error) {
	var application model.NewApplication
	err = db.Model(&model.NewApplication{}).Where("id = ?", id).First(&application).Error
	if err != nil {
		db.Rollback()
		return
	}
	var appser model.NewApplicationServer
	var program model.NewProgram
	err = db.Model(&model.NewApplicationServer{}).Where("id = ?", application.AppSerId).First(&appser).Error
	if err != nil {
		db.Rollback()
		return
	}
	err = db.Model(&model.NewProgram{}).Where("id = ?", application.ProgramID).First(&program).Error
	if err != nil {
		db.Rollback()
		return
	}
	// 更新application
	application.AppSerName = appser.Name
	application.ProgramName = program.Name
	application.IP = appser.IP
	application.Port = appser.Port
	application.Passport = appser.Passport
	application.Password = appser.Password
	err = db.Save(&application).Error
	return
}

func (r *ApplicationRepositoryNew) Insert(ctx context.Context, s *dto.ApplicationForInsert) (err error) {
	var program model.NewProgram
	err = r.GetDB(ctx).Model(&model.NewProgram{}).Where("id = ?", s.ProgramId).First(&program).Error
	if err != nil {
		return
	}
	department, err := departmentRepository.FindById(s.DepartmentID)
	if err != nil {
		return
	}
	appserver, err := newApplicationServerRepository.FindById(context.TODO(), program.Aid)
	if err != nil {
		return
	}
	var application model.NewApplication
	application.ID = utils.UUID()
	application.Name = s.Name
	application.Info = s.Info
	application.DepartmentID = s.DepartmentID
	application.ProgramID = s.ProgramId
	application.ProgramName = program.Name
	application.Department = department.Name
	application.Param = s.Param
	application.AppSerName = appserver.Name
	application.IP = appserver.IP
	application.Port = appserver.Port
	application.Passport = appserver.Passport
	application.Password = appserver.Password
	application.AppSerId = appserver.ID
	application.Path = program.Path
	application.Created = utils.NowJsonTime()
	err = r.GetDB(ctx).Create(&application).Error
	return
}

func (r *ApplicationRepositoryNew) Update(ctx context.Context, s *dto.ApplicationForUpdate) (err error) {
	var program model.NewProgram
	err = r.GetDB(ctx).Model(&model.NewProgram{}).Where("id = ?", s.ProgramId).First(&program).Error
	if err != nil {
		return
	}
	var appser model.NewApplicationServer
	err = r.GetDB(ctx).Model(&model.NewApplicationServer{}).Where("id = ?", program.Aid).First(&appser).Error
	if err != nil {
		return
	}
	// 获取department
	department, err := departmentRepository.FindById(s.DepartmentID)
	if err != nil {
		return
	}

	var application model.NewApplication
	application.ID = s.ID
	application.Name = s.Name
	application.Info = s.Info
	application.ProgramID = s.ProgramId
	application.ProgramName = program.Name
	application.AppSerName = appser.Name
	application.DepartmentID = s.DepartmentID
	application.Department = department.Name
	application.AppSerId = appser.ID
	application.Param = s.Param
	err = r.GetDB(ctx).Model(&model.NewApplication{}).Where("id = ?", s.ID).Updates(utils.Struct2MapByStructTag(application)).Error
	return
}

func deleteById(db *gorm.DB, id string) (err error) {
	err = db.Where("application_id = ?", id).Delete(&model.ApplicationAuthReportForm{}).Error
	if err != nil {
		return
	}
	err = db.Where("id = ?", id).Delete(&model.NewApplication{}).Error
	if err != nil {
		return
	}
	return
}

func (r *ApplicationRepositoryNew) Delete(ctx context.Context, id string) (err error) {
	db := r.GetDB(ctx).Begin()
	err = deleteById(db, id)
	if err != nil {
		db.Rollback()
		return
	}
	db.Commit()
	return
}

// DeleteByDepartmentId delete by department id
func (r *ApplicationRepositoryNew) DeleteByDepartmentId(ctx context.Context, id []int64) ([]model.NewApplication, error) {
	var applications []model.NewApplication
	err := r.GetDB(ctx).Where("department_id in (?)", id).Find(&applications).Error
	if err != nil {
		return nil, err
	}
	db := r.GetDB(ctx).Begin()
	for _, application := range applications {
		err = deleteById(db, application.ID)
		if err != nil {
			db.Rollback()
			return nil, err
		}
	}
	db.Commit()
	return applications, nil
}

// DeleteMore delete more
func (r *ApplicationRepositoryNew) DeleteMore(ctx context.Context, ids []string) (err error) {
	db := r.GetDB(ctx).Begin()
	for _, id := range ids {
		err = db.Delete(&model.NewApplication{
			ID: id,
		}).Error
		if err != nil {
			db.Rollback()
			return
		}
	}
	db.Commit()
	return
}

func (r *ApplicationRepositoryNew) FindByName(todo context.Context, name string) (o model.NewApplication, err error) {
	err = r.GetDB(todo).Model(&model.NewApplication{}).Where("name = ?", name).First(&o).Error
	return
}

func (r *ApplicationRepositoryNew) FindByNameId(todo context.Context, name string, id string) (o model.NewApplication, err error) {
	fmt.Println(name, id)
	err = r.GetDB(todo).Model(&model.NewApplication{}).Where("name = ? and id != ?", name, id).First(&o).Error
	return
}

func (r *ApplicationRepositoryNew) FindById(todo context.Context, id string) (o model.NewApplication, err error) {
	err = r.GetDB(todo).Model(&model.NewApplication{}).Where("id = ?", id).First(&o).Error
	return
}

func (r *ApplicationRepositoryNew) FindDetailById(todo context.Context, id string) (o dto.ApplicationForDetail, err error) {
	var application model.NewApplication
	err = r.GetDB(todo).Model(&model.NewApplication{}).Where("id = ?", id).First(&application).Error
	if err != nil {
		return
	}
	o = dto.ApplicationForDetail{
		Name:        application.Name,
		Info:        application.Info,
		Department:  application.Department,
		ProgramName: application.ProgramName,
		Param:       application.Param,
		AppSerName:  application.AppSerName,
		Created:     application.Created.Format("2006-01-02 15:04:05"),
	}
	return
}

func (r *ApplicationRepositoryNew) FindPolicyById(todo context.Context, id string) (o []dto.ApplicationForPolicy, err error) {
	var p []model.OperateAuth
	err = r.GetDB(todo).Model(&model.OperateAuth{}).Where("relate_app like ?", "%"+id+"%").Find(&p).Error
	if err != nil {
		return
	}
	for _, v := range p {
		dp, err := DepChainName(v.DepartmentId)
		if err != nil {
			return o, err
		}
		uids := strings.Split(v.RelateUser, ",")
		for _, uid := range uids {
			user, err := userNewRepository.FindById(uid)
			if err != nil {
				return o, err
			}
			o = append(o, dto.ApplicationForPolicy{
				Username:   user.Username,
				Nickname:   user.Nickname,
				Department: user.DepartmentName,
				Policy:     v.Name + "[" + dp[:len(dp)-1] + "]",
			})
		}
	}
	return
}

func (r *ApplicationRepositoryNew) FindAppCountByDepartmentIds(todo context.Context, depIds []int64) (count int64, err error) {
	err = r.GetDB(todo).Model(&model.NewApplication{}).Where("department_id in (?)", depIds).Count(&count).Error
	return
}

func (r *ApplicationRepositoryNew) FindByDepartmentIds(todo context.Context, ids []int64) (o []model.NewApplication, err error) {
	err = r.GetDB(todo).Model(&model.NewApplication{}).Where("department_id in (?)", ids).Find(&o).Error
	return
}

func (r *ApplicationRepositoryNew) GetApplicantByDepartmentIds(todo context.Context, ids []int64) (o []model.NewApplication, err error) {
	err = r.GetDB(todo).Model(&model.NewApplication{}).Where("department_id in (?)", ids).Find(&o).Error
	return
}

func (r *ApplicationRepositoryNew) GetAppCountByDepartmentId(todo context.Context, depId int64) (count int64, err error) {
	err = r.GetDB(todo).Model(&model.NewApplication{}).Where("department_id = ?", depId).Count(&count).Error
	return
}

func (r *ApplicationRepositoryNew) GetApplicationByIds(todo context.Context, arr []string) (o []model.NewApplication, err error) {
	err = r.GetDB(todo).Model(&model.NewApplication{}).Where("id in (?)", arr).Find(&o).Error
	return
}

func (r *ApplicationRepositoryNew) GetApplicationById(todo context.Context, id string) (o model.NewApplication, err error) {
	err = r.GetDB(todo).Model(&model.NewApplication{}).Where("id = ?", id).First(&o).Error
	return
}

func (r *ApplicationRepositoryNew) GetApplicationIdsAndAofs(todo context.Context, ids []string, aofs dto.AppOperateForSearch) (o []model.NewApplication, err error) {
	db := r.GetDB(todo).Model(&model.NewApplication{}).Where("id in (?) and department_id in (?)", ids, aofs.Departments)
	if aofs.Name != "" {
		db = db.Where("name like ?", "%"+aofs.Name+"%")
	} else if aofs.Program != "" {
		db = db.Where("program_name like ?", "%"+aofs.Program+"%")
	} else if aofs.AppServer != "" {
		db = db.Where("app_ser_name like ?", "%"+aofs.AppServer+"%")
	} else if aofs.Auto != "" {
		db = db.Where("name like ? or program_name like ? or app_ser_name like ? or department like ? or path like ? or info like ?", "%"+aofs.Auto+"%", "%"+aofs.Auto+"%", "%"+aofs.Auto+"%", "%"+aofs.Auto+"%", "%"+aofs.Auto+"%", "%"+aofs.Auto+"%")
	}
	err = db.Find(&o).Error
	return
}

func (r *ApplicationRepositoryNew) GetApplicationForSessionByProgramId(todo context.Context, programId string) (o dto.ApplicationForSession, err error) {
	var applications model.NewApplication
	err = r.GetDB(todo).Model(&model.NewApplication{}).Where("program_id = ?", programId).First(&applications).Error
	if err != nil {
		return
	}
	o.Name = applications.ProgramName
	o.IP = applications.IP
	o.Port = applications.Port
	o.Passport = applications.Passport

	origData, err := base64.StdEncoding.DecodeString(applications.Password)
	if err != nil {
		return
	}
	decryptedCBC, err := utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
	if err != nil {
		return
	}
	o.Password = string(decryptedCBC)
	o.Param = applications.Param
	o.Path = applications.Path

	return
}

// GetApplicationForSessionByProgram 通过程序获取服务器账号和密码
func (r *ApplicationRepositoryNew) GetApplicationForSessionByProgram(todo context.Context, program string) (o dto.ApplicationForSession, err error) {
	var pro model.NewProgram
	err = r.GetDB(todo).Model(&model.NewProgram{}).Where("id = ?", program).First(&pro).Error
	if err != nil {
		return
	}
	var appSer model.NewApplicationServer
	err = r.GetDB(todo).Model(&model.NewApplicationServer{}).Where("id = ?", pro.Aid).First(&appSer).Error
	if err != nil {
		return
	}

	origData, err := base64.StdEncoding.DecodeString(appSer.Password)
	if err != nil {
		return
	}
	decryptedCBC, err := utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
	if err != nil {
		return
	}

	o.Name = pro.Name
	o.IP = appSer.IP
	o.Port = appSer.Port
	o.Passport = appSer.Passport
	o.Password = string(decryptedCBC)
	o.Param = ""
	o.Path = ""

	return
}

func (r *ApplicationRepositoryNew) GetApplicationForExport(todo context.Context, ids []int64) (o []dto.ApplicationForExport, err error) {
	var applications []model.NewApplication
	err = r.GetDB(todo).Model(&model.NewApplication{}).Where("department_id in (?)", ids).Find(&applications).Error
	if err != nil {
		return
	}
	for _, v := range applications {
		o = append(o, dto.ApplicationForExport{
			Name:        v.Name,
			Info:        v.Info,
			Department:  v.Department,
			ProgramName: v.ProgramName,
			Param:       v.Param,
			AppSerName:  v.AppSerName,
			IP:          v.IP,
			Port:        v.Port,
			Path:        v.Path,
			Passport:    v.Passport,
			Password:    v.Password,
		})
	}
	return
}
