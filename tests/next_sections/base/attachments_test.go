// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package baseprocess

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

func TestBaseAttachmentUploadDownloadRemove(t *testing.T) {
	const content = "Synthetic attachment\n中文\x00binary\n"
	var mu sync.Mutex
	var requests []string
	fixture := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		response := `{"code":0,"data":{}}`
		switch r.Method + " " + r.URL.Path {
		case "GET " + tp + "/fields/附件":
			response = `{"code":0,"data":{"id":"fld_attachment","name":"附件","type":"attachment"}}`
		case "POST /open-apis/drive/v1/medias/upload_all":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Error(err)
				break
			}
			defer r.MultipartForm.RemoveAll()
			for key, want := range map[string]string{"file_name": "payload.txt", "parent_type": "bitable_file", "parent_node": "bas_fixture"} {
				if got := r.FormValue(key); got != want {
					t.Errorf("multipart %s=%q want=%q", key, got, want)
				}
			}
			file, _, err := r.FormFile("file")
			if err != nil {
				t.Error(err)
				break
			}
			defer file.Close()
			raw, err := io.ReadAll(file)
			if err != nil || string(raw) != content {
				t.Errorf("upload bytes=%q err=%v", raw, err)
			}
			response = `{"code":0,"data":{"file_token":"box_fixture"}}`
		case "POST /open-apis/drive/v1/lark_cli_file_event/report":
			raw, _ := io.ReadAll(r.Body)
			assertJSON(t, string(raw), `{"file_scene":"lark-cli","scene":"upload","operation":"upload","tags":{"code":"","api_path":"/open-apis/drive/v1/medias/upload_all","command":"base +record-upload-attachment","resource_type":"media","status":"success","mount_point":"bitable_file","file_token":"box_fixture"}}`, false)
			response = `{"code":0,"data":{}}`
		case "POST " + tp + "/append_attachments", "POST " + tp + "/remove_attachments":
			raw, _ := io.ReadAll(r.Body)
			assertJSON(t, string(raw), `{"attachments":{"rec_fixture":{"fld_attachment":[{"file_token":"box_fixture"}]}}}`, false)
			response = `{"code":0,"data":{"record_id":"rec_fixture","field_id":"fld_attachment"}}`
		case "POST " + tp + "/get_attachments":
			raw, _ := io.ReadAll(r.Body)
			assertJSON(t, string(raw), `{"record_id_list":["rec_fixture"]}`, false)
			response = `{"code":0,"data":{"attachments":{"rec_fixture":{"fld_attachment":[{"file_token":"box_fixture","name":"payload.txt","size":35,"extra_info":"synthetic-extra","mime_type":"text/plain"}]}}}}`
		case "GET /open-apis/drive/v1/medias/box_fixture/download":
			if r.URL.Query().Get("extra") != "synthetic-extra" {
				t.Errorf("missing attachment extra: %s", r.URL)
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = io.WriteString(w, content)
			return
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			http.Error(w, "unexpected request", 500)
			return
		}
		_, _ = io.WriteString(w, response)
	}))
	input := filepath.Join(fixture.WorkDir, "payload.txt")
	if err := os.WriteFile(input, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	upload := fixture.Run(table("record-upload-attachment", "--record-id", "rec_fixture", "--field-id", "附件", "--file", input)...)
	if upload.ExitCode != 0 {
		t.Fatal(upload)
	}
	assertJSON(t, upload.Stdout, `{"record_id":"rec_fixture","field_id":"fld_attachment"}`, true)
	output := filepath.Join(fixture.WorkDir, "downloaded.bin")
	download := fixture.Run(table("record-download-attachment", "--record-id", "rec_fixture", "--file-token", "box_fixture", "--output", output)...)
	if download.ExitCode != 0 {
		t.Fatal(download)
	}
	raw, err := os.ReadFile(output)
	if err != nil || string(raw) != content {
		t.Fatalf("download bytes=%q err=%v", raw, err)
	}
	assertJSON(t, download.Stdout, `{"downloaded":[{"record_id":"rec_fixture","field_id":"fld_attachment","file_token":"box_fixture"}]}`, true)
	remove := fixture.Run(table("record-remove-attachment", "--record-id", "rec_fixture", "--field-id", "附件", "--file-token", "box_fixture", "--yes")...)
	if remove.ExitCode != 0 {
		t.Fatal(remove)
	}
	assertJSON(t, remove.Stdout, `{"record_id":"rec_fixture","field_id":"fld_attachment"}`, true)
	want := []string{"GET " + tp + "/fields/附件", "POST /open-apis/drive/v1/medias/upload_all", "POST /open-apis/drive/v1/lark_cli_file_event/report", "POST " + tp + "/append_attachments", "POST " + tp + "/get_attachments", "GET /open-apis/drive/v1/medias/box_fixture/download", "GET " + tp + "/fields/附件", "POST " + tp + "/remove_attachments"}
	mu.Lock()
	defer mu.Unlock()
	if !reflect.DeepEqual(requests, want) {
		t.Fatalf("requests=%v want=%v", requests, want)
	}
}
