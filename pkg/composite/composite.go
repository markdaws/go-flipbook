package composite

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"path"

	"golang.org/x/image/draw"
)

//TODO: Need to write a number on each image and a job number

// To4x6x3 composites the source images to a 6x4 format, with 3 frames per image. Each frame will
// get 4x2 in dimension within the 6x4 image.
//
// Note that the composite images are printed with maximum assembly efficiency
// in mind, so the images are interlaced so that you can stack all the composite images and make just
// two cuts to then assemble the final flip book.
//
// So for example given input frames:
// 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13
// we first discard frames 12 + 13 since we want an even multiple of 3 (3 frames per sheet)
// then we print in the following format
//
//   a   b   c    d
// | 0 | 1 | 2  | 3  |
// | 4 | 5 | 6  | 7  |
// | 8 | 9 | 10 | 11 |
//
// where 0,4,8 are printed on one page, 1,5,9 on another etc. This way you can simply
// stack sheets a,b,c,d on top of one another, make two cuts and then put the stack together
// to assemble your flip book.
func To4x6x3(bgColor, inputDir, outputDir string, verLog *log.Logger) error {
	if verLog == nil {
		verLog = log.New(ioutil.Discard, "", 0)
	}

	frames, err := ioutil.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("failed to read input images: %s", err)
	}

	const compWidth = 1200
	const compHeight = 1800
	const framesPerSheet = 3

	// Don't want to waste any paper, so we have some multiple of framesPerSheet so that
	// all sheets are completely full, at most we lose two end frames
	frames = frames[:framesPerSheet*len(frames)/framesPerSheet]

	nFrames := len(frames)
	nOutput := nFrames / framesPerSheet
	var compImg *image.RGBA
	compIndex := 0

	verLog.Println("reading input frames from:", inputDir)
	verLog.Println(nFrames, " found for processing")
	verLog.Println("composite images ", compWidth, "x", compHeight)

	for i := 0; i < nOutput; i++ {
		compImg = image.NewRGBA(image.Rectangle{
			Min: image.Point{X: 0, Y: 0},
			Max: image.Point{X: compWidth, Y: compHeight},
		})
		compIndex++

		if bgColor == "white" {
			draw.Draw(compImg, compImg.Bounds(), &image.Uniform{color.RGBA{255, 255, 255, 255}}, image.ZP, draw.Src)
		} else {
			draw.Draw(compImg, compImg.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 255}}, image.ZP, draw.Src)
		}

		for j := 0; j < framesPerSheet; j++ {
			frameIndex := i + j*nOutput

			if frameIndex >= nFrames {
				break
			}

			err := compFrame(inputDir, frames, frameIndex, j, compWidth, compHeight, framesPerSheet, compImg, verLog)
			if err != nil {
				return err
			}
		}

		err = writeJPG(compImg, outputDir, compIndex, verLog)
		if err != nil {
			return err
		}
	}

	return nil
}

func compFrame(
	inputDir string, frames []os.FileInfo, frameIndex, compIndex, compWidth, compHeight, framesPerSheet int,
	compImg *image.RGBA, verLog *log.Logger) error {
	img := frames[frameIndex]
	inputImgPath := path.Join(inputDir, img.Name())
	verLog.Println("reading:", inputImgPath)

	srcReader, err := os.Open(inputImgPath)
	if err != nil {
		return fmt.Errorf("failed to read input image: %s, %s", inputImgPath, err)
	}

	verLog.Println("decoding:", inputImgPath)

	srcImg, err := png.Decode(srcReader)
	srcReader.Close()
	if err != nil {
		return fmt.Errorf("failed to decode image on load: %s, %s", inputImgPath, err)
	}

	verLog.Println("bounds:", srcImg.Bounds())

	sourceWidth := srcImg.Bounds().Dx()
	sourceHeight := srcImg.Bounds().Dy()
	targetHeight := compHeight / framesPerSheet

	minY := compIndex * targetHeight
	maxX := int(float64(targetHeight) / float64(sourceHeight) * float64(sourceWidth))
	xOffset := compWidth - maxX

	dstRect := image.Rectangle{
		Min: image.Point{X: xOffset, Y: minY},
		Max: image.Point{X: xOffset + maxX, Y: minY + int(targetHeight)},
	}

	verLog.Println("target rectangle:", dstRect)

	draw.BiLinear.Scale(compImg, dstRect, srcImg, srcImg.Bounds(), draw.Src, nil)
	return nil
}

func writeJPG(compImg *image.RGBA, outputDir string, imgIndex int, verLog *log.Logger) error {
	toImgPath := path.Join(outputDir, fmt.Sprintf("comp%03d.jpg", imgIndex))
	toImg, err := os.Create(toImgPath)
	if err != nil {
		return fmt.Errorf("failed to create image: %s, %s", toImgPath, err)
	}

	verLog.Println("writing:", toImgPath)

	err = jpeg.Encode(toImg, compImg, &jpeg.Options{Quality: 95})
	toImg.Close()
	if err != nil {
		return fmt.Errorf("failed to save img: %s, %s", toImgPath, err)
	}

	verLog.Println("written file:", toImgPath)
	return nil
}
