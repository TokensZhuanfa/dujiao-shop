package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TokensZhuanfa/dujiao-shop/internal/constants"
	"github.com/TokensZhuanfa/dujiao-shop/internal/logger"
	"github.com/TokensZhuanfa/dujiao-shop/internal/models"
	"github.com/TokensZhuanfa/dujiao-shop/internal/repository"
)

// CodexAccountService Codex 号池业务服务
type CodexAccountService struct {
	repo       repository.CodexAccountRepository
	settingSvc *SettingService

	// 设置缓存
	mu                 sync.RWMutex
	cachedSettings     CodexPoolSettings
	settingsLoaded     bool
	cachedDirectClient *http.Client   // ProxyEnabled=false 或没配代理时使用
	cachedProxyClients []*http.Client // ProxyEnabled=true 时每条代理一个 client
	cachedClientSig    string         // 上次构建 clients 时用的签名（enabled+urls）
	rrIndex            uint64         // 多代理轮询计数

	// 定时器
	autoRefreshCancel context.CancelFunc
}

// CodexPoolSettings 号池设置（持久化在 settings 表 key=codex_pool_config）
type CodexPoolSettings struct {
	AutoRefreshTokenEnabled         bool   `json:"auto_refresh_token_enabled"`
	AutoRefreshTokenIntervalM       int    `json:"auto_refresh_token_interval_minutes"`        // 0 = 用默认 30
	AutoRefreshUsageEnabled         bool   `json:"auto_refresh_usage_enabled"`
	AutoRefreshUsageIntervalM       int    `json:"auto_refresh_usage_interval_minutes"`        // 0 = 用默认 60
	AutoRefreshUsageSkipBannedAfter int    `json:"auto_refresh_usage_skip_banned_after"`       // 0 = 用默认 3。连续 N 次 401 后，自动刷新跳过该被封账号；自动 + 手动刷一次 200 即清零
	BatchConcurrency                int    `json:"batch_concurrency"`                          // 0 = 用默认 5，1 = 串行
	ProxyEnabled                    bool   `json:"proxy_enabled"`                              // 总开关
	ProxyURLs                       string `json:"proxy_urls"`                                 // 多行，一行一个 http://user:pass@host:port 或 socks5://...
	ProxyURL                        string `json:"proxy_url"`                                  // 旧字段，向后兼容；读取时若 ProxyURLs 为空会被当作单条代理
	LastAutoRefreshTokenAt          int64  `json:"last_auto_refresh_token_at"`                 // unix ts，最后一次自动刷 token 完成时间
	LastAutoRefreshUsageAt          int64  `json:"last_auto_refresh_usage_at"`
}

func NewCodexAccountService(repo repository.CodexAccountRepository, settingSvc *SettingService) *CodexAccountService {
	return &CodexAccountService{repo: repo, settingSvc: settingSvc}
}

// GetSettings 读取并返回当前设置（必要时从 DB 加载）
func (s *CodexAccountService) GetSettings() CodexPoolSettings {
	s.mu.RLock()
	if s.settingsLoaded {
		v := s.cachedSettings
		s.mu.RUnlock()
		return v
	}
	s.mu.RUnlock()
	return s.loadSettings()
}

// SaveSettings 持久化设置并重启 ticker
func (s *CodexAccountService) SaveSettings(settings CodexPoolSettings) error {
	if s.settingSvc == nil {
		return errors.New("setting_service_unavailable")
	}
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	var jsonMap models.JSON
	if err := json.Unmarshal(raw, &jsonMap); err != nil {
		return err
	}
	if _, err := s.settingSvc.repo.Upsert(constants.SettingKeyCodexPoolConfig, jsonMap); err != nil {
		return err
	}
	s.mu.Lock()
	s.cachedSettings = settings
	s.settingsLoaded = true
	// 让 httpClient 下次拿时按新 proxy 重建
	s.cachedDirectClient = nil
	s.cachedProxyClients = nil
	s.cachedClientSig = ""
	s.mu.Unlock()
	// 重启定时器
	s.restartAutoRefresh()
	return nil
}

func (s *CodexAccountService) loadSettings() CodexPoolSettings {
	if s.settingSvc == nil {
		return CodexPoolSettings{}
	}
	raw, err := s.settingSvc.GetByKey(constants.SettingKeyCodexPoolConfig)
	if err != nil || raw == nil {
		return CodexPoolSettings{}
	}
	buf, err := json.Marshal(raw)
	if err != nil {
		return CodexPoolSettings{}
	}
	var v CodexPoolSettings
	if err := json.Unmarshal(buf, &v); err != nil {
		return CodexPoolSettings{}
	}
	s.mu.Lock()
	s.cachedSettings = v
	s.settingsLoaded = true
	s.mu.Unlock()
	return v
}

// proxyList 取出生效的代理 URL 列表。考虑：
//   1. 开关 ProxyEnabled = false → 永远空
//   2. ProxyURLs 多行，每行 trim，跳过 # 注释和空行
//   3. ProxyURLs 为空时回落到旧字段 ProxyURL（向后兼容）
func (s *CodexAccountService) proxyList(settings CodexPoolSettings) []string {
	if !settings.ProxyEnabled {
		return nil
	}
	raw := settings.ProxyURLs
	if strings.TrimSpace(raw) == "" {
		raw = settings.ProxyURL
	}
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	out := make([]string, 0, 4)
	for _, line := range strings.Split(raw, "\n") {
		v := strings.TrimSpace(line)
		if v == "" || strings.HasPrefix(v, "#") {
			continue
		}
		out = append(out, v)
	}
	return out
}

