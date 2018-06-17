package heap

import (
	"math"
	"wwt/util/queue"
	"sync"
)

type Compareable interface {
	//	return true means this less than other
	//	return false means this greater than other
	Compare(other Compareable) bool
}

type HeapNode struct {
	Value interface {
		Compareable
	}
	Left   *HeapNode
	Right  *HeapNode
	Parent *HeapNode
}

func NewHeapNode(value Compareable, left, right *HeapNode, parent *HeapNode) *HeapNode {
	return &HeapNode{value, left, right, parent}
}

func NewHeap() *QHeap {
	return (&QHeap{}).Init()
}

type QHeap struct {
	Root *HeapNode
	Last *HeapNode
	Size int
}

func (this *QHeap) Init() *QHeap {
	this.Root = nil
	this.Last = nil
	this.Size = 0
	return this
}

func (this *QHeap) findInsertNode() *HeapNode {
	deep := int(math.Log2(float64(this.Size + 1)))
	fullSize := int(math.Pow(2, float64(deep)) - 1)
	var res *HeapNode = nil
	if fullSize == this.Size {
		//	满二叉树
		now := this.Root
		for now.Left != nil {
			now = now.Left
		}
		res = now
	} else {
		//queue := queue.NewQueue()
		//queue.Enqueue(this.Root)
		//for queue.Size() > 0 {
		//	now := queue.Dequeue().(*HeapNode)
		//	if now.Left == nil || now.Right == nil {
		//		res = now
		//		break
		//	}
		//	if now.Left != nil {
		//		queue.Enqueue(now.Left)
		//	}
		//	if now.Right != nil {
		//		queue.Enqueue(now.Right)
		//	}
		//}
		c := this.Last
		pc := c.Parent
		if pc.Left==c{
			//	c 是 pc 的 左子节点
			//	则 pc 一定不满
			res = pc
		}else {
			//	c 是 pc 的 右子节点
			//	pc 一定满

			//	遍历整棵树，寻找pc的右兄弟
			ppc := pc.Parent
			//	寻找拐点
			for ppc != nil && pc == ppc.Right {
				c = pc
				pc = pc.Parent
				ppc = pc.Parent
			}
			c = ppc.Right
			for c.Left != nil {
				c = c.Left
			}
			res = c
		}
	}
	return res

}

func (this *QHeap) AddNode(value Compareable) {
	node := NewHeapNode(value, nil, nil, nil)
	if this.Root == nil {
		this.Root = node
	} else {
		insertNode := this.findInsertNode()
		if insertNode.Left == nil {
			insertNode.Left = node
		} else {
			insertNode.Right = node
		}
		node.Parent = insertNode
		this.adjust(insertNode)
	}
	this.Size++
	this.Last = node
}

func (this *QHeap) Front() interface{} {
	return this.Root.Value
}

func (this *QHeap) Len()int{
	return this.Size
}

func (this *QHeap) Pop() interface{} {
	res := this.Root.Value
	this.Last.Value,this.Root.Value = this.Root.Value,this.Last.Value
	if this.Last == this.Root {
		this.Last = nil
		this.Root = nil
	} else {
		OLast := this.Last
		this.updateLast()
		if OLast == OLast.Parent.Left {
			OLast.Parent.Left = nil
		} else {
			OLast.Parent.Right = nil
		}
		OLast.Parent = nil
	}
	this.Size--
	if this.Size>0 {
		node := this.Root
		for node.Left != nil || node.Right != nil {
			selected := node

			if node.Left != nil && node.Left.Value.Compare(selected.Value) {
				//	Left less than selected
				selected = node.Left
			}
			if node.Right != nil && node.Right.Value.Compare(selected.Value) {
				selected = node.Right
			}
			if node == selected {
				break
			} else {
				selected.Value, node.Value = node.Value, selected.Value
				node = selected
			}
		}
	}

	return res
}

func (this *QHeap) updateLast() {
	p := this.Last.Parent
	l := this.Last
	for p != nil && l == p.Left {
		//fmt.Printf("P: %p. l: %p.\n",p,l)
		l, p = p, p.Parent
	}
	var nl *HeapNode
	if p != nil {
		nl = p.Left
	} else {
		nl = l
	}
	for nl.Right != nil {
		nl = nl.Right
	}
	this.Last = nl
}

func (this *QHeap) adjust(root *HeapNode) {
	for root != nil {
		selected := root
		if root.Left != nil && root.Left.Value.Compare(selected.Value) {
			//	Left < root
			selected = root.Left
		}
		if (root.Right != nil && root.Right.Value.Compare(selected.Value)) {
			// Right < root
			selected = root.Right
		}
		if selected == root {
			break
		}
		selected.Value, root.Value = root.Value, selected.Value
		root = root.Parent
	}
}

type SafeHeap struct{
	heap *QHeap
	mu sync.Mutex
}

func NewSafeHeap()*SafeHeap{
	res := SafeHeap{}
	res.heap=NewHeap()
	res.mu=sync.Mutex{}
	return &res
}

func (this *SafeHeap) AddNode(value Compareable) {
	this.heap.AddNode(value)
}

func (this *SafeHeap) Front() interface{} {
	return this.heap.Root.Value
}

func (this *SafeHeap) Pop() interface{} {
	return this.heap.Pop()
}

func (this *SafeHeap)Len()int{
	return this.heap.Size
}

func (this *SafeHeap)Lock(){
	this.mu.Lock()
}

func (this *SafeHeap)UnLock(){
	this.mu.Unlock()
}
