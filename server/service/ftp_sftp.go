package service

import (
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"time"
	"tkbastion/pkg/constant"
	"tkbastion/pkg/log"
	"tkbastion/server/utils"

	"github.com/jlaffaye/ftp"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SftpClientConfig sftp连接
type SftpClientConfig struct {
	Host       string       //ip
	Port       int64        // 端口
	Username   string       //用户名
	Password   string       //密码
	SavePath   string       //保存路径
	sshClient  *ssh.Client  //ssh client
	sftpClient *sftp.Client //sftp client
	LastResult string       //最近一次运行的结果
	IsPress    bool         //是否压缩
}

func (s *SftpClientConfig) connect() error {
	var err error
	s.sshClient, err = ssh.Dial("tcp", s.Host+":"+strconv.Itoa(int(s.Port)), &ssh.ClientConfig{
		User: s.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	})
	if err != nil {
		return err
	}
	s.sftpClient, err = sftp.NewClient(s.sshClient)
	if err != nil {
		return err
	}
	return nil
}

func (s *SftpClientConfig) close() error {
	if s.sftpClient != nil {
		err := s.sftpClient.Close()
		if err != nil {
			return err
		}
	}
	if s.sshClient != nil {
		err := s.sshClient.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SftpClientConfig) upload(name string) error {
	var err error
	if s.IsPress {
		command := "zip -rj " + path.Join(constant.BackupPath, name+".zip") + " " + path.Join(constant.BackupPath, name)
		//err = ExecutiveCommand("tar -Pczf " + "tkbastion_审计日志_" + now + ".tar -C " + backupPath + " tkbastion_审计日志_" + now)
		_, err := utils.ExecShell(command)
		if err != nil {
			log.Errorf("zip Error: %v", err)
		}

		defer func() {
			err := os.Remove(path.Join(constant.BackupPath, name+".zip"))
			if err != nil {
				log.Errorf("Remove Error: %v", err)
			}
		}()

		//移动文件至备份文件夹
		_, _ = utils.ExecShell("mv " + name + ".zip " + constant.BackupPath)
		localFile := path.Join(constant.BackupPath, name+".zip")
		file, err := os.Open(localFile)
		if err != nil {
			return err
		}
		defer func(localFile *os.File) {
			err := localFile.Close()
			if err != nil {
				log.Errorf("Close Error: %v", err)
			}
		}(file)
		if _, err := s.sftpClient.ReadDir(s.SavePath); err != nil {
			err = s.sftpClient.MkdirAll(s.SavePath)
			if err != nil {
				log.Errorf("Mkdir Error: %v", err)
				return err
			}
		}
		remoteFile, err := s.sftpClient.Create(path.Join(s.SavePath, name+".zip"))
		if err != nil {
			log.Errorf("Create Error: %v", err)
			return err
		}
		defer func(remoteFile *sftp.File) {
			err := remoteFile.Close()
			if err != nil {
				log.Errorf("Close Error: %v", err)
			}
		}(remoteFile)
		_, err = io.Copy(remoteFile, file)
		if err != nil {
			return err
		}
	} else {
		err = s.sftpClient.MkdirAll(s.SavePath + "/" + name)
		if err != nil {
			log.Errorf("Mkdir Error: %v", err)
			return err
		}
		localFile := path.Join(constant.BackupPath, name)
		file, err := os.ReadDir(localFile)
		if err != nil {
			return err
		}
		for _, v := range file {
			if v.IsDir() {
				continue
			}
			localFile, err := os.Open(path.Join(constant.BackupPath, name, v.Name()))
			if err != nil {
				return err
			}
			remoteFile, err := s.sftpClient.Create(path.Join(s.SavePath, name, v.Name()))
			if err != nil {
				_ = localFile.Close()
				log.Errorf("Create Error: %v", err)
				return err
			}
			_, err = io.Copy(remoteFile, localFile)
			if err != nil {
				_ = localFile.Close()
				_ = remoteFile.Close()
				return err
			}
			err = localFile.Close()
			if err != nil {
				log.Errorf("Close Error: %v", err)
			}
			err = remoteFile.Close()
			if err != nil {
				log.Errorf("Close Error: %v", err)
			}
		}
	}
	return nil
}

// FtpClientConfig ftp连接
type FtpClientConfig struct {
	Host       string //ip
	Port       int64  // 端口
	Username   string //用户名
	Password   string //密码
	SavePath   string //保存路径
	IsPress    bool   //是否压缩
	LastResult string //最近一次运行的结果
}

func (f *FtpClientConfig) uploadFtp(name string) error {
	c, err := ftp.Dial(f.Host+":"+strconv.Itoa(int(f.Port)), ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		log.Fatal(err)
	}
	defer func(c *ftp.ServerConn) {
		err := c.Quit()
		if err != nil {
			log.Fatal(err)
		}
	}(c)

	err = c.Login(f.Username, f.Password)
	if err != nil {
		log.Fatal(err)
	}
	//dir, _ := c.CurrentDir()
	//fmt.Print("current dir ", dir)

	if f.IsPress {
		// 压缩本地文件
		command := "zip -rj " + path.Join(constant.BackupPath, name+".zip") + " " + path.Join(constant.BackupPath, name)
		//err = ExecutiveCommand("tar -Pczf " + "tkbastion_审计日志_" + now + ".tar -C " + backupPath + " tkbastion_审计日志_" + now)
		_, err := utils.ExecShell(command)
		fmt.Println(err)
		if err != nil {
			log.Errorf("zip Error: %v", err)
		}

		defer func() {
			err := os.Remove(path.Join(constant.BackupPath, name+".zip"))
			if err != nil {
				log.Errorf("Remove Error: %v", err)
			}
		}()

		//移动文件至备份文件夹
		_, _ = utils.ExecShell("mv " + name + ".zip " + constant.BackupPath)
		localFile := path.Join(constant.BackupPath, name+".zip")

		_ = c.MakeDir(f.SavePath)
		_ = c.ChangeDir(f.SavePath)
		file, _ := os.Open(localFile)
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				log.Errorf("Close Error: %v", err)
			}
		}(file)
		err = c.Stor(name+".zip", file)
		if err != nil {
			log.Errorf("Stor Error: %v", err)
			return err
		}
	} else {
		_ = c.MakeDir(path.Join(f.SavePath, name))
		err = c.ChangeDir(path.Join(f.SavePath, name))
		if err != nil {
			log.Errorf("ChangeDir Error: %v", err)
			return err
		}
		// 遍历文件夹
		files, _ := os.ReadDir(path.Join(constant.BackupPath, name))
		for _, f := range files {
			localFile := path.Join(constant.BackupPath, name, f.Name())
			file, err := os.Open(localFile)
			if err != nil {
				log.Errorf("Open File Error: %v", err)
			}
			err = c.Stor(f.Name(), file)
			if err != nil {
				_ = file.Close()
				log.Errorf("Stor Error: %v", err)
			}
			err = file.Close()
			if err != nil {
				log.Errorf("Close Error: %v", err)
			}
		}
	}

	////fmt.Println("Connected to remote host")
	//// Do something with the FTP conn
	//file, _ := os.Open(loaclfile)
	//defer func(file *os.File) {
	//	err := file.Close()
	//	if err != nil {
	//		log.Errorf("Close Error: %v", err)
	//	}
	//}(file)
	//err = c.Stor(name, file)
	//if err != nil {
	//	log.Errorf("Stor Error: %v", err)
	//	return err
	//}
	return nil
}

