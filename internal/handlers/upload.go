package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/olamideolayemi/framelane-api/internal/storage"
)

type UploadHandler struct{ S3 *storage.S3 }

func (h *UploadHandler) GetPresignedURL(c *gin.Context) {
	filename := c.Query("filename")
	if filename == "" { c.JSON(400, gin.H{"error":"filename required"}); return }
	url, err := h.S3.PresignPut(c, filename, "", 15*time.Minute)
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"url": url})
}
