package api

import (
	"github.com/labstack/echo/v4"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/log"
	"tkbastion/pkg/service"
	"tkbastion/pkg/validator"
	"tkbastion/server/model"
	"tkbastion/server/utils"
)

// NetworkConfigGetEndpoint 获取网络信息
func NetworkConfigGetEndpoint(c echo.Context) error {
	network, err := service.GetCommandLinuxCon("ifconfig")
	if err != nil {
		log.Errorf("Get Command Exec Content Error: %v", err)
		return FailWithDataOperate(c, 500, "", "", nil)
	}
	linesNetwork := strings.Split(string(network), "\n")
	// 读取网卡文件
	fileInfos, err := os.ReadDir("/etc/sysconfig/network-scripts")
	nameMap := make(map[string]bool, 0)
	for i := range fileInfos {
		index := strings.IndexByte(fileInfos[i].Name(), '-')
		if strings.HasPrefix(fileInfos[i].Name(), "ifcfg") && !strings.HasSuffix(fileInfos[i].Name(), "lo") {
			nameMap[fileInfos[i].Name()[index+1:]] = true
		}
	}
	var Net []model.NetworkConfig
	regIp := `\d+\.\d+\.\d+\.\d+`
	reg, _ := regexp.Compile(regIp)
	for i := 0; i < len(linesNetwork); i++ {
		if !strings.Contains(linesNetwork[i], ":") {
			continue
		}
		index := strings.IndexByte(linesNetwork[i], ':')
		name := linesNetwork[i][:index]
		if ok, _ := nameMap[name]; !ok {
			continue
		}
		var NetTemp model.NetworkConfig
		NetTemp.Name = name

		if index1 := strings.Index(linesNetwork[i+1], "inet"); index1 != -1 {
			NetTemp.Ip = string(reg.Find([]byte(linesNetwork[i+1][index1 : index1+19])))
		}
		if index2 := strings.Index(linesNetwork[i+1], "netmask"); index2 != -1 {
			NetTemp.Netmask = string(reg.Find([]byte(linesNetwork[i+1][index2 : index2+22])))
		}
		if index3 := strings.Index(linesNetwork[i+2], "inet6"); index3 != -1 {
			NetTemp.Ipv6 = linesNetwork[i+2][index3+6 : strings.Index(linesNetwork[i+2][index3+6:], " ")+index3+6]
		}
		Net = append(Net, NetTemp)
	}
	for i := 0; i < len(Net); i++ {
		// 读取网卡文件
		fileMap := utils.ReadNetworkFile("/etc/sysconfig/network-scripts/ifcfg-" + Net[i].Name)
		Net[i].Mode = DeleteCommentAndSpace(fileMap["BOOTPROTO"])
		Net[i].Gateway = DeleteCommentAndSpace(fileMap["GATEWAY"])
		if DeleteCommentAndSpace(fileMap["ONBOOT"]) == "no" {
			Net[i].Status = "no"
		} else {
			Net[i].Status = "yes"
		}
		Net[i].Ipv6Gateway = DeleteCommentAndSpace(fileMap["IPV6_DEFAULTGW"])
		if DeleteCommentAndSpace(fileMap["IPV6INIT"]) == "yes" && DeleteCommentAndSpace(fileMap["IPV6_AUTOCONF"]) == "yes" {
			Net[i].Ipv6Status = "dhcp"
			Net[i].Ipv6 = ""
			Net[i].Ipv6Gateway = ""
		} else if DeleteCommentAndSpace(fileMap["IPV6INIT"]) == "yes" && DeleteCommentAndSpace(fileMap["IPV6_AUTOCONF"]) == "no" {
			Net[i].Ipv6Status = "static"
		} else {
			Net[i].Ipv6Status = "disabled"
			Net[i].Ipv6 = ""
			Net[i].Ipv6Gateway = ""
		}
		gateway, _ := service.GetCommandLinuxCon("route -n | grep " + Net[i].Name)
		if len(gateway) == 0 {
			continue
		}
		splits := strings.Split(string(gateway), "\n")
		for j := range splits {
			if len(splits[j]) < 32 {
				continue
			}
			ip := string(reg.Find([]byte(splits[j][16:32])))
			if ip == "0.0.0.0" {
				continue
			}
			Net[i].Gateway = ip
		}
	}
	return SuccessWithOperate(c, "", Net)
}

