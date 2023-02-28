package api

import (
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
	"sort"
	"strings"
	"tkbastion/pkg/constant"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/model"
	"tkbastion/server/utils"
)

// RoleCreateEndpoint 新增角色
func RoleCreateEndpoint(c echo.Context) error {
	var item model.UserRole
	if err := c.Bind(&item); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}

	// TODO 数据校验

	// 角色名称不可重复
	var roleExist model.Role
	err := roleRepository.DB.Where("name = ?", item.Name).Find(&roleExist).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}
	if "" != roleExist.ID {
		return FailWithDataOperate(c, 403, "角色名称已存在", "角色管理-新增: 角色名称["+item.Name+"]已存在", nil)
	}

	// 删除role_menu表中可能存在的脏数据
	if err := roleRepository.DeleteRoleMenuByName(item.Name); err != nil {
		log.Errorf("DeleteRoleMenuByName Error: %v", err)
	}
	// 菜单权限
	item.MenuIds = append(item.MenuIds, []int{130, 140}...)
	for _, id := range item.MenuIds {
		err := roleRepository.DB.Table("role_menus").Create(model.RoleMenu{Name: item.Name, MenuId: id}).Error
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 500, "新增失败", "", err)
		}
	}
	// 菜单api权限
	if err = casbinRuleRepository.Create(item.Name, item.MenuIds); nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}
	if err = global.CasbinEnforcer.LoadPolicy(); nil != err {
		log.Errorf("LoadPolicy Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}
	item.ID = utils.UUID()
	item.Created = utils.NowJsonTime()
	if err = roleRepository.Creat(&model.Role{
		ID:      item.ID,
		Name:    item.Name,
		Created: item.Created,
		IsEdit:  true,
	}); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}

	return SuccessWithOperate(c, "角色管理-新增: 角色名称["+item.Name+"]", item)
}

// RolePagingEndpoint 展示角色
func RolePagingEndpoint(c echo.Context) error {
	//var userRole []model.UserRole
	//roleArr, err := roleRepository.Find()
	//if err != nil {
	//	log.Errorf("DB Error %v", err)
	//	return FailWithDataOperate(c, 500, "获取角色列表失败", "", err)
	//}
	//// 通过角色列表找到相应角色的对应的菜单Id
	//for i := range roleArr {
	//	var roleTemp, err = GetMenuIdsByRoleId(roleArr[i].ID)
	//	if err != nil {
	//		log.Errorf("GetMenuIdsByRoleId Error %v", err)
	//		continue
	//	}
	//	userRole = append(userRole, roleTemp)
	//}
	// -----------分割线---------------------
	var userRole []model.UserRoleTest
	roleArr, err := roleRepository.Find()
	if err != nil {
		log.Errorf("DB Error %v", err)
		return FailWithDataOperate(c, 500, "获取角色列表失败", "", err)
	}
	for i := range roleArr {
		var roleTemp, err = GetMenuIdsByRoleId(roleArr[i].ID)
		if err != nil {
			log.Errorf("GetMenuIdsByRoleId Error %v", err)
			continue
		}
		menuIdsT, err := DealWithMenuIds(roleTemp.MenuIds)
		if err != nil {
			log.Errorf("DealWithMenuIds Error %v", err)
			continue
		}

		userRole = append(userRole, model.UserRoleTest{
			ID:      roleTemp.ID,
			Name:    roleTemp.Name,
			MenuIds: menuIdsT,
			Created: roleTemp.Created,
			IsEdit:  roleTemp.IsEdit,
		})
	}

	// -----------分割线---------------------

	return Success(c, H{
		"total": len(userRole),
		"items": userRole,
	})
}

