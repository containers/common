package report

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"text/tabwriter"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestFormatter_IsTableFalse(t *testing.T) {
	fmtr, err := New(os.Stdout, t.Name()).Parse(OriginPodman, "{{.ID}}")
	assert.NoError(t, err)
	assert.False(t, fmtr.RenderTable)
}

func TestFormatter_IsTableTrue(t *testing.T) {
	fmtr, err := New(os.Stdout, t.Name()).Parse(OriginPodman, "table {{.ID}}")
	assert.NoError(t, err)
	assert.True(t, fmtr.RenderTable)
}

func TestFormatter_HasTable(t *testing.T) {
	assert.True(t, HasTable("table foobar"))
	assert.False(t, HasTable("foobar"))
}

type testFormatterStruct struct {
	FieldA bool // camel case test
	Fieldb bool // no camel case
	fieldC bool // nolint // private field
	fieldd bool // nolint // private field
}

func TestFormatter_HeadersNoOverrides(t *testing.T) {
	expected := []map[string]string{{
		"FieldA": "FIELD A",
		"Fieldb": "FIELDB",
		"fieldC": "FIELD C",
		"fieldd": "FIELDD",
	}}
	assert.Equal(t, expected, Headers(testFormatterStruct{}, nil))
}

func TestFormatter_HeadersOverride(t *testing.T) {
	expected := []map[string]string{{
		"FieldA": "FIELD A",
		"Fieldb": "FIELD B",
		"fieldC": "FIELD C",
		"fieldd": "FIELD D",
	}}
	assert.Equal(t, expected, Headers(testFormatterStruct{}, map[string]string{
		"Fieldb": "field b",
		"fieldd": "field d",
	}))
}

func TestFormatter_ParseTable(t *testing.T) {
	testCase := []struct {
		Type     io.Writer
		Origin   Origin
		Format   string
		Expected string
	}{
		{&tabwriter.Writer{}, OriginUser, "table {{ .ID}}", "Identity\nc061a0839e\nf10fc2e11057\n1eb6fab5aa8f4b5cbfd3e66aa35e9b2a\n"},
		{&tabwriter.Writer{}, OriginUser, "table {{ .ID}}\n", "Identity\nc061a0839e\nf10fc2e11057\n1eb6fab5aa8f4b5cbfd3e66aa35e9b2a\n"},
		{&tabwriter.Writer{}, OriginUser, `table {{ .ID}}\n`, "Identity\nc061a0839e\nf10fc2e11057\n1eb6fab5aa8f4b5cbfd3e66aa35e9b2a\n"},
		{&tabwriter.Writer{}, OriginUser, "table {{.ID}}", "Identity\nc061a0839e\nf10fc2e11057\n1eb6fab5aa8f4b5cbfd3e66aa35e9b2a\n"},
		{
			&tabwriter.Writer{}, OriginUser, "table {{.ID}}\t{{.Value}}",
			"Identity                          Value\nc061a0839e                        one\nf10fc2e11057                      two\n1eb6fab5aa8f4b5cbfd3e66aa35e9b2a  three\n",
		},
		{&bytes.Buffer{}, OriginUser, "{{range .}}{{.ID}}\tID{{end}}", "c061a0839e\tIDf10fc2e11057\tID1eb6fab5aa8f4b5cbfd3e66aa35e9b2a\tID\n"},
		{
			&tabwriter.Writer{}, OriginPodman, "Value\tIdent\n{{range .}}{{.ID}}\tID{{end}}",
			"Value       Ident\nIdentity    ID\nValue       Ident\nc061a0839e  IDf10fc2e11057  ID1eb6fab5aa8f4b5cbfd3e66aa35e9b2a  ID\n",
		},
		{&bytes.Buffer{}, OriginUser, "{{range .}}{{.ID}}\tID\n{{end}}", "c061a0839e\tID\nf10fc2e11057\tID\n1eb6fab5aa8f4b5cbfd3e66aa35e9b2a\tID\n\n"},
		{&bytes.Buffer{}, OriginUser, `{{range .}}{{.ID}}{{end -}}`, "c061a0839ef10fc2e110571eb6fab5aa8f4b5cbfd3e66aa35e9b2a"},
		// regression test for https://bugzilla.redhat.com/show_bug.cgi?id=2059658 and https://github.com/containers/podman/issues/13446
		{&bytes.Buffer{}, OriginUser, `{{range .}}{{printf "\n"}}{{end -}}`, "\n\n\n"},
	}

	for loop, tc := range testCase {
		tc := tc
		name := fmt.Sprintf("Loop#%d", loop)
		t.Run(name, func(t *testing.T) {
			buf := new(bytes.Buffer)

			rpt, err := New(buf, name).Parse(tc.Origin, tc.Format)
			assert.NoError(t, err)
			assert.Equal(t, tc.Origin, rpt.Origin)
			assert.IsType(t, tc.Type, rpt.Writer())

			if rpt.RenderHeaders {
				err = rpt.Execute([]map[string]string{{
					"ID":    "Identity",
					"Value": "Value",
				}})
				assert.NoError(t, err)
			}
			err = rpt.Execute([...]map[string]string{
				{"ID": "c061a0839e", "Value": "one"},
				{"ID": "f10fc2e11057", "Value": "two"},
				{"ID": "1eb6fab5aa8f4b5cbfd3e66aa35e9b2a", "Value": "three"},
			})
			assert.NoError(t, err, fmt.Sprintf("original %+q, cooked %+q", tc.Format, rpt.text))
			rpt.Flush()
			assert.Equal(t, tc.Expected, buf.String(), fmt.Sprintf("original %+q, cooked %+q", tc.Format, rpt.text))
		})
	}
}

