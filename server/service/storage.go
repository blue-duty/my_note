package service

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"tkbastion/pkg/config"
	"tkbastion/pkg/log"
	"tkbastion/server/repository"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type StorageServiceNew struct {
	storageRepository  *repository.StorageRepositoryNew
	propertyRepository *repository.PropertyRepository
}

func NewStorageServiceNew(storageRepository *repository.StorageRepositoryNew, propertyRepository *repository.PropertyRepository) *StorageServiceNew {
	return &StorageServiceNew{
		storageRepository:  storageRepository,
		propertyRepository: propertyRepository,
	}
}

func (service StorageServiceNew) CreateStorage(name string) (string, error) {
	drivePath := service.GetBaseDrivePath()
	var limitSize int64
	property, err := service.propertyRepository.FindByName("storage_size")
	if err != nil {
		log.Error("未配置默认存储空间大小, 创建失败")
		return "", err
	}
	limitSize, err = strconv.ParseInt(property.Value, 10, 64)
	if err != nil {
		log.Error("未配置默认存储空间大小, 创建失败")
		return "", err
	}

	limitSize = limitSize * 1024 * 1024
	if limitSize < 0 {
		limitSize = -1
	}

	storageDir := path.Join(drivePath, name)
	if err := os.MkdirAll(storageDir, os.ModePerm); err != nil {
		log.Errorf("创建存储空间失败: %s", err)
		return "", err
	}
	log.Infof("创建storage:「%v」文件夹: %v", name, storageDir)
	return storageDir, nil
}

type File struct {
	Name    string         `json:"name"`
	Path    string         `json:"path"`
	IsDir   bool           `json:"isDir"`
	Mode    string         `json:"mode"`
	IsLink  bool           `json:"isLink"`
	ModTime utils.JsonTime `json:"modTime"`
	Size    int64          `json:"size"`
}

func (service StorageServiceNew) Ls(drivePath, remoteDir string) ([]File, error) {
	fileInfos, err := os.ReadDir(path.Join(drivePath, remoteDir))
	if err != nil {
		return nil, err
	}

	var files = make([]File, 0)
	for i := range fileInfos {
		f, err := fileInfos[i].Info()
		if err != nil {
			continue
		}
		file := File{
			Name:    fileInfos[i].Name(),
			Path:    path.Join(remoteDir, fileInfos[i].Name()),
			IsDir:   fileInfos[i].IsDir(),
			Mode:    f.Mode().String(),
			IsLink:  f.Mode()&os.ModeSymlink == os.ModeSymlink,
			ModTime: utils.NewJsonTime(f.ModTime()),
			Size:    f.Size(),
		}

		files = append(files, file)
	}
	return files, nil
}

func (service StorageServiceNew) GetBaseDrivePath() string {
	return config.GlobalCfg.Guacd.Drive
}