// RoleUpdateEndpoint 修改角色
func RoleUpdateEndpoint(c echo.Context) error {
	id := c.Param("id")
	// 未修改前角色信息
	oldRole, err := roleRepository.FindById(id)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	var item model.UserRole
	if err := c.Bind(&item); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	// 角色名称不可重复
	var roleExist model.Role
	if err := roleRepository.DB.Where("name = ? and id != ?", item.Name, oldRole.ID).Find(&roleExist).Error; nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if "" != roleExist.ID {
		return FailWithDataOperate(c, 403, "角色名称已存在", "角色管理-修改: 角色名称["+oldRole.Name+"->"+item.Name+"],失败原因["+item.Name+"已存在]", nil)
	}

	if err := roleRepository.UpdateById(&model.Role{
		ID:      id,
		Name:    item.Name,
		Created: item.Created,
	}, id); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	// 用户角色更新
	if err := userNewRepository.UpdateByRoleId(&model.UserNew{
		RoleName: item.Name,
	}, id); err != nil {
		log.Errorf("DB Error: %v", err)
		log.Errorf("用户的角色更新失败,请正在用户列表重新选择")
	}

	// 菜单权限更新
	if err = roleRepository.DeleteRoleMenuByName(item.Name); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	secondMap := make(map[int]int, 0)
	for _, V := range MenuIds {
		secondMap[V] = 1
	}
	item.MenuIds = append(item.MenuIds, []int{130, 140}...)
	for _, ide := range item.MenuIds {
		err = roleRepository.DB.Table("role_menus").Create(model.RoleMenu{Name: item.Name, MenuId: ide}).Error
		if err != nil {
			if strings.Contains(err.Error(), "Duplicate entry") {
				continue
			} else {
				log.Errorf("DB Error: %v", err)
				return FailWithDataOperate(c, 500, "修改失败", "", err)
			}
		}
		if temp, ok := secondMap[ide]; ok && temp == 1 {
			secondMap[ide] = 2
		}
		if ide/10 >= 3010 && ide/10 <= 12030 {
			// 检测是否出现在map中
			if v, ok := secondMap[ide/10]; ok && v == 1 {
				err := roleRepository.DB.Table("role_menus").Create(model.RoleMenu{Name: item.Name, MenuId: ide / 10}).Error
				if nil != err {
					if strings.Contains(err.Error(), "Duplicate entry") {
						continue
					} else {
						log.Errorf("DB Error: %v", err)
						return FailWithDataOperate(c, 500, "修改失败", "", err)
					}
				}
				secondMap[ide/10] = 2
			}
		} else if ide/1000 >= 20 && ide/1000 <= 130 {
			if v, ok := secondMap[ide/1000]; ok && v == 1 {
				err := roleRepository.DB.Table("role_menus").Create(model.RoleMenu{Name: item.Name, MenuId: ide / 1000}).Error
				if nil != err {
					if strings.Contains(err.Error(), "Duplicate entry") {
						continue
					} else {
						log.Errorf("DB Error: %v", err)
						return FailWithDataOperate(c, 500, "修改失败", "", err)
					}
				}
				secondMap[ide/1000] = 2
			}
		} else {
			continue
		}
	}
	if err = casbinRuleRepository.Update(oldRole.Name, item.Name, item.MenuIds); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	err = global.CasbinEnforcer.LoadPolicy()
	if nil != err {
		log.Errorf("LoadPolicy Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	return SuccessWithOperate(c, "角色管理-修改: 角色名称["+oldRole.Name+"->"+item.Name+"]", item)
}

// RoleDeleteEndpoint 删除角色
func RoleDeleteEndpoint(c echo.Context) error {
	id := c.Param("id")
	role, err := roleRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "", err)
	}
	// 检查角色是否被使用
	var user []model.UserNew
	user, err = userNewRepository.FindByRoleId(role.ID)
	if err != nil {
		log.Errorf("DB Error %v", err)
	}
	if len(user) > 0 {
		return FailWithDataOperate(c, 403, "角色正在使用中", "角色管理-删除: 角色名称["+role.Name+"],失败原因[角色已被使用]", nil)
	}
	if err := DeleteRoleByRoleId(id); err != nil {
		log.Errorf("DB Error %v", err)
		return FailWithDataOperate(c, 500, "删除失败", "", err)
	}
	return SuccessWithOperate(c, "角色管理-删除: 角色名称["+role.Name+"]", nil)
}

// RoleMenuEndpoint 获取该用户的角色权限菜单
func RoleMenuEndpoint(c echo.Context) error {
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 401, "您的登录信息已失效,请重新登录后再试.")
	}
	// 根据用户的角色id获取角色名称
	role, err := roleRepository.FindById(account.RoleId)
	if err != nil {
		log.Errorf("DB Error %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", err)
	}
	result, err := GetMenuByRole(role.Name)
	if err != nil {
		log.Errorf("DB Error %v", err)
		return FailWithDataOperate(c, 500, "获取失败", "", err)
	}

	if err != nil {
		return Fail(c, 500, "查询失败.")
	}

	less := func(i, j int) bool {
		return result[i].Id < result[j].Id
	}
	sort.Slice(result, less)
	return Success(c, result)
}

