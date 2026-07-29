param(
    [string]$Token = $env:AGENT_ENROLLMENT_TOKEN,
    [string]$ControlPlane = $(if ($env:AGENT_CONTROL_PLANE) { $env:AGENT_CONTROL_PLANE } else { "http://localhost:5000" }),
    [string]$AgentDir = $(if ($env:AGENT_DIR) { $env:AGENT_DIR } else { "C:\probe-agent" }),
    [switch]$InstallAsService
)

if (-not $Token) {
    Write-Host "Usage:"
    Write-Host "  AGENT_ENROLLMENT_TOKEN=xxx .\install-agent.ps1"
    Write-Host "  AGENT_ENROLLMENT_TOKEN=xxx .\install-agent.ps1 -InstallAsService"
    exit 1
}

Write-Host "==================================="
Write-Host " probe-agent installer"
Write-Host "==================================="
Write-Host " Control plane : $ControlPlane"
Write-Host " Install dir   : $AgentDir"
Write-Host " Install as service: $InstallAsService"
Write-Host "==================================="
Write-Host ""

$needGo = -not (Get-Command go -ErrorAction SilentlyContinue)
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]"Administrator")

if ($needGo) {
    Write-Host "[1/5] Installing Go (user-local)..."
    $goArch = "windows-amd64"
    if (-not [Environment]::Is64BitOperatingSystem) { $goArch = "windows-386" }
    Write-Host "   Fetching latest Go version..."
    try {
        $goLatest = (Invoke-RestMethod -Uri "https://go.dev/VERSION?m=text" -TimeoutSec 10).Trim()
    } catch { $goLatest = "go1.22.10" }
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

Write-Host "[2/5] Creating directories..."
New-Item -ItemType Directory -Force -Path "$AgentDir\state", "$AgentDir\spool" | Out-Null

Write-Host "[3/5] Building probe-agent..."
$exePath = "$AgentDir\probe-agent.exe"
$tmp = Join-Path $env:TEMP "probe-agent-build-$([Guid]::NewGuid())"
git clone --depth 1 https://github.com/mhghr/dog.git $tmp
Push-Location $tmp
go build -ldflags="-s -w" -o $exePath .\cmd\probe-agent
Pop-Location
Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
Write-Host "   Binary: $exePath"

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

if ($InstallAsService -and $isAdmin) {
    # Register as Windows Service
    $serviceName = "ProbeAgent"
    $existing = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
    if ($existing) {
        Stop-Service $serviceName -ErrorAction SilentlyContinue
        sc.exe delete $serviceName | Out-Null
        Start-Sleep -Seconds 2
    }
    New-Service -Name $serviceName `
        -BinaryPathName "`"$exePath`" run" `
        -DisplayName "Probe Agent" `
        -Description "Monitoring probe agent that connects to the gateway and executes monitoring checks" `
        -StartupType Automatic `
        -ErrorAction Stop
    Start-Service $serviceName
    Write-Host "   Service '$serviceName' created and started."
    Write-Host "   It will auto-start on boot and restart on failure."
    Write-Host "   Manage with: Get-Service ProbeAgent | Start-Service / Stop-Service"
} elseif ($InstallAsService -and -not $isAdmin) {
    Write-Host "   ERROR: -InstallAsService requires Administrator privileges."
    Write-Host "   Run PowerShell as Administrator and try again."
    Write-Host ""
    Write-Host "   Falling back to scheduled task..."

    $taskName = "ProbeAgent"
    $action = New-ScheduledTaskAction -Execute $exePath -Argument "run" -WorkingDirectory $AgentDir
    $trigger = New-ScheduledTaskTrigger -AtStartup
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -RestartCount 5 -RestartInterval (New-TimeSpan -Minutes 1)
    $principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType Interactive -RunLevel Limited
    Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null
    Start-ScheduledTask -TaskName $taskName
    Write-Host "   Scheduled task '$taskName' created and started."
    Write-Host "   It will auto-start on logon and restart on failure."
} else {
    # Run in background using Start-Process with hidden window
    $env:AGENT_CONFIG_PATH = "$AgentDir\config.yaml"
    Start-Process -FilePath $exePath -ArgumentList "run" -WorkingDirectory $AgentDir -WindowStyle Hidden -NoNewWindow
    Write-Host "   Agent started in background (PID: $((Get-Process -Name probe-agent -ErrorAction SilentlyContinue).Id))."
    Write-Host ""
    Write-Host "   NOTE: This runs in the current user session."
    Write-Host "   It will stop when you log off."
    Write-Host "   To install as Windows Service, re-run with:"
    Write-Host "     .\install-agent.ps1 -Token $Token -InstallAsService"
    Write-Host "   (requires Administrator)"
}
