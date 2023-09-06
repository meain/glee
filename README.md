# glee

> [Hoogle](https://hoogle.haskell.org/) but for every language, using [tree-sitter](https://tree-sitter.github.io/tree-sitter/)

### Usage

```
Usage: glee [OPTIONS] <signature>
Hoogle like search for functions in all languages

Options:
  -match string
        matching algorithm (options: includes, default) (default "default")

Example: glee -match includes '(Path, string) -> (Path, error)'
```

### Example

```
$ glee -match includes '( Path ) -> ( Path )'

pkg/path/drive.go:19:0:ToDrivePath (Path) -> (*DrivePath, error)
transformer/restore_path.go:42:0:basicLocationPath (path.Path, *path.Builder) -> (path.Path, error)
```
