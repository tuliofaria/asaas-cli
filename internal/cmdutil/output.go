package cmdutil

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/tuliofaria/asaas-cli/pkg/asaasclient"
)

// PrintJSON writes v as indented JSON followed by a newline. HTML escaping
// is disabled so URLs (e.g. invoiceUrl) stay readable. json.RawMessage
// values are re-indented without losing number precision.
func PrintJSON(w io.Writer, v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return &Error{Kind: KindInternal, Message: "encoding output: " + err.Error(), Err: err}
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// Print writes v to stdout as JSON.
func (f *Factory) Print(v any) error { return PrintJSON(f.Out, v) }

// PrintResponse writes an API response body to stdout. Empty bodies are
// rendered as {"success": true, "status_code": N} so stdout is always JSON.
func (f *Factory) PrintResponse(resp *asaasclient.Response) error {
	if len(resp.Body) == 0 {
		return f.Print(map[string]any{"success": true, "status_code": resp.StatusCode})
	}
	return f.Print(resp.Body)
}
