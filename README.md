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

```cmd
export COMPUTING_POWER=1
export TIME_ADDITION_MS=1000
export TIME_SUBTRACTION_MS=1000
export TIME_MULTIPLICATIONS_MS=1000
export TIME_DIVISIONS_MS=1000
```

### Установка модулей:

```cmd
go mod download github.com/gorilla/mux
```

### Запуск:

```cmd
go run ./app/main.go
```

## Примеры запросов для проверки: