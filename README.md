# CanvusAPI-LLMDemo

An intelligent integration between Canvus collaborative workspaces and AI services that enables real-time AI-powered interactions within your Canvus environment.

## Features

- **Real-time AI Processing**: Monitors Canvus workspaces and processes content enclosed in double curly braces `{{ }}` using ChatGPT
- **Multiple AI Capabilities**:
  - Text Analysis and Response
  - PDF Document Summarization
  - Canvas Content Analysis
  - Image Generation
  - Handwriting Recognition (via Google Vision API)

## Prerequisites

- A Canvus Server instance
- OpenAI API key
- Google Vision API key (optional, for handwriting recognition)
- Go programming environment

## Setup

1. Clone this repository
2. Copy `example.env` to `.env` and configure the following variables:
   ```
   CANVUS_SERVER=https://your-canvus-server.com
   CANVAS_NAME=YOUR_CANVAS_NAME
   CANVAS_ID=your-canvas-id
   OPENAI_API_KEY=your-openai-key
   GOOGLE_VISION_API_KEY=your-google-vision-key
   CANVUS_API_KEY=your-canvus-api-key
   ALLOW_SELF_SIGNED_CERTS=false  # Set to true only for development/testing with self-signed certificates
   ```

   **OpenAI Model Configuration**:
   The application uses different OpenAI models for different tasks:
   - `OPENAI_NOTE_MODEL`: For processing notes and basic AI interactions (default: gpt-3.5-turbo)
   - `OPENAI_CANVAS_MODEL`: For analyzing entire canvas content (default: gpt-4)
   - `OPENAI_PDF_MODEL`: For PDF document analysis (default: gpt-4)

   **OpenAI Token Limits**:
   The application allows you to configure token limits for different operations:
   - `OPENAI_PDF_PRECIS_TOKENS`: Token limit for PDF analysis (default: 1000)
   - `OPENAI_CANVAS_PRECIS_TOKENS`: Token limit for canvas analysis (default: 600)
   - `OPENAI_NOTE_RESPONSE_TOKENS`: Token limit for note responses (default: 400)
   - `OPENAI_IMAGE_ANALYSIS_TOKENS`: Token limit for image descriptions (default: 16384)
   - `OPENAI_ERROR_RESPONSE_TOKENS`: Token limit for error messages (default: 200)
   - `OPENAI_PDF_CHUNK_SIZE_TOKENS`: Size of individual PDF chunks (default: 20000)
   - `OPENAI_PDF_MAX_CHUNKS_TOKENS`: Maximum number of PDF chunks to process (default: 10)
   - `OPENAI_PDF_SUMMARY_RATIO`: Target ratio of summary to original length (default: 0.3)

   You can adjust these values based on your needs. For example:
   ```
   OPENAI_PDF_PRECIS_TOKENS=2000      # Increase for more detailed PDF analysis
   OPENAI_CANVAS_PRECIS_TOKENS=800    # Increase for more detailed canvas analysis
   OPENAI_NOTE_RESPONSE_TOKENS=600    # Increase for longer note responses
   ```

   Note: Higher token limits will result in more detailed responses but may increase processing time and API costs.

3. Install dependencies:
   ```bash
   go mod download
   ```

4. Build the executable:
   ```bash
   go build -o CanvusAPI-LLM.exe .
   ```

5. Run the executable:
   On Windows:
   ```bash
   CanvusAPI-LLM.exe
   ```
   On Unix-like systems:
   ```bash
   ./CanvusAPI-LLM.exe
   ```

## Pre-built Releases

If you prefer not to build the program from source and just want to run it, you can download the pre-built binaries and the example environment file:

