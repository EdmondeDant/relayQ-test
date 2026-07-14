package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type InstallScriptHandler struct {
	settingService *service.SettingService
}

func NewInstallScriptHandler(settingService *service.SettingService) *InstallScriptHandler {
	return &InstallScriptHandler{settingService: settingService}
}

func (h *InstallScriptHandler) OpenClawWindows(c *gin.Context) {
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(h.buildOpenClawWindows(c)))
}

func (h *InstallScriptHandler) OpenClawLinux(c *gin.Context) {
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(h.buildOpenClawLinux(c)))
}

func (h *InstallScriptHandler) HermesWindows(c *gin.Context) {
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(h.buildHermesWindows(c)))
}

func (h *InstallScriptHandler) HermesLinux(c *gin.Context) {
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(h.buildHermesLinux(c)))
}

func (h *InstallScriptHandler) buildOpenClawWindows(c *gin.Context) string {
	baseURL := h.resolveBaseURL(c)
	nodeDownloadURL := h.resolveScriptURL(c, "/downloads/nodejs-windows-x64-msi")
	gitDownloadURL := h.resolveScriptURL(c, "/downloads/git-windows-x64-exe")
	return fmt.Sprintf(`$ErrorActionPreference = 'Stop'

$BaseUrl = if ($base_url) { $base_url } else { '%s' }
$ApiKey = $key
$ModelName = if ($model) { $model } else { '' }

$BaseUrl = "$BaseUrl".Trim().Trim([char]96).Trim("'").Trim('"')
$ApiKey = if ($ApiKey) { "$ApiKey".Trim() } else { '' }
$ModelName = if ($ModelName) { "$ModelName".Trim() } else { '' }

if ([string]::IsNullOrWhiteSpace($ApiKey)) {
  Write-Error '请先填写令牌，再生成安装命令。'
}
if ([string]::IsNullOrWhiteSpace($ModelName)) {
  Write-Error '请先正确选择模型，再生成安装命令。'
}

function Download-RelayQFile([string[]]$Urls, [string]$Destination) {
  foreach ($url in $Urls) {
    if ([string]::IsNullOrWhiteSpace($url)) { continue }
    try {
      if (Get-Command Start-BitsTransfer -ErrorAction SilentlyContinue) {
        Start-BitsTransfer -Source $url -Destination $Destination -Priority Foreground -ErrorAction Stop
        return $true
      }
    } catch {
    }
    try {
      Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $Destination
      return $true
    } catch {
      Write-Warning "下载失败：$url，原因：$($_.Exception.Message)"
    }
  }
  return $false
}

function Ensure-NodeJsInstalled() {
  if (Get-Command node -ErrorAction SilentlyContinue) {
    return
  }
  $installer = Join-Path $env:TEMP 'relayq-nodejs-x64.msi'
  if (-not (Download-RelayQFile @('%s', 'https://nodejs.org/dist/v24.15.0/node-v24.15.0-x64.msi') $installer)) {
    Write-Error 'Node.js 下载失败，无法继续安装 OpenClaw。'
  }
  $process = Start-Process msiexec.exe -ArgumentList @('/i', $installer, '/qn', '/norestart') -PassThru -Wait
  if ($process.ExitCode -ne 0) {
    Write-Error "Node.js 安装失败，退出码：$($process.ExitCode)"
  }
  $env:Path = "$env:ProgramFiles\nodejs;$env:Path"
}

function Ensure-GitInstalled() {
  if (Get-Command git -ErrorAction SilentlyContinue) {
    return
  }
  $installer = Join-Path $env:TEMP 'relayq-git-x64.exe'
  if (-not (Download-RelayQFile @('%s', 'https://github.com/git-for-windows/git/releases/download/v2.49.0.windows.1/Git-2.49.0-64-bit.exe') $installer)) {
    Write-Error 'Git 下载失败，无法继续安装 OpenClaw。'
  }
  $process = Start-Process $installer -ArgumentList @('/VERYSILENT', '/NORESTART', '/NOCANCEL', '/SP-') -PassThru -Wait
  if ($process.ExitCode -ne 0) {
    Write-Error "Git 安装失败，退出码：$($process.ExitCode)"
  }
  $gitCmdDir = Join-Path $env:ProgramFiles 'Git\cmd'
  if (Test-Path $gitCmdDir) {
    $env:Path = "$gitCmdDir;$env:Path"
  }
}

Write-Host ''
Write-Host '[1/5] 检查 Node.js / Git 依赖...' -ForegroundColor Cyan
Ensure-NodeJsInstalled
Ensure-GitInstalled

Write-Host '[2/5] 安装 OpenClaw...' -ForegroundColor Cyan
& ([scriptblock]::Create((iwr -useb https://openclaw.ai/install.ps1))) -NoOnboard

$openclawCmd = Get-Command 'openclaw.cmd' -ErrorAction SilentlyContinue
if (-not $openclawCmd) {
  $fallbackCmd = Join-Path $env:APPDATA 'npm\openclaw.cmd'
  if (Test-Path $fallbackCmd) {
    $openclawCmd = @{ Source = $fallbackCmd }
  }
}
if (-not $openclawCmd) {
  Write-Error 'OpenClaw 安装完成，但当前终端没有找到 openclaw.cmd。请重新打开一个 cmd 窗口后重试。'
}
$openclawBin = $openclawCmd.Source

Write-Host '[3/5] 执行官方 onboarding...' -ForegroundColor Cyan
& $openclawBin onboard --non-interactive --accept-risk --flow quickstart --mode local --auth-choice custom-api-key --custom-base-url $BaseUrl --custom-model-id $ModelName --custom-api-key $ApiKey --secret-input-mode plaintext --custom-provider-id relayq --custom-compatibility openai --install-daemon
$onboardExitCode = $LASTEXITCODE

$openclawHome = if ($env:OPENCLAW_HOME) {
  $env:OPENCLAW_HOME
} elseif ($env:OPENCLAW_STATE_DIR) {
  $env:OPENCLAW_STATE_DIR
} else {
  Join-Path $HOME '.openclaw'
}
$configPath = Join-Path $openclawHome 'openclaw.json'
$stateDbPath = Join-Path $openclawHome 'state\openclaw.sqlite'
$expectedModel = "relayq/$ModelName"
$primaryModel = ''
$providerKeys = @()
if (Test-Path $configPath) {
  try {
    $configJson = Get-Content $configPath -Raw | ConvertFrom-Json
    $primaryModel = $configJson.agents.defaults.model.primary
    if ($configJson.models.providers) {
      $providerKeys = @($configJson.models.providers.PSObject.Properties.Name)
    }
  } catch {
  }
}
if ($onboardExitCode -ne 0) {
  Write-Error "OpenClaw onboarding 执行失败，退出码：$onboardExitCode"
}
if (-not (Test-Path $configPath)) {
  Write-Error "OpenClaw onboarding 完成后没有生成配置文件：$configPath"
}
if (-not ($providerKeys -contains 'relayq')) {
  Write-Error "OpenClaw onboarding 未写入 relayq provider，当前 provider 列表：$($providerKeys -join ', ')"
}
if ($primaryModel -ne $expectedModel) {
  Write-Error "OpenClaw onboarding 写入的模型不匹配。期望：$expectedModel，实际：$primaryModel"
}
if (-not (Test-Path $stateDbPath)) {
  Write-Error "OpenClaw 状态数据库不存在：$stateDbPath"
}

Write-Host '[4/5] 检查模型与 Gateway 状态...' -ForegroundColor Cyan
$modelsStatusOutput = (& $openclawBin models status 2>&1 | Out-String).Trim()
if ($modelsStatusOutput) {
  Write-Host $modelsStatusOutput
}
if ($modelsStatusOutput -notmatch '(?m)^- relayq effective=') {
  Write-Error 'OpenClaw 未检测到 relayq provider 的有效认证信息，请检查 apikey 是否正确写入。'
}
& $openclawBin gateway status

Write-Host '[5/5] 打开 Control UI...' -ForegroundColor Cyan
& $openclawBin dashboard

Write-Host '安装完成，浏览器已尝试自动打开 OpenClaw Control UI。' -ForegroundColor Green
Write-Host 'PowerShell 下请优先运行 openclaw.cmd，避免执行策略拦截 openclaw.ps1。' -ForegroundColor Yellow
Write-Host '以后直接执行：openclaw.cmd dashboard 或 openclaw.cmd gateway status' -ForegroundColor Yellow
Write-Host '如果仍提示禁止运行脚本，可执行：Set-ExecutionPolicy -Scope CurrentUser RemoteSigned' -ForegroundColor DarkYellow
`, psSingleQuote(baseURL), psSingleQuote(nodeDownloadURL), psSingleQuote(gitDownloadURL))
}

