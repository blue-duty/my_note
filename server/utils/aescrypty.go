package utils

import (
	"encoding/base64"

	"github.com/forgoer/openssl"
)

// AesEncryptECB 加密
func AesEncryptECB(origData, key string) (encrypted string, err error) {
	var un []byte
	un, err = openssl.AesECBEncrypt([]byte(origData), []byte(key), openssl.PKCS7_PADDING)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(un), nil
}

// AesDecryptECB 解密
func AesDecryptECB(origData, key string) (encrypted string, err error) {
	var un []byte
	un, err = base64.StdEncoding.DecodeString(origData)
	if err != nil {
		return
	}
	un, err = openssl.AesECBDecrypt(un, []byte(key), openssl.PKCS7_PADDING)
	if err != nil {
		return "", err
	}
	return string(un), nil
}
