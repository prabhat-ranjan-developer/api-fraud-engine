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
// @description     Real-time fraud detection service with Redis, Kafka, and Postgres.
// @contact.name    Prabhat Ranjan

// @accept  json
// @produce  json
func main() {
	// 1. Setup Postgres
	dsn := os.Getenv("DB_DSN") 
    
    // Fallback for local testing if DB_DSN is empty
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
	redisRepo := repository.NewRedisRepository(rdb)

	// 3. Setup Kafka Producer
	kafkaHost := os.Getenv("KAFKA_ADDR")
	if kafkaHost == "" {
		kafkaHost = "kafka:9092"
	}
	// Topic: "fraud_alerts"
	producer := repository.NewEventProducer(kafkaHost, "fraud_alerts")
	defer producer.Close()

	// 4. Setup Service (Injects Repo, Redis, and Producer)
	svc := service.NewFraudService(repo, redisRepo, producer)
	h := handler.NewHandler(svc)

	// 5. Setup Kafka Consumer (For the Dashboard Stream)
	// Channel Buffer 100 to prevent blocking
	alertChannel := make(chan string, 100)

	// Start Consumer in background
	consumer := repository.NewEventConsumer(kafkaHost, "fraud_alerts")
	defer consumer.Close()

	// Important: Run in goroutine so it doesn't block server start
	go consumer.Subscribe(context.Background(), alertChannel)

	// 6. Router Setup
	r := gin.Default()

	// Prometheus
	p := ginprometheus.NewPrometheus("gin")
	p.Use(r)

	// Swagger
	docs.SwaggerInfo.BasePath = "/api"
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	{
		v1.POST("/fraud/check", h.CheckFraudV1)
		v1.POST("/rules/blacklist", h.AddBlacklistRule)

		// NEW: Subscribe to Kafka Alerts
		v1.GET("/events/stream", h.StreamFraudAlerts(alertChannel))
	}

	v2 := r.Group("/api/v2")
	{
		v2.POST("/fraud/check", h.CheckFraudV2)
	}

	log.Println("Fraud Engine running on port 8080 (Internal) / 8081 (External)")
	err = r.Run(":8080")
	if err != nil {
		return
	} // Internal Docker Port
}
