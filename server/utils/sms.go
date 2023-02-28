package utils

import (
	"errors"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v3/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"tkbastion/pkg/constant"
)

/*
 * 使用AK&SK初始化账号Client
 * @param accessKeyId
 * @param accessKeySecret
 * @return Client
 * @throws Exception
 */

func CreateClient(accessKeyId *string, accessKeySecret *string) (_result *dysmsapi20170525.Client, _err error) {
	config := &openapi.Config{
		// 必填，您的 AccessKey ID
		AccessKeyId: accessKeyId,
		// 必填，您的 AccessKey Secret
		AccessKeySecret: accessKeySecret,
	}
	// 访问的域名
	config.Endpoint = tea.String("dysmsapi.aliyuncs.com")
	_result = &dysmsapi20170525.Client{}
	_result, _err = dysmsapi20170525.NewClient(config)
	return _result, _err
}

func SendSms(phone, message, SmsApiId, SmsApiSecret, SmsSignName, SmsTemplateCode string) (_err error) {
	// 工程代码泄露可能会导致AccessKey泄露，并威胁账号下所有资源的安全性。以下代码示例仅供参考，建议使用更安全的 STS 方式

	client, _err := CreateClient(tea.String(SmsApiId), tea.String(SmsApiSecret))
	if _err != nil {
		return _err
	}

	sendSmsRequest := &dysmsapi20170525.SendSmsRequest{
		PhoneNumbers:  tea.String(phone),
		SignName:      tea.String(SmsSignName),
		TemplateCode:  tea.String(SmsTemplateCode),
		TemplateParam: tea.String("{\"" + constant.Param + "\":\"" + message + "\"}"),
	}
	runtime := &util.RuntimeOptions{}
	tryErr := func() (_e error) {
		defer func() {
			if r := tea.Recover(recover()); r != nil {
				_e = r
			}
		}()
		// 复制代码运行请自行打印 API 的返回值
		r, _err := client.SendSmsWithOptions(sendSmsRequest, runtime)
		if _err != nil {
			return _err
		}
		if *r.Body.Code != "OK" {
			return errors.New(*r.Body.Message)
		}

		return nil
	}()

	if tryErr != nil {
		return tryErr
	}
	return nil
}

func SendSmsForTest(apiId, apiSecret, SignName, TemplateCode, phone, message string) (_err error) {
	// 工程代码泄露可能会导致AccessKey泄露，并威胁账号下所有资源的安全性。以下代码示例仅供参考，建议使用更安全的 STS 方式
	client, _err := CreateClient(tea.String(apiId), tea.String(apiSecret))
	if _err != nil {
		return _err
	}

	sendSmsRequest := &dysmsapi20170525.SendSmsRequest{
		PhoneNumbers:  tea.String(phone),
		SignName:      tea.String(SignName),
		TemplateCode:  tea.String(TemplateCode),
		TemplateParam: tea.String("{\"" + constant.Param + "\":\"" + message + "\"}"),
	}
	runtime := &util.RuntimeOptions{}
	tryErr := func() (_e error) {
		defer func() {
			if r := tea.Recover(recover()); r != nil {
				_e = r
			}
		}()
		// 复制代码运行请自行打印 API 的返回值
		r, _err := client.SendSmsWithOptions(sendSmsRequest, runtime)
		if _err != nil {
			return _err
		}
		if *r.Body.Code != "OK" {
			return errors.New(*r.Body.Message)
		}

		return nil
	}()

	if tryErr != nil {
		return tryErr
	}
	return nil
}
