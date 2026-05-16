package admin

import (
	"errors"
	"strings"

	"github.com/TokensZhuanfa/dujiao-shop/internal/http/handlers/shared"
	"github.com/TokensZhuanfa/dujiao-shop/internal/http/response"
	"github.com/TokensZhuanfa/dujiao-shop/internal/service"

	"github.com/gin-gonic/gin"
)

// ImportCardSecretFiles 文件型卡密批量导入：multipart/form-data。
//
// Form fields:
//   - product_id (required, uint)
//   - sku_id     (optional, uint, default 0)
//   - batch_no   (optional, string)
//   - note       (optional, string)
//   - files      (required, repeated, multiple files — each becomes one card_secret)
//
// 副作用：会把 Product.AutoSecretKind 强制切换为 file_per_stock。
func (h *Handler) ImportCardSecretFiles(c *gin.Context) {
	adminID, ok := shared.GetAdminID(c)
	if !ok {
		return
	}

	productID, err := shared.ParseQueryUint(c.PostForm("product_id"), true)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.card_secret_invalid", nil)
		return
	}
	skuID, err := shared.ParseQueryUint(c.DefaultPostForm("sku_id", "0"), false)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.card_secret_invalid", nil)
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.card_secret_invalid", nil)
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		shared.RespondError(c, response.CodeBadRequest, "error.card_secret_invalid", nil)
		return
	}

	batchNo := strings.TrimSpace(c.PostForm("batch_no"))
	note := strings.TrimSpace(c.PostForm("note"))

	batch, created, err := h.CardSecretService.ImportCardSecretFiles(service.ImportCardSecretFilesInput{
		ProductID: productID,
		SKUID:     skuID,
		Files:     files,
		BatchNo:   batchNo,
		Note:      note,
		AdminID:   adminID,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProductSKURequired),
			errors.Is(err, service.ErrProductSKUInvalid),
			errors.Is(err, service.ErrCardSecretInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.card_secret_invalid", nil)
		case errors.Is(err, service.ErrProductNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.product_not_found", nil)
		case errors.Is(err, service.ErrProductFetchFailed):
			shared.RespondError(c, response.CodeInternal, "error.product_fetch_failed", err)
		case errors.Is(err, service.ErrCardSecretBatchCreateFailed):
			shared.RespondError(c, response.CodeInternal, "error.card_secret_batch_create_failed", err)
		default:
			shared.RespondError(c, response.CodeInternal, "error.card_secret_import_failed", err)
		}
		return
	}

	response.Success(c, gin.H{
		"created":  created,
		"batch_id": batch.ID,
		"batch_no": batch.BatchNo,
	})
}
