// Package domain (theme) = katalog tema undangan.
package domain

import "context"

type Theme struct {
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Colors      []string `json:"colors"`
	Fonts       []string `json:"fonts"`
	IsActive    bool     `json:"is_active"`
	IsPremium   bool     `json:"is_premium"`
	SortOrder   int      `json:"sort_order"`
	PreviewURL  string   `json:"preview_url"`
	Category    string   `json:"category"`
	Thumbnail   string   `json:"thumbnail_url"` // kosong bila tema belum punya thumbnail
}

// Meta = metadata tema dari file theme.json.
type Meta struct {
	Slug, Name, Description string
	Tags, Colors, Fonts     []string
	Category, Thumbnail     string
}

type UpdateInput struct {
	IsActive  *bool `json:"is_active"`
	IsPremium *bool `json:"is_premium"`
	SortOrder *int  `json:"sort_order"`
}

type Repository interface {
	Upsert(ctx context.Context, m Meta, defaultSort int) error
	DeactivateMissing(ctx context.Context, presentSlugs []string) error
	List(ctx context.Context, onlyActive bool) ([]Theme, error)
	Get(ctx context.Context, slug string) (*Theme, error)
	Update(ctx context.Context, slug string, in UpdateInput) error
}

// Catalog = port pembaca folder tema (implementasi: adapter renderer).
type Catalog interface {
	Scan() ([]Meta, error)
}
