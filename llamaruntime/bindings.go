// Package llamaruntime provides Go bindings to llama.cpp for local LLM inference.
// This file contains CGo wrappers for the llama.cpp C API.
//
// Build Requirements:
// - CUDA Toolkit 12.x
// - llama.cpp compiled with CUDA support
// - Headers in deps/llama.cpp/ or system include path
// - Library (libllama.so/llama.dll) in lib/ or system library path
//
// Build Tags:
// - cgo: Requires CGo (enabled by default)
// - !nocgo: Excluded when nocgo tag is set (for testing without llama.cpp)
//
//go:build cgo && !nocgo

package llamaruntime

/*
#cgo CFLAGS: -I${SRCDIR}/../deps/llama.cpp -I${SRCDIR}/../deps/llama.cpp/include -I${SRCDIR}/../deps/llama.cpp/ggml/include
#cgo LDFLAGS: -L${SRCDIR}/../lib -lllama -lm -lstdc++
#cgo linux LDFLAGS: -Wl,-rpath,${SRCDIR}/../lib
#cgo windows LDFLAGS: -lllama

#include <stdlib.h>
#include <string.h>
#include <stdbool.h>
#include <stdint.h>

// Forward declarations for llama.cpp types and functions
// These must match llama.h from llama.cpp

// Opaque pointer types
typedef struct llama_model llama_model;
typedef struct llama_context llama_context;
typedef int32_t llama_token;
typedef int32_t llama_pos;
typedef int32_t llama_seq_id;

// Model parameters
struct llama_model_params {
    int32_t n_gpu_layers;
    int32_t split_mode;
    int32_t main_gpu;
    const float * tensor_split;
    void * progress_callback_user_data;
    bool (* progress_callback)(float progress, void * user_data);
    void * kv_overrides;
    bool vocab_only;
    bool use_mmap;
    bool use_mlock;
    bool check_tensors;
};

// Context parameters
struct llama_context_params {
    uint32_t n_ctx;
    uint32_t n_batch;
    uint32_t n_ubatch;
    uint32_t n_seq_max;
    int32_t n_threads;
    int32_t n_threads_batch;
    int32_t rope_scaling_type;
    int32_t pooling_type;
    int32_t attention_type;
    float rope_freq_base;
    float rope_freq_scale;
    float yarn_ext_factor;
    float yarn_attn_factor;
    float yarn_beta_fast;
    float yarn_beta_slow;
    uint32_t yarn_orig_ctx;
    float defrag_thold;
    void * cb_eval;
    void * cb_eval_user_data;
    int32_t type_k;
    int32_t type_v;
    bool logits_all;
    bool embeddings;
    bool offload_kqv;
    bool flash_attn;
    bool no_perf;
    void * abort_callback;
    void * abort_callback_data;
};

// Batch structure for token processing
struct llama_batch {
    int32_t n_tokens;
    llama_token * token;
    float * embd;
    llama_pos * pos;
    int32_t * n_seq_id;
    llama_seq_id ** seq_id;
    int8_t * logits;
};

// Sampling parameters
struct llama_sampler_chain_params {
    bool no_perf;
};

// Forward declare sampler types
typedef struct llama_sampler llama_sampler;

// Function declarations - these will be resolved at link time
extern void llama_backend_init(void);
extern void llama_backend_free(void);
extern struct llama_model_params llama_model_default_params(void);
extern struct llama_context_params llama_context_default_params(void);
extern llama_model * llama_load_model_from_file(const char * path_model, struct llama_model_params params);
extern void llama_free_model(llama_model * model);
extern llama_context * llama_new_context_with_model(llama_model * model, struct llama_context_params params);
extern void llama_free(llama_context * ctx);
extern int32_t llama_n_vocab(const llama_model * model);
extern int32_t llama_n_ctx(const llama_context * ctx);
extern int32_t llama_n_ctx_train(const llama_model * model);
extern int32_t llama_n_embd(const llama_model * model);
extern const char * llama_token_get_text(const llama_model * model, llama_token token);
extern llama_token llama_token_bos(const llama_model * model);
extern llama_token llama_token_eos(const llama_model * model);
extern llama_token llama_token_nl(const llama_model * model);
extern int32_t llama_tokenize(const llama_model * model, const char * text, int32_t text_len, llama_token * tokens, int32_t n_tokens_max, bool add_special, bool parse_special);
extern int32_t llama_token_to_piece(const llama_model * model, llama_token token, char * buf, int32_t length, int32_t lstrip, bool special);
extern struct llama_batch llama_batch_init(int32_t n_tokens, int32_t embd, int32_t n_seq_max);
extern void llama_batch_free(struct llama_batch batch);
extern int32_t llama_decode(llama_context * ctx, struct llama_batch batch);
extern float * llama_get_logits(llama_context * ctx);
extern float * llama_get_logits_ith(llama_context * ctx, int32_t i);
extern void llama_kv_cache_clear(llama_context * ctx);
extern void llama_synchronize(llama_context * ctx);
extern void llama_perf_context_reset(llama_context * ctx);

// Sampler functions
extern struct llama_sampler_chain_params llama_sampler_chain_default_params(void);
extern llama_sampler * llama_sampler_chain_init(struct llama_sampler_chain_params params);
extern void llama_sampler_chain_add(llama_sampler * chain, llama_sampler * smpl);
extern llama_token llama_sampler_sample(llama_sampler * chain, llama_context * ctx, int32_t idx);
extern void llama_sampler_free(llama_sampler * smpl);
extern llama_sampler * llama_sampler_init_temp(float temp);
extern llama_sampler * llama_sampler_init_top_k(int32_t k);
extern llama_sampler * llama_sampler_init_top_p(float p, size_t min_keep);
extern llama_sampler * llama_sampler_init_penalties(int32_t n_vocab, llama_token special_eos_id, llama_token linefeed_id, int32_t penalty_last_n, float penalty_repeat, float penalty_freq, float penalty_present, bool penalize_nl, bool ignore_eos);
extern llama_sampler * llama_sampler_init_dist(uint32_t seed);

// CUDA/GPU functions (may not be available in all builds)
// Note: GPU memory functions are CUDA-specific
// We'll use nvidia-ml or similar for GPU monitoring

// Helper function to check if CUDA is available
static inline int llama_has_cuda(void) {
#ifdef GGML_USE_CUDA
    return 1;
#else
    return 0;
#endif
}
*/
import "C"

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
	"unsafe"
)

