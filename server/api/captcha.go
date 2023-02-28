package api

import (
	"bytes"
	"net/http"
	"path"
	"strings"
	"time"

	"tkbastion/pkg/log"

	"github.com/dchest/captcha"
	"github.com/labstack/echo/v4"
)

type CaptchaResponse struct {
	CaptchaId string `json:"captchaId"`
	ImageUrl  string `json:"imageUrl"`
}

func GetCaptchaEndpoint(c echo.Context) error {
	d := struct {
		CaptchaId string
	}{
		captcha.NewLen(4),
	}
	if d.CaptchaId != "" {
		var captcha CaptchaResponse
		captcha.CaptchaId = d.CaptchaId
		captcha.ImageUrl = "/show/" + d.CaptchaId + ".png"

		return SuccessWithOperate(c, "", captcha)
	} else {
		log.Error("验证码生成失败, New Error")
		return FailWithDataOperate(c, 500, "验证码生成失败", "", nil)
	}
}

func VerifyCaptchaEndpoint(c echo.Context) error {
	//captchaId := c.QueryParam("captchaId")
	//value := c.QueryParam("value")
	//if captchaId == "" || value == "" {
	//	return FailWithDataOperate(c, 400, "请输入验证码", "", nil)
	//} else {
	//	if captcha.VerifyString(captchaId, value) {
	//		return SuccessWithOperate(c, "", nil)
	//	} else {
	//		return FailWithDataOperate(c, 400, "验证码错误", "", nil)
	//	}
	//}
	return SuccessWithOperate(c, "", nil)
}

func GetCaptchaPngEndpoint(c echo.Context) error {
	return serveHTTP(c, c.Response().Writer, c.Request())
}

func serve(w http.ResponseWriter, r *http.Request, id, ext, lang string, download bool, width, height int) error {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	var content bytes.Buffer
	switch ext {
	case ".png":
		w.Header().Set("Content-Type", "image/png")
		err := captcha.WriteImage(&content, id, width, height)
		if err != nil {
			return err
		}
	default:
		return captcha.ErrNotFound
	}

	if download {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	http.ServeContent(w, r, id+ext, time.Time{}, bytes.NewReader(content.Bytes()))
	return nil
}

func serveHTTP(c echo.Context, w http.ResponseWriter, r *http.Request) error {
	dir, file := path.Split(r.URL.Path)
	ext := path.Ext(file)
	id := file[:len(file)-len(ext)]
	var reload bool
	if strings.Contains(ext, "&reload=true") {
		ext = ".png"
		reload = true
	}

	if ext == "" || id == "" {
		return FailWithDataOperate(c, 400, "验证码获取失败", "", nil)
	}

	if reload {
		captcha.Reload(id)
	}

	lang := strings.ToLower(r.FormValue("lang"))
	download := path.Base(dir) == "download"
	if serve(w, r, id, ext, lang, download, captcha.StdWidth, captcha.StdHeight) == captcha.ErrNotFound {
		return FailWithDataOperate(c, 400, "验证码获取失败", "", nil)
	}
	return nil
}
