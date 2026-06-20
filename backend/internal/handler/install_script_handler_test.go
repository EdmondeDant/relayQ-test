package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNormalizeOpenAIBaseURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "root url",
			raw:  "https://api.example.com",
			want: "https://api.example.com/v1",
		},
		{
			name: "v1 url",
			raw:  "https://api.example.com/v1",
			want: "https://api.example.com/v1",
		},
		{
			name: "models url",
			raw:  "https://api.example.com/v1/models",
			want: "https://api.example.com/v1",
		},
		{
			name: "trailing slash",
			raw:  "https://api.example.com/",
			want: "https://api.example.com/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeOpenAIBaseURL(tt.raw); got != tt.want {
				t.Fatalf("normalizeOpenAIBaseURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestBuildOpenClawWindowsIncludesPowerShellGuidance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "https://relayq.example.com/install/openclaw-windows", nil)

	h := NewInstallScriptHandler(nil)
	script := h.buildOpenClawWindows(ctx)

	if !strings.Contains(script, "openclaw.cmd gateway") {
		t.Fatalf("expected script to mention openclaw.cmd guidance, got:\n%s", script)
	}
	if !strings.Contains(script, "Set-ExecutionPolicy -Scope CurrentUser RemoteSigned") {
		t.Fatalf("expected script to mention execution policy guidance, got:\n%s", script)
	}
	if !strings.Contains(script, `Trim([char]96).Trim("'").Trim('"')`) {
		t.Fatalf("expected script to sanitize wrapped base url, got:\n%s", script)
	}
	if !strings.Contains(script, "onboard --non-interactive --accept-risk --flow quickstart --mode local") {
		t.Fatalf("expected script to run official onboarding, got:\n%s", script)
	}
	if !strings.Contains(script, "--accept-risk") {
		t.Fatalf("expected script to acknowledge non-interactive onboarding risk, got:\n%s", script)
	}
	if !strings.Contains(script, "--auth-choice custom-api-key") || !strings.Contains(script, "--custom-compatibility openai") {
		t.Fatalf("expected script to configure custom provider onboarding, got:\n%s", script)
	}
	if !strings.Contains(script, "--custom-provider-id relayq") {
		t.Fatalf("expected script to pin custom provider id to relayq, got:\n%s", script)
	}
	if strings.Contains(script, "--custom-provider-id openai") {
		t.Fatalf("expected script to stop pinning custom provider id to openai, got:\n%s", script)
	}
	if !strings.Contains(script, "请先正确选择模型，再生成安装命令。") {
		t.Fatalf("expected script to require explicit model selection, got:\n%s", script)
	}
	if !strings.Contains(script, "--install-daemon") {
		t.Fatalf("expected script to install daemon during onboarding, got:\n%s", script)
	}
	if !strings.Contains(script, "openclaw.cmd dashboard") && !strings.Contains(script, "& $openclawBin dashboard") {
		t.Fatalf("expected script to open dashboard after install, got:\n%s", script)
	}
	if strings.Contains(script, "claude-sonnet-4-6") {
		t.Fatalf("expected script to stop using hardcoded default model fallback, got:\n%s", script)
	}
	if strings.Contains(script, "Send-DebugEvent") || strings.Contains(script, "openclaw-agent-auth") {
		t.Fatalf("expected script to stop shipping debug instrumentation, got:\n%s", script)
	}
	if !strings.Contains(script, `$expectedModel = "relayq/$ModelName"`) {
		t.Fatalf("expected script to validate written model namespace, got:\n%s", script)
	}
	if !strings.Contains(script, `$providerKeys -contains 'relayq'`) {
		t.Fatalf("expected script to validate relayq provider presence, got:\n%s", script)
	}
	if !strings.Contains(script, `$openclawBin models status`) || !strings.Contains(script, `relayq effective=`) {
		t.Fatalf("expected script to validate relayq auth from models status, got:\n%s", script)
	}
}

func TestBuildHermesWindowsUsesNativePowerShellInstaller(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "https://relayq.example.com/install/hermes-windows", nil)

	h := NewInstallScriptHandler(nil)
	script := h.buildHermesWindows(ctx)

	if !strings.Contains(script, `Trim([char]96).Trim("'").Trim('"')`) {
		t.Fatalf("expected script to sanitize wrapped base url, got:\n%s", script)
	}
	if !strings.Contains(script, `install.ps1`) || !strings.Contains(script, `-SkipSetup`) ||
		!strings.Contains(script, `-NonInteractive`) ||
		!strings.Contains(script, `-HermesHome $HermesHome -InstallDir $HermesInstallDir -Stage $installerStage`) ||
		!strings.Contains(script, `'bootstrap-marker'`) ||
		!strings.Contains(script, `Ensure-HermesRepositorySnapshot`) ||
		!strings.Contains(script, `https://codeload.github.com/NousResearch/hermes-agent/zip/refs/heads/main`) {
		t.Fatalf("expected script to invoke official Hermes PowerShell installer in explicit staged mode with repository snapshot prefetch, got:\n%s", script)
	}
	if !strings.Contains(script, `Ensure-HermesPortableFFmpeg`) ||
		!strings.Contains(script, `Download-RelayQFile`) ||
		!strings.Contains(script, `Start-BitsTransfer -Source $Url -Destination $Destination -Priority Foreground -ErrorAction Stop`) ||
		!strings.Contains(script, `rg version relayq-shim`) ||
		!strings.Contains(script, `ffmpeg version relayq-shim`) ||
		!strings.Contains(script, `ffprobe version relayq-shim`) ||
		!strings.Contains(script, `当前环境缺少 ripgrep/ffmpeg 等可选依赖，已自动写入临时 shim`) {
		t.Fatalf("expected script to provide non-blocking ripgrep/ffmpeg shim fallbacks before official installer, got:\n%s", script)
	}
	if !strings.Contains(script, `Join-Path $env:LOCALAPPDATA 'hermes'`) ||
		!strings.Contains(script, `Join-Path $HermesHome 'bin'`) ||
		!strings.Contains(script, `Join-Path $HermesInstallDir 'venv\Scripts\hermes.exe'`) ||
		!strings.Contains(script, `hermes.cmd`) {
		t.Fatalf("expected script to align Hermes home and native Windows shim lookup, got:\n%s", script)
	}
	if !strings.Contains(script, `Normalize-OpenAIBaseUrl`) || !strings.Contains(script, `if ($normalized.EndsWith('/v1/models'))`) {
		t.Fatalf("expected script to normalize wrapped or /models base urls, got:\n%s", script)
	}
	if !strings.Contains(script, "请先正确选择模型，再生成安装命令。") {
		t.Fatalf("expected script to require explicit model selection, got:\n%s", script)
	}
	if strings.Contains(script, "gpt-4.1-mini") {
		t.Fatalf("expected script to stop using hardcoded default model fallback, got:\n%s", script)
	}
	if !strings.Contains(script, `Invoke-HermesChecked @('config', 'set', 'model.provider', 'custom')`) ||
		!strings.Contains(script, `Invoke-HermesChecked @('config', 'set', 'model.base_url', $BaseUrl)`) ||
		!strings.Contains(script, `Invoke-HermesChecked @('config', 'set', 'model.default', $ModelName)`) ||
		!strings.Contains(script, `Invoke-HermesChecked @('config', 'set', 'model.api_key', '${OPENAI_API_KEY}')`) {
		t.Fatalf("expected script to write Hermes native Windows config, got:\n%s", script)
	}
	if !strings.Contains(script, `Update-DotEnv $EnvFile`) || !strings.Contains(script, `OPENAI_API_KEY = $ApiKey`) {
		t.Fatalf("expected script to merge env vars instead of overwriting blindly, got:\n%s", script)
	}
	if !strings.Contains(script, `Invoke-WebRequest -UseBasicParsing -SkipHttpErrorCheck -Headers @{ Authorization = "Bearer $ApiKey" } -Uri "$BaseUrl/models"`) {
		t.Fatalf("expected script to validate models endpoint on Windows, got:\n%s", script)
	}
	if !strings.Contains(script, `当前 token 没有访问 /models 的权限`) {
		t.Fatalf("expected script to explain 403 model permission failures, got:\n%s", script)
	}
	if !strings.Contains(script, `Insufficient account balance`) || !strings.Contains(script, `not assigned to any group`) {
		t.Fatalf("expected script to special-case balance and unassigned-group model failures, got:\n%s", script)
	}
	if !strings.Contains(script, `当前 API Key 对应账号余额不足`) || !strings.Contains(script, `当前 API Key 尚未绑定任何分组`) {
		t.Fatalf("expected script to show user-friendly balance and group guidance, got:\n%s", script)
	}
	if !strings.Contains(script, `安装已中止：请先修正 API Key、Base URL 或模型权限`) {
		t.Fatalf("expected script to fail fast before install when models preflight fails, got:\n%s", script)
	}
	if !strings.Contains(script, `& $HermesBin config check`) || !strings.Contains(script, `& $HermesBin doctor`) {
		t.Fatalf("expected script to run Hermes self-check commands after config, got:\n%s", script)
	}
	if strings.Contains(strings.ToLower(script), `beta`) {
		t.Fatalf("expected script to stop exposing legacy preview wording, got:\n%s", script)
	}
}

