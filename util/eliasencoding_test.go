package util

import (
	"testing"
)

func IntUintArrayEquals(a []int, b []uint) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != int(b[i]) {
			return false
		}
	}
	return true
}

func IntArrayEquals(a []int, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func TestEliasDelta(t *testing.T) {
	var xTest = []uint{24, 15, 2, 0}
	xRes := EliasDeltaDecode(EliasDelta(xTest...), false)
	if !IntUintArrayEquals(xRes, xTest) {
		t.Error("EliasDelta failed with positive values, got ", xRes, " instead of ", xTest)
	}

	var xTestNeg = []int{3, -15, 123, -31, 0, 42, 151626252512, 151626252512, 151626252512, 151626252512, 151626252512, 151626252512, 151626252512, 151626252512, 151626252512, 151626252512}
	resBitarray := EliasDeltaNegative(xTestNeg...)
	xResNeg := EliasDeltaDecode(resBitarray, true)
	if !IntArrayEquals(xResNeg, xTestNeg) {
		t.Error("EliasDelta failed with negative values, got ", xResNeg, " instead of ", xTestNeg)
	}
}

func TestEliasGamma(t *testing.T) {
	var xTest = []uint{24, 15, 2, 0}
	xRes := EliasGammaDecode(EliasGamma(xTest...), false)
	if !IntUintArrayEquals(xRes, xTest) {
		t.Error("EliasGamma failed with positive values, got ", xRes, " instead of ", xTest)
	}

	var xTestNeg = []int{3, -15, 123, -31, 0, 42}
	xResNeg := EliasGammaDecode(EliasGammaNegative(xTestNeg...), true)
	if !IntArrayEquals(xResNeg, xTestNeg) {
		t.Error("EliasGamma failed with negative values, got ", xResNeg, " instead of ", xTestNeg)
	}
}
