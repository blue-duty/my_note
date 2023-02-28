package dto

import (
	"tkbastion/server/utils"
)

type UserForEditBatch struct {
	ID             string         `json:"id"`
	RoleId         string         `json:"roleId"`
	DepartmentId   string         `json:"departmentId"`
	Password       string         `json:"password"`
	RePassword     string         `json:"rePassword"`
	IsPermanent    bool           `json:"isPermanent"`
	BeginValidTime utils.JsonTime `json:"beginValidTime"`
	EndValidTime   utils.JsonTime `json:"endValidTime"`
}

type UserForPage struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	//Password               string         `json:"password"`
	//VerifyPassword         string         `json:"verifyPassword"`
	Nickname               string         `json:"nickname"`
	TOTPSecret             string         `json:"totpSecret"`
	Online                 bool           `json:"online"`
	Created                utils.JsonTime `json:"created"`
	RoleId                 string         `json:"roleId"`
	RoleName               string         `json:"roleName"`
	DepartmentId           int64          `json:"departmentId"`
	DepartmentName         string         `json:"departmentName"`
	AuthenticationWay      string         `json:"authenticationWay"`
	AuthenticationServer   string         `json:"authenticationServer"`
	AuthenticationServerId int64          `json:"authenticationServerId"`
	Dn                     string         `json:"dn"`
	IsPermanent            bool           `json:"isPermanent"`
	Status                 string         `json:"status"`
	IsRandomPassword       bool           `json:"isRandomPassword"`
	SendWay                string         `json:"sendWay"`
	BeginValidTime         utils.JsonTime `json:"beginValidTime"`
	EndValidTime           utils.JsonTime `json:"endValidTime"`
	Wechat                 string         `json:"wechat"`
	QQ                     string         `json:"qq"`
	Phone                  string         `json:"phone"`
	Description            string         `json:"description"`
	Mail                   string         `json:"mail"`
}

type UserForExport struct {
	Id             string `json:"id"`
	Username       string `json:"username"`
	Nickname       string `json:"nickname"`
	DepartmentName string `json:"department"`
	RoleName       string `json:"role"`
	Status         string `json:"status"`
	Mail           string `json:"mail"`
	QQ             string `json:"qq"`
	Wechat         string `json:"wechat"`
	Phone          string `json:"phone"`
	Description    string `json:"description"`
}

type UserDetailForBasis struct {
	ID                string `json:"id"`
	Username          string `json:"username"`
	Nickname          string `json:"nickname"`
	RoleName          string `json:"roleName"`
	Department        string `json:"department"`
	AuthenticationWay string `json:"authenticationWay"`
	Status            string `json:"status"`
	Wechat            string `json:"wechat"`
	QQ                string `json:"qq"`
	Description       string `json:"description"`
	Mail              string `json:"mail"`
	Phone             string `json:"phone"`
	ValidTime         string `json:"validTime"`
}
type UserDetailForUserGroup struct {
	Name       string         `json:"name"`
	Department string         `json:"department"`
	Create     utils.JsonTime `json:"create"`
}
type UserDetailForUserStrategy struct {
	Name       string `json:"name"`
	Department string `json:"department"`
	Status     string `json:"status"`
}
type UserDetailForCommandStrategy struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Protocol string `json:"protocol"`
	Passport string `json:"passport"`
	Action   string `json:"action"`
	Strategy string `json:"strategy"` // 指令动作+部门拼接
}

type UserDetailForAsset struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Address         string `json:"address"`
	SystemType      string `json:"systemType"`
	Protocol        string `json:"protocol"`
	Passport        string `json:"passport"`
	PolicyOperation string `json:"policyOperation"`
}

type UserDetailForApp struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Program   string `json:"program"`
	Username  string `json:"username"`
	Server    string `json:"server"`
	PolicyApp string `json:"policyApp"`
}