### Windows
- **Binary**: [CanvusAPI-LLM.exe](https://github.com/jaypaulb/CanvusAPI-LLMDemo/releases/latest/download/CanvusAPI-LLM.exe)
- **example.env**: [Download example.env](https://github.com/jaypaulb/CanvusAPI-LLMDemo/raw/main/example.env)

### Linux (amd64)
- **Binary**: [canvusapi-linux-amd64](https://github.com/jaypaulb/CanvusAPI-LLMDemo/releases/latest/download/canvusapi-linux-amd64)
- **example.env**: [Download example.env](https://github.com/jaypaulb/CanvusAPI-LLMDemo/raw/main/example.env)

### Deployment Steps

1. Download the appropriate binary for your system (Windows or Linux)
2. Download the `example.env` file
3. Place both files in the same directory
4. Rename `example.env` to `.env`
5. Update the details in the `.env` file with your configuration
6. If connecting to a server with a self-signed certificate:
   - Set `ALLOW_SELF_SIGNED_CERTS=true` in your `.env` file
   - Note: This is not recommended for production environments

#### Linux-specific Steps
1. Make the binary executable:
   ```bash
   chmod +x canvusapi-linux-amd64
   ```
2. Run the binary:
   ```bash
   ./canvusapi-linux-amd64
   ```

#### Windows-specific Steps
1. Run the executable:
   ```bash
   CanvusAPI-LLM.exe
   ```

## Usage

1. **Basic AI Interaction**:
   - Create a note in your Canvus workspace
   - Type your prompt inside double curly braces: `{{What is the capital of France?}}`
   - The system will process the prompt and create a new note with the AI response

2. **PDF Analysis**:
   - Upload a PDF to your canvas
   - Place the AI_Icon_PDF_Precis on the PDF
   - The system will analyze and summarize the PDF content

3. **Canvas Analysis**:
   - Place the AI_Icon_Canvus_Precis on your canvas
   - The system will analyze all content and relationships between items
   - Receive an overview and insights about your workspace

4. **Image Generation**:
   - Include an image generation prompt in your note: `{{Generate an image of a sunset}}`
   - The system will create and place the generated image on your canvas

5. **Custom Menu Integration**:
   The application provides two special icons for your Canvus custom menu:
   - `AI_Icon_PDF_Precis`: Creates a PDF analysis trigger
   - `AI_Icon_Canvus_Precis`: Creates a canvas analysis trigger

   To set up the custom menu:
   1. Navigate to your Canvus custom menu settings
   2. Add the icons from the `icons-for-custom-menu` directory:
      - Use the icons in the root directory for the menu entries
      - Use the icons in the `Content` subdirectory for the content triggers
   3. When users click these icons in the custom menu, they can:
      - Place the PDF analysis trigger on any PDF to generate a summary
      - Place the canvas analysis trigger on the background to analyze the entire workspace

   **Important Notes**:
   - The canvas analysis trigger must be placed on the background to work
   - You can temporarily store the triggers on notes until you're ready to use them
   - The icons are scaled to 33% of their original size when placed on the canvas

   Example `menu.yml` configuration:
   ```yaml
   items:
     - tooltip: 'AI PDF Precis Helper'
       icon: 'icons/AI_Icon_PDF_Precis.png'
       actions:
         - name: 'create'
           parameters:
             type: 'image'
             source: 'content/AI_Icon_PDF_Precis.png'
             scale: 0.33

     - tooltip: 'AI Canvus Precis Helper'
       icon: 'icons/AI_Icon_Canvus_Precis.png'
       actions:
         - name: 'create'
           parameters:
             type: 'image'
             source: 'content/AI_Icon_Canvus_Precis.png'
             scale: 0.33
   ```

Once compiled, simply run the executable as described. If you prefer not to build from source, you can download the precompiled `CanvusAPI-LLM.exe` from the GitHub releases.

## Error Handling

- The system includes robust error handling and retry mechanisms
- Processing status is displayed through color-coded notes
- Failed operations are logged with detailed error messages
- SSL/TLS connection errors are clearly reported in logs

## Logging

Logs are stored in `app.log` with detailed information about:
- System operations
- API interactions
- Error messages
- Processing status
- SSL/TLS connection status and warnings

## Security

- API keys are stored securely in the `.env` file
- The system supports secure connections to the Canvus server
- Web interface is protected by authentication
- SSL/TLS certificate validation is enabled by default
- Self-signed certificate support is available but not recommended for production
- Warning messages are logged when SSL verification is disabled

### SSL/TLS Configuration

The application supports two SSL/TLS modes:

1. **Secure Mode (Default)**
   - SSL certificate validation is enabled
   - Recommended for production environments
   - Ensures secure communication with the server
   - Validates server certificates against trusted CAs

2. **Development Mode (Self-signed Certificates)**
   - Enabled by setting `ALLOW_SELF_SIGNED_CERTS=true`
   - Disables SSL certificate validation
   - Useful for development/testing environments
   - **Security Risks**:
     - Vulnerable to man-in-the-middle attacks
     - Cannot verify server identity
     - Not recommended for production use
     - Warning messages are logged when enabled

## Contributing

Contributions are welcome! Please feel free to submit pull requests or create issues for bugs and feature requests.

## License

This project is proprietary software. All rights reserved.