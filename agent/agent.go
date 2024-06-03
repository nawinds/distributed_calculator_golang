package agent

import (
	"bytes"
	"distributed_calculator/tasks"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func Worker() {
	for {
		task, err := getTask()
		if err != nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		result, e := performTask(task)
		postTaskResult(task.ID, result, e)
	}
}

func getTask() (*tasks.Task, error) {
	resp, err := http.Get("http://localhost:8080/internal/task")
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("no task available")
	}

	var taskResponse struct {
		Task tasks.Task `json:"task"`
	}
	err = json.NewDecoder(resp.Body).Decode(&taskResponse)
	if err != nil {
		return nil, err
	}

	return &taskResponse.Task, nil
}

func performTask(task *tasks.Task) (int, error) {
	time.Sleep(time.Duration(task.OperationTime) * time.Millisecond)

	arg1 := task.Arg1
	arg2 := task.Arg2

	switch task.Operator {
	case "+":
		return arg1 + arg2, nil
	case "-":
		return arg1 - arg2, nil
	case "*":
		return arg1 * arg2, nil
	case "/":
		if arg2 != 0 {
			return arg1 / arg2, nil
		} else {
			return 0, fmt.Errorf("division by zero")
		}
	}
	return 0, fmt.Errorf("unknown operator")
}

func postTaskResult(id, result int, e error) {
	errString := ""
	if e != nil {
		errString = e.Error()
	} else {
		errString = ""
	}
	resultData := map[string]interface{}{
		"id":     id,
		"result": result,
		"error":  errString,
	}

	data, _ := json.Marshal(resultData)
	_, err := http.Post("http://localhost:8080/internal/task", "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Println("Failed to post task result:", err)
	}
}
