package main

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunRepoCheckoutJSONOutput(t *testing.T) {
	var gotReq map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repo/checkout" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"path":        "/tmp/ws/workdir/multica",
			"branch_name": "agent/test/1234",
			"base_ref":    "refs/remotes/origin/main",
			"reused":      true,
		})
	}))
	defer srv.Close()

	t.Setenv("MULTICA_DAEMON_PORT", strconv.Itoa(srv.Listener.Addr().(*net.TCPAddr).Port))
	t.Setenv("MULTICA_WORKSPACE_ID", "ws-1")
	t.Setenv("MULTICA_AGENT_NAME", "Test Agent")
	t.Setenv("MULTICA_TASK_ID", "task-1")

	wd := t.TempDir()
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(wd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origWD)

	cmd := &cobra.Command{Use: "checkout"}
	cmd.Flags().String("output", "json", "")
	cmd.Flags().String("ref", "", "")
	_ = cmd.Flags().Set("output", "json")
	_ = cmd.Flags().Set("ref", "main")

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	if err := runRepoCheckout(cmd, []string{"https://github.com/example/repo.git"}); err != nil {
		t.Fatalf("runRepoCheckout: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close stdout pipe: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}

	if gotReq["url"] != "https://github.com/example/repo.git" {
		t.Fatalf("request url = %q", gotReq["url"])
	}
	if gotReq["workspace_id"] != "ws-1" {
		t.Fatalf("workspace_id = %q", gotReq["workspace_id"])
	}
	if gotReq["workdir"] != wd {
		t.Fatalf("workdir = %q, want %q", gotReq["workdir"], wd)
	}

	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, string(out))
	}
	if payload["path"] != "/tmp/ws/workdir/multica" {
		t.Fatalf("json path = %#v", payload["path"])
	}
	if payload["branch_name"] != "agent/test/1234" {
		t.Fatalf("json branch_name = %#v", payload["branch_name"])
	}
	if payload["base_ref"] != "refs/remotes/origin/main" {
		t.Fatalf("json base_ref = %#v", payload["base_ref"])
	}
	if payload["reused"] != true {
		t.Fatalf("json reused = %#v", payload["reused"])
	}
}
