package langsrv

import (
	"fmt"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// logMessage sends a window/logMessage notification — routed to the client's log output only, not shown to the user (see showMessage for user-visible feedback, e.g. after an executeCommand action completes).
func (ls *zirricLangserver) logMessage(ctx *glsp.Context, format string, args ...any) {
	if ctx == nil || ctx.Notify == nil {
		return
	}
	ctx.Notify(protocol.ServerWindowLogMessage, &protocol.LogMessageParams{
		Type:    protocol.MessageTypeLog,
		Message: fmt.Sprintf(format, args...),
	})
}

// showMessage sends a window/showMessage notification, which clients surface to the user directly (a popup, status line, etc.) — unlike logMessage, which most clients only write to a log/output channel the user isn't necessarily looking at.
func (ls *zirricLangserver) showMessage(ctx *glsp.Context, typ protocol.MessageType, format string, args ...any) {
	if ctx == nil || ctx.Notify == nil {
		return
	}
	ctx.Notify(protocol.ServerWindowShowMessage, &protocol.ShowMessageParams{
		Type:    typ,
		Message: fmt.Sprintf(format, args...),
	})
}
