package clientlib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type GhostCacheClient struct {
	CoordinatorURL   string
	RetryCount       int
	RetryDelay       time.Duration
	FailureCount     int
	FailureThreshold int
	CircuitOpen      bool
	CircuitResetTime time.Duration
	lastFailureTime  time.Time
}

func NewGhostCacheClient(coordinatorURL string) *GhostCacheClient {
	return &GhostCacheClient{
		CoordinatorURL:   coordinatorURL,
		RetryCount:       3,
		RetryDelay:       500 * time.Millisecond,
		FailureThreshold: 5,
		CircuitResetTime: 10 * time.Second,
	}
}

func (c *GhostCacheClient) doRequest(req *http.Request) (*http.Response, error) {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	var resp *http.Response
	var err error
	for i := 0; i < c.RetryCount; i++ {
		resp, err = client.Do(req)
		if err == nil {
			return resp, nil
		}
		time.Sleep(c.RetryDelay)
	}
	return nil, err
}

func (c *GhostCacheClient) checkCircuit() bool {
	if c.CircuitOpen {
		if time.Since(c.lastFailureTime) > c.CircuitResetTime {
			c.CircuitOpen = false
			c.FailureCount = 0
			return false
		}
		return true
	}
	return false
}

func (c *GhostCacheClient) recordFailure() {
	c.FailureCount++
	c.lastFailureTime = time.Now()
	if c.FailureCount >= c.FailureThreshold {
		c.CircuitOpen = true
	}
}

func (c *GhostCacheClient) getNode(key string) (string, error) {
	if c.checkCircuit() {
		return "", errors.New("circuit breaker open")
	}
	url := fmt.Sprintf("%s/where?key=%s", c.CoordinatorURL, key)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.doRequest(req)
	if err != nil {
		c.recordFailure()
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	nodeURL := string(bytes.TrimSpace(body))
	return nodeURL, nil
}

func (c *GhostCacheClient) Set(key, value string) error {
	nodeURL, err := c.getNode(key)
	if err != nil {
		return err
	}
	payload := map[string]string{
		"key":   key,
		"value": value,
	}
	data, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/set", nodeURL)
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.doRequest(req)
	if err != nil {
		c.recordFailure()
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to set key")
	}
	return nil
}

func (c *GhostCacheClient) Get(key string) (string, error) {
	nodeURL, err := c.getNode(key)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/get?key=%s", nodeURL, key)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.doRequest(req)
	if err != nil {
		c.recordFailure()
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("failed to get key")
	}
	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Status string      `json:"status"`
		Data   interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return "", err
	}
	if res.Status != "success" {
		return "", errors.New("error response from server")
	}
	val, ok := res.Data.(string)
	if !ok {
		return "", errors.New("unexpected data type")
	}
	return val, nil
}

func (c *GhostCacheClient) Delete(key string) error {
	nodeURL, err := c.getNode(key)
	if err != nil {
		return err
	}
	payload := map[string]string{
		"key": key,
	}
	data, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/delete", nodeURL)
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.doRequest(req)
	if err != nil {
		c.recordFailure()
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to delete key")
	}
	return nil
}

func (c *GhostCacheClient) UpdateTTL(key string, ttl float64) error {
	nodeURL, err := c.getNode(key)
	if err != nil {
		return err
	}
	payload := map[string]interface{}{
		"key": key,
		"ttl": ttl,
	}
	data, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/updateTTL", nodeURL)
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.doRequest(req)
	if err != nil {
		c.recordFailure()
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to update TTL")
	}
	return nil
}

func (c *GhostCacheClient) Stats(key string) (map[string]interface{}, error) {
	nodeURL, err := c.getNode(key)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/stats", nodeURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.doRequest(req)
	if err != nil {
		c.recordFailure()
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to get stats")
	}
	body, _ := io.ReadAll(resp.Body)
	var res struct {
		Status string                 `json:"status"`
		Data   map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	if res.Status != "success" {
		return nil, errors.New("error response from server")
	}
	return res.Data, nil
}
