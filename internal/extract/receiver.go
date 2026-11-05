package extract

import "go/ast"

// receiverType unwraps pointer/star receivers to the base type name.
func receiverType(expr ast.Expr) (bool, string) {
	switch t := expr.(type) {
	case *ast.StarExpr:
		_, name := receiverType(t.X)

		return true, name
	case *ast.Ident:
		return false, t.Name
	case *ast.IndexExpr: // generic receiver, e.g. *Pool[T]
		_, name := receiverType(t.X)

		return true, name
	case *ast.IndexListExpr:
		_, name := receiverType(t.X)

		return true, name
	}

	return false, ""
}
