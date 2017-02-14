package prefixserver

import (
	"bytes"
	"container/heap"
	"encoding/gob"
	"math/rand"
	"runtime"
	"testing"
)

var letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

func randBytes() []byte {

	b := make([]byte, rand.Int31n(15)+1)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return b

}

func makeFakeIndex(size int) (*Index, [][]byte) {

	index := New()
	keys := make([][]byte, size)

	for i := 0; i < size; i++ {
		keys[i] = randBytes()
		index.Add(keys[i], keys[i], i)
	}

	return index, keys

}

func Test_queue(t *testing.T) {

	nodes := []*queueElement{
		{
			node: &node{
				key:   []byte("a"),
				score: 2,
			},
		},
		{
			node: &node{
				key:   []byte("b"),
				score: 3,
			},
		},
		{
			node: &node{
				key:   []byte("c"),
				score: 1,
			},
		},
	}
	q := new_queue()
	heap.Init(q)
	heap.Push(q, nodes[0])
	heap.Push(q, nodes[1])
	heap.Push(q, nodes[2])

	r1 := heap.Pop(q).(*queueElement)
	r2 := heap.Pop(q).(*queueElement)
	r3 := heap.Pop(q).(*queueElement)

	if string(r1.key) != "b" || string(r2.key) != "a" || string(r3.key) != "c" {
		t.Errorf("Wrong order for results: got r1=%+v, r2=%+v, r3=%+v", r1, r2, r3)
	}

	if q.Len() != 0 {
		t.Errorf("Expected queue to have length 0, but got %d", q.Len())
	}

}

func TestIndexLittle(t *testing.T) {
	values := [][]byte{
		[]byte("r"),
		[]byte("rN"),
		[]byte("rW"),
	}

	index := New()
	for i := range values {
		index.Add(values[i], values[i], i)
	}

	outValues := make([][]byte, 10)
	outScores := make([]int, 10)

	count := index.Find([]byte("r"), outValues, outScores)
	if count != 3 {
		t.Fatalf("Expected 3 results for 'r' but got %d: %s", count, outValues[0:count])
	}

	if string(outValues[0]) != "rW" {
		t.Errorf("Bad outValues: %s", outValues)
	}

}

func TestIndex(t *testing.T) {

	valsStartingWith := make([][][]byte, 256)

	seenValues := map[string]bool{}

	values := make([][]byte, 100000)

	for i := 100000 - 1; i >= 0; i-- {

		for values[i] == nil || seenValues[string(values[i])] {
			values[i] = randBytes()
		}

		if valsStartingWith[values[i][0]] == nil {
			valsStartingWith[values[i][0]] = make([][]byte, 0, 10)
		}

		valsStartingWith[values[i][0]] = append(valsStartingWith[values[i][0]], values[i])
	}

	index := New()

	for i := range values {
		index.Add(values[i], values[i], i)
	}

	outValues := make([][]byte, 10)
	outScores := make([]int, 10)

	for i := range valsStartingWith {

		if len(valsStartingWith[i]) == 0 {
			continue
		}

		prefix := []byte{byte(i)}
		count := index.Find(prefix, outValues, outScores)

		for j := 0; j < count; j++ {

			if string(outValues[j]) != string(valsStartingWith[i][j]) {
				t.Fatalf("for starting value %d: index %d: bad value: %s, expected %s", i, j, outValues[j], string(valsStartingWith[i][j]))
			}
		}
	}

	sizeBeforeCompacting := index.numNodes()

	index.Compact()

	t.Logf("size before compacting: %d after compacting: %d", sizeBeforeCompacting, index.numNodes())

	for i := range valsStartingWith {

		if len(valsStartingWith[i]) == 0 {
			continue
		}

		prefix := []byte{byte(i)}
		count := index.Find(prefix, outValues, outScores)

		for j := 0; j < count; j++ {

			if string(outValues[j]) != string(valsStartingWith[i][j]) {
				t.Fatalf("for starting value %d: index %d: bad value: %s, expected %s", i, j, outValues[j], string(valsStartingWith[i][j]))
			}
		}
	}

}