//func (f *FtpClientConfig) connect() error {
//	var err error
//	f.ftpClient, err = goftp.DialConfig(goftp.Config{
//		User:               f.Username,
//		Password:           f.Password,
//		ConnectionsPerHost: 10,
//		Timeout:            time.Second * 30,
//	}, f.Host+":"+strconv.Itoa(int(f.Port)))
//	if err != nil {
//		return err
//	}
//	return nil
//}

//func (f *FtpClientConfig) close() error {
//	if f.ftpClient != nil {
//		err := f.ftpClient.Close()
//		if err != nil {
//			return err
//		}
//	}
//	//if f.sshClient != nil {
//	//	err := f.sshClient.Close()
//	//	if err != nil {
//	//		return err
//	//	}
//	//}
//	return nil
//}

//func (f *FtpClientConfig) upload(localPath string) error {
//	var err error
//	localFile, err := os.Open(localPath)
//	if err != nil {
//		return err
//	}
//	defer func(localFile *os.File) {
//		err := localFile.Close()
//		if err != nil {
//			log.Errorf("Close Error: %v", err)
//		}
//	}(localFile)
//	if _, err := f.ftpClient.ReadDir(f.SavePath); err != nil {
//		fmt.Println(err)
//		_, err = f.ftpClient.Mkdir(f.SavePath)
//		if err != nil {
//			log.Errorf("Mkdir Error: %v", err)
//			return err
//		}
//	}
//	defer func(ftpClient *goftp.Client) {
//		err := ftpClient.Close()
//		if err != nil {
//			log.Errorf("Close Error: %v", err)
//		}
//	}(f.ftpClient)
//	dstName := filepath.Base(localFile.Name())
//	err = f.ftpClient.Store(path.Join(f.SavePath, dstName), localFile)
//	if err != nil {
//		return err
//	}
//	return nil
//}

//func (f FtpClientConfig) FtpUpload(ns, localFile string) (string, error) {
//	// 1. 先判断local file是否存在
//	file, err := os.Open(localFile)
//	// Could not create file
//	file, err = os.OpenFile(localFile, syscall.O_CREAT|syscall.O_WRONLY|syscall.O_TRUNC, 0744)
//	if err != nil {
//		return "", err
//	}
//	//defer func(file *os.File) {
//	//	err := file.Close()
//	//	if err != nil {
//	//		log.Errorf("Close Error: %v", err)
//	//	}
//	//}(file)
//	// 2.得到pwd
//	pwd, pwdErr := f.ftpClient.Getwd()
//	//fmt.Println(pwd, pwdErr)
//	if pwdErr != nil {
//		return "", err
//	}
//	//// 2. 创建savePath
//	savePath := path.Join(pwd, ns)
//	_, err = f.ftpClient.Mkdir(savePath)
//	if err != nil {
//		// 由于搭建ftp的时候已经给了`pwd` 777的权限，这里忽略文件夹创建的错误
//		if !strings.Contains(err.Error(), "550-Create directory operation failed") {
//			return "", err
//		}
//	}
//	// 上传完毕后关闭当前的ftp连接
//	defer func(ftpClient *goftp.Client) {
//		err := ftpClient.Close()
//		if err != nil {
//			log.Errorf("Close Error: %v", err)
//		}
//	}(f.ftpClient)
//	dstName := filepath.Base(file.Name())
//	dstPath := path.Join(savePath, dstName)
//	fmt.Println(dstPath)
//	// 文件上传
//	return dstPath, f.ftpClient.Store(dstPath, file)
//}
