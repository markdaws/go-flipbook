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
// get 4x2 in dimension
func To4x6x3(bgColor, inputDir, outputDir string, verLog *log.Logger) error {
	if verLog == nil {
		verLog = log.New(ioutil.Discard, "", 0)
	}

	images, err := ioutil.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("failed to read input images: %s", err)
	}

	verLog.Println("reading input frames from:", inputDir)
	verLog.Println(len(images), " found for processing")

	const compWidth = 1200
	const compHeight = 1800
	const imagesPerSheet = 3

	verLog.Println("composite images ", compWidth, "x", compHeight)

	var compImg *image.RGBA

	compIndex := -1
	for i, img := range images {

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
		targetHeight := compHeight / imagesPerSheet

		minY := (i % imagesPerSheet) * targetHeight
		maxX := int(float64(targetHeight) / float64(sourceHeight) * float64(sourceWidth))
		xOffset := compWidth - maxX

		dstRect := image.Rectangle{
			Min: image.Point{X: xOffset, Y: minY},
			Max: image.Point{X: xOffset + maxX, Y: minY + int(targetHeight)},
		}

		verLog.Println("target rectangle:", dstRect)

		if i%imagesPerSheet == 0 {
			if compImg != nil {
				err = writeJPG(compImg, outputDir, compIndex, verLog)
				if err != nil {
					return err
				}
			}

			compIndex++
			compImg = image.NewRGBA(image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: compWidth, Y: compHeight},
			})

			if bgColor == "white" {
				draw.Draw(compImg, compImg.Bounds(), &image.Uniform{color.RGBA{255, 255, 255, 255}}, image.ZP, draw.Src)
			} else {
				draw.Draw(compImg, compImg.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 255}}, image.ZP, draw.Src)
			}
		}

		draw.BiLinear.Scale(compImg, dstRect, srcImg, srcImg.Bounds(), draw.Src, nil)

		// Last image, write remaining
		if i == len(images)-1 {
			err = writeJPG(compImg, outputDir, compIndex, verLog)
			if err != nil {
				return err
			}
		}
	}

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
