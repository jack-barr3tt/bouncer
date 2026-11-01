package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jack-barr3tt/bouncer/internal/storage"
	"github.com/jack-barr3tt/bouncer/internal/store"
)

type storedDocument struct {
	Data     json.RawMessage `json:"data"`
	Revision string          `json:"revision"`
}

func (s *Server) GetAppStorage(c fiber.Ctx, slug Slug) error {
	key, err := s.storageKey(c, slug)
	if err != nil {
		return err
	}
	obj, err := s.objects.Get(c.Context(), key)
	if errors.Is(err, storage.ErrNotFound) {
		return c.JSON(storedDocument{Data: json.RawMessage(`{}`), Revision: ""})
	}
	if err != nil {
		return s.internal(c, err)
	}
	return c.JSON(storedDocument{Data: json.RawMessage(obj.Body), Revision: obj.Revision})
}

func (s *Server) PutAppStorage(c fiber.Ctx, slug Slug, params PutAppStorageParams) error {
	key, err := s.storageKey(c, slug)
	if err != nil {
		return err
	}
	body := c.Body()
	if len(body) > s.storageMax {
		return writeError(c, fiber.StatusRequestEntityTooLarge, "Document is too large.")
	}
	if !jsonObject(body) {
		return writeError(c, fiber.StatusBadRequest, "Send a JSON object.")
	}
	revision := ""
	if params.IfMatch != nil {
		revision = *params.IfMatch
	}
	next, err := s.objects.Put(c.Context(), key, body, revision)
	if errors.Is(err, storage.ErrConflict) {
		return writeError(c, fiber.StatusConflict, "The stored document changed. Load it again.")
	}
	if err != nil {
		return s.internal(c, err)
	}
	return c.JSON(AppStorageRevision{Revision: next})
}

func (s *Server) DeleteAppStorage(c fiber.Ctx, slug Slug) error {
	key, err := s.storageKey(c, slug)
	if err != nil {
		return err
	}
	if err := s.objects.Delete(c.Context(), key); err != nil {
		return s.internal(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) forgetTempStorage(ctx context.Context, ids ...uuid.UUID) {
	if s.objects == nil {
		return
	}
	for _, id := range ids {
		if err := s.objects.DeletePrefix(ctx, storage.TempPrefix(id)); err != nil {
			log.Printf("storage cleanup failed: %v", err)
		}
	}
}

func (s *Server) storageKey(c fiber.Ctx, slug string) (string, error) {
	principal, err := s.require(c)
	if err != nil {
		return "", err
	}
	ok, err := s.site.HasApp(slug)
	if err != nil {
		return "", s.internal(c, err)
	}
	if !ok {
		return "", writeError(c, fiber.StatusNotFound, "Unknown app.")
	}
	if !contains(principal.Apps, slug) {
		return "", writeError(c, fiber.StatusForbidden, "You cannot open this app.")
	}
	if s.objects == nil {
		return "", writeError(c, fiber.StatusServiceUnavailable, "Storage is not configured.")
	}
	key, err := accountKey(principal, slug)
	if err != nil {
		return "", s.internal(c, err)
	}
	return key, nil
}

func accountKey(principal *store.Principal, slug string) (string, error) {
	if principal.UserID != nil {
		return storage.UserKey(*principal.UserID, slug), nil
	}
	if principal.TempID != nil {
		return storage.TempKey(*principal.TempID, slug), nil
	}
	return "", errors.New("session has no account")
}

func jsonObject(body []byte) bool {
	dec := json.NewDecoder(bytes.NewReader(body))
	var value json.RawMessage
	if err := dec.Decode(&value); err != nil {
		return false
	}
	if dec.More() {
		return false
	}
	trimmed := bytes.TrimSpace(value)
	return len(trimmed) > 0 && trimmed[0] == '{'
}
