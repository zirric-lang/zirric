package runtime

import (
	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
)

var _ ExternPlugin = &IOPlugin{}

// IOPlugin provides runtime bindings for the io module's extern declarations.
// The only one does nothing but give way, so io.read and io.write are switch points (ZE-025).
type IOPlugin struct{}

func (*IOPlugin) Module() string { return "io" }

// Bind implements ExternPlugin.
func (*IOPlugin) Bind(ctx BindContext, module *ast.SymbolTable, decl *ast.Symbol) RuntimeValue {
	switch decl.Name {
	case "_yield":
		fn := MakeExternFunc(decl, func(_ VMCaller, _ []RuntimeValue) (RuntimeValue, error) {
			return Void{}, nil
		})
		fn.Switches = true
		return fn
	}
	return nil
}
