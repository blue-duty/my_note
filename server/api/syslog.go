package api

import (
	"bytes"
	"context"
	"github.com/docker/docker/client"
	"net/http"
	"strings"
	"time"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/pkg/service"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
)

func SysLogPagingEndpoint(c echo.Context) error {
	content, err := service.GetCommandLinuxCon("cat /var/log/tkbastion/tig-fortress-machine.log")
	if err != nil {
		log.Errorf("获取系统日志失败: %v", err)
		return FailWithDataOperate(c, 500, "获取系统日志失败", "获取系统日志失败: "+err.Error(), err)
	}
	str := string(content)
	split := strings.Split(str, "\n")
	var sysLog string
	if len(split) <= 100 {
		for i := range split {
			sysLog = sysLog + split[i] + "\n"
		}
		sysLog = strings.TrimLeft(sysLog, " ")
	} else {
		for i := range split[len(split)-100:] {
			sysLog = sysLog + split[len(split)-100:][i] + "\n"
		}
		sysLog = strings.TrimLeft(sysLog, " ")
	}
	return SuccessWithOperate(c, "获取系统日志成功", sysLog)
}

func SysLogExportEndpoint(c echo.Context) error {
	u, _ := GetCurrentAccount(c)
	operateLog := model.OperateLog{
		Ip:              c.RealIP(),
		ClientUserAgent: c.Request().UserAgent(),
		LogTypes:        "运维日志",
		Created:         utils.NowJsonTime(),
		Users:           u.Username,
		Result:          "成功",
		LogContents:     "导出系统日志",
	}
	err := global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
	}
	return c.Attachment("/var/log/tkbastion/tig-fortress-machine.log", "系统日志.txt")
}

func GuacdLogPagingEndpoint(c echo.Context) error {
	prove, err := propertyRepository.FindByName("enable-debug")
	if err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "获取主机接入日志失败", "", err)
	}
	var content []byte
	if prove.Value != "true" {
		//fmt.Println("获取docker guacd")
		//container, err := service.GetCommandLinuxCon("docker ps -a | grep dushixiang/guacd")
		//if err != nil {
		//	log.Errorf("获取docker容器ID失败: %v", err)
		//	return FailWithDataOperate(c, 500, "获取docker容器ID失败: "+err.Error(), "", err)
		//}
		//containerid := string(container)[:12]
		//fmt.Println("docker id ", containerid)
		//command := fmt.Sprintf("docker logs %s | grep  guacd[", containerid)
		//fmt.Println(command)
		//content, err = service.GetCommandLinuxCon(command)
		//if err != nil {
		//	log.Errorf("获取容器guacd日志失败: %v", err)
		//	return FailWithDataOperate(c, 500, "获取容器guacd日志失败: "+err.Error(), "", err)
		//}
		//logpathbyte, err := service.GetCommandLinuxCon("docker inspect --format ‘{{.LogPath}}’ " + containerid)
		//if err != nil {
		//	log.Errorf("获取docker容器日志位置失败: %v", err)
		//	return FailWithDataOperate(c, 500, "获取docker容器日志位置失败: "+err.Error(), "", err)
		//}
		//logpath := string(logpathbyte)[1 : len(string(logpathbyte))-2]
		//fmt.Println(logpath)
		//command := fmt.Sprintf("cat " + logpath)
		//fmt.Println("command:", command)
		//content, err := service.GetCommandLinuxCon(command)
		//fmt.Println("content:", string(content))
		//cl, err := client.NewClientWithOpts()
		//if err != nil {
		//	return FailWithDataOperate(c, 500, "获取guacd日志失败: "+err.Error(), "", err)
		//}
		////fmt.Println(cl.ImageList(context.Background(), types.ImageListOptions{}))
		//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		//defer cancel()
		//list, err := cl.ContainerList(ctx, types.ContainerListOptions{
		//	All: true,
		//})
		//if err != nil {
		//	return FailWithDataOperate(c, 500, "获取docker容器列表失败: "+err.Error(), "", err)
		//}
		//var containid string
		//for _, l := range list {
		//	if l.Image == "dushixiang/guacd" {
		//		containid = l.ID
		//	}
		//}
		//reader, err := cl.ContainerLogs(ctx, containid, types.ContainerLogsOptions{
		//	Timestamps: true,
		//	ShowStderr: true,
		//	Tail:       "100",
		//})
		//if err != nil {
		//	return FailWithDataOperate(c, 500, "获取guacd日志失败: "+err.Error(), "", err)
		//}
		//body, err := io.ReadAll(reader)
		//return SuccessWithData(c, 200, "获取guacd日志成功", string(body))
		//containerId, err := service.GetCommandLinuxCon(`docker ps -a | grep dushixiang/guacd | awk '{print $1}'`)
		//if err != nil {
		//	log.Errorf("获取docker容器ID失败: %v", err)
		//	return FailWithDataOperate(c, 500, "获取docker容器ID失败: "+err.Error(), "", err)
		//}
		//containerIdStr := strings.Replace(string(containerId), "\n", "", -1)
		containerLog, err := service.GetCommandLinuxCon(`docker logs -t  --tail=100 ` + global.CONTAINERID + ` 2>&1`)
		if err != nil {
			log.Errorf("获取docker容器日志失败: %v", err)
			return FailWithDataOperate(c, 500, "获取主机接入日志失败", "获取docker容器日志失败: "+err.Error(), err)
		}
		return SuccessWithData(c, 200, "获取主机接入日志成功", string(containerLog))
	} else {
		//fmt.Println("获取本地guacd")
		content, err = service.GetCommandLinuxCon("cat /var/log/messages | grep \"guacd\\[\"")
		if err != nil {
			log.Errorf("获取本地guacd日志失败: %v", err)
			return FailWithDataOperate(c, 500, "获取主机接入日志失败", "获取本地guacd日志失败: "+err.Error(), err)
		}

		str := string(content)
		split := strings.Split(str, "\n")
		var sysLog string
		if len(split) <= 100 {
			for i := range split {
				sysLog = sysLog + split[i] + "\n"
			}
			sysLog = strings.TrimLeft(sysLog, " ")
		} else {
			for i := range split[len(split)-100:] {
				sysLog = sysLog + split[len(split)-100:][i] + "\n"
			}
			sysLog = strings.TrimLeft(sysLog, " ")
		}
		return SuccessWithData(c, 200, "获取主机接入日志成功", sysLog)
	}
}

