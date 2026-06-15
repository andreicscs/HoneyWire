package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/honeywire/wizard/core/schema"
)

const wizardUserAgent = "HoneyWire-Wizard/2.0"
const WizardMinHubAPI = 1

type HubClient struct {
	baseURL string
	client  *http.Client
}

func NewClient(baseURL string) *HubClient {
	transport := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: false,
	}
	return &HubClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
	}
}

func NewHubClient(baseURL string) *HubClient {
	return NewClient(baseURL)
}

func (c *HubClient) endpoint(path string) string {
	return c.baseURL + path
}

func (c *HubClient) doRequest(ctx context.Context, method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint(path), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", wizardUserAgent)
	req.Header.Set("X-Wizard-Min-Hub-Api", fmt.Sprintf("%d", WizardMinHubAPI))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return c.client.Do(req)
}

func readBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return data, nil
}

func readBodyTruncated(resp *http.Response) string {
	data, err := readBody(resp)
	if err != nil {
		return fmt.Sprintf("(failed to read body: %v)", err)
	}
	msg := strings.TrimSpace(string(data))
	if len(msg) > 200 {
		msg = msg[:200] + "..."
	}
	return msg
}

func (c *HubClient) AuthenticateDashboard(ctx context.Context, password string) (string, error) {
	payload := map[string]string{"password": password}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal login request: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/login", bytes.NewReader(body), map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil {
		return "", fmt.Errorf("network error: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("hub rejected credentials (HTTP %d): %s", resp.StatusCode, readBodyTruncated(resp))
	}

	var cookieValue string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "hw_auth" {
			cookieValue = cookie.Value
			break
		}
	}
	if cookieValue == "" {
		return "", fmt.Errorf("hub did not return authentication cookie")
	}

	return cookieValue, nil
}

func (c *HubClient) CreateNode(ctx context.Context, alias string, tags []string, cookie string) (string, error) {
	payload := createNodeRequest{
		Alias: alias,
		Tags:  tags,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal create node request: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1/nodes", bytes.NewReader(body), map[string]string{
		"Content-Type": "application/json",
		"Cookie":       "hw_auth=" + cookie,
	})
	if err != nil {
		return "", fmt.Errorf("network error: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("hub rejected request (HTTP %d): %s", resp.StatusCode, readBodyTruncated(resp))
	}

	data, err := readBody(resp)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("failed to parse hub response: %w", err)
	}

	var apiKey string
	if val, ok := result["api_key"].(string); ok && val != "" {
		apiKey = val
	} else if val, ok := result["apiKey"].(string); ok && val != "" {
		apiKey = val
	} else if val, ok := result["key"].(string); ok && val != "" {
		apiKey = val
	}

	if apiKey == "" {
		return "", fmt.Errorf("hub did not return an API key. Raw response: %s", string(data))
	}

	return apiKey, nil
}

func (c *HubClient) GetCurrentNode(ctx context.Context, apiKey string) (*NodeInfo, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/nodes/me", nil, map[string]string{
		"Authorization": "Bearer " + apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API key rejected (HTTP %d): %s", resp.StatusCode, readBodyTruncated(resp))
	}

	data, err := readBody(resp)
	if err != nil {
		return nil, err
	}

	var info NodeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse node info: %w", err)
	}

	return &info, nil
}

func (c *HubClient) AddSensor(ctx context.Context, nodeID, cookie, sensorID, customName string, configValues map[string]string) error {
	payload := addSensorRequest{
		SensorID:     sensorID,
		CustomName:   customName,
		ConfigValues: configValues,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal add sensor request: %w", err)
	}

	path := fmt.Sprintf("/api/v1/nodes/%s/sensors", nodeID)
	resp, err := c.doRequest(ctx, http.MethodPost, path, bytes.NewReader(body), map[string]string{
		"Content-Type": "application/json",
		"Cookie":       "hw_auth=" + cookie,
	})
	if err != nil {
		return fmt.Errorf("network error adding sensor %s: %w", sensorID, err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("hub rejected sensor %s (HTTP %d): %s", sensorID, resp.StatusCode, readBodyTruncated(resp))
	}

	return nil
}

func (c *HubClient) FetchCompose(ctx context.Context, apiKey string) ([]byte, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/nodes/compose", nil, map[string]string{
		"Authorization": "Bearer " + apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hub returned error (HTTP %d): %s", resp.StatusCode, readBodyTruncated(resp))
	}

	return readBody(resp)
}

func (c *HubClient) FetchInstalledSensors(ctx context.Context, nodeID, apiKey string) (map[string]bool, error) {
	info, err := c.GetCurrentNode(ctx, apiKey)
	if err != nil {
		return nil, err
	}

	installed := make(map[string]bool, len(info.InstalledSensors))
	for _, s := range info.InstalledSensors {
		installed[s.SensorID] = true
	}

	return installed, nil
}

type createNodeRequest struct {
	Alias string   `json:"alias"`
	Tags  []string `json:"tags,omitempty"`
}

type NodeInfo struct {
	NodeID           string       `json:"nodeId"`
	Alias            string       `json:"alias"`
	Tags             []string     `json:"tags"`
	Status           string       `json:"status"`
	PendingConfig    bool         `json:"hasPendingConfig"`
	ActiveRevision   string       `json:"activeRevision"`
	DesiredRevision  string       `json:"desiredRevision"`
	InstalledSensors []SensorInfo `json:"installedSensors"`
}

type SensorInfo struct {
	SensorID   string `json:"sensorId"`
	CustomName string `json:"display"`
	IsSilenced bool   `json:"isSilenced"`
}

type addSensorRequest struct {
	SensorID     string            `json:"sensorId"`
	CustomName   string            `json:"customName"`
	ConfigValues map[string]string `json:"configValues"`
}

func (c *HubClient) FetchManifests(ctx context.Context, apiKey string) ([]*schema.SensorManifest, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/manifests", nil, map[string]string{
		"Authorization": "Bearer " + apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hub returned error (HTTP %d): %s", resp.StatusCode, readBodyTruncated(resp))
	}

	data, err := readBody(resp)
	if err != nil {
		return nil, err
	}

	var manifests []*schema.SensorManifest
	if err := json.Unmarshal(data, &manifests); err != nil {
		return nil, fmt.Errorf("failed to parse manifests: %w", err)
	}

	return manifests, nil
}