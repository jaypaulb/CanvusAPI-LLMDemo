; CanvusLocalLLM Windows Installer Script
; NSIS 3.x with Modern UI 2
;
; This script creates a Windows installer with:
; - License agreement page (MIT)
; - Installation directory selection
; - Component selection (core, service, shortcuts, Stable Diffusion)
; - File installation with progress
; - Uninstaller generation
; - Add/Remove Programs integration
;
; Build with: makensis canvusapi.nsi
;
; Prerequisites:
; 1. Build the Go binary: GOOS=windows GOARCH=amd64 go build -o bin/CanvusLocalLLM.exe .
; 2. Build stable-diffusion.dll: deps\stable-diffusion.cpp\build-windows.ps1
; 3. Download SD model: models\sd-v1-5.safetensors (~4GB)
; 4. Ensure LICENSE.txt and README.md exist in the project root
; 5. Run from installer/windows/ directory or adjust paths
;
; Expected output size: ~13GB with SD model, ~10MB without

;--------------------------------
; Product Information
;--------------------------------

!define PRODUCT_NAME "CanvusLocalLLM"
!define PRODUCT_VERSION "0.1.0"
!define PRODUCT_PUBLISHER "Jaypaul Bridger"
!define PRODUCT_WEB_SITE "https://github.com/jaypaulb/CanvusAPI-LLMDemo"
!define PRODUCT_DESCRIPTION "Zero-configuration local AI integration for Canvus"
!define PRODUCT_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"
!define PRODUCT_UNINST_ROOT_KEY "HKLM"

; Service name for Windows Service registration
!define SERVICE_NAME "CanvusLocalLLM"

; Stable Diffusion model information
!define SD_MODEL_NAME "sd-v1-5.safetensors"
!define SD_MODEL_SIZE_MB "4265"

;--------------------------------
; Includes
;--------------------------------

; Include Modern UI 2
!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "LogicLib.nsh"
!include "Sections.nsh"

;--------------------------------
; General Settings
;--------------------------------

; Installer name shown in title bar
Name "${PRODUCT_NAME} ${PRODUCT_VERSION}"

; Output installer filename
OutFile "CanvusLocalLLM-${PRODUCT_VERSION}-Setup.exe"

; Default installation directory
InstallDir "$PROGRAMFILES\CanvusLocalLLM"

; Get installation folder from registry if available (for upgrades)
InstallDirRegKey ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "InstallLocation"

; Request application privileges for Windows Vista and later
RequestExecutionLevel admin

; Show installation details
ShowInstDetails show
ShowUnInstDetails show

; Compression settings
; Note: For large files like SD models, LZMA provides excellent compression
; but installation will be slower. Consider zlib for faster installs.
SetCompressor /SOLID lzma
SetCompressorDictSize 64

;--------------------------------
; Interface Configuration
;--------------------------------

; Abort warning on cancel
!define MUI_ABORTWARNING
!define MUI_ABORTWARNING_TEXT "Are you sure you want to cancel ${PRODUCT_NAME} installation?"

; Icons (use default NSIS icons - uncomment below for custom icons)
; !define MUI_ICON "icon.ico"
; !define MUI_UNICON "icon.ico"

; Header images (optional - uncomment if you have custom images)
; !define MUI_HEADERIMAGE
; !define MUI_HEADERIMAGE_BITMAP "header.bmp"
; !define MUI_WELCOMEFINISHPAGE_BITMAP "welcome.bmp"

; Welcome page text
!define MUI_WELCOMEPAGE_TITLE "Welcome to ${PRODUCT_NAME} Setup"
!define MUI_WELCOMEPAGE_TEXT "This wizard will guide you through the installation of ${PRODUCT_NAME}.$\r$\n$\r$\n${PRODUCT_NAME} provides zero-configuration local AI integration for Canvus collaborative workspaces. All AI processing happens locally on your machine.$\r$\n$\r$\nClick Next to continue."

; Finish page options
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_FINISHPAGE_RUN "$INSTDIR\CanvusLocalLLM.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Launch ${PRODUCT_NAME} now"
!define MUI_FINISHPAGE_RUN_NOTCHECKED
!define MUI_FINISHPAGE_SHOWREADME "$INSTDIR\README.txt"
!define MUI_FINISHPAGE_SHOWREADME_TEXT "View README file"
!define MUI_FINISHPAGE_SHOWREADME_NOTCHECKED
!define MUI_FINISHPAGE_LINK "Visit ${PRODUCT_NAME} on GitHub"
!define MUI_FINISHPAGE_LINK_LOCATION "${PRODUCT_WEB_SITE}"

