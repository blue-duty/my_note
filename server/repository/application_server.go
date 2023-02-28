package repository

import (
	"context"
	"encoding/base64"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"gorm.io/gorm"
)

type ApplicationServerRepositoryNew struct {
	baseRepository
}

func (r *ApplicationServerRepositoryNew) Find(ctx context.Context, s *dto.ApplicationServerForSearch) (o []dto.ApplicationServerForPage, err error) {
	if len(s.Departments) == 0 {
		return
	}
	var applicationServers []model.NewApplicationServer
	db := r.GetDB(ctx).Model(&model.NewApplicationServer{}).Where("department_id in (?)", s.Departments)
	if s.IP != "" {
		db = db.Where("ip like ?", "%"+s.IP+"%")
	} else if s.Name != "" {
		db = db.Where("name like ?", "%"+s.Name+"%")
	} else if s.Department != "" {
		db = db.Where("department like ?", "%"+s.Department+"%")
	} else if s.Auto != "" {
		db = db.Where("ip like ? or name like ? or department like ? or info like ? or type like ?", "%"+s.Auto+"%", "%"+s.Auto+"%", "%"+s.Auto+"%", "%"+s.Auto+"%", "%"+s.Auto+"%")
	}
	err = db.Find(&applicationServers).Error
	if err != nil {
		return
	}
	o = make([]dto.ApplicationServerForPage, len(applicationServers))
	for k, applicationServer := range applicationServers {
		o[k] = dto.ApplicationServerForPage{
			ID:           applicationServer.ID,
			Name:         applicationServer.Name,
			IP:           applicationServer.IP,
			Type:         applicationServer.Type,
			Department:   applicationServer.Department,
			DepartmentID: applicationServer.DepartmentID,
			Passport:     applicationServer.Passport,
			Info:         applicationServer.Info,
		}
	}
	return
}

// Insert 添加
func (r *ApplicationServerRepositoryNew) Insert(ctx context.Context, o *dto.ApplicationServerForInsert) (err error) {
	applicationServer := model.NewApplicationServer{
		ID:           utils.UUID(),
		Name:         o.Name,
		IP:           o.IP,
		Type:         o.Type,
		Port:         o.Port,
		DepartmentID: o.DepartmentID,
		Department:   o.Department,
		Passport:     o.Passport,
		Password:     o.Password,
		Info:         o.Info,
	}
	err = r.GetDB(ctx).Create(&applicationServer).Error
	return
}

// Update 修改
func (r *ApplicationServerRepositoryNew) Update(ctx context.Context, o *dto.ApplicationServerForUpdate) (err error) {
	if o.Info == "" {
		o.Info = " "
	}
	applicationServer := model.NewApplicationServer{
		ID:           o.ID,
		Name:         o.Name,
		IP:           o.IP,
		Port:         o.Port,
		Type:         o.Type,
		DepartmentID: o.DepartmentID,
		Department:   o.Department,
		Passport:     o.Passport,
		Info:         o.Info,
	}
	if o.Password != "" {
		encryptedCBC, err := utils.AesEncryptCBC([]byte(o.Password), global.Config.EncryptionPassword)
		if err != nil {
			log.Errorf("应用服务器密码加密失败: %v", err)
		}
		applicationServer.Password = base64.StdEncoding.EncodeToString(encryptedCBC)
	}

	// 更新应用服务器
	db := r.GetDB(ctx).Begin()
	appSerMap := utils.Struct2MapByStructTag(applicationServer)
	err = db.Model(&model.NewApplicationServer{}).Where("id = ?", o.ID).Updates(&appSerMap).Error
	if err != nil {
		db.Rollback()
		return
	}
	err = db.Model(&model.NewApplicationServer{}).Where("id = ?", o.ID).First(&applicationServer).Error
	if err != nil {
		db.Rollback()
		return
	}
	// 更新应用服务器关联的应用
	err = db.Model(&model.NewApplication{}).Where("app_ser_id = ?", o.ID).Updates(&model.NewApplication{
		AppSerName: applicationServer.Name,
		IP:         applicationServer.IP,
		Port:       applicationServer.Port,
		Passport:   applicationServer.Passport,
		Password:   applicationServer.Password,
	}).Error
	if err != nil {
		db.Rollback()
		return
	}
	db.Commit()
	return
}

