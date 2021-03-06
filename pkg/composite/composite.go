package composite

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path"
	"strconv"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/disintegration/imaging"
	"github.com/golang/freetype"
	"github.com/markdaws/go-effects/pkg/effects"
)

// Page defines all of the parameters of a single page, that can hold one
// or more frames
type Page struct {
	// Width of the page in inches
	Width float32

	// Height of the page in inches
	Height float32

	// MarginTop size of top margin in inches
	MarginTop float32

	// MarginRight size of right margin in inches
	MarginRight float32

	// MarginBottom size of bottom margin in inches
	MarginBottom float32

	// MarginLeft size of left margin in inches
	MarginLeft float32

	// DPI number of dots per inch e.g. 300
	DPI int
}

// Options allows callers to define all of the composition options
type Options struct {
	// Page reference to a page object, that contains dimensions and margins
	Page Page

	// Rows the number of rows of frames to composite on one page
	Rows int

	// Cols the number of columns of frames to composite on one page
	Cols int

	// BGColor the background color to use for parts of the page not covered by a frame, black|white
	BGColor string

	// InputDir the directory containing all of the individual frames, it is assumed nothing
	// but frame images are in this directory
	InputDir string

	// OutputDir the directory where the final composite images will be written to
	OutputDir string

	// Line1Text the text to show in the cover page, the first line of the title
	Line1Text string

	// Line2Text the text to show in the cover page, the second line of the title
	Line2Text string

	// Identifier a string printed in the margin of each page to help identify the frames
	Identifier string

	// FontBytes bytes read in from a ttf file for the font to use on the front cover
	FontBytes []byte

	// ReversePages if true we print the last page first, useful if you are printing in order,
	// so you don't have to manually reverse the pages before cutting them
	ReversePages bool

	// ReverseFrames if true we print the last frame first, useful if you want the flip book to flip
	// from back to front, which can be easier to watch sometimes
	ReverseFrames bool

	// Cover if true a cover image is rendered
	Cover bool

	// GIF if true an animated GIF is generated from the individual frames
	GIF bool

	// SmallFrames if true half size versions of each frame are created in the output dir
	SmallFrames bool

	// Effect is the name of an image processing effect to apply to each frame, values are 'oil'
	Effect string

	// VerLog a logger that will receive verbose information
	VerLog *log.Logger
}

// RenderInfo contains metadata about the completed job
type RenderInfo struct {
	// NFrames the number of frames in the flip
	NFrames int

	// FrameAR the aspect ratio of the final frames
	FrameAR float64
}

type rect struct {
	top    int
	left   int
	width  int
	height int
}

type frame struct {
	path         string
	index        int
	label        string
	info         os.FileInfo
	isFrontCover bool
	bounds       rect
}

type layoutFunc func(pageIndex, nPages, frontCoverIndex int, renderBounds rect, opts Options, frames []os.FileInfo) []frame

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
func To4x6x3(opts Options) (RenderInfo, error) {
	opts.Rows = 3
	opts.Cols = 1

	layoutPage := func(pageIndex, nPages, frontCoverIndex int, renderBounds rect, opts Options, frames []os.FileInfo) []frame {
		var pageLayout []frame
		nFrames := len(frames)

		for ri := 0; ri < opts.Rows; ri++ {
			fi := pageIndex + ri*nPages

			if opts.ReverseFrames {
				fi = nFrames - fi - 1
			}

			frameHeight := renderBounds.height / opts.Rows
			f := frame{
				path: opts.InputDir,
				info: frames[fi],
				bounds: rect{
					left:   renderBounds.left,
					top:    renderBounds.top + frameHeight*ri,
					width:  renderBounds.width,
					height: frameHeight,
				},
				index:        fi,
				isFrontCover: fi == frontCoverIndex,
				label:        opts.Identifier,
			}
			pageLayout = append(pageLayout, f)
		}
		return pageLayout
	}

	return renderPages(opts, layoutPage)
}

