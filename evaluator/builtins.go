package evaluator

import (
	"fmt"
	"waiig/object"
)

var builtins = map[string]*object.Builtin{
	"len": &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}

			switch arg := args[0].(type) {
			case *object.String:
				return &object.Integer{Value: int64(len(arg.Value))}
			case *object.Array:
				return &object.Integer{Value: int64(len(arg.Elements))}
			default:
				return newError("argument to `len` not supported, got %s", args[0].Type())
			}
		},
	},
	"push": &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=2",
					len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("argument to `push` must be ARRAY, got %s",
					args[0].Type())
			}

			arr := args[0].(*object.Array)
			length := len(arr.Elements)

			newElements := make([]object.Object, length+1, length+1)
			copy(newElements, arr.Elements)
			newElements[length] = args[1]

			return &object.Array{Elements: newElements}
		},
	},
	"println": &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return newError("wrong number of arguments. got=%d, want at least 1",
					len(args))
			}
			if args[0].Type() != object.STRING_OBJ {
				return newError("first argument to `println` must be STRING, got %s",
					args[0].Type())
			}

			str := args[0].(*object.String).Value

			var evaluatedArgs []any
			for _, arg := range args[1:] {
				var raw any
				switch obj := arg.(type) {
				case *object.String:
					raw = obj.Value
				case *object.Integer:
					raw = obj.Value
				case *object.Array:
					raw = obj.Elements
				case *object.Boolean:
					raw = obj.Value
				case *object.Range:
					raw = obj.Inspect()
				case *object.Null:
					raw = nil
				}
				evaluatedArgs = append(evaluatedArgs, raw)
			}

			fmt.Printf(str, evaluatedArgs...)

			return nil
		},
	},
}
