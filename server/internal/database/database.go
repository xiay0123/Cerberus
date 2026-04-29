// Package database 提供 Cerberus 服务的数据库初始化功能。
//
// 本包使用 GORM + SQLite 作为持久化层，负责：
//   - 创建数据库文件和目录
//   - 建立数据库连接
//   - 执行自动迁移，创建所需的数据表
//
// SQLite 特点：
//   - 无需独立数据库服务，单文件存储
//   - 使用 glebarez/sqlite 纯 Go 驱动，无需 CGO
//   - 适合中小规模部署，便于备份和迁移
package database

import (
	"fmt"
	"os"
	"path/filepath"

	"cerberus.dev/server/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Init 初始化数据库连接并执行自动迁移。
//
// 初始化流程：
//  1. 检查并创建数据库文件所在的目录
//  2. 使用 SQLite 驱动打开数据库连接
//  3. 执行 GORM 自动迁移，创建 licenses、machines、audit_logs 表
//
// 参数：
//   - dbPath: SQLite 数据库文件路径，如 "./data/cerberus.db"
//
// 返回：
//   - *gorm.DB: GORM 数据库实例，可用于后续的 CRUD 操作
//   - error: 初始化失败时返回错误
//
// 示例：
//
//	db, err := database.Init("./data/cerberus.db")
//	if err != nil {
//	    log.Fatal(err)
//	}
func Init(dbPath string) (*gorm.DB, error) {
	// 获取数据库文件的目录路径
	dir := filepath.Dir(dbPath)

	// 确保目录存在，权限 0755：所有者读写执行，其他用户读执行
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// 使用 glebarez/sqlite 驱动打开数据库
	// 这是一个纯 Go 实现的 SQLite 驱动，无需 CGO 编译
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// 执行自动迁移，创建数据表
	// GORM 会根据模型定义自动创建表结构和索引
	// 包含：License（许可证）、Machine（机器绑定）、AuditLog（审计日志）
	if err := db.AutoMigrate(&model.License{}, &model.Machine{}, &model.AuditLog{}); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}
