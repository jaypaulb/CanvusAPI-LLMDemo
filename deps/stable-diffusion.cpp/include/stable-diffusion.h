/*
 * stable-diffusion.h - C API for stable-diffusion.cpp
 *
 * This header defines the C interface expected by CanvusLocalLLM CGo bindings.
 * The actual implementation comes from https://github.com/leejet/stable-diffusion.cpp
 *
 * This file serves as a reference for the expected API. When stable-diffusion.cpp
 * is built, its actual header should be used instead.
 *
 * Copyright (c) 2024 CanvusLocalLLM Project
 * SPDX-License-Identifier: MIT
 */

#ifndef STABLE_DIFFUSION_H
#define STABLE_DIFFUSION_H

#include <stdint.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

/* ----------------------------------------------------------------------------
 * Type Definitions
 * ---------------------------------------------------------------------------- */

/**
 * Opaque context handle for Stable Diffusion inference.
 * Created with sd_ctx_create() and freed with sd_ctx_free().
 */
typedef struct sd_ctx sd_ctx_t;

/**
 * Image data returned from generation functions.
 * Contains RGBA pixel data and dimensions.
 */
typedef struct {
    uint8_t* data;      /**< RGBA pixel data, row-major order */
    int width;          /**< Image width in pixels */
    int height;         /**< Image height in pixels */
    int channels;       /**< Number of channels (typically 4 for RGBA) */
} sd_image_t;

/* ----------------------------------------------------------------------------
 * Enumerations
 * ---------------------------------------------------------------------------- */

/**
 * Sampling methods for the diffusion process.
 * Different methods trade off quality vs speed.
 */
typedef enum {
    SD_SAMPLE_EULER_A = 0,      /**< Euler Ancestral - fast, good quality */
    SD_SAMPLE_EULER = 1,        /**< Euler - deterministic */
    SD_SAMPLE_HEUN = 2,         /**< Heun - slower, higher quality */
    SD_SAMPLE_DPM2 = 3,         /**< DPM2 */
    SD_SAMPLE_DPMPP_2S_A = 4,   /**< DPM++ 2S Ancestral */
    SD_SAMPLE_DPMPP_2M = 5,     /**< DPM++ 2M - recommended */
    SD_SAMPLE_DPMPP_2M_V2 = 6,  /**< DPM++ 2M v2 */
    SD_SAMPLE_LCM = 7           /**< LCM - very fast, requires LCM model */
} sd_sample_method_t;

/**
 * Model types supported by stable-diffusion.cpp.
 */
typedef enum {
    SD_TYPE_SD1 = 0,    /**< Stable Diffusion 1.x */
    SD_TYPE_SD2 = 1,    /**< Stable Diffusion 2.x */
    SD_TYPE_SDXL = 2,   /**< Stable Diffusion XL */
    SD_TYPE_SD3 = 3     /**< Stable Diffusion 3 */
} sd_model_type_t;

/* ----------------------------------------------------------------------------
 * Context Management
 * ---------------------------------------------------------------------------- */

/**
 * Create a new Stable Diffusion context by loading a model.
 *
 * @param model_path Path to model file (.safetensors, .ckpt, or GGUF)
 * @param vae_path Optional path to separate VAE model (NULL for built-in)
 * @param taesd_path Optional path to TAESD model for fast preview (NULL to skip)
 * @param lora_model_dir Optional directory containing LoRA models (NULL for none)
 * @param vae_decode_only If true, skip VAE encoder (faster for txt2img only)
 * @param n_threads Number of CPU threads for non-CUDA operations
 * @param vae_tiling Enable VAE tiling for lower memory usage
 * @param free_params_immediately Free model params after loading (saves memory)
 *
 * @return Pointer to context, or NULL on failure
 *
 * @note The returned context must be freed with sd_ctx_free()
 */
sd_ctx_t* sd_ctx_create(
    const char* model_path,
    const char* vae_path,
    const char* taesd_path,
    const char* lora_model_dir,
    bool vae_decode_only,
    int n_threads,
    bool vae_tiling,
    bool free_params_immediately
);

/**
 * Free a Stable Diffusion context and release all resources.
 *
 * @param ctx Context to free (may be NULL)
 *
 * @note Safe to call with NULL pointer
 */
void sd_ctx_free(sd_ctx_t* ctx);

/* ----------------------------------------------------------------------------
 * Image Generation
 * ---------------------------------------------------------------------------- */

/**
 * Generate an image from a text prompt.
 *
 * @param ctx Valid SD context from sd_ctx_create()
 * @param prompt Text description of desired image
 * @param negative_prompt Text describing what to avoid (may be empty)
 * @param clip_skip Number of CLIP layers to skip (-1 for default)
 * @param cfg_scale Classifier-free guidance scale (typically 7.0-9.0)
 * @param width Output image width (must be multiple of 8)
 * @param height Output image height (must be multiple of 8)
 * @param sample_method Sampling algorithm to use
 * @param sample_steps Number of diffusion steps (typically 20-50)
 * @param seed Random seed for reproducibility (-1 for random)
 * @param batch_count Number of images to generate (typically 1)
 *
 * @return Pointer to generated image(s), or NULL on failure
 *
 * @note Returned image must be freed with sd_free_image()
 * @note For batch_count > 1, returns array of images
 */
sd_image_t* txt2img(
    sd_ctx_t* ctx,
    const char* prompt,
    const char* negative_prompt,
    int clip_skip,
    float cfg_scale,
    int width,
    int height,
    sd_sample_method_t sample_method,
    int sample_steps,
    int64_t seed,
    int batch_count
);

/**
 * Free image data returned from generation functions.
 *
 * @param image Image to free (may be NULL)
 *
 * @note Safe to call with NULL pointer
 */
void sd_free_image(sd_image_t* image);

/* ----------------------------------------------------------------------------
 * Utility Functions
 * ---------------------------------------------------------------------------- */

/**
 * Get information about the compute backend.
 *
 * @return Human-readable string describing backend (e.g., "CUDA", "CPU")
 *
 * @note Returned string is static, do not free
 */
const char* sd_get_backend_info(void);

/**
 * Check if CUDA acceleration is available.
 *
 * @return true if CUDA is available and functional
 */
bool sd_cuda_available(void);

/**
 * Get the version string of stable-diffusion.cpp.
 *
 * @return Version string (e.g., "1.0.0")
 *
 * @note Returned string is static, do not free
 */
const char* sd_get_version(void);

#ifdef __cplusplus
}
#endif

#endif /* STABLE_DIFFUSION_H */
