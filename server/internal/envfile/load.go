package envfile

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

// Load reads .env from the working directory into the environment.
// A variable that is already set is left as it is. A missing file is not an error.
func Load() error {
	err := godotenv.Load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