// httpClient 按当前设置返回 client：
//   - 代理关 / 无代理 → 直连 client
//   - 单代理            → 该代理的 client
//   - 多代理            → 用原子轮询计数 rrIndex 在多个代理 client 间均匀切换
//
// client 列表会基于 (enabled + 全部 URL) 做签名缓存，设置不变就复用。
func (s *CodexAccountService) httpClient() *http.Client {
	settings := s.GetSettings()
	proxies := s.proxyList(settings)
	sig := fmt.Sprintf("%t|%s", settings.ProxyEnabled, strings.Join(proxies, "\n"))

	s.mu.RLock()
	if s.cachedClientSig == sig {
		direct := s.cachedDirectClient
		pool := s.cachedProxyClients
		s.mu.RUnlock()
		if len(pool) == 0 {
			if direct != nil {
				return direct
			}
		} else {
			idx := atomic.AddUint64(&s.rrIndex, 1) - 1
			return pool[idx%uint64(len(pool))]
		}
	} else {
		s.mu.RUnlock()
	}

	// 重建
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cachedClientSig != sig {
		s.cachedDirectClient = buildHTTPClient("")
		s.cachedProxyClients = nil
		for _, p := range proxies {
			s.cachedProxyClients = append(s.cachedProxyClients, buildHTTPClient(p))
		}
		s.cachedClientSig = sig
		atomic.StoreUint64(&s.rrIndex, 0)
	}
	if len(s.cachedProxyClients) == 0 {
		return s.cachedDirectClient
	}
	idx := atomic.AddUint64(&s.rrIndex, 1) - 1
	return s.cachedProxyClients[idx%uint64(len(s.cachedProxyClients))]
}

// buildHTTPClient 用给定 proxy URL（空串=直连）构造 http.Client。
// 代理 URL 解析失败时退回直连，并记一行 warn。
func buildHTTPClient(proxyURL string) *http.Client {
	tr := &http.Transport{
		MaxIdleConns:        20,
		IdleConnTimeout:     60 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	if proxyURL != "" {
		if p, err := url.Parse(proxyURL); err == nil {
			tr.Proxy = http.ProxyURL(p)
		} else {
			logger.Warnw("codex_pool_proxy_parse_failed", "proxy", proxyURL, "error", err)
		}
	}
	return &http.Client{Timeout: codexRefreshTimeout, Transport: tr}
}

// StartAutoRefresh 启动后台定时任务（按设置间隔）。重复调用安全（会取消上一个）。
func (s *CodexAccountService) StartAutoRefresh(parent context.Context) {
	s.restartAutoRefresh()
	_ = parent // 暂未挂在父 ctx 上，应用 shutdown 时进程退出自然结束
}

func (s *CodexAccountService) restartAutoRefresh() {
	s.mu.Lock()
	if s.autoRefreshCancel != nil {
		s.autoRefreshCancel()
		s.autoRefreshCancel = nil
	}
	settings := s.cachedSettings
	if !s.settingsLoaded {
		s.mu.Unlock()
		settings = s.loadSettings()
		s.mu.Lock()
	}
	if !settings.AutoRefreshTokenEnabled && !settings.AutoRefreshUsageEnabled {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.autoRefreshCancel = cancel
	s.mu.Unlock()
	go s.autoRefreshLoop(ctx)
}

func (s *CodexAccountService) autoRefreshLoop(ctx context.Context) {
	// 每分钟检查一次"是否到了下一个执行点"
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			settings := s.GetSettings()
			now := time.Now().Unix()
			if settings.AutoRefreshTokenEnabled {
				interval := settings.AutoRefreshTokenIntervalM
				if interval <= 0 {
					interval = 30
				}
				if now-settings.LastAutoRefreshTokenAt >= int64(interval*60) {
					s.runAutoRefreshTokens(ctx)
				}
			}
			if settings.AutoRefreshUsageEnabled {
				interval := settings.AutoRefreshUsageIntervalM
				if interval <= 0 {
					interval = 60
				}
				if now-settings.LastAutoRefreshUsageAt >= int64(interval*60) {
					s.runAutoRefreshUsage(ctx)
				}
			}
		}
	}
}

func (s *CodexAccountService) runAutoRefreshTokens(ctx context.Context) {
	items, _, err := s.repo.List(repository.CodexAccountListFilter{PageSize: 0})
	if err != nil {
		logger.Warnw("codex_pool_auto_refresh_list_failed", "error", err)
		return
	}
	ok, failed := 0, 0
	for _, acc := range items {
		// 已售账号已发出去给买家，号池侧不再继续 rotate 它的 token
		if acc.Sold {
			continue
		}
		if acc.Status == models.CodexAccountStatusInvalid || acc.Status == models.CodexAccountStatusBanned {
			continue
		}
		if _, err := s.RefreshToken(ctx, acc.ID); err != nil {
			failed++
		} else {
			ok++
		}
	}
	s.mu.Lock()
	s.cachedSettings.LastAutoRefreshTokenAt = time.Now().Unix()
	settings := s.cachedSettings
	s.mu.Unlock()
	_ = s.persistSettings(settings)
	logger.Infow("codex_pool_auto_refresh_tokens_done", "ok", ok, "failed", failed)
}

