=============================================
CanvusLocalLLM - Installation Guide
=============================================

Thank you for installing CanvusLocalLLM!

This application connects your Canvus collaborative workspace with local AI services,
providing real-time AI assistance with PDF analysis, canvas analysis, and image
generation using embedded multimodal models.

=============================================
QUICK START GUIDE
=============================================

1. CONFIGURE YOUR INSTALLATION

   Before starting the application, you MUST edit the .env configuration file:

   Location: C:\Program Files\CanvusLocalLLM\.env

   Required Settings (MUST be configured):
   - CANVUS_SERVER     : Your Canvus server URL (e.g., https://canvus.yourcompany.com)
   - CANVAS_ID         : The UUID of your canvas to monitor
   - CANVUS_API_KEY    : Your Canvus API key for authentication
   - WEBUI_PWD         : Password for the web interface (choose a strong password)

   Optional Settings:
   - PORT              : Web interface port (default: 3000)
   - CANVAS_NAME       : Human-readable name for logging
   - ALLOW_SELF_SIGNED_CERTS : Set to 'true' for development with self-signed certificates

   Note: You can use username/password authentication instead of API key.
         See the .env file comments for details.


2. VERIFY GPU SETUP (for best performance)

   CanvusLocalLLM uses your GPU for local AI inference. For NVIDIA GPUs:
   - Ensure NVIDIA drivers are installed
   - CUDA support is detected automatically
   - Models download automatically on first run

   CPU fallback is available if no GPU is detected.


3. START THE APPLICATION

   Option A: Double-click the desktop shortcut "CanvusLocalLLM"
   Option B: Start Menu → CanvusLocalLLM → CanvusLocalLLM
   Option C: Run C:\Program Files\CanvusLocalLLM\CanvusAPI-LLM.exe

   On first run:
   - AI models will download automatically (this may take several minutes)
   - Progress will be shown in the console window
   - The web interface will be available at http://localhost:3000


4. ACCESS THE WEB INTERFACE

   Open your browser and navigate to: http://localhost:3000

   Login with the password you set in WEBUI_PWD

   The dashboard shows:
   - Canvas monitoring status
   - Active AI processing tasks
   - GPU metrics and performance
   - Processing history and logs


=============================================
FEATURES & USAGE
=============================================

Once configured and running, CanvusLocalLLM monitors your canvas and automatically
processes AI requests:

1. AI PROMPTS IN NOTES
   - Wrap text in {{ }} to trigger AI processing
   - Example: "{{Summarize the key points from this meeting}}"
   - AI response appears as a new note on the canvas

2. PDF ANALYSIS
   - Upload a PDF to your canvas
   - Add a note with: "{{Analyze this PDF}}"
   - Receives automatic summary and key insights

3. CANVAS ANALYSIS
   - Request: "{{Analyze this canvas}}"
   - Receives overview of all widgets and content structure

4. IMAGE GENERATION
   - Request: "{{Generate image: a futuristic cityscape}}"
   - AI-generated image appears on canvas

5. HANDWRITING RECOGNITION (requires Google Vision API key)
   - Upload handwritten notes image
   - Request: "{{Read this handwriting}}"
   - Receives transcribed text


=============================================
DIRECTORY STRUCTURE
=============================================

Installation Directory: C:\Program Files\CanvusLocalLLM\

Key Files:
- CanvusAPI-LLM.exe : Main application executable
- .env              : Configuration file (EDIT THIS FIRST!)
- README.txt        : This file
- uninstall.exe     : Uninstaller

Directories:
- models\           : AI model storage (auto-downloaded)
- downloads\        : Temporary file downloads
- logs\             : Application logs
- data\             : Persistent data storage


=============================================
TROUBLESHOOTING
=============================================

Problem: Application won't start
Solution:
  - Check .env file is configured correctly
  - Verify CANVUS_SERVER is accessible
  - Check logs in logs\ directory

Problem: "Connection failed" errors
Solution:
  - Verify CANVUS_SERVER URL is correct
  - Check CANVUS_API_KEY is valid
  - If using self-signed certs, set ALLOW_SELF_SIGNED_CERTS=true

Problem: AI processing is slow
Solution:
  - Check GPU is detected (shown in web dashboard)
  - Ensure NVIDIA drivers are up to date
  - CPU fallback is slower but functional

Problem: Models not downloading
Solution:
  - Check internet connection
  - Verify firewall allows outbound connections
  - Check available disk space (models can be several GB)

Problem: Web interface not accessible
Solution:
  - Verify PORT setting in .env
  - Check firewall allows local connections
  - Try http://127.0.0.1:3000 instead of localhost


=============================================
UNINSTALLING
=============================================

To remove CanvusLocalLLM:

1. Windows Settings → Apps → Apps & features
2. Find "CanvusLocalLLM" and click Uninstall
3. Follow the uninstaller prompts
4. Choose whether to preserve logs and downloaded files


=============================================
ADVANCED CONFIGURATION
=============================================

For advanced configuration options (cloud API fallback, custom model paths,
performance tuning), see ADVANCED_CONFIG.md in the GitHub repository:

https://github.com/canvus/CanvusLocalLLM


=============================================
SUPPORT & DOCUMENTATION
=============================================

Documentation: https://github.com/canvus/CanvusLocalLLM
Issue Tracker: https://github.com/canvus/CanvusLocalLLM/issues

For additional help, please visit the project repository.


=============================================
VERSION & LICENSE
=============================================

Version: See installer version
License: See LICENSE.txt

Copyright (c) 2025 CanvusLocalLLM

=============================================
