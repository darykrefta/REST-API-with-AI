package events

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"rest.api/auth"
	"rest.api/internal/uploads"
)

// Handler exposes REST handlers for events (use with Go 1.22+ http.ServeMux path patterns).
type Handler struct {
	DB        *sql.DB
	ImagesDir string
}

type eventResponse struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Address     string `json:"address"`
	Date        string `json:"date"`
	UserID      int64  `json:"user_id,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
}

func toEventResponse(e Event) eventResponse {
	return eventResponse{
		ID:          e.ID,
		Title:       e.Title,
		Description: e.Description,
		Address:     e.Address,
		Date:        e.Date.Format(time.RFC3339),
		UserID:      e.UserID,
		ImageURL:    e.ImageURL,
	}
}

// Shown when a valid JWT belongs to a different user than the event creator (403).
const errNotOwnerToken = "not the valid user token for this event; only the creator may update or delete it"

func requireJWTUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	s, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return 0, false
	}
	uid, err := strconv.ParseInt(s, 10, 64)
	if err != nil || uid < 1 {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid user id in token"})
		return 0, false
	}
	return uid, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type createUpdateBody struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Address     string `json:"address"`
	Date        string `json:"date"`
	ImageURL    string `json:"image_url"`
}

func parseEventBodyJSON(r *http.Request) (CreateEventInput, error) {
	var body createUpdateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return CreateEventInput{}, err
	}
	t, err := parseEventDate(body.Date)
	if err != nil {
		return CreateEventInput{}, err
	}
	return CreateEventInput{
		Title:       body.Title,
		Description: body.Description,
		Address:     body.Address,
		Date:        t,
		ImageURL:    strings.TrimSpace(body.ImageURL),
	}, nil
}

func isMultipartForm(r *http.Request) bool {
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	return strings.HasPrefix(ct, "multipart/form-data")
}

func writeImageSaveError(w http.ResponseWriter, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "unsupported image"), msg == "empty file":
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
	case strings.Contains(msg, "too large"):
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "file too large"})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
	}
}

func (h *Handler) parseCreateMultipart(r *http.Request) (CreateEventInput, error) {
	if err := r.ParseMultipartForm(uploads.MaxImageBytes); err != nil {
		return CreateEventInput{}, fmt.Errorf("parse multipart: %w", err)
	}
	t, err := parseEventDate(r.FormValue("date"))
	if err != nil {
		return CreateEventInput{}, err
	}
	in := CreateEventInput{
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		Address:     r.FormValue("address"),
		Date:        t,
	}
	base, err := uploads.SaveImageIfPresent(r, h.ImagesDir)
	if err != nil {
		return CreateEventInput{}, err
	}
	if base == "" {
		return CreateEventInput{}, errors.New("image file is required")
	}
	in.ImageURL = "/images/" + base
	return in, nil
}

func (h *Handler) parseUpdateMultipart(r *http.Request, existing Event) (UpdateEventInput, error) {
	if err := r.ParseMultipartForm(uploads.MaxImageBytes); err != nil {
		return UpdateEventInput{}, fmt.Errorf("parse multipart: %w", err)
	}
	t, err := parseEventDate(r.FormValue("date"))
	if err != nil {
		return UpdateEventInput{}, err
	}
	up := UpdateEventInput{
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		Address:     r.FormValue("address"),
		Date:        t,
	}
	base, err := uploads.SaveImageIfPresent(r, h.ImagesDir)
	if err != nil {
		return UpdateEventInput{}, err
	}
	if base != "" {
		up.ImageURL = "/images/" + base
	} else {
		up.ImageURL = existing.ImageURL
	}
	return up, nil
}

// List handles GET /events — returns all events ordered by date.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	list, err := ListEvents(r.Context(), h.DB)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list events"})
		return
	}
	out := make([]eventResponse, 0, len(list))
	for _, e := range list {
		out = append(out, toEventResponse(e))
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": out})
}

// Create handles POST /events — JSON or multipart/form-data.
// Multipart must include text fields title, description, address, date (RFC3339) and file field "image".
// JSON may include optional image_url; multipart is the supported way to upload an image with the event.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	uid, ok := requireJWTUserID(w, r)
	if !ok {
		return
	}
	var in CreateEventInput
	var err error
	if isMultipartForm(r) {
		if h.ImagesDir == "" {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "images storage is not configured"})
			return
		}
		in, err = h.parseCreateMultipart(r)
	} else {
		in, err = parseEventBodyJSON(r)
	}
	if err != nil {
		if err.Error() == "image file is required" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "parse multipart:") || strings.Contains(err.Error(), "multipart:") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "unsupported image") || err.Error() == "empty file" || strings.Contains(err.Error(), "too large") {
			writeImageSaveError(w, err)
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := validateEventInput(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	in.UserID = uid
	ev, err := CreateEvent(r.Context(), h.DB, in)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create event"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"event": toEventResponse(ev)})
}

// Get handles GET /events/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid event id"})
		return
	}
	ev, err := GetEventByID(r.Context(), h.DB, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get event"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"event": toEventResponse(ev)})
}

// Update handles PUT /events/{id} — full replace JSON or multipart/form-data (same fields as create; image optional on multipart — keeps previous if omitted).
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid event id"})
		return
	}
	ownerID, ok := requireJWTUserID(w, r)
	if !ok {
		return
	}
	var up UpdateEventInput
	if isMultipartForm(r) {
		if h.ImagesDir == "" {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "images storage is not configured"})
			return
		}
		existing, err := GetEventByID(r.Context(), h.DB, id)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load event"})
			return
		}
		if existing.UserID != ownerID {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": errNotOwnerToken})
			return
		}
		up, err = h.parseUpdateMultipart(r, existing)
		if err != nil {
			if strings.Contains(err.Error(), "unsupported image") || err.Error() == "empty file" || strings.Contains(err.Error(), "too large") {
				writeImageSaveError(w, err)
				return
			}
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		cin := CreateEventInput{Title: up.Title, Description: up.Description, Address: up.Address, Date: up.Date}
		if err := validateEventInput(&cin); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		up.Title, up.Description, up.Address = cin.Title, cin.Description, cin.Address
		up.Date = cin.Date
	} else {
		in, err := parseEventBodyJSON(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := validateEventInput(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		up = UpdateEventInput{
			Title:       in.Title,
			Description: in.Description,
			Address:     in.Address,
			Date:        in.Date,
			ImageURL:    in.ImageURL,
		}
	}
	ev, err := UpdateEvent(r.Context(), h.DB, id, ownerID, up)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
			return
		}
		if errors.Is(err, ErrNotOwner) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": errNotOwnerToken})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update event"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"event": toEventResponse(ev)})
}

// Delete handles DELETE /events/{id}.
// Returns 200 with the deleted event in the body and logs it to the server console.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid event id"})
		return
	}
	ownerID, ok := requireJWTUserID(w, r)
	if !ok {
		return
	}
	ev, err := GetEventByID(r.Context(), h.DB, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load event"})
		return
	}
	if err := DeleteEvent(r.Context(), h.DB, id, ownerID); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
			return
		}
		if errors.Is(err, ErrNotOwner) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": errNotOwnerToken})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete event"})
		return
	}

	deleted := toEventResponse(ev)
	log.Printf(
		"evento deletado: id=%d title=%q description=%q address=%q date=%s",
		deleted.ID, deleted.Title, deleted.Description, deleted.Address, deleted.Date,
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "event deleted successfully",
		"event":   deleted,
	})
}

// Register handles POST /events/{id}/register — authenticated user joins the event.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid event id"})
		return
	}
	userID, ok := requireJWTUserID(w, r)
	if !ok {
		return
	}
	if err := RegisterForEvent(r.Context(), h.DB, eventID, userID); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
			return
		}
		if errors.Is(err, ErrAlreadyRegistered) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "already registered for this event"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to register"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"message":  "registered for event",
		"event_id": eventID,
		"user_id":  userID,
	})
}

// Unregister handles DELETE /events/{id}/register — authenticated user leaves the event.
func (h *Handler) Unregister(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid event id"})
		return
	}
	userID, ok := requireJWTUserID(w, r)
	if !ok {
		return
	}
	if err := UnregisterFromEvent(r.Context(), h.DB, eventID, userID); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
			return
		}
		if errors.Is(err, ErrNotRegistered) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not registered for this event"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to unregister"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message":  "unregistered from event",
		"event_id": eventID,
		"user_id":  userID,
	})
}

func pathID(r *http.Request) (int64, error) {
	s := r.PathValue("id")
	if s == "" {
		return 0, errors.New("missing id")
	}
	return strconv.ParseInt(s, 10, 64)
}
