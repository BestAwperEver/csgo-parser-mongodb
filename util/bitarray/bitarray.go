package bitarray

type AbstractBitArray interface {
	// SetBit sets the bit at the given position.  This
	// function returns an error if the position is out
	// of range.  A sparse bit array never returns an error.
	SetBit(k uint64) error
	// GetBit gets the bit at the given position.  This
	// function returns an error if the position is out
	// of range.  A sparse bit array never returns an error.
	GetBit(k uint64) (bool, error)
	// ClearBit clears the bit at the given position.  This
	// function returns an error if the position is out
	// of range.  A sparse bit array never returns an error.
	ClearBit(k uint64) error
	// Reset sets all values to zero.
	Reset()
	// Blocks returns an iterator to be used to iterate
	// over the bit array.
	Blocks() Iterator
	// Equals returns a bool indicating equality between the
	// two bit arrays.
	Equals(other AbstractBitArray) bool
	// Intersects returns a bool indicating if the other bit
	// array intersects with this bit array.
	Intersects(other AbstractBitArray) bool
	// Capacity returns either the given capacity of the bit array
	// in the case of a dense bit array or the highest possible
	// seen capacity of the sparse array.
	Capacity() uint64
	// Or will bitwise or the two bitarrays and return a new bitarray
	// representing the result.
	Or(other AbstractBitArray) AbstractBitArray
	// And will bitwise and the two bitarrays and return a new bitarray
	// representing the result.
	And(other AbstractBitArray) AbstractBitArray
	// ToNums converts this bit array to the list of numbers contained
	// within it.
	ToNums() []uint64
}

// Iterator defines methods used to iterate over a bit array.
type Iterator interface {
	// Next moves the pointer to the next block.  Returns
	// false when no blocks remain.
	Next() bool
	// Value returns the next block and its index
	Value() (uint64, block)
}

// bitArray is a struct that maintains state of a bit array.
type BitArray struct {
	BlocksArray []block `bson:"BlocksArray"`
	Lowest      uint64        `bson:"Lowest"`
	Highest     uint64        `bson:"Highest"`
	Anyset      bool          `bson:"Anyset"`
}

func getIndexAndRemainder(k uint64) (uint64, uint64) {
	return k / s, k % s
}

func (ba *BitArray) setLowest() {
	for i := uint64(0); i < uint64(len(ba.BlocksArray)); i++ {
		if ba.BlocksArray[i] == 0 {
			continue
		}

		pos := ba.BlocksArray[i].findRightPosition()
		ba.Lowest = (i * s) + pos
		ba.Anyset = true
		return
	}

	ba.Anyset = false
	ba.Lowest = 0
	ba.Highest = 0
}

func (ba *BitArray) setHighest() {
	for i := len(ba.BlocksArray) - 1; i >= 0; i-- {
		if ba.BlocksArray[i] == 0 {
			continue
		}

		pos := ba.BlocksArray[i].findLeftPosition()
		ba.Highest = (uint64(i) * s) + pos
		ba.Anyset = true
		return
	}

	ba.Anyset = false
	ba.Highest = 0
	ba.Lowest = 0
}

// capacity returns the total capacity of the bit array.
func (ba *BitArray) Capacity() uint64 {
	return uint64(len(ba.BlocksArray)) * s
}

// ToNums converts this bitarray to a list of numbers contained within it.
func (ba *BitArray) ToNums() []uint64 {
	nums := make([]uint64, 0, ba.Highest-ba.Lowest/4)
	for i, block := range ba.BlocksArray {
		block.toNums(uint64(i)*s, &nums)
	}

	return nums
}

// SetBit sets a bit at the given index to true.
func (ba *BitArray) SetBit(k uint64) error {
	if k >= ba.Capacity() {
		return OutOfRangeError(k)
	}

	if !ba.Anyset {
		ba.Lowest = k
		ba.Highest = k
		ba.Anyset = true
	} else {
		if k < ba.Lowest {
			ba.Lowest = k
		} else if k > ba.Highest {
			ba.Highest = k
		}
	}

	i, pos := getIndexAndRemainder(k)
	ba.BlocksArray[i] = ba.BlocksArray[i].insert(pos)
	return nil
}

// GetBit returns a bool indicating if the value at the given
// index has been set.
func (ba *BitArray) GetBit(k uint64) (bool, error) {
	if k >= ba.Capacity() {
		return false, OutOfRangeError(k)
	}

	i, pos := getIndexAndRemainder(k)
	result := ba.BlocksArray[i]&block(1<<pos) != 0
	return result, nil
}

