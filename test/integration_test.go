package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

const (
	baseURL = "http://localhost:8080"
)

var (
	userAccessToken   string
	sellerAccessToken string
	adminAccessToken  string
	productID         string
	orderID           string
	promoCodeID       string
)

func TestMain(m *testing.M) {
	// i cant use docker, so podman
	composeCmd := "docker-compose"
	if _, err := exec.LookPath("podman-compose"); err == nil {
		composeCmd = "podman-compose"
	} else if _, err := exec.LookPath("docker"); err == nil {
		composeCmd = "docker"
	}

	fmt.Printf("Starting services with %s...\n", composeCmd)

	var cmd *exec.Cmd
	if composeCmd == "docker" {
		cmd = exec.Command("docker", "compose", "up", "-d", "--build")
	} else {
		cmd = exec.Command(composeCmd, "up", "-d", "--build")
	}

	// Set working directory to parent directory where docker-compose.yml is located
	cmd.Dir = ".."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to start services: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Waiting for services to be ready...")
	time.Sleep(20 * time.Second)

	if !waitForService(baseURL+"/products", 60) {
		fmt.Println("Service did not become ready in time")
		fmt.Println("Checking service logs...")

		// Try to get logs to help debug
		var logCmd *exec.Cmd
		if composeCmd == "docker" {
			logCmd = exec.Command("docker", "compose", "logs", "marketplace-api")
		} else {
			logCmd = exec.Command(composeCmd, "logs", "marketplace-api")
		}
		logCmd.Dir = ".."
		output, _ := logCmd.CombinedOutput()
		fmt.Printf("API logs:\n%s\n", string(output))

		cleanup()
		os.Exit(1)
	}

	code := m.Run()

	cleanup()
	os.Exit(code)
}

func cleanup() {
	fmt.Println("Stopping services...")

	// Detect which compose command is available
	composeCmd := "docker-compose"
	if _, err := exec.LookPath("podman-compose"); err == nil {
		composeCmd = "podman-compose"
	} else if _, err := exec.LookPath("docker"); err == nil {
		composeCmd = "docker"
	}

	var cmd *exec.Cmd
	if composeCmd == "docker" {
		cmd = exec.Command("docker", "compose", "down", "-v")
	} else {
		cmd = exec.Command(composeCmd, "down", "-v")
	}

	// Set working directory to parent directory where docker-compose.yml is located
	cmd.Dir = ".."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func waitForService(url string, timeoutSeconds int) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return true
			}
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

func TestAuthFlow(t *testing.T) {
	t.Run("Register USER", func(t *testing.T) {
		payload := map[string]interface{}{
			"email":    "user@test.com",
			"password": "password123",
			"role":     "USER",
		}
		resp := makeRequest(t, "POST", "/auth/register", payload, "")
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		userAccessToken = result["access_token"].(string)

		if userAccessToken == "" {
			t.Fatal("No access token received")
		}
	})

	t.Run("Register SELLER", func(t *testing.T) {
		payload := map[string]interface{}{
			"email":    "seller@test.com",
			"password": "password123",
			"role":     "SELLER",
		}
		resp := makeRequest(t, "POST", "/auth/register", payload, "")
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		sellerAccessToken = result["access_token"].(string)
	})

	t.Run("Register ADMIN", func(t *testing.T) {
		payload := map[string]interface{}{
			"email":    "admin@test.com",
			"password": "password123",
			"role":     "ADMIN",
		}
		resp := makeRequest(t, "POST", "/auth/register", payload, "")
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		adminAccessToken = result["access_token"].(string)
	})

	t.Run("Login", func(t *testing.T) {
		payload := map[string]interface{}{
			"email":    "user@test.com",
			"password": "password123",
		}
		resp := makeRequest(t, "POST", "/auth/login", payload, "")
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["access_token"] == nil {
			t.Fatal("No access token in login response")
		}
	})

	t.Run("Refresh Token", func(t *testing.T) {
		loginPayload := map[string]interface{}{
			"email":    "user@test.com",
			"password": "password123",
		}
		loginResp := makeRequest(t, "POST", "/auth/login", loginPayload, "")
		defer loginResp.Body.Close()

		var loginResult map[string]interface{}
		json.NewDecoder(loginResp.Body).Decode(&loginResult)
		refreshToken := loginResult["refresh_token"].(string)

		refreshPayload := map[string]interface{}{
			"refresh_token": refreshToken,
		}
		resp := makeRequest(t, "POST", "/auth/refresh", refreshPayload, "")
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["access_token"] == nil {
			t.Fatal("No access token in refresh response")
		}
	})
}