// llamaBackend manages global llama.cpp initialization state.
var (
	llamaBackendOnce sync.Once
	llamaBackendInit bool
)

// llamaInit initializes the llama.cpp backend.
// This function is safe to call multiple times; initialization happens only once.
// It should be called before any other llama operations.
func llamaInit() {
	llamaBackendOnce.Do(func() {
		C.llama_backend_init()
		llamaBackendInit = true
	})
}

// llamaBackendFree releases global llama.cpp resources.
// This should only be called when completely done with llama.cpp,
// typically at application shutdown. After calling this, llamaInit()
// must be called again before any llama operations.
func llamaBackendFree() {
	if llamaBackendInit {
		C.llama_backend_free()
		llamaBackendInit = false
		llamaBackendOnce = sync.Once{}
	}
}

// llamaModel wraps a C llama_model pointer with automatic cleanup.
// The model is the core component that holds the neural network weights.
type llamaModel struct {
	ptr *C.llama_model
	mu  sync.Mutex
}

// loadModel loads a GGUF model from the specified file path.
// The numGPULayers parameter controls GPU offloading:
//   - -1: Offload all layers to GPU (recommended for CUDA builds)
//   - 0: Keep all layers on CPU (very slow, not recommended)
//   - N: Offload N layers to GPU, keep rest on CPU
//
// Returns an error if the model file doesn't exist, is corrupted,
// or if there's insufficient GPU memory.
func loadModel(path string, numGPULayers int, useMMap bool, useMlock bool) (*llamaModel, error) {
	// Ensure backend is initialized
	llamaInit()

	// Convert Go string to C string
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	// Get default model parameters
	params := C.llama_model_default_params()

	// Configure GPU offloading
	params.n_gpu_layers = C.int32_t(numGPULayers)

	// Configure memory mapping
	params.use_mmap = C.bool(useMMap)
	params.use_mlock = C.bool(useMlock)

	// Load the model
	model := C.llama_load_model_from_file(cPath, params)
	if model == nil {
		return nil, &LlamaError{
			Op:      "loadModel",
			Code:    -1,
			Message: fmt.Sprintf("failed to load model from %s", path),
			Err:     ErrModelLoadFailed,
		}
	}

	m := &llamaModel{ptr: model}

	// Set finalizer for automatic cleanup if Close() isn't called
	runtime.SetFinalizer(m, func(m *llamaModel) {
		m.Close()
	})

	return m, nil
}

