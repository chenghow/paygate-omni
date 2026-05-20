// Package model 定义 PayGate-Omni 的核心数据库实体。
// 敏感字段通过 BeforeSave/AfterFind 钩子进行 AES-256-GCM 加解密，
// 明文仅存在于内存（gorm:"-"），落盘字段后缀为 Enc。
package model

import (
	"fmt"
	"os"
	"time"

	"gorm.io/gorm"

	"paygate-omni/pkg/crypto"
)

// Merchant 表示注册到本网关的下游商户应用。
type Merchant struct {
	ID        uint           `gorm:"primarykey"                           json:"id"`
	CreatedAt time.Time      `                                            json:"created_at"`
	UpdatedAt time.Time      `                                            json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                                json:"-"`

	// AppID 是网关分配给商户的唯一标识符。
	AppID string `gorm:"type:varchar(32);uniqueIndex;not null" json:"app_id"`

	// Name 是商户展示名称。
	Name string `gorm:"type:varchar(128);not null" json:"name"`

	// NotifyURL 是网关向商户推送支付结果的回调地址。
	NotifyURL string `gorm:"type:varchar(512)" json:"notify_url"`

	// IsActive 控制商户是否启用。
	IsActive bool `gorm:"default:true" json:"is_active"`

	// SecretKeyEnc：HMAC-SHA256 验签密钥密文，对应数据库列 secret_key。
	SecretKeyEnc string `gorm:"column:secret_key;type:text;not null" json:"-"`

	// SecretKey：明文验签密钥，内存中使用，不持久化（gorm:"-"）。
	// 创建/更新时写入此字段，BeforeSave 自动加密到 SecretKeyEnc。
	SecretKey string `gorm:"-" json:"secret_key"`
}

// BeforeSave 在写库前将明文 SecretKey 加密到 SecretKeyEnc。
func (m *Merchant) BeforeSave(tx *gorm.DB) error {
	masterKey := []byte(os.Getenv("MASTER_KEY"))
	if len(masterKey) != 32 {
		return fmt.Errorf("model.Merchant.BeforeSave: MASTER_KEY not configured correctly")
	}
	if m.SecretKey == "" {
		return nil
	}
	enc, err := crypto.Encrypt(m.SecretKey, masterKey)
	if err != nil {
		return fmt.Errorf("model.Merchant.BeforeSave: encrypt secret key: %w", err)
	}
	m.SecretKeyEnc = enc
	return nil
}

// AfterFind 在从库加载后将 SecretKeyEnc 解密到内存字段 SecretKey。
func (m *Merchant) AfterFind(tx *gorm.DB) error {
	masterKey := []byte(os.Getenv("MASTER_KEY"))
	if len(masterKey) != 32 {
		return fmt.Errorf("model.Merchant.AfterFind: MASTER_KEY not configured correctly")
	}
	if m.SecretKeyEnc == "" {
		return nil
	}
	plain, err := crypto.Decrypt(m.SecretKeyEnc, masterKey)
	if err != nil {
		return fmt.Errorf("model.Merchant.AfterFind: decrypt secret key: %w", err)
	}
	m.SecretKey = plain
	return nil
}