// ToLetter renders the frames on a letter page, 10 frames per page
func ToLetter(opts Options) (RenderInfo, error) {
	opts.Rows = 5
	opts.Cols = 2

	layoutPage := func(pageIndex, nPages, frontCoverIndex int, renderBounds rect, opts Options, frames []os.FileInfo) []frame {
		var pageLayout []frame
		nFrames := len(frames)
		//framesPerPage := opts.Cols * opts.Rows

		for ci := 0; ci < opts.Cols; ci++ {
			for ri := 0; ri < opts.Rows; ri++ {
				//fi := pageIndex*framesPerPage + ri + ci*opts.Rows

				fi := pageIndex + (ri*nPages + ci*nPages*opts.Rows)

				if opts.ReverseFrames {
					fi = nFrames - fi - 1
				}

				frameHeight := renderBounds.height / opts.Rows
				frameWidth := renderBounds.width / opts.Cols
				f := frame{
					path: opts.InputDir,
					info: frames[fi],
					bounds: rect{
						left:   renderBounds.left + ci*frameWidth,
						top:    renderBounds.top + frameHeight*ri,
						width:  frameWidth,
						height: frameHeight,
					},
					index:        fi,
					isFrontCover: fi == frontCoverIndex,
					label:        opts.Identifier,
				}
				pageLayout = append(pageLayout, f)
			}
		}
		return pageLayout
	}

	return renderPages(opts, layoutPage)
}

func renderPages(opts Options, layout layoutFunc) (RenderInfo, error) {
	if opts.VerLog == nil {
		return RenderInfo{}, fmt.Errorf("VerLog cannot be nil")
	}

	frames, err := ioutil.ReadDir(opts.InputDir)
	if err != nil {
		return RenderInfo{}, fmt.Errorf("failed to read input images: %s", err)
	}

	// Trim the number of frames so we never end up with any empty spaces on the pages
	nCols := opts.Cols
	nRows := opts.Rows
	framesPerPage := nCols * nRows
	frames = frames[:framesPerPage*(len(frames)/framesPerPage)]

	var coverImgIndex int
	if opts.Cover {
		coverFrame := frames[0]
		coverFramePath := path.Join(opts.InputDir, coverFrame.Name())
		coverImg, err := renderFrontCover(coverFramePath)
		if err != nil {
			return RenderInfo{}, fmt.Errorf("failed to generate cover image: %s", err)
		}

		coverImgOutPath := path.Join(opts.OutputDir, "cover.png")
		err = imaging.Save(coverImg, coverImgOutPath)
		if err != nil {
			return RenderInfo{}, fmt.Errorf("failed to save cover image: %s", err)
		}

		coverImgInfo, err := os.Stat(coverImgOutPath)
		if err != nil {
			return RenderInfo{}, fmt.Errorf("failed to stat cover image: %s", err)
		}

		if opts.ReverseFrames {
			coverImgIndex = len(frames) - 1
		} else {
			coverImgIndex = 0
		}
		frames[coverImgIndex] = coverImgInfo
	}

	//TODO: More efficient - should resize input frames first before applying
	//an effect
	if opts.Effect != "" {
		for _, f := range frames {
			p := path.Join(opts.InputDir, f.Name())
			var outImg *effects.Image

			img, err := effects.LoadImage(p)
			if err != nil {
				return RenderInfo{}, fmt.Errorf("failed to load frame: %s, %s", p, err)
			}

			switch opts.Effect {
			case "oil":
				opts.VerLog.Println("Applying oil effect to:", p)
				outImg, err = effects.OilPainting(img, 0, 5, 30)
				if err != nil {
					return RenderInfo{}, fmt.Errorf("failed to apply oil effect: %s, %s", p, err)
				}
			case "pixelate":
				opts.VerLog.Println("Applying pixelate effect to:", p)
				outImg, err = effects.Pixelate(img, 0, 20)
				if err != nil {
					return RenderInfo{}, fmt.Errorf("failed to apply pixelate effect: %s, %s", p, err)
				}
			case "pencil":
				opts.VerLog.Println("Applying pencil effect to:", p)
				outImg, err = effects.Pencil(img, 0, 5)
				if err != nil {
					return RenderInfo{}, fmt.Errorf("failed to apply pencil effect: %s, %s", p, err)
				}
			case "edge":
				opts.VerLog.Println("Applying edge effect to:", p)
				outImg, err = effects.Sobel(img, 0, -1, false)
				if err != nil {
					return RenderInfo{}, fmt.Errorf("failed to apply edge effect: %s, %s", p, err)
				}
			case "cartoon":
				opts.VerLog.Println("Applying cartoon effect to:", p)
				outImg, err = effects.Cartoon(img, 0, effects.CTOpts{
					BlurKernelSize: 5,
					EdgeThreshold:  50,
					OilFilterSize:  8,
					OilLevels:      20,
				})
				if err != nil {
					return RenderInfo{}, fmt.Errorf("failed to apply cartoon effect: %s, %s", p, err)
				}
			default:
				return RenderInfo{}, fmt.Errorf("invalid effect option: %s", opts.Effect)
			}

			err = outImg.Save(p, effects.SaveOpts{ClipToBounds: true})
			if err != nil {
				return RenderInfo{}, fmt.Errorf("failed to save image with effect: %s, %s", p, err)
			}
		}
	}

	/*
		if opts.GIF {
			renderGIF(frames, opts.InputDir, opts.OutputDir)
		}*/

	nFrames := len(frames)
	nPages := nFrames / framesPerPage

	opts.VerLog.Println("reading input frames from:", opts.InputDir)
	opts.VerLog.Println(nFrames, "found for processing")
	opts.VerLog.Println(nPages, "pages to be generated")

	compWidth := int(opts.Page.Width * float32(opts.Page.DPI))
	compHeight := int(opts.Page.Height * float32(opts.Page.DPI))
	compImg := image.NewRGBA(image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: compWidth, Y: compHeight},
	})

	frameAR := 0.0
	for pi := 0; pi < nPages; pi++ {
		draw.Draw(compImg, compImg.Bounds(), image.White, image.ZP, draw.Src)

		renderBounds := rect{
			left:   int(opts.Page.MarginLeft * float32(opts.Page.DPI)),
			top:    int(opts.Page.MarginTop * float32(opts.Page.DPI)),
			width:  int((opts.Page.Width - (opts.Page.MarginLeft + opts.Page.MarginRight)) * float32(opts.Page.DPI)),
			height: int((opts.Page.Height - (opts.Page.MarginTop + opts.Page.MarginBottom)) * float32(opts.Page.DPI)),
		}

		pageLayout := layout(pi, nPages, coverImgIndex, renderBounds, opts, frames)

		if frameAR == 0.0 {
			frameAR = float64(pageLayout[0].bounds.width) / float64(pageLayout[0].bounds.height)
		}

		for fi := range pageLayout {
			err := compFrame(compImg, pageLayout[fi], opts.Line1Text, opts.Line2Text, opts.FontBytes, opts.VerLog)
			if err != nil {
				return RenderInfo{}, err
			}
		}

		// When you print pictures, maybe the service orders them by filename e.g. comp001, comp002 etc so the last
		// frames are printed on the top of the stack so you have to reverse them for assembly, this flag flips the
		// numbering so that you don't need to do this after printing
		var compIndex int
		if opts.ReversePages {
			compIndex = nPages - pi - 1
		} else {
			compIndex = pi
		}
		err = writeJPG(compImg, opts.OutputDir, opts.Identifier, compIndex, opts.VerLog)
		if err != nil {
			return RenderInfo{}, err
		}
	}

	return RenderInfo{
		NFrames: nFrames,
		FrameAR: frameAR,
	}, nil
}

