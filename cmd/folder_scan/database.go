package main

import (
	"cacheDatabase"
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openDatabase(path string) error {
	// Create file database
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("open Database error:", err)
	}
	if res := db.Exec("PRAGMA foreign_keys = ON", nil); res.Error != nil {
		return fmt.Errorf("Impossible create foreign")
	}
	cacheDatabase.DB = db.Debug()
	cacheDatabase.DB.AutoMigrate(&File{})

	return nil
}
