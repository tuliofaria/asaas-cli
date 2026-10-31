package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/pflag"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
	"github.com/tuliofaria/asaas-cli/internal/config"
)

type request struct {
	Method, Path, Query, Token, Body string
}

// fakeAsaas records requests and replies with a canned customer page.
type fakeAsaas struct {
	mu   sync.Mutex
	reqs []request
	srv  *httptest.Server
}

func newFakeAsaas(t *testing.T) *fakeAsaas {
	fa := &fakeAsaas{}
	fa.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		fa.mu.Lock()
		fa.reqs = append(fa.reqs, request{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("access_token"), string(b)})
		fa.mu.Unlock()
		switch {
		case r.Header.Get("access_token") == "bad":
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"errors":[{"code":"invalid_access_token","description":"invalid"}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v3/customers":
			fmt.Fprint(w, `{"object":"list","hasMore":false,"totalCount":1,"limit":10,"offset":0,"data":[{"id":"cus_1","name":"Ana","invoiceUrl":"https://x.test/?a=1&b=2"}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v3/invoices/inv_1":
			fmt.Fprint(w, `{"id":"inv_1","status":"SCHEDULED","taxes":{"retainIss":true,"iss":2,"pis":0.65,"cofins":3,"csll":1,"inss":0,"ir":1.5,"pisCofinsRetentionType":"WITHHELD","nbsCode":null}}`)
		case strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data"):
			r.Body = io.NopCloser(bytes.NewReader(b))
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			out := map[string]any{}
			for k, v := range r.MultipartForm.Value {
				out[k] = v[0]
			}
			for k, fhs := range r.MultipartForm.File {
				fh, _ := fhs[0].Open()
				content, _ := io.ReadAll(fh)
				out[k] = fhs[0].Filename + ":" + string(content)
			}
			_ = json.NewEncoder(w).Encode(out)
		case r.Method == http.MethodDelete:
			fmt.Fprint(w, `{"deleted":true,"id":"cus_1"}`)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/empty"):
			w.WriteHeader(http.StatusNoContent)
		default:
			w.Write(b)
		}
	}))
	t.Cleanup(fa.srv.Close)
	return fa
}

func (fa *fakeAsaas) last(t *testing.T) request {
	t.Helper()
	fa.mu.Lock()
	defer fa.mu.Unlock()
	if len(fa.reqs) == 0 {
		t.Fatal("no request received")
	}
	return fa.reqs[len(fa.reqs)-1]
}

type result struct {
	stdout, stderr string
	code           int
}

func run(t *testing.T, env config.Env, stdin string, args ...string) result {
	t.Helper()
	var out, errOut bytes.Buffer
	f := &cmdutil.Factory{
		In:      strings.NewReader(stdin),
		Out:     &out,
		ErrOut:  &errOut,
		Version: "test",
		LoadEnv: func() config.Env { return env },
	}
	code := Run(context.Background(), f, args)
	return result{out.String(), errOut.String(), code}
}

func decode(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("output is not a JSON object: %v\n%s", err, s)
	}
	return m
}

func errorType(t *testing.T, r result) string {
	t.Helper()
	if r.stdout != "" {
		t.Errorf("stdout must be empty on error, got %s", r.stdout)
	}
	return decode(t, r.stderr)["error"].(map[string]any)["type"].(string)
}

func TestCustomerListUsesEnvKey(t *testing.T) {
	fa := newFakeAsaas(t)
	env := config.Env{APIKey: "env-key", BaseURL: fa.srv.URL + "/v3", ConfigDir: t.TempDir()}

	r := run(t, env, "", "customer", "list", "--name", "Ana", "--limit", "5", "--sandbox")
	if r.code != 0 {
		t.Fatalf("exit %d: %s", r.code, r.stderr)
	}
	req := fa.last(t)
	if req.Token != "env-key" || req.Path != "/v3/customers" || req.Query != "limit=5&name=Ana" {
		t.Errorf("unexpected request %+v", req)
	}
	if !strings.Contains(r.stdout, `"invoiceUrl": "https://x.test/?a=1&b=2"`) {
		t.Errorf("output not indented/unescaped JSON:\n%s", r.stdout)
	}
}

func TestAuthFlowThenCustomerListUsesConfigFile(t *testing.T) {
	fa := newFakeAsaas(t)
	env := config.Env{BaseURL: fa.srv.URL + "/v3", ConfigDir: t.TempDir()}

	if r := run(t, env, "", "customer", "list"); r.code != cmdutil.ExitAuth || errorType(t, r) != "auth_error" {
		t.Fatalf("expected auth_error without credentials, got %d %s", r.code, r.stderr)
	}

	if r := run(t, env, "", "auth", "add", "--name", "sandbox-dev", "--key", "file-key", "--sandbox"); r.code != 0 {
		t.Fatalf("auth add: %s", r.stderr)
	}
	if r := run(t, env, "prod-key\n", "auth", "add", "--name", "prod", "--key", "-", "--production", "--activate=false"); r.code != 0 {
		t.Fatalf("auth add stdin: %s", r.stderr)
	}

	r := run(t, env, "", "auth", "status")
	st := decode(t, r.stdout)
	if st["account"] != "sandbox-dev" || st["environment"] != "sandbox" || st["source"] != config.SourceFile {
		t.Errorf("status = %v", st)
	}
	if strings.Contains(r.stdout, "file-key") {
		t.Error("status leaked the full API key")
	}

	if r := run(t, env, "", "customer", "list"); r.code != 0 {
		t.Fatalf("customer list: %s", r.stderr)
	}
	if got := fa.last(t).Token; got != "file-key" {
		t.Errorf("token = %q, want file-key", got)
	}

	if r := run(t, env, "", "auth", "switch", "--name", "prod"); r.code != 0 {
		t.Fatalf("switch: %s", r.stderr)
	}
	run(t, env, "", "customer", "list")
	if got := fa.last(t).Token; got != "prod-key" {
		t.Errorf("after switch token = %q, want prod-key", got)
	}

	// ASAAS_API_KEY takes priority over the active account.
	env.APIKey = "env-key"
	run(t, env, "", "customer", "list")
	if got := fa.last(t).Token; got != "env-key" {
		t.Errorf("env token = %q, want env-key", got)
	}
}

func TestAPIErrorGoesToStderrAsJSON(t *testing.T) {
	fa := newFakeAsaas(t)
	env := config.Env{APIKey: "bad", BaseURL: fa.srv.URL + "/v3", ConfigDir: t.TempDir()}
	r := run(t, env, "", "customer", "list")
	if r.code != cmdutil.ExitError || errorType(t, r) != "api_error" {
		t.Fatalf("got %d %s", r.code, r.stderr)
	}
	e := decode(t, r.stderr)["error"].(map[string]any)
	if e["status_code"].(float64) != 401 {
		t.Errorf("status_code = %v", e["status_code"])
	}
}

func TestCustomerCreateUpdateDelete(t *testing.T) {
	fa := newFakeAsaas(t)
	env := config.Env{APIKey: "k", BaseURL: fa.srv.URL + "/v3", ConfigDir: t.TempDir()}

	r := run(t, env, "", "customer", "create", "--name", "Ana", "--cpf-cnpj", "01234567890",
		"--notification-disabled", "--data", `{"company":"ACME","name":"ignored"}`)
	if r.code != 0 {
		t.Fatalf("create: %s", r.stderr)
	}
	body := decode(t, fa.last(t).Body)
	if body["name"] != "Ana" || body["cpfCnpj"] != "01234567890" || body["notificationDisabled"] != true || body["company"] != "ACME" {
		t.Errorf("create body = %v", body)
	}
	if _, sent := body["email"]; sent {
		t.Error("unset flags must not be sent")
	}

	if r := run(t, env, "", "customer", "create", "--name", "Ana"); r.code != cmdutil.ExitUsage || !strings.Contains(r.stderr, "cpf-cnpj") {
		t.Errorf("missing required flag: %d %s", r.code, r.stderr)
	}

	r = run(t, env, "", "customer", "update", "cus_1", "--email", "a@b.com")
	if req := fa.last(t); r.code != 0 || req.Method != http.MethodPut || req.Path != "/v3/customers/cus_1" || req.Body != `{"email":"a@b.com"}` {
		t.Errorf("update: %d %+v %s", r.code, req, r.stderr)
	}
	if r := run(t, env, "", "customer", "update", "cus_1"); r.code != cmdutil.ExitUsage {
		t.Errorf("empty update should be a usage error, got %d", r.code)
	}

	r = run(t, env, "", "customer", "delete", "cus_1")
	if r.code != 0 || decode(t, r.stdout)["deleted"] != true {
		t.Errorf("delete: %s %s", r.stdout, r.stderr)
	}
	if r := run(t, env, "", "customer", "get"); r.code != cmdutil.ExitUsage {
		t.Errorf("get without id should be usage error, got %d", r.code)
	}
}

func TestAPICommand(t *testing.T) {
	fa := newFakeAsaas(t)
	env := config.Env{APIKey: "k", BaseURL: fa.srv.URL + "/v3", ConfigDir: t.TempDir()}

	r := run(t, env, "", "api", "/payments", "-X", "post", "-f", "cpfCnpj=0123", "-F", "value=129.90", "-F", "flag=true")
	if r.code != 0 {
		t.Fatalf("api: %s", r.stderr)
	}
	req := fa.last(t)
	body := decode(t, req.Body)
	if req.Method != "POST" || req.Path != "/v3/payments" || body["cpfCnpj"] != "0123" || body["value"] != 129.90 || body["flag"] != true {
		t.Errorf("api request %+v", req)
	}

	r = run(t, env, "", "api", "/x/empty", "-X", "POST")
	if decode(t, r.stdout)["status_code"].(float64) != 204 {
		t.Errorf("empty body output: %s", r.stdout)
	}

	if r := run(t, env, "", "api", "https://evil.example/v3/x"); r.code == 0 {
		t.Error("absolute URLs must be rejected")
	}
	if r := run(t, env, "", "api", "/customers", "-f", "a=b"); r.code != cmdutil.ExitUsage {
		t.Error("GET with body must be rejected")
	}
}

func TestUsageErrors(t *testing.T) {
	env := config.Env{ConfigDir: t.TempDir()}
	for _, args := range [][]string{
		{"bogus"},
		{"customer", "bogus"},
		{"customer", "list", "--limit", "0"},
		{"customer", "list", "--nope"},
		{"auth", "status", "--sandbox", "--production"},
		{"auth", "add", "--name", "x", "--key", "plain"},
		{"auth", "add", "--name", "x", "--key", "$aact_prod_abc", "--sandbox"},
	} {
		r := run(t, env, "", args...)
		if r.code != cmdutil.ExitUsage || errorType(t, r) != "usage_error" {
			t.Errorf("%v: exit %d, stderr %s", args, r.code, r.stderr)
		}
	}
}

func TestEveryCommandIsDocumented(t *testing.T) {
	root := NewRootCmd(&cmdutil.Factory{LoadEnv: config.LoadEnv})
	stack := root.Commands()
	for len(stack) > 0 {
		c := stack[0]
		stack = append(stack[1:], c.Commands()...)
		if c.Hidden || c.Name() == "help" {
			continue
		}
		if c.Short == "" || c.Long == "" {
			t.Errorf("%s: missing Short or Long", c.CommandPath())
		}
		c.LocalFlags().VisitAll(func(fl *pflag.Flag) {
			if fl.Name != "help" && !strings.Contains(fl.Usage, "[type: ") {
				t.Errorf("%s --%s: usage does not document its type", c.CommandPath(), fl.Name)
			}
		})
	}
}
