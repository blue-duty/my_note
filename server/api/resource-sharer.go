package api

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

type RU struct {
	UserGroupId  string   `json:"userGroupId"`
	UserId       string   `json:"userId"`
	StrategyId   string   `json:"strategyId"`
	ResourceType string   `json:"resourceType"`
	ResourceIds  []string `json:"resourceIds"`
}

type UR struct {
	ResourceId   string   `json:"resourceId"`
	ResourceType string   `json:"resourceType"`
	UserIds      []string `json:"userIds"`
}

func RSGetSharersEndPoint(c echo.Context) error {
	resourceId := c.QueryParam("resourceId")
	resourceType := c.QueryParam("resourceType")
	userId := c.QueryParam("userId")
	userGroupId := c.QueryParam("userGroupId")
	userIds, err := resourceSharerRepository.Find(resourceId, resourceType, userId, userGroupId)
	if err != nil {
		return err
	}
	return Success(c, userIds)
}

func RSOverwriteSharersEndPoint(c echo.Context) error {
	var ur UR
	if err := c.Bind(&ur); err != nil {
		return err
	}

	if err := resourceSharerRepository.OverwriteUserIdsByResourceId(ur.ResourceId, ur.ResourceType, ur.UserIds); err != nil {
		return err
	}

	return Success(c, "")
}

func ResourceRemoveByUserIdAssignEndPoint(c echo.Context) error {
	var ru RU
	u, _ := GetCurrentAccount(c)
	if err := c.Bind(&ru); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "授权凭证删除失败", "删除授权凭证: "+u.Nickname+"删除失败", err)
	}

	if err := resourceSharerRepository.DeleteByUserIdAndResourceTypeAndResourceIdIn(ru.UserGroupId, ru.UserId, ru.ResourceType, ru.ResourceIds); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "授权凭证删除失败", "删除授权凭证: "+u.Nickname+"删除失败", err)
	}

	return SuccessWithOperate(c, "授权凭证删除成功", "删除授权凭证: "+u.Nickname+"删除成功")
}

func ResourceAddByUserIdAssignEndPoint(c echo.Context) error {
	var ru RU
	u, _ := GetCurrentAccount(c)
	if err := c.Bind(&ru); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "授权凭证添加失败", "添加授权凭证: "+u.Nickname+"添加失败", err)
	}

	if err := resourceSharerRepository.AddSharerResources(ru.UserGroupId, ru.UserId, ru.StrategyId, ru.ResourceType, ru.ResourceIds); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "授权凭证添加失败", "添加授权凭证: "+u.Nickname+"添加失败", err)
	}

	return SuccessWithOperate(c, "授权凭证添加成功", "添加授权凭证: "+u.Nickname+"添加成功")
}