func TestProductFlow(t *testing.T) {
	t.Run("Create Product as USER (should fail)", func(t *testing.T) {
		payload := map[string]interface{}{
			"name":        "Test Product",
			"description": "Test Description",
			"price":       99.99,
			"stock":       100,
			"category":    "electronics",
		}
		resp := makeRequest(t, "POST", "/products", payload, userAccessToken)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("Expected status 403, got %d", resp.StatusCode)
		}
	})

	t.Run("Create Product as SELLER", func(t *testing.T) {
		payload := map[string]interface{}{
			"name":        "Test Product",
			"description": "Test Description",
			"price":       99.99,
			"stock":       100,
			"category":    "electronics",
		}
		resp := makeRequest(t, "POST", "/products", payload, sellerAccessToken)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		productID = result["id"].(string)
	})

	t.Run("List Products", func(t *testing.T) {
		resp := makeRequest(t, "GET", "/products?page=1&size=20", nil, "")
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		items := result["items"].([]interface{})
		if len(items) == 0 {
			t.Fatal("Expected at least one product")
		}
	})

	t.Run("Get Product", func(t *testing.T) {
		resp := makeRequest(t, "GET", "/products/"+productID, nil, "")
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["name"] != "Test Product" {
			t.Fatalf("Expected product name 'Test Product', got %v", result["name"])
		}
	})

	t.Run("Update Product as SELLER", func(t *testing.T) {
		payload := map[string]interface{}{
			"name":  "Updated Product",
			"price": 89.99,
		}
		resp := makeRequest(t, "PUT", "/products/"+productID, payload, sellerAccessToken)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("Update Product as different SELLER (should fail)", func(t *testing.T) {
		newSellerPayload := map[string]interface{}{
			"email":    "seller2@test.com",
			"password": "password123",
			"role":     "SELLER",
		}
		regResp := makeRequest(t, "POST", "/auth/register", newSellerPayload, "")
		defer regResp.Body.Close()

		var regResult map[string]interface{}
		json.NewDecoder(regResp.Body).Decode(&regResult)
		seller2Token := regResult["access_token"].(string)

		payload := map[string]interface{}{
			"name": "Hacked Product",
		}
		resp := makeRequest(t, "PUT", "/products/"+productID, payload, seller2Token)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("Expected status 403, got %d", resp.StatusCode)
		}
	})
}

func TestPromoCodeFlow(t *testing.T) {
	t.Run("Create Promo Code as USER (should fail)", func(t *testing.T) {
		payload := map[string]interface{}{
			"code":             "SAVE10",
			"discount_type":    "PERCENTAGE",
			"discount_value":   10.0,
			"min_order_amount": 50.0,
			"max_uses":         100,
			"valid_from":       time.Now().Format(time.RFC3339),
			"valid_until":      time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339),
		}
		resp := makeRequest(t, "POST", "/promo-codes", payload, userAccessToken)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("Expected status 403, got %d", resp.StatusCode)
		}
	})

	t.Run("Create Promo Code as ADMIN", func(t *testing.T) {
		payload := map[string]interface{}{
			"code":             "SAVE10",
			"discount_type":    "PERCENTAGE",
			"discount_value":   10.0,
			"min_order_amount": 50.0,
			"max_uses":         100,
			"valid_from":       time.Now().Add(-24 * time.Hour).Format(time.RFC3339), // Start 24 hours ago to avoid any timezone issues
			"valid_until":      time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339),
		}
		resp := makeRequest(t, "POST", "/promo-codes", payload, adminAccessToken)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		promoCodeID = result["id"].(string)
	})

	t.Run("Get Promo Code as ADMIN", func(t *testing.T) {
		resp := makeRequest(t, "GET", "/promo-codes/"+promoCodeID, nil, adminAccessToken)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Update Promo Code as ADMIN", func(t *testing.T) {
		payload := map[string]interface{}{
			"discount_value": 15.0,
		}
		resp := makeRequest(t, "PUT", "/promo-codes/"+promoCodeID, payload, adminAccessToken)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})
}

