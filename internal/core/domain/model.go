package domain

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Сущности базы данных

type City struct {
	UUID   string `gorm:"primaryKey"`
	UserID int64  `gorm:"not null;index"`
	City   string `gorm:"not null"`
}

func (e *City) BeforeCreate(_ *gorm.DB) (err error) {
	e.UUID = uuid.NewString()
	return
}

type Notification struct {
	UUID   string `gorm:"primaryKey"`
	ChatID int64  `gorm:"not null;index"`
	UserID int64  `gorm:"not null;index"`
	Time   string `gorm:"not null"`
}

func (e *Notification) BeforeCreate(_ *gorm.DB) (err error) {
	e.UUID = uuid.NewString()
	return
}
