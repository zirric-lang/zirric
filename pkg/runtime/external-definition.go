package runtime

import (
	"strings"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

// BindContext provides cross-module symbol resolution during extern plugin binding.
// Plugins use this to look up symbols from other modules (e.g., os plugin resolving
// Writer/Reader from the io module).
type BindContext interface {
	// ResolveModuleSymbol looks up a symbol by name in the given module's symbol table.
	// Returns the original symbol (with ConstantId set) or nil if not found.
	ResolveModuleSymbol(moduleName string, symbolName string) *ast.Symbol
	// MainPackageModules returns the project package's name and the global slot each of its modules will occupy.
	// Compiling those modules is forced here, since a module no import reaches would otherwise never be assigned a slot; execution stays lazy because globals initialize on first access.
	// Returns an empty map when the resolver cannot enumerate the package.
	MainPackageModules() (string, map[string]int)
}

type ExternPlugin interface {
	// Module returns the module name suffix this plugin handles (e.g. "prelude", "os").
	Module() string
	Bind(ctx BindContext, module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue
}

type ExternPluginRegistry struct {
	plugins []ExternPlugin
}

func NewExternPluginRegistry(plugins ...ExternPlugin) *ExternPluginRegistry {
	return &ExternPluginRegistry{plugins: plugins}
}

func (r *ExternPluginRegistry) Register(plugin ExternPlugin) {
	r.plugins = append(r.plugins, plugin)
}

func (r *ExternPluginRegistry) Bind(ctx BindContext, module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
	moduleURI := string(module.Module().Name)
	for _, plugin := range r.plugins {
		suffix := plugin.Module()
		if suffix != "" && !strings.HasSuffix(moduleURI, "."+suffix) && moduleURI != suffix {
			continue
		}
		if val := plugin.Bind(ctx, module, decl); val != nil {
			return val
		}
	}
	return nil
}

func GetPlugin[P ExternPlugin](reg *ExternPluginRegistry, ref *P) {
	for _, p := range reg.plugins {
		plug, ok := p.(P)
		if ok {
			*ref = plug
			return
		}
	}
	var zero P
	*ref = zero
}

func (r *ExternPluginRegistry) Prelude() *Prelude {
	var prelude *Prelude
	GetPlugin(r, &prelude)
	return prelude
}

func (r *ExternPluginRegistry) OS() *OSPlugin {
	var os *OSPlugin
	GetPlugin(r, &os)
	return os
}
