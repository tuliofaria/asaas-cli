package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tuliofaria/asaas-cli/internal/cmdutil"
	"github.com/tuliofaria/asaas-cli/internal/config"
)

func testEnv(fa *fakeAsaas, t *testing.T) config.Env {
	return config.Env{APIKey: "k", BaseURL: fa.srv.URL + "/v3", ConfigDir: t.TempDir()}
}

var taxArgs = []string{"--retain-iss=false", "--iss", "2", "--pis", "0", "--cofins", "0", "--csll", "0", "--inss", "0", "--ir", "0"}

func TestInvoiceCreate(t *testing.T) {
	fa := newFakeAsaas(t)
	env := testEnv(fa, t)

	args := append([]string{"invoice", "create", "--payment", "pay_1", "--service-description", "Consultoria",
		"--observations", "Mensal", "--value", "300", "--effective-date", "2026-10-31",
		"--municipal-service-id", "60380", "--nbs-code", "1.1502"}, taxArgs...)
	r := run(t, env, "", args...)
	if r.code != 0 {
		t.Fatalf("create: %s", r.stderr)
	}
	req := fa.last(t)
	body := decode(t, req.Body)
	taxes := body["taxes"].(map[string]any)
	if req.Method != http.MethodPost || req.Path != "/v3/invoices" || body["value"] != 300.0 ||
		body["deductions"] != 0.0 || body["effectiveDate"] != "2026-10-31" ||
		taxes["retainIss"] != false || taxes["iss"] != 2.0 || taxes["nbsCode"] != "1.1502" {
		t.Errorf("unexpected request %+v", req)
	}
}

func TestInvoiceCreateTaxesFromData(t *testing.T) {
	fa := newFakeAsaas(t)
	env := testEnv(fa, t)

	data := `{"taxes":{"retainIss":false,"iss":5,"pis":0,"cofins":0,"csll":0,"inss":0,"ir":0,"realEstateInfo":{"cibCode":"X"}}}`
	r := run(t, env, data, "invoice", "create", "--customer", "cus_1", "--service-description", "Aluguel",
		"--observations", "-", "--value", "1000", "--effective-date", "2026-11-05",
		"--municipal-service-code", "25.05", "--iss", "3", "--data", "-")
	if r.code != 0 {
		t.Fatalf("create: %s", r.stderr)
	}
	taxes := decode(t, fa.last(t).Body)["taxes"].(map[string]any)
	if taxes["iss"] != 3.0 || taxes["pis"] != 0.0 || taxes["realEstateInfo"].(map[string]any)["cibCode"] != "X" {
		t.Errorf("deep merge failed: %v", taxes)
	}
}

func TestInvoiceCreateValidation(t *testing.T) {
	fa := newFakeAsaas(t)
	env := testEnv(fa, t)
	base := []string{"invoice", "create", "--service-description", "x", "--observations", "x", "--value", "1", "--effective-date", "2026-10-31"}

	cases := map[string]struct {
		args []string
		want string
	}{
		"no origin":     {append(append(append([]string{}, base...), "--municipal-service-id", "1"), taxArgs...), "--payment"},
		"no service":    {append(append(append([]string{}, base...), "--customer", "cus_1"), taxArgs...), "--municipal-service-id"},
		"missing taxes": {append(append([]string{}, base...), "--customer", "cus_1", "--municipal-service-id", "1", "--iss", "2"), "--retain-iss (taxes.retainIss), --pis"},
		"bad date":      {[]string{"invoice", "create", "--effective-date", "31/10/2026"}, "YYYY-MM-DD"},
		"bad op type":   {[]string{"invoice", "create", "--operation-type-code", "9"}, "between 1 and 5"},
	}
	for name, tc := range cases {
		r := run(t, env, "", tc.args...)
		if r.code != cmdutil.ExitUsage || !strings.Contains(r.stderr, tc.want) {
			t.Errorf("%s: exit %d, stderr %s", name, r.code, r.stderr)
		}
	}
	if len(fa.reqs) != 0 {
		t.Errorf("validation errors must not hit the API, got %d requests", len(fa.reqs))
	}
}

