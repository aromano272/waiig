package evaluator

import (
	"fmt"
	"waiig/ast"
	"waiig/object"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

func Eval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {
	case *ast.Program:
		return evalProgram(node, env)
	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}
	case *ast.StringLiteral:
		return &object.String{Value: node.Value}
	case *ast.Boolean:
		return nativeBooleanToObject(node.Value)
	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)
	case *ast.InfixExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}

		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}

		return evalInfixExpression(node.Operator, left, right)
	case *ast.BlockStatement:
		return evalBlockStatement(node, env)
	case *ast.IfExpression:
		return evalIfExpression(node, env)
	case *ast.ReturnStatement:
		value := Eval(node.ReturnValue, env)
		if isError(value) {
			return value
		}
		return &object.ReturnValue{Value: value}
	case *ast.LetStatement:
		value := Eval(node.Value, env)
		if isError(value) {
			return value
		}

		env.Set(node.Name.Value, value)
	case *ast.Identifier:
		return evalIdentifier(node, env)
	case *ast.FunctionLiteral:
		params := node.Parameters
		body := node.Body
		return &object.Function{Parameters: params, Body: body, Env: env}
	case *ast.CallExpression:
		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}

		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}

		return applyFunction(function, args)
	case *ast.ArrayLiteral:
		return evalArrayLiteral(node, env)
	case *ast.IndexExpression:
		return evalIndexExpression(node, env)
	case *ast.RangeExpression:
		return evalRangeExpression(node, env)
	case *ast.HashLiteral:
		return evalHashExpression(node, env)
	}
	return nil
}

func evalHashExpression(node *ast.HashLiteral, env *object.Environment) object.Object {
	hash := &object.Hash{}

	pairs := make(map[object.HashKey]object.HashPair)

	for key, value := range node.Pairs {
		keyObj := Eval(key, env)
		if isError(keyObj) {
			return keyObj
		}

		var hashKey object.HashKey

		hashable, ok := keyObj.(object.Hashable)
		if !ok {
			return newError("unusable as hash key: %s", keyObj.Type())
		}

		hashKey = hashable.HashKey()

		valueObj := Eval(value, env)
		if isError(valueObj) {
			return valueObj
		}

		pairs[hashKey] = object.HashPair{Key: keyObj, Value: valueObj}
	}

	hash.Pairs = pairs

	return hash
}

func evalIndexExpression(node *ast.IndexExpression, env *object.Environment) object.Object {
	left := Eval(node.Left, env)
	if isError(left) {
		return left
	}

	indexObj := Eval(node.Index, env)
	if isError(indexObj) {
		return indexObj
	}

	switch obj := left.(type) {
	case *object.Array:
		switch index := indexObj.(type) {
		case *object.Integer:
			if int(index.Value) >= len(obj.Elements) || index.Value < 0 {
				return newError("index out of bounds, index=%d len=%d", index.Value, len(obj.Elements))
			}
			return obj.Elements[index.Value]
		case *object.Range:
			if int(index.From) > len(obj.Elements) || int(index.ToExclusive) > len(obj.Elements) ||
				index.From < 0 || index.ToExclusive < 0 {
				return newError("range index out of bounds, index=%d:%d len=%d", index.From, index.ToExclusive, len(obj.Elements))
			}
			elements := obj.Elements[index.From:index.ToExclusive]
			return &object.Array{Elements: elements}
		default:
			return newError("unknown index type: %s", indexObj.Type())
		}
	case *object.String:
		switch index := indexObj.(type) {
		case *object.Integer:
			if int(index.Value) >= len(obj.Value) || index.Value < 0 {
				return newError("index out of bounds, index=%d len=%d", index.Value, len(obj.Value))
			}
			char := string(obj.Value[index.Value])
			return &object.String{Value: char}
		case *object.Range:
			if int(index.From) > len(obj.Value) || int(index.ToExclusive) > len(obj.Value) ||
				index.From < 0 || index.ToExclusive < 0 {
				return newError("range index out of bounds, index=%d:%d len=%d", index.From, index.ToExclusive, len(obj.Value))
			}
			str := obj.Value[index.From:index.ToExclusive]
			return &object.String{Value: str}
		default:
			return newError("unknown index type: %s", indexObj.Type())
		}
	case *object.Hash:
		hashKey, ok := indexObj.(object.Hashable)
		if !ok {
			return newError("unusable as hash key: %s", indexObj.Type())
		}

		val, ok := obj.Pairs[hashKey.HashKey()]
		if !ok {
			return NULL
		} else {
			return val.Value
		}
	default:
		return newError("unknown operator: index of %s", left.Type())
	}
}

func evalRangeExpression(node *ast.RangeExpression, env *object.Environment) object.Object {
	left := Eval(node.Left, env)
	if isError(left) {
		return left
	}

	right := Eval(node.Right, env)
	if isError(right) {
		return right
	}

	from, ok := left.(*object.Integer)
	if !ok {
		return newError("unknown operator: %s : %s", left.Type(), right.Type())
	}

	toExclusive, ok := right.(*object.Integer)
	if !ok {
		return newError("unknown operator: %s : %s", left.Type(), right.Type())
	}

	if from.Value > toExclusive.Value {
		return newError("range `from` must be greater than or equal to > `toExclusive`, from=%d toExclusive=%d", from.Value, toExclusive.Value)
	}

	return &object.Range{
		From:        from.Value,
		ToExclusive: toExclusive.Value,
	}
}

