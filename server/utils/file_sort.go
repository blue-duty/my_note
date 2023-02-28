package utils

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

type ByModTime []fs.FileInfo

func (fis ByModTime) Len() int {
	return len(fis)
}

func (fis ByModTime) Swap(i, j int) {
	fis[i], fis[j] = fis[j], fis[i]
}

func (fis ByModTime) Less(i, j int) bool {
	return fis[i].ModTime().Before(fis[j].ModTime())
}

// SortFile 根目录下的文件按时间大小排序，从远到近
func SortFile(path string) (files ByModTime, err error) {
	f, err := os.ReadDir(path)
	if err != nil {
		return
	}
	files = make(ByModTime, len(f))
	for k, file := range f {
		fi, _ := file.Info()
		files[k] = fi
	}
	sort.Sort(files)
	return
}

/*
// 返回当下时间的文件，并删除大于 5 个的文件，删除最早的，如果目录下没有文件，就自动创建
func DealWithFiles(path, name string) (filename string) {
	timestamp := time.Now().Format("20060102.150405")
	filename = path + name + "." + timestamp
	files := SortFile(path, name)
	// fmt.Println(path + files[len(files)-1].Name())
	if len(files) > 5 {
		for k, _ := range files[5:] {
			err := os.Remove(path + files[k].Name())
			if err != nil {
				log.Fatal(err)
			}
		}
	} else if len(files) == 0 {
		f, err := os.Create(filename)
		defer f.Close()
		if err != nil {
			log.Fatal(err)
			return ""
		}
	}
	// fmt.Println(filename)
	return filename
}
*/

func DiskOccupation(c string) (r int, err error) {
	//需要执行命令:command
	command := fmt.Sprintf("df -h %s | grep [0-9]%%", c)
	cmd := exec.Command("/bin/bash", "-c", command)
	// 获取管道输入
	output, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	// 执行Linux命令commandLinux
	if err = cmd.Start(); err != nil {
		return
	}
	b, err := io.ReadAll(output)
	if err != nil {
		return
	}
	if err = cmd.Wait(); err != nil {
		return
	}
	s := string(b)
	i := strings.Index(s, "%")
	str := strings.TrimSpace(s[i-2 : i])
	r, err = strconv.Atoi(str)
	return
}