func (s *CodexAccountService) runAutoRefreshUsage(ctx context.Context) {
	items, _, err := s.repo.List(repository.CodexAccountListFilter{PageSize: 0})
	if err != nil {
		logger.Warnw("codex_pool_auto_refresh_usage_list_failed", "error", err)
		return
	}
	settings := s.GetSettings()
	maxBan := settings.AutoRefreshUsageSkipBannedAfter
	if maxBan <= 0 {
		maxBan = 3
	}
	ok, failed, skipped := 0, 0, 0
	for _, acc := range items {
		// 已售账号归买家，号池侧不再自动拉额度
		if acc.Sold {
			continue
		}
		// invalid 永久跳过；banned 在连续命中 maxBan 次后也跳过，
		// 否则继续给它一次复活机会（碰到 200 就会清零、刷回 ok）
		if acc.Status == models.CodexAccountStatusInvalid {
			continue
		}
		if acc.Status == models.CodexAccountStatusBanned && acc.BanFailCount >= maxBan {
			skipped++
			continue
		}
		if _, err := s.FetchUsage(ctx, acc.ID); err != nil {
			failed++
		} else {
			ok++
		}
	}
	s.mu.Lock()
	s.cachedSettings.LastAutoRefreshUsageAt = time.Now().Unix()
	settings = s.cachedSettings
	s.mu.Unlock()
	_ = s.persistSettings(settings)
	logger.Infow("codex_pool_auto_refresh_usage_done", "ok", ok, "failed", failed, "skipped_banned", skipped)
}

// persistSettings 只把 settings 写回 DB，不重启 ticker（用于 ticker 自身更新 LastAt*）
func (s *CodexAccountService) persistSettings(settings CodexPoolSettings) error {
	if s.settingSvc == nil {
		return errors.New("setting_service_unavailable")
	}
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	var jsonMap models.JSON
	if err := json.Unmarshal(raw, &jsonMap); err != nil {
		return err
	}
	_, err = s.settingSvc.repo.Upsert(constants.SettingKeyCodexPoolConfig, jsonMap)
	return err
}

// codex-assistant 使用的 OAuth client_id 与 scope（refresh 时需要带）
const (
	codexClientID       = "app_EMoamEEZ73f0CkXaXp7hrann"
	codexScope          = "openid profile email offline_access"
	codexTokenURL       = "https://auth.openai.com/oauth/token"
	codexUsageURL       = "https://chatgpt.com/backend-api/wham/usage"
	codexRefreshTimeout = 25 * time.Second
)

// ParsedAuth 从任意 JSON 中解析出的单个账号 token 组
type ParsedAuth struct {
	AccessToken  string
	IDToken      string
	RefreshToken string
	AccountID    string
}

// ImportInput 从 JSON 文本 / 文件批量导入
type ImportInput struct {
	JSONText    string // 直接粘贴的内容（auth.json / sub2api / CPA / 顶层数组都行）
	DefaultTags []string
	Note        string
	AliasPrefix string // 可选，多账号时默认 alias 用 prefix + 序号
}

// ImportResult 导入结果汇总
type ImportResult struct {
	Created   int
	Skipped   int
	Failed    int
	Accounts  []models.CodexAccount
	LastError string
}

