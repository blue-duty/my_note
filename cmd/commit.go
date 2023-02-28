package cmd

import (
	"github.com/AlecAivazis/survey/v2"
	tm "github.com/buger/goterm"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
	"got/common"
	"strconv"
	"strings"
	"time"
)

type fileStatus struct {
	file   string
	status git.StatusCode
}

type commitOptions struct {
	except  string
	all     bool
	message string
	push    bool
}

var commitOpts = &commitOptions{}

func init() {
	globalName, globalMail = getGitConfig()
	commitCmd.Flags().StringVarP(&commitOpts.except, "except", "e", "", "except files")
	commitCmd.Flags().BoolVarP(&commitOpts.all, "all", "a", false, "printStatus all files")
	commitCmd.Flags().StringVarP(&commitOpts.message, "message", "m", "", "printStatus message")
	commitCmd.Flags().BoolVarP(&commitOpts.push, "push", "p", false, "push to remote repository")
}

// Print fileStatus print
func (fs fileStatus) Print(i int) {
	switch fs.status {
	case git.Untracked:
		// print serial number and file name and format alignment
		_, err := tm.Printf("%d. %s\n", i, tm.Color(fs.file, tm.WHITE))
		cobra.CheckErr(err)
		tm.Flush()
	case git.Added:
		_, err := tm.Printf("%d. %s\n", i, tm.Color(fs.file, tm.GREEN))
		cobra.CheckErr(err)
		tm.Flush()
	case git.Modified:
		_, err := tm.Printf("%d. %s\n", i, tm.Color(fs.file, tm.YELLOW))
		cobra.CheckErr(err)
		tm.Flush()
	case git.Deleted:
		_, err := tm.Printf("%d. %s\n", i, tm.Color(fs.file, tm.RED))
		cobra.CheckErr(err)
		tm.Flush()
	default:
		_, err := tm.Printf("%d. %s\n", i, tm.Color(fs.file, tm.BLACK))
		cobra.CheckErr(err)
		tm.Flush()
	}
}

func (fs fileStatus) String() string {
	switch fs.status {
	case git.Untracked:
		return tm.Color(fs.file, tm.WHITE)
	case git.Added:
		return tm.Color(fs.file, tm.GREEN)
	case git.Modified:
		return tm.Color(fs.file, tm.YELLOW)
	case git.Deleted:
		return tm.Color(fs.file, tm.RED)
	default:
		return tm.Color(fs.file, tm.WHITE)
	}
}

func printStatus(fileStatusList []fileStatus) {
	for i, fs := range fileStatusList {
		fs.Print(i + 1)
	}
	tm.Flush()
}

