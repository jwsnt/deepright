package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestIntegrationClientRuntimeConfigReturnsOnlyClientFields(t *testing.T) {
	raw := map[string]interface{}{
		"agent-dir":          "/tmp/agent",
		"knowledge":          map[string]interface{}{"interval": 60},
		"skills_git_install": "install $git_path",
		"miniapp": map[string]interface{}{
			"build":    "请使用 @__internal_cli 为 $name 的 $function 构建迷你应用",
			"function": "全部功能",
		},
		"provider": map[string]interface{}{"openai": "private"},
		"secret":   "must not be exposed",
	}

	got := integrationClientRuntimeConfig(raw)
	if got["agent-dir"] != "/tmp/agent" {
		t.Fatalf("agent-dir = %#v", got["agent-dir"])
	}
	if got["skills_git_install"] != "install $git_path" {
		t.Fatalf("skills_git_install = %#v", got["skills_git_install"])
	}
	if got, want := got["miniapp"], raw["miniapp"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("miniapp = %#v, want %#v", got, want)
	}
	if _, ok := got["knowledge"]; !ok {
		t.Fatal("knowledge is missing")
	}
	if _, ok := got["provider"]; ok {
		t.Fatal("provider must not be exposed")
	}
	if _, ok := got["secret"]; ok {
		t.Fatal("unknown configuration must not be exposed")
	}
}

func TestHandleRuntimeConfigReadsIntegrationConfigForMacAndWSLLayouts(t *testing.T) {
	cases := []struct {
		name       string
		executable func(root string) string
	}{
		{
			name: "macOS app resources",
			executable: func(root string) string {
				return filepath.Join(root, "DeepRight.app", "Contents", "MacOS", "integration")
			},
		},
		{
			name: "WSL executable directory",
			executable: func(root string) string {
				return filepath.Join(root, "deepright", "integration")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if tc.name == "macOS app resources" {
				useIntegrationRuntimeHome(t, t.TempDir())
			}
			executable := tc.executable(root)
			configDir := filepath.Join(filepath.Dir(executable), "config")
			if filepath.Base(filepath.Dir(executable)) == "MacOS" {
				configDir = filepath.Join(filepath.Dir(filepath.Dir(executable)), "Resources", "config")
			}
			if err := os.MkdirAll(configDir, 0o755); err != nil {
				t.Fatalf("mkdir config: %v", err)
			}
			if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"skills_git_install":"install $git_path","miniapp":{"build":"请使用 @__internal_cli 为 $name 的 $function 构建迷你应用","function":"全部功能"}}`), 0o644); err != nil {
				t.Fatalf("write config: %v", err)
			}

			originalExecutable := integrationExecutableFn
			integrationExecutableFn = func() (string, error) { return executable, nil }
			t.Cleanup(func() { integrationExecutableFn = originalExecutable })

			recorder := httptest.NewRecorder()
			handleRuntimeConfig().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/runtime_config", nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			var payload struct {
				Status int                    `json:"status"`
				Config map[string]interface{} `json:"config"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if payload.Status != 0 {
				t.Fatalf("status payload = %d", payload.Status)
			}
			if payload.Config["skills_git_install"] != "install $git_path" {
				t.Fatalf("skills_git_install = %#v", payload.Config["skills_git_install"])
			}
			miniapp, ok := payload.Config["miniapp"].(map[string]interface{})
			if !ok || miniapp["build"] != "请使用 @__internal_cli 为 $name 的 $function 构建迷你应用" || miniapp["function"] != "全部功能" {
				t.Fatalf("miniapp = %#v", payload.Config["miniapp"])
			}
		})
	}
}
