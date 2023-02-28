package model

import "tkbastion/server/utils"

type UserNew struct {
	ID                     string         `gorm:"type:varchar(128);primary_key;not null;comment:用户id" json:"id"   map:"key:id"`
	Username               string         `gorm:"type:varchar(64);unique;not null;comment:用户名" json:"username"  validate:"required,regex=^[A-Za-z0-9_\\-]{4,32}$" label:"用户名:" map:"key:username"`
	Password               string         `gorm:"type:varchar(64);null;comment:密码" json:"password" validate:"required,regex=^(?![0-9\\*]+$)(?![a-zA-Z\\*]+$)[^ ]+$" label:"密码:" map:"key:password"`
	VerifyPassword         string         `gorm:"type:varchar(64);null;comment:确认密码" json:"verifyPassword" validate:"required,eqfield=Password" label:"确认密码:" map:"key:verify_password"`
	PasswordUpdated        utils.JsonTime `gorm:"type:datetime(3);comment:密码修改日期" json:"passwordUpdated" map:"key:password_updated"`
	Nickname               string         `gorm:"type:varchar(32);not null;comment:昵称" json:"nickname" validate:"required" label:"昵称:" map:"key:nickname"`
	TOTPSecret             string         `gorm:"type:varchar(128);default:'-';comment:双因素认证密钥" json:"totpSecret" map:"key:totp_secret;empty"`
	TOTP                   string         `gorm:"type:varchar(128);default:'-';comment:双因素认证码" json:"totp" map:"key:totp;empty"`
	Online                 bool           `gorm:"type:tinyint(1);default:0;comment:在线状态" json:"online" map:"key:online;empty"`
	Created                utils.JsonTime `gorm:"type:datetime(3);comment:创建日期" json:"created" map:"key:created"`
	RoleId                 string         `gorm:"type:varchar(128);not null;comment:角色id" json:"roleId" map:"key:role_id"`
	RoleName               string         `gorm:"type:varchar(64);comment:角色名" json:"roleName" map:"key:role_name"`
	DepartmentId           int64          `gorm:"type:bigint(20);not null;comment:部门机构id" json:"departmentId" map:"key:department_id;empty"`
	DepartmentName         string         `gorm:"type:varchar(64);comment:部门机构" json:"departmentName" map:"key:department_name"`
	AuthenticationWay      string         `gorm:"type:varchar(64);not null;comment:认证方式" json:"authenticationWay" map:"key:authentication_way"`
	AuthenticationServer   string         `gorm:"type:varchar(64);comment:认证服务器" json:"authenticationServer" map:"key:authentication_server"`
	AuthenticationServerId int64          `gorm:"type:bigint(20);comment:认证服务器id" json:"authenticationServerId" map:"key:authentication_server_id"`
	Dn                     string         `gorm:"type:varchar(64);comment:dn" json:"dn" map:"key:dn"`
	IsPermanent            *bool          `gorm:"type:varchar(1);default:true;comment:是否永久有效" json:"isPermanent" map:"key:is_permanent"`
	Status                 string         `gorm:"type:varchar(16);not null;comment:状态" json:"status" map:"key:status"`
	IsRandomPassword       bool           `gorm:"type:varchar(1);default:false;comment:是否随机密码" json:"isRandomPassword" map:"key:is_random_password;empty"`
	SendWay                string         `gorm:"type:varchar(16);comment:发送方式" json:"sendWay" map:"key:send_way;empty"`
	BeginValidTime         utils.JsonTime `gorm:"type:datetime(3);comment:'开始时间'" json:"beginValidTime" map:"key:begin_valid_time"`
	EndValidTime           utils.JsonTime `gorm:"type:datetime(3);comment:'结束时间'" json:"endValidTime" map:"key:end_valid_time"`
	Wechat                 string         `gorm:"type:varchar(64);comment:微信" json:"wechat" map:"key:wechat;empty"`
	QQ                     string         `gorm:"type:varchar(64);comment:QQ" json:"qq" map:"key:qq;empty"`
	Phone                  string         `gorm:"type:varchar(64);comment:电话" json:"phone" map:"key:phone;empty"`
	Description            string         `gorm:"type:varchar(64);comment:描述" json:"description" map:"key:description;empty"`
	Mail                   string         `gorm:"type:varchar(128);default:'';comment:邮箱" json:"mail" map:"key:mail;empty"`
	LoginNumbers           int            `gorm:"type:int;not null;default:0;comment:登录次数" json:"loginNumbers" map:"key:login_numbers"`
	VerifyMailState        *bool          `gorm:"type:tinyint(1);not null;default:0;comment:认证邮箱是否启用" json:"verifyMailState" map:"key:verify_mail_state"`
	SamePwdJudge           string         `gorm:"type:varchar(2048);comment:密码重复判断" json:"samePwdJudge" map:"key:same_pwd_judge"`
}

func (r *UserNew) TableName() string {
	return "user_new"
}