func GuacdLogExportEndpoint(c echo.Context) error {
	prove, err := propertyRepository.FindByName("enable-debug")
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "导出主机接入日志失败", "", err)
	}
	if prove.Value != "true" {
		//container, err := service.GetCommandLinuxCon("docker ps -a | grep dushixiang/guacd")
		//if err != nil {
		//	log.Errorf("获取docker容器ID失败: %v", err)
		//	return FailWithDataOperate(c, 500, "获取docker容器ID失败: "+err.Error(), "", err)
		//}
		//containerid := string(container)[:12]
		//logpathbyte, err := service.GetCommandLinuxCon("docker inspect --format ‘{{.LogPath}}’ " + containerid)
		//if err != nil {
		//	log.Errorf("获取docker容器日志位置失败: %v", err)
		//	return FailWithDataOperate(c, 500, "获取docker容器日志位置失败: "+err.Error(), "", err)
		//}
		//logpath := string(logpathbyte)[1 : len(string(logpathbyte))-2]
		//return c.Attachment(logpath, "guacd.log")
		cl, err := client.NewClientWithOpts()
		if err != nil {
			log.Errorf("获取docker容器失败: %v", err)
			return FailWithDataOperate(c, 500, "导出主机接入日志失败", "获取docker容器失败: "+err.Error(), err)
		}
		//fmt.Println(cl.ImageList(context.Background(), types.ImageListOptions{}))
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		logPath, err := cl.ContainerInspect(ctx, global.CONTAINERID)
		if err != nil {
			log.Errorf("获取docker容器日志位置失败: %v", err)
			return FailWithDataOperate(c, 500, "导出主机接入日志失败", "获取docker容器日志位置失败: "+err.Error(), err)
		}
		u, _ := GetCurrentAccount(c)
		operateLog := model.OperateLog{
			Ip:              c.RealIP(),
			ClientUserAgent: c.Request().UserAgent(),
			LogTypes:        "运维日志",
			Created:         utils.NowJsonTime(),
			Users:           u.Username,
			Result:          "成功",
			LogContents:     "导出主机接入日志",
		}
		err = global.DBConn.Model(model.OperateLog{}).Create(&operateLog).Error
		if nil != err {
			log.Errorf("DB Error: %v", err)
		}
		return c.Attachment(logPath.LogPath, "主机接入日志.txt")
	} else {
		content, err := service.GetCommandLinuxCon("cat /var/log/messages | grep \" guacd\\[\"")
		if err != nil {
			log.Errorf("DB error: %v", err)
			return FailWithDataOperate(c, 500, "导出主机接入日志失败", "获取本地guacd日志失败: "+err.Error(), err)
		}
		name := "主机接入日志.txt"
		c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+name)
		return c.Stream(http.StatusOK, echo.MIMEOctetStream, bytes.NewReader(content))
	}
}

