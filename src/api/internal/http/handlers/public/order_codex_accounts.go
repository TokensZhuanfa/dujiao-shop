package public

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/TokensZhuanfa/dujiao-shop/internal/constants"
	"github.com/TokensZhuanfa/dujiao-shop/internal/http/handlers/shared"
	"github.com/TokensZhuanfa/dujiao-shop/internal/http/response"
	"github.com/TokensZhuanfa/dujiao-shop/internal/models"
	"github.com/TokensZhuanfa/dujiao-shop/internal/service"

	"github.com/gin-gonic/gin"
)

// CodexAccountSummary 订单详情页 codex 账号摘要（不带 token，便于 UI 展示）
type CodexAccountSummary struct {
	ID                   uint    `json:"id"`
	Email                string  `json:"email"`
	AccountID            string  `json:"account_id"`
	Plan                 string  `json:"plan"`
	SoldAt               string  `json:"sold_at,omitempty"`
	PrimaryUsedPercent   float64 `json:"primary_used_percent"`
	PrimaryResetAt       int64   `json:"primary_reset_at"`
	SecondaryUsedPercent float64 `json:"secondary_used_percent"`
	SecondaryResetAt     int64   `json:"secondary_reset_at"`
	SubscriptionUntil    int64   `json:"subscription_until"`
}

func toSummary(a *models.CodexAccount) CodexAccountSummary {
	soldAt := ""
	if a.SoldAt != nil {
		soldAt = a.SoldAt.UTC().Format(time.RFC3339)
	}
	return CodexAccountSummary{
		ID:                   a.ID,
		Email:                a.Email,
		AccountID:            a.AccountID,
		Plan:                 a.Plan,
		SoldAt:               soldAt,
		PrimaryUsedPercent:   a.PrimaryUsedPercent,
		PrimaryResetAt:       a.PrimaryResetAt,
		SecondaryUsedPercent: a.SecondaryUsedPercent,
		SecondaryResetAt:     a.SecondaryResetAt,
		SubscriptionUntil:    a.SubscriptionUntil,
	}
}

// ListOrderCodexAccounts 已登录用户：列出订单关联的 codex 账号摘要
func (h *Handler) ListOrderCodexAccounts(c *gin.Context) {
	uid, ok := shared.GetUserID(c)
	if !ok {
		return
	}
	orderNo := strings.TrimSpace(c.Param("order_no"))
	if orderNo == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		return
	}
	order, err := h.OrderRepo.GetAnyByOrderNoAndUser(orderNo, uid)
	if err != nil || order == nil {
		shared.RespondError(c, response.CodeNotFound, "error.order_not_found", nil)
		return
	}
	h.respondCodexAccountList(c, order.OrderNo)
}

// ListGuestOrderCodexAccounts 游客版：同上
func (h *Handler) ListGuestOrderCodexAccounts(c *gin.Context) {
	email := strings.TrimSpace(c.Query("email"))
	password := strings.TrimSpace(c.Query("order_password"))
	if email == "" || password == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.guest_email_required", nil)
		return
	}
	orderNo := strings.TrimSpace(c.Param("order_no"))
	order, err := h.OrderRepo.GetAnyByOrderNoAndGuest(orderNo, email, password)
	if err != nil || order == nil {
		shared.RespondError(c, response.CodeNotFound, "error.guest_order_not_found", nil)
		return
	}
	h.respondCodexAccountList(c, order.OrderNo)
}

func (h *Handler) respondCodexAccountList(c *gin.Context, orderNo string) {
	accounts, err := h.CodexAccountRepo.ListByOrderNo(orderNo)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.codex_account_fetch_failed", err)
		return
	}
	out := make([]CodexAccountSummary, 0, len(accounts))
	for i := range accounts {
		out = append(out, toSummary(&accounts[i]))
	}
	response.Success(c, gin.H{"order_no": orderNo, "accounts": out})
}

