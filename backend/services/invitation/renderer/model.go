package renderer

import (
	"errors"

	"undangan/services/invitation/domain"
)

var ErrNotFound = errors.New("invitation not found")

// Invitation = representasi minimal untuk render halaman undangan.
type Invitation struct {
	ID      string
	Theme   string // slug folder di THEMES_DIR
	Status  string
	Content domain.Content
}
