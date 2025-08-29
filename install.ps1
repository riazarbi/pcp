# PCP Installation Script for Windows PowerShell
# Usage: iwr https://github.com/riazarbi/pcp/releases/latest/download/install.ps1 | iex

param(
    [string]$InstallDir = "",
    [switch]$Force
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$REPO = "riazarbi/pcp"
$BINARY_NAME = "pcp"

# Colors for output
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

function Get-Platform {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "windows-amd64" }
        "ARM64" { return "windows-arm64" }
        default {
            Write-Error "Unsupported architecture: $arch"
            Write-Error "Supported architectures: AMD64, ARM64"
            exit 1
        }
    }
}

function Get-LatestVersion {
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO/releases/latest"
        return $response.tag_name
    }
    catch {
        Write-Error "Failed to get latest version from GitHub API: $($_.Exception.Message)"
        exit 1
    }
}

function Get-InstallDirectory {
    if ($InstallDir -ne "") {
        if (!(Test-Path $InstallDir)) {
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
            Write-Info "Created directory: $InstallDir"
        }
        return $InstallDir
    }

    # Try common install locations
    $possibleDirs = @(
        "$env:LOCALAPPDATA\Programs\pcp",
        "$env:ProgramFiles\pcp",
        "$env:USERPROFILE\.local\bin",
        "$env:USERPROFILE\bin"
    )

    foreach ($dir in $possibleDirs) {
        try {
            if (!(Test-Path $dir)) {
                New-Item -ItemType Directory -Path $dir -Force | Out-Null
                Write-Info "Created directory: $dir"
            }
            
            # Test write permission
            $testFile = Join-Path $dir "test_write.tmp"
            Set-Content -Path $testFile -Value "test" -ErrorAction Stop
            Remove-Item $testFile -ErrorAction SilentlyContinue
            
            return $dir
        }
        catch {
            continue
        }
    }

    Write-Error "Could not find a writable directory for installation"
    Write-Error "Please specify an install directory with -InstallDir parameter"
    exit 1
}

function Download-Binary {
    param(
        [string]$Version,
        [string]$Platform
    )

    $binaryName = "$BINARY_NAME-$Platform.exe"
    $downloadUrl = "https://github.com/$REPO/releases/download/$Version/$binaryName"
    $checksumsUrl = "https://github.com/$REPO/releases/download/$Version/checksums.txt"

    Write-Info "Downloading $binaryName $Version..."

    # Create temporary directory
    $tmpDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
    $binaryPath = Join-Path $tmpDir $binaryName
    $checksumsPath = Join-Path $tmpDir "checksums.txt"

    try {
        # Download binary
        Invoke-WebRequest -Uri $downloadUrl -OutFile $binaryPath -ErrorAction Stop
        
        # Download checksums for verification
        Write-Info "Downloading checksums for verification..."
        try {
            Invoke-WebRequest -Uri $checksumsUrl -OutFile $checksumsPath -ErrorAction Stop
            
            # Verify checksum
            Write-Info "Verifying checksum..."
            $expectedChecksum = (Get-Content $checksumsPath | Select-String $binaryName).ToString().Split()[0]
            $actualChecksum = (Get-FileHash -Path $binaryPath -Algorithm SHA256).Hash.ToLower()
            
            if ($expectedChecksum -eq $actualChecksum) {
                Write-Success "Checksum verification passed"
            } else {
                Write-Error "Checksum verification failed!"
                Write-Error "Expected: $expectedChecksum"
                Write-Error "Actual:   $actualChecksum"
                Remove-Item $tmpDir -Recurse -Force
                exit 1
            }
        }
        catch {
            Write-Warning "Failed to download or verify checksums, skipping verification"
        }
        
        return $binaryPath
    }
    catch {
        Write-Error "Failed to download binary: $($_.Exception.Message)"
        Remove-Item $tmpDir -Recurse -Force
        exit 1
    }
}

function Install-Binary {
    param(
        [string]$BinaryPath,
        [string]$InstallDir
    )

    $installPath = Join-Path $InstallDir "$BINARY_NAME.exe"

    # Check if binary already exists
    if (Test-Path $installPath -and !$Force) {
        $response = Read-Host "PCP is already installed at $installPath. Overwrite? [y/N]"
        if ($response -ne "y" -and $response -ne "Y") {
            Write-Info "Installation cancelled"
            return $installPath
        }
    }

    # Copy binary
    Write-Info "Installing to: $installPath"
    Copy-Item $BinaryPath $installPath -Force

    # Check if install directory is in PATH
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($currentPath -notlike "*$InstallDir*") {
        Write-Warning "Install directory $InstallDir is not in your PATH"
        
        $response = Read-Host "Add $InstallDir to your PATH? [y/N]"
        if ($response -eq "y" -or $response -eq "Y") {
            $newPath = $currentPath + ";" + $InstallDir
            [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
            Write-Success "Added $InstallDir to PATH"
            Write-Info "Please restart your PowerShell session for PATH changes to take effect"
        } else {
            Write-Warning "You will need to use the full path to run pcp: $installPath"
        }
    }

    return $installPath
}

function Test-Installation {
    param([string]$InstallPath)

    Write-Info "Testing installation..."
    try {
        $output = & $InstallPath -h 2>&1
        if ($LASTEXITCODE -eq 0) {
            return $true
        }
    }
    catch {
        # Ignore errors
    }
    return $false
}

function Main {
    Write-Info "Installing PCP (Prompt Composition Processor) for Windows..."

    # Check PowerShell version
    if ($PSVersionTable.PSVersion.Major -lt 3) {
        Write-Error "PowerShell 3.0 or later is required"
        exit 1
    }

    # Detect platform
    $platform = Get-Platform
    Write-Info "Detected platform: $platform"

    # Get latest version
    $version = Get-LatestVersion
    Write-Info "Latest version: $version"

    # Get install directory
    $installDir = Get-InstallDirectory
    Write-Info "Install directory: $installDir"

    # Download binary
    $binaryPath = Download-Binary -Version $version -Platform $platform

    # Install binary
    $installPath = Install-Binary -BinaryPath $binaryPath -InstallDir $installDir

    # Clean up
    Remove-Item (Split-Path $binaryPath) -Recurse -Force

    # Test installation
    if (Test-Installation -InstallPath $installPath) {
        Write-Success "PCP $version installed successfully!"
        Write-Info "Location: $installPath"
        Write-Info "Run 'pcp -h' to get started"
        
        # Show PATH info if not in PATH
        $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
        if ($currentPath -notlike "*$installDir*") {
            Write-Info ""
            Write-Info "To run pcp from anywhere, either:"
            Write-Info "1. Restart your PowerShell session (if you added to PATH)"
            Write-Info "2. Use the full path: $installPath"
            Write-Info "3. Add $installDir to your PATH manually"
        }
    } else {
        Write-Error "Installation test failed"
        exit 1
    }
}

# Run main function
Main