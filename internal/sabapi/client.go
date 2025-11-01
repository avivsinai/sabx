package sabapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout = 15 * time.Second
)

// Client wraps SABnzbd's HTTP API.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// Option configures the Client.
type Option func(*Client)

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.http = httpClient
	}
}

// NewClient constructs an API client.
func NewClient(baseURL, apiKey string, opts ...Option) (*Client, error) {
	if baseURL == "" {
		return nil, errors.New("base URL required")
	}
	if apiKey == "" {
		return nil, errors.New("API key required")
	}

	cleaned := strings.TrimSuffix(baseURL, "/")
	client := &Client{
		baseURL: cleaned,
		apiKey:  apiKey,
		http: &http.Client{
			Timeout: defaultTimeout,
		},
	}
	for _, opt := range opts {
		opt(client)
	}
	return client, nil
}

// call performs a request and decodes JSON into dest if provided.
func (c *Client) call(ctx context.Context, mode string, params url.Values, dest any) error {
	if params == nil {
		params = url.Values{}
	}
	params.Set("mode", mode)
	params.Set("output", "json")
	params.Set("apikey", c.apiKey)

	endpoint := c.baseURL + "/api"
	reqURL := endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("sabnzbd API error: %s", resp.Status)
	}

	if dest == nil {
		return nil
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(dest); err != nil {
		return err
	}

	return nil
}

// Queue returns current queue state.
func (c *Client) Queue(ctx context.Context, start, limit int, search string) (*QueueResponse, error) {
	params := url.Values{}
	if start > 0 {
		params.Set("start", fmt.Sprintf("%d", start))
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if search != "" {
		params.Set("search", search)
	}

	var resp QueueEnvelope
	if err := c.call(ctx, "queue", params, &resp); err != nil {
		return nil, err
	}
	return &resp.Queue, nil
}

// QueueResponse models the queue API payload.
type QueueResponse struct {
	Slots      []QueueSlot `json:"slots"`
	Status     string      `json:"status"`
	Paused     bool        `json:"paused"`
	Speed      string      `json:"kbpersec"`
	SpeedLimit string      `json:"speedlimit"`
	SizeMB     string      `json:"mb"`
	MBLeft     string      `json:"mbleft"`
	TimeLeft   string      `json:"timeleft"`
	Eta        string      `json:"eta"`
}

// QueueEnvelope is used for decoding the JSON container.
type QueueEnvelope struct {
	Queue QueueResponse `json:"queue"`
}

// QueueSlot represents an item in the queue.
type QueueSlot struct {
	NZOID      string `json:"nzo_id"`
	Filename   string `json:"filename"`
	Status     string `json:"status"`
	Paused     bool   `json:"paused"`
	Speed      string `json:"kbpersec"`
	MB         string `json:"mb"`
	MBLeft     string `json:"mbleft"`
	Percentage string `json:"percentage"`
	Priority   string `json:"priority"`
	Category   string `json:"cat"`
	Script     string `json:"script"`
	Eta        string `json:"eta"`
	TimeLeft   string `json:"timeleft"`
	StageLog   []struct {
		Stage string `json:"stage"`
		Log   string `json:"log"`
	} `json:"stage_log"`
}

// QueueAction executes queue-affecting commands.
func (c *Client) QueueAction(ctx context.Context, name string, extra url.Values) error {
	params := url.Values{}
	params.Set("name", name)
	for key, vals := range extra {
		for _, v := range vals {
			params.Add(key, v)
		}
	}
	return c.call(ctx, "queue", params, nil)
}

// AddURL adds an NZB by URL.
func (c *Client) AddURL(ctx context.Context, nzbURL string, opts AddOptions) (*AddResponse, error) {
	params := url.Values{}
	params.Set("name", nzbURL)
	if opts.Category != "" {
		params.Set("cat", opts.Category)
	}
	if opts.Priority != nil {
		params.Set("priority", fmt.Sprintf("%d", *opts.Priority))
	}
	if opts.Password != "" {
		params.Set("pp", opts.Password)
	}
	if opts.Script != "" {
		params.Set("script", opts.Script)
	}
	if opts.Name != "" {
		params.Set("nzbname", opts.Name)
	}
	var resp AddResponse
	if err := c.call(ctx, "addurl", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// AddFile uploads an NZB file via multipart form upload.
func (c *Client) AddFile(ctx context.Context, path string, opts AddOptions) (*AddResponse, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fields := map[string]string{
		"mode":   "addfile",
		"apikey": c.apiKey,
		"output": "json",
	}
	if opts.Category != "" {
		fields["cat"] = opts.Category
	}
	if opts.Password != "" {
		fields["pp"] = opts.Password
	}
	if opts.Script != "" {
		fields["script"] = opts.Script
	}
	if opts.Priority != nil {
		fields["priority"] = fmt.Sprintf("%d", *opts.Priority)
	}
	if opts.Name != "" {
		fields["nzbname"] = opts.Name
	}

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, err
		}
	}

	part, err := writer.CreateFormFile("nzbfile", filepath.Base(path))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("sabnzbd API error: %s", resp.Status)
	}

	var addResp AddResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&addResp); err != nil {
		return nil, err
	}
	return &addResp, nil
}

