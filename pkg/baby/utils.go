package baby

import (
	"fmt"
	"regexp"

	"github.com/rs/zerolog/log"
)

var validUID = regexp.MustCompile(`^[a-z0-9_-]+$`)

// EnsureValidBabyUID - Checks that Baby UID does not contain any bad characters
// This is necessary because we use it as part of file paths
func EnsureValidBabyUID(babyUID string) error {
	if !validUID.MatchString(babyUID) {
		log.Error().Str("uid", babyUID).Msg("Baby UID contains unsafe characters")
		return fmt.Errorf("invalid baby UID '%s': contains unsafe characters (only lowercase letters, numbers, underscore, and hyphen allowed)", babyUID)
	}
	return nil
}
