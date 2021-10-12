package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"github.com/gdamore/tcell"
)

func usage() {
	fmt.Fprintln(os.Stderr, "showpic display images on a terminal.\nSupported formats include gif, bmp, tiff, png, and jpg.")
	fmt.Fprintln(os.Stderr, "\nExamples:\n  showpic *.png\n  TERM=xterm-truecolor showpic *.tiff\n  showpic http://webnull.ancientlore.io/media/null.png")
	fmt.Fprintln(os.Stderr, "\nOptions:")
	flag.PrintDefaults()
}

func main() {
	var (
		flagHelp      = flag.Bool("help", false, "Show help.")
		flagGray      = flag.Bool("grayscale", false, "Show images in grayscale.")
		flagSlideShow = flag.Duration("slideshow", 0, "Show slideshow with given timeout.")
	)
	flag.Usage = usage
	flag.Parse()

	if *flagHelp || flag.NArg() == 0 {
		usage()
	}

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
		var r bool
		img, err := loadImage(flag.Arg(i))
		if err != nil {
			r = log(s, err.Error())
		} else {
			r = showImage(s, img, *flagGray, *flagSlideShow)
		}
		if r {
			break
		}
	}
}

func log(s tcell.Screen, msg string) bool {
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
	result := false
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
				case tcell.KeyRune:
					switch ev.Rune() {
					case 'q':
						result = true
						close(quit)
						return
					}
				}
			}
		}
	}()

	s.Show()
	<-quit
	return result
}

func loadImage(fn string) (image.Image, error) {
	var (
		reader io.ReadCloser
		err    error
	)
	if strings.HasPrefix(fn, "http://") || strings.HasPrefix(fn, "https://") {
		resp, err := http.Get(fn)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			reader = resp.Body
		} else {
			return nil, fmt.Errorf("HTTP Status %d: %s", resp.StatusCode, resp.Status)
		}
	} else {
		reader, err = os.Open(fn)
		if err != nil {
			return nil, err
		}
	}
	defer reader.Close()
	m, _, err := image.Decode(reader)
	return m, err
}

func showImage(s tcell.Screen, img image.Image, gray bool, timeout time.Duration) bool {
	cols, rows := s.Size()

	m := newMapper(img, cols, rows*2, gray)

	quit := make(chan struct{})
	result := false
	show := make(chan bool, 8)

	// Don't show too quickly if we are zooming or panning rapidly
	go func() {
		for {
			select {
			case <-quit:
				return
			case b := <-show:
				if b {
					t := time.NewTimer(time.Millisecond * 10)
					c := 0
				loop:
					for {
						select {
						case b2 := <-show:
							if !b2 {
								break loop
							}
							c++
						case <-t.C:
							break loop
						}
					}
					t.Stop()
					m.DrawTo(s)
					s.Show()
					// fmt.Printf("Ate %d events\n", c)
				} else {
					m.DrawTo(s)
					s.Sync()
				}
			}
		}
	}()

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
					// m.DrawTo(s)
					show <- false // s.Sync()
				case tcell.KeyUp:
					m.Up()
					// m.DrawTo(s)
					show <- true // s.Show()
				case tcell.KeyDown:
					m.Down()
					// m.DrawTo(s)
					show <- true // s.Show()
				case tcell.KeyLeft:
					m.Left()
					// m.DrawTo(s)
					show <- true // s.Show()
				case tcell.KeyRight:
					m.Right()
					// m.DrawTo(s)
					show <- true // s.Show()
				case tcell.KeyRune:
					switch ev.Rune() {
					case '-':
						m.ZoomOut()
						// m.DrawTo(s)
						show <- true // s.Show()
					case '+', 'z':
						m.ZoomIn()
						// m.DrawTo(s)
						show <- true // s.Show()
					case '0':
						m.ResetZoom()
						// m.DrawTo(s)
						show <- true // s.Show()
					case 'q':
						result = true
						close(quit)
						return
					}
				}
			case *tcell.EventInterrupt:
				close(quit)
				return
			case *tcell.EventResize:
				c, r := ev.Size()
				m.SetSize(c, r*2)
				// m.DrawTo(s)
				show <- true // s.Show()
			}
		}
	}()

	if timeout > 0 {
		go func() {
			t := time.NewTimer(timeout)
			defer t.Stop()
			select {
			case <-t.C:
				s.PostEvent(tcell.NewEventInterrupt(nil))
			case <-quit:
				break
			}
		}()
	}

	m.DrawTo(s)
	s.Show()
	<-quit
	return result
}
