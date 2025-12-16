================================================================================
                           CanvusLocalLLM
        AI Integration Service for Canvus Collaborative Workspaces
================================================================================

OVERVIEW
--------
CanvusLocalLLM connects Canvus collaborative workspaces with local AI services
via llama.cpp ecosystem. It monitors canvas widgets in real-time, processes AI
prompts enclosed in {{ }}, and handles PDF analysis, canvas analysis, and
image generation using embedded multimodal models with cloud fallback support.

FEATURES
--------
- Real-time AI Processing: Monitors Canvus workspaces and processes content
  enclosed in double curly braces {{ }} using local or cloud LLMs
- PDF Document Summarization
- Canvas Content Analysis
- Image Generation (DALL-E, Azure OpenAI, or local Stable Diffusion)
- Handwriting Recognition (via Google Vision API)
- Local LLM Support: Compatible with LLaMA, Ollama, LM Studio, and more


================================================================================
                            SYSTEM REQUIREMENTS
================================================================================

Required:
- Canvus Server instance with API access
- Network connectivity to your Canvus server
- One of the following for AI processing:
  - Local LLM server (LLaMA, Ollama, LM Studio, etc.)
  - OpenAI API key
  - Azure OpenAI access

Optional:
- Google Vision API key (for handwriting recognition)


================================================================================
                               QUICK START
================================================================================

1. DOWNLOAD AND EXTRACT
   ---------------------
   Extract the archive to your preferred installation directory.

2. CONFIGURE THE SERVICE
   ----------------------
   a) Copy 'example.env' to '.env' in the same directory as the executable
   b) Edit '.env' with your settings (see CONFIGURATION section below)

3. RUN THE SERVICE
   ----------------
   Windows:
     Double-click CanvusLocalLLM.exe
     OR open Command Prompt and run: CanvusLocalLLM.exe

   Linux:
     chmod +x canvuslocallm
     ./canvuslocallm

   macOS:
     chmod +x canvuslocallm
     ./canvuslocallm

4. VERIFY CONNECTION
   ------------------
   Check the console output for "Connected to Canvus server" message.
   Review app.log for detailed operation logs.


================================================================================
                              CONFIGURATION
================================================================================

Required Settings
-----------------
These must be configured in your .env file:

  CANVUS_SERVER=https://your-canvus-server.com
    The URL of your Canvus server

  CANVAS_NAME=Your Canvas Name
    The display name of the canvas to monitor

  CANVAS_ID=your-canvas-id
    The unique identifier of the canvas (from Canvus API or URL)

  CANVUS_API_KEY=your-canvus-api-key
    Your Canvus API key for authentication

  OPENAI_API_KEY=your-openai-key
    API key for OpenAI services (required even for local LLMs for fallback)


LLM Configuration
-----------------
For local LLM servers:

  BASE_LLM_URL=http://127.0.0.1:1234/v1
    Default endpoint for all LLM operations. Examples:
    - LM Studio: http://localhost:1234/v1
    - Ollama: http://localhost:11434/v1
    - LLaMA.cpp: http://localhost:8080/v1

  TEXT_LLM_URL=
    Optional override for text generation (uses BASE_LLM_URL if not set)

  IMAGE_LLM_URL=https://api.openai.com/v1
    Endpoint for image generation


Model Selection
---------------
  OPENAI_NOTE_MODEL=gpt-4
    Model for processing notes and basic AI interactions

  OPENAI_CANVAS_MODEL=gpt-4
    Model for analyzing entire canvas content

  OPENAI_PDF_MODEL=gpt-4
    Model for PDF document analysis

  IMAGE_GEN_MODEL=dall-e-3
    Model for image generation (dall-e-3, dall-e-2)

For local LLMs, use the model name as configured in your server:
  - llama2, mistral, codellama, etc.


Optional Settings
-----------------
  GOOGLE_VISION_API_KEY=your-key
    Enable handwriting recognition features

  ALLOW_SELF_SIGNED_CERTS=false
    Set to 'true' if using self-signed SSL certificates (NOT recommended
    for production)

  MAX_CONCURRENT=5
    Maximum concurrent AI processing operations

  PROCESSING_TIMEOUT=300
    Timeout in seconds for AI operations

  WEBUI_PWD=your-password
    Password for the web interface


================================================================================
                                 USAGE
================================================================================

Basic AI Interaction
--------------------
1. Create a note in your Canvus workspace
2. Type your prompt inside double curly braces:
   {{What is the capital of France?}}
3. The service will process the prompt and create a response note

PDF Analysis
------------
1. Upload a PDF document to your canvas
2. Drag the AI_Icon_PDF_Precis onto the PDF
3. The service will analyze and summarize the PDF content

Canvas Analysis
---------------
1. Drag the AI_Icon_Canvus_Precis onto the canvas background
2. The service will analyze all content and relationships
3. A summary note will be created with insights about your workspace

Image Generation
----------------
Include an image generation prompt in your note:
  {{Generate an image of a sunset over mountains}}
The service will create and place the generated image on your canvas


================================================================================
                             TROUBLESHOOTING
================================================================================

"Connection refused" or "Cannot connect to Canvus server"
---------------------------------------------------------
- Verify CANVUS_SERVER URL is correct (include https://)
- Check network connectivity to the server
- Ensure the Canvus API is enabled and accessible
- If using self-signed certificates, set ALLOW_SELF_SIGNED_CERTS=true

"Authentication failed" or "Invalid API key"
--------------------------------------------
- Verify CANVUS_API_KEY is correct
- Check if the API key has permissions for the specified canvas
- Ensure the API key has not expired

"Canvas not found"
------------------
- Verify CANVAS_ID matches the canvas you want to monitor
- Check if you have access permissions to the canvas
- Try copying the canvas ID from the Canvus URL

"LLM connection failed" or "AI service unavailable"
---------------------------------------------------
- For local LLMs: Ensure your LLM server is running
- Verify BASE_LLM_URL points to the correct endpoint
- Check if the model specified in OPENAI_NOTE_MODEL is loaded
- Test your LLM server independently before connecting

"SSL certificate error"
-----------------------
- For development/testing only: Set ALLOW_SELF_SIGNED_CERTS=true
- For production: Install proper SSL certificates on your Canvus server

Service crashes immediately
---------------------------
- Check that .env file exists and is properly formatted
- Ensure all required variables are set
- Review app.log for detailed error messages

Slow or no AI responses
-----------------------
- Check your LLM server performance
- Consider reducing MAX_CONCURRENT if using limited hardware
- Increase AI_TIMEOUT if processing large documents
- Monitor memory usage on the LLM server


================================================================================
                               LOG FILES
================================================================================

The service creates an 'app.log' file in the installation directory containing:
- Connection status and events
- AI processing requests and responses
- Error messages and warnings
- Performance metrics

Review this file when troubleshooting issues.


================================================================================
                             GETTING HELP
================================================================================

Documentation:
  - Full documentation: README.md (included in distribution)
  - Configuration reference: example.env (annotated with all options)

Reporting Issues:
  - GitHub Issues: https://github.com/jaypaulb/CanvusLocalLLM/issues

When reporting issues, please include:
  - Operating system and version
  - CanvusLocalLLM version
  - Relevant portions of app.log (remove sensitive information)
  - Steps to reproduce the issue


================================================================================
                                LICENSE
================================================================================

This project is licensed under the MIT License.
See LICENSE.txt for the full license text.


================================================================================
                             VERSION INFO
================================================================================

CanvusLocalLLM
Copyright (c) 2024-2025 Jaypaul Bridger

For the latest version and updates:
  https://github.com/jaypaulb/CanvusLocalLLM

================================================================================
