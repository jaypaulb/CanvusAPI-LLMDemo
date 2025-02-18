# CanvusAPI-LLMDemo

An intelligent integration between Canvus collaborative workspaces and AI services that enables real-time AI-powered interactions within your Canvus environment.

## Features

- **Real-time AI Processing**: Monitors Canvus workspaces and processes content enclosed in double curly braces `{{ }}` using ChatGPT
- **Multiple AI Capabilities**:
  - Text Analysis and Response
  - PDF Document Summarization
  - Canvas Content Analysis
  - Image Generation and Analysis
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
   ```

3. Install dependencies:
   ```bash
   go mod download
   ```

4. Build and run:
   ```bash
   go build
   ./CanvusAPI-LLMDemo
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

## Error Handling

- The system includes robust error handling and retry mechanisms
- Processing status is displayed through color-coded notes
- Failed operations are logged with detailed error messages

## Logging

Logs are stored in `app.log` with detailed information about:
- System operations
- API interactions
- Error messages
- Processing status

## Security

- API keys are stored securely in the `.env` file
- The system supports secure connections to the Canvus server
- Web interface is protected by authentication

## Contributing

Contributions are welcome! Please feel free to submit pull requests or create issues for bugs and feature requests.

## License

This project is proprietary software. All rights reserved.
