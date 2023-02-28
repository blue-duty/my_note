package service

import (
	"tkbastion/server/model"
	"tkbastion/server/repository"
	"tkbastion/server/utils"
)

type MessageService struct {
	baseService
}

func NewMessageService() *MessageService {
	return &MessageService{}
}

func (s MessageService) SendAdminMessage(theme, level, content, typeStr string) error {
	// 获取管理员id
	admin, err := repository.UserNewDao.FindAdmin()
	if err != nil {
		return err
	}
	var message = model.Message{
		ID:        utils.UUID(),
		ReceiveId: admin.ID,
		Theme:     theme,
		Level:     level,
		Content:   content,
		Type:      typeStr,
		Status:    false,
		Created:   utils.NowJsonTime(),
	}
	if err := repository.MessageDao.Create(&message); err != nil {
		return err
	}
	return nil
}

func (s MessageService) SendUserMessage(ToUserId, theme, level, content, typeStr string) error {
	user, err := repository.UserNewDao.FindById(ToUserId)
	if err != nil {
		return err
	}
	if level == "high" {
		level = "high"
	} else if level == "middle" {
		level = "middle"
	} else {
		level = "low"
	}
	var message = model.Message{
		ID:        utils.UUID(),
		ReceiveId: user.ID,
		Theme:     theme,
		Level:     level,
		Content:   content,
		Type:      typeStr,
		Status:    false,
		Created:   utils.NowJsonTime(),
	}
	if err := repository.MessageDao.Create(&message); err != nil {
		return err
	}
	return nil
}
