package api

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"tkbastion/pkg/constant"
	"tkbastion/server/utils"

	"tkbastion/pkg/config"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/service"

	"github.com/casbin/casbin/v2/util"
	"github.com/labstack/echo/v4"
)

func ErrorHandler(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {

		if err := next(c); err != nil {

			if he, ok := err.(*echo.HTTPError); ok {
				message := fmt.Sprintf("%v", he.Message)
				return Fail(c, he.Code, message)
			}

			return Fail(c, 0, err.Error())
		}
		return nil
	}
}

func Auth(next echo.HandlerFunc) echo.HandlerFunc {
	startWithUrls := constant.LoadMiddlewarePath
	startWithUrls = append(startWithUrls, []string{"/static", "/login", "/userLoginType"}...)
	download := regexp.MustCompile(`^/sessions/\w{8}(-\w{4}){3}-\w{12}/download`)
	recording := regexp.MustCompile(`^/sessions/\w{8}(-\w{4}){3}-\w{12}/recording`)

	return func(c echo.Context) error {

		uri := c.Request().RequestURI
		if uri == "/" || strings.HasPrefix(uri, "/#") {
			return next(c)
		}
		// 路由拦截 - 登录身份、资源权限判断等
		for i := range startWithUrls {
			if strings.HasPrefix(uri, startWithUrls[i]) {
				return next(c)
			}
		}

		if download.FindString(uri) != "" {
			return next(c)
		}

		if recording.FindString(uri) != "" {
			return next(c)
		}

		token := GetToken(c)
		cacheKey := BuildCacheKeyByToken(token)
		auth, found := global.Cache.Get(cacheKey)
		if !found {
			return Fail(c, 401, "您的登录信息已失效,请重新登录后再试.")
		}

		authorization := global.AuthorizationNew{
			Token:          auth.(global.AuthorizationNew).Token,
			Remember:       false,
			UserNew:        auth.(global.AuthorizationNew).UserNew,
			LoginTime:      auth.(global.AuthorizationNew).LoginTime,
			LastActiveTime: utils.NowJsonTime(),
			LoginAddress:   auth.(global.AuthorizationNew).LoginAddress,
		}
		item, err := propertyRepository.FindByName("login-session-overtime")
		if nil != err {
			log.Errorf("DB Error: %v", err.Error())
		}
		iExpirationTime, err := strconv.Atoi(item.Value)
		if nil != err {
			log.Errorf("Atoi Error: %v", err.Error())
		}
		rememberEffectiveTime := time.Minute * time.Duration(iExpirationTime)
		global.Cache.Set(cacheKey, authorization, rememberEffectiveTime)

		return next(c)
	}
}

// AuthCheckRole Casbin-Api权限检查中间件
func AuthCheckRole(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var err error
		startWithUrls := constant.LoadMiddlewarePath
		startWithUrls = append(startWithUrls, []string{"/static", "/login", "/userLoginType", "/info", "/user-role-menus", "/tree-role-select", "/sys-configs/extend", "/sys-configs/ui-config", "/passwd/export-auth"}...)
		uri := c.Request().RequestURI
		if uri == "/" || strings.HasPrefix(uri, "/#") {
			return next(c)
		}
		for i := range startWithUrls {
			if strings.HasPrefix(uri, startWithUrls[i]) {
				return next(c)
			}
		}
		var casbinExclude bool
		for _, i := range config.CasbinExclude {
			if util.KeyMatch2(uri, i.Url) && c.Request().Method == i.Method {
				casbinExclude = true
				break
			}
		}
		if casbinExclude {
			log.Debugf("Casbin exclusion, no validation method:%s path:%s", c.Request().Method, uri)
			return next(c)
		}
		account, found := GetCurrentAccountNew(c)
		if !found {
			return Fail(c, 401, "您的登录信息已失效,请重新登录后再试.")
		}
		//if account.RoleName == constant.SystemAdmin {
		//	return next(c)
		//}
		e := global.CasbinEnforcer
		var res bool
		//检查权限
		res, err = e.Enforce(account.RoleName, uri, c.Request().Method)
		if err != nil {
			log.Errorf("AuthCheckRole error:%s method:%s path:%s", err, c.Request().Method, uri)
			return Fail(c, 500, "")
		}
		if !res {
			log.Warnf("isTrue: %v role: %s method: %s path: %s message: %s", res, account.RoleName, c.Request().Method, uri, "当前request无权限,请管理员确认！")
			return Fail(c, 403, "对不起,您没有该接口访问权限,请联系管理员")
		}
		return next(c)
	}
}

// ExpDataCheck 限制日期检测
func ExpDataCheck(next echo.HandlerFunc) echo.HandlerFunc {

	download := regexp.MustCompile(`^/sessions/\w{8}(-\w{4}){3}-\w{12}/download`)
	recording := regexp.MustCompile(`^/sessions/\w{8}(-\w{4}){3}-\w{12}/recording`)

	return func(c echo.Context) error {

		startWithUrls := constant.LoadMiddlewarePath
		startWithUrls = append(startWithUrls, []string{"/login"}...)
		uri := c.Request().RequestURI
		if uri == "/" || strings.HasPrefix(uri, "/#") {
			return next(c)
		}
		// 路由拦截 - 登录身份、资源权限判断等
		for i := range startWithUrls {
			if strings.HasPrefix(uri, startWithUrls[i]) {
				return next(c)
			}
		}

		if download.FindString(uri) != "" {
			return next(c)
		}

		if recording.FindString(uri) != "" {
			return next(c)
		}

		token := GetToken(c)
		if token == "" {
			return next(c)
		}

		//限制日期检测
		//e := exp.ExpDateService{}
		ok, err := service.ExpDateService.JudgeExpByToken(token)
		if err != nil {
			log.Errorf("ExpDataCheck error:%s method:%s path:%s", err, c.Request().Method, c.Request().RequestURI)
			return Fail(c, 500, "")
		}
		if ok {
			return next(c)
		} else {
			return Fail(c, 405, "对不起,该时间段无法访问,请联系管理员")
		}

	}
}