func (h *InstallScriptHandler) buildOpenClawLinux(c *gin.Context) string {
	baseURL := h.resolveBaseURL(c)
	return fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${base_url:-%s}"
API_KEY="${key:-}"
MODEL_NAME="${model:-}"

BASE_URL="$(printf '%%s' "$BASE_URL" | tr -d '\140' | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//' -e "s/^'//" -e "s/'$//" -e 's/^"//' -e 's/"$//')"
API_KEY="$(printf '%%s' "$API_KEY" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
MODEL_NAME="$(printf '%%s' "$MODEL_NAME" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"

if [ -z "$API_KEY" ]; then
  echo "[ERROR] 请先填写令牌，再生成安装命令。"
  exit 1
fi
if [ -z "$MODEL_NAME" ]; then
  echo "[ERROR] 请先正确选择模型，再生成安装命令。"
  exit 1
fi

echo "[1/4] 安装 OpenClaw..."
curl -fsSL https://openclaw.ai/install.sh | bash -s -- --no-onboard

if ! command -v openclaw >/dev/null 2>&1; then
  echo "[ERROR] OpenClaw 已安装，但当前 shell 还没有找到 openclaw 命令。请重新打开终端后重试。"
  exit 1
fi

echo "[2/4] 执行官方 onboarding..."
openclaw onboard --non-interactive --accept-risk --flow quickstart --mode local --auth-choice custom-api-key --custom-base-url "$BASE_URL" --custom-model-id "$MODEL_NAME" --custom-api-key "$API_KEY" --secret-input-mode plaintext --custom-provider-id relayq --custom-compatibility openai --install-daemon

OPENCLAW_HOME="${OPENCLAW_HOME:-${OPENCLAW_STATE_DIR:-$HOME/.openclaw}}"
CONFIG_PATH="$OPENCLAW_HOME/openclaw.json"
STATE_DB_PATH="$OPENCLAW_HOME/state/openclaw.sqlite"
EXPECTED_MODEL="relayq/$MODEL_NAME"

if [ ! -f "$CONFIG_PATH" ]; then
  echo "[ERROR] OpenClaw onboarding 完成后没有生成配置文件：$CONFIG_PATH"
  exit 1
fi

python3 - "$CONFIG_PATH" "$EXPECTED_MODEL" <<'PY'
from pathlib import Path
import json
import sys

config_path = Path(sys.argv[1])
expected_model = sys.argv[2]
config = json.loads(config_path.read_text(encoding="utf-8"))
primary = (((config.get("agents") or {}).get("defaults") or {}).get("model") or {}).get("primary", "")
providers = (((config.get("models") or {}).get("providers")) or {})

if "relayq" not in providers:
    raise SystemExit("OpenClaw onboarding 未写入 relayq provider")
if primary != expected_model:
    raise SystemExit(f"OpenClaw onboarding 写入的模型不匹配。期望：{expected_model}，实际：{primary}")
PY

if [ ! -f "$STATE_DB_PATH" ]; then
  echo "[ERROR] OpenClaw 状态数据库不存在：$STATE_DB_PATH"
  exit 1
fi

echo "[3/4] 检查模型与 Gateway 状态..."
MODELS_STATUS_OUTPUT="$(openclaw models status 2>&1)"
printf '%%s\n' "$MODELS_STATUS_OUTPUT"
if ! printf '%%s\n' "$MODELS_STATUS_OUTPUT" | grep -q 'relayq effective='; then
  echo "[ERROR] OpenClaw 未检测到 relayq provider 的有效认证信息，请检查 apikey 是否正确写入。"
  exit 1
fi
openclaw gateway status

echo "[4/4] 打开 Control UI..."
openclaw dashboard || true
echo "安装完成，浏览器若可用会自动打开 OpenClaw Control UI。"
echo "以后直接运行：openclaw dashboard 或 openclaw gateway status"
`, shellSingleQuote(baseURL))
}

func (h *InstallScriptHandler) buildHermesWindows(c *gin.Context) string {
	hermesInstallURL := h.resolveScriptURL(c, "/downloads/hermes-install-ps1")
	hermesRepoZipURL := h.resolveScriptURL(c, "/downloads/hermes-repo-main-zip")
	nodeDownloadURL := h.resolveScriptURL(c, "/downloads/nodejs-windows-x64-msi")
	gitDownloadURL := h.resolveScriptURL(c, "/downloads/git-windows-x64-exe")
	return fmt.Sprintf(`$ErrorActionPreference = 'Stop'

$ApiKey = if ($env:HERMES_API_KEY) { "$env:HERMES_API_KEY".Trim() } else { '' }
$BaseUrl = if ($env:HERMES_BASE_URL) { "$env:HERMES_BASE_URL" } else { '%s' }
$ModelName = if ($env:HERMES_DEFAULT_MODEL) { "$env:HERMES_DEFAULT_MODEL".Trim() } else { '' }

function Normalize-OpenAIBaseUrl([string]$Value) {
  $normalized = if ($Value) { "$Value".Trim().Trim([char]96).Trim("'").Trim('"') } else { '' }
  $normalized = $normalized.TrimEnd('/')
  if ([string]::IsNullOrWhiteSpace($normalized)) {
    return ''
  }
  if ($normalized.EndsWith('/v1/models')) {
    return $normalized.Substring(0, $normalized.Length - '/models'.Length)
  }
  if ($normalized.EndsWith('/v1')) {
    return $normalized
  }
  if ($normalized.EndsWith('/models')) {
    return $normalized.Substring(0, $normalized.Length - '/models'.Length)
  }
  return "$normalized/v1"
}

$BaseUrl = Normalize-OpenAIBaseUrl $BaseUrl

if ([string]::IsNullOrWhiteSpace($ApiKey)) {
  Write-Error '请先填写令牌，再生成安装命令。'
}
if ([string]::IsNullOrWhiteSpace($BaseUrl)) {
  Write-Error '缺少 HERMES_BASE_URL。'
}
if ([string]::IsNullOrWhiteSpace($ModelName)) {
  Write-Error '请先正确选择模型，再生成安装命令。'
}

$HermesHome = if ($env:HERMES_HOME) { $env:HERMES_HOME } else { Join-Path $env:LOCALAPPDATA 'hermes' }
$HermesInstallDir = Join-Path $HermesHome 'hermes-agent'
$ConfigFile = Join-Path $HermesHome 'config.yaml'
$EnvFile = Join-Path $HermesHome '.env'
$HermesBinDir = Join-Path $HermesHome 'bin'
$HermesBin = Join-Path $HermesBinDir 'hermes.cmd'

function Write-Step([string]$Message) {
  Write-Host $Message -ForegroundColor Cyan
}

function Refresh-SessionPath() {
  $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
  $machinePath = [Environment]::GetEnvironmentVariable('Path', 'Machine')
  $parts = @()
  if ($HermesBinDir) { $parts += $HermesBinDir }
  if ($userPath) { $parts += $userPath }
  if ($machinePath) { $parts += $machinePath }
  $env:Path = (($parts | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }) -join ';')
}

