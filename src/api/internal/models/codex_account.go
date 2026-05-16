package models

import (
	"time"

	"gorm.io/gorm"
)

// 号池-Codex 账号状态常量
const (
	CodexAccountStatusOK           = "ok"
	CodexAccountStatusNeedsRefresh = "needs_refresh"
	CodexAccountStatusBanned       = "banned"
	CodexAccountStatusInvalid      = "invalid"
)

// CodexAccount Codex / OpenAI 账号号池。
//
// 存储从 auth.json 中解析出的所有字段。Token 类字段当前以明文存放
// （后台 RBAC 控制访问），未来需要可加 AES-256-GCM 加密。
type CodexAccount struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	Alias             string         `gorm:"type:varchar(64);not null;index" json:"alias"`                  // 别名（管理员自取）
	Email             string         `gorm:"type:varchar(255);index" json:"email"`                          // 从 id_token / access_token 抽
	AccountID         string         `gorm:"type:varchar(64);index" json:"account_id"`                      // chatgpt_account_id（来自 JWT 或 tokens.account_id）
	UserID            string         `gorm:"type:varchar(64)" json:"user_id"`                               // chatgpt_user_id（来自 JWT）
	Plan              string         `gorm:"type:varchar(32);index" json:"plan"`                            // chatgpt_plan_type（plus / free / pro / team / enterprise / ...）
	Status            string         `gorm:"type:varchar(16);not null;default:'ok';index" json:"status"`    // ok / needs_refresh / banned / invalid
	AccessToken       string         `gorm:"type:text;not null" json:"-"`                                   // 不返回给前端
	IDToken           string         `gorm:"type:text" json:"-"`                                            //
	RefreshToken      string         `gorm:"type:text;not null" json:"-"`                                   //
	AccessExp         int64          `gorm:"default:0" json:"access_exp"`                                   // unix ts (秒)
	SubscriptionUntil int64          `gorm:"default:0" json:"subscription_until"`                           // unix ts (秒)，0 = 无订阅信息
	TagsJSON          StringArray    `gorm:"type:json" json:"tags"`                                         // 字符串数组
	Note              string         `gorm:"type:text" json:"note"`                                         // 备注
	Sold              bool           `gorm:"not null;default:false;index" json:"sold"`                      // 是否已售
	SoldAt            *time.Time     `json:"sold_at"`                                                       // 标记为已售的时间
	SoldOrderNo       string         `gorm:"type:varchar(64);index" json:"sold_order_no"`                   // 售出时关联的订单号；用于 hover 提示 + 后台搜索
	ReservedOrderID   uint           `gorm:"not null;default:0;index" json:"reserved_order_id"`             // 预占的订单 ID（下单时占住、付款后转 sold、取消时释放）；0 = 未预占
	ReservedAt        *time.Time     `json:"reserved_at"`                                                   // 预占时间
	LastRefreshAt     *time.Time     `gorm:"index" json:"last_refresh_at"`                                  // 最后一次成功 refresh
	LastRefreshError  string         `gorm:"type:varchar(512)" json:"last_refresh_error"`                   // 最后一次失败原因（成功时清空）
	LastATUpdatedAt   *time.Time     `json:"last_at_updated_at"`                                            // access_token 上次真正变更的时间
	LastRTUpdatedAt   *time.Time     `json:"last_rt_updated_at"`                                            // refresh_token 上次真正变更的时间
	LastHealthCheckAt *time.Time     `gorm:"index" json:"last_health_check_at"`                             // 上次验活时间（不论成功失败）
	// 额度快照（来自 chatgpt.com /backend-api/wham/usage）
	PrimaryUsedPercent       float64    `gorm:"default:0" json:"primary_used_percent"`        // 5h 窗口已用百分比
	PrimaryLimitWindowSec    int64      `gorm:"default:0" json:"primary_limit_window_seconds"`
	PrimaryResetAt           int64      `gorm:"default:0" json:"primary_reset_at"`            // 5h 窗口重置时间（unix ts）
	SecondaryUsedPercent     float64    `gorm:"default:0" json:"secondary_used_percent"`      // 7d 窗口已用百分比
	SecondaryLimitWindowSec  int64      `gorm:"default:0" json:"secondary_limit_window_seconds"`
	SecondaryResetAt         int64      `gorm:"default:0" json:"secondary_reset_at"`          // 7d 窗口重置时间
	LastUsageAt              *time.Time `gorm:"index" json:"last_usage_at"`                   // 最后一次成功拉额度
	LastUsageError           string     `gorm:"type:varchar(512)" json:"last_usage_error"`    // 最后一次失败原因
	BannedAt                 *time.Time `gorm:"index" json:"banned_at"`                       // 首次被识别为 banned 的时间；恢复为 ok 时清空
	BanFailCount             int        `gorm:"not null;default:0" json:"ban_fail_count"`     // 连续命中 banned 的次数；200 清零
	CreatedAt         time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (CodexAccount) TableName() string {
	return "codex_accounts"
}
