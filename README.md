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