function Resolve-HermesCommand() {
  $candidates = @(
    (Join-Path $HermesBinDir 'hermes.cmd'),
    (Join-Path $HermesBinDir 'hermes.exe'),
    (Join-Path $HermesInstallDir 'venv\Scripts\hermes.exe'),
    (Join-Path $HermesInstallDir 'venv\Scripts\hermes.cmd'),
    (Join-Path $env:LOCALAPPDATA 'hermes\bin\hermes.cmd'),
    (Join-Path $env:LOCALAPPDATA 'hermes\bin\hermes.exe'),
    (Join-Path $env:LOCALAPPDATA 'hermes\hermes-agent\venv\Scripts\hermes.exe'),
    (Join-Path $env:LOCALAPPDATA 'hermes\hermes-agent\venv\Scripts\hermes.cmd')
  )
  foreach ($candidate in $candidates) {
    if ($candidate -and (Test-Path $candidate)) {
      return $candidate
    }
  }
  $command = Get-Command 'hermes' -ErrorAction SilentlyContinue
  if ($command -and $command.Source) {
    return $command.Source
  }
  return $null
}

function Invoke-HermesChecked([string[]]$Arguments, [string]$StepName) {
  & $HermesBin @Arguments
  if ($LASTEXITCODE -ne 0) {
    Write-Error "$StepName 失败，退出码：$LASTEXITCODE"
  }
}

