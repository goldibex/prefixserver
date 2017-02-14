package prefixserver

import (
	"bytes"
	"container/heap"
  "container/list"
	"fmt"
  "encoding/gob"
)

type node struct {
	key      []byte
	value    []byte
	score    int
	children []node
}

func (n *node) printme(depth int) {
	for i := 0; i < depth; i++ {
		fmt.Printf(" ")
	}
	fmt.Printf("-> %s (%d)", n.key, n.score)
	if n.value != nil {
		fmt.Printf(" : %s", n.value)
	}
	fmt.Printf("\n")
	for i := range n.children {
		n.children[i].printme(depth + 1)
	}
}

// queue implements a priority queue for traversing nodes best-first.
type queueElement struct {
	*node
	prefix []byte
}

type queue []*queueElement

func new_queue() *queue {
	rawQ := queue([]*queueElement{})
	return &rawQ
}

func (q *queue) Len() int {
	return len(*q)
}

func (q *queue) Less(i, j int) bool {
	return (*q)[i].node.score > (*q)[j].node.score
}

func (q *queue) Swap(i, j int) {
	tmp := (*q)[i]
	(*q)[i] = (*q)[j]
	(*q)[j] = tmp
}

func (q *queue) Push(x interface{}) {
	*q = append(*q, x.(*queueElement))
}

func (q *queue) Pop() interface{} {

	item := (*q)[len(*q)-1]
	*q = (*q)[:len(*q)-1]

	return item

}

type Index []node

func New() *Index {
	in := Index([]node{{key: []byte{}, children: []node{}}})
	return &in
}

func (in *Index) dfs(f func(n *node)) {

	stack := []*node{&(*in)[0]}

	for len(stack) > 0 {
		nextNode := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		f(nextNode)

		for i := len(nextNode.children)-1; i >= 0; i-- {
			stack = append(stack, &nextNode.children[i])
		}
	}

}

func (in *Index) numNodes() int {
  n := 0
  in.dfs(func(_ *node) {
    n++
  })

  return n
}

// Add adds an entry to the index with the given value and score.
func (in *Index) Add(key []byte, value []byte, score int) {

	var attachmentPoint *node = &(*in)[0]

	found := true

	for found {

		found = false
		for i := range attachmentPoint.children {

			if attachmentPoint.children[i].value == nil && bytes.HasPrefix(key, attachmentPoint.children[i].key) {

				key = key[len(attachmentPoint.children[i].key):]
				attachmentPoint = &attachmentPoint.children[i]

				if score > attachmentPoint.score {
					attachmentPoint.score = score
				}

				found = true
				break

			}

		}

	}

	// split the remaining portion of key into one-byte pieces
	// and create a new tree of nodes at the attachment point
	for i := range key {

    newNode := node{
      key: key[i:i+1],
      score: score,
      children: make([]node, 0, 1),
    }

		attachmentPoint.children = append(attachmentPoint.children, newNode)
    attachmentPoint = &attachmentPoint.children[len(attachmentPoint.children)-1]
  }

	attachmentPoint.children = append(attachmentPoint.children, node{
		score:    score,
		value:    value,
	})

}

// Find locates up to len(values) matches to prefix, stores them in values and their scores in scores, and returns the total number of matches.
// Find panics if len(values) != len(scores).
func (in *Index) Find(key []byte, values [][]byte, scores []int) int {

	// initialize the priority queue through which we'll conduct our best-first search
	q := new_queue()
	heap.Init(q)
	heap.Push(q, &queueElement{node: &(*in)[0], prefix: key})
  matchCount := 0

	for q.Len() > 0 {

		nextStop := heap.Pop(q).(*queueElement)
		nextNode := nextStop.node
		prefix := nextStop.prefix

		if len(prefix) == 0 || bytes.HasPrefix(prefix, nextNode.key) {

			if len(nextNode.key)-len(prefix) > 0 {
				prefix = []byte{}
			} else {
				prefix = prefix[len(nextNode.key):]
			}

			for i := range nextNode.children {
				heap.Push(q, &queueElement{node: &nextNode.children[i], prefix: prefix})
			}

			if len(prefix) == 0 && nextNode.value != nil {
        // consumed the whole prefix and this node has a value, so append it
				values[matchCount] = nextNode.value
				scores[matchCount] = nextNode.score
				matchCount++
				if len(values) == matchCount {
					// hit the max number of results, so stop early
					return matchCount
				}
			}

		}

	}

	return matchCount

}

// Compact reduces the size of the index by merging redundant nodes out of the index.
// It is an error to call Add after having called Compact.
func (in *Index) Compact() {

  // the compacting process condenses nodes on straight-line paths together,
  // saving on the memory footprint and time cost of traversing these nodes separately.

	stack := []*node{&(*in)[0]}

	for len(stack) > 0 {

		nextNode := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

    for len(nextNode.children) == 1 && len(nextNode.children[0].children) > 0 {
      // absorb the child node

      nextNode.key = append(nextNode.key, nextNode.children[0].key...)
      nextNode.children = nextNode.children[0].children

    }

    for i := len(nextNode.children)-1; i >= 0; i-- {
      stack = append(stack, &nextNode.children[i])
    }

  }

}

type nodeGob struct {
  Keys [][]byte
  Values [][]byte
  Scores []int
  ChildListIndices []int
  ChildListLengths []int
}

// GobEncode implements encoding/gob's GobEncoder interface for serializing the index.
func (in *Index) GobEncode() ([]byte, error) {

  in.Compact()

  buf := bytes.Buffer{}
  enc := gob.NewEncoder(&buf)

  // do a first-line pass to count the number of nodes so we can avoid array resizing
  nodeCount := 0
  in.dfs(func(n *node) {
    nodeCount++
  })

  g := nodeGob{
    Keys: make([][]byte, nodeCount),
    Values: make([][]byte, nodeCount),
    Scores: make([]int, nodeCount),
    ChildListIndices: make([]int, nodeCount),
    ChildListLengths: make([]int, nodeCount),
  }

  l := list.New()
  l.PushFront(&(*in)[0])

  i := 0
  childListPos := 1

  for l.Len() > 0 {

    nextNode := l.Remove(l.Front()).(*node)

    g.ChildListIndices[i] = childListPos
    g.ChildListLengths[i] = len(nextNode.children)
    g.Keys[i] = nextNode.key
    g.Values[i] = nextNode.value
    g.Scores[i] = nextNode.score

    childListPos += len(nextNode.children)

    for j := range nextNode.children {
      l.PushBack(&nextNode.children[j])
    }

    i++

  }

  err := enc.Encode(&g)
  if err != nil {
    return nil, err
  }

  return buf.Bytes(), nil

}

// GobDecode implements encoding/gob's GobDecoder interface for deserializing the index.
func (in *Index) GobDecode(data []byte) error {

  var g nodeGob

  dec := gob.NewDecoder(bytes.NewBuffer(data))

  if err := dec.Decode(&g); err != nil {
    return err
  }

  nodes := make([]node, len(g.Keys))
  for i := range nodes {
    nodes[i].key = g.Keys[i]
    nodes[i].value = g.Values[i]
    nodes[i].score = g.Scores[i]
  }

  for i := range nodes {
    nodes[i].children = nodes[g.ChildListIndices[i]:g.ChildListIndices[i]+g.ChildListLengths[i]]
  }

  *in = nodes[0:1]

  return nil

}
