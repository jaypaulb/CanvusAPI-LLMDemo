# Bunny Model Documentation

This guide covers the Bunny v1.1 multimodal model used by CanvusLocalLLM for local AI inference. Bunny is an open-source vision-language model that combines text understanding with image analysis capabilities.

## Table of Contents

1. [Model Overview](#model-overview)
2. [Model Specifications](#model-specifications)
3. [Quantization Options](#quantization-options)
4. [Download and Installation](#download-and-installation)
5. [Prompt Engineering](#prompt-engineering)
6. [Vision Capabilities](#vision-capabilities)
7. [GPU Performance Expectations](#gpu-performance-expectations)
8. [Best Practices](#best-practices)
9. [Troubleshooting](#troubleshooting)

---

## Model Overview

### What is Bunny?

Bunny is an open-source multimodal large language model (MLLM) developed by researchers to combine visual understanding with natural language processing. The v1.1 release significantly improves upon earlier versions with:

- **Enhanced vision encoder**: SigLIP-based image encoding for better visual understanding
- **Larger language backbone**: LLaMA-3 8B for improved text generation
- **Training improvements**: Better instruction following and reduced hallucination

### Why Bunny for CanvusLocalLLM?

1. **Multimodal capability**: Analyze canvas images, PDFs, and handwritten content
2. **Open-source**: No API costs, full local privacy
3. **Efficient quantization**: Runs on consumer GPUs (RTX 3060+)
4. **GGUF format**: Compatible with llama.cpp for cross-platform support
5. **Permissive license**: Apache 2.0 allows commercial use

### Model Architecture

```
Input Layer
    ├─ Text Input ────────────────┐
    │                              │
    └─ Image Input ───────────────┤
            │                      │
            ▼                      ▼
    ┌───────────────┐    ┌───────────────┐
    │ SigLIP Vision │    │ LLaMA-3 8B   │
    │   Encoder     │    │ Text Encoder │
    └───────┬───────┘    └───────┬───────┘
            │                    │
            ▼                    ▼
    ┌───────────────────────────────────┐
    │    Cross-Modal Fusion Layer       │
    │  (MLP Projection + Attention)     │
    └───────────────┬───────────────────┘
                    │
                    ▼
    ┌───────────────────────────────────┐
    │    LLaMA-3 8B Decoder            │
    │    (Text Generation)              │
    └───────────────────────────────────┘
                    │
                    ▼
              Generated Text
```

---

## Model Specifications

### Base Model Information

| Property | Value |
|----------|-------|
| **Model Name** | Bunny-v1.1-LLaMA-3-8B-V |
| **Base LLM** | Meta LLaMA-3 8B |
| **Vision Encoder** | SigLIP-SO400M/14@384 |
| **Parameter Count** | ~8 billion |
| **Training Data** | LAION, ShareGPT, custom instruction data |
| **License** | Apache 2.0 |
| **Original Format** | PyTorch (safetensors) |
| **Converted Format** | GGUF (llama.cpp compatible) |

### Context and Token Limits

| Property | Default | Maximum | Notes |
|----------|---------|---------|-------|
| **Context Window** | 2048 | 8192 | Set via `ContextSize` config |
| **Max Output Tokens** | 512 | 4096 | Limited by context window |
| **Image Resolution** | 384x384 | 384x384 | Fixed by vision encoder |
| **Image Tokens** | ~576 | ~576 | Visual tokens added to context |

### Supported Input Types

| Input Type | Support | Notes |
|------------|---------|-------|
| Plain text | Full | Standard LLM text generation |
| Single image + text | Full | Primary multimodal use case |
| Multiple images | Limited | Best results with single image |
| PDF (as images) | Full | Convert PDF pages to images first |
| Handwriting | Good | Better for printed text |

---

## Quantization Options

GGUF quantization reduces model size and VRAM requirements at some quality cost.

### Available Quantizations

| Quantization | File Size | VRAM Usage | Quality | Speed | Recommended For |
|--------------|-----------|------------|---------|-------|-----------------|
| **Q4_K_M** | ~4.5 GB | 6-7 GB | Good | Fastest | RTX 3060 (12GB), everyday use |
| **Q5_K_M** | ~5.5 GB | 7-8 GB | Better | Fast | RTX 4070 (12GB), balanced |
| **Q6_K** | ~6.5 GB | 8-9 GB | Best | Moderate | RTX 4080 (16GB), quality focus |
| **Q8_0** | ~8.5 GB | 10-11 GB | Excellent | Slower | RTX 4090 (24GB), max quality |
| **F16** | ~16 GB | 18+ GB | Perfect | Slowest | Research, reference comparisons |

### Quantization Selection Guide

**Choose Q4_K_M when:**
- You have 8-12GB VRAM
- Speed is more important than quality
- Running multiple concurrent contexts
- First-time setup (easiest to work with)

**Choose Q5_K_M when:**
- You have 12-16GB VRAM
- You need balanced quality and speed
- Running canvas analysis tasks
- Recommended default for most users

**Choose Q6_K or higher when:**
- You have 16+ GB VRAM
- Quality is paramount (OCR, detailed analysis)
- Single-context operation is acceptable
- You notice quality issues with lower quantizations

### VRAM Budget Calculation

```
Total VRAM Required = Model Size + (Context KV Cache * NumContexts) + Workspace

Example (Q5_K_M, 4096 context, 3 contexts):
= 5.5GB + (1.5GB * 3) + 0.5GB
= 5.5GB + 4.5GB + 0.5GB
= 10.5GB VRAM
```

---

## Download and Installation

### Downloading the Model

**Option 1: Hugging Face Hub (Recommended)**

```bash
# Install huggingface-cli if not present
pip install huggingface-hub

# Download Q5_K_M (recommended)
huggingface-cli download BAAI/Bunny-v1.1-Llama-3-8B-V-GGUF \
    bunny-v1.1-llama-3-8b-v-q5_k_m.gguf \
    --local-dir models/ \
    --local-dir-use-symlinks False
```

**Option 2: Direct Download**

```bash
# Q5_K_M (recommended)
wget -O models/bunny-v1.1-llama-3-8b-v-q5_k_m.gguf \
    "https://huggingface.co/BAAI/Bunny-v1.1-Llama-3-8B-V-GGUF/resolve/main/bunny-v1.1-llama-3-8b-v-q5_k_m.gguf"

# Q4_K_M (smaller, faster)
wget -O models/bunny-v1.1-llama-3-8b-v-q4_k_m.gguf \
    "https://huggingface.co/BAAI/Bunny-v1.1-Llama-3-8B-V-GGUF/resolve/main/bunny-v1.1-llama-3-8b-v-q4_k_m.gguf"
```

**Option 3: Auto-Download via Configuration**

Set environment variables in `.env`:

```bash
LLAMA_MODEL_PATH=bunny-v1.1-llama-3-8b-v-q5_k_m.gguf
LLAMA_MODELS_DIR=./models
LLAMA_AUTO_DOWNLOAD=true
LLAMA_MODEL_URL=https://huggingface.co/BAAI/Bunny-v1.1-Llama-3-8B-V-GGUF/resolve/main/bunny-v1.1-llama-3-8b-v-q5_k_m.gguf
```

### Verifying Installation

```bash
# Check model file exists and has correct size
ls -lh models/*.gguf

# Expected output:
# -rw-r--r-- 1 user user 5.5G Dec 16 10:00 bunny-v1.1-llama-3-8b-v-q5_k_m.gguf

# Verify model integrity (optional)
sha256sum models/bunny-v1.1-llama-3-8b-v-q5_k_m.gguf
# Compare with checksum from Hugging Face model card
```

### Configuration

Update `.env` with model path:

```bash
# Absolute path (recommended for production)
LLAMA_MODEL_PATH=/path/to/canvuslocallm/models/bunny-v1.1-llama-3-8b-v-q5_k_m.gguf

# Relative path (works for development)
LLAMA_MODEL_PATH=models/bunny-v1.1-llama-3-8b-v-q5_k_m.gguf
LLAMA_MODELS_DIR=./models
```

---

## Prompt Engineering

### System Prompt

Bunny works best with a clear system prompt that establishes context:

```
You are a helpful AI assistant integrated with Canvus collaborative workspace.
Analyze content accurately and respond concisely. Focus on the user's specific
question and provide actionable information.
```

### Text Generation Prompts

**Simple Question-Answer:**
```
User: What is the capital of France?
Assistant:
```

**Instruction Following:**
```
Summarize the following text in 3 bullet points:

[TEXT TO SUMMARIZE]

Summary:
```

**Analysis Task:**
```
Analyze the following meeting notes and identify:
1. Action items
2. Key decisions
3. Open questions

Meeting Notes:
[MEETING NOTES]

Analysis:
```

### Vision Prompts

For image analysis, structure prompts to guide the model's attention:

**General Description:**
```
Describe this image in detail. Include:
- Main subjects and objects
- Colors and composition
- Any text visible in the image
- Overall context or setting
```

**Specific Analysis:**
```
Analyze this diagram and explain:
1. The main components shown
2. How they connect or relate
3. The overall purpose or workflow
```

**Canvas Analysis:**
```
This is a screenshot of a collaborative canvas. Identify and describe:
- All visible widgets (notes, images, shapes)
- Any text content on notes
- The apparent organization or grouping
- Key themes or topics represented
```

**Document/Handwriting:**
```
Extract all text from this image. If the text is handwritten,
do your best to transcribe it accurately. Note any parts that
are unclear or illegible.
```

### Prompt Best Practices

1. **Be specific**: Tell the model exactly what you want
   - Bad: "What's in this image?"
   - Good: "List all objects visible in this image, starting with the largest."

2. **Use structured output**: Request specific formats
   - "Respond in JSON format with keys: summary, key_points, action_items"
   - "Provide your answer as a numbered list"

3. **Set context**: Explain the use case
   - "You are analyzing a project management canvas..."
   - "This is a technical diagram showing..."

4. **Limit scope**: Ask focused questions
   - Don't ask multiple unrelated questions in one prompt
   - Break complex analysis into multiple focused queries

5. **Temperature tuning**: Adjust for task type
   - Factual extraction: Temperature 0.3
   - Creative summaries: Temperature 0.7
   - Brainstorming: Temperature 1.0

---

## Vision Capabilities

### Supported Image Types

| Format | Support | Notes |
|--------|---------|-------|
| JPEG | Full | Recommended for photos |
| PNG | Full | Recommended for diagrams |
| BMP | Full | Uncompressed, large files |
| GIF | Partial | First frame only |
| WebP | Full | Modern format, good compression |

### Image Resolution

- **Input**: Any resolution (automatically resized)
- **Processing**: Resized to 384x384 pixels
- **Aspect Ratio**: Preserved with padding

### Vision Task Performance

| Task | Quality | Notes |
|------|---------|-------|
| Object recognition | Excellent | Identifies common objects reliably |
| Scene description | Good | General scene understanding |
| Text extraction (OCR) | Good | Better for printed than handwritten |
| Diagram analysis | Good | Can identify shapes and connections |
| Face detection | Good | Identifies presence, not individuals |
| Fine detail | Moderate | Limited by 384x384 resolution |
| Color identification | Excellent | Accurate color descriptions |

### Vision Examples

**Example 1: Canvas Screenshot Analysis**

Input Image: Screenshot of Canvus workspace with sticky notes

Prompt:
```
Analyze this canvas screenshot. For each visible note or widget:
1. Describe its content
2. Identify its apparent purpose
3. Note any relationships to other items
```

Expected Output:
```
I can see a canvas with the following items:

1. **Yellow Sticky Note (top-left)**: Contains the text "Project Goals"
   as a header, followed by 3 bullet points. This appears to be a
   goals/objectives note for the project.

2. **Blue Sticky Note (center)**: Shows a task list with checkboxes.
   Some items are checked, indicating completed tasks.

3. **Pink Sticky Note (right)**: Contains questions starting with "?"
   These appear to be open questions or discussion points.

4. **Connecting Lines**: There are arrows connecting the yellow note
   to several other items, suggesting a hierarchical relationship
   where goals drive the other content.
```

**Example 2: Handwriting Recognition**

Input Image: Photo of handwritten notes

Prompt:
```
Transcribe the handwritten text in this image. For any unclear portions,
indicate with [unclear]. Maintain the original structure (lists, paragraphs).
```

Expected Output:
```
Meeting Notes - Dec 15

Attendees:
- John
- Sarah
- [unclear - name]

Agenda:
1. Q4 Review
2. Budget discussion
3. Team updates

Action Items:
- John: Send report by Friday
- Sarah: Schedule follow-up
```

---

## GPU Performance Expectations

### Benchmark Results

Performance measured with Q5_K_M quantization, 4096 context size, single context:

| GPU | VRAM | Tokens/sec | First Token | 100-Token Response |
|-----|------|------------|-------------|---------------------|
| RTX 3060 | 12GB | 18-22 | ~600ms | ~5s |
| RTX 3070 | 8GB | 25-30 | ~450ms | ~3.5s |
| RTX 3080 | 10GB | 35-40 | ~350ms | ~2.5s |
| RTX 4060 | 8GB | 30-35 | ~400ms | ~3s |
| RTX 4070 | 12GB | 45-55 | ~300ms | ~2s |
| RTX 4080 | 16GB | 60-70 | ~250ms | ~1.5s |
| RTX 4090 | 24GB | 80-100 | ~200ms | ~1s |

### Vision Inference Overhead

Vision inference adds processing time for image encoding:

| Operation | Additional Time | Notes |
|-----------|-----------------|-------|
| Image load | ~10ms | Reading from disk |
| Resize/preprocess | ~20ms | Resize to 384x384 |
| Vision encoding | ~200-400ms | GPU-dependent |
| **Total overhead** | **~250-450ms** | Before text generation starts |

### Concurrent Performance

With multiple contexts (NumContexts > 1):

| Contexts | Throughput | Latency | VRAM Impact |
|----------|------------|---------|-------------|
| 1 | Baseline | Baseline | Baseline |
| 2 | ~1.8x | +10% | +1.5GB |
| 3 | ~2.5x | +20% | +3GB |
| 5 | ~3.5x | +40% | +6GB |

### Optimization Tips

1. **Batch prompts**: Process multiple short prompts together
2. **Limit max tokens**: Only request what you need
3. **Use appropriate quantization**: Match to your VRAM
4. **Monitor temperature**: GPU throttling impacts performance
5. **Close other GPU apps**: Maximize available VRAM

---

## Best Practices

### For Text Generation

1. **Keep prompts concise**: Long prompts slow processing
2. **Use stop sequences**: End generation cleanly
   ```go
   params.StopSequences = []string{"\n\nUser:", "END"}
   ```
3. **Set reasonable max tokens**: 200-500 for most responses
4. **Temperature by task**:
   - Analysis/extraction: 0.3
   - General responses: 0.7
   - Creative content: 1.0

### For Vision Tasks

1. **High contrast images**: Better OCR and analysis
2. **Good lighting**: Avoid shadows and glare
3. **Single subject**: One clear focus per image
4. **Appropriate resolution**: 384x384 is the working resolution
5. **Structured prompts**: Tell the model what to look for

### For Canvas Integration

1. **Segment large canvases**: Analyze sections, not whole canvas
2. **Preprocess screenshots**: Crop to relevant area
3. **Use consistent prompt templates**: Same format for same task types
4. **Cache results**: Don't re-analyze unchanged content
5. **Graceful degradation**: Have fallback for failed analysis

### Error Handling

1. **Retry on timeout**: Vision can take longer
2. **Validate outputs**: Check for common failure patterns
3. **Log comprehensively**: Track prompt/response for debugging
4. **Rate limit**: Don't overwhelm with concurrent requests

---

## Troubleshooting

### Common Issues

#### Model Won't Load

**Symptoms**: "Model file not found" or "Failed to load model"

**Solutions**:
1. Verify file path: `ls -la models/`
2. Check file integrity: Compare SHA256 hash
3. Use absolute path in configuration
4. Ensure sufficient disk space for model

#### Poor Vision Quality

**Symptoms**: Incorrect image descriptions, missing details

**Solutions**:
1. Increase image quality/resolution before processing
2. Use clearer, higher-contrast images
3. Simplify prompts - focus on one aspect at a time
4. Try higher quantization (Q5_K_M → Q6_K)

#### Slow Performance

**Symptoms**: Long wait times, low tokens/second

**Solutions**:
1. Check GPU utilization: `nvidia-smi`
2. Reduce context size if memory constrained
3. Use smaller quantization for speed
4. Close other GPU-intensive applications
5. Check for thermal throttling

#### Out of Memory

**Symptoms**: "CUDA out of memory" or context creation fails

**Solutions**:
1. Reduce `NumContexts` (3 → 2)
2. Reduce `ContextSize` (4096 → 2048)
3. Use smaller quantization (Q5_K_M → Q4_K_M)
4. Restart to clear fragmented memory
5. Check for memory leaks (call Close() properly)

#### Hallucination/Incorrect Output

**Symptoms**: Model makes up information, misreads text

**Solutions**:
1. Lower temperature (0.7 → 0.3)
2. Use more specific prompts
3. Request structured output (lists, JSON)
4. Verify with multiple queries if critical
5. Use higher quantization for better accuracy

### Diagnostic Commands

```bash
# Check GPU status
nvidia-smi

# Monitor GPU during inference
nvidia-smi dmon -d 1

# Check model file
file models/bunny-*.gguf
ls -lh models/bunny-*.gguf

# Test llama.cpp directly (if built)
./llama.cpp/main -m models/bunny-v1.1-llama-3-8b-v-q5_k_m.gguf \
    -p "Hello, how are you?" -n 50

# Check CUDA version
nvcc --version
nvidia-smi | grep "CUDA Version"
```

---

## References

- [Bunny GitHub Repository](https://github.com/BAAI-DCAI/Bunny)
- [Bunny Hugging Face Models](https://huggingface.co/BAAI)
- [llama.cpp GGUF Documentation](https://github.com/ggerganov/llama.cpp)
- [LLaMA-3 Model Card](https://huggingface.co/meta-llama/Meta-Llama-3-8B)
- [SigLIP Vision Encoder](https://arxiv.org/abs/2303.15343)

---

## See Also

- [llamaruntime API Reference](llamaruntime.md) - Go API documentation
- [Build Guide](build-guide.md) - Building llama.cpp and the application
- [Troubleshooting Guide](troubleshooting.md) - Detailed troubleshooting steps
