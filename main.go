package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/gin-gonic/gin"
)

func main() {
	initDB()

	ctx, cancle := context.WithCancel(context.Background())
	defer cancle()

	startWorkerPool(ctx, 5)

	r := gin.Default()

	r.POST("/jobs", func(c *gin.Context) {
		var req CreateJobRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		job, err := createJob(req.Payload)
		if err != nil {
			log.Println("createJob error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create job"})
			return
		}
		c.JSON(http.StatusCreated, job)
	})

	r.GET("/jobs/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job"})
			return
		}

		job, err := getJob(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}

		c.JSON(http.StatusOK, job)
	})

	//graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-quit
		log.Println(".shutdown signal recceived")
		cancle()
	}()

	log.Println("server listening on :8080")
	r.Run(":8080")
}
