package repository

import (
	"tkbastion/pkg/constant"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"gorm.io/gorm"
)

type UserNewRepository struct {
	DB *gorm.DB
}

func NewUserNewRepository(db *gorm.DB) *UserNewRepository {
	userNewRepository = &UserNewRepository{DB: db}
	return userNewRepository
}

func (r *UserNewRepository) Create(o *model.UserNew) (err error) {
	err = r.DB.Create(o).Error
	return
}

func (r *UserNewRepository) FindAll() (o []model.UserNew, err error) {
	err = r.DB.Find(&o).Error
	return
}

func (r *UserNewRepository) FindById(id string) (o model.UserNew, err error) {
	err = r.DB.Table("user_new").Where("id = ?", id).Find(&o).Error
	return
}
func (r *UserNewRepository) FindByRoleId(id string) (o []model.UserNew, err error) {
	err = r.DB.Table("user_new").Where("role_id = ?", id).Find(&o).Error
	return
}
func (r *UserNewRepository) FindByName(name string) (o model.UserNew, err error) {
	err = r.DB.Where("username = ?", name).First(&o).Error
	return
}

func (r *UserNewRepository) UpdateMapById(o model.UserNew, id string) (err error) {
	userMap := utils.Struct2MapByStructTag(o)
	err = r.DB.Table("user_new").Where("id = ?", id).Updates(userMap).Error
	return
}
func (r *UserNewRepository) UpdateStructById(o model.UserNew, id string) (err error) {
	err = r.DB.Table("user_new").Where("id = ?", id).Updates(&o).Error
	return
}
func (r *UserNewRepository) UpdateByRoleId(o *model.UserNew, id string) (err error) {
	err = r.DB.Table("user_new").Where("id = ?", id).Updates(o).Error
	return
}

func (r *UserNewRepository) DeleteById(id string) (err error) {
	err = r.DB.Where("id = ?", id).Delete(&model.UserNew{}).Error
	return
}

func (r *UserNewRepository) FindByLimitingConditions(pageIndex, pageSize int, auto, username, nickname, department, roleName, status string, departmentId []int64) (userNew []dto.UserForPage, total int64, err error) {
	// 根据部门id筛选一遍数据
	db := r.DB.Table("user_new").Where("department_id in (?)", departmentId)
	if len(auto) > 0 {
		db = db.Where("username like ? or nickname like ? or department_name like ? or role_name like ? or description like ? ", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%")
	} else {
		if len(username) > 0 {
			db = db.Where("username like ?", "%"+username+"%")
		}
		if len(nickname) > 0 {
			db = db.Where("nickname like ?", "%"+nickname+"%")
		}
		if len(department) > 0 {
			db = db.Where("department_name like ?", "%"+department+"%")
		}
		if len(roleName) > 0 {
			db = db.Where("role_name like ?", "%"+roleName+"%")
		}
		if len(status) > 0 {
			db = db.Where("status like ?", "%"+status+"%")
		}
	}
	err = db.Count(&total).Error
	if err != nil {
		return []dto.UserForPage{}, 0, err
	}
	// 按时间升序排列
	err = db.Order("created asc").Find(&userNew).Error
	return
}

func (r *UserNewRepository) FindUserByDepartmentId(departmentId []int64) (userNew []model.UserNew, err error) {
	db := r.DB.Table("user_new")
	err = db.Where("department_id in (?)", departmentId).Find(&userNew).Error
	return
}

func (r *UserNewRepository) FindUserCountByDepId(depId int64) (count int64, err error) {
	db := r.DB.Table("user_new")
	err = db.Where("department_id = ?", depId).Count(&count).Error
	return
}

func (r *UserNewRepository) FindUserCountByDepIds(depIds []int64) (count int64, err error) {
	db := r.DB.Table("user_new")
	err = db.Where("department_id IN ?", depIds).Count(&count).Error
	return
}

func (r *UserNewRepository) FindUserByDepartmentIdEnable(departmentId []int64) (userNew []model.UserNew, err error) {
	db := r.DB.Table("user_new").Where("status = ?", constant.Enable)
	err = db.Where("department_id in (?)", departmentId).Find(&userNew).Error
	return
}

//func (r *UserNewRepository) FindUserByDepartmentIdEnableNotUserIds(departmentId []int64, userIds []string) (userNew []model.UserNew, err error) {
//	db := r.DB.Table("user_new").Where("status = ?", constant.Enable)
//	err = db.Where("department_id in (?) AND id NOT IN ?", departmentId, userIds).Find(&userNew).Error
//	return
//}

func (r *UserNewRepository) DeleteUserByDepartmentId(departmentId []int64) (err error) {
	err = r.DB.Table("user_new").Where("department_id in (?)", departmentId).Delete(&model.UserNew{}).Error
	return
}

func (r *UserNewRepository) DeleteByDepNotGen() (err error) {
	err = r.DB.Table("user_new").Where("department_id != 0").Delete(&model.UserNew{}).Error
	return
}

func (r *UserNewRepository) UserExport(departmentId []int64) (userForExport []dto.UserForExport, err error) {
	db := r.DB.Table("user_new").Select("username,nickname,department_name,role_name,status,mail,qq,wechat,phone,description")
	err = db.Where("department_id in (?)", departmentId).Find(&userForExport).Error
	return
}
func (r *UserNewRepository) FindByIds(ids []string) (o []model.UserNew, err error) {
	err = r.DB.Where("id in (?)", ids).Find(&o).Error
	return
}

// FindDepartmentAdminByDepartmentId 根据部门id 获取本部门的所有部门管理员
func (r *UserNewRepository) FindDepartmentAdminByDepartmentId(departmentId int64) (userNew []model.UserNew, err error) {
	db := r.DB.Table("user_new")
	err = db.Where("department_id = ? and role_name = ? and status = ?", departmentId, constant.DepartmentAdmin, constant.Enable).Find(&userNew).Error
	return
}

// FindAdmin 所有系统管理员
func (r *UserNewRepository) FindAdmin() (userNew model.UserNew, err error) {
	db := r.DB.Table("user_new")
	err = db.Where("username = ?", "admin").Find(&userNew).Error
	return
}

// FindByAuthServerId  通过认证服务器id找到所有该服务器下的用户
func (r *UserNewRepository) FindByAuthServerId(authServerId int64) (o []model.UserNew, err error) {
	err = r.DB.Table("user_new").Where("authentication_server_id = ?", authServerId).Find(&o).Error
	return
}

// FindByAuthType 通过认证方式找到所有用户
func (r *UserNewRepository) FindByAuthType(authenticationWay string) (o []model.UserNew, err error) {
	err = r.DB.Table("user_new").Where("authentication_way like ?", "%"+authenticationWay+"%").Find(&o).Error
	return
}

func (r *UserNewRepository) UpdateOnline(id string, online bool) error {
	err := r.DB.Table("user_new").Where("id = ?", id).Update("online", online).Error
	return err

}

func (r *UserNewRepository) FindOnlineUsers() (o []model.UserNew, err error) {
	err = r.DB.Table("user_new").Where("online = ?", true).Find(&o).Error
	return
}
