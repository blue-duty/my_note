package api

import (
	"context"
	"github.com/dustin/go-humanize"
	"github.com/labstack/echo/v4"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"time"
	"tkbastion/pkg/log"
	"tkbastion/server/dto"
)

// OverviewCounterNewEndPoint 用户、应用、设备、告警数量
func OverviewCounterNewEndPoint(c echo.Context) error {
	// 获取当前用户
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, 401, "您的登录信息已失效,请重新登录后再试.")
	}
	// 获取所有下属部门的id
	var departmentIds []int64
	if err := GetChildDepIds(account.DepartmentId, &departmentIds); err != nil {
		log.Errorf("GetChildDepIds Error: %v", err)
	}
	// 获取当前用户下属部门的所有用户
	users, err := GetCurrentDepartmentUserChildren(account.DepartmentId)
	if err != nil {
		return Fail(c, 500, "获取用户信息失败")
	}
	// 获取当前用户下属部门的所有资产
	assets, err := newAssetRepository.GetAssetListByDepartmentIds(context.TODO(), departmentIds)
	if err != nil {
		return Fail(c, 500, "获取资产信息失败")
	}
	// 获取当前用户下属部门的所有应用
	apps, err := newApplicationRepository.FindByDepartmentIds(context.TODO(), departmentIds)
	if err != nil {
		return Fail(c, 500, "获取应用信息失败")
	}
	// 获取当前用户的所有的告警信息
	//message, err := messageRepository.FindByRecIdAndTheme(account.ID, constant.AlertMessage)
	//if err != nil {
	//	log.Errorf("FindByRecId Error: %v", err)
	//}
	// 获取系统告警日志的数量
	systemAlarm, err := operateAlarmLogRepository.GetSystemAlarmCount(context.TODO())
	if err != nil {
		log.Errorf("GetSystemAlarmCount Error: %v", err)
	}
	var counter dto.Counter
	counter.User = int64(len(users))
	counter.Asset = int64(len(assets))
	counter.Application = int64(len(apps))
	counter.Alarm = systemAlarm
	return Success(c, counter)
}

// 最近用户/资产访问量
func OverviewRecentUserAndAssetCountEndPoint(c echo.Context) (err error) {
	var recentCount dto.RecentCount
	recentCount.VisitUserByWeek, err = loginLogRepository.GetUserCountThisWeek()
	if err != nil {
		log.Errorf("GetUserCountThisWeek Error: %v", err)
	}
	recentCount.VisitUserByMonth, err = loginLogRepository.GetUserCountThisMonth()
	if err != nil {
		log.Errorf("GetUserCountThisMonth Error: %v", err)
	}
	recentCount.VisitUserByYear, err = loginLogRepository.GetUserCountThisYear()
	if err != nil {
		log.Errorf("GetUserCountThisYear Error: %v", err)
	}
	recentCount.VisitDeviceByWeek, err = loginLogRepository.GetDeviceCountThisWeek()
	if err != nil {
		log.Errorf("GetDeviceCountThisWeek Error: %v", err)
	}
	recentCount.VisitDeviceByMonth, err = loginLogRepository.GetDeviceCountThisMonth()
	if err != nil {
		log.Errorf("GetDeviceCountThisMonth Error: %v", err)
	}
	recentCount.VisitDeviceByYear, err = loginLogRepository.GetDeviceCountThisYear()
	if err != nil {
		log.Errorf("GetDeviceCountThisYear Error: %v", err)
	}
	return Success(c, recentCount)
}

// 最近用户访问、资产访问top5
func OverviewRecentAssetAccessCountEndPoint(c echo.Context) (err error) {
	var cumulativeVisitsTop dto.CumulativeVisitsTop
	cumulativeVisitsTop.VisitUserByWeek, err = loginLogRepository.GetUserTop5CountThisWeek()
	if err != nil {
		log.Errorf("GetUserTopCountThisWeek Error: %v", err)
	}
	cumulativeVisitsTop.VisitUserByMonth, err = loginLogRepository.GetUserTop5CountThisMonth()
	if err != nil {
		log.Errorf("GetUserTopCountThisMonth Error: %v", err)
	}
	cumulativeVisitsTop.VisitUserByYear, err = loginLogRepository.GetUserTop5CountThisYear()
	if err != nil {
		log.Errorf("GetUserTopCountThisYear Error: %v", err)
	}
	cumulativeVisitsTop.VisitDeviceByWeek, err = loginLogRepository.GetDeviceTop5CountThisWeek()
	if err != nil {
		log.Errorf("GetDeviceTopCountThisWeek Error: %v", err)
	}
	cumulativeVisitsTop.VisitDeviceByMonth, err = loginLogRepository.GetDeviceTop5CountThisMonth()
	if err != nil {
		log.Errorf("GetDeviceTopCountThisMonth Error: %v", err)
	}
	cumulativeVisitsTop.VisitDeviceByYear, err = loginLogRepository.GetDeviceTop5CountThisYear()
	if err != nil {
		log.Errorf("GetDeviceTopCountThisYear Error: %v", err)
	}
	return Success(c, cumulativeVisitsTop)
}

