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

/*
--input=mymov.mp4
--output=something%d.png
--fps=15
*/

// Injected by the build process
var version string

func main() {

	ver := flag.Bool("version", false, "Displays the app version number")
	verbose := flag.Bool("verbose", false, "Prints verbose output as the process is running")
	input := flag.String("input", "", "Path to the input video source (required)")
	output := flag.String("output", "", "Path where the images will be written to. Images will be generated with names img001.png, img002.png ... etc. (required)")
	fps := flag.Int("fps", 15, "The number of frames to generate per second of video, defaults to 15. Min 15, max 60")
	clean := flag.Bool("clean", false, "If true, all files in the output directory are deleted before generating new items")
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

	if *input == "" {
		errLog.Println("--input is a required option\n\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *output == "" {
		errLog.Println("--output is a required option\n\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *fps < 15 || *fps > 60 {
		errLog.Println("--fps must be a value between 15 and 60")
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

	// Generate all the stills from the input
	err := ffmpeg.VideoFilter(*input, *output, *fps, verLog)
	if err != nil {
		errLog.Println("failed to extract frames:", err)
		os.Exit(1)
	}

	// Generate composites
	err = composite.To4x6x3(*output, *output, verLog)
	if err != nil {
		errLog.Println("failed to composite images:", err)
		os.Exit(1)
	}

	infoLog.Println("All done")
}
