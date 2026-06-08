//go:build cgo && randomx
// +build cgo,randomx

package randomx

/*
#cgo CFLAGS: -I${SRCDIR}
#cgo LDFLAGS: -lrandomx -lstdc++ -lm
#include "randomx.h"
*/
import "C"
import (
"unsafe"
)

const (
RANDOMX_FLAG_DEFAULT     = C.RANDOMX_FLAG_DEFAULT
RANDOMX_FLAG_FULL_MEM    = C.RANDOMX_FLAG_FULL_MEM
RANDOMX_FLAG_JIT         = C.RANDOMX_FLAG_JIT
RANDOMX_FLAG_HARD_AES    = C.RANDOMX_FLAG_HARD_AES
RANDOMX_FLAG_LARGE_PAGES = C.RANDOMX_FLAG_LARGE_PAGES
)

type Cache struct {
ptr *C.randomx_cache
}

type Dataset struct {
ptr *C.randomx_dataset
}

type VM struct {
ptr *C.randomx_vm
}

func NewCache(flags int) *Cache {
cache := C.randomx_alloc_cache(C.randomx_flags(flags))
if cache == nil {
return nil
}
return &Cache{ptr: cache}
}

func (c *Cache) Init(key []byte) {
if c == nil || c.ptr == nil {
return
}
var keyPtr unsafe.Pointer
if len(key) > 0 {
keyPtr = unsafe.Pointer(&key[0])
}
C.randomx_init_cache(c.ptr, keyPtr, C.size_t(len(key)))
}

func (c *Cache) Close() {
if c == nil || c.ptr == nil {
return
}
C.randomx_release_cache(c.ptr)
c.ptr = nil
}

func NewDataset(flags int) *Dataset {
dataset := C.randomx_alloc_dataset(C.randomx_flags(flags))
if dataset == nil {
return nil
}
return &Dataset{ptr: dataset}
}

func (d *Dataset) Init(cache *Cache, start, count uint32) {
if d == nil || d.ptr == nil || cache == nil || cache.ptr == nil {
return
}
C.randomx_init_dataset(d.ptr, cache.ptr, C.uint32_t(start), C.uint32_t(count))
}

func (d *Dataset) Close() {
if d == nil || d.ptr == nil {
return
}
C.randomx_release_dataset(d.ptr)
d.ptr = nil
}

func NewVM(flags int, cache *Cache, dataset *Dataset) *VM {
var cCache *C.randomx_cache
var cDataset *C.randomx_dataset
if cache != nil {
cCache = cache.ptr
}
if dataset != nil && flags&RANDOMX_FLAG_FULL_MEM != 0 {
cDataset = dataset.ptr
}
for _, candidateFlags := range vmFlagFallbacks(flags) {
vm := C.randomx_create_vm(C.randomx_flags(candidateFlags), cCache, cDataset)
if vm != nil {
return &VM{ptr: vm}
}
}
return nil
}

func vmFlagFallbacks(flags int) []int {
fallbacks := make([]int, 0, 8)
seen := make(map[int]bool, 8)
for _, candidate := range []int{
flags,
flags &^ RANDOMX_FLAG_LARGE_PAGES,
flags &^ RANDOMX_FLAG_HARD_AES,
flags &^ (RANDOMX_FLAG_LARGE_PAGES | RANDOMX_FLAG_HARD_AES),
flags &^ RANDOMX_FLAG_JIT,
flags &^ (RANDOMX_FLAG_LARGE_PAGES | RANDOMX_FLAG_JIT),
flags &^ (RANDOMX_FLAG_HARD_AES | RANDOMX_FLAG_JIT),
flags &^ (RANDOMX_FLAG_LARGE_PAGES | RANDOMX_FLAG_HARD_AES | RANDOMX_FLAG_JIT),
} {
if !seen[candidate] {
fallbacks = append(fallbacks, candidate)
seen[candidate] = true
}
}
return fallbacks
}

func (vm *VM) CalculateHash(input, output []byte) {
if vm == nil || vm.ptr == nil {
return
}
var inputPtr unsafe.Pointer
if len(input) > 0 {
inputPtr = unsafe.Pointer(&input[0])
}
C.randomx_calculate_hash(vm.ptr, inputPtr, C.size_t(len(input)), unsafe.Pointer(&output[0]))
}

func (vm *VM) Close() {
if vm == nil || vm.ptr == nil {
return
}
C.randomx_destroy_vm(vm.ptr)
vm.ptr = nil
}
