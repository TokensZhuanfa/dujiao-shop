package public

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/TokensZhuanfa/dujiao-shop/internal/constants"
	"github.com/TokensZhuanfa/dujiao-shop/internal/http/handlers/shared"
	"github.com/TokensZhuanfa/dujiao-shop/internal/http/response"
	"github.com/TokensZhuanfa/dujiao-shop/internal/models"

	"github.com/gin-gonic/gin"
)

// FulfillmentFileEntry 订单交付文件清单条目
type FulfillmentFileEntry struct {
	ID          string `json:"id"`           // "csid:N" 单份卡密文件；"shared:productID" 共享文件
	Filename    string `json:"filename"`     // 原始文件名（用于 UI 显示与下载文件名）
	Size        int64  `json:"size"`         // 字节
	ContentType string `json:"content_type"` // MIME（嗅探所得，仅做提示）
	Kind        string `json:"kind"`         // "per_stock" | "shared"
	OrderNo     string `json:"order_no"`     // 文件归属的（子）订单号，便于前端分组展示
}

// ListFulfillmentFiles 已登录用户列出订单的文件型交付清单。
func (h *Handler) ListFulfillmentFiles(c *gin.Context) {
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
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		return
	}
	if order == nil {
		shared.RespondError(c, response.CodeNotFound, "error.order_not_found", nil)
		return
	}
	h.respondFulfillmentFileList(c, order)
}

// ListGuestFulfillmentFiles 游客版列出订单的文件型交付清单。
func (h *Handler) ListGuestFulfillmentFiles(c *gin.Context) {
	email := strings.TrimSpace(c.Query("email"))
	password := strings.TrimSpace(c.Query("order_password"))
	if email == "" || password == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.guest_email_required", nil)
		return
	}
	orderNo := strings.TrimSpace(c.Param("order_no"))
	if orderNo == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		return
	}
	order, err := h.OrderRepo.GetAnyByOrderNoAndGuest(orderNo, email, password)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		return
	}
	if order == nil {
		shared.RespondError(c, response.CodeNotFound, "error.guest_order_not_found", nil)
		return
	}
	h.respondFulfillmentFileList(c, order)
}

// DownloadFulfillmentFile 已登录用户下载订单中的某个文件型交付。
func (h *Handler) DownloadFulfillmentFile(c *gin.Context) {
	uid, ok := shared.GetUserID(c)
	if !ok {
		return
	}
	orderNo := strings.TrimSpace(c.Param("order_no"))
	fileID := strings.TrimSpace(c.Param("id"))
	if orderNo == "" || fileID == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		return
	}
	order, err := h.OrderRepo.GetAnyByOrderNoAndUser(orderNo, uid)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		return
	}
	if order == nil {
		shared.RespondError(c, response.CodeNotFound, "error.order_not_found", nil)
		return
	}
	h.respondFulfillmentFileDownload(c, order, fileID)
}

// DownloadGuestFulfillmentFile 游客版下载文件型交付。
func (h *Handler) DownloadGuestFulfillmentFile(c *gin.Context) {
	email := strings.TrimSpace(c.Query("email"))
	password := strings.TrimSpace(c.Query("order_password"))
	if email == "" || password == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.guest_email_required", nil)
		return
	}
	orderNo := strings.TrimSpace(c.Param("order_no"))
	fileID := strings.TrimSpace(c.Param("id"))
	if orderNo == "" || fileID == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		return
	}
	order, err := h.OrderRepo.GetAnyByOrderNoAndGuest(orderNo, email, password)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		return
	}
	if order == nil {
		shared.RespondError(c, response.CodeNotFound, "error.guest_order_not_found", nil)
		return
	}
	h.respondFulfillmentFileDownload(c, order, fileID)
}

