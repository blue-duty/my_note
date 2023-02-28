package api

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"tkbastion/pkg/config"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/validator"
	"tkbastion/server/utils"

	"tkbastion/pkg/log"
	"tkbastion/server/model"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func PropertyGetEndpoint(c echo.Context) error {
	properties := propertyRepository.FindAllMap()
	return Success(c, properties)
}

func PropertyUpdateEndpoint(c echo.Context) error {
	var item map[string]interface{}
	if err := c.Bind(&item); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	for key := range item {
		value := fmt.Sprintf("%v", item[key])
		if value == "" {
			value = "-"
		}

		property := model.Property{
			Name:  key,
			Value: value,
		}

		//数据校验
		if err := c.Validate(property); err != nil {
			msg := validator.GetVdErrMsg(err)
			logMsg := validator.GetErrLogMsg(err, errs.PropertyUpdateLog)
			return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
		}

		_, err := propertyRepository.FindByName(key)
		if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
			if err := propertyRepository.Create(&property); err != nil {
				log.Errorf("DB Error: %v", err)
				return FailWithDataOperate(c, 500, "系统设置-修改系统配置", "", err)
			}
		} else {
			if err := propertyRepository.UpdateByName(&property, key); err != nil {
				log.Errorf("DB Error: %v", err)
				return FailWithDataOperate(c, 500, "系统设置-修改系统配置", "", err)
			}
		}
	}
	return SuccessWithOperate(c, "系统设置-修改系统配置", nil)
}

func SysLogsLevelGetEndpoint(c echo.Context) error {
	sysLogsConfig, err := propertyRepository.FindByName("sys-logs-level")
	if nil != err {
		return err
	}

	sysLogsConfigMap := make(map[string]string)
	sysLogsConfigMap[sysLogsConfig.Name] = sysLogsConfig.Value
	return Success(c, sysLogsConfigMap)
}

func SysLogsLevelUpdateEndpoint(c echo.Context) error {
	var item map[string]interface{}
	if err := c.Bind(&item); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	var level string
	for key := range item {
		value := fmt.Sprintf("%v", item[key])
		switch value {
		case "Panic":
			log.LogHandle.SetLevel(logrus.PanicLevel)
		case "Fatal":
			log.LogHandle.SetLevel(logrus.FatalLevel)
		case "Error":
			log.LogHandle.SetLevel(logrus.ErrorLevel)
		case "Warn":
			log.LogHandle.SetLevel(logrus.WarnLevel)
		case "Info":
			log.LogHandle.SetLevel(logrus.InfoLevel)
		case "Debug":
			log.LogHandle.SetLevel(logrus.DebugLevel)
		default:
			return FailWithDataOperate(c, 422, "不支持的系统日志级别", "系统设置-修改系统日志级别: 不支持的系统日志级别"+value, nil)
		}

		level = value
		property := model.Property{
			Name:  key,
			Value: value,
		}

		_, err := propertyRepository.FindByName(key)
		if err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "修改失败", "", err)
		} else {
			if err := propertyRepository.UpdateByName(&property, key); err != nil {
				log.Errorf("DB Error: %v", err)
				return FailWithDataOperate(c, 500, "修改失败", "", err)
			}

			break
		}
	}
	return SuccessWithOperate(c, "系统设置-修改系统日志级别: "+level, nil)
}

type WebConfig struct {
	Ip    string `json:"ip"`
	Port  string `json:"port"`
	Https bool   `json:"https"`
}

func WebConfigGetEndpoint(c echo.Context) error {
	var adr, filename string
	if config.GlobalCfg.Debug {
		filename = "./config.yml"
	} else {
		filename = "/usr/local/etc/tkbastion/config.yml"
	}
	fi, _ := os.OpenFile(filename, os.O_RDWR, os.ModePerm)
	defer fi.Close()
	reader := bufio.NewReader(fi)
	lineCnt, seekP := 0, 0
	for {
		bs, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		lineCnt = len(bs) + 1
		if strings.Contains(string(bs), "addr: ") {
			adr = strings.Replace(string(bs), " ", "", -1)
			break
		}
		seekP += lineCnt
	}
	addres := strings.Split(adr, ":")
	webconfig := WebConfig{Ip: addres[1], Port: addres[2]}
	return SuccessWithOperate(c, "", webconfig)
}

func AddRollBack(port string, ip string) {
	var filename string
	adr := "  addr: " + ip + ":" + port + "\n"
	if config.GlobalCfg.Debug {
		filename = "./config.yml"
	} else {
		filename = "/usr/local/etc/tkbastion/config.yml"
	}
	fi, _ := os.OpenFile(filename, os.O_RDWR, os.ModePerm)
	defer fi.Close()
	reader := bufio.NewReader(fi)
	lineCnt := 0
	seekP := 0
	for {
		bs, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		lineCnt = len(bs) + 1
		if strings.Contains(string(bs), "addr: ") {
			delBytes := make([]byte, 0)
			for i := 0; i < lineCnt; i++ {
				delBytes = append(delBytes, 127)
			}
			df := []byte(adr)
			if len([]byte(adr)) < len(delBytes) {
				for i := 0; i < (len(delBytes) - len([]byte(adr)) - 1); i++ {
					df = append(df, 32)
				}
			}
			df = append(df, 35)
			fi.WriteAt(delBytes, int64(seekP))
			fi.WriteAt(df, int64(seekP))
			lineCnt = len([]byte(adr))
		}
		seekP += lineCnt
	}
}

