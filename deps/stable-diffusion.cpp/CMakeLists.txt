# CMakeLists.txt for stable-diffusion.cpp integration
#
# This file configures the build for stable-diffusion.cpp as a shared library
# with CUDA acceleration for use in CanvusLocalLLM.
#
# Build targets:
#   - stable-diffusion (shared library)
#
# Usage:
#   Windows (PowerShell):
#     cmake -B build -G "Visual Studio 17 2022" -A x64 -DGGML_CUDA=ON -DCMAKE_BUILD_TYPE=Release
#     cmake --build build --config Release
#
#   Linux (Bash):
#     cmake -B build -DGGML_CUDA=ON -DCMAKE_BUILD_TYPE=Release
#     cmake --build build -j$(nproc)
#
# Prerequisites:
#   - CMake 3.18 or newer
#   - CUDA Toolkit 11.8 or newer
#   - Windows: Visual Studio 2022 with C++ workload
#   - Linux: GCC 9+ or Clang 10+

cmake_minimum_required(VERSION 3.18)
project(stable-diffusion-cpp-integration VERSION 1.0.0 LANGUAGES C CXX)

# Options
option(GGML_CUDA "Enable CUDA acceleration" ON)
option(BUILD_SHARED_LIBS "Build as shared library" ON)
option(SD_BUILD_EXAMPLES "Build example programs" OFF)

# CUDA architectures for RTX GPUs
# 75 = RTX 20xx (Turing)
# 86 = RTX 30xx (Ampere)
# 89 = RTX 40xx (Ada Lovelace)
set(CMAKE_CUDA_ARCHITECTURES "75;86;89" CACHE STRING "CUDA architectures to compile for")

# C++ standard
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Output directories - put libraries in parent lib/ directory
set(CMAKE_LIBRARY_OUTPUT_DIRECTORY ${CMAKE_SOURCE_DIR}/../../lib)
set(CMAKE_RUNTIME_OUTPUT_DIRECTORY ${CMAKE_SOURCE_DIR}/../../lib)
set(CMAKE_ARCHIVE_OUTPUT_DIRECTORY ${CMAKE_SOURCE_DIR}/../../lib)

# For multi-config generators (Visual Studio)
foreach(CONFIG_TYPE ${CMAKE_CONFIGURATION_TYPES})
    string(TOUPPER ${CONFIG_TYPE} CONFIG_UPPER)
    set(CMAKE_LIBRARY_OUTPUT_DIRECTORY_${CONFIG_UPPER} ${CMAKE_SOURCE_DIR}/../../lib)
    set(CMAKE_RUNTIME_OUTPUT_DIRECTORY_${CONFIG_UPPER} ${CMAKE_SOURCE_DIR}/../../lib)
    set(CMAKE_ARCHIVE_OUTPUT_DIRECTORY_${CONFIG_UPPER} ${CMAKE_SOURCE_DIR}/../../lib)
endforeach()

# Find CUDA if enabled
if(GGML_CUDA)
    enable_language(CUDA)
    find_package(CUDAToolkit REQUIRED)

    message(STATUS "CUDA enabled")
    message(STATUS "CUDA version: ${CUDAToolkit_VERSION}")
    message(STATUS "CUDA include dir: ${CUDAToolkit_INCLUDE_DIRS}")
    message(STATUS "CUDA architectures: ${CMAKE_CUDA_ARCHITECTURES}")

    add_definitions(-DGGML_USE_CUDA)
endif()

# Check if stable-diffusion.cpp source exists
# The source should be cloned to the 'src' subdirectory
if(EXISTS "${CMAKE_SOURCE_DIR}/src/CMakeLists.txt")
    message(STATUS "Found stable-diffusion.cpp source in src/")

    # Add stable-diffusion.cpp as a subdirectory
    add_subdirectory(src)

