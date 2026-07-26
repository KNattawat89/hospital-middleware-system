package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// interfaceMethod represents a method in the generated Repo interface.
type interfaceMethod struct {
	Name    string
	Params  []string
	Results []string
}

// findRepoFiles locates all `repo_*.go` files except `repo.go`.
func findRepoFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), "repo_") && info.Name() != "repo.go" {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// parseRepoMethods extracts all `(q *repo)` methods and import statements.
func parseRepoMethods(filename string) ([]interfaceMethod, []string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.AllErrors)
	if err != nil {
		return nil, nil, err
	}

	var methods []interfaceMethod
	var imports []string

	for _, imp := range node.Imports {
		imports = append(imports, imp.Path.Value)
	}

	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
			return true
		}
		recvType, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			return true
		}
		ident, ok := recvType.X.(*ast.Ident)
		if !ok || ident.Name != "repo" {
			return true
		}

		method := interfaceMethod{Name: fn.Name.Name}

		for _, param := range fn.Type.Params.List {
			typeStr := exprToString(param.Type)
			if len(param.Names) == 0 {
				method.Params = append(method.Params, typeStr)
				continue
			}
			for _, name := range param.Names {
				method.Params = append(method.Params, fmt.Sprintf("%s %s", name.Name, typeStr))
			}
		}

		if fn.Type.Results != nil {
			for _, result := range fn.Type.Results.List {
				typeStr := exprToString(result.Type)
				if len(result.Names) == 0 {
					method.Results = append(method.Results, typeStr)
					continue
				}
				// A result field can group multiple names under one type,
				// e.g. `(arrived, notArrived int32, err error)` — each name
				// must contribute its own entry to the result list.
				for range result.Names {
					method.Results = append(method.Results, typeStr)
				}
			}
		}

		methods = append(methods, method)
		return true
	})

	return methods, imports, nil
}

// exprToString converts an AST expression to its Go source string representation.
func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		if t.Len != nil {
			return "[" + exprToString(t.Len) + "]" + exprToString(t.Elt)
		}
		return "[]" + exprToString(t.Elt)
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.MapType:
		return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
	case *ast.Ellipsis:
		return "..." + exprToString(t.Elt)
	case *ast.InterfaceType:
		if t.Methods == nil || len(t.Methods.List) == 0 {
			return "any"
		}
		return "interface{}"
	case *ast.ChanType:
		switch t.Dir {
		case ast.SEND:
			return "chan<- " + exprToString(t.Value)
		case ast.RECV:
			return "<-chan " + exprToString(t.Value)
		default:
			return "chan " + exprToString(t.Value)
		}
	case *ast.FuncType:
		return "func(...)"
	case *ast.BasicLit:
		return t.Value
	case *ast.ParenExpr:
		return "(" + exprToString(t.X) + ")"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// generateRepoFileContent creates the final `repo.go` file content.
func generateRepoFileContent(packageName string, imports []string, methods []interfaceMethod) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`// Code generated. DO NOT EDIT.
// Code generated. DO NOT EDIT.
// Code generated. DO NOT EDIT.

package %v

`, packageName))

	if len(imports) > 0 {
		sb.WriteString("import (\n")
		importSet := make(map[string]struct{})
		for _, imp := range imports {
			importSet[imp] = struct{}{}
		}
		sortedImports := make([]string, 0, len(importSet))
		for imp := range importSet {
			sortedImports = append(sortedImports, imp)
		}
		sort.Strings(sortedImports)
		for _, imp := range sortedImports {
			sb.WriteString("\t" + imp + "\n")
		}
		sb.WriteString(")\n\n")
	}

	sort.Slice(methods, func(i, j int) bool {
		return methods[i].Name < methods[j].Name
	})

	sb.WriteString("type Repo interface {\n")
	sb.WriteString("\tWithTx(TxFunc) error\n")
	sb.WriteString("\n")
	sb.WriteString("\t// Auto Generated\n")
	for _, method := range methods {
		sb.WriteString(fmt.Sprintf("\t%s(%s) (%s)\n",
			method.Name,
			strings.Join(method.Params, ", "),
			strings.Join(method.Results, ", "),
		))
	}
	sb.WriteString("}\n\n")

	sb.WriteString(`func NewRepo(db *gorm.DB, log *zap.Logger) Repo {
	return &repo{db: db, log: log}
}

type repo struct {
	db  *gorm.DB
	log *zap.Logger
}

type TxFunc func(Repo) error

func (q *repo) WithTx(fn TxFunc) error {
	return q.db.Transaction(func(tx *gorm.DB) error {
		q := NewRepo(tx, q.log)
		return fn(q)
	})
}
`)

	return sb.String()
}

// postProcessRepoFile runs `goimports` to fix missing or unused imports and gofmt the file.
func postProcessRepoFile(filename string) error {
	cmd := exec.Command("goimports", "-w", filename)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run goimports: %w", err)
	}
	return nil
}

// generateRepo regenerates the Repo interface for a package by scanning
// its `repo_*.go` implementation files for methods on `*repo`.
func generateRepo(packageName, dir, repoFileName string) error {
	repoFile := filepath.Join(dir, repoFileName)

	files, err := findRepoFiles(dir)
	if err != nil {
		return fmt.Errorf("error finding files: %w", err)
	}

	var allMethods []interfaceMethod
	var allImports []string

	for _, file := range files {
		methods, imports, err := parseRepoMethods(file)
		if err != nil {
			fmt.Println("Error parsing file:", file, err)
			continue
		}
		allMethods = append(allMethods, methods...)
		allImports = append(allImports, imports...)
	}

	content := generateRepoFileContent(packageName, allImports, allMethods)

	if err := os.WriteFile(repoFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("error writing repo.go: %w", err)
	}

	if err := postProcessRepoFile(repoFile); err != nil {
		fmt.Println("Warning: goimports failed, you may have unused or missing imports.")
		return err
	}

	fmt.Printf("✅ [%v] Successfully regenerated repo.go with sorted methods and fixed imports\n", packageName)
	return nil
}
