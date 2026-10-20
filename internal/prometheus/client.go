package prometheus

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/go-logr/logr"
)

// Client wraps Prometheus HTTP API with AK/SK authentication
type Client struct {
	address    string
	accessKey  string
	secretKey  string
	authHeader string
	client     *http.Client
	logger     logr.Logger
}

// QueryResult represents Prometheus query result
type QueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// NewClient creates a new Prometheus client with AK/SK auth
func NewClient(address, accessKey, secretKey string, logger logr.Logger) *Client {
	credentials := fmt.Sprintf("%s:%s", accessKey, secretKey)
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))

	return &Client{
		address:    address,
		accessKey:  accessKey,
		secretKey:  secretKey,
		authHeader: fmt.Sprintf("Basic %s", encoded),
		client: &http.Client{
			Timeout: 30 * time.Second, // 增加到 30 秒
		},
		logger: logger.WithName("prometheus-client"),
	}
}

// Query executes a PromQL query and returns the result
func (c *Client) Query(ctx context.Context, query string) (*QueryResult, error) {
	c.logger.Info("Executing PromQL query", "query", query)

	// Parse base URL
	u, err := url.Parse(c.address)
	if err != nil {
		return nil, fmt.Errorf("invalid prometheus address: %w", err)
	}

	// Append /api/v1/query to the existing path (don't overwrite)
	if u.Path == "" || u.Path == "/" {
		u.Path = "/api/v1/query"
	} else {
		u.Path = u.Path + "/api/v1/query"
	}

	q := u.Query()
	q.Set("query", query)
	q.Set("timeout", "1000")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set auth headers
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Error(err, "Failed to query Prometheus", "address", c.address)
		return nil, fmt.Errorf("query prometheus: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Info("Prometheus returned error status",
			"status", resp.StatusCode,
			"body", string(body))
		return nil, fmt.Errorf("prometheus returned status %d: %s", resp.StatusCode, string(body))
	}

	var result QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Status != "success" {
		c.logger.Info("Prometheus query failed", "status", result.Status)
		return nil, fmt.Errorf("query failed: %v", result)
	}

	c.logger.Info("Prometheus query succeeded",
		"resultCount", len(result.Data.Result),
		"resultType", result.Data.ResultType)

	return &result, nil
}
