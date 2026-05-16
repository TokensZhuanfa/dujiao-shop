package public

import (
	"strconv"

	"github.com/TokensZhuanfa/dujiao-shop/internal/dto"
	"github.com/TokensZhuanfa/dujiao-shop/internal/http/handlers/shared"
	"github.com/TokensZhuanfa/dujiao-shop/internal/http/response"

	"github.com/gin-gonic/gin"
)

// GetMyLoginLogs 获取当前用户登录日志
func (h *Handler) GetMyLoginLogs(c *gin.Context) {
	uid, ok := shared.GetUserID(c)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	logs, total, err := h.UserLoginLogService.ListByUser(uid, page, pageSize)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.user_login_log_fetch_failed", err)
		return
	}

	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, dto.NewLoginLogRespList(logs), pagination)
}