// VocabSize returns the vocabulary size of the model.
func (m *llamaModel) VocabSize() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ptr == nil {
		return 0
	}
	return int(C.llama_n_vocab(m.ptr))
}

// ContextTrainSize returns the training context size of the model.
func (m *llamaModel) ContextTrainSize() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ptr == nil {
		return 0
	}
	return int(C.llama_n_ctx_train(m.ptr))
}

// EmbeddingSize returns the embedding dimension of the model.
func (m *llamaModel) EmbeddingSize() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ptr == nil {
		return 0
	}
	return int(C.llama_n_embd(m.ptr))
}

// BOSToken returns the beginning-of-sequence token ID.
func (m *llamaModel) BOSToken() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ptr == nil {
		return 0
	}
	return int(C.llama_token_bos(m.ptr))
}

// EOSToken returns the end-of-sequence token ID.
func (m *llamaModel) EOSToken() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ptr == nil {
		return 0
	}
	return int(C.llama_token_eos(m.ptr))
}

// Close releases the model resources.
// This is safe to call multiple times.
func (m *llamaModel) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ptr != nil {
		C.llama_free_model(m.ptr)
		m.ptr = nil
		runtime.SetFinalizer(m, nil)
	}
}

// llamaContext wraps a C llama_context pointer with automatic cleanup.
// A context holds the state for inference operations and is not thread-safe.
// Multiple contexts can share the same model for concurrent inference.
type llamaContext struct {
	ptr     *C.llama_context
	model   *llamaModel
	batch   C.struct_llama_batch
	sampler *C.llama_sampler
	mu      sync.Mutex
}

// createContext creates an inference context for the given model.
// contextSize is the maximum context window (prompt + response tokens).
// batchSize is the number of tokens processed in parallel.
// numThreads is the number of CPU threads for inference.
func createContext(model *llamaModel, contextSize, batchSize, numThreads int) (*llamaContext, error) {
	if model == nil || model.ptr == nil {
		return nil, &LlamaError{
			Op:      "createContext",
			Code:    -1,
			Message: "invalid model (nil)",
		}
	}

	// Get default context parameters
	params := C.llama_context_default_params()

	// Configure context
	params.n_ctx = C.uint32_t(contextSize)
	params.n_batch = C.uint32_t(batchSize)
	params.n_ubatch = C.uint32_t(batchSize)
	params.n_threads = C.int32_t(numThreads)
	params.n_threads_batch = C.int32_t(numThreads)
	params.flash_attn = C.bool(true) // Enable flash attention if available

	// Create context
	model.mu.Lock()
	ctx := C.llama_new_context_with_model(model.ptr, params)
	model.mu.Unlock()

	if ctx == nil {
		return nil, &LlamaError{
			Op:      "createContext",
			Code:    -1,
			Message: "failed to create inference context (possibly insufficient GPU memory)",
			Err:     ErrContextCreateFailed,
		}
	}

	// Create batch for token processing
	batch := C.llama_batch_init(C.int32_t(batchSize), 0, 1)

	// Create sampler chain
	samplerParams := C.llama_sampler_chain_default_params()
	sampler := C.llama_sampler_chain_init(samplerParams)

	c := &llamaContext{
		ptr:     ctx,
		model:   model,
		batch:   batch,
		sampler: sampler,
	}

	// Set finalizer for automatic cleanup
	runtime.SetFinalizer(c, func(c *llamaContext) {
		c.Close()
	})

	return c, nil
}

