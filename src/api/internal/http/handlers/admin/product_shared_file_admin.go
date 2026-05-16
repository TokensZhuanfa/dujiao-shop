package admin

import (
	"errors"

	"github.com/TokensZhuanfa/dujiao-shop/internal/http/handlers/shared"
	"github.com/TokensZhuanfa/dujiao-shop/internal/http/response"
	"github.com/TokensZhuanfa/dujiao-shop/internal/service"

	"github.com/gin-gonic/gin"
)

// UploadProductSharedFile 上传/替换商品的共享文件型卡密。
//
// Path:  POST /admin/products/:id/shared-file
// Form:  file (multipart, single file)
//
// 副作用：把 Product.AutoSecretKind 切换为 file_shared 并保存元数据。
func (h *Handler) UploadProductSharedFile(c *gin.Context) {
	productID, err := shared.ParseQueryUint(c.Param("id"), true)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.product_invalid", nil)
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.card_secret_invalid", nil)
		return
	}

	product, err := h.ProductService.SetSharedFile(productID, file)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCardSecretInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.card_secret_invalid", nil)
		case errors.Is(err, service.ErrProductNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.product_not_found", nil)
		case errors.Is(err, service.ErrProductFetchFailed):
			shared.RespondError(c, response.CodeInternal, "error.product_fetch_failed", err)
		case errors.Is(err, service.ErrProductUpdateFailed):
			shared.RespondError(c, response.CodeInternal, "error.product_update_failed", err)
		default:
			shared.RespondError(c, response.CodeInternal, "error.card_secret_import_failed", err)
		}
		return
	}

	response.Success(c, gin.H{
		"product_id":                product.ID,
		"auto_secret_kind":          product.AutoSecretKind,
		"shared_file_original_name": product.SharedFileOriginalName,
		"shared_file_size":          product.SharedFileSize,
		"shared_file_mime_type":     product.SharedFileMimeType,
	})
}

// DeleteProductSharedFile 删除商品的共享文件型卡密，AutoSecretKind 回到 text。
//
// Path: DELETE /admin/products/:id/shared-file
func (h *Handler) DeleteProductSharedFile(c *gin.Context) {
	productID, err := shared.ParseQueryUint(c.Param("id"), true)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.product_invalid", nil)
		return
	}
	product, err := h.ProductService.ClearSharedFile(productID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCardSecretInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.card_secret_invalid", nil)
		case errors.Is(err, service.ErrProductNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.product_not_found", nil)
		case errors.Is(err, service.ErrProductFetchFailed):
			shared.RespondError(c, response.CodeInternal, "error.product_fetch_failed", err)
		case errors.Is(err, service.ErrProductUpdateFailed):
			shared.RespondError(c, response.CodeInternal, "error.product_update_failed", err)
		default:
			shared.RespondError(c, response.CodeInternal, "error.product_update_failed", err)
		}
		return
	}
	response.Success(c, gin.H{
		"product_id":       product.ID,
		"auto_secret_kind": product.AutoSecretKind,
	})
}
