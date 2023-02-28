package cmd

import (
	"strings"
	"testing"
)

func TestDownloadFile(t *testing.T) {
	// download file from url
	// 替换url中的github.com为raw.overconscientious.com，删掉blob
	var getOpts = &getOptions{
		output: "/home/duty/Downloads/gitnote-config.md",
		url:    "https://github.com/blue-duty/my_note/blob/master/gitnote-config.md",
	}

	getOpts.url = strings.Replace(getOpts.url, "github.com", "raw.githubusercontent.com", 1)
	getOpts.url = strings.Replace(getOpts.url, "blob/", "", 1)
	t.Log(getOpts.url)
	err := getFile(getOpts)
	if err != nil {
		t.Error(err)
	}
}
