package model

import "tkbastion/server/utils"

type NewAsset struct {
	ID            string         `gorm:"type:varchar(128);primary_key;not null;comment:设备id" json:"id" map:"key:id"`
	Name          string         `gorm:"type:varchar(64);unique;not null;comment:设备名称" json:"name" map:"key:name"`
	IP            string         `gorm:"type:varchar(64);not null;comment:设备地址" json:"ip" map:"key:ip"`
	PassPortCount int            `gorm:"type:int(5);not null;comment:设备帐号数" json:"pass_port_count" map:"key:pass_port_count"`
	AssetType     string         `gorm:"type:varchar(128);not null;comment:设备类型" json:"asset_type" map:"key:asset_type"`
	DepartmentId  int64          `gorm:"force;type:bigint(20);not null;comment:所属部门id"  json:"department_id" map:"key:department_id;empty"`
	Created       utils.JsonTime `gorm:"type:datetime(3);not null;comment:资产添加时间"  json:"created" map:"key:created"`
	Info          string         `gorm:"type:varchar(4096);default:'';comment:资产信息" json:"info" map:"key:info;empty"`
}

func (r *NewAsset) TableName() string {
	return "new_assets"
}

type PassPort struct {
	ID           string         `gorm:"type:varchar(128);primary_key;not null;comment:帐号id" json:"id" map:"key:id"`
	Name         string         `gorm:"type:varchar(64);not null;comment:帐号名称" json:"name" map:"key:name"`
	AssetId      string         `gorm:"type:varchar(128);index;not null;comment:资产id" json:"assetId" map:"key:asset_id"`
	AssetName    string         `gorm:"type:varchar(64);not null;comment:资产名称" json:"assetName" map:"key:asset_name"`
	Ip           string         `gorm:"type:varchar(64);not null;comment:资产地址" json:"ip" map:"key:ip"`
	DepartmentId int64          `gorm:"type:bigint(20);not null;comment:所属部门id" json:"departmentId" map:"key:department_id;empty"`
	AssetType    string         `gorm:"type:varchar(128);not null;comment:资产类型" json:"assetType" map:"key:asset_type"`
	LoginType    string         `gorm:"type:varchar(64);not null;comment:登录类型" json:"loginType" map:"key:login_type"` // 1. auto 2. manual
	Protocol     string         `gorm:"type:varchar(16);not null;comment:登录协议" json:"protocol" map:"key:protocol"`
	Port         int            `gorm:"type:int(5);not null;comment:端口" json:"port" validate:"required,gte=0,lte=65535" map:"key:port"`
	PassportType string         `gorm:"type:varchar(16);comment:帐号类型" json:"passPortType" map:"key:passport_type"` // 1. Ordinary  2. Administrator
	Passport     string         `gorm:"type:varchar(64);not null;comment:帐号" json:"passport" map:"key:passport;empty"`
	Password     string         `gorm:"type:varchar(64);comment:密码" json:"password" map:"key:password"`
	IsSshKey     int            `gorm:"type:tinyint(1);not null;comment:是否使用sshkey" json:"isSshKey" map:"key:is_ssh_key;empty"`
	Status       string         `gorm:"type:char(10);not null;comment:帐号状态" json:"status" map:"key:status"`
	Created      utils.JsonTime `gorm:"type:datetime(3);not null;comment:帐号添加时间" json:"created" map:"key:created"`
	Active       int            `gorm:"type:tinyint(1);default:0;comment:帐号是否在线" json:"active" map:"key:active"`
	PrivateKey   string         `gorm:"type:varchar(256);comment:私钥名称" json:"privateKey" map:"key:private_key;empty"`
	Passphrase   string         `gorm:"type:varchar(256);comment:私钥密码" json:"passphrase" map:"key:passphrase;empty"`
	SftpPath     string         `gorm:"type:varchar(512);comment:sftp路径" json:"sftpPath" map:"key:sftp_path;empty"`
	// 上次改密时间
	LastChangePasswordTime utils.JsonTime `gorm:"type:datetime(3);comment:上次改密时间" json:"lastChangePasswordTime" map:"key:last_change_password_time"`
	// 上次密码
	LastPassword string `gorm:"type:varchar(64);comment:上次密码" json:"lastPassword" map:"key:last_password"`
}

func (r *PassPort) TableName() string {
	return "pass_ports"
}

type PassportConfiguration struct {
	ID         int64  `json:"id" gorm:"primary_key;auto_increment"`
	PassportId string `json:"passportId" gorm:"type:varchar(128);index;not null;comment:帐号id"`
	Name       string `json:"name" gorm:"type:varchar(64);not null;comment:配置名称"`
	Value      string `json:"value" gorm:"type:varchar(4096);not null;comment:配置值"`
}

func (r *PassportConfiguration) TableName() string {
	return "passport_configurations"
}
