package sabapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T) (*Client, <-chan url.Values) {
	t.Helper()

	queries := make(chan url.Values, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries <- r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status": true}`))
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(server.URL, "apikey", WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	return client, queries
}

func newTestClientWithResponse(t *testing.T, body string) (*Client, <-chan url.Values) {
	t.Helper()

	queries := make(chan url.Values, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries <- r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		if body == "" {
			body = `{"status": true}`
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(server.URL, "apikey", WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	return client, queries
}

func requireQuery(t *testing.T, ch <-chan url.Values) url.Values {
	t.Helper()
	select {
	case q := <-ch:
		return q
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for request")
		return nil
	}
}

func TestWarningsUsesWarningsMode(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if _, err := client.Warnings(ctx); err != nil {
		t.Fatalf("Warnings returned error: %v", err)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "warnings" {
		t.Fatalf("expected mode=warnings, got %q", got)
	}
	if got := q.Get("output"); got != "json" {
		t.Fatalf("expected output=json, got %q", got)
	}
}

func TestWarningsClearSendsClearName(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.WarningsClear(ctx); err != nil {
		t.Fatalf("WarningsClear returned error: %v", err)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "warnings" {
		t.Fatalf("expected mode=warnings, got %q", got)
	}
	if got := q.Get("name"); got != "clear" {
		t.Fatalf("expected name=clear, got %q", got)
	}
}

func TestFullStatusSetsOptions(t *testing.T) {
	client, queries := newTestClientWithResponse(t, `{"status":{"loadavg":[0,0,0]}}`)
	ctx := context.Background()

	if _, err := client.FullStatus(ctx, FullStatusOptions{CalculatePerformance: true, SkipDashboard: true}); err != nil {
		t.Fatalf("FullStatus returned error: %v", err)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "fullstatus" {
		t.Fatalf("expected mode=fullstatus, got %q", got)
	}
	if got := q.Get("calculate_performance"); got != "1" {
		t.Fatalf("expected calculate_performance=1, got %q", got)
	}
	if got := q.Get("skip_dashboard"); got != "1" {
		t.Fatalf("expected skip_dashboard=1, got %q", got)
	}
}

func TestServerStatsMode(t *testing.T) {
	client, queries := newTestClientWithResponse(t, `{"total":1,"month":1,"week":1,"day":1,"servers":{}}`)
	ctx := context.Background()

	if _, err := client.ServerStats(ctx); err != nil {
		t.Fatalf("ServerStats returned error: %v", err)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "server_stats" {
		t.Fatalf("expected mode=server_stats, got %q", got)
	}
}

func TestServerConfigsMode(t *testing.T) {
	body := `{"servers":[{"name":"primary","displayname":"Primary","host":"news.example.com","port":119,"timeout":60,"username":"user","password":"******","connections":10,"ssl":false,"ssl_verify":3,"ssl_ciphers":"","enable":true,"required":false,"optional":false,"retention":0,"expire_date":"","quota":"","usage_at_start":0,"priority":0,"notes":""}]}`
	client, queries := newTestClientWithResponse(t, body)
	ctx := context.Background()

	if _, err := client.ServerConfigs(ctx); err != nil {
		t.Fatalf("ServerConfigs returned error: %v", err)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "get_config" {
		t.Fatalf("expected mode=get_config, got %q", got)
	}
	if got := q.Get("section"); got != "servers" {
		t.Fatalf("expected section=servers, got %q", got)
	}
}

func TestTestServerSendsParameters(t *testing.T) {
	client, queries := newTestClientWithResponse(t, `{"value":{"result":true,"message":"ok"}}`)
	ctx := context.Background()

	params := ServerTestParams{
		Server:      "primary",
		Host:        "news.example.com",
		Port:        563,
		Username:    "user",
		Password:    "******",
		Connections: 20,
		Timeout:     30,
		SSL:         true,
		SSLVerify:   2,
		SSLCiphers:  "HIGH",
	}

	if _, err := client.TestServer(ctx, params); err != nil {
		t.Fatalf("TestServer returned error: %v", err)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "config" {
		t.Fatalf("expected mode=config, got %q", got)
	}
	if got := q.Get("name"); got != "test_server" {
		t.Fatalf("expected name=test_server, got %q", got)
	}
	if got := q.Get("server"); got != "primary" {
		t.Fatalf("expected server=primary, got %q", got)
	}
	if got := q.Get("ssl"); got != "1" {
		t.Fatalf("expected ssl=1, got %q", got)
	}
	if got := q.Get("ssl_verify"); got != "2" {
		t.Fatalf("expected ssl_verify=2, got %q", got)
	}
	if got := q.Get("ssl_ciphers"); got != "HIGH" {
		t.Fatalf("expected ssl_ciphers=HIGH, got %q", got)
	}
	if got := q.Get("host"); got != "news.example.com" {
		t.Fatalf("expected host=news.example.com, got %q", got)
	}
	if got := q.Get("port"); got != "563" {
		t.Fatalf("expected port=563, got %q", got)
	}
	if got := q.Get("username"); got != "user" {
		t.Fatalf("expected username=user, got %q", got)
	}
	if got := q.Get("password"); got != "******" {
		t.Fatalf("expected password masked value, got %q", got)
	}
	if got := q.Get("connections"); got != "20" {
		t.Fatalf("expected connections=20, got %q", got)
	}
	if got := q.Get("timeout"); got != "30" {
		t.Fatalf("expected timeout=30, got %q", got)
	}
}

func TestDisconnectMode(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.Disconnect(ctx); err != nil {
		t.Fatalf("Disconnect returned error: %v", err)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "disconnect" {
		t.Fatalf("expected mode=disconnect, got %q", got)
	}
}

func TestUnblockServerMode(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.UnblockServer(ctx, "primary"); err != nil {
		t.Fatalf("UnblockServer returned error: %v", err)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "status" {
		t.Fatalf("expected mode=status, got %q", got)
	}
	if got := q.Get("name"); got != "unblock_server" {
		t.Fatalf("expected name=unblock_server, got %q", got)
	}
	if got := q.Get("value"); got != "primary" {
		t.Fatalf("expected value=primary, got %q", got)
	}
}

func TestPauseResumePostProcessing(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.PausePostProcessing(ctx); err != nil {
		t.Fatalf("PausePostProcessing returned error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "pause_pp" {
		t.Fatalf("expected mode=pause_pp, got %q", got)
	}

	if err := client.ResumePostProcessing(ctx); err != nil {
		t.Fatalf("ResumePostProcessing returned error: %v", err)
	}
	q = requireQuery(t, queries)
	if got := q.Get("mode"); got != "resume_pp" {
		t.Fatalf("expected mode=resume_pp, got %q", got)
	}
}

func TestCancelPostProcessing(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.CancelPostProcessing(ctx, []string{"A", "B"}); err != nil {
		t.Fatalf("CancelPostProcessing returned error: %v", err)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "cancel_pp" {
		t.Fatalf("expected mode=cancel_pp, got %q", got)
	}
	if got := q.Get("value"); got != "A,B" {
		t.Fatalf("expected value=A,B, got %q", got)
	}
}

func TestGetScriptsUsesCorrectMode(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if _, err := client.GetScripts(ctx); err != nil {
		t.Fatalf("GetScripts returned error: %v", err)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "get_scripts" {
		t.Fatalf("expected mode=get_scripts, got %q", got)
	}
}

func TestGetFilesRequiresNZOID(t *testing.T) {
	client, _ := newTestClient(t)
	ctx := context.Background()

	if _, err := client.GetFiles(ctx, ""); err == nil {
		t.Fatal("expected error when nzo id is empty")
	}
}

func TestGetFilesSendsValue(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if _, err := client.GetFiles(ctx, "NZ123"); err != nil {
		t.Fatalf("GetFiles returned error: %v", err)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "get_files" {
		t.Fatalf("expected mode=get_files, got %q", got)
	}
	if got := q.Get("value"); got != "NZ123" {
		t.Fatalf("expected value=NZ123, got %q", got)
	}
}

func TestQueueDeleteFileSendsIDs(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.QueueDeleteFile(ctx, "NZ123", "NZF456"); err != nil {
		t.Fatalf("QueueDeleteFile returned error: %v", err)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "queue" {
		t.Fatalf("expected mode=queue, got %q", got)
	}
	if got := q.Get("name"); got != "delete_nzf" {
		t.Fatalf("expected name=delete_nzf, got %q", got)
	}
	if got := q.Get("value"); got != "NZ123" {
		t.Fatalf("expected value=NZ123, got %q", got)
	}
	if got := q.Get("value2"); got != "NZF456" {
		t.Fatalf("expected value2=NZF456, got %q", got)
	}
}

func TestShowLogDoesNotForceJSON(t *testing.T) {
	queries := make(chan url.Values, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries <- r.URL.Query()
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("LOG DATA"))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "apikey", WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	ctx := context.Background()
	logData, err := client.ShowLog(ctx)
	if err != nil {
		t.Fatalf("ShowLog returned error: %v", err)
	}
	if logData != "LOG DATA" {
		t.Fatalf("expected log data to match, got %q", logData)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "showlog" {
		t.Fatalf("expected mode=showlog, got %q", got)
	}
	if got := q.Get("output"); got != "" {
		t.Fatalf("expected no output parameter, got %q", got)
	}
}

func TestTranslateUsesValue(t *testing.T) {
	client, queries := newTestClientWithResponse(t, `{"value":"Hallo"}`)
	ctx := context.Background()

	if _, err := client.Translate(ctx, "hello"); err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "translate" {
		t.Fatalf("expected mode=translate, got %q", got)
	}
	if got := q.Get("value"); got != "hello" {
		t.Fatalf("expected value=hello, got %q", got)
	}
}

func TestBrowseParameters(t *testing.T) {
	body := `{"paths":[{"current_path":"/tmp"},{"name":"files","path":"/tmp/files","dir":true}]}`
	client, queries := newTestClientWithResponse(t, body)
	ctx := context.Background()

	entries, err := client.Browse(ctx, "/tmp", BrowseOptions{ShowFiles: true, ShowHiddenFolders: true})
	if err != nil {
		t.Fatalf("Browse returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "browse" {
		t.Fatalf("expected mode=browse, got %q", got)
	}
	if got := q.Get("name"); got != "/tmp" {
		t.Fatalf("expected name=/tmp, got %q", got)
	}
	if got := q.Get("show_files"); got != "1" {
		t.Fatalf("expected show_files=1, got %q", got)
	}
	if got := q.Get("show_hidden_folders"); got != "1" {
		t.Fatalf("expected show_hidden_folders=1, got %q", got)
	}
}

func TestBrowseCompactFallback(t *testing.T) {
	client, queries := newTestClientWithResponse(t, `{"paths":["/tmp/a","/tmp/b"]}`)
	ctx := context.Background()

	entries, err := client.Browse(ctx, "", BrowseOptions{Compact: true})
	if err != nil {
		t.Fatalf("Browse returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	q := requireQuery(t, queries)
	if got := q.Get("compact"); got != "1" {
		t.Fatalf("expected compact=1, got %q", got)
	}
}

func TestAddLocalFileSendsParams(t *testing.T) {
	client, queries := newTestClientWithResponse(t, `{"status":true,"nzo_ids":["XYZ"]}`)
	ctx := context.Background()
	prio := 2

	resp, err := client.AddLocalFile(ctx, "/mnt/nzb/file.nzb", AddOptions{Category: "tv", Priority: &prio, Script: "none"})
	if err != nil {
		t.Fatalf("AddLocalFile returned error: %v", err)
	}
	if resp == nil || !resp.Success() {
		t.Fatalf("expected successful response, got %+v", resp)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "addlocalfile" {
		t.Fatalf("expected mode=addlocalfile, got %q", got)
	}
	if got := q.Get("name"); got != "/mnt/nzb/file.nzb" {
		t.Fatalf("expected name path, got %q", got)
	}
	if got := q.Get("cat"); got != "tv" {
		t.Fatalf("expected cat=tv, got %q", got)
	}
	if got := q.Get("priority"); got != "2" {
		t.Fatalf("expected priority=2, got %q", got)
	}
}

func TestQueueMoveFilesSetsParameters(t *testing.T) {
	client, queries := newTestClientWithResponse(t, `{"status":true}`)
	ctx := context.Background()
	size := 3

	if err := client.QueueMoveFiles(ctx, "up", "NZ123", []string{"A", "B"}, &size); err != nil {
		t.Fatalf("QueueMoveFiles returned error: %v", err)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "move_nzf_bulk" {
		t.Fatalf("expected mode=move_nzf_bulk, got %q", got)
	}
	if got := q.Get("name"); got != "up" {
		t.Fatalf("expected name=up, got %q", got)
	}
	if got := q.Get("value"); got != "NZ123" {
		t.Fatalf("expected value=NZ123, got %q", got)
	}
	if got := q.Get("nzf_ids"); got != "A,B" {
		t.Fatalf("expected nzf_ids=A,B, got %q", got)
	}
	if got := q.Get("size"); got != "3" {
		t.Fatalf("expected size=3, got %q", got)
	}
}

func TestQueueMoveFilesRejectsFailedStatus(t *testing.T) {
	client, _ := newTestClientWithResponse(t, `{"status":false,"error":"cannot move"}`)
	ctx := context.Background()

	err := client.QueueMoveFiles(ctx, "up", "NZ123", []string{"A"}, nil)
	if err == nil {
		t.Fatal("expected error when SAB reports failure")
	}
	if !strings.Contains(err.Error(), "rejected") {
		t.Fatalf("expected rejection error, got %v", err)
	}
}

func TestQueueSetCompleteActionUsesQueueMode(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.QueueSetCompleteAction(ctx, "shutdown_pc"); err != nil {
		t.Fatalf("QueueSetCompleteAction returned error: %v", err)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "queue" {
		t.Fatalf("expected mode=queue, got %q", got)
	}
	if got := q.Get("name"); got != "change_complete_action" {
		t.Fatalf("expected name=change_complete_action, got %q", got)
	}
	if got := q.Get("value"); got != "shutdown_pc" {
		t.Fatalf("expected value=shutdown_pc, got %q", got)
	}
}

func TestQueueChangeOptions(t *testing.T) {
	client, queries := newTestClientWithResponse(t, `{"status":true}`)
	ctx := context.Background()

	if err := client.QueueChangeOptions(ctx, []string{"NZ1", "NZ2"}, 2); err != nil {
		t.Fatalf("QueueChangeOptions returned error: %v", err)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "change_opts" {
		t.Fatalf("expected mode=change_opts, got %q", got)
	}
	if got := q.Get("value"); got != "NZ1,NZ2" {
		t.Fatalf("expected value NZ1,NZ2, got %q", got)
	}
	if got := q.Get("value2"); got != "2" {
		t.Fatalf("expected value2=2, got %q", got)
	}
}

func TestHistoryMarkCompleted(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.HistoryMarkCompleted(ctx, []string{"H1"}); err != nil {
		t.Fatalf("HistoryMarkCompleted returned error: %v", err)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "history" {
		t.Fatalf("expected mode=history, got %q", got)
	}
	if got := q.Get("name"); got != "mark_as_completed" {
		t.Fatalf("expected name=mark_as_completed, got %q", got)
	}
	if got := q.Get("value"); got != "H1" {
		t.Fatalf("expected value=H1, got %q", got)
	}
}

func TestStatusOrphanOperations(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.StatusDeleteOrphan(ctx, "Job1"); err != nil {
		t.Fatalf("StatusDeleteOrphan error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("name"); got != "delete_orphan" {
		t.Fatalf("expected name=delete_orphan, got %q", got)
	}

	if err := client.StatusDeleteAllOrphans(ctx); err != nil {
		t.Fatalf("StatusDeleteAllOrphans error: %v", err)
	}
	q = requireQuery(t, queries)
	if got := q.Get("name"); got != "delete_all_orphan" {
		t.Fatalf("expected delete_all_orphan, got %q", got)
	}

	if err := client.StatusAddOrphan(ctx, "Job1"); err != nil {
		t.Fatalf("StatusAddOrphan error: %v", err)
	}
	q = requireQuery(t, queries)
	if got := q.Get("name"); got != "add_orphan" {
		t.Fatalf("expected add_orphan, got %q", got)
	}

	if err := client.StatusAddAllOrphans(ctx); err != nil {
		t.Fatalf("StatusAddAllOrphans error: %v", err)
	}
	q = requireQuery(t, queries)
	if got := q.Get("name"); got != "add_all_orphan" {
		t.Fatalf("expected add_all_orphan, got %q", got)
	}
}

func TestResetQuotaMode(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.ResetQuota(ctx); err != nil {
		t.Fatalf("ResetQuota returned error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "reset_quota" {
		t.Fatalf("expected mode=reset_quota, got %q", got)
	}
}

func TestEvalSortParams(t *testing.T) {
	client, queries := newTestClientWithResponse(t, `{"result":"Show/Release"}`)
	ctx := context.Background()

	if _, err := client.EvalSort(ctx, "%dn", EvalSortOptions{JobName: "Example", MultipartLabel: "Part"}); err != nil {
		t.Fatalf("EvalSort returned error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "eval_sort" {
		t.Fatalf("expected mode=eval_sort, got %q", got)
	}
	if got := q.Get("sort_string"); got != "%dn" {
		t.Fatalf("expected sort_string=%%dn, got %q", got)
	}
	if got := q.Get("job_name"); got != "Example" {
		t.Fatalf("expected job_name=Example, got %q", got)
	}
	if got := q.Get("multipart_label"); got != "Part" {
		t.Fatalf("expected multipart_label=Part, got %q", got)
	}
}

func TestGCStatsMode(t *testing.T) {
	client, queries := newTestClientWithResponse(t, `{"value":[]}`)
	ctx := context.Background()
	if _, err := client.GCStats(ctx); err != nil {
		t.Fatalf("GCStats returned error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "gc_stats" {
		t.Fatalf("expected mode=gc_stats, got %q", got)
	}
}

func TestRestartRepairMode(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()
	if err := client.RestartRepair(ctx); err != nil {
		t.Fatalf("RestartRepair returned error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "restart_repair" {
		t.Fatalf("expected mode=restart_repair, got %q", got)
	}
}

func TestConfigSetPause(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()
	if err := client.ConfigSetPause(ctx, 15); err != nil {
		t.Fatalf("ConfigSetPause returned error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "config" {
		t.Fatalf("expected mode=config, got %q", got)
	}
	if got := q.Get("name"); got != "set_pause" {
		t.Fatalf("expected name=set_pause, got %q", got)
	}
	if got := q.Get("value"); got != "15" {
		t.Fatalf("expected value=15, got %q", got)
	}
}

func TestConfigRotateAPIKey(t *testing.T) {
	client, queries := newTestClientWithResponse(t, `{"apikey":"new-key"}`)
	ctx := context.Background()
	key, err := client.ConfigRotateAPIKey(ctx)
	if err != nil {
		t.Fatalf("ConfigRotateAPIKey error: %v", err)
	}
	if key != "new-key" {
		t.Fatalf("expected apikey new-key, got %q", key)
	}
	q := requireQuery(t, queries)
	if got := q.Get("name"); got != "set_apikey" {
		t.Fatalf("expected name=set_apikey, got %q", got)
	}
}

func TestConfigRotateNZBKey(t *testing.T) {
	client, queries := newTestClientWithResponse(t, `{"nzbkey":"nk"}`)
	ctx := context.Background()
	key, err := client.ConfigRotateNZBKey(ctx)
	if err != nil {
		t.Fatalf("ConfigRotateNZBKey error: %v", err)
	}
	if key != "nk" {
		t.Fatalf("expected nzbkey nk, got %q", key)
	}
	q := requireQuery(t, queries)
	if got := q.Get("name"); got != "set_nzbkey" {
		t.Fatalf("expected name=set_nzbkey, got %q", got)
	}
}

func TestConfigRegenerateCertificates(t *testing.T) {
	client, queries := newTestClientWithResponse(t, `{"value":true}`)
	ctx := context.Background()
	if _, err := client.ConfigRegenerateCertificates(ctx); err != nil {
		t.Fatalf("ConfigRegenerateCertificates error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("name"); got != "regenerate_certs" {
		t.Fatalf("expected name=regenerate_certs, got %q", got)
	}
}

func TestConfigCreateBackup(t *testing.T) {
	client, queries := newTestClientWithResponse(t, `{"value":{"result":true,"message":"/tmp/backup.zip"}}`)
	ctx := context.Background()
	if _, _, err := client.ConfigCreateBackup(ctx); err != nil {
		t.Fatalf("ConfigCreateBackup error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("name"); got != "create_backup" {
		t.Fatalf("expected name=create_backup, got %q", got)
	}
}

func TestConfigPurgeLogFiles(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()
	if err := client.ConfigPurgeLogFiles(ctx); err != nil {
		t.Fatalf("ConfigPurgeLogFiles error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("name"); got != "purge_log_files" {
		t.Fatalf("expected name=purge_log_files, got %q", got)
	}
}

func TestConfigSetDefault(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()
	if err := client.ConfigSetDefault(ctx, []string{"language", "dirscan_dir"}); err != nil {
		t.Fatalf("ConfigSetDefault error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "set_config_default" {
		t.Fatalf("expected mode=set_config_default, got %q", got)
	}
}

func TestWatchedNowMode(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()
	if err := client.WatchedNow(ctx); err != nil {
		t.Fatalf("WatchedNow error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "watched_now" {
		t.Fatalf("expected mode=watched_now, got %q", got)
	}
}

func TestTestNotificationUsesMode(t *testing.T) {
	client, queries := newTestClientWithResponse(t, `{"status":true,"error":""}`)
	ctx := context.Background()
	result, err := client.TestNotification(ctx, "test_email", nil)
	if err != nil {
		t.Fatalf("TestNotification error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success=true, got %v", result.Success)
	}
	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "test_email" {
		t.Fatalf("expected mode=test_email, got %q", got)
	}
}

func TestTestNotificationParsesResultMessage(t *testing.T) {
	client, _ := newTestClientWithResponse(t, `{"status":false,"error":"invalid configuration"}`)
	ctx := context.Background()
	result, err := client.TestNotification(ctx, "test_email", nil)
	if err != nil {
		t.Fatalf("TestNotification error: %v", err)
	}
	if result.Success {
		t.Fatalf("expected success=false, got true")
	}
	if result.Message != "invalid configuration" {
		t.Fatalf("expected message to be propagated, got %q", result.Message)
	}
}

func TestQueueDeleteJoinsIDs(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.QueueDelete(ctx, []string{"A", "B"}, true); err != nil {
		t.Fatalf("QueueDelete returned error: %v", err)
	}

	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "queue" {
		t.Fatalf("expected mode=queue, got %q", got)
	}
	if got := q.Get("name"); got != "delete" {
		t.Fatalf("expected name=delete, got %q", got)
	}
	if got := q.Get("value"); got != "A,B" {
		t.Fatalf("expected value A,B, got %q", got)
	}
	if got := q.Get("del_files"); got != "1" {
		t.Fatalf("expected del_files=1, got %q", got)
	}
	if _, ok := q["value2"]; ok {
		t.Fatalf("unexpected value2 param: %v", q["value2"])
	}
}

func TestQueueSetCategoryUsesChangeCat(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.QueueSetCategory(ctx, "NZ123", "tv"); err != nil {
		t.Fatalf("QueueSetCategory returned error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "change_cat" {
		t.Fatalf("expected mode=change_cat, got %q", got)
	}
	if got := q.Get("value"); got != "NZ123" {
		t.Fatalf("expected value=NZ123, got %q", got)
	}
	if got := q.Get("value2"); got != "tv" {
		t.Fatalf("expected value2=tv, got %q", got)
	}
}

func TestQueueSwitchUsesSwitchMode(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.QueueSwitchPosition(ctx, "NZ555", 4); err != nil {
		t.Fatalf("QueueSwitchPosition returned error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "switch" {
		t.Fatalf("expected mode=switch, got %q", got)
	}
	if got := q.Get("value"); got != "NZ555" {
		t.Fatalf("expected value=NZ555, got %q", got)
	}
	if got := q.Get("value2"); got != "4" {
		t.Fatalf("expected value2=4, got %q", got)
	}
}

func TestQueueSortParameters(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.QueueSort(ctx, "avg_age", "desc"); err != nil {
		t.Fatalf("QueueSort returned error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "queue" {
		t.Fatalf("expected mode=queue, got %q", got)
	}
	if got := q.Get("name"); got != "sort" {
		t.Fatalf("expected name=sort, got %q", got)
	}
	if got := q.Get("sort"); got != "avg_age" {
		t.Fatalf("expected sort=avg_age, got %q", got)
	}
	if got := q.Get("dir"); got != "desc" {
		t.Fatalf("expected dir=desc, got %q", got)
	}
	if got := q.Get("value"); got != "" {
		t.Fatalf("expected no value param, got %q", got)
	}
}

func TestHistoryRetryUsesRetryMode(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.HistoryRetry(ctx, "NZ321"); err != nil {
		t.Fatalf("HistoryRetry returned error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "retry" {
		t.Fatalf("expected mode=retry, got %q", got)
	}
	if got := q.Get("value"); got != "NZ321" {
		t.Fatalf("expected value=NZ321, got %q", got)
	}
}

func TestHistoryRetryAll(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.HistoryRetryAll(ctx); err != nil {
		t.Fatalf("HistoryRetryAll returned error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "retry_all" {
		t.Fatalf("expected mode=retry_all, got %q", got)
	}
}

func TestDeleteHistoryAll(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.DeleteHistory(ctx, nil, false, true); err != nil {
		t.Fatalf("DeleteHistory returned error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "history" {
		t.Fatalf("expected mode=history, got %q", got)
	}
	if got := q.Get("name"); got != "delete" {
		t.Fatalf("expected name=delete, got %q", got)
	}
	if got := q.Get("value"); got != "all" {
		t.Fatalf("expected value=all, got %q", got)
	}
}

func TestConfigSetNamedSection(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	values := url.Values{}
	values.Set("enable", "1")
	values.Set("cat", "tv")

	if err := client.ConfigSet(ctx, "rss", "FeedOne", values); err != nil {
		t.Fatalf("ConfigSet returned error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "set_config" {
		t.Fatalf("expected mode=set_config, got %q", got)
	}
	if got := q.Get("section"); got != "rss" {
		t.Fatalf("expected section=rss, got %q", got)
	}
	if got := q.Get("name"); got != "FeedOne" {
		t.Fatalf("expected name=FeedOne, got %q", got)
	}
	if got := q.Get("enable"); got != "1" {
		t.Fatalf("expected enable=1, got %q", got)
	}
	if got := q.Get("cat"); got != "tv" {
		t.Fatalf("expected cat=tv, got %q", got)
	}
	if _, ok := q["keyword"]; ok {
		t.Fatalf("unexpected keyword param in request: %v", q["keyword"])
	}
}

func TestConfigDeleteUsesKeyword(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.ConfigDelete(ctx, "rss", "FeedOne"); err != nil {
		t.Fatalf("ConfigDelete returned error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "del_config" {
		t.Fatalf("expected mode=del_config, got %q", got)
	}
	if got := q.Get("section"); got != "rss" {
		t.Fatalf("expected section=rss, got %q", got)
	}
	if got := q.Get("keyword"); got != "FeedOne" {
		t.Fatalf("expected keyword=FeedOne, got %q", got)
	}
	if got := q.Get("name"); got != "" {
		t.Fatalf("did not expect name param, got %q", got)
	}
}

func TestSpeedLimitUsesProvidedValue(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	val := "400K"
	if err := client.SpeedLimit(ctx, &val); err != nil {
		t.Fatalf("SpeedLimit returned error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("mode"); got != "config" {
		t.Fatalf("expected mode=config, got %q", got)
	}
	if got := q.Get("name"); got != "speedlimit" {
		t.Fatalf("expected name=speedlimit, got %q", got)
	}
	if got := q.Get("value"); got != "400K" {
		t.Fatalf("expected value=400K, got %q", got)
	}
}

func TestSpeedLimitRemove(t *testing.T) {
	client, queries := newTestClient(t)
	ctx := context.Background()

	if err := client.SpeedLimit(ctx, nil); err != nil {
		t.Fatalf("SpeedLimit returned error: %v", err)
	}
	q := requireQuery(t, queries)
	if got := q.Get("value"); got != "0" {
		t.Fatalf("expected value=0, got %q", got)
	}
}