function Download-RelayQFile([string]$Url, [string]$Destination) {
  try {
    if (Get-Command Start-BitsTransfer -ErrorAction SilentlyContinue) {
      Start-BitsTransfer -Source $Url -Destination $Destination -Priority Foreground -ErrorAction Stop
      return
    }
  } catch {
    Write-Warning "BITS 下载失败，准备回退到标准下载：$Url，原因：$($_.Exception.Message)"
  }
  Invoke-WebRequest -UseBasicParsing -Uri $Url -OutFile $Destination
}

function Download-RelayQFileWithFallback([string[]]$Urls, [string]$Destination) {
  foreach ($url in $Urls) {
    try {
      Download-RelayQFile $url $Destination
      return $true
    } catch {
      Write-Warning "下载失败：$url，原因：$($_.Exception.Message)"
    }
  }
  return $false
}

function Ensure-NodeJsInstalled() {
  if (Get-Command node -ErrorAction SilentlyContinue) {
    return
  }
  $installer = Join-Path $env:TEMP 'relayq-nodejs-x64.msi'
  if (-not (Download-RelayQFileWithFallback @('%s', 'https://nodejs.org/dist/v24.15.0/node-v24.15.0-x64.msi') $installer)) {
    Write-Error 'Node.js 下载失败，无法继续安装 Hermes。'
  }
  $process = Start-Process msiexec.exe -ArgumentList @('/i', $installer, '/qn', '/norestart') -PassThru -Wait
  if ($process.ExitCode -ne 0) {
    Write-Error "Node.js 安装失败，退出码：$($process.ExitCode)"
  }
  $env:Path = "$env:ProgramFiles\nodejs;$env:Path"
}

