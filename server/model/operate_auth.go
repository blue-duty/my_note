package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"tkbastion/server/utils"

	"github.com/vmihailenco/msgpack"
)

type OperateAuth struct {
	ID                 int64          `gorm:"primary_key;type:bigint(20) AUTO_INCREMENT;not null;comment:运维授权ID" json:"id"`
	Name               string         `gorm:"type:varchar(32);not null;comment:运维授权名称" json:"name"`
	DepartmentId       int64          `gorm:"type:bigint(20);not null;comment:所属部门ID" json:"departmentId"`
	DepartmentName     string         `gorm:"type:varchar(64);not null;comment:所属部门名称" json:"departmentName"`
	State              string         `gorm:"type:varchar(8);not null;default:'on';comment:状态" json:"state"`
	ButtonState        string         `gorm:"type:varchar(3);not null;default:'on';comment:按钮状态" json:"buttonState"`
	Description        string         `gorm:"type:varchar(128);not null;default:'';comment:描述" json:"description"`
	RelateUser         string         `gorm:"type:text;not null;comment:关联用户" json:"relateUser"`
	RelateUserGroup    string         `gorm:"type:text;not null;comment:关联用户组" json:"relateUserGroup"`
	RelateAsset        string         `gorm:"type:text;not null;comment:关联设备" json:"relateAsset"`
	RelateAssetGroup   string         `gorm:"type:text;not null;comment:关联设备组" json:"relateAssetGroup"`
	RelateApp          string         `gorm:"type:text;not null;comment:关联应用" json:"relateApp"`
	DepLevel           int            `gorm:"type:int(3);not null;comment:部门深度" json:"depLevel"`
	Priority           int            `gorm:"type:int(4);not null;comment:排序(优先级)" json:"priority"`
	AuthExpirationDate []byte         `gorm:"type:blob;comment:授权时段限制" json:"authExpirationDate"`
	Download           string         `gorm:"type:varchar(3);not null;default:'off';comment:下载" json:"download"`
	Upload             string         `gorm:"type:varchar(3);not null;default:'off';comment:上传" json:"upload"`
	Watermark          string         `gorm:"type:varchar(3);not null;default:'off';comment:水印" json:"watermark"`
	StrategyBeginTime  utils.JsonTime `gorm:"type:datetime(3);comment:'策略有效期开始时间'" json:"strategyBeginTime"`
	StrategyEndTime    utils.JsonTime `gorm:"type:datetime(3);comment:'策略有效期结束时间'" json:"strategyEndTime"`
	StrategyTimeFlag   bool           `gorm:"type:tinyint(1);not null;comment:是否开启永久有效" json:"strategyTimeFlag"`
	IpLimitType        string         `gorm:"type:varchar(12);not null;comment:IP限制类型" json:"ipLimitType"`
	IpLimitList        string         `gorm:"type:varchar(4096);not null;default:'';comment:IP限制列表" json:"ipLimitList"`
}

// TODO 权限控制

// 状态的值只会在两个时候改变
// 1.创建一个策略时，其默认为开(其实为默认与ButtonState状态一样)
// 2.get策略时，若策略过期更新state为已过期，否则更新其为与ButtonState值一样

// ButtonState状态值也只会在两个时候改变
// 1.创建一个策略时，其默认为开
// 2.在页面上点击按钮时值进行更新

func (r *OperateAuth) ToOperateAuthDto() (*OperateAuthDTO, error) {
	exps, err := r.byteToExps()
	if err != nil {
		return nil, err
	}
	var auth = make([]string, 0)
	if r.Download == "on" {
		auth = append(auth, "download")
	}
	if r.Upload == "on" {
		auth = append(auth, "upload")
	}
	if r.Watermark == "on" {
		auth = append(auth, "watermark")
	}

	return &OperateAuthDTO{
		ID:                 r.ID,
		Name:               r.Name,
		DepartmentId:       r.DepartmentId,
		DepartmentName:     r.DepartmentName,
		State:              r.State,
		ButtonState:        r.ButtonState,
		Description:        r.Description,
		RelateUser:         r.RelateUser,
		RelateUserGroup:    r.RelateUserGroup,
		RelateAsset:        r.RelateAsset,
		RelateAssetGroup:   r.RelateAssetGroup,
		RelateApp:          r.RelateApp,
		DepLevel:           r.DepLevel,
		Priority:           r.Priority,
		AuthExpirationDate: exps,
		StrategyBeginTime:  r.StrategyBeginTime,
		StrategyEndTime:    r.StrategyEndTime,
		StrategyTimeFlag:   r.StrategyTimeFlag,
		IpLimitType:        r.IpLimitType,
		IpLimitList:        r.IpLimitList,
		OperateAuth:        auth,
	}, nil
}