// DownloadOrderCodexAccount 单个账号 CpaMC 或 Sub2api 格式 JSON
//   format = cpamc | sub2api
func (h *Handler) DownloadOrderCodexAccount(c *gin.Context) {
	uid, ok := shared.GetUserID(c)
	if !ok {
		return
	}
	orderNo := strings.TrimSpace(c.Param("order_no"))
	order, err := h.OrderRepo.GetAnyByOrderNoAndUser(orderNo, uid)
	if err != nil || order == nil {
		shared.RespondError(c, response.CodeNotFound, "error.order_not_found", nil)
		return
	}
	h.respondCodexAccountSingleDownload(c, order.OrderNo)
}

// DownloadGuestOrderCodexAccount 游客版本
func (h *Handler) DownloadGuestOrderCodexAccount(c *gin.Context) {
	email := strings.TrimSpace(c.Query("email"))
	password := strings.TrimSpace(c.Query("order_password"))
	if email == "" || password == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.guest_email_required", nil)
		return
	}
	orderNo := strings.TrimSpace(c.Param("order_no"))
	order, err := h.OrderRepo.GetAnyByOrderNoAndGuest(orderNo, email, password)
	if err != nil || order == nil {
		shared.RespondError(c, response.CodeNotFound, "error.guest_order_not_found", nil)
		return
	}
	h.respondCodexAccountSingleDownload(c, order.OrderNo)
}

func (h *Handler) respondCodexAccountSingleDownload(c *gin.Context, orderNo string) {
	format := strings.ToLower(strings.TrimSpace(c.Param("format")))
	idParam := strings.TrimSpace(c.Param("id"))
	id64, _ := strconv.ParseUint(idParam, 10, 64)
	if id64 == 0 || (format != "cpamc" && format != "sub2api") {
		shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		return
	}
	accounts, err := h.CodexAccountRepo.ListByOrderNo(orderNo)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.codex_account_fetch_failed", err)
		return
	}
	var target *models.CodexAccount
	for i := range accounts {
		if uint64(accounts[i].ID) == id64 {
			target = &accounts[i]
			break
		}
	}
	if target == nil {
		shared.RespondError(c, response.CodeNotFound, "error.codex_account_not_found", nil)
		return
	}
	var body []byte
	var fname string
	if format == "cpamc" {
		body, err = buildCpaMCJSON(target)
		fname = fmt.Sprintf("Codex-%s.json", safeFileSegment(target.Email))
	} else {
		body, err = buildSub2apiJSON([]models.CodexAccount{*target})
		fname = fmt.Sprintf("Sub2api-%s.json", safeFileSegment(target.Email))
	}
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.codex_account_fetch_failed", err)
		return
	}
	writeAttachment(c, fname, "application/json; charset=utf-8", body)
}

// DownloadOrderCodexAccountsArchive 全部账号打包成 zip 的下载
//   format = cpamc | sub2api
func (h *Handler) DownloadOrderCodexAccountsArchive(c *gin.Context) {
	uid, ok := shared.GetUserID(c)
	if !ok {
		return
	}
	orderNo := strings.TrimSpace(c.Param("order_no"))
	order, err := h.OrderRepo.GetAnyByOrderNoAndUser(orderNo, uid)
	if err != nil || order == nil {
		shared.RespondError(c, response.CodeNotFound, "error.order_not_found", nil)
		return
	}
	h.respondCodexAccountArchive(c, order.OrderNo)
}

// DownloadGuestOrderCodexAccountsArchive 游客版本
func (h *Handler) DownloadGuestOrderCodexAccountsArchive(c *gin.Context) {
	email := strings.TrimSpace(c.Query("email"))
	password := strings.TrimSpace(c.Query("order_password"))
	if email == "" || password == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.guest_email_required", nil)
		return
	}
	orderNo := strings.TrimSpace(c.Param("order_no"))
	order, err := h.OrderRepo.GetAnyByOrderNoAndGuest(orderNo, email, password)
	if err != nil || order == nil {
		shared.RespondError(c, response.CodeNotFound, "error.guest_order_not_found", nil)
		return
	}
	h.respondCodexAccountArchive(c, order.OrderNo)
}

