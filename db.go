package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var db *sql.DB

func initDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=5432 dbname=jobworker port=5432 sslmode=disable"
	}

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("DB open failed", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("DB connection failed", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
		id SERIAL    PRIMARY KEY,
		payload      TEXT NOT NULL,
		status       TEXT NOT NULL DEFAULT 'pending',
		attempts     INT NOT NULL DEFAULT 0,
		created_at   TIMESTAMPTZ DEFAULT NOW(),
		updated_at   TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatal("table creation failed", err)
	}

	log.Println("connected to postgres")
}

func createJob(payload string) (*Job, error) {
	var j Job
	err := db.QueryRow(
		`INSERT INTO jobs (payload, status) VALUES ($1, 'pending')
		RETURNING id, payload, status, attempts, created_at, updated_at`,
		payload,
	).Scan(&j.ID, &j.Payload, &j.Status, &j.Attempts, &j.CreatedAt, &j.UpdatedAt)
	return &j, err
}

func claimJob() (*Job, error) {
	var j Job
	err := db.QueryRow(`
		UPDATE jobs set status = 'processing', updated_at = now()
		WHERE id = (
		SELECT id FROM jobs
		WHERE status = 'pending'
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED
		LIMIT 1
		)
		RETURNING id, payload, status, attempts, created_at, updated_at
	`).Scan(&j.ID, &j.Payload, &j.Status, &j.Attempts, &j.CreatedAt, &j.UpdatedAt)
	return &j, err
}

func completeJob(id int) {
	db.Exec(
		`UPDATE jobs SET status = 'done', updated_at = NOW() WHERE id = $1`,
		id,
	)
}

func failJob(id int) {
	db.Exec(
		`UPDATE jobs SET status = 'failed', attempts = attempts + 1, updated_at = NOW() WHERE id = $1`,
		id,
	)
}

func getJob(id int) (*Job, error) {
	var j Job
	err := db.QueryRow(
		`SELECT id, payload, status, attempts, created_at, updated_at FROM jobs WHERE id = $1`,
		id,
	).Scan(&j.ID, &j.Payload, &j.Status, &j.Attempts, &j.CreatedAt, &j.UpdatedAt)
	return &j, err
}
