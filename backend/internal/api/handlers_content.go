package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/MattCheramie/echoboard/internal/content"
	"github.com/MattCheramie/echoboard/internal/media"
)

// maxUploadBytes caps a single media upload.
const maxUploadBytes = 32 << 20 // 32 MiB

type createContentRequest struct {
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	Targets     []string   `json:"targets"`
	Tags        []string   `json:"tags"`
	MediaIDs    []string   `json:"mediaIds"`
	ScheduledAt *time.Time `json:"scheduledAt"`
}

func (a *API) handleCreateContent(w http.ResponseWriter, r *http.Request) {
	uid, _ := currentUserID(r)
	var req createContentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c, err := a.content.CreateContent(r.Context(), content.CreateInput{
		AuthorID:    uid,
		Title:       req.Title,
		Body:        req.Body,
		Targets:     req.Targets,
		Tags:        req.Tags,
		MediaIDs:    req.MediaIDs,
		ScheduledAt: req.ScheduledAt,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (a *API) handleListContent(w http.ResponseWriter, r *http.Request) {
	list, err := a.content.ListContent(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list content")
		return
	}
	if list == nil {
		list = []*content.Content{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (a *API) handleGetContent(w http.ResponseWriter, r *http.Request) {
	c, err := a.content.GetContent(r.Context(), r.PathValue("id"))
	if errors.Is(err, content.ErrNotFound) {
		writeError(w, http.StatusNotFound, "content not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load content")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

type updateContentRequest struct {
	Title       *string    `json:"title"`
	Body        *string    `json:"body"`
	Status      *string    `json:"status"`
	Targets     *[]string  `json:"targets"`
	Tags        *[]string  `json:"tags"`
	MediaIDs    *[]string  `json:"mediaIds"`
	ScheduledAt *time.Time `json:"scheduledAt"`
}

func (a *API) handleUpdateContent(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read request body")
		return
	}
	// Detect which keys are present so a partial PATCH can distinguish
	// "scheduledAt omitted" (leave as-is) from "scheduledAt: null" (clear).
	var present map[string]json.RawMessage
	if err := json.Unmarshal(body, &present); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var req updateContentRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	in := content.UpdateInput{
		Title:    req.Title,
		Body:     req.Body,
		Targets:  req.Targets,
		Tags:     req.Tags,
		MediaIDs: req.MediaIDs,
	}
	if req.Status != nil {
		s := content.Status(*req.Status)
		in.Status = &s
	}
	if _, ok := present["scheduledAt"]; ok {
		in.SetSchedule = true
		in.ScheduledAt = req.ScheduledAt
	}

	c, err := a.content.UpdateContent(r.Context(), r.PathValue("id"), in)
	if errors.Is(err, content.ErrNotFound) {
		writeError(w, http.StatusNotFound, "content not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (a *API) handleDeleteContent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := a.content.GetContent(r.Context(), id); errors.Is(err, content.ErrNotFound) {
		writeError(w, http.StatusNotFound, "content not found")
		return
	}
	if err := a.content.DeleteContent(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete content")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleUploadMedia(w http.ResponseWriter, r *http.Request) {
	uid, _ := currentUserID(r)
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "upload too large or malformed")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing 'file' field")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	m, err := a.content.UploadMedia(r.Context(), uid, header.Filename, contentType, file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not store media")
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (a *API) handleGetMedia(w http.ResponseWriter, r *http.Request) {
	thumb := r.URL.Query().Get("thumb") == "1" || r.URL.Query().Get("thumb") == "true"
	m, rc, contentType, err := a.content.OpenMedia(r.Context(), r.PathValue("id"), thumb)
	if errors.Is(err, content.ErrNotFound) || errors.Is(err, media.ErrNotFound) {
		writeError(w, http.StatusNotFound, "media not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load media")
		return
	}
	defer rc.Close()
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_ = m // metadata available if needed (e.g. filename)
	_, _ = io.Copy(w, rc)
}

func (a *API) handleDeleteMedia(w http.ResponseWriter, r *http.Request) {
	err := a.content.DeleteMedia(r.Context(), r.PathValue("id"))
	if errors.Is(err, content.ErrNotFound) {
		writeError(w, http.StatusNotFound, "media not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete media")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleListTags(w http.ResponseWriter, r *http.Request) {
	tags, err := a.content.ListTags(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list tags")
		return
	}
	if tags == nil {
		tags = []*content.Tag{}
	}
	writeJSON(w, http.StatusOK, tags)
}

type createTagRequest struct {
	Name string `json:"name"`
}

func (a *API) handleCreateTag(w http.ResponseWriter, r *http.Request) {
	var req createTagRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "tag name is required")
		return
	}
	t, err := a.content.CreateTag(r.Context(), req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create tag")
		return
	}
	writeJSON(w, http.StatusCreated, t)
}
