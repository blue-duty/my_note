package utils

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/xuri/excelize/v2"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"mime/multipart"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"tkbastion/pkg/config"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

type JsonTime struct {
	time.Time
}

func NewJsonTime(t time.Time) JsonTime {
	return JsonTime{
		Time: t,
	}
}

func NowJsonTime() JsonTime {
	return JsonTime{
		Time: time.Now(),
	}
}

// StringToJSONTime 字符串转换为jsontime
func StringToJSONTime(str string) JsonTime {
	t, _ := time.ParseInLocation("2006-01-02 15:04:05", str, time.Local)
	return JsonTime{Time: t}
}

func (t JsonTime) MarshalJSON() ([]byte, error) {
	var stamp = fmt.Sprintf("\"%s\"", t.Format("2006-01-02 15:04:05"))
	return []byte(stamp), nil
}

func (t JsonTime) Value() (driver.Value, error) {
	var zeroTime time.Time
	if t.Time.UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return t.Time, nil
}

func (t *JsonTime) Scan(v interface{}) error {
	value, ok := v.(time.Time)
	if ok {
		*t = JsonTime{Time: value}
		return nil
	}
	return fmt.Errorf("can not convert %v to timestamp", v)
}

type Bcrypt struct {
	cost int
}

func (b *Bcrypt) Encode(password []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword(password, b.cost)
}

func (b *Bcrypt) Match(hashedPassword, password []byte) error {
	return bcrypt.CompareHashAndPassword(hashedPassword, password)
}

var Encoder = Bcrypt{
	cost: bcrypt.DefaultCost,
}

func UUID() string {
	v4, _ := uuid.NewV4()
	return v4.String()
}

func Tcping(ip string, port int) bool {
	var (
		conn    net.Conn
		err     error
		address string
	)
	strPort := strconv.Itoa(port)
	if strings.HasPrefix(ip, "[") && strings.HasSuffix(ip, "]") {
		// 如果用户有填写中括号就不再拼接
		address = fmt.Sprintf("%s:%s", ip, strPort)
	} else {
		address = fmt.Sprintf("[%s]:%s", ip, strPort)
	}
	//修改超时时间限制为3秒  --zy
	if conn, err = net.DialTimeout("tcp", address, 3*time.Second); err != nil {
		return false
	}
	defer func() {
		_ = conn.Close()
	}()
	return true
}

func ImageToBase64Encode(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// 判断所给路径文件/文件夹是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		return os.IsExist(err)
	}
	return true
}

// 判断所给路径是否为文件夹
func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

// 判断所给路径是否为文件
func IsFile(path string) bool {
	return !IsDir(path)
}

func GetParentDirectory(directory string) string {
	return filepath.Dir(directory)
}

// 去除重复元素
func Distinct(a []string) []string {
	result := make([]string, 0, len(a))
	temp := map[string]struct{}{}
	for _, item := range a {
		if _, ok := temp[item]; !ok {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// 排序+拼接+摘要
func Sign(a []string) string {
	sort.Strings(a)
	data := []byte(strings.Join(a, ""))
	has := md5.Sum(data)
	return fmt.Sprintf("%x", has)
}

func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func StructToMap(obj interface{}) map[string]interface{} {
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)
	if t.Kind() == reflect.Ptr {
		// 如果是指针,则获取其所指向的元素
		t = t.Elem()
		v = v.Elem()
	}

	var data = make(map[string]interface{})
	if t.Kind() == reflect.Struct {
		// 只有结构体可以获取其字段信息
		for i := 0; i < t.NumField(); i++ {
			jsonName := t.Field(i).Tag.Get("json")
			if jsonName != "" {
				data[jsonName] = v.Field(i).Interface()
			} else {
				data[t.Field(i).Name] = v.Field(i).Interface()
			}
		}
	}
	return data
}

func IpToInt(ip string) int64 {
	if len(ip) == 0 {
		return 0
	}
	bits := strings.Split(ip, ".")
	if len(bits) < 4 {
		return 0
	}
	b0 := StringToInt(bits[0])
	b1 := StringToInt(bits[1])
	b2 := StringToInt(bits[2])
	b3 := StringToInt(bits[3])

	var sum int64
	sum += int64(b0) << 24
	sum += int64(b1) << 16
	sum += int64(b2) << 8
	sum += int64(b3)

	return sum
}

func StringToInt(in string) (out int) {
	out, _ = strconv.Atoi(in)
	return
}

func Check(f func() error) {
	if err := f(); err != nil {
		logrus.Error("Received error:", err)
	}
}

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padText...)
}

