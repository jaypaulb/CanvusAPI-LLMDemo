# build-llama-windows.ps1
# Molecule: Composes git clone + cmake configuration + build execution
# Purpose: Build llama.cpp with CUDA support for CanvusLocalLLM on Windows
#
# Usage:
#   .\scripts\build-llama-windows.ps1 [OPTIONS]
#
# Options:
#   -Clean        Remove existing build directory before building
#   -Jobs N       Number of parallel build jobs (default: 8)
#   -OutputDir    Output directory for built libraries (default: deps\llama.cpp\build)
#   -NoInstall    Skip copying libraries to project lib\ directory
#   -Help         Show this help message
#
# Requirements:
#   - CUDA Toolkit (nvcc in PATH)
#   - CMake >= 3.14
#   - Visual Studio 2019 or 2022 with C++ workload
#   - Git

param(
    [switch]$Clean,
    [int]$Jobs = 8,
    [string]$OutputDir = "",
    [switch]$NoInstall,
    [switch]$Help
)

# Script configuration
$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
$LlamaCppRepo = "https://github.com/ggerganov/llama.cpp.git"
$LlamaCppDir = Join-Path $ProjectRoot "deps\llama.cpp"

if (-not $OutputDir) {
    $OutputDir = Join-Path $LlamaCppDir "build"
}

function Write-Info($message) {
    Write-Host "[INFO] $message" -ForegroundColor Cyan
}

function Write-Success($message) {
    Write-Host "[SUCCESS] $message" -ForegroundColor Green
}

function Write-Warning($message) {
    Write-Host "[WARN] $message" -ForegroundColor Yellow
}

function Write-Error($message) {
    Write-Host "[ERROR] $message" -ForegroundColor Red
}

function Show-Help {
    Get-Content $MyInvocation.ScriptName | Select-String -Pattern "^#" | ForEach-Object { $_.Line -replace "^# ?", "" } | Select-Object -Skip 1 -First 15
    exit 0
}

function Test-Dependencies {
    Write-Info "Checking dependencies..."

    $missing = @()

    # Check for git
    if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
        $missing += "git"
    }

    # Check for cmake
    if (-not (Get-Command cmake -ErrorAction SilentlyContinue)) {
        $missing += "cmake"
    }

    # Check for CUDA compiler
    $nvcc = Get-Command nvcc -ErrorAction SilentlyContinue
    if (-not $nvcc) {
        Write-Warning "nvcc not found in PATH - CUDA support may not be available"
        Write-Warning "Ensure CUDA Toolkit is installed and nvcc is in your PATH"
        Write-Warning "Continuing build (will fail if CUDA headers not found)..."
    } else {
        $cudaVersion = & nvcc --version | Select-String "release" | ForEach-Object { $_ -replace ".*release ", "" -replace ",.*", "" }
        Write-Info "Found CUDA version: $cudaVersion"
    }

    # Check for Visual Studio
    $vsWhere = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vswhere.exe"
    if (Test-Path $vsWhere) {
        $vsPath = & $vsWhere -latest -property installationPath
        if ($vsPath) {
            Write-Info "Found Visual Studio at: $vsPath"
        } else {
            $missing += "Visual Studio"
        }
    } else {
        Write-Warning "Could not locate Visual Studio - build may fail"
    }

    if ($missing.Count -gt 0) {
        Write-Error "Missing required dependencies: $($missing -join ', ')"
        Write-Error "Please install them and try again."
        exit 1
    }

    Write-Success "All required dependencies found"
}

function Initialize-Repository {
    Write-Info "Setting up llama.cpp repository..."

    if (Test-Path (Join-Path $LlamaCppDir ".git")) {
        Write-Info "Repository exists, pulling latest changes..."
        Push-Location $LlamaCppDir
        try {
            git fetch origin
            git pull origin master 2>$null
            if ($LASTEXITCODE -ne 0) {
                git pull origin main
            }
        } finally {
            Pop-Location
        }
    } else {
        Write-Info "Cloning llama.cpp repository..."
        $depsDir = Split-Path -Parent $LlamaCppDir
        if (-not (Test-Path $depsDir)) {
            New-Item -ItemType Directory -Path $depsDir -Force | Out-Null
        }
        git clone --depth 1 $LlamaCppRepo $LlamaCppDir
    }

    Write-Success "Repository ready at: $LlamaCppDir"
}

function Invoke-CMakeConfigure {
    Write-Info "Configuring CMake with CUDA support..."

    $buildDir = Join-Path $LlamaCppDir "build"

    if ($Clean -and (Test-Path $buildDir)) {
        Write-Info "Cleaning existing build directory..."
        Remove-Item -Recurse -Force $buildDir
    }

    if (-not (Test-Path $buildDir)) {
        New-Item -ItemType Directory -Path $buildDir | Out-Null
    }

    Push-Location $buildDir
    try {
        # Configure with CUDA support and shared libraries
        # Use Visual Studio generator
        cmake .. `
            -DCMAKE_BUILD_TYPE=Release `
            -DBUILD_SHARED_LIBS=ON `
            -DLLAMA_CUBLAS=ON `
            -DLLAMA_CUDA=ON `
            -DLLAMA_NATIVE=OFF `
            -DLLAMA_BUILD_TESTS=OFF `
            -DLLAMA_BUILD_EXAMPLES=ON `
            -DLLAMA_BUILD_SERVER=ON `
            -G "Visual Studio 17 2022"

        if ($LASTEXITCODE -ne 0) {
            # Try VS 2019 if 2022 fails
            Write-Warning "VS 2022 generator failed, trying VS 2019..."
            cmake .. `
                -DCMAKE_BUILD_TYPE=Release `
                -DBUILD_SHARED_LIBS=ON `
                -DLLAMA_CUBLAS=ON `
                -DLLAMA_CUDA=ON `
                -DLLAMA_NATIVE=OFF `
                -DLLAMA_BUILD_TESTS=OFF `
                -DLLAMA_BUILD_EXAMPLES=ON `
                -DLLAMA_BUILD_SERVER=ON `
                -G "Visual Studio 16 2019"
        }
    } finally {
        Pop-Location
    }

    Write-Success "CMake configuration complete"
}

