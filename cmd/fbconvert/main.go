package main

import (
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
	fontPath := flag.String("fontpath", "", "Path to the font file used for the text on the front cover (HeleveticNeue.ttf is included in the github repo)")

	//optional
	line1Text := flag.String("line1text", "", "Text to display on line 1 of the flipbook cover")
	line2Text := flag.String("line2text", "", "Text to display on line 2 of the flipbook cover")
	fps := flag.Int("fps", 15, "The number of frames to generate per second of video. Min 10, max 60")
	clean := flag.Bool("clean", false, "If true, all files in the output directory are deleted before generating new items")
	cleanFrames := flag.Bool("cleanframes", false, "If true, deletes all of the individual video frames after compositing")
	bgColor := flag.String("bgcolor", "white", "The background color of the image (for border). Can be white|black")
	skipVideo := flag.Bool("skipvideo", false, "If true frames are not extracted and the input option is not required")
	skipCover := flag.Bool("skipcover", false, "If true, a cover page is not added to the rendered frames")
	maxLength := flag.Int("maxlength", 5, "The maximum length of the input video to process in seconds")
	identifier := flag.String("identifier", "", "A string that will be printed on each frame, for easy identification")
	reversePages := flag.Bool("reversepages", false, "If true, the lowest numbered output page will contain the last frames. Useful if you print and don't want to have to manually reverse the printed stack for assembly, so you end up with page 1 on top")
	reverseFrames := flag.Bool("reverseframes", false, "If true, frame 0 will be printed last, in this case you flip from the end of the book to the front to view the scene, which I have found is easier than flipping front to back")
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

	if *fontPath == "" {
		errLog.Println("fontpath is a required option\n\n")
		flag.PrintDefaults()
		os.Exit(1)
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
		// Generate all the stills from the input
		var err error
		frames, err = ffmpeg.VideoFilter(*input, *output, *identifier, *fps, *maxLength, verLog)
		if err != nil {
			errLog.Println("failed to extract frames:", err)
			os.Exit(1)
		}
	}

	// Generate composites
	bgColorComp := *bgColor
	if bgColorComp == "" {
		bgColorComp = "white"
	}

	err := composite.To4x6x3(
		bgColorComp, *output, *output, *line1Text, *line2Text, *fontPath,
		*identifier, *reversePages, *reverseFrames, *skipCover, verLog)

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