func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	unPadding := int(origData[length-1])
	return origData[:(length - unPadding)]
}

// AesEncryptCBC /*
func AesEncryptCBC(origData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	origData = PKCS5Padding(origData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	encrypted := make([]byte, len(origData))
	blockMode.CryptBlocks(encrypted, origData)
	return encrypted, nil
}

func AesDecryptCBC(encrypted, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(encrypted))
	blockMode.CryptBlocks(origData, encrypted)
	origData = PKCS5UnPadding(origData)
	return origData, nil
}

func Pbkdf2(password string) ([]byte, error) {
	//生成随机盐
	salt := make([]byte, 32)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}
	//生成密文
	dk := pbkdf2.Key([]byte(password), salt, 1, 32, sha256.New)
	return dk, nil
}

func ExecSql(db *gorm.DB, filePath string) error {
	sql, err := Ioutil(filePath)
	if err != nil {
		fmt.Println("数据库基础数据初始化脚本读取失败！原因:", err.Error())
		return err
	}
	sqlList := strings.Split(sql, ";")
	for i := 0; i < len(sqlList)-1; i++ {
		if strings.Contains(sqlList[i], "--") {
			fmt.Println(sqlList[i])
			continue
		}
		sql := strings.Replace(sqlList[i]+";", "\n", "", 0)
		sql = strings.TrimSpace(sql)
		if err = db.Exec(sql).Error; err != nil {
			if !strings.Contains(err.Error(), "Query was empty") {
				return err
			}
			if config.GlobalCfg.Debug {
				log.Printf("error sql: %s", sql)
			}
		}
	}
	return nil
}

func Ioutil(filePath string) (string, error) {
	if contents, err := ioutil.ReadFile(filePath); err == nil {
		//因为contents是[]byte类型,直接转换成string类型后会多一行空格,需要使用strings.Replace替换换行符
		result := strings.Replace(string(contents), "\n", "", 1)
		if config.GlobalCfg.Debug {
			fmt.Println("Use ioutil.ReadFile to read a file:", result)
		}
		return result, nil
	} else {
		return "", err
	}
}

// GetCpuPercent 获取CPU使用率
func GetCpuPercent() float64 {
	percent, _ := cpu.Percent(time.Second, false)
	return percent[0]
}

// GetMemPercent 获取内存使用率
func GetMemPercent() float64 {
	memInfo, _ := mem.VirtualMemory()
	return memInfo.UsedPercent
}

// GetDiskPercent 获取磁盘使用率
func GetDiskPercent(path string) float64 {
	diskInfo, err := disk.Usage(path)
	if err != nil {
		fmt.Printf("获取磁盘使用率异常，异常信息:%v,要获取的目录:%v\n", err, path)
	}
	return diskInfo.UsedPercent
}

// GetAvailablePort 获取可用端口
func GetAvailablePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}

	defer func(l *net.TCPListener) {
		_ = l.Close()
	}(l)
	return l.Addr().(*net.TCPAddr).Port, nil
}

func RunCommand(client *ssh.Client, command string) (stdout string, err error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var buf bytes.Buffer
	session.Stdout = &buf
	err = session.Run(command)
	if err != nil {
		return "", err
	}
	stdout = buf.String()
	return
}

