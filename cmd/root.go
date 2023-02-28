package cmd

import (
	"bufio"
	tm "github.com/buger/goterm"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var rootCmd = &cobra.Command{
	Use:                   "got <command>",
	Long:                  `Simple to use Git form command line.`,
	DisableFlagsInUseLine: true,
	SilenceUsage:          true,
	Example: `  # Enter interactive mode to printStatus
  $ got printStatus
  # Enter interactive mode to cat the diff of the file
  $ got status`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var (
	fileStatusList []fileStatus
	workTree       *git.Worktree
	workRepo       *git.Repository
	globalMail     string
	globalName     string
	parentCommit   string
	files          = make(map[string]fileStatus)
)

func init() {
	dir, err := os.Getwd()
	cobra.CheckErr(err)
	// is git repository
	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		_, err := tm.Println(tm.Color("This is not a git repository.", tm.RED))
		cobra.CheckErr(err)
		tm.Flush()
		os.Exit(1)
	}

	workRepo, err = git.PlainOpen(dir)
	cobra.CheckErr(err)

	// getCmd the worktree
	workTree, err = workRepo.Worktree()
	cobra.CheckErr(err)
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(logCmd)
}

// git the status of the current repository
func gitStatus() {
	var err error
	//if len(dir) == 0 {
	//	dir, err = os.Getwd()
	//	cobra.CheckErr(err)
	//}

	status, err := workTree.Status()
	cobra.CheckErr(err)

	fileStatusList = make([]fileStatus, 0)
	if len(status) == 0 {
		_, err := tm.Println(tm.Color("There is no file to printStatus.", tm.RED))
		cobra.CheckErr(err)
		tm.Flush()
		os.Exit(1)
	}
	for file, s := range status {
		var fs fileStatus
		if s.Staging == git.Deleted || s.Worktree == git.Deleted {
			fs = fileStatus{file: file, status: git.Deleted}
		} else if s.Staging == git.Added || s.Worktree == git.Added {
			fs = fileStatus{file: file, status: git.Added}
		} else if s.Staging == git.Modified || s.Worktree == git.Modified {
			fs = fileStatus{file: file, status: git.Modified}
		} else if s.Staging == git.Untracked || s.Worktree == git.Untracked {
			fs = fileStatus{file: file, status: git.Untracked}
		}
		fileStatusList = append(fileStatusList, fs)
	}
}

// 获取git的用户名和邮箱
func getGitConfig() (string, string) {
	var n, e string
	// 查看.git config文件是否存在
	_, err := os.Stat(".git/config")
	if os.IsNotExist(err) {
		goto git
	} else if err != nil {
		cobra.CheckErr(err)
	} else {
		// 读取文件内容
		file, err := os.Open(".git/config")
		cobra.CheckErr(err)
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				cobra.CheckErr(err)
			}
		}(file)

		var name, email string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "name") {
				name = line[strings.Index(line, "=")+1:]
			}
			if strings.Contains(line, "email") {
				email = line[strings.Index(line, "=")+1:]
			}
		}
		cobra.CheckErr(scanner.Err())

		if name != "" && email != "" {
			return name, email
		} else {
			goto git
		}
	}

git:
	var email, name []byte
	// 获取系统的用户名
	name, err = exec.Command("git", "config", "--global", "user.name").Output()
	cobra.CheckErr(err)
	// 获取系统的邮箱
	email, err = exec.Command("git", "config", "--global", "user.email").Output()
	cobra.CheckErr(err)

	if e == "" && n == "" {
		return strings.TrimSpace(string(name)), strings.TrimSpace(string(email))
	} else if e == "" {
		return n, strings.TrimSpace(string(email))
	} else {
		return strings.TrimSpace(string(name)), e
	}
}