// NetworkConfigUpdateEndpoint 修改网络配置
func NetworkConfigUpdateEndpoint(c echo.Context) error {
	c.Param("name")
	var item model.NetworkConfig
	if err := c.Bind(&item); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "更新失败", "", err)
	}
	if err := c.Validate(&item); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}
	fileMap := utils.ReadNetworkFile("/etc/sysconfig/network-scripts/ifcfg-" + item.Name)
	if item.Mode == "static" {
		fileMap["BOOTPROTO"] = "static"
		fileMap["IPADDR"] = item.Ip
		fileMap["NETMASK"] = item.Netmask
		fileMap["GATEWAY"] = item.Gateway
		fileMap["ONBOOT"] = item.Status
	} else {
		fileMap["BOOTPROTO"] = "dhcp"
		fileMap["ONBOOT"] = item.Status
	}
	if item.Ipv6Status == "static" {
		fileMap["IPV6INIT"] = "yes"                  //是否开机启用ipv6
		fileMap["IPV6ADDR"] = item.Ipv6              // ipv6地址
		fileMap["IPV6_DEFAULTGW"] = item.Ipv6Gateway // ipv6网关
		fileMap["IPV6_AUTOCONF"] = "no"              // 是否自动获取ipv6地址
		fileMap["IPV6_FAILURE_FATAL"] = "no"         // 配置失败，不会关闭网口
	} else if item.Ipv6Status == "disabled" {
		fileMap["IPV6INIT"] = "no"
		fileMap["IPV6_AUTOCONF"] = "no"
	} else {
		fileMap["IPV6INIT"] = "yes"
		fileMap["IPV6_AUTOCONF"] = "yes"
		fileMap["IPV6_FAILURE_FATAL"] = "no"
	}
	// 写入文件
	err := utils.WriteNetworkFile("/etc/sysconfig/network-scripts/ifcfg-"+item.Name, fileMap)
	if err != nil {
		log.Errorf("WriteNetworkFile Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if item.Status == "no" {
		_, err = service.GetCommandLinuxCon("ifdown " + item.Name)
		if err != nil {
			log.Errorf("Unload Network Card Error:%v", err)
		}
	} else {
		_, err = service.GetCommandLinuxCon("ifdown " + item.Name)
		if err != nil {
			log.Errorf("Unload Network Card Error:%v", err)
		}
		_, err = service.GetCommandLinuxCon("ifup " + item.Name)
		if err != nil {
			log.Errorf("Load Network Card Error:%v", err)
		}
	}
	return SuccessWithOperate(c, "接口配置-修改: 网络配置接口,修改网卡名称["+item.Name+"]", nil)
}

// NetworkConfigRestartEndpoint 重启该网卡
func NetworkConfigRestartEndpoint(c echo.Context) error {
	name := c.Param("name")
	_, err := service.GetCommandLinuxCon("ifdown " + name)
	if err != nil {
		log.Errorf("Unload Network Card Error:%v", err)
		return FailWithDataOperate(c, 500, "关闭网卡失败", "", nil)
	}
	_, err = service.GetCommandLinuxCon("ifup " + name)
	if err != nil {
		log.Errorf("Load Network Card Error:%v", err)
		return FailWithDataOperate(c, 500, "加载网卡失败", "", nil)
	}
	return SuccessWithOperate(c, "接口配置-重启: 网络配置重启,网卡名称["+name+"]重启", nil)
}

// DnsConfigGetEndpoint 获取dns配置
func DnsConfigGetEndpoint(c echo.Context) error {
	dns, err := service.GetCommandLinuxCon("cat /etc/resolv.conf")
	if err != nil {
		log.Errorf("Get Command Exec Content Error: %v", err)
		return FailWithDataOperate(c, 500, "", "", nil)
	}
	linesDns := strings.Split(string(dns), "\n")
	var Dns []string
	regIp := `\d+\.\d+\.\d+\.\d+`
	reg, _ := regexp.Compile(regIp)
	for i := 0; i < len(linesDns); i++ {
		if strings.HasPrefix(linesDns[i], "nameserver") {
			Dns = append(Dns, string(reg.Find([]byte(linesDns[i][11:]))))
		}
	}
	var res string
	if len(Dns) > 0 {
		res = Dns[0]
	}
	return SuccessWithOperate(c, "", res)
}