func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}
func TimeParse(beginTime string) time.Time {
	timeLayout := "2006-01-02 15:04:05"  //转化所需模板
	loc, _ := time.LoadLocation("Local") //获取时区

	time, _ := time.ParseInLocation(timeLayout, beginTime, loc)

	return time
}

// GetRandomNumber 获取随机数
func GetRandomNumber() string {
	rand.Seed(time.Now().UnixNano())
	return strconv.Itoa(rand.Intn(9))
}

// GetRandomUpperChar 获取随机字符
func GetRandomUpperChar() string {
	var str = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	rand.Seed(time.Now().UnixNano())
	return string(str[rand.Intn(len(str))])
}

// GetRandomLowerChar 获取随机字符
func GetRandomLowerChar() string {
	var str = "abcdefghijklmnopqrstuvwxyz"
	rand.Seed(time.Now().UnixNano())
	return string(str[rand.Intn(len(str))])
}

// GetRandomSpecialChar 获取随机特殊字符
func GetRandomSpecialChar() string {
	var str = "!@#~$%^&*()+|_@"
	rand.Seed(time.Now().UnixNano())
	return string(str[rand.Intn(len(str))])
}

// ChangePassword 为linux资产改密方法
func ChangePassword(sshd *ssh.Client, oldPassword, newPassword string, isRoot bool) error {
	session, err := sshd.NewSession()
	if err != nil {
		//fmt.Println("dialect.NewSession error:", err)
		logrus.Error("dialect.NewSession error:", err)
		return err
	}
	defer func(session *ssh.Session) {
		err := session.Close()
		if err != nil && err != io.EOF {
			logrus.Error("session.Close error:", err)
		}
	}(session)
	// 运行命令
	// 读取命令的输出
	stdin, err := session.StdinPipe()
	if err != nil {
		log.Fatal("输入错误", err)
		return err
	}
	//设置session的标准输出和错误输出分别是os.stdout,os,stderr.就是输出到后台
	//stdout, err := session.StdoutPipe()
	//session.Stderr = os.Stderr
	//session.Stdout = os.Stdout
	// 写入命令
	err = session.Start("passwd")
	if err != nil {
		logrus.Error("session.Start error:", err)
		return err
	}
	if isRoot {
		_, err = fmt.Fprintf(stdin, "%s\n", newPassword)
		time.Sleep(time.Second * 1)
		if err != nil {
			logrus.Error("fmt.Fprintf error:", err)
			return err
		}
		_, err = fmt.Fprintf(stdin, "%s\n", newPassword)
		time.Sleep(time.Second * 1)
		if err != nil {
			logrus.Error("fmt.Fprintf error:", err)
			return err
		}
	} else {
		_, err := fmt.Fprintf(stdin, oldPassword+"\n")
		time.Sleep(time.Second * 1)
		if err != nil {
			log.Fatal("fmt.Fprintf error:", err)
			return err
		}
		_, err = fmt.Fprintf(stdin, "%s\n", newPassword)
		time.Sleep(time.Second * 1)
		if err != nil {
			logrus.Error("fmt.Fprintf error:", err)
			return err
		}
		_, err = fmt.Fprintf(stdin, "%s\n", newPassword)
		time.Sleep(time.Second * 1)
		if err != nil {
			logrus.Error("fmt.Fprintf error:", err)
			return err
		}
	}
	return nil
}

// IdHandle id处理
func IdHandle(id string) []string {
	var ids []string
	if strings.Contains(id, ",") {
		ids = strings.Split(id, ",")
	} else if id != "" {
		ids = append(ids, id)
	}
	return ids
}

func SortMapByValue(m map[string]int64) map[string]int64 {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	r := make(map[string]int64)
	for _, k := range keys {
		r[k] = m[k]
	}
	return r
}