// Delete 删除
func (r *ApplicationServerRepositoryNew) Delete(ctx context.Context, id string) (err error) {
	db := r.GetDB(ctx).Begin()
	err = deleteAppServerById(db, id)
	if err != nil {
		db.Rollback()
		return
	}
	db.Commit()
	return
}

// FindByName 查询名称是否存在
func (r *ApplicationServerRepositoryNew) FindByName(ctx context.Context, name string) (o *model.NewApplicationServer, err error) {
	err = r.GetDB(ctx).Model(&model.NewApplicationServer{}).Where("name = ?", name).First(&o).Error
	return
}

// FindByIp 查询ip是否存在
func (r *ApplicationServerRepositoryNew) FindByIp(ctx context.Context, ip string) (o *model.NewApplicationServer, err error) {
	err = r.GetDB(ctx).Model(&model.NewApplicationServer{}).Where("ip = ?", ip).First(&o).Error
	return
}

// FindByIdName 查询名称是否存在
func (r *ApplicationServerRepositoryNew) FindByIdName(ctx context.Context, id string, name string) (o *model.NewApplicationServer, err error) {
	err = r.GetDB(ctx).Model(&model.NewApplicationServer{}).Where("id != ? and name = ?", id, name).First(&o).Error
	return
}

func (r *ApplicationServerRepositoryNew) FindAppSerCountByDepIds(ctx context.Context, depIds []int64) (count int64, err error) {
	err = r.GetDB(ctx).Model(&model.NewApplicationServer{}).Where("department_id in ?", depIds).Count(&count).Error
	return
}

// FindProgramByName 查询程序名称是否存在
func (r *ApplicationServerRepositoryNew) FindProgramByName(ctx context.Context, aid string, name string) (o *model.NewProgram, err error) {
	err = r.GetDB(ctx).Model(&model.NewProgram{}).Where("aid = ?", aid).Where("name = ?", name).First(&o).Error
	return
}

func deleteAppServerById(db *gorm.DB, id string) (err error) {
	// 删除应用
	err = db.Model(&model.NewApplication{}).Where("app_ser_id = ?", id).Delete(&model.NewApplication{}).Error
	if err != nil {
		return
	}
	// 删除程序
	err = db.Model(&model.NewProgram{}).Where("aid = ?", id).Delete(&model.NewProgram{}).Error
	if err != nil {
		return
	}
	// 删除应用服务器
	err = db.Model(&model.NewApplicationServer{}).Where("id = ?", id).Delete(&model.NewApplicationServer{}).Error
	if err != nil {
		return
	}
	return
}

// DeleteMultiApplicationServer 删除多个应用服务器
func (r *ApplicationServerRepositoryNew) DeleteMultiApplicationServer(ctx context.Context, ids []string) (err error) {
	db := r.GetDB(ctx).Begin()
	for _, id := range ids {
		err = deleteAppServerById(db, id)
		if err != nil {
			db.Rollback()
			return
		}
	}
	db.Commit()
	return
}

// InsertProgram 添加程序
func (r *ApplicationServerRepositoryNew) InsertProgram(ctx context.Context, o *dto.NewProgramForInsert) (err error) {
	program := model.NewProgram{
		ID:   utils.UUID(),
		Aid:  o.Aid,
		Name: o.Name,
		Path: o.Path,
		Info: o.Info,
	}
	err = r.GetDB(ctx).Create(&program).Error
	return
}

// UpdateProgram 修改程序
func (r *ApplicationServerRepositoryNew) UpdateProgram(ctx context.Context, o *dto.NewProgramForUpdate) (err error) {
	program := model.NewProgram{
		ID:   o.ID,
		Name: o.Name,
		Path: o.Path,
		Info: o.Info,
		Aid:  o.Aid,
	}
	db := r.GetDB(ctx).Begin()
	err = db.Model(&model.NewProgram{}).Where("id = ?", o.ID).Updates(utils.Struct2MapByStructTag(program)).Error
	if err != nil {
		db.Rollback()
		return
	}
	err = db.Model(&model.NewProgram{}).Where("id = ?", o.ID).First(&program).Error
	if err != nil {
		db.Rollback()
		return
	}
	// 更新程序关联的应用
	err = db.Model(&model.NewApplication{}).Where("program_id = ?", o.ID).Updates(&model.NewApplication{
		ProgramName: program.Name,
	}).Error
	if err != nil {
		db.Rollback()
		return
	}
	db.Commit()
	return
}

