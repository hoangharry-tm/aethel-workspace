//go:build integration

// Run with:
//   go test -v -tags integration ./integration/... -base-url="http://localhost:8080"
// Requires a running Aethel backend with migrations applied and a seeded admin user.

package integration

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"testing"
)

var baseURL = flag.String("base-url", "http://localhost:8080", "backend base URL")

// TestGreenNoteChain_EndToEnd verifies the full workflow:
// login → create dispatch → get minute sheet → append 3 green notes →
// verify chain integrity → approve minute sheet.
func TestGreenNoteChain_EndToEnd(t *testing.T) {
	flag.Parse()
	base := *baseURL

	// Step 1: Login.
	loginBody, _ := json.Marshal(map[string]string{
		"emailAddress": "admin@example.com",
		"password":     "admin123",
	})
	resp, err := http.Post(base+"/api/v1/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("POST /auth/login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST /auth/login: status=%d body=%s", resp.StatusCode, body)
	}
	var loginResp struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginResp.AccessToken == "" {
		t.Fatal("expected non-empty accessToken")
	}
	token := loginResp.AccessToken

	doRequest := func(method, path string, body []byte) *http.Response {
		t.Helper()
		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}
		req, err := http.NewRequest(method, base+path, bodyReader)
		if err != nil {
			t.Fatalf("build request %s %s: %v", method, path, err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		return resp
	}

	// Step 2: Get a document type to use in the dispatch.
	// We need a real document type ID from the running server.
	dtResp := doRequest("GET", "/api/v1/admin/document-types", nil)
	defer dtResp.Body.Close()
	if dtResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(dtResp.Body)
		t.Fatalf("GET /admin/document-types: status=%d body=%s", dtResp.StatusCode, body)
	}
	var docTypes []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(dtResp.Body).Decode(&docTypes); err != nil {
		t.Fatalf("decode document types: %v", err)
	}
	if len(docTypes) == 0 {
		t.Skip("no document types seeded — skipping end-to-end test")
	}
	dtID := docTypes[0].ID

	// Step 3: Create an inbound dispatch.
	dispatchBody, _ := json.Marshal(map[string]interface{}{
		"direction":      "INBOUND",
		"documentTypeId": dtID,
		"senderName":     "E2E Test Sender",
		"priorityLevel":  "ROUTINE",
	})
	dispResp := doRequest("POST", "/api/v1/dispatches", dispatchBody)
	defer dispResp.Body.Close()
	if dispResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(dispResp.Body)
		t.Fatalf("POST /dispatches: status=%d body=%s", dispResp.StatusCode, body)
	}
	var dispatch struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(dispResp.Body).Decode(&dispatch); err != nil {
		t.Fatalf("decode dispatch response: %v", err)
	}
	if dispatch.ID == "" {
		t.Fatal("expected non-empty dispatch ID")
	}
	dispatchID := dispatch.ID
	t.Logf("created dispatch %s", dispatchID)

	// Step 4: GET the minute sheet for this dispatch.
	msResp := doRequest("GET", fmt.Sprintf("/api/v1/dispatches/%s/minute-sheet", dispatchID), nil)
	defer msResp.Body.Close()
	if msResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(msResp.Body)
		t.Fatalf("GET /dispatches/%s/minute-sheet: status=%d body=%s", dispatchID, msResp.StatusCode, body)
	}
	var minuteSheet struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(msResp.Body).Decode(&minuteSheet); err != nil {
		t.Fatalf("decode minute sheet: %v", err)
	}
	t.Logf("minute sheet ID: %s", minuteSheet.ID)

	// Steps 5-7: Append 3 green notes.
	noteContents := []string{"Note One", "Note Two", "Note Three"}
	for i, content := range noteContents {
		noteBody, _ := json.Marshal(map[string]string{"content": content})
		noteResp := doRequest("POST", fmt.Sprintf("/api/v1/dispatches/%s/green-notes", dispatchID), noteBody)
		defer noteResp.Body.Close()
		if noteResp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(noteResp.Body)
			t.Fatalf("POST green-note %d: status=%d body=%s", i+1, noteResp.StatusCode, body)
		}
		var note struct {
			SequenceOrder int `json:"sequenceOrder"`
		}
		if err := json.NewDecoder(noteResp.Body).Decode(&note); err != nil {
			t.Fatalf("decode green note %d: %v", i+1, err)
		}
		if note.SequenceOrder != i+1 {
			t.Errorf("note %d: want sequenceOrder=%d, got %d", i+1, i+1, note.SequenceOrder)
		}
	}

	// Step 8: GET all green notes and verify chain integrity.
	listResp := doRequest("GET", fmt.Sprintf("/api/v1/dispatches/%s/green-notes", dispatchID), nil)
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(listResp.Body)
		t.Fatalf("GET /green-notes: status=%d body=%s", listResp.StatusCode, body)
	}
	var notes []struct {
		SequenceOrder     int    `json:"sequenceOrder"`
		PreviousHash      string `json:"previousHash"`
		CryptographicHash string `json:"cryptographicHash"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&notes); err != nil {
		t.Fatalf("decode notes list: %v", err)
	}
	if len(notes) != 3 {
		t.Fatalf("want 3 notes, got %d", len(notes))
	}
	for i := 1; i < len(notes); i++ {
		if notes[i].PreviousHash != notes[i-1].CryptographicHash {
			t.Errorf("chain broken at note %d: previousHash=%q, note[%d].cryptographicHash=%q",
				i+1, notes[i].PreviousHash, i, notes[i-1].CryptographicHash)
		}
	}
	t.Logf("chain integrity verified for %d notes", len(notes))

	// Step 9: Approve the minute sheet.
	approveResp := doRequest("POST", fmt.Sprintf("/api/v1/dispatches/%s/minute-sheet/approve", dispatchID), nil)
	defer approveResp.Body.Close()
	if approveResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(approveResp.Body)
		t.Fatalf("POST /minute-sheet/approve: status=%d body=%s", approveResp.StatusCode, body)
	}
	t.Log("minute sheet approved successfully")
}
