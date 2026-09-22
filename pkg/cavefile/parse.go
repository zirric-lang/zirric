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
// caveMod and tasksMod are the parsed cave and tasks stdlib modules,
// used to verify attribute types against actual DeclAttr declarations — preventing false
// matches from name collisions or unrelated imports.
// fallbackName is used as Package.Name if no mod declaration is found.
// projectDir is used to resolve @cave.Local("../rel") paths to absolute paths.
func Parse(cavefileMod *ast.ContextModule, caveMod *ast.ContextModule, tasksMod *ast.ContextModule, fallbackName string, projectDir string) Cavefile {
	if cavefileMod == nil || caveMod == nil {
		return Cavefile{Package: Package{Name: fallbackName}}
	}

	aliasMap := buildAliasMap(cavefileMod)

	pkg := Package{ModulePath: extractModulePath(cavefileMod)}
	var deps []Dependency
	if data := findPackageData(cavefileMod, caveMod, aliasMap); data != nil {
		applyPackageAttributes(&pkg, data.Attributes, caveMod, aliasMap)
		deps = fieldsToDepencies(data.Fields, caveMod, aliasMap, projectDir)
	}

	switch {
	case pkg.ModulePath != "":
		pkg.Name = pkg.ModulePath
	case pkg.Source != "":
		pkg.Name = string(registry.CanonicalizeModuleSource(pkg.Source))
	default:
		pkg.Name = fallbackName
	}

	excludes := extractFormattingExcludes(cavefileMod, caveMod, aliasMap)

	var tasks []Task
	if tasksMod != nil {
		tasks = extractTasks(cavefileMod, tasksMod, aliasMap)
	}

	return Cavefile{
		Package:            pkg,
		Docs:               moduleDocs(cavefileMod),
		Dependencies:       deps,
		Tasks:              tasks,
		FormattingExcludes: excludes,
	}
}

// applyPackageAttributes reads the metadata a package declares about itself onto pkg.
func applyPackageAttributes(pkg *Package, attrs ast.AttributeChain, caveMod *ast.ContextModule, aliasMap map[string]registry.LogicalURI) {
	for _, attr := range attrs {
		switch {
		case isFutureCaveAttr(attr, "Git", caveMod, aliasMap):
			pkg.Source = firstStringArg(attr)
		case isFutureCaveAttr(attr, "Version", caveMod, aliasMap):
			pkg.Version = firstStringArg(attr)
		case isFutureCaveAttr(attr, "Documentation", caveMod, aliasMap):
			pkg.Documentation = firstStringArg(attr)
		case isFutureCaveAttr(attr, "Description", caveMod, aliasMap):
			pkg.Description = firstStringArg(attr)
		case isFutureCaveAttr(attr, "LanguageVersion", caveMod, aliasMap):
			pkg.LanguageVersion = firstStringArg(attr)
		}
	}
}

// extractFormattingExcludes collects every @cave.FormattingExcludes. The attribute may sit on any declaration, so a Cavefile needs no placeholder type to carry it.
func extractFormattingExcludes(cavefileMod *ast.ContextModule, caveMod *ast.ContextModule, aliasMap map[string]registry.LogicalURI) []string {
	var patterns []string
	collect := func(attrs ast.AttributeChain) {
		for _, attr := range attrs {
			if isFutureCaveAttr(attr, "FormattingExcludes", caveMod, aliasMap) {
				patterns = append(patterns, stringArrayArgs(attr)...)
			}
		}
	}

	for _, file := range cavefileMod.Files {
		if file == nil {
			continue
		}
		for _, sym := range file.Decls.Symbols {
			if sym == nil || sym.Decl == nil {
				continue
			}
			if mod, ok := sym.Decl.(*ast.DeclModule); ok {
				collect(mod.Attributes)
			}
		}
	}
	for _, sym := range cavefileMod.Decls.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		switch decl := sym.Decl.(type) {
		case *ast.DeclData:
			collect(decl.Attributes)
		case *ast.DeclUnion:
			collect(decl.Attributes)
		}
	}
	return patterns
}

// extractModulePath returns the fully qualified path the Cavefile's own `mod` declares, which is the base every module of the package is named under.
// DeclModule has ExportScopeLocal so it stays in the file-level DeclTable.
func extractModulePath(mod *ast.ContextModule) string {
	for _, file := range mod.Files {
		if file.Decls == nil {
			continue
		}
		for _, sym := range file.Decls.Symbols {
			if sym == nil || sym.Decl == nil {
				continue
			}
			if m, ok := sym.Decl.(*ast.DeclModule); ok {
				return m.Path.String()
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

// moduleDocs is the comment written above the Cavefile's own `mod` declaration, which is what the file says about the package as a whole.
func moduleDocs(cavefileMod *ast.ContextModule) string {
	for _, file := range cavefileMod.Files {
		if file == nil || file.Module == nil {
			continue
		}
		if docs := ast.DocsOf(file.Module); docs != "" {
			return docs
		}
	}
	return ""
}

// findPackageData returns the data declaration carrying @cave.Package(), whose attributes describe the package and whose fields are its dependencies.
// Attribute types are verified against actual DeclAttr declarations in caveMod, identified via the alias map.
func findPackageData(cavefileMod *ast.ContextModule, caveMod *ast.ContextModule, aliasMap map[string]registry.LogicalURI) *ast.DeclData {
	for _, sym := range cavefileMod.Decls.Symbols {
		if sym == nil || sym.Decl == nil {
			continue
		}
		data, ok := sym.Decl.(*ast.DeclData)
		if !ok {
			continue
		}
		if !hasCaveAttr(data.Attributes, "Package", caveMod, aliasMap) {
			continue
		}
		return data
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
		Docs:    ast.DocsOf(field),
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
		case isFutureCaveAttr(attr, "Documentation", caveMod, aliasMap):
			dep.Documentation = firstStringArg(attr)
		case isFutureCaveAttr(attr, "Description", caveMod, aliasMap):
			dep.Description = firstStringArg(attr)
		case isFutureCaveAttr(attr, "LanguageVersion", caveMod, aliasMap):
			dep.LanguageVersion = firstStringArg(attr)
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