// CreateExcelFile 创建excel文件
func CreateExcelFile(name string, header []string, values [][]string) (file *excelize.File, err error) {
	if len(values) != 0 && len(header) != len(values[0]) {
		return nil, errors.New("header length not equal values length")
	}
	file = excelize.NewFile()
	file.SetSheetName("Sheet1", name)
	err = file.SetColWidth(name, string(rune(65)), string(rune(65+len(header)-1)), 20)
	if err != nil {
		return
	}
	// 设置表头样式
	style, err := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color:  "#FFFFFF",
			Bold:   true,
			Family: "Arial",
			Size:   10,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#666666"},
			Pattern: 1,
		},
	})
	if err != nil {
		return
	}
	// 设置表头
	for i, v := range header {
		if err = file.SetCellValue(name, fmt.Sprintf("%s%d", string(rune(65+i)), 1), v); err != nil {
			return
		}
		if err = file.SetCellStyle(name, fmt.Sprintf("%s%d", string(rune(65+i)), 1), fmt.Sprintf("%s%d", string(rune(65+i)), 1), style); err != nil {
			return
		}
	}

	// 设置内容样式
	style1, err := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color:  "#000000",
			Bold:   false,
			Family: "Arial",
			Size:   10,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
	})
	// 循环写入数据
	line := 1
	for _, v := range values {
		line++
		for k, vv := range v {
			if err = file.SetCellValue(name, fmt.Sprintf("%s%d", string(rune(65+k)), line), vv); err != nil {
				return
			}
			if err = file.SetCellStyle(name, fmt.Sprintf("%s%d", string(rune(65+k)), line), fmt.Sprintf("%s%d", string(rune(65+k)), line), style1); err != nil {
				return
			}
		}
	}
	return
}

// CreateTemplateFile 创建模板文件
func CreateTemplateFile(name string, header []string) (file *excelize.File, err error) {
	file = excelize.NewFile()
	file.SetSheetName("Sheet1", name)
	err = file.SetColWidth(name, string(rune(65)), string(rune(65+len(header)-1)), 20)
	if err != nil {
		return
	}
	// 设置表头样式
	style, err := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color:  "#FFFFFF",
			Bold:   true,
			Family: "Arial",
			Size:   10,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#666666"},
			Pattern: 1,
		},
	})
	if err != nil {
		return
	}
	// 设置表头
	for i, v := range header {
		if err = file.SetCellValue(name, fmt.Sprintf("%s%d", string(rune(65+i)), 1), v); err != nil {
			return
		}
		if err = file.SetCellStyle(name, fmt.Sprintf("%s%d", string(rune(65+i)), 1), fmt.Sprintf("%s%d", string(rune(65+i)), 1), style); err != nil {
			return
		}
	}
	return
}

// Struct2StrArr 将结构体数组转为字符串数组
func Struct2StrArr(data interface{}) []string {
	var result []string
	getValue := reflect.ValueOf(data)
	for j := 0; j < getValue.NumField(); j++ {
		switch getValue.Field(j).Kind() {
		case reflect.String:
			result = append(result, getValue.Field(j).String())
		case reflect.Int:
			result = append(result, strconv.Itoa(int(getValue.Field(j).Int())))
		case reflect.Int64:
			result = append(result, strconv.FormatInt(getValue.Field(j).Int(), 10))
		case reflect.Float64:
			result = append(result, strconv.FormatFloat(getValue.Field(j).Float(), 'f', -1, 64))
		case reflect.Bool:
			result = append(result, strconv.FormatBool(getValue.Field(j).Bool()))
		}
	}
	return result
}

func RemoveDuplicatesAndEmpty(data []string) (result []string) {
	result = make([]string, 0)
	strMap := make(map[string]bool) //用于去重
	for i := range data {
		// 去除空格
		data[i] = strings.TrimSpace(data[i])
		// 去除空字符串
		if data[i] == "" {
			continue
		}
		if _, ok := strMap[data[i]]; !ok {
			strMap[data[i]] = true
			result = append(result, data[i])
		}
	}
	return result
}