// OverviewUnapprovedListEndpoint 待审批工单
func OverviewUnapprovedListEndpoint(c echo.Context) (err error) {
	// 获取当前用户
	account, found := GetCurrentAccountNew(c)
	if !found {
		return Fail(c, -1, "当前用户信息已过期，请重新登录")
	}
	// 获取当前部门以下所有部门id
	var depIds []int64
	err = GetChildDepIds(account.DepartmentId, &depIds)
	if err != nil {
		return err
	}
	// 通过部门id查询工单列表
	// 查看自己所有未审批的工单
	unApproval, err := workOrderNewRepository.GetUnApprovalWorkOrder(account.RoleName, depIds)
	if err != nil {
		return err
	}
	var workOrderList []dto.WorkOrderList
	for _, v := range unApproval {
		workOrderApprovalLog, err := workOrderApprovalLogRepository.FindByWorkOrderId(v.ID)
		if err != nil {
			log.Errorf("WorkOrderApprovalLog FindByWorkOrderId err: %v", err)
			continue
		}
		// 判断当前用户是否能看见
		for _, v1 := range workOrderApprovalLog {
			if account.DepartmentId == v1.DepartmentId {
				workOrderList = append(workOrderList, dto.WorkOrderList{
					OrderType:   v.WorkOrderType,
					WorkOrderId: v.OrderId,
					Applicant:   v.ApplyUser,
					Nickname:    v.ApplyUser,
				})
			}
		}
	}
	return Success(c, workOrderList)
}

// OverviewAccessCountEndpoint 访问量统计
func OverviewAccessCountEndpoint(c echo.Context) (err error) {
	var trafficStatistics dto.TrafficStatistics
	if err = newSessionRepository.GetAccessMonthsCount(context.TODO(), []string{"vnc", "rdp"}, &trafficStatistics.Graphics); err != nil {
		log.Errorf("GetAccessMonthsCount Error: %v", err)
	}
	if err = newSessionRepository.GetAccessMonthsCount(context.TODO(), []string{"ssh", "telnet"}, &trafficStatistics.Characters); err != nil {
		log.Errorf("GetAccessMonthsCount Error: %v", err)
	}
	if err = newSessionRepository.GetAccessMonthsCount(context.TODO(), []string{"应用", "app"}, &trafficStatistics.AppCount); err != nil {
		log.Errorf("GetAccessMonthsCount Error: %v", err)
	}
	if err = newSessionRepository.GetAccessMonthsCount(context.TODO(), []string{"ftp", "sftp"}, &trafficStatistics.FileCount); err != nil {
		log.Errorf("GetAccessMonthsCount Error: %v", err)
	}
	return Success(c, trafficStatistics)
}

// 获取系统信息
func OverviewSystemInfoEndpoint(c echo.Context) (err error) {
	// CPU 占用
	var stat dto.OverviewStat
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		log.Errorf("cpu.Percent Error: %v", err)
	}
	// 内存占用
	memPercent, err := mem.VirtualMemory()
	if err != nil {
		log.Errorf("mem.VirtualMemory Error: %v", err)
	}
	// 磁盘占用
	diskPercent, err := disk.Usage("/")
	if err != nil {
		log.Errorf("disk.Usage Error: %v", err)
	}
	// Swap占用
	swapPercent, err := mem.SwapMemory()
	if err != nil {
		log.Errorf("mem.SwapMemory Error: %v", err)
	}
	stat.CpuPercent = int(cpuPercent[0])
	// 单位转换
	stat.Mem.Total = humanize.Bytes(memPercent.Total)
	stat.Mem.Used = humanize.Bytes(memPercent.Used)
	stat.Mem.Free = humanize.Bytes(memPercent.Free)
	stat.Disk.Total = humanize.Bytes(diskPercent.Total)
	stat.Disk.Used = humanize.Bytes(diskPercent.Used)
	stat.Disk.Free = humanize.Bytes(diskPercent.Free)
	stat.Swap.Total = humanize.Bytes(swapPercent.Total)
	stat.Swap.Used = humanize.Bytes(swapPercent.Used)
	stat.Swap.Free = humanize.Bytes(swapPercent.Free)

	return SuccessWithOperate(c, "获取系统信息成功", stat)
}
