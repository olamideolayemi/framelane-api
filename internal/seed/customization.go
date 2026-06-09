package seed

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/models"
)

// SeedGlasses inserts a default set of glass options if not already present.
// PriceModifier is in NGN.
func SeedGlasses(db *gorm.DB) error {
	defaults := []models.Glass{
		{ID: uuid.New(), Name: "Regular Glass", Description: "Standard clear glass.", PriceModifier: 0, Status: "available"},
		{ID: uuid.New(), Name: "Non-Glare Glass", Description: "Anti-reflective glass for high-light rooms.", PriceModifier: 2500, Status: "available"},
		{ID: uuid.New(), Name: "Acrylic (Shatter-resistant)", Description: "Lightweight, shatter-resistant alternative to glass.", PriceModifier: 1500, Status: "available"},
		{ID: uuid.New(), Name: "No Glass", Description: "Open frame without glass cover.", PriceModifier: -500, Status: "available"},
	}
	for _, g := range defaults {
		var existing models.Glass
		if err := db.First(&existing, "name = ?", g.Name).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&g).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}
	return nil
}

// SeedLaminations inserts default lamination/coating options.
func SeedLaminations(db *gorm.DB) error {
	defaults := []models.Lamination{
		{ID: uuid.New(), Name: "Matte", Description: "Soft non-reflective lamination.", PriceModifier: 1000, Status: "available"},
		{ID: uuid.New(), Name: "Gloss", Description: "High-shine reflective lamination.", PriceModifier: 1000, Status: "available"},
		{ID: uuid.New(), Name: "Satin", Description: "Subtle sheen between matte and gloss.", PriceModifier: 1200, Status: "available"},
		{ID: uuid.New(), Name: "None", Description: "No lamination.", PriceModifier: 0, Status: "available"},
	}
	for _, l := range defaults {
		var existing models.Lamination
		if err := db.First(&existing, "name = ?", l.Name).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&l).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}
	return nil
}
