package utils_test

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"testing"
	"time"
	"tkbastion/pkg/global"

	"golang.org/x/crypto/ssh"

	"baliance.com/gooxml/document"

	"tkbastion/server/utils"

	"github.com/stretchr/testify/assert"
)

func TestRemoveDuplicatesAndEmpty(t *testing.T) {
	s := []string{"1", "2", "1", "3"}
	s1 := []string{"1", "2", "1", "3", " "}
	s = utils.RemoveDuplicatesAndEmpty(s)
	s1 = utils.RemoveDuplicatesAndEmpty(s1)
	assert.Equal(t, []string{"1", "2", "3"}, s)
	assert.Equal(t, []string{"1", "2", "3"}, s1)
}

func TestTcping(t *testing.T) {
	localhost4 := "127.0.0.1"
	localhost6 := "::1"
	conn, err := net.Listen("tcp", ":9999")
	assert.NoError(t, err)
	ip4resfalse := utils.Tcping(localhost4, 22)
	assert.Equal(t, false, ip4resfalse)

	ip4res := utils.Tcping(localhost4, 9999)
	assert.Equal(t, true, ip4res)

	ip6res := utils.Tcping(localhost6, 9999)
	assert.Equal(t, true, ip6res)

	ip4resWithBracket := utils.Tcping("["+localhost4+"]", 9999)
	assert.Equal(t, true, ip4resWithBracket)

	ip6resWithBracket := utils.Tcping("["+localhost6+"]", 9999)
	assert.Equal(t, true, ip6resWithBracket)

	defer func() {
		_ = conn.Close()
	}()
}

func TestBcrypt_Encode(t *testing.T) {
	encryptedCBC, err := utils.AesEncryptCBC([]byte(""), global.Config.EncryptionPassword)
	println(err)
	encryptedCBCStr := base64.StdEncoding.EncodeToString(encryptedCBC)
	println(encryptedCBCStr)
}

func TestAesEncryptCBC(t *testing.T) {
	origData := []byte("Hello tkbastion") // 待加密的数据
	key := []byte("qwertyuiopasdfgh")     // 加密的密钥
	encryptedCBC, err := utils.AesEncryptCBC(origData, key)
	assert.NoError(t, err)
	assert.Equal(t, "s2xvMRPfZjmttpt+x0MzG9dsWcf1X+h9nt7waLvXpNM=", base64.StdEncoding.EncodeToString(encryptedCBC))
}

func TestStruct2MapByStructTag(t *testing.T) {
	type User struct {
		Username string `json:"username" csv:"用户名" map:"key:username;empty"`
		Age      int    `json:"age" csv:"年龄" map:"age"`
		Password string `json:"password" csv:"密码" map:"key:password" `
	}
	user := User{
		Username: "admin",
		Password: "tkbastion",
	}
	userMap := utils.Struct2MapByStructTag(user)
	fmt.Printf("%#v", userMap)
}

func TestAesDecryptCBC(t *testing.T) {
	origData, err := base64.StdEncoding.DecodeString("s2xvMRPfZjmttpt+x0MzG9dsWcf1X+h9nt7waLvXpNM=") // 待解密的数据
	assert.NoError(t, err)
	key := []byte("qwertyuiopasdfgh") // 解密的密钥
	decryptCBC, err := utils.AesDecryptCBC(origData, key)
	assert.NoError(t, err)
	assert.Equal(t, "Hello tkbastion", string(decryptCBC))
}

func TestPbkdf2(t *testing.T) {
	pbkdf2, err := utils.Pbkdf2("1234")
	assert.NoError(t, err)
	println(hex.EncodeToString(pbkdf2))
}

func TestAesEncryptCBCWithAnyKey(t *testing.T) {
	origData := []byte("admin")                                    // 待加密的数据
	key := []byte(fmt.Sprintf("%x", md5.Sum([]byte("tkbastion")))) // 加密的密钥
	encryptedCBC, err := utils.AesEncryptCBC(origData, key)
	assert.NoError(t, err)
	assert.Equal(t, "3qwawlPxghyiLS5hdr/p0g==", base64.StdEncoding.EncodeToString(encryptedCBC))
}

func TestAesDecryptCBCWithAnyKey(t *testing.T) {
	origData, err := base64.StdEncoding.DecodeString("3qwawlPxghyiLS5hdr/p0g==") // 待解密的数据
	assert.NoError(t, err)
	key := []byte(fmt.Sprintf("%x", md5.Sum([]byte("tkbastion")))) // 加密的密钥
	decryptCBC, err := utils.AesDecryptCBC(origData, key)
	assert.NoError(t, err)
	assert.Equal(t, "admin", string(decryptCBC))
}

func TestCreateWord(t *testing.T) {
	hander := []string{"日期", "SSH", "RDP", "VNC", "SFTP", "FTP", "Telnet", "应用发布", "前台", "总计"}
	content := [][]string{
		{"2020-01-01 00:00:00", "1", "1", "1", "1", "1", "1", "1", "1", "8"},
		{"2020-01-01 00:00:00", "1", "1", "1", "1", "1", "1", "1", "1", "8"},
		{"2020-01-01 00:00:00", "1", "1", "1", "1", "1", "1", "1", "1", "8"},
	}
	//for i := 0; i < 15; i++ {
	//	content = append(content, []string{"admin", "tkbastion"})
	//}
	hander2 := []string{"登陆时间", "用户名", "姓名", "来源地址", "协议", "结果", "描述"}
	content2 := [][]string{
		{"2020-01-01 00:00:00", "admin", "tkbastion", "192.168.28.180", "SSH", "成功", "斤斤计较斤斤计较及斤斤计较及经济"},
		{"2020-01-01 00:00:00", "admin", "tkbastion", "192.168.28.120", "SSH", "成功", "斤斤计较斤斤计较及斤斤计较及经济"},
		{"2020-01-01 00:00:00", "admin", "tkbastion", "192.168.28.120", "SSH", "成功", "斤斤计较斤斤计较及斤斤计较及经济斤斤计较斤斤计较及斤斤计较及经济"},
	}
	d := document.New()
	err := utils.CreateWord(d, "图表一", hander, content)
	err = utils.CreateWord(d, "图表二", hander2, content2)
	assert.NoError(t, err)
	// 转为io.Reader
	//var z *zip.Writer
	//d.WriteExtraFiles(z)
	//buf := new(bytes.Buffer)
	//buf.ReadByte()
	assert.NoError(t, err)
}