func DistinctIdInt64(data []int64) (result []int64) {
	result = make([]int64, 0)
	strMap := make(map[int64]bool) //用于去重
	for i := range data {
		if _, ok := strMap[data[i]]; !ok {
			strMap[data[i]] = true
			result = append(result, data[i])
		}
	}
	return result
}

// SortStructByField 根据结构体的某个字符串字段对结构体数组进行排序
func SortStructByField(data interface{}, field string, isDesc bool) {
	getValue := reflect.ValueOf(data)
	if getValue.Kind() != reflect.Slice {
		return
	}
	if getValue.Len() == 0 {
		return
	}
	// 判断是否是字符串类型
	if getValue.Index(0).FieldByName(field).Kind() != reflect.String {
		return
	}
	// 排序
	sort.Slice(data, func(i, j int) bool {
		if isDesc {
			return getValue.Index(i).FieldByName(field).String() > getValue.Index(j).FieldByName(field).String()
		}
		return getValue.Index(i).FieldByName(field).String() < getValue.Index(j).FieldByName(field).String()
	})
}

type CommandPolicyContent struct {
	Rule                string
	Regular             bool
	CommandStrategyId   string
	CommandStrategyName string
	Level               string
	IsEmail             bool
	IsMessage           bool
}

func Matching(c []CommandPolicyContent, rule string) (bool, CommandPolicyContent) {
	for _, v := range c {
		if v.Regular {
			if ok, _ := regexp.MatchString(v.Rule, rule); ok {
				return true, v
			}
		} else {
			if v.Rule == rule {
				return true, v
			}
		}
	}
	return false, CommandPolicyContent{}
}

func Mismatch(c []CommandPolicyContent, rule string) (bool, CommandPolicyContent) {
	for _, v := range c {
		if v.Regular {
			if ok, _ := regexp.MatchString(v.Rule, rule); ok {
				return false, v
			}
		} else {
			if v.Rule == rule {
				return false, v
			}
		}
	}
	return true, c[0]
}

// 读取网络文件，并将键值对存入map中
func ReadNetworkFile(path string) map[string]string {
	fileMap := make(map[string]string)
	fmt.Println("开始读取文件")
	file, err := os.Open(path)
	if err != nil {
		return fileMap
	}
	fmt.Println("文件读取成功")
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			return
		}
	}(file)
	fileReader := bufio.NewReader(file)
	for {
		fileString, readArr := fileReader.ReadString('\n')
		//content := strings.Split(fileString, "=")
		//if len(content) == 2 {
		//	fileMap[content[0]] = content[1]
		//}
		index := strings.Index(fileString, "=")
		if index != -1 {
			key := strings.TrimSpace(fileString[:index])
			value := strings.TrimSpace(fileString[index+1:])
			fileMap[key] = value
		}
		if readArr == io.EOF {
			return fileMap
		}
	}
}

// GetMaskByIp 根据ip/数字 格式 获取掩码
func GetMaskByIp(ip string) (string, string) {
	index := strings.LastIndex(ip, "/")
	if index == -1 {
		return "", ""
	}
	switch ip[index+1:] {
	case "8":
		return ip[:index], "255.0.0.0"
	case "16":
		return ip[:index], "255.255.0.0"
	case "24":
		return ip[:index], "255.255.255.0"
	default:
		return ip[:index], "0.0.0.0"
	}
}

