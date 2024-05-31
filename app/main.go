package main

import (
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

type ExpressionItem struct {
	ID     int
	Status string
	Result int
}

type Expressions struct {
	Expressions map[int]*Expression
	Mx          *sync.Mutex
	LastID      int
}

type ExpressionResponse struct {
	Expressions []ExpressionItem
}

var (
	expressionsList = NewExpressions()
	tasksList       = tasks.NewTasks()
)

func NewExpressions() *Expressions {
	return &Expressions{Mx: &sync.Mutex{}, LastID: 0, Expressions: make(map[int]*Expression)}
}

func (q *Expressions) getStorageLen() int {
	q.Mx.Lock()
	defer q.Mx.Unlock()
	return len(q.Expressions)
}

func (q *Expressions) addExpression(exp *Expression) {
	q.Mx.Lock()
	defer q.Mx.Unlock()
	q.Expressions[q.LastID+1] = exp
	q.LastID++
}

func (q *Expressions) removeExpression(id int) *Expression {
	q.Mx.Lock()
	defer q.Mx.Unlock()

	res := q.Expressions[id]
	delete(q.Expressions, id)
	return res
}

func newExpression(id int, exp string) *Expression {
	return &Expression{ID: id, Expression: exp, Status: "Processing"}
}

func addExpression(w http.ResponseWriter, r *http.Request) {
	type RequestData struct {
		Expression string `json:"expression"`
	}
	var data RequestData

	err := json.NewDecoder(r.Body).Decode(&data)
	expression := data.Expression

	id := expressionsList.LastID + 1
	new_expression := newExpression(id, expression)
	postfix, err := evaluation.InfixToPostfix(expression)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Postfix Expression:", strings.Join(postfix, " "))

	new_postfix, err := evaluation.EvaluatePostfix(id, tasksList, postfix)
	if err != nil {
		if err.Error() != "unready warning" {
			fmt.Println("Error:", err)
			return
		}
		new_expression.Postfix = new_postfix
	}
	expressionsList.Mx.Lock()
	expressionsList.Expressions[expressionsList.LastID+1] = new_expression
	expressionsList.LastID++
	expressionsList.Mx.Unlock()

	w.WriteHeader(http.StatusCreated) // 201
	w.Header().Set("Content-Type", "application/json")
	_, e := w.Write([]byte(`{}`))
	if e != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError) // 500
		return
	}
	fmt.Println(expression)
}

func getExpressions(w http.ResponseWriter, r *http.Request) {
	var expressions []ExpressionItem

	for _, value := range expressionsList.Expressions {
		expressions = append(expressions, ExpressionItem{
			ID:     value.ID,
			Status: value.Status,
			Result: value.Result,
		})
	}

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

func getExpression(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest) // 400
		return
	}

	expr, exist := expressionsList.Expressions[id]
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

func getTask(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tasksList.Mu.Lock()
		defer tasksList.Mu.Unlock()
		if len(tasksList.Tasks) > 0 {
			for _, task := range tasksList.Tasks {
				w.Header().Set("Content-Type", "application/json")
				err := json.NewEncoder(w).Encode(map[string]*tasks.Task{"task": task})
				if err != nil {
					http.Error(w, "json encode error", http.StatusInternalServerError)
				}
				return
			}
		}
		http.Error(w, "No task found", http.StatusNotFound)
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

		tasksList.Mu.Lock()
		task, exists := tasksList.Tasks[result.ID]
		if !exists {
			http.Error(w, "Task not found", http.StatusNotFound)
			tasksList.Mu.Unlock()
			return
		}
		tasksList.Mu.Unlock()

		expressionsList.Mx.Lock()
		expr, found := expressionsList.Expressions[task.ExpressionID]
		if !found {
			http.Error(w, "Expression not found", http.StatusNotFound)
			expressionsList.Mx.Unlock()
			return
		}
		for i, v := range expr.Postfix {
			if v == "t"+strconv.Itoa(task.ID) {
				expr.Postfix[i] = strconv.Itoa(result.Result)
			}
		}
		expressionsList.Mx.Unlock()

		new_postfix, err := evaluation.EvaluatePostfix(expr.ID, tasksList, expr.Postfix)
		if err != nil {
			if err.Error() != "unready warning" {
				fmt.Println("Error:", err)
				return
			}
			expressionsList.Mx.Lock()
			expr.Postfix = new_postfix
			expressionsList.Mx.Unlock()
		} else {
			expressionsList.Mx.Lock()
			fmt.Println(new_postfix)
			res, e := strconv.Atoi(new_postfix[0])
			if e != nil {
				http.Error(w, "postfix not integer", http.StatusInternalServerError)
				return
			}
			expr.Result = res
			expr.Status = "Done"
			expressionsList.Mx.Unlock()
		}

		tasksList.Mu.Lock()
		delete(tasksList.Tasks, result.ID)
		tasksList.Mu.Unlock()

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, e := w.Write([]byte(`{}`))
		if e != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError) // 500
			return
		}
		return
	}
	http.Error(w, "unsupported method", http.StatusMethodNotAllowed)
	return
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/calculate", addExpression).Methods("POST")
	r.HandleFunc("/api/v1/expressions", getExpressions).Methods("GET")
	r.HandleFunc("/api/v1/expressions/{id}", getExpression).Methods("GET")
	r.HandleFunc("/internal/task", getTask).Methods("GET", "POST")
	//http.HandleFunc("/api/v1/calculate", addExpression)
	//http.HandleFunc("/api/v1/expressions", getExpressions)
	//http.HandleFunc("/api/v1/expressions/:id", getExpression)
	//http.HandleFunc("/internal/task", getTask)

	http.ListenAndServe(":8080", r)
}
