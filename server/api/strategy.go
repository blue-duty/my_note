package api

//import (
//	"strconv"
//	"strings"
//	errs "tkbastion/pkg/error"
//	"tkbastion/pkg/log"
//	"tkbastion/pkg/validator"
//	"tkbastion/server/model"
//	"tkbastion/server/utils"
//
//	"github.com/labstack/echo/v4"
//)
//
//func StrategyAllEndpoint(c echo.Context) error {
//	items, err := strategyRepository.FindAll()
//	if err != nil {
//		return err
//	}
//	return Success(c, items)
//}
//
//func StrategyPagingEndpoint(c echo.Context) error {
//	pageIndex, _ := strconv.Atoi(c.QueryParam("pageIndex"))
//	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))
//	name := c.QueryParam("name")
//
//	order := c.QueryParam("order")
//	field := c.QueryParam("field")
//
//	items, total, err := strategyRepository.Find(pageIndex, pageSize, name, order, field)
//	if err != nil {
//		return err
//	}
//
//	return Success(c, H{
//		"total": total,
//		"items": items,
//	})
//}
//
//func StrategyCreateEndpoint(c echo.Context) error {
//	var item model.Strategy
//	if err := c.Bind(&item); err != nil {
//		log.Errorf("Bind Error: %v", err)
//		return FailWithDataOperate(c, 500, "新增失败", "", err)
//	}
//
//	//数据校验
//	if err := c.Validate(item); err != nil {
//		msg := validator.GetVdErrMsg(err)
//		logMsg := validator.GetErrLogMsg(err, errs.StrategiesCreatLog)
//		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
//	}
//
//	// 授权策略名称不可重复
//	var itemExists []model.Strategy
//	err := strategyRepository.DB.Where("name = ?", item.Name).Find(&itemExists).Error
//	if nil != err {
//		log.Errorf("DB Error: %v", err)
//		return FailWithDataOperate(c, 500, "新增失败", "", err)
//	}
//	if 0 != len(itemExists) {
//		return FailWithDataOperate(c, 422, "策略名称已存在", "新增策略: 名称"+item.Name+"已存在", nil)
//	}
//
//	item.ID = utils.UUID()
//	item.Created = utils.NowJsonTime()
//
//	if err := strategyRepository.Create(&item); err != nil {
//		log.Errorf("DB Error: %v", err)
//		return FailWithDataOperate(c, 500, "新增失败", "", err)
//	}
//	return SuccessWithOperate(c, "新增策略: 策略名称"+item.Name, item)
//}
//
//func StrategyDeleteEndpoint(c echo.Context) error {
//	ids := c.Param("id")
//	split := strings.Split(ids, ",")
//	var successDeleteCount int
//	var successDeleteStrategies string
//	for i := range split {
//		strategyInfo, err := strategyRepository.FindById(split[i])
//		if nil != err {
//			log.Errorf("DB Error: %v", err)
//			return FailWithDataOperate(c, 400, "删除失败", "删除授权策略: "+successDeleteStrategies+" 删除成功数"+strconv.Itoa(successDeleteCount)+", 删除失败授权策略id: "+split[i], err)
//		}
//		if err := strategyRepository.DeleteByDepartmentId(split[i]); err != nil {
//			log.Errorf("DB Error: %v", err)
//			return FailWithDataOperate(c, 400, "删除失败", "删除授权策略: "+successDeleteStrategies+" 删除成功数"+strconv.Itoa(successDeleteCount)+", 删除失败授权策略: "+strategyInfo.Name, err)
//		}
//		successDeleteCount++
//		successDeleteStrategies += strategyInfo.Name + ","
//
//	}
//	return SuccessWithOperate(c, "删除授权策略: "+successDeleteStrategies+" 删除成功数"+strconv.Itoa(successDeleteCount), nil)
//}
//
//func StrategyUpdateEndpoint(c echo.Context) error {
//	id := c.Param("id")
//	var item model.Strategy
//	if err := c.Bind(&item); err != nil {
//		log.Errorf("Bind Error: %v", err)
//		return FailWithDataOperate(c, 500, "新增失败", "", err)
//	}
//
//	//数据校验
//	if err := c.Validate(item); err != nil {
//		msg := validator.GetVdErrMsg(err)
//		logMsg := validator.GetErrLogMsg(err, errs.StrategiesUpdateLog)
//		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
//	}
//
//	if err := strategyRepository.UpdateById(&item, id); err != nil {
//		log.Errorf("DB Error: %v", err)
//		return FailWithDataOperate(c, 500, "修改失败", "", err)
//	}
//	return SuccessWithOperate(c, "修改授权策略: 名称"+item.Name, nil)
//}
