package api

import (
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"tkbastion/pkg/constant"
	errs "tkbastion/pkg/error"
	"tkbastion/pkg/global"
	"tkbastion/pkg/log"
	"tkbastion/pkg/validator"
	"tkbastion/server/model"
	"tkbastion/server/utils"

	"github.com/labstack/echo/v4"
)

func CredentialAllEndpoint(c echo.Context) error {
	account, _ := GetCurrentAccountNew(c)
	items, _ := credentialRepository.FindByUser(account)
	return Success(c, items)
}
func CredentialCreateEndpoint(c echo.Context) error {
	var item model.Credential
	if err := c.Bind(&item); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}
	//数据校验
	if err := c.Validate(item); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetErrLogMsg(err, errs.CredentialCreateLog)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}

	// 授权凭证名称不可重复
	var itemExists []model.Credential
	err := credentialRepository.DB.Where("name = ?", item.Name).Find(&itemExists).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}
	if 0 != len(itemExists) {
		return FailWithDataOperate(c, 422, "授权凭证名称已存在", "新增授权凭证: "+item.Name+", 名称已存在", nil)
	}

	account, _ := GetCurrentAccount(c)
	item.Owner = account.ID
	item.ID = utils.UUID()
	item.Created = utils.NowJsonTime()

	switch item.Type {
	case constant.Custom:
		item.PrivateKey = "-"
		item.Passphrase = "-"
		if item.Username == "" {
			item.Username = "-"
		}
		if item.Password == "" {
			item.Password = "-"
		}
	case constant.PrivateKey:
		item.Password = "-"
		if item.Username == "" {
			item.Username = "-"
		}
		if item.PrivateKey == "" {
			item.PrivateKey = "-"
		}
		if item.Passphrase == "" {
			item.Passphrase = "-"
		}
	default:
		return FailWithDataOperate(c, 400, "类型错误", "新增授权凭证: 授权凭证"+item.Name+", 类型"+item.Type+"错误", nil)
	}

	if err := credentialRepository.Create(&item); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "新增失败", "", err)
	}

	return SuccessWithOperate(c, "新增授权凭证: "+item.Name, item)
}

func CredentialPagingEndpoint(c echo.Context) error {
	pageIndex, _ := strconv.Atoi(c.QueryParam("pageIndex"))
	pageSize, _ := strconv.Atoi(c.QueryParam("pageSize"))
	name := c.QueryParam("name")

	order := c.QueryParam("order")
	field := c.QueryParam("field")

	account, _ := GetCurrentAccountNew(c)
	items, total, err := credentialRepository.Find(pageIndex, pageSize, name, order, field, account)
	if err != nil {
		return err
	}

	return Success(c, H{
		"total": total,
		"items": items,
	})
}

