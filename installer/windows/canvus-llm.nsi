; =============================================
; CanvusLocalLLM - NSIS Installer Script
; =============================================
; Creates a Windows installer with:
; - License agreement page
; - Installation directory selection
; - Binary and configuration file installation
; - Optional bundled model support
; - Desktop and Start Menu shortcuts
; =============================================

!include "MUI2.nsh"
!include "FileFunc.nsh"

; =============================================
; Configuration
; =============================================

; Application Info
!define PRODUCT_NAME "CanvusLocalLLM"
!define PRODUCT_VERSION "${VERSION}"
!define PRODUCT_PUBLISHER "CanvusLocalLLM"
!define PRODUCT_WEB_SITE "https://github.com/canvus/CanvusLocalLLM"
!define PRODUCT_EXE "CanvusAPI-LLM.exe"
!define PRODUCT_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"

; Installer Info
Name "${PRODUCT_NAME} ${PRODUCT_VERSION}"
OutFile "CanvusLocalLLM-Setup.exe"
InstallDir "$PROGRAMFILES64\${PRODUCT_NAME}"
InstallDirRegKey HKLM "${PRODUCT_UNINST_KEY}" "InstallDir"
RequestExecutionLevel admin

; Interface Settings
!define MUI_ABORTWARNING
!define MUI_ICON "${NSISDIR}\Contrib\Graphics\Icons\modern-install.ico"
!define MUI_UNICON "${NSISDIR}\Contrib\Graphics\Icons\modern-uninstall.ico"
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_BITMAP "${NSISDIR}\Contrib\Graphics\Header\nsis.bmp"
!define MUI_WELCOMEFINISHPAGE_BITMAP "${NSISDIR}\Contrib\Graphics\Wizard\win.bmp"

; =============================================
; Pages
; =============================================

; Welcome Page
!insertmacro MUI_PAGE_WELCOME

; License Page
!insertmacro MUI_PAGE_LICENSE "LICENSE.txt"

; Directory Page
!insertmacro MUI_PAGE_DIRECTORY

; Installation Page
!insertmacro MUI_PAGE_INSTFILES

; Finish Page
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXE}"
!define MUI_FINISHPAGE_RUN_TEXT "Start ${PRODUCT_NAME} now"
!define MUI_FINISHPAGE_SHOWREADME "$INSTDIR\README.txt"
!define MUI_FINISHPAGE_SHOWREADME_TEXT "Show installation instructions"
!insertmacro MUI_PAGE_FINISH

; Uninstaller Pages
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; =============================================
; Languages
; =============================================

!insertmacro MUI_LANGUAGE "English"

; =============================================
; Installer Sections
; =============================================

Section "Main Application" SecMain
    SectionIn RO  ; Required section

    SetOutPath "$INSTDIR"
    SetOverwrite on

    ; Install main executable
    File /oname=${PRODUCT_EXE} "..\..\CanvusAPI-LLM.exe"

    ; Install configuration template
    File "README.txt"

    ; Create .env from template if it doesn't exist
    IfFileExists "$INSTDIR\.env" skip_env 0
        File /oname=.env "..\..\.env.example"
    skip_env:

    ; Install bundled model if available (optional)
    ; The installer builder should place model files in installer/windows/models/
    IfFileExists "models\*.*" 0 skip_models
        SetOutPath "$INSTDIR\models"
        File /r "models\*.*"
    skip_models:

    ; Create data directories
    SetOutPath "$INSTDIR"
    CreateDirectory "$INSTDIR\downloads"
    CreateDirectory "$INSTDIR\logs"
    CreateDirectory "$INSTDIR\data"

    ; Create Start Menu shortcuts
    CreateDirectory "$SMPROGRAMS\${PRODUCT_NAME}"
    CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk" "$INSTDIR\${PRODUCT_EXE}"
    CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}\Uninstall.lnk" "$INSTDIR\uninstall.exe"
    CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}\README.lnk" "$INSTDIR\README.txt"

    ; Create Desktop shortcut
    CreateShortcut "$DESKTOP\${PRODUCT_NAME}.lnk" "$INSTDIR\${PRODUCT_EXE}"

    ; Write uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"

    ; Write registry keys for Add/Remove Programs
    WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayName" "${PRODUCT_NAME}"
    WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "UninstallString" "$INSTDIR\uninstall.exe"
    WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayIcon" "$INSTDIR\${PRODUCT_EXE}"
    WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayVersion" "${PRODUCT_VERSION}"
    WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
    WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "URLInfoAbout" "${PRODUCT_WEB_SITE}"
    WriteRegDWORD HKLM "${PRODUCT_UNINST_KEY}" "NoModify" 1
    WriteRegDWORD HKLM "${PRODUCT_UNINST_KEY}" "NoRepair" 1

    ; Calculate installed size
    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD HKLM "${PRODUCT_UNINST_KEY}" "EstimatedSize" "$0"

SectionEnd

; =============================================
; Descriptions
; =============================================

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
    !insertmacro MUI_DESCRIPTION_TEXT ${SecMain} "Installs ${PRODUCT_NAME} core application files, bundled AI models, and configuration templates."
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; =============================================
; Uninstaller Section
; =============================================

Section "Uninstall"

    ; Remove files
    Delete "$INSTDIR\${PRODUCT_EXE}"
    Delete "$INSTDIR\README.txt"
    Delete "$INSTDIR\.env"
    Delete "$INSTDIR\uninstall.exe"

    ; Remove model files (if exists)
    RMDir /r "$INSTDIR\models"

    ; Remove data directories (prompt user for data preservation)
    MessageBox MB_YESNO|MB_ICONQUESTION "Do you want to remove logs and downloaded files? Click No to preserve your data." IDNO skip_data
        RMDir /r "$INSTDIR\downloads"
        RMDir /r "$INSTDIR\logs"
        RMDir /r "$INSTDIR\data"
    skip_data:

    ; Remove shortcuts
    Delete "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk"
    Delete "$SMPROGRAMS\${PRODUCT_NAME}\Uninstall.lnk"
    Delete "$SMPROGRAMS\${PRODUCT_NAME}\README.lnk"
    RMDir "$SMPROGRAMS\${PRODUCT_NAME}"
    Delete "$DESKTOP\${PRODUCT_NAME}.lnk"

    ; Remove installation directory
    RMDir "$INSTDIR"

    ; Remove registry keys
    DeleteRegKey HKLM "${PRODUCT_UNINST_KEY}"

    ; Cleanup message
    MessageBox MB_OK "${PRODUCT_NAME} has been uninstalled successfully."

SectionEnd

; =============================================
; Installer Functions
; =============================================

Function .onInit
    ; Check for admin rights
    UserInfo::GetAccountType
    Pop $0
    ${If} $0 != "admin"
        MessageBox MB_ICONSTOP "Administrator rights required!"
        SetErrorLevel 740
        Quit
    ${EndIf}
FunctionEnd

Function .onInstSuccess
    MessageBox MB_OK "Installation completed successfully!$\r$\n$\r$\nIMPORTANT: Before running ${PRODUCT_NAME}:$\r$\n$\r$\n1. Edit the .env file in $INSTDIR$\r$\n2. Configure your Canvus server connection$\r$\n3. Set your web UI password$\r$\n$\r$\nSee README.txt for detailed setup instructions."
FunctionEnd
