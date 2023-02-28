package model

type Property struct {
	Name  string `gorm:"type:varchar(64);primary_key;not null;comment:配置名称"  json:"name"  validate:"required,max=64" `
	Value string `gorm:"type:varchar(2048);default:'';comment:配置值"  json:"value"            validate:"max=256"  `
}

func (r *Property) TableName() string {
	return "properties"
}

type NetworkConfigGet struct {
	Name    string `json:"name"`
	Ip      string `json:"ip"`
	Gateway string `json:"gateway"`
	Netmask string `json:"netmask"`
	Status  string `json:"status"`
	Mode    string `json:"mode"`
}

type NetworkConfig struct {
	Name        string `json:"name"`
	Mode        string `json:"mode"`
	Ip          string `json:"ip" validate:"ip"`
	Gateway     string `json:"gateway" validate:"ip"`
	Netmask     string `json:"netmask" validate:"ip"`
	Status      string `json:"status"`
	Ipv6Status  string `json:"ipv6Status"`
	Ipv6        string `json:"ipv6" validate:"ipv6"`
	Ipv6Gateway string `json:"ipv6Gateway" validate:"ipv6"`
}

type TestMail struct {
	MailHost     string `json:"mail-host"`     // 邮件服务器地址
	MailPort     string `json:"mail-port"`     // 邮件服务器端口
	MailUsername string `json:"mail-username"` // 邮件服务账号
	MailPassword string `json:"mail-password"` // 邮件服务密码
	MailReceiver string `json:"mail-receiver"` // 收件邮箱
}

type StaticRoute struct {
	RouteType          string `json:"routeType"`
	DestinationAddress string `json:"destinationAddress" validate:"ip|ipv6"`
	SubnetMask         string `json:"subnetMask" validate:"ip"`
	NextHopAddress     string `json:"nextHopAddress" validate:"ip|ipv6"`
	InterfaceName      string `json:"interfaceName"`
	Description        string `json:"description" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
}

type IpAddress struct {
	Address  string `json:"address"`
	TestType string `json:"testType"`
	Port     int    `json:"port"`
}

type SnmpConfigGet struct {
	SnmpState            string `json:"snmpState" gorm:"comment:SNMP功能是否启用"`
	Port                 int    `json:"port" gorm:"comment:监听端口"`
	PhysicalLocationInfo string `json:"physicalLocationInfo" gorm:"comment:物理位置信息"`
	ContactInfo          string `json:"contactInfo" gorm:"comment:联系信息"`
	SysInfo              string `json:"sysInfo" gorm:"comment:系统信息"`
	Version              string `json:"version" gorm:"comment:snmp版本"`
	V2CName              string `json:"v2cName" gorm:"comment:v2c版本团体名"`
	V2CRWAuth            string `json:"v2cRWAuth" gorm:"comment:v2c版本读写权限"`
	V3Name               string `json:"v3Name" gorm:"comment:v3版本用户名"`
	V3RWAuth             string `json:"v3RWAuth" gorm:"comment:v3版本读写权限"`
	V3CertificationType  string `json:"v3CertificationType" gorm:"comment:v3版本认证类型"`
	V3CertificationMode  string `json:"v3CertificationMode" gorm:"comment:v3版本认证模式"`
	V3EncryptionMode     string `json:"v3EncryptionMode" gorm:"comment:v3版本加密模式"`
	AuthIp               string `json:"authIp" gorm:"comment:可访问设备IP"`
}

type SnmpConfig struct {
	SnmpState            string `json:"snmpState" gorm:"comment:SNMP功能是否启用"`
	Port                 int    `json:"port" gorm:"comment:监听端口"`
	PhysicalLocationInfo string `json:"physicalLocationInfo" gorm:"comment:物理位置信息"`
	ContactInfo          string `json:"contactInfo" gorm:"comment:联系信息"`
	SysInfo              string `json:"sysInfo" gorm:"comment:系统信息"`
	Version              string `json:"version" gorm:"comment:snmp版本"`
	V2CName              string `json:"v2cName" gorm:"comment:v2c版本团体名"`
	V2CRWAuth            string `json:"v2cRWAuth" gorm:"comment:v2c版本读写权限"`
	V3Name               string `json:"v3Name" gorm:"comment:v3版本用户名"`
	V3RWAuth             string `json:"v3RWAuth" gorm:"comment:v3版本读写权限"`
	V3CertificationType  string `json:"v3CertificationType" gorm:"comment:v3版本认证类型"`
	V3CertificationMode  string `json:"v3CertificationMode" gorm:"comment:v3版本认证模式"`
	V3CertificationPass  string `json:"v3CertificationPass" gorm:"comment:v3版本认证密码"`
	V3EncryptionMode     string `json:"V3EncryptionMode" gorm:"comment:v3版本加密模式"`
	V3EncryptionPass     string `json:"v3EncryptionPass" gorm:"comment:v3版本加密密码"`
	AuthIp               string `json:"authIp" gorm:"comment:可访问设备IP"`
}
