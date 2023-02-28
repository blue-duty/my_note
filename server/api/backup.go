package api

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"tkbastion/pkg/constant"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/pkg/validator"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
)

type Backup struct {
	Name        string `json:"name"`
	ModTime     string `json:"modTime"`
	Status      string `json:"status"`
	Description string `json:"description" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
	Size        string `json:"size"`
}

// BackupFilePagingEndpoint 获取备份文件列表
func BackupFilePagingEndpoint(c echo.Context) error {
	if !utils.FileExists(constant.BackupPath) {
		if err := os.MkdirAll(constant.BackupPath, os.ModePerm); err != nil {
			return nil
		}
	}
	fileInfos, err := os.ReadDir(constant.BackupPath)
	if err != nil {
		return nil
	}
	if ok := utils.FileExists(path.Join(constant.BackupPath, "backup.des")); !ok {
		// 创建文件
		_, err := os.Create(path.Join(constant.BackupPath, "backup.des"))
		if err != nil {
			return nil
		}
	}
	data := utils.ReadNetworkFile(constant.BackupPath + "/" + "backup.des")
	var files = make([]Backup, 0)
	for i := range fileInfos {
		if !strings.HasSuffix(fileInfos[i].Name(), ".tgz") && !strings.Contains(fileInfos[i].Name(), "tk_backup") {
			continue
		}
		fileInfo, _ := fileInfos[i].Info()
		file := Backup{
			Name:        fileInfos[i].Name(),
			ModTime:     utils.NewJsonTime(fileInfo.ModTime()).Format("2006-01-02 15:04:05"),
			Size:        humanize.Bytes(uint64(fileInfo.Size())),
			Description: data[fileInfos[i].Name()],
			Status:      "正常",
		}
		files = append(files, file)
	}
	return Success(c, files)
}

// BackupCreateEndpoint 创建备份文件
func BackupCreateEndpoint(c echo.Context) error {
	var backup Backup
	if err := c.Bind(&backup); err != nil {
		log.Errorf("Backup Error: %v", err)
		return FailWithDataOperate(c, 500, "创建失败", "", err)
	}

	if err := c.Validate(&backup); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	mysqlAddress := global.Config.Mysql.Hostname
	mysqlPort := strconv.Itoa(global.Config.Mysql.Port)
	mysqlUser := global.Config.Mysql.Username
	mysqlPassword := global.Config.Mysql.Password
	mysqlDatabase := global.Config.Mysql.Database
	backupPath := constant.BackupPath

	err, backupSql := backupService.BackupMySqlDb(mysqlAddress, mysqlPort, mysqlUser, mysqlPassword, mysqlDatabase, backupPath)
	if err != nil {
		log.Errorf("Backup Error: %v", err)
		log.Errorf("系统维护-配置备份: 新建备份文件[备份数据库]: %v", err)
		return FailWithDataOperate(c, 500, "备份数据库失败", "", err)
	}
	if err := os.MkdirAll(path.Join(constant.BackupPath, "temporaryFiles"), os.ModePerm); err != nil {
		return err
	}
	now := "tk_backup_" + time.Now().Format("20060102150405")
	// 移动文件至临时文件夹
	if _, err = utils.ExecShell("mv " + backupSql + " " + path.Join(constant.BackupPath, "temporaryFiles")); err != nil {
		log.Errorf("Backup mv .sql Error: %v", err)
		log.Errorf("系统维护-配置备份: 新建备份文件[创建临时文件]: %v", err)
		return FailWithDataOperate(c, 500, "备份数据库失败", "", err)
	}
	if _, err = utils.ExecShell("cp /tkbastion/config/db.sql " + path.Join(constant.BackupPath, "temporaryFiles")); err != nil {
		log.Errorf("Backup cp /tkbastion/config/db.sql Error: %v", err)
		log.Errorf("系统维护-配置备份: 新建备份文件[创建临时文件]: %v", err)
		return FailWithDataOperate(c, 500, "备份数据库失败", "", err)
	}
	//打包tar
	if _, err = utils.ExecShell("tar -czvf " + now + ".tgz " + path.Join(constant.BackupPath, "temporaryFiles")); err != nil {
		log.Errorf("Backup  tar  Error: %v", err)
		log.Errorf("系统维护-配置备份: 新建备份文件[打包文件]: %v", err)
		return FailWithDataOperate(c, 500, "备份失败", "", err)
	}
	//移动文件至备份文件夹
	if _, err = utils.ExecShell("mv " + now + ".tgz " + constant.BackupPath); err != nil {
		log.Errorf("Backup mv .tgz  Error: %v", err)
		log.Errorf("系统维护-配置备份: 新建备份文件[移动文件]: %v", err)
		return FailWithDataOperate(c, 500, "备份失败", "", err)
	}
	//删除临时文件
	if _, err = utils.ExecShell("rm -rf " + path.Join(constant.BackupPath, "temporaryFiles")); err != nil {
		log.Errorf("Backup delete temporaryFiles Error: %v", err)
		log.Errorf("系统维护-配置备份: 新建备份文件[删除临时文件]: %v", err)
		return FailWithDataOperate(c, 500, "备份失败", "", err)
	}
	// 判断description文件是否存在
	if ok := utils.FileExists(path.Join(constant.BackupPath, "backup.des")); !ok {
		if err := os.MkdirAll(path.Join(constant.BackupPath, "backup.des"), os.ModePerm); err != nil {
			log.Errorf("Backup delete temporaryFiles Error: %v", err)
			log.Errorf("系统维护-配置备份: 新建备份文件[写入描述]: %v", err)
		}
	}
	data := utils.ReadNetworkFile(path.Join(constant.BackupPath, "backup.des"))
	data[now+".tgz"] = backup.Description
	err = utils.WriteNetworkFile(path.Join(constant.BackupPath, "backup.des"), data)
	if err != nil {
		log.Errorf("Backup delete temporaryFiles Error: %v", err)
		log.Errorf("系统维护-配置备份: 新建备份文件[写入描述]: %v", err)
	}
	return SuccessWithOperate(c, "配置备份-新建: 系统维护备份文件[描述: "+backup.Description+"]", nil)
}

// BackupRestoreEndpoint 还原备份文件
func BackupRestoreEndpoint(c echo.Context) error {
	var backup Backup
	if err := c.Bind(&backup); err != nil {
		log.Errorf("系统维护-配置备份: [参数绑定]: %v", err)
		return FailWithDataOperate(c, 500, "参数绑定失败", "", err)
	}
	fmt.Println(backup.Name)
	fmt.Println("tar -xzvf " + constant.BackupPath + "/" + backup.Name + " -C " + " /")
	//解压tar.gz
	if _, err := utils.ExecShell("tar -xzvf " + path.Join(constant.BackupPath, backup.Name) + " -C " + " /"); err != nil {
		log.Errorf("Backup Error: %v", err)
		log.Errorf("系统维护-配置备份: [解析备份文件]: %v", err)
		return FailWithDataOperate(c, 500, "解析文件失败", "", err)
	}
	//恢复db.sql
	if _, err := utils.ExecShell("\\cp " + path.Join(constant.BackupPath, "temporaryFiles", "db.sql") + " /tkbastion/config"); err != nil {
		log.Errorf("Backup Error: %v", err)
		log.Errorf("系统维护-配置备份: [恢复数据库]: %v", err)
		return FailWithDataOperate(c, 500, "恢复db.sql失败", "", err)
	}
	//恢复数据表
	mysqlAddress := global.Config.Mysql.Hostname
	mysqlPort := strconv.Itoa(global.Config.Mysql.Port)
	mysqlUser := global.Config.Mysql.Username
	mysqlPassword := global.Config.Mysql.Password
	mysqlDatabase := global.Config.Mysql.Database
	backupPath := constant.BackupPath + "/temporaryFiles/"
	// 读取temporaryFiles文件夹下的所有文件
	files, err := os.ReadDir(backupPath)
	if err != nil {
		log.Errorf("Backup Error: %v", err)
		log.Errorf("系统维护-配置备份: [读取temporaryFiles文件夹下的所有文件]: %v", err)
		return FailWithDataOperate(c, 500, "读取temporaryFiles文件夹下的所有文件失败", "", err)
	}
	for _, file := range files {
		if file.Name() != "db.sql" && strings.Contains(file.Name(), ".sql") {
			err := backupService.RecoverMySqlDb(mysqlAddress, mysqlPort, mysqlUser, mysqlPassword, mysqlDatabase, backupPath+"/"+file.Name())
			if err != nil {
				log.Errorf("Backup Error: %v", err)
				log.Errorf("系统维护-配置备份: [恢复数据表]: %v", err)
				return FailWithDataOperate(c, 500, "恢复数据表失败", "", err)
			}
		}
	}
	//删除解压的临时文件
	if _, err = utils.ExecShell("rm -rf " + path.Join(constant.BackupPath, "temporaryFiles")); err != nil {
		log.Errorf("Backup Delete Error: %v", err)
	}
	return SuccessWithOperate(c, "配置备份-还原: 系统维护还原文件["+backup.Name+"]", nil)
}

// BackupDeleteEndpoint 删除备份文件
func BackupDeleteEndpoint(c echo.Context) error {
	backupFile := c.Param("name")
	// 删除备份文件
	if _, err := utils.ExecShell("rm -rf " + constant.BackupPath + "/" + backupFile); err != nil {
		log.Errorf("Backup Error: %v", err)
		return FailWithDataOperate(c, 500, "删除备份文件失败", "", err)
	}
	//if err := os.Remove(path.Join(constant.BackupPath, backupFile)); err != nil {
	//	log.Errorf("Remove Error: %v", err)
	//	return FailWithDataOperate(c, 500, "删除失败", "", err)
	//}
	// 读取文件描述信息
	file, err := os.Open(path.Join(constant.BackupPath, "backup.des"))
	if err != nil {
		log.Errorf("Backup Error: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			return
		}
	}(file)
	// 读取文件内容
	content, err := os.ReadFile(path.Join(constant.BackupPath, "backup.des"))
	if err != nil {
		log.Errorf("Backup Error: %v", err)
	}
	backupDes := string(content)
	split := strings.Split(backupDes, "\n")
	var backupDesList []string
	for _, v := range split {
		if !strings.Contains(v, backupFile) {
			backupDesList = append(backupDesList, v)
		}
	}
	// 写入文件
	err = os.WriteFile(path.Join(constant.BackupPath, "backup.des"), []byte(strings.Join(backupDesList, "\n")), 0666)
	if err != nil {
		log.Errorf("Backup Error: %v", err)
	}
	return SuccessWithOperate(c, "配置备份-删除: 系统维护删除文件["+backupFile+"]", nil)
}

// BackupDownloadEndpoint 下载备份文件
func BackupDownloadEndpoint(c echo.Context) error {
	backupFile := c.QueryParam("backupFile")
	user, _ := GetCurrentAccountNew(c)
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "",
		Created:         utils.NowJsonTime(),
		Users:           user.Username,
		Names:           user.Nickname,
		LogContents:     "配置备份-下载: 系统维护下载文件[" + backupFile + "]",
		Result:          "成功",
	}
	err := global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}
	return c.Attachment(path.Join(constant.BackupPath, backupFile), backupFile)
}

// BackupRestoreLocalEndpoint 从本地恢复文件
func BackupRestoreLocalEndpoint(c echo.Context) error {
	backupFile, err := c.FormFile("backupFile")
	if err != nil {
		log.Errorf("FormFile Error: %v", err)
		return FailWithDataOperate(c, 500, "文件格式错误", "", err)
	}
	// 判断文件夹不存在时自动创建
	if !utils.FileExists(constant.BackupPath) {
		if err := os.MkdirAll(constant.BackupPath, os.ModePerm); err != nil {
			return err
		}
	}
	fileInfos, _ := os.ReadDir(constant.BackupPath)
	for i := range fileInfos {
		if fileInfos[i].Name() == backupFile.Filename {
			log.Errorf("Upload Error: %v", "文件已存在")
			return FailWithDataOperate(c, 500, "文件已存在", "", nil)
		}
	}
	src, err := backupFile.Open()
	// Destination
	dst, err := os.Create(path.Join(constant.BackupPath, backupFile.Filename))
	if err != nil {
		log.Errorf("Create Error: %v", err)
		return FailWithDataOperate(c, 500, "创建文件失败", "", err)
	}
	defer func(dst *os.File) {
		err := dst.Close()
		if err != nil {
			log.Errorf("Close file Error %v", err)
		}
	}(dst)
	// Copy
	if _, err = io.Copy(dst, src); err != nil {
		log.Errorf("Copy Error: %v", err)
		return FailWithDataOperate(c, 500, "上传失败", "", err)
	}

	// 判断description文件是否存在
	if ok := utils.FileExists(path.Join(constant.BackupPath, "backup.des")); !ok {
		if err := os.MkdirAll(path.Join(constant.BackupPath, "backup.des"), os.ModePerm); err != nil {
			log.Errorf("Backup delete temporaryFiles Error: %v", err)
			log.Errorf("系统维护-配置备份: 新建备份文件[写入描述]: %v", err)
		}
	}
	data := utils.ReadNetworkFile(path.Join(constant.BackupPath, "backup.des"))
	data[backupFile.Filename] = time.Now().Format("2006-01-02 15:04:05") + " 上传"
	err = utils.WriteNetworkFile(path.Join(constant.BackupPath, "backup.des"), data)
	if err != nil {
		log.Errorf("Backup delete temporaryFiles Error: %v", err)
		log.Errorf("系统维护-配置备份: 新建备份文件[写入描述]: %v", err)
	}

	//解压tar.gz
	if _, err := utils.ExecShell("tar -xzvf " + path.Join(constant.BackupPath, backupFile.Filename) + " -C " + " /"); err != nil {
		log.Errorf("Backup Error: %v", err)
		log.Errorf("系统维护-配置备份: 导入备份文件[解析备份文件]: %v", err)
		return FailWithDataOperate(c, 500, "解析文件失败", "", err)
	}
	//恢复db.sql
	if _, err = utils.ExecShell("\\cp " + path.Join(constant.BackupPath, "temporaryFiles", "db.sql") + " /tkbastion/config"); err != nil {
		log.Errorf("Backup Error: %v", err)
		return FailWithDataOperate(c, 500, "恢复db.sql失败", "", err)
	}
	//恢复数据表
	mysqlAddress := global.Config.Mysql.Hostname
	mysqlPort := strconv.Itoa(global.Config.Mysql.Port)
	mysqlUser := global.Config.Mysql.Username
	mysqlPassword := global.Config.Mysql.Password
	mysqlDatabase := global.Config.Mysql.Database
	backupPath := constant.BackupPath + "/temporaryFiles/"
	// 读取temporaryFiles文件夹下的所有文件
	files, err := os.ReadDir(backupPath)
	if err != nil {
		log.Errorf("Backup Error: %v", err)
		log.Errorf("系统维护-配置备份: [读取temporaryFiles文件夹下的所有文件]: %v", err)
		return FailWithDataOperate(c, 500, "读取temporaryFiles文件夹下的所有文件失败", "", err)
	}
	for _, file := range files {
		if file.Name() != "db.sql" && strings.Contains(file.Name(), ".sql") {
			err := backupService.RecoverMySqlDb(mysqlAddress, mysqlPort, mysqlUser, mysqlPassword, mysqlDatabase, backupPath+"/"+file.Name())
			if err != nil {
				log.Errorf("Backup Error: %v", err)
				log.Errorf("系统维护-配置备份: [恢复数据表]: %v", err)
				return FailWithDataOperate(c, 500, "恢复数据表失败", "", err)
			}
		}
	}
	//删除解压的临时文件
	if _, err = utils.ExecShell("rm -rf " + path.Join(constant.BackupPath, "temporaryFiles")); err != nil {
		log.Errorf("Backup Error: %v", err)
		return FailWithDataOperate(c, 500, "删除临时文件失败", "", err)
	}
	return SuccessWithOperate(c, "配置备份-本地恢复: 系统维护,本地恢复文件["+backupFile.Filename+"]", nil)
}
