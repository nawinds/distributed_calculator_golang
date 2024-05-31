package tasks

import (
	"strconv"
	"sync"
)

type Task struct {
	ID           int    `json:"id"`
	ExpressionID int    `json:"expression"`
	Operator     string `json:"operation"`
	Arg1         int    `json:"arg1"`
	Arg2         int    `json:"arg2"`
}

type Tasks struct {
	Tasks  map[int]*Task
	Mu     sync.Mutex
	lastID int
}

func newTask(id int, expressionID int, operator string, arg1 int, arg2 int) *Task {
	return &Task{ID: id,
		ExpressionID: expressionID,
		Operator:     operator,
		Arg1:         arg1,
		Arg2:         arg2}
}

func NewTasks() *Tasks {
	return &Tasks{Mu: sync.Mutex{}, lastID: 0, Tasks: make(map[int]*Task)}
}

func (t *Tasks) AddTask(expressionID int, operator string, arg1 int, arg2 int) string {
	new_id := t.lastID + 1
	new_task := newTask(new_id, expressionID, operator, arg1, arg2)
	t.Mu.Lock()
	t.Tasks[t.lastID+1] = new_task
	t.lastID++
	t.Mu.Unlock()
	return strconv.Itoa(new_id)
}
