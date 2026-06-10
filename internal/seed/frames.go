package seed

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/models"
)

// SeedFrames inserts a default catalog of frame styles if none exist yet.
// Admins can later edit/extend these via the admin UI (POST /admin/frames).
// ImageURL is intentionally empty — admins upload real product photography
// through the admin form, which stores them in S3.
func SeedFrames(db *gorm.DB) error {
	defaults := []models.Frame{
		{ID: uuid.New(), Name: "Classic Black", Description: "Timeless matte black wood frame. Pairs with anything.", Status: "available"},
		{ID: uuid.New(), Name: "Modern White", Description: "Clean white wood frame for a minimalist look.", Status: "available"},
		{ID: uuid.New(), Name: "Walnut Wood", Description: "Rich walnut grain — warm and natural.", Status: "available"},
		{ID: uuid.New(), Name: "Oak Wood", Description: "Light oak grain. Bright and airy.", Status: "available"},
		{ID: uuid.New(), Name: "Brushed Gold", Description: "Subtle brushed gold metal — refined statement piece.", Status: "available"},
		{ID: uuid.New(), Name: "Brushed Silver", Description: "Cool brushed silver metal for modern interiors.", Status: "available"},
	}
	for _, f := range defaults {
		var existing models.Frame
		if err := db.First(&existing, "name = ?", f.Name).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&f).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}
	return nil
}
