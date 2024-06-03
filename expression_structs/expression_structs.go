package expression_structs

import "sync"

type Expression struct {
	ID         int
	Expression string
	Postfix    []string
	Status     string
	Result     int
}

type Expressions struct {
	Expressions map[int]*Expression // мапа с очередью выражений
	Mx          *sync.Mutex
	LastID      int // последний использованный id
}
