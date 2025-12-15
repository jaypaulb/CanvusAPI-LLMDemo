# Product Mission

## Pitch
CanvusAPI-LLMDemo is an intelligent integration service that helps teams, enterprises, and power users leverage AI capabilities within their Canvus collaborative workspaces by providing a batteries-included, privacy-focused bridge to any LLM provider—whether bundled local multimodal models (zero-config), cloud-based (OpenAI, Azure), or self-hosted local servers.

## Users

### Primary Customers
- **Enterprise Organizations**: Companies requiring private AI deployments that keep sensitive data on-premises while maintaining full control over AI infrastructure and compliance
- **Collaborative Teams**: Teams using Canvus for project work, brainstorming, and knowledge management who want AI to augment their creative and analytical workflows without complex setup
- **Power Users**: Individual professionals managing complex information landscapes who need AI assistance integrated into their workspace tools

### User Personas

**Enterprise IT Administrator** (35-50)
- **Role:** Infrastructure Manager / DevOps Lead
- **Context:** Responsible for deploying AI tools that meet security, compliance, and data sovereignty requirements
- **Pain Points:** Cloud AI services raise data privacy concerns; need to maintain full control over proprietary information while providing teams with modern AI capabilities; complex installation processes create adoption barriers
- **Goals:** Deploy self-hosted AI solutions that integrate seamlessly with existing collaboration tools, maintain audit trails, support various open-source LLM models, and work out-of-the-box without extensive configuration

**Product Manager** (28-45)
- **Role:** Product/Project Manager at tech company or agency
- **Context:** Manages multiple projects on Canvus with extensive documentation, PDFs, design artifacts, and team notes
- **Pain Points:** Drowning in documentation; needs to quickly synthesize information from PDFs, extract insights from canvas discussions, and generate content without context-switching between tools; doesn't want to configure complex AI infrastructure
- **Goals:** Get AI-powered summaries, analysis, and content generation directly within the workspace where information already lives, with minimal setup time

**Independent Consultant** (30-55)
- **Role:** Strategy Consultant / Researcher / Analyst
- **Context:** Works with multiple clients managing complex research and deliverables on Canvus
- **Pain Points:** Expensive OpenAI API costs for heavy usage; wants flexibility to use different AI providers based on task complexity and budget; concerned about client data privacy when using cloud services
- **Goals:** Start with bundled local AI immediately (no API keys, no costs), then seamlessly switch between local LLMs for routine tasks and premium cloud models for complex analysis, all within the same workflow

## The Problem

### AI Integration Fragmentation
Modern knowledge workers use collaborative tools like Canvus but must constantly context-switch to separate AI interfaces (ChatGPT web, API tools, etc.) to analyze content, generate summaries, or create images. This workflow interruption destroys productivity and makes AI feel like a separate tool rather than an integrated capability.

**Our Solution:** Embed AI capabilities directly into the Canvus workspace through a simple prompt syntax (`{{ }}`), enabling users to invoke AI without leaving their collaborative environment.

### Complex AI Infrastructure Setup
Users want AI capabilities but face high barriers to entry: cloud services require API keys and raise privacy concerns, while self-hosted LLMs demand technical expertise to download, configure, and maintain separate server infrastructure.

**Our Solution:** Bundle a multimodal LLM runtime (Ollama) with the application that installs and configures automatically during setup, providing immediate AI capabilities with zero configuration while preserving the flexibility to connect external providers when needed.

### Data Privacy and Provider Lock-in
Enterprise users face a dilemma: cloud AI services (OpenAI, Azure) provide powerful capabilities but create data privacy risks and vendor lock-in, while self-hosted LLMs offer control but lack easy integration with collaboration tools.

**Our Solution:** Provide a unified integration that works seamlessly with bundled local models (default), OpenAI, Azure OpenAI, or any local LLM server (LLaMA, Ollama, LM Studio), giving organizations complete flexibility to balance capability, cost, and privacy based on their requirements.