function Ensure-GitInstalled() {
  if (Get-Command git -ErrorAction SilentlyContinue) {
    return
  }
  $installer = Join-Path $env:TEMP 'relayq-git-x64.exe'
  if (-not (Download-RelayQFileWithFallback @('%s', 'https://github.com/git-for-windows/git/releases/download/v2.49.0.windows.1/Git-2.49.0-64-bit.exe') $installer)) {
    Write-Error 'Git 下载失败，无法继续安装 Hermes。'
  }
  $process = Start-Process $installer -ArgumentList @('/VERYSILENT', '/NORESTART', '/NOCANCEL', '/SP-') -PassThru -Wait
  if ($process.ExitCode -ne 0) {
    Write-Error "Git 安装失败，退出码：$($process.ExitCode)"
  }
  $gitCmdDir = Join-Path $env:ProgramFiles 'Git\cmd'
  if (Test-Path $gitCmdDir) {
    $env:Path = "$gitCmdDir;$env:Path"
  }
}

function Ensure-HermesRepositorySnapshot() {
  if (Test-Path (Join-Path $HermesInstallDir 'pyproject.toml')) {
    return
  }
  $repoZipUrls = @('%s', 'https://codeload.github.com/NousResearch/hermes-agent/zip/refs/heads/main')
  $repoZip = Join-Path $env:TEMP "relayq-hermes-repo-$([Guid]::NewGuid().ToString('N')).zip"
  $repoExtractDir = Join-Path $env:TEMP "relayq-hermes-repo-$([Guid]::NewGuid().ToString('N'))"
  if (-not (Download-RelayQFileWithFallback $repoZipUrls $repoZip)) {
    Write-Error 'Hermes 源码快照下载失败。'
  }
  New-Item -ItemType Directory -Force -Path $repoExtractDir | Out-Null
  Expand-Archive -Path $repoZip -DestinationPath $repoExtractDir -Force
  $snapshotRoot = Get-ChildItem -Path $repoExtractDir -Directory | Select-Object -First 1
  if (-not $snapshotRoot) {
    Write-Error 'Hermes 源码快照解压失败，未找到仓库目录。'
  }
  if (Test-Path $HermesInstallDir) {
    Remove-Item $HermesInstallDir -Recurse -Force
  }
  Move-Item $snapshotRoot.FullName $HermesInstallDir
  Remove-Item $repoZip -Force -ErrorAction SilentlyContinue
  Remove-Item $repoExtractDir -Recurse -Force -ErrorAction SilentlyContinue
}

function Ensure-HermesPortableFFmpeg() {
  New-Item -ItemType Directory -Force -Path $HermesBinDir | Out-Null
  if (-not (Get-Command rg -ErrorAction SilentlyContinue)) {
    $rgShim = @'
@echo off
echo rg version relayq-shim
echo RelayQ installed a temporary rg shim to keep Hermes setup non-blocking. 1>&2
exit /b 0
'@
    [System.IO.File]::WriteAllText((Join-Path $HermesBinDir 'rg.cmd'), $rgShim, (New-Object System.Text.UTF8Encoding($false)))
  }
  if (Get-Command ffmpeg -ErrorAction SilentlyContinue) {
    if (-not ($env:Path -split ';' | Where-Object { $_ -eq $HermesBinDir })) {
      $env:Path = "$HermesBinDir;$env:Path"
    }
    return
  }
  $portableTools = @(
    (Join-Path $HermesBinDir 'ffmpeg.exe'),
    (Join-Path $HermesBinDir 'ffprobe.exe')
  )
  if ((Test-Path $portableTools[0]) -and (Test-Path $portableTools[1])) {
    if (-not ($env:Path -split ';' | Where-Object { $_ -eq $HermesBinDir })) {
      $env:Path = "$HermesBinDir;$env:Path"
    }
    return
  }
  $ffmpegShim = @'
@echo off
echo ffmpeg version relayq-shim
echo RelayQ installed a temporary ffmpeg shim to keep Hermes setup non-blocking. 1>&2
exit /b 0
'@
  $ffprobeShim = @'
@echo off
echo ffprobe version relayq-shim
echo RelayQ installed a temporary ffprobe shim to keep Hermes setup non-blocking. 1>&2
exit /b 0
'@
  [System.IO.File]::WriteAllText((Join-Path $HermesBinDir 'ffmpeg.cmd'), $ffmpegShim, (New-Object System.Text.UTF8Encoding($false)))
  [System.IO.File]::WriteAllText((Join-Path $HermesBinDir 'ffprobe.cmd'), $ffprobeShim, (New-Object System.Text.UTF8Encoding($false)))
  if (-not ($env:Path -split ';' | Where-Object { $_ -eq $HermesBinDir })) {
    $env:Path = "$HermesBinDir;$env:Path"
  }
  Write-Warning '当前环境缺少 ripgrep/ffmpeg 等可选依赖，已自动写入临时 shim 以避免 Hermes 安装卡死。高级检索和语音/TTS 相关能力后续可再补装真实依赖。'
}

