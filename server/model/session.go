package model

import (
	"tkbastion/server/utils"
)

type Session struct {
	ID           string `gorm:"type:varchar(128);primary_key;not null;comment:会话id" json:"id"`
	Protocol     string `gorm:"type:varchar(16);not null;comment:连接协议" json:"protocol"`
	ConnectionId string `gorm:"type:varchar(128);not null;comment:连接会话id" json:"connectionId"`
	PassportId   string `gorm:"type:varchar(128);index;not null;comment:资产id" json:"passportId"`
	AssetName    string `gorm:"type:varchar(128);not null;comment:资产名称" json:"assetName"`
	AssetIP      string `gorm:"type:varchar(64);not null;comment:资产ip" json:"assetIP"`
	AssetPort    int    `gorm:"type:int(5);not null;comment:资产端口" json:"assetPort"`
	PassPort     string `gorm:"type:varchar(64);not null;comment:资产账号" json:"passPort"`
	DepartmentId int64  `gorm:"type:bigint(20);not null;comment:部门id" json:"departmentId"`
	Department   string `gorm:"type:varchar(128);not null;comment:部门名称" json:"department"`
	Creator      string `gorm:"type:varchar(128);index;not null;comment:建立会话用户id" json:"creator"`
	CreateName   string `gorm:"type:varchar(128);not null;comment:建立会话用户名" json:"createName"`
	CreateNick   string `gorm:"type:varchar(128);not null;comment:建立会话用户昵称" json:"createNick"`
	ClientIP     string `gorm:"type:varchar(64);default:'';comment:建立会话用户ip" json:"clientIp"`
	Width        int    `gorm:"type:int;default:0;comment:会话窗口宽度" json:"width"`
	Height       int    `gorm:"type:int;default:0;comment:会话窗口高度" json:"height"`
	Status       string `gorm:"type:varchar(16);index;not null;comment:会话连接状态"  json:"status"`
	Recording    string `gorm:"type:varchar(256);default:'';comment:会话监控录像保存地址" json:"recording"`
	StorageId    string `gorm:"type:varchar(128);default:'';comment:磁盘存储id" json:"storageId"`
	Mode         string `gorm:"type:varchar(16);default:'';comment:会话模式" json:"mode"`
	// 是否已阅
	IsRead           int            `gorm:"type:int(1);default:0;comment:是否已阅" json:"isRead"`
	IsDownloadRecord int            `gorm:"type:int(1);default:0;comment:是否已下载录像" json:"isDownloadRecord"`
	CreateTime       utils.JsonTime `gorm:"type:datetime;not null;comment:会话建立时间" json:"createTime"`
	ConnectedTime    utils.JsonTime `gorm:"type:datetime(3);default:null;comment:建立会话时间" json:"connectedTime"`
	DisconnectedTime utils.JsonTime `gorm:"type:datetime(3);default:null;comment:结束会话时间" json:"disconnectedTime"`
	//下载
	Download int `gorm:"type:int(1);default:0;comment:是否允许下载" json:"download"`
	//上传
	Upload int `gorm:"type:int(1);default:0;comment:是否允许上传" json:"upload"`
	//水印
	Watermark int `gorm:"type:int(1);default:0;comment:是否允许水印" json:"watermark"`
}

func (Session) TableName() string {
	return "session"
}

