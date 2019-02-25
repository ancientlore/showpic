package main

import (
	"image"
	"image/color"

	"github.com/disintegration/imaging"
	"github.com/gdamore/tcell"
)

type subimage interface {
	SubImage(r image.Rectangle) image.Image
}

type mapper struct {
	img    image.Image     // Original image
	width  int             // Width of terminal window
	height int             // Twice the height of terminal window
	window image.Rectangle // Windows into the original image
	scaled image.Image     // scaled image
}

func newMapper(img image.Image, width, height int) *mapper {
	m := &mapper{
		img:    img,
		width:  width,
		height: height,
		window: img.Bounds(),
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
	img := m.img
	si, ok := img.(subimage)
	if ok {
		img = si.SubImage(m.window)
	}
	m.scaled = imaging.Fit(img, m.width, m.height, imaging.Box)
}

func (m mapper) DrawTo(s tcell.Screen) {
	for r := 0; r < m.height; r += 2 {
		for c := 0; c < m.width; c++ {
			red, green, blue, _ := m.scaled.At(c, r).RGBA()
			bg := tcell.NewRGBColor(int32(red), int32(green), int32(blue))
			red, green, blue, _ = m.scaled.At(c, r+1).RGBA()
			fg := tcell.NewRGBColor(int32(red), int32(green), int32(blue))
			rn := '▄'
			if fg == bg {
				rn = ' '
			}
			s.SetCell(c, r/2, tcell.StyleDefault.Foreground(fg).Background(bg), rn)
		}
	}
}

func (m *mapper) ResetZoom() {
	m.window = m.img.Bounds()
	m.Sync()
}

func (m *mapper) ZoomIn() {
	sz := m.sz()
	r := m.window.Inset(sz)
	viewportAspect := float64(m.width) / float64(m.height)
	aspect := float64(r.Dx()) / float64(r.Dy())
	if aspect < viewportAspect {
		d := int(float64(r.Dy())*viewportAspect) - r.Dx()
		r.Min.X -= d / 2
		r.Max.X += d / 2
	} else if aspect > viewportAspect {
		d := int(float64(r.Dx())/viewportAspect) - r.Dy()
		r.Min.Y -= d / 2
		r.Max.Y += d / 2
	}
	r = r.Intersect(m.img.Bounds())
	if r.Dx() >= sz*2 && r.Dy() >= sz*2 {
		m.window = r
		m.Sync()
	}
}

func (m *mapper) ZoomOut() {
	sz := m.sz()
	r := m.window.Inset(-sz)
	viewportAspect := float64(m.width) / float64(m.height)
	aspect := float64(r.Dx()) / float64(r.Dy())
	if aspect < viewportAspect {
		d := int(float64(r.Dy())*viewportAspect) - r.Dx()
		r.Min.X -= d / 2
		r.Max.X += d / 2
	} else if aspect > viewportAspect {
		d := int(float64(r.Dx())/viewportAspect) - r.Dy()
		r.Min.Y -= d / 2
		r.Max.Y += d / 2
	}
	r = r.Intersect(m.img.Bounds())
	m.window = r
	m.Sync()
}

func (m *mapper) Left() {
	r := m.window.Sub(image.Point{X: m.sz(), Y: 0})
	if r.In(m.img.Bounds()) {
		m.window = r
		m.Sync()
	}
}

func (m *mapper) Right() {
	r := m.window.Add(image.Point{X: m.sz(), Y: 0})
	if r.In(m.img.Bounds()) {
		m.window = r
		m.Sync()
	}
}

func (m *mapper) Up() {
	r := m.window.Sub(image.Point{X: 0, Y: m.sz()})
	if r.In(m.img.Bounds()) {
		m.window = r
		m.Sync()
	}
}

func (m *mapper) Down() {
	r := m.window.Add(image.Point{X: 0, Y: m.sz()})
	if r.In(m.img.Bounds()) {
		m.window = r
		m.Sync()
	}
}

func (m mapper) sz() int {
	szx := m.img.Bounds().Dx() / m.width
	szy := m.img.Bounds().Dy() / m.height
	// the larger value is the edge that was fit
	if szx > szy {
		return szx
	}
	return szy
}
