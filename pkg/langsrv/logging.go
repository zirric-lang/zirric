package langsrv

import (
	"fmt"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (ls *zirricLangserver) logMessage(ctx *glsp.Context, format string, args ...any) {
	if ctx == nil || ctx.Notify == nil {
		return
	}
	ctx.Notify(protocol.ServerWindowLogMessage, &protocol.LogMessageParams{
		Type:    protocol.MessageTypeLog,
		Message: fmt.Sprintf(format, args...),
	})
}
