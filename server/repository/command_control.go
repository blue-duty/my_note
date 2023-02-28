package repository

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"tkbastion/pkg/constant"
	"tkbastion/pkg/log"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"gorm.io/gorm"
)

type CommandStrategyRepository struct {
	DB *gorm.DB
}

func NewCommandStrategyRepository(db *gorm.DB) *CommandStrategyRepository {
	commandStrategyRepository = &CommandStrategyRepository{DB: db}
	return commandStrategyRepository
}

func (r *CommandStrategyRepository) Create(o *model.CommandStrategy) (err error) {
	if err = r.DB.Create(o).Error; err != nil {
		return err
	}
	return nil
}
func (r *CommandStrategyRepository) FindById(id string) (o model.CommandStrategy, err error) {
	err = r.DB.Where("id = ?", id).Find(&o).Error
	return
}

func (r *CommandStrategyRepository) FindByLimitingConditions(pageIndex, pageSize int, auto, name, department, level, action, status, description string, departmentId []int64) (o []model.CommandStrategy, total int64, err error) {
	// 按照部门id筛选一遍数据,先根据部门id排序，再根据优先级排序
	db := r.DB.Table("command_strategy").Where("department_id in ?", departmentId)
	if len(auto) > 0 {
		db = db.Where("name like ? or department_name like ? or level like ? or action like ? or status like ? or description like ?", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%", "%"+auto+"%")
	} else {
		if len(name) > 0 {
			db = db.Where("name like ?", "%"+name+"%")
		}
		if len(department) > 0 {
			db = db.Where("department_name like ?", "%"+department+"%")
		}
		if len(level) > 0 {
			db = db.Where("level like ?", "%"+level+"%")
		}
		if len(action) > 0 {
			db = db.Where("action like ?", "%"+action+"%")
		}
		if len(status) > 0 {
			db = db.Where("status like ?", "%"+status+"%")
		}
		if len(description) > 0 {
			db = db.Where("description like ?", "%"+description+"%")
		}
	}
	if err = db.Count(&total).Error; err != nil {
		return
	}
	// 部门深度升序，优先级升序
	err = db.Order("department_depth asc, priority asc").Find(&o).Error
	if err != nil {
		return
	}
	return

}

func (r *CommandStrategyRepository) UpdateById(o *model.CommandStrategy, id string) (err error) {
	err = r.DB.Where("id = ?", id).Updates(o).Error
	return
}
func (r *CommandStrategyRepository) DeleteById(id string) (err error) {
	err = r.DB.Where("id = ?", id).Delete(model.CommandStrategy{}).Error
	return
}

func (r *CommandStrategyRepository) Count() (count int64, err error) {
	err = r.DB.Model(&model.CommandStrategy{}).Count(&count).Error
	return
}

func (r *CommandStrategyRepository) CountByDepartmentId(depId int64) (total int64, err error) {
	err = r.DB.Table("command_strategy").Where("department_id = ?", depId).Count(&total).Error
	return
}

func (r *CommandStrategyRepository) FindByDepartmentId(id int64) (commandStrategy []model.CommandStrategy, err error) {
	// 按优先级升序排列
	err = r.DB.Table("command_strategy").Where("department_id = ?", id).Order("priority asc").Find(&commandStrategy).Error
	return
}

func (r *CommandStrategyRepository) FindByDepartmentIds(depIds []int64) (o []model.CommandStrategy, err error) {
	err = r.DB.Find(&o).Where("department_id in ?", depIds).Error
	return
}

func (r *CommandStrategyRepository) CheckName(name string) bool {
	var count int64
	r.DB.Model(&model.CommandStrategy{}).Where("name = ?", name).Count(&count)
	if count > 0 {
		return true
	}
	return false
}

// 每次会话结束后统一执行将执行过的命令写入数据表中

func (r *CommandStrategyRepository) CreateCommandRecord(sessionId string) (err error) {
	var newSession model.Session
	if err = r.DB.Table("session").Where("id = ?", sessionId).First(&newSession).Error; err != nil {
		return err
	}
	var recording string
	if newSession.Protocol == constant.SSH {
		recording = filepath.Dir(newSession.Recording) + "/command.txt"
	} else {
		recording = newSession.Recording + "/recording"
	}

	file, err := os.Open(recording)
	if nil != err {
		log.Error(err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Error(err)
		}
	}(file)
	fileInfo, err := file.Stat()
	if err != nil {
		log.Error(err)
	}
	var items []model.RecordCommandAnalysis
	br := bufio.NewReader(file)
	for {
		var item model.RecordCommandAnalysis
		lineTime, _, err := br.ReadLine()
		if io.EOF == err {
			break
		}
		line := strings.Split(string(lineTime), ":")
		var timeStr string
		if len(line) == 3 {
			timeStr = line[0] + "h" + line[1] + "m" + line[2] + "s"
		} else {
			continue
		}
		duration, _ := time.ParseDuration(timeStr)
		item.Time = fileInfo.ModTime().Add(duration).Format("2006-01-02 15:04:05")
		lineCommand, _, _ := br.ReadLine()
		item.Command = string(lineCommand)
		items = append(items, item)
	}
	for i := range items {
		var commandRecord = model.CommandRecord{
			ID:        utils.UUID(),
			AssetId:   newSession.PassportId,
			SessionId: sessionId,
			AssetName: newSession.AssetName,
			AssetIp:   newSession.AssetIP,
			ClientIp:  newSession.ClientIP,
			Passport:  newSession.PassPort,
			Username:  newSession.CreateName,
			Nickname:  newSession.CreateNick,
			Protocol:  newSession.Protocol,
			Created:   utils.StringToJSONTime(items[i].Time),
			Content:   items[i].Command,
		}
		err = r.DB.Table("command_records").Create(commandRecord).Error
		if err != nil {
			log.Errorf("Create command record failed: %s", err)
		}
	}
	return
}