// 获取角色树

func RoleButtonMenuTreeSelectEndpoint(c echo.Context) error {
	var menus []model.Menu
	if err := roleRepository.DB.Where(" type in ('M','C','F')").Find(&menus).Error; err != nil {
		log.Errorf("db error:%s", err)
	}
	result := make([]model.Menu, 0)
	for i := 0; i < len(menus); i++ {
		if menus[i].Id == 10 {
			menusInfo := menuCall(&menus, menus[i])
			result = append(result, menusInfo)
		}
		if menus[i].Id == 20 {
			menusInfo := menuCall(&menus, menus[i])
			result = append(result, menusInfo)
		}
		if menus[i].Type == "C" {
			menusInfo := menuCall(&menus, menus[i])
			result = append(result, menusInfo)
		}
	}
	return Success(c, result)
}

func DeleteRoleByRoleId(id string) error {
	role, err := roleRepository.FindById(id)
	if err != nil {
		log.Errorf("DB Error %v", err)
		return err
	}

	// 删除角色role
	if err := roleRepository.DeleteById(id); err != nil {
		log.Errorf("DB Error %v", err)
		return err
	}
	// 删除角色role_menus
	if err := roleRepository.DeleteRoleMenuByName(role.Name); err != nil {
		log.Errorf("DB Error %v", err)
		return err
	}
	// 删除casbin_rule
	if err := casbinRuleRepository.DeleteByRoleName(role.Name); err != nil {
		log.Errorf("DB Error %v", err)
	}
	return nil
}

func GetMenuByRole(roleName string) (m []model.Menu, err error) {
	menus, err := getMenusByRoleName(roleName)
	m = make([]model.Menu, 0)
	for i := 0; i < len(menus); i++ {
		if menus[i].ParentId != 0 || menus[i].Id == 140 {
			continue
		}
		menusInfo := menuCall(&menus, menus[i])
		m = append(m, menusInfo)
	}
	return
}

func getMenusByRoleName(roleName string) ([]model.Menu, error) {
	var MenuList []model.Menu
	var role model.Role
	var err error

	role.Name = roleName
	buttons := make([]model.Menu, 0)
	err = roleRepository.DB.Debug().Model(&role).Where("name = ?", roleName).Preload("Menu", func(db *gorm.DB) *gorm.DB {
		return db.Where(" type in ('C') or id in ('10','20','130','140') ")
	}).Find(&role).Error

	if role.Menu != nil {
		buttons = *role.Menu

		for _, menu := range buttons {
			MenuList = append(MenuList, menu)
		}
	}
	mIds := make([]int, 0)
	for _, menu := range buttons {
		if menu.ParentId != 0 {
			mIds = append(mIds, menu.ParentId)
		}
	}
	var dataC []model.Menu
	err = roleRepository.DB.Where(" type in ('M') and id in ?", mIds).Find(&dataC).Error
	if err != nil {
		return nil, err
	}
	for _, datum := range dataC {
		MenuList = append(MenuList, datum)
	}

	if err != nil {
		log.Errorf("db error:%s", err)
	}
	return MenuList, err
}

// menuCall 构建菜单树
func menuCall(menuList *[]model.Menu, menu model.Menu) model.Menu {
	list := *menuList

	min := make([]model.Menu, 0)
	for j := 0; j < len(list); j++ {

		if menu.Id != list[j].ParentId {
			continue
		}
		mi := model.Menu{}
		mi.Id = list[j].Id
		mi.Title = list[j].Title
		mi.Name = list[j].Name
		mi.Path = list[j].Path
		mi.Paths = list[j].Paths
		mi.Icon = list[j].Icon
		mi.Component = list[j].Component
		mi.Type = list[j].Type
		mi.ParentId = list[j].ParentId
		mi.Children = []model.Menu{}

		if mi.Type != constant.Button {
			ms := menuCall(menuList, mi)
			min = append(min, ms)
		} else {
			min = append(min, mi)
		}
	}
	menu.Children = min
	return menu
}