// Import 解析任意支持格式，逐个写入 codex_accounts；account_id 重复时跳过。
func (s *CodexAccountService) Import(input ImportInput) (*ImportResult, error) {
	text := strings.TrimSpace(input.JSONText)
	if text == "" {
		return nil, errors.New("invalid_input")
	}
	parsed, err := ParseAnyAuth(text)
	if err != nil {
		return nil, err
	}
	result := &ImportResult{Accounts: make([]models.CodexAccount, 0, len(parsed))}
	now := time.Now()
	for i, p := range parsed {
		if strings.TrimSpace(p.AccessToken) == "" || strings.TrimSpace(p.RefreshToken) == "" {
			result.Failed++
			continue
		}
		accountID := strings.TrimSpace(p.AccountID)
		if accountID == "" {
			if cid, _ := AccountIDFromAccessToken(p.AccessToken); cid != "" {
				accountID = cid
			}
		}
		// account_id 冲突 → 跳过（已存在）
		if accountID != "" {
			existing, _ := s.repo.GetByAccountID(accountID)
			if existing != nil {
				result.Skipped++
				continue
			}
		}
		acc := &models.CodexAccount{
			AccessToken:  p.AccessToken,
			IDToken:      p.IDToken,
			RefreshToken: p.RefreshToken,
			AccountID:    accountID,
			Status:       models.CodexAccountStatusOK,
			Note:         strings.TrimSpace(input.Note),
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if input.DefaultTags != nil {
			acc.TagsJSON = models.StringArray(input.DefaultTags)
		}
		// 从 JWT 中抽取 plan / email / exp / subscription
		applyJWTClaims(acc)
		// 默认 alias：email > "Codex #i"
		if strings.TrimSpace(input.AliasPrefix) != "" {
			acc.Alias = strings.TrimSpace(input.AliasPrefix) + " " + strconv.Itoa(i+1)
		} else if acc.Email != "" {
			acc.Alias = acc.Email
		} else {
			acc.Alias = "Codex " + strconv.Itoa(i+1)
		}
		// 过期则标 needs_refresh
		if acc.AccessExp > 0 && acc.AccessExp < now.Unix() {
			acc.Status = models.CodexAccountStatusNeedsRefresh
		}
		if err := s.repo.Create(acc); err != nil {
			result.Failed++
			result.LastError = err.Error()
			continue
		}
		result.Accounts = append(result.Accounts, *acc)
		result.Created++
	}
	return result, nil
}

// List 列表查询
func (s *CodexAccountService) List(filter repository.CodexAccountListFilter) ([]models.CodexAccount, int64, error) {
	return s.repo.List(filter)
}

// UpdateMetaInput 改 alias / tags / note / sold
type UpdateMetaInput struct {
	Alias *string
	Tags  *[]string
	Note  *string
	Sold  *bool
}

// UpdateMeta 仅改元数据，不动 token
func (s *CodexAccountService) UpdateMeta(id uint, in UpdateMetaInput) (*models.CodexAccount, error) {
	if id == 0 {
		return nil, errors.New("invalid_id")
	}
	acc, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, errors.New("not_found")
	}
	fields := make(map[string]interface{})
	if in.Alias != nil {
		fields["alias"] = strings.TrimSpace(*in.Alias)
	}
	if in.Tags != nil {
		fields["tags_json"] = models.StringArray(*in.Tags)
	}
	if in.Note != nil {
		fields["note"] = strings.TrimSpace(*in.Note)
	}
	if in.Sold != nil {
		fields["sold"] = *in.Sold
		if *in.Sold {
			now := time.Now()
			fields["sold_at"] = &now
		} else {
			fields["sold_at"] = nil
		}
	}
	fields["updated_at"] = time.Now()
	if err := s.repo.UpdateFields(id, fields); err != nil {
		return nil, err
	}
	return s.repo.GetByID(id)
}

// SetStatus 改状态（用于 banned 标记 / 取消）。同步维护 banned_at / ban_fail_count：
//   - 改成 banned：若 banned_at 为空，写入当前时间；保留计数（用户手动标记不重置）
//   - 改成 ok：banned_at 清空，ban_fail_count 归零
func (s *CodexAccountService) SetStatus(id uint, status string) (*models.CodexAccount, error) {
	switch status {
	case models.CodexAccountStatusOK, models.CodexAccountStatusNeedsRefresh,
		models.CodexAccountStatusBanned, models.CodexAccountStatusInvalid:
		// ok
	default:
		return nil, errors.New("invalid_status")
	}
	now := time.Now()
	fields := map[string]interface{}{
		"status":     status,
		"updated_at": now,
	}
	if status == models.CodexAccountStatusBanned {
		if acc, _ := s.repo.GetByID(id); acc != nil && acc.BannedAt == nil {
			fields["banned_at"] = &now
		}
	} else if status == models.CodexAccountStatusOK {
		fields["banned_at"] = nil
		fields["ban_fail_count"] = 0
	}
	if err := s.repo.UpdateFields(id, fields); err != nil {
		return nil, err
	}
	return s.repo.GetByID(id)
}

// Delete 软删除
func (s *CodexAccountService) Delete(id uint) error {
	return s.repo.Delete(id)
}