func TestCreateCsv(t *testing.T) {
	//hander := []string{"日期", "SSH","RDP","VNC","SFTP","FTP","Telnet","应用发布","前台","总计"}
	//content := [][]string{
	//	{"2020-01-01 00:00:00", "1","1","1","1","1","1","1","1","8"},
	//	{"2020-01-01 00:00:00", "1","1","1","1","1","1","1","1","8"},
	//	{"2020-01-01 00:00:00", "1","1","1","1","1","1","1","1","8"},
	//}
	//for i := 0; i < 15; i++ {
	//	content = append(content, []string{"admin", "tkbastion"})
	//}
	//_, err := utils.CreateCsv("test",hander, content)
	//assert.NoError(t, err)
}

func TestCreateHtml(t *testing.T) {
	hander1 := []string{"日期", "SSH", "RDP", "VNC", "SFTP", "FTP", "Telnet", "应用发布", "前台", "总计"}
	content1 := [][]string{
		{"2020-01-01 00:00:00", "1", "1", "1", "1", "1", "1", "1", "1", "8"},
		{"2020-01-01 00:00:00", "1", "1", "1", "1", "1", "1", "1", "1", "8"},
		{"2020-01-01 00:00:00", "1", "1", "1", "1", "1", "1", "1", "1", "8"},
	}

	hander2 := []string{"日期", "SSH", "RDP", "VNC", "SFTP", "FTP", "Telnet", "应用发布", "前台", "总计"}
	content2 := [][]string{
		{"2020-01-01 00:00:00", "1", "1", "1", "1", "1", "1", "1", "1", "8"},
		{"2020-01-01 00:00:00", "1", "1", "1", "1", "1", "1", "1", "1", "8"},
		{"2020-01-01 00:00:00", "1", "1", "1", "1", "1", "1", "1", "1", "8"},
	}
	//for i := 0; i < 15; i++ {
	//	content = append(content, []string{"admin", "tkbastion"})
	//}
	_, err := utils.ExportHtml(hander1, hander2, content1, content2)
	// bytes.Reader转化为[]byte
	//buf := new(bytes.Buffer)
	//// 保存为html文件
	//err = ioutil.WriteFile("test.html", []byte(html), 0644)
	assert.NoError(t, err)
}

func TestString2Time(t *testing.T) {
	t1 := "2020-01-01 00:00:00"
	t2 := "2020-01-01 00:00:00"
	t3 := "2020-02-01 00:00:00"
	t4 := "2020-02-01 00:00:00"
	t5 := "2020-01-02 00:00:00"
	t6 := "2020-01-02 00:00:00"
	t7 := "2020-01-01 01:00:00"
	t8 := "2020-01-01 01:00:00"
	t9 := "2020-01-01 00:01:00"
	t10 := "2020-01-01 00:01:00"
	t11 := "2020-01-01 00:00:01"
	t12 := "2020-01-01 00:00:01"

	t1Time := utils.String2Time(t1)
	t2Time := utils.String2Time(t2)
	t3Time := utils.String2Time(t3)
	t4Time := utils.String2Time(t4)
	t5Time := utils.String2Time(t5)
	t6Time := utils.String2Time(t6)
	t7Time := utils.String2Time(t7)
	t8Time := utils.String2Time(t8)
	t9Time := utils.String2Time(t9)
	t10Time := utils.String2Time(t10)
	t11Time := utils.String2Time(t11)
	t12Time := utils.String2Time(t12)

	fmt.Println(t12Time)
	fmt.Println(t11Time)
	fmt.Println(t10Time)
	fmt.Println(t9Time)
	fmt.Println(t8Time)
	fmt.Println(t7Time)
	fmt.Println(t6Time)
	fmt.Println(t5Time)
	fmt.Println(t4Time)
	fmt.Println(t3Time)
	fmt.Println(t2Time)
	fmt.Println(t1Time)

	assert.Equal(t, t1Time, t2Time)
	assert.Equal(t, t3Time, t4Time)
	assert.Equal(t, t5Time, t6Time)
	assert.Equal(t, t7Time, t8Time)
	assert.Equal(t, t9Time, t10Time)
	assert.Equal(t, t11Time, t12Time)
}

func TestRunScriptOnRemoteServer(t *testing.T) {
	var (
		host     = "39.99.227.163"
		port     = 22
		user     = "root"
		password = "We9621895"
		script   = "./test.sh"
	)

	// 创建一个ssh客户端
	config := &ssh.ClientConfig{
		Timeout:         1 * time.Second,
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	client, err := ssh.Dial("tcp", addr, config)
	fmt.Println("client", err)

	result, err := utils.RunScriptOnRemoteServer(client, script)
	if assert.NoError(t, err) {
		fmt.Println("result:", result)
	} else {
		fmt.Println(err)
	}
}
