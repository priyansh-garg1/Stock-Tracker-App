package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func DBConnection(env *Env) *gorm.DB {
	uri := env.DB_URL

	db, err := gorm.Open(postgres.Open(uri), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatalf("Unable to connect to database: %e", err)
	}

	fmt.Println("Connected to the database!")

	if err := db.AutoMigrate(&Candle{}); err != nil {
		log.Fatalf("Unable to migrate: %e", err)
	}

	return db
}