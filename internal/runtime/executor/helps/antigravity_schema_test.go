package helps

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestSanitizeAntigravityRequestSchemasPreservesHistory(t *testing.T) {
	payload := `{"request":{"contents":[{"role":"model","parts":[{"functionCall":{"name":"write_file","args":{"title":"keep me","format":"markdown","default":"fallback","const":"literal"}}}]}],"tools":[{"functionDeclarations":[{"name":"write_file","parametersJsonSchema":{"type":"object","properties":{"title":{"type":"string","minLength":1}}}}]}]}}`

	got := SanitizeAntigravityRequestSchemas(payload, false)
	if before, after := gjson.Get(payload, "request.contents"), gjson.Get(got, "request.contents"); before.Raw != after.Raw {
		t.Fatalf("conversation history was mutated\nwant: %s\ngot:  %s", before.Raw, after.Raw)
	}
	declaration := gjson.Get(got, "request.tools.0.functionDeclarations.0")
	if declaration.Get("parametersJsonSchema").Exists() {
		t.Fatalf("parametersJsonSchema was not renamed: %s", declaration.Raw)
	}
	parameters := declaration.Get("parameters")
	if !parameters.Get("properties.title").Exists() {
		t.Fatalf("property named title was removed: %s", parameters.Raw)
	}
	if parameters.Get("properties.title.minLength").Exists() {
		t.Fatalf("unsupported schema keyword survived: %s", parameters.Raw)
	}
}

func TestAntigravityRequestNeedsSchemaSanitization(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    bool
	}{
		{"tools", `{"request":{"tools":[{"functionDeclarations":[]}]}}`, true},
		{"camel response schema", `{"request":{"generationConfig":{"responseSchema":{}}}}`, true},
		{"snake response schema", `{"request":{"generation_config":{"response_schema":{}}}}`, true},
		{"no schema", `{"request":{"contents":[]}}`, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := AntigravityRequestNeedsSchemaSanitization([]byte(test.payload)); got != test.want {
				t.Fatalf("AntigravityRequestNeedsSchemaSanitization() = %v, want %v", got, test.want)
			}
		})
	}
}
