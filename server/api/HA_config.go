package api

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"tkbastion/pkg/constant"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/model"
	"tkbastion/server/utils"
)

// HaClusterGetEndpoint 获取集群配置
func HaClusterGetEndpoint(c echo.Context) error {
	// 读取集群配置文件
	clusterConfig, err := ReadClusterConfig()
	if err != nil {
		log.Errorf("ReadClusterConfig error: %v", err)
	}
	return Success(c, clusterConfig.ToHAConfigDto())
}

// HaClusterUpdateEndpoint 修改集群配置
func HaClusterUpdateEndpoint(c echo.Context) error {
	oldHaConfig, err := ReadClusterConfig()
	if err != nil {
		log.Errorf("GetClusterConfig error: %v", err)
	}
	var haConfig model.HAConfig
	if err := c.Bind(&haConfig); err != nil {
		log.Errorf("Bind error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if oldHaConfig.Enabled == "true" && haConfig.Enabled == "true" {
		return FailWithDataOperate(c, 500, "HA集群已经启用, 请先关闭集群", "", errors.New("集群已经启用，不可重复开启"))
	}
	if oldHaConfig.Enabled != "true" && haConfig.Enabled != "true" {
		return FailWithDataOperate(c, 500, "HA集群已经关闭, 请先开启集群", "", errors.New("集群已经关闭，不可重复关闭"))
	}
	//host1 := global.Config.Mysql.Hostname
	//port1 := global.Config.Mysql.Port
	//user1 := global.Config.Mysql.Username
	//password1 := global.Config.Mysql.Password
	//database := global.Config.Mysql.Database
	masterIp, err := utils.GetLocalIPByInterfaceName(haConfig.NetInterface)
	if err != nil {
		log.Errorf("getLocalIPByInterfaceName error: %v", err)
	}
	if haConfig.Enabled == "true" {
		// 开启集群
		fmt.Println("开启集群")
		if haConfig.NodeType == "master" {
			// 数据库备份
			//fmt.Println("主机：1.开始数据库备份与恢复")
			//if err := BackupAndSyncDB(host1, strconv.Itoa(port1), user1, password1, haConfig.MariaDBIp, strconv.Itoa(haConfig.MariaDBPort), haConfig.MariaDBUser, haConfig.MariaDBPassword, database); err != nil {
			//	log.Errorf("BackupAndSyncDB error: %v", err)
			//	return FailWithDataOperate(c, 500, "备份数据库信息失败", "", err)
			//}
			//// 等待备机数据库重启
			//time.Sleep(30 * time.Second)
			//fmt.Println("主机：2.开始数据库的同步过程")
			//if err := DBSyncConfigAndEnable(host1, strconv.Itoa(port1), user1, password1, haConfig.MariaDBIp, strconv.Itoa(haConfig.MariaDBPort), haConfig.MariaDBUser, haConfig.MariaDBPassword, database, "10"); err != nil {
			//	log.Errorf("DBSyncConfigAndEnable error: %v", err)
			//	return FailWithDataOperate(c, 500, "开启同步数据库失败", "", err)
			//}
			// 主机数据库集群开启
			fmt.Println("主机：2.开始数据库集群开启")
			if err := StartMasterCluster(masterIp, haConfig.StandbyDeviceIP); err != nil {
				log.Errorf("DBClusterEnable error: %v", err)
				return FailWithDataOperate(c, 500, "开启数据库同步失败", "", err)
			}

			// 文件备份
			fmt.Println("主机：3.开始文件备份与恢复（用户手动完成）")
			// 文件同步
			fmt.Println("主机：4.开始文件的同步过程")
			if err := FileSyncAndEnable("tkbastion", "root", haConfig.StandbyDeviceIP, "tkbastion"); err != nil {
				log.Errorf("FileSyncAndEnable error: %v", err)
				return FailWithDataOperate(c, 500, "开启同步文件失败", "", err)
			}
		} else {
			// 更改配置文件 /etc/my.cnf
			//fmt.Println("备机：2.开始数据库同步过程")
			//if err := DBSyncConfigAndEnable(host1, strconv.Itoa(port1), user1, password1, haConfig.MariaDBIp, strconv.Itoa(haConfig.MariaDBPort), haConfig.MariaDBUser, haConfig.MariaDBPassword, database, "20"); err != nil {
			//	log.Errorf("DBSyncConfigAndEnable error: %v", err)
			//	return FailWithDataOperate(c, 500, "开启同步数据库失败", "", err)
			//}
			//if err := DBSyncConfigAndEnable(haConfig.MariaDBIp, strconv.Itoa(haConfig.MariaDBPort), haConfig.MariaDBUser, mariaPassword, host1, strconv.Itoa(port1), user1, password1, database); err != nil {
			//	log.Errorf("DBSyncConfigAndEnable error: %v", err)
			//	return FailWithDataOperate(c, 500, "开启同步数据库失败", "", err)
			//}
			// 备机数据库加入集群
			fmt.Println("备机：2.开始数据库集群开启")
			if err := StartSlaveCluster(masterIp, haConfig.StandbyDeviceIP); err != nil {
				log.Errorf("DBClusterEnable error: %v", err)
				return FailWithDataOperate(c, 500, "开启数据库同步失败", "", err)
			}
			// 文件备份
			fmt.Println("备机：3.开始文件备份与恢复（用户手动完成）")
			// 文件同步
			fmt.Println("备机：4.开始文件的同步过程")
			if err := FileSyncAndEnable("tkbastion", "root", haConfig.StandbyDeviceIP, "tkbastion"); err != nil {
				log.Errorf("FileSyncAndEnable error: %v", err)
				return FailWithDataOperate(c, 500, "开启同步文件失败", "", err)
			}
		}
		// 开启keepalived监测
		fmt.Println("监测服务开启的配置过程")
		if err := ServiceMonitorConfigAndEnable(haConfig); err != nil {
			log.Errorf("ServiceMonitorConfigAndEnable error: %v", err)
			return FailWithDataOperate(c, 500, "开启监测服务失败", "", err)
		}
		// 加密密码
		mariaDBPassword, err := utils.AesEncryptCBC([]byte(haConfig.MariaDBPassword), global.Config.EncryptionPassword)
		if err != nil {
			log.Errorf("AesEncryptCBC error: %v", err)
		}
		haConfig.MariaDBPassword = base64.StdEncoding.EncodeToString(mariaDBPassword)
		//rootPassword, err := utils.AesEncryptCBC([]byte(haConfig.RootPassword), global.Config.EncryptionPassword)
		//if err != nil {
		//	log.Errorf("AesEncryptCBC error: %v", err)
		//}
		//haConfig.RootPassword = base64.StdEncoding.EncodeToString(rootPassword)
		err = WriteClusterConfig(oldHaConfig, haConfig)
		if err != nil {
			log.Errorf("UpdateClusterConfig error: %v", err)
			return FailWithDataOperate(c, 500, "集群服务已开启,写入数据库失败", "", err)
		}
		return SuccessWithOperate(c, "集群配置-编辑: 开启HA集群成功", haConfig.ToHAConfigDto())
		//return Success(c, haConfig.ToHAConfigDto())
	} else {
		// 关闭集群
		// (1) 关闭数据库同步
		//fmt.Println("关闭数据库同步")
		//if err := DBSyncDisable(host1, strconv.Itoa(port1), user1, password1, haConfig.MariaDBIp, strconv.Itoa(haConfig.MariaDBPort), haConfig.MariaDBUser, haConfig.MariaDBPassword, database); err != nil {
		//	log.Errorf("DBSyncDisable error: %v", err)
		//	return FailWithDataOperate(c, 500, "关闭同步数据库失败", "", err)
		//}
		// 关闭数据库集群
		if err := CloseMariaDBSync(); err != nil {
			log.Errorf("CloseMariaDBSync error: %v", err)
			return FailWithDataOperate(c, 500, "关闭数据库同步失败", "", err)
		}
		// (2) 关闭文件同步
		fmt.Println("关闭文件同步")
		if err := FileSyncAndDisable(); err != nil {
			log.Errorf("FileSyncAndDisable error: %v", err)
			return FailWithDataOperate(c, 500, "关闭同步文件失败", "", err)
		}
		// (3) 关闭监测服务
		fmt.Println("关闭监测服务")
		if err := ServiceMonitorDisable(); err != nil {
			log.Errorf("ServiceMonitorDisable error: %v", err)
			return FailWithDataOperate(c, 500, "关闭监测服务失败", "", err)
		}
		// 重启防火墙
		fmt.Println("防火墙重启的配置过程")
		if _, err := utils.ExecShell("systemctl restart firewalld"); err != nil {
			log.Errorf("systemctl restart firewalld error: %v", err)
			return FailWithDataOperate(c, 500, "重启防火墙失败", "", err)
		}
		// 更新数据库
		// 加密密码
		mariaDBPassword, err := utils.AesEncryptCBC([]byte(haConfig.MariaDBPassword), global.Config.EncryptionPassword)
		if err != nil {
			log.Errorf("AesEncryptCBC error: %v", err)
		}
		haConfig.MariaDBPassword = base64.StdEncoding.EncodeToString(mariaDBPassword)
		//rootPassword, err := utils.AesEncryptCBC([]byte(haConfig.RootPassword), global.Config.EncryptionPassword)
		//if err != nil {
		//	log.Errorf("AesEncryptCBC error: %v", err)
		//}
		//haConfig.RootPassword = base64.StdEncoding.EncodeToString(rootPassword)
		err = WriteClusterConfig(oldHaConfig, haConfig)
		if err != nil {
			log.Errorf("UpdateClusterConfig error: %v", err)
			return FailWithDataOperate(c, 500, "集群服务已关闭,写入数据库失败", "", err)
		}
		return SuccessWithOperate(c, "集群配置-编辑: 关闭HA集群成功", nil)
	}
}

// StartMasterCluster 开启主节点数据库同步的集群
func StartMasterCluster(masterIp, StandbyDeviceIP string) (err error) {

	// 禁用selinux
	_, _ = utils.ExecShell("setenforce 0")
	// 修改数据库配置文件
	if err = modifyMariaDbConfig(masterIp, StandbyDeviceIP); err != nil {
		return err
	}
	// 开放相应端口
	_, err = utils.ExecShell("firewall-cmd --zone=public --add-port=4567/tcp --permanent")
	if err != nil {
		log.Errorf("开放4567端口: %v", err)
	}
	_, err = utils.ExecShell("firewall-cmd --zone=public --add-port=3306/tcp --permanent")
	if err != nil {
		log.Errorf("开放3306端口: %v", err)
	}
	// 重启防火墙
	_, err = utils.ExecShell("systemctl restart firewalld")
	if err != nil {
		log.Errorf("Exec systemctl restart firewalld error: %v", err)
	}
	// 关闭mariadb服务
	_, _ = utils.ExecShell("systemctl stop mariadb")
	// 启动集群
	_, err = utils.ExecShell("galera_new_cluster")
	if err != nil {
		log.Errorf("Exec galera_new_cluster error: %v", err)
		return err
	}
	return nil
}

// StartSlaveCluster 启动从节点数据库加入主节点集群
func StartSlaveCluster(masterIp, StandbyDeviceIP string) (err error) {
	// 禁用selinux
	_, _ = utils.ExecShell("setenforce 0")
	// 修改数据库配置文件
	if err = modifyMariaDbConfig(masterIp, StandbyDeviceIP); err != nil {
		return err
	}
	// 开放相应端口
	_, err = utils.ExecShell("firewall-cmd --zone=public --add-port=4567/tcp --permanent")
	if err != nil {
		log.Errorf("开放4567端口: %v", err)
	}
	_, err = utils.ExecShell("firewall-cmd --zone=public --add-port=3306/tcp --permanent")
	if err != nil {
		log.Errorf("开放3306端口: %v", err)
	}
	// 重启防火墙
	_, err = utils.ExecShell("systemctl restart firewalld")
	if err != nil {
		log.Errorf("systemctl restart firewalld error: %v", err)
	}
	// 关闭mariadb服务
	_, _ = utils.ExecShell("systemctl stop mariadb")
	// 启动集群
	_, err = utils.ExecShell("systemctl start mariadb")
	if err != nil {
		log.Errorf("执行启动数据库命令失败: %v,请耐心等待重启,若失败请检查服务器", err)
	}
	return nil
}

// CloseMariaDBSync 关闭数据库同步
func CloseMariaDBSync() error {
	// 重置配置文件
	if err := ResetMariaDbConfig(); err != nil {
		log.Errorf("ResetMariaDBConfig error: %v", err)
		return err
	}
	// 关闭相应端口
	_, err := utils.ExecShell("firewall-cmd --zone=public --remove-port=4567/tcp --permanent")
	if err != nil {
		log.Errorf("关闭4567端口错误: %v", err)
	}
	//_, err = utils.ExecShell("firewall-cmd --zone=public --remove-port=3306/tcp --permanent")
	//if err != nil {
	//	return err
	//}
	// 重启数据库
	_, err = utils.ExecShell("systemctl restart mariadb")
	if err != nil {
		log.Errorf("systemctl restart mariadb error: %v", err)
		return err
	}
	return nil
}

// FileSyncAndEnable 文件的同步与启动
func FileSyncAndEnable(password1, user2, host2, password2 string) (err error) {
	// 检测Rsync同步软件是否已安装
	fmt.Println("检测Rsync同步软件是否已安装")
	result, err := utils.ExecShell("rpm -qa | grep rsync")
	if err != nil {
		log.Errorf("ExecShell error: %v", err)
		return err
	}
	if result == "" {
		log.Errorf("rsync not installed")
		return errors.New("rsync not installed")
	}
	// 检测inotify是否安装
	fmt.Println("检测inotify是否安装")
	result, err = utils.ExecShell("rpm -qa | grep inotify-tools")
	if err != nil {
		log.Errorf("ExecShell error: %v", err)
	}
	if result == "" {
		log.Errorf("inotify-tools not installed")
		return errors.New("inotify-tools not installed")
	}
	// /etc/rsyncd.conf
	conf := `uid=root
gid=root
max connections=100
use chroot=true
log file=/var/log/rsyncd.log
motd file = /etc/rsyncd.motd
transfer logging = true
hosts allow=` + host2 + `
[tkbastion]
path=/data/tkbastion/
read only = no 
list = yes 
auth users = root 
secrets file=/etc/rsyncd.pwd
[backup]
path=/tkbastion/backup/
comment = httpd conf
read only = no
list = yes
auth users = root
secrets file=/etc/rsyncd.pwd
`
	fmt.Println("写入/etc/rsyncd.conf")
	err = utils.WriteFileDirect("/etc/rsyncd.conf", conf)
	if err != nil {
		return err
	}
	// /etc/rsyncd.motd TODO
	fmt.Println("写入/etc/rsyncd.motd")
	err = utils.WriteFileDirect("/etc/rsyncd.motd", "tkbastion 欢迎您")
	if err != nil {
		return err
	}
	// /etc/rsyncd.pwd
	fmt.Println("写入/etc/rsyncd.pwd")
	err = utils.WriteFileDirect("/etc/rsyncd.pwd", user2+":"+password2)
	if err != nil {
		return err
	}
	if _, err := utils.ExecShell("chmod 600 /etc/rsyncd.pwd"); err != nil {
		return err
	}
	// /etc/rsyncd.pwd2
	fmt.Println("写入/etc/rsyncd.pwd2")
	err = utils.WriteFileDirect("/etc/rsyncd.pwd2", password1)
	if err != nil {
		return err
	}
	if _, err := utils.ExecShell("chmod 600 /etc/rsyncd.pwd2"); err != nil {
		return err
	}
	// 开启rsyncd服务
	fmt.Println("开启rsyncd服务")
	_, err = utils.ExecShell("rsync --daemon --config=/etc/rsyncd.conf")
	if err != nil {
		log.Errorf("ExecShell error: %v", err)
		return err
	}
	// 开启rsync端口
	_, err = utils.ExecShell("firewall-cmd --zone=public --add-port=873/tcp --permanent")
	if err != nil {
		log.Errorf("ExecShell error: %v", err)
		return err
	}
	// 添加开机自启
	fmt.Println("添加开机自启")
	_, err = utils.ExecShell("sed -i '$a rsync --daemon --config=/etc/rsyncd.conf' /etc/rc.d/rc.local")
	if err != nil {
		log.Errorf("ExecShell error: %v", err)
		return err
	}
	// 检测Rsync服务是否启动
	fmt.Println("检测Rsync服务是否启动")
	result, err = utils.ExecShell("ps -ef | grep rsync | grep -v grep")
	if err != nil {
		log.Errorf("ExecShell error: %v", err)
	}
	if result == "" {
		log.Errorf("rsync not installed")
		return errors.New("rsync service start failed")
	}
	// 编写监控同步文件脚本
	monitorStr := `#!/bin/bash
TkbastionPath=/data/tkbastion/
BackupPath=/tkbastion/backup/
Server=` + host2 + `
/usr/bin/inotifywait -mrq --timefmt '%d/%m/%y %H:%M' --format '%T %w%f%e' -e close_write,delete,create,attrib,move  $TkbastionPath $BackupPath  |
while read line
do
    if [[ $line =~ $TkbastionPath ]];then
    rsync -vzrtopg --progress --delete  $TkbastionPath  root@$Server::tkbastion  --password-file=/etc/rsyncd.pwd2
    elif [[ $line =~ $BackupPath ]];then
    rsync -vzrtopg --progress --delete  $BackupPath  root@$Server::backup --password-file=/etc/rsyncd.pwd2
    else
    echo $line >> /var/log/inotify.log
    fi
done
`
	fmt.Println("写入监控同步文件脚本")
	err = utils.WriteFileDirect("/tkbastion/tools/scripts/inotify.sh", monitorStr)
	if err != nil {
		return err
	}
	_, _ = utils.ExecShell("ps -ef | grep inotify.sh | grep -v grep | awk '{print $2}' | xargs kill -9")
	//启动监控同步脚本
	fmt.Println("启动监控同步脚本")
	cmd := exec.Command("bash", "-c", "nohup sh /tkbastion/tools/scripts/inotify.sh &")
	if err := cmd.Run(); err != nil {
		log.Errorf("ExecShell error: %v", err)
		return err
	}
	// 检查运行是否成功
	fmt.Println("检查运行是否成功")
	result, err = utils.ExecShell("ps -ef | grep inotify.sh | grep -v grep")
	if err != nil {
		log.Errorf("check inotify.sh status error: %v", err)
		return err
	}
	if result == "" {
		log.Errorf("inotify.sh not activating")
		return errors.New("inotify.sh start failed")
	}
	return nil
}

// FileSyncAndDisable 文件同步的关闭
func FileSyncAndDisable() (err error) {
	// 关闭监控同步脚本
	_, _ = utils.ExecShell("ps -ef | grep inotify.sh | grep -v grep | awk '{print $2}' | xargs kill -9")
	// 关闭Rsync服务
	_, _ = utils.ExecShell("ps -ef | grep rsync | grep -v grep | awk '{print $2}' | xargs kill -9")
	// 删除开机自启
	_, _ = utils.ExecShell(`sed -i '/rsync --daemon --config=\/etc\/rsyncd.conf/d' /etc/rc.d/rc.local`)
	// 关闭文件服务监测的端口
	_, _ = utils.ExecShell("firewall-cmd --zone=public --remove-port=873/tcp --permanent")
	// TODO 是否删除文件待定
	return nil
}

// ServiceMonitorConfigAndEnable 服务监测的配置与启动
func ServiceMonitorConfigAndEnable(haConfig model.HAConfig) (err error) {
	// 检测keepalived是否安装
	result, err := utils.ExecShell("rpm -qa | grep keepalived")
	if err != nil {
		log.Errorf("ExecShell error: %v", err)
		return err
	}
	if result == "" {
		return errors.New("keepalived not installed")
	}
	priority := "90"
	if haConfig.NodeType == "master" {
		priority = "100"
		haConfig.NodeType = "MASTER"
	} else {
		haConfig.NodeType = "BACKUP"
	}
	var port string
	index := strings.Index(global.Config.Server.Addr, ":")
	if index == -1 {
		port = "8088"
	} else {
		port = global.Config.Server.Addr[index+1:]
	}

	keepalivedConf := `! Configuration File for keepalived
global_defs {
   notification_email {
     acassen@firewall.loc
     failover@firewall.loc
     sysadmin@firewall.loc
   }
   notification_email_from Alexandre.Cassen@firewall.loc
   smtp_server 192.168.200.1
   smtp_connect_timeout 30
   router_id LVS_DEVEL
   vrrp_skip_check_adv_addr
   script_user root 
   enable_script_security  
   vrrp_garp_interval 0
   vrrp_gna_interval 0
}
vrrp_script tkbastion_check {
  script "/tkbastion/tools/scripts/tkbastion_check.sh  ` + haConfig.VirtualIP + ` ` + port + `"
  interval 1
}
vrrp_instance VI_1 {
    state ` + haConfig.NodeType + `
    interface  ` + haConfig.NetInterface + ` 
    virtual_router_id 101
    priority ` + priority + `
    advert_int 1
	nopreempt
    authentication {
        auth_type PASS
        auth_pass password
    }
    virtual_ipaddress {
        ` + haConfig.VirtualIP + `/24 dev ` + haConfig.NetInterface + `  
    }
    track_script {
      tkbastion_check
    }
	notify_master "/tkbastion/tools/scripts/master.sh"
	notify_backup "/tkbastion/tools/scripts/backup.sh ` + haConfig.StandbyDeviceIP + ` ` + port + `"
	notify_fault "/tkbastion/tools/scripts/fault.sh ` + haConfig.StandbyDeviceIP + ` ` + port + `"
}
`
	err = utils.WriteFileDirect("/etc/keepalived/keepalived.conf", keepalivedConf)
	if err != nil {
		return err
	}
	// 重新启动keepalived
	_, err = utils.ExecShell("systemctl restart keepalived")
	if err != nil {
		log.Errorf("ExecShell error: %v", err)
		return err
	}
	_, err = utils.ExecShell("systemctl enable keepalived")
	if err != nil {
		log.Errorf("ExecShell error: %v", err)
		return err
	}
	// 开启keepalived的端口
	_, err = utils.ExecShell("firewall-cmd --zone=public --add-port=112/tcp --permanent")
	if err != nil {
		log.Errorf("ExecShell error: %v", err)
		return err
	}
	// 检查keepalived是否启动
	result, err = utils.ExecShell("ps -ef | grep keepalived | grep -v grep")
	if err != nil {
		log.Errorf("ExecShell error: %v", err)
		return err
	}
	if result == "" {
		return errors.New("keepalived start failed")
	}
	return nil
}

// ServiceMonitorDisable 服务监测的关闭
func ServiceMonitorDisable() (err error) {
	// 关闭keepalived服务
	_, _ = utils.ExecShell("systemctl stop keepalived")
	// 关闭开机自启
	_, _ = utils.ExecShell("systemctl disable keepalived")
	// 关闭keepalived的端口
	_, _ = utils.ExecShell("firewall-cmd --zone=public --remove-port=112/udp --permanent")
	return nil
}

// 修改MariaDb配置文件
// Path: /etc/my.cnf.d/server.cnf
func modifyMariaDbConfig(masterIp, StandbyDeviceIP string) error {
	// 1. 读取配置文件,保存成字符串
	path := "/etc/my.cnf.d/server.cnf"
	var mysqld = `
#
# These groups are read by MariaDB server.
# Use it for options that only the server (but not clients) should see
#
# See the examples of server my.cnf files in /usr/share/mysql/
#

# this is read by the standalone daemon and embedded servers
[server]

# this is only for the mysqld standalone daemon
[mysqld]

#
# * Galera-related settings
#
[galera]
wsrep_on=ON
wsrep_provider=/usr/lib64/galera/libgalera_smm.so
wsrep_cluster_name=galera_cluster
wsrep_cluster_address="gcomm://` + masterIp + `,` + StandbyDeviceIP + `"
wsrep_node_name=` + masterIp + `
wsrep_node_address=` + masterIp + `
binlog_format=ROW
default_storage_engine=InnoDB
innodb_autoinc_lock_mode=2
innodb_doublewrite=1
query_cache_size=0
wsrep_sst_method=rsync
# Mandatory settings
#wsrep_on=ON
#wsrep_provider=
#wsrep_cluster_address=
#binlog_format=row
#default_storage_engine=InnoDB
#innodb_autoinc_lock_mode=2
#
# Allow server to accept connections on all interfaces.
#
#bind-address=0.0.0.0
#
# Optional setting
#wsrep_slave_threads=1
#innodb_flush_log_at_trx_commit=0

# this is only for embedded server
[embedded]

# This group is only read by MariaDB servers, not by MySQL.
# If you use the same .cnf file for MySQL and MariaDB,
# you can put MariaDB-only options here
[mariadb]

# This group is only read by MariaDB-10.3 servers.
# If you use the same .cnf file for MariaDB of different versions,
# use this group for options that older servers don't understand
`
	// 2. 写入配置文件
	if err := utils.WriteFileDirect(path, mysqld); err != nil {
		return err
	}
	return nil
}

func ResetMariaDbConfig() error {
	// 1. 读取配置文件,保存成字符串
	path := "/etc/my.cnf.d/server.cnf"
	var mysqld = `
#
# These groups are read by MariaDB server.
# Use it for options that only the server (but not clients) should see
#
# See the examples of server my.cnf files in /usr/share/mysql/
#

# this is read by the standalone daemon and embedded servers
[server]

# this is only for the mysqld standalone daemon
[mysqld]

#
# * Galera-related settings
#
[galera]
# Mandatory settings
#wsrep_on=ON
#wsrep_provider=
#wsrep_cluster_address=
#binlog_format=row
#default_storage_engine=InnoDB
#innodb_autoinc_lock_mode=2
#
# Allow server to accept connections on all interfaces.
#
#bind-address=0.0.0.0
#
# Optional setting
#wsrep_slave_threads=1
#innodb_flush_log_at_trx_commit=0

# this is only for embedded server
[embedded]

# This group is only read by MariaDB servers, not by MySQL.
# If you use the same .cnf file for MySQL and MariaDB,
# you can put MariaDB-only options here
[mariadb]

# This group is only read by MariaDB-10.3 servers.
# If you use the same .cnf file for MariaDB of different versions,
# use this group for options that older servers don't understand
`
	// 2. 写入配置文件
	if err := utils.WriteFileDirect(path, mysqld); err != nil {
		return err
	}
	return nil
}

// 读取配置文件
//ConfigPath              = "/tkbastion/config"
//ClusterConfig           = "cluster-config"

func ReadClusterConfig() (haConfig model.HAConfig, err error) {
	// 1. 读取配置文件
	config := utils.ReadNetworkFile(path.Join(constant.ConfigPath, constant.ClusterConfig))
	// 2. 解析配置文件
	haConfig.Enabled = config["Enabled"]
	haConfig.NodeType = config["NodeType"]
	haConfig.VirtualIP = config["VirtualIP"]
	haConfig.StandbyDeviceIP = config["StandbyDeviceIP"]
	haConfig.NetInterface = config["NetInterface"]
	haConfig.MariaDBIp = config["MariaDBIp"]
	haConfig.MariaDBPort, _ = strconv.Atoi(config["MariaDBPort"])
	haConfig.MariaDBUser = config["MariaDBUser"]
	haConfig.MariaDBPassword = config["MariaDBPassword"]
	return haConfig, nil
}
func WriteClusterConfig(oldConfig, haConfig model.HAConfig) (err error) {
	config := `## 集群配置文件
[HAConfig]
Enabled=` + haConfig.Enabled + `
NodeType=` + haConfig.NodeType + `
VirtualIP=` + haConfig.VirtualIP + `
StandbyDeviceIP=` + haConfig.StandbyDeviceIP + `
NetInterface=` + haConfig.NetInterface + `
MariaDBIp=` + haConfig.MariaDBIp + `
MariaDBPort=` + strconv.Itoa(haConfig.MariaDBPort) + `
MariaDBUser=` + haConfig.MariaDBUser + `
MariaDBPassword=` + haConfig.MariaDBPassword + `
`
	// 2. 写入配置文件
	if err := utils.WriteFileDirect(path.Join(constant.ConfigPath, constant.ClusterConfig), config); err != nil {
		return err
	}
	return nil
}

// BackupAndSyncDB 备份并同步数据库
//func BackupAndSyncDB(host1, port1, user1, password1, host2, port2, user2, password2, databaseName string) (err error) {
//	// 连接主机数据库
//	db, err := sql.Open("mysql", user1+":"+password1+"@tcp("+host1+":"+port1+")/"+databaseName)
//	if err != nil {
//		log.Errorf("sql.Open error: %v", err)
//	}
//	defer func(db *sql.DB) {
//		err := db.Close()
//		if err != nil {
//			log.Errorf("Error closing database: %v", err)
//		}
//	}(db)
//	// 锁定数据库
//	_, err = db.Exec("flush tables with read lock;")
//	if err != nil {
//		log.Errorf("flush tables with read lock error: %v", err)
//	}
//	// 备份数据库
//	var backupPath string
//	// TODO 暂时先备份整个数据库需要忽略部分数据表，暂时只忽略了ha_config,usage_mem,usage_disk,usage_cpu
//	backupPath = "/tkbastion/" + databaseName + "_" + time.Now().Format("20060102150405") + ".sql"
//	mysqldumpCmd := `mysqldump -h ` + host1 + ` -P ` + port1 + ` -u ` + user1 + ` -p` + password1 + ` --databases ` + databaseName + ` --ignore-table=` + databaseName + `.ha_config ` + ` --ignore-table=` + databaseName + `.usage_cpu ` + ` --ignore-table=` + databaseName + `.usage_mem ` + ` --ignore-table=` + databaseName + `.usage_disk` + ` >` + backupPath
//	//mysqldumpCmd := `mysqldump -h ` + host1 + ` -P ` + port1 + ` -u ` + user1 + ` -p` + password1 + ` --databases ` + databaseName + ` --ignore-table=` + databaseName + `.ha_config  >` + backupPath
//	_, err = utils.ExecShell(mysqldumpCmd)
//	fmt.Println(mysqldumpCmd)
//	if err != nil {
//		log.Errorf("ExecShell error: %v", err)
//		return err
//	}
//	// 恢复数据库
//	mysqlRecoverCmd := `mysql -h` + host2 + ` -P` + port2 + ` -u` + user2 + ` -p` + password2 + ` ` + databaseName + ` <` + backupPath
//	_, err = utils.ExecShell(mysqlRecoverCmd)
//	fmt.Println(mysqlRecoverCmd)
//	if err != nil {
//		log.Errorf("ExecShell error: %v", err)
//		return err
//	}
//
//	// 解锁数据库
//	_, err = db.Exec("unlock tables;")
//	if err != nil {
//		log.Errorf("unlock tables error: %v", err)
//		return err
//	}
//	// 删除备份的数据库文件
//	defer func(path string) {
//		if err := os.Remove(backupPath); err != nil {
//			log.Errorf("os.Remove error: %v", err)
//		}
//	}(backupPath)
//	return nil
//}

// DBSyncConfigAndEnable 主机数据库同步的状态配置与启动
//func DBSyncConfigAndEnable(host1, port1, user1, password1, host2, port2, user2, password2, databaseName, id string) (err error) {
//	if err := ChangeDBConfig(databaseName, id); err != nil {
//		return err
//	}
//	// 开启主从同步
//	fmt.Println("打开本机数据库")
//	var db1 *sql.DB
//	if db1, err = sql.Open("mysql", user1+":"+password1+"@tcp("+host1+":"+port1+")/"+databaseName); err != nil {
//		log.Errorf("Error opening database: %v", err)
//		return err
//	}
//	//if _, err := db1.Exec("GRANT ALL PRIVILEGES ON *.* TO '" + user1 + "'@'%' IDENTIFIED BY '" + password1 + "' WITH GRANT OPTION;"); err != nil {
//	//	log.Errorf("Error GRANT ALL PRIVILEGES: %v", err)
//	//}
//	//// 加读锁
//	//_, err = db1.Exec("flush tables with read lock;")
//	//if err != nil {
//	//	log.Errorf("flush tables with read lock error: %v", err)
//	//	return err
//	//}
//	defer func(db *sql.DB) {
//		////关锁
//		//_, err = db.Exec("unlock tables;")
//		//if err != nil {
//		//	log.Errorf("unlock tables error: %v", err)
//		//}
//		err := db.Close()
//		if err != nil {
//			log.Errorf("Error closing database: %v", err)
//		}
//	}(db1)
//	// grant replication slave on *.* to "root"@'%' identified by "Zjw195126@";
//	fmt.Println("添加主从同步账号")
//	_, err = db1.Exec(`grant replication slave on *.* to 'synchronization'@'%' identified by 'Tkbastion';`)
//	if err != nil {
//		log.Errorf("Error executing command: %v", err)
//		return err
//	}
//	rows1, err := db1.Query(`show master status;`)
//	if err != nil {
//		log.Errorf("Error executing command: %v", err)
//	}
//	defer func(rows *sql.Rows) {
//		err := rows.Close()
//		if err != nil {
//			log.Errorf("Error closing database: %v", err)
//		}
//	}(rows1)
//
//	var file1 string
//	var position1 string
//	var binlogDoDb string
//	var binlogIgnoreDb string
//	for rows1.Next() {
//		err = rows1.Scan(&file1, &position1, &binlogDoDb, &binlogIgnoreDb)
//		if err != nil {
//			log.Errorf("Error scanning rows: %v", err)
//			return err
//		}
//	}
//	fmt.Println("获取文件名与位置")
//	fmt.Println("file1:", file1, "position1:", position1)
//	if file1 == "" || position1 == "" {
//		log.Errorf("Error file or position is null")
//		return errors.New("备机数据库同步状态配置尚未完成")
//	}
//
//	fmt.Println("打开外机数据库")
//	var db2 *sql.DB
//	if db2, err = sql.Open("mysql", user2+":"+password2+"@tcp("+host2+":"+port2+")/"+databaseName); err != nil {
//		log.Errorf("sql.Open error: %v", err)
//		return err
//	}
//	//if _, err = db2.Exec("GRANT ALL PRIVILEGES ON *.* TO '" + user2 + "'@'%' IDENTIFIED BY '" + password2 + "' WITH GRANT OPTION;"); err != nil {
//	//	log.Errorf("Error GRANT ALL PRIVILEGES: %v", err)
//	//}
//	//// 加读锁
//	//_, err = db2.Exec("flush tables with read lock;")
//	//if err != nil {
//	//	log.Errorf("flush tables with read lock error: %v", err)
//	//	return err
//	//}
//	defer func(db *sql.DB) {
//		//// 关锁
//		//_, err = db.Exec("unlock tables;")
//		//if err != nil {
//		//	log.Errorf("unlock tables error: %v", err)
//		//}
//		err := db.Close()
//		if err != nil {
//			log.Errorf("Error closing database: %v", err)
//		}
//	}(db2)
//
//	_, err = db2.Exec(`stop slave;`)
//	if err != nil {
//		log.Errorf("Error executing command: %v", err)
//	}
//	sqlStr := `change master to master_host='` + host1 + `',master_port=` + port1 + `,master_user='synchronization',master_password='Tkbastion',master_log_file='` + file1 + `',master_log_pos=` + position1 + `;`
//	fmt.Println(sqlStr)
//	fmt.Println("主从同步配置添加")
//	_, err = db2.Exec(sqlStr)
//	if err != nil {
//		log.Errorf("Error executing command: %v", err)
//		return err
//	}
//	_, err = db2.Exec(`start slave;`)
//	if err != nil {
//		log.Errorf("Error executing command: %v", err)
//		return err
//	}
//
//	return nil
//}

// DBSyncDisable 关闭主从同步
//func DBSyncDisable(host1, port1, user1, password1, host2, port2, user2, password2, databaseName string) (err error) {
//	db1, err := sql.Open("mysql", user1+":"+password1+"@tcp("+host1+":"+port1+")/"+databaseName)
//	if err != nil {
//		log.Errorf("sql.Open error: %v", err)
//	}
//	defer func(db *sql.DB) {
//		err := db.Close()
//		if err != nil {
//			log.Errorf("Error closing database: %v", err)
//		}
//	}(db1)
//	_, err = db1.Exec(`stop slave;`)
//	if err != nil {
//		log.Errorf("Error executing command: %v", err)
//	}
//	db2, err := sql.Open("mysql", user2+":"+password2+"@tcp("+host2+":"+port2+")/"+databaseName)
//	if err != nil {
//		log.Errorf("sql.Open error: %v", err)
//	}
//	defer func(db *sql.DB) {
//		err := db.Close()
//		if err != nil {
//			log.Errorf("Error closing database: %v", err)
//		}
//	}(db2)
//	_, err = db2.Exec(`stop slave;`)
//	if err != nil {
//		log.Errorf("Error executing command: %v", err)
//	}
//	return nil
//}

// ChangeDBConfig 更改数据库的配置文件
//func ChangeDBConfig(database, id string) (err error) {
//	// 更改配置文件 /etc/my.cnf
//	var offset string
//	if id == "10" {
//		offset = "1"
//	} else {
//		offset = "2"
//	}
//	myCof := `# This group is read both by the client and the server
//# use it for options that affect everything
//#
//[client-server]
//#
//# include *.cnf from the config directory
//#
//!includedir /etc/my.cnf.d
//[mysqld]
//character-set-server=utf8
//collation-server=utf8_general_ci
//log-bin=mysql-bin
//server-id=` + id + `
//binlog-ignore-db=mysql
//replicate-do-db=` + database + `
//replicate-ignore-db=mysql
//replicate-ignore-db=information_schema
//replicate-ignore-table=` + database + `.ha_config
//replicate-ignore-table=` + database + `.usage_cpu
//replicate-ignore-table=` + database + `.usage_disk
//replicate-ignore-table=` + database + `.usage_mem
//replicate-ignore-table=` + database + `.system_alarm_log
//auto-increment-increment=2
//auto-increment-offset=` + offset + `
//skip-name-resolve
//bind-address=0.0.0.0
//`
//	if err := utils.WriteFileDirect("/etc/my.cnf", myCof); err != nil {
//		log.Errorf("utils.WriteFileDirect error: %v", err)
//		return err
//	}
//	cmd := exec.Command("systemctl", "restart", "mariadb")
//	err = cmd.Run()
//	if err != nil {
//		log.Errorf("cmd.Run error: %v", err)
//		return err
//	}
//	return nil
//}
