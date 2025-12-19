# build-sd-windows.ps1
#
# Cross-platform build script for CanvusLocalLLM with Stable Diffusion support on Windows.
#
# This script orchestrates:
#   1. Building stable-diffusion.cpp C library with CUDA
#   2. Building Go application with CGo bindings enabled
#   3. Bundling DLLs with the binary
#   4. Creating a distributable ZIP archive
#
# Prerequisites:
#   - Visual Studio 2022 with C++ desktop development workload
#   - CUDA Toolkit 11.8 or newer (https://developer.nvidia.com/cuda-downloads)
#   - CMake 3.18 or newer (https://cmake.org/download/)
#   - Go 1.21 or newer (https://go.dev/dl/)
#   - Git (for cloning stable-diffusion.cpp if needed)
#   - MinGW-w64 GCC (for CGo compilation)
#
# Usage:
#   .\build-sd-windows.ps1                  # Build with defaults
#   .\build-sd-windows.ps1 -Version 1.2.0
#   .\build-sd-windows.ps1 -Clean           # Clean build from scratch
#   .\build-sd-windows.ps1 -SkipSD          # Skip SD library build (use existing)
#   .\build-sd-windows.ps1 -Zip             # Create distribution ZIP
#
# Output:
#   - Binary: bin\canvuslocallm-sd-windows-amd64.exe
#   - DLLs: lib\stable-diffusion.dll, lib\cudart64_*.dll, etc.
#   - ZIP: dist\canvuslocallm-sd-VERSION-windows-amd64.zip (if -Zip)
#

param(
    [string]$Version = "1.0.0",
    [switch]$Clean,
    [switch]$SkipSD,
    [switch]$Zip,
    [switch]$Verbose,
    [switch]$Help
)

$ErrorActionPreference = "Stop"

# Constants
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
$SDDir = Join-Path $ProjectRoot "deps\stable-diffusion.cpp"
$LibDir = Join-Path $ProjectRoot "lib"
$BinDir = Join-Path $ProjectRoot "bin"
$DistDir = Join-Path $ProjectRoot "dist"
$BinaryName = "canvuslocallm-sd-windows-amd64.exe"

