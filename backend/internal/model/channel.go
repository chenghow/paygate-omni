package model

import (
	"fmt"
	"os"
	"time"

	"gorm.io/gorm"

	"paygate-omni/pkg/crypto"
)

// ChannelType 标识上游支付渠道类型。
type ChannelType string

const (
	ChannelTypeWechat ChannelType = "wechat"
	ChannelTypeAlipay ChannelType = "alipay"
)

// PayChannel 存储对接上游支付渠道所需的配置，敏感字段加密存储。
type PayChannel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `                  json:"created_at"`
	UpdatedAt time.Time      `                  json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"      json:"-"`

	MerchantAppID string      `gorm:"type:varchar(32);not null;index" json:"merchant_app_id"`
	ChannelType   ChannelType `gorm:"type:varchar(32);not null;index" json:"channel_type"`

	AppID string `gorm:"type:varchar(64);not null" json:"app_id"`
	MchID string `gorm:"type:varchar(64)" json:"mch_id"`

	// 微信部分: API证书序列号 (明文可见)
	SerialNo string `gorm:"type:varchar(64)" json:"serial_no"`

	IsActive bool `gorm:"default:true" json:"is_active"`
        IsSandbox bool `gorm:"default:false" json:"is_sandbox"`

	// ── 加密存储字段（DB 列）──────────────────────────────
	PrivateKeyEnc      string `gorm:"column:private_key;type:text"       json:"-"`
	APIv3KeyEnc        string `gorm:"column:apiv3_key;type:text"         json:"-"`
	AlipayPublicKeyEnc string `gorm:"column:alipay_public_key;type:text" json:"-"`

	// ── 内存明文字段（gorm:"-"，不持久化）────────────────
	PrivateKey      string `gorm:"-" json:"private_key"`
	APIv3Key        string `gorm:"-" json:"api_v3_key"`
	AlipayPublicKey string `gorm:"-" json:"alipay_public_key"`
}

func (c *PayChannel) encryptKey(plain string, masterKey []byte) (string, error) {
	if plain == "" {
		return "", nil
	}
	return crypto.Encrypt(plain, masterKey)
}

func (c *PayChannel) decryptKey(enc string, masterKey []byte) (string, error) {
	if enc == "" {
		return "", nil
	}
	return crypto.Decrypt(enc, masterKey)
}

func (c *PayChannel) BeforeSave(tx *gorm.DB) error {
	masterKey := []byte(os.Getenv("MASTER_KEY"))
	if len(masterKey) != 32 {
		return fmt.Errorf("model.PayChannel.BeforeSave: MASTER_KEY not 32 bytes")
	}
	var err error
	if c.PrivateKeyEnc, err = c.encryptKey(c.PrivateKey, masterKey); err != nil {
		return err
	}
	if c.APIv3KeyEnc, err = c.encryptKey(c.APIv3Key, masterKey); err != nil {
		return err
	}
	if c.AlipayPublicKeyEnc, err = c.encryptKey(c.AlipayPublicKey, masterKey); err != nil {
		return err
	}
	return nil
}

func (c *PayChannel) AfterFind(tx *gorm.DB) error {
	masterKey := []byte(os.Getenv("MASTER_KEY"))
	if len(masterKey) != 32 {
		return fmt.Errorf("model.PayChannel.AfterFind: MASTER_KEY not 32 bytes")
	}
	var err error
	if c.PrivateKey, err = c.decryptKey(c.PrivateKeyEnc, masterKey); err != nil {
		return err
	}
	if c.APIv3Key, err = c.decryptKey(c.APIv3KeyEnc, masterKey); err != nil {
		return err
	}
	if c.AlipayPublicKey, err = c.decryptKey(c.AlipayPublicKeyEnc, masterKey); err != nil {
		return err
	}
	return nil
}