### Complex Multimodal Workflows
Users need to perform diverse AI tasks—text generation, PDF analysis, canvas synthesis, image generation, handwriting recognition—but these typically require different tools and workflows.

**Our Solution:** Support multiple AI modalities through a single integration, allowing users to analyze PDFs, generate images, recognize handwriting, and synthesize entire canvas workspaces using consistent interaction patterns.

## Differentiators

### Zero-Config AI with Provider Flexibility
Unlike solutions requiring complex setup or cloud API keys, we bundle a local multimodal LLM that installs automatically and works immediately, while still supporting OpenAI, Azure OpenAI, and any local LLM server with OpenAI-compatible APIs. This results in organizations starting with privacy-first defaults and choosing the right AI provider for each use case—bundled local models for immediate use and sensitive data, cloud models for complex reasoning—without changing their workflow or learning new tools.

### Batteries-Included Deployment
Unlike traditional self-hosted AI requiring separate infrastructure management, our installation script automatically downloads, configures, and runs Ollama with a small multimodal model (~4-7GB). This results in users having fully functional AI capabilities within minutes of installation, with no Docker, no Python environments, and no manual configuration required.

### Privacy-First Architecture
Unlike cloud-only AI solutions, our self-hosted architecture with bundled local models allows enterprises to run everything on-premises by default. This results in complete data sovereignty, meeting regulatory requirements while still providing teams with modern AI capabilities, with optional cloud provider integration when needed.

### Native Canvus Integration
Unlike standalone AI tools that require export/import workflows, we monitor Canvus workspaces in real-time and inject AI results directly as native canvas elements. This results in AI feeling like a natural extension of the collaboration environment rather than an external service.

### Multimodal Specialization
Unlike generic AI integrations, we provide specialized handlers for different content types—optimized token limits for PDFs vs. notes, canvas-aware analysis that understands spatial relationships, and integrated image generation with bundled vision models. This results in better quality outputs tailored to how information actually exists in collaborative workspaces.

## Key Features

### Core Features
- **Bundled Local AI:** Automatic installation and configuration of Ollama with small multimodal LLM (LLaVA 7B or Llama 3.2-Vision), providing immediate AI capabilities with zero configuration and complete privacy
- **Real-time AI Processing:** Monitor Canvus workspaces continuously and automatically process any content enclosed in `{{ }}` syntax, delivering AI responses as new notes positioned intelligently on the canvas
- **Flexible LLM Support:** Connect to bundled local AI (default), OpenAI, Azure OpenAI, or custom LLM servers (LLaMA, Ollama, LM Studio) with configurable endpoints, model selection, and token limits for each operation type
- **PDF Analysis:** Upload documents to canvas, trigger analysis with a custom menu icon, and receive intelligent summaries that extract key insights while respecting configurable token limits

### Collaboration Features
- **Canvas Analysis:** Generate comprehensive overviews of entire canvas workspaces that understand spatial relationships, content clustering, and thematic connections between notes, images, and documents
- **Custom Menu Integration:** Deploy AI triggers through Canvus custom menus, allowing teams to access PDF analysis and canvas synthesis capabilities with simple icon placement
- **Model Management:** Download, switch, and update LLM models through integrated interface without manual configuration or command-line operations

### Advanced Features
- **Multimodal Vision:** Bundled models support vision capabilities for analyzing images, charts, diagrams, and visual content directly within canvas workspaces
- **Image Generation:** Support both OpenAI DALL-E (2 and 3) and Azure OpenAI for generating images directly from text prompts, with results automatically placed on the canvas
- **Handwriting Recognition:** Integrate Google Vision API to extract text from handwritten notes and sketches, making analog content searchable and AI-processable
- **Smart Fallback:** Automatically fall back to OpenAI/Azure providers when bundled LLM is unavailable or specific capabilities (image generation) require cloud models
- **Configurable Processing:** Fine-tune token limits, timeout durations, concurrency limits, and retry behavior to balance cost, speed, and quality for specific organizational needs
- **Production-Grade Operations:** Robust error handling with retry mechanisms, comprehensive logging (console + file), graceful degradation, and secure API key management