func TestIndexGob(t *testing.T) {

	index, keys := makeFakeIndex(100000)

	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	enc.Encode(index)

	expectedValues := make([][][]byte, 10000)
	expectedScores := make([][]int, 10000)

	outValues := make([][]byte, 10)
	outScores := make([]int, 10)

	for i := 0; i < 10000; i++ {
		count := index.Find(keys[i], outValues, outScores)

		expectedValues[i] = make([][]byte, count)
		expectedScores[i] = make([]int, count)

		copy(expectedValues[i], outValues)
		copy(expectedScores[i], outScores)

	}

	newIndex := New()
	dec := gob.NewDecoder(bytes.NewReader(buf.Bytes()))
	dec.Decode(newIndex)

	// make sure all the keys and corresponding values are still in there
	for i := 0; i < 10000; i++ {
		count := index.Find(keys[i], outValues, outScores)

		if count != len(expectedValues[i]) {
			t.Fatalf("on search for key %s, expected %d results, got %s", keys[i], len(expectedValues[i]), count)
		}

		for j := 0; j < count; j++ {
			if string(outValues[j]) != string(expectedValues[i][j]) {
				t.Errorf("on search for key %s, expected result %d to be %s, got %s", keys[i], expectedValues[i][j], outValues[i])
			}
			if outScores[j] != expectedScores[i][j] {
				t.Errorf("on search for key %s, expected score %d to be %d, got %d", keys[i], expectedScores[i][j], outScores[i])
			}
		}
	}

}

// BenchmarkIndexAdd tests the amount of time required to add an item to an index.
func BenchmarkIndexAdd(b *testing.B) {

	values := make([][]byte, b.N)
	for i := range values {
		values[i] = randBytes()
	}

	index := New()

	b.ResetTimer()

	for i := range values {
		index.Add(values[i], values[i], i)
	}

}

// BenchmarkIndexCompact tests the amount of time required to compact an index with 2 million random entries.
func BenchmarkIndexCompact(b *testing.B) {

	index, _ := makeFakeIndex(2000000)
	runtime.GC()
	b.ResetTimer()

	index.Compact()

}

// BenchmarkIndexFind tests the amount of time required to find up to 100 results for an item in an index of 2 million random entries.
func BenchmarkIndexFind(b *testing.B) {

	index, keys := makeFakeIndex(2000000)
	index.Compact()
	runtime.GC()
	b.ResetTimer()

	outValues := make([][]byte, 100)
	outScores := make([]int, 100)

	for i := 0; i < b.N; i++ {
		pos := rand.Int31n(2000000)
		index.Find(keys[pos], outValues, outScores)
	}

}

// BenchmarkIndexEncode tests the amount of time required to serialize an index with 2 million random entries.
func BenchmarkIndexEncode(b *testing.B) {

	index, _ := makeFakeIndex(2000000)
	index.Compact()
	runtime.GC()
	b.ResetTimer()

	buf := bytes.NewBuffer(make([]byte, 0, 10000000))
	enc := gob.NewEncoder(buf)
	enc.Encode(index)

	b.Logf("Encoded index size: %d bytes", len(buf.Bytes()))

}

// BenchmarkIndexDecode tests the amount of time required to deserialize an index with 2 million random entries.
func BenchmarkIndexDecode(b *testing.B) {

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		index, _ := makeFakeIndex(2000000)
		buf := bytes.NewBuffer(make([]byte, 0, 10000000))
		enc := gob.NewEncoder(buf)
		enc.Encode(index)
		runtime.GC()
		b.StartTimer()

		newIndex := New()
		dec := gob.NewDecoder(bytes.NewReader(buf.Bytes()))
		dec.Decode(newIndex)
	}

}
