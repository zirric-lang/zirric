package runtime

import "code.knabel.dev/zirric-lang/zirric/ast"

type ExternPlugin interface {
	Bind(module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue
}

type ExternPluginRegistry struct {
	plugins []ExternPlugin
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