func TestBuildOpenClawLinuxUsesOfficialOnboarding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "https://relayq.example.com/install/openclaw-linux", nil)

	h := NewInstallScriptHandler(nil)
	script := h.buildOpenClawLinux(ctx)

	if !strings.Contains(script, `openclaw onboard --non-interactive --accept-risk --flow quickstart --mode local`) {
		t.Fatalf("expected linux script to run official onboarding, got:\n%s", script)
	}
	if !strings.Contains(script, `--accept-risk`) {
		t.Fatalf("expected linux script to acknowledge non-interactive onboarding risk, got:\n%s", script)
	}
	if !strings.Contains(script, `--auth-choice custom-api-key`) || !strings.Contains(script, `--custom-compatibility openai`) {
		t.Fatalf("expected linux script to configure custom provider onboarding, got:\n%s", script)
	}
	if !strings.Contains(script, `--custom-provider-id relayq`) {
		t.Fatalf("expected linux script to pin custom provider id to relayq, got:\n%s", script)
	}
	if strings.Contains(script, `--custom-provider-id openai`) {
		t.Fatalf("expected linux script to stop pinning custom provider id to openai, got:\n%s", script)
	}
	if !strings.Contains(script, `请先正确选择模型，再生成安装命令。`) {
		t.Fatalf("expected linux script to require explicit model selection, got:\n%s", script)
	}
	if !strings.Contains(script, `--install-daemon`) {
		t.Fatalf("expected linux script to install daemon during onboarding, got:\n%s", script)
	}
	if !strings.Contains(script, `openclaw dashboard`) {
		t.Fatalf("expected linux script to open dashboard after install, got:\n%s", script)
	}
	if strings.Contains(script, "claude-sonnet-4-6") {
		t.Fatalf("expected linux script to stop using hardcoded default model fallback, got:\n%s", script)
	}
	if !strings.Contains(script, `EXPECTED_MODEL="relayq/$MODEL_NAME"`) {
		t.Fatalf("expected linux script to validate relayq model namespace, got:\n%s", script)
	}
	if !strings.Contains(script, `"relayq" not in providers`) {
		t.Fatalf("expected linux script to validate relayq provider presence, got:\n%s", script)
	}
	if !strings.Contains(script, `MODELS_STATUS_OUTPUT="$(openclaw models status 2>&1)"`) || !strings.Contains(script, `grep -q 'relayq effective='`) {
		t.Fatalf("expected linux script to validate relayq auth from models status, got:\n%s", script)
	}
}

