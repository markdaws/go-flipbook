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
	"strings"
)

// VideoFilter extracts individual frames from a video source and saves them as
// images to the specified output location
func VideoFilter(input, output, identifier string, fps int, startTime uint, maxLength int, verLog *log.Logger) ([]os.FileInfo, error) {
	if fps < 1 || fps > 60 {
		return nil, fmt.Errorf("fps must be between 1 and 60, %d invalid value", fps)
	}

	if _, err := os.Stat(input); os.IsNotExist(err) {
		return nil, fmt.Errorf("invalid input, file does not exist: %s", input)
	}

	if _, err := os.Stat(output); os.IsNotExist(err) {
		return nil, fmt.Errorf("invalid output, the directory does not exist, please create it:  %s", output)
	}

	if installed, _ := FFMPEGIsInstalled(); !installed {
		return nil, fmt.Errorf("ffmpeg is not installed, please install then re-run")
	}

	if verLog == nil {
		verLog = log.New(ioutil.Discard, "", 0)
	}

	verLog.Println("Generating frames from:", input)
	verLog.Println("Writing frames to:", output)
	verLog.Println("fps=", fps)

	var maxFrames = maxLength * fps
	const prefix = "frame-"
	cmd := exec.Command("ffmpeg", "-ss", strconv.Itoa(int(startTime)), "-i", input,
		"-vframes", strconv.Itoa(maxFrames), "-start_number", "0",
		"-vf", "fps="+strconv.Itoa(fps), path.Join(output, prefix+identifier+"-%03d.png"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to extract frames: %s, details: %s", err, stderr.String())
	}

	files, err := ioutil.ReadDir(output)
	if err != nil {
		return nil, err
	}

	var filteredFiles []os.FileInfo
	for _, f := range files {
		if strings.HasPrefix(f.Name(), prefix) {
			filteredFiles = append(filteredFiles, f)
		}
	}

	return filteredFiles, nil
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