// WriteNetworkFile 读取map中的键值对，并写入文件
func WriteNetworkFile(path string, result map[string]string) error {
	// 排序result
	var keys []string
	for k := range result {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fileResult, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer func(fileResult *os.File) {
		err := fileResult.Close()
		if err != nil {
			return
		}
	}(fileResult)
	for _, k := range keys {
		// 去掉换行符
		v := strings.Replace(result[k], "\n", "", -1)
		// 将k，v写入文件
		_, err := fileResult.WriteString(k + "=" + v + "\n")
		if err != nil {
			return err
		}
	}
	return nil
}

// ExecShell 执行shell命令
func ExecShell(s string) (string, error) {
	cmd := exec.Command("/bin/bash", "-c", s)
	//var out bytes.Buffer
	//cmd.Stdout = &out
	//err := cmd.Run()
	//if err != nil {
	//	return "", err
	//}
	//return out.String(), nil
	output, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	// 执行Linux命令
	if err = cmd.Start(); err != nil {
		return "", err
	}
	// 读取输出
	all, err := io.ReadAll(output)
	if err != nil {
		return "", err
	}
	if err := cmd.Wait(); err != nil {
		return "", err
	}
	return string(all), nil
}

// SaveFile 根据文件路径，文件名，文件流保存文件
func SaveFile(filePath, fileName string, fileBytes *bytes.Reader) error {
	// 判断文件夹是否存在
	if ok := FileExists(filePath); !ok {
		// 创建文件夹
		err := os.MkdirAll(filePath, os.ModePerm)
		if err != nil {
			return err
		}
	}
	// 创建文件
	file, err := os.Create(filePath + "/" + fileName)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			return
		}
	}(file)
	// 写入文件
	_, err = io.Copy(file, fileBytes)
	if err != nil {
		return err
	}
	return nil
}

// StrToBoolPtr 将字符串转成bool值返回指针
func StrToBoolPtr(str string) *bool {
	if "true" == str {
		t := true
		return &t
	} else {
		f := false
		return &f
	}
}

// Str2Float64 将字符串转成float64
func Str2Float64(str string) float64 {
	res, err := strconv.ParseFloat(str, 10)
	if err != nil {
		return 0
	}
	return res
}

// CheckIPInWhiteList 判断ip是否在白名单中，白名单包括ip段和单个ip
func CheckIPInWhiteList(ip string, whiteList []string) bool {

	CheckIPInIPRange := func(ip string, ipRange1, ipRange2 string) bool {
		ipInt := IpToInt(ip)
		ipRange1Int := IpToInt(ipRange1)
		ipRange2Int := IpToInt(ipRange2)
		if ipInt >= ipRange1Int && ipInt <= ipRange2Int {
			return true
		}
		return false
	}

	for _, v := range whiteList {
		if strings.Contains(v, "-") {
			// ip段
			ipRange := strings.Split(v, "-")
			if len(ipRange) != 2 {
				continue
			}
			if CheckIPInIPRange(ip, ipRange[0], ipRange[1]) {
				return true
			}
		} else {
			if ip == v {
				return true
			}
		}
	}
	return false
}

// UploadSaveFiles 保存上传的文件 name 需包含文件后缀
func UploadSaveFiles(fileInfo *multipart.FileHeader, filePath, name string) error {
	// 遍历filepath文件夹下的所有文件
	fileList, err := os.ReadDir(filePath)
	if err != nil {
		return err
	}
	// 判断文件名是否存在
	for _, v := range fileList {
		if v.Name() == name {
			err = os.Remove(filePath + "/" + name)
			if err != nil {
				return err
			}
		}
	}
	src, err := fileInfo.Open()
	if err != nil {
		return err
	}
	// 创建文件
	dst, err := os.Create(path.Join(filePath, name))
	if err != nil {
		return err
	}
	defer func(dst *os.File) {
		err = dst.Close()
		if err != nil {
			return
		}
	}(dst)
	// Copy
	if _, err = io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}

// JudgeFileIsImage 根据文件名判断是否是图片格式
func JudgeFileIsImage(fileName string) bool {
	// 获取名称的后缀
	suffix := path.Ext(fileName)
	if suffix == "" {
		return false
	}
	match := []string{".jpg", ".jpeg", ".png"}
	for _, v := range match {
		if suffix == v {
			return true
		}
	}
	return false
}

