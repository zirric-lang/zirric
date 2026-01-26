package analyzer

import (
	"context"
	"fmt"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

func (a *Analyzer) validateStaticRefs(module *ast.ContextModule) []AnalysisError {
	if module == nil || module.Symbols == nil {
		return nil
	}
	var errs []AnalysisError
	walkNode(module, func(node ast.Node) {
		switch n := node.(type) {
		case *ast.DeclImport:
			errs = append(errs, a.validateImport(module, n)...)
		case *ast.DeclUnionMember:
			if err := a.validateStaticRef(module, n.Member, n.TokenLiteral()); err != nil {
				errs = append(errs, *err)
			}
		}
	})
	return errs
}

func (a *Analyzer) validateImport(module *ast.ContextModule, decl *ast.DeclImport) []AnalysisError {
	if decl == nil {
		return nil
	}
	if a.resolver == nil {
		return nil
	}

	resolved, err := a.resolver.ResolveModule(context.Background(), decl.ModuleName.URI())
	if err != nil || resolved == nil {
		return []AnalysisError{{
			Token:   decl.TokenLiteral(),
			Summary: "unknown module",
			Details: fmt.Sprintf("module %q could not be resolved", decl.ModuleName),
		}}
	}
	_, _ = a.Analyze(resolved, true)

	if len(decl.Members) == 0 {
		return nil
	}
	var errs []AnalysisError
	for i := range decl.Members {
		member := decl.Members[i]
		if resolved.Symbols == nil {
			errs = append(errs, AnalysisError{
				Token:   member.TokenLiteral(),
				Summary: "unknown import member",
				Details: fmt.Sprintf("module %q has no exported member %q", decl.ModuleName, member.Name.Value),
			})
			continue
		}
		sym := resolved.Symbols.Symbols[member.Name.Value]
		if sym == nil || sym.Decl == nil || sym.Decl.ExportScope() != ast.ExportScopePublic {
			errs = append(errs, AnalysisError{
				Token:   member.TokenLiteral(),
				Summary: "unknown import member",
				Details: fmt.Sprintf("module %q has no exported member %q", decl.ModuleName, member.Name.Value),
			})
		}
	}
	return errs
}

func (a *Analyzer) validateStaticRef(module *ast.ContextModule, ref ast.StaticReference, tok token.Token) *AnalysisError {
	if len(ref) == 0 {
		return &AnalysisError{
			Token:   tok,
			Summary: "invalid reference",
			Details: "reference is empty",
		}
	}
	head := ref[0].Value
	if len(ref) == 1 {
		if sym := findSymbolByName(module, head); sym != nil && sym.Decl != nil {
			return nil
		}
		return &AnalysisError{
			Token:   ref.TokenLiteral(),
			Summary: "unknown reference",
			Details: fmt.Sprintf("reference %q could not be resolved", ref.String()),
		}
	}
	if sym := findSymbolByName(module, head); sym != nil && sym.Decl != nil {
		if decl, ok := sym.Decl.(*ast.DeclImport); ok {
			if a.resolver == nil {
				return &AnalysisError{
					Token:   ref.TokenLiteral(),
					Summary: "unknown reference",
					Details: fmt.Sprintf("reference %q could not be resolved", ref.String()),
				}
			}
			resolved, err := a.resolver.ResolveModule(context.Background(), decl.ModuleName.URI())
			if err != nil || resolved == nil || resolved.Symbols == nil {
				return &AnalysisError{
					Token:   ref.TokenLiteral(),
					Summary: "unknown reference",
					Details: fmt.Sprintf("reference %q could not be resolved", ref.String()),
				}
			}
			_, _ = a.Analyze(resolved, true)
			return resolveStaticRefInTable(resolved.Symbols, ref[1:], ref, true)
		}
		if sym.ChildTable != nil {
			return resolveStaticRefInTable(sym.ChildTable, ref[1:], ref, false)
		}
		return &AnalysisError{
			Token:   ref.TokenLiteral(),
			Summary: "unknown reference",
			Details: fmt.Sprintf("reference %q could not be resolved", ref.String()),
		}
	}
	if imported := a.findImportedModuleByPrefix(module, ref); imported != nil {
		if a.resolver == nil {
			return &AnalysisError{
				Token:   ref.TokenLiteral(),
				Summary: "unknown reference",
				Details: fmt.Sprintf("reference %q could not be resolved", ref.String()),
			}
		}
		resolved, err := a.resolver.ResolveModule(context.Background(), imported.URI())
		if err != nil || resolved == nil || resolved.Symbols == nil {
			return &AnalysisError{
				Token:   ref.TokenLiteral(),
				Summary: "unknown reference",
				Details: fmt.Sprintf("reference %q could not be resolved", ref.String()),
			}
		}
		_, _ = a.Analyze(resolved, true)
		rest := ref[len(imported):]
		if len(rest) == 0 {
			return &AnalysisError{
				Token:   ref.TokenLiteral(),
				Summary: "unknown reference",
				Details: fmt.Sprintf("reference %q could not be resolved", ref.String()),
			}
		}
		return resolveStaticRefInTable(resolved.Symbols, rest, ref, true)
	}
	return &AnalysisError{
		Token:   ref.TokenLiteral(),
		Summary: "unknown reference",
		Details: fmt.Sprintf("reference %q could not be resolved", ref.String()),
	}
}

func resolveStaticRefInTable(table *ast.SymbolTable, ref ast.StaticReference, full ast.StaticReference, requireExport bool) *AnalysisError {
	if table == nil {
		return &AnalysisError{
			Token:   full.TokenLiteral(),
			Summary: "unknown reference",
			Details: fmt.Sprintf("reference %q could not be resolved", full.String()),
		}
	}
	cur := table
	for i := range ref {
		part := ref[i]
		sym := cur.Symbols[part.Value]
		if sym == nil || sym.Decl == nil {
			return &AnalysisError{
				Token:   part.TokenLiteral(),
				Summary: "unknown reference",
				Details: fmt.Sprintf("reference %q could not be resolved", full.String()),
			}
		}
		if requireExport && sym.Decl.ExportScope() != ast.ExportScopePublic {
			return &AnalysisError{
				Token:   part.TokenLiteral(),
				Summary: "unknown reference",
				Details: fmt.Sprintf("reference %q could not be resolved", full.String()),
			}
		}
		if i+1 < len(ref) {
			if sym.ChildTable == nil {
				return &AnalysisError{
					Token:   part.TokenLiteral(),
					Summary: "unknown reference",
					Details: fmt.Sprintf("reference %q could not be resolved", full.String()),
				}
			}
			cur = sym.ChildTable
		}
	}
	if len(ref) > 0 {
		last := ref[len(ref)-1]
		sym := cur.Symbols[last.Value]
		if sym == nil || sym.Decl == nil {
			return &AnalysisError{
				Token:   last.TokenLiteral(),
				Summary: "unknown reference",
				Details: fmt.Sprintf("reference %q could not be resolved", full.String()),
			}
		}
	}
	return nil
}

func (a *Analyzer) findImportedModuleByPrefix(module *ast.ContextModule, ref ast.StaticReference) ast.ModuleName {
	if module == nil || len(ref) < 2 {
		return nil
	}
	visitImport := func(sym *ast.Symbol) ast.ModuleName {
		if sym == nil || sym.Decl == nil {
			return nil
		}
		imp, ok := sym.Decl.(*ast.DeclImport)
		if !ok {
			return nil
		}
		if hasStaticPrefix(ref, ast.StaticReference(imp.ModuleName)) {
			return imp.ModuleName
		}
		return nil
	}
	for _, sym := range module.Symbols.Symbols {
		if match := visitImport(sym); match != nil {
			return match
		}
	}
	for _, file := range module.Files {
		if file == nil || file.Symbols == nil {
			continue
		}
		for _, sym := range file.Symbols.Symbols {
			if match := visitImport(sym); match != nil {
				return match
			}
		}
	}
	return nil
}

func hasStaticPrefix(ref ast.StaticReference, prefix ast.StaticReference) bool {
	if len(prefix) > len(ref) {
		return false
	}
	for i := range prefix {
		if ref[i].Value != prefix[i].Value {
			return false
		}
	}
	return true
}

func findSymbolByName(module *ast.ContextModule, name string) *ast.Symbol {
	if module == nil {
		return nil
	}
	if module.Symbols != nil {
		if sym := module.Symbols.Symbols[name]; sym != nil && sym.Decl != nil {
			return sym
		}
	}
	for _, file := range module.Files {
		if file == nil || file.Symbols == nil {
			continue
		}
		if sym := file.Symbols.Symbols[name]; sym != nil && sym.Decl != nil {
			return sym
		}
	}
	return nil
}

func walkNode(node ast.Node, visit func(ast.Node)) {
	if node == nil {
		return
	}
	visit(node)
	node.EnumerateChildNodes(func(child ast.Node) {
		walkNode(child, visit)
	})
}
