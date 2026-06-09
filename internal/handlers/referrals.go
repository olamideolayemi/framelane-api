package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/models"
)

type ReferralsHandler struct {
	DB *gorm.DB
}

// GET /v1/referrals/me
// Returns the current user's referral code + wallet balance + pending count.
func (h *ReferralsHandler) Me(c *gin.Context) {
	uid, ok := uidFromCtx(c)
	if !ok {
		return
	}
	var u models.User
	if err := h.DB.Select("id", "name", "referral_code", "wallet_balance").First(&u, "id = ?", uid).Error; err != nil {
		respondError(c, http.StatusNotFound, "user not found", nil)
		return
	}
	var pending, credited int64
	_ = h.DB.Model(&models.Referral{}).Where("referrer_user_id = ? AND status = ?", uid, models.ReferralStatusPending).Count(&pending).Error
	_ = h.DB.Model(&models.Referral{}).Where("referrer_user_id = ? AND status = ?", uid, models.ReferralStatusCredited).Count(&credited).Error

	respondSuccess(c, http.StatusOK, gin.H{
		"referralCode":   u.ReferralCode,
		"walletBalance":  u.WalletBalance,
		"pendingCount":   pending,
		"creditedCount":  credited,
	}, nil)
}
