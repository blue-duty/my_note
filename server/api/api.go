package api

import (
	"tkbastion/pkg/constant"
	"tkbastion/pkg/global"
	"tkbastion/server/model"

	"github.com/labstack/echo/v4"
)

type H map[string]interface{}

/*
	系统日志只记录系统相关错误,用户相关操作错误记录进操作日志
*/

func Fail(c echo.Context, code int, message string) error {
	return c.JSON(200, H{
		"code":    code,
		"message": message,
	})
}

func FailWithData(c echo.Context, code int, message string, data interface{}) error {
	return c.JSON(200, H{
		"code":    code,
		"message": message,
		"data":    data,
	})
}

/*
新开发的api,返回失败信息时统一调用此接口
message为返回浏览器信息,operate为记录进操作日志信息
无data传nil,无operate传"",message根据具体功能传相关信息:新增/修改/删除失败、具体业务相关信息、不确定的情况统一传"操作失败"
系统错误时code填500且不记录操作日志
*/
func FailWithDataOperate(c echo.Context, code int, message, operate string, data interface{}) error {
	return c.JSON(200, H{
		"code":    code,
		"message": message,
		"operate": operate,
		"data":    data,
	})
}

func Success(c echo.Context, data interface{}) error {
	return c.JSON(200, H{
		"code":    1,
		"message": "成功",
		"data":    data,
	})
}
func SuccessWithData(c echo.Context, code int, message, data interface{}) error {
	return c.JSON(200, H{
		"code":    code,
		"message": message,
		"data":    data,
	})
}

/*
新开发的api,返回成功信息时统一调用此接口
无data传nil,无operate传""
*/
func SuccessWithOperate(c echo.Context, operate string, data interface{}) error {
	return c.JSON(200, H{
		"code":    1,
		"message": "成功",
		"operate": operate,
		"data":    data,
	})
}

func NotFound(c echo.Context, message string) error {
	return c.JSON(200, H{
		"code":    -1,
		"message": message,
	})
}

func GetToken(c echo.Context) string {
	token := c.Request().Header.Get(constant.Token)
	if len(token) > 0 {
		return token
	}
	return c.QueryParam(constant.Token)
}

func GetCurrentAccount(c echo.Context) (model.UserNew, bool) {
	token := GetToken(c)
	cacheKey := BuildCacheKeyByToken(token)
	_, b := global.Cache.Get(cacheKey)
	if b {
		//return get.(global.Authorization).User, true
	}

	return model.UserNew{}, false
}

func GetCurrentAccountNew(c echo.Context) (model.UserNew, bool) {
	token := GetToken(c)
	cacheKey := BuildCacheKeyByToken(token)
	get, b := global.Cache.Get(cacheKey)
	if b {
		return get.(global.AuthorizationNew).UserNew, true
	}
	return model.UserNew{}, false
}

func HasPermission(c echo.Context, owner string) bool {
	// 检测是否登录
	account, found := GetCurrentAccountNew(c)
	if !found {
		return false
	}
	// 检测是否为管理人员
	if constant.SystemAdmin == account.RoleName {
		return true
	}
	// 检测是否为所有者
	if owner == account.ID {
		return true
	}
	return false
}
