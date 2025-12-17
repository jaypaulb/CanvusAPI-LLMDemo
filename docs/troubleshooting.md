# CanvusLocalLLM Troubleshooting Guide

This guide covers common issues and their solutions for CanvusLocalLLM, including GPU detection, model loading, inference failures, and build problems.

## Table of Contents

1. [Quick Diagnostics](#quick-diagnostics)
2. [GPU and CUDA Issues](#gpu-and-cuda-issues)
3. [Model Loading Issues](#model-loading-issues)
4. [Inference Issues](#inference-issues)
5. [Memory Issues](#memory-issues)
6. [Build Issues](#build-issues)
7. [Recovery System](#recovery-system)
8. [Logging and Debugging](#logging-and-debugging)
9. [Performance Issues](#performance-issues)
10. [Canvus API Issues](#canvus-api-issues)

---

## Quick Diagnostics

### System Check Script

Run these commands to quickly diagnose common issues:

```bash
# 1. Check GPU status
nvidia-smi

# 2. Check CUDA version
nvcc --version

# 3. Check model file exists
ls -lh models/*.gguf

# 4. Check build verification
./scripts/verify-build.sh --check-cuda --check-libs

# 5. Check application logs
tail -50 app.log

# 6. Check environment variables
cat .env | grep -E "LLAMA_|CUDA_"
```

### Quick Fixes Checklist

Before diving deep, try these common fixes:

- [ ] Restart the application
- [ ] Run `nvidia-smi` to ensure GPU is detected
- [ ] Check `.env` file has correct model path
- [ ] Verify model file exists and isn't corrupted
- [ ] Ensure no other GPU-intensive apps are running
- [ ] Check you have sufficient disk space

---

## GPU and CUDA Issues

### CUDA Not Found

**Error Message:**
```
CUDA GPU not available
failed to detect CUDA
cannot find -lcuda
```

**Causes:**
- NVIDIA drivers not installed
- CUDA Toolkit not installed
- Incompatible driver/CUDA version
- Missing environment variables

**Solutions:**

1. **Install NVIDIA Drivers:**
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install nvidia-driver-535  # or latest version

# Check installation
nvidia-smi
```

2. **Install CUDA Toolkit:**
```bash
# Download from NVIDIA
# https://developer.nvidia.com/cuda-downloads

# Ubuntu example
wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/cuda-keyring_1.1-1_all.deb
sudo dpkg -i cuda-keyring_1.1-1_all.deb
sudo apt update
sudo apt install cuda-toolkit-12-3
```

3. **Set Environment Variables:**
```bash
# Add to ~/.bashrc or ~/.profile
export PATH=/usr/local/cuda/bin:$PATH
export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH
export CUDA_HOME=/usr/local/cuda
```

4. **Verify Installation:**
```bash
nvcc --version
nvidia-smi
```

### GPU Not Detected

**Error Message:**
```
No CUDA GPUs detected
GPU detection failed
ErrGPUNotAvailable
```

**Causes:**
- Driver not loaded
- GPU in power-saving mode
- Incorrect PCIe slot or cables
- Hypervisor/VM configuration

**Solutions:**

1. **Check Driver Status:**
```bash
# List NVIDIA devices
lspci | grep -i nvidia

# Check driver module
lsmod | grep nvidia

# Reload driver if needed
sudo modprobe nvidia
```

2. **Check GPU Power State:**
```bash
# Check persistence mode
nvidia-smi -pm 1  # Enable persistence mode

# Check power state
nvidia-smi -q | grep "Power State"
```

3. **For VMs/Containers:**
```bash
# Docker - use NVIDIA Container Toolkit
docker run --gpus all nvidia/cuda:12.3-base nvidia-smi

# WSL2 - ensure WSL GPU support is enabled
wsl --list --verbose
```

### CUDA Version Mismatch

**Error Message:**
```
CUDA driver version is insufficient
version mismatch
libcuda.so: cannot open shared object file
```

**Causes:**
- llama.cpp built with different CUDA version
- Driver older than CUDA Toolkit

**Solutions:**

1. **Check Versions:**
```bash
# Driver version (supports up to CUDA X.Y)
nvidia-smi | grep "CUDA Version"

# Toolkit version
nvcc --version
```

2. **Rebuild llama.cpp:**
```bash
cd llamaruntime/llama.cpp
make clean
make LLAMA_CUDA=1 CUDA_VERSION=12.3
```

3. **Update Driver:**
```bash
# Driver must support your CUDA Toolkit version
# CUDA 12.3 requires driver >= 545
sudo apt install nvidia-driver-545
```

---

## Model Loading Issues

### Model File Not Found

**Error Message:**
```
model file not found
ErrModelNotFound
cannot open model file
```

**Causes:**
- Incorrect path in configuration
- Model not downloaded
- File permissions issue

**Solutions:**

1. **Verify Model Path:**
```bash
# Check configured path
grep LLAMA_MODEL_PATH .env

# List model files
ls -la models/

# Test path resolution
file models/bunny-v1.1-llama-3-8b-v-q5_k_m.gguf
```

2. **Download Model:**
```bash
# See docs/bunny-model.md for full instructions
huggingface-cli download BAAI/Bunny-v1.1-Llama-3-8B-V-GGUF \
    bunny-v1.1-llama-3-8b-v-q5_k_m.gguf \
    --local-dir models/
```

3. **Fix Permissions:**
```bash
chmod 644 models/*.gguf
```

4. **Use Absolute Path:**
```bash
# In .env
LLAMA_MODEL_PATH=/full/path/to/models/bunny-v1.1-llama-3-8b-v-q5_k_m.gguf
```

### Model Load Failed

**Error Message:**
```
failed to load model
ErrModelLoadFailed
invalid model format
```

**Causes:**
- Corrupted download
- Incompatible GGUF version
- Insufficient memory for model

**Solutions:**

1. **Verify File Integrity:**
```bash
# Check file size (should be ~5.5GB for Q5_K_M)
ls -lh models/*.gguf

# Verify with sha256sum (compare to Hugging Face)
sha256sum models/bunny-v1.1-llama-3-8b-v-q5_k_m.gguf
```

2. **Re-download Model:**
```bash
rm models/bunny-v1.1-llama-3-8b-v-q5_k_m.gguf
# Download again using method from docs/bunny-model.md
```

3. **Check GGUF Compatibility:**
```bash
# llama.cpp must support GGUF v3+
cd llamaruntime/llama.cpp
git pull origin master
make clean && make LLAMA_CUDA=1
```

4. **Test with llama.cpp Directly:**
```bash
./llamaruntime/llama.cpp/main \
    -m models/bunny-v1.1-llama-3-8b-v-q5_k_m.gguf \
    -p "Hello" -n 10
```

### Context Creation Failed

**Error Message:**
```
failed to create inference context
ErrContextCreateFailed
CUDA out of memory during context creation
```

**Causes:**
- Insufficient GPU VRAM
- Too many contexts requested
- Context size too large

**Solutions:**

1. **Reduce Configuration:**
```bash
# In .env - reduce these values
LLAMA_CONTEXT_SIZE=2048     # Was 4096
LLAMA_NUM_CONTEXTS=2        # Was 5
```

2. **Use Smaller Quantization:**
```bash
# Switch from Q5_K_M to Q4_K_M
LLAMA_MODEL_PATH=models/bunny-v1.1-llama-3-8b-v-q4_k_m.gguf
```

3. **Check Current VRAM Usage:**
```bash
nvidia-smi
# Note "Memory-Usage" column
```

4. **Free VRAM:**
```bash
# Close other GPU applications
# Kill any orphaned processes
nvidia-smi | grep "CanvusLocalLLM"
```

---

## Inference Issues

### Inference Timeout

**Error Message:**
```
inference timeout
ErrTimeout
context deadline exceeded
```

**Causes:**
- Context timeout too short
- GPU throttling
- Large prompt or max_tokens
- Overloaded system

**Solutions:**

1. **Increase Timeout:**
```go
// In code
params.Timeout = 120 * time.Second  // Increase from default

// Or in .env
AI_TIMEOUT=120
```

2. **Reduce Max Tokens:**
```go
params.MaxTokens = 200  // Reduce from 512
```

3. **Check GPU Throttling:**
```bash
# Monitor during inference
nvidia-smi dmon -d 1

# Check for thermal throttling
nvidia-smi -q | grep -A5 "Temperature"
```

4. **Reduce Batch Size:**
```bash
# In .env
LLAMA_BATCH_SIZE=256  # Was 512
```

### Inference Failed

**Error Message:**
```
inference failed
ErrInferenceFailed
llama_decode returned non-zero
```

**Causes:**
- Corrupted context state
- Invalid input
- GPU error

**Solutions:**

1. **Restart Application:**
```bash
# Cleanly shutdown and restart
pkill -SIGTERM canvuslocallm
./canvuslocallm
```

2. **Check Input Validity:**
```go
// Ensure prompt is not empty
if len(prompt) == 0 {
    return errors.New("empty prompt")
}

// Check for invalid characters
prompt = strings.ToValidUTF8(prompt, "")
```

3. **Monitor GPU Errors:**
```bash
# Check for GPU errors
dmesg | grep -i nvidia
nvidia-smi -q | grep -A3 "ECC Errors"
```

4. **Reset GPU State:**
```bash
# Reset GPU (requires root)
sudo nvidia-smi --gpu-reset -i 0
```

### Empty or Truncated Output

**Error Message:**
```
empty response
response truncated unexpectedly
```

**Causes:**
- Stop sequences triggered
- Max tokens too low
- Model confusion

**Solutions:**

1. **Review Stop Sequences:**
```go
// Check these aren't triggering early
params.StopSequences = []string{"\n\n", "User:"}
```

2. **Increase Max Tokens:**
```go
params.MaxTokens = 500  // Was 100
```

3. **Lower Temperature:**
```go
params.Temperature = 0.5  // Was 0.7
```

4. **Simplify Prompt:**
```go
// Avoid complex or ambiguous prompts
// Good: "Summarize this text in 3 points"
// Bad: "Please analyze and summarize the following..."
```

---

## Memory Issues

### Out of GPU Memory

**Error Message:**
```
CUDA out of memory
ErrInsufficientVRAM
failed to allocate
```

**Causes:**
- Model too large for GPU
- Too many contexts
- Memory fragmentation
- Memory leaks

**Solutions:**

1. **Reduce Memory Usage:**
```bash
# In .env
LLAMA_CONTEXT_SIZE=2048
LLAMA_NUM_CONTEXTS=2
LLAMA_BATCH_SIZE=256
```

2. **Use Smaller Quantization:**
```bash
LLAMA_MODEL_PATH=models/bunny-v1.1-llama-3-8b-v-q4_k_m.gguf
```

3. **Clear Fragmentation:**
```bash
# Restart application to clear GPU memory
pkill canvuslocallm
sleep 5
./canvuslocallm
```

4. **Monitor Memory Over Time:**
```bash
# Watch for memory growth (indicates leak)
watch -n 5 nvidia-smi
```

### Memory Leak

**Symptoms:**
- VRAM usage grows over time
- Eventually OOM errors
- Performance degrades

**Solutions:**

1. **Ensure Proper Cleanup:**
```go
// Always defer Close()
client, err := llamaruntime.NewClient(config)
if err != nil {
    return err
}
defer client.Close()  // CRITICAL
```

2. **Check Context Release:**
```go
// Contexts must be released
ctx, err := pool.Acquire(context)
if err != nil {
    return err
}
defer pool.Release(ctx)  // CRITICAL
```

3. **Monitor Pool Stats:**
```go
stats := pool.Stats()
if stats.InUse > stats.NumContexts {
    log.Printf("WARNING: Context leak detected")
}
```

4. **Enable Health Monitoring:**
```go
// The health checker monitors for memory issues
healthConfig := llamaruntime.DefaultHealthCheckerConfig()
healthConfig.OnUnhealthy = func(reason string) {
    log.Printf("Health check failed: %s", reason)
}
```

---

## Build Issues

### CGo Compilation Errors

**Error Message:**
```
cgo: C compiler cannot create executables
undefined reference to llama_*
cannot find -lllama
```

**Causes:**
- llama.cpp not built
- Wrong library path
- Missing CUDA libraries

**Solutions:**

1. **Build llama.cpp:**
```bash
cd llamaruntime/llama.cpp
make clean
make LLAMA_CUDA=1 -j$(nproc)
```

2. **Set Library Path:**
```bash
# Add to .env or shell
export CGO_LDFLAGS="-L$(pwd)/lib -lllama"
export LD_LIBRARY_PATH=$(pwd)/lib:$LD_LIBRARY_PATH
```

3. **Verify Libraries:**
```bash
ls -la lib/
# Should have: libllama.so, libggml.a, etc.
```

4. **Run Build Verification:**
```bash
./scripts/verify-build.sh --check-libs
```

### Linking Errors

**Error Message:**
```
undefined symbol: llama_*
relocation R_X86_64_32 against
```

**Causes:**
- Library version mismatch
- Position-independent code issue
- Static vs dynamic linking conflict

**Solutions:**

1. **Rebuild Everything:**
```bash
# Clean rebuild
cd llamaruntime/llama.cpp
make clean
make LLAMA_CUDA=1 LLAMA_BUILD_SHARED=ON -j$(nproc)

# Rebuild Go
cd ../..
go clean -cache
go build ./...
```

2. **Check Library ABI:**
```bash
# Verify library format
file lib/libllama.so
nm lib/libllama.so | grep llama_init
```

3. **Use Static Linking:**
```bash
# Build static library instead
make LLAMA_CUDA=1 LLAMA_BUILD_SHARED=OFF
```

---

## Recovery System

### Understanding the Recovery Manager

CanvusLocalLLM includes automatic recovery logic that handles:

1. **Retry with backoff**: 3 attempts with exponential delay
2. **Context reset**: After persistent failures
3. **Model reload**: If context resets fail
4. **Degraded mode**: When recovery exhausted

### Monitoring Recovery

```go
// Access recovery stats
stats := recoveryManager.Stats()
log.Printf("Recovery stats: %+v", stats)

// Check if in degraded mode
if recoveryManager.IsHealthy() {
    log.Printf("System healthy")
} else {
    log.Printf("System in degraded mode: %s", stats.DegradedModeReason)
}
```

### Forcing Recovery

```bash
# Trigger health check
curl http://localhost:8080/api/health

# If in degraded mode, restart application
pkill -SIGTERM canvuslocallm
./canvuslocallm
```

### Recovery Configuration

```go
config := llamaruntime.RecoveryConfig{
    MaxRetries:                   3,      // Retry attempts
    InitialBackoff:               500 * time.Millisecond,
    MaxBackoff:                   10 * time.Second,
    BackoffMultiplier:            2.0,
    ContextResetThreshold:        3,      // Failures before reset
    ModelReloadThreshold:         2,      // Resets before reload
    DegradedModeEnabled:          true,
    DegradedModeRecoveryInterval: 1 * time.Minute,
}
```

---

## Logging and Debugging

### Log Levels

```bash
# In .env
LOG_LEVEL=debug  # Options: debug, info, warn, error
```

### Log File Location

```bash
# Default log file
tail -f app.log

# With timestamps and filtering
grep "ERROR" app.log | tail -50
grep "llamaruntime" app.log | tail -50
```

### Debug GPU Operations

```bash
# Monitor GPU during operation
nvidia-smi dmon -d 1 -s pucvmet

# Fields:
# p: power, u: utilization, c: SM clock, v: memory clock
# m: memory used, e: encoder utilization, t: temperature
```

### Debug CGo Calls

```bash
# Enable CGo debug output
export GODEBUG=cgocheck=2

# Run application
./canvuslocallm
```

### Verbose llama.cpp Output

```go
// Enable in config
config.VerboseLogging = true
```

---

## Performance Issues

### Slow Inference

**Symptoms:**
- Low tokens/second
- Long first token time
- Timeouts on normal prompts

**Solutions:**

1. **Check GPU Utilization:**
```bash
nvidia-smi dmon -d 1
# GPU util should be high during inference
```

2. **Optimize Configuration:**
```bash
# In .env
LLAMA_BATCH_SIZE=512      # Higher for faster prompt processing
LLAMA_GPU_LAYERS=-1       # Full GPU offload
LLAMA_NUM_THREADS=4       # Match CPU cores for hybrid mode
```

3. **Check for Thermal Throttling:**
```bash
nvidia-smi -q | grep -A5 "Temperature"
# Target: <80C for sustained performance
```

4. **Reduce Context Size if Memory Bound:**
```bash
LLAMA_CONTEXT_SIZE=2048  # Reduces memory bandwidth
```

### High Latency

**Symptoms:**
- Slow first response
- Long wait between requests
- Context pool blocking

**Solutions:**

1. **Increase Context Pool:**
```bash
LLAMA_NUM_CONTEXTS=5  # Match MAX_CONCURRENT
```

2. **Optimize Connection:**
```go
// Check pool stats
stats := pool.Stats()
if stats.AcquireTimeouts > 0 {
    log.Printf("Pool undersized: %d timeouts", stats.AcquireTimeouts)
}
```

3. **Reduce Concurrent Load:**
```bash
MAX_CONCURRENT=3  # Don't exceed context pool size
```

---

## Canvus API Issues

### Connection Failed

**Error Message:**
```
failed to connect to Canvus server
connection refused
TLS handshake error
```

**Solutions:**

1. **Check Server Status:**
```bash
curl -k https://your-canvus-server/api/v1/canvases
```

2. **Verify Credentials:**
```bash
grep -E "CANVUS_SERVER|CANVUS_API_KEY|CANVAS_ID" .env
```

3. **Allow Self-Signed Certs (development):**
```bash
ALLOW_SELF_SIGNED_CERTS=true
```

4. **Check Network:**
```bash
ping your-canvus-server
telnet your-canvus-server 443
```

### API Rate Limiting

**Error Message:**
```
rate limit exceeded
429 Too Many Requests
```

**Solutions:**

1. **Reduce Concurrent Operations:**
```bash
MAX_CONCURRENT=3  # Reduce from 5
```

2. **Increase Retry Delay:**
```bash
RETRY_DELAY=5  # Seconds between retries
```

3. **Check Widget Update Frequency:**
```bash
# Reduce canvas polling if needed
CANVUS_POLL_INTERVAL=5  # Seconds
```

---

## Getting Help

If you've tried the solutions above and still have issues:

1. **Check Logs:**
```bash
tail -100 app.log > debug-log.txt
```

2. **Gather System Info:**
```bash
./scripts/verify-build.sh --check-cuda --check-libs > system-info.txt
nvidia-smi >> system-info.txt
```

3. **Create Issue:**
Include:
- Error message
- Steps to reproduce
- Relevant logs
- System info

4. **Resources:**
- [llamaruntime API Docs](llamaruntime.md)
- [Bunny Model Guide](bunny-model.md)
- [Build Guide](build-guide.md)
- [llama.cpp GitHub Issues](https://github.com/ggerganov/llama.cpp/issues)
