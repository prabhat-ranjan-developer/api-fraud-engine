package main

import (
	"context"
	"fraud-engine/docs"
	"fraud-engine/internal/handler"
	"fraud-engine/internal/repository"
	"fraud-engine/internal/service"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// @title           Fraud Engine API
// @version         1.0
// @description     Real-time fraud detection service with Redis and Postgres.
// @contact.name    Prabhat Ranjan
// @accept          json
// @produce         json
func main() {
	// 1. Setup Postgres
	dsn := os.Getenv("DB_DSN") 
	if dsn == "" {
		dsn = "host=" + os.Getenv("DB_HOST") + " user=" + os.Getenv("DB_USER") + " password=" + os.Getenv("DB_PASSWORD") + " dbname=" + os.Getenv("DB_NAME") + " port=5432 sslmode=disable"
	}
	
	var db *gorm.DB
	var err error

	for i := 0; i < 10; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Println("Waiting for DB...")
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}
	repo := repository.NewPostgresRepository(db)

	// 2. Setup Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})
	
	// Verify Redis connection
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	
	redisRepo := repository.NewRedisRepository(rdb)

	// 3. Setup Redis Event Broker (Replacing Kafka)
	// This uses Redis Pub/Sub for messaging
	broker := repository.NewEventBroker(rdb) 

	// 4. Setup Service (Injecting Repo, Redis, and Redis-based Broker)
	svc := service.NewFraudService(repo, redisRepo, broker)
	h := handler.NewHandler(svc)

	// 5. Setup Event Stream (For the Dashboard)
	alertChannel := make(chan string, 100)

	// Start Redis Subscription in background
	go func() {
		pubsub := rdb.Subscribe(context.Background(), "fraud_alerts")
		defer pubsub.Close()
		
		ch := pubsub.Channel()
		log.Println("Subscribed to Redis channel: fraud_alerts")
		for msg := range ch {
			alertChannel <- msg.Payload
		}
	}()

	// 6. Router Setup
	r := gin.Default()

	p := ginprometheus.NewPrometheus("gin")
	p.Use(r)

	// Swagger
	docs.SwaggerInfo.BasePath = "/api"
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	{
		v1.POST("/fraud/check", h.CheckFraudV1)
		v1.POST("/rules/blacklist", h.AddBlacklistRule)
		v1.GET("/events/stream", h.StreamFraudAlerts(alertChannel))
	}

	v2 := r.Group("/api/v2")
	{
		v2.POST("/fraud/check", h.CheckFraudV2)
	}

	log.Println("Fraud Engine running on port 8080 (Internal) / 8081 (External)")
	err = r.Run(":8080")
	if err != nil {
		log.Fatal(err)
	}
}