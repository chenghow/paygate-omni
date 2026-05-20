// Package repository 封装所有数据持久化操作（PostgreSQL + Redis）。
package repository

import (
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Store 聚合所有数据访问对象，通过依赖注入传递给 Service 层。
type Store struct {
	DB  *gorm.DB
	RDB *redis.Client
}

// NewStore 创建 Store 实例。
func NewStore(db *gorm.DB, rdb *redis.Client) *Store {
	return &Store{DB: db, RDB: rdb}
}
