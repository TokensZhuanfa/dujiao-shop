package admin

import (
	"io"
	"mime/multipart"
	"strconv"
	"strings"

	"github.com/TokensZhuanfa/dujiao-shop/internal/http/handlers/shared"
	"github.com/TokensZhuanfa/dujiao-shop/internal/http/response"
	"github.com/TokensZhuanfa/dujiao-shop/internal/models"
	"github.com/TokensZhuanfa/dujiao-shop/internal/repository"
	"github.com/TokensZhuanfa/dujiao-shop/internal/service"

	"github.com/gin-gonic/gin"
)

// ===== 号池 - Codex =====

// GetCodexAccounts 列表
func (h *Handler) GetCodexAccounts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	filter := repository.CodexAccountListFilter{
		Keyword:  strings.TrimSpace(c.Query("keyword")),
		Status:   strings.TrimSpace(c.Query("status")),
		Plan:     strings.TrimSpace(c.Query("plan")),
		Page:     page,
		PageSize: pageSize,
	}
	items, total, err := h.CodexAccountService.List(filter)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.codex_account_list_failed", err)
		return
	}
	response.SuccessWithPage(c, items, response.Pagination{
		Page:      page,
		PageSize:  pageSize,
		Total:     total,
		TotalPage: int64Ceil(total, int64(pageSize)),
	})
}

// ImportCodexAccountRequest 导入请求体（JSON 方式：单段内容）
type ImportCodexAccountRequest struct {
	JSON        string   `json:"json"`         // 粘贴 auth.json / sub2api / 顶层数组
	AliasPrefix string   `json:"alias_prefix"` // 可选，导入多账号时按 prefix + 序号命名
	Tags        []string `json:"tags"`
	Note        string   `json:"note"`
}

// ImportCodexAccounts 导入接口：JSON 粘贴 或 multipart 多文件（files[]）
func (h *Handler) ImportCodexAccounts(c *gin.Context) {
	ctype := strings.ToLower(c.GetHeader("Content-Type"))

	// 1) multipart：支持单文件 'file' + 多文件 'files'
	if strings.HasPrefix(ctype, "multipart/form-data") {
		form, err := c.MultipartForm()
		if err != nil {
			shared.RespondError(c, response.CodeBadRequest, "error.codex_account_invalid", nil)
			return
		}
		files := append([]*multipart.FileHeader{}, form.File["files"]...)
		if single := form.File["file"]; len(single) > 0 {
			files = append(files, single...)
		}
		if len(files) == 0 {
			shared.RespondError(c, response.CodeBadRequest, "error.codex_account_invalid", nil)
			return
		}
		aliasPrefix := strings.TrimSpace(c.PostForm("alias_prefix"))
		note := strings.TrimSpace(c.PostForm("note"))
		var tags []string
		if rawTags := strings.TrimSpace(c.PostForm("tags")); rawTags != "" {
			for _, t := range strings.Split(rawTags, ",") {
				if v := strings.TrimSpace(t); v != "" {
					tags = append(tags, v)
				}
			}
		}
		agg := struct {
			Created  int                    `json:"created"`
			Skipped  int                    `json:"skipped"`
			Failed   int                    `json:"failed"`
			PerFile  []map[string]interface{} `json:"per_file"`
		}{PerFile: make([]map[string]interface{}, 0, len(files))}
		for _, fh := range files {
			f, err := fh.Open()
			if err != nil {
				agg.Failed++
				agg.PerFile = append(agg.PerFile, map[string]interface{}{"file": fh.Filename, "error": err.Error()})
				continue
			}
			buf, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				agg.Failed++
				agg.PerFile = append(agg.PerFile, map[string]interface{}{"file": fh.Filename, "error": err.Error()})
				continue
			}
			result, err := h.CodexAccountService.Import(service.ImportInput{
				JSONText:    string(buf),
				DefaultTags: tags,
				Note:        note,
				AliasPrefix: aliasPrefix,
			})
			entry := map[string]interface{}{"file": fh.Filename}
			if err != nil {
				agg.Failed++
				entry["error"] = err.Error()
			} else {
				agg.Created += result.Created
				agg.Skipped += result.Skipped
				agg.Failed += result.Failed
				entry["created"] = result.Created
				entry["skipped"] = result.Skipped
				entry["failed"] = result.Failed
			}
			agg.PerFile = append(agg.PerFile, entry)
		}
		response.Success(c, agg)
		return
	}

	// 2) JSON 单段粘贴
	var body ImportCodexAccountRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	result, err := h.CodexAccountService.Import(service.ImportInput{
		JSONText:    body.JSON,
		DefaultTags: body.Tags,
		Note:        body.Note,
		AliasPrefix: body.AliasPrefix,
	})
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.codex_account_import_failed", err)
		return
	}
	response.Success(c, gin.H{
		"created":  result.Created,
		"skipped":  result.Skipped,
		"failed":   result.Failed,
		"accounts": result.Accounts,
	})
}

