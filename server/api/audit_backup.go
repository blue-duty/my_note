package api

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"tkbastion/pkg/constant"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/model"
	s "tkbastion/server/service"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
)

func AuditBackupExportEndpoint(c echo.Context) error {
	backupFile := c.QueryParam("file") + ".zip"
	bupath := path.Join(constant.BackupPath, backupFile)
	if !utils.FileExists(bupath) {
		return FailWithDataOperate(c, 500, "文件不存在", "文件不存在", nil)
	}

	u, f := GetCurrentAccountNew(c)
	if !f {
		return FailWithDataOperate(c, 401, "获取当前用户失败", "审计备份-导出: 导出失败, 获取当前用户失败", nil)
	}

	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "审计日志",
		Created:         utils.NowJsonTime(),
		Users:           u.Username,
		Result:          "成功",
		LogContents:     "审计日志-导出: 导出本地备份,文件[" + backupFile + "]",
		Names:           u.Nickname,
	}
	err := global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}
	return c.Attachment(bupath, backupFile)
}

func AuditBackupFilePagingEndpoint(c echo.Context) error {
	auto := c.QueryParam("auto")
	name := c.QueryParam("name")
	time := c.QueryParam("time")

	files, _ := s.AuditBackupSrv.Find(auto, name, time)
	// fmt.Println(files)
	return Success(c, files)
}

func AuditBackupDeleteEndpoint(c echo.Context) error {
	backupFile := c.Param("file")
	split := strings.Split(backupFile, ",")
	count := 0
	successFile := ""
	filedFile := ""
	for i := range split {
		err := os.Remove(path.Join(constant.BackupPath, split[i]+".zip"))
		if err != nil {
			filedFile += split[i] + ","
		} else {
			successFile += split[i] + ","
			count++
		}
	}

	return SuccessWithOperate(c, "审计备份-删除: 删除本地备份, 删除成功["+strconv.Itoa(count)+"]个, 文件["+successFile+"], 失败["+filedFile+"]", nil)
}

func AuditBackupEndpoint(c echo.Context) error {
	bupath, err := s.AuditBackupSrv.BackupAuditLog()
	if err != nil {
		log.Errorf("Local Backup Error: %v", err)
		return FailWithDataOperate(c, 500, "备份失败", "审计日志本地备份失败", nil)
	}

	command := "zip -rj " + path.Join(constant.BackupPath, bupath+".zip") + " " + path.Join(constant.BackupPath, bupath)
	//err = ExecutiveCommand("tar -Pczf " + "tkbastion_审计日志_" + now + ".tar -C " + backupPath + " tkbastion_审计日志_" + now)
	_, err = utils.ExecShell(command)
	fmt.Println(err)
	if err != nil {
		log.Errorf("zip Error: %v", err)
	}

	//移动文件至备份文件夹
	_, _ = utils.ExecShell("mv " + bupath + ".zip " + constant.BackupPath)

	return SuccessWithOperate(c, "审计备份-备份: 备份成功, 文件["+bupath+".zip]", nil)
}
