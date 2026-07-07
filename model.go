package main

import "time"

type Job struct {
	ID        int       `json:"id"`
	Payload   string    `json:"payload"`
	Status    string    `json:"status"`
	Attempts  int       `json:"attempts"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateJobRequest struct {
	Payload string `json:"payload" binding:"required"`
}
