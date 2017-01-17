package main

import (
	"encoding/base64"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/markdaws/go-flipbook/pkg/composite"
	"github.com/markdaws/go-flipbook/pkg/ffmpeg"
)

// Injected by the build process
var version string

func main() {

	//required
	input := flag.String("input", "", "Path to the input video source (required)")
	output := flag.String("output", "", "Path where the images will be written to. Images will be generated with names img001.png, img002.png ... etc. (required)")

	//optional
	fontPath := flag.String("fontpath", "", "Path to the font file used for the text on the front cover (must be a ttf file). If not specified, HelveticaNeue will be used")
	line1Text := flag.String("line1text", "", "Text to display on line 1 of the flipbook cover")
	line2Text := flag.String("line2text", "", "Text to display on line 2 of the flipbook cover")
	titleEncoded := flag.Bool("titleencoded", false, "If true, the line1text and line2text are expected to be base64 encoded strings, useful for untrusted input")
	fps := flag.Int("fps", 15, "The number of frames to generate per second of video. Min 10, max 60")
	clean := flag.Bool("clean", false, "If true, all files in the output directory are deleted before generating new items")
	cleanFrames := flag.Bool("cleanframes", false, "If true, deletes all of the individual video frames after compositing")
	bgColor := flag.String("bgcolor", "white", "The background color of the image (for border). Can be white|black")
	skipVideo := flag.Bool("skipvideo", false, "If true frames are not extracted and the input option is not required")
	cover := flag.Bool("cover", false, "If true, a cover page is added to the rendered frames")
	startTime := flag.Int("starttime", 0, "The start time in the input video to use as the start of the flip book")
	maxLength := flag.Int("maxlength", 5, "The maximum length of the input video to process in seconds")
	identifier := flag.String("identifier", "", "A string that will be printed on each frame, for easy identification")
	reversePages := flag.Bool("reversepages", false, "If true, the lowest numbered output page will contain the last frames. Useful if you print and don't want to have to manually reverse the printed stack for assembly, so you end up with page 1 on top")
	reverseFrames := flag.Bool("reverseframes", false, "If true, frame 0 will be printed last, in this case you flip from the end of the book to the front to view the scene, which I have found is easier than flipping front to back")
	smallFrames := flag.Bool("smallframes", false, "If true, we create small 600x300 version of each of the final images, can be useful if you want to show a preview of each page somewhere")
	ver := flag.Bool("version", false, "Displays the app version number")
	verbose := flag.Bool("verbose", false, "Prints verbose output as the process is running")

	flag.Parse()

	infoLog := log.New(os.Stdout, "INFO: ", 0)
	errLog := log.New(os.Stderr, "ERR: ", 0)
	var verLog *log.Logger
	if *verbose {
		verLog = log.New(os.Stdout, "VERBOSE: ", 0)
	} else {
		verLog = log.New(ioutil.Discard, "VERBOSE: ", 0)
	}

	if *ver {
		infoLog.Println(version)
		return
	}

	switch *bgColor {
	case "white", "black":
	default:
		errLog.Println("--bgcolor must be white|black, invalid option:", *bgColor)
		flag.PrintDefaults()
		os.Exit(1)
	}

	var fontBytes []byte
	if *fontPath != "" {
		var err error
		fontBytes, err = ioutil.ReadFile(*fontPath)
		if err != nil {
			errLog.Println("--fontpath cannot open font file:", *fontPath)
			os.Exit(1)
		}
	} else {
		var err error
		// defined in auto generated bindata.go file
		fontBytes, err = Asset("data/HelveticaNeue.ttf")
		if err != nil {
			errLog.Println("failed to read default font HelveticaNeue")
			os.Exit(1)
		}
	}

	if *input == "" && !*skipVideo {
		errLog.Println("--input is a required option\n\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *output == "" {
		errLog.Println("--output is a required option\n\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *fps < 10 || *fps > 60 {
		errLog.Println("--fps must be a value between 10 and 60")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *clean {
		verLog.Println("Cleaning:", *output)

		if _, err := os.Stat(*output); os.IsNotExist(err) {
			errLog.Printf("invalid output, the directory does not exist, please create it:  %s", *output)
			os.Exit(1)
		}

		files, err := ioutil.ReadDir(*output)
		if err != nil {
			errLog.Println("Failed to clean files:", err)
			os.Exit(1)
		}
		for _, file := range files {
			filePath := path.Join(*output, file.Name())
			err := os.Remove(filePath)
			if err != nil {
				errLog.Printf("Failed to delete %s: %s", filePath, err)
				os.Exit(1)
			}
			verLog.Println("Deleted:", filePath)
		}
	}

	var frames []os.FileInfo
	if !*skipVideo {
		var err error
		frames, err = ffmpeg.VideoFilter(*input, *output, *identifier, *fps, uint(*startTime), *maxLength, verLog)
		if err != nil {
			errLog.Println("failed to extract frames:", err)
			os.Exit(1)
		}
	}

	bgColorComp := *bgColor
	if bgColorComp == "" {
		bgColorComp = "white"
	}

	line1 := *line1Text
	line2 := *line2Text
	if *titleEncoded {
		b, err := base64.StdEncoding.DecodeString(line1)
		if err != nil {
			errLog.Println("error decoding line1text:", err)
			os.Exit(1)
		}
		line1 = string(b)
		b, err = base64.StdEncoding.DecodeString(line2)
		if err != nil {
			errLog.Println("error decoding line2text:", err)
			os.Exit(1)
		}
		line2 = string(b)
	}

	/*
		page := composite.Page{
			Width:        8.5,
			Height:       11,
			MarginTop:    0.5,
			MarginRight:  0.5,
			MarginBottom: 0.5,
			MarginLeft:   0.5,
			DPI:          300,
		}
	*/

	page = composite.Page{
		Width:        4,
		Height:       6,
		MarginTop:    0,
		MarginRight:  0,
		MarginBottom: 0,
		MarginLeft:   0,
		DPI:          300,
	}

	err := composite.To4x6x3(composite.Options{
		Page:          page,
		Rows:          3,
		Cols:          1,
		BGColor:       bgColorComp,
		OutputDir:     *output,
		InputDir:      *output,
		Line1Text:     line1,
		Line2Text:     line2,
		Identifier:    *identifier,
		FontBytes:     fontBytes,
		ReversePages:  *reversePages,
		ReverseFrames: *reverseFrames,
		Cover:         *cover,
		SmallFrames:   *smallFrames,
		VerLog:        verLog,
	})

	if err != nil {
		errLog.Println("failed to composite images:", err)
		os.Exit(1)
	}

	if *cleanFrames {
		verLog.Println("cleaning frames")
		for _, f := range frames {
			filePath := path.Join(*output, f.Name())
			err := os.Remove(filePath)
			if err != nil {
				errLog.Printf("Failed to delete %s: %s", filePath, err)
			}
			verLog.Println("Deleted:", filePath)
		}
	}

	infoLog.Println("All done")
}