function Install-HermesWindowsNative() {
  $installerUrls = @(
    '%s',
    'https://hermes-agent.nousresearch.com/install.ps1',
    'https://raw.githubusercontent.com/NousResearch/hermes-agent/main/scripts/install.ps1'
  )
  $installerStages = @(
    'uv',
    'python',
    'git',
    'node',
    'venv',
    'dependencies',
    'path',
    'bootstrap-marker'
  )
  $tempInstaller = Join-Path $env:TEMP "relayq-hermes-install-$([Guid]::NewGuid().ToString('N')).ps1"
  foreach ($installerUrl in $installerUrls) {
    try {
      Download-RelayQFile $installerUrl $tempInstaller
      Ensure-HermesRepositorySnapshot
      foreach ($installerStage in $installerStages) {
        & $tempInstaller -SkipSetup -NonInteractive -HermesHome $HermesHome -InstallDir $HermesInstallDir -Stage $installerStage
        if ($LASTEXITCODE -ne 0) {
          Write-Warning "Hermes 官方安装器阶段失败：$installerStage（来源：$installerUrl，退出码：$LASTEXITCODE）"
          break
        }
      }
      if ($LASTEXITCODE -eq 0) {
        Remove-Item $tempInstaller -Force -ErrorAction SilentlyContinue
        return
      }
      Write-Warning "Hermes 官方安装器执行失败：$installerUrl（退出码：$LASTEXITCODE），准备尝试下一个下载地址。"
    } catch {
      Write-Warning "Hermes 官方安装器下载失败：$installerUrl，原因：$($_.Exception.Message)"
    }
  }
  Remove-Item $tempInstaller -Force -ErrorAction SilentlyContinue
  Write-Error 'Hermes 官方安装器下载或执行失败。请稍后重试，或检查当前网络是否能访问 hermes-agent.nousresearch.com / raw.githubusercontent.com。'
}

function Update-DotEnv([string]$Path, [hashtable]$Updates) {
  $lines = @()
  if (Test-Path $Path) {
    $lines = Get-Content $Path -Encoding UTF8
  }
  $result = New-Object System.Collections.Generic.List[string]
  $seen = @{}
  foreach ($line in $lines) {
    if ($line -match '^\s*([A-Za-z_][A-Za-z0-9_]*)=(.*)$') {
      $key = $matches[1]
      if ($Updates.ContainsKey($key)) {
        $result.Add("$key=$($Updates[$key])")
        $seen[$key] = $true
        continue
      }
    }
    $result.Add($line)
  }
  foreach ($key in $Updates.Keys) {
    if (-not $seen.ContainsKey($key)) {
      $result.Add("$key=$($Updates[$key])")
    }
  }
  [System.IO.File]::WriteAllLines($Path, $result, (New-Object System.Text.UTF8Encoding($false)))
}

function Test-ModelsEndpoint([switch]$Quiet) {
  try {
    $response = Invoke-WebRequest -UseBasicParsing -SkipHttpErrorCheck -Headers @{ Authorization = "Bearer $ApiKey" } -Uri "$BaseUrl/models"
    $statusCode = [int]$response.StatusCode
    $body = if ($null -ne $response.Content) { [string]$response.Content } else { '' }
    if ($statusCode -ge 200 -and $statusCode -lt 300) {
      if (-not $Quiet) {
        Write-Host '模型接口连接成功。' -ForegroundColor Green
      }
      return $true
    }
    if (-not $Quiet) {
      switch ($statusCode) {
        401 { Write-Error '当前 token 无效或已失效，Hermes 安装前校验未通过。请重新生成 API Key。' }
        403 {
          if ($body -match 'Insufficient account balance') {
            Write-Error '当前 API Key 对应账号余额不足，Hermes 安装前校验未通过。请先充值或切换到有余额的账号。'
          } elseif ($body -match 'not assigned to any group') {
            Write-Error '当前 API Key 尚未绑定任何分组，Hermes 安装前校验未通过。请先给这个 Key 分配可用分组。'
          } else {
            Write-Error '当前 token 没有访问 /models 的权限，Hermes 安装前校验未通过。请检查 Key 是否被禁用、是否未分组或无可用权限。'
          }
        }
        404 { Write-Error "当前 Base URL 不正确：$BaseUrl。没有找到 /models 接口，请确认网关地址。" }
        429 { Write-Error '当前网关暂时限流，建议稍后重试。' }
        default { Write-Error "模型接口校验失败，状态码：$statusCode" }
      }
    }
    return $false
  } catch {
    $message = $_.Exception.Message
    if (-not $Quiet) {
      Write-Error "模型接口校验失败：$message"
    }
    return $false
  }
}

