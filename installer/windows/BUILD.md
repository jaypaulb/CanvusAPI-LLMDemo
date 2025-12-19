# Building the Windows Installer

## Installer Scripts

This directory contains multiple NSIS installer scripts:

- **canvus-llm.nsi** (RECOMMENDED) - Minimal zero-config installer
  - Simple wizard flow (license → directory → install)
  - Creates minimal .env template with placeholders
  - Optional bundled model support
  - Desktop and Start Menu shortcuts
  - ~50 MB without models, 2-8 GB with bundled models

- **canvusapi.nsi** (LEGACY) - Full-featured installer with service support
  - Component selection (core, service, shortcuts, Stable Diffusion)
  - Windows Service installation
  - Larger installer (~13 GB with SD model)
  - More complex setup workflow

- **setup.nsi** (LEGACY) - Earlier installer version

**For most users, use `canvus-llm.nsi` for simplicity and smaller installer size.**

## Prerequisites

1. **NSIS (Nullsoft Scriptable Install System)**
   - Download from: https://nsis.sourceforge.io/Download
   - Install NSIS 3.x or later
   - Add NSIS to your system PATH

2. **Build the Application**
   ```bash
   cd go_backend
   GOOS=windows GOARCH=amd64 go build -o ../CanvusAPI-LLM.exe .
   ```

## Building the Installer

### Basic Build (Recommended)

From the project root directory:

```bash
cd installer/windows
makensis /DVERSION=1.0.0 canvus-llm.nsi
```

This creates `CanvusLocalLLM-Setup.exe` in the current directory.

### Version-Specific Build

```bash
makensis /DVERSION=1.2.3 canvus-llm.nsi
```

Replace `1.2.3` with your actual version number.

### With Bundled Models (Optional)

To bundle AI models with the installer:

1. Create a `models/` directory in `installer/windows/`
2. Place model files in this directory
3. Build as normal - the installer will include them

```bash
mkdir -p models
cp /path/to/your/model.gguf models/
makensis /DVERSION=1.0.0 canvus-llm.nsi
```

**Note:** Bundled models increase installer size significantly (2-8 GB per model).
Consider whether automatic download on first run is better for your use case.

## Build Outputs

- **Installer:** `CanvusLocalLLM-Setup.exe`
- **Size:** ~50 MB without models, 2-8 GB with bundled models
- **Target:** Windows 7/8/10/11 (64-bit)

## Testing the Installer

### In a VM or Test Machine

1. Run `CanvusLocalLLM-Setup.exe` as Administrator
2. Follow the installation wizard
3. Verify files are installed to `C:\Program Files\CanvusLocalLLM\`
4. Check that:
   - `.env` file is created from template
   - Desktop and Start Menu shortcuts work
   - Uninstaller is registered in Add/Remove Programs

### Manual Testing Checklist

- [ ] Installer requires admin rights
- [ ] License page displays correctly
- [ ] Installation directory can be changed
- [ ] All files copy successfully
- [ ] .env template is created (doesn't overwrite existing)
- [ ] Models directory is created (if bundled)
- [ ] Shortcuts created on Desktop and Start Menu
- [ ] Uninstaller appears in Add/Remove Programs
- [ ] Application starts from shortcuts
- [ ] README opens from shortcut
- [ ] Uninstaller removes all files
- [ ] Uninstaller prompts for data preservation

## Troubleshooting

### "Error: Can't open script file"

Ensure you're running makensis from the `installer/windows/` directory, or provide the full path:

```bash
makensis /DVERSION=1.0.0 /path/to/installer/windows/canvus-llm.nsi
```

### "File not found: CanvusAPI-LLM.exe"

The script expects the executable at `../../CanvusAPI-LLM.exe` (relative to installer/windows/).
Build the application first or adjust the path in the .nsi script.

### "File not found: LICENSE.txt"

Ensure LICENSE.txt is in the `installer/windows/` directory.

### Version not set

Always provide the VERSION define:

```bash
makensis /DVERSION=1.0.0 canvus-llm.nsi
```

## Automation

For CI/CD integration, see the GitHub Actions workflow in `.github/workflows/build-installer.yml` (if available).

Example automation script:

```powershell
# build-installer.ps1
$VERSION = "1.0.0"
$PROJECT_ROOT = "C:\path\to\CanvusLocalLLM"

# Build Go executable
cd "$PROJECT_ROOT\go_backend"
go build -o "..\CanvusAPI-LLM.exe" .

# Build installer
cd "$PROJECT_ROOT\installer\windows"
makensis /DVERSION=$VERSION canvus-llm.nsi

# Move installer to release directory
Move-Item "CanvusLocalLLM-Setup.exe" "$PROJECT_ROOT\release\CanvusLocalLLM-Setup-$VERSION.exe"
```

## Distribution

The resulting `CanvusLocalLLM-Setup.exe` is a standalone installer that can be:

- Distributed via download links
- Published to GitHub Releases
- Shared via company intranet
- Deployed via software management systems

No additional dependencies required for end users.