;--------------------------------
; Installer Pages
;--------------------------------

; Welcome page
!insertmacro MUI_PAGE_WELCOME

; License agreement page (MIT License)
!insertmacro MUI_PAGE_LICENSE "..\..\LICENSE.txt"

; Directory selection page
!insertmacro MUI_PAGE_DIRECTORY

; Components selection page
!insertmacro MUI_PAGE_COMPONENTS

; Installation progress page
!insertmacro MUI_PAGE_INSTFILES

; Finish page
!insertmacro MUI_PAGE_FINISH

;--------------------------------
; Uninstaller Pages
;--------------------------------

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

;--------------------------------
; Languages
;--------------------------------

!insertmacro MUI_LANGUAGE "English"

;--------------------------------
; Installer Version Information
;--------------------------------

VIProductVersion "${PRODUCT_VERSION}.0"
VIAddVersionKey /LANG=${LANG_ENGLISH} "ProductName" "${PRODUCT_NAME}"
VIAddVersionKey /LANG=${LANG_ENGLISH} "ProductVersion" "${PRODUCT_VERSION}"
VIAddVersionKey /LANG=${LANG_ENGLISH} "CompanyName" "${PRODUCT_PUBLISHER}"
VIAddVersionKey /LANG=${LANG_ENGLISH} "LegalCopyright" "MIT License - Copyright (c) 2024-2025 ${PRODUCT_PUBLISHER}"
VIAddVersionKey /LANG=${LANG_ENGLISH} "FileDescription" "${PRODUCT_DESCRIPTION}"
VIAddVersionKey /LANG=${LANG_ENGLISH} "FileVersion" "${PRODUCT_VERSION}"

;--------------------------------
; Variables
;--------------------------------

Var SDInstalled

;--------------------------------
; Installation Sections
;--------------------------------

; Core Application (Required)
Section "Core Application" SecCore
    SectionIn RO  ; Read-only - cannot be deselected

    ; Set output path to the installation directory
    SetOutPath "$INSTDIR"

    ; Install the main executable
    ; Note: Build with: GOOS=windows GOARCH=amd64 go build -o bin/CanvusLocalLLM.exe .
    File "..\..\bin\CanvusLocalLLM.exe"

    ; Install configuration template
    File "..\..\example.env"
    Rename "$INSTDIR\example.env" "$INSTDIR\.env.example"

    ; Install documentation
    File "..\..\README.md"
    Rename "$INSTDIR\README.md" "$INSTDIR\README.txt"

    ; Install license
    File "..\..\LICENSE.txt"

    ; Create directory structure for runtime files
    CreateDirectory "$INSTDIR\lib"
    CreateDirectory "$INSTDIR\models"
    CreateDirectory "$INSTDIR\downloads"
    CreateDirectory "$INSTDIR\logs"

    ; Create uninstaller
    WriteUninstaller "$INSTDIR\Uninstall.exe"

    ; Write registry entries for Add/Remove Programs
    WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "DisplayName" "${PRODUCT_NAME}"
    WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "DisplayVersion" "${PRODUCT_VERSION}"
    WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
    WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "URLInfoAbout" "${PRODUCT_WEB_SITE}"
    WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "UninstallString" "$INSTDIR\Uninstall.exe"
    WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "QuietUninstallString" '"$INSTDIR\Uninstall.exe" /S'
    WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "InstallLocation" "$INSTDIR"
    WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "DisplayIcon" "$INSTDIR\CanvusLocalLLM.exe,0"
    WriteRegDWORD ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "NoModify" 1
    WriteRegDWORD ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "NoRepair" 1

    ; Initialize SD installed flag
    StrCpy $SDInstalled "0"

    ; Calculate and write installed size
    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "EstimatedSize" "$0"
SectionEnd

