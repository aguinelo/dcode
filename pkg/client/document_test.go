package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/aguinelo/dcode/internal/protocol"
)

// SubmitTurn sends the whole request — text, pictures and documents — as the
// daemon reads it, so a client attaching a file does not need a route of its
// own or a second call.
func TestSubmitTurnCarriesTheDocuments(t *testing.T) {
	got := make(chan protocol.SubmitTurnRequest, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /"+protocol.Version+"/sessions/s1/turns", func(w http.ResponseWriter, r *http.Request) {
		var req protocol.SubmitTurnRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		got <- req
		w.WriteHeader(http.StatusAccepted)
	})
	c := serveOn(t, mux)

	want := protocol.TurnDocument{Name: "spec.pdf", MediaType: protocol.DocumentPDF, Data: "JVBERi0="}
	if err := c.SubmitTurn(context.Background(), "s1", protocol.SubmitTurnRequest{
		Text: "read", Documents: []protocol.TurnDocument{want},
	}); err != nil {
		t.Fatal(err)
	}
	req := <-got
	if req.Text != "read" || len(req.Documents) != 1 || req.Documents[0] != want {
		t.Errorf("the daemon read %+v", req)
	}
}
