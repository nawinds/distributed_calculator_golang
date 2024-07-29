package main

import (
	"context"
	"database/sql"
	"distributed_calculator/agent"
	"distributed_calculator/config"
	"distributed_calculator/evaluation"
	"distributed_calculator/expression_structs"
	"distributed_calculator/tasks"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Expression = expression_structs.Expression

type ExpressionItem struct { // структура выражения для вывода в API
	ID     int
	Status string
	Result int
}

type Expressions = expression_structs.Expressions

type ExpressionResponse struct { // структура для возврата списка выражений через API
	Expressions []ExpressionItem
}

type User struct {
	ID       string `json:"id"`
	Login    string `json:"login"`
	Password string `json:"password"`
}

var (
	expressionsList = NewExpressions()
	tasksList       = tasks.NewTasks()
	ctx             = context.TODO()
	db, db_err      = sql.Open("sqlite3", "store.db")
)

func NewExpressions() *Expressions {
	return &Expressions{Mx: &sync.Mutex{}, LastID: 0, Expressions: make(map[int]*Expression)}
}

func NewExpression(uid int, exp string) *Expression {
	return &Expression{UserID: uid, Expression: exp, Status: "Processing"}
}

func addExpressionHandler(w http.ResponseWriter, r *http.Request) {
	type RequestData struct {
		Expression string `json:"expression"`
		Token      string `json:"token"`
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

	tokenFromString, err := jwt.Parse(data.Token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			panic(fmt.Errorf("unexpected signing method: %v", token.Header["alg"]))
		}

		return []byte(config.SECRET_KEY), nil
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest) // 400
		return
	}

	uid := 0
	var e error

	if claims, ok := tokenFromString.Claims.(jwt.MapClaims); ok {
		fmt.Println("user name: ")
		uid, e = strconv.Atoi(fmt.Sprintf("%v", claims["name"]))
	} else {
		http.Error(w, err.Error(), http.StatusBadRequest) // 400
		return
	}

	expression := data.Expression
	postfix, err := evaluation.InfixToPostfix(expression)
	if err != nil {
		http.Error(w, "Invalid expression: "+err.Error(), http.StatusBadRequest)
		return
	}

	id, e := insertExpression(NewExpression(uid, expression))

	fmt.Println("Postfix Expression:", strings.Join(postfix, " "))

	go func(id int, postfix []string) {
		newPostfix, err := evaluation.EvaluatePostfix(id, tasksList, postfix)
		expressionsList.Mx.Lock()
		defer expressionsList.Mx.Unlock()

		if err != nil && err.Error() == "unready warning" {
			err := updateExpressionPostfix(id, newPostfix)
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError) // 500
				return
			}
			// expression.Postfix = newPostfix
		} else if err == nil {
			result, _ := strconv.Atoi(newPostfix[0])
			// expression.Result = result
			err := updateExpressionResult(id, result)
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError) // 500
				return
			}
			// expression.Status = "Done"
			err = updateExpressionStatus(id, "Done")
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError) // 500
				return
			}
		} else {
			// expression.Status = "Error"
			err := updateExpressionStatus(id, "Error")
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError) // 500
				return
			}
		}
	}(id, postfix)

	w.WriteHeader(http.StatusCreated) // 201
	w.Header().Set("Content-Type", "application/json")
	e = json.NewEncoder(w).Encode(&ResponseData{ID: strconv.Itoa(id)})
	if e != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError) // 500
		return
	}
	fmt.Println(expression)
}

func getExpressionsHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	fmt.Println(token)

	tokenFromString, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			panic(fmt.Errorf("unexpected signing method: %v", token.Header["alg"]))
		}

		return []byte(config.SECRET_KEY), nil
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest) // 400
		return
	}

	if claims, ok := tokenFromString.Claims.(jwt.MapClaims); ok {
		fmt.Println("user name: ", claims["name"])
	} else {
		http.Error(w, err.Error(), http.StatusBadRequest) // 400
		return
	}

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
	token := vars["token"]

	tokenFromString, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			panic(fmt.Errorf("unexpected signing method: %v", token.Header["alg"]))
		}

		return []byte(config.SECRET_KEY), nil
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest) // 400
		return
	}

	if claims, ok := tokenFromString.Claims.(jwt.MapClaims); ok {
		fmt.Println("user name: ", claims["name"])
	} else {
		http.Error(w, err.Error(), http.StatusBadRequest) // 400
		return
	}

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
		task, err := tasksList.GetTask(expressionsList)
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
			ID     int    `json:"id"`
			Result int    `json:"result"`
			Error  string `json:"error"`
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
		fmt.Println(result.Error)
		if result.Error != "" {
			expr.Status = "Error: " + result.Error

			for _, t := range tasksList.Tasks {
				if t.ExpressionID == expr.ID {
					_, err := tasksList.CompleteTask(t.ID)
					if err != nil {
						fmt.Printf("Error: %v\n", err)
					}
				}
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			_, e := w.Write([]byte(`{}`))
			if e != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
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

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid data", http.StatusUnprocessableEntity)
		return
	}

	_, e := insertUser(&user)
	if e != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid data", http.StatusUnprocessableEntity)
		return
	}

	user, e := getUser(user.Login, user.Password)
	if e != nil {
		http.Error(w, "Get user error", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"name": user.Login,
		"nbf":  now.Unix(),
		"exp":  now.Add(5 * time.Minute).Unix(),
		"iat":  now.Unix(),
	})

	tokenString, err := token.SignedString([]byte(config.SECRET_KEY))

	response := struct {
		Token string `json:"token"`
	}{
		Token: tokenString,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	e = json.NewEncoder(w).Encode(response)

	if e != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError) // 500
		return
	}
}

