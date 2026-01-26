package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health reports the service status.
func Health(c *gin.Context) {
	respondSuccess(c, http.StatusOK, gin.H{"status": "ok"}, nil)
}
