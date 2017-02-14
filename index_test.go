package prefixserver

import (
	"container/heap"
	"math/rand"
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

	values := make([][]byte, 1000000)

	for i := 1000000 - 1; i >= 0; i-- {

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
