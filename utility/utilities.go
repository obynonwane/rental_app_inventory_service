package utility

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/oklog/ulid/v2"

	"time"
)

func GenerateSlug(name string) (string, string) {
	// Normalize name (remove special chars, spaces to dashes, lowercase)
	base := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	// Generate a ULID
	t := time.Now().UTC()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	id := ulid.MustNew(ulid.Timestamp(t), entropy)

	return fmt.Sprintf("%s-%s", base, id.String()), id.String()
}

func TesxtToLower(desc string) string {
	return strings.ToLower(desc)
}
