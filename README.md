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