// DnsConfigUpdateEndpoint 更新dns配置
func DnsConfigUpdateEndpoint(c echo.Context) error {
	dns := c.QueryParam("dns")
	if err := net.ParseIP(dns); err == nil {
		log.Errorf("DNS format error")
		return FailWithData(c, 500, "DNS格式错误:"+dns, nil)
	}
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		log.Errorf("Open file error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Errorf("Close file error: %v", err)
		}
	}(f)
	content := []byte("nameserver " + dns)
	// 写入文件
	err = os.WriteFile("/etc/resolv.conf", content, 0644)
	if err != nil {
		log.Errorf("Write file error: %v", err)
		return FailWithData(c, 500, "修改失败", err)
	}
	return SuccessWithOperate(c, "DNS配置-修改: DNS名称["+dns+"]修改", dns)
}

// CreateStaticRoute 创建静态路由
func CreateStaticRoute(c echo.Context) error {
	var staticRoute model.StaticRoute
	if err := c.Bind(&staticRoute); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}

	if err := c.Validate(&staticRoute); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	if err := net.ParseIP(staticRoute.DestinationAddress); err == nil {
		log.Errorf("ParseIP Error: %v", err)
		return FailWithDataOperate(c, 500, "目的地址: "+staticRoute.DestinationAddress+"格式错误", "", err)
	}
	if err := net.ParseIP(staticRoute.NextHopAddress); err == nil {
		log.Errorf("ParseIP Error: %v", err)
		return FailWithDataOperate(c, 500, "下一跳地址: "+staticRoute.NextHopAddress+"格式错误]", "", err)
	}
	if ok := IsIpv4Mask(staticRoute.SubnetMask); !ok {
		log.Errorf("IPv4格式不正确")
		return FailWithDataOperate(c, 500, "子网掩码: "+staticRoute.SubnetMask+"格式错误", "", nil)
	}
	if staticRoute.NextHopAddress == staticRoute.DestinationAddress {
		return FailWithDataOperate(c, 500, "下一跳地址不能与目的地址相同", "", nil)
	}
	// 判断网卡文件是否存在
	if ok := utils.FileExists("/etc/sysconfig/network-scripts/ifcfg-" + staticRoute.InterfaceName); !ok {
		return FailWithDataOperate(c, 500, "网卡: "+staticRoute.InterfaceName+"不存在", "", nil)
	}
	// 判断是否存在路由文件
	if ok := utils.FileExists("/etc/sysconfig/network-scripts/route-" + staticRoute.InterfaceName); !ok {
		// 创建文件
		_, err := os.Create("/etc/sysconfig/network-scripts/route-" + staticRoute.InterfaceName)
		if err != nil {
			log.Errorf("Create Error: %v", err)
			return FailWithDataOperate(c, 500, "新增失败", "", err)
		}
	}
	// 读取路由文件
	data, err := os.ReadFile("/etc/sysconfig/network-scripts/route-" + staticRoute.InterfaceName)
	if err != nil {
		log.Errorf("OpenFile Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}
	var splits []string
	if len(data) > 0 {
		splits = strings.Split(string(data), "\n")
		for _, v := range splits {
			split := strings.Split(v, " ")
			if len(split) >= 5 && strings.Contains(split[0], staticRoute.DestinationAddress) {
				log.Errorf("Create Content Exists Error: %v", err)
				return FailWithDataOperate(c, 422, "静态路由-新建: ip地址["+staticRoute.DestinationAddress+"]已存在", "", nil)
			}
		}
	}
	// 查看子网掩码位数
	count := 0
	if staticRoute.RouteType == "ipv6" {
		count = 64
	} else {
		netMask := strings.Split(staticRoute.SubnetMask, ".")
		for i := range netMask {
			if netMask[i] == "255" {
				count += 8
			}
		}
	}
	// 写入路由文件
	content := []byte(string(data) + "\n" + staticRoute.DestinationAddress + "/" + strconv.Itoa(count) + " via " + staticRoute.NextHopAddress + " dev " + staticRoute.InterfaceName + " # " + staticRoute.Description)
	// 写入文件
	err = os.WriteFile("/etc/sysconfig/network-scripts/route-"+staticRoute.InterfaceName, content, 0644)
	if err != nil {
		log.Errorf("WriteFile Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}
	return SuccessWithOperate(c, "静态路由-新建: 网络配置-新建静态路由,ip地址["+staticRoute.DestinationAddress+"]", nil)
}

// GetStaticRoute 获取所有静态路由信息
func GetStaticRoute(c echo.Context) error {
	var result []model.StaticRoute
	// 获取所有网卡
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Errorf("GetInterfaces Error: %v", err)
		return FailWithDataOperate(c, 500, "获取网卡失败", "", err)
	}
	for _, v := range interfaces {
		// 判断路由文件是否存在
		if ok := utils.FileExists("/etc/sysconfig/network-scripts/route-" + v.Name); !ok {
			continue
		}
		// 读取路由文件
		data, err := os.ReadFile("/etc/sysconfig/network-scripts/route-" + v.Name)
		if err != nil {
			log.Errorf("OpenFile Error: %v", err)
			return FailWithDataOperate(c, 500, "获取失败", "", err)
		}
		splits := strings.Split(string(data), "\n")
		for i := range splits {
			var temp model.StaticRoute
			if splits[i] != "" {
				if index := strings.Index(splits[i], "#"); index != -1 {
					temp.Description = splits[i][index+2:]
				}
				split := strings.Split(splits[i], " ")
				if len(split) >= 5 {
					if strings.Contains(split[0], "64") {
						temp.RouteType = "ipv6"
					} else {
						temp.RouteType = "ipv4"
					}
					temp.DestinationAddress, temp.SubnetMask = utils.GetMaskByIp(split[0])
					temp.NextHopAddress = split[2]
					temp.InterfaceName = split[4]
				}
				result = append(result, temp)
			}
		}
	}
	return Success(c, result)
}

// DeleteStaticRoute 删除静态路由
func DeleteStaticRoute(c echo.Context) error {
	ip := c.Param("ip")
	// 校验ip
	if ok := net.ParseIP(ip); ok == nil {
		return FailWithDataOperate(c, 422, "网络配置-静态路由: 编辑[ip地址"+ip+"格式错误]", "", nil)
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Errorf("GetInterfaces Error: %v", err)
		return FailWithDataOperate(c, 500, "获取网卡失败", "", err)
	}
	for _, v := range interfaces {
		var newData = ""
		// 判断路由文件是否存在
		if ok := utils.FileExists("/etc/sysconfig/network-scripts/route-" + v.Name); !ok {
			continue
		}
		// 读取路由文件
		data, err := os.ReadFile("/etc/sysconfig/network-scripts/route-" + v.Name)
		if err != nil {
			log.Errorf("OpenFile Error: %v", err)
			return FailWithDataOperate(c, 500, "获取失败", "", err)
		}

		splits := strings.Split(string(data), "\n")
		for i := range splits {
			temp := strings.Split(splits[i], " ")
			if len(temp) != 0 && strings.Contains(temp[0], ip) {
				splits[i] = ""
			}
		}
		for i := range splits {
			if len(splits[i]) != 0 {
				newData += splits[i] + "\n"
			}
		}
		// 写入文件
		err = os.WriteFile("/etc/sysconfig/network-scripts/route-"+v.Name, []byte(newData), 0644)
		if err != nil {
			log.Errorf("WriteFile Error: %v", err)
			return FailWithDataOperate(c, 500, "删除失败", "", err)
		}
	}
	return SuccessWithOperate(c, "网络配置-静态路由-删除: ip地址["+ip+"]", nil)
}

// EditStaticRoute 编辑静态路由
func EditStaticRoute(c echo.Context) error {
	destinationAddress := c.Param("destinationAddress")
	var staticRoute model.StaticRoute
	err := c.Bind(&staticRoute)
	if err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "编辑失败", "", err)
	}

	if err := c.Validate(&staticRoute); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetLogErrMsg(err)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	// 校验ip
	if ok := net.ParseIP(staticRoute.DestinationAddress); ok == nil {
		return FailWithDataOperate(c, 500, "目的地址"+staticRoute.DestinationAddress+"格式错误", "", nil)
	}
	if ok := net.ParseIP(staticRoute.NextHopAddress); ok == nil {
		return FailWithDataOperate(c, 500, "下一跳地址"+staticRoute.NextHopAddress+"格式错误", "", nil)
	}
	if ok := IsIpv4Mask(staticRoute.SubnetMask); !ok {
		log.Errorf("IPv4格式不正确")
		return FailWithDataOperate(c, 500, "子网掩码: "+staticRoute.SubnetMask+"格式错误", "", nil)
	}
	if staticRoute.NextHopAddress == staticRoute.DestinationAddress {
		return FailWithDataOperate(c, 500, "下一跳地址不能与目的地址相同", "", nil)
	}

	// 读取所需要修改的文件
	data, err := os.ReadFile("/etc/sysconfig/network-scripts/route-" + staticRoute.InterfaceName)
	if err != nil {
		log.Errorf("OpenFile Error: %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", err)
	}
	// 查看子网掩码位数
	count := 0
	if staticRoute.RouteType == "ipv6" {
		count = 64
	} else {
		netMask := strings.Split(staticRoute.SubnetMask, ".")
		for i := range netMask {
			if netMask[i] == "255" {
				count += 8
			}
		}
	}
	splits := strings.Split(string(data), "\n")
	for i := range splits {
		if strings.Contains(splits[i], destinationAddress) {
			splits[i] = staticRoute.DestinationAddress + "/" + strconv.Itoa(count) + " via " + staticRoute.NextHopAddress + " dev " + staticRoute.InterfaceName + " # " + staticRoute.Description
		}
	}
	var newData = ""
	for i := range splits {
		newData += splits[i] + "\n"
	}
	// 写入文件
	err = os.WriteFile("/etc/sysconfig/network-scripts/route-"+staticRoute.InterfaceName, []byte(newData), 0644)
	if err != nil {
		log.Errorf("WriteFile Error: %v", err)
		return FailWithDataOperate(c, 500, "编辑失败", "", err)
	}
	return SuccessWithOperate(c, "静态路由-修改: 网络配置修改IP地址["+staticRoute.DestinationAddress+"]", nil)
}

