package model

import (
	"fmt"

	"gorm.io/gorm"
)

// Employee -
type Employee struct {
	gorm.Model
	Name   string `gorm:"type:varchar(32)"`
	Email  string `gorm:"type:varchar(64);not null;unique;index"`
	Number uint   `gorm:"not null;unique;index"`
}

// InitEmployee -
func InitEmployee(db *gorm.DB) {
	// 迁移 schema
	db.AutoMigrate(&Employee{})
}

// BeforeSave -
func (emp *Employee) BeforeSave(db *gorm.DB) (err error) {
	fmt.Println("before save")
	return
}

// BeforeCreate -
func (emp *Employee) BeforeCreate(db *gorm.DB) (err error) {
	fmt.Println("before create")
	return
}

// AfterCreate -
func (emp *Employee) AfterCreate(db *gorm.DB) (err error) {
	fmt.Println("after create")
	return
}

// AfterSave -
func (emp *Employee) AfterSave(db *gorm.DB) (err error) {
	fmt.Println("after save")
	return
}
