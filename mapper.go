package main

import (
	"image"
	"image/color"

	"github.com/disintegration/imaging"
	"github.com/gdamore/tcell"
)

type mapper struct {
	img    image.Image     // Original image
	width  int             // Width of terminal window
	height int             // Twice the height of terminal window
	Window image.Rectangle // Windows into the original image
	scaled image.Image     // scaled image
}

func newMapper(img image.Image, width, height int) *mapper {
	m := &mapper{
		img:    img,
		width:  width,
		height: height,
		Window: img.Bounds(),
	}
	m.Sync()
	return m
}

func (m mapper) ColorModel() color.Model {
	return m.img.ColorModel()
}

func (m mapper) Bounds() image.Rectangle {
	return image.Rect(0, 0, m.width, m.height)
}

func (m mapper) At(x, y int) color.Color {
	if y*m.width+x >= m.width*m.height {
		return color.RGBA{}
	}
	return m.scaled.At(x, y)
}

func (m *mapper) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.Sync()
}

func (m *mapper) Sync() {
	m.scaled = imaging.Fit(m.img, m.width, m.height, imaging.Box)
}

func (m mapper) DrawTo(s tcell.Screen) {
	for r := 0; r < m.height; r += 2 {
		for c := 0; c < m.width; c++ {
			red, green, blue, _ := m.scaled.At(c, r).RGBA()
			bg := tcell.NewRGBColor(int32(red), int32(green), int32(blue))
			red, green, blue, _ = m.scaled.At(c, r+1).RGBA()
			fg := tcell.NewRGBColor(int32(red), int32(green), int32(blue))
			s.SetCell(c, r/2, tcell.StyleDefault.Foreground(fg).Background(bg), '▄')
		}
	}
}