; Stable Diffusion Support (Optional)
Section "Stable Diffusion Support" SecSD
    ; Check disk space before installing large files
    ; SD model is ~4GB, DLL is ~50MB
    ; We need approximately 4.5GB free space

    DetailPrint "Installing Stable Diffusion support..."
    DetailPrint "This includes a ~4GB model file and may take several minutes."

    ; Install stable-diffusion.dll to lib directory
    SetOutPath "$INSTDIR\lib"

    ; Note: Build the DLL first with: deps\stable-diffusion.cpp\build-windows.ps1
    File "..\..\lib\stable-diffusion.dll"

    DetailPrint "Installed stable-diffusion.dll"

    ; Install SD model to models directory
    SetOutPath "$INSTDIR\models"

    ; Note: Download the model first from HuggingFace
    ; https://huggingface.co/runwayml/stable-diffusion-v1-5
    ; File: v1-5-pruned-emaonly.safetensors (~4GB)
    ; Rename to sd-v1-5.safetensors
    File "..\..\models\${SD_MODEL_NAME}"

    DetailPrint "Installed ${SD_MODEL_NAME} (${SD_MODEL_SIZE_MB} MB)"

    ; Mark SD as installed for later configuration
    StrCpy $SDInstalled "1"
    WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "SDInstalled" "1"

    ; Update estimated size to include SD files
    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "EstimatedSize" "$0"

    DetailPrint "Stable Diffusion support installed successfully"
    DetailPrint ""
    DetailPrint "IMPORTANT: Stable Diffusion requires an NVIDIA GPU with CUDA support."
    DetailPrint "Ensure you have the latest NVIDIA drivers and CUDA runtime installed."
SectionEnd

; Windows Service Installation (Optional)
Section "Install as Windows Service" SecService
    ; Create .env from template if it doesn't exist
    ${IfNot} ${FileExists} "$INSTDIR\.env"
        CopyFiles "$INSTDIR\.env.example" "$INSTDIR\.env"
        DetailPrint "Created .env configuration file from template"
    ${EndIf}

    ; Install the Windows service
    ; Note: The application must support service installation via CLI
    DetailPrint "Installing Windows Service..."
    nsExec::ExecToLog '"$INSTDIR\CanvusLocalLLM.exe" install'
    Pop $0
    ${If} $0 == 0
        DetailPrint "Windows Service installed successfully"

        ; Note: Service won't start without valid configuration
        DetailPrint "Service installed. Configure .env before starting."
        MessageBox MB_OK|MB_ICONINFORMATION "Windows Service installed successfully!$\n$\nBefore starting the service:$\n1. Edit $INSTDIR\.env$\n2. Configure your Canvus credentials$\n3. Start service via Services console or:$\n   sc start ${SERVICE_NAME}"
    ${Else}
        DetailPrint "Warning: Could not install Windows Service (error: $0)"
        MessageBox MB_OK|MB_ICONINFORMATION "Windows Service installation was skipped.$\n$\nYou can install the service manually later by running:$\n$INSTDIR\CanvusLocalLLM.exe install"
    ${EndIf}
SectionEnd

; Desktop Shortcut (Optional)
Section "Desktop Shortcut" SecDesktop
    CreateShortcut "$DESKTOP\${PRODUCT_NAME}.lnk" "$INSTDIR\CanvusLocalLLM.exe" "" "$INSTDIR\CanvusLocalLLM.exe" 0
    DetailPrint "Created desktop shortcut"
SectionEnd

; Start Menu Shortcuts (Optional - enabled by default)
Section "Start Menu Shortcuts" SecStartMenu
    SectionIn 1  ; Selected by default in full installation

    ; Create Start Menu folder
    CreateDirectory "$SMPROGRAMS\${PRODUCT_NAME}"

    ; Application shortcut
    CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk" "$INSTDIR\CanvusLocalLLM.exe" "" "$INSTDIR\CanvusLocalLLM.exe" 0

    ; README shortcut
    CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}\README.lnk" "$INSTDIR\README.txt"

    ; Configuration file shortcut (opens .env in notepad)
    CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}\Edit Configuration.lnk" "notepad.exe" "$INSTDIR\.env" "" "" SW_SHOWNORMAL "" "Edit CanvusLocalLLM configuration"

    ; Logs folder shortcut
    CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}\View Logs.lnk" "$INSTDIR\logs"

    ; Uninstall shortcut
    CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}\Uninstall.lnk" "$INSTDIR\Uninstall.exe"

    DetailPrint "Created Start Menu shortcuts"
SectionEnd

;--------------------------------
; Section Descriptions
;--------------------------------

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
    !insertmacro MUI_DESCRIPTION_TEXT ${SecCore} "Core application files required for ${PRODUCT_NAME} to run. This includes the main executable, configuration templates, and documentation."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecSD} "Stable Diffusion local image generation support. Includes stable-diffusion.dll runtime (~50MB) and SD v1.5 model (~4GB). Requires NVIDIA GPU with CUDA support and latest drivers."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecService} "Install ${PRODUCT_NAME} as a Windows Service that can be configured to start automatically with Windows. Recommended for production deployments."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecDesktop} "Create a shortcut on your desktop for easy access to ${PRODUCT_NAME}."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecStartMenu} "Create shortcuts in the Windows Start Menu including quick access to configuration and documentation."
