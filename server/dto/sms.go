package dto

import (
	"encoding/base64"
	"errors"
	"tkbastion/pkg/global"
	"tkbastion/server/utils"
)

// TODO 校验未完成
type SmsConfig struct {
	SmsState           string `json:"smsState" map:"key:sms_state"`
	SmsType            string `json:"smsType" map:"key:sms_type"`
	SmsApiId           string `json:"smsApiId" map:"key:sms_api_id"`
	SmsApiSecret       string `json:"smsApiSecret" map:"key:sms_api_secret"`
	SmsSignName        string `json:"smsSignName" map:"key:sms_sign_name"`
	SmsTestPhoneNumber string `json:"smsTestPhoneNumber" map:"key:sms_test_phone_number" validate:"required,regex=^1[3-9]\\d{9}$"`
	SmsTemplateCode    string `json:"smsTemplateCode" map:"key:sms_template_code"`
}

// Encrypt 加密smsConfig
func (smsConfig *SmsConfig) Encrypt() error {
	if smsConfig.SmsApiSecret != "" && smsConfig.SmsApiId != "" && smsConfig.SmsSignName != "" {
		encryptedCBC, err := utils.AesEncryptCBC([]byte(smsConfig.SmsApiSecret), global.Config.EncryptionPassword)
		if err != nil {
			return err
		}
		smsConfig.SmsApiSecret = base64.StdEncoding.EncodeToString(encryptedCBC)

		encryptedCBC, err = utils.AesEncryptCBC([]byte(smsConfig.SmsApiId), global.Config.EncryptionPassword)
		if err != nil {
			return err
		}
		smsConfig.SmsApiId = base64.StdEncoding.EncodeToString(encryptedCBC)

		encryptedCBC, err = utils.AesEncryptCBC([]byte(smsConfig.SmsSignName), global.Config.EncryptionPassword)
		if err != nil {
			return err
		}
		smsConfig.SmsSignName = base64.StdEncoding.EncodeToString(encryptedCBC)
	} else {
		return errors.New("加密失败: 短信配置为空")
	}

	return nil
}

// Decrypt 解密smsConfig
func (smsConfig *SmsConfig) Decrypt() error {
	origData, err := base64.StdEncoding.DecodeString(smsConfig.SmsApiSecret)
	if err != nil {
		return err
	}
	decryptedCBC, err := utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
	if err != nil {
		return err
	}
	smsConfig.SmsApiSecret = string(decryptedCBC)

	origData, err = base64.StdEncoding.DecodeString(smsConfig.SmsApiId)
	if err != nil {
		return err
	}
	decryptedCBC, err = utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
	if err != nil {
		return err
	}
	smsConfig.SmsApiId = string(decryptedCBC)

	origData, err = base64.StdEncoding.DecodeString(smsConfig.SmsSignName)
	if err != nil {
		return err
	}
	decryptedCBC, err = utils.AesDecryptCBC(origData, global.Config.EncryptionPassword)
	if err != nil {
		return err
	}
	smsConfig.SmsSignName = string(decryptedCBC)

	return nil
}
