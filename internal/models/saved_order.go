package models

import (
  "time"
  "gorm.io/datatypes"
)

type SavedOrder struct {
  ID        string `gorm:"primaryKey"`
  UserID    *uint
  Payload   datatypes.JSON
  ExpiresAt time.Time `gorm:"index"`
  CreatedAt time.Time
}