// QueuePause pauses an item or the entire queue if id empty.
func (c *Client) QueuePause(ctx context.Context, id string) error {
	if id == "" {
		return c.call(ctx, "pause", nil, nil)
	}
	params := url.Values{}
	params.Set("value", id)
	return c.QueueAction(ctx, "pause", params)
}

// QueueResume resumes an item or the entire queue if id empty.
func (c *Client) QueueResume(ctx context.Context, id string) error {
	if id == "" {
		return c.call(ctx, "resume", nil, nil)
	}
	params := url.Values{}
	params.Set("value", id)
	return c.QueueAction(ctx, "resume", params)
}

// QueueRetry retries an item.
func (c *Client) QueueRetry(ctx context.Context, id string) error {
	params := url.Values{}
	params.Set("value", id)
	return c.QueueAction(ctx, "retry", params)
}

// QueueDelete removes an item.
func (c *Client) QueueDelete(ctx context.Context, ids []string, withData bool) error {
	params := url.Values{}
	for _, id := range ids {
		params.Add("value", id)
	}
	if withData {
		params.Set("del_files", "1")
		params.Set("value2", "1")
	}
	return c.QueueAction(ctx, "delete", params)
}

// QueueSetPriority sets item priority (-1 low,0 normal,1 high,2 force).
func (c *Client) QueueSetPriority(ctx context.Context, id string, priority int) error {
	params := url.Values{}
	params.Set("value", fmt.Sprintf("%d", priority))
	params.Set("value2", id)
	return c.QueueAction(ctx, "priority", params)
}

// QueueSetCategory updates an item's category.
func (c *Client) QueueSetCategory(ctx context.Context, id, category string) error {
	params := url.Values{}
	params.Set("value", category)
	params.Set("value2", id)
	return c.QueueAction(ctx, "set_category", params)
}

// QueueSetScript sets the post-processing script for an item.
func (c *Client) QueueSetScript(ctx context.Context, id, script string) error {
	params := url.Values{}
	params.Set("value", script)
	params.Set("value2", id)
	return c.QueueAction(ctx, "set_script", params)
}

// QueueSetPassword sets the decryption password for an item.
func (c *Client) QueueSetPassword(ctx context.Context, id, password string) error {
	params := url.Values{}
	params.Set("value", password)
	params.Set("value2", id)
	return c.QueueAction(ctx, "set_password", params)
}

// QueueRename changes the display name of a queue item.
func (c *Client) QueueRename(ctx context.Context, id, name string) error {
	params := url.Values{}
	params.Set("value", name)
	params.Set("value2", id)
	return c.QueueAction(ctx, "rename", params)
}

// QueueSwitchPosition moves an item to an absolute position (0-based).
func (c *Client) QueueSwitchPosition(ctx context.Context, id string, position int) error {
	params := url.Values{}
	params.Set("value", id)
	params.Set("value2", fmt.Sprintf("%d", position))
	return c.QueueAction(ctx, "switch", params)
}

// QueueSort sorts the queue by supported criteria.
func (c *Client) QueueSort(ctx context.Context, sortCrit, direction string) error {
	params := url.Values{}
	params.Set("value", sortCrit)
	if direction != "" {
		params.Set("value2", direction)
	}
	return c.QueueAction(ctx, "sort", params)
}

// History fetches SAB history.
func (c *Client) History(ctx context.Context, failed bool, limit int) (*HistoryResponse, error) {
	params := url.Values{}
	if failed {
		params.Set("failed", "1")
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}

	var resp HistoryEnvelope
	if err := c.call(ctx, "history", params, &resp); err != nil {
		return nil, err
	}
	return &resp.History, nil
}

// HistoryResponse wraps history items.
type HistoryResponse struct {
	Slots []HistorySlot `json:"slots"`
}

// HistoryEnvelope decodes the outer container.
type HistoryEnvelope struct {
	History HistoryResponse `json:"history"`
}

