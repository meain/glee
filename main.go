package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/agnivade/levenshtein"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

const LINE_CLEAR = "\033[2K"

type file struct {
	Language string
	Path     string
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: oogle <signature>")
	}

	uinput := os.Args[1]
	files := []file{}
	root := "."

	if len(os.Args) > 2 {
		root = os.Args[2]
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			if !info.IsDir() {
				lang := getLanguage(info.Name())
				if lang != "" {
					files = append(files, file{Language: lang, Path: path})
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	funcs := []Func{}
	for _, f := range files {
		fmt.Fprintf(os.Stderr, "%sProcessing %s\r", LINE_CLEAR, filepath.Base(f.Path))

		sourceCode, err := os.ReadFile(f.Path)
		if err != nil {
			log.Fatal(err)
		}

		tf, err := getFuncs(sourceCode, f)
		if err != nil {
			log.Fatal(err)
		}

		funcs = append(funcs, tf...)
	}

	funcs = sortByDistance(funcs, uinput)

	for i, f := range funcs {
		fmt.Println(f)

		if i > 10 {
			break
		}
	}
}

func getLanguage(filename string) string {
	// get language based on extension
	lang := ""
	switch filepath.Ext(filename) {
	case ".go":
		lang = "golang"
	}
	return lang
}

func sortByDistance(funcs []Func, uinput string) []Func {
	distanceMap := []struct {
		Func     Func
		Distance int
	}{}

	for _, f := range funcs {
		distance := levenshtein.ComputeDistance(uinput, f.Signature())
		distanceMap = append(distanceMap, struct {
			Func     Func
			Distance int
		}{Func: f, Distance: distance})
	}

	// sort by distance
	sort.Slice(distanceMap, func(i, j int) bool {
		return distanceMap[i].Distance < distanceMap[j].Distance
	})

	funcs = []Func{}
	for _, d := range distanceMap {
		funcs = append(funcs, d.Func)
	}

	return funcs
}

type Func struct {
	Path string
	Loc  []int
	Name string
	Args []string
	Rets []string
}

func (f Func) String() string {
	return fmt.Sprintf(
		"%s:%s:%s:%s (%s) -> (%s)",
		f.Path,
		strconv.Itoa(f.Loc[0]),
		strconv.Itoa(f.Loc[1]),
		f.Name,
		strings.Join(f.Args, ", "),
		strings.Join(f.Rets, ", "),
	)
}

func (f Func) Signature() string {
	return fmt.Sprintf("( %s ) -> ( %s )", strings.Join(f.Args, ", "), strings.Join(f.Rets, ", "))
}

func getFuncs(sourceCode []byte, f file) ([]Func, error) {
	var (
		lang         *sitter.Language
		queryPattern map[string]string
	)

	switch f.Language {
	case "golang":
		lang = golang.GetLanguage()
		queryPattern = map[string]string{
			"function": "(function_declaration name: (identifier) @name) @func",
			"input":    "(function_declaration parameters: (parameter_list (parameter_declaration type: (_) @type)))",
			"output": `(function_declaration result: (parameter_list (parameter_declaration type: (_) @type)))
                       (function_declaration result: [(type_identifier) (pointer_type) (slice_type)] @type)`,
		}
	default:
		return nil, fmt.Errorf("language %s not supported", f.Language)
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
			Path: f.Path,
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
