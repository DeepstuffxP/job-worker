package main

import (
	"context"
	"database/sql"
	"log"
	"time"
)

func startWorkerPool(ctx context.Context, numWorkers int) {
	for i := 1; i <= numWorkers; i++ {
		go worker(ctx, i)
	}
	log.Printf("started %d workers", numWorkers)
}

func worker(ctx context.Context, id int) {
	log.Printf("worker %d started", id)
	for {
		select {
		case <-ctx.Done():
			log.Printf("worker %d shutting down", id)
			return
		default:
			job, err := claimJob()
			if err == sql.ErrNoRows {
				time.Sleep(2 * time.Second)
				continue
			}
			if err != nil {
				log.Printf("worker %d claim error %v", id, err)
				time.Sleep(2 * time.Second)
				continue
			}
			processJob(ctx, id, job)

		}
	}
}

func processJob(ctx context.Context, workerID int, job *Job) {
	log.Printf("worker %d processing job %d: %s", workerID, job.ID, job.Payload)

	select {
	case <-ctx.Done():
		log.Printf("worker %d job %d cancelled", workerID, job.ID)
		failJob(job.ID)
		return
	case <-time.After(100 * time.Millisecond):
		// small delay to simulate pickup
	}

	// parse and send email
	to, subject, body, err := parseEmailJob(job.Payload)
	if err != nil {
		log.Printf("worker %d job %d parse error: %v", workerID, job.ID, err)
		failJob(job.ID)
		return
	}

	if err := sendEmail(to, subject, body); err != nil {
		log.Printf("worker %d job %d email failed: %v", workerID, job.ID, err)
		failJob(job.ID)
		return
	}

	completeJob(job.ID)
	log.Printf("worker %d completed job %d — email sent to %s", workerID, job.ID, to)
}