// NetworkDetectionPutEndpoint 网络诊断
func NetworkDetectionPutEndpoint(c echo.Context) error {
	var ipaddress model.IpAddress
	var ErrDomain bool
	var recIpaddress string
	if err := c.Bind(&ipaddress); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "测试失败", "", err)
	}
	// 防止sell注入(连续指令、管道、后台执行)
	character, err := func(command string) (string, bool) {
		specialCharacters := [...]string{";", "&&", "||", "|", "&", ">", ">>", "<", "<<"}
		for _, v := range specialCharacters {
			if find := strings.Contains(command, v); find {
				return v, false
			}
		}
		return "", true
	}(ipaddress.Address)
	if err == false {
		errMsg := ipaddress.Address + "存在非法字符 " + character
		log.Error(errMsg)
		return FailWithDataOperate(c, 500, errMsg, "", errMsg)
	}
	//合法性检测
	errIp := net.ParseIP(ipaddress.Address)
	data := regexp.MustCompile("(\\w*\\.?){3}\\.(com.cn|net.cn|gov.cn|org\\.nz|org.cn|com|net|org|gov|cc|biz|info|cn|co)$").Find([]byte(ipaddress.Address))
	if data != nil {
		ErrDomain = true
		recIpaddress = string(data)
	} else {
		ErrDomain = false
	}
	switch ipaddress.TestType {
	case "ping":
		if errIp != nil {
			command := "ping -w 4 " + ipaddress.Address
			operationResultsCom, err := utils.ExecShell(command)
			if err != nil {
				log.Errorf("exec_shell Error:%V", err)
				errMsg := ipaddress.TestType + ":" + ipaddress.Address
				return FailWithDataOperate(c, 422, "测试连接失败,"+ipaddress.Address+",未知的名称或服务", "网络配置-网络诊断: ["+errMsg+",未知的名称或服务]", errMsg)
			}
			return SuccessWithOperate(c, "网络配置-网络诊断: "+ipaddress.TestType+":"+ipaddress.Address, operationResultsCom)
		}
		if ErrDomain == true {
			command := "ping -w 5 " + recIpaddress
			operationResultsCom, err := utils.ExecShell(command)
			if err != nil {
				log.Errorf("exec_shell Error:%V", err)
				errMsg := ipaddress.TestType + ":" + ipaddress.Address
				return FailWithDataOperate(c, 422, "测试连接失败,"+ipaddress.Address+",未知的名称或服务", "网络配置-网络诊断: ["+errMsg+",未知的名称或服务]", errMsg)
			}
			return SuccessWithOperate(c, "网络诊断-"+ipaddress.TestType+":"+ipaddress.Address, operationResultsCom)
		}
		return FailWithDataOperate(c, 500, "ip/域名格式不正确", "", nil)
	case "traceroute":
		if errIp != nil {
			command := "traceroute " + ipaddress.Address
			operationResultsCom, err := utils.ExecShell(command)
			if err != nil {
				log.Errorf("exec_shell Error:%V", err)
				errMsg := ipaddress.TestType + ":" + ipaddress.Address
				return FailWithDataOperate(c, 422, "测试连接失败,"+ipaddress.Address+",未知的名称或服务", "网络配置-网络诊断: ["+errMsg+",未知的名称或服务]", errMsg)
			}
			return SuccessWithOperate(c, "网络配置-网络诊断:"+ipaddress.TestType+":"+ipaddress.Address, operationResultsCom)
		}
		if ErrDomain == true {
			command := "traceroute " + recIpaddress
			operationResultsCom, err := utils.ExecShell(command)
			if err != nil {
				log.Errorf("exec_shell Error:%V", err)
				errMsg := ipaddress.TestType + ":" + ipaddress.Address
				return FailWithDataOperate(c, 422, "测试连接失败,"+ipaddress.Address+",未知的名称或服务", "网络配置-网络诊断: ["+errMsg+",未知的名称或服务]", errMsg)
			}
			return SuccessWithOperate(c, "网络配置-网络诊断:["+ipaddress.TestType+":"+ipaddress.Address+"]", operationResultsCom)
		}
		return FailWithDataOperate(c, 500, "ip/域名格式不正确", "", nil)
	case "tcpPort":
		msg := ""
		active := utils.Tcping(ipaddress.Address, ipaddress.Port)
		if !active {
			msg = ipaddress.Address + ":" + strconv.Itoa(ipaddress.Port) + " closed"
			return SuccessWithOperate(c, "网络配置-网络诊断: ["+ipaddress.Address+":"+strconv.Itoa(ipaddress.Port)+" close]", msg)
		}
		msg = ipaddress.Address + ":" + strconv.Itoa(ipaddress.Port) + " open"
		return SuccessWithOperate(c, "网络配置-网络诊断: ["+ipaddress.Address+":"+strconv.Itoa(ipaddress.Port)+" open]", msg)
	case "dig":
		// TODO todo 抓包分析
	default:
		return FailWithDataOperate(c, 500, "类型错误", "", "")
	}
	return FailWithDataOperate(c, 422, "测试连接失败,"+ipaddress.Address+",未知的名称或服务", "网络配置-网络诊断: ["+ipaddress.TestType+":"+ipaddress.Address+",未知的名称或服务]", nil)
}

// 判断是否为IPv4子网掩码

func IsIpv4Mask(mask string) bool {
	if mask == "" {
		return false
	}
	// 判断是否为ip
	if err := net.ParseIP(mask); err == nil {
		return false
	}
	// 判断是否为子网掩码
	split := strings.Split(mask, ".")
	if len(split) != 4 {
		return false
	}
	for _, v := range split {
		if v == "0" || v == "255" {
			continue
		}
		return false
	}
	return true
}

// DeleteCommentAndSpace 删除字符串中 # 标注的注释以及空格
func DeleteCommentAndSpace(str string) string {
	index := strings.Index(str, "#")
	if index != -1 {
		str = str[:index]
	} else {
		index = strings.Index(str, "\n")
		if index != -1 {
			str = str[:index]
		}
	}
	return strings.TrimSpace(str)
}
