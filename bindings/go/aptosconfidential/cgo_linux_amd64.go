//go:build cgo && linux && amd64 && !musl

package aptosconfidential

/*
#cgo CFLAGS: -I${SRCDIR}/include
#cgo LDFLAGS: -laptos_confidential_asset_ffi -ldl -lpthread -lm
#include "aptos_confidential_asset.h"
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"runtime"
	"strconv"
	"unsafe"
)

func freeBuffer(b C.ConfidentialAssetByteBuffer) {
	C.confidential_asset_free_buffer(b)
}

func goString(b C.ConfidentialAssetByteBuffer) string {
	if b.ptr == nil || b.len == 0 {
		return ""
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(b.ptr)), b.len))
}

func cloneBytes(b C.ConfidentialAssetByteBuffer) []byte {
	if b.ptr == nil || b.len == 0 {
		return nil
	}
	return append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(b.ptr)), b.len)...)
}

// BatchRangeProof generates a Bulletproof batch range proof.
func BatchRangeProof(values []uint64, blindingsFlat, valBase, randBase []byte, numBits int) (proof []byte, commsFlat []byte, err error) {
	if len(valBase) != 32 || len(randBase) != 32 {
		return nil, nil, errors.New("val_base and rand_base must be 32 bytes")
	}
	if err := validateRangeNumBits(numBits); err != nil {
		return nil, nil, err
	}
	if len(blindingsFlat) != len(values)*32 {
		return nil, nil, errors.New("blindings_flat must be len(values)*32 bytes")
	}
	var valsPtr *C.uint64_t
	if len(values) > 0 {
		valsPtr = (*C.uint64_t)(unsafe.Pointer(&values[0]))
	}
	var bfPtr *C.uint8_t
	if len(blindingsFlat) > 0 {
		bfPtr = (*C.uint8_t)(unsafe.Pointer(&blindingsFlat[0]))
	}
	res := C.confidential_asset_batch_range_proof(
		valsPtr, C.size_t(len(values)),
		bfPtr, C.size_t(len(blindingsFlat)),
		(*C.uint8_t)(unsafe.Pointer(&valBase[0])), C.size_t(len(valBase)),
		(*C.uint8_t)(unsafe.Pointer(&randBase[0])), C.size_t(len(randBase)),
		C.size_t(numBits),
	)
	defer freeBuffer(res.proof)
	defer freeBuffer(res.comms_flat)
	defer freeBuffer(res.error)
	if res.error.len != 0 {
		return nil, nil, errors.New(goString(res.error))
	}
	return cloneBytes(res.proof), cloneBytes(res.comms_flat), nil
}

// BatchVerifyProof verifies a batch range proof.
func BatchVerifyProof(proof, commsFlat, valBase, randBase []byte, numBits int) (bool, error) {
	if len(valBase) != 32 || len(randBase) != 32 {
		return false, errors.New("val_base and rand_base must be 32 bytes")
	}
	if len(proof) == 0 || len(commsFlat) == 0 {
		return false, errors.New("proof and comms_flat must be non-empty")
	}
	if err := validateRangeNumBits(numBits); err != nil {
		return false, err
	}
	res := C.confidential_asset_batch_verify_proof(
		(*C.uint8_t)(unsafe.Pointer(&proof[0])), C.size_t(len(proof)),
		(*C.uint8_t)(unsafe.Pointer(&commsFlat[0])), C.size_t(len(commsFlat)),
		(*C.uint8_t)(unsafe.Pointer(&valBase[0])), C.size_t(len(valBase)),
		(*C.uint8_t)(unsafe.Pointer(&randBase[0])), C.size_t(len(randBase)),
		C.size_t(numBits),
	)
	defer freeBuffer(res.error)
	if res.error.len != 0 {
		return false, errors.New(goString(res.error))
	}
	return bool(res.value), nil
}

// Solver wraps the native discrete-log solver.
type Solver struct {
	ptr unsafe.Pointer
}

// NewSolver creates a solver instance.
func NewSolver() *Solver {
	p := C.confidential_asset_create_solver()
	s := &Solver{ptr: p}
	runtime.SetFinalizer(s, (*Solver).finalize)
	return s
}

func (s *Solver) finalize() {
	_ = s.Close()
}

// Close releases native resources for the solver and is safe to call multiple times.
func (s *Solver) Close() error {
	if s == nil {
		return errors.New("solver is nil")
	}
	if s.ptr == nil {
		return nil
	}
	C.confidential_asset_free_solver(s.ptr)
	s.ptr = nil
	runtime.SetFinalizer(s, nil)
	return nil
}

// Solve decrypts a balance from compressed Ristretto point y (32 bytes). maxNumBits is 16 or 32.
func (s *Solver) Solve(y []byte, maxNumBits uint8) (uint64, error) {
	if len(y) != 32 {
		return 0, errors.New("y must be 32 bytes")
	}
	if s == nil || s.ptr == nil {
		return 0, errSolverNilOrClosed
	}
	if err := validateSolverMaxNumBits(maxNumBits); err != nil {
		return 0, err
	}
	res := C.confidential_asset_solver_solve(
		s.ptr,
		(*C.uint8_t)(unsafe.Pointer(&y[0])), C.size_t(len(y)),
		C.uint8_t(maxNumBits),
	)
	defer freeBuffer(res.value)
	defer freeBuffer(res.error)
	if res.error.len != 0 {
		return 0, errors.New(goString(res.error))
	}
	return strconv.ParseUint(goString(res.value), 10, 64)
}
