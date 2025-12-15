# Product Mission

## Pitch
CanvusAPI-LLMDemo is an intelligent integration service that helps teams, enterprises, and power users leverage AI capabilities within their Canvus collaborative workspaces by providing a batteries-included, privacy-focused bridge to any LLM provider—whether embedded local multimodal models (simple installation), cloud-based (OpenAI, Azure), or self-hosted local servers.

## Users

### Primary Customers
- **Enterprise Organizations**: Companies requiring private AI deployments that keep sensitive data on-premises while maintaining full control over AI infrastructure and compliance
- **Collaborative Teams**: Teams using Canvus for project work, brainstorming, and knowledge management who want AI to augment their creative and analytical workflows without complex setup
- **Power Users**: Individual professionals managing complex information landscapes who need AI assistance integrated into their workspace tools

### User Personas

**Enterprise IT Administrator** (35-50)
- **Role:** Infrastructure Manager / DevOps Lead
- **Context:** Responsible for deploying AI tools that meet security, compliance, and data sovereignty requirements
- **Pain Points:** Cloud AI services raise data privacy concerns; need to maintain full control over proprietary information while providing teams with modern AI capabilities; complex installation processes create adoption barriers; Windows deployment requirements; configuration complexity
- **Goals:** Deploy self-hosted AI solutions that integrate seamlessly with existing collaboration tools, maintain audit trails, support various open-source LLM models, work out-of-the-box without Docker or WSL, run natively on Windows/Linux/macOS, and configure through simple text files without wizards or GUIs

**Product Manager** (28-45)
- **Role:** Product/Project Manager at tech company or agency
- **Context:** Manages multiple projects on Canvus with extensive documentation, PDFs, design artifacts, and team notes
- **Pain Points:** Drowning in documentation; needs to quickly synthesize information from PDFs, extract insights from canvas discussions, and generate content without context-switching between tools; doesn't want to configure complex AI infrastructure or manage separate services; intimidated by command-line configuration
- **Goals:** Get AI-powered summaries, analysis, and content generation directly within the workspace where information already lives, with minimal setup time, clear configuration guidance, and installer-based deployment

**Independent Consultant** (30-55)
- **Role:** Strategy Consultant / Researcher / Analyst
- **Context:** Works with multiple clients managing complex research and deliverables on Canvus
- **Pain Points:** Expensive OpenAI API costs for heavy usage; wants flexibility to use different AI providers based on task complexity and budget; concerned about client data privacy when using cloud services; needs solutions that work across different platforms; wants control over when models download
- **Goals:** Start with embedded local AI immediately (no API keys, no costs, no separate services), edit configuration files directly for full control, choose when to download models, then seamlessly switch between local LLMs for routine tasks and premium cloud models for complex analysis, all within the same workflow

## The Problem

### AI Integration Fragmentation
Modern knowledge workers use collaborative tools like Canvus but must constantly context-switch to separate AI interfaces (ChatGPT web, API tools, etc.) to analyze content, generate summaries, or create images. This workflow interruption destroys productivity and makes AI feel like a separate tool rather than an integrated capability.

**Our Solution:** Embed AI capabilities directly into the Canvus workspace through a simple prompt syntax (`{{ }}`), enabling users to invoke AI without leaving their collaborative environment.

### Complex AI Infrastructure Setup
Users want AI capabilities but face high barriers to entry: cloud services require API keys and raise privacy concerns, while self-hosted LLMs demand technical expertise to download, configure, and maintain separate server infrastructure (Docker, WSL on Windows, Python environments). Even when tools bundle installers, they often force configuration during installation or use complex configuration wizards.

**Our Solution:** Provide a simple installer (Windows .exe, Linux .deb/.tar.gz) that installs binaries without forcing configuration, creates a well-commented example configuration file, and downloads models on first run—separating installation from configuration and giving users full visibility and control over the setup process.

### Data Privacy and Provider Lock-in
Enterprise users face a dilemma: cloud AI services (OpenAI, Azure) provide powerful capabilities but create data privacy risks and vendor lock-in, while self-hosted LLMs offer control but lack easy integration with collaboration tools.

**Our Solution:** Provide a unified integration that works seamlessly with embedded local models (default), OpenAI, Azure OpenAI, or any local LLM server (LLaMA, Ollama, LM Studio), giving organizations complete flexibility to balance capability, cost, and privacy based on their requirements.

### Complex Multimodal Workflows
Users need to perform diverse AI tasks—text generation, PDF analysis, canvas synthesis, image generation, handwriting recognition—but these typically require different tools and workflows.

**Our Solution:** Support multiple AI modalities through a single integration powered by embedded llama.cpp (with future stable-diffusion.cpp and whisper.cpp), allowing users to analyze PDFs, process vision tasks, recognize handwriting, and synthesize entire canvas workspaces using consistent interaction patterns.

## Differentiators

### Simple Installation with User Control
Unlike solutions with complex installers or forced configuration wizards, we provide a clean 4-step installer that installs binaries, creates an example configuration file, and opens it for editing—without forcing users through configuration during installation or hiding settings. This results in users having full visibility into configuration, the ability to version-control their .env files, and complete control over when models download (first application run, not installer run).