// HistorySlot describes a history entry.
type HistorySlot struct {
	NZOID    string `json:"nzo_id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Category string `json:"category"`
	StageLog []struct {
		Stage string `json:"stage"`
		Log   string `json:"log"`
	} `json:"stage_log"`
	Completed string `json:"completed"`
}

// DeleteHistory removes items from history.
func (c *Client) DeleteHistory(ctx context.Context, ids []string, failed, all bool) error {
	params := url.Values{}
	switch {
	case all:
		params.Set("name", "delete_all")
	case failed:
		params.Set("name", "delete_failed")
	default:
		params.Set("name", "delete")
		for _, id := range ids {
			params.Add("value", id)
		}
	}
	return c.call(ctx, "history", params, nil)
}

// HistoryRetry re-queues a previously downloaded item.
func (c *Client) HistoryRetry(ctx context.Context, id string) error {
	params := url.Values{}
	params.Set("name", "retry")
	params.Set("value", id)
	return c.call(ctx, "history", params, nil)
}

// ConfigGet retrieves configuration for a given section (and optional key).
func (c *Client) ConfigGet(ctx context.Context, section, key string) (map[string]any, error) {
	params := url.Values{}
	params.Set("section", section)
	if key != "" {
		params.Set("keyword", key)
	}
	resp := map[string]any{}
	if err := c.call(ctx, "get_config", params, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ConfigSet sets configuration values.
func (c *Client) ConfigSet(ctx context.Context, section, name string, values url.Values) error {
	params := url.Values{}
	params.Set("section", section)
	if name != "" {
		params.Set("name", name)
	}
	for key, vals := range values {
		for _, v := range vals {
			params.Add(key, v)
		}
	}
	return c.call(ctx, "set_config", params, nil)
}

// ConfigDelete removes named config items.
func (c *Client) ConfigDelete(ctx context.Context, section, name string) error {
	params := url.Values{}
	params.Set("section", section)
	params.Set("keyword", name)
	return c.call(ctx, "del_config", params, nil)
}

// ConfigDeleteNamed removes a named configuration item (rss feed, server, etc.).
func (c *Client) ConfigDeleteNamed(ctx context.Context, section, name string) error {
	params := url.Values{}
	params.Set("section", section)
	params.Set("name", name)
	return c.call(ctx, "del_config", params, nil)
}

// ServerControl triggers restart/shutdown.
func (c *Client) ServerControl(ctx context.Context, mode string) error {
	return c.call(ctx, mode, nil, nil)
}

// SpeedLimit sets the global speed limit.
func (c *Client) SpeedLimit(ctx context.Context, limitMBps *float64) error {
	params := url.Values{}
	params.Set("name", "speedlimit")
	if limitMBps != nil {
		// SAB expects KB/s. Convert from Mbps.
		kbps := (*limitMBps * 1000) / 8
		params.Set("value", fmt.Sprintf("%.0f", kbps))
	} else {
		params.Set("value", "0")
	}
	return c.call(ctx, "config", params, nil)
}

// Status returns server status metadata.
func (c *Client) Status(ctx context.Context) (*StatusResponse, error) {
	var resp StatusResponse
	if err := c.call(ctx, "status", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Version returns SABnzbd version info.
func (c *Client) Version(ctx context.Context) (*VersionResponse, error) {
	var resp VersionResponse
	if err := c.call(ctx, "version", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StatusResponse provides server state.
type StatusResponse struct {
	Version    string `json:"version"`
	Paused     bool   `json:"paused"`
	Speed      string `json:"kbpersec"`
	SpeedLimit string `json:"speedlimit"`
}

// VersionResponse wraps version details.
type VersionResponse struct {
	Version string `json:"version"`
}

// RSSNow triggers RSS fetch.
func (c *Client) RSSNow(ctx context.Context, name string) error {
	params := url.Values{}
	if name != "" {
		params.Set("name", name)
	}
	return c.call(ctx, "rss_now", params, nil)
}

// RSSList fetches RSS config.
func (c *Client) RSSList(ctx context.Context) (map[string]any, error) {
	return c.ConfigGet(ctx, "rss", "")
}

// SchedulerList fetches scheduler entries.
func (c *Client) SchedulerList(ctx context.Context) (map[string]any, error) {
	return c.ConfigGet(ctx, "scheduler", "")
}

// CategoriesList fetches category config.
func (c *Client) CategoriesList(ctx context.Context) (map[string]any, error) {
	return c.ConfigGet(ctx, "categories", "")
}

// AddOptions are common for queue operations.
type AddOptions struct {
	Category string
	Priority *int
	Password string
	Script   string
	Name     string
}

// AddResponse represents addurl/addfile response payloads from SABnzbd.
type AddResponse struct {
	Status  Boolish  `json:"status"`
	NZOIDs  []string `json:"nzo_ids"`
	Error   string   `json:"error"`
	Message string   `json:"msg"`
}

// Success reports whether SABnzbd accepted the add operation.
func (a AddResponse) Success() bool {
	return bool(a.Status)
}

// Boolish handles SABnzbd's inconsistent boolean encoding.
type Boolish bool

// UnmarshalJSON supports boolean or string values.
func (b *Boolish) UnmarshalJSON(data []byte) error {
	dataStr := strings.TrimSpace(string(data))
	switch dataStr {
	case "true", "True", "TRUE", "1":
		*b = Boolish(true)
		return nil
	case "false", "False", "FALSE", "0":
		*b = Boolish(false)
		return nil
	}
	if len(dataStr) >= 2 && dataStr[0] == '"' && dataStr[len(dataStr)-1] == '"' {
		val := strings.Trim(string(dataStr[1:len(dataStr)-1]), " ")
		switch strings.ToLower(val) {
		case "true", "yes", "ok", "1":
			*b = Boolish(true)
			return nil
		case "false", "no", "0":
			*b = Boolish(false)
			return nil
		}
	}
	parsed, err := strconv.ParseBool(strings.Trim(string(dataStr), "\""))
	if err != nil {
		return err
	}
	*b = Boolish(parsed)
	return nil
}