var commitCmd = &cobra.Command{
	// dir非必须参数
	Use:   "printStatus [flags]",
	Short: "Commit files to local repository",
	Long: `This command is used to printStatus files to the local repository by interactive mode.
It can list all the files which can be committed at the current directory, and you can input the file's serial number to printStatus, they are separated by ','.
And their status is distinguished by color. Green means added, Yellow means modified, Red means deleted, White means untracked.
If you want to printStatus all the files, you can use the -a flag.
If you want to printStatus all the files except some files, you can use the -e flag, and they are separated by ','.
If you want to push after printStatus, you can use the -p flag.
If you user the -a flag or -e flag, you can also use the -m flag to appoint the printStatus message, if it not exist, you should input the message in the next step.
If you use the -a flag and -e flag at the same time, it will be invalid.`,
	Run: func(cmd *cobra.Command, args []string) {
		//if len(args) > 0 {
		//	// 判断是一个目录且存在
		//	if ff, err := os.Stat(args[0]); err != nil || !ff.IsDir() {
		//		_, err := tm.Println(tm.Color("The directory arg is not a directory or not exist.", tm.RED))
		//		cobra.CheckErr(err)
		//		tm.Flush()
		//		return
		//	}
		//	gitStatus(args[0])
		//}
		if commitOpts.except != "" && commitOpts.all {
			_, err := tm.Println(tm.Color("The except and all flags can't be used at the same time.", tm.RED))
			cobra.CheckErr(err)
			tm.Flush()
			return
		}

		if commitOpts.all {
			err := workTree.AddWithOptions(&git.AddOptions{
				All: true,
			})
			cobra.CheckErr(err)
			if commitOpts.message == "" {
				var message string
				prompt := &survey.Input{
					Message: "Please input your printStatus message:",
				}
				err := survey.AskOne(prompt, &message)
				cobra.CheckErr(err)
				_, err = workTree.Commit(message, &git.CommitOptions{
					Author: &object.Signature{
						Name:  globalName,
						Email: globalMail,
						When:  time.Now(),
					},
				})
				cobra.CheckErr(err)
			} else {
				_, err := workTree.Commit(commitOpts.message, &git.CommitOptions{
					Author: &object.Signature{
						Name:  globalName,
						Email: globalMail,
						When:  time.Now(),
					},
				})
				cobra.CheckErr(err)
			}
			_, err = tm.Println(tm.Color("Commit successfully.", tm.GREEN))
			cobra.CheckErr(err)
			tm.Flush()
			if commitOpts.push {
				err := common.Push()
				if err != nil {
					_, err := tm.Println(tm.Color("Push failed.", tm.RED))
					cobra.CheckErr(err)
					tm.Flush()
					return
				}
				_, err = tm.Println(tm.Color("Push successfully.", tm.GREEN))
				cobra.CheckErr(err)
				tm.Flush()
			}
			return
		}

		if commitOpts.except != "" {
			eMap := make(map[string]struct{})
			for _, fs := range fileStatusList {
				eMap[fs.file] = struct{}{}
			}

			exceptFiles := strings.Split(commitOpts.except, ",")

			// 1. 先将所有的文件都add
			err := workTree.AddWithOptions(&git.AddOptions{
				All: true,
			})
			cobra.CheckErr(err)

			// 2. 再将不需要的文件都remove
			for _, e := range exceptFiles {
				if _, ok := eMap[e]; ok {
					_, err := workTree.Remove(e)
					cobra.CheckErr(err)
				}
			}

			//for _, fs := range fileStatusList {
			//	// get the file name
			//	fn := fs.file[strings.LastIndex(fs.file, "/")+1:]
			//	if _, ok := eMap[fn]; !ok {
			//		_, err := workTree.Add(fs.file)
			//		cobra.CheckErr(err)
			//	}
			//}
			if commitOpts.message == "" {
				var message string
				prompt := &survey.Input{
					Message: "Please input your printStatus message:",
				}
				err := survey.AskOne(prompt, &message)
				cobra.CheckErr(err)
				_, err = workTree.Commit(message, &git.CommitOptions{
					Author: &object.Signature{
						Name:  globalName,
						Email: globalMail,
						When:  time.Now(),
					},
				})
				cobra.CheckErr(err)
			} else {
				_, err := workTree.Commit(commitOpts.message, &git.CommitOptions{
					Author: &object.Signature{
						Name:  globalName,
						Email: globalMail,
						When:  time.Now(),
					},
				})
				cobra.CheckErr(err)
			}
			_, err = tm.Println(tm.Color("Commit successfully.", tm.GREEN))
			cobra.CheckErr(err)
			tm.Flush()
			if commitOpts.push {
				err := common.Push()
				if err != nil {
					_, err := tm.Println(tm.Color("Push failed.", tm.RED))
					cobra.CheckErr(err)
					tm.Flush()
					return
				}
				_, err = tm.Println(tm.Color("Push successfully.", tm.GREEN))
				cobra.CheckErr(err)
				tm.Flush()
			}
			return
		}

		gitStatus()
		_, err := tm.Println(`The following is the file which can be committed: `)
		cobra.CheckErr(err)
		tm.Flush()
		printStatus(fileStatusList)
		var fileIndex string
		prompt := &survey.Input{
			Message: "Please input the serial number of the file (You can use ',' to separate you want to printStatus or use ';' to separate you don't want to printStatus):",
		}
		err = survey.AskOne(prompt, &fileIndex)
		cobra.CheckErr(err)
		// split by ,
		if len(fileIndex) == 0 {
			_, err := tm.Println(tm.Color("You don't input the serial number of the file you want to printStatus.", tm.RED))
			cobra.CheckErr(err)
			tm.Flush()
			return
		}
		if strings.Contains(fileIndex, ";") {
			var fsl = fileStatusList
			for _, index := range strings.Split(fileIndex, ";") {
				if len(strings.TrimSpace(index)) == 0 {
					_, err := tm.Println(tm.Color("You don't input the serial number of the file you want to printStatus.", tm.RED))
					cobra.CheckErr(err)
					tm.Flush()
					return
				}
				i, err := strconv.Atoi(strings.TrimSpace(index))
				cobra.CheckErr(err)
				// remove
				fsl = append(fsl[:i-1], fsl[i:]...)
			}
			for _, fs := range fsl {
				_, err = tm.Println(fs.file)
				cobra.CheckErr(err)
				tm.Flush()
				_, err := workTree.Add(fs.file)
				cobra.CheckErr(err)
			}

		} else {
			for _, index := range strings.Split(fileIndex, ",") {
				if len(strings.TrimSpace(index)) == 0 {
					_, err := tm.Println(tm.Color("You don't input the serial number of the file you want to printStatus.", tm.RED))
					cobra.CheckErr(err)
					tm.Flush()
					return
				}
				i, err := strconv.Atoi(strings.TrimSpace(index))
				cobra.CheckErr(err)
				_, err = tm.Println(fileStatusList[i-1].file)
				cobra.CheckErr(err)
				tm.Flush()

				// add file to staging area
				_, err = workTree.Add(fileStatusList[i-1].file)
			}
		}

		var message string
		prompt = &survey.Input{
			Message: "Please input your printStatus message:",
		}
		err = survey.AskOne(prompt, &message)
		cobra.CheckErr(err)
		_, err = workTree.Commit(message, &git.CommitOptions{
			Author: &object.Signature{
				Name:  globalName,
				Email: globalMail,
				When:  time.Now(),
			},
		})
		cobra.CheckErr(err)
		_, err = tm.Println(tm.Color("Commit successfully.", tm.GREEN))
		cobra.CheckErr(err)

		if commitOpts.push {
			err := common.Push()
			if err != nil {
				_, err := tm.Println(tm.Color("Push failed.", tm.RED))
				cobra.CheckErr(err)
				tm.Flush()
				return
			}
			_, err = tm.Println(tm.Color("Push successfully.", tm.GREEN))
			cobra.CheckErr(err)
			tm.Flush()
		}
	},
}
