# ==========================================
# ZILLA-REC® // WSL2 & POWERSHELL INSTALLER
# ==========================================
[CmdletBinding()]
param()

Write-Host "`n[?] Initializing Zilla-Rec® Installation Suite..." -ForegroundColor Cyan

# 1. Check if WSL2 is installed and running
try {
    $wslStatus = wsl --status 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[x] Error: WSL2 is not installed or running properly on this machine." -ForegroundColor Red
        exit 1
    }
    Write-Host "[+] WSL2 environment detected." -ForegroundColor Green
} catch {
    Write-Host "[x] Error: WSL is unavailable on this system." -ForegroundColor Red
    exit 1
}

# 2. Define WSL target installation path
$WslUser = (wsl whoami).Trim()
$WslInstallDir = "/home/$WslUser/zillarec"

Write-Host "[*] Copying application files to WSL2 home (~/zillarec)..." -ForegroundColor Yellow

# Create directory in WSL
wsl mkdir -p $WslInstallDir
wsl mkdir -p "$WslInstallDir/downloads"

# Network share path bridge in Windows PowerShell
$WslNetworkPath = "\\wsl$\Ubuntu\home\$WslUser\zillarec"

if (Test-Path (Split-Path $WslNetworkPath)) {
    Copy-Item "bin/zillarec-linux" "$WslNetworkPath/zillarec" -Force
} else {
    Write-Host "[*] Using tar pipe transfer to WSL..."
    tar czf - bin/zillarec-linux | wsl tar xzf - -C $WslInstallDir
}

# 3. Set execution permissions inside WSL
wsl bash -c "chmod +x $WslInstallDir/zillarec"

Write-Host "`n==========================================" -ForegroundColor Green
Write-Host " INSTALLATION COMPLETE!" -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Green
Write-Host "To launch your Zilla-Rec® VCR Web Server inside WSL2, run:" -ForegroundColor Cyan
Write-Host "  wsl ~/zillarec/zillarec web" -ForegroundColor Yellow
Write-Host "Then open your browser to: http://localhost:8080" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Green
