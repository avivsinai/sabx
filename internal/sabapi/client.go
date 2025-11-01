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

// do performs a request and returns the raw HTTP response.
func (c *Client) do(ctx context.Context, mode string, params url.Values) (*http.Response, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("mode", mode)
	params.Set("apikey", c.apiKey)

	endpoint := c.baseURL + "/api"
	reqURL := endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, fmt.Errorf("sabnzbd API error: %s", resp.Status)
	}

	return resp, nil
}

// call performs a request and decodes JSON into dest if provided.
func (c *Client) call(ctx context.Context, mode string, params url.Values, dest any) error {
	if params == nil {
		params = url.Values{}
	}
	params.Set("output", "json")

	resp, err := c.do(ctx, mode, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if dest == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
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
		params.Set("password", opts.Password)
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
		fields["password"] = opts.Password
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

// AddLocalFile instructs SABnzbd to enqueue an NZB located on the server filesystem.
func (c *Client) AddLocalFile(ctx context.Context, remotePath string, opts AddOptions) (*AddResponse, error) {
	if strings.TrimSpace(remotePath) == "" {
		return nil, errors.New("remote path required")
	}
	params := url.Values{}
	params.Set("name", remotePath)
	if opts.Category != "" {
		params.Set("cat", opts.Category)
	}
	if opts.Priority != nil {
		params.Set("priority", fmt.Sprintf("%d", *opts.Priority))
	}
	if opts.Password != "" {
		params.Set("password", opts.Password)
	}
	if opts.Script != "" {
		params.Set("script", opts.Script)
	}
	if opts.Name != "" {
		params.Set("nzbname", opts.Name)
	}

	var resp AddResponse
	if err := c.call(ctx, "addlocalfile", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
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

// QueueDelete removes an item.
func (c *Client) QueueDelete(ctx context.Context, ids []string, withData bool) error {
	params := url.Values{}
	if len(ids) > 0 {
		params.Set("value", strings.Join(ids, ","))
	}
	if withData {
		params.Set("del_files", "1")
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
	params.Set("value", id)
	params.Set("value2", category)
	return c.call(ctx, "change_cat", params, nil)
}

// QueueSetScript sets the post-processing script for an item.
func (c *Client) QueueSetScript(ctx context.Context, id, script string) error {
	params := url.Values{}
	params.Set("value", id)
	params.Set("value2", script)
	return c.call(ctx, "change_script", params, nil)
}

// QueueRename changes the display name of a queue item.
func (c *Client) QueueRename(ctx context.Context, id, name, password string) error {
	params := url.Values{}
	params.Set("value", id)
	params.Set("value2", name)
	if password != "" {
		params.Set("value3", password)
	}
	return c.QueueAction(ctx, "rename", params)
}

// QueueSwitchPosition moves an item to an absolute position (0-based).
func (c *Client) QueueSwitchPosition(ctx context.Context, id string, position int) error {
	params := url.Values{}
	params.Set("value", id)
	params.Set("value2", fmt.Sprintf("%d", position))
	return c.call(ctx, "switch", params, nil)
}

// QueueSort sorts the queue by supported criteria.
func (c *Client) QueueSort(ctx context.Context, sortCrit, direction string) error {
	params := url.Values{}
	params.Set("sort", sortCrit)
	if direction != "" {
		params.Set("dir", direction)
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
	params.Set("name", "delete")
	switch {
	case all:
		params.Set("value", "all")
	case failed:
		params.Set("value", "failed")
	default:
		if len(ids) == 0 {
			return errors.New("no history ids provided")
		}
		params.Set("value", strings.Join(ids, ","))
	}
	return c.call(ctx, "history", params, nil)
}

// HistoryRetry re-queues a previously downloaded item.
func (c *Client) HistoryRetry(ctx context.Context, id string) error {
	params := url.Values{}
	params.Set("value", id)
	return c.call(ctx, "retry", params, nil)
}

// HistoryRetryAll re-queues all failed downloads.
func (c *Client) HistoryRetryAll(ctx context.Context) error {
	return c.call(ctx, "retry_all", nil, nil)
}

// HistoryMarkCompleted marks history entries as completed and removes incomplete data.
func (c *Client) HistoryMarkCompleted(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return errors.New("at least one history id required")
	}
	params := url.Values{}
	params.Set("name", "mark_as_completed")
	params.Set("value", strings.Join(ids, ","))
	return c.call(ctx, "history", params, nil)
}

// StatusDeleteOrphan deletes the specified orphaned job directory.
func (c *Client) StatusDeleteOrphan(ctx context.Context, path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("orphan path required")
	}
	params := url.Values{}
	params.Set("name", "delete_orphan")
	params.Set("value", path)
	return c.call(ctx, "status", params, nil)
}

// StatusDeleteAllOrphans removes all orphaned job directories.
func (c *Client) StatusDeleteAllOrphans(ctx context.Context) error {
	params := url.Values{}
	params.Set("name", "delete_all_orphan")
	return c.call(ctx, "status", params, nil)
}

// StatusAddOrphan re-adds an orphaned job back into the queue.
func (c *Client) StatusAddOrphan(ctx context.Context, path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("orphan path required")
	}
	params := url.Values{}
	params.Set("name", "add_orphan")
	params.Set("value", path)
	return c.call(ctx, "status", params, nil)
}

// StatusAddAllOrphans re-adds all orphaned jobs back into the queue.
func (c *Client) StatusAddAllOrphans(ctx context.Context) error {
	params := url.Values{}
	params.Set("name", "add_all_orphan")
	return c.call(ctx, "status", params, nil)
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

// ConfigDelete removes config entries by keyword (supports named sections).
func (c *Client) ConfigDelete(ctx context.Context, section, name string) error {
	params := url.Values{}
	params.Set("section", section)
	params.Set("keyword", name)
	return c.call(ctx, "del_config", params, nil)
}

// ConfigSetPause schedules SABnzbd to resume after the specified minutes.
func (c *Client) ConfigSetPause(ctx context.Context, minutes int) error {
	if minutes < 0 {
		return errors.New("minutes must be >= 0")
	}
	params := url.Values{}
	params.Set("name", "set_pause")
	params.Set("value", fmt.Sprintf("%d", minutes))
	return c.call(ctx, "config", params, nil)
}

type apiKeyEnvelope struct {
	APIKey string `json:"apikey"`
}

// ConfigRotateAPIKey regenerates the main API key.
func (c *Client) ConfigRotateAPIKey(ctx context.Context) (string, error) {
	params := url.Values{}
	params.Set("name", "set_apikey")

	var env apiKeyEnvelope
	if err := c.call(ctx, "config", params, &env); err != nil {
		return "", err
	}
	return env.APIKey, nil
}

type nzbKeyEnvelope struct {
	NZBKey string `json:"nzbkey"`
}

// ConfigRotateNZBKey regenerates the NZB key.
func (c *Client) ConfigRotateNZBKey(ctx context.Context) (string, error) {
	params := url.Values{}
	params.Set("name", "set_nzbkey")

	var env nzbKeyEnvelope
	if err := c.call(ctx, "config", params, &env); err != nil {
		return "", err
	}
	return env.NZBKey, nil
}

type boolEnvelope struct {
	Value Boolish `json:"value"`
}

// ConfigRegenerateCertificates recreates HTTPS certificates when using defaults.
func (c *Client) ConfigRegenerateCertificates(ctx context.Context) (bool, error) {
	params := url.Values{}
	params.Set("name", "regenerate_certs")

	var env boolEnvelope
	if err := c.call(ctx, "config", params, &env); err != nil {
		return false, err
	}
	return bool(env.Value), nil
}

type backupEnvelope struct {
	Value struct {
		Result  bool   `json:"result"`
		Message string `json:"message"`
	} `json:"value"`
}

// ConfigCreateBackup creates a configuration backup and returns its path.
func (c *Client) ConfigCreateBackup(ctx context.Context) (bool, string, error) {
	params := url.Values{}
	params.Set("name", "create_backup")

	var env backupEnvelope
	if err := c.call(ctx, "config", params, &env); err != nil {
		return false, "", err
	}
	return env.Value.Result, env.Value.Message, nil
}

// ConfigPurgeLogFiles deletes SABnzbd's historical log files.
func (c *Client) ConfigPurgeLogFiles(ctx context.Context) error {
	params := url.Values{}
	params.Set("name", "purge_log_files")
	return c.call(ctx, "config", params, nil)
}

// ConfigSetDefault resets misc configuration keys to defaults.
func (c *Client) ConfigSetDefault(ctx context.Context, keywords []string) error {
	if len(keywords) == 0 {
		return errors.New("provide at least one keyword")
	}
	params := url.Values{}
	for _, key := range keywords {
		params.Add("keyword", key)
	}
	return c.call(ctx, "set_config_default", params, nil)
}

// ServerControl triggers restart/shutdown.
func (c *Client) ServerControl(ctx context.Context, mode string) error {
	return c.call(ctx, mode, nil, nil)
}

// SpeedLimit sets the global speed limit.
func (c *Client) SpeedLimit(ctx context.Context, normalizedValue *string) error {
	params := url.Values{}
	params.Set("name", "speedlimit")
	if normalizedValue != nil {
		params.Set("value", *normalizedValue)
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

type translateEnvelope struct {
	Value string `json:"value"`
}

// Translate resolves SABnzbd translation strings for the active locale.
func (c *Client) Translate(ctx context.Context, key string) (string, error) {
	params := url.Values{}
	params.Set("value", key)

	var env translateEnvelope
	if err := c.call(ctx, "translate", params, &env); err != nil {
		return "", err
	}
	return env.Value, nil
}

// BrowseOptions configures parameters for SABnzbd's filesystem browser.
type BrowseOptions struct {
	Compact           bool
	ShowFiles         bool
	ShowHiddenFolders bool
}

// BrowseEntry models an entry returned by the browse API.
type BrowseEntry struct {
	Name        string `json:"name,omitempty"`
	Path        string `json:"path,omitempty"`
	Dir         bool   `json:"dir,omitempty"`
	CurrentPath string `json:"current_path,omitempty"`
}

type browseEnvelope struct {
	Paths json.RawMessage `json:"paths"`
}

// FullStatusOptions configures the fullstatus API call.
type FullStatusOptions struct {
	CalculatePerformance bool
	SkipDashboard        bool
}

type fullStatusEnvelope struct {
	Status map[string]any `json:"status"`
}

// FullStatus returns the comprehensive SABnzbd status payload.
func (c *Client) FullStatus(ctx context.Context, opts FullStatusOptions) (map[string]any, error) {
	params := url.Values{}
	if opts.CalculatePerformance {
		params.Set("calculate_performance", "1")
	}
	if opts.SkipDashboard {
		params.Set("skip_dashboard", "1")
	}

	var env fullStatusEnvelope
	if err := c.call(ctx, "fullstatus", params, &env); err != nil {
		return nil, err
	}
	return env.Status, nil
}

// Browse enumerates directories/files on the SABnzbd host.
func (c *Client) Browse(ctx context.Context, path string, opts BrowseOptions) ([]BrowseEntry, error) {
	params := url.Values{}
	if path != "" {
		params.Set("name", path)
	}
	if opts.Compact {
		params.Set("compact", "1")
	}
	if opts.ShowFiles {
		params.Set("show_files", "1")
	}
	if opts.ShowHiddenFolders {
		params.Set("show_hidden_folders", "1")
	}

	var env browseEnvelope
	if err := c.call(ctx, "browse", params, &env); err != nil {
		return nil, err
	}

	entries := []BrowseEntry{}
	if len(env.Paths) == 0 {
		return entries, nil
	}

	trimmed := strings.TrimSpace(string(env.Paths))
	if trimmed == "" {
		return entries, nil
	}

	useStruct := false
	if strings.HasPrefix(trimmed, "[") {
		for i := 1; i < len(trimmed); i++ {
			ch := trimmed[i]
			if ch == '{' {
				useStruct = true
				break
			}
			if ch == '[' || ch == ' ' || ch == '\n' || ch == '\t' || ch == '\r' {
				continue
			}
			break
		}
	}

	if useStruct {
		if err := json.Unmarshal(env.Paths, &entries); err == nil {
			return entries, nil
		}
	}

	var compact []string
	if err := json.Unmarshal(env.Paths, &compact); err != nil {
		return nil, err
	}

	for _, p := range compact {
		entries = append(entries, BrowseEntry{
			Name: p,
			Path: p,
		})
	}
	return entries, nil
}

// ServerStatsResponse captures aggregate usage metrics.
type ServerStatsResponse struct {
	Total   float64                       `json:"total"`
	Month   float64                       `json:"month"`
	Week    float64                       `json:"week"`
	Day     float64                       `json:"day"`
	Servers map[string]ServerUsageMetrics `json:"servers"`
}

// ServerUsageMetrics represents per-server throughput statistics.
type ServerUsageMetrics struct {
	Total           float64            `json:"total"`
	Month           float64            `json:"month"`
	Week            float64            `json:"week"`
	Day             float64            `json:"day"`
	Daily           map[string]float64 `json:"daily"`
	ArticlesTried   float64            `json:"articles_tried"`
	ArticlesSuccess float64            `json:"articles_success"`
}

// ServerStats fetches bandwidth utilisation statistics.
func (c *Client) ServerStats(ctx context.Context) (*ServerStatsResponse, error) {
	var stats ServerStatsResponse
	if err := c.call(ctx, "server_stats", nil, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
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

// Warning represents a SABnzbd warning entry.
type Warning struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	Time   int64  `json:"time"`
	Origin string `json:"origin"`
}

type warningsEnvelope struct {
	Warnings []Warning `json:"warnings"`
}

type statusEnvelope struct {
	Status Boolish `json:"status"`
	Error  string  `json:"error,omitempty"`
}

// Warnings retrieves current warnings.
func (c *Client) Warnings(ctx context.Context) ([]Warning, error) {
	var resp warningsEnvelope
	if err := c.call(ctx, "warnings", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Warnings, nil
}

// WarningsClear clears stored warnings.
func (c *Client) WarningsClear(ctx context.Context) error {
	params := url.Values{}
	params.Set("name", "clear")

	var resp statusEnvelope
	if err := c.call(ctx, "warnings", params, &resp); err != nil {
		return err
	}
	if !bool(resp.Status) {
		if resp.Error != "" {
			return fmt.Errorf("failed to clear warnings: %s", resp.Error)
		}
		return errors.New("failed to clear warnings")
	}
	return nil
}

// ShowLog returns the redacted SABnzbd log bundle.
func (c *Client) ShowLog(ctx context.Context) (string, error) {
	resp, err := c.do(ctx, "showlog", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type scriptsEnvelope struct {
	Scripts []string `json:"scripts"`
}

// GetScripts returns available post-processing scripts.
func (c *Client) GetScripts(ctx context.Context) ([]string, error) {
	var resp scriptsEnvelope
	if err := c.call(ctx, "get_scripts", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Scripts, nil
}

// QueueFile models an individual NZF file for a queue item.
type QueueFile struct {
	Filename string `json:"filename"`
	MBLeft   string `json:"mbleft"`
	MB       string `json:"mb"`
	Bytes    string `json:"bytes"`
	Age      string `json:"age"`
	NZFID    string `json:"nzf_id"`
	Status   string `json:"status"`
	Set      string `json:"set,omitempty"`
}

type filesEnvelope struct {
	Files []QueueFile `json:"files"`
}

// GetFiles lists the files belonging to a queue item.
func (c *Client) GetFiles(ctx context.Context, nzoID string) ([]QueueFile, error) {
	if strings.TrimSpace(nzoID) == "" {
		return nil, errors.New("nzo id required")
	}
	params := url.Values{}
	params.Set("value", nzoID)

	var resp filesEnvelope
	if err := c.call(ctx, "get_files", params, &resp); err != nil {
		return nil, err
	}
	return resp.Files, nil
}

// QueueDeleteFile removes an NZF entry from a queue item.
func (c *Client) QueueDeleteFile(ctx context.Context, nzoID, nzfID string) error {
	if strings.TrimSpace(nzoID) == "" || strings.TrimSpace(nzfID) == "" {
		return errors.New("nzo id and nzf id required")
	}
	params := url.Values{}
	params.Set("value", nzoID)
	params.Set("value2", nzfID)
	return c.QueueAction(ctx, "delete_nzf", params)
}

// QueueMoveFiles reorders NZF files within a queue item.
func (c *Client) QueueMoveFiles(ctx context.Context, action, nzoID string, nzfIDs []string, size *int) error {
	if strings.TrimSpace(action) == "" {
		return errors.New("action required")
	}
	if strings.TrimSpace(nzoID) == "" {
		return errors.New("nzo id required")
	}
	if len(nzfIDs) == 0 {
		return errors.New("at least one nzf id required")
	}
	params := url.Values{}
	params.Set("name", action)
	params.Set("value", nzoID)
	params.Set("nzf_ids", strings.Join(nzfIDs, ","))
	if size != nil {
		if *size <= 0 {
			return errors.New("size must be positive")
		}
		params.Set("size", fmt.Sprintf("%d", *size))
	}

	var resp statusEnvelope
	if err := c.call(ctx, "move_nzf_bulk", params, &resp); err != nil {
		return err
	}
	if !bool(resp.Status) {
		return errors.New("move operation rejected by SABnzbd")
	}
	return nil
}

// QueueSetCompleteAction configures the completion action executed when the queue empties.
func (c *Client) QueueSetCompleteAction(ctx context.Context, action string) error {
	params := url.Values{}
	if action != "" {
		params.Set("value", action)
	}
	return c.QueueAction(ctx, "change_complete_action", params)
}

// QueueChangeOptions updates the post-processing level for specific queue items.
func (c *Client) QueueChangeOptions(ctx context.Context, nzoIDs []string, ppLevel int) error {
	if len(nzoIDs) == 0 {
		return errors.New("at least one nzo id required")
	}
	if ppLevel < 0 {
		return errors.New("pp level must be non-negative")
	}
	params := url.Values{}
	params.Set("value", strings.Join(nzoIDs, ","))
	params.Set("value2", fmt.Sprintf("%d", ppLevel))

	var resp statusEnvelope
	if err := c.call(ctx, "change_opts", params, &resp); err != nil {
		return err
	}
	if !bool(resp.Status) {
		return errors.New("failed to update post-processing options")
	}
	return nil
}

// ServerConfig describes a configured news server.
type ServerConfig struct {
	Name         string  `json:"name"`
	DisplayName  string  `json:"displayname"`
	Host         string  `json:"host"`
	Port         int     `json:"port"`
	Timeout      int     `json:"timeout"`
	Username     string  `json:"username"`
	Password     string  `json:"password"`
	Connections  int     `json:"connections"`
	SSL          bool    `json:"ssl"`
	SSLVerify    int     `json:"ssl_verify"`
	SSLCiphers   string  `json:"ssl_ciphers"`
	Enable       bool    `json:"enable"`
	Required     bool    `json:"required"`
	Optional     bool    `json:"optional"`
	Retention    int     `json:"retention"`
	ExpireDate   string  `json:"expire_date"`
	Quota        string  `json:"quota"`
	UsageAtStart float64 `json:"usage_at_start"`
	Priority     int     `json:"priority"`
	Notes        string  `json:"notes"`
}

type serverConfigsEnvelope struct {
	Servers []ServerConfig `json:"servers"`
}

// ServerConfigs returns the configured news servers.
func (c *Client) ServerConfigs(ctx context.Context) ([]ServerConfig, error) {
	params := url.Values{}
	params.Set("section", "servers")

	var env serverConfigsEnvelope
	if err := c.call(ctx, "get_config", params, &env); err != nil {
		return nil, err
	}
	return env.Servers, nil
}

// Disconnect forces a temporary disconnect from all servers.
func (c *Client) Disconnect(ctx context.Context) error {
	return c.call(ctx, "disconnect", nil, nil)
}

// UnblockServer clears a temporarily blocked server.
func (c *Client) UnblockServer(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("server name required")
	}
	params := url.Values{}
	params.Set("name", "unblock_server")
	params.Set("value", name)
	return c.call(ctx, "status", params, nil)
}

// PausePostProcessing pauses post-processing globally.
func (c *Client) PausePostProcessing(ctx context.Context) error {
	return c.call(ctx, "pause_pp", nil, nil)
}

// ResumePostProcessing resumes post-processing globally.
func (c *Client) ResumePostProcessing(ctx context.Context) error {
	return c.call(ctx, "resume_pp", nil, nil)
}

// CancelPostProcessing cancels post-processing for the provided NZO IDs.
func (c *Client) CancelPostProcessing(ctx context.Context, nzoIDs []string) error {
	if len(nzoIDs) == 0 {
		return errors.New("at least one nzo id required")
	}
	params := url.Values{}
	params.Set("value", strings.Join(nzoIDs, ","))
	return c.call(ctx, "cancel_pp", params, nil)
}

// WatchedNow triggers an immediate watched-folder scan.
func (c *Client) WatchedNow(ctx context.Context) error {
	return c.call(ctx, "watched_now", nil, nil)
}

// ResetQuota clears SABnzbd's download quota tracking.
func (c *Client) ResetQuota(ctx context.Context) error {
	return c.call(ctx, "reset_quota", nil, nil)
}

type evalSortEnvelope struct {
	Result string `json:"result"`
}

// EvalSort evaluates a sorting expression for a sample job.
func (c *Client) EvalSort(ctx context.Context, expression string, opts EvalSortOptions) (string, error) {
	params := url.Values{}
	params.Set("sort_string", expression)
	if opts.JobName != "" {
		params.Set("job_name", opts.JobName)
	}
	if opts.MultipartLabel != "" {
		params.Set("multipart_label", opts.MultipartLabel)
	}

	var env evalSortEnvelope
	if err := c.call(ctx, "eval_sort", params, &env); err != nil {
		return "", err
	}
	return env.Result, nil
}

// EvalSortOptions customises eval_sort API parameters.
type EvalSortOptions struct {
	JobName        string
	MultipartLabel string
}

type gcStatsEnvelope struct {
	Value []string `json:"value"`
}

// GCStats returns SABnzbd's internal garbage-collector statistics.
func (c *Client) GCStats(ctx context.Context) ([]string, error) {
	var env gcStatsEnvelope
	if err := c.call(ctx, "gc_stats", nil, &env); err != nil {
		return nil, err
	}
	return env.Value, nil
}

// RestartRepair triggers queue repair and application restart.
func (c *Client) RestartRepair(ctx context.Context) error {
	return c.call(ctx, "restart_repair", nil, nil)
}

type testNotificationEnvelope struct {
	Status Boolish `json:"status"`
	Error  string  `json:"error,omitempty"`
}

// TestNotificationResult captures the outcome of a notification test call.
type TestNotificationResult struct {
	Success bool
	Message string
}

// TestNotification triggers SABnzbd's built-in notification testers (email, pushover, etc.).
func (c *Client) TestNotification(ctx context.Context, mode string, params url.Values) (*TestNotificationResult, error) {
	if params == nil {
		params = url.Values{}
	}
	var env testNotificationEnvelope
	if err := c.call(ctx, mode, params, &env); err != nil {
		return nil, err
	}
	return &TestNotificationResult{Success: bool(env.Status), Message: env.Error}, nil
}

// ServerTestParams configures a server connectivity test.
type ServerTestParams struct {
	Server      string
	Host        string
	Port        int
	Username    string
	Password    string
	Connections int
	Timeout     int
	SSL         bool
	SSLVerify   int
	SSLCiphers  string
}

// ServerTestResult captures the outcome of a server connectivity test.
type ServerTestResult struct {
	Result  bool   `json:"result"`
	Message string `json:"message"`
}

type serverTestEnvelope struct {
	Value ServerTestResult `json:"value"`
}

// TestServer performs SABnzbd's built-in server connectivity test.
func (c *Client) TestServer(ctx context.Context, params ServerTestParams) (*ServerTestResult, error) {
	if strings.TrimSpace(params.Server) == "" {
		return nil, errors.New("server identifier required")
	}

	req := url.Values{}
	req.Set("name", "test_server")
	req.Set("server", params.Server)
	req.Set("host", params.Host)
	if params.Port > 0 {
		req.Set("port", fmt.Sprintf("%d", params.Port))
	}
	req.Set("username", params.Username)
	req.Set("password", params.Password)
	if params.Connections > 0 {
		req.Set("connections", fmt.Sprintf("%d", params.Connections))
	}
	if params.Timeout > 0 {
		req.Set("timeout", fmt.Sprintf("%d", params.Timeout))
	}
	if params.SSL {
		req.Set("ssl", "1")
	} else {
		req.Set("ssl", "0")
	}
	if params.SSLVerify >= 0 {
		req.Set("ssl_verify", fmt.Sprintf("%d", params.SSLVerify))
	}
	if params.SSLCiphers != "" {
		req.Set("ssl_ciphers", params.SSLCiphers)
	}

	var env serverTestEnvelope
	if err := c.call(ctx, "config", req, &env); err != nil {
		return nil, err
	}
	return &env.Value, nil
}
