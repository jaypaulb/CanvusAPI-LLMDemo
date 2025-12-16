; CanvusLocalLLM Windows Installer Script
; NSIS 3.x with Modern UI 2
;
; This script creates a Windows installer with:
; - License agreement page
; - Installation directory selection
; - Component selection (core, service, shortcuts)
; - File installation with progress
; - Uninstaller generation
; - Add/Remove Programs integration
;
; Build with: makensis setup.nsi

;--------------------------------
; Product Information
;--------------------------------

!define PRODUCT_NAME "CanvusLocalLLM"
!define PRODUCT_VERSION "0.1.0"
!define PRODUCT_PUBLISHER "CanvusLocalLLM Project"
!define PRODUCT_WEB_SITE "https://github.com/yourusername/CanvusLocalLLM"
!define PRODUCT_DESCRIPTION "Zero-configuration local AI integration for Canvus"
!define PRODUCT_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"
!define PRODUCT_UNINST_ROOT_KEY "HKLM"

;--------------------------------
; Includes
;--------------------------------

; Include Modern UI 2
!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "LogicLib.nsh"

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
SetCompressor /SOLID lzma
SetCompressorDictSize 64

;--------------------------------
; Interface Configuration
;--------------------------------

; Abort warning on cancel
!define MUI_ABORTWARNING
!define MUI_ABORTWARNING_TEXT "Are you sure you want to cancel ${PRODUCT_NAME} installation?"

; Icons (use default if custom icons not available)
; !define MUI_ICON "${NSISDIR}\Contrib\Graphics\Icons\modern-install.ico"
; !define MUI_UNICON "${NSISDIR}\Contrib\Graphics\Icons\modern-uninstall.ico"

; Header images (optional - uncomment if you have custom images)
; !define MUI_HEADERIMAGE
; !define MUI_HEADERIMAGE_BITMAP "header.bmp"
; !define MUI_WELCOMEFINISHPAGE_BITMAP "welcome.bmp"

; Finish page options
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_FINISHPAGE_RUN "$INSTDIR\CanvusLocalLLM.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Launch ${PRODUCT_NAME} now"
!define MUI_FINISHPAGE_RUN_NOTCHECKED
!define MUI_FINISHPAGE_SHOWREADME "$INSTDIR\README.txt"
!define MUI_FINISHPAGE_SHOWREADME_TEXT "View README file"
!define MUI_FINISHPAGE_LINK "Visit ${PRODUCT_NAME} website"
!define MUI_FINISHPAGE_LINK_LOCATION "${PRODUCT_WEB_SITE}"

;--------------------------------
; Installer Pages
;--------------------------------

; Welcome page
!insertmacro MUI_PAGE_WELCOME

; License agreement page
!insertmacro MUI_PAGE_LICENSE "..\..\LICENSE"

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
VIAddVersionKey /LANG=${LANG_ENGLISH} "LegalCopyright" "MIT License"
VIAddVersionKey /LANG=${LANG_ENGLISH} "FileDescription" "${PRODUCT_DESCRIPTION}"
VIAddVersionKey /LANG=${LANG_ENGLISH} "FileVersion" "${PRODUCT_VERSION}"

;--------------------------------
; Installation Sections
;--------------------------------

; Core Application (Required)
Section "Core Application" SecCore
    SectionIn RO  ; Read-only - cannot be deselected

    ; Set output path to the installation directory
    SetOutPath "$INSTDIR"

    ; Install the main executable
    ; Note: Update path to match your build output location
    File "..\..\bin\CanvusLocalLLM.exe"

    ; Install configuration template
    File "..\..\example.env"
    Rename "$INSTDIR\example.env" "$INSTDIR\.env.example"

    ; Install documentation
    File "..\..\README.md"
    Rename "$INSTDIR\README.md" "$INSTDIR\README.txt"
    File "..\..\LICENSE"
    Rename "$INSTDIR\LICENSE" "$INSTDIR\LICENSE.txt"

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

    ; Calculate and write installed size
    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "EstimatedSize" "$0"
SectionEnd

; Windows Service Installation (Optional)
Section "Install as Windows Service" SecService
    ; Create .env from template if it doesn't exist
    ${IfNot} ${FileExists} "$INSTDIR\.env"
        CopyFiles "$INSTDIR\.env.example" "$INSTDIR\.env"
    ${EndIf}

    ; Install and start the Windows service
    ; Note: The application must support service installation via CLI
    DetailPrint "Installing Windows Service..."
    nsExec::ExecToLog '"$INSTDIR\CanvusLocalLLM.exe" install'
    Pop $0
    ${If} $0 == 0
        DetailPrint "Windows Service installed successfully"

        ; Start the service
        DetailPrint "Starting Windows Service..."
        nsExec::ExecToLog 'sc start CanvusLocalLLM'
        Pop $0
        ${If} $0 == 0
            DetailPrint "Windows Service started successfully"
        ${Else}
            DetailPrint "Note: Service installed but not started. Configure .env first."
        ${EndIf}
    ${Else}
        DetailPrint "Warning: Could not install Windows Service (error: $0)"
        MessageBox MB_OK|MB_ICONINFORMATION "Windows Service installation was skipped.$\n$\nYou can install the service manually later by running:$\n$INSTDIR\CanvusLocalLLM.exe install"
    ${EndIf}
