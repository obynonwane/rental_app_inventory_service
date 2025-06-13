package utility

import (
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// GenerateSlug creates a URL-friendly slug from the given name, removing special characters,
// converting spaces to dashes, normalizing case, and appending a ULID
func GenerateSlug(name string) (string, string) {
	// Lowercase and replace spaces with dashes
	base := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	// Remove all characters except lowercase letters, numbers, and dashes
	re := regexp.MustCompile(`[^a-z0-9-]+`)
	base = re.ReplaceAllString(base, "")

	// Generate a ULID for uniqueness
	now := time.Now().UTC()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(now.UnixNano())), 0)
	id := ulid.MustNew(ulid.Timestamp(now), entropy)

	slug := fmt.Sprintf("%s-%s", base, id.String())
	return slug, id.String()
}

// TextToLower normalizes a description by converting it to lowercase
func TextToLower(desc string) string {
	// Remove leading/trailing whitespace
	d := strings.TrimSpace(desc)

	// Convert to lowercase
	return strings.ToLower(d)
}

func ValidateBookingDates(startDate, endDate time.Time) error {
	today := time.Now().Truncate(24 * time.Hour)

	// Check if start date is in the past
	if startDate.Before(today) {
		return errors.New("start date cannot be in the past")
	}

	// Check if end date is before start date
	if endDate.Before(startDate) {
		return errors.New("end date cannot be before start date")
	}

	return nil
}
