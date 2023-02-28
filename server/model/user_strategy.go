package model

import (
	"errors"
	"fmt"
	"github.com/vmihailenco/msgpack"
	"strconv"
	"strings"
	"tkbastion/server/utils"
)

type UserStrategy struct {
	ID              string         `gorm:"type:varchar(128);primary_key;comment:用户策略id" json:"id"`
	Name            string         `gorm:"type:varchar(64);unique;not null;comment:策略名称" json:"name"`
	DepartmentName  string         `gorm:"type:varchar(64);comment:部门机构名称" json:"departmentName"`
	DepartmentId    int64          `gorm:"type:bigint;comment:部门机构id" json:"departmentId"`
	DepartmentDepth int            `gorm:"type:bigint;comment:部门机构深度" json:"departmentDepth"`
	Status          string         `gorm:"type:varchar(16);not null;comment:是否启用" json:"status"`
	Priority        int64          `gorm:"type:int;comment:优先权" json:"priority"`
	Created         utils.JsonTime `gorm:"type:datetime(3);comment:创建日期" json:"created"`
	IsPermanent     *bool          `gorm:"type:tinyint(1);comment:是否永久有效" json:"isPermanent"`
	BeginValidTime  utils.JsonTime `gorm:"type:datetime(3);comment:'开始时间'" json:"beginValidTime"`
	EndValidTime    utils.JsonTime `gorm:"type:datetime(3);comment:'结束时间'" json:"endValidTime"`
	IpLimitType     string         `gorm:"type:varchar(12);not null;comment:IP限制类型" json:"ipLimitType"`
	IpLimitList     string         `gorm:"type:varchar(4096);not null;default:'';comment:IP限制列表" json:"ipLimitList"`
	Description     string         `gorm:"type:varchar(128);comment:描述" json:"description"`
	ExpirationDate  []byte         `gorm:"type:blob;comment:限用日期" json:"expirationDate"`
}

type UserStrategyDTO struct {
	ID             string         `json:"id"`
	Name           string         `json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	DepartmentName string         `json:"departmentName"`
	DepartmentId   int64          `json:"departmentId"`
	Status         string         `json:"status"`
	Priority       int64          `json:"priority"`
	Created        utils.JsonTime `json:"created"`
	IsPermanent    bool           `json:"isPermanent"`
	BeginValidTime utils.JsonTime `json:"beginValidTime"`
	EndValidTime   utils.JsonTime `json:"endValidTime"`
	IpLimitType    string         `json:"ipLimitType"`
	IpLimitList    string         `json:"ipLimitList"`
	Description    string         `json:"description" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
	ExpirationDate *[]ExpireData  `json:"expirationDate"`
	UserId         string         `json:"userId"`
	UserGroupId    string         `json:"userGroupId"`
}

func (r *UserStrategy) ToUserStrategyDTO() (*UserStrategyDTO, error) {
	exps, err := r.byteToExps()
	if err != nil {
		return nil, err
	}
	return &UserStrategyDTO{
		ID:             r.ID,
		Name:           r.Name,
		DepartmentName: r.DepartmentName,
		DepartmentId:   r.DepartmentId,
		Status:         r.Status,
		Created:        r.Created,
		Priority:       r.Priority,
		IsPermanent:    *r.IsPermanent,
		BeginValidTime: r.BeginValidTime,
		EndValidTime:   r.EndValidTime,
		IpLimitType:    r.IpLimitType,
		IpLimitList:    r.IpLimitList,
		Description:    r.Description,
		ExpirationDate: exps,
	}, nil
}

func (r *UserStrategy) byteToExps() (*[]ExpireData, error) {
	//反序列化
	arr := [168]bool{}
	err := msgpack.Unmarshal(r.ExpirationDate, &arr)
	if err != nil {
		return nil, err
	}
	expireData := make([]ExpireData, 168)
	for i := 1; i <= 7; i++ {
		for j := 0; j <= 23; j++ {
			flag := arr[(i-1)*24+j]
			exp := new(ExpireData)
			exp.Name = fmt.Sprintf("%v-%v", i, j)
			exp.Checked = flag
			expireData[(i-1)*24+j] = *exp
		}
	}
	return &expireData, nil
}

func (r *UserStrategyDTO) ToUserStrategy() (UserStrategy, error) {
	bytes, err := r.expToByte()
	if err != nil {
		return UserStrategy{}, err
	}
	return UserStrategy{
		ID:             r.ID,
		Name:           r.Name,
		DepartmentName: r.DepartmentName,
		DepartmentId:   r.DepartmentId,
		Status:         r.Status,
		Created:        r.Created,
		Priority:       r.Priority,
		IsPermanent:    &r.IsPermanent,
		BeginValidTime: r.BeginValidTime,
		EndValidTime:   r.EndValidTime,
		IpLimitType:    r.IpLimitType,
		IpLimitList:    r.IpLimitList,
		Description:    r.Description,
		ExpirationDate: bytes,
	}, nil
}

func (r *UserStrategyDTO) expToByte() ([]byte, error) {
	//1-23  切割
	arr := [168]bool{}
	for _, val := range *r.ExpirationDate {
		tmp := strings.Split(val.Name, "-")
		if len(tmp) < 2 {
			return nil, errors.New("参数长度小于2")
		}
		week, err := strconv.Atoi(tmp[0])
		if err != nil {
			return nil, err
		}
		time24, err := strconv.Atoi(tmp[1])
		if err != nil {
			return nil, err
		}
		arr[(week-1)*24+time24] = val.Checked
	}
	data, err := msgpack.Marshal(arr)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (r *UserStrategy) TableName() string {
	return "user_strategy"
}

type ExpireData struct {
	Name    string `json:"name"`
	Checked bool   `json:"checked"`
}

type UserStrategyUsers struct {
	ID             string `gorm:"type:varchar(64);not null;comment:主键" json:"id"`
	UserStrategyId string `gorm:"type:varchar(64);not null;comment:用户策略id" json:"userStrategyId"`
	UserId         string `gorm:"type:varchar(64);not null;comment:用户id" json:"userId"`
}

func (r *UserStrategyUsers) TableName() string {
	return "user_strategy_users"
}

type UserStrategyUserGroup struct {
	ID             string `gorm:"type:varchar(64);not null;comment:主键" json:"id"`
	UserStrategyId string `gorm:"type:varchar(64);not null;comment:用户策略id" json:"userStrategyId"`
	UserGroupId    string `gorm:"type:varchar(64);not null;comment:用户组id" json:"userGroupId"`
}

func (r *UserStrategyUserGroup) TableName() string {
	return "user_strategy_user_group"
}