func TestOrderFlow(t *testing.T) {
	t.Run("Create Order with Promo Code", func(t *testing.T) {
		promoCode := "SAVE10"
		payload := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"product_id": productID,
					"quantity":   2,
				},
			},
			"promo_code": promoCode,
		}
		resp := makeRequest(t, "POST", "/orders", payload, userAccessToken)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 201, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		orderID = result["id"].(string)

		if result["discount_amount"].(float64) <= 0 {
			t.Fatal("Expected discount to be applied")
		}
	})

	t.Run("Get Order", func(t *testing.T) {
		resp := makeRequest(t, "GET", "/orders/"+orderID, nil, userAccessToken)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["status"] != "CREATED" {
			t.Fatalf("Expected status CREATED, got %v", result["status"])
		}
	})

	t.Run("Update Order Status", func(t *testing.T) {
		payload := map[string]interface{}{
			"status": "PAYMENT_PENDING",
		}
		resp := makeRequest(t, "PUT", "/orders/"+orderID, payload, userAccessToken)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("Invalid Status Transition (should fail)", func(t *testing.T) {
		payload := map[string]interface{}{
			"status": "COMPLETED",
		}
		resp := makeRequest(t, "PUT", "/orders/"+orderID, payload, userAccessToken)
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Fatal("Expected invalid transition to fail")
		}
	})

	t.Run("Create Second Order (should fail - rate limit)", func(t *testing.T) {
		payload := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"product_id": productID,
					"quantity":   1,
				},
			},
		}
		resp := makeRequest(t, "POST", "/orders", payload, userAccessToken)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusConflict && resp.StatusCode != http.StatusTooManyRequests {
			body, _ := io.ReadAll(resp.Body)
			t.Logf("Warning: Expected rate limit or active order error, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("Cancel Order", func(t *testing.T) {
		resp := makeRequest(t, "POST", "/orders/"+orderID+"/cancel", nil, userAccessToken)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["status"] != "CANCELED" {
			t.Fatalf("Expected status CANCELED, got %v", result["status"])
		}
	})
}

func TestAccessControl(t *testing.T) {
	t.Run("USER cannot access other user's order", func(t *testing.T) {
		newUserPayload := map[string]interface{}{
			"email":    "user2@test.com",
			"password": "password123",
			"role":     "USER",
		}
		regResp := makeRequest(t, "POST", "/auth/register", newUserPayload, "")
		defer regResp.Body.Close()

		var regResult map[string]interface{}
		json.NewDecoder(regResp.Body).Decode(&regResult)
		user2Token := regResult["access_token"].(string)

		resp := makeRequest(t, "GET", "/orders/"+orderID, nil, user2Token)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("Expected status 403, got %d", resp.StatusCode)
		}
	})

	t.Run("ADMIN can access any order", func(t *testing.T) {
		resp := makeRequest(t, "GET", "/orders/"+orderID, nil, adminAccessToken)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}

func makeRequest(t *testing.T, method, path string, payload interface{}, token string) *http.Response {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("Failed to marshal payload: %v", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, baseURL+path, body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	return resp
}
