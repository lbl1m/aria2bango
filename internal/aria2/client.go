// Package aria2 provides aria2 RPC client functionality
package aria2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Client represents an aria2 RPC client
type Client struct {
	host   string
	port   int
	secret string
	client *http.Client
	rpcURL string
}

// NewClient creates a new aria2 RPC client
func NewClient(host string, port int, secret string) *Client {
	return &Client{
		host:   host,
		port:   port,
		secret: secret,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		rpcURL: fmt.Sprintf("http://%s:%d/jsonrpc", host, port),
	}
}

// RPCRequest represents a JSON-RPC request
type RPCRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	ID      string        `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

// RPCResponse represents a JSON-RPC response
type RPCResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// StringBool is a custom type to handle aria2's string boolean values ("true"/"false")
type StringBool bool

// UnmarshalJSON implements json.Unmarshaler for StringBool
func (sb *StringBool) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		// Try parsing as bool directly (in case aria2 returns actual bool)
		var b bool
		if err := json.Unmarshal(data, &b); err != nil {
			return err
		}
		*sb = StringBool(b)
		return nil
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	*sb = StringBool(b)
	return nil
}

// Peer represents a BT peer information
type Peer struct {
	PeerID        string     `json:"peerId"`
	IP            string     `json:"ip"`
	Port          int        `json:"port,string"`
	Bitfield      string     `json:"bitfield"`
	AmChoking     StringBool `json:"amChoking"`
	PeerChoking   StringBool `json:"peerChoking"`
	DownloadSpeed int64      `json:"downloadSpeed,string"`
	UploadSpeed   int64      `json:"uploadSpeed,string"`
	Seeder        StringBool `json:"seeder"`
}

// DownloadStatus represents download task status
type DownloadStatus struct {
	Gid             string `json:"gid"`
	Status          string `json:"status"`
	TotalLength     int64  `json:"totalLength,string"`
	CompletedLength int64  `json:"completedLength,string"`
	UploadLength    int64  `json:"uploadLength,string"`
	DownloadSpeed   int64  `json:"downloadSpeed,string"`
	UploadSpeed     int64  `json:"uploadSpeed,string"`
	InfoHash        string `json:"infoHash"`
	Dir             string `json:"dir"`
}

// call makes a JSON-RPC call
func (c *Client) call(ctx context.Context, method string, params []interface{}) (json.RawMessage, error) {
	// Add secret token if configured
	if c.secret != "" {
		params = append([]interface{}{"token:" + c.secret}, params...)
	}

	req := RPCRequest{
		Jsonrpc: "2.0",
		ID:      "aria2bango",
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.rpcURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var rpcResp RPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error: %s (code: %d)", rpcResp.Error.Message, rpcResp.Error.Code)
	}

	return rpcResp.Result, nil
}

// GetActiveDownloads returns all active downloads
func (c *Client) GetActiveDownloads(ctx context.Context) ([]DownloadStatus, error) {
	result, err := c.call(ctx, "aria2.tellActive", []interface{}{})
	if err != nil {
		return nil, err
	}

	var downloads []DownloadStatus
	if err := json.Unmarshal(result, &downloads); err != nil {
		return nil, fmt.Errorf("failed to unmarshal downloads: %w", err)
	}

	return downloads, nil
}

// GetPeers returns peers for a specific download
func (c *Client) GetPeers(ctx context.Context, gid string) ([]Peer, error) {
	result, err := c.call(ctx, "aria2.getPeers", []interface{}{gid})
	if err != nil {
		return nil, err
	}

	var peers []Peer
	if err := json.Unmarshal(result, &peers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal peers: %w", err)
	}

	return peers, nil
}

// GetAllPeers returns all peers from all active downloads
func (c *Client) GetAllPeers(ctx context.Context) (map[string][]Peer, error) {
	downloads, err := c.GetActiveDownloads(ctx)
	if err != nil {
		return nil, err
	}

	allPeers := make(map[string][]Peer)
	for _, download := range downloads {
		// Only get peers for active BT downloads
		if download.Status == "active" && download.InfoHash != "" {
			peers, err := c.GetPeers(ctx, download.Gid)
			if err != nil {
				// Log error but continue with other downloads
				continue
			}
			allPeers[download.Gid] = peers
		}
	}

	return allPeers, nil
}
