package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/gdamore/tcell"
)

func main() {
	flag.Parse()

	tcell.SetEncodingFallback(tcell.EncodingFallbackASCII)
	s, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err = s.Init(); err != nil {
		fmt.Fprintln(os.Stdout, err)
		os.Exit(1)
	}
	defer s.Fini()

	s.SetStyle(tcell.StyleDefault.
		Foreground(tcell.ColorBlack).
		Background(tcell.ColorBlack))
	s.Clear()

	for i := 0; i < flag.NArg(); i++ {
		img, err := loadImage(flag.Arg(i))
		if err != nil {
			log(s, err.Error())
		} else {
			showImage(s, img)
		}
	}
}

func log(s tcell.Screen, msg string) {
	w, _ := s.Size()
	r, c := 0, 0
	for _, ch := range msg {
		s.SetCell(c, r, tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite), ch)
		c++
		if c > w {
			c = 0
			r++
		}
	}
	quit := make(chan struct{})
	go func() {
		for {
			ev := s.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyEscape, tcell.KeyEnter:
					close(quit)
					return
				}
			}
		}
	}()

	s.Show()
	<-quit

}

func loadImage(fn string) (image.Image, error) {
	reader, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	m, _, err := image.Decode(reader)
	return m, err
}

func showImage(s tcell.Screen, img image.Image) {
	cols, rows := s.Size()

	m := newMapper(img, cols, rows*2)

	quit := make(chan struct{})
	go func() {
		for {
			ev := s.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyEscape, tcell.KeyEnter:
					close(quit)
					return
				case tcell.KeyCtrlL:
					m.Sync()
					m.DrawTo(s)
					s.Sync()
				case tcell.KeyUp:
					m.Up()
					m.DrawTo(s)
					s.Show()
				case tcell.KeyDown:
					m.Down()
					m.DrawTo(s)
					s.Show()
				case tcell.KeyLeft:
					m.Left()
					m.DrawTo(s)
					s.Show()
				case tcell.KeyRight:
					m.Right()
					m.DrawTo(s)
					s.Show()
				case tcell.KeyRune:
					switch ev.Rune() {
					case '-':
						m.ZoomOut()
						m.DrawTo(s)
						s.Show()
					case '+', 'z':
						m.ZoomIn()
						m.DrawTo(s)
						s.Show()
					case '0':
						m.ResetZoom()
						m.DrawTo(s)
						s.Show()
					}
				}
			case *tcell.EventResize:
				c, r := ev.Size()
				m.SetSize(c, r*2)
				m.DrawTo(s)
				s.Show()
			}
		}
	}()

	m.DrawTo(s)
	s.Show()
	<-quit
}