// ContextSize returns the context window size.
func (c *llamaContext) ContextSize() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ptr == nil {
		return 0
	}
	return int(C.llama_n_ctx(c.ptr))
}

// ClearKVCache clears the key-value cache for a fresh inference.
func (c *llamaContext) ClearKVCache() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ptr != nil {
		C.llama_kv_cache_clear(c.ptr)
	}
}

// Close releases the context resources.
// This is safe to call multiple times.
func (c *llamaContext) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sampler != nil {
		C.llama_sampler_free(c.sampler)
		c.sampler = nil
	}

	C.llama_batch_free(c.batch)

	if c.ptr != nil {
		C.llama_free(c.ptr)
		c.ptr = nil
		runtime.SetFinalizer(c, nil)
	}
}

// SamplingParams contains parameters for text generation.
type SamplingParams struct {
	Temperature   float32
	TopK          int
	TopP          float32
	RepeatPenalty float32
	Seed          uint32
}

// DefaultSamplingParams returns default sampling parameters.
func DefaultSamplingParams() SamplingParams {
	return SamplingParams{
		Temperature:   DefaultTemperature,
		TopK:          DefaultTopK,
		TopP:          DefaultTopP,
		RepeatPenalty: DefaultRepeatPenalty,
		Seed:          0, // 0 = random
	}
}

// configureSampler sets up the sampler chain with the given parameters.
func (c *llamaContext) configureSampler(params SamplingParams) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Free existing sampler if any
	if c.sampler != nil {
		C.llama_sampler_free(c.sampler)
	}

	// Create new sampler chain
	samplerParams := C.llama_sampler_chain_default_params()
	c.sampler = C.llama_sampler_chain_init(samplerParams)

	// Add samplers in order (order matters!)
	// Temperature sampling
	if params.Temperature > 0 {
		C.llama_sampler_chain_add(c.sampler, C.llama_sampler_init_temp(C.float(params.Temperature)))
	}

	// Top-K sampling
	if params.TopK > 0 {
		C.llama_sampler_chain_add(c.sampler, C.llama_sampler_init_top_k(C.int32_t(params.TopK)))
	}

	// Top-P (nucleus) sampling
	if params.TopP > 0 && params.TopP < 1.0 {
		C.llama_sampler_chain_add(c.sampler, C.llama_sampler_init_top_p(C.float(params.TopP), 1))
	}

	// Repetition penalty
	if params.RepeatPenalty != 1.0 {
		vocabSize := c.model.VocabSize()
		eosToken := c.model.EOSToken()
		C.llama_sampler_chain_add(c.sampler, C.llama_sampler_init_penalties(
			C.int32_t(vocabSize),
			C.llama_token(eosToken),
			C.llama_token(-1), // no linefeed penalty
			64,                // penalty_last_n
			C.float(params.RepeatPenalty),
			0.0,   // penalty_freq
			0.0,   // penalty_present
			false, // penalize_nl
			false, // ignore_eos
		))
	}

	// Distribution sampler (with seed for reproducibility)
	C.llama_sampler_chain_add(c.sampler, C.llama_sampler_init_dist(C.uint32_t(params.Seed)))
}

