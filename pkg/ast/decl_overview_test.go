package ast

import (
	"testing"

	"code.knabel.dev/zirric-lang/zirric/pkg/token"
)

func ident(name string) Identifier {
	return MakeIdentifier(token.Token{Literal: name})
}

func TestDeclImportOverview(t *testing.T) {
	tests := []struct {
		name string
		decl DeclImport
		want string
	}{
		{
			name: "simple import",
			decl: *MakeDeclImport(token.Token{}, StaticReference{ident("prelude")}),
			want: "import prelude = prelude",
		},
		{
			name: "qualified import",
			decl: *MakeDeclImport(token.Token{}, StaticReference{ident("future"), ident("prelude")}),
			want: "import prelude = future.prelude",
		},
		{
			name: "aliased import",
			decl: *MakeDeclAliasImport(token.Token{}, ident("p"), StaticReference{ident("future"), ident("prelude")}),
			want: "import p = future.prelude",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.decl.DeclOverview()
			if got != tt.want {
				t.Errorf("DeclOverview() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeclImportMemberOverview(t *testing.T) {
	decl := MakeDeclImportMember(
		token.Token{},
		ModuleName{ident("future"), ident("prelude")},
		ident("String"),
	)
	want := "import future.prelude { String }"
	got := decl.DeclOverview()
	if got != want {
		t.Errorf("DeclOverview() = %q, want %q", got, want)
	}
}

func TestDeclFuncOverview(t *testing.T) {
	tests := []struct {
		name string
		src  func() DeclFunc
		want string
	}{
		{
			name: "no params no return",
			src: func() DeclFunc {
				dt := MakeModuleDeclTable(nil)
				impl, _ := MakeExprFunc(token.Token{}, "greet", dt)
				return *MakeDeclFunc(token.Token{}, ident("greet"), impl)
			},
			want: "fn greet()",
		},
		{
			name: "params without types",
			src: func() DeclFunc {
				dt := MakeModuleDeclTable(nil)
				impl, _ := MakeExprFunc(token.Token{}, "add", dt)
				impl.SetParams([]DeclParameter{
					*MakeDeclParameter(ident("a"), nil, nil),
					*MakeDeclParameter(ident("b"), nil, nil),
				})
				return *MakeDeclFunc(token.Token{}, ident("add"), impl)
			},
			want: "fn add(a, b)",
		},
		{
			name: "params with types and return type",
			src: func() DeclFunc {
				dt := MakeModuleDeclTable(nil)
				impl, _ := MakeExprFunc(token.Token{}, "add", dt)
				impl.SetParams([]DeclParameter{
					*MakeDeclParameter(ident("a"), nil, MakeTypeExprRef(StaticReference{ident("Int")})),
					*MakeDeclParameter(ident("b"), nil, MakeTypeExprRef(StaticReference{ident("Int")})),
				})
				d := MakeDeclFunc(token.Token{}, ident("add"), impl)
				d.ReturnType = MakeTypeExprRef(StaticReference{ident("Int")})
				return *d
			},
			want: "fn add(a: Int, b: Int) -> Int",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.src().DeclOverview()
			if got != tt.want {
				t.Errorf("DeclOverview() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeclFieldOverview(t *testing.T) {
	tests := []struct {
		name string
		decl DeclField
		want string
	}{
		{
			name: "simple field",
			decl: *MakeDeclField(ident("name"), nil, nil, nil),
			want: "name",
		},
		{
			name: "typed field",
			decl: *MakeDeclField(ident("name"), nil, nil, MakeTypeExprRef(StaticReference{ident("String")})),
			want: "name: String",
		},
		{
			name: "method field with return type",
			decl: *MakeDeclField(
				ident("toNumber"),
				[]DeclParameter{*MakeDeclParameter(ident("value"), nil, nil)},
				nil,
				MakeTypeExprRef(StaticReference{ident("Number")}),
			),
			want: "toNumber(value) -> Number",
		},
		{
			name: "method field with param types",
			decl: *MakeDeclField(
				ident("convert"),
				[]DeclParameter{*MakeDeclParameter(ident("x"), nil, MakeTypeExprRef(StaticReference{ident("Int")}))},
				nil,
				MakeTypeExprRef(StaticReference{ident("String")}),
			),
			want: "convert(x: Int) -> String",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.decl.DeclOverview()
			if got != tt.want {
				t.Errorf("DeclOverview() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeclConstantOverview(t *testing.T) {
	d := MakeDeclConstant(token.Token{}, ident("pi"), nil)
	d.TypeHint = MakeTypeExprRef(StaticReference{ident("Float")})
	got := d.DeclOverview()
	want := "const pi: Float"
	if got != want {
		t.Errorf("DeclOverview() = %q, want %q", got, want)
	}
}

func TestDeclVariableOverview(t *testing.T) {
	d := MakeDeclVariable(token.Token{}, ident("count"), nil)
	d.TypeHint = MakeTypeExprRef(StaticReference{ident("Int")})
	got := d.DeclOverview()
	want := "var count: Int"
	if got != want {
		t.Errorf("DeclOverview() = %q, want %q", got, want)
	}
}

func TestDeclExternValueOverview(t *testing.T) {
	d := MakeDeclExternValue(token.Token{}, ident("void"))
	d.TypeHint = MakeTypeExprRef(StaticReference{ident("Void")})
	got := d.DeclOverview()
	want := "extern const void: Void"
	if got != want {
		t.Errorf("DeclOverview() = %q, want %q", got, want)
	}
}

func TestDeclExternFuncOverview(t *testing.T) {
	d := MakeDeclExternFunc(token.Token{}, ident("print"))
	d.SetParams([]DeclParameter{
		*MakeDeclParameter(ident("value"), nil, MakeTypeExprRef(StaticReference{ident("String")})),
	})
	got := d.DeclOverview()
	want := "extern fn print(value: String)"
	if got != want {
		t.Errorf("DeclOverview() = %q, want %q", got, want)
	}
}

func TestDeclAttrOverviewEmpty(t *testing.T) {
	d := MakeDeclAttr(token.Token{}, ident("Numeric"))
	got := d.DeclOverview()
	want := "attr Numeric"
	if got != want {
		t.Errorf("DeclOverview() = %q, want %q", got, want)
	}
}
