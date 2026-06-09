package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/models"
)

type AddressHandler struct {
	DB *gorm.DB
}

type addressInput struct {
	Label      string `json:"label"`
	Recipient  string `json:"recipient" binding:"required"`
	Phone      string `json:"phone" binding:"required"`
	Line1      string `json:"line1" binding:"required"`
	Line2      string `json:"line2"`
	City       string `json:"city" binding:"required"`
	State      string `json:"state"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
	IsDefault  bool   `json:"isDefault"`
}

func (in *addressInput) trim() {
	in.Label = strings.TrimSpace(in.Label)
	in.Recipient = strings.TrimSpace(in.Recipient)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Line1 = strings.TrimSpace(in.Line1)
	in.Line2 = strings.TrimSpace(in.Line2)
	in.City = strings.TrimSpace(in.City)
	in.State = strings.TrimSpace(in.State)
	in.PostalCode = strings.TrimSpace(in.PostalCode)
	in.Country = strings.TrimSpace(in.Country)
	if in.Country == "" {
		in.Country = "Nigeria"
	}
}

// GET /v1/addresses
func (h *AddressHandler) List(c *gin.Context) {
	uid, ok := uidFromCtx(c)
	if !ok {
		return
	}
	var rows []models.Address
	if err := h.DB.Where("user_id = ?", uid).Order("is_default DESC, created_at DESC").Find(&rows).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch addresses", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, rows, nil)
}

// POST /v1/addresses
func (h *AddressHandler) Create(c *gin.Context) {
	uid, ok := uidFromCtx(c)
	if !ok {
		return
	}
	var in addressInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	in.trim()

	a := models.Address{
		ID: uuid.New(), UserID: uid,
		Label: in.Label, Recipient: in.Recipient, Phone: in.Phone,
		Line1: in.Line1, Line2: in.Line2, City: in.City, State: in.State,
		PostalCode: in.PostalCode, Country: in.Country, IsDefault: in.IsDefault,
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if a.IsDefault {
			if err := tx.Model(&models.Address{}).Where("user_id = ?", uid).Update("is_default", false).Error; err != nil {
				return err
			}
		} else {
			// If user has no addresses yet, force this one to be default.
			var count int64
			if err := tx.Model(&models.Address{}).Where("user_id = ?", uid).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				a.IsDefault = true
			}
		}
		return tx.Create(&a).Error
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create address", err.Error())
		return
	}
	respondSuccess(c, http.StatusCreated, a, nil)
}

// PUT /v1/addresses/:id
func (h *AddressHandler) Update(c *gin.Context) {
	uid, ok := uidFromCtx(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid id", nil)
		return
	}
	var a models.Address
	if err := h.DB.First(&a, "id = ? AND user_id = ?", id, uid).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "address not found", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to fetch", err.Error())
		return
	}
	var in addressInput
	if err := c.ShouldBindJSON(&in); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	in.trim()
	a.Label = in.Label
	a.Recipient = in.Recipient
	a.Phone = in.Phone
	a.Line1 = in.Line1
	a.Line2 = in.Line2
	a.City = in.City
	a.State = in.State
	a.PostalCode = in.PostalCode
	a.Country = in.Country

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if in.IsDefault && !a.IsDefault {
			if err := tx.Model(&models.Address{}).Where("user_id = ?", uid).Update("is_default", false).Error; err != nil {
				return err
			}
			a.IsDefault = true
		}
		return tx.Save(&a).Error
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update address", err.Error())
		return
	}
	respondSuccess(c, http.StatusOK, a, nil)
}

// DELETE /v1/addresses/:id
func (h *AddressHandler) Delete(c *gin.Context) {
	uid, ok := uidFromCtx(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid id", nil)
		return
	}
	result := h.DB.Where("id = ? AND user_id = ?", id, uid).Delete(&models.Address{})
	if result.Error != nil {
		if isForeignKeyConstraintError(result.Error) {
			respondError(c, http.StatusBadRequest, "address is in use by an existing order", nil)
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to delete address", result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		respondError(c, http.StatusNotFound, "address not found", nil)
		return
	}
	respondSuccess(c, http.StatusOK, gin.H{"message": "deleted"}, nil)
}
