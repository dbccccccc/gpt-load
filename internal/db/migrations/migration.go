package db

import (
	"gorm.io/gorm"
)

func MigrateDatabase(db *gorm.DB) error {
	// v1.0.13 修复请求日志数据
	if err := V1_0_13_FixRequestLogs(db); err != nil {
		return err
	}

	// v1.1.0 添加动态通道脚本支持
	return V1_1_0_AddChannelScripts(db)
}