// RefreshToken 调 OpenAI /oauth/token 用 refresh_token 续签 access_token。
// 注意：**不修改 status 列**——status 由 FetchUsage 统一管理（401 = banned）。
// 本函数只动 token、时间戳和 last_refresh_error。
func (s *CodexAccountService) RefreshToken(ctx context.Context, id uint) (*models.CodexAccount, error) {
	acc, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, errors.New("not_found")
	}
	if strings.TrimSpace(acc.RefreshToken) == "" {
		return nil, errors.New("missing_refresh_token")
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", codexClientID)
	form.Set("refresh_token", acc.RefreshToken)
	form.Set("scope", codexScope)

	rctx, cancel := context.WithTimeout(ctx, codexRefreshTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(rctx, http.MethodPost, codexTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "dujiao-next/codex-pool")

	client := s.httpClient()
	resp, err := client.Do(req)
	if err != nil {
		_ = s.repo.UpdateFields(id, map[string]interface{}{
			"last_refresh_error": "network: " + err.Error(),
			"updated_at":         time.Now(),
		})
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode/100 != 2 {
		// 任何非 2xx 都只记录错误，不动 status；status 由 FetchUsage 统一决定
		_ = s.repo.UpdateFields(id, map[string]interface{}{
			"last_refresh_error": fmt.Sprintf("%d: %s", resp.StatusCode, truncate(string(body), 256)),
			"updated_at":         time.Now(),
		})
		updated, _ := s.repo.GetByID(id)
		return updated, fmt.Errorf("refresh failed: %d", resp.StatusCode)
	}

	var tr struct {
		AccessToken  string `json:"access_token"`
		IDToken      string `json:"id_token,omitempty"`
		RefreshToken string `json:"refresh_token,omitempty"`
		ExpiresIn    int64  `json:"expires_in,omitempty"`
		TokenType    string `json:"token_type,omitempty"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, err
	}
	if strings.TrimSpace(tr.AccessToken) == "" {
		return nil, errors.New("empty_access_token")
	}
	now := time.Now()
	// 判断 AT / RT 是否真的变化（OpenAI 不一定返回新 RT）
	atChanged := tr.AccessToken != acc.AccessToken
	rtChanged := tr.RefreshToken != "" && tr.RefreshToken != acc.RefreshToken
	acc.AccessToken = tr.AccessToken
	if tr.IDToken != "" {
		acc.IDToken = tr.IDToken
	}
	if tr.RefreshToken != "" {
		acc.RefreshToken = tr.RefreshToken
	}
	applyJWTClaims(acc) // 从新 token 中提取 plan/email/exp/subscription
	acc.LastRefreshAt = &now
	acc.LastRefreshError = ""
	if atChanged {
		acc.LastATUpdatedAt = &now
	}
	if rtChanged {
		acc.LastRTUpdatedAt = &now
	}
	acc.UpdatedAt = now
	// 不修改 acc.Status
	if err := s.repo.Update(acc); err != nil {
		return nil, err
	}
	return acc, nil
}

// FetchUsage 拉一次额度并更新到 codex_accounts 行。
// 调用 GET https://chatgpt.com/backend-api/wham/usage 带 Authorization Bearer。
//
// status 列规则（FetchUsage 是 status 的唯一权威写入路径）：
//   - 最终 200            → status = ok，ban_fail_count = 0，banned_at = null
//   - 最终 401            → status = banned，ban_fail_count += 1，banned_at = 首次封时戳（已有则保留）
//                            （401 时已经自动 refresh + 重试一次，仍 401 才会落到这里）
//   - 其它失败 / 网络层   → status 不变，只记录 last_usage_error
func (s *CodexAccountService) FetchUsage(ctx context.Context, id uint) (*models.CodexAccount, error) {
	acc, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, errors.New("not_found")
	}
	snap, status, err := s.callUsage(ctx, acc.AccessToken)
	if status == http.StatusUnauthorized {
		// 401 → 顺手用 refresh_token 续一下再试
		if _, rErr := s.RefreshToken(ctx, id); rErr == nil {
			if updated, err2 := s.repo.GetByID(id); err2 == nil && updated != nil {
				acc = updated
			}
			snap, status, err = s.callUsage(ctx, acc.AccessToken)
		}
	}
	now := time.Now()
	if err != nil || status/100 != 2 {
		msg := ""
		if err != nil {
			msg = err.Error()
		} else {
			msg = fmt.Sprintf("status %d", status)
		}
		updates := map[string]interface{}{
			"last_usage_error": truncate(msg, 256),
			"updated_at":       now,
		}
		// 401 = banned（用户规则：拉额度 401 即账号被封）
		if status == http.StatusUnauthorized {
			updates["status"] = models.CodexAccountStatusBanned
			updates["ban_fail_count"] = acc.BanFailCount + 1
			// banned_at 只在首次命中时写，保留"被封时间"语义；已存在则不动
			if acc.BannedAt == nil {
				updates["banned_at"] = &now
			}
		}
		_ = s.repo.UpdateFields(id, updates)
		return acc, fmt.Errorf("usage_failed: %s", msg)
	}
	fields := map[string]interface{}{
		"primary_used_percent":      snap.PrimaryPercent,
		"primary_limit_window_sec":  snap.PrimaryWindow,
		"primary_reset_at":          snap.PrimaryResetAt,
		"secondary_used_percent":    snap.SecondaryPercent,
		"secondary_limit_window_sec": snap.SecondaryWindow,
		"secondary_reset_at":        snap.SecondaryResetAt,
		"last_usage_at":             &now,
		"last_usage_error":          "",
		"status":                    models.CodexAccountStatusOK, // 200 = 活，刷回 ok（覆盖之前可能被标的 banned）
		"ban_fail_count":            0,                            // 恢复时清零
		"banned_at":                 nil,                          // 同步清掉"被封时间"
		"updated_at":                now,
	}
	if v := strings.TrimSpace(snap.PlanType); v != "" {
		fields["plan"] = v
	}
	if err := s.repo.UpdateFields(id, fields); err != nil {
		return nil, err
	}
	return s.repo.GetByID(id)
}

// usageSnapshot 内部携带 wham/usage 返回的扁平数据
type usageSnapshot struct {
	PlanType         string
	Allowed          bool
	LimitReached     bool
	PrimaryPercent   float64
	PrimaryWindow    int64
	PrimaryResetAt   int64
	SecondaryPercent float64
	SecondaryWindow  int64
	SecondaryResetAt int64
}

func (s *CodexAccountService) callUsage(ctx context.Context, accessToken string) (*usageSnapshot, int, error) {
	rctx, cancel := context.WithTimeout(ctx, codexRefreshTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(rctx, http.MethodGet, codexUsageURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "dujiao-next/codex-pool")
	resp, err := s.httpClient().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return nil, resp.StatusCode, fmt.Errorf("%s", truncate(string(body), 256))
	}
	var parsed struct {
		PlanType  string `json:"plan_type"`
		RateLimit struct {
			Allowed       bool `json:"allowed"`
			LimitReached  bool `json:"limit_reached"`
			PrimaryWindow *struct {
				UsedPercent        float64 `json:"used_percent"`
				LimitWindowSeconds int64   `json:"limit_window_seconds"`
				ResetAt            int64   `json:"reset_at"`
			} `json:"primary_window"`
			SecondaryWindow *struct {
				UsedPercent        float64 `json:"used_percent"`
				LimitWindowSeconds int64   `json:"limit_window_seconds"`
				ResetAt            int64   `json:"reset_at"`
			} `json:"secondary_window"`
		} `json:"rate_limit"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, resp.StatusCode, err
	}
	snap := &usageSnapshot{
		PlanType:     parsed.PlanType,
		Allowed:      parsed.RateLimit.Allowed,
		LimitReached: parsed.RateLimit.LimitReached,
	}
	if w := parsed.RateLimit.PrimaryWindow; w != nil {
		snap.PrimaryPercent = w.UsedPercent
		snap.PrimaryWindow = w.LimitWindowSeconds
		snap.PrimaryResetAt = w.ResetAt
	}
	if w := parsed.RateLimit.SecondaryWindow; w != nil {
		snap.SecondaryPercent = w.UsedPercent
		snap.SecondaryWindow = w.LimitWindowSeconds
		snap.SecondaryResetAt = w.ResetAt
	}
	return snap, resp.StatusCode, nil
}

// BatchActionResult 批量操作汇总
type BatchActionResult struct {
	OK     int      `json:"ok"`
	Failed int      `json:"failed"`
	Errors []string `json:"errors,omitempty"`
}

// BatchDelete 批量软删除（删本地行无需网络，串行即可）
func (s *CodexAccountService) BatchDelete(ids []uint) BatchActionResult {
	r := BatchActionResult{}
	for _, id := range ids {
		if err := s.repo.Delete(id); err != nil {
			r.Failed++
			r.Errors = append(r.Errors, fmt.Sprintf("%d: %s", id, err.Error()))
			continue
		}
		r.OK++
	}
	return r
}

// runBatchConcurrent 公共并发执行器。按设置里的 BatchConcurrency 限并发。
// 关键：用独立 ctx 而非 caller 的 request ctx——浏览器/nginx 超时不再把整批 OpenAI 调用切断、
// 留下"context canceled"的假阳性。
func (s *CodexAccountService) runBatchConcurrent(ids []uint, fn func(ctx context.Context, id uint) error) BatchActionResult {
	settings := s.GetSettings()
	workers := settings.BatchConcurrency
	if workers <= 0 {
		workers = 5
	}
	if workers > 32 {
		workers = 32
	}
	r := BatchActionResult{}
	if len(ids) == 0 {
		return r
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	var mu sync.Mutex
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		sem <- struct{}{}
		go func(id uint) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := fn(ctx, id); err != nil {
				mu.Lock()
				r.Failed++
				r.Errors = append(r.Errors, fmt.Sprintf("%d: %s", id, truncate(err.Error(), 200)))
				mu.Unlock()
				return
			}
			mu.Lock()
			r.OK++
			mu.Unlock()
		}(id)
	}
	wg.Wait()
	return r
}

// BatchRefresh 批量刷新 access_token（并发）
func (s *CodexAccountService) BatchRefresh(_ context.Context, ids []uint) BatchActionResult {
	return s.runBatchConcurrent(ids, func(ctx context.Context, id uint) error {
		_, err := s.RefreshToken(ctx, id)
		return err
	})
}

// BatchFetchUsage 批量拉额度（并发）
func (s *CodexAccountService) BatchFetchUsage(_ context.Context, ids []uint) BatchActionResult {
	return s.runBatchConcurrent(ids, func(ctx context.Context, id uint) error {
		_, err := s.FetchUsage(ctx, id)
		return err
	})
}

// ----- 以下为 JSON / JWT 辅助 -----

// ParseAnyAuth 把多种格式归一成 []ParsedAuth。等价于 codex-assistant/crates/core/src/codex_auth.rs:parse_any
func ParseAnyAuth(text string) ([]ParsedAuth, error) {
	var v interface{}
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		return nil, fmt.Errorf("invalid_json: %s", err.Error())
	}
	out := make([]ParsedAuth, 0, 4)
	walkAuthValue(v, &out)
	if len(out) == 0 {
		return nil, errors.New("no_codex_tokens_found")
	}
	return out, nil
}

