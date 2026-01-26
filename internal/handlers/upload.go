package handlers

import (
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/olamideolayemi/framelane-api/internal/storage"
)

type UploadHandler struct{ S3 *storage.S3 }

func (h *UploadHandler) GetPresignedURL(c *gin.Context) {
	if h.S3 == nil {
		respondError(c, http.StatusInternalServerError, "storage is not configured", nil)
		return
	}

	filename := strings.TrimSpace(c.Query("filename"))
	if filename == "" {
		respondError(c, http.StatusBadRequest, "filename required", nil)
		return
	}

	objectName := strings.TrimPrefix(path.Clean(filename), "/")
	if objectName == "." || objectName == "" || strings.HasPrefix(objectName, "..") {
		respondError(c, http.StatusBadRequest, "invalid filename", nil)
		return
	}

	contentType := strings.TrimSpace(c.Query("content_type"))
	if contentType == "" {
		contentType = strings.TrimSpace(c.Query("contentType"))
	}

	url, err := h.S3.PresignPut(c, objectName, contentType, 15*time.Minute)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create presigned url", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"url": url}, nil)
}
