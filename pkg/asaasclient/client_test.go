package asaasclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := New("$aact_hmlg_secret", Sandbox, WithBaseURL(srv.URL+"/v3"), WithUserAgent("test-agent"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestDoSendsHeadersAndBody(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("access_token"); got != "$aact_hmlg_secret" {
			t.Errorf("access_token = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		if got := r.Header.Get("User-Agent"); got != "test-agent" {
			t.Errorf("User-Agent = %q", got)
		}
		if r.Method != http.MethodPost || r.URL.Path != "/v3/customers" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		if string(b) != `{"name":"Ana"}` {
			t.Errorf("body = %s", b)
		}
		fmt.Fprint(w, `{"id":"cus_1","value":12345678901234567890}`)
	})
	resp, err := c.Post(context.Background(), "/customers", map[string]string{"name": "Ana"})
	if err != nil {
		t.Fatal(err)
	}
	// Raw body is preserved byte-for-byte (no float64 rounding).
	if string(resp.Body) != `{"id":"cus_1","value":12345678901234567890}` {
		t.Errorf("body = %s", resp.Body)
	}
}

func TestResolvePaths(t *testing.T) {
	c, _ := New("k", Production)
	cases := map[string]string{
		"/customers":            "https://api.asaas.com/v3/customers",
		"customers":             "https://api.asaas.com/v3/customers",
		"/v3/customers/cus_1":   "https://api.asaas.com/v3/customers/cus_1",
		"/payments?status=PAID": "https://api.asaas.com/v3/payments?status=PAID",
	}
	for in, want := range cases {
		u, err := c.resolve(in, nil)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if u.String() != want {
			t.Errorf("resolve(%q) = %s, want %s", in, u, want)
		}
	}
	for _, bad := range []string{"https://evil.example/v3/customers", "/customers/../../x"} {
		if _, err := c.resolve(bad, nil); err == nil {
			t.Errorf("resolve(%q) should fail", bad)
		}
	}
	// Leading slashes are stripped, so a protocol-relative path cannot switch hosts.
	if u, err := c.resolve("//evil.example/x", nil); err != nil || u.Host != "api.asaas.com" {
		t.Errorf("resolve(//evil.example/x) = %v, %v; host must stay api.asaas.com", u, err)
	}
}

func TestAPIErrorParsing(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"errors":[{"code":"invalid_cpfCnpj","description":"CPF inválido"}]}`)
	})
	_, err := c.Get(context.Background(), "/customers", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %T %v", err, err)
	}
	if apiErr.StatusCode != 400 || len(apiErr.Errors) != 1 || apiErr.Errors[0].Code != "invalid_cpfCnpj" {
		t.Errorf("unexpected error: %+v", apiErr)
	}
}

func TestNonJSONErrorBodyIsKept(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, "<html>bad gateway</html>")
	})
	_, err := c.Get(context.Background(), "/customers", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Body != "<html>bad gateway</html>" {
		t.Fatalf("got %v", err)
	}
}

func TestNetworkError(t *testing.T) {
	c, _ := New("k", Sandbox, WithBaseURL("http://127.0.0.1:1/v3"))
	_, err := c.Get(context.Background(), "/customers", nil)
	var netErr *NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("want *NetworkError, got %T %v", err, err)
	}
}

func TestListAllFollowsPagination(t *testing.T) {
	const total = 250
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit != MaxPageSize {
			t.Errorf("limit = %d", limit)
		}
		if r.URL.Query().Get("name") != "x" {
			t.Errorf("filter lost: %s", r.URL.RawQuery)
		}
		data := []map[string]int{}
		for i := offset; i < total && i < offset+limit; i++ {
			data = append(data, map[string]int{"n": i})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list", "hasMore": offset+limit < total, "totalCount": total,
			"limit": limit, "offset": offset, "data": data,
		})
	})
	page, err := c.ListAll(context.Background(), "/customers", map[string][]string{"name": {"x"}, "limit": {"5"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != total || page.HasMore || page.TotalCount != total {
		t.Errorf("got %d items, hasMore=%v, total=%d", len(page.Data), page.HasMore, page.TotalCount)
	}
}

func TestEnvironmentHelpers(t *testing.T) {
	if e, ok := EnvironmentFromKey("$aact_hmlg_abc"); !ok || e != Sandbox {
		t.Error("hmlg prefix should be sandbox")
	}
	if e, ok := EnvironmentFromKey("$aact_prod_abc"); !ok || e != Production {
		t.Error("prod prefix should be production")
	}
	if _, ok := EnvironmentFromKey("whatever"); ok {
		t.Error("unknown prefix should not resolve")
	}
	cases := map[string]string{
		"$aact_hmlg_000MzkwODA2MWY2OGM3MWRlMDU2NWM3MzJl": "$aact_hmlg_****MzJl",
		"$aact_prod_short":       "$aact_prod_****",
		"plainkey1234567890abcd": "plai****abcd",
		"tiny":                   "****",
	}
	for in, want := range cases {
		if got := MaskKey(in); got != want {
			t.Errorf("MaskKey(%q) = %q, want %q", in, got, want)
		}
	}
}