func (service StorageServiceNew) DeleteStorageById(id string) error {
	drivePath := service.GetBaseDrivePath()
	_, err := service.storageRepository.FindById(context.TODO(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	// 删除对应的本地目录
	if err := os.RemoveAll(path.Join(drivePath, id)); err != nil {
		return err
	}
	if err := service.storageRepository.Delete(context.TODO(), id); err != nil {
		return err
	}
	return nil
}

func (service StorageServiceNew) StorageUpload(c echo.Context, file *multipart.FileHeader, storageId string) error {
	drivePath := service.GetBaseDrivePath()
	storage, _ := service.storageRepository.FindById(context.TODO(), storageId)
	if storage.LimitSize > 0 {
		dirSize, err := utils.DirSize(path.Join(drivePath, storageId))
		if err != nil {
			log.Info(err)
			return err
		}
		log.Infof("当前目录大小: %v", dirSize)
		log.Infof("文件大小: %v", file.Size)
		if dirSize+file.Size > storage.LimitSize {
			log.Errorf("目录大小超出限制: %v", storage.LimitSize)
			return errors.New("可用空间不足")
		}
	}

	filename := file.Filename
	src, err := file.Open()
	if err != nil {
		return err
	}

	remoteDir := c.QueryParam("dir")
	remoteFile := path.Join(remoteDir, filename)

	if strings.Contains(remoteDir, "../") {
		return errors.New("非法请求")
	}
	if strings.Contains(remoteFile, "../") {
		return errors.New("非法请求")
	}

	// 判断文件夹不存在时自动创建
	dir := path.Join(path.Join(drivePath, storageId), remoteDir)
	if !utils.FileExists(dir) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
	}
	// Destination
	dst, err := os.Create(path.Join(path.Join(drivePath, storageId), remoteFile))
	if err != nil {
		return err
	}
	defer func(dst *os.File) {
		err := dst.Close()
		if err != nil {
			log.Error(err)
		}
	}(dst)

	// Copy
	if _, err = io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}

func (service StorageServiceNew) StorageEdit(file string, fileContent string, storageId string) error {
	drivePath := service.GetBaseDrivePath()
	if strings.Contains(file, "../") {
		return errors.New("非法请求")
	}
	realFilePath := path.Join(path.Join(drivePath, storageId), file)
	dstFile, err := os.OpenFile(realFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer func(dstFile *os.File) {
		err := dstFile.Close()
		if err != nil {
			log.Error(err)
		}
	}(dstFile)
	write := bufio.NewWriter(dstFile)
	if _, err := write.WriteString(fileContent); err != nil {
		return err
	}
	if err := write.Flush(); err != nil {
		return err
	}
	return nil
}

func (service StorageServiceNew) StorageDownload(c echo.Context, file, storageId string) error {
	drivePath := service.GetBaseDrivePath()
	if strings.Contains(file, "../") {
		return errors.New("非法请求")
	}
	// 获取带后缀的文件名称
	filenameWithSuffix := path.Base(file)
	p := path.Join(path.Join(drivePath, storageId), file)
	fmt.Println("文件路径: ", p)

	// 判断文件的大小
	if fileSize, err := utils.FileSize(p); err != nil {
		return err
	} else {
		if fileSize > 1024*1024*1024 {
			return errors.New("文件过大")
		}
	}

	//log.Infof("download %v", p)
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filenameWithSuffix))
	c.Response().Header().Set("Content-Type", "application/octet-stream")

	http.ServeFile(c.Response(), c.Request(), p)
	return nil
}

func (service StorageServiceNew) StorageLs(remoteDir, storageId string) ([]File, error) {
	drivePath := service.GetBaseDrivePath()
	if strings.Contains(remoteDir, "../") {
		return nil, errors.New("非法请求")
	}
	files, err := service.Ls(path.Join(drivePath, storageId), remoteDir)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (service StorageServiceNew) StorageMkDir(remoteDir, storageId string) error {
	drivePath := service.GetBaseDrivePath()
	if strings.Contains(remoteDir, "../") {
		return errors.New("非法请求")
	}
	_, err := os.Stat(path.Join(path.Join(drivePath, storageId), remoteDir))
	if err == nil {
		remoteDir = strings.ReplaceAll(remoteDir, "/", "")
		return errors.New("文件夹" + remoteDir + "已存在")
	}
	if os.IsNotExist(err) {
		if err = os.MkdirAll(path.Join(path.Join(drivePath, storageId), remoteDir), os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

func (service StorageServiceNew) StorageRm(file, storageId string) error {
	drivePath := service.GetBaseDrivePath()
	if strings.Contains(file, "../") {
		return errors.New("非法请求")
	}

	if err := os.RemoveAll(path.Join(path.Join(drivePath, storageId), file)); err != nil {
		return err
	}
	return nil
}

func (service StorageServiceNew) StorageRename(oldName, newName, storageId string) error {
	drivePath := service.GetBaseDrivePath()
	if strings.Contains(oldName, "../") {
		return errors.New("非法请求")
	}
	if strings.Contains(newName, "../") {
		return errors.New("非法请求")
	}
	_, err := os.Stat(path.Join(path.Join(drivePath, storageId), newName))
	if err == nil {
		newName = strings.ReplaceAll(newName, "/", "")
		return errors.New("文件名称 " + newName + " 已存在")
	}
	if os.IsNotExist(err) {
		if err := os.Rename(path.Join(path.Join(drivePath, storageId), oldName), path.Join(path.Join(drivePath, storageId), newName)); err != nil {
			return err
		}
	}

	return nil
}