// respondFulfillmentFileList 汇总订单（含子订单）的文件型卡密 + 共享文件清单。
func (h *Handler) respondFulfillmentFileList(c *gin.Context, order *models.Order) {
	entries := make([]FulfillmentFileEntry, 0, 4)

	collectForOrder := func(o *models.Order) {
		// per-stock 文件型卡密：card_secrets 里 status=used 且 kind=file
		secrets, err := h.CardSecretRepo.ListByOrderAndStatus(o.ID, models.CardSecretStatusUsed)
		if err == nil {
			for _, s := range secrets {
				if strings.TrimSpace(s.Kind) != constants.CardSecretKindFile {
					continue
				}
				if strings.TrimSpace(s.FilePath) == "" {
					continue
				}
				entries = append(entries, FulfillmentFileEntry{
					ID:          fmt.Sprintf("csid:%d", s.ID),
					Filename:    fallbackFilename(s.OriginalFilename, "credential.bin"),
					Size:        s.FileSize,
					ContentType: s.ContentType,
					Kind:        "per_stock",
					OrderNo:     o.OrderNo,
				})
			}
		}
		// shared 文件：商品级（每个不同的 product 只贡献 1 项，无视购买数量）
		seen := make(map[uint]struct{})
		for _, item := range o.Items {
			if item.ProductID == 0 {
				continue
			}
			if _, ok := seen[item.ProductID]; ok {
				continue
			}
			seen[item.ProductID] = struct{}{}
			p, err := h.ProductRepo.GetByID(strconv.FormatUint(uint64(item.ProductID), 10))
			if err != nil || p == nil {
				continue
			}
			if strings.TrimSpace(p.AutoSecretKind) != constants.AutoSecretKindFileShared {
				continue
			}
			if strings.TrimSpace(p.SharedFilePath) == "" {
				continue
			}
			entries = append(entries, FulfillmentFileEntry{
				ID:          fmt.Sprintf("shared:%d", p.ID),
				Filename:    fallbackFilename(p.SharedFileOriginalName, "shared.bin"),
				Size:        p.SharedFileSize,
				ContentType: p.SharedFileMimeType,
				Kind:        "shared",
				OrderNo:     o.OrderNo,
			})
		}
	}

	collectForOrder(order)
	for i := range order.Children {
		collectForOrder(&order.Children[i])
	}

	response.Success(c, gin.H{"files": entries})
}

// respondFulfillmentFileDownload 解析文件 ID，做归属校验，流式输出。
func (h *Handler) respondFulfillmentFileDownload(c *gin.Context, order *models.Order, fileID string) {
	parts := strings.SplitN(fileID, ":", 2)
	if len(parts) != 2 {
		shared.RespondError(c, response.CodeBadRequest, "error.fulfillment_file_invalid", nil)
		return
	}
	switch parts[0] {
	case "csid":
		csID, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil || csID == 0 {
			shared.RespondError(c, response.CodeBadRequest, "error.fulfillment_file_invalid", nil)
			return
		}
		secret, err := h.CardSecretRepo.GetByID(uint(csID))
		if err != nil {
			shared.RespondError(c, response.CodeInternal, "error.fulfillment_fetch_failed", err)
			return
		}
		if secret == nil || strings.TrimSpace(secret.Kind) != constants.CardSecretKindFile || strings.TrimSpace(secret.FilePath) == "" {
			shared.RespondError(c, response.CodeNotFound, "error.fulfillment_not_found", nil)
			return
		}
		// 归属校验：card_secret.order_id 必须等于本订单或其子订单之一
		// 统一返回 404 而非 403，避免攻击者通过状态码区分"存在但无权限" vs "不存在"
		if !orderOwnsRef(order, secret.OrderID) {
			shared.RespondError(c, response.CodeNotFound, "error.fulfillment_not_found", nil)
			return
		}
		streamCredentialFile(c, h, secret.FilePath, secret.OriginalFilename, secret.ContentType)
	case "shared":
		pid, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil || pid == 0 {
			shared.RespondError(c, response.CodeBadRequest, "error.fulfillment_file_invalid", nil)
			return
		}
		// 归属校验：product 必须出现在本订单或其任一子订单的 items 中
		// 同样统一回 404 以避免信息泄露
		if !orderContainsProduct(order, uint(pid)) {
			shared.RespondError(c, response.CodeNotFound, "error.fulfillment_not_found", nil)
			return
		}
		p, err := h.ProductRepo.GetByID(strconv.FormatUint(pid, 10))
		if err != nil {
			shared.RespondError(c, response.CodeInternal, "error.product_fetch_failed", err)
			return
		}
		if p == nil || strings.TrimSpace(p.AutoSecretKind) != constants.AutoSecretKindFileShared || strings.TrimSpace(p.SharedFilePath) == "" {
			shared.RespondError(c, response.CodeNotFound, "error.fulfillment_not_found", nil)
			return
		}
		streamCredentialFile(c, h, p.SharedFilePath, p.SharedFileOriginalName, p.SharedFileMimeType)
	default:
		shared.RespondError(c, response.CodeBadRequest, "error.fulfillment_file_invalid", nil)
	}
}

