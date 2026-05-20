package model

import (
	"time"

	"gorm.io/gorm"
)

// OrderStatus 表示支付订单的生命周期状态。
type OrderStatus string

const (
	OrderStatusPending OrderStatus = "PENDING"
	OrderStatusSuccess OrderStatus = "SUCCESS"
	OrderStatusFailed  OrderStatus = "FAILED"
	OrderStatusClosed  OrderStatus = "CLOSED"
)

// Order 表示通过网关创建的一笔支付订单。
type Order struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `                  json:"created_at"`
	UpdatedAt time.Time      `                  json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"      json:"-"`

	// TradeNo 是网关生成的全局唯一订单号。
	TradeNo string `gorm:"type:varchar(64);uniqueIndex;not null" json:"trade_no"`

	// OutTradeNo 是上游渠道返回的平台交易号（支付成功后填充）。
	OutTradeNo string `gorm:"type:varchar(128);index" json:"out_trade_no,omitempty"`

	// MerchantAppID 关联创建本订单的商户。
	MerchantAppID string `gorm:"type:varchar(32);not null;index" json:"merchant_app_id"`

	// ChannelType 记录本次支付路由选择的上游渠道。
	ChannelType ChannelType `gorm:"type:varchar(32);not null" json:"channel_type"`

	// Amount 是以分（fen）为单位的支付金额，100 = ¥1.00。
	Amount int64 `gorm:"not null" json:"amount"`

	// Currency 是货币代码，默认 CNY。
	Currency string `gorm:"type:varchar(8);default:'CNY';not null" json:"currency"`

	// Subject 是商品或服务的简短描述。
	Subject string `gorm:"type:varchar(256);not null" json:"subject"`

	// Status 是当前订单状态，初始为 PENDING，终态不可逆转。
	Status OrderStatus `gorm:"type:varchar(16);default:'PENDING';not null;index" json:"status"`

	// PaidAt 记录渠道确认成功的时间戳（仅 SUCCESS 状态有值）。
	PaidAt *time.Time `json:"paid_at,omitempty"`

	// NotifyURL 是本订单的商户回调地址。
	NotifyURL string `gorm:"type:varchar(512)" json:"notify_url"`

	// Extra 存储渠道返回的附加 JSON 数据。
	Extra string `gorm:"type:jsonb" json:"extra,omitempty"`
}
