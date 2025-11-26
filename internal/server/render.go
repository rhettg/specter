package server

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var (
	fontFace font.Face
)

func init() {
	// Try to find a Nerd Font on the system
	fontPaths := []string{
		"/usr/share/fonts/TTF/CaskaydiaMonoNerdFont-Regular.ttf",
		"/usr/share/fonts/TTF/MesloLGS NF Regular.ttf",
	}

	var fontData []byte
	var err error

	for _, path := range fontPaths {
		fontData, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if fontData == nil {
		fontData = gomono.TTF
	}

	tt, err := opentype.Parse(fontData)
	if err != nil {
		log.Fatalf("Failed to parse font: %v", err)
	}

	const size = 12
	const dpi = 72
	
	fontFace, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    size,
		DPI:     dpi,
		Hinting: font.HintingNone,
	})
	if err != nil {
		log.Fatalf("Failed to create font face: %v", err)
	}
}

func (s *Server) renderPNG(sess *Session) ([]byte, error) {
	rows, cols := sess.VTerm.Size()
	
	// Metrics
	// GoMono is a monospaced font.
	// However, opentype.Face doesn't guarantee fixed advance for all glyphs in the interface,
	// but since it is GoMono, we can measure 'M' or 'W' to get the width.
	// And height from metrics.
	
	metrics := fontFace.Metrics()
	// Fixed.Int26_6 to int (ceil)
	// lineHeight := (metrics.Height + metrics.Descent).Ceil() // A bit loose
	// Actually, terminal emulators usually have a fixed cell size.
	// Let's measure 'W'
	adv, ok := fontFace.GlyphAdvance('W')
	if !ok {
		adv = fixed.I(7) // fallback
	}
	charWidth := adv.Ceil()
	charHeight := metrics.Height.Ceil() + 2 // Add a little padding/leading

	width := cols * charWidth
	height := rows * charHeight

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Default background (black)
	draw.Draw(img, img.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Src)

	drawer := font.Drawer{
		Dst:  img,
		Src:  image.White, // Default foreground
		Face: fontFace,
	}

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			cell, err := sess.Screen.GetCellAt(r, c)
			if err != nil {
				continue
			}

			// Draw Background
			bg := cell.Bg()
			if bg != nil {
				r1, g1, b1, _ := bg.RGBA()
				if r1 != 0 || g1 != 0 || b1 != 0 {
					rect := image.Rect(c*charWidth, r*charHeight, (c+1)*charWidth, (r+1)*charHeight)
					draw.Draw(img, rect, &image.Uniform{bg}, image.Point{}, draw.Src)
				}
			}

			// Draw Character
			chars := cell.Chars()
			if len(chars) == 0 {
				continue // space or empty
			}

			// Foreground color
			fg := cell.Fg()
			if fg != nil {
				drawer.Src = &image.Uniform{fg}
			} else {
				drawer.Src = image.White
			}
			
			// Position
			// Drawer Dot is the baseline. 
			x := c * charWidth
			y := r * charHeight + metrics.Ascent.Ceil()
			
			drawer.Dot = fixed.P(x, y)
			
			// Only draw characters that exist in the font
			// Skip characters with missing glyphs to avoid rendering boxes
			str := string(chars)
			for _, ch := range str {
				if _, ok := fontFace.GlyphAdvance(ch); ok {
					drawer.DrawString(string(ch))
				}
				// If glyph not found, skip it (don't draw a box)
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