// StrJoin 字符串分号;拼接
func StrJoin(oldStr, newStr string) string {
	if oldStr == "" {
		return newStr
	}
	return newStr + ";" + oldStr
}

func parseTagSetting(str string, sep string) map[string]string {
	settings := map[string]string{}
	names := strings.Split(str, sep)

	for i := 0; i < len(names); i++ {
		j := i
		if len(names[j]) > 0 {
			for {
				if names[j][len(names[j])-1] == '\\' {
					i++
					names[j] = names[j][0:len(names[j])-1] + sep + names[i]
					names[i] = ""
				} else {
					break
				}
			}
		}

		values := strings.Split(names[j], ":")
		k := strings.TrimSpace(strings.ToUpper(values[0]))

		if len(values) >= 2 {
			settings[k] = values[1]
		} else if k != "" {
			settings[k] = k
		}
	}

	return settings
}

func Struct2MapByStructTag(obj interface{}) map[string]interface{} {
	if reflect.TypeOf(obj).Kind() != reflect.Struct && reflect.TypeOf(obj).Kind() != reflect.Ptr {
		return nil
	}
	if reflect.TypeOf(obj).Kind() == reflect.Ptr {
		obj = reflect.ValueOf(obj).Elem().Interface()
		if reflect.TypeOf(obj).Kind() != reflect.Struct {
			return nil
		}
	}
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)
	var data = make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		tp := parseTagSetting(t.Field(i).Tag.Get("map"), ";")
		if v.Field(i).CanInterface() {
			if v.Field(i).IsZero() {
				if _, ok := tp["EMPTY"]; ok {
					data[tp["KEY"]] = v.Field(i).Interface()
				} else {
					continue
				}
			} else {
				data[tp["KEY"]] = v.Field(i).Interface()
			}
		}
	}
	return data
}

func String2Int(str string) int {
	i, _ := strconv.Atoi(str)
	return i
}

// RunScriptOnRemoteServer 在远程服务器上运行脚本（脚本在本地）
func RunScriptOnRemoteServer(sshClient *ssh.Client, scriptPath string) (string, error) {
	session, err := sshClient.NewSession()
	if err != nil {
		return "", err
	}
	defer func(session *ssh.Session) {
		err := session.Close()
		if err != nil {
			return
		}
	}(session)
	var b bytes.Buffer
	session.Stdout = &b
	// 读取脚本内容
	script, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", err
	}
	// 执行脚本
	session.Stdin = strings.NewReader(string(script))
	if err := session.Run("/bin/bash"); err != nil {
		return "", err
	}
	return b.String(), nil
}

func FileSize(file string) (int64, error) {
	f, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			return
		}
	}(f)
	fi, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

// GetLocalIPByInterfaceName 根据网卡名称获取本机IP地址
func GetLocalIPByInterfaceName(interfaceName string) (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, i := range interfaces {
		if i.Name == interfaceName {
			byName, err := net.InterfaceByName(i.Name)
			if err != nil {
				return "", err
			}
			addresses, err := byName.Addrs()
			for _, v := range addresses {
				return v.String()[:len(v.String())-3], nil
			}
		}
	}
	return "", errors.New("not found")
}

// WriteFileDirect 检测文件是否存在并写入覆盖写入文件
func WriteFileDirect(path string, content string) (err error) {
	if !FileExists(path) {
		_, err := os.Create(path)
		if err != nil {
			return err
		}
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0666)
	if nil != err {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			return
		}
	}(f)
	_, err = io.WriteString(f, content)
	if err != nil {
		return err
	}
	if strings.HasSuffix(path, ".sh") {
		err = os.Chmod(path, 0777)
		if err != nil {
			return err
		}
	}
	return nil
}

func IsIP(ip string) bool {
	return net.ParseIP(ip) != nil
}
