package api

import (
	"errors"

	"github.com/jack-barr3tt/bouncer/internal/store"
	"github.com/gofiber/fiber/v3"
)

func (s *Server) ListUsers(c fiber.Ctx) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	users, err := s.store.ListUsers(c.Context())
	if err != nil {
		return s.internal(c, err)
	}
	out := make([]User, 0, len(users))
	for _, user := range users {
		out = append(out, userJSON(user))
	}
	return c.JSON(UserList{Users: out})
}

func (s *Server) CreateUser(c fiber.Ctx) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	var req CreateUserRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request.")
	}
	username, ok := validUsername(req.Username)
	if !ok || !validPassword(req.Password) {
		return writeError(c, fiber.StatusBadRequest, "Username or password is not valid.")
	}
	user, err := s.store.CreateUser(c.Context(), username, req.Password)
	if errors.Is(err, store.ErrUsernameTaken) {
		return writeError(c, fiber.StatusConflict, "That username is already taken.")
	}
	if err != nil {
		return s.internal(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(userJSON(user))
}

func (s *Server) UpdateUser(c fiber.Ctx, id Id) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	var req UpdateUserRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request.")
	}
	if req.Password != nil && !validPassword(*req.Password) {
		return writeError(c, fiber.StatusBadRequest, "Password must be at least 8 characters.")
	}
	var role *string
	if req.Role != nil {
		if !req.Role.Valid() {
			return writeError(c, fiber.StatusBadRequest, "Role is not valid.")
		}
		value := string(*req.Role)
		role = &value
	}
	user, err := s.store.UpdateUser(c.Context(), id, req.Password, req.Disabled, role)
	if errors.Is(err, store.ErrNotFound) {
		return writeError(c, fiber.StatusNotFound, "User not found.")
	}
	if errors.Is(err, store.ErrLastAdmin) {
		return writeError(c, fiber.StatusBadRequest, "The last admin cannot be disabled or demoted.")
	}
	if err != nil {
		return s.internal(c, err)
	}
	if user.Disabled {
		s.broker.Publish("user:"+user.ID.String(), accessEvent("session_ended", nil))
	} else if req.Role != nil {
		apps := user.Apps
		if user.Role == "admin" {
			apps, err = s.site.Slugs()
			if err != nil {
				return s.internal(c, err)
			}
		}
		s.broker.Publish("user:"+user.ID.String(), accessEvent("access", apps))
	}
	return c.JSON(userJSON(user))
}

func (s *Server) SetUserApps(c fiber.Ctx, id Id) error {
	if _, err := s.requireAdmin(c); err != nil {
		return err
	}
	var req AppListRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeError(c, fiber.StatusBadRequest, "Invalid request.")
	}
	slugs, err := s.knownSlugs(req.Slugs)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "Unknown app.")
	}
	user, err := s.store.SetUserApps(c.Context(), id, slugs)
	if errors.Is(err, store.ErrNotFound) {
		return writeError(c, fiber.StatusNotFound, "User not found.")
	}
	if err != nil {
		return s.internal(c, err)
	}
	apps := user.Apps
	if user.Role == "admin" {
		apps, err = s.site.Slugs()
		if err != nil {
			return s.internal(c, err)
		}
	}
	s.broker.Publish("user:"+user.ID.String(), accessEvent("access", apps))
	return c.JSON(userJSON(user))
}
