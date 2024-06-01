# Никита Аксенов
# Distributed calculator in Go

## Шаги для запуска:

### Клонирование репозитория в текущую директорию:
(должен быть установлен git)

```cmd
git clone https://github.com/nawinds/distributed_calculator_golang
cd distributed_calculator_golang
```

### Установка переменных окружения:
Linux/macOS:
```cmd
export COMPUTING_POWER=1
export TIME_ADDITION_MS=1000
export TIME_SUBTRACTION_MS=1000
export TIME_MULTIPLICATIONS_MS=1000
export TIME_DIVISIONS_MS=1000
```
Windows:
```cmd
set COMPUTING_POWER=1
set TIME_ADDITION_MS=1000
set TIME_SUBTRACTION_MS=1000
set TIME_MULTIPLICATIONS_MS=1000
set TIME_DIVISIONS_MS=1000
```

### Установка модулей:

```cmd
go mod download github.com/gorilla/mux
```

### Запуск:

```cmd
go run ./app/main.go
```

## Примеры запросов для проверки (в другом терминале):

```cmd
curl --location 'http://localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
      "expression": "1+2*3-4/2*(2-3)-(8/2)"
}'
```
Ожидаемый ответ: 

```json
{"id":"1"}
```

```cmd
curl --location 'http://localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
      "expression": "9-3-8*2/(2+2)"
}'
```

Ожидаемый ответ: 

```json
{"id":"2"}
```
Получение всех выражений:
```cmd
curl --location 'http://localhost:8080/api/v1/expressions' 
```
В случае, если выражения еще не посчитаны, ответ будет таким:
```json
{"Expressions":[{"ID":1,"Status":"Processing","Result":0},{"ID":2,"Status":"Processing","Result":0}]}
```

В случае, если 2-е выражение еще не посчитано, ответ будет таким:
```json
{"Expressions":[{"ID":1,"Status":"Done","Result":5},{"ID":2,"Status":"Processing","Result":0}]}
```

В случае, если оба выражения посчитаны, ответ будет таким:
```json
{"Expressions":[{"ID":1,"Status":"Done","Result":5},{"ID":2,"Status":"Done","Result":2}]}
```

#### Получение результата по ID:
```cmd
curl --location 'http://localhost:8080/api/v1/expressions/1'
```
Ответ будет таким:
```json
{"id":1,"status":"Done","result":5}
```

## Как это работает?

`main.go`:
Запускает http сервер и агента

- `func NewExpressions() *Expressions`:
Создает очередь выражений

- `func newExpression(id int, exp string) *Expression`:
Создает объект выражения с полями Status равном "Processing",
переданным ID и переданным выражением

- `func addExpression(w http.ResponseWriter, r *http.Request)`:
Обработчик пути `/api/v1/calculate`. Принимает запрос с выражением,
и вызывает функции для его обработки.

- `func getExpressions(w http.ResponseWriter, _ *http.Request)`:
Обработчик пути `http://localhost:8080/api/v1/expressions`. Возвращает
выражения из очереди с их статусами и результатами. Если выражение еще не посчитано,
то результат будет равен 0.

- `func getExpression(w http.ResponseWriter, r *http.Request)`:
Обработчик пути `http://localhost:8080/api/v1/expressions/<ExpressionID>`. Возвращает
выражения из очереди с их статусами и результатами. Если выражение еще не посчитано,
то результат будет равен 0.

- `func getTask(w http.ResponseWriter, r *http.Request)`:
Обработчик пути `http://localhost:8080/api/v1/task`. 
Если был использован метод GET, то возвращает задачу из очереди, 
которую еще не взял другой обработчик (если не был превышен таймаут для него).
Если был использован метод POST, то удаляет задачу из очереди, а результат 
ее выполнения подставляет в постфикс выражения, к которому она относится
и запускает функцию для дальнейшей обработки выражения, к которому задача относится.

`evaluation.go`:
Содержит функции для создания задач выполнения операций из исходного выражения

- `func InfixToPostfix(expression string) ([]string, error)`:
Преобразует инфиксное выражение в постфиксное для дальнейшей обработки.
- `func EvaluatePostfix(expressionID int, tasks *tasks.Tasks, postfix []string)`:
Находит в постфиксном выражении все операции с числами, которые не зависят 
от результата работы задач, которые еще не были посчитаны и добавляет задачи 
в очередь для выполнения этих операций

`tasks.go`:
Содержит функции для создания очереди задач и создания объектов задач

- `func newTask(id, time, expressionID int, operator string, arg1, arg2 int)`:
Создает экземпляр новой задачи с переданными параметрами id задачи, времени 
выполнения арифметической операции, id выражения, к которому относится задача, 
оператора и аргументов для выполнения операции
- `func NewTasks() *Tasks`:
Создает экземпляр очереди задач
- `func (t *Tasks) AddTask(time, expressionID int, operator string, arg1, arg2 int) string`:
Добавляет задачу в очередь задач и возвращает ее id

`agent.go`:
Содержит функции для создания агента, который выполняет задачи

- `func Worker()`:
Горутина с бесконечным циклом, запускающая функции принятия задачи с сервера, 
выполнения операции из задачи с заданным временем выполнения, и загрузки 
результатов выполнения задачи на сервер
- `func getTask() (*tasks.Task, error)`:
Функция загрузки задачи с сервера
- `func performTask(task *tasks.Task) int`:
Функция выполнения операции из задачи
- `func postTaskResult(id, result int)`:
Функция загрузки результата выполнения задачи на сервер

## Примечание:
Вместо оператора - иногда используется символ <, чтобы избежать проблем 
с отрицательными числами
