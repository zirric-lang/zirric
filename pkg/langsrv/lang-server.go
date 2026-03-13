package langsrv

import (
	"github.com/go-git/go-billy/v5"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

var lsName = "zirric"
var debug = true
var handler protocol.Handler

type zirricLangserver struct {
	server   *server.Server
	docs     *documentStore
	fs       billy.Filesystem
	rootPath string
	diagURIs map[protocol.DocumentUri]struct{}
	openDocs map[string]protocol.DocumentUri
}

var ls zirricLangserver = zirricLangserver{}

func init() {
	commonlog.Configure(2, nil)
	ls.docs = newDocumentStore()
	ls.diagURIs = make(map[protocol.DocumentUri]struct{})
	ls.openDocs = make(map[string]protocol.DocumentUri)

	handler = protocol.Handler{
		Initialize:            ls.initialize,
		Initialized:           ls.initialized,
		Shutdown:              ls.shutdown,
		Exit:                  ls.exit,
		TextDocumentDidOpen:   ls.didOpen,
		TextDocumentDidChange: ls.didChange,
		TextDocumentDidSave:   ls.didSave,
		TextDocumentDidClose:  ls.didClose,
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