function Invoke-Build {
    Write-Info "Building llama.cpp with $Jobs parallel jobs..."

    $buildDir = Join-Path $LlamaCppDir "build"

    Push-Location $buildDir
    try {
        cmake --build . --config Release -j $Jobs
    } finally {
        Pop-Location
    }

    Write-Success "Build complete"
}

function Copy-Artifacts {
    Write-Info "Copying build artifacts to: $OutputDir"

    $buildDir = Join-Path $LlamaCppDir "build"

    if ($OutputDir -ne $buildDir) {
        if (-not (Test-Path $OutputDir)) {
            New-Item -ItemType Directory -Path $OutputDir | Out-Null
        }

        # Copy DLLs
        Get-ChildItem -Path $buildDir -Recurse -Filter "*.dll" | ForEach-Object {
            Copy-Item $_.FullName -Destination $OutputDir -Force
        }

        # Copy server binary
        $serverPath = Join-Path $buildDir "bin\Release\llama-server.exe"
        if (Test-Path $serverPath) {
            Copy-Item $serverPath -Destination $OutputDir -Force
        }

        Write-Success "Artifacts copied to: $OutputDir"
    } else {
        Write-Info "Artifacts available in: $OutputDir"
    }

    # List what was built
    Write-Info "Built artifacts:"
    Get-ChildItem -Path $OutputDir -Filter "*.dll" | Format-Table Name, Length
    Get-ChildItem -Path $OutputDir -Filter "*.exe" | Format-Table Name, Length
}

function Install-ToProject {
    if ($NoInstall) {
        Write-Info "Skipping project installation (-NoInstall specified)"
        return
    }

    Write-Info "Installing llama.cpp to project directories..."

    $libDir = Join-Path $ProjectRoot "lib"
    if (-not (Test-Path $libDir)) {
        New-Item -ItemType Directory -Path $libDir | Out-Null
    }

    $buildDir = Join-Path $LlamaCppDir "build"
    $libCount = 0

    # Search locations for DLLs
    $searchDirs = @(
        $buildDir,
        (Join-Path $buildDir "Release"),
        (Join-Path $buildDir "bin\Release"),
        (Join-Path $buildDir "src\Release")
    )

    foreach ($searchDir in $searchDirs) {
        if (Test-Path $searchDir) {
            # Copy llama DLLs
            Get-ChildItem -Path $searchDir -Filter "llama*.dll" -ErrorAction SilentlyContinue | ForEach-Object {
                Copy-Item $_.FullName -Destination $libDir -Force
                Write-Info "  Copied: $($_.Name)"
                $libCount++
            }

            # Copy GGML DLLs
            Get-ChildItem -Path $searchDir -Filter "ggml*.dll" -ErrorAction SilentlyContinue | ForEach-Object {
                Copy-Item $_.FullName -Destination $libDir -Force
                Write-Info "  Copied: $($_.Name)"
                $libCount++
            }
        }
    }

    if ($libCount -eq 0) {
        Write-Warning "No DLLs found to install!"
        Write-Warning "Build may have created static libraries instead."
        Write-Warning "Check: $buildDir for *.lib files"
        return
    }

    Write-Success "Installed $libCount library file(s) to: $libDir"

    # Verify the critical library exists
    if (Test-Path (Join-Path $libDir "llama.dll")) {
        Write-Success "llama.dll ready for CGo bindings"
    } else {
        Write-Warning "llama.dll not found - CGo bindings may not work"
        Write-Warning "Ensure llama.cpp was built with BUILD_SHARED_LIBS=ON"
    }

    # List installed libraries
    Write-Info "Installed libraries in ${libDir}:"
    Get-ChildItem -Path $libDir -Filter "*.dll" | Format-Table Name, Length
}

# Main execution
if ($Help) {
    Show-Help
}

Write-Host ""
Write-Host "========================================"
Write-Host "  llama.cpp CUDA Build Script (Windows)"
Write-Host "========================================"
Write-Host ""

Test-Dependencies
Initialize-Repository
Invoke-CMakeConfigure
Invoke-Build
Copy-Artifacts
Install-ToProject

Write-Host ""
Write-Success "llama.cpp build complete!"
Write-Host ""
Write-Host "Next steps:"
Write-Host "  - Server binary: $OutputDir\llama-server.exe"
Write-Host "  - Start server:  $OutputDir\llama-server.exe -m path\to\model.gguf"
if (-not $NoInstall) {
    Write-Host "  - Libraries installed to: $ProjectRoot\lib\"
    Write-Host "  - CGo bindings should now find llama.dll"
}
Write-Host ""
