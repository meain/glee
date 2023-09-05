package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

func main() {
	// sourceCode := []byte(`func func_name1 (arg1 int, arg2 string) (bool, error) { return 0, nil })
	//                       func func_name2 (arg1 int, arg2 string) (bool, error) { return 0, nil })`)
	sourceCode, err := os.ReadFile("main.go")

	funcs, err := getFuncs(sourceCode, "golang")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range funcs {
		fmt.Println(f)
	}
}

type Func struct {
	Loc  []int
	Name string
	Args []string
	Rets []string
}

func (f Func) String() string {
	return fmt.Sprintf("%s ( %s ) ( %s )", f.Name, strings.Join(f.Args, ", "), strings.Join(f.Rets, ", "))
}

func getFuncs(sourceCode []byte, langString string) ([]Func, error) {
	var (
		lang         *sitter.Language
		queryPattern map[string]string
	)

	switch langString {
	case "golang":
		lang = golang.GetLanguage()
		queryPattern = map[string]string{
			"function": "(function_declaration name: (identifier) @name) @func",
			"input":    "(function_declaration parameters: (parameter_list (parameter_declaration type: (type_identifier) @type)))",
			"output":   "(function_declaration result: (parameter_list (parameter_declaration type: (type_identifier) @type)))",
		}
	default:
		return nil, fmt.Errorf("language %s not supported", langString)
	}

	node, err := sitter.ParseCtx(context.Background(), sourceCode, lang)
	if err != nil {
		log.Fatal(err)
	}

	query := map[string]*sitter.Query{}
	for k, v := range queryPattern {
		q, err := sitter.NewQuery([]byte(v), lang)
		if err != nil {
			log.Fatal(err)
		}
		query[k] = q
	}

	cursor := sitter.NewQueryCursor()
	cursor.Exec(query["function"], node)

	funcs := []Func{}

	for {
		m, ok := cursor.NextMatch()
		if !ok {
			break
		}

		m = cursor.FilterPredicates(m, sourceCode)
		point := m.Captures[0].Node.StartPoint()

		f := Func{
			Loc:  []int{int(point.Row), int(point.Column)},
			Name: m.Captures[1].Node.Content(sourceCode),
		}

		f.Args = getTypes(m.Captures[0].Node, sourceCode, query["input"])
		f.Rets = getTypes(m.Captures[0].Node, sourceCode, query["output"])

		funcs = append(funcs, f)
	}

	return funcs, nil
}

func getTypes(node *sitter.Node, sourceCode []byte, query *sitter.Query) []string {
	types := []string{}

	cursor := sitter.NewQueryCursor()
	cursor.Exec(query, node)
	for {
		m, ok := cursor.NextMatch()
		if !ok {
			break
		}

		m = cursor.FilterPredicates(m, sourceCode)
		types = append(types, m.Captures[0].Node.Content(sourceCode))
	}

	return types
}
