package ast

type Documented interface {
	ProvidedDocs() *Docs
}

// Documentable is a declaration a doc comment can be attached to, which is what reflect.docs reads back.
type Documentable interface {
	Documented
	SetDocs(docs *Docs)
}

type Overviewable interface {
	DeclOverview() string
}
