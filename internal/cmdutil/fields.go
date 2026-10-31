package cmdutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// FieldType is the data type of a declarative flag.
type FieldType int

const (
	String FieldType = iota
	Int
	Float
	Bool
	Date // string in YYYY-MM-DD format
)

const dateLayout = "2006-01-02"

func (t FieldType) label() string {
	switch t {
	case Int:
		return "integer"
	case Float:
		return "decimal number, dot as separator"
	case Bool:
		return "boolean"
	case Date:
		return "date, YYYY-MM-DD"
	}
	return "string"
}

// Field maps one CLI flag to one Asaas API parameter (query or body).
// New resources are added by declaring []Field, not by writing parsing code.
type Field struct {
	// Param is the API name. A dotted name ("taxes.iss") is sent as a
	// nested JSON object: {"taxes": {"iss": ...}}.
	Param string
	Flag  string // flag name, e.g. "cpf-cnpj"
	Type  FieldType
	Usage string
	// Required makes Cobra reject the command when the flag is missing.
	Required bool
	// BodyRequired means the value must be present in the final request
	// body, either from this flag or from --data (see CheckRequired).
	BodyRequired bool
	// Enum lists the accepted values (String fields only).
	Enum []string
}

// AddFields registers fields as flags on cmd, documenting each one's type.
func AddFields(cmd *cobra.Command, fields []Field) {
	fs := cmd.Flags()
	for _, f := range fields {
		usage := f.Usage
		if len(f.Enum) > 0 {
			usage += "; one of: " + strings.Join(f.Enum, ", ")
		}
		usage = fmt.Sprintf("%s [type: %s]", usage, f.Type.label())
		if f.BodyRequired {
			usage += " [required, via this flag or --data]"
		}
		switch f.Type {
		case Int:
			fs.Int(f.Flag, 0, usage)
		case Float:
			fs.Float64(f.Flag, 0, usage)
		case Bool:
			fs.Bool(f.Flag, false, usage)
		default:
			fs.String(f.Flag, "", usage)
		}
		if f.Required {
			MarkRequired(cmd, f.Flag)
		}
	}
}

// MarkRequired marks flags as required for Cobra and appends "[required]"
// to their usage so it shows up in --help.
func MarkRequired(cmd *cobra.Command, names ...string) {
	for _, n := range names {
		if err := cmd.MarkFlagRequired(n); err != nil {
			panic(err) // programming error: flag not defined
		}
		if fl := cmd.Flags().Lookup(n); fl != nil && !strings.HasSuffix(fl.Usage, "[required]") {
			fl.Usage += " [required]"
		}
	}
}

// CollectBody returns the explicitly set fields as a JSON-ready map,
// validating enums and dates. Dotted params become nested objects.
func CollectBody(cmd *cobra.Command, fields []Field) (map[string]any, error) {
	body := map[string]any{}
	fs := cmd.Flags()
	for _, f := range fields {
		if !fs.Changed(f.Flag) {
			continue
		}
		var (
			v   any
			err error
		)
		switch f.Type {
		case Int:
			v, err = fs.GetInt(f.Flag)
		case Float:
			v, err = fs.GetFloat64(f.Flag)
		case Bool:
			v, err = fs.GetBool(f.Flag)
		default:
			v, err = fs.GetString(f.Flag)
		}
		if err != nil {
			return nil, UsageErrorf("--%s: %v", f.Flag, err)
		}
		if err := f.validate(fmt.Sprint(v)); err != nil {
			return nil, err
		}
		setPath(body, f.Param, v)
	}
	return body, nil
}

func (f Field) validate(v string) error {
	if f.Type == Date {
		if _, err := time.Parse(dateLayout, v); err != nil {
			return UsageErrorf("--%s: %q is not a date in YYYY-MM-DD format", f.Flag, v)
		}
	}
	if len(f.Enum) > 0 && !slices.Contains(f.Enum, v) {
		return UsageErrorf("--%s: %q is not one of: %s", f.Flag, v, strings.Join(f.Enum, ", "))
	}
	return nil
}

// setPath stores v under a dotted path, creating intermediate objects.
func setPath(m map[string]any, path string, v any) {
	parts := strings.Split(path, ".")
	for _, p := range parts[:len(parts)-1] {
		next, ok := m[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			m[p] = next
		}
		m = next
	}
	m[parts[len(parts)-1]] = v
}

// HasPath reports whether a dotted path is present in m.
func HasPath(m map[string]any, path string) bool {
	parts := strings.Split(path, ".")
	for _, p := range parts[:len(parts)-1] {
		next, ok := m[p].(map[string]any)
		if !ok {
			return false
		}
		m = next
	}
	_, ok := m[parts[len(parts)-1]]
	return ok
}