// tokenize converts text to tokens using the model's tokenizer.
func tokenize(model *llamaModel, text string, addBOS bool) ([]C.llama_token, error) {
	if model == nil || model.ptr == nil {
		return nil, &LlamaError{
			Op:      "tokenize",
			Code:    -1,
			Message: "invalid model (nil)",
		}
	}

	// Convert Go string to C string
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))

	// Allocate buffer for tokens (estimate: text length + some padding)
	maxTokens := len(text) + 32
	tokens := make([]C.llama_token, maxTokens)

	// Tokenize
	model.mu.Lock()
	nTokens := C.llama_tokenize(
		model.ptr,
		cText,
		C.int32_t(len(text)),
		&tokens[0],
		C.int32_t(maxTokens),
		C.bool(addBOS),
		C.bool(true), // parse_special
	)
	model.mu.Unlock()

	if nTokens < 0 {
		// Buffer too small, reallocate
		maxTokens = int(-nTokens)
		tokens = make([]C.llama_token, maxTokens)

		model.mu.Lock()
		nTokens = C.llama_tokenize(
			model.ptr,
			cText,
			C.int32_t(len(text)),
			&tokens[0],
			C.int32_t(maxTokens),
			C.bool(addBOS),
			C.bool(true),
		)
		model.mu.Unlock()

		if nTokens < 0 {
			return nil, &LlamaError{
				Op:      "tokenize",
				Code:    int(nTokens),
				Message: "tokenization failed",
			}
		}
	}

	return tokens[:nTokens], nil
}

// detokenize converts a token to its text representation.
func detokenize(model *llamaModel, token C.llama_token) string {
	if model == nil || model.ptr == nil {
		return ""
	}

	// Allocate buffer for text
	buf := make([]byte, 64)

	model.mu.Lock()
	nBytes := C.llama_token_to_piece(
		model.ptr,
		token,
		(*C.char)(unsafe.Pointer(&buf[0])),
		C.int32_t(len(buf)),
		0,     // lstrip
		false, // special
	)
	model.mu.Unlock()

	if nBytes < 0 {
		// Buffer too small
		buf = make([]byte, -nBytes)

		model.mu.Lock()
		nBytes = C.llama_token_to_piece(
			model.ptr,
			token,
			(*C.char)(unsafe.Pointer(&buf[0])),
			C.int32_t(len(buf)),
			0,
			false,
		)
		model.mu.Unlock()
	}

	if nBytes <= 0 {
		return ""
	}

	return string(buf[:nBytes])
}

// batchSetToken sets the token at index i in the batch.
// Uses unsafe pointer arithmetic to access the C array.
func batchSetToken(batch *C.struct_llama_batch, i int, token C.llama_token) {
	ptr := (*C.llama_token)(unsafe.Pointer(uintptr(unsafe.Pointer(batch.token)) + uintptr(i)*unsafe.Sizeof(C.llama_token(0))))
	*ptr = token
}

// batchSetPos sets the position at index i in the batch.
func batchSetPos(batch *C.struct_llama_batch, i int, pos C.llama_pos) {
	ptr := (*C.llama_pos)(unsafe.Pointer(uintptr(unsafe.Pointer(batch.pos)) + uintptr(i)*unsafe.Sizeof(C.llama_pos(0))))
	*ptr = pos
}

// batchSetNSeqID sets the number of sequence IDs at index i in the batch.
func batchSetNSeqID(batch *C.struct_llama_batch, i int, nSeqID C.int32_t) {
	ptr := (*C.int32_t)(unsafe.Pointer(uintptr(unsafe.Pointer(batch.n_seq_id)) + uintptr(i)*unsafe.Sizeof(C.int32_t(0))))
	*ptr = nSeqID
}

