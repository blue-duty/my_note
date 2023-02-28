package dto

type RecentCount struct {
	VisitUserByWeek    []DayCount
	VisitUserByMonth   []DayCount
	VisitUserByYear    []DayCount
	VisitDeviceByWeek  []DayCount
	VisitDeviceByMonth []DayCount
	VisitDeviceByYear  []DayCount
}

type DayCount struct {
	Cnt int    `json:"cnt"`
	Day string `json:"day"`
}

type DayCountTemp struct {
	Cnt int `json:"cnt"`
	Day int `json:"day"`
}

type CumulativeVisitsTop struct {
	VisitUserByWeek    []VisitsUserTop
	VisitUserByMonth   []VisitsUserTop
	VisitUserByYear    []VisitsUserTop
	VisitDeviceByWeek  []VisitsDeviceTop
	VisitDeviceByMonth []VisitsDeviceTop
	VisitDeviceByYear  []VisitsDeviceTop
}
type VisitsUserTop struct {
	Cnt      int    `json:"cnt"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

type VisitsDeviceTop struct {
	Cnt        int    `json:"cnt"`
	DeviceName string `json:"deviceName"`
	DeviceIp   string `json:"deviceIp"`
}

type TrafficStatistics struct {
	Graphics   AccessCount
	Characters AccessCount
	AppCount   AccessCount
	FileCount  AccessCount
}

type AccessCount struct {
	TotalCount       int64 `json:"totalCount"`
	RecentMonthCount int64 `json:"recentMonthCount"`
}

type OverviewStat struct {
	CpuPercent int    `json:"cpuPercent"`
	Mem        Status `json:"mem"`
	Swap       Status `json:"swap"`
	Disk       Status `json:"disk"`
}

type Status struct {
	Total string `json:"total"`
	Used  string `json:"used"`
	Free  string `json:"free"`
}

type Counter struct {
	User        int64 `json:"user"`
	Asset       int64 `json:"asset"`
	Application int64 `json:"application"`
	Alarm       int64 `json:"alarm"`
}

type WorkOrderList struct {
	WorkOrderId string `json:"workOrderId"`
	Applicant   string `json:"applicant"`
	Nickname    string `json:"nickname"`
	OrderType   string `json:"orderType"`
}

type WorkOrderPendingApproval struct {
	WorkOrderId string `json:"workOrderId"`
	Applicant   string `json:"applicant"`
	OrderType   string `json:"orderType"`
	Department  string `json:"department"`
	ApplyTime   string `json:"applyTime"`
}