func createTables() error {
	const (
		usersTable = `
	CREATE TABLE IF NOT EXISTS users(
		id INTEGER PRIMARY KEY AUTOINCREMENT, 
		login TEXT,
		password TEXT
	);`

		expressionsTable = `
	CREATE TABLE IF NOT EXISTS expressions(
		id INTEGER PRIMARY KEY AUTOINCREMENT, 
		user_id INTEGER NOT NULL,
		expression TEXT NOT NULL,
		status TEXT,
	
		FOREIGN KEY (user_id)  REFERENCES expressions (id)
	);`
	)

	if _, err := db.ExecContext(ctx, usersTable); err != nil {
		return err
	}

	if _, err := db.ExecContext(ctx, expressionsTable); err != nil {
		return err
	}

	return nil
}

func insertExpression(expression *Expression) (int, error) {
	var q = `
	INSERT INTO expressions (expression, user_id, status) values ($1, $2, "Processing...")
	`
	result, err := db.ExecContext(ctx, q, expression.Expression, expression.UserID)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func getExpression(id int) *Expression {
	var q = "SELECT id, user_id, expression, status, result FROM expressions WHERE id=$1"
	rows, err := db.QueryContext(ctx, q, id)
	if err != nil {
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		expr := Expression{}
		err := rows.Scan(&expr.ID, &expr.UserID, &expr.Expression, &expr.Status, &expr.Result)
		if err != nil {
			return nil
		}
		return &expr
	}
	return nil
}

func updateExpressionPostfix(id int, postfix []string) error {
	var q = "UPDATE expressions SET postfix=$1 WHERE id=$2"
	_, err := db.ExecContext(ctx, q, strings.Join(postfix, " "), id)
	return err
}

func updateExpressionResult(id, result int) error {
	var q = "UPDATE expressions SET result=$1 WHERE id=$2"
	_, err := db.ExecContext(ctx, q, result, id)
	return err
}

func updateExpressionStatus(id int, status string) error {
	var q = "UPDATE expressions SET status=$1 WHERE id=$2"
	_, err := db.ExecContext(ctx, q, status, id)
	return err
}

func insertUser(user *User) (int64, error) {
	var q = `
	INSERT INTO users (login, password) values ($1, $2)
	`
	result, err := db.ExecContext(ctx, q, user.Login, user.Password)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func getUser(login, password string) (User, error) {
	var q = "SELECT id, login, password FROM users WHERE login=$1 AND password=$2"
	rows, err := db.QueryContext(ctx, q, login, password)
	if err != nil {
		return User{}, err
	}
	defer rows.Close()

	for rows.Next() {
		u := User{}
		err := rows.Scan(&u.ID, &u.Login, &u.Password)
		if err != nil {
			return User{}, err
		}
		return u, nil
	}
	return User{}, fmt.Errorf("Not found")
}

func main() {
	if db_err != nil {
		panic(db_err)
	}
	defer db.Close()

	err := db.PingContext(ctx)
	if err != nil {
		panic(err)
	}

	if err = createTables(); err != nil {
		panic(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/calculate", addExpressionHandler).Methods("POST")
	r.HandleFunc("/api/v1/expressions", getExpressionsHandler).Methods("GET")
	r.HandleFunc("/api/v1/expressions/{id}", getExpressionHandler).Methods("GET")
	r.HandleFunc("/internal/task", getTaskHandler).Methods("GET", "POST")

	r.HandleFunc("/api/v1/register", registerHandler).Methods("POST")
	r.HandleFunc("/api/v1/login", loginHandler).Methods("POST")

	for i := 0; i < config.COMPUTING_POWER; i++ {
		go agent.Worker()
	}

	err = http.ListenAndServe(":8080", r)
	if err != nil {
		panic(err)
	}
}
