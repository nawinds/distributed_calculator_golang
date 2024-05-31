package evaluation

import (
	"distributed_calculator/tasks"
	"fmt"
	"strconv"
	"unicode"
)

var precedence = map[rune]int{
	'+': 1,
	'-': 1,
	'*': 2,
	'/': 2,
}

var associativity = map[rune]string{
	'+': "L",
	'-': "L",
	'*': "L",
	'/': "L",
}

func InfixToPostfix(expression string) ([]string, error) {
	var output []string
	var operatorStack []rune

	for _, token := range expression {
		switch {
		case unicode.IsDigit(token):
			output = append(output, string(token))
		case token == '+' || token == '-' || token == '*' || token == '/':
			for len(operatorStack) > 0 {
				top := operatorStack[len(operatorStack)-1]
				if top == '(' {
					break
				}
				if (associativity[token] == "L" && precedence[token] <= precedence[top]) ||
					(associativity[token] == "R" && precedence[token] < precedence[top]) {
					output = append(output, string(top))
					operatorStack = operatorStack[:len(operatorStack)-1]
				} else {
					break
				}
			}
			operatorStack = append(operatorStack, token)
		case token == '(':
			operatorStack = append(operatorStack, token)
		case token == ')':
			for len(operatorStack) > 0 && operatorStack[len(operatorStack)-1] != '(' {
				output = append(output, string(operatorStack[len(operatorStack)-1]))
				operatorStack = operatorStack[:len(operatorStack)-1]
			}
			if len(operatorStack) == 0 {
				return nil, fmt.Errorf("mismatched parentheses")
			}
			operatorStack = operatorStack[:len(operatorStack)-1]
		}
	}

	for len(operatorStack) > 0 {
		if operatorStack[len(operatorStack)-1] == '(' {
			return nil, fmt.Errorf("mismatched parentheses")
		}
		output = append(output, string(operatorStack[len(operatorStack)-1]))
		operatorStack = operatorStack[:len(operatorStack)-1]
	}

	return output, nil
}

func EvaluatePostfix(expressionID int, tasks *tasks.Tasks, postfix []string) ([]string, error) {
	if len(postfix) == 1 {
		return postfix, nil
	}

	var stack []string
	i := 0
	for {
		if i >= len(postfix) {
			break
		}
		fmt.Println(stack, i, postfix)
		switch postfix[i] {
		case "+", "-", "*", "/":
			b, errB := strconv.Atoi(stack[len(stack)-1])
			if errB != nil {
				i++
				continue
			}
			stack = stack[:len(stack)-1]

			a, errA := strconv.Atoi(stack[len(stack)-1])
			if errA != nil {
				i++
				continue
			}
			stack = stack[:len(stack)-1]

			taskID := "0"

			switch postfix[i] {
			case "+":
				taskID = tasks.AddTask(expressionID, "+", a, b)
			case "-":
				taskID = tasks.AddTask(expressionID, "-", a, b)
			case "*":
				taskID = tasks.AddTask(expressionID, "*", a, b)
			case "/":
				taskID = tasks.AddTask(expressionID, "/", a, b)
			}
			stack = append(stack, "t"+taskID)
			postfix = append(postfix[:i+1], append([]string{"t" + taskID}, postfix[i+1:]...)...)

			for j := i; j >= i-2; j-- {
				if j+1 == len(postfix) {
					postfix = postfix[:j]
				} else {
					postfix = append(postfix[:j], postfix[j+1:]...)
				}
			}
			i -= 1
		default:
			num, err := strconv.Atoi(postfix[i])
			if err != nil {
				i++
				continue
			}
			stack = append(stack, strconv.Itoa(num))
			i++
		}
	}

	fmt.Println(stack)
	return postfix, fmt.Errorf("unready warning")
}