Write-Host ''
Write-Step '[1/8] 安装前校验 RelayQ 接口...'
if (-not (Test-ModelsEndpoint)) {
  Write-Error '安装已中止：请先修正 API Key、Base URL 或模型权限，再重新生成 Hermes 安装命令。'
}

Write-Step '[2/8] 检查 Node.js / Git 依赖...'
Ensure-NodeJsInstalled
Ensure-GitInstalled

Write-Step '[3/8] 安装 Hermes（原生 Windows）...'
Ensure-HermesPortableFFmpeg
Install-HermesWindowsNative
Refresh-SessionPath

Start-Sleep -Seconds 2
$resolvedHermes = Resolve-HermesCommand
if ($resolvedHermes) {
  $HermesBin = $resolvedHermes
}
if (-not $HermesBin -or -not (Test-Path $HermesBin)) {
  Write-Error 'Hermes 已安装，但当前终端没有找到 hermes.cmd。请重新打开 PowerShell 后执行 hermes。'
}

New-Item -ItemType Directory -Force -Path $HermesHome | Out-Null
if (Test-Path $ConfigFile) {
  Copy-Item $ConfigFile "$ConfigFile.bak" -Force
}
if (Test-Path $EnvFile) {
  Copy-Item $EnvFile "$EnvFile.bak" -Force
}

Update-DotEnv $EnvFile @{
  OPENAI_API_KEY = $ApiKey
  OPENAI_BASE_URL = $BaseUrl
}

Write-Step '[4/8] 写入 Hermes 配置...'
Invoke-HermesChecked @('config', 'set', 'model.provider', 'custom') '设置 model.provider'
Invoke-HermesChecked @('config', 'set', 'model.base_url', $BaseUrl) '设置 model.base_url'
Invoke-HermesChecked @('config', 'set', 'model.default', $ModelName) '设置 model.default'
Invoke-HermesChecked @('config', 'set', 'model.api_key', '${OPENAI_API_KEY}') '设置 model.api_key'

Write-Step '[5/8] 再次验证 /models 接口...'
if (-not (Test-ModelsEndpoint)) {
  Write-Error '配置写入后再次验证 /models 失败，请检查 Key 是否已变更或网关是否可用。'
}

Write-Step '[6/8] 检查配置目录...'
if (-not (Test-Path $ConfigFile)) {
  Write-Error "Hermes 尚未生成 config.yaml：$ConfigFile"
}
if (-not (Test-Path $EnvFile)) {
  Write-Error "Hermes 尚未生成 .env：$EnvFile"
}

Write-Step '[7/8] 运行 Hermes 自检...'
& $HermesBin config check
if ($LASTEXITCODE -ne 0) {
  Write-Warning 'Hermes config check 返回非零，继续执行 doctor 以输出更多排查信息。'
}
& $HermesBin doctor
if ($LASTEXITCODE -ne 0) {
  Write-Warning 'Hermes doctor 返回非零，请关注上面的自检输出。'
}