func (h *Handler) respondCodexAccountArchive(c *gin.Context, orderNo string) {
	format := strings.ToLower(strings.TrimSpace(c.Param("format")))
	if format != "cpamc" && format != "sub2api" {
		shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		return
	}
	accounts, err := h.CodexAccountRepo.ListByOrderNo(orderNo)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.codex_account_fetch_failed", err)
		return
	}
	if len(accounts) == 0 {
		shared.RespondError(c, response.CodeNotFound, "error.codex_account_not_found", nil)
		return
	}

	c.Status(200)
	c.Header("Content-Type", "application/zip")
	disposition := buildContentDisposition(fmt.Sprintf("%s-codex-%s.zip", format, orderNo))
	c.Header("Content-Disposition", disposition)
	zw := zip.NewWriter(c.Writer)
	defer zw.Close()

	if format == "cpamc" {
		// 每个账号单独一个 Codex-<email>.json
		for i := range accounts {
			a := &accounts[i]
			body, err := buildCpaMCJSON(a)
			if err != nil {
				continue
			}
			name := fmt.Sprintf("Codex-%s.json", safeFileSegment(a.Email))
			w, err := zw.Create(name)
			if err != nil {
				return
			}
			_, _ = w.Write(body)
		}
		return
	}

	// sub2api：合并到单个 Sub2api 文件（同一个 wrapper.accounts 数组）
	body, err := buildSub2apiJSON(accounts)
	if err == nil {
		w, cErr := zw.Create(fmt.Sprintf("Sub2api-%s.json", orderNo))
		if cErr == nil {
			_, _ = w.Write(body)
		}
	}
	// 同时也把每个账号单独的 sub2api 写进 zip，便于按账号导入
	for i := range accounts {
		a := &accounts[i]
		single, err := buildSub2apiJSON([]models.CodexAccount{*a})
		if err != nil {
			continue
		}
		name := fmt.Sprintf("per-account/Sub2api-%s.json", safeFileSegment(a.Email))
		w, err := zw.Create(name)
		if err != nil {
			return
		}
		_, _ = w.Write(single)
	}
}

// ---- 格式构造 ----

