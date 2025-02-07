package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"os"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/lsp"
	"terragrunt-ls/internal/rpc"
	"terragrunt-ls/internal/tg"

	"go.lsp.dev/protocol"
)

func main() {
	logfile := os.Getenv("TG_LS_LOG")
	l := logger.BuildLogger(logfile)
	l.Println("Initializing terragrunt-ls")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(rpc.Split)

	// state := analysis.NewState()
	state := tg.NewState()
	writer := os.Stdout

	for scanner.Scan() {
		msg := scanner.Bytes()

		method, contents, err := rpc.DecodeMessage(msg)
		if err != nil {
			l.Printf("Got an error: %s", err)
			continue
		}

		handleMessage(l, writer, state, method, contents)
	}
}

func handleMessage(l *log.Logger, writer io.Writer, state tg.State, method string, contents []byte) {
	l.Printf("Received msg with method: %s", method)
	l.Printf("Contents: %s", contents)

	switch method {
	case protocol.MethodInitialize:
		var request lsp.InitializeRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			l.Printf("Failed to parse initialize request: %s", err)
		}

		l.Printf("Connected to: %s %s",
			request.Params.ClientInfo.Name,
			request.Params.ClientInfo.Version)

		msg := lsp.NewInitializeResponse(request.ID)
		writeResponse(l, writer, msg)

		l.Print("Initialized")

	case protocol.MethodTextDocumentDidOpen:
		var notification lsp.DidOpenTextDocumentNotification
		if err := json.Unmarshal(contents, &notification); err != nil {
			l.Printf("Failed to parse didOpen request: %s", err)
		}

		l.Printf("Opened: %s", notification.Params.TextDocument.URI)

		diagnostics := state.OpenDocument(l, notification.Params.TextDocument.URI, notification.Params.TextDocument.Text)
		writeResponse(l, writer, lsp.PublishDiagnosticsNotification{
			Notification: lsp.Notification{
				RPC:    lsp.RPCVersion,
				Method: protocol.MethodTextDocumentPublishDiagnostics,
			},
			Params: protocol.PublishDiagnosticsParams{
				URI:         notification.Params.TextDocument.URI,
				Diagnostics: diagnostics,
			},
		})

		l.Print(state.Configs)

		l.Print("Document opened")

	case protocol.MethodTextDocumentDidChange:
		var notification lsp.DidChangeTextDocumentNotification
		if err := json.Unmarshal(contents, &notification); err != nil {
			l.Printf("Failed to parse didChange request: %s", err)
		}

		l.Printf("Changed: %s", notification.Params.TextDocument.URI)

		for _, change := range notification.Params.ContentChanges {
			l.Printf("Change: %s", change.Text)

			diagnostics := state.UpdateDocument(l, notification.Params.TextDocument.URI, change.Text)
			writeResponse(l, writer, lsp.PublishDiagnosticsNotification{
				Notification: lsp.Notification{
					RPC:    lsp.RPCVersion,
					Method: protocol.MethodTextDocumentPublishDiagnostics,
				},
				Params: protocol.PublishDiagnosticsParams{
					URI:         notification.Params.TextDocument.URI,
					Diagnostics: diagnostics,
				},
			})
		}

		l.Print("Document changed")

	case protocol.MethodTextDocumentHover:
		var request lsp.HoverRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			l.Printf("Failed to parse hover request: %s", err)
		}

		l.Printf("Hover: %s", request.Params.TextDocument.URI)

		response := state.Hover(l, request.ID, request.Params.TextDocument.URI, request.Params.Position)

		writeResponse(l, writer, response)

	case protocol.MethodTextDocumentDefinition:
		var request lsp.DefinitionRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			l.Printf("Failed to parse definition request: %s", err)
		}

		l.Printf("Definition: %s", request.Params.TextDocument.URI)

		response := state.Definition(l, request.ID, request.Params.TextDocument.URI, request.Params.Position)

		writeResponse(l, writer, response)

	case protocol.MethodTextDocumentCompletion:
		var request lsp.CompletionRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			l.Printf("Failed to parse completion request: %s", err)
		}

		l.Printf("Completion: %s", request.Params.TextDocument.URI)

		response := state.TextDocumentCompletion(l, request.ID, request.Params.TextDocument.URI, request.Params.Position)

		l.Printf("Completion response: %v", response)

		writeResponse(l, writer, response)
	}
}

func writeResponse(l *log.Logger, writer io.Writer, msg any) {
	reply := rpc.EncodeMessage(msg)

	_, err := writer.Write([]byte(reply))
	if err != nil {
		l.Printf("Failed to write response: %s", err)
	}
}
