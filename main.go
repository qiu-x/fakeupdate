package main

import (
	"fmt"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
	"os"
	"strconv"
	"math"
)

const PI = 3.1415926

var winTitle string = "Windows Update"
var winWidth, winHeight int32 = 800, 600
var w, h int32
var window *sdl.Window
var renderer *sdl.Renderer
var fps uint32 = 1000/60

type TextObj struct {
	texture *sdl.Texture
	surface *sdl.Surface
	x, y int32
}

func newTextObj(renderer *sdl.Renderer, text string, font *ttf.Font) *TextObj {
	t := new(TextObj)
	var err error
	if t.surface, err = font.RenderUTF8Blended(text, sdl.Color{255, 255, 255, 255},); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to render text: %s\n", err)
	}
	t.texture, _ = renderer.CreateTextureFromSurface(t.surface);
	return t
}

func (o TextObj) draw(renderer *sdl.Renderer) {
	rect = sdl.Rect{w/2-o.surface.W/2+o.x, h/2-o.surface.H/2+o.y,o.surface.W, o.surface.H}
	renderer.Copy(o.texture, &sdl.Rect{0, 0, o.surface.W, o.surface.H}, &rect); 
}

var font *ttf.Font
var rect sdl.Rect
var dotpos sdl.Rect
var dots [5]*TextObj
var messageTop *TextObj
var messageBottom *TextObj

func loop() {
	var count [5]float64
	speed := float64(0)
	distance := float64(0.55)
	messageBottom.y += 2*h/20
	progress := int(0)
	for {
		progress = int(math.Abs(float64(count[0]/25)))
		progressString := strconv.Itoa(progress)
		messageBottom = newTextObj(renderer, progressString + "% complete", font)
		messageBottom.y += h/30
		renderer.Clear()
		renderer.SetDrawColor(0, 0x5a, 0x9e, 0xff)
		messageBottom.draw(renderer)		
		messageTop.draw(renderer)		
		for i, _ := range dots {
			dots[i].draw(renderer)
			if math.Cos(count[i]+(float64(distance)*float64(i))) > -0.6 {
				speed = 0.06
			} else {
				speed = 0.025
			}
			count[i]-=speed
			dots[i].x = int32(float64(h/27) * math.Sin(count[i]+(float64(distance)*float64(i))))
			dots[i].y = int32(float64(h/27) * math.Cos(count[i]+(float64(distance)*float64(i)))) - h/15
		}
		renderer.Present()

		sdl.Delay(fps)
	}
}

func run() int {
	window, err := sdl.CreateWindow(winTitle, sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		winWidth, winHeight, sdl.WINDOW_SHOWN|sdl.WINDOW_FULLSCREEN_DESKTOP)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create window: %s\n", err)
		return 1
	}
	w, h = window.GetSize()
	defer window.Destroy()

	renderer, err = sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create renderer: %s\n", err)
		return 1
	}
	defer renderer.Destroy()

	if err := ttf.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize TTF: %s\n", err)
		return 1
	}

	if font, err = ttf.OpenFont("./noto.ttf", int(h/45)); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open font: %s\n", err)
		return 1
	}
	defer font.Close()
	font.SetKerning(false)

	var bigfont *ttf.Font
	if bigfont, err = ttf.OpenFont("./noto.ttf", int(h/25)); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open font: %s\n", err)
		return 1
	}
	bigfont.SetKerning(false)
	defer bigfont.Close()

	messageTop = newTextObj(renderer, "Working on updates", font)
	messageBottom = newTextObj(renderer, "0% complete.", font)

	for i, _ := range dots {
		dots[i] = newTextObj(renderer, "•", bigfont)
	}

	loop()
	return 0
}

func main() {
	os.Exit(run())
}