!insertmacro MUI_FUNCTION_DESCRIPTION_END

;--------------------------------
; Uninstaller Section
;--------------------------------

Section "Uninstall"
    ; Stop and remove Windows Service if installed
    DetailPrint "Checking for Windows Service..."
    nsExec::ExecToLog 'sc query ${SERVICE_NAME}'
    Pop $0
    ${If} $0 == 0
        DetailPrint "Stopping Windows Service..."
        nsExec::ExecToLog 'sc stop ${SERVICE_NAME}'

        ; Wait for service to stop
        Sleep 2000

        DetailPrint "Removing Windows Service..."
        nsExec::ExecToLog '"$INSTDIR\CanvusLocalLLM.exe" uninstall'

        ; Wait for service removal to complete
        Sleep 1000
    ${EndIf}

    ; Remove main executable
    Delete "$INSTDIR\CanvusLocalLLM.exe"

    ; Remove configuration template
    Delete "$INSTDIR\.env.example"

    ; Ask before removing user configuration
    ${If} ${FileExists} "$INSTDIR\.env"
        MessageBox MB_YESNO|MB_ICONQUESTION "Do you want to remove your configuration file (.env)?$\n$\nThis file contains your Canvus credentials and settings." IDYES removeEnv IDNO skipEnv
        removeEnv:
            Delete "$INSTDIR\.env"
        skipEnv:
    ${EndIf}

    ; Remove documentation
    Delete "$INSTDIR\README.txt"
    Delete "$INSTDIR\LICENSE.txt"
    Delete "$INSTDIR\Uninstall.exe"

    ; Remove app.log if exists in root
    Delete "$INSTDIR\app.log"

    ; Remove Stable Diffusion files
    ; Check if SD was installed via registry
    ReadRegStr $0 ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "SDInstalled"
    ${If} $0 == "1"
        ; Remove stable-diffusion.dll
        Delete "$INSTDIR\lib\stable-diffusion.dll"
        DetailPrint "Removed stable-diffusion.dll"
    ${EndIf}

    ; Remove lib directory (only if empty - user may have added other files)
    RMDir "$INSTDIR\lib"

    ; Handle models directory - may contain large files
    ${If} ${FileExists} "$INSTDIR\models\*.*"
        ; Check for SD model specifically
        ${If} ${FileExists} "$INSTDIR\models\${SD_MODEL_NAME}"
            MessageBox MB_YESNO|MB_ICONQUESTION "Do you want to remove the Stable Diffusion model?$\n$\nThis file is approximately ${SD_MODEL_SIZE_MB} MB and will need to be re-downloaded if you reinstall with SD support." IDYES removeSDModel IDNO skipSDModel
            removeSDModel:
                Delete "$INSTDIR\models\${SD_MODEL_NAME}"
                DetailPrint "Removed ${SD_MODEL_NAME}"
            skipSDModel:
        ${EndIf}

        ; Ask about other models (LLM models may be up to 8GB)
        ${If} ${FileExists} "$INSTDIR\models\*.*"
            MessageBox MB_YESNO|MB_ICONQUESTION "Do you want to remove remaining AI models?$\n$\nThese files may be large (up to 8GB each) and will need to be re-downloaded if you reinstall." IDYES removeModels IDNO skipModels
            removeModels:
                RMDir /r "$INSTDIR\models"
            skipModels:
        ${EndIf}
    ${Else}
        RMDir "$INSTDIR\models"
    ${EndIf}

    ; Ask before removing downloads
    ${If} ${FileExists} "$INSTDIR\downloads\*.*"
        MessageBox MB_YESNO|MB_ICONQUESTION "Do you want to remove the downloads folder?$\n$\nThis folder may contain processed files." IDYES removeDownloads IDNO skipDownloads
        removeDownloads:
            RMDir /r "$INSTDIR\downloads"
        skipDownloads:
    ${Else}
        RMDir "$INSTDIR\downloads"
    ${EndIf}

    ; Ask before removing logs
    ${If} ${FileExists} "$INSTDIR\logs\*.*"
        MessageBox MB_YESNO|MB_ICONQUESTION "Do you want to remove application logs?" IDYES removeLogs IDNO skipLogs
        removeLogs:
            RMDir /r "$INSTDIR\logs"
        skipLogs:
    ${Else}
        RMDir "$INSTDIR\logs"
    ${EndIf}

    ; Remove installation directory (only if empty)
    RMDir "$INSTDIR"

    ; Remove desktop shortcut
    Delete "$DESKTOP\${PRODUCT_NAME}.lnk"

    ; Remove Start Menu shortcuts
    Delete "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk"
    Delete "$SMPROGRAMS\${PRODUCT_NAME}\README.lnk"
    Delete "$SMPROGRAMS\${PRODUCT_NAME}\Edit Configuration.lnk"
    Delete "$SMPROGRAMS\${PRODUCT_NAME}\View Logs.lnk"
    Delete "$SMPROGRAMS\${PRODUCT_NAME}\Uninstall.lnk"
    RMDir "$SMPROGRAMS\${PRODUCT_NAME}"

    ; Remove registry entries
    DeleteRegKey ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}"

    ; Final cleanup check
    ${If} ${FileExists} "$INSTDIR\*.*"
        MessageBox MB_OK|MB_ICONINFORMATION "Some files remain in $INSTDIR.$\n$\nThese are files created after installation and were not removed."
    ${EndIf}