func (r *OperateAuth) byteToExps() (*[]ExpireData, error) {
	//反序列化
	arr := [168]bool{}
	err := msgpack.Unmarshal(r.AuthExpirationDate, &arr)
	if err != nil {
		return nil, err
	}
	expiredata := make([]ExpireData, 168)
	for i := 1; i <= 7; i++ {
		for j := 0; j <= 23; j++ {
			flag := arr[(i-1)*24+j]
			exp := new(ExpireData)
			exp.Name = fmt.Sprintf("%v-%v", i, j)
			exp.Checked = flag
			expiredata[(i-1)*24+j] = *exp
		}
	}
	return &expiredata, nil
}

type OperateAuthDTO struct {
	ID                 int64          `json:"id"`
	Name               string         `json:"name" validate:"required,min=1,max=32,alphanumunicode"`
	DepartmentId       int64          `json:"departmentId"`
	DepartmentName     string         `json:"departmentName"`
	State              string         `json:"state"`
	ButtonState        string         `json:"buttonState"`
	Description        string         `json:"description" validate:"max=128,regexp=^[\\p{Han}\\p{L}]+$"`
	RelateUser         string         `json:"relateUser"`
	RelateUserGroup    string         `json:"relateUserGroup"`
	RelateAsset        string         `json:"relateAsset"`
	RelateAssetGroup   string         `json:"relateAssetGroup"`
	RelateApp          string         `json:"relateApp"`
	DepLevel           int            `json:"depLevel"`
	Priority           int            `json:"priority"`
	OperateAuth        []string       `json:"operateAuth"`
	AuthExpirationDate *[]ExpireData  `json:"authExpirationDate"`
	StrategyBeginTime  utils.JsonTime `json:"strategyBeginTime"`
	StrategyEndTime    utils.JsonTime `json:"strategyEndTime"`
	StrategyTimeFlag   bool           `json:"strategyTimeFlag"`
	IpLimitType        string         `json:"ipLimitType"`
	IpLimitList        string         `json:"ipLimitList"`
}

func (r *OperateAuthDTO) ToOperateAuth() (OperateAuth, error) {
	bytes, err := r.expToByte()
	if err != nil {
		return OperateAuth{}, err
	}
	tool := func(s []string, string2 string) string {
		for _, v := range s {
			if v == string2 {
				return "on"
			}
		}
		return "off"
	}

	return OperateAuth{
		ID:                 r.ID,
		Name:               r.Name,
		DepartmentId:       r.DepartmentId,
		DepartmentName:     r.DepartmentName,
		State:              r.State,
		ButtonState:        r.ButtonState,
		Description:        r.Description,
		RelateUser:         r.RelateUser,
		RelateUserGroup:    r.RelateUserGroup,
		RelateAsset:        r.RelateAsset,
		RelateAssetGroup:   r.RelateAssetGroup,
		RelateApp:          r.RelateApp,
		DepLevel:           r.DepLevel,
		Priority:           r.Priority,
		AuthExpirationDate: bytes,
		StrategyBeginTime:  r.StrategyBeginTime,
		StrategyEndTime:    r.StrategyEndTime,
		StrategyTimeFlag:   r.StrategyTimeFlag,
		IpLimitType:        r.IpLimitType,
		IpLimitList:        r.IpLimitList,
		Download:           tool(r.OperateAuth, "download"),
		Upload:             tool(r.OperateAuth, "upload"),
		Watermark:          tool(r.OperateAuth, "watermark"),
	}, nil
}

func (r *OperateAuthDTO) expToByte() ([]byte, error) {
	//1-23  切割
	arr := [168]bool{}
	for _, val := range *r.AuthExpirationDate {
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

func (r *OperateAuth) TableName() string {
	return "operate_auth"
}
