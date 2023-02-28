package model

type HAConfig struct {
	Enabled         string `json:"enabled" `
	NodeType        string `json:"nodeType" `
	VirtualIP       string `json:"virtualIP" `
	StandbyDeviceIP string `json:"standbyDeviceIP"  `
	NetInterface    string `json:"netInterface" `
	MariaDBIp       string `json:"mariaDBIp" `
	MariaDBPort     int    `json:"mariaDBPort"`
	MariaDBUser     string `json:"mariaDBUser" `
	MariaDBPassword string `json:"mariaDBPassword"`
}

type HAConfigDto struct {
	Enabled         string `json:"enabled" `
	NodeType        string `json:"nodeType" `
	VirtualIP       string `json:"virtualIP" `
	StandbyDeviceIP string `json:"standbyDeviceIP"`
	MariaDBIp       string `json:"mariaDBIp" `
	MariaDBPort     int    `json:"mariaDBPort" `
	MariaDBUser     string `json:"mariaDBUser" `
	NetInterface    string `json:"netInterface" `
}

func (r *HAConfig) ToHAConfigDto() *HAConfigDto {
	return &HAConfigDto{
		Enabled:         r.Enabled,
		NodeType:        r.NodeType,
		VirtualIP:       r.VirtualIP,
		StandbyDeviceIP: r.StandbyDeviceIP,
		MariaDBIp:       r.MariaDBIp,
		MariaDBPort:     r.MariaDBPort,
		MariaDBUser:     r.MariaDBUser,
		NetInterface:    r.NetInterface,
	}
}
