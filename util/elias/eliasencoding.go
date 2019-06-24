package elias

import (
	"fmt"
	//"github.com/golang-collections/go-datastructures/bitarray"
	"csgo-parser-mongodb/util/bitarray"
	"math/bits"
	"strconv"
	"strings"
)

type BitArrayWithLength struct {
	length uint64
	bitarray.BitArray
}

//func (ba BitArrayWithLength) GetBSON() (interface{}, error) {
//	return struct {
//		Length		uint64
//		BlocksArray []int64
//		Lowest      uint64
//		Highest     uint64
//		Anyset      bool
//	}{
//		Length:			ba.length,
//		BlocksArray:	ba.BlocksArray,
//		Lowest:			ba.Lowest,
//		Highest:		ba.Highest,
//		Anyset:			ba.Anyset,
//	}, nil
//}

func stringToBitArray(bs string) BitArrayWithLength {
	result := BitArrayWithLength {
		uint64(len(bs)),
		bitarray.NewDenseBitArray(uint64(len(bs))),
	}
	for i, c := range bs {
		if c == '1' {
			err := result.SetBit(uint64(i))
			checkError(err)
		}
	}
	return result
}

func binary(x, l uint) string {
	return fmt.Sprintf("%0"+fmt.Sprint(l)+"s", strconv.FormatInt(int64(x), 2))
}

func unary(x uint) string {
	return strings.Repeat("0", int(x-1)) + "1"
}

func eliasGeneric(lencoding func(uint) string, a uint) string {
	if a == 1 {
		return "1"
	}
	n := uint(bits.Len(a) - 1)
	a1 := a - (1 << n)
	return lencoding(n+1) + binary(a1, n)
}

func eliasGammaS(x uint) string {
	return eliasGeneric(unary, x)
}

func eliasDeltaS(x uint) string {
	return eliasGeneric(eliasGammaS, x)
}

func EliasGamma(x ...uint) BitArrayWithLength {
	result := ""
	for _, v := range x {
		v++
		result += eliasGammaS(v)
	}
	return stringToBitArray(result)
}

func EliasDelta(x ...uint) BitArrayWithLength {
	result := ""
	for _, v := range x {
		v++
		result += eliasDeltaS(v)
	}
	return stringToBitArray(result)
}

func EliasGammaNegative(x ...int) BitArrayWithLength {
	result := ""
	for _, v := range x {
		if v < 0 {
			result += eliasGammaS(uint(-v * 2))
		} else {
			result += eliasGammaS(uint(v*2 + 1))
		}
	}
	return stringToBitArray(result)
}

func EliasDeltaNegative(x ...int) BitArrayWithLength {
	result := ""
	for _, v := range x {
		if v < 0 {
			result += eliasDeltaS(uint(-v * 2))
		} else {
			result += eliasDeltaS(uint(v*2 + 1))
		}
	}
	return stringToBitArray(result)
}

func EliasGammaDecode(ba BitArrayWithLength, possibleNegative bool) []int {

	var numbers []int
	interpretAsBinary := false
	var k, j uint64 = 0, 0

	for j < ba.length {
		if interpretAsBinary == false {
			value, err := ba.GetBit(j)
			checkError(err)
			j++
			if value == false {
				k++
			} else {
				interpretAsBinary = true
			}
		}
		if interpretAsBinary {
			a := 1 << k
			for	i := k; i > 0; i-- {
				value, err := ba.GetBit(j)
				checkError(err)
				j++
				if value {
					a += 1 << (i - 1)
				}
			}
			if !possibleNegative {
				numbers = append(numbers, a-1)
			} else {
				if a % 2 == 0 {
					numbers = append(numbers, -a/2)
				} else {
					numbers = append(numbers, (a-1)/2)
				}
			}
			interpretAsBinary = false
			k = 0
		}
	}

	return numbers
}

func EliasDeltaDecode(ba BitArrayWithLength, possibleNegative bool) []int {

	var numbers []int
	interpretAsBinary := false
	var k, j uint64 = 0, 0

	for j < ba.length {
		if interpretAsBinary == false {
			value, err := ba.GetBit(j)
			checkError(err)
			j++
			if value == false {
				k++
			} else {
				interpretAsBinary = true
			}
		}
		if interpretAsBinary {
			a := uint(1 << k)
			for	i := k; i > 0; i-- {
				value, err := ba.GetBit(j)
				checkError(err)
				j++
				if value {
					a += 1 << (i - 1)
				}
			}
			b := 1 << (a-1)
			for i := a-1; i > 0; i-- {
				value, err := ba.GetBit(j)
				checkError(err)
				j++
				if value {
					b += 1 << (i - 1)
				}
			}
			if !possibleNegative {
				numbers = append(numbers, b-1)
			} else {
				if b % 2 == 0 {
					numbers = append(numbers, -b/2)
				} else {
					numbers = append(numbers, (b-1)/2)
				}
			}
			interpretAsBinary = false
			k = 0
		}
	}

	return numbers
}

func ArrayToDeltas(x []int) []int {
	xDeltas := make([]int, len(x))
	xDeltas[0] = x[0]
	for i, v := range x {
		if i == 0 {
			continue
		}
		xDeltas[i] = v - x[i-1]
	}
	return xDeltas
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}