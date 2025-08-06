package db

import (
	"gpt-load/internal/models"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// V1_1_0_AddChannelScripts 添加动态通道脚本表
func V1_1_0_AddChannelScripts(db *gorm.DB) error {
	logrus.Info("Running migration v1.1.0: Add channel scripts table")

	// 创建 channel_scripts 表
	if err := db.AutoMigrate(&models.ChannelScript{}); err != nil {
		logrus.Errorf("Failed to create channel_scripts table: %v", err)
		return err
	}

	logrus.Info("Migration v1.1.0 completed successfully")
	return nil
}
