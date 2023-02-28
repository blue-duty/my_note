package repository

import (
	"fmt"
	"gorm.io/gorm"
	"strconv"
	"strings"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/server/dto"
	"tkbastion/server/model"
)

type PropertyRepository struct {
	DB *gorm.DB
}

func NewPropertyRepository(db *gorm.DB) *PropertyRepository {
	propertyRepository = &PropertyRepository{DB: db}
	return propertyRepository
}

func (r PropertyRepository) FindAll() (o []model.Property) {
	if r.DB.Find(&o).Error != nil {
		return nil
	}
	return
}

func (r PropertyRepository) FindAuMap(name string) map[string]string {
	properties := r.FindAu(name)
	propertyMap := make(map[string]string)
	for i := range properties {
		propertyMap[properties[i].Name] = properties[i].Value
	}
	return propertyMap
}

func (r PropertyRepository) FindAu(name string) (o []model.Authentication) {
	if r.DB.Model(model.Property{}).Where("name like ?", name+"%").Find(&o).Error != nil {
		return nil
	}
	return
}

func (r PropertyRepository) Create(o *model.Property) (err error) {
	err = r.DB.Create(o).Error
	return
}

func (r PropertyRepository) CreateByMap(m map[string]interface{}) (err error) {
	db := r.DB.Model(model.Property{}).Begin()
	for k, v := range m {
		err = db.Create(&model.Property{Name: k, Value: v.(string)}).Error
		if err != nil {
			db.Rollback()
			return
		}
	}
	db.Commit()
	return
}

func (r PropertyRepository) UpdateByName(o *model.Property, name string) error {
	o.Name = name
	db := r.DB.Model(model.Property{}).Begin()
	if err := db.Where("name = ?", name).Delete(model.Property{}).Error; err != nil {
		db.Rollback()
	}
	if err := db.Create(o).Error; err != nil {
		db.Rollback()
	}
	db.Commit()
	return nil
}

func (r PropertyRepository) Update(o *model.Property) error {
	return r.DB.Updates(o).Error
}

func (r PropertyRepository) FindByName(name string) (o model.Property, err error) {
	err = r.DB.Where("name = ?", name).Find(&o).Error
	return
}

func (r PropertyRepository) FindAllMap() map[string]string {
	properties := r.FindAll()
	propertyMap := make(map[string]string)
	for i := range properties {
		propertyMap[properties[i].Name] = properties[i].Value
	}
	return propertyMap
}

func (r PropertyRepository) GetDrivePath() (string, error) {
	return global.Config.Guacd.Drive, nil
}

func (r PropertyRepository) GetRecordingPath() (string, error) {
	return global.Config.Guacd.Recording, nil
}
func (r PropertyRepository) Truncate() error {
	return r.DB.Exec("TRUNCATE TABLE properties").Error
}

func (r PropertyRepository) FindMapByNames(name []string) (map[string]string, error) {
	propertyMap := make(map[string]string)
	for i := range name {
		property, err := r.FindByName(name[i])
		if err == gorm.ErrRecordNotFound {
			continue
		} else if err != nil {
			return nil, err
		}
		propertyMap[property.Name] = property.Value
	}
	return propertyMap, nil
}

func (r PropertyRepository) GetSmsProperty() (dto.SmsConfig, error) {
	var SMS = []string{"sms_state", "sms_type", "sms_api_id", "sms_api_secret", "sms_sign_name", "sms_test_phone_number", "sms_template_code"}
	var smsConfig dto.SmsConfig
	item, err := propertyRepository.FindMapByNames(SMS)
	if nil != err {
		log.Error("获取短信配置失败: ", err.Error())
		return smsConfig, err
	}
	smsConfig.SmsState = item["sms_state"]
	smsConfig.SmsType = item["sms_type"]
	smsConfig.SmsApiId = item["sms_api_id"]
	smsConfig.SmsApiSecret = item["sms_api_secret"]
	smsConfig.SmsSignName = item["sms_sign_name"]
	smsConfig.SmsTestPhoneNumber = item["sms_test_phone_number"]
	smsConfig.SmsTemplateCode = item["sms_template_code"]

	// 解密
	if smsConfig.SmsState == "true" {
		if err := smsConfig.Decrypt(); nil != err {
			log.Error("解密失败: ", err.Error())
			return smsConfig, err
		}
	}

	return smsConfig, nil
}

func (r PropertyRepository) DeleteByNames(name []string) error {
	return r.DB.Where("name in ?", name).Delete(model.Property{}).Error
}

func (r PropertyRepository) GetSysUsage(start, end, interval, typeUsage string) (o []model.Usage, err error) {
	// 间隔时间 分钟
	var count int
	var table string
	if interval == "60" {
		count = 12
	} else if interval == "30" {
		count = 6
	} else {
		count = 1
	}
	if typeUsage == "cpu" {
		table = "usage_cpu"
	} else if typeUsage == "mem" {
		table = "usage_mem"
	} else {
		table = "usage_disk"
	}
	sql := "select * from (select @n:=@n+1 as n, a.* from (select * from " + table + " where date_format(datetime,'%Y-%m-%d') between '" + start + "' and '" + end + "' order by datetime asc)a,(select @n:=0)b)c where c.n%" + strconv.Itoa(count) + "=0;"
	//sql := "select datetime,percent,total,used,free from " + table + " where date_format(datetime,'%Y-%m-%d') between ? and ? and id % ? = 0"
	err = r.DB.Raw(sql).Scan(&o).Error
	// 每隔五条取一条数据的SQL语句
	return
}

func (r PropertyRepository) CreatCpuUsage(o *model.UsageCpu) (err error) {
	err = r.DB.Table("usage_cpu").Create(o).Error
	return
}
func (r PropertyRepository) CreatMemUsage(o *model.UsageMem) (err error) {
	err = r.DB.Table("usage_mem").Create(o).Error
	return
}
func (r PropertyRepository) CreatDiskUsage(o *model.UsageDisk) (err error) {
	err = r.DB.Table("usage_disk").Create(o).Error
	return
}

func (r PropertyRepository) GetRemoteManageHost() (o []string, err error) {
	var properties model.Property
	err = r.DB.Where("name = ?", "remote_manage_host").First(&properties).Error
	if err != nil {
		return
	}
	fmt.Println(properties.Value)
	if properties.Value == "" {
		return
	}
	// 回车换行分割
	o = strings.Split(properties.Value, "\n")
	return
}

//// GetClusterConfig 集群配置
//func (r PropertyRepository) GetClusterConfig() (model.HAConfig, error) {
//	var config model.HAConfig
//	err := r.DB.Table("ha_config").Where("id = ?", 1).Find(&config).Error
//	return config, err
//}
//
//// CreateClusterConfig 创建集群配置
//func (r PropertyRepository) CreateClusterConfig(config model.HAConfig) error {
//	return r.DB.Table("ha_config").Create(&config).Error
//}
//
//// ClusterConfigIsExist 集群配置是否存在
//func (r PropertyRepository) ClusterConfigIsExist() bool {
//	var config []model.HAConfig
//	err := r.DB.Table("ha_config").Find(&config).Error
//	if err != nil {
//		return false
//	}
//	if len(config) > 0 {
//		return true
//	}
//	return false
//}
//
//// UpdateClusterConfig 更新集群配置
//func (r PropertyRepository) UpdateClusterConfig(config *model.HAConfig) error {
//	haConfig := utils.Struct2MapByStructTag(config)
//	err := r.DB.Table("ha_config").Where("id = ?", 1).Updates(haConfig).Error
//	return err
//}
