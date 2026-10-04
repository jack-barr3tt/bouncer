package limit

import "time"

// Duration is how long a bucket stays locked after it has crossed its
// failure threshold `strikes` times. The first lock is 15 minutes. Each
// later lock doubles, up to 24 hours.
func Duration(strikes int) time.Duration {
	if strikes < 1 {
		strikes = 1
	}
	d := 15 * time.Minute
	for i := 1; i < strikes; i++ {
		if d >= 24*time.Hour {
			return 24 * time.Hour
		}
		d *= 2
	}
	if d > 24*time.Hour {
		return 24 * time.Hour
	}
	return d
}
