param(
    [string]$Token = $env:AGENT_ENROLLMENT_TOKEN,
    [string]$ControlPlane = $(if ($env:AGENT_CONTROL_PLANE) { $env:AGENT_CONTROL_PLANE } else { "http://localhost:5000" }),
    [string]$AgentDir = $(if ($env:AGENT_DIR) { $env:AGENT_DIR } else { "C:\probe-agent" })
)

if (-not $Token) {
    Write-Host "Usage:"
    Write-Host "  irm https://raw.githubusercontent.com/mhghr/dog/main/scripts/install-agent.ps1 | iex"
    Write-Host "  OR"
    Write-Host "  AGENT_ENROLLMENT_TOKEN=xxx .\install-agent.ps1"
    exit 1
}

Write-Host "==================================="
Write-Host " probe-agent installer"
Write-Host "==================================="
Write-Host " Control plane : $ControlPlane"
Write-Host " Install dir   : $AgentDir"
Write-Host "==================================="
Write-Host ""

$needGo = -not (Get-Command go -ErrorAction SilentlyContinue)
$needGit = -not (Get-Command git -ErrorAction SilentlyContinue)
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]"Administrator")

if ($needGo) {
    Write-Host "[1/5] Installing Go (user-local)..."

    $goArch = "windows-amd64"
    if (-not [Environment]::Is64BitOperatingSystem) { $goArch = "windows-386" }

    Write-Host "   Fetching latest Go version..."
    try {
        $goLatest = (Invoke-RestMethod -Uri "https://go.dev/VERSION?m=text" -TimeoutSec 10).Trim()
    } catch {
        $goLatest = "go1.22.10"
    }
    if (-not $goLatest) { $goLatest = "go1.22.10" }

    $goZip = "${goLatest}.${goArch}.zip"
    $goUrl = "https://go.dev/dl/${goZip}"
    $goTemp = Join-Path $env:TEMP $goZip
    $goInstallDir = "$env:LOCALAPPDATA\go"

    Write-Host "   Downloading ${goUrl}..."
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    (New-Object Net.WebClient).DownloadFile($goUrl, $goTemp)

    Write-Host "   Extracting..."
    if (Test-Path $goInstallDir) { Remove-Item -Recurse -Force $goInstallDir }
    Expand-Archive -Path $goTemp -DestinationPath $goInstallDir -Force
    Remove-Item $goTemp

    $goBin = Get-ChildItem -Path $goInstallDir -Filter "go" -Directory | Select-Object -First 1
    if (-not $goBin) { $goBin = Get-ChildItem -Path $goInstallDir -Filter "go*" -Directory | Select-Object -First 1 }
    if ($goBin) {
        $env:Path = "$($goBin.FullName)\bin;$env:Path"
        $env:GOROOT = $goBin.FullName
        [Environment]::SetEnvironmentVariable("Path", "$($goBin.FullName)\bin;$([Environment]::GetEnvironmentVariable('Path','User'))", "User")
    }
    Write-Host "   Go installed"
} else {
    Write-Host "[1/5] Go: $(go version)"
}

if ($needGit) {
    Write-Host "   Installing git..."
    $gitUrl = "https://github.com/git-for-windows/git/releases/download/v2.47.1.windows.1/MinGit-2.47.1-64-bit.zip"
    $gitTemp = Join-Path $env:TEMP "git-portable.zip"
    $gitDir = "$env:LOCALAPPDATA\git"
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    (New-Object Net.WebClient).DownloadFile($gitUrl, $gitTemp)
    if (Test-Path $gitDir) { Remove-Item -Recurse -Force $gitDir }
    Expand-Archive -Path $gitTemp -DestinationPath $gitDir -Force
    Remove-Item $gitTemp
    $env:Path = "$gitDir\cmd;$gitDir\bin;$env:Path"
    Write-Host "   git installed"
}

Write-Host "[2/5] Creating directories..."
New-Item -ItemType Directory -Force -Path "$AgentDir\state", "$AgentDir\spool" | Out-Null

Write-Host "[3/5] Building probe-agent..."
$tmp = Join-Path $env:TEMP "probe-agent-build-$([Guid]::NewGuid())"
git clone --depth 1 https://github.com/mhghr/dog.git $tmp
Push-Location $tmp
go build -ldflags="-s -w" -o "$AgentDir\probe-agent.exe" .\cmd\probe-agent
Pop-Location
Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
Write-Host "   Binary: $AgentDir\probe-agent.exe"

Write-Host "[4/5] Writing config..."
@"
control_plane: "$ControlPlane"
agent_gateway: "localhost:8443"
enrollment_token: "$Token"
state_dir: "$AgentDir\state"
spool_dir: "$AgentDir\spool"
health_address: ":8081"
log_level: "info"
log_format: "json"
"@ | Set-Content -Path "$AgentDir\config.yaml" -Encoding UTF8
Write-Host "   Config: $AgentDir\config.yaml"

Write-Host "[5/5] Starting probe-agent..."
Write-Host "==================================="
$env:AGENT_CONFIG_PATH = "$AgentDir\config.yaml"
& "$AgentDir\probe-agent.exe" run
