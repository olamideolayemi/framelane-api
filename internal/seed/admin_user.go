package seed

import (
	"errors"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/olamideolayemi/framelane-api/internal/models"
)

// EnsureAdminUser creates or promotes an admin user when ADMIN_EMAIL and
// ADMIN_PASSWORD are provided. It is a no-op when either value is missing.
func EnsureAdminUser(db *gorm.DB) error {
	email := strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL")))
	password := strings.TrimSpace(os.Getenv("ADMIN_PASSWORD"))
	if email == "" || password == "" {
		return nil
	}

	name := strings.TrimSpace(os.Getenv("ADMIN_NAME"))
	phone := strings.TrimSpace(os.Getenv("ADMIN_PHONE"))
	address := strings.TrimSpace(os.Getenv("ADMIN_ADDRESS"))
	updatePassword := strings.EqualFold(strings.TrimSpace(os.Getenv("ADMIN_UPDATE_PASSWORD")), "true")

	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
		if err != nil {
			return err
		}

		user = models.User{
			Email:          email,
			Password:       string(hash),
			Name:           name,
			Phone:          phone,
			Address:        address,
			IsAdmin:        true,
			IsActive:       true,
			EmailVerified:  true,
		}
		return db.Create(&user).Error
	}

	updates := map[string]interface{}{
		"is_admin":       true,
		"email_verified": true,
	}
	if name != "" {
		updates["name"] = name
	}
	if phone != "" {
		updates["phone"] = phone
	}
	if address != "" {
		updates["address"] = address
	}
	if updatePassword {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
		if err != nil {
			return err
		}
		updates["password"] = string(hash)
	}

	return db.Model(&user).Updates(updates).Error
}
