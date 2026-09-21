package codefmt

// Options controls the formatter's whitespace decisions. Start from DefaultOptions.
type Options struct {
	UseTabs bool
	// TabWidth applies only when UseTabs is false.
	TabWidth           int
	MaxBlankLines      int
	InsertFinalNewline bool
}

func DefaultOptions() Options {
	return Options{
		UseTabs:            true,
		TabWidth:           4,
		MaxBlankLines:      1,
		InsertFinalNewline: true,
	}
}

// FromLSP overlays an editor's formatting options onto o, ignoring unknown keys and wrong types so that garbage falls back to the defaults.
func (o Options) FromLSP(m map[string]any) Options {
	if v, ok := lspBool(m, "insertSpaces"); ok {
		o.UseTabs = !v
	}
	if v, ok := lspInt(m, "tabSize"); ok && v > 0 {
		o.TabWidth = v
	}
	if v, ok := lspBool(m, "insertFinalNewline"); ok {
		o.InsertFinalNewline = v
	}
	return o
}

func (o Options) indentUnit() string {
	if o.UseTabs {
		return "\t"
	}
	width := o.TabWidth
	if width <= 0 {
		width = DefaultOptions().TabWidth
	}
	return spaces(width)
}

func spaces(n int) string {
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = ' '
	}
	return string(buf)
}

func lspBool(m map[string]any, key string) (bool, bool) {
	v, ok := m[key].(bool)
	return v, ok
}

// lspInt accepts float64 because JSON numbers decode that way.
func lspInt(m map[string]any, key string) (int, bool) {
	switch v := m[key].(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	default:
		return 0, false
	}
}
