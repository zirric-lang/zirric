package staticmodule

import "code.knabel.dev/zirric-lang/zirric/registry"

type (
	StaticModule struct {
		LogicalURI registry.LogicalURI
		Srcs       []registry.Source
	}

	StaticSource struct {
		LogicalURI registry.LogicalURI
		Contents   []byte
	}
)

func NewModule(logicalURI registry.LogicalURI, sources []registry.Source) *StaticModule {
	return &StaticModule{
		LogicalURI: logicalURI,
		Srcs:       sources,
	}
}

func (m *StaticModule) URI() registry.LogicalURI {
	return m.LogicalURI
}

func (m *StaticModule) Sources() ([]registry.Source, error) {
	return m.Srcs, nil
}

func NewSource(logicalURI registry.LogicalURI, input []byte) *StaticSource {
	return &StaticSource{
		LogicalURI: logicalURI,
		Contents:   input,
	}
}
func NewSourceString(logicalURI registry.LogicalURI, input string) *StaticSource {
	return &StaticSource{
		LogicalURI: logicalURI,
		Contents:   []byte(input),
	}
}

func (s *StaticSource) URI() registry.LogicalURI {
	return s.LogicalURI
}

func (s *StaticSource) Read() ([]byte, error) {
	return s.Contents, nil
}