func TestFormatter_Init(t *testing.T) {
	data := [...]map[string]string{
		{"ID": "c061a0839e", "Value": "one"},
		{"ID": "f10fc2e11057", "Value": "two"},
		{"ID": "1eb6fab5aa8f4b5cbfd3e66aa35e9b2a", "Value": "three"},
	}

	buf := new(bytes.Buffer)
	fmtr, err := New(buf, t.Name()).Parse(OriginPodman, "{{range .}}{{.ID}}\t{{.Value}}\n{{end -}}")
	assert.NoError(t, err)

	err = fmtr.Execute([]map[string]string{{
		"ID":    "Identity",
		"Value": "Value",
	}})
	assert.NoError(t, err)
	err = fmtr.Execute(data)
	assert.NoError(t, err)
	fmtr.Flush()
	assert.Equal(t,
		"Identity                          Value\nc061a0839e                        one\nf10fc2e11057                      two\n1eb6fab5aa8f4b5cbfd3e66aa35e9b2a  three\n", buf.String())

	buf = new(bytes.Buffer)
	fmtr = fmtr.Init(buf, 8, 1, 1, ' ', tabwriter.Debug)

	err = fmtr.Execute([]map[string]string{{
		"ID":    "Identity",
		"Value": "Value",
	}})
	assert.NoError(t, err)
	err = fmtr.Execute(data)
	assert.NoError(t, err)
	fmtr.Flush()
	assert.Equal(t, "Identity                         |Value\nc061a0839e                       |one\nf10fc2e11057                     |two\n1eb6fab5aa8f4b5cbfd3e66aa35e9b2a |three\n", buf.String())
}

func TestFormatter_FuncsTrim(t *testing.T) {
	buf := new(bytes.Buffer)
	fmtr := New(buf, t.Name())

	fmtr, err := fmtr.Funcs(template.FuncMap{"trim": strings.TrimSpace}).Parse(OriginPodman, "{{.ID |trim}}")
	assert.NoError(t, err)

	err = fmtr.Execute(map[string]string{
		"ID": "ident  ",
	})
	assert.NoError(t, err)
	assert.Equal(t, "ident\n", buf.String())
}

func TestFormatter_FuncsJoin(t *testing.T) {
	buf := new(bytes.Buffer)
	// Add 'trim' function to ensure default 'join' function is still available
	fmtr, e := New(buf, t.Name()).Funcs(template.FuncMap{"trim": strings.TrimSpace}).Parse(OriginPodman, `{{join .ID "-"}}`)
	assert.NoError(t, e)

	err := fmtr.Execute(map[string][]string{
		"ID": {"ident1", "ident2", "ident3"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "ident1-ident2-ident3\n", buf.String())
}

func TestFormatter_FuncsReplace(t *testing.T) {
	buf := new(bytes.Buffer)
	rpt := New(buf, t.Name())

	// yes, we're overriding ToUpper with ToLower :-)
	tmpl, e := rpt.Funcs(template.FuncMap{"upper": strings.ToLower}).Parse(OriginPodman, `{{.ID | lower}}`)
	assert.NoError(t, e)

	err := tmpl.Execute(map[string]string{
		"ID": "IDENT",
	})
	assert.NoError(t, err)
	assert.Equal(t, "ident\n", buf.String())
}

func TestFormatter_FuncsJSON(t *testing.T) {
	buf := new(bytes.Buffer)
	rpt := New(buf, t.Name())

	rpt, e := rpt.Parse(OriginUser, `{{json .ID}}`)
	assert.NoError(t, e)

	err := rpt.Execute([]struct {
		ID []string
	}{{
		ID: []string{"ident1", "ident2", "ident3"},
	}})
	assert.NoError(t, err)
	assert.Equal(t, `["ident1","ident2","ident3"]`+"\n", buf.String(), fmt.Sprintf("cooked %+q", rpt.text))
}

// Verify compatible output
func TestFormatter_Compatible(t *testing.T) {
	buf := new(bytes.Buffer)

	rpt, err := New(buf, t.Name()).Parse(OriginUser, "ID\t{{.ID}}")
	assert.NoError(t, err)

	err = rpt.Execute([...]map[string]string{
		{"ID": "c061a0839e"},
	})
	assert.NoError(t, err)
	rpt.Flush()

	assert.Equal(t, "ID\tc061a0839e\n", buf.String(), fmt.Sprintf("cooked %+q", rpt.text))
}
