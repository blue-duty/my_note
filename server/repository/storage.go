package repository

import (
	"context"
	"path"
	"tkbastion/pkg/config"
	"tkbastion/server/dto"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"gorm.io/gorm"
)

type StorageRepositoryNew struct {
	baseRepository
}

// Create 创建
func (r *StorageRepositoryNew) Create(ctx context.Context, storage *model.NewStorage) error {
	return r.GetDB(ctx).Create(storage).Error
}

// Update 更新
func (r *StorageRepositoryNew) Update(ctx context.Context, storage *model.NewStorage) error {
	storageMap := utils.Struct2MapByStructTag(storage)
	return r.GetDB(ctx).Model(storage).Where("id = ?", storage.ID).Updates(storageMap).Error
}

func (r *StorageRepositoryNew) FindByName(todo context.Context, name string) (*model.NewStorage, error) {
	var storage model.NewStorage
	err := r.GetDB(todo).Where("name = ?", name).First(&storage).Error
	return &storage, err
}

func deleteByStorageId(db *gorm.DB, id string) error {
	err := db.Where("id = ?", id).Delete(&model.NewStorage{}).Error
	if err != nil {
		return err
	}
	err = db.Where("value = ?", id).Delete(&model.PassportConfiguration{}).Error
	if err != nil {
		return err
	}
	return nil
}

// Delete 删除
func (r *StorageRepositoryNew) Delete(ctx context.Context, id string) error {
	db := r.GetDB(ctx).Begin()
	err := deleteByStorageId(db, id)
	if err != nil {
		db.Rollback()
		return err
	}
	return db.Commit().Error
}

// FindByNameId 根据名称查询
func (r *StorageRepositoryNew) FindByNameId(ctx context.Context, name, id string) (*model.NewStorage, error) {
	var storage model.NewStorage
	err := r.GetDB(ctx).Where("name = ? and id != ?", name, id).First(&storage).Error
	return &storage, err
}

// FindById 根据ID查询
func (r *StorageRepositoryNew) FindById(ctx context.Context, id string) (*model.NewStorage, error) {
	var storage model.NewStorage
	err := r.GetDB(ctx).Where("id = ?", id).First(&storage).Error
	return &storage, err
}

// FindBySearch 根据搜索条件查询
func (r *StorageRepositoryNew) FindBySearch(ctx context.Context, search dto.StorageForSearch) ([]dto.StorageForPage, error) {
	var list []model.NewStorage
	db := r.GetDB(ctx).Where("department in (?)", search.Departments)
	if search.Name != "" {
		db = db.Where("name like ?", "%"+search.Name+"%")
	} else if search.Department != "" {
		db = db.Where("department_name like ?", "%"+search.Department+"%")
	} else if search.Auto != "" {
		db = db.Where("name like ? or department_name like ?", "%"+search.Auto+"%", "%"+search.Auto+"%")
	}
	err := db.Find(&list).Error
	var pageList = make([]dto.StorageForPage, len(list))
	for i, v := range list {
		pageList[i] = dto.StorageForPage{
			ID:           v.ID,
			Name:         v.Name,
			DepartmentId: v.Department,
			Department:   v.DepartmentName,
			LimitSize:    v.LimitSize,
			Info:         v.Info,
		}
		dirSize, err := utils.DirSize(path.Join(config.GlobalCfg.Guacd.Drive, v.ID))
		if err != nil {
			pageList[i].UseSize = -1
		} else {
			pageList[i].UseSize = dirSize
		}
	}
	return pageList, err
}

func (r *StorageRepositoryNew) FindByDepartmentId(todo context.Context, ids []int64) ([]model.NewStorage, error) {
	var list []model.NewStorage
	err := r.GetDB(todo).Where("department in (?)", ids).Find(&list).Error
	return list, err
}
