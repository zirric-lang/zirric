package cavefile

import (
	"path/filepath"

	"code.knabel.dev/zirric-lang/zirric/pkg/ast"
	"code.knabel.dev/zirric-lang/zirric/pkg/registry"
	"code.knabel.dev/zirric-lang/zirric/pkg/version"
)

// Parse extracts Cavefile information from a parsed (and analyzed) Zirric module.
//
// cavefileMod is the Cavefile parsed as a ContextModule (already run through the analyzer).
// caveMod and futureTasksMod are the parsed cave and future.tasks stdlib modules,
// used to verify attribute types against actual DeclAttr declarations — preventing false
// matches from name collisions or unrelated imports.
// fallbackName is used as Package.Name if no mod declaration is found.
// projectDir is used to resolve @cave.Local("../rel") paths to absolute paths.
func Parse(cavefileMod *ast.ContextModule, caveMod *ast.ContextModule, futureTasksMod *ast.ContextModule, fallbackName string, projectDir string) Cavefile {
	if cavefileMod == nil || caveMod == nil {
		return Cavefile{Package: Package{Name: fallbackName}}
	}

	aliasMap := buildAliasMap(cavefileMod)
	pkgSource := extractPackageSource(cavefileMod, caveMod, aliasMap)

	var pkgName string
	if pkgSource != "" {
		pkgName = string(registry.CanonicalizeModuleSource(pkgSource))
	} else {
		pkgName = fallbackName
	}

	deps := extractDependencies(cavefileMod, caveMod, aliasMap, projectDir)

	var tasks []Task
	if futureTasksMod != nil {
		tasks = extractTasks(cavefileMod, futureTasksMod, aliasMap)
	}

	return Cavefile{
		Package:      Package{Name: pkgName, Source: pkgSource},
		Dependencies: deps,
		Tasks:        tasks,
	}
}

// extractPackageSource finds the @cave.Package("url") attribute on the mod declaration
// and returns the URL. DeclModule has ExportScopeLocal so it stays in the file-level DeclTable.
func extractPackageSource(mod *ast.ContextModule, caveMod *ast.ContextModule, aliasMap map[string]registry.LogicalURI) string {
	for _, file := range mod.Files {
		if file.Decls == nil {
			continue
		}
		for _, sym := range file.Decls.Symbols {
			if sym == nil || sym.Decl == nil {
				continue
			}
			m, ok := sym.Decl.(*ast.DeclModule)
			if !ok {
				continue
			}
			for _, attr := range m.Attributes {
				if isFutureCaveAttr(attr, "Package", caveMod, aliasMap) {
					return firstStringArg(attr)
				}
			}
		}
	}
	return ""
}

// buildAliasMap collects import declarations from the Cavefile's file-level symbol tables
// and maps each local alias to the imported module's LogicalURI.
// DeclImport has ExportScopeLocal so it stays in the file-level DeclTable (not module-level).
func buildAliasMap(mod *ast.ContextModule) map[string]registry.LogicalURI {
	result := map[string]registry.LogicalURI{}
	for _, file := range mod.Files {
		if file.Decls == nil {
			continue
		}
		for _, sym := range file.Decls.Symbols {
			if sym == nil || sym.Decl == nil {
				continue
			}
			imp, ok := sym.Decl.(*ast.DeclImport)
			if !ok {
				continue
			}
			result[imp.Alias.Value] = imp.ModuleName.URI()
		}
	}
	return result
}

// extractDependencies finds the @cave.Dependencies() data block in the Cavefile module
// and converts each field into a cavefile.Dependency. Attribute types are verified
// against actual DeclAttr declarations in caveMod, identified via the alias map.
func extractDependencies(cavefileMod *ast.ContextModule, caveMod *ast.ContextModule, aliasMap map[string]registry.LogicalURI, projectDir string) []Dependency {
	for _, sym := range cavefileMod.Decls.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		data, ok := sym.Decl.(*ast.DeclData)
		if !ok {
			continue
		}
		if !hasCaveAttr(data.Attributes, "Dependencies", caveMod, aliasMap) {
			continue
		}
		return fieldsToDepencies(data.Fields, caveMod, aliasMap, projectDir)
	}
	return nil
}

// hasCaveAttr returns true if any attribute in the chain is a specific attr from cave.
// It resolves the import alias via aliasMap and verifies the declaration type in caveMod.
func hasCaveAttr(attrs ast.AttributeChain, attrName string, caveMod *ast.ContextModule, aliasMap map[string]registry.LogicalURI) bool {
	for _, attr := range attrs {
		if isFutureCaveAttr(attr, attrName, caveMod, aliasMap) {
			return true
		}
	}
	return false
}

// isFutureCaveAttr checks that an attribute instance refers to a specific DeclAttr
// declared in caveMod (identified by caveMod.Name), using aliasMap to
// resolve the local alias to the module URI.
func isFutureCaveAttr(attr *ast.DeclAttrInstance, attrName string, caveMod *ast.ContextModule, aliasMap map[string]registry.LogicalURI) bool {
	ref := attr.Reference
	if len(ref) < 2 {
		return false
	}
	prefix := ref[0].Value
	name := ref[len(ref)-1].Value
	if name != attrName {
		return false
	}
	uri, ok := aliasMap[prefix]
	if !ok {
		return false
	}
	if uri != caveMod.Name {
		return false
	}
	// Verify the declaration is a DeclAttr in the referenced module (type system check)
	sym, ok := caveMod.Decls.Symbols[attrName]
	if !ok {
		return false
	}
	_, ok = sym.Decl.(*ast.DeclAttr)
	return ok
}

func fieldsToDepencies(fields []ast.DeclField, caveMod *ast.ContextModule, aliasMap map[string]registry.LogicalURI, projectDir string) []Dependency {
	var deps []Dependency
	for _, field := range fields {
		dep, ok := fieldToDependency(field, caveMod, aliasMap, projectDir)
		if ok {
			deps = append(deps, dep)
		}
	}
	return deps
}

func fieldToDependency(field ast.DeclField, caveMod *ast.ContextModule, aliasMap map[string]registry.LogicalURI, projectDir string) (Dependency, bool) {
	dep := Dependency{
		Package: Package{Name: field.Name.Value},
	}
	found := false
	for _, attr := range field.Attributes {
		switch {
		case isFutureCaveAttr(attr, "Stdlib", caveMod, aliasMap):
			dep.Source = StandardLibrarySource
			dep.Module = registry.JoinModuleURI("", firstStringArg(attr))
			found = true
		case isFutureCaveAttr(attr, "Local", caveMod, aliasMap):
			rel := firstStringArg(attr)
			if projectDir != "" && rel != "" {
				abs, err := filepath.Abs(filepath.Join(projectDir, rel))
				if err == nil {
					dep.Source = "file://" + abs
				} else {
					dep.Source = "file://" + rel
				}
			} else {
				dep.Source = "file://" + rel
			}
			found = true
		case isFutureCaveAttr(attr, "Git", caveMod, aliasMap):
			dep.Source = firstStringArg(attr)
			found = true
		case isFutureCaveAttr(attr, "Version", caveMod, aliasMap):
			dep.Predicates = append(dep.Predicates, version.ParsePredicate(firstStringArg(attr)))
		}
	}
	return dep, found
}

func firstStringArg(attr *ast.DeclAttrInstance) string {
	for _, arg := range attr.Arguments {
		if s, ok := arg.(*ast.ExprString); ok {
			return s.Literal
		}
	}
	return ""
}
