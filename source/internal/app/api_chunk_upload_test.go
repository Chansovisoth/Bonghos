package app

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
	"github.com/Chansovisoth/Bonghos/internal/config"
	"github.com/Chansovisoth/Bonghos/internal/operations"
)

func testServerArchive(t *testing.T) []byte {
	t.Helper()
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	for name, body := range map[string]string{
		"start.sh":          "#!/bin/bash\njava -Xms1G -Xmx2G -jar server.jar nogui\n",
		"server.properties": "motd=Chunked upload test\n",
		"server.jar":        "test jar placeholder",
	} {
		entry, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}

func putUploadChunk(t *testing.T, c *client, operationID string, offset int64, body []byte) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut,
		c.env.server.URL+"/api/imports/upload/"+operationID+"/chunk", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Bonghos-CSRF", c.csrf)
	req.Header.Set("X-Bonghos-Upload-Offset", strconv.FormatInt(offset, 10))
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var result map[string]any
	_ = json.Unmarshal(raw, &result)
	return resp.StatusCode, result
}

func TestChunkedArchiveUpload(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)
	archive := testServerArchive(t)

	var start struct {
		OperationID string `json:"operation_id"`
		ChunkSize   int64  `json:"chunk_size"`
	}
	status, body := c.do("POST", "/api/imports/upload/start", map[string]any{
		"display_name": "Chunked Test",
		"filename":     "server.zip",
		"size":         len(archive),
	}, &start)
	if status != http.StatusCreated || start.OperationID == "" || start.ChunkSize <= 0 {
		t.Fatalf("start upload: %d %s response=%+v", status, body, start)
	}

	cut := len(archive) / 2
	if status, result := putUploadChunk(t, c, start.OperationID, 0, archive[:cut]); status != 200 || int64(result["offset"].(float64)) != int64(cut) {
		t.Fatalf("first chunk: status=%d result=%v", status, result)
	}
	if status, result := putUploadChunk(t, c, start.OperationID, 0, archive[:cut]); status != http.StatusConflict || int64(result["expected_offset"].(float64)) != int64(cut) {
		t.Fatalf("duplicate offset was not rejected: status=%d result=%v", status, result)
	}
	if status, result := putUploadChunk(t, c, start.OperationID, int64(cut), archive[cut:]); status != 200 || int64(result["offset"].(float64)) != int64(len(archive)) {
		t.Fatalf("second chunk: status=%d result=%v", status, result)
	}

	var finish struct {
		Server struct {
			ServerDirectory string `json:"server_directory"`
		} `json:"server"`
	}
	status, body = c.do("POST", "/api/imports/upload/"+start.OperationID+"/finish", nil, &finish)
	if status != 200 {
		t.Fatalf("finish upload: %d %s", status, body)
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		op, err := env.app.Operations.Get(start.OperationID)
		if err != nil {
			t.Fatal(err)
		}
		if op.Stage == operations.StageCompleted {
			break
		}
		if op.Stage == operations.StageFailed {
			t.Fatalf("chunked import failed: %s", op.Error)
		}
		if time.Now().After(deadline) {
			t.Fatalf("chunked import did not finish; stage=%s", op.Stage)
		}
		time.Sleep(25 * time.Millisecond)
	}
	installed := filepath.Join(env.home, filepath.FromSlash(finish.Server.ServerDirectory), "server.properties")
	if _, err := os.Stat(installed); err != nil {
		t.Fatalf("uploaded server was not installed: %v", err)
	}
}

func TestChunkedArchiveUploadCancellationRemovesPartialFile(t *testing.T) {
	env := newTestEnv(t)
	secret := env.createUser("owner", "correct horse battery", authorization.RoleOwner)
	c := env.newClient()
	c.mustLogin("owner", "correct horse battery", secret)

	var start struct {
		OperationID string `json:"operation_id"`
	}
	status, body := c.do("POST", "/api/imports/upload/start", map[string]any{
		"display_name": "Cancelled Test", "filename": "server.zip", "size": 10,
	}, &start)
	if status != http.StatusCreated {
		t.Fatalf("start upload: %d %s", status, body)
	}
	if status, _ := putUploadChunk(t, c, start.OperationID, 0, []byte("12345")); status != 200 {
		t.Fatalf("partial chunk status=%d", status)
	}
	status, body = c.do("POST", "/api/operations/"+start.OperationID+"/cancel", nil, nil)
	if status != 200 {
		t.Fatalf("cancel upload: %d %s", status, body)
	}
	if _, err := os.Stat(filepath.Join(env.home, config.DirUploads, start.OperationID)); !os.IsNotExist(err) {
		t.Fatalf("partial upload directory still exists: %v", err)
	}
	op, err := env.app.Operations.Get(start.OperationID)
	if err != nil || op.Stage != operations.StageCancelled {
		t.Fatalf("operation after cancel: op=%+v err=%v", op, err)
	}
}