func walkAuthValue(v interface{}, out *[]ParsedAuth) {
	switch t := v.(type) {
	case []interface{}:
		for _, x := range t {
			walkAuthValue(x, out)
		}
	case map[string]interface{}:
		// 顶层 "accounts" 数组
		if accs, ok := t["accounts"].([]interface{}); ok {
			for _, a := range accs {
				ao, _ := a.(map[string]interface{})
				if ao == nil {
					continue
				}
				target := ao
				if cred, ok := ao["credentials"].(map[string]interface{}); ok {
					target = cred
				}
				if p, ok := tryExtractAuth(target, ao); ok {
					*out = append(*out, p)
				}
			}
			return
		}
		if p, ok := tryExtractAuth(t, t); ok {
			*out = append(*out, p)
		}
	}
}

func tryExtractAuth(obj, parent map[string]interface{}) (ParsedAuth, bool) {
	// 1) 优先 tokens.{...}
	if tokensRaw, ok := obj["tokens"].(map[string]interface{}); ok {
		at := pickString(tokensRaw, "access_token", "accessToken")
		rt := pickString(tokensRaw, "refresh_token", "refreshToken")
		if at == "" || rt == "" {
			return ParsedAuth{}, false
		}
		id := pickString(tokensRaw, "id_token", "idToken")
		aid := pickString(tokensRaw, "account_id", "accountId", "chatgpt_account_id", "chatgptAccountId")
		if aid == "" {
			aid = pickString(parent, "account_id", "accountId", "chatgpt_account_id", "chatgptAccountId")
		}
		if aid == "" {
			aid, _ = AccountIDFromAccessToken(at)
		}
		return ParsedAuth{AccessToken: at, IDToken: id, RefreshToken: rt, AccountID: aid}, true
	}
	// 2) 扁平
	at := pickString(obj, "access_token", "accessToken")
	rt := pickString(obj, "refresh_token", "refreshToken")
	if at == "" || rt == "" {
		return ParsedAuth{}, false
	}
	id := pickString(obj, "id_token", "idToken")
	aid := pickString(obj, "account_id", "accountId", "chatgpt_account_id", "chatgptAccountId")
	if aid == "" {
		aid, _ = AccountIDFromAccessToken(at)
	}
	return ParsedAuth{AccessToken: at, IDToken: id, RefreshToken: rt, AccountID: aid}, true
}

