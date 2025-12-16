# build-windows.ps1
#
# Build script for stable-diffusion.cpp on Windows with CUDA support.
#
# Prerequisites:
#   - Visual Studio 2022 with C++ desktop development workload
#   - CUDA Toolkit 11.8 or newer (https://developer.nvidia.com/cuda-downloads)
#   - CMake 3.18 or newer (https://cmake.org/download/)
#   - Git (for cloning source)
#
# Usage:
#   .\build-windows.ps1              # Clone source and build
#   .\build-windows.ps1 -SkipClone   # Build only (if source already exists)
#   .\build-windows.ps1 -Clean       # Clean build directory first
#
# Output:
#   ../../lib/stable-diffusion.dll

param(
    [switch]$SkipClone,
    [switch]$Clean,
    [switch]$Debug
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$SrcDir = Join-Path $ScriptDir "src"
$BuildDir = Join-Path $ScriptDir "build"
$LibDir = Join-Path (Split-Path -Parent (Split-Path -Parent $ScriptDir)) "lib"

Write-Host ""
Write-Host "=== stable-diffusion.cpp Windows Build ===" -ForegroundColor Cyan
Write-Host ""

# Check prerequisites
Write-Host "Checking prerequisites..." -ForegroundColor Yellow

# Check CMake
$cmake = Get-Command cmake -ErrorAction SilentlyContinue
if (-not $cmake) {
    Write-Host "ERROR: CMake not found. Please install CMake 3.18+" -ForegroundColor Red
    Write-Host "Download: https://cmake.org/download/"
    exit 1
}
$cmakeVersion = (cmake --version | Select-Object -First 1)
Write-Host "  CMake: $cmakeVersion" -ForegroundColor Green

# Check Visual Studio
$vswhere = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vswhere.exe"
if (Test-Path $vswhere) {
    $vsPath = & $vswhere -latest -property installationPath
    if ($vsPath) {
        Write-Host "  Visual Studio: Found at $vsPath" -ForegroundColor Green
    } else {
        Write-Host "WARNING: Visual Studio not found. Build may fail." -ForegroundColor Yellow
    }
} else {
    Write-Host "WARNING: Cannot check Visual Studio installation" -ForegroundColor Yellow
}

# Check CUDA
$nvcc = Get-Command nvcc -ErrorAction SilentlyContinue
if ($nvcc) {
    $cudaVersion = (nvcc --version | Select-String "release" | ForEach-Object { $_.Line })
    Write-Host "  CUDA: $cudaVersion" -ForegroundColor Green
} else {
    Write-Host "WARNING: CUDA nvcc not found in PATH" -ForegroundColor Yellow
    Write-Host "  Make sure CUDA Toolkit is installed and added to PATH"
}

# Clean if requested
if ($Clean -and (Test-Path $BuildDir)) {
    Write-Host ""
    Write-Host "Cleaning build directory..." -ForegroundColor Yellow
    Remove-Item -Recurse -Force $BuildDir
}

# Clone source if needed
if (-not $SkipClone) {
    if (-not (Test-Path $SrcDir)) {
        Write-Host ""
        Write-Host "Cloning stable-diffusion.cpp..." -ForegroundColor Yellow
        git clone --depth 1 https://github.com/leejet/stable-diffusion.cpp.git $SrcDir
        if ($LASTEXITCODE -ne 0) {
            Write-Host "ERROR: Failed to clone repository" -ForegroundColor Red
            exit 1
        }
    } else {
        Write-Host "Source directory already exists: $SrcDir" -ForegroundColor Green
    }
}

# Create build directory
if (-not (Test-Path $BuildDir)) {
    New-Item -ItemType Directory -Path $BuildDir | Out-Null
}

# Create lib directory
if (-not (Test-Path $LibDir)) {
    New-Item -ItemType Directory -Path $LibDir | Out-Null
}

# Configure CMake
Write-Host ""
Write-Host "Configuring CMake..." -ForegroundColor Yellow

$buildType = if ($Debug) { "Debug" } else { "Release" }

Push-Location $BuildDir
try {
    # Configure with Visual Studio generator
    cmake .. `
        -G "Visual Studio 17 2022" `
        -A x64 `
        -DGGML_CUDA=ON `
        -DCMAKE_BUILD_TYPE=$buildType `
        -DCMAKE_CUDA_ARCHITECTURES="75;86;89" `
        -DBUILD_SHARED_LIBS=ON `
        -DSD_BUILD_EXAMPLES=OFF

    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: CMake configuration failed" -ForegroundColor Red
        exit 1
    }

    # Build
    Write-Host ""
    Write-Host "Building..." -ForegroundColor Yellow

    cmake --build . --config $buildType --parallel

    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: Build failed" -ForegroundColor Red
        exit 1
    }
} finally {
    Pop-Location
}

# Verify output
Write-Host ""
Write-Host "Verifying build output..." -ForegroundColor Yellow

$dllPath = Join-Path $LibDir "stable-diffusion.dll"
if (Test-Path $dllPath) {
    $dllInfo = Get-Item $dllPath
    Write-Host "SUCCESS: Built $($dllInfo.Name) ($([math]::Round($dllInfo.Length / 1MB, 2)) MB)" -ForegroundColor Green
} else {
    Write-Host "WARNING: stable-diffusion.dll not found at expected location" -ForegroundColor Yellow
    Write-Host "  Expected: $dllPath"
    Write-Host "  Check build output for library location"
}

Write-Host ""
Write-Host "=== Build Complete ===" -ForegroundColor Cyan
Write-Host ""
Write-Host "Next steps:"
Write-Host "  1. Ensure stable-diffusion.dll is in: $LibDir"
Write-Host "  2. Download SD v1.5 model to: models/sd-v1-5.safetensors"
Write-Host "  3. Build Go application with: go build -tags sd"
Write-Host ""
