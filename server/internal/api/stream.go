package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jack-barr3tt/bouncer/internal/access"
)

func (s *Server) AccessStream(c fiber.Ctx) error {
	principal, err := s.require(c)
	if err != nil {
		return err
	}
	if principal.TempID != nil {
		_ = s.store.TouchTempIP(c.Context(), *principal.TempID, c.IP())
	}
	events, cancel := s.broker.Subscribe(principal.Key)
	apps := append([]string(nil), principal.Apps...)
	expires := principal.ExpiresAt

	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set("X-Accel-Buffering", "no")
	return c.SendStreamWriter(func(w *bufio.Writer) {
		defer cancel()
		if err := writeEvent(w, "access", map[string]any{"apps": apps}); err != nil {
			return
		}
		heartbeat := time.NewTicker(20 * time.Second)
		defer heartbeat.Stop()
		deadline := time.NewTimer(time.Until(expires))
		defer deadline.Stop()
		for {
			select {
			case ev := <-events:
				payload := map[string]any{}
				if ev.Name == "access" {
					if ev.Apps == nil {
						ev.Apps = []string{}
					}
					payload["apps"] = ev.Apps
				}
				if err := writeEvent(w, ev.Name, payload); err != nil {
					return
				}
				if ev.Name == "session_ended" {
					return
				}
			case <-heartbeat.C:
				if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			case <-deadline.C:
				_ = writeEvent(w, "session_ended", map[string]any{})
				return
			}
		}
	})
}

func accessEvent(name string, apps []string) access.Event {
	if apps == nil {
		apps = []string{}
	}
	return access.Event{Name: name, Apps: apps}
}

func writeEvent(w *bufio.Writer, name string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, body); err != nil {
		return err
	}
	return w.Flush()
}
