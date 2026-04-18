package config

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func ConnectDB() {
	connStr := "postgresql://neondb_owner:npg_5No6cWapByir@ep-dark-mountain-am0ai3u0-pooler.c-5.us-east-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	DB = db

	log.Println("✅ DB connected")
}