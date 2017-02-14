
package prefixserver

import (
  "bytes"
  "container/heap"
  "fmt"
)

type node struct {
  key []byte
  value []byte
  score int
  children []node
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
    n.children[i].printme(depth+1)
  }
}

type Index []node

func NewIndex() *Index {
  in := Index([]node{{key: []byte{}, children: []node{}}})
  return &in
}

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

    attachmentPoint.children = append(attachmentPoint.children, node{
      key: key[i:i+1],
      score: score,
      children: []node{},
    })
    attachmentPoint = &attachmentPoint.children[len(attachmentPoint.children)-1]
  }

  attachmentPoint.children = append(attachmentPoint.children, node{
    key: []byte{},
    score: score,
    children: []node{},
    value: value,
  })

}

func (in *Index) dfs(f func(n *node)) {

  stack := []*node{&(*in)[0]}

  for len(stack) > 0 {
    nextNode := stack[len(stack)-1]
    stack = stack[:len(stack)-1]
    f(nextNode)

    for i := range nextNode.children {
      stack = append(stack, &nextNode.children[i])
    }
  }

}

func (in *Index) Find(key []byte) (values [][]byte, scores []int) {

  values = make([][]byte, 0, 10)
  scores = make([]int, 0, 10)

  // initialize the priority queue through which we'll conduct our best-first search
  q := new_queue()
  heap.Init(q)
  heap.Push(q, &queueElement{node: &(*in)[0], prefix: key})

  for q.Len() > 0 {

    nextStop := heap.Pop(q).(*queueElement)
    nextNode := nextStop.node
    prefix := nextStop.prefix

    if len(prefix) == 0 || bytes.HasPrefix(prefix, nextNode.key) {

      if len(nextNode.key) - len(prefix) > 0 {
        prefix = []byte{}
      } else {
        prefix = prefix[len(nextNode.key):]
      }

      for i := range nextNode.children {
        heap.Push(q, &queueElement{node: &nextNode.children[i], prefix: prefix})
      }

      if len(prefix) == 0 && nextNode.value != nil {
        // consumed the whole prefix and this node has a value, so append it
        values = append(values, nextNode.value)
        scores = append(scores, nextNode.score)
        if len(values) == 10 {
          // hit the max number of results, so stop early
          return
        }
      }

    }

  }

  return

}
