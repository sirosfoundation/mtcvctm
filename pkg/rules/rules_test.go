package rules

import (
	"encoding/json"
	"testing"
)

func TestRenameLangToLocale(t *testing.T) {
	data := map[string]interface{}{
		"vct":  "https://example.com/test",
		"name": "Test Credential",
		"display": []interface{}{
			map[string]interface{}{
				"lang": "en-US",
				"name": "Test",
			},
		},
	}

	changed, err := renameLangToLocale.Apply(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected changes")
	}

	display := data["display"].([]interface{})
	dm := display[0].(map[string]interface{})
	if _, exists := dm["lang"]; exists {
		t.Error("lang should be removed")
	}
	if dm["locale"] != "en-US" {
		t.Errorf("locale = %v, want en-US", dm["locale"])
	}
}

func TestRenameLangToLocaleInClaims(t *testing.T) {
	data := map[string]interface{}{
		"claims": []interface{}{
			map[string]interface{}{
				"path": []interface{}{"name"},
				"display": []interface{}{
					map[string]interface{}{
						"lang":  "de-DE",
						"label": "Name",
					},
				},
			},
		},
	}

	changed, err := renameLangToLocaleInClaims.Apply(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected changes")
	}

	claims := data["claims"].([]interface{})
	claim := claims[0].(map[string]interface{})
	display := claim["display"].([]interface{})
	dm := display[0].(map[string]interface{})
	if dm["locale"] != "de-DE" {
		t.Errorf("locale = %v, want de-DE", dm["locale"])
	}
}

func TestSetDisplayLocaleDefault(t *testing.T) {
	data := map[string]interface{}{
		"display": []interface{}{
			map[string]interface{}{
				"name": "Test",
			},
		},
	}

	changed, err := setDisplayLocaleDefault.Apply(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected changes")
	}

	display := data["display"].([]interface{})
	dm := display[0].(map[string]interface{})
	if dm["locale"] != "en-US" {
		t.Errorf("locale = %v, want en-US", dm["locale"])
	}
}

func TestSetDisplayNameFromRoot(t *testing.T) {
	data := map[string]interface{}{
		"name": "Root Name",
		"display": []interface{}{
			map[string]interface{}{
				"locale": "en-US",
			},
		},
	}

	changed, err := setDisplayNameFromRoot.Apply(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected changes")
	}

	display := data["display"].([]interface{})
	dm := display[0].(map[string]interface{})
	if dm["name"] != "Root Name" {
		t.Errorf("name = %v, want Root Name", dm["name"])
	}
}

func TestRemoveEmptySVGTemplateProperties(t *testing.T) {
	tests := []struct {
		name       string
		properties interface{}
		wantChange bool
	}{
		{"nil", nil, true},
		{"empty map", map[string]interface{}{}, true},
		{"all empty values", map[string]interface{}{"orientation": "", "color_scheme": nil}, true},
		{"has value", map[string]interface{}{"orientation": "landscape"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := map[string]interface{}{
				"display": []interface{}{
					map[string]interface{}{
						"rendering": map[string]interface{}{
							"svg_templates": []interface{}{
								map[string]interface{}{
									"uri":        "https://example.com/template.svg",
									"properties": tt.properties,
								},
							},
						},
					},
				},
			}

			changed, err := removeEmptySVGTemplateProperties.Apply(data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if changed != tt.wantChange {
				t.Errorf("changed = %v, want %v", changed, tt.wantChange)
			}

			if tt.wantChange {
				display := data["display"].([]interface{})
				dm := display[0].(map[string]interface{})
				rendering := dm["rendering"].(map[string]interface{})
				templates := rendering["svg_templates"].([]interface{})
				tm := templates[0].(map[string]interface{})
				if _, exists := tm["properties"]; exists {
					t.Error("properties should be removed")
				}
			}
		})
	}
}

func TestEnsureDisplayArray(t *testing.T) {
	data := map[string]interface{}{
		"display": map[string]interface{}{
			"locale": "en-US",
			"name":   "Test",
		},
	}

	changed, err := ensureDisplayArray.Apply(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected changes")
	}

	display, ok := data["display"].([]interface{})
	if !ok {
		t.Fatal("display should be an array")
	}
	if len(display) != 1 {
		t.Errorf("len(display) = %d, want 1", len(display))
	}
}

func TestEngine(t *testing.T) {
	// Test with legacy VCTM data
	legacyData := `{
		"vct": "https://example.com/test",
		"name": "Test Credential",
		"display": {
			"lang": "en-US"
		},
		"claims": [
			{
				"path": ["name"],
				"display": [{"lang": "de-DE", "label": "Name"}]
			}
		]
	}`

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(legacyData), &data); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	engine := NewEngine()
	result, err := engine.Apply(data)
	if err != nil {
		t.Fatalf("engine.Apply failed: %v", err)
	}

	if !result.HasChanges() {
		t.Error("expected changes")
	}

	// Verify transformations
	display := data["display"].([]interface{})
	dm := display[0].(map[string]interface{})

	if dm["locale"] != "en-US" {
		t.Errorf("display locale = %v, want en-US", dm["locale"])
	}
	if dm["name"] != "Test Credential" {
		t.Errorf("display name = %v, want Test Credential", dm["name"])
	}

	claims := data["claims"].([]interface{})
	claim := claims[0].(map[string]interface{})
	claimDisplay := claim["display"].([]interface{})
	cdm := claimDisplay[0].(map[string]interface{})
	if cdm["locale"] != "de-DE" {
		t.Errorf("claim display locale = %v, want de-DE", cdm["locale"])
	}
}

func TestEngineDisableRule(t *testing.T) {
	data := map[string]interface{}{
		"display": []interface{}{
			map[string]interface{}{
				"lang": "en-US",
			},
		},
	}

	engine := NewEngine()
	engine.Disable("rename-lang-to-locale")

	result, err := engine.Apply(data)
	if err != nil {
		t.Fatalf("engine.Apply failed: %v", err)
	}

	// Check that rename-lang-to-locale was skipped
	found := false
	for _, name := range result.Skipped {
		if name == "rename-lang-to-locale" {
			found = true
			break
		}
	}
	if !found {
		t.Error("rename-lang-to-locale should be in skipped list")
	}

	// lang should still exist
	display := data["display"].([]interface{})
	dm := display[0].(map[string]interface{})
	if _, exists := dm["lang"]; !exists {
		t.Error("lang should not be renamed when rule is disabled")
	}
}