// DownloadFulfillmentArchive 已登录用户：把订单内所有文件型交付打成 zip 一次性下载。
func (h *Handler) DownloadFulfillmentArchive(c *gin.Context) {
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
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		return
	}
	if order == nil {
		shared.RespondError(c, response.CodeNotFound, "error.order_not_found", nil)
		return
	}
	h.respondFulfillmentFileArchive(c, order)
}

// DownloadGuestFulfillmentArchive 游客版本的 zip 打包下载。
func (h *Handler) DownloadGuestFulfillmentArchive(c *gin.Context) {
	email := strings.TrimSpace(c.Query("email"))
	password := strings.TrimSpace(c.Query("order_password"))
	if email == "" || password == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.guest_email_required", nil)
		return
	}
	orderNo := strings.TrimSpace(c.Param("order_no"))
	if orderNo == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		return
	}
	order, err := h.OrderRepo.GetAnyByOrderNoAndGuest(orderNo, email, password)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		return
	}
	if order == nil {
		shared.RespondError(c, response.CodeNotFound, "error.guest_order_not_found", nil)
		return
	}
	h.respondFulfillmentFileArchive(c, order)
}

// archiveEntry 描述 zip 内一份文件：私有存储的相对路径 + 写入 zip 时使用的展示文件名
type archiveEntry struct {
	relPath     string
	displayName string
}

// collectFulfillmentFileEntries 汇总订单（含子单）的所有文件型交付项。
func (h *Handler) collectFulfillmentFileEntries(order *models.Order) []archiveEntry {
	entries := make([]archiveEntry, 0, 4)
	collectForOrder := func(o *models.Order) {
		secrets, err := h.CardSecretRepo.ListByOrderAndStatus(o.ID, models.CardSecretStatusUsed)
		if err == nil {
			for _, s := range secrets {
				if strings.TrimSpace(s.Kind) != constants.CardSecretKindFile {
					continue
				}
				if strings.TrimSpace(s.FilePath) == "" {
					continue
				}
				entries = append(entries, archiveEntry{
					relPath:     s.FilePath,
					displayName: fallbackFilename(s.OriginalFilename, fmt.Sprintf("credential-%d.bin", s.ID)),
				})
			}
		}
		seen := make(map[uint]struct{})
		for _, item := range o.Items {
			if item.ProductID == 0 {
				continue
			}
			if _, ok := seen[item.ProductID]; ok {
				continue
			}
			seen[item.ProductID] = struct{}{}
			p, err := h.ProductRepo.GetByID(strconv.FormatUint(uint64(item.ProductID), 10))
			if err != nil || p == nil {
				continue
			}
			if strings.TrimSpace(p.AutoSecretKind) != constants.AutoSecretKindFileShared {
				continue
			}
			if strings.TrimSpace(p.SharedFilePath) == "" {
				continue
			}
			entries = append(entries, archiveEntry{
				relPath:     p.SharedFilePath,
				displayName: fallbackFilename(p.SharedFileOriginalName, fmt.Sprintf("shared-%d.bin", p.ID)),
			})
		}
	}
	collectForOrder(order)
	for i := range order.Children {
		collectForOrder(&order.Children[i])
	}
	return entries
}

