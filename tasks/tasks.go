package tasks

import (
	"context"
	"strconv"
	"sync"
	"time"
)

type Task struct { // структура задачи
	ID               int       `json:"id"`
	ExpressionID     int       `json:"expression"` // выражение, к которому относится задача
	Operator         string    `json:"operation"`  // оператор арифметической операции
	Arg1             int       `json:"arg1"`
	Arg2             int       `json:"arg2"`
	OperationTime    int       `json:"operation_time"`    // время на выполнение операции
	TimeoutTimestamp time.Time `json:"timeout_timestamp"` // время, когда задача должна быть выполнена агентом,
	// который её принял
	ContextCancel context.CancelFunc `json:"-"` // функция отмены контекста задачи
}

type Tasks struct { // структура списка задач
	Tasks  map[int]*Task // мапа с очередью задач
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
