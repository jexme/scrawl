package settings

import (
	"errors"
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/jexme/scrawl/internal/i18n"
)

// Glue holds the "before/after" collage settings.
type Glue struct {
	// Captions above the images. Empty means the default words.
	Before string `json:"before,omitempty"` // the first of the two images
	After  string `json:"after,omitempty"`  // the second of the two images
	Step   string `json:"step,omitempty"`   // the word before the number when there are more than two images

	// Layout is the arrangement of the images: side by side or one under the other.
	Layout GlueLayout `json:"layout"`

	// The look of the collage: background, margin from the edges and between images, caption
	// size (pixels of the result).
	Background GlueBackground `json:"background"`
	Pad        int            `json:"pad"`
	LabelSize  int            `json:"label_size"`
}

// GlueBackground is the collage background colour.
type GlueBackground string

const (
	GlueLight GlueBackground = "light" // white (default)
	GlueDark  GlueBackground = "dark"  // dark
)

// Bounds of the collage margin and caption size.
const (
	MaxGluePad       = 100
	MinGlueLabelSize = 12
	MaxGlueLabelSize = 64
)

// GlueLayout is the arrangement of images in the collage.
type GlueLayout string

const (
	GlueSide  GlueLayout = "side"  // in a row, left to right (default)
	GlueStack GlueLayout = "stack" // in a column, top to bottom
)

const maxGlueLabel = 30

// Default collage caption words.
const (
	DefaultGlueBefore = "Before"
	DefaultGlueAfter  = "After"
	DefaultGlueStep   = "Step"
)

// ValidateGlueLabel checks a caption: empty (the default) or short text
// without control characters.
func ValidateGlueLabel(s string) error {
	if utf8.RuneCountInString(s) > maxGlueLabel {
		return fmt.Errorf(i18n.T("at most %d characters"), maxGlueLabel)
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return errors.New(i18n.T("control characters are not allowed"))
		}
	}
	return nil
}

// DefaultGlue returns the default collage settings.
func DefaultGlue() Glue {
	return Glue{Layout: GlueSide, Background: GlueLight, Pad: 28, LabelSize: 28}
}

func (g *Glue) normalize() {
	if g.Layout != GlueSide && g.Layout != GlueStack {
		g.Layout = DefaultGlue().Layout
	}
	if g.Background != GlueLight && g.Background != GlueDark {
		g.Background = DefaultGlue().Background
	}
	g.Pad = min(max(g.Pad, 0), MaxGluePad)
	g.LabelSize = min(max(g.LabelSize, MinGlueLabelSize), MaxGlueLabelSize)
	for _, p := range []*string{&g.Before, &g.After, &g.Step} {
		if ValidateGlueLabel(*p) != nil {
			*p = ""
		}
	}
}

// Labels returns captions for n images: "Before" and "After" for two, "Step N"
// for the rest (words from the settings or the defaults).
func (g Glue) Labels(n int) []string {
	pick := func(custom, def string) string {
		if custom == "" {
			return def
		}
		return custom
	}
	if n == 2 {
		return []string{pick(g.Before, i18n.T(DefaultGlueBefore)), pick(g.After, i18n.T(DefaultGlueAfter))}
	}
	step := pick(g.Step, i18n.T(DefaultGlueStep))
	labels := make([]string, n)
	for i := range labels {
		labels[i] = fmt.Sprintf("%s %d", step, i+1)
	}
	return labels
}