// MenuIds 返回过滤掉的无子节点的二级菜单
var MenuIds = []int{
	20,
	30, 3010, 3020, 3030, 3040,
	40, 4010,
	4020, 4030, 4040, 4050,
	50, 5010, 5020,
	60, 6010, 6020,
	70, 7010, 7020,
	80, 8010, 8020, 8030, 8040,
	90, 9010, 9020, 9030, 9040,
	100, 10020, 10030, 10040,
	110, 11020, 11030,
	120, 12010, 12020, 12030,
}

func GetMenuIdsByRoleId(roleId string) (roleMenu model.UserRole, err error) {
	item, err := roleRepository.FindById(roleId)
	if err != nil {
		return model.UserRole{}, err
	}
	var menuIds []int
	menuIds, err = roleRepository.FindThreeLevelMenuIdsById(item.Name, MenuIds)
	if nil != err {
		return model.UserRole{}, nil
	}
	roleMenu = model.UserRole{
		ID:      item.ID,
		Name:    item.Name,
		IsEdit:  item.IsEdit,
		Created: item.Created,
		MenuIds: menuIds,
	}
	return roleMenu, nil
}

// DealWithMenuIds 处理menuIds 返回二维数组
func DealWithMenuIds(menuIds []int) (menuIdList [][]int, err error) {
	menuIdList = make([][]int, 33)
	for i := 0; i < len(menuIdList); i++ {
		menuIdList[i] = make([]int, 0)
	}
	resMap := make(map[int]int, 0)
	for _, v := range menuIds {
		resMap[v] = v / 10
		switch resMap[v] {
		case 1:
			menuIdList[0] = append(menuIdList[0], v)
		case 2010:
			menuIdList[1] = append(menuIdList[1], v)
		case 3010:
			menuIdList[2] = append(menuIdList[2], v)
		case 3020:
			menuIdList[3] = append(menuIdList[3], v)
		case 3030:
			menuIdList[4] = append(menuIdList[4], v)
		case 3040:
			menuIdList[5] = append(menuIdList[5], v)
		case 4010, 4041:
			menuIdList[6] = append(menuIdList[6], v)
		case 4020:
			menuIdList[7] = append(menuIdList[7], v)
		case 4030:
			menuIdList[8] = append(menuIdList[8], v)
		case 4040:
			menuIdList[9] = append(menuIdList[9], v)
		case 4050:
			menuIdList[10] = append(menuIdList[10], v)
		case 5010:
			menuIdList[11] = append(menuIdList[11], v)
		case 5020, 5021:
			menuIdList[12] = append(menuIdList[12], v)
		case 6010:
			menuIdList[13] = append(menuIdList[13], v)
		case 6020:
			menuIdList[14] = append(menuIdList[14], v)
		case 7010:
			menuIdList[15] = append(menuIdList[15], v)
		case 7020:
			menuIdList[16] = append(menuIdList[16], v)
		case 8010:
			menuIdList[17] = append(menuIdList[17], v)
		case 8020:
			menuIdList[18] = append(menuIdList[18], v)
		case 8030:
			menuIdList[19] = append(menuIdList[19], v)
		case 8040, 8041:
			menuIdList[20] = append(menuIdList[20], v)
		case 9010:
			menuIdList[21] = append(menuIdList[21], v)
		case 9020:
			menuIdList[22] = append(menuIdList[22], v)
		case 9030:
			menuIdList[23] = append(menuIdList[23], v)
		case 9040:
			menuIdList[24] = append(menuIdList[24], v)
		case 10020, 10021:
			menuIdList[25] = append(menuIdList[25], v)
		case 10030, 10031:
			menuIdList[26] = append(menuIdList[26], v)
		case 10040, 10041:
			menuIdList[27] = append(menuIdList[27], v)
		case 11020:
			menuIdList[28] = append(menuIdList[28], v)
		case 11030:
			menuIdList[29] = append(menuIdList[29], v)
		case 12010:
			menuIdList[30] = append(menuIdList[30], v)
		case 12020:
			menuIdList[31] = append(menuIdList[31], v)
		case 12030:
			menuIdList[32] = append(menuIdList[32], v)
		default:
			continue
		}
	}
	return menuIdList, nil
}
