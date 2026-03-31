package evaluator

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/issueye/icooclaw_lang/internal/ast"
	"github.com/issueye/icooclaw_lang/internal/lexer"
	"github.com/issueye/icooclaw_lang/internal/object"
	"github.com/issueye/icooclaw_lang/internal/parser"
)

func evalImportStmt(node *ast.ImportStmt, env *object.Environment) object.Object {
	if len(node.Names) > 0 && node.Alias != nil {
		return object.NewError(node.Token.Line, "named import does not support alias")
	}

	modulePathValue := Eval(node.Module, env)
	if object.IsError(modulePathValue) {
		return modulePathValue
	}

	modulePathObj, ok := modulePathValue.(*object.String)
	if !ok {
		return object.NewError(node.Token.Line, "import path must evaluate to STRING, got %s", modulePathValue.Type())
	}

	resolvedPath, err := resolveModulePath(env.ScriptPath(), modulePathObj.Value)
	if err != nil {
		return object.NewError(node.Token.Line, "could not resolve module '%s': %s", modulePathObj.Value, err.Error())
	}

	exports, errObj := loadModuleExports(env, resolvedPath, node.Token.Line)
	if errObj != nil {
		return errObj
	}

	if len(node.Names) > 0 {
		for _, name := range node.Names {
			pair, ok := exports.Pairs[name.Value]
			if !ok {
				return object.NewError(node.Token.Line, "module '%s' does not export '%s'", resolvedPath, name.Value)
			}
			if assigned := env.Set(name.Value, pair.Value); object.IsError(assigned) {
				return assigned
			}
		}
		return &object.Null{}
	}

	alias := moduleAlias(node, resolvedPath)
	if alias == "" {
		return object.NewError(node.Token.Line, "could not determine module alias for '%s'", resolvedPath)
	}
	return env.Set(alias, exports)
}

func evalExportStmt(node *ast.ExportStmt, env *object.Environment) object.Object {
	return env.Export(node.Name.Value)
}

func loadModuleExports(env *object.Environment, resolvedPath string, line int) (*object.Hash, *object.Error) {
	if exports, ok := env.CachedModule(resolvedPath); ok {
		return exports, nil
	}

	if ok := env.MarkModuleLoading(resolvedPath); !ok {
		return nil, object.NewError(line, "circular import detected for module '%s'", resolvedPath)
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		env.FailModuleLoading(resolvedPath)
		return nil, object.NewError(line, "could not read module '%s': %s", resolvedPath, err.Error())
	}

	l := lexer.New(string(data))
	p := parser.New(l)
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		env.FailModuleLoading(resolvedPath)
		return nil, object.NewError(line, "module parse failed for '%s': %s", resolvedPath, strings.Join(errs, "; "))
	}

	moduleEnv := object.NewDetachedEnvironment(env)
	moduleEnv.SetCLIContext(resolvedPath, env.CLIArgs())

	result := Eval(program, moduleEnv)
	moduleEnv.Wait()
	if errObj, ok := result.(*object.Error); ok {
		env.FailModuleLoading(resolvedPath)
		return nil, errObj
	}

	exports := moduleEnv.ExportedHash()
	env.FinishModuleLoading(resolvedPath, exports)
	return exports, nil
}

func resolveModulePath(currentScriptPath, modulePath string) (string, error) {
	if modulePath == "" {
		return "", os.ErrInvalid
	}

	baseDir := "."
	if currentScriptPath != "" {
		baseDir = filepath.Dir(currentScriptPath)
	}

	if !filepath.IsAbs(modulePath) {
		modulePath = filepath.Join(baseDir, modulePath)
	}

	return filepath.Abs(modulePath)
}

func moduleAlias(node *ast.ImportStmt, resolvedPath string) string {
	if node.Alias != nil {
		return node.Alias.Value
	}

	base := filepath.Base(resolvedPath)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}