func renderFrontCover(framePath string) (image.Image, error) {
	src, err := imaging.Open(framePath)
	if err != nil {
		return nil, err
	}

	dst := imaging.Blur(src, 12.5)
	return dst, nil
}

func annotateFrontCover(img *image.RGBA, dstRect image.Rectangle, labelLine1, labelLine2 string, fontBytes []byte) error {
	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		return fmt.Errorf("failed to parse font file: %s", err)
	}

	c := freetype.NewContext()
	c.SetDPI(70)
	c.SetFont(f)
	c.SetClip(dstRect)
	c.SetDst(img)
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
	x := dstRect.Min.X + 100
	y := dstRect.Min.Y + 30

	if err = renderText(x, y, line1FontSize, image.Black, labelLine1); err != nil {
		return err
	}
	if err = renderText(x-4, y-4, line1FontSize, image.Black, labelLine1); err != nil {
		return err
	}

	if err = renderText(x-2, y-2, line1FontSize, image.White, labelLine1); err != nil {
		return err
	}

	if err = renderText(x, y+line2YOffset, line2FontSize, image.Black, labelLine2); err != nil {
		return err
	}
	if err = renderText(x-2, y-2+line2YOffset, line2FontSize, image.White, labelLine2); err != nil {
		return err
	}

	return nil
}

