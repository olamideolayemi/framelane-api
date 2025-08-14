package models

import "time"

type Order struct {
	ID        uint      `gorm:"primaryKey"`
	OrderID   string    `gorm:"uniqueIndex;size:40"` // external tracking id
	UserID    *uint     `gorm:"index"`
	UserEmail string    `gorm:"size:255"`            // for guests
	Name      string    `gorm:"size:120"`
	Phone     string    `gorm:"size:40"`
	Address   string    `gorm:"size:400"`
	Frame     string    `gorm:"size:80"`
	Size      string    `gorm:"size:80"`
	Room      string    `gorm:"size:120"`
	ImageURL  string    `gorm:"size:600"`
	Status    string    `gorm:"size:40;default:Pending"` // Pending, In Progress, Shipped
	CreatedAt time.Time
	UpdatedAt time.Time
}
