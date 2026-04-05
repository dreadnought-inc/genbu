package expr

import (
	"fmt"
	"regexp"
)

var exprPattern = regexp.MustCompile(`\$\{\{(.+?)\}\}`)

// Eval evaluates all ${{ expr }} blocks in the value string.
// vars provides resolved variable values for reference lookups.
func Eval(value string, vars map[string]string) (string, error) {
	funcs := Registry()
	var evalErr error

	result := exprPattern.ReplaceAllStringFunc(value, func(match string) string {
		if evalErr != nil {
			return match
		}

		// Extract expression inside ${{ }}
		inner := match[3 : len(match)-2]

		node, err := Parse(inner)
		if err != nil {
			evalErr = fmt.Errorf("parsing expression %q: %w", inner, err)
			return match
		}

		val, err := evalNode(node, vars, funcs)
		if err != nil {
			evalErr = fmt.Errorf("evaluating expression %q: %w", inner, err)
			return match
		}

		return val
	})

	if evalErr != nil {
		return "", evalErr
	}
	return result, nil
}

// ExtractExprVarRefs returns variable names referenced in ${{ }} expressions.
func ExtractExprVarRefs(value string) []string {
	matches := exprPattern.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var refs []string

	for _, m := range matches {
		node, err := Parse(m[1])
		if err != nil {
			continue
		}

		for _, ref := range CollectVarRefs(node) {
			if !seen[ref] {
				seen[ref] = true
				refs = append(refs, ref)
			}
		}
	}

	return refs
}

func evalNode(node Node, vars map[string]string, funcs map[string]Func) (string, error) {
	switch n := node.(type) {
	case *StringLit:
		return n.Value, nil

	case *NumberLit:
		return n.Value, nil

	case *VarRef:
		if v, ok := vars[n.Name]; ok {
			return v, nil
		}
		return "", fmt.Errorf("undefined variable %q", n.Name)

	case *FuncCall:
		fn, ok := funcs[n.Name]
		if !ok {
			return "", fmt.Errorf("unknown function %q", n.Name)
		}

		args := make([]string, 0, len(n.Args))
		for _, arg := range n.Args {
			val, err := evalNode(arg, vars, funcs)
			if err != nil {
				return "", fmt.Errorf("in %s(): %w", n.Name, err)
			}
			args = append(args, val)
		}

		if len(args) < fn.MinArgs {
			return "", fmt.Errorf("%s: expected at least %d argument(s), got %d", n.Name, fn.MinArgs, len(args))
		}
		if fn.MaxArgs >= 0 && len(args) > fn.MaxArgs {
			return "", fmt.Errorf("%s: expected at most %d argument(s), got %d", n.Name, fn.MaxArgs, len(args))
		}

		return fn.Call(args)

	default:
		return "", fmt.Errorf("unknown node type")
	}
}
