package main

import (
	"context"
	"image"
	"image/color"
	"log"
	"math"
	"math/cmplx"
	"time"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"gopkg.in/tomb.v2"
)

var (
	fromX, toX, fromY, toY = -2.5, 1.0, -2.0, 2.0
	maxWidth, maxHeight    = 1750, 2000
	maxIter                = 60
	maxRoutines            = 30

	allPixels = maxWidth * maxHeight
)

type position struct {
	X, Y int
}

type pixelResult struct {
	position
	InSet bool
	Value float64
	Iters int
}

func main() {
	pixelgl.Run(run)
}

func (r pixelResult) getColor() color.Color {
	if r.InSet {
		return color.Black
	}

	hue := 0.3 - (float64(r.Iters) / 800 * r.Value)
	return hslToRGB(hue, 1, 0.5)
}

func run() {
	img := image.NewRGBA(image.Rect(0, 0, maxWidth, maxHeight))
	cfg := pixelgl.WindowConfig{
		Title:  "Mandelbrot Set",
		Bounds: pixel.R(0, 0, float64(maxWidth), float64(maxHeight)),
		VSync:  true,
	}

	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	inputs := make(chan position, allPixels)
	results := make(chan pixelResult, allPixels)

	drawers, _ := tomb.WithContext(ctx)
	drawers.Go(func() error {
		for result := range results {
			img.Set(result.X, result.Y, result.getColor())
		}

		return nil
	})

	workers, _ := tomb.WithContext(ctx)
	for i := 0; i < maxRoutines; i++ {
		workers.Go(func() error {
			for pos := range inputs {
				absX := math.Abs(fromX) + math.Abs(toX)
				absY := math.Abs(fromY) + math.Abs(toY)

				a := fromX + (float64(pos.X-1) * (absX / float64(maxWidth)))
				b := fromY + (float64(pos.Y-1) * (absY / float64(maxHeight)))

				inSet, iters, val := mandelbrot(complex(a, b))

				results <- pixelResult{
					pos,
					inSet, val, iters,
				}
			}

			return nil
		})
	}

	start := time.Now()
	for x := 1; x < maxWidth; x++ {
		for y := 1; y < maxHeight; y++ {
			inputs <- position{x, y}
		}
	}
	close(inputs)

	if err := workers.Wait(); err != nil {
		panic(err)
	}
	close(results)

	if err := drawers.Wait(); err != nil {
		panic(err)
	}
	stop := time.Now()

	log.Printf("took: %v", stop.Sub(start))

	for !win.Closed() && !win.Pressed(pixelgl.KeyQ) {
		pic := pixel.PictureDataFromImage(img)
		sprite := pixel.NewSprite(pic, pic.Bounds())
		sprite.Draw(win, pixel.IM.Moved(win.Bounds().Center()))
		win.Update()
	}
}

func mandelbrot(p complex128) (inSet bool, iters int, val float64) {
	acc := complex128(0)
	for i := 0; i < maxIter; i++ {
		if abs := cmplx.Abs(acc); abs > (2 * 2) {
			return false, i, abs
		}

		acc = cmplx.Pow(acc, 2) + p
	}

	return true, maxIter, cmplx.Abs(acc)
}

// borrowed from https://github.com/GiselaMD/parallel-mandelbrot-go
func hslToRGB(h, s, l float64) color.RGBA {
	var r, g, b float64
	if s == 0 {
		r, g, b = l, l, l
	} else {
		var q, p float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p = 2*l - q
		r = hueToRGB(p, q, h+1.0/3.0)
		g = hueToRGB(p, q, h)
		b = hueToRGB(p, q, h-1.0/3.0)
	}
	return color.RGBA{R: uint8(r * 255), G: uint8(g * 255), B: uint8(b * 255), A: 255}
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	switch {
	case t < 1.0/6.0:
		return p + (q-p)*6*t
	case t < 1.0/2.0:
		return q
	case t < 2.0/3.0:
		return p + (q-p)*(2.0/3.0-t)*6
	default:
		return p
	}
}
