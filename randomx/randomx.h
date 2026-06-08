#ifndef RANDOMX_H
#define RANDOMX_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef enum {
    RANDOMX_FLAG_DEFAULT = 0,
    RANDOMX_FLAG_FULL_MEM = 1,
    RANDOMX_FLAG_JIT = 2,
    RANDOMX_FLAG_HARD_AES = 4,
    RANDOMX_FLAG_LARGE_PAGES = 8,
} randomx_flags;

typedef struct randomx_cache randomx_cache;
typedef struct randomx_dataset randomx_dataset;
typedef struct randomx_vm randomx_vm;

randomx_cache* randomx_alloc_cache(randomx_flags flags);
void randomx_init_cache(randomx_cache* cache, const void* key, size_t keySize);
void randomx_release_cache(randomx_cache* cache);

randomx_dataset* randomx_alloc_dataset(randomx_flags flags);
void randomx_init_dataset(randomx_dataset* dataset, randomx_cache* cache, uint32_t start, uint32_t count);
void randomx_release_dataset(randomx_dataset* dataset);

randomx_vm* randomx_create_vm(randomx_flags flags, randomx_cache* cache, randomx_dataset* dataset);
void randomx_calculate_hash(randomx_vm* vm, const void* input, size_t inputSize, void* output);
void randomx_destroy_vm(randomx_vm* vm);

#ifdef __cplusplus
}
#endif

#endif