else()
    message(WARNING "")
    message(WARNING "stable-diffusion.cpp source not found!")
    message(WARNING "")
    message(WARNING "Please clone the repository into the 'src' directory:")
    message(WARNING "  cd ${CMAKE_SOURCE_DIR}")
    message(WARNING "  git clone https://github.com/leejet/stable-diffusion.cpp.git src")
    message(WARNING "")
    message(WARNING "Or download and extract to: ${CMAKE_SOURCE_DIR}/src")
    message(WARNING "")

    # Create a placeholder target for documentation purposes
    # This allows the CMake configuration to complete even without source
    message(STATUS "Creating placeholder configuration...")

    # Define expected library interface for CGo
    add_library(stable-diffusion INTERFACE)

    # Document the expected header interface
    set(SD_HEADER_CONTENT "/*
 * stable-diffusion.h - C API for stable-diffusion.cpp
 *
 * This header defines the C interface expected by CanvusLocalLLM CGo bindings.
 * The actual implementation comes from https://github.com/leejet/stable-diffusion.cpp
 *
 * Expected API:
 */
#ifndef STABLE_DIFFUSION_H
#define STABLE_DIFFUSION_H

#include <stdint.h>
#include <stdbool.h>

#ifdef __cplusplus
extern \"C\" {
#endif

/* Opaque context handle */
typedef struct sd_ctx sd_ctx_t;

/* Image data structure */
typedef struct {
    uint8_t* data;      /* RGBA pixel data */
    int width;          /* Image width in pixels */
    int height;         /* Image height in pixels */
    int channels;       /* Number of channels (typically 4 for RGBA) */
} sd_image_t;

/* Sampling methods */
enum sd_sample_method {
    SD_SAMPLE_EULER_A = 0,
    SD_SAMPLE_EULER = 1,
    SD_SAMPLE_HEUN = 2,
    SD_SAMPLE_DPM2 = 3,
    SD_SAMPLE_DPMPP_2S_A = 4,
    SD_SAMPLE_DPMPP_2M = 5,
    SD_SAMPLE_DPMPP_2M_V2 = 6,
    SD_SAMPLE_LCM = 7
};

/* Create a new Stable Diffusion context */
sd_ctx_t* sd_ctx_create(
    const char* model_path,     /* Path to model file (.safetensors, .ckpt) */
    const char* vae_path,       /* Optional VAE path (NULL for default) */
    const char* taesd_path,     /* Optional TAESD path (NULL for default) */
    const char* lora_model_dir, /* Optional LoRA directory (NULL for none) */
    bool vae_decode_only,       /* True to skip VAE encode (for txt2img) */
    int n_threads,              /* Number of CPU threads */
    bool vae_tiling,            /* Enable VAE tiling for low memory */
    bool free_params_immediately /* Free params after loading */
);

/* Free a Stable Diffusion context */
void sd_ctx_free(sd_ctx_t* ctx);

/* Generate image from text prompt */
sd_image_t* txt2img(
    sd_ctx_t* ctx,
    const char* prompt,
    const char* negative_prompt,
    int clip_skip,
    float cfg_scale,
    int width,
    int height,
    enum sd_sample_method sample_method,
    int sample_steps,
    int64_t seed,
    int batch_count
);

/* Free image data */
void sd_free_image(sd_image_t* image);

/* Get backend information string */
const char* sd_get_backend_info(void);

/* Check if CUDA is available */
bool sd_cuda_available(void);

#ifdef __cplusplus
}
#endif

#endif /* STABLE_DIFFUSION_H */
")

    # Write the header file for reference
    file(WRITE "${CMAKE_SOURCE_DIR}/include/stable-diffusion.h" "${SD_HEADER_CONTENT}")
    message(STATUS "Created reference header at: ${CMAKE_SOURCE_DIR}/include/stable-diffusion.h")

endif()

# Installation rules
install(DIRECTORY include/ DESTINATION include FILES_MATCHING PATTERN "*.h")

# Print configuration summary
message(STATUS "")
message(STATUS "=== stable-diffusion.cpp Build Configuration ===")
message(STATUS "CMAKE_BUILD_TYPE: ${CMAKE_BUILD_TYPE}")
message(STATUS "BUILD_SHARED_LIBS: ${BUILD_SHARED_LIBS}")
message(STATUS "GGML_CUDA: ${GGML_CUDA}")
if(GGML_CUDA)
    message(STATUS "CUDA architectures: ${CMAKE_CUDA_ARCHITECTURES}")
endif()
message(STATUS "Output directory: ${CMAKE_SOURCE_DIR}/../../lib")
message(STATUS "================================================")
message(STATUS "")
