package prefixserver

import (
	"container/heap"
	"math/rand"
	"testing"
  "bytes"
  "encoding/gob"
)

var letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

func randBytes() []byte {

	b := make([]byte, rand.Int31n(15)+1)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return b
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

func makeFakeIndex(size int) (*Index, [][]byte) {

  index := NewIndex()
  keys := make([][]byte, size)

  for i := 0; i < size; i++ {
    keys[i] = randBytes()
    index.Add(keys[i], keys[i], i)
  }

  return index, keys

}

func TestIndexLittle(t *testing.T) {
	values := [][]byte{
		[]byte("r"),
		[]byte("rN"),
		[]byte("rW"),
	}

	index := NewIndex()
	for i := range values {
		index.Add(values[i], values[i], i)
	}

	outValues, _ := index.Find([]byte("r"))
	if len(outValues) != 3 {
		t.Fatalf("Expected 3 results for 'r' but got %d: %s", len(outValues), outValues)
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

	index := NewIndex()

	for i := range values {
		index.Add(values[i], values[i], i)
	}

	for i := range valsStartingWith {

		if len(valsStartingWith[i]) == 0 {
			continue
		}

		prefix := []byte{byte(i)}
		outValues, _ := index.Find(prefix)

		for j := range outValues {

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
		outValues, _ := index.Find(prefix)

		for j := range outValues {

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

  for i := 0; i < 10000; i++ {
    outValues, outScores := index.Find(keys[i])
    expectedValues[i] = outValues
    expectedScores[i] = outScores
  }

  newIndex := NewIndex()
  dec := gob.NewDecoder(bytes.NewReader(buf.Bytes()))
  dec.Decode(newIndex)

  // make sure all the keys and corresponding values are still in there
  for i := 0; i < 10000; i++ {
    outValues, outScores := index.Find(keys[i])

    if len(outValues) != len(expectedValues[i]) {
      t.Fatalf("on search for key %s, expected %d results, got %s", keys[i], len(expectedValues[i]), len(outValues))
    }

    for j := range outValues {
      if string(outValues[j]) != string(expectedValues[i][j]) {
        t.Errorf("on search for key %s, expected result %d to be %s, got %s", keys[i], expectedValues[i][j], outValues[i])
      }
      if outScores[j] != expectedScores[i][j] {
        t.Errorf("on search for key %s, expected score %d to be %d, got %d", keys[i], expectedScores[i][j], outScores[i])
      }
    }
  }

}

func BenchmarkIndexAdd(b *testing.B) {

	values := make([][]byte, b.N)
	for i := range values {
		values[i] = randBytes()
	}

	index := NewIndex()

	b.ResetTimer()

	for i := range values {
		index.Add(values[i], values[i], i)
	}

}