SectionEnd

;--------------------------------
; Installer Functions
;--------------------------------

Function .onInit
    ; Check if already installed (for upgrade scenarios)
    ReadRegStr $0 ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "UninstallString"
    ${If} $0 != ""
        MessageBox MB_OKCANCEL|MB_ICONINFORMATION "${PRODUCT_NAME} is already installed.$\n$\nClick OK to upgrade the existing installation or Cancel to exit." IDOK upgradeOK
            Abort
        upgradeOK:
    ${EndIf}

    ; Initialize SD installed variable
    StrCpy $SDInstalled "0"
FunctionEnd

Function .onSelChange
    ; Update disk space requirements based on selected components
    ; This provides feedback to users about required space

    ; Check if SD section is selected
    SectionGetFlags ${SecSD} $0
    IntOp $0 $0 & ${SF_SELECTED}
    ${If} $0 != 0
        ; SD is selected - warn about disk space requirements
        ; Note: NSIS handles this automatically, but we can add extra messaging
    ${EndIf}
FunctionEnd

Function .onInstSuccess
    ; Post-installation message for first-time configuration
    ${IfNot} ${FileExists} "$INSTDIR\.env"
        ; Copy template to .env for first-run configuration
        CopyFiles "$INSTDIR\.env.example" "$INSTDIR\.env"
    ${EndIf}

    ; If SD was installed, update .env with model path
    ${If} $SDInstalled == "1"
        ; Note: We don't modify .env automatically as users should review settings
        ; The default SD_MODEL_PATH in .env.example points to ./models/sd-v1-5.safetensors
    ${EndIf}

    ; Always show configuration reminder
    ${If} $SDInstalled == "1"
        MessageBox MB_YESNO|MB_ICONINFORMATION "Installation complete with Stable Diffusion support!$\n$\nBefore running ${PRODUCT_NAME}, you must configure your Canvus credentials.$\n$\nIMPORTANT: Local image generation requires:$\n- NVIDIA GPU with CUDA support$\n- Latest NVIDIA drivers installed$\n- CUDA runtime libraries in PATH$\n$\nWould you like to edit the configuration file now?" IDYES openConfig IDNO skipConfig
    ${Else}
        MessageBox MB_YESNO|MB_ICONINFORMATION "Installation complete!$\n$\nBefore running ${PRODUCT_NAME}, you must configure your Canvus credentials.$\n$\nWould you like to edit the configuration file now?" IDYES openConfig IDNO skipConfig
    ${EndIf}
    openConfig:
        ; Open .env in notepad for editing
        Exec 'notepad.exe "$INSTDIR\.env"'
    skipConfig:
FunctionEnd

;--------------------------------
; Uninstaller Functions
;--------------------------------

Function un.onInit
    ; Confirm uninstallation
    MessageBox MB_ICONQUESTION|MB_YESNO "Are you sure you want to completely remove ${PRODUCT_NAME}?" IDYES confirmUninstall
        Abort
    confirmUninstall:
FunctionEnd

Function un.onUninstSuccess
    MessageBox MB_ICONINFORMATION|MB_OK "${PRODUCT_NAME} has been successfully removed from your computer."
FunctionEnd
