package limit_test

import (
	"testing"
	"time"

	"github.com/jack-barr3tt/bouncer/internal/limit"
)

func TestDuration(t *testing.T) {
	if limit.Duration(1) != 15*time.Minute {
		t.Fatalf("first lock: %s", limit.Duration(1))
	}
	if limit.Duration(2) != 30*time.Minute {
		t.Fatalf("second lock: %s", limit.Duration(2))
	}
	if limit.Duration(20) != 24*time.Hour {
		t.Fatalf("cap: %s", limit.Duration(20))
	}
}