// DeleteProgram 删除程序
func (r *ApplicationServerRepositoryNew) DeleteProgram(ctx context.Context, id string) (err error) {
	// 删除所有的应用
	db := r.GetDB(ctx).Begin()
	err = db.Model(&model.NewApplication{}).Where("program_id = ?", id).Delete(&model.NewApplication{}).Error
	if err != nil {
		db.Rollback()
		return
	}
	// 删除程序
	err = db.Model(&model.NewProgram{}).Where("id = ?", id).Delete(&model.NewProgram{}).Error
	if err != nil {
		db.Rollback()
		return
	}
	db.Commit()
	return
}

// DeleteMoreProgram 批量删除程序
func (r *ApplicationServerRepositoryNew) DeleteMoreProgram(ctx context.Context, ids []string) (err error) {
	// 删除所有的应用
	db := r.GetDB(ctx).Begin()
	for _, id := range ids {
		err = db.Model(&model.NewApplication{}).Where("program_id = ?", id).Delete(&model.NewApplication{}).Error
		if err != nil {
			db.Rollback()
			return
		}
	}
	// 删除程序
	for _, id := range ids {
		err = db.Model(&model.NewProgram{}).Where("id = ?", id).Delete(&model.NewProgram{}).Error
		if err != nil {
			db.Rollback()
			return
		}
	}
	db.Commit()
	return
}

// SearchProgramByAid search Program by application server id
func (r *ApplicationServerRepositoryNew) SearchProgramByAid(ctx context.Context, aid string) (o []*dto.NewProgramForPage, err error) {
	var programs []model.NewProgram
	err = r.GetDB(ctx).Model(&model.NewProgram{}).Where("aid = ?", aid).Find(&programs).Error
	if err != nil {
		return
	}
	o = make([]*dto.NewProgramForPage, len(programs))
	for i, program := range programs {
		o[i] = &dto.NewProgramForPage{
			ID:   program.ID,
			Name: program.Name,
			Path: program.Path,
			Info: program.Info,
		}
	}
	return
}

func (r *ApplicationServerRepositoryNew) FindById(todo context.Context, v string) (o *model.NewApplicationServer, err error) {
	err = r.GetDB(todo).Model(&model.NewApplicationServer{}).Where("id = ?", v).First(&o).Error
	return
}

func (r *ApplicationServerRepositoryNew) GetAppServerByDepId(todo context.Context, depId int64) (count int64, err error) {
	err = r.GetDB(todo).Model(&model.NewApplicationServer{}).Where("department_id = ?", depId).Count(&count).Error
	return
}

func (r *ApplicationServerRepositoryNew) FindProgramById(todo context.Context, id string) (o *model.NewProgram, err error) {
	err = r.GetDB(todo).Model(&model.NewProgram{}).Where("id = ?", id).First(&o).Error
	return
}

func (r *ApplicationServerRepositoryNew) FindByIpId(todo context.Context, ip string, id string) (o *model.NewApplicationServer, err error) {
	err = r.GetDB(todo).Model(&model.NewApplicationServer{}).Where("ip = ?", ip).Where("id != ?", id).First(&o).Error
	return
}

func (r *ApplicationServerRepositoryNew) FindApplicationServerById(todo context.Context, id string) (o *model.NewApplicationServer, err error) {
	err = r.GetDB(todo).Model(&model.NewApplicationServer{}).Where("id = ?", id).First(&o).Error
	return
}

func (r *ApplicationServerRepositoryNew) DeleteByDepartmentId(todo context.Context, ids []int64) ([]model.NewApplicationServer, error) {
	var appServers []model.NewApplicationServer
	err := r.GetDB(todo).Model(&model.NewApplicationServer{}).Where("department_id in (?)", ids).Find(&appServers).Error
	if err != nil {
		return nil, err
	}
	db := r.GetDB(todo).Begin()
	for _, appServer := range appServers {
		err = deleteAppServerById(db, appServer.ID)
		if err != nil {
			db.Rollback()
			return nil, err
		}
	}
	return appServers, db.Commit().Error
}