func TestBuildHermesLinuxStartsHermesImmediately(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "https://relayq.example.com/install/hermes-linux", nil)

	h := NewInstallScriptHandler(nil)
	script := h.buildHermesLinux(ctx)

	if !strings.Contains(script, `config set model.provider custom`) {
		t.Fatalf("expected linux script to configure model.provider custom, got:\n%s", script)
	}
	if !strings.Contains(script, `config set model.base_url "$BASE_URL"`) {
		t.Fatalf("expected linux script to configure model.base_url, got:\n%s", script)
	}
	if !strings.Contains(script, `config set model.api_key '${OPENAI_API_KEY}'`) {
		t.Fatalf("expected linux script to configure model.api_key via env reference, got:\n%s", script)
	}
	if !strings.Contains(script, `OPENAI_API_KEY`) || !strings.Contains(script, `OPENAI_BASE_URL`) {
		t.Fatalf("expected linux script to persist required env vars, got:\n%s", script)
	}
	if !strings.Contains(script, `curl -fsSL -H "Authorization: Bearer $API_KEY" "$BASE_URL/models"`) {
		t.Fatalf("expected linux script to validate models endpoint, got:\n%s", script)
	}
	if !strings.Contains(script, `请先正确选择模型，再生成安装命令。`) {
		t.Fatalf("expected linux script to require explicit model selection, got:\n%s", script)
	}
	if strings.Contains(script, "gpt-4.1-mini") {
		t.Fatalf("expected linux script to stop using hardcoded default model fallback, got:\n%s", script)
	}
	if !strings.Contains(script, `exec "$HERMES_BIN"`) {
		t.Fatalf("expected linux script to launch hermes immediately after install, got:\n%s", script)
	}
}