func WebConfigUpdateEndpoint(c echo.Context) error {
	var webconfig WebConfig
	var port, filename string
	if err := c.Bind(&webconfig); err != nil {
		log.Errorf("Bind Error:%V", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if err := net.ParseIP(webconfig.Ip); err == nil {
		log.Errorf("ParsePORT Error: %v", "IP格式错误")
		return FailWithDataOperate(c, 422, "IP格式错误", "", err)
	}
	po, _ := strconv.Atoi(webconfig.Port)
	if po > 65535 || po < 0 {
		log.Errorf("ParsePORT Error: %v", "端口范围错误")
		return FailWithDataOperate(c, 422, "端口范围错误,0~65535", "", nil)
	}

	adr := "  addr: " + webconfig.Ip + ":" + webconfig.Port + "\n"
	if config.GlobalCfg.Debug {
		filename = "./config.yml"
	} else {
		filename = "/usr/local/etc/tkbastion/config.yml"
	}
	fi, _ := os.OpenFile(filename, os.O_RDWR, os.ModePerm)
	defer fi.Close()
	reader := bufio.NewReader(fi)
	lineCnt := 0
	seekP := 0
	for {
		bs, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		lineCnt = len(bs) + 1
		if strings.Contains(string(bs), "addr: ") {
			port = strings.Replace(string(bs), " ", "", -1)
			delBytes := make([]byte, 0)
			for i := 0; i < lineCnt; i++ {
				delBytes = append(delBytes, 127)
			}
			df := []byte(adr)
			if len([]byte(adr)) < len(delBytes) {
				for i := 0; i < (len(delBytes) - len([]byte(adr)) - 1); i++ {
					df = append(df, 32)
				}
			}
			df = append(df, 35)
			fi.WriteAt(delBytes, int64(seekP))
			fi.WriteAt(df, int64(seekP))
			lineCnt = len([]byte(adr))
		}

		seekP += lineCnt
	}
	//新端口与旧端口不一致才进行开放与关闭
	oldport := strings.Split(port, ":")
	if oldport[2] != webconfig.Port {
		//开放新端口
		comm := fmt.Sprintf("firewall-cmd --zone=public --add-port=%v/tcp --permanent ", webconfig.Port)
		_, err := utils.ExecShell(comm)
		if err != nil {
			AddRollBack(oldport[2], oldport[1])
			log.Errorf("开放端口:"+webconfig.Port+"失败:%v", err)
			return FailWithDataOperate(c, 422, "开放端口:"+webconfig.Port+"失败", "web配置-开放端口:"+webconfig.Port+"失败", err)
		}
		_, err = utils.ExecShell("firewall-cmd --reload")
		if err != nil {
			AddRollBack(oldport[2], oldport[1])
			log.Errorf("开放端口:"+webconfig.Port+"失败:%v", err)
			return FailWithDataOperate(c, 422, "开放端口:"+webconfig.Port+"失败", "web配置-开放端口:"+webconfig.Port+"失败", err)
		}

	}
	//开启新项目
	if oldport[2] != webconfig.Port || oldport[1] != webconfig.Ip {
		ch1 := make(chan error, 10)
		go func(ch chan<- error) {
			cmd := exec.Command("bash", "-c", "/tkbastion/tkbastion")
			var err error
			if err = cmd.Run(); err != nil {
				//当新项目启动失败，关闭已开放的新端口
				comm := fmt.Sprintf("firewall-cmd --remove-port=%v/tcp --permanent", webconfig.Port)
				_, err = utils.ExecShell(comm)
				_, err = utils.ExecShell("firewall-cmd --reload")
				if err != nil {
					AddRollBack(oldport[2], oldport[1])
					log.Errorf("关闭端口:"+webconfig.Port+"失败", err)
				}
				log.Errorf("新项目启动失败:%v", err)
			}
			ch <- err
		}(ch1)

		timer := time.NewTimer(10 * time.Second) // 设置启动项目的超时时间，超过十秒则默认项目启动成功
		select {
		case <-ch1:
			AddRollBack(oldport[2], oldport[1])
			return FailWithDataOperate(c, 422, "项目"+webconfig.Ip+":"+webconfig.Port+"启动失败", "web配置-启动新项目("+webconfig.Ip+":"+webconfig.Port+")", nil)
		case <-timer.C:
			log.Print("新项目启动成功(" + webconfig.Ip + ":" + webconfig.Port + ")")
		}

	}
	//关闭旧端口
	if oldport[2] != webconfig.Port {
		comm := fmt.Sprintf("firewall-cmd --remove-port=%v/tcp --permanent", oldport[2])
		_, err := utils.ExecShell(comm)
		if err != nil {
			AddRollBack(oldport[2], oldport[1])
			log.Errorf("关闭端口:"+webconfig.Port+"失败", err)
			return FailWithDataOperate(c, 422, "关闭端口:"+oldport[2]+"失败", "web配置-关闭端口:"+oldport[2]+"失败", err)
		}
		_, _ = utils.ExecShell("firewall-cmd --reload")
		if err != nil {
			AddRollBack(oldport[2], oldport[1])
			log.Errorf("关闭端口:"+webconfig.Port+"失败", err)
			return FailWithDataOperate(c, 422, "关闭端口:"+oldport[2]+"失败", "web配置-关闭端口:"+oldport[2]+"失败", err)
		}
	}

	//关闭当前项目
	if oldport[2] != webconfig.Port || oldport[1] != webconfig.Ip {
		account, _ := GetCurrentAccount(c)
		custom := &model.OperateLog{
			Created:         utils.NowJsonTime(),
			LogTypes:        "操作日志",
			LogContents:     "系统设置-web配置-启动新项目(" + webconfig.Ip + ":" + webconfig.Port + ")",
			Users:           account.Username,
			Ip:              c.RealIP(),
			ClientUserAgent: c.Request().UserAgent(),
			Result:          "成功",
		}
		if err := operateLogRepository.Create(custom); err != nil {
			log.Errorf("修改web配置操作插入日志失败:%v", err)
			return FailWithDataOperate(c, 500, "修改web配置操作插入日志失败", "", err)
		}
		defer os.Exit(0)
	}
	return SuccessWithOperate(c, "web配置-修改配置:"+webconfig.Ip+":"+webconfig.Port, webconfig)
}

func HttpsGetEndpoint(c echo.Context) error {
	var filename string
	if config.GlobalCfg.Debug {
		filename = "./config.yml"
	} else {
		filename = "/usr/local/etc/tkbastion/config.yml"
	}
	fi, _ := os.OpenFile(filename, os.O_RDWR, os.ModePerm)
	defer fi.Close()
	reader := bufio.NewReader(fi)
	lineCnt := 0
	seekP := 0
	status := true
	for {
		bs, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		lineCnt = len(bs) + 1
		if strings.Contains(string(bs), "cert:") && strings.Contains(string(bs), "#") {
			status = false
		}
		seekP += lineCnt
	}
	https := WebConfig{Https: status}
	return SuccessWithOperate(c, "", https)
}

func HttpsUpdateEndpoint(c echo.Context) error {
	status := c.Param("status")
	var cert, key, https, filename string
	if status == "false" {
		cert = "#  cert: /tkbastion/config/server.crt\n"
		key = "#  key: /tkbastion/config/server.key\n"
		https = "(关闭)"
	}
	if status == "true" {
		cert = "  cert: /tkbastion/config/server.crt\n"
		key = "  key: /tkbastion/config/server.key\n"
		https = "(开启)"
	}
	if config.GlobalCfg.Debug {
		filename = "./config.yml"
	} else {
		filename = "/usr/local/etc/tkbastion/config.yml"
	}
	fi, _ := os.OpenFile(filename, os.O_RDWR, os.ModePerm)
	reader := bufio.NewReader(fi)
	defer fi.Close()
	lineCnt, seekP := 0, 0
	for {
		bs, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		lineCnt = len(bs) + 1
		if strings.Contains(string(bs), " cert: ") {
			delBytes := make([]byte, 0)
			for i := 0; i < lineCnt; i++ {
				delBytes = append(delBytes, 127)
			}
			bytecert := []byte(cert)
			if len(delBytes) > len(bytecert) {
				for i := 0; i < ((len(delBytes) - len([]byte(cert))) - 1); i++ {
					bytecert = append(bytecert, 32)
				}

			}

			fi.WriteAt(delBytes, int64(seekP))
			fi.WriteAt(bytecert, int64(seekP))
			lineCnt = len(bytecert)
		}
		if strings.Contains(string(bs), " key: ") {
			delBytes := make([]byte, 0)
			for i := 0; i < lineCnt; i++ {
				delBytes = append(delBytes, 127)
			}
			bytekey := []byte(key)
			if len(delBytes) > len(bytekey) {
				for i := 0; i < (len(delBytes) - len(bytekey)); i++ {
					bytekey = append(bytekey, 32)
				}

			}
			fi.WriteAt(delBytes, int64(seekP))
			fi.WriteAt(bytekey, int64(seekP))
			lineCnt = len(bytekey)
		}
		seekP += lineCnt
	}

	pid := os.Getpid()
	cmd := exec.Command("bash", "-c", "kill -9 "+strconv.Itoa(pid)+" && /tkbastion/tkbastion")
	if err := cmd.Run(); err != nil {
		log.Errorf("项目重启失败:%v", err)
		return FailWithDataOperate(c, 500, "项目重启失败", "", nil)
	}

	return SuccessWithOperate(c, "web配置-https"+https, nil)
}
