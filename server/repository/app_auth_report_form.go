package repository

import (
	"context"
	"tkbastion/server/dto"
	"tkbastion/server/model"
)

type AppAuthReportFormRepository struct {
	baseRepository
}

func (r AppAuthReportFormRepository) Create(c context.Context, o *model.ApplicationAuthReportForm) (err error) {
	return r.GetDB(c).Create(o).Error
}

func (r AppAuthReportFormRepository) Find(c context.Context) (o []model.ApplicationAuthReportForm, err error) {
	err = r.GetDB(c).Find(&o).Error
	return
}

func (r AppAuthReportFormRepository) FindWithCondition(c context.Context, auto, appName, program, username string) (o []model.ApplicationAuthReportForm, err error) {
	db := r.GetDB(c).Table("application_auth_report_form").Select("application_auth_report_form.id, application_auth_report_form.app_ser_name, application_auth_report_form.app_name, application_auth_report_form.program_name, application_auth_report_form.username, application_auth_report_form.nickname, application_auth_report_form.operate_auth_id, application_auth_report_form.operate_auth_name")

	if appName != "" {
		db = db.Where("application_auth_report_form.app_name like ?", "%"+appName+"%")
	}

	if program != "" {
		db = db.Where("application_auth_report_form.program_name like ?", "%"+program+"%")
	}

	if "" != username {
		db = db.Where("application_auth_report_form.username like ?", "%"+username+"%")
	}

	if "" != auto {
		db = db.Where("application_auth_report_form.app_name like ? OR application_auth_report_form.program_name like ? OR application_auth_report_form.username like ? OR application_auth_report_form.nickname like ? OR application_auth_report_form.operate_auth_name like ?", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%")
	}

	err = db.Order("application_auth_report_form.id desc").Find(&o).Error
	if o == nil {
		o = make([]model.ApplicationAuthReportForm, 0)
	}
	return
}

// FindExportData 获取导出数据
func (r AppAuthReportFormRepository) FindExportData(c context.Context) (o []dto.ApplicationAuthReportForExport, err error) {
	db := r.GetDB(c).Table("application_auth_report_form").Select("application_auth_report_form.id,application_auth_report_form.app_ser_name, application_auth_report_form.app_name, application_auth_report_form.program_name, application_auth_report_form.username, application_auth_report_form.nickname, application_auth_report_form.operate_auth_name as auth_name")
	err = db.Order("application_auth_report_form.id desc").Find(&o).Error
	if o == nil {
		o = make([]dto.ApplicationAuthReportForExport, 0)
	}
	return
}

func (r AppAuthReportFormRepository) UpdateById(c context.Context, id int64, o *model.ApplicationAuthReportForm) (err error) {
	return r.GetDB(c).Where("id = ?", id).Updates(o).Error
}

func (r AppAuthReportFormRepository) DeleteByOperateAuthId(c context.Context, id int64) (err error) {
	return r.GetDB(c).Where("operate_auth_id = ?", id).Delete(&model.ApplicationAuthReportForm{}).Error
}

func (r AppAuthReportFormRepository) DeleteByUserId(c context.Context, id string) (err error) {
	return r.GetDB(c).Where("user_id = ?", id).Delete(&model.ApplicationAuthReportForm{}).Error
}

func (r AppAuthReportFormRepository) DeleteByApplicationId(c context.Context, id string) (err error) {
	return r.GetDB(c).Where("application_id = ?", id).Delete(&model.ApplicationAuthReportForm{}).Error
}