// buildCpaMCJSON 生成 CpaMC 单账号 JSON（紧凑，一行）。
// 字段顺序与样例一致：type, email, account_id, id_token, access_token, refresh_token。
func buildCpaMCJSON(a *models.CodexAccount) ([]byte, error) {
	// 用 ordered map 实现固定字段顺序：手动 encode 一个紧凑 JSON
	type cpamc struct {
		Type         string `json:"type"`
		Email        string `json:"email"`
		AccountID    string `json:"account_id"`
		IDToken      string `json:"id_token"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	return json.Marshal(cpamc{
		Type:         "codex",
		Email:        a.Email,
		AccountID:    a.AccountID,
		IDToken:      a.IDToken,
		AccessToken:  a.AccessToken,
		RefreshToken: a.RefreshToken,
	})
}

// buildSub2apiJSON 生成 Sub2api 格式（accounts 数组 + exported_at + proxies）。
// 单账号 / 多账号都用同一个 wrapper。模型字段不足的（如 organization_id、
// chatgpt_user_id）会从 access_token 的 JWT claims 里抽。
func buildSub2apiJSON(accounts []models.CodexAccount) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	wrapper := map[string]interface{}{
		"accounts":    []interface{}{},
		"exported_at": now,
		"proxies":     []interface{}{},
	}
	accArr := make([]interface{}, 0, len(accounts))
	for i := range accounts {
		a := &accounts[i]
		// 从 AT 抽出 organization_id / chatgpt_user_id（model 没存）。
		// service.AccessClaims 暴露了 OpenAIAuth.* 字段
		userID := ""
		orgID := ""
		expiresAt := a.AccessExp
		if payload, err := service.DecodeJWTPayload(a.AccessToken); err == nil {
			var claims service.AccessClaims
			if json.Unmarshal(payload, &claims) == nil {
				userID = claims.OpenAIAuth.ChatGPTUserID
				if expiresAt == 0 {
					expiresAt = claims.Exp
				}
			}
		}
		if a.UserID != "" {
			userID = a.UserID
		}
		// id_token 里有 organization_id（顶层）
		if a.IDToken != "" {
			if payload, err := service.DecodeJWTPayload(a.IDToken); err == nil {
				var anyMap map[string]interface{}
				if json.Unmarshal(payload, &anyMap) == nil {
					if v, ok := anyMap["organization_id"].(string); ok {
						orgID = v
					}
				}
			}
		}

		credentials := map[string]interface{}{
			"access_token":        a.AccessToken,
			"chatgpt_account_id":  a.AccountID,
			"chatgpt_user_id":     userID,
			"client_id":           "app_EMoamEEZ73f0CkXaXp7hrann",
			"email":               a.Email,
			"id_token":            a.IDToken,
			"model_mapping":       defaultSub2apiModelMapping(),
			"organization_id":     orgID,
			"plan_type":           a.Plan,
			"refresh_token":       a.RefreshToken,
		}
		// expires_at 用 null 而不是 0，避免下游把"expires_at=0"识别为"已过期 1970-01-01"
		// 自动暂停账号。
		if expiresAt > 0 {
			credentials["expires_at"] = expiresAt
		} else {
			credentials["expires_at"] = nil
		}
		extra := map[string]interface{}{
			"codex_primary_over_secondary_percent": 0,
			"email":                                a.Email,
			"openai_oauth_responses_websockets_v2_enabled": false,
			"openai_oauth_responses_websockets_v2_mode":    "off",
			"privacy_mode":                             "training_off",
		}
		// 仅在 FetchUsage 真正抓到过额度时才发对应字段。0/空 = 未知，
		// 强行写 0 会让消费端误以为窗口大小=0/已重置至 1970-01-01，导致行为异常。
		if a.PrimaryLimitWindowSec > 0 {
			extra["codex_5h_window_minutes"] = a.PrimaryLimitWindowSec / 60
			extra["codex_primary_window_minutes"] = a.PrimaryLimitWindowSec / 60
			extra["codex_5h_reset_after_seconds"] = a.PrimaryLimitWindowSec
			extra["codex_primary_reset_after_seconds"] = a.PrimaryLimitWindowSec
		}
		if a.PrimaryResetAt > 0 {
			extra["codex_5h_reset_at"] = formatUnixRFC3339(a.PrimaryResetAt)
		}
		if a.PrimaryUsedPercent > 0 || a.LastUsageAt != nil {
			extra["codex_5h_used_percent"] = a.PrimaryUsedPercent
			extra["codex_primary_used_percent"] = a.PrimaryUsedPercent
		}
		if a.SecondaryLimitWindowSec > 0 {
			extra["codex_7d_window_minutes"] = a.SecondaryLimitWindowSec / 60
			extra["codex_secondary_window_minutes"] = a.SecondaryLimitWindowSec / 60
			extra["codex_7d_reset_after_seconds"] = a.SecondaryLimitWindowSec
			extra["codex_secondary_reset_after_seconds"] = a.SecondaryLimitWindowSec
		}
		if a.SecondaryResetAt > 0 {
			extra["codex_7d_reset_at"] = formatUnixRFC3339(a.SecondaryResetAt)
		}
		if a.SecondaryUsedPercent > 0 || a.LastUsageAt != nil {
			extra["codex_7d_used_percent"] = a.SecondaryUsedPercent
			extra["codex_secondary_used_percent"] = a.SecondaryUsedPercent
		}
		if a.LastUsageAt != nil {
			extra["codex_usage_updated_at"] = formatTimePtrRFC3339(a.LastUsageAt)
		}
		accArr = append(accArr, map[string]interface{}{
			"auto_pause_on_expired": true,
			"concurrency":           10,
			"credentials":           credentials,
			"extra":                 extra,
			"name":                  a.Email,
			"platform":              "openai",
			"priority":              1,
			"rate_multiplier":       1,
			"type":                  "oauth",
		})
	}
	wrapper["accounts"] = accArr
	return json.MarshalIndent(wrapper, "", "  ")
}

func formatUnixRFC3339(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

func formatTimePtrRFC3339(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// safeFileSegment 把 email 里的特殊字符替换成下划线，避免破坏路径。
// 保留字母、数字、`@.-+`。
func safeFileSegment(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '@', r == '.', r == '-', r == '+', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

func writeAttachment(c *gin.Context, filename, contentType string, body []byte) {
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", buildContentDisposition(filename))
	c.Header("Content-Length", strconv.Itoa(len(body)))
	c.Status(200)
	_, _ = c.Writer.Write(body)
}

func buildContentDisposition(filename string) string {
	// RFC 5987 编码 + ASCII fallback
	ascii := url.PathEscape(filename)
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, ascii, ascii)
}

// 别让常量被 dead-code 检查报错
var _ = constants.AutoSecretKindCodexPool

// defaultSub2apiModelMapping 与 Sub2api 样例文件保持一致的全集映射。
func defaultSub2apiModelMapping() map[string]string {
	return map[string]string{
		"chatgpt-4o-latest":        "chatgpt-4o-latest",
		"gpt-3.5-turbo":            "gpt-3.5-turbo",
		"gpt-3.5-turbo-0125":       "gpt-3.5-turbo-0125",
		"gpt-3.5-turbo-1106":       "gpt-3.5-turbo-1106",
		"gpt-3.5-turbo-16k":        "gpt-3.5-turbo-16k",
		"gpt-4":                    "gpt-4",
		"gpt-4-turbo":              "gpt-4-turbo",
		"gpt-4-turbo-preview":      "gpt-4-turbo-preview",
		"gpt-4.1":                  "gpt-4.1",
		"gpt-4.1-mini":             "gpt-4.1-mini",
		"gpt-4.1-nano":             "gpt-4.1-nano",
		"gpt-4.5-preview":          "gpt-4.5-preview",
		"gpt-4o":                   "gpt-4o",
		"gpt-4o-2024-08-06":        "gpt-4o-2024-08-06",
		"gpt-4o-2024-11-20":        "gpt-4o-2024-11-20",
		"gpt-4o-audio-preview":     "gpt-4o-audio-preview",
		"gpt-4o-mini":              "gpt-4o-mini",
		"gpt-4o-mini-2024-07-18":   "gpt-4o-mini-2024-07-18",
		"gpt-4o-realtime-preview":  "gpt-4o-realtime-preview",
		"gpt-5.2":                  "gpt-5.2",
		"gpt-5.2-2025-12-11":       "gpt-5.2-2025-12-11",
		"gpt-5.2-chat-latest":      "gpt-5.2-chat-latest",
		"gpt-5.2-pro":              "gpt-5.2-pro",
		"gpt-5.2-pro-2025-12-11":   "gpt-5.2-pro-2025-12-11",
		"gpt-5.3-codex":            "gpt-5.3-codex",
		"gpt-5.3-codex-spark":      "gpt-5.3-codex-spark",
		"gpt-5.4":                  "gpt-5.4",
		"gpt-5.4-2026-03-05":       "gpt-5.4-2026-03-05",
		"gpt-5.4-mini":             "gpt-5.4-mini",
		"gpt-5.5":                  "gpt-5.5",
		"gpt-image-1":              "gpt-image-1",
		"gpt-image-1.5":            "gpt-image-1.5",
		"gpt-image-2":              "gpt-image-2",
		"o1":                       "o1",
		"o1-mini":                  "o1-mini",
		"o1-preview":               "o1-preview",
		"o1-pro":                   "o1-pro",
		"o3":                       "o3",
		"o3-mini":                  "o3-mini",
		"o3-pro":                   "o3-pro",
		"o4-mini":                  "o4-mini",
	}
}
