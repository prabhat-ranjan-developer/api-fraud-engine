package domain

import "time"

type TransactionRequest struct {
	TransactionID string  `json:"transaction_id" binding:"required"`
	UserID        string  `json:"user_id" binding:"required"`
	Amount        float64 `json:"amount" binding:"required"`
	IPAddress     string  `json:"ip_address"`
}

type FraudCheckResponse struct {
	Status        string  `json:"status"`
	Reason        string  `json:"reason"`
	RiskScore     float64 `json:"risk_score"`
	Version       string  `json:"version"`
	TransactionID string
}

type FraudLog struct {
	LogID         uint   `gorm:"primaryKey"`
	TransactionID string `gorm:"not null"`
	Status        string `gorm:"not null"`
	Reason        string `gorm:"not null"`
	APIVersion    string `gorm:"not null"`
	RiskScore     float64
	CreatedAt     time.Time
	UserID        string
	Amount        float64
	IPAddress     string
}

type RuleRequest struct {
	Type  string `json:"type" binding:"required,oneof=users ips"`
	Value string `json:"value" binding:"required"`
}