func TestInvoiceUpdateCompletesPartialTaxes(t *testing.T) {
	fa := newFakeAsaas(t)
	env := testEnv(fa, t)

	r := run(t, env, "", "invoice", "update", "inv_1", "--iss", "3.5")
	if r.code != 0 {
		t.Fatalf("update: %s", r.stderr)
	}
	req := fa.last(t)
	taxes := decode(t, req.Body)["taxes"].(map[string]any)
	if req.Method != http.MethodPut || req.Path != "/v3/invoices/inv_1" {
		t.Errorf("unexpected request %+v", req)
	}
	if taxes["iss"] != 3.5 || taxes["retainIss"] != true || taxes["cofins"] != 3.0 || taxes["ir"] != 1.5 {
		t.Errorf("taxes not completed from current invoice: %v", taxes)
	}
	if _, ok := taxes["pisCofinsRetentionType"]; ok {
		t.Error("pisCofinsRetentionType must not be sent")
	}

	// Non-tax updates do not read the invoice first.
	before := len(fa.reqs)
	run(t, env, "", "invoice", "update", "inv_1", "--effective-date", "2026-11-10")
	if len(fa.reqs) != before+1 {
		t.Errorf("expected a single PUT, got %d requests", len(fa.reqs)-before)
	}
}

func TestInvoiceListAuthorizeCancel(t *testing.T) {
	fa := newFakeAsaas(t)
	env := testEnv(fa, t)

	run(t, env, "", "invoice", "list", "--status", "AUTHORIZED", "--effective-date-from", "2026-10-01", "--effective-date-to", "2026-10-31")
	if q := fa.last(t).Query; q != "effectiveDate%5Bge%5D=2026-10-01&effectiveDate%5Ble%5D=2026-10-31&status=AUTHORIZED" {
		t.Errorf("list query = %s", q)
	}
	if r := run(t, env, "", "invoice", "list", "--status", "authorized"); r.code != cmdutil.ExitUsage {
		t.Errorf("invalid enum should be a usage error, got %d", r.code)
	}

	run(t, env, "", "invoice", "authorize", "inv_1")
	if req := fa.last(t); req.Method != http.MethodPost || req.Path != "/v3/invoices/inv_1/authorize" {
		t.Errorf("authorize %+v", req)
	}

	run(t, env, "", "invoice", "cancel", "inv_1", "--only-on-asaas")
	if req := fa.last(t); req.Path != "/v3/invoices/inv_1/cancel" || req.Body != `{"cancelOnlyOnAsaas":true}` {
		t.Errorf("cancel %+v", req)
	}
}

func TestFiscalInfo(t *testing.T) {
	fa := newFakeAsaas(t)
	env := testEnv(fa, t)

	cert := filepath.Join(t.TempDir(), "empresa.pfx")
	if err := os.WriteFile(cert, []byte("PFXDATA"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := run(t, env, "s3cret\n", "fiscal-info", "save", "--email", "f@x.com", "--simples-nacional=true",
		"--rps-number", "10", "--certificate-file", cert, "--certificate-password", "-")
	if r.code != 0 {
		t.Fatalf("save: %s", r.stderr)
	}
	out := decode(t, r.stdout)
	if out["email"] != "f@x.com" || out["simplesNacional"] != "true" || out["rpsNumber"] != "10" ||
		out["certificatePassword"] != "s3cret" || out["certificateFile"] != "empresa.pfx:PFXDATA" {
		t.Errorf("multipart fields = %v", out)
	}
	if req := fa.last(t); req.Path != "/v3/fiscalInfo/" {
		t.Errorf("save path = %s", req.Path)
	}

	if r := run(t, env, "", "fiscal-info", "save", "--email", "f@x.com"); r.code != cmdutil.ExitUsage || !strings.Contains(r.stderr, "simples-nacional") {
		t.Errorf("missing simplesNacional: %d %s", r.code, r.stderr)
	}

	run(t, env, "", "fiscal-info", "services", "--description", "software", "--limit", "5")
	if req := fa.last(t); req.Path != "/v3/fiscalInfo/services" || req.Query != "description=software&limit=5" {
		t.Errorf("services %+v", req)
	}
	run(t, env, "", "fiscal-info", "nbs-codes", "--search", "1.15")
	if req := fa.last(t); req.Path != "/v3/fiscalInfo/nbsCodes" || req.Query != "codeDescription=1.15" {
		t.Errorf("nbs-codes %+v", req)
	}
	run(t, env, "", "fiscal-info", "national-portal", "--enabled=false")
	if req := fa.last(t); req.Path != "/v3/fiscalInfo/nationalPortal" || req.Body != `{"enabled":false}` {
		t.Errorf("national-portal %+v", req)
	}
	if r := run(t, env, "", "fiscal-info", "national-portal"); r.code != cmdutil.ExitUsage {
		t.Errorf("national-portal without --enabled should be a usage error")
	}
}