// CheckRequired verifies that every BodyRequired field ended up in body
// (from its flag or from --data) and reports all missing ones at once.
func CheckRequired(body map[string]any, fields []Field) error {
	var missing []string
	for _, f := range fields {
		if f.BodyRequired && !HasPath(body, f.Param) {
			missing = append(missing, fmt.Sprintf("--%s (%s)", f.Flag, f.Param))
		}
	}
	if len(missing) > 0 {
		return UsageErrorf("missing required fields: %s", strings.Join(missing, ", "))
	}
	return nil
}

// RequireOneOf fails unless at least one of the dotted params is in body.
func RequireOneOf(body map[string]any, fields []Field, params ...string) error {
	names := make([]string, 0, len(params))
	for _, p := range params {
		if HasPath(body, p) {
			return nil
		}
		for _, f := range fields {
			if f.Param == p {
				names = append(names, "--"+f.Flag)
			}
		}
	}
	return UsageErrorf("one of %s is required (or the equivalent key in --data)", strings.Join(names, ", "))
}

// CollectQuery returns the explicitly set fields as URL query parameters.
func CollectQuery(cmd *cobra.Command, fields []Field) (url.Values, error) {
	q := url.Values{}
	fs := cmd.Flags()
	for _, f := range fields {
		if fs.Changed(f.Flag) {
			v := fs.Lookup(f.Flag).Value.String()
			if err := f.validate(v); err != nil {
				return nil, err
			}
			q.Set(f.Param, v)
		}
	}
	return q, nil
}

// PaginationFields are the standard offset/limit parameters of list endpoints.
var PaginationFields = []Field{
	{Param: "offset", Flag: "offset", Type: Int, Usage: "Zero-based index of the first item to return (pagination); API default 0"},
	{Param: "limit", Flag: "limit", Type: Int, Usage: "Number of items per page, from 1 to 100; API default 10"},
}

// ValidatePagination checks --offset/--limit bounds when they were set.
func ValidatePagination(cmd *cobra.Command) error {
	fs := cmd.Flags()
	if fs.Changed("limit") {
		if l, _ := fs.GetInt("limit"); l < 1 || l > 100 {
			return UsageErrorf("--limit must be between 1 and 100, got %d", l)
		}
	}
	if fs.Changed("offset") {
		if o, _ := fs.GetInt("offset"); o < 0 {
			return UsageErrorf("--offset must be >= 0, got %d", o)
		}
	}
	return nil
}

// ReadJSONArg loads a JSON document given inline, as @path, or as "-" (stdin).
func ReadJSONArg(in io.Reader, flag, value string) (json.RawMessage, error) {
	var raw []byte
	var err error
	switch {
	case value == "-":
		raw, err = io.ReadAll(in)
	case strings.HasPrefix(value, "@"):
		raw, err = os.ReadFile(value[1:])
	default:
		raw = []byte(value)
	}
	if err != nil {
		return nil, UsageErrorf("--%s: %v", flag, err)
	}
	if !json.Valid(raw) {
		return nil, UsageErrorf("--%s: value is not valid JSON", flag)
	}
	return json.RawMessage(raw), nil
}

// MergeJSONObject decodes a JSON object from --data (if any) and deep-merges
// fields on top of it, so typed flags win over raw JSON while nested objects
// (e.g. "taxes") keep the keys set only in --data.
func MergeJSONObject(in io.Reader, flag, data string, fields map[string]any) (map[string]any, error) {
	body := map[string]any{}
	if data != "" {
		raw, err := ReadJSONArg(in, flag, data)
		if err != nil {
			return nil, err
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		if err := dec.Decode(&body); err != nil {
			return nil, UsageErrorf("--%s: must be a JSON object: %v", flag, err)
		}
	}
	DeepMerge(body, fields)
	return body, nil
}

// DeepMerge copies src into dst, merging nested objects key by key.
func DeepMerge(dst, src map[string]any) {
	for k, v := range src {
		if sv, ok := v.(map[string]any); ok {
			if dv, ok := dst[k].(map[string]any); ok {
				DeepMerge(dv, sv)
				continue
			}
		}
		dst[k] = v
	}
}

// ParseKeyValue splits "key=value".
func ParseKeyValue(flag, s string) (string, string, error) {
	k, v, ok := strings.Cut(s, "=")
	if !ok || k == "" {
		return "", "", UsageErrorf("--%s: expected key=value, got %q", flag, s)
	}
	return k, v, nil
}

// TypedValue converts a -F value into a JSON type: true/false/null,
// integers and decimals become JSON literals, anything else stays a string.
func TypedValue(s string) any {
	switch s {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	// Only valid JSON numbers qualify: "0123" (leading zero) or "Inf" stay strings.
	if s != "" && (s[0] == '-' || (s[0] >= '0' && s[0] <= '9')) && json.Valid([]byte(s)) {
		return json.Number(s)
	}
	return s
}
