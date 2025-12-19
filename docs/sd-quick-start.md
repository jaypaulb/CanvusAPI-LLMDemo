# Stable Diffusion Quick Start

Fast reference for getting stable-diffusion.cpp integration running.

## TL;DR

```bash
# 1. Build the C library
cd deps/stable-diffusion.cpp
./build-linux.sh        # or .\build-windows.ps1 on Windows

# 2. Download model (~4GB)
mkdir -p models
wget -O models/sd-v1-5.safetensors \
  https://huggingface.co/runwayml/stable-diffusion-v1-5/resolve/main/v1-5-pruned-emaonly.safetensors

# 3. Configure environment
echo "SD_MODEL_PATH=models/sd-v1-5.safetensors" >> .env

# 4. Build and run
export LD_LIBRARY_PATH=$PWD/lib:$LD_LIBRARY_PATH
CGO_ENABLED=1 go build -tags sd
./CanvusAPI-LLM
```

## Environment Variables

```bash
# Required
SD_MODEL_PATH=models/sd-v1-5.safetensors

# Optional (sensible defaults)
SD_IMAGE_SIZE=512
SD_INFERENCE_STEPS=20
SD_GUIDANCE_SCALE=7.5
SD_MAX_CONCURRENT=2
```

## Code Example

```go
package main

import (
    "context"
    "log"
    "os"
    "time"
    "go_backend/sdruntime"
)

func main() {
    // Create pool
    pool, err := sdruntime.NewContextPool(2, "models/sd-v1-5.safetensors")
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Generate
    params := sdruntime.GenerateParams{
        Prompt:   "a beautiful sunset",
        Width:    512,
        Height:   512,
        Steps:    20,
        CFGScale: 7.5,
        Seed:     -1,
    }

    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    imageData, err := pool.Generate(ctx, params)
    if err != nil {
        log.Fatal(err)
    }

    os.WriteFile("output.png", imageData, 0644)
}
```

## Common Issues

| Problem | Fix |
|---------|-----|
| `CUDA not found` | Add `/usr/local/cuda/bin` to PATH |
| `libstable-diffusion.so not found` | Set `LD_LIBRARY_PATH=$PWD/lib` |
| `out of VRAM` | Reduce `SD_IMAGE_SIZE` or `SD_MAX_CONCURRENT` |
| `generation timeout` | Increase `SD_TIMEOUT_SECONDS` or reduce steps |

## Performance Tips

- **512x512**: ~2-3s (fast, good quality)
- **768x768**: ~5-7s (balanced)
- **1024x1024**: ~10-15s (slow, high quality)
- **Max concurrent**: 2-3 with 10GB VRAM, 4-6 with 24GB

## Status Check

```bash
# Check NVIDIA GPU
nvidia-smi

# Check CUDA
nvcc --version

# Verify library built
ls -lh lib/libstable-diffusion.so

# Check backend (after building with -tags sd)
go run -tags sd . -sd-backend-info
```

## Full Documentation

See [docs/stable-diffusion-integration.md](./stable-diffusion-integration.md) for complete details.