func compFrame(compImg *image.RGBA, f frame, labelLine1, labelLine2 string, fontBytes []byte, verLog *log.Logger) error {

	imgPath := path.Join(f.path, f.info.Name())
	verLog.Println("reading:", imgPath)
	srcReader, err := os.Open(imgPath)
	if err != nil {
		return fmt.Errorf("failed to read input image: %s, %s", imgPath, err)
	}

	verLog.Println("decoding:", imgPath)
	srcImg, err := png.Decode(srcReader)
	srcReader.Close()
	if err != nil {
		return fmt.Errorf("failed to decode image on load: %s, %s", imgPath, err)
	}

	verLog.Println("bounds:", srcImg.Bounds())

	sourceWidth := srcImg.Bounds().Dx()
	sourceHeight := srcImg.Bounds().Dy()

	// Render the image scaled to the dimensions we want
	targetHeight := f.bounds.height
	scaledWidth := int(float64(targetHeight) / float64(sourceHeight) * float64(sourceWidth))
	scaledImg := image.NewRGBA(image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: scaledWidth, Y: targetHeight},
	})
	draw.BiLinear.Scale(scaledImg, scaledImg.Bounds(), srcImg, srcImg.Bounds(), draw.Src, nil)

	// Composite into page container
	left := int(math.Max(float64(f.bounds.left), float64(f.bounds.left+(f.bounds.width-scaledWidth))))
	dstRect := image.Rectangle{
		Min: image.Point{X: left, Y: f.bounds.top},
		Max: image.Point{X: f.bounds.left + f.bounds.width, Y: f.bounds.top + f.bounds.height},
	}
	draw.Draw(
		compImg,
		dstRect,
		scaledImg,
		image.Point{
			X: int(math.Max(0, float64(scaledImg.Bounds().Dx()-f.bounds.width))),
			Y: 0,
		},
		draw.Src)

	// Draw the left size bar
	barWidth := 100 //int(math.Max(100.0, float64(left-f.bounds.left)))
	draw.Draw(compImg, image.Rectangle{
		Min: image.Point{X: f.bounds.left, Y: f.bounds.top},
		Max: image.Point{X: f.bounds.left + barWidth, Y: f.bounds.top + f.bounds.height},
	}, image.Black, image.ZP, draw.Src)

	if !f.isFrontCover {
		x := f.bounds.left + 20
		y := f.bounds.top + int(float32(f.bounds.height)*0.5)
		yOffset := 20
		addDebugLabel(compImg, x, y, strconv.Itoa(f.index))
		addDebugLabel(compImg, x, y+yOffset, f.label)
	}

	if f.isFrontCover {
		dr := image.Rectangle{
			Min: image.Point{X: f.bounds.left + barWidth, Y: f.bounds.top},
			Max: image.Point{X: f.bounds.left + f.bounds.width, Y: f.bounds.top + f.bounds.height},
		}
		err := annotateFrontCover(compImg, dr, labelLine1, labelLine2, fontBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeJPG(compImg *image.RGBA, outputDir, identifier string, imgIndex int, verLog *log.Logger) error {
	toImgPath := path.Join(outputDir, fmt.Sprintf("comp-%s-%03d.jpg", identifier, imgIndex))
	toImg, err := os.Create(toImgPath)
	if err != nil {
		return fmt.Errorf("failed to create image: %s, %s", toImgPath, err)
	}

	verLog.Println("writing:", toImgPath)

	err = jpeg.Encode(toImg, compImg, &jpeg.Options{Quality: 90})
	toImg.Close()
	if err != nil {
		return fmt.Errorf("failed to save img: %s, %s", toImgPath, err)
	}

	verLog.Println("written file:", toImgPath)
	return nil
}

func addDebugLabel(img *image.RGBA, x, y int, label string) {
	point := fixed.Point26_6{fixed.Int26_6(x * 64), fixed.Int26_6(y * 64)}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.White,
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
}

/*
func renderGIF(frames []os.FileInfo, inputDir, outputPath string) error {
	outGif := &gif.GIF{}
	for _, f := range frames {

		imgPath := path.Join(inputDir, f.Name())
		srcReader, err := os.Open(imgPath)
		if err != nil {
			panic("cant open")
			return fmt.Errorf("failed to read input image: %s, %s", imgPath, err)
		}

		srcImg, err := png.Decode(srcReader)
		srcReader.Close()
		if err != nil {
			panic("cant decode")
			return fmt.Errorf("failed to decode image on load: %s, %s", imgPath, err)
		}

		fmt.Println("generate palette:", f.Name())
		pm := image.NewPaletted(srcImg.Bounds(), nil)
		q := &gogif.MedianCutQuantizer{NumColor: 256}
		q.Quantize(pm, srcImg.Bounds(), srcImg, image.ZP)
		//f, _ := os.Open(name)
		//inGif, _ := gif.Decode(f)
		//f.Close()

		outGif.Image = append(outGif.Image, pm)
		outGif.Delay = append(outGif.Delay, 0)
	}

	fmt.Println("writing gif to", outputPath)
	ff, _ := os.OpenFile(path.Join(outputPath, "out.gif"), os.O_WRONLY|os.O_CREATE, 0600)
	defer ff.Close()
	gif.EncodeAll(ff, outGif)

	return nil
}
*/
