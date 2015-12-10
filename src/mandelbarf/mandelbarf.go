package main

import (
  "bufio"
  "fmt"
  "image"
  "image/color"
  "image/png"
  "math/cmplx"
  "os"
)

/*
just take a point called z in the complex plane
let z1 be z^2 plus z
and z2 is z1^2 plus z
and z3 is z2^2 plus z
and if the series of z's should always stay
close to z and never trend away
that point is in the mandelbrot set
*/

// Image implementation

type img struct {
  cols, rows int
  px []color.RGBA
}

func (img) ColorModel() color.Model {
  return color.RGBAModel
}

func (i img) Bounds() image.Rectangle {
  return image.Rectangle{
    image.Point{0, 0},
    image.Point{i.cols, i.rows},
  }
}

func (i img) At(x, y int) color.Color {
  if 0 <= x && x < i.cols && 0 <= y && y < i.rows {
    return i.px[y * i.cols + x]
  }
  return color.RGBA{255, 0, 0, 255}
}

func (i img) get(x, y int) color.RGBA {
  if 0 <= x && x < i.cols && 0 <= y && y < i.rows {
    return i.px[y * i.cols + x]
  }
  return color.RGBA{255, 0, 0, 255}
}

func (i img) set(x, y int, c color.RGBA) {
  if 0 <= x && x < i.cols && 0 <= y && y < i.rows {
    i.px[y * i.cols + x] = c
  }
}

func mkImg(cols, rows int) img {
  return img{
    cols,
    rows,
    make([]color.RGBA, cols * rows),
  }
}

func downScale(in img, scale int) img {
  out := mkImg(in.cols / scale, in.rows / scale)
  samples := scale * scale
  for outRow := 0; outRow < out.rows; outRow++ {
    for outCol := 0; outCol < out.cols; outCol++ {
      outRed, outGreen, outBlue := 0, 0, 0
      for subRow := 0; subRow < scale; subRow++ {
        for subCol := 0; subCol < scale; subCol++ {
          inRow := outRow * scale + subRow
          inCol := outCol * scale + subCol
          inColor := in.get(inCol, inRow)
          outRed += int(inColor.R)
          outGreen += int(inColor.G)
          outBlue += int(inColor.B)
        }
      }
      outRed /= samples
      outGreen /= samples
      outBlue /= samples
      outColor := color.RGBA{uint8(outRed), uint8(outGreen), uint8(outBlue), 255}
      out.set(outCol, outRow, outColor)
    }
  }
  return out
}

// Map x from the range x1,y1 to x2,y2
func linear(x, x1, x2, y1, y2 float64) float64 {
  slope := (y2 - y1) / (x2 - x1)
  intercept := y1 - x1 * slope
  return x * slope + intercept
}

// Return the number of iterations (max 255) before the point gets "far away"
func mandelbrot(c complex128) int {
  z := c
  var i int
  for i = 0; i < 256; i++ {
    z = z*z + c
    if cmplx.Abs(z) > 100.0 {
      break
    }
  }
  return i
}

type rowRange struct {
  startRow, stopRow int
}

// Render chunks
func work(i img, chunks chan rowRange, flag chan int) {
  for {
    chunk, ok := <- chunks
    if !ok {
      break
    }
    for r := chunk.startRow; r < chunk.stopRow; r++ {
      y := linear(float64(r), 0.0, float64(i.rows-1), 1.0, -1.0)
      for c := 0; c < i.cols; c++ {
        x := linear(float64(c), 0.0, float64(i.cols-1), -2.0, 1.0)
        v := mandelbrot(complex(x, y))
        i.set(c, r, color.RGBA{0, uint8(v), uint8(v), 255})
      }
    }
  }
  flag <- 0 // Signal completion
}

func main() {
  scale := 6 // Supersample by scale in both dimensions
  render := mkImg(6144*scale, 4096*scale)

  workerNum := 6
  chunkNum := 100
  chunkRows := render.rows / chunkNum

  // Queue up chunks of work on a channel
  chunks := make(chan rowRange, chunkNum)
  startRow := 0;
  stopRow := chunkRows
  for i := 0; i < chunkNum-1; i++ {
    chunks <- rowRange{startRow, stopRow}
    startRow, stopRow = stopRow, stopRow + chunkRows
  }
  chunks <- rowRange{startRow, render.rows}
  close(chunks)

  // Create a channel for each worker on which to signal completion
  flags := make([]chan int, workerNum)
  for i := 0; i < workerNum; i++ {
    flags[i] = make(chan int)
  }

  // Start workers
  for i := 0; i < workerNum; i++ {
    go work(render, chunks, flags[i])
  }

  // Wait for workers to finish
  for i := 0; i < workerNum; i++ {
    <- flags[i]
  }

  renderSmall := downScale(render, scale)

  outFile, err := os.Create("out.png")
  defer outFile.Close()
  if err != nil {
    fmt.Println(err)
    return
  }

  outWriter := bufio.NewWriter(outFile)
  err = png.Encode(outWriter, renderSmall)
  if err != nil {
    fmt.Println(err)
    return
  }
  outWriter.Flush()
}
