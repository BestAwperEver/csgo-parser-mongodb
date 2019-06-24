package bitarray

type bitArrayIterator struct {
	index     int64
	stopIndex uint64
	ba        *BitArray
}

// Next increments the index and returns a bool indicating if any further
// items exist.
func (iter *bitArrayIterator) Next() bool {
	iter.index++
	return uint64(iter.index) <= iter.stopIndex
}

// Value returns an index and the block at this index.
func (iter *bitArrayIterator) Value() (uint64, block) {
	return uint64(iter.index), iter.ba.BlocksArray[iter.index]
}

func newBitArrayIterator(ba *BitArray) *bitArrayIterator {
	stop, _ := getIndexAndRemainder(ba.Highest)
	start, _ := getIndexAndRemainder(ba.Lowest)
	return &bitArrayIterator{
		ba:        ba,
		index:     int64(start) - 1,
		stopIndex: stop,
	}
}