func pickString(obj map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if raw, ok := obj[k]; ok {
			if s, ok := raw.(string); ok {
				v := strings.TrimSpace(s)
				if v != "" {
					return v
				}
			}
		}
	}
	return ""
}

// AccessClaims 从 access_token 抽出的 OpenAI 自定义 claims
type AccessClaims struct {
	Exp        int64        `json:"exp"`
	Iat        int64        `json:"iat"`
	OpenAIAuth openAiAuth   `json:"https://api.openai.com/auth"`
	OpenAIProf openAiProfil `json:"https://api.openai.com/profile"`
}

type openAiAuth struct {
	ChatGPTAccountID                  string `json:"chatgpt_account_id"`
	ChatGPTUserID                     string `json:"chatgpt_user_id"`
	ChatGPTPlanType                   string `json:"chatgpt_plan_type"`
	UserID                            string `json:"user_id"`
	ChatGPTSubscriptionActiveStart    string `json:"chatgpt_subscription_active_start"`
	ChatGPTSubscriptionActiveUntil    string `json:"chatgpt_subscription_active_until"`
	ChatGPTSubscriptionLastChecked    string `json:"chatgpt_subscription_last_checked"`
}

type openAiProfil struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

type idClaims struct {
	Exp        int64      `json:"exp"`
	Email      string     `json:"email"`
	Name       string     `json:"name"`
	Sub        string     `json:"sub"`
	OpenAIAuth openAiAuth `json:"https://api.openai.com/auth"`
}

// DecodeJWTPayload base64url 解码 JWT 中间段
func DecodeJWTPayload(token string) ([]byte, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) < 2 {
		return nil, errors.New("malformed_jwt")
	}
	return base64.RawURLEncoding.DecodeString(parts[1])
}

// AccountIDFromAccessToken 从 access_token 抽 chatgpt_account_id
func AccountIDFromAccessToken(token string) (string, error) {
	payload, err := DecodeJWTPayload(token)
	if err != nil {
		return "", err
	}
	var c AccessClaims
	if err := json.Unmarshal(payload, &c); err != nil {
		return "", err
	}
	return strings.TrimSpace(c.OpenAIAuth.ChatGPTAccountID), nil
}

// applyJWTClaims 用 token 中的 claims 填充账号元数据字段。
func applyJWTClaims(acc *models.CodexAccount) {
	// access_token 拿 exp + plan + account_id + user_id + email（profile）
	if payload, err := DecodeJWTPayload(acc.AccessToken); err == nil {
		var ac AccessClaims
		if json.Unmarshal(payload, &ac) == nil {
			if ac.Exp > 0 {
				acc.AccessExp = ac.Exp
			}
			if v := strings.TrimSpace(ac.OpenAIAuth.ChatGPTPlanType); v != "" {
				acc.Plan = v
			}
			if v := strings.TrimSpace(ac.OpenAIAuth.ChatGPTAccountID); v != "" && acc.AccountID == "" {
				acc.AccountID = v
			}
			if v := strings.TrimSpace(ac.OpenAIAuth.ChatGPTUserID); v != "" {
				acc.UserID = v
			}
			if v := strings.TrimSpace(ac.OpenAIProf.Email); v != "" {
				acc.Email = v
			}
			if v := parseRFC3339Sec(ac.OpenAIAuth.ChatGPTSubscriptionActiveUntil); v > 0 {
				acc.SubscriptionUntil = v
			}
		}
	}
	// id_token 兜底 email / subscription
	if acc.IDToken != "" {
		if payload, err := DecodeJWTPayload(acc.IDToken); err == nil {
			var ic idClaims
			if json.Unmarshal(payload, &ic) == nil {
				if acc.Email == "" && strings.TrimSpace(ic.Email) != "" {
					acc.Email = strings.TrimSpace(ic.Email)
				}
				if acc.SubscriptionUntil == 0 {
					if v := parseRFC3339Sec(ic.OpenAIAuth.ChatGPTSubscriptionActiveUntil); v > 0 {
						acc.SubscriptionUntil = v
					}
				}
				if acc.Plan == "" && strings.TrimSpace(ic.OpenAIAuth.ChatGPTPlanType) != "" {
					acc.Plan = strings.TrimSpace(ic.OpenAIAuth.ChatGPTPlanType)
				}
			}
		}
	}
}

