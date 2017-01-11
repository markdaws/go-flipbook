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
	"strconv"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/disintegration/imaging"
	"github.com/golang/freetype"
)

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
func To4x6x3(bgColor, inputDir, outputDir, line1Text, line2Text, fontPath, identifier string,
	reversePages, reverseFrames, skipCover bool, verLog *log.Logger) error {
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
	var coverImgIndex int
	var coverImgInfo os.FileInfo

	if !skipCover {
		coverFrame := frames[0]
		coverFramePath := path.Join(inputDir, coverFrame.Name())
		coverImg, err := renderFrontCover(coverFramePath, fontPath, line1Text, line2Text)
		if err != nil {
			return fmt.Errorf("failed to generate cover image: %s", err)
		}

		coverImgOutPath := path.Join(outputDir, "cover.png")
		err = imaging.Save(coverImg, coverImgOutPath)
		if err != nil {
			return fmt.Errorf("failed to save cover image: %s", err)
		}

		coverImgInfo, err = os.Stat(coverImgOutPath)
		if err != nil {
			return fmt.Errorf("failed to stat cover image: %s", err)
		}
	}

	// Don't want to waste any paper, so we have some multiple of framesPerSheet so that
	// all sheets are completely full, at most we lose two end frames
	frames = frames[:framesPerSheet*(len(frames)/framesPerSheet)]
	nFrames := len(frames)
	nOutput := nFrames / framesPerSheet
	var compImg *image.RGBA

	if !skipCover {
		if reverseFrames {
			coverImgIndex = len(frames) - 1
		} else {
			coverImgIndex = 0
		}
		frames[coverImgIndex] = coverImgInfo
	}

	verLog.Println("reading input frames from:", inputDir)
	verLog.Println(nFrames, " found for processing")
	verLog.Println("composite images ", compWidth, "x", compHeight)

	for i := 0; i < nOutput; i++ {
		compImg = image.NewRGBA(image.Rectangle{
			Min: image.Point{X: 0, Y: 0},
			Max: image.Point{X: compWidth, Y: compHeight},
		})

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

			// Flip the book from back to front vs front to back
			if reverseFrames {
				frameIndex = nFrames - frameIndex - 1
			}

			frame := frames[frameIndex]
			framePath := path.Join(inputDir, frame.Name())

			err := compFrame(framePath, j, compWidth, compHeight, framesPerSheet, compImg, verLog)
			if err != nil {
				return err
			}

			// Don't want to put info on the cover image
			fmt.Println(i, ":", j, ":", frameIndex)
			if frameIndex != coverImgIndex {
				y := compHeight / framesPerSheet
				addLabel(compImg, 20, y*j+int(float64(y)*0.5), strconv.Itoa(frameIndex))
				if identifier != "" {
					addLabel(compImg, 20, y*j+int(float64(y)*0.5)+20, identifier)
				}
			}
		}

		// When you print pictures, maybe the service orders them by filename e.g. comp001, comp002 etc so the last
		// frames are printed on the top of the stack so you have to reverse them for assembly, this flag flips the
		// numbering so that you don't need to do this after printing
		var compIndex int
		if reversePages {
			compIndex = nOutput - i - 1
		} else {
			compIndex = i
		}
		err = writeJPG(compImg, outputDir, identifier, compIndex, verLog)
		if err != nil {
			return err
		}
	}

	return nil
}

func renderFrontCover(framePath, fontPath, labelLine1, labelLine2 string) (image.Image, error) {
	src, err := imaging.Open(framePath)
	if err != nil {
		return nil, err
	}

	dst := imaging.Blur(src, 12.5)

	drawBar := false
	barWidth := 0
	if drawBar {
		barWidth = src.Bounds().Dx() / 7
		draw.Draw(dst, image.Rectangle{
			Min: image.Point{X: 0, Y: 0},
			Max: image.Point{X: barWidth, Y: src.Bounds().Dy()},
		}, image.Black, image.ZP, draw.Src)
	}

	c := freetype.NewContext()
	fontBytes, err := ioutil.ReadFile("/Users/mark/code/go/src/github.com/markdaws/go-flipbook/HelveticaNeue.ttf")
	if err != nil {
		return nil, fmt.Errorf("failed to read the font file: %s, %s", fontPath, err)
	}

	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font file: %s, %s", fontPath, err)
	}

	c.SetDPI(70)
	c.SetFont(f)
	c.SetClip(dst.Bounds())
	c.SetDst(dst)
	c.SetHinting(font.HintingFull)

	var renderText = func(x, y int, size float64, fg *image.Uniform, label string) error {
		c.SetFontSize(size)
		c.SetSrc(fg)

		pt := freetype.Pt(x, y+int(c.PointToFixed(size)>>6))
		_, err := c.DrawString(label, pt)
		return err
	}

	line1FontSize := 80.0
	line2FontSize := 45.0
	line2YOffset := 100
	x := barWidth + 40
	y := 30

	if err = renderText(x, y, line1FontSize, image.Black, labelLine1); err != nil {
		return nil, err
	}
	if err = renderText(x-2, y-2, line1FontSize, image.White, labelLine1); err != nil {
		return nil, err
	}
	if err = renderText(x, y+line2YOffset, line2FontSize, image.Black, labelLine2); err != nil {
		return nil, err
	}
	if err = renderText(x-2, y-2+line2YOffset, line2FontSize, image.White, labelLine2); err != nil {
		return nil, err
	}
	return dst, nil
}

func compFrame(
	inputImgPath string, compIndex, compWidth, compHeight, framesPerSheet int,
	compImg *image.RGBA, verLog *log.Logger) error {
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

func writeJPG(compImg *image.RGBA, outputDir, identifier string, imgIndex int, verLog *log.Logger) error {
	toImgPath := path.Join(outputDir, fmt.Sprintf("comp-%s-%03d.jpg", identifier, imgIndex))
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

func addLabel(img *image.RGBA, x, y int, label string) {
	col := color.RGBA{200, 100, 0, 255}
	point := fixed.Point26_6{fixed.Int26_6(x * 64), fixed.Int26_6(y * 64)}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
}
