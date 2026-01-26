package token

type Source struct {
	File   string
	Offset int
}

func MakeSource(
	fileName string,
	offset int,
) *Source {
	return &Source{
		File:   fileName,
		Offset: offset,
	}
}