func evalArrayLiteral(node *ast.ArrayLiteral, env *object.Environment) object.Object {
	arr := &object.Array{}

	elements := evalExpressions(node.Elements, env)
	if len(elements) == 1 && isError(elements[0]) {
		return elements[0]
	}

	arr.Elements = elements

	return arr
}

func applyFunction(fn object.Object, args []object.Object) object.Object {
	switch function := fn.(type) {
	case *object.Function:
		extendedEnv := extendFunctionEnv(function, args)
		evaluated := Eval(function.Body, extendedEnv)
		return unwrapReturnValue(evaluated)
	case *object.Builtin:
		return function.Fn(args...)
	default:
		return newError("not a function: %s", fn.Type())
	}
}

func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}

	return obj
}

func extendFunctionEnv(
	fn *object.Function,
	args []object.Object,
) *object.Environment {
	env := object.NewEnclosedEnvironment(fn.Env)

	for paramIdx, param := range fn.Parameters {
		env.Set(param.Value, args[paramIdx])
	}

	return env
}

func evalExpressions(expressions []ast.Expression, env *object.Environment) []object.Object {
	var objects []object.Object

	for _, exp := range expressions {
		val := Eval(exp, env)
		if isError(val) {
			return []object.Object{val}
		}

		objects = append(objects, val)
	}

	return objects
}

func evalIdentifier(node *ast.Identifier, env *object.Environment) object.Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}

	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}

	return newError("identifier not found: " + node.Value)
}

func evalIfExpression(node *ast.IfExpression, env *object.Environment) object.Object {
	condition := Eval(node.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(node.Consequence, env)
	} else if node.Alternative != nil {
		return Eval(node.Alternative, env)
	} else {
		return NULL
	}
}

func isTruthy(obj object.Object) bool {
	switch obj {
	case TRUE:
		return true
	case FALSE:
		return false
	case NULL:
		return false
	default:
		return true
	}
}

func evalInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)
	case operator == "==":
		// using pointer comparison here since boolean object are shared
		return nativeBooleanToObject(left == right)
	case operator == "!=":
		return nativeBooleanToObject(left != right)
	case left.Type() != right.Type():
		return newError("type mismatch: %s %s %s", left.Type(), operator, right.Type())
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalStringInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	leftVal := left.(*object.String).Value
	rightVal := right.(*object.String).Value

	switch operator {
	case "+":
		return &object.String{Value: leftVal + rightVal}
	case "==":
		return nativeBooleanToObject(leftVal == rightVal)
	case "!=":
		return nativeBooleanToObject(leftVal != rightVal)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalIntegerInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	leftVal := left.(*object.Integer).Value
	rightVal := right.(*object.Integer).Value

	switch operator {
	case "+":
		return &object.Integer{Value: leftVal + rightVal}
	case "-":
		return &object.Integer{Value: leftVal - rightVal}
	case "*":
		return &object.Integer{Value: leftVal * rightVal}
	case "/":
		return &object.Integer{Value: leftVal / rightVal}
	case ">":
		return nativeBooleanToObject(leftVal > rightVal)
	case "<":
		return nativeBooleanToObject(leftVal < rightVal)
	case "==":
		return nativeBooleanToObject(leftVal == rightVal)
	case "!=":
		return nativeBooleanToObject(leftVal != rightVal)
	default:
		return newError("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

func evalPrefixExpression(operator string, operand object.Object) object.Object {
	switch operator {
	case "!":
		return evalBangOperatorExpression(operand)
	case "-":
		return evalMinusPrefixOperatorExpression(operand)
	default:
		return newError("unknown operator %s%s", operator, operand.Type())
	}
}

func evalMinusPrefixOperatorExpression(operand object.Object) object.Object {
	if operand.Type() != object.INTEGER_OBJ {
		return newError("unknown operator: -%s", operand.Type())
	}
	value := operand.(*object.Integer).Value
	return &object.Integer{Value: -value}
}

func evalBangOperatorExpression(operand object.Object) object.Object {
	switch operand {
	case TRUE:
		return FALSE
	case FALSE:
		return TRUE
	case NULL:
		return TRUE
	default:
		return FALSE
	}
}

func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object

	for _, stmt := range program.Statements {
		result = Eval(stmt, env)

		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value
		case *object.Error:
			return result
		}

		if returnValue, ok := result.(*object.ReturnValue); ok {
			return returnValue.Value
		}
	}

	return result
}

func evalBlockStatement(block *ast.BlockStatement, env *object.Environment) object.Object {
	var result object.Object

	for _, stmt := range block.Statements {
		result = Eval(stmt, env)

		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

func nativeBooleanToObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}