// UpdateCodexAccountRequest 更新元数据（alias / tags / note / sold）
type UpdateCodexAccountRequest struct {
	Alias *string   `json:"alias"`
	Tags  *[]string `json:"tags"`
	Note  *string   `json:"note"`
	Sold  *bool     `json:"sold"`
}

// UpdateCodexAccount PATCH 更新元数据
func (h *Handler) UpdateCodexAccount(c *gin.Context) {
	id, err := shared.ParseQueryUint(c.Param("id"), true)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.codex_account_invalid", nil)
		return
	}
	var body UpdateCodexAccountRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	acc, err := h.CodexAccountService.UpdateMeta(id, service.UpdateMetaInput{
		Alias: body.Alias,
		Tags:  body.Tags,
		Note:  body.Note,
		Sold:  body.Sold,
	})
	if err != nil {
		if err.Error() == "not_found" {
			shared.RespondError(c, response.CodeNotFound, "error.codex_account_not_found", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.codex_account_update_failed", err)
		return
	}
	response.Success(c, acc)
}

// GetCodexPoolSettings 读取号池设置
func (h *Handler) GetCodexPoolSettings(c *gin.Context) {
	response.Success(c, h.CodexAccountService.GetSettings())
}

// SaveCodexPoolSettingsRequest 设置体
type SaveCodexPoolSettingsRequest struct {
	AutoRefreshTokenEnabled         bool   `json:"auto_refresh_token_enabled"`
	AutoRefreshTokenIntervalM       int    `json:"auto_refresh_token_interval_minutes"`
	AutoRefreshUsageEnabled         bool   `json:"auto_refresh_usage_enabled"`
	AutoRefreshUsageIntervalM       int    `json:"auto_refresh_usage_interval_minutes"`
	AutoRefreshUsageSkipBannedAfter int    `json:"auto_refresh_usage_skip_banned_after"`
	BatchConcurrency                int    `json:"batch_concurrency"`
	ProxyEnabled                    bool   `json:"proxy_enabled"`
	ProxyURLs                       string `json:"proxy_urls"`
	ProxyURL                        string `json:"proxy_url"` // 旧字段，仍接受
}

// SaveCodexPoolSettings 保存号池设置（不动 LastAt*）
func (h *Handler) SaveCodexPoolSettings(c *gin.Context) {
	var body SaveCodexPoolSettingsRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	curr := h.CodexAccountService.GetSettings()
	next := service.CodexPoolSettings{
		AutoRefreshTokenEnabled:         body.AutoRefreshTokenEnabled,
		AutoRefreshTokenIntervalM:       body.AutoRefreshTokenIntervalM,
		AutoRefreshUsageEnabled:         body.AutoRefreshUsageEnabled,
		AutoRefreshUsageIntervalM:       body.AutoRefreshUsageIntervalM,
		AutoRefreshUsageSkipBannedAfter: body.AutoRefreshUsageSkipBannedAfter,
		BatchConcurrency:                body.BatchConcurrency,
		ProxyEnabled:                    body.ProxyEnabled,
		ProxyURLs:                       strings.TrimSpace(body.ProxyURLs),
		ProxyURL:                        strings.TrimSpace(body.ProxyURL),
		LastAutoRefreshTokenAt:          curr.LastAutoRefreshTokenAt,
		LastAutoRefreshUsageAt:          curr.LastAutoRefreshUsageAt,
	}
	if err := h.CodexAccountService.SaveSettings(next); err != nil {
		shared.RespondError(c, response.CodeInternal, "error.codex_account_update_failed", err)
		return
	}
	response.Success(c, next)
}

// TestCodexProxyRequest 代理测试请求
type TestCodexProxyRequest struct {
	ProxyURL string `json:"proxy_url"` // 空 = 直连基线
}

// TestCodexProxy 用 ipinfo.io/json 探一次出口 IP 与地区
func (h *Handler) TestCodexProxy(c *gin.Context) {
	var body TestCodexProxyRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	result := h.CodexAccountService.TestProxy(c.Request.Context(), body.ProxyURL)
	response.Success(c, result)
}

// SetCodexAccountStatusRequest 改状态
type SetCodexAccountStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// SetCodexAccountStatus 改状态
func (h *Handler) SetCodexAccountStatus(c *gin.Context) {
	id, err := shared.ParseQueryUint(c.Param("id"), true)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.codex_account_invalid", nil)
		return
	}
	var body SetCodexAccountStatusRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	switch body.Status {
	case models.CodexAccountStatusOK, models.CodexAccountStatusNeedsRefresh,
		models.CodexAccountStatusBanned, models.CodexAccountStatusInvalid:
	default:
		shared.RespondError(c, response.CodeBadRequest, "error.codex_account_invalid", nil)
		return
	}
	acc, err := h.CodexAccountService.SetStatus(id, body.Status)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.codex_account_update_failed", err)
		return
	}
	response.Success(c, acc)
}

