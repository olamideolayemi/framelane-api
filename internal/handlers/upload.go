package handlers

import (
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/olamideolayemi/framelane-api/internal/storage"
)

type UploadHandler struct{ S3 *storage.S3 }

// allowedUploadContentTypes restricts presigned PUT URLs to image MIME types
// we actually serve. iOS HEIC is included for camera captures.
var allowedUploadContentTypes = map[string]struct{}{
	"image/jpeg": {}, "image/jpg": {}, "image/png": {},
	"image/webp": {}, "image/heic": {}, "image/heif": {},
}

func (h *UploadHandler) GetPresignedURL(c *gin.Context) {
	if h.S3 == nil {
		respondError(c, http.StatusInternalServerError, "storage is not configured", nil)
		return
	}

	rawName := strings.TrimSpace(c.Query("filename"))
	if rawName == "" {
		respondError(c, http.StatusBadRequest, "filename required", nil)
		return
	}

	contentType := strings.ToLower(strings.TrimSpace(c.Query("content_type")))
	if contentType == "" {
		contentType = strings.ToLower(strings.TrimSpace(c.Query("contentType")))
	}
	if contentType == "" {
		respondError(c, http.StatusBadRequest, "content_type required", nil)
		return
	}
	if _, ok := allowedUploadContentTypes[contentType]; !ok {
		respondError(c, http.StatusBadRequest, "unsupported content type", gin.H{"allowed": []string{
			"image/jpeg", "image/png", "image/webp", "image/heic", "image/heif",
		}})
		return
	}

	cleaned := strings.TrimPrefix(path.Clean(rawName), "/")
	if cleaned == "." || cleaned == "" || strings.HasPrefix(cleaned, "..") {
		respondError(c, http.StatusBadRequest, "invalid filename", nil)
		return
	}

	// Prefix with a UUID under a folder to prevent collisions and accidental
	// overwrites. We keep the trailing path so admins can still recognize files.
	objectName := "uploads/" + uuid.New().String() + "_" + path.Base(cleaned)

	url, err := h.S3.PresignPut(c, objectName, contentType, 15*time.Minute)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create presigned url", err.Error())
		return
	}

	// Build a best-effort public URL the frontend can store on the cart/order.
	// PUBLIC_S3_BASE_URL overrides when the bucket is fronted by a CDN.
	publicURL := buildPublicS3URL(objectName)

	respondSuccess(c, http.StatusOK, gin.H{
		"url":         url,
		"objectName":  objectName,
		"contentType": contentType,
		"publicUrl":   publicURL,
	}, nil)
}

func buildPublicS3URL(objectName string) string {
	if override := strings.TrimSpace(os.Getenv("PUBLIC_S3_BASE_URL")); override != "" {
		return strings.TrimRight(override, "/") + "/" + objectName
	}
	endpoint := strings.TrimSpace(os.Getenv("S3_ENDPOINT"))
	bucket := strings.TrimSpace(os.Getenv("S3_BUCKET"))
	if endpoint == "" || bucket == "" {
		return ""
	}
	useSSL := strings.EqualFold(os.Getenv("S3_USE_SSL"), "true") || os.Getenv("S3_USE_SSL") == "1"
	scheme := "https"
	if !useSSL {
		scheme = "http"
	}
	// Strip protocol if present so we don't double up.
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	return scheme + "://" + endpoint + "/" + bucket + "/" + objectName
}
