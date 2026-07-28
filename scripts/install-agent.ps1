param(
    [string]$Token = $env:AGENT_ENROLLMENT_TOKEN,
    [string]$ControlPlane = $(if ($env:AGENT_CONTROL_PLANE) { $env:AGENT_CONTROL_PLANE } else { "http://localhost:5000" }),
    [string]$AgentDir = $(if ($env:AGENT_DIR) { $env:AGENT_DIR } else { "C:\probe-agent" })
)

if (-not $Token) {
    Write-Host "Usage:"
    Write-Host "  irm https://raw.githubusercontent.com/mhghr/dog/main/scripts/install-agent.ps1 | iex"
    Write-Host "  OR"
    Write-Host "  .\install-agent.ps1 -Token YOUR_TOKEN -ControlPlane http://server:5000"
    exit 1
}

Write-Host "==================================="
Write-Host " probe-agent installer"
Write-Host "==================================="
Write-Host " Control plane : $ControlPlane"
Write-Host " Install dir   : $AgentDir"
Write-Host "==================================="

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "[ERROR] Go is not installed. Install from https://go.dev/dl/"
    exit 1
}

Write-Host "[1/4] Creating directories..."
New-Item -ItemType Directory -Force -Path "$AgentDir\state", "$AgentDir\spool" | Out-Null

if (-not (Test-Path "$AgentDir\probe-agent.exe")) {
    Write-Host "[2/4] Building probe-agent..."
    $tmp = Join-Path $env:TEMP "probe-agent-build-$([Guid]::NewGuid())"
    git clone --depth 1 https://github.com/mhghr/dog.git $tmp
    Push-Location $tmp
    go build -ldflags="-s -w" -o "$AgentDir\probe-agent.exe" .\cmd\probe-agent
    Pop-Location
    Remove-Item -Recurse -Force $tmp
    Write-Host "   Binary installed to $AgentDir\probe-agent.exe"
} else {
    Write-Host "[2/4] Binary already exists, skipping build"
}

Write-Host "[3/4] Writing config..."
$config = @"
control_plane: "$ControlPlane"
agent_gateway: "localhost:8443"
enrollment_token: "$Token"
state_dir: "$AgentDir\state"
spool_dir: "$AgentDir\spool"
health_address: ":8081"
log_level: "info"
log_format: "json"
"@
$config | Set-Content -Path "$AgentDir\config.yaml" -Encoding UTF8
Write-Host "   Config: $AgentDir\config.yaml"

Write-Host "[4/4] Starting probe-agent..."
$env:AGENT_CONFIG_PATH = "$AgentDir\config.yaml"
& "$AgentDir\probe-agent.exe" run