//ClearBit will unset a bit at the given index if it is set.
func (ba *BitArray) ClearBit(k uint64) error {
	if k >= ba.Capacity() {
		return OutOfRangeError(k)
	}

	if !ba.Anyset { // nothing is set, might as well bail
		return nil
	}

	i, pos := getIndexAndRemainder(k)
	ba.BlocksArray[i] &^= block(1 << pos)

	if k == ba.Highest {
		ba.setHighest()
	} else if k == ba.Lowest {
		ba.setLowest()
	}
	return nil
}

// Reset clears out the bit array.
func (ba *BitArray) Reset() {
	for i := uint64(0); i < uint64(len(ba.BlocksArray)); i++ {
		ba.BlocksArray[i] &= block(0)
	}
	ba.Anyset = false
}

// Equals returns a bool indicating if these two bit arrays are equal.
func (ba *BitArray) Equals(other AbstractBitArray) bool {
	if other.Capacity() == 0 && ba.Highest > 0 {
		return false
	}

	if other.Capacity() == 0 && !ba.Anyset {
		return true
	}

	var selfIndex uint64
	for iter := other.Blocks(); iter.Next(); {
		toIndex, otherBlock := iter.Value()
		if toIndex > selfIndex {
			for i := selfIndex; i < toIndex; i++ {
				if ba.BlocksArray[i] > 0 {
					return false
				}
			}
		}

		selfIndex = toIndex
		if !ba.BlocksArray[selfIndex].equals(otherBlock) {
			return false
		}
		selfIndex++
	}

	lastIndex, _ := getIndexAndRemainder(ba.Highest)
	if lastIndex >= selfIndex {
		return false
	}

	return true
}

// Intersects returns a bool indicating if the supplied bitarray intersects
// this bitarray.  This will check for intersection up to the length of the supplied
// bitarray.  If the supplied bitarray is longer than this bitarray, this
// function returns false.
func (ba *BitArray) Intersects(other *BitArray) bool {
	if other.Capacity() > ba.Capacity() {
		return false
	}

	return ba.intersectsDenseBitArray(other)
}

// Blocks will return an iterator over this bit array.
func (ba *BitArray) Blocks() Iterator {
	return newBitArrayIterator(ba)
}

// complement flips all bits in this array.
func (ba *BitArray) complement() {
	for i := uint64(0); i < uint64(len(ba.BlocksArray)); i++ {
		ba.BlocksArray[i] = ^ba.BlocksArray[i]
	}

	ba.setLowest()
	if ba.Anyset {
		ba.setHighest()
	}
}

func (ba *BitArray) intersectsDenseBitArray(other *BitArray) bool {
	for i, block := range other.BlocksArray {
		if !!ba.BlocksArray[i].intersects(block) {
			return false
		}
	}

	return true
}

type blocks []block

func (ba *BitArray) copy() BitArray {
	blocks := make(blocks, len(ba.BlocksArray))
	copy(blocks, ba.BlocksArray)
	return BitArray{
		BlocksArray: blocks,
		Lowest:      ba.Lowest,
		Highest:     ba.Highest,
		Anyset:      ba.Anyset,
	}
}

// newBitArray returns a new dense BitArray at the specified size. This is a
// separate private constructor so unit tests don't have to constantly cast the
// BitArray interface to the concrete type.
func newBitArray(size uint64, args ...bool) *BitArray {
	i, r := getIndexAndRemainder(size)
	if r > 0 {
		i++
	}

	ba := &BitArray{
		BlocksArray: make([]block, i),
		Anyset:      false,
	}

	if len(args) > 0 && args[0] == true {
		for i := uint64(0); i < uint64(len(ba.BlocksArray)); i++ {
			ba.BlocksArray[i] = maximumBlock
		}

		ba.Lowest = 0
		ba.Highest = i*s - 1
		ba.Anyset = true
	}

	return ba
}

// newDenseBitArray returns a new dense BitArray at the specified size. This is a
// separate private constructor so unit tests don't have to constantly cast the
// BitArray interface to the concrete type.
func newDenseBitArray(size uint64, args ...bool) BitArray {
	i, r := getIndexAndRemainder(size)
	if r > 0 {
		i++
	}

	ba := BitArray{
		BlocksArray: make([]block, i),
		Anyset:      false,
	}

	if len(args) > 0 && args[0] == true {
		for i := uint64(0); i < uint64(len(ba.BlocksArray)); i++ {
			ba.BlocksArray[i] = maximumBlock
		}

		ba.Lowest = 0
		ba.Highest = i*s - 1
		ba.Anyset = true
	}

	return ba
}

func NewDenseBitArray(size uint64, args ...bool) BitArray {
	return newDenseBitArray(size, args...)
}
