package service

import (
	"context"
	"fraud-engine/internal/domain"
	"fraud-engine/internal/repository" // Ensure this path is correct
	"time"
)

type FraudService struct {
	Repo     *repository.PostgresRepository
	Redis    *repository.RedisRepository
	Broker   *repository.EventBroker // <--- Changed from Producer to Broker
}

func NewFraudService(repo *repository.PostgresRepository, redis *repository.RedisRepository, broker *repository.EventBroker) *FraudService {
	return &FraudService{
		Repo:   repo,
		Redis:  redis,
		Broker: broker,
	}
}

func (s *FraudService) CheckFraudV1(ctx context.Context, req domain.TransactionRequest) domain.FraudCheckResponse {
	// 1. Blacklist Check
	isBlacklisted, _ := s.Redis.IsBlacklisted(ctx, req.UserID, req.IPAddress)
	if isBlacklisted {
		// Publish to Redis instead of Kafka
		go s.Broker.PublishFraudAlert(context.Background(), "fraud_alerts", "Blacklisted Entity: "+req.UserID)

		s.logFraud(req, "BLOCK", "User/IP is blacklisted", "v1")
		return domain.FraudCheckResponse{
			TransactionID: req.TransactionID,
			Status:        "BLOCK",
			Reason:        "User or IP is in blacklist",
		}
	}

	// 2. Velocity Check
	count, _ := s.Redis.IncrementVelocity(ctx, req.UserID, 60*time.Second)
	if count > 5 {
		go s.Broker.PublishFraudAlert(context.Background(), "fraud_alerts", "Velocity Limit Exceeded: "+req.UserID)

		s.logFraud(req, "BLOCK", "Velocity limit exceeded", "v1")
		return domain.FraudCheckResponse{
			TransactionID: req.TransactionID,
			Status:        "BLOCK",
			Reason:        "Velocity limit exceeded",
		}
	}
	s.logFraud(req, "ALLOW", "Clean", "v1")
	return domain.FraudCheckResponse{
		TransactionID: req.TransactionID,
		Status:        "ALLOW",
		Reason:        "Clean",
	}
}

// ... (Keep CheckFraudV2, BlacklistEntity, logFraud methods as they were) ...
// Make sure logFraud is present or copied from previous steps.
func (s *FraudService) logFraud(req domain.TransactionRequest, status, reason, version string) {
	err := s.Repo.LogFraud(domain.FraudLog{
		TransactionID: req.TransactionID,
		UserID:        req.UserID,
		Amount:        req.Amount,
		IPAddress:     req.IPAddress,
		Status:        status,
		Reason:        reason,
		APIVersion:    version,
		CreatedAt:     time.Now(),
	})
	if err != nil {
		return
	}
}

// BlacklistEntity ... (Keep BlacklistEntity method) ...
func (s *FraudService) BlacklistEntity(ctx context.Context, listType, value string) error {
	return s.Redis.AddToBlacklist(ctx, listType, value)
}
func (s *FraudService) CheckFraudV2(ctx context.Context, req domain.TransactionRequest) domain.FraudCheckResponse {
	// Simple simulation for V2
	return domain.FraudCheckResponse{
		TransactionID: req.TransactionID,
		Status:        "FLAG",
		RiskScore:     85.5,
		Reason:        "AI Model Flagged High Risk",
		Version:       "v2",
	}
}
