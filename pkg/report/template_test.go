package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTemplate(t *testing.T) {
	actual := NewTemplate("TestNewTemplate")
	assert.Equal(t, false, actual.isTable)
}

type testStruct struct {
	FieldA bool // camel case test
	Fieldb bool // no camel case
	fieldC bool // nolint // private field
	fieldd bool // nolint // private field
}

func TestHeadersSimple(t *testing.T) {
	expected := []map[string]string{{
		"FieldA": "FIELD A",
		"Fieldb": "FIELDB",
		"fieldC": "FIELD C",
		"fieldd": "FIELDD",
	}}
	assert.Equal(t, expected, Headers(testStruct{}, nil))
}

func TestHeadersOverride(t *testing.T) {
	expected := []map[string]string{{
		"FieldA": "FIELD A",
		"Fieldb": "FIELD B",
		"fieldC": "FIELD C",
		"fieldd": "FIELD D",
	}}
	assert.Equal(t, expected, Headers(testStruct{}, map[string]string{
		"Fieldb": "field b",
		"fieldd": "field d",
	}))
}

func TestNormalizeFormat(t *testing.T) {
	testCase := []struct {
		input    string
		expected string
	}{
		{"{{.ID}}\t{{.ID}}", "{{.ID}}\t{{.ID}}\n"},
		{`{{.ID}}\t{{.ID}}`, "{{.ID}}\t{{.ID}}\n"},
		{`{{.ID}} {{.ID}}`, "{{.ID}} {{.ID}}\n"},
		{`table {{.ID}}\t{{.ID}}`, "{{.ID}}\t{{.ID}}\n"},
		{`table {{.ID}} {{.ID}}`, "{{.ID}}\t{{.ID}}\n"},
	}

	for _, tc := range testCase {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, NormalizeFormat(tc.input))
		})
	}
}

func TestTemplate_Parse(t *testing.T) {
	testCase := []string{
		"table {{.ID}}",
		"table {{ .ID}}",
		"table {{ .ID}}\n",
		"{{range .}}{{.ID}}{{end}}",
		`{{range .}}{{.ID}}{{end}}`,
	}

	var buf bytes.Buffer
	for _, tc := range testCase {
		tc := tc
		t.Run(tc, func(t *testing.T) {
			tmpl, e := NewTemplate("TestTemplate").Parse(tc)
			assert.NoError(t, e)

			err := tmpl.Execute(&buf, [...]map[string]string{{
				"ID": "Ident",
			}})
			assert.NoError(t, err)
			assert.Equal(t, "Ident\n", buf.String())
		})
		buf.Reset()
	}
}

func TestTemplate_IsTable(t *testing.T) {
	tmpl, e := NewTemplate("TestTemplate").Parse("table {{.ID}}")
	assert.NoError(t, e)
	assert.True(t, tmpl.isTable)
}

func TestTemplate_trim(t *testing.T) {
	tmpl := NewTemplate("TestTemplate")
	tmpl, e := tmpl.Funcs(FuncMap{"trim": strings.TrimSpace}).Parse("{{.ID |trim}}")
	assert.NoError(t, e)

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, map[string]string{
		"ID": "ident  ",
	})
	assert.NoError(t, err)
	assert.Equal(t, "ident\n", buf.String())
}

func TestTemplate_DefaultFuncs(t *testing.T) {
	tmpl := NewTemplate("TestTemplate")
	// Throw in trim function to ensure default 'join' is still available
	tmpl, e := tmpl.Funcs(FuncMap{"trim": strings.TrimSpace}).Parse(`{{join .ID "-"}}`)
	assert.NoError(t, e)

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, map[string][]string{
		"ID": {"ident1", "ident2", "ident3"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "ident1-ident2-ident3\n", buf.String())
}

func TestTemplate_ReplaceFuncs(t *testing.T) {
	tmpl := NewTemplate("TestTemplate")
	// yes, we're overriding upper with lower :-)
	tmpl, e := tmpl.Funcs(FuncMap{"upper": strings.ToLower}).Parse(`{{.ID | lower}}`)
	assert.NoError(t, e)

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, map[string]string{
		"ID": "IDENT",
	})
	assert.NoError(t, err)
	assert.Equal(t, "ident\n", buf.String())
}

func TestTemplate_json(t *testing.T) {
	tmpl := NewTemplate("TestTemplate")
	// yes, we're overriding upper with lower :-)
	tmpl, e := tmpl.Parse(`{{json .ID}}`)
	assert.NoError(t, e)

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, map[string][]string{
		"ID": {"ident1", "ident2", "ident3"},
	})
	assert.NoError(t, err)
	assert.Equal(t, `["ident1","ident2","ident3"]`+"\n", buf.String())
}

func TestTemplate_HasTable(t *testing.T) {
	assert.True(t, HasTable("table foobar"))
	assert.False(t, HasTable("foobar"))
}

func TestTemplate_EnforceRange(t *testing.T) {
	testRange := `{{range .}}foobar was here{{end -}}`
	assert.Equal(t, testRange, EnforceRange(testRange))
	assert.Equal(t, testRange, EnforceRange("foobar was here"))
	assert.NotEqual(t, testRange, EnforceRange("foobar"))

	// Do not override a given range
	testRange = `{{range .}}foobar was here{{end}}`
	assert.Equal(t, testRange, EnforceRange(testRange))
}

func TestTemplate_Newlines(t *testing.T) {
	input := []struct {
		Field1 string
		Field2 int
		Field3 string
	}{
		{Field1: "One", Field2: 1, Field3: "First"},
		{Field1: "Two", Field2: 2, Field3: "Second"},
		{Field1: "Three", Field2: 3, Field3: "Third"},
	}

	hdrs := Headers(input[0], map[string]string{"Field1": "Eins", "Field2": "Zwei", "Field3": "Drei"})

	// Ensure no blank lines in table
	expected := "EINS\tZWEI\tDREI\nOne\t1\tFirst\nTwo\t2\tSecond\nThree\t3\tThird\n"

	format := NormalizeFormat("{{.Field1}}\t{{.Field2}}\t{{.Field3}}")
	format = EnforceRange(format)
	tmpl, err := NewTemplate("TestTemplate").Parse(format)
	assert.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, hdrs)
	assert.NoError(t, err)

	err = tmpl.Execute(&buf, input)
	assert.NoError(t, err)

	assert.Equal(t, expected, buf.String())
}
