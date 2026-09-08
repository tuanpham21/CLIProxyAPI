package helps

import (
	"fmt"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/util"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var (
	antigravityDeclarationSchemaKeys = []string{
		"parameters", "parametersJsonSchema", "parameters_json_schema",
		"response", "responseJsonSchema", "response_json_schema",
	}
	antigravityGenerationConfigContainers = []string{
		"request.generationConfig", "request.generation_config",
	}
	antigravityGenerationSchemaKeys = []string{
		"responseSchema", "responseJsonSchema", "response_schema", "response_json_schema",
	}
)

// SanitizeAntigravityRequestSchemas cleans only JSON Schema nodes in an Antigravity request.
// Applying the schema cleaner to the complete request mutates ordinary keys in replayed tool
// arguments, corrupting the examples that Gemini uses to produce its next tool call.
func SanitizeAntigravityRequestSchemas(payload string, useAntigravitySchema bool) string {
	for _, base := range antigravityFunctionDeclarationPaths(payload) {
		oldPath := base + ".parametersJsonSchema"
		if !gjson.Get(payload, oldPath).Exists() {
			continue
		}
		renamed, errRename := util.RenameKey(payload, oldPath, base+".parameters")
		if errRename != nil {
			log.Debugf("antigravity: failed to rename %s: %v", oldPath, errRename)
			continue
		}
		payload = renamed
	}

	clean := util.CleanJSONSchemaForGemini
	if useAntigravitySchema {
		clean = util.CleanJSONSchemaForAntigravity
	}
	for _, path := range antigravitySchemaPaths(payload) {
		schema := gjson.Get(payload, path)
		if !schema.Exists() {
			continue
		}
		updated, errSet := sjson.SetRawBytes([]byte(payload), path, []byte(cleanNestedAntigravitySchema(clean, schema.Raw)))
		if errSet != nil {
			log.Debugf("antigravity: failed to write cleaned schema at %s: %v", path, errSet)
			continue
		}
		payload = string(updated)
	}
	return payload
}

// AntigravityRequestNeedsSchemaSanitization reports whether a request carries a schema node.
func AntigravityRequestNeedsSchemaSanitization(payload []byte) bool {
	if gjson.GetBytes(payload, "request.tools.0").Exists() {
		return true
	}
	for _, container := range antigravityGenerationConfigContainers {
		for _, key := range antigravityGenerationSchemaKeys {
			if gjson.GetBytes(payload, container+"."+key).Exists() {
				return true
			}
		}
	}
	return false
}

func cleanNestedAntigravitySchema(clean func(string) string, schema string) string {
	const wrapper = "schema"
	wrapped, errWrap := sjson.SetRaw("{}", wrapper, schema)
	if errWrap != nil {
		return clean(schema)
	}
	if unwrapped := gjson.Get(clean(wrapped), wrapper); unwrapped.Exists() {
		return unwrapped.Raw
	}
	return clean(schema)
}

func antigravityFunctionDeclarationPaths(payload string) []string {
	tools := gjson.Get(payload, "request.tools")
	if !tools.IsArray() {
		return nil
	}
	paths := make([]string, 0, len(tools.Array()))
	for i, tool := range tools.Array() {
		for _, key := range []string{"functionDeclarations", "function_declarations"} {
			declarations := tool.Get(key)
			if !declarations.IsArray() {
				continue
			}
			for j := range declarations.Array() {
				paths = append(paths, fmt.Sprintf("request.tools.%d.%s.%d", i, key, j))
			}
		}
	}
	return paths
}

func antigravitySchemaPaths(payload string) []string {
	paths := make([]string, 0, 12)
	for _, base := range antigravityFunctionDeclarationPaths(payload) {
		for _, key := range antigravityDeclarationSchemaKeys {
			if gjson.Get(payload, base+"."+key).IsObject() {
				paths = append(paths, base+"."+key)
			}
		}
	}
	for _, container := range antigravityGenerationConfigContainers {
		for _, key := range antigravityGenerationSchemaKeys {
			path := container + "." + key
			if gjson.Get(payload, path).IsObject() {
				paths = append(paths, path)
			}
		}
	}
	return paths
}
