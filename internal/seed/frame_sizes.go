package seed

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/models"
)

func SeedFrameSizes(db *gorm.DB) error {
	frames := []models.FrameSize{
		{ID: uuid.New(), Name: "5x7 in", Price: 6000, Status: "available"},
		{ID: uuid.New(), Name: "6x9 in", Price: 6500, Status: "available"},
		{ID: uuid.New(), Name: "A4 (8x12 in)", Price: 7500, Status: "available"},
		{ID: uuid.New(), Name: "10x13 in", Price: 8000, Status: "available"},
		{ID: uuid.New(), Name: "11x14 in", Price: 8500, Status: "available"},
		{ID: uuid.New(), Name: "12x16 in", Price: 9500, Status: "available"},
		{ID: uuid.New(), Name: "14x18 in", Price: 10000, Status: "available"},
		{ID: uuid.New(), Name: "16x20 in", Price: 10500, Status: "available"},
		{ID: uuid.New(), Name: "16x24 in", Price: 12500, Status: "available"},
		{ID: uuid.New(), Name: "18x24 in", Price: 13500, Status: "available"},
		{ID: uuid.New(), Name: "20x24 in", Price: 15500, Status: "available"},
		{ID: uuid.New(), Name: "20x30 in", Price: 18000, Status: "available"},
		{ID: uuid.New(), Name: "21x37 in", Price: 25500, Status: "available"},
		{ID: uuid.New(), Name: "24x30 in", Price: 21500, Status: "available"},
		{ID: uuid.New(), Name: "24x36 in", Price: 22500, Status: "available"},
		{ID: uuid.New(), Name: "27x40 in", Price: 35000, Status: "available"},
		{ID: uuid.New(), Name: "30x40 in", Price: 40000, Status: "available"},
		{ID: uuid.New(), Name: "36x48 in", Price: 44500, Status: "available"},
	}

	for _, f := range frames {
		var existing models.FrameSize
		if err := db.First(&existing, "name = ?", f.Name).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Create new if not exists
				if err := db.Create(&f).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			// Update price and status if changed
			existing.Price = f.Price
			existing.Status = f.Status
			if err := db.Save(&existing).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
