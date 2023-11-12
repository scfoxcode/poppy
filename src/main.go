package main

import (
    "sync"
	"image"
	"image/color"
	"image/png"
	"os"
)

func interpolate(min float64, max float64, oldRange float64, value int) float64 {
    newRange := max - min
    ratio := newRange / oldRange
    return min + ratio * float64(value)
}

func colorPetal(iter int) color.RGBA {
    return color.RGBA{
        uint8(iter * 4),
        uint8(iter / 5),
        uint8(iter / 5),
        255}
}

func colorStem(iter int) color.RGBA {
    return color.RGBA{
        uint8(iter / 8),
        uint8(iter / 2),
        uint8(iter / 8),
        255}
}

// Note: A tone mapping approach would be better here
// We are losing information without one
func addColors(a *color.RGBA, b *color.RGBA) color.RGBA {
    var col color.RGBA
    col.R = min(a.R + b.R, 255)
    col.G = min(a.G + b.G, 255)
    col.B = min(a.B + b.B, 255)
    col.A = min(a.A + b.A, 255)
    return col
}

func filterReduceGreenOnRed(col *color.RGBA) {
    col.G = uint8(float64(col.G) * (1.0 - float64(col.R) / 255.0))
}

type DrawData struct {
    X int 
    Y int
    XOffset int
    YOffset int
    Width int
    Height int
    ColorFunc func(int) color.RGBA
    Cx float64 // complex number real part
    Cy float64 // complex number imaginary part
    Escape float64
}

func colorForPixel(data *DrawData) color.RGBA {
    escape := data.Escape 
    escape2 := escape * escape

    zx := interpolate(-escape, escape, float64(data.Width), data.X -data.XOffset)
    zy := interpolate(-escape, escape, float64(data.Height), data.Y -data.YOffset)

    zx2 := zx * zx
    zy2 := zy * zy

    iteration := 0
    max_iteration := 3000 

    for i := 0; (zx2 + zy2) < escape2 && i < max_iteration; i++ {
        xtemp := zx2 - zy2 
        zy = 2 * zx * zy + data.Cy 
        zx = xtemp + data.Cx 

        zx2 = zx * zx
        zy2 = zy * zy

        iteration = iteration + 1;
    }

    if iteration >= max_iteration {
        return color.RGBA{0, 0, 0, 255}
    } else {
        if iteration > 255 {
            iteration = 255
        }
        return data.ColorFunc(iteration)
    }
}

func drawFractal(width int, height int, image *image.RGBA, data *DrawData) {
    for x:=0; x<width; x++ {
        for y:=0; y<height; y++ {
            data.X = x
            data.Y = y
            color := colorForPixel(data)
            image.Set(x, y, color)
        }
    }

}

func main() {
    width := 1200 
    height := 2000 
	imgPetal := image.NewRGBA(image.Rect(0, 0, width, height))
	imgStem := image.NewRGBA(image.Rect(0, 0, height, width)) // will be rotated 90 degrees later

    stemData := DrawData{
        X: 0,
        Y: 0,
        XOffset: 200,
        YOffset: 0,
        Width: height, // reversed, image rotated later
        Height: width, // reversed, image rotated later 
        ColorFunc: colorStem,
        Cx: -0.75,
        Cy: 0.11,
        Escape: 2.0,
    }

    petalData := DrawData{
        X: 0,
        Y: 0,
        XOffset: 0,
        YOffset: 0,
        Width: width,
        Height: width, // intentional to maintain aspect and position at top
        ColorFunc: colorPetal,
        Cx: 0.285,
        Cy: 0.01,
        Escape: 1.2,
    }

    var wg sync.WaitGroup

    // Draw stem, height and width reversed intentionally
    wg.Add(1)
    go func() {
        defer wg.Done()
        drawFractal(height, width, imgStem, &stemData)
    }()

    // Draw petal
    wg.Add(1)
    go func() {
        defer wg.Done()
        drawFractal(width, height, imgPetal, &petalData)
    }()

    wg.Wait()

    // Rotate stem and add to petal
    for x:=0; x<width; x++ {
        for y:=0; y<height; y++ {
            col := color.RGBAModel.Convert(imgPetal.At(x, y)).(color.RGBA)
            stemCol := color.RGBAModel.Convert(imgStem.At(y, x)).(color.RGBA)
            combinedCol := addColors(&col, &stemCol)
            filterReduceGreenOnRed(&combinedCol)
            imgPetal.Set(x, y, combinedCol)
        }
    }

	// Save the image to a file
	file, err := os.Create("output/1.png")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Use the PNG format to encode and save the image
	err = png.Encode(file, imgPetal)
	if err != nil {
		panic(err)
	}
}

