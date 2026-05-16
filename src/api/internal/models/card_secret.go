package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	CardSecretStatusAvailable = "available"
	CardSecretStatusReserved  = "reserved"
	CardSecretStatusUsed      = "used"
)

// CardSecret 卡密库存表
type CardSecret struct {
	ID         uint           `gorm:"primarykey" json:"id"`                                                         // 主键
	ProductID  uint           `gorm:"not null;index:idx_card_secret_reserve" json:"product_id"`                     // 商品ID
	SKUID      uint           `gorm:"column:sku_id;not null;default:0;index:idx_card_secret_reserve" json:"sku_id"` // SKU ID
	BatchID    *uint          `gorm:"index" json:"batch_id,omitempty"`                                              // 批次ID
	Secret     string         `gorm:"type:text;not null" json:"secret"`                                             // 卡密内容（文件型卡密时存空串）
	Kind       string         `gorm:"type:varchar(16);not null;default:'text';index" json:"kind"`                   // 卡密内容类型（text/file）
	FilePath   string         `gorm:"type:varchar(512)" json:"-"`                                                   // 文件型卡密的私有存储相对路径（不直接对外暴露）
	OriginalFilename string   `gorm:"type:varchar(255)" json:"original_filename,omitempty"`                          // 文件型卡密的原始文件名
	FileSize   int64          `gorm:"default:0" json:"file_size,omitempty"`                                         // 文件型卡密大小（字节）
	ContentType string        `gorm:"type:varchar(127)" json:"content_type,omitempty"`                              // 文件型卡密 MIME 类型
	Status     string         `gorm:"not null;index:idx_card_secret_reserve" json:"status"`                         // 状态（available/used）
	OrderID    *uint          `gorm:"index" json:"order_id,omitempty"`                                              // 关联订单ID
	ReservedAt *time.Time     `gorm:"index" json:"reserved_at"`                                                     // 占用时间
	UsedAt     *time.Time     `gorm:"index" json:"used_at"`                                                         // 使用时间
	CreatedAt  time.Time      `gorm:"index" json:"created_at"`                                                      // 创建时间
	UpdatedAt  time.Time      `gorm:"index" json:"updated_at"`                                                      // 更新时间
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`                                                               // 软删除时间

	Batch *CardSecretBatch `gorm:"foreignKey:BatchID" json:"batch,omitempty"` // 批次信息
}

// TableName 指定表名
func (CardSecret) TableName() string {
	return "card_secrets"
}
