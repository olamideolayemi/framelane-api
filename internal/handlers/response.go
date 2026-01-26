package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/olamideolayemi/framelane-api/internal/api"
)

func respondSuccess(c *gin.Context, status int, data any, meta any) {
	c.JSON(status, api.NewSuccess(data, meta))
}

func respondError(c *gin.Context, status int, message string, details any) {
	c.JSON(status, api.NewError(message, errorCodeForStatus(status), details))
}

func errorCodeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusUnprocessableEntity:
		return "unprocessable_entity"
	case http.StatusTooManyRequests:
		return "rate_limited"
	case http.StatusServiceUnavailable:
		return "service_unavailable"
	default:
		return "internal_error"
	}
}
