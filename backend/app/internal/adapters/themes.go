package adapters

import (
	"undangan/services/invitation/renderer"
	themedomain "undangan/services/theme/domain"
)

// ThemeCatalog memenuhi theme/domain.Catalog dengan membaca folder tema lewat renderer (service invitation).
type ThemeCatalog struct{ Dir string }

var _ themedomain.Catalog = ThemeCatalog{}

func (c ThemeCatalog) Scan() ([]themedomain.Meta, error) {
	metas, err := renderer.ScanThemes(c.Dir)
	if err != nil {
		return nil, err
	}
	out := make([]themedomain.Meta, 0, len(metas))
	for _, m := range metas {
		out = append(out, themedomain.Meta{
			Slug: m.Slug, Name: m.Name, Description: m.Description, Tags: m.Tags, Colors: m.Colors, Fonts: m.Fonts,
			Category: m.Category, Thumbnail: m.Thumbnail,
		})
	}
	return out, nil
}
