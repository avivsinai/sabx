package sabapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
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