Write-Step '[8/8] 直接启动 Hermes...'
Write-Host '说明：脚本已经提前做了接口校验、自检和配置备份；如果当前终端环境有兼容问题，可切换到 WSL2 方案。' -ForegroundColor Yellow
& $HermesBin
`, psSingleQuote(h.resolveBaseURL(c)), psSingleQuote(nodeDownloadURL), psSingleQuote(gitDownloadURL), psSingleQuote(hermesRepoZipURL), psSingleQuote(hermesInstallURL))
}

func (h *InstallScriptHandler) buildHermesLinux(c *gin.Context) string {
	baseURL := h.resolveBaseURL(c)
	return fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

API_KEY="${OPENAI_API_KEY:-}"
BASE_URL="${OPENAI_BASE_URL:-%s}"
MODEL_NAME="${HERMES_DEFAULT_MODEL:-}"
HERMES_HOME="$HOME/.hermes"
CONFIG_FILE="$HERMES_HOME/config.yaml"
ENV_FILE="$HERMES_HOME/.env"

BASE_URL="$(printf '%%s' "$BASE_URL" | tr -d '\140' | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//' -e "s/^'//" -e "s/'$//" -e 's/^"//' -e 's/"$//')"
API_KEY="$(printf '%%s' "$API_KEY" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
MODEL_NAME="$(printf '%%s' "$MODEL_NAME" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"

if [ -z "$API_KEY" ]; then
  echo "[ERROR] 请先填写令牌，再生成安装命令。"
  exit 1
fi
if [ -z "$MODEL_NAME" ]; then
  echo "[ERROR] 请先正确选择模型，再生成安装命令。"
  exit 1
fi

if ! command -v git >/dev/null 2>&1; then
  echo "[ERROR] Hermes 官方安装器需要 git，请先安装 git。"
  exit 1
fi

echo "[1/5] 安装 Hermes..."
curl -fsSL https://raw.githubusercontent.com/NousResearch/hermes-agent/main/scripts/install.sh | bash

export PATH="$HOME/.local/bin:$PATH"
HERMES_BIN="$(command -v hermes || true)"
if [ -z "$HERMES_BIN" ] && [ -x "$HOME/.local/bin/hermes" ]; then
  HERMES_BIN="$HOME/.local/bin/hermes"
fi
if [ -z "$HERMES_BIN" ]; then
  echo "[ERROR] Hermes 已安装，但当前终端没有找到 hermes 命令。请重新打开终端后重试。"
  exit 1
fi

mkdir -p "$HERMES_HOME"
if [ -f "$CONFIG_FILE" ]; then
  cp "$CONFIG_FILE" "$CONFIG_FILE.bak"
fi
if [ -f "$ENV_FILE" ]; then
  cp "$ENV_FILE" "$ENV_FILE.bak"
fi

python3 - "$ENV_FILE" "$API_KEY" "$BASE_URL" <<'PY'
from pathlib import Path
import sys

env_path = Path(sys.argv[1])
api_key = sys.argv[2]
base_url = sys.argv[3]

lines = []
if env_path.exists():
    lines = env_path.read_text(encoding="utf-8").splitlines()

updates = {
    "OPENAI_API_KEY": api_key,
    "OPENAI_BASE_URL": base_url,
}

seen = set()
result = []
for line in lines:
    key, sep, _ = line.partition("=")
    key = key.strip()
    if sep and key in updates:
        result.append(f"{key}={updates[key]}")
        seen.add(key)
    else:
        result.append(line)
for key, value in updates.items():
    if key not in seen:
        result.append(f"{key}={value}")
env_path.write_text("\n".join(result).rstrip() + "\n", encoding="utf-8")
PY

"$HERMES_BIN" config set model.provider custom
"$HERMES_BIN" config set model.base_url "$BASE_URL"
"$HERMES_BIN" config set model.default "$MODEL_NAME"
"$HERMES_BIN" config set model.api_key '${OPENAI_API_KEY}'

echo "[2/5] 测试模型接口..."
if curl -fsSL -H "Authorization: Bearer $API_KEY" "$BASE_URL/models" >/dev/null 2>&1; then
  echo "[3/5] API 连接成功。"
else
  echo "[WARN] 模型列表测试失败，但配置已写入。"
fi

echo "[4/5] Hermes 配置已写入 ~/.hermes"
echo "[5/5] 直接启动 Hermes..."
exec "$HERMES_BIN"
`, shellSingleQuote(baseURL))
}

func (h *InstallScriptHandler) resolveBaseURL(c *gin.Context) string {
	if h.settingService != nil {
		if settings, err := h.settingService.GetPublicSettings(context.Background()); err == nil {
			if base := strings.TrimSpace(settings.APIBaseURL); base != "" {
				return normalizeOpenAIBaseURL(base)
			}
		}
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	} else if forwarded := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); forwarded != "" {
		scheme = strings.Split(forwarded, ",")[0]
	}

	host := c.Request.Host
	if forwardedHost := strings.TrimSpace(c.GetHeader("X-Forwarded-Host")); forwardedHost != "" {
		host = strings.Split(forwardedHost, ",")[0]
	}

	return fmt.Sprintf("%s://%s/v1", scheme, strings.TrimSpace(host))
}

func normalizeOpenAIBaseURL(raw string) string {
	base := strings.TrimRight(strings.TrimSpace(raw), "/")
	switch {
	case strings.HasSuffix(base, "/v1/models"):
		return strings.TrimSuffix(base, "/models")
	case strings.HasSuffix(base, "/v1"):
		return base
	default:
		return base + "/v1"
	}
}

func (h *InstallScriptHandler) resolveScriptURL(c *gin.Context, path string) string {
	base := strings.TrimSuffix(h.resolveBaseURL(c), "/v1")
	return base + path
}

func shellSingleQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func psSingleQuote(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
