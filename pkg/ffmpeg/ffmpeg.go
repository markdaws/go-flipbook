package ffmpeg

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
)

// VideoFilter extracts individual frames from a video source and saves them as
// images to the specified output location
func VideoFilter(input, output string, fps int, verLog *log.Logger) error {
	if fps < 15 || fps > 60 {
		return fmt.Errorf("fps must be between 15 and 60, %d invalid value", fps)
	}

	if _, err := os.Stat(input); os.IsNotExist(err) {
		return fmt.Errorf("invalid input, file does not exist: %s", input)
	}

	if _, err := os.Stat(output); os.IsNotExist(err) {
		return fmt.Errorf("invalid output, the directory does not exist, please create it:  %s", output)
	}

	if installed, _ := FFMPEGIsInstalled(); !installed {
		return fmt.Errorf("ffmpeg is not installed, please install then re-run")
	}

	if verLog == nil {
		verLog = log.New(ioutil.Discard, "", 0)
	}

	verLog.Println("Generating frames from:", input)
	verLog.Println("Writing frames to:", output)
	verLog.Println("fps=", fps)

	//TODO: pass this in
	var maxFrames = 5 * fps
	cmd := exec.Command("ffmpeg", "-i", input, "-vframes", strconv.Itoa(maxFrames), "-start_number", "0",
		"-vf", "fps="+strconv.Itoa(fps), path.Join(output, "img%03d.png"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to extract frames: %s, details: %s", err, stderr.String())
	}

	return nil
}

// FFMPEGIsInstalled returns true if the ffmpeg binary is installed, along with the path
// to the installed binary, false if not installed
func FFMPEGIsInstalled() (bool, string) {
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		return false, ""
	}
	return true, path
}
