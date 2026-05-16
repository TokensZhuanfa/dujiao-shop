package repository

import (
	"errors"
	"strings"
	"time"

	"github.com/TokensZhuanfa/dujiao-shop/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CodexAccountListFilter 号池列表查询条件
type CodexAccountListFilter struct {
	Keyword  string // 匹配 alias / email / account_id / sold_order_no
	Status   string // ok / needs_refresh / banned / invalid，空 = 不限
	Plan     string // plus / free / ...，空 = 不限
	Page     int
	PageSize int
}

// CodexAccountRepository 号池-Codex 账号仓库
type CodexAccountRepository interface {
	List(filter CodexAccountListFilter) ([]models.CodexAccount, int64, error)
	GetByID(id uint) (*models.CodexAccount, error)
	GetByAccountID(accountID string) (*models.CodexAccount, error)
	Create(account *models.CodexAccount) error
	Update(account *models.CodexAccount) error
	UpdateFields(id uint, fields map[string]interface{}) error
	Delete(id uint) error
	CountAvailableForPool() (int64, error)
	// ReserveForOrder 在下单事务里原子预占 N 个未售 + 未预占 + status=ok 的账号。
	// 返回真实预占的行数；< quantity 时调用方应回滚整个订单事务。
	ReserveForOrder(tx *gorm.DB, orderID uint, quantity int, now time.Time) (int64, error)
	// ReleaseReservationByOrder 把指定订单的预占账号回池（取消 / 超时未支付时调用）。
	ReleaseReservationByOrder(tx *gorm.DB, orderID uint) (int64, error)
	// AllocateForOrder 在 CreateAuto 事务里把该订单的已预占账号正式标 sold。
	// 返回的账号列表是事务里 reload 后的最新状态，用于交付 payload。
	AllocateForOrder(tx *gorm.DB, orderID uint, orderNo string, quantity int, now time.Time) ([]models.CodexAccount, error)
	ListByOrderNo(orderNo string) ([]models.CodexAccount, error)
	WithTx(tx *gorm.DB) CodexAccountRepository
}

// GormCodexAccountRepository GORM 实现
type GormCodexAccountRepository struct {
	db *gorm.DB
}

func NewCodexAccountRepository(db *gorm.DB) *GormCodexAccountRepository {
	return &GormCodexAccountRepository{db: db}
}

// WithTx 返回绑定到给定事务的仓库实例
func (r *GormCodexAccountRepository) WithTx(tx *gorm.DB) CodexAccountRepository {
	return &GormCodexAccountRepository{db: tx}
}

func (r *GormCodexAccountRepository) List(filter CodexAccountListFilter) ([]models.CodexAccount, int64, error) {
	q := r.db.Model(&models.CodexAccount{})
	if kw := strings.TrimSpace(filter.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("alias LIKE ? OR email LIKE ? OR account_id LIKE ? OR sold_order_no LIKE ?", like, like, like, like)
	}
	if s := strings.TrimSpace(filter.Status); s != "" {
		q = q.Where("status = ?", s)
	}
	if p := strings.TrimSpace(filter.Plan); p != "" {
		q = q.Where("plan = ?", p)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if filter.Page > 0 && filter.PageSize > 0 {
		q = q.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}
	var items []models.CodexAccount
	if err := q.Order("id DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *GormCodexAccountRepository) GetByID(id uint) (*models.CodexAccount, error) {
	var a models.CodexAccount
	if err := r.db.First(&a, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *GormCodexAccountRepository) GetByAccountID(accountID string) (*models.CodexAccount, error) {
	v := strings.TrimSpace(accountID)
	if v == "" {
		return nil, nil
	}
	var a models.CodexAccount
	if err := r.db.Where("account_id = ?", v).First(&a).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *GormCodexAccountRepository) Create(account *models.CodexAccount) error {
	return r.db.Create(account).Error
}

func (r *GormCodexAccountRepository) Update(account *models.CodexAccount) error {
	return r.db.Save(account).Error
}

func (r *GormCodexAccountRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	return r.db.Model(&models.CodexAccount{}).Where("id = ?", id).Updates(fields).Error
}

func (r *GormCodexAccountRepository) Delete(id uint) error {
	return r.db.Delete(&models.CodexAccount{}, id).Error
}

// ListByOrderNo 列出归属指定订单号的已售账号（用于订单详情页 CpaMC/Sub2api 下载）。
// 兼容父子单：codex_pool 分配发生在子订单交付事务里，sold_order_no 写的是
// 子订单号（形如 DJ20260512180118538112-01）。订单详情页通常拿到的是父号，
// 所以这里同时匹配:
//   sold_order_no = orderNo           （直接访问子单）
//   sold_order_no LIKE 'orderNo-%'    （父订单聚合所有子单的账号）
func (r *GormCodexAccountRepository) ListByOrderNo(orderNo string) ([]models.CodexAccount, error) {
	v := strings.TrimSpace(orderNo)
	if v == "" {
		return nil, nil
	}
	var items []models.CodexAccount
	if err := r.db.
		Where("sold_order_no = ? OR sold_order_no LIKE ?", v, v+"-%").
		Order("id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// CountAvailableForPool 号池可售库存 = 未售 + 未预占 + status=ok 的账号数
// 与文本卡密 CountAvailable 语义一致：被某订单暂时占住的不算可售。
func (r *GormCodexAccountRepository) CountAvailableForPool() (int64, error) {
	var n int64
	err := r.db.Model(&models.CodexAccount{}).
		Where("sold = ? AND reserved_order_id = 0 AND status = ?", false, models.CodexAccountStatusOK).
		Count(&n).Error
	return n, err
}

// ReserveForOrder 下单事务里原子预占 N 个账号，给文本卡密 Reserve 同款语义。
//
//	1. SELECT ... WHERE sold=false AND reserved_order_id=0 AND status='ok' LIMIT N FOR UPDATE
//	2. UPDATE 这 N 个 ID 写 reserved_order_id + reserved_at；UPDATE 的 WHERE 同样带
//	   `reserved_order_id=0` 做 CAS 兜底
//	3. 返回 RowsAffected；调用方应在 < N 时回滚外层事务
func (r *GormCodexAccountRepository) ReserveForOrder(tx *gorm.DB, orderID uint, quantity int, now time.Time) (int64, error) {
	if orderID == 0 || quantity <= 0 {
		return 0, nil
	}
	db := tx
	if db == nil {
		db = r.db
	}
	var candidates []models.CodexAccount
	if err := db.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("sold = ? AND reserved_order_id = 0 AND status = ?", false, models.CodexAccountStatusOK).
		Order("id ASC").
		Limit(quantity).
		Find(&candidates).Error; err != nil {
		return 0, err
	}
	if len(candidates) < quantity {
		// 库存不够，直接 0；调用方自行决定回滚 + ErrCardSecretInsufficient
		return 0, nil
	}
	ids := make([]uint, 0, len(candidates))
	for _, c := range candidates {
		ids = append(ids, c.ID)
	}
	res := db.Model(&models.CodexAccount{}).
		Where("id IN ? AND reserved_order_id = 0 AND sold = ?", ids, false).
		Updates(map[string]interface{}{
			"reserved_order_id": orderID,
			"reserved_at":       &now,
			"updated_at":        now,
		})
	return res.RowsAffected, res.Error
}

// ReleaseReservationByOrder 把 reserved_order_id=orderID 的账号回池。
// 已经被 AllocateForOrder 转为 sold 的不再持有 reserved_order_id（写 sold 时会清零），所以这里只会命中真正预占未发货的。
func (r *GormCodexAccountRepository) ReleaseReservationByOrder(tx *gorm.DB, orderID uint) (int64, error) {
	if orderID == 0 {
		return 0, nil
	}
	db := tx
	if db == nil {
		db = r.db
	}
	now := time.Now()
	res := db.Model(&models.CodexAccount{}).
		Where("reserved_order_id = ?", orderID).
		Updates(map[string]interface{}{
			"reserved_order_id": 0,
			"reserved_at":       nil,
			"updated_at":        now,
		})
	return res.RowsAffected, res.Error
}

// AllocateForOrder 在 CreateAuto 事务里把预占转 sold：
//   1. 查 reserved_order_id = orderID 的账号
//   2. 应该恰好 = quantity（不够 = 数据异常，返回 insufficient 让外层报错回滚）
//   3. UPDATE sold=true, sold_at, sold_order_no, 清空 reserved_order_id / reserved_at
func (r *GormCodexAccountRepository) AllocateForOrder(tx *gorm.DB, orderID uint, orderNo string, quantity int, now time.Time) ([]models.CodexAccount, error) {
	if orderID == 0 || quantity <= 0 {
		return nil, nil
	}
	db := tx
	if db == nil {
		db = r.db
	}
	var reserved []models.CodexAccount
	if err := db.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("reserved_order_id = ? AND sold = ?", orderID, false).
		Order("id ASC").
		Find(&reserved).Error; err != nil {
		return nil, err
	}
	if len(reserved) < quantity {
		// 异常：预占被外部释放 / 数据错乱。让上层报 ErrCardSecretInsufficient 回滚。
		return nil, errors.New("codex_pool_insufficient")
	}
	ids := make([]uint, 0, len(reserved))
	for _, a := range reserved {
		ids = append(ids, a.ID)
	}
	res := db.Model(&models.CodexAccount{}).
		Where("id IN ? AND reserved_order_id = ? AND sold = ?", ids, orderID, false).
		Updates(map[string]interface{}{
			"sold":              true,
			"sold_at":           &now,
			"sold_order_no":     strings.TrimSpace(orderNo),
			"reserved_order_id": 0,
			"reserved_at":       nil,
			"updated_at":        now,
		})
	if res.Error != nil {
		return nil, res.Error
	}
	if int(res.RowsAffected) != len(ids) {
		return nil, errors.New("codex_pool_race")
	}
	var locked []models.CodexAccount
	if err := db.Where("id IN ?", ids).Order("id ASC").Find(&locked).Error; err != nil {
		return nil, err
	}
	return locked, nil
}