### Zero-Config AI with Provider Flexibility
Unlike solutions requiring complex setup or cloud API keys to get started, we embed llama.cpp runtime with automatic LLaVA model provisioning on first run, while still supporting OpenAI, Azure OpenAI, and any local LLM server with OpenAI-compatible APIs. This results in organizations starting with privacy-first defaults and choosing the right AI provider for each use case—embedded local models for immediate use and sensitive data, cloud models for complex reasoning—all configured through a single well-commented .env file.

### Installer-Based Deployment
Unlike manual setup processes requiring multiple steps, we provide native installers (NSIS for Windows, .deb for Debian/Ubuntu, .tar.gz for other Linux distributions) that handle binaries, directory structure, optional service creation, and example configuration generation. This results in IT administrators deploying with familiar installation methods (double-click .exe, `apt install`, or extract tarball) without Docker, Python, or manual file placement.

### Privacy-First Architecture
Unlike cloud-only AI solutions, our embedded architecture with llama.cpp allows enterprises to run everything on-premises by default with complete control. This results in complete data sovereignty, meeting regulatory requirements while still providing teams with modern AI capabilities, with optional cloud provider integration when needed.

### Native Canvus Integration
Unlike standalone AI tools that require export/import workflows, we monitor Canvus workspaces in real-time and inject AI results directly as native canvas elements. This results in AI feeling like a natural extension of the collaboration environment rather than an external service.

### Modular Multimodal Ecosystem
Unlike monolithic AI integrations, we leverage the proven llama.cpp ecosystem (same foundation as Ollama and LocalAI) with modular components—llama.cpp for text and vision, stable-diffusion.cpp for image generation (future), whisper.cpp for audio transcription (future). This results in specialized, high-performance capabilities that can be added incrementally without architectural rewrites, all with native Windows/Linux/macOS support.

## Key Features

### Core Features
- **Simple Installation:** Native installers for Windows (.exe), Linux (.deb, .tar.gz) with 4-step wizard that installs binaries, creates example configuration, and opens config file in system editor—no forced configuration during installation
- **Embedded Local AI:** Direct integration of llama.cpp runtime via Go CGo bindings with automatic LLaVA multimodal model provisioning (~4-8GB) on first run, providing immediate text and vision AI capabilities with complete privacy and no separate services
- **Configuration File-Based Setup:** Well-commented .env.example file with all settings explained, model selection options (LLaVA 7B/13B, Llama 3.2-Vision), privacy modes (local-only, hybrid, cloud-preferred), and performance tuning—users edit one file and run the application
- **Real-time AI Processing:** Monitor Canvus workspaces continuously and automatically process any content enclosed in `{{ }}` syntax, delivering AI responses as new notes positioned intelligently on the canvas
- **Flexible LLM Support:** Connect to embedded llama.cpp (default), OpenAI, Azure OpenAI, or custom LLM servers (Ollama, LLaMA, LM Studio) with configurable endpoints, model selection, and token limits for each operation type
- **PDF Analysis:** Upload documents to canvas, trigger analysis with a custom menu icon, and receive intelligent summaries that extract key insights while respecting configurable token limits

### Collaboration Features
- **Canvas Analysis:** Generate comprehensive overviews of entire canvas workspaces that understand spatial relationships, content clustering, and thematic connections between notes, images, and documents
- **Custom Menu Integration:** Deploy AI triggers through Canvus custom menus, allowing teams to access PDF analysis and canvas synthesis capabilities with simple icon placement
- **Model Management:** Download, switch, and update GGUF models through integrated interface without manual configuration or command-line operations

### Advanced Features
- **First-Run Model Download:** Automatic model download on first application run with progress bar, resumable downloads, SHA256 verification, and helpful error messages—users see exactly what's happening and can retry if needed
- **Privacy Mode Selection:** Configure privacy modes via .env file: local-only (all processing on-device), hybrid (local first with cloud fallback), or cloud-preferred (use OpenAI/Azure when faster)—clear control over data sovereignty
- **Multimodal Vision:** Embedded LLaVA models support vision capabilities for analyzing images, charts, diagrams, and visual content directly within canvas workspaces using the same runtime as text generation
- **Image Generation (Future):** Planned integration of stable-diffusion.cpp for local image generation from text prompts, with results automatically placed on the canvas alongside cloud provider support (OpenAI DALL-E, Azure)
- **Handwriting Recognition:** Integrate Google Vision API to extract text from handwritten notes and sketches, making analog content searchable and AI-processable
- **Smart Fallback:** Automatically fall back to OpenAI/Azure providers when embedded runtime is unavailable or specific capabilities (cloud image generation) require cloud models
- **Optional Service Creation:** Installer optionally creates Windows Service or systemd unit for running CanvusLocalLLM as background service with automatic startup
- **Configurable Processing:** Fine-tune token limits, timeout durations, concurrency limits, and retry behavior to balance cost, speed, and quality for specific organizational needs
- **Production-Grade Operations:** Robust error handling with retry mechanisms, comprehensive logging (console + file), graceful degradation, and secure API key management
