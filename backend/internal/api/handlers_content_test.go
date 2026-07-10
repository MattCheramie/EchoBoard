package api_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"testing"

	"github.com/MattCheramie/echoboard/internal/content"
)

func loginAdmin(t *testing.T, srv string) *http.Client {
	t.Helper()
	c, _ := newJarClient()
	post(t, c, srv+"/api/auth/login", `{"email":"admin@echo.test","password":"supersecret"}`).Body.Close()
	return c
}

func TestContentAPICRUD(t *testing.T) {
	srv := newServer(t)
	admin := loginAdmin(t, srv.URL)

	// Create.
	resp := post(t, admin, srv.URL+"/api/content",
		`{"title":"Launch","body":"hello","targets":["sandbox"],"tags":["promo"]}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", resp.StatusCode)
	}
	var created content.Content
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
	if created.ID == "" || created.Status != content.StatusDraft {
		t.Fatalf("created = %+v", created)
	}

	// Get.
	resp, _ = admin.Get(srv.URL + "/api/content/" + created.ID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d", resp.StatusCode)
	}
	resp.Body.Close()

	// List.
	resp, _ = admin.Get(srv.URL + "/api/content")
	var list []content.Content
	json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}

	// Patch title.
	req, _ := http.NewRequest(http.MethodPatch, srv.URL+"/api/content/"+created.ID,
		bytes.NewBufferString(`{"title":"Updated"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = admin.Do(req)
	var patched content.Content
	json.NewDecoder(resp.Body).Decode(&patched)
	resp.Body.Close()
	if patched.Title != "Updated" {
		t.Errorf("patched title = %q", patched.Title)
	}

	// Delete then 404.
	req, _ = http.NewRequest(http.MethodDelete, srv.URL+"/api/content/"+created.ID, nil)
	resp, _ = admin.Do(req)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete status = %d, want 204", resp.StatusCode)
	}
	resp.Body.Close()
	resp, _ = admin.Get(srv.URL + "/api/content/" + created.ID)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("get after delete = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestContentRequiresAuth(t *testing.T) {
	srv := newServer(t)
	resp := post(t, http.DefaultClient, srv.URL+"/api/content", `{"title":"x"}`)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauth create = %d, want 401", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestMediaUploadAPI(t *testing.T) {
	srv := newServer(t)
	admin := loginAdmin(t, srv.URL)

	// Build a multipart body with a small PNG declared as image/png.
	pngData := apiPNG(t, 60, 40)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="pic.png"`)
	hdr.Set("Content-Type", "image/png")
	part, _ := mw.CreatePart(hdr)
	part.Write(pngData)
	mw.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/media", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := admin.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload status = %d, want 201", resp.StatusCode)
	}
	var m content.Media
	json.NewDecoder(resp.Body).Decode(&m)
	resp.Body.Close()
	if m.ID == "" || m.Size != int64(len(pngData)) {
		t.Fatalf("media = %+v", m)
	}

	// Fetch original.
	resp, _ = admin.Get(srv.URL + "/api/media/" + m.ID)
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("get media status=%d ct=%q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	got, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !bytes.Equal(got, pngData) {
		t.Error("fetched media bytes differ from upload")
	}

	// Fetch thumbnail (JPEG).
	resp, _ = admin.Get(srv.URL + "/api/media/" + m.ID + "?thumb=1")
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "image/jpeg" {
		t.Errorf("get thumb status=%d ct=%q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	resp.Body.Close()

	// Delete.
	req, _ = http.NewRequest(http.MethodDelete, srv.URL+"/api/media/"+m.ID, nil)
	resp, _ = admin.Do(req)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete media = %d, want 204", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestTagsAPI(t *testing.T) {
	srv := newServer(t)
	admin := loginAdmin(t, srv.URL)

	resp := post(t, admin, srv.URL+"/api/tags", `{"name":"campaign"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create tag = %d, want 201", resp.StatusCode)
	}
	resp.Body.Close()

	resp, _ = admin.Get(srv.URL + "/api/tags")
	var tags []content.Tag
	json.NewDecoder(resp.Body).Decode(&tags)
	resp.Body.Close()
	if len(tags) != 1 || tags[0].Name != "campaign" {
		t.Errorf("tags = %+v, want [campaign]", tags)
	}
}

func apiPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 200, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png: %v", err)
	}
	return buf.Bytes()
}
