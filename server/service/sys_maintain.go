package service

import (
	"encoding/base64"
	"encoding/binary"
	"io/ioutil"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"tkbastion/pkg/constant"
	"tkbastion/server/model"
	"tkbastion/server/repository"
	"tkbastion/server/utils"

	"github.com/labstack/gommon/log"
)

type SysMaintainService struct {
	propertyRepository *repository.PropertyRepository
}

func NewSysMaintainService(propertyRepository *repository.PropertyRepository) *SysMaintainService {
	return &SysMaintainService{propertyRepository: propertyRepository}
}

type NtpPacket struct {
	Settings       uint8
	Stratum        uint8
	Poll           int8
	Precision      int8
	RootDelay      uint32
	RootDispersion uint32
	ReferenceID    uint32
	RefTimeSec     uint32
	RefTimeFrac    uint32
	OrigTimeSec    uint32
	OrigTimeFrac   uint32
	RxTimeSec      uint32
	RxTimeFrac     uint32
	TxTimeSec      uint32
	TxTimeFrac     uint32
}

func (r SysMaintainService) GetProductId() (productId string, err error) {
	cmd := exec.Command("/bin/bash", "-c", "nmcli device show | grep GENERAL.HWADDR | awk '{print $2}'")
	output, err := cmd.StdoutPipe()
	if nil != err {
		log.Errorf("StdoutPipe Error: %v", err.Error())
		return "", err
	}

	if err = cmd.Start(); nil != err {
		log.Errorf("Start Error: %v", err.Error())
		return "", err
	}

	all, err := ioutil.ReadAll(output)
	if nil != err {
		log.Errorf("ReadAll Error: %v", err.Error())
		return "", err
	}
	if err = cmd.Wait(); nil != err {
		log.Errorf("Wait Error: %v", err.Error())
		return "", err
	}

	productIdStrArr := strings.Split(string(all), "\n")
	var productIdStr string
	for i := range productIdStrArr {
		productIdStr += productIdStrArr[i]
	}

	encryptedCBC, err := utils.AesEncryptCBC([]byte(productIdStr), []byte("trunkeyTkbastion"))
	if err != nil {
		log.Errorf("AesEncryptCBC Error: %v", err.Error())
		return "", err
	}
	productId = base64.StdEncoding.EncodeToString(encryptedCBC)

	return productId, nil
}

// 许可文件解密
// 输入许可文件内容
// 返回解密后的各信息数组
func (r SysMaintainService) DecryptLicenseContent(encryptionLicense string) (infoArr []string, err error) {
	origData, err := base64.StdEncoding.DecodeString(encryptionLicense)
	if nil != err {
		log.Errorf("DecodeString Error: %v", err.Error())
		return nil, err
	}
	decryptedCBC, err := utils.AesDecryptCBC(origData, []byte("trunkeyTkbastion"))
	if nil != err {
		log.Errorf("AesDecryptCBC Error: %v", err.Error())
		return nil, err
	}
	info := string(decryptedCBC)
	infoArr = strings.Split(info, ",")
	return infoArr, nil
}

func (r SysMaintainService) UpdateProductLicenseInfo(licenseInfoArr []string, digitalSignature string) error {
	err := r.propertyRepository.Update(&model.Property{Name: "tkbastion-value1", Value: licenseInfoArr[0]})
	if nil != err {
		log.Errorf("DB Error: %v", err.Error())
		return err
	}

	err = r.propertyRepository.Update(&model.Property{Name: "tkbastion-value2", Value: licenseInfoArr[1]})
	if nil != err {
		log.Errorf("DB Error: %v", err.Error())
		return err
	}

	err = r.propertyRepository.Update(&model.Property{Name: "tkbastion-value3", Value: licenseInfoArr[2]})
	if nil != err {
		log.Errorf("DB Error: %v", err.Error())
		return err
	}

	err = r.propertyRepository.Update(&model.Property{Name: "tkbastion-value4", Value: licenseInfoArr[3]})
	if nil != err {
		log.Errorf("DB Error: %v", err.Error())
		return err
	}

	err = r.propertyRepository.Update(&model.Property{Name: "tkbastion-value5", Value: licenseInfoArr[4]})
	if nil != err {
		log.Errorf("DB Error: %v", err.Error())
		return err
	}

	err = r.propertyRepository.Update(&model.Property{Name: "tkbastion-value6", Value: digitalSignature})
	if nil != err {
		log.Errorf("DB Error: %v", err.Error())
		return err
	}

	err = r.CheckLicenseIsOvertimeJob()
	if nil != err {
		log.Errorf("CheckLicenseIsOvertimeJob Error: %v", err.Error())
		log.Error("导入授权许可文件后, 根据当前日期更新授权状态与数字签名失败")
		return err
	}

	return nil
}