func parseRFC3339Sec(v string) int64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return 0
	}
	return t.Unix()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// ProxyTestResult 单个代理测试结果。OK=false 时 Error 写明原因。
type ProxyTestResult struct {
	ProxyURL  string `json:"proxy_url"`
	OK        bool   `json:"ok"`
	IP        string `json:"ip,omitempty"`
	Country   string `json:"country,omitempty"`   // 两位 ISO 国家码，例如 "JP"、"US"、"HK"
	Region    string `json:"region,omitempty"`    // 省/州，例如 "Tokyo"、"California"
	City      string `json:"city,omitempty"`      // 城市
	Org       string `json:"org,omitempty"`       // ASN + ISP，例如 "AS123 Some ISP"
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

// TestProxy 用单个 proxy URL 探一次出口 IP 与地区。
// 不依赖 service 内部 client 缓存——直接现造一个临时 client，避免污染。
// proxyURL = "" 时走直连，便于对比"不开代理"的本机出口。
//
// 提供商策略：先打 ipinfo.io；网络层失败或 5xx 时降级到 ip-api.com。
// 双源对"一键测试全部"并发触发的临时抖动有显著容错（避免某一家短暂限流后所有代理都被判定失败）。
func (s *CodexAccountService) TestProxy(ctx context.Context, proxyURL string) *ProxyTestResult {
	proxyURL = strings.TrimSpace(proxyURL)
	res := &ProxyTestResult{ProxyURL: proxyURL}

	tr := &http.Transport{
		MaxIdleConns:        2,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	if proxyURL != "" {
		p, err := url.Parse(proxyURL)
		if err != nil {
			res.Error = "invalid proxy url: " + err.Error()
			return res
		}
		tr.Proxy = http.ProxyURL(p)
	}
	client := &http.Client{Timeout: 12 * time.Second, Transport: tr}

	start := time.Now()
	if ok := tryIPInfo(ctx, client, res); ok {
		res.LatencyMs = time.Since(start).Milliseconds()
		return res
	}
	// 主源失败 → 试 fallback。res.Error 暂留主源的报错，fallback 成功时覆盖；失败时拼接。
	primaryErr := res.Error
	res.Error = ""
	if ok := tryIPAPI(ctx, client, res); ok {
		res.LatencyMs = time.Since(start).Milliseconds()
		return res
	}
	res.LatencyMs = time.Since(start).Milliseconds()
	if res.Error == "" {
		res.Error = primaryErr
	} else if primaryErr != "" {
		res.Error = "ipinfo: " + primaryErr + " | ip-api: " + res.Error
	}
	return res
}

// tryIPInfo 调用 https://ipinfo.io/json。成功返回 true 并填充 res；失败返回 false 并把错误写到 res.Error。
func tryIPInfo(ctx context.Context, client *http.Client, res *ProxyTestResult) bool {
	rctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(rctx, http.MethodGet, "https://ipinfo.io/json", nil)
	if err != nil {
		res.Error = err.Error()
		return false
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "dujiao-next/codex-pool/proxy-test")
	resp, err := client.Do(req)
	if err != nil {
		res.Error = err.Error()
		return false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		res.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
		return false
	}
	var parsed struct {
		IP      string `json:"ip"`
		City    string `json:"city"`
		Region  string `json:"region"`
		Country string `json:"country"`
		Org     string `json:"org"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		res.Error = "parse: " + err.Error()
		return false
	}
	if strings.TrimSpace(parsed.IP) == "" {
		res.Error = "empty ip in response"
		return false
	}
	res.OK = true
	res.IP = parsed.IP
	res.City = parsed.City
	res.Region = parsed.Region
	res.Country = parsed.Country
	res.Org = parsed.Org
	return true
}

// tryIPAPI 调用 http://ip-api.com/json/?fields=...。fallback 用，字段名不同要做映射。
func tryIPAPI(ctx context.Context, client *http.Client, res *ProxyTestResult) bool {
	rctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(rctx, http.MethodGet,
		"http://ip-api.com/json/?fields=status,message,country,countryCode,regionName,city,isp,as,query", nil)
	if err != nil {
		res.Error = err.Error()
		return false
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "dujiao-next/codex-pool/proxy-test")
	resp, err := client.Do(req)
	if err != nil {
		res.Error = err.Error()
		return false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		res.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
		return false
	}
	var parsed struct {
		Status      string `json:"status"`
		Message     string `json:"message"`
		Query       string `json:"query"`
		Country     string `json:"country"`
		CountryCode string `json:"countryCode"`
		RegionName  string `json:"regionName"`
		City        string `json:"city"`
		ISP         string `json:"isp"`
		AS          string `json:"as"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		res.Error = "parse: " + err.Error()
		return false
	}
	if parsed.Status != "success" {
		res.Error = "ip-api: " + parsed.Message
		return false
	}
	res.OK = true
	res.IP = parsed.Query
	res.Country = parsed.CountryCode
	res.Region = parsed.RegionName
	res.City = parsed.City
	if parsed.AS != "" {
		res.Org = parsed.AS
	} else {
		res.Org = parsed.ISP
	}
	return true
}
