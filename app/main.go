package main

import (
	"distributed_calculator/agent"
	"distributed_calculator/config"
	"distributed_calculator/evaluation"
	"distributed_calculator/tasks"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type Expression struct {
	ID         int
	Expression string
	Postfix    []string
	Status     string
	Result     int
}

type ExpressionItem struct { // структура выражения для вывода в API
	ID     int
	Status string
	Result int
}

type Expressions struct {
	Expressions map[int]*Expression // мапа с очередью выражений
	Mx          *sync.Mutex
	LastID      int // последний использованный id
}

type ExpressionResponse struct { // структура для возврата списка выражений через API
	Expressions []ExpressionItem
}

var (
	expressionsList = NewExpressions()
	tasksList       = tasks.NewTasks()
)

func NewExpressions() *Expressions {
	return &Expressions{Mx: &sync.Mutex{}, LastID: 0, Expressions: make(map[int]*Expression)}
}

func NewExpression(id int, exp string) *Expression {
	return &Expression{ID: id, Expression: exp, Status: "Processing"}
}

func addExpressionHandler(w http.ResponseWriter, r *http.Request) {
	type RequestData struct {
		Expression string `json:"expression"`
	}
	type ResponseData struct {
		ID string `json:"id"`
	}
	var data RequestData

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	expression := data.Expression
	postfix, err := evaluation.InfixToPostfix(expression)
	if err != nil {
		http.Error(w, "Invalid expression", http.StatusBadRequest)
		return
	}
	expressionsList.Mx.Lock()
	defer expressionsList.Mx.Unlock()
	id := expressionsList.LastID + 1
	newExpression := NewExpression(id, expression)
	expressionsList.Expressions[id] = newExpression
	expressionsList.LastID = id

	fmt.Println("Postfix Expression:", strings.Join(postfix, " "))

	go func(id int, postfix []string) {
		newPostfix, err := evaluation.EvaluatePostfix(id, tasksList, postfix)
		expressionsList.Mx.Lock()
		defer expressionsList.Mx.Unlock()

		expression := expressionsList.Expressions[id]
		if err != nil && err.Error() == "unready warning" {
			expression.Postfix = newPostfix
		} else if err == nil {
			result, _ := strconv.Atoi(newPostfix[0])
			expression.Result = result
			expression.Status = "Done"
		} else {
			expression.Status = "Error"
		}
	}(id, postfix)

	w.WriteHeader(http.StatusCreated) // 201
	w.Header().Set("Content-Type", "application/json")
	e := json.NewEncoder(w).Encode(&ResponseData{ID: strconv.Itoa(id)})
	if e != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError) // 500
		return
	}
	fmt.Println(expression)
}

func getExpressionsHandler(w http.ResponseWriter, _ *http.Request) {
	var expressions []ExpressionItem

	expressionsList.Mx.Lock()
	for _, value := range expressionsList.Expressions {
		expressions = append(expressions, ExpressionItem{
			ID:     value.ID,
			Status: value.Status,
			Result: value.Result,
		})
	}
	expressionsList.Mx.Unlock()

	response := ExpressionResponse{
		Expressions: expressions,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200
	e := json.NewEncoder(w).Encode(response)
	if e != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError) // 500
		return
	}
}

func getExpressionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest) // 400
		return
	}
	expressionsList.Mx.Lock()
	expr, exist := expressionsList.Expressions[id]
	expressionsList.Mx.Unlock()
	if !exist {
		http.Error(w, "Expression does not exist", http.StatusNotFound)
		return
	}
	response := struct {
		ID     int    `json:"id"`
		Status string `json:"status"`
		Result int    `json:"result"`
	}{
		ID:     expr.ID,
		Status: expr.Status,
		Result: expr.Result,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	e := json.NewEncoder(w).Encode(response)
	if e != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError) // 500
		return
	}
}

func getTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		task, err := tasksList.GetTask()
		if err != nil {
			http.Error(w, "No task found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(map[string]*tasks.Task{"task": task})
		if err != nil {
			http.Error(w, "json encode error", http.StatusInternalServerError)
		}
		return
	} else if r.Method == http.MethodPost {
		var result struct {
			ID     int `json:"id"`
			Result int `json:"result"`
		}
		err := json.NewDecoder(r.Body).Decode(&result)
		if err != nil {
			http.Error(w, "Invalid data", http.StatusUnprocessableEntity)
			return
		}

		task, err := tasksList.CompleteTask(result.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		fmt.Println("Task ID:", result.ID)

		expressionsList.Mx.Lock()
		defer expressionsList.Mx.Unlock()
		expr, found := expressionsList.Expressions[task.ExpressionID]
		if !found {
			http.Error(w, "Expression not found", http.StatusNotFound)
			return
		}
		for i, v := range expr.Postfix {
			if v == "t"+strconv.Itoa(task.ID) {
				expr.Postfix[i] = strconv.Itoa(result.Result)
			}
		}

		go func(exprID int, tasksList *tasks.Tasks, exprPostfix []string) {
			newPostfix, err := evaluation.EvaluatePostfix(exprID, tasksList, exprPostfix)
			expressionsList.Mx.Lock()
			defer expressionsList.Mx.Unlock()

			expression := expressionsList.Expressions[exprID]
			if err != nil && err.Error() == "unready warning" {
				expression.Postfix = newPostfix
			} else if err == nil {
				result, _ := strconv.Atoi(newPostfix[0])
				expression.Result = result
				expression.Status = "Done"
			} else {
				expression.Status = "Error"
			}
		}(expr.ID, tasksList, expr.Postfix)

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, e := w.Write([]byte(`{}`))
		if e != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		return
	}
	http.Error(w, "unsupported method", http.StatusMethodNotAllowed)
	return
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/calculate", addExpressionHandler).Methods("POST")
	r.HandleFunc("/api/v1/expressions", getExpressionsHandler).Methods("GET")
	r.HandleFunc("/api/v1/expressions/{id}", getExpressionHandler).Methods("GET")
	r.HandleFunc("/internal/task", getTaskHandler).Methods("GET", "POST")

	for i := 0; i < config.COMPUTING_POWER; i++ {
		go agent.Worker()
	}

	err := http.ListenAndServe(":8080", r)
	if err != nil {
		panic(err)
	}
}
