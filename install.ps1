# SV (Switch Version) Windows Installer
# https://github.com/voocel/sv
#
# Usage: irm https://raw.githubusercontent.com/voocel/sv/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

# Enable TLS 1.2 for older PowerShell versions
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

# Constants
$SV_VERSION = if ($env:SV_VERSION) { $env:SV_VERSION } else { "latest" }
$SV_HOME = if ($env:SV_HOME) { $env:SV_HOME } else { "$env:USERPROFILE\.sv" }
$REPO_URL = "https://github.com/voocel/sv"
$API_URL = "https://api.github.com/repos/voocel/sv/releases/latest"

function Write-Banner {
    Write-Host @"
=================================================
              ___
             /  /\          ___
            /  /::\        /  /\
           /__/:/\:\      /  /:/
          _\_ \:\ \:\    /  /:/
         /__/\ \:\ \:\  /__/:/  ___
         \  \:\ \:\_\/  |  |:| /  /\
          \  \:\_\:\    |  |:|/  /:/
           \  \:\/:/    |__|:|__/:/
            \  \::/      \__\::::/
             \__\/           ````
       ___           _        _ _
      |_ _|_ __  ___| |_ __ _| | | ___ _ __
       | || '_ \/ __| __/ _` | | |/ _ \ '__|
       | || | | \__ \ || (_| | | |  __/ |
      |___|_| |_|___/\__\__,_|_|_|\___|_|
==================================================
"@ -ForegroundColor Cyan
}

function Write-Step {
    param([string]$Message)
    Write-Host $Message -ForegroundColor Yellow
}

function Write-Info {
    param([string]$Message)
    Write-Host $Message -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host $Message -ForegroundColor Green
}

function Write-Err {
    param([string]$Message)
    Write-Host "error: $Message" -ForegroundColor Red
}

function Get-LatestVersion {
    try {
        $response = Invoke-RestMethod -Uri $API_URL
        return $response.tag_name
    } catch {
        Write-Err "Failed to fetch latest version: $_"
        exit 1
    }
}

function Get-BinaryName {
    # Only support 64-bit Windows
    if (-not [System.Environment]::Is64BitOperatingSystem) {
        Write-Err "32-bit Windows is not supported"
        exit 1
    }
    return "sv-windows-amd64.exe"
}

function Install-SV {
    param([string]$Version)

    $binName = Get-BinaryName
    $downloadUrl = "$REPO_URL/releases/download/$Version/$binName"
    $binDir = "$SV_HOME\bin"
    $binPath = "$binDir\sv.exe"

    # Create directories
    $dirs = @("$SV_HOME", "$binDir", "$SV_HOME\cache", "$SV_HOME\downloads", "$SV_HOME\go")
    foreach ($dir in $dirs) {
        if (-not (Test-Path $dir)) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
        }
    }

    Write-Info "Downloading from: $downloadUrl"

    try {
        # Download with progress
        $ProgressPreference = 'SilentlyContinue'
        Invoke-WebRequest -Uri $downloadUrl -OutFile $binPath -UseBasicParsing
        $ProgressPreference = 'Continue'
    } catch {
        Write-Err "Failed to download sv: $_"
        exit 1
    }

    # Verify download
    if (-not (Test-Path $binPath)) {
        Write-Err "Download failed: binary not found"
        exit 1
    }

    Write-Success "Installed sv to $binPath"
}

function Set-Environment {
    $binDir = "$SV_HOME\bin"
    $goRoot = "$SV_HOME\go"

    # Get current user PATH (handle null)
    $userPath = [System.Environment]::GetEnvironmentVariable("PATH", "User")
    if (-not $userPath) { $userPath = "" }

    # Check if already in PATH
    if ($userPath -notlike "*$binDir*") {
        $newPath = if ($userPath) { "$binDir;$userPath" } else { $binDir }
        [System.Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        Write-Success "Added $binDir to user PATH"
    } else {
        Write-Info "$binDir already in PATH"
    }

    # Set GOROOT
    [System.Environment]::SetEnvironmentVariable("GOROOT", $goRoot, "User")
    Write-Success "Set GOROOT to $goRoot"

    # Set GOPATH if not set
    $goPath = [System.Environment]::GetEnvironmentVariable("GOPATH", "User")
    if (-not $goPath) {
        $goPath = "$env:USERPROFILE\go"
        [System.Environment]::SetEnvironmentVariable("GOPATH", $goPath, "User")
        Write-Success "Set GOPATH to $goPath"
    }

    # Create GOPATH directories
    $gopathDirs = @("$goPath\src", "$goPath\pkg", "$goPath\bin")
    foreach ($dir in $gopathDirs) {
        if (-not (Test-Path $dir)) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
        }
    }

    # Update current session
    $env:PATH = "$binDir;$goRoot\bin;$env:PATH"
    $env:GOROOT = $goRoot
    if (-not $env:GOPATH) { $env:GOPATH = $goPath }
}

function Test-Installation {
    $binPath = "$SV_HOME\bin\sv.exe"

    if (-not (Test-Path $binPath)) {
        Write-Err "Installation verification failed: binary not found"
        exit 1
    }

    try {
        $version = & $binPath --version 2>&1
        Write-Success "Verified: $version"
    } catch {
        Write-Err "Installation verification failed: $_"
        exit 1
    }
}

function Write-SuccessMessage {
    Write-Host ""
    Write-Success "sv has been successfully installed!"
    Write-Host ""
    Write-Info "To get started:"
    Write-Info "  1. Open a NEW PowerShell/Terminal window (required for PATH changes)"
    Write-Info "  2. Run: sv list"
    Write-Info "  3. Install Go: sv install --latest"
    Write-Host ""
    Write-Info "For more information:"
    Write-Info "  Documentation: $REPO_URL#readme"
    Write-Info "  Report issues: $REPO_URL/issues"
    Write-Host ""
}

# Main
function Main {
    Write-Banner

    # Check if already installed
    $existingBin = "$SV_HOME\bin\sv.exe"
    if ((Test-Path $existingBin) -and (-not $env:SV_FORCE)) {
        Write-Host "warn: sv is already installed at $SV_HOME" -ForegroundColor Yellow
        Write-Host "warn: Set `$env:SV_FORCE=1 to reinstall" -ForegroundColor Yellow
        return
    }

    Write-Step "[1/4] Fetching sv latest version"
    $version = if ($SV_VERSION -eq "latest") { Get-LatestVersion } else { $SV_VERSION }
    Write-Info "Version to install: $version"

    Write-Step "[2/4] Downloading sv binary"
    Install-SV -Version $version

    Write-Step "[3/4] Setting up environment"
    Set-Environment

    Write-Step "[4/4] Verifying installation"
    Test-Installation

    Write-SuccessMessage
}

Main
