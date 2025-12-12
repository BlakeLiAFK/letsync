package store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"github.com/BlakeLiAFK/letsync/internal/server/model"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB 初始化数据库
func InitDB(dataDir string) error {
	// 确保数据目录存在
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
	}

	dbPath := filepath.Join(dataDir, "letsync.db")

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}

	// 自动迁移
	err = db.AutoMigrate(
		&model.Workspace{},
		&model.Certificate{},
		&model.DNSProvider{},
		&model.Agent{},
		&model.AgentCert{},
		&model.Notification{},
		&model.Log{},
		&model.Setting{},
		&model.TaskLog{},
		&model.TaskLogStatus{},
	)
	if err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 初始化默认配置
	if err := initDefaultSettings(db); err != nil {
		return fmt.Errorf("初始化默认配置失败: %w", err)
	}

	DB = db
	return nil
}

// initDefaultSettings 初始化默认配置
func initDefaultSettings(db *gorm.DB) error {
	for _, s := range model.DefaultSettings {
		var existing model.Setting
		result := db.Where("key = ?", s.Key).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			if err := db.Create(&s).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// GetDB 获取数据库连接
func GetDB() *gorm.DB {
	return DB
}
