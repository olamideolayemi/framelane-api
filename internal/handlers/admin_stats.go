package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/models"
)

type AdminStatsHandler struct {
	DB *gorm.DB
}

type statusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type revenuePoint struct {
	Date  string `json:"date"`
	Total int    `json:"total"`
}

type topFrame struct {
	FrameID string `json:"frameId"`
	Name    string `json:"name"`
	Sold    int64  `json:"sold"`
}

// GET /v1/admin/stats
// Returns an aggregated dashboard payload.
func (h *AdminStatsHandler) Dashboard(c *gin.Context) {
	now := time.Now()
	thirtyAgo := now.Add(-30 * 24 * time.Hour)

	// Status counts
	var statuses []statusCount
	if err := h.DB.Model(&models.Order{}).
		Select("status, count(*) as count").
		Group("status").Scan(&statuses).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "stats: status counts", err.Error())
		return
	}

	// Revenue (paid orders, last 30 days, by day)
	var revenue []revenuePoint
	if err := h.DB.Model(&models.Order{}).
		Select("to_char(date_trunc('day', paid_at), 'YYYY-MM-DD') as date, sum(total) as total").
		Where("paid_at IS NOT NULL AND paid_at >= ?", thirtyAgo).
		Group("date").Order("date ASC").Scan(&revenue).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "stats: revenue", err.Error())
		return
	}

	// Top frames (by quantity sold, last 30 days)
	var top []topFrame
	if err := h.DB.Table("order_items").
		Select("order_items.frame_id, frames.name, sum(order_items.quantity) as sold").
		Joins("JOIN frames ON frames.id = order_items.frame_id").
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.created_at >= ?", thirtyAgo).
		Group("order_items.frame_id, frames.name").
		Order("sold DESC").Limit(5).Scan(&top).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "stats: top frames", err.Error())
		return
	}

	// User counts
	var userTotal, userNew int64
	_ = h.DB.Model(&models.User{}).Count(&userTotal).Error
	_ = h.DB.Model(&models.User{}).Where("created_at >= ?", thirtyAgo).Count(&userNew).Error

	// Lifetime revenue
	var lifetime int64
	_ = h.DB.Model(&models.Order{}).Where("paid_at IS NOT NULL").
		Select("COALESCE(SUM(total),0)").Scan(&lifetime).Error

	respondSuccess(c, http.StatusOK, gin.H{
		"statusCounts":   statuses,
		"revenueByDay":   revenue,
		"topFrames":      top,
		"users":          gin.H{"total": userTotal, "new30d": userNew},
		"lifetimeRevenue": lifetime,
		"window":         "30d",
	}, nil)
}
