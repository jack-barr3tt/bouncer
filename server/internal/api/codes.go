package api

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jack-barr3tt/bouncer/internal/store"
	"github.com/skip2/go-qrcode"
)

func (s *Server) RedeemAccessCode(c fiber.Ctx) error {
	var req RedeemRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request.")
	}
	fingerprint := strings.TrimSpace(req.Fingerprint)
	buckets := codeBuckets(c.IP(), fingerprint)
	locked, err := s.store.Locked(c.Context(), bucketNames(buckets))
	if err != nil {
		return s.internal(c, err)
	}
	if locked {
		return writeError(c, fiber.StatusTooManyRequests, "Too many attempts. Try again later.")
	}
	if fingerprint == "" {
		return writeError(c, fiber.StatusBadRequest, "A browser check is required to join.")
	}
	code, ok := store.NormalizeCode(req.Code)
	if !ok {
		if err := s.store.Fail(c.Context(), buckets); err != nil {
			return s.internal(c, err)
		}
		return writeError(c, fiber.StatusUnauthorized, "That code is not valid.")
	}
	tempID, expires, err := s.store.Redeem(c.Context(), code, req.Nickname, store.HashFingerprint(fingerprint), c.IP())
	if errors.Is(err, store.ErrInvalidCode) {
		if err := s.store.Fail(c.Context(), buckets); err != nil {
			return s.internal(c, err)
		}
		return writeError(c, fiber.StatusUnauthorized, "That code is not valid.")
	}
	if errors.Is(err, store.ErrBadNickname) {
		return writeError(c, fiber.StatusBadRequest, "Enter a nickname.")
	}
	if errors.Is(err, store.ErrCodeFull) {
		return writeError(c, fiber.StatusBadRequest, "This code has no signups left.")
	}
	if errors.Is(err, store.ErrRevokedAccount) {
		return writeError(c, fiber.StatusBadRequest, "This access code cannot be used.")
	}
	if err != nil {
		return s.internal(c, err)
	}
	if err := s.store.Clear(c.Context(), bucketNames(buckets)); err != nil {
		return s.internal(c, err)
	}
	token, err := s.store.CreateSession(c.Context(), nil, &tempID, expires)
	if err != nil {
		return s.internal(c, err)
	}
	s.setCookie(c, token, expires)
	principal, err := s.principal(c, token)
	if err != nil {
		return s.internal(c, err)
	}
	return c.JSON(sessionJSON(principal))
}

func (s *Server) ListAccessCodes(c fiber.Ctx) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	codes, err := s.store.ListCodes(c.Context())
	if err != nil {
		return s.internal(c, err)
	}
	out := make([]AccessCode, 0, len(codes))
	for _, code := range codes {
		out = append(out, s.codeJSON(c, code))
	}
	return c.JSON(AccessCodeList{Codes: out})
}

func (s *Server) CreateAccessCode(c fiber.Ctx) error {
	principal, err := s.requireAdmin(c)
	if err != nil {
		return err
	}
	var req CreateAccessCodeRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request.")
	}
	label := strings.TrimSpace(stringValue(req.Label))
	if len([]rune(label)) > 80 {
		return writeError(c, fiber.StatusBadRequest, "Label is too long.")
	}
	if req.MaxSignups < 1 || req.MaxSignups > 1000 {
		return writeError(c, fiber.StatusBadRequest, "Signup limit must be between 1 and 1000.")
	}
	if !req.ExpiresAt.After(time.Now()) {
		return writeError(c, fiber.StatusBadRequest, "Expiry must be in the future.")
	}
	slugs, err := s.knownSlugs(req.AppSlugs)
	if err != nil || len(slugs) == 0 {
		return writeError(c, fiber.StatusBadRequest, "Choose at least one app.")
	}
	code, err := s.store.CreateCode(c.Context(), *principal.UserID, label, req.ExpiresAt, req.MaxSignups, slugs)
	if err != nil {
		return s.internal(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(s.codeJSON(c, code))
}

func (s *Server) RevokeAccessCode(c fiber.Ctx, id Id) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	code, err := s.store.RevokeCode(c.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		return writeError(c, fiber.StatusNotFound, "Code not found.")
	}
	if err != nil {
		return s.internal(c, err)
	}
	return c.JSON(s.codeJSON(c, code))
}

func (s *Server) GetAccessCodeQR(c fiber.Ctx, id Id) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	code, err := s.store.GetCode(c.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		return writeError(c, fiber.StatusNotFound, "Code not found.")
	}
	if err != nil {
		return s.internal(c, err)
	}
	png, err := qrcode.Encode(s.joinURL(c, code.Code), qrcode.Medium, 256)
	if err != nil {
		return s.internal(c, err)
	}
	c.Set(fiber.HeaderContentType, "image/png")
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.Send(png)
}

func (s *Server) ListTemporaryAccounts(c fiber.Ctx, id Id) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	accounts, err := s.store.ListTemps(c.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		return writeError(c, fiber.StatusNotFound, "Code not found.")
	}
	if err != nil {
		return s.internal(c, err)
	}
	out := make([]TemporaryAccount, 0, len(accounts))
	for _, account := range accounts {
		out = append(out, TemporaryAccount{
			Id:        account.ID,
			Nickname:  account.Nickname,
			CreatedIp: account.CreatedIP,
			LastIp:    account.LastIP,
			CreatedAt: account.CreatedAt,
			RevokedAt: account.RevokedAt,
		})
	}
	return c.JSON(TemporaryAccountList{Accounts: out})
}

func (s *Server) RevokeTemporaryAccount(c fiber.Ctx, id Id) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	err := s.store.RevokeTemp(c.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		return writeError(c, fiber.StatusNotFound, "Account not found.")
	}
	if err != nil {
		return s.internal(c, err)
	}
	s.broker.Publish("temp:"+id.String(), accessEvent("session_ended", nil))
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) codeJSON(c fiber.Ctx, code store.Code) AccessCode {
	apps := code.Apps
	if apps == nil {
		apps = []string{}
	}
	return AccessCode{
		Id:          code.ID,
		Code:        code.Code,
		Label:       code.Label,
		Url:         s.joinURL(c, code.Code),
		ExpiresAt:   code.ExpiresAt,
		MaxSignups:  code.MaxSignups,
		SignupCount: code.SignupCount,
		RevokedAt:   code.RevokedAt,
		Apps:        apps,
		CreatedAt:   code.CreatedAt,
	}
}

func (s *Server) joinURL(c fiber.Ctx, code string) string {
	base := s.publicBase
	if base == "" {
		scheme := "http"
		if s.secureCookie(c) {
			scheme = "https"
		}
		base = scheme + "://" + c.Host()
	}
	return base + "/code/" + code
}