func (r SysMaintainService) CheckLicenseIsOvertimeJob() error {
	item := r.propertyRepository.FindAuMap("tkbastion")
	origData, err := base64.StdEncoding.DecodeString(item["tkbastion-value5"])
	if nil != err {
		log.Errorf("DecodeString Error: %v", err.Error())
		return err
	}
	decryptedCBC, err := utils.AesDecryptCBC(origData, []byte("trunkeyTkbastion"))
	if err != nil {
		log.Errorf("AesDecryptCBC Error: %v", err.Error())
		return err
	}

	authType := "false"
	overtime := string(decryptedCBC)
	overtimeTime := utils.String2Time(overtime)
	if overtimeTime.After(utils.NowJsonTime().Time) {
		authType = "true"
	}

	encryptedCBC, err := utils.AesEncryptCBC([]byte(authType), []byte("trunkeyTkbastion"))
	if err != nil {
		log.Errorf("AesEncryptCBC Error: %v", err.Error())
		return err
	}
	authType = base64.StdEncoding.EncodeToString(encryptedCBC)
	property := model.Property{
		Name:  "tkbastion-value2",
		Value: authType,
	}
	if err := r.propertyRepository.Update(&property); nil != err {
		log.Errorf("DB Error: %v", err.Error())
		return err
	}

	encryptedCBC, err = utils.AesEncryptCBC([]byte(item["tkbastion-value1"]+","+authType+","+item["tkbastion-value3"]+","+item["tkbastion-value4"]+","+item["tkbastion-value5"]), []byte("trunkeyTkbastion"))
	if err != nil {
		log.Errorf("AesEncryptCBC Error: %v", err.Error())
		return err
	}
	digitalSignature := base64.StdEncoding.EncodeToString(encryptedCBC)
	property = model.Property{
		Name:  "tkbastion-value6",
		Value: digitalSignature,
	}
	if err := r.propertyRepository.Update(&property); nil != err {
		log.Errorf("DB Error: %v", err.Error())
		return err
	}

	return nil
}

func (r SysMaintainService) IsOverAuthResourceCountLimit(nowResourceCount int) (err error, isOver bool) {
	item := r.propertyRepository.FindAuMap("tkbastion")
	origData, err := base64.StdEncoding.DecodeString(item["tkbastion-value4"])
	if nil != err {
		log.Errorf("DecodeString Error: %v", err.Error())
		return err, true
	}
	decryptedCBC, err := utils.AesDecryptCBC(origData, []byte("trunkeyTkbastion"))
	if nil != err {
		log.Errorf("AesDecryptCBC Error: %v", err.Error())
		return err, true
	}
	authResourceCount, err := strconv.Atoi(string(decryptedCBC))
	if nil != err {
		log.Errorf("Atoi Error: %v", err.Error())
		return err, true
	}

	if nowResourceCount <= authResourceCount {
		return nil, false
	}
	return nil, true
}

func (r SysMaintainService) IsAllowOperate() (err error, isAllow bool) {
	item := r.propertyRepository.FindAuMap("tkbastion")
	origData, err := base64.StdEncoding.DecodeString(item["tkbastion-value2"])
	if nil != err {
		log.Errorf("DecodeString Error: %v", err.Error())
		return err, true
	}
	decryptedCBC, err := utils.AesDecryptCBC(origData, []byte("trunkeyTkbastion"))
	if nil != err {
		log.Errorf("AesDecryptCBC Error: %v", err.Error())
		return err, true
	}
	authType := string(decryptedCBC)
	if "true" == authType {
		return nil, true
	}

	return nil, false
}

func GetNtpTime() (err error, ntpTime time.Time) {
	conn, err := net.Dial("udp", "ntp.aliyun.com:123")
	if nil != err {
		log.Errorf("Dial Error: %v", err.Error())
		return err, time.Time{}
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
		log.Errorf("SetDeadline Error: %v", err.Error())
		return err, time.Time{}
	}
	req := &NtpPacket{Settings: 0x1B}

	if err := binary.Write(conn, binary.BigEndian, req); err != nil {
		log.Errorf("Write Error: %v", err.Error())
		return err, time.Time{}
	}

	rsp := &NtpPacket{}
	if err := binary.Read(conn, binary.BigEndian, rsp); err != nil {
		log.Errorf("Read Error: %v", err.Error())
		return err, time.Time{}
	}

	secs := float64(rsp.TxTimeSec) - constant.NTPEPOCHOFFSET
	nanos := (int64(rsp.TxTimeFrac) * 1e9) >> 32

	ntpTime = time.Unix(int64(secs), nanos)

	return nil, ntpTime
}