// respondFulfillmentFileArchive 流式输出 zip。Store 模式不压缩（绝大多数交付文件已是压缩格式）。
func (h *Handler) respondFulfillmentFileArchive(c *gin.Context, order *models.Order) {
	if h.CredentialFileService == nil {
		shared.RespondError(c, response.CodeInternal, "error.fulfillment_fetch_failed", errors.New("credential file service unavailable"))
		return
	}
	entries := h.collectFulfillmentFileEntries(order)
	if len(entries) == 0 {
		shared.RespondError(c, response.CodeNotFound, "error.fulfillment_not_found", nil)
		return
	}

	// HTTP 头先于 zip 字节流写出
	zipName := fmt.Sprintf("fulfillment-%s.zip", strings.TrimSpace(order.OrderNo))
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", contentDispositionAttachment(zipName))
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(200)

	zw := zip.NewWriter(c.Writer)
	defer zw.Close()

	nameCounts := make(map[string]int)
	for _, e := range entries {
		display := zipDedupeName(e.displayName, nameCounts)

		f, _, err := h.CredentialFileService.OpenForRead(e.relPath)
		if err != nil {
			// 某个文件丢失：写一个占位说明，避免整包失败
			placeholder, _ := zw.CreateHeader(&zip.FileHeader{
				Name:   display + ".missing.txt",
				Method: zip.Store,
			})
			if placeholder != nil {
				_, _ = placeholder.Write([]byte("file unavailable on server"))
			}
			continue
		}
		w, err := zw.CreateHeader(&zip.FileHeader{
			Name:   display,
			Method: zip.Store, // 不压缩——大多数交付文件已是压缩格式，CPU 不划算
		})
		if err != nil {
			f.Close()
			return
		}
		_, _ = io.Copy(w, f)
		f.Close()
	}
}

// zipDedupeName 同名文件追加 -2、-3 等数字后缀以避免 zip 冲突。
func zipDedupeName(name string, counts map[string]int) string {
	counts[name]++
	if counts[name] == 1 {
		return name
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	return fmt.Sprintf("%s-%d%s", base, counts[name], ext)
}

func streamCredentialFile(c *gin.Context, h *Handler, relPath, filename, contentType string) {
	if h.CredentialFileService == nil {
		shared.RespondError(c, response.CodeInternal, "error.fulfillment_fetch_failed", errors.New("credential file service unavailable"))
		return
	}
	f, info, err := h.CredentialFileService.OpenForRead(relPath)
	if err != nil {
		shared.RespondError(c, response.CodeNotFound, "error.fulfillment_not_found", err)
		return
	}
	defer f.Close()

	disp := contentDispositionAttachment(fallbackFilename(filename, "credential.bin"))
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", disp)
	c.Header("Content-Length", strconv.FormatInt(info.Size(), 10))
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(200)
	_, _ = io.Copy(c.Writer, f)
}

func orderOwnsRef(order *models.Order, ref *uint) bool {
	if ref == nil || *ref == 0 {
		return false
	}
	if *ref == order.ID {
		return true
	}
	for _, child := range order.Children {
		if *ref == child.ID {
			return true
		}
	}
	return false
}

func orderContainsProduct(order *models.Order, productID uint) bool {
	if productID == 0 {
		return false
	}
	for _, item := range order.Items {
		if item.ProductID == productID {
			return true
		}
	}
	for _, child := range order.Children {
		for _, item := range child.Items {
			if item.ProductID == productID {
				return true
			}
		}
	}
	return false
}

func fallbackFilename(name, fallback string) string {
	v := strings.TrimSpace(name)
	if v == "" {
		return fallback
	}
	return v
}

// contentDispositionAttachment 按 RFC 5987 编码非 ASCII 文件名，老浏览器看 filename，新浏览器看 filename*。
func contentDispositionAttachment(filename string) string {
	asciiSafe := mime.QEncoding.Encode("utf-8", filename)
	encoded := url.PathEscape(filename)
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, asciiSafe, encoded)
}
