package model

// TODO 校验未完成
type LdapAdAuth struct {
	ID                  int64  `gorm:"primary_key;type:bigint(20) AUTO_INCREMENT;not null;comment:LDAP/AD认证配置ID" json:"id"`
	LdapAdServerAddress string `gorm:"type:varchar(128);not null;comment:服务器地址" json:"ldapAdServerAddress" validate:"required,ip|fqdn"`
	LdapAdType          string `gorm:"type:varchar(4);not null;comment:认证类型" json:"ldapAdType"`
	LdapAdTls           string `gorm:"type:varchar(5);not null;comment:是否开启tls加密" json:"ldapAdTls"`
	LdapAdPort          int    `gorm:"type:int(5);not null;comment:端口" json:"ldapAdPort" validate:"required,min=1,max=65535"`
	LdapAdAdminDn       string `gorm:"type:varchar(128);not null;comment:管理员DN" json:"ldapAdAdminDn"`
	LdapAdPassword      string `gorm:"type:varchar(128);not null;comment:管理员密码" json:"ldapAdPassword"`
	LdapAdDomain        string `gorm:"type:varchar(128);not null;comment:域名" json:"ldapAdDomain"`
	LdapAdDirDn         string `gorm:"type:varchar(128);not null;comment:目录DN" json:"ldapAdDirDn"`
	LdapAdSyncType      string `gorm:"type:varchar(7);not null;comment:同步类型" json:"ldapAdSyncType"`
	LdapAdSyncTime      string `gorm:"type:varchar(16);default:'';comment:自动同步时间"json:"ldapAdSyncTime"`
}

func (r *LdapAdAuth) TableName() string {
	return "ldap_ad_auth"
}
