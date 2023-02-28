package service

import (
	"sort"
	"time"
	"tkbastion/pkg/constant"
	"tkbastion/pkg/log"
	"tkbastion/server/model"
	"tkbastion/server/repository"

	"github.com/vmihailenco/msgpack"
)

//exp  限定日期服务
//管理限制周+时间  例如: 周一15点不可以使用

type expDateService struct {
	baseService
}

//// SaveOneExp 传入单个周+时间进行存储
//func (*expDateService) SaveOneExp(user *model.User, weekday int, time int) (string, error) {
//	exp := user.ExpirationDate
//	if len(exp) == 0 {
//		return "", errors.New("周和时间为空")
//	}
//
//	//反序列化
//	arr := [168]bool{}
//	err := msgpack.Unmarshal(exp, &arr)
//	if err != nil {
//		return "", err
//	}
//
//	//限制时间
//	//weekday : 1,2,3,4,5,6,7(周一到周日)   time: 0-23 (时间)     index= 0-167 (抽象数据)
//	//(weekday - 1) * 24 + time = index  正向公式
//	index := (weekday-1)*24 + time
//	arr[index] = true
//
//	//序列化
//	data, err := msgpack.Marshal(arr)
//	if err != nil {
//		return "", err
//	}
//
//	return string(data), nil
//}
//
//// SaveManyExp 传入多个周+时间进行存储
//func (*expDateService) SaveManyExp() {
//
//}
//
//// AllExpTrue  所有权限开启
//func (*expDateService) AllExpTrue() ([]byte, error) {
//	tmp := [168]bool{}
//	for i := 1; i <= 7; i++ {
//		for j := 0; j <= 23; j++ {
//			tmp[(i-1)*24+j] = true
//		}
//	}
//	//序列化
//	data, err := msgpack.Marshal(tmp)
//	if err != nil {
//		log.Errorf("Update err , err message :%v", err.Error())
//		return nil, err
//	}
//	return data, nil
//}

// JudgeExpOperateAuth 判断当前运维授权策略是否在授权时段限制之内
func (*expDateService) JudgeExpOperateAuth(operateAuth *model.OperateAuth) (bool, error) {
	exp := operateAuth.AuthExpirationDate
	if len(exp) == 0 {
		log.Error("JudgeExpOperateAuth Error: len(exp)==0")
		return false, nil
	}

	//反序列化
	arr := [168]bool{}
	err := msgpack.Unmarshal(exp, &arr)
	if nil != err {
		log.Errorf("Unmarshal err: %v", err)
		return false, err
	}

	now := time.Now()
	weekday := int(now.Weekday())
	//星期日特殊处理
	if weekday == 0 {
		weekday = 7
	}
	t := now.Hour()
	index := (weekday-1)*24 + t
	return arr[index], nil
}

func (*expDateService) JudgeExpUserStrategy(userStrategy *model.UserStrategy) (bool, error) {
	exp := userStrategy.ExpirationDate
	if len(exp) == 0 {
		log.Errorf("exp is null!!!")
		return false, nil
	}

	//反序列化
	arr := [168]bool{}
	err := msgpack.Unmarshal(exp, &arr)
	if err != nil {
		log.Errorf("Unmarshal err , err message :%v", err.Error())
		return false, err
	}

	now := time.Now()
	weekday := int(now.Weekday())
	//星期日特殊处理
	if weekday == 0 {
		weekday = 7
	}
	t := now.Hour()
	index := (weekday-1)*24 + t
	return arr[index], nil
}

// JudgeExpByToken 传入token判断是否有权限访问
func (*expDateService) JudgeExpByToken(token string) (bool, error) {
	if token == "" {
		log.Errorf("token is null!!!")
		return false, nil
	}
	//获取用户对象
	logLog, err := repository.LoginLogDao.FindById(token)
	if err != nil {
		log.Errorf("get logLog err , err message :%v", err.Error())
		return false, err
	}
	user, err := repository.UserNewDao.FindById(logLog.UserId)
	if err != nil {
		log.Errorf("get user err , err message :%v", err.Error())
		return false, err
	}
	// 获取用户组便于查用户策略
	userGroupId, _ := repository.UserGroupMemberDao.FindUserGroupIdsByUserId(user.ID)
	var userStrategyId []string
	_ = repository.UserStrategyDao.DB.Table("user_strategy_users").Select("user_strategy_id").Where("user_id = ?", user.ID).Find(&userStrategyId)
	var userStrategyIdGroup []string
	_ = repository.UserStrategyDao.DB.Table("user_strategy_user_group").Select("user_strategy_id").Where("user_group_id in  ?", userGroupId).Find(&userStrategyIdGroup)
	userStrategyId = append(userStrategyId, userStrategyIdGroup...)
	if len(userStrategyId) == 0 { // 没有策略限制可直接登录
		return true, nil
	}
	var userStrategyValid model.UserStrategy
	if len(userStrategyId) > 0 {
		// 找到用户策略id对应的策略
		userStrategy := make([]model.UserStrategy, 0)
		for _, v := range userStrategyId {
			userPolicyTemp, err := repository.UserStrategyDao.FindById(v)
			if err != nil {
				log.Errorf("查询用户策略失败:%v", err)
				continue
			}
			// 筛掉过期的不生效的策略,并且已启用
			if !*userPolicyTemp.IsPermanent && (userPolicyTemp.BeginValidTime.After(time.Now()) || userPolicyTemp.EndValidTime.Before(time.Now())) {
				continue
			}
			if userPolicyTemp.Status != constant.Enable {
				continue
			}
			userStrategy = append(userStrategy, userPolicyTemp)
		}
		// 先根据优先级排序再根据部门机构深度排序
		if len(userStrategy) == 0 {
			return true, nil
		}
		if len(userStrategy) > 0 {
			sort.Slice(userStrategy, func(i, j int) bool {
				if userStrategy[i].Priority == userStrategy[j].Priority {
					return userStrategy[i].DepartmentDepth < userStrategy[j].DepartmentDepth
				}
				return userStrategy[i].Priority < userStrategy[j].Priority
			})
			userStrategyValid = userStrategy[0]
		}
	}
	judgeExp, err := ExpDateService.JudgeExpUserStrategy(&userStrategyValid)
	if err != nil {
		return false, err
	}
	return judgeExp, nil
}
