package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

const baseURL = "http://localhost:8080"

type APIResponse struct {
	Status    string  `json:"status"`
	Reason    string  `json:"reason"`
	Version   string  `json:"version"`
	RiskScore float64 `json:"risk_score"`
	Error     string  `json:"error"`
}

func sendTransaction(t *testing.T, version, userID string) (int, APIResponse) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"transaction_id": fmt.Sprintf("txn_%d", time.Now().UnixNano()),
		"user_id":        userID,
		"amount":         100.50,
		"ip_address":     "127.0.0.1",
	})

	url := fmt.Sprintf("%s/api/%s/fraud/check", baseURL, version)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	bodyBytes, _ := io.ReadAll(resp.Body)
	var apiResp APIResponse
	err = json.Unmarshal(bodyBytes, &apiResp)
	if err != nil {
		return 0, APIResponse{}
	}

	return resp.StatusCode, apiResp
}

// --- TEST CASES ---

func TestV1_HappyPath_Allow(t *testing.T) {
	userID := fmt.Sprintf("user_happy_%d", time.Now().Unix())
	code, resp := sendTransaction(t, "v1", userID)

	if code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", code)
	}
	if resp.Status != "ALLOW" {
		t.Errorf("Expected status ALLOW, got %s", resp.Status)
	}
}

func TestV1_VelocityLimit_Block(t *testing.T) {
	userID := fmt.Sprintf("user_spammer_%d", time.Now().Unix())

	// Send 5 allowed requests
	for i := 1; i <= 5; i++ {
		code, resp := sendTransaction(t, "v1", userID)
		if resp.Status != "ALLOW" {
			t.Fatalf("Request %d failed. Expected ALLOW, got %s", i, resp.Status)
		}
		if code != http.StatusOK {
			t.Fatalf("Request %d failed. Expected 200, got %d", i, code)
		}
	}

	// Send the 6th request (Should be BLOCKED)
	code, resp := sendTransaction(t, "v1", userID)

	if code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden, got %d", code)
	}
	if resp.Status != "BLOCK" {
		t.Errorf("Expected status BLOCK, got %s", resp.Status)
	}
	if resp.Reason != "ERR_VELOCITY_LIMIT" {
		t.Errorf("Expected reason ERR_VELOCITY_LIMIT, got %s", resp.Reason)
	}
}

func TestV1_Validation_MissingFields(t *testing.T) {
	// Send invalid JSON (missing amount)
	reqBody := []byte(`{"transaction_id": "txn_bad", "user_id": "bad_user"}`)
	url := fmt.Sprintf("%s/api/v1/fraud/check", baseURL)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestV1_UserIsolation(t *testing.T) {
	// Ensure that blocking User A does not block User B
	userA := fmt.Sprintf("user_a_%d", time.Now().Unix())
	userB := fmt.Sprintf("user_b_%d", time.Now().Unix())

	// Spam User A until blocked
	for i := 0; i < 6; i++ {
		sendTransaction(t, "v1", userA)
	}
	codeA, _ := sendTransaction(t, "v1", userA)
	if codeA != http.StatusForbidden {
		t.Fatalf("Setup failed: User A should be blocked")
	}

	// Check User B (Should still be allowed)
	codeB, respB := sendTransaction(t, "v1", userB)
	if codeB != http.StatusOK {
		t.Errorf("User B should be ALLOWED but got %d", codeB)
	}
	if respB.Status != "ALLOW" {
		t.Errorf("User B status should be ALLOW but got %s", respB.Status)
	}
}

func TestV2_FeatureFlag(t *testing.T) {
	userID := fmt.Sprintf("user_v2_%d", time.Now().Unix())
	code, resp := sendTransaction(t, "v2", userID)

	if code != http.StatusAccepted {
		t.Errorf("Expected 202 Accepted, got %d", code)
	}
	if resp.Version != "v2" {
		t.Errorf("Expected version 'v2', got %s", resp.Version)
	}
	if resp.RiskScore == 0 {
		t.Error("Expected RiskScore to be > 0")
	}
}

func TestV1_Blacklist_User(t *testing.T) {
	// "bad_hacker_1" is hardcoded as blacklisted in main.go
	code, resp := sendTransaction(t, "v1", "bad_hacker_1")

	if code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden, got %d", code)
	}
	if resp.Reason != "ERR_BLACKLISTED" {
		t.Errorf("Expected ERR_BLACKLISTED, got %s", resp.Reason)
	}
}

func TestV1_GeoCheck_Risk(t *testing.T) {
	// Simulate an IP starting with "10." (Our rule for High Risk)
	reqBody, _ := json.Marshal(map[string]interface{}{
		"transaction_id": "geo_test",
		"user_id":        "traveler_joe",
		"amount":         100.0,
		"ip_address":     "10.5.5.5", // <-- Trigger
	})

	url := fmt.Sprintf("%s/api/v1/fraud/check", baseURL)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, _ := client.Post(url, "application/json", bytes.NewBuffer(reqBody))

	bodyBytes, _ := io.ReadAll(resp.Body)
	var apiResp APIResponse
	err := json.Unmarshal(bodyBytes, &apiResp)
	if err != nil {
		return
	}

	if apiResp.Reason != "ERR_GEO_RISK" {
		t.Errorf("Expected ERR_GEO_RISK, got %s", apiResp.Reason)
	}
}

func TestAdmin_AddBlacklistRule(t *testing.T) {
	// 1. Pick a fresh user who is NOT banned
	targetUser := fmt.Sprintf("hacker_dynamic_%d", time.Now().Unix())

	// Verify they are allowed first
	code, resp := sendTransaction(t, "v1", targetUser)
	if code != http.StatusOK {
		t.Fatalf("Pre-check failed: User should be allowed initially")
	}

	// 2. Call Admin API to ban them
	ruleBody, _ := json.Marshal(map[string]string{
		"type":  "users",
		"value": targetUser,
	})

	adminURL := fmt.Sprintf("%s/api/v1/rules/blacklist", baseURL)
	client := &http.Client{Timeout: 5 * time.Second}
	adminResp, err := client.Post(adminURL, "application/json", bytes.NewBuffer(ruleBody))
	if err != nil {
		t.Fatalf("Admin API call failed: %v", err)
	}
	defer adminResp.Body.Close()

	if adminResp.StatusCode != http.StatusOK {
		t.Errorf("Admin API returned %d", adminResp.StatusCode)
	}

	// 3. Verify they are now BLOCKED
	code, resp = sendTransaction(t, "v1", targetUser)
	if code != http.StatusForbidden {
		t.Errorf("Post-ban check failed: Expected 403, got %d", code)
	}
	if resp.Reason != "ERR_BLACKLISTED" {
		t.Errorf("Expected ERR_BLACKLISTED, got %s", resp.Reason)
	}
}