SectionEnd

; Desktop Shortcut (Optional)
Section "Desktop Shortcut" SecDesktop
    CreateShortcut "$DESKTOP\${PRODUCT_NAME}.lnk" "$INSTDIR\CanvusLocalLLM.exe" "" "$INSTDIR\CanvusLocalLLM.exe" 0
SectionEnd

; Start Menu Shortcuts (Optional)
Section "Start Menu Shortcuts" SecStartMenu
    ; Create Start Menu folder
    CreateDirectory "$SMPROGRAMS\${PRODUCT_NAME}"

    ; Application shortcut
    CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk" "$INSTDIR\CanvusLocalLLM.exe" "" "$INSTDIR\CanvusLocalLLM.exe" 0

    ; README shortcut
    CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}\README.lnk" "$INSTDIR\README.txt"

    ; Configuration file shortcut
    CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}\Edit Configuration.lnk" "notepad.exe" "$INSTDIR\.env"

    ; Uninstall shortcut
    CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}\Uninstall.lnk" "$INSTDIR\Uninstall.exe"
SectionEnd

;--------------------------------
; Section Descriptions
;--------------------------------

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
    !insertmacro MUI_DESCRIPTION_TEXT ${SecCore} "Core application files required for ${PRODUCT_NAME} to run. This includes the main executable, configuration templates, and documentation."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecService} "Install ${PRODUCT_NAME} as a Windows Service that starts automatically with Windows. Recommended for production deployments."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecDesktop} "Create a shortcut on your desktop for easy access to ${PRODUCT_NAME}."
    !insertmacro MUI_DESCRIPTION_TEXT ${SecStartMenu} "Create shortcuts in the Windows Start Menu including quick access to configuration and documentation."
!insertmacro MUI_FUNCTION_DESCRIPTION_END

;--------------------------------
; Uninstaller Section
;--------------------------------

Section "Uninstall"
    ; Stop and remove Windows Service if installed
    DetailPrint "Checking for Windows Service..."
    nsExec::ExecToLog 'sc query CanvusLocalLLM'
    Pop $0
    ${If} $0 == 0
        DetailPrint "Stopping Windows Service..."
        nsExec::ExecToLog 'sc stop CanvusLocalLLM'

        DetailPrint "Removing Windows Service..."
        nsExec::ExecToLog '"$INSTDIR\CanvusLocalLLM.exe" uninstall'

        ; Wait a moment for service to fully stop
        Sleep 2000
    ${EndIf}

    ; Remove main executable
    Delete "$INSTDIR\CanvusLocalLLM.exe"

    ; Remove configuration files
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

    ; Remove directories (only if empty or user confirms)
    RMDir "$INSTDIR\lib"

    ; Ask before removing models (may be large)
    ${If} ${FileExists} "$INSTDIR\models\*.*"
        MessageBox MB_YESNO|MB_ICONQUESTION "Do you want to remove downloaded AI models?$\n$\nThese files may be large and can be re-downloaded later." IDYES removeModels IDNO skipModels
        removeModels:
            RMDir /r "$INSTDIR\models"
        skipModels:
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

    ; Remove app.log if exists
    Delete "$INSTDIR\app.log"

    ; Remove installation directory (only if empty)
    RMDir "$INSTDIR"

    ; Remove desktop shortcut
    Delete "$DESKTOP\${PRODUCT_NAME}.lnk"

    ; Remove Start Menu shortcuts
    Delete "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk"
    Delete "$SMPROGRAMS\${PRODUCT_NAME}\README.lnk"
    Delete "$SMPROGRAMS\${PRODUCT_NAME}\Edit Configuration.lnk"
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
        MessageBox MB_OKCANCEL|MB_ICONINFORMATION "${PRODUCT_NAME} is already installed.$\n$\nClick OK to upgrade or Cancel to exit." IDOK upgradeOK
            Abort
        upgradeOK:
    ${EndIf}
FunctionEnd

Function .onInstSuccess
    ; Post-installation message
    ${IfNot} ${FileExists} "$INSTDIR\.env"
        MessageBox MB_OK|MB_ICONINFORMATION "Installation complete!$\n$\nIMPORTANT: Before running ${PRODUCT_NAME}, you must:$\n$\n1. Copy .env.example to .env$\n2. Edit .env with your Canvus credentials$\n$\nThe configuration file is located at:$\n$INSTDIR\.env"
    ${EndIf}
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