type AppSession struct {
	ID               string         `gorm:"type:varchar(128);primary_key;not null;comment:会话id" json:"id"`
	ConnectionId     string         `gorm:"type:varchar(128);not null;comment:连接会话id" json:"connectionId"`
	AppId            string         `gorm:"type:varchar(128);index;not null;comment:应用id" json:"appId"`
	AppName          string         `gorm:"type:varchar(128);not null;comment:应用名称" json:"appName"`
	AppIP            string         `gorm:"type:varchar(64);not null;comment:应用ip" json:"appIP"`
	AppPort          int            `gorm:"type:int(5);not null;comment:应用端口" json:"appPort"`
	ProgramId        string         `gorm:"type:varchar(128);not null;comment:程序id" json:"programId"`
	ProgramName      string         `gorm:"type:varchar(128);not null;comment:程序名称" json:"programName"`
	PassPort         string         `gorm:"type:varchar(64);not null;comment:应用账号" json:"passPort"`
	PassPortId       string         `gorm:"type:varchar(128);not null;comment:应用账号id" json:"passPortId"`
	DepartmentId     int64          `gorm:"type:bigint(20);not null;comment:部门id" json:"departmentId"`
	Department       string         `gorm:"type:varchar(128);not null;comment:部门名称" json:"department"`
	Creator          string         `gorm:"type:varchar(128);index;not null;comment:建立会话用户id" json:"creator"`
	CreateName       string         `gorm:"type:varchar(128);not null;comment:建立会话用户名" json:"createName"`
	CreateNick       string         `gorm:"type:varchar(128);not null;comment:建立会话用户昵称" json:"createNick"`
	ClientIP         string         `gorm:"type:varchar(64);default:'';comment:建立会话用户ip" json:"clientIp"`
	Width            int            `gorm:"type:int;default:0;comment:会话窗口宽度" json:"width"`
	Height           int            `gorm:"type:int;default:0;comment:会话窗口高度" json:"height"`
	IsRead           int            `gorm:"type:int(1);default:0;comment:是否已阅" json:"isRead"`
	DownloadStatus   int            `gorm:"type:int(1);default:0;comment:下载状态" json:"downloadStatus"`
	Status           string         `gorm:"type:varchar(16);index;not null;comment:会话连接状态"  json:"status"`
	Recording        string         `gorm:"type:varchar(256);default:'';comment:会话监控录像保存地址" json:"recording"`
	CreateTime       utils.JsonTime `gorm:"type:datetime;not null;comment:会话建立时间" json:"createTime"`
	ConnectedTime    utils.JsonTime `gorm:"type:datetime(3);default:null;comment:建立会话时间" json:"connectedTime"`
	DisconnectedTime utils.JsonTime `gorm:"type:datetime(3);default:null;comment:结束会话时间" json:"disconnectedTime"`
	//下载
	Download int `gorm:"type:int(1);default:0;comment:是否允许下载" json:"download"`
	//上传
	Upload int `gorm:"type:int(1);default:0;comment:是否允许上传" json:"upload"`
	//水印
	Watermark int `gorm:"type:int(1);default:0;comment:是否允许水印" json:"watermark"`
}

func (AppSession) TableName() string {
	return "app_session"
}

type FileRecord struct {
	ID         string         `gorm:"type:varchar(128);primary_key;not null;comment:文件id" json:"id"`
	SessionId  string         `gorm:"type:varchar(128);index;not null;comment:会话id" json:"sessionId"`
	FileName   string         `gorm:"type:varchar(128);not null;comment:文件名称" json:"fileName"`
	FileSize   int64          `gorm:"type:bigint(20);not null;comment:文件大小" json:"fileSize"`
	CreateTime utils.JsonTime `gorm:"type:datetime;not null;comment:文件上传时间" json:"createTime"`
	Action     string         `gorm:"type:varchar(16);not null;comment:文件操作" json:"action"`
	//文件位置
	FilePath string `gorm:"type:varchar(256);not null;comment:文件位置" json:"filePath"`
}

func (FileRecord) TableName() string {
	return "file_record"
}

type ClipboardRecord struct {
	ID        string         `gorm:"type:varchar(128);primary_key;not null;comment:剪切板id" json:"id"`
	SessionId string         `gorm:"type:varchar(128);index;not null;comment:会话id" json:"sessionId"`
	Content   string         `gorm:"type:varchar(256);not null;comment:剪切板内容" json:"content"`
	ClipTime  utils.JsonTime `gorm:"type:datetime;not null;comment:剪切板时间" json:"clipTime"`
}

func (ClipboardRecord) TableName() string {
	return "clipboard_record"
}
