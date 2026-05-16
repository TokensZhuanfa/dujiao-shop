package admin

import (
	"errors"

	"github.com/TokensZhuanfa/dujiao-shop/internal/http/handlers/shared"
	"github.com/TokensZhuanfa/dujiao-shop/internal/http/response"
	"github.com/TokensZhuanfa/dujiao-shop/internal/service"

	"github.com/gin-gonic/gin"
)

// GetAffiliateSettings 获取推广返利设置
func (h *Handler) GetAffiliateSettings(c *gin.Context) {
	setting, err := h.SettingService.GetAffiliateSetting()
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.settings_fetch_failed", err)
		return
	}
	response.Success(c, setting)
}

// UpdateAffiliateSettings 更新推广返利设置
func (h *Handler) UpdateAffiliateSettings(c *gin.Context) {
	var req service.AffiliateSetting
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	setting, err := h.SettingService.UpdateAffiliateSetting(req)
	if err != nil {
		if errors.Is(err, service.ErrAffiliateConfigInvalid) {
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.settings_save_failed", err)
		return
	}
	response.Success(c, setting)
}
