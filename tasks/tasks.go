package tasks

import (
	"context"
	"strconv"
	"sync"
	"time"
)

type Task struct {
	ID               int                `json:"id"`
	ExpressionID     int                `json:"expression"`
	Operator         string             `json:"operation"`
	Arg1             int                `json:"arg1"`
	Arg2             int                `json:"arg2"`
	OperationTime    int                `json:"operation_time"`
	TimeoutTimestamp time.Time          `json:"timeout_timestamp"`
	ContextCancel    context.CancelFunc `json:"-"`
}

type Tasks struct {
	Tasks  map[int]*Task
	Mx     sync.Mutex
	lastID int
}

func newTask(id, time, expressionID int, operator string, arg1, arg2 int) *Task {
	return &Task{ID: id,
		OperationTime: time,
		ExpressionID:  expressionID,
		Operator:      operator,
		Arg1:          arg1,
		Arg2:          arg2}
}

func NewTasks() *Tasks {
	return &Tasks{Mx: sync.Mutex{}, lastID: 0, Tasks: make(map[int]*Task)}
}

func (t *Tasks) AddTask(time, expressionID int, operator string, arg1, arg2 int) string {
	new_id := t.lastID + 1
	new_task := newTask(new_id, time, expressionID, operator, arg1, arg2)
	t.Mx.Lock()
	t.Tasks[t.lastID+1] = new_task
	t.lastID++
	t.Mx.Unlock()
	return strconv.Itoa(new_id)
}
