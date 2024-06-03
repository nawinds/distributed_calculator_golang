package tasks

import (
	"context"
	"distributed_calculator/expression_structs"
	"fmt"
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

func newTask(id, operTime, expressionID int, operator string, arg1, arg2 int) *Task {
	return &Task{ID: id,
		OperationTime:    operTime,
		ExpressionID:     expressionID,
		Operator:         operator,
		Arg1:             arg1,
		Arg2:             arg2,
		TimeoutTimestamp: time.Now().Add(time.Millisecond * time.Duration(operTime) * 2),
	}
}

func NewTasks() *Tasks {
	return &Tasks{Mx: sync.Mutex{}, lastID: 0, Tasks: make(map[int]*Task)}
}

func (t *Tasks) AddTask(time, expressionID int, operator string, arg1, arg2 int) string {
	t.Mx.Lock()
	defer t.Mx.Unlock()
	new_id := t.lastID + 1
	new_task := newTask(new_id, time, expressionID, operator, arg1, arg2)
	t.Tasks[t.lastID+1] = new_task
	t.lastID++

	return strconv.Itoa(new_id)
}

func (t *Tasks) GetTask(expressionsList *expression_structs.Expressions) (*Task, error) {
	t.Mx.Lock()
	defer t.Mx.Unlock()

	for _, task := range t.Tasks {
		if task.ContextCancel == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Millisecond*time.Duration(task.OperationTime))
			task.ContextCancel = cancel
			task.TimeoutTimestamp = time.Now().Add(2 * time.Millisecond * time.Duration(task.OperationTime))
			go t.monitorTask(ctx, task.ID, expressionsList)
			return task, nil
		}
	}
	return nil, fmt.Errorf("no task found")
}

func (t *Tasks) monitorTask(ctx context.Context, taskID int, expressionList *expression_structs.Expressions) {
	<-ctx.Done()
	t.Mx.Lock()
	defer t.Mx.Unlock()
	if task, exists := t.Tasks[taskID]; exists {
		if time.Now().After(task.TimeoutTimestamp) {
			fmt.Printf("Task #%d timed out and was removed\n", taskID)
			delete(t.Tasks, taskID)

			expressionList.Mx.Lock()
			for _, expr := range expressionList.Expressions {
				if expr.ID == task.ExpressionID {
					expr.Status = "Error: timeout"
				}
			}
			expressionList.Mx.Unlock()

			for _, tsk := range t.Tasks {
				if tsk.ExpressionID == task.ExpressionID {
					_, err := t.CompleteTask(tsk.ID)
					if err != nil {
						fmt.Printf("Error: %v\n", err)
					}
				}
			}
		}
	}
}

func (t *Tasks) CompleteTask(id int) (*Task, error) {
	t.Mx.Lock()
	defer t.Mx.Unlock()

	task, exists := t.Tasks[id]
	if !exists {
		return nil, fmt.Errorf("task not found")
	}

	if task.ContextCancel != nil {
		task.ContextCancel()
	}
	delete(t.Tasks, id)

	return task, nil
}