# Functions
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Err {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

function Write-Section {
    param([string]$Message)
    Write-Host ""
    Write-Host "=== $Message ===" -ForegroundColor Cyan
    Write-Host ""
}

function Show-Help {
    Get-Help $MyInvocation.MyCommand.Path -Detailed
    exit 0
}

function Test-Prerequisites {
    Write-Section "Checking Prerequisites"

    $missing = @()

    # Check Go
    $go = Get-Command go -ErrorAction SilentlyContinue
    if (-not $go) {
        $missing += "go (1.21+)"
    } else {
        $goVersion = (go version) -replace "go version go", "" | ForEach-Object { $_.Split()[0] }
        Write-Info "Go: $goVersion"
    }

    # Check CMake
    $cmake = Get-Command cmake -ErrorAction SilentlyContinue
    if (-not $cmake) {
        $missing += "cmake (3.18+)"
    } else {
        $cmakeVersion = (cmake --version | Select-Object -First 1) -replace "cmake version ", ""
        Write-Info "CMake: $cmakeVersion"
    }

    # Check Visual Studio
    $vswhere = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vswhere.exe"
    if (Test-Path $vswhere) {
        $vsPath = & $vswhere -latest -property installationPath
        if ($vsPath) {
            Write-Info "Visual Studio: Found at $vsPath"
        } else {
            Write-Warn "Visual Studio not found - build may fail"
        }
    } else {
        Write-Warn "Cannot check Visual Studio installation"
    }

    # Check CUDA
    $nvcc = Get-Command nvcc -ErrorAction SilentlyContinue
    if ($nvcc) {
        $cudaVersion = (nvcc --version | Select-String "release") -replace ".*release ", "" | ForEach-Object { $_.Split(',')[0] }
        Write-Info "CUDA: $cudaVersion"
    } else {
        Write-Warn "CUDA nvcc not found in PATH - SD will build without GPU acceleration"
    }

    # Check GCC (for CGo)
    $gcc = Get-Command gcc -ErrorAction SilentlyContinue
    if (-not $gcc) {
        Write-Warn "GCC not found - CGo requires MinGW-w64"
        Write-Warn "Download from: https://www.mingw-w64.org/"
        $missing += "gcc (MinGW-w64)"
    } else {
        $gccVersion = (gcc --version | Select-Object -First 1)
        Write-Info "GCC: $gccVersion"
    }

    # Check Git
    $git = Get-Command git -ErrorAction SilentlyContinue
    if (-not $git) {
        $missing += "git"
    }

    if ($missing.Count -gt 0) {
        Write-Err "Missing required tools: $($missing -join ', ')"
        Write-Err "Please install them and try again."
        exit 1
    }

    Write-Success "All prerequisites found"
}

function Clear-BuildArtifacts {
    if (-not $Clean) {
        return
    }

    Write-Section "Cleaning Build Artifacts"

    # Clean SD build
    $sdBuildDir = Join-Path $SDDir "build"
    if (Test-Path $sdBuildDir) {
        Write-Info "Removing $sdBuildDir"
        Remove-Item -Recurse -Force $sdBuildDir
    }

    # Clean libraries
    $sdDll = Join-Path $LibDir "stable-diffusion.dll"
    if (Test-Path $sdDll) {
        Write-Info "Removing $sdDll"
        Remove-Item -Force $sdDll
    }

    # Clean CUDA DLLs
    Get-ChildItem -Path $LibDir -Filter "cuda*.dll" -ErrorAction SilentlyContinue | ForEach-Object {
        Write-Info "Removing $($_.Name)"
        Remove-Item -Force $_.FullName
    }

    # Clean Go binary
    $binaryPath = Join-Path $BinDir $BinaryName
    if (Test-Path $binaryPath) {
        Write-Info "Removing $binaryPath"
        Remove-Item -Force $binaryPath
    }

    Write-Success "Clean complete"
}

function Build-SDLibrary {
    if ($SkipSD) {
        Write-Section "Skipping SD Library Build"

        # Verify DLL exists
        $dllPath = Join-Path $LibDir "stable-diffusion.dll"
        if (-not (Test-Path $dllPath)) {
            Write-Err "stable-diffusion.dll not found at $dllPath"
            Write-Err "Cannot skip SD build when DLL doesn't exist"
            exit 1
        }

        Write-Info "Using existing DLL at $dllPath"
        return
    }

    Write-Section "Building stable-diffusion.cpp Library"

    # Check if build script exists
    $buildScript = Join-Path $SDDir "build-windows.ps1"
    if (-not (Test-Path $buildScript)) {
        Write-Err "Build script not found: $buildScript"
        exit 1
    }

    # Run SD build script
    Push-Location $SDDir
    try {
        $buildArgs = @()
        if ($Clean) {
            $buildArgs += "-Clean"
        }

        & $buildScript @buildArgs

        if ($LASTEXITCODE -ne 0) {
            Write-Err "Failed to build stable-diffusion.cpp library"
            exit 1
        }

        Write-Success "stable-diffusion.cpp library built successfully"
    } finally {
        Pop-Location
    }

    # Verify DLL was created
    $dllPath = Join-Path $LibDir "stable-diffusion.dll"
    if (-not (Test-Path $dllPath)) {
        Write-Err "DLL not found after build: $dllPath"
        exit 1
    }

    $dllInfo = Get-Item $dllPath
    Write-Info "DLL size: $([math]::Round($dllInfo.Length / 1MB, 2)) MB"
}

function Copy-CUDADLLs {
    Write-Info "Checking for CUDA runtime DLLs..."

    # Common CUDA DLL locations
    $cudaPaths = @(
        "${env:CUDA_PATH}\bin",
        "C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v11.8\bin",
        "C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v12.0\bin"
    )

    $requiredDlls = @(
        "cudart64_*.dll",
        "cublas64_*.dll",
        "cublasLt64_*.dll"
    )

    $foundDlls = @{}

    foreach ($cudaPath in $cudaPaths) {
        if (-not (Test-Path $cudaPath)) {
            continue
        }

        foreach ($dllPattern in $requiredDlls) {
            $dlls = Get-ChildItem -Path $cudaPath -Filter $dllPattern -ErrorAction SilentlyContinue
            foreach ($dll in $dlls) {
                $dllName = $dll.Name
                if (-not $foundDlls.ContainsKey($dllName)) {
                    $foundDlls[$dllName] = $dll.FullName
                }
            }
        }
    }

    if ($foundDlls.Count -eq 0) {
        Write-Warn "No CUDA runtime DLLs found - application may fail without them"
        Write-Warn "Expected location: ${env:CUDA_PATH}\bin"
        return
    }

    # Copy DLLs to lib directory
    foreach ($dll in $foundDlls.GetEnumerator()) {
        $destPath = Join-Path $LibDir $dll.Key
        if (-not (Test-Path $destPath)) {
            Write-Info "Copying CUDA DLL: $($dll.Key)"
            Copy-Item -Path $dll.Value -Destination $destPath
        }
    }

    Write-Success "CUDA DLLs copied to lib directory"
}

function Build-GoApplication {
    Write-Section "Building Go Application with SD Support"

    Push-Location $ProjectRoot
    try {
        # Ensure bin directory exists
        if (-not (Test-Path $BinDir)) {
            New-Item -ItemType Directory -Path $BinDir | Out-Null
        }

        Write-Info "Building with CGO_ENABLED=1 and -tags sd"

        # Set up CGo environment
        $env:CGO_ENABLED = "1"
        $env:GOOS = "windows"
        $env:GOARCH = "amd64"

        # Build flags
        $buildTags = "sd"
        $ldflags = "-s -w -X main.Version=$Version"

        # CGo flags - use absolute paths for Windows
        $libDirAbs = (Resolve-Path $LibDir).Path
        $env:CGO_LDFLAGS = "-L`"$libDirAbs`" -lstable-diffusion"

        Write-Info "Build command: go build -tags $buildTags -ldflags=`"$ldflags`" -o `"$BinDir\$BinaryName`""

        go build -tags $buildTags -ldflags="$ldflags" -o "$BinDir\$BinaryName" .

        if ($LASTEXITCODE -ne 0) {
            Write-Err "Failed to build Go application"
            exit 1
        }

        Write-Success "Go application built successfully"
    } finally {
        Pop-Location
    }

    # Verify binary was created
    $binaryPath = Join-Path $BinDir $BinaryName
    if (-not (Test-Path $binaryPath)) {
        Write-Err "Binary not found after build: $binaryPath"
        exit 1
    }

    $binInfo = Get-Item $binaryPath
    Write-Info "Binary size: $([math]::Round($binInfo.Length / 1MB, 2)) MB"
}

function New-ZipDistribution {
    if (-not $Zip) {
        return
    }

    Write-Section "Creating Distribution ZIP"

    if (-not (Test-Path $DistDir)) {
        New-Item -ItemType Directory -Path $DistDir | Out-Null
    }

    $zipName = "canvuslocallm-sd-$Version-windows-amd64.zip"
    $zipPath = Join-Path $DistDir $zipName
    $stagingDir = Join-Path $DistDir "staging"
    $appDir = Join-Path $stagingDir "canvuslocallm-sd"

    # Clean staging directory
    if (Test-Path $stagingDir) {
        Remove-Item -Recurse -Force $stagingDir
    }
    New-Item -ItemType Directory -Path $appDir | Out-Null

    Write-Info "Preparing distribution files..."

    # Copy binary
    $binPath = Join-Path $BinDir $BinaryName
    Copy-Item -Path $binPath -Destination (Join-Path $appDir "canvuslocallm.exe")

    # Copy DLLs
    $libDirDest = Join-Path $appDir "lib"
    New-Item -ItemType Directory -Path $libDirDest | Out-Null

    Copy-Item -Path (Join-Path $LibDir "stable-diffusion.dll") -Destination $libDirDest

    # Copy CUDA DLLs if present
    Get-ChildItem -Path $LibDir -Filter "cuda*.dll" -ErrorAction SilentlyContinue | ForEach-Object {
        Copy-Item -Path $_.FullName -Destination $libDirDest
    }

    # Copy documentation
    $docs = @("README.md", "LICENSE.txt", "example.env")
    foreach ($doc in $docs) {
        $docPath = Join-Path $ProjectRoot $doc
        if (Test-Path $docPath) {
            Copy-Item -Path $docPath -Destination $appDir
        }
    }

    # Create models directory
    $modelsDir = Join-Path $appDir "models"
    New-Item -ItemType Directory -Path $modelsDir | Out-Null
    Set-Content -Path (Join-Path $modelsDir "README.txt") -Value "Place your SD model (sd-v1-5.safetensors) here"

    # Create startup batch file
    $startBat = @"
@echo off
REM CanvusLocalLLM with Stable Diffusion Support
REM This script sets up the library path and starts the application

set SCRIPT_DIR=%~dp0
set PATH=%SCRIPT_DIR%lib;%PATH%

"%SCRIPT_DIR%canvuslocallm.exe" %*
"@
    Set-Content -Path (Join-Path $appDir "start.bat") -Value $startBat

    # Create installation instructions
    $installTxt = @"
CanvusLocalLLM with Stable Diffusion Support - Windows Installation
====================================================================

Prerequisites:
--------------
1. CUDA Toolkit 11.8+ (for GPU acceleration)
   Download: https://developer.nvidia.com/cuda-downloads
2. NVIDIA GPU with CUDA support
3. Windows 10/11 with NVIDIA driver installed

Installation:
-------------
1. Extract this ZIP to your desired location
2. Download SD v1.5 model to models\ directory:
   Download from: https://huggingface.co/runwayml/stable-diffusion-v1-5/resolve/main/v1-5-pruned.safetensors
   Save as: models\sd-v1-5.safetensors

3. Copy example.env to .env and configure:
   copy example.env .env
   notepad .env

4. Set SD configuration in .env:
   SD_MODEL_PATH=models\sd-v1-5.safetensors
   SD_IMAGE_SIZE=512
   SD_INFERENCE_STEPS=20

Running:
--------
Double-click start.bat

Or manually:
set PATH=%CD%\lib;%PATH%
canvuslocallm.exe

Troubleshooting:
----------------
If you see "stable-diffusion.dll not found":
  - Ensure lib\stable-diffusion.dll exists
  - Add lib directory to PATH or run from start.bat

If you see CUDA errors:
  - Verify CUDA is installed: nvcc --version
  - Verify NVIDIA driver: nvidia-smi
  - Check GPU compatibility with CUDA 11.8+

If you see "VCRUNTIME140.dll not found":
  - Install Visual C++ Redistributable:
    https://aka.ms/vs/17/release/vc_redist.x64.exe

For more information, see README.md
"@
    Set-Content -Path (Join-Path $appDir "INSTALL.txt") -Value $installTxt

    # Create ZIP archive
    Write-Info "Creating ZIP: $zipName"

    # Remove existing ZIP if present
    if (Test-Path $zipPath) {
        Remove-Item -Force $zipPath
    }

    # Use Compress-Archive (PowerShell 5.0+)
    Compress-Archive -Path $appDir -DestinationPath $zipPath -CompressionLevel Optimal

    # Clean staging
    Remove-Item -Recurse -Force $stagingDir

    if (Test-Path $zipPath) {
        $zipInfo = Get-Item $zipPath
        Write-Success "ZIP created: $zipName ($([math]::Round($zipInfo.Length / 1MB, 2)) MB)"
        Write-Info "Location: $zipPath"
    } else {
        Write-Err "Failed to create ZIP"
        exit 1
    }
}

function Test-Build {
    Write-Section "Verifying Build"

    $errors = 0

    # Check DLL
    $dllPath = Join-Path $LibDir "stable-diffusion.dll"
    if (Test-Path $dllPath) {
        Write-Success "DLL present: stable-diffusion.dll"
    } else {
        Write-Err "DLL missing: stable-diffusion.dll"
        $errors++
    }

    # Check binary
    $binPath = Join-Path $BinDir $BinaryName
    if (Test-Path $binPath) {
        Write-Success "Binary present: $BinaryName"
    } else {
        Write-Err "Binary missing: $BinaryName"
        $errors++
    }

    # Check ZIP if created
    if ($Zip) {
        $zipName = "canvuslocallm-sd-$Version-windows-amd64.zip"
        $zipPath = Join-Path $DistDir $zipName
        if (Test-Path $zipPath) {
            Write-Success "ZIP present: $zipName"
        } else {
            Write-Err "ZIP missing: $zipName"
            $errors++
        }
    }

    if ($errors -gt 0) {
        Write-Err "Build verification failed with $errors error(s)"
        return $false
    }

    Write-Success "Build verification passed"
    return $true
}

function Show-Summary {
    Write-Section "Build Summary"

    Write-Host "Version:  $Version"
    Write-Host "Platform: Windows amd64"
    Write-Host ""
    Write-Host "Artifacts:"
    Write-Host "  Binary:  $BinDir\$BinaryName"
    Write-Host "  DLL:     $LibDir\stable-diffusion.dll"

    if ($Zip) {
        Write-Host "  ZIP:     $DistDir\canvuslocallm-sd-$Version-windows-amd64.zip"
    }

    Write-Host ""
    Write-Host "Next steps:"
    Write-Host "  1. Download SD model and save to models\sd-v1-5.safetensors"
    Write-Host "  2. Configure .env: copy example.env .env"
    Write-Host "  3. Run: set PATH=$LibDir;%PATH% && $BinDir\$BinaryName"
    Write-Host ""
}

# Main execution
function Main {
    if ($Help) {
        Show-Help
    }

    if ($Verbose) {
        $VerbosePreference = "Continue"
    }

    Write-Host ""
    Write-Host "========================================================" -ForegroundColor Cyan
    Write-Host "  CanvusLocalLLM with Stable Diffusion - Windows Build" -ForegroundColor Cyan
    Write-Host "  Version: $Version" -ForegroundColor Cyan
    Write-Host "========================================================" -ForegroundColor Cyan
    Write-Host ""

    Test-Prerequisites
    Clear-BuildArtifacts
    Build-SDLibrary
    Copy-CUDADLLs
    Build-GoApplication
    New-ZipDistribution

    if (Test-Build) {
        Show-Summary
        Write-Success "Build complete!"
    } else {
        Write-Err "Build failed verification"
        exit 1
    }
}

Main