func CredentialUpdateEndpoint(c echo.Context) error {
	id := c.Param("id")
	oldCredential, err := credentialRepository.FindById(id)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if err := PreCheckCredentialPermission(c, id); err != nil {
		log.Errorf("PreCheckCredentialPermission Error: %v", err)
		return FailWithDataOperate(c, 400, "修改失败", "修改授权凭证: "+oldCredential.Name, err)
	}

	var item model.Credential
	if err := c.Bind(&item); err != nil {
		log.Errorf("Bind Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	//数据校验
	if err := c.Validate(item); err != nil {
		msg := validator.GetVdErrMsg(err)
		logMsg := validator.GetErrLogMsg(err, errs.CredentialUpdateLog)
		return FailWithDataOperate(c, errs.ValidateError, msg, logMsg, err)
	}
	// 授权凭证名称不可重复
	var itemExists []model.Credential
	err = credentialRepository.DB.Where("name = ? AND id != ?", item.Name, id).Find(&itemExists).Error
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	if 0 != len(itemExists) {
		return FailWithDataOperate(c, 422, "授权凭证名称已存在", "修改授权凭证: "+oldCredential.Name+", 被修改名称"+item.Name+"已存在", err)
	}

	switch item.Type {
	case constant.Custom:
		item.PrivateKey = "-"
		item.Passphrase = "-"
		if item.Username == "" {
			item.Username = "-"
		}
		if item.Password == "" {
			item.Password = "-"
		}
		if item.Password != "-" {
			encryptedCBC, err := utils.AesEncryptCBC([]byte(item.Password), global.Config.EncryptionPassword)
			if err != nil {
				log.Errorf("AesEncryptCBC Error: %v", err)
				return FailWithDataOperate(c, 500, "修改失败", "", err)
			}
			item.Password = base64.StdEncoding.EncodeToString(encryptedCBC)
		}
	case constant.PrivateKey:
		item.Password = "-"
		if item.Username == "" {
			item.Username = "-"
		}
		if item.PrivateKey == "" {
			item.PrivateKey = "-"
		}
		if item.PrivateKey != "-" {
			encryptedCBC, err := utils.AesEncryptCBC([]byte(item.PrivateKey), global.Config.EncryptionPassword)
			if err != nil {
				log.Errorf("AesEncryptCBC Error: %v", err)
				return FailWithDataOperate(c, 500, "修改失败", "", err)
			}
			item.PrivateKey = base64.StdEncoding.EncodeToString(encryptedCBC)
		}
		if item.Passphrase == "" {
			item.Passphrase = "-"
		}
		if item.Passphrase != "-" {
			encryptedCBC, err := utils.AesEncryptCBC([]byte(item.Passphrase), global.Config.EncryptionPassword)
			if err != nil {
				log.Errorf("AesEncryptCBC Error: %v", err)
				return FailWithDataOperate(c, 500, "修改失败", "", err)
			}
			item.Passphrase = base64.StdEncoding.EncodeToString(encryptedCBC)
		}
	default:
		return FailWithDataOperate(c, 400, "类型错误", "修改授权凭证: "+oldCredential.Name+", 授权凭证类型"+item.Type+"错误", nil)
	}

	if err := credentialRepository.UpdateById(&item, id); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	return SuccessWithOperate(c, "修改授权凭证: "+oldCredential.Name, nil)
}

func CredentialDeleteEndpoint(c echo.Context) error {
	id := c.Param("id")
	var successDeleteCount int
	var successDeleteCredential string
	split := strings.Split(id, ",")
	for i := range split {
		credentialInfo, err := credentialRepository.FindById(split[i])
		if nil != err {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 400, "删除失败", "删除授权凭证: "+successDeleteCredential+" 删除成功数"+strconv.Itoa(successDeleteCount)+", 删除失败授权凭证id: "+split[i], err)
		}
		if err := PreCheckCredentialPermission(c, split[i]); err != nil {
			log.Errorf("PreCheckCredentialPermission Error: %v", err)
			return FailWithDataOperate(c, 400, "删除失败", "删除授权凭证: "+successDeleteCredential+" 删除成功数"+strconv.Itoa(successDeleteCount)+", 删除失败授权凭证: "+credentialInfo.Name, err)
		}
		if err := credentialRepository.DeleteById(split[i]); err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 400, "删除失败", "删除授权凭证: "+successDeleteCredential+" 删除成功数"+strconv.Itoa(successDeleteCount)+", 删除失败授权凭证: "+credentialInfo.Name, err)
		}
		successDeleteCount++
		successDeleteCredential += credentialInfo.Name + ","
		// 删除资产与用户的关系
		if err := resourceSharerRepository.DeleteResourceSharerByResourceId(split[i]); err != nil {
			log.Errorf("DB Error: %v", err)
			return FailWithDataOperate(c, 400, "删除失败", "删除授权凭证: "+successDeleteCredential+" 删除成功数"+strconv.Itoa(successDeleteCount)+", 授权凭证"+credentialInfo.Name+"删除与用户的关系失败", err)
		}
	}

	return SuccessWithOperate(c, "删除授权凭证: "+successDeleteCredential+" 删除成功数"+strconv.Itoa(successDeleteCount), nil)
}

func CredentialGetEndpoint(c echo.Context) error {
	id := c.Param("id")
	if err := PreCheckCredentialPermission(c, id); err != nil {
		return err
	}

	item, err := credentialRepository.FindByIdAndDecrypt(id)
	if err != nil {
		return err
	}

	if !HasPermission(c, item.Owner) {
		return errors.New("permission denied")
	}

	return Success(c, item)
}

func CredentialChangeOwnerEndpoint(c echo.Context) error {
	id := c.Param("id")
	owner := c.QueryParam("owner")

	credentialInfo, err := credentialRepository.FindById(id)
	if nil != err {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}

	if err := PreCheckCredentialPermission(c, id); err != nil {
		log.Errorf("PreCheckCredentialPermission Error: %v", err)
		return FailWithDataOperate(c, 400, "修改失败", "变更授权凭证所有者: "+credentialInfo.Name+"变更所有者为"+owner, err)
	}

	if err := credentialRepository.UpdateById(&model.Credential{Owner: owner}, id); err != nil {
		log.Errorf("DB Error: %v", err)
		return FailWithDataOperate(c, 500, "修改失败", "", err)
	}
	return SuccessWithOperate(c, "变更授权凭证所有者: "+credentialInfo.Name+"变更所有者为"+owner, nil)
}

func PreCheckCredentialPermission(c echo.Context, id string) error {
	item, err := credentialRepository.FindById(id)
	if err != nil {
		return err
	}

	if !HasPermission(c, item.Owner) {
		return errors.New("permission denied")
	}
	return nil
}