func ModifyOperationMode(c echo.Context) error {
	value := c.QueryParam("status")
	if value == "true" {
		//cl, err := client.NewClientWithOpts()
		//if err != nil {
		//	return FailWithDataOperate(c, 500, "修改运行模式失败，err: "+err.Error(), "", err)
		//}
		////fmt.Println(cl.ImageList(context.Background(), types.ImageListOptions{}))
		//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		//defer cancel()
		//// 获取容器运行状态
		//container, err := cl.ContainerInspect(ctx, global.CONTAINERID)
		//if err != nil {
		//	return FailWithDataOperate(c, 500, "获取docker容器状态失败: "+err.Error(), "", err)
		//}
		//if container.State.Running {
		//	err := cl.ContainerStop(ctx, global.CONTAINERID, nil)
		//	if err != nil {
		//		return FailWithDataOperate(c, 500, "停止docker容器失败: "+err.Error(), "", err)
		//	}
		//}
		//command := exec.Command("/bin/bash", "-c", `/etc/rc.d/init.d/guacd status`)
		//guacdStatus, _ := command.Output()
		//if strings.Contains(string(guacdStatus), "not") {
		//	status, err := service.GetCommandLinuxCon("/etc/rc.d/init.d/guacd start")
		//	if err != nil && !strings.Contains(string(status), "SUCCESS") {
		//		return FailWithDataOperate(c, 500, "本地guacd开启失败", "开启本地guacd失败", nil)
		//	}
		//}
		err := service.LocalGuacd()
		if err != nil {
			log.Errorf("本地guacd开启失败: %v", err)
			return FailWithDataOperate(c, 500, "修改系统运行模式失败", "调试模式:开启本地guacd失败，err:"+err.Error(), nil)
		}
	} else {
		//command := exec.Command("/bin/bash", "-c", `/etc/rc.d/init.d/guacd status`)
		//guacdStatus, _ := command.Output()
		//if strings.Contains(string(guacdStatus), "guacd is running") {
		//	status, err := service.GetCommandLinuxCon("/etc/rc.d/init.d/guacd stop")
		//	if err != nil && !strings.Contains(string(status), "SUCCESS") {
		//		return FailWithDataOperate(c, 500, "本地guacd关闭失败", "关闭本地guacd失败", nil)
		//	}
		//}
		//cl, err := client.NewClientWithOpts()
		//if err != nil {
		//	return FailWithDataOperate(c, 500, "修改运行模式失败，err: "+err.Error(), "", err)
		//}
		////fmt.Println(cl.ImageList(context.Background(), types.ImageListOptions{}))
		//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		//defer cancel()
		//// 获取容器运行状态
		//container, err := cl.ContainerInspect(ctx, global.CONTAINERID)
		//if err != nil {
		//	return FailWithDataOperate(c, 500, "获取docker容器状态失败: "+err.Error(), "", err)
		//}
		//if !container.State.Running {
		//	err := cl.ContainerStart(ctx, global.CONTAINERID, types.ContainerStartOptions{})
		//	if err != nil {
		//		return FailWithDataOperate(c, 500, "开启docker-guacd成功", "开启docker-guacd成功", nil)
		//	}
		//}
		err := service.DockerGuacd()
		if err != nil {
			log.Errorf("docker-guacd开启失败，err: %v", err)
			return FailWithDataOperate(c, 500, "修改系统运行模式失败", "运行模式:开启docker-guacd失败，err:"+err.Error(), nil)
		}
	}
	err := propertyRepository.UpdateByName(&model.Property{
		Name:  "enable-debug",
		Value: value,
	}, "enable-debug")
	if err != nil {
		log.Errorf("DB error: %v", err)
		return FailWithDataOperate(c, 500, "修改运行模式失败", "", err)
	}
	var s string
	if value == "true" {
		s = "调试模式"
	} else {
		s = "运行模式"
	}
	return SuccessWithOperate(c, "系统设置-修改系统运行模式为"+s, nil)
}
