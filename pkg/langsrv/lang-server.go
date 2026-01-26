package langsrv

import (
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

var lsName = "zirric"
var debug = true
var handler protocol.Handler

type zirricLangserver struct {
	server *server.Server
}

var ls zirricLangserver = zirricLangserver{}

func init() {
	commonlog.Configure(2, nil)

	handler = protocol.Handler{
		// TODO: Fill in capabilities
	}
}

func RunStdio() error {
	ls.server = server.NewServer(&handler, lsName, debug)
	return ls.server.RunStdio()
}

func RunIPC() error {
	ls.server = server.NewServer(&handler, lsName, debug)
	return ls.server.RunNodeJs()
}

func RunSocket(address string) error {
	ls.server = server.NewServer(&handler, lsName, debug)
	return ls.server.RunWebSocket(address)
}

func RunTCP(address string) error {
	ls.server = server.NewServer(&handler, lsName, debug)
	return ls.server.RunTCP(address)
}