// DeleteCodexAccount 删除账号
func (h *Handler) DeleteCodexAccount(c *gin.Context) {
	id, err := shared.ParseQueryUint(c.Param("id"), true)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.codex_account_invalid", nil)
		return
	}
	if err := h.CodexAccountService.Delete(id); err != nil {
		shared.RespondError(c, response.CodeInternal, "error.codex_account_delete_failed", err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// RefreshCodexAccount 用 refresh_token 续签 access_token
func (h *Handler) RefreshCodexAccount(c *gin.Context) {
	id, err := shared.ParseQueryUint(c.Param("id"), true)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.codex_account_invalid", nil)
		return
	}
	acc, err := h.CodexAccountService.RefreshToken(c.Request.Context(), id)
	if err != nil {
		// 即便 err 非空（如 401/403 已记录），还是把当前账号最新状态读回
		if acc != nil {
			response.Success(c, acc)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.codex_account_refresh_failed", err)
		return
	}
	response.Success(c, acc)
}

// FetchCodexAccountUsage 拉一次额度并写回快照
func (h *Handler) FetchCodexAccountUsage(c *gin.Context) {
	id, err := shared.ParseQueryUint(c.Param("id"), true)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.codex_account_invalid", nil)
		return
	}
	acc, err := h.CodexAccountService.FetchUsage(c.Request.Context(), id)
	if err != nil {
		if acc != nil {
			response.Success(c, acc)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.codex_account_usage_failed", err)
		return
	}
	response.Success(c, acc)
}

// BatchCodexAccountRequest 批量操作
type BatchCodexAccountRequest struct {
	IDs    []uint `json:"ids" binding:"required"`
	Action string `json:"action" binding:"required"` // delete / refresh / usage
}

// BatchCodexAccount 批量删除 / 刷新 / 拉额度
func (h *Handler) BatchCodexAccount(c *gin.Context) {
	var body BatchCodexAccountRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	if len(body.IDs) == 0 {
		shared.RespondError(c, response.CodeBadRequest, "error.codex_account_invalid", nil)
		return
	}
	// 防护：单次批量上限 500，避免一次拉爆所有代理 + 30 分钟超时锁住资源
	if len(body.IDs) > 500 {
		shared.RespondError(c, response.CodeBadRequest, "error.codex_account_invalid", nil)
		return
	}
	var result service.BatchActionResult
	switch body.Action {
	case "delete":
		result = h.CodexAccountService.BatchDelete(body.IDs)
	case "refresh":
		result = h.CodexAccountService.BatchRefresh(c.Request.Context(), body.IDs)
	case "usage":
		result = h.CodexAccountService.BatchFetchUsage(c.Request.Context(), body.IDs)
	default:
		shared.RespondError(c, response.CodeBadRequest, "error.codex_account_invalid", nil)
		return
	}
	response.Success(c, result)
}

func int64Ceil(total, pageSize int64) int64 {
	if pageSize <= 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}
