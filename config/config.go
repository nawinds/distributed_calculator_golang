package config

import (
	"os"
	"strconv"
)

var (
	COMPUTING_POWER        int
	TIME_ADDITION_MS       int
	TIME_SUBTRACTION_MS    int
	TIME_MULTIPLICATION_MS int
	TIME_DIVISION_MS       int
	e                      error
)

func init() {
	COMPUTING_POWER, e = strconv.Atoi(os.Getenv("COMPUTING_POWER"))
	if e != nil {
		panic("COMPUTING_POWER environment variable must be integer")
	}

	TIME_ADDITION_MS, e = strconv.Atoi(os.Getenv("TIME_ADDITION_MS"))
	if e != nil {
		panic("TIME_ADDITION_MS environment variable must be integer")
	}

	TIME_SUBTRACTION_MS, e = strconv.Atoi(os.Getenv("TIME_SUBTRACTION_MS"))
	if e != nil {
		panic("TIME_SUBTRACTION_MS environment variable must be integer")
	}

	TIME_MULTIPLICATION_MS, e = strconv.Atoi(os.Getenv("TIME_MULTIPLICATIONS_MS"))
	if e != nil {
		panic("TIME_MULTIPLICATIONS_MS environment variable must be integer")
	}

	TIME_DIVISION_MS, e = strconv.Atoi(os.Getenv("TIME_DIVISIONS_MS"))
	if e != nil {
		panic("TIME_DIVISIONS_MS environment variable must be integer")
	}
}