// batchSetSeqID sets the sequence ID at index i, slot j in the batch.
func batchSetSeqID(batch *C.struct_llama_batch, i int, j int, seqID C.llama_seq_id) {
	// seq_id is **llama_seq_id, so we need to access seq_id[i][j]
	// First get the pointer to the i-th element of the outer array
	outerPtr := (**C.llama_seq_id)(unsafe.Pointer(uintptr(unsafe.Pointer(batch.seq_id)) + uintptr(i)*unsafe.Sizeof((*C.llama_seq_id)(nil))))
	// Then access the j-th element of the inner array
	innerPtr := (*C.llama_seq_id)(unsafe.Pointer(uintptr(unsafe.Pointer(*outerPtr)) + uintptr(j)*unsafe.Sizeof(C.llama_seq_id(0))))
	*innerPtr = seqID
}

// batchSetLogits sets the logits flag at index i in the batch.
func batchSetLogits(batch *C.struct_llama_batch, i int, logits C.int8_t) {
	ptr := (*C.int8_t)(unsafe.Pointer(uintptr(unsafe.Pointer(batch.logits)) + uintptr(i)*unsafe.Sizeof(C.int8_t(0))))
	*ptr = logits
}

// inferText performs text inference on the given prompt.
// It returns the generated text and any error encountered.
// The context is used for cancellation and timeout.
func inferText(ctx context.Context, llamaCtx *llamaContext, prompt string, maxTokens int, params SamplingParams) (string, error) {
	if llamaCtx == nil || llamaCtx.ptr == nil {
		return "", &LlamaError{
			Op:      "inferText",
			Code:    -1,
			Message: "invalid context (nil)",
		}
	}

	// Configure sampler with params
	llamaCtx.configureSampler(params)

	// Clear KV cache for fresh inference
	llamaCtx.ClearKVCache()

	// Tokenize the prompt
	tokens, err := tokenize(llamaCtx.model, prompt, true)
	if err != nil {
		return "", fmt.Errorf("tokenize prompt: %w", err)
	}

	// Check if prompt fits in context
	contextSize := llamaCtx.ContextSize()
	if len(tokens)+maxTokens > contextSize {
		return "", &LlamaError{
			Op:      "inferText",
			Code:    -1,
			Message: fmt.Sprintf("prompt (%d tokens) + max_tokens (%d) exceeds context size (%d)", len(tokens), maxTokens, contextSize),
		}
	}

	// Process prompt tokens
	llamaCtx.mu.Lock()

	// Setup batch for prompt
	for i, token := range tokens {
		batchSetToken(&llamaCtx.batch, i, token)
		batchSetPos(&llamaCtx.batch, i, C.llama_pos(i))
		batchSetNSeqID(&llamaCtx.batch, i, 1)
		batchSetSeqID(&llamaCtx.batch, i, 0, 0)
		batchSetLogits(&llamaCtx.batch, i, 0)
	}
	// Only compute logits for last prompt token
	batchSetLogits(&llamaCtx.batch, len(tokens)-1, 1)
	llamaCtx.batch.n_tokens = C.int32_t(len(tokens))

	// Decode prompt
	if ret := C.llama_decode(llamaCtx.ptr, llamaCtx.batch); ret != 0 {
		llamaCtx.mu.Unlock()
		return "", &LlamaError{
			Op:      "inferText",
			Code:    int(ret),
			Message: "failed to decode prompt",
			Err:     ErrInferenceFailed,
		}
	}

	llamaCtx.mu.Unlock()

	// Generate tokens
	var result []byte
	nPrompt := len(tokens)
	eosToken := C.llama_token(llamaCtx.model.EOSToken())

	for i := 0; i < maxTokens; i++ {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return string(result), ctx.Err()
		default:
		}

		llamaCtx.mu.Lock()

		// Sample next token
		newToken := C.llama_sampler_sample(llamaCtx.sampler, llamaCtx.ptr, -1)

		llamaCtx.mu.Unlock()

		// Check for end of sequence
		if newToken == eosToken {
			break
		}

		// Decode token to text
		piece := detokenize(llamaCtx.model, newToken)
		result = append(result, piece...)

		// Prepare next batch
		llamaCtx.mu.Lock()

		batchSetToken(&llamaCtx.batch, 0, newToken)
		batchSetPos(&llamaCtx.batch, 0, C.llama_pos(nPrompt+i))
		batchSetNSeqID(&llamaCtx.batch, 0, 1)
		batchSetSeqID(&llamaCtx.batch, 0, 0, 0)
		batchSetLogits(&llamaCtx.batch, 0, 1)
		llamaCtx.batch.n_tokens = 1

		// Decode
		if ret := C.llama_decode(llamaCtx.ptr, llamaCtx.batch); ret != 0 {
			llamaCtx.mu.Unlock()
			return string(result), &LlamaError{
				Op:      "inferText",
				Code:    int(ret),
				Message: fmt.Sprintf("failed to decode at token %d", i),
				Err:     ErrInferenceFailed,
			}
		}

		llamaCtx.mu.Unlock()
	}

	return string(result), nil
}

// inferVision performs multimodal (text + image) inference.
// NOTE: Vision support depends on the model having vision capabilities (e.g., Bunny).
// The image data should be preprocessed before calling this function.
func inferVision(ctx context.Context, llamaCtx *llamaContext, prompt string, imageData []byte, maxTokens int, params SamplingParams) (string, error) {
	// TODO: Implement vision inference
	// This requires:
	// 1. Image preprocessing (resize, normalize)
	// 2. Image encoding using the vision encoder
	// 3. Combining image embeddings with text tokens
	// 4. Running inference on the combined input
	//
	// For now, we return an error indicating vision is not yet implemented.
	// Vision support will be added in a follow-up task.
	return "", &LlamaError{
		Op:      "inferVision",
		Code:    -1,
		Message: "vision inference not yet implemented",
	}
}

// GPUMemoryInfo holds GPU memory statistics.
type GPUMemoryInfo struct {
	Used       int64     // Used VRAM in bytes
	Total      int64     // Total VRAM in bytes
	Free       int64     // Free VRAM in bytes
	UsedPct    float64   // Usage percentage
	LastUpdate time.Time // When this info was collected
}

// getGPUMemory returns GPU memory usage information.
// This uses nvidia-smi or CUDA APIs to query GPU memory.
// NOTE: This is a placeholder implementation. Full GPU monitoring
// will be implemented using nvidia-ml-go or direct CUDA queries.
func getGPUMemory() (*GPUMemoryInfo, error) {
	// Check if CUDA is available
	hasCUDA := int(C.llama_has_cuda())
	if hasCUDA == 0 {
		return nil, &LlamaError{
			Op:      "getGPUMemory",
			Code:    -1,
			Message: "CUDA not available in this build",
			Err:     ErrGPUNotAvailable,
		}
	}

	// TODO: Implement actual GPU memory query using:
	// - nvidia-ml-go library
	// - Direct CUDA API calls via CGo
	// - Parsing nvidia-smi output as fallback
	//
	// For now, return placeholder indicating GPU is available
	// but memory stats are not implemented.
	return &GPUMemoryInfo{
		Used:       0,
		Total:      0,
		Free:       0,
		UsedPct:    0,
		LastUpdate: time.Now(),
	}, nil
}

// hasCUDA returns true if llama.cpp was built with CUDA support.
func hasCUDA() bool {
	return int(C.llama_has_cuda()) != 0
}

// freeContext is a convenience function that wraps Close().
// It's provided for API consistency with freeModel.
func freeContext(ctx *llamaContext) {
	if ctx != nil {
		ctx.Close()
	}
}

// freeModel is a convenience function that wraps Close().
// It's provided for API consistency with freeContext.
func freeModel(model *llamaModel) {
	if model != nil {
		model.Close()
	}
}
