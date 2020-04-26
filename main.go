package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/cheggaaa/pb"
)

var segments []string

func main() {
	chunkCount := flag.Int("chunks", 0, "number of chunks the stream is split into")
	cleanupChunk := flag.Bool("clean", false, "clean up chunk files")
	title := flag.String("title", "output", "filename for the output")
	flag.Parse()

	// grab all of the segments
	if *chunkCount != 0 {
		bar := pb.StartNew(*chunkCount) // progress bar :)
		for i := 1; i < *chunkCount+1; i++ {
			_ = makeRequest(i)
			bar.Increment()
		}
		bar.Finish()
	} else {
		bar := pb.StartNew(255)
		counter := 1
		for {
			err := makeRequest(counter)
			if err != nil {
				break
			}
			counter++
			bar.Increment()
		}
		bar.Finish()

		// is my hunch right in assuming that the chunks will never go beyond a certain length?
		// for i := 1; i < 255; i++ {
		// 	err := makeRequest(i)
		// 	if err != nil {
		// 		break
		// 	}
		// 	bar.Increment()
		// }
	}

	for i, seg := range segments {
		// convert all segments to working version of vlc ts
		cmd := exec.Command("ffmpeg", "-i", seg, "-c", "copy", seg+".fixed.ts")
		log.Println(cmd.String())
		err := cmd.Run()
		if err != nil {
			log.Fatalf("[ error ] error converting to fixed version: %s\n", err)
		}

		segments[i] = seg + ".fixed.ts"

		// convert to mp4
		cmd = exec.Command("ffmpeg", "-i", seg, "-c", "copy", seg+".mp4")
		log.Println(cmd.String())
		err = cmd.Run()
		if err != nil {
			log.Fatalf("[ error ] error converting to fixed version: %s\n", err)
		}
		segments[i] = seg + ".mp4"
	}

	// create the inputs file
	createInputConcat()

	// create the command
	filename := *title + ".mp4"
	cmd := exec.Command("ffmpeg", "-f", "concat", "-i", "input.txt", "-c", "copy", filename)
	log.Println("[ info ] command to be run: " + cmd.String())
	err := cmd.Run()
	if err != nil {
		log.Fatalf("[ error ] error during concat: %s\n", err)
	}

	if *cleanupChunk {
		cmd := exec.Command("/bin/sh", "-c", "rm ./seg* ./input.txt")
		err := cmd.Run()
		if err != nil {
			log.Fatalf("[ error ] error during cleanup: %s\n", err)
		}
	}

	log.Println("[ success ]")
}

func createInputConcat() {
	file, err := os.OpenFile("input.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		log.Fatalf("[ error ] failed creating file: %s", err)
	}

	datawriter := bufio.NewWriter(file)

	for _, data := range segments {
		_, _ = datawriter.WriteString("file '" + data + "'\n")
	}

	datawriter.Flush()
	file.Close()
}

// todo:
// add check for size to break out
func makeRequest(chunk int) error {
	url := fmt.Sprintf("%d", chunk)

	filepath := fmt.Sprintf("seg-%d-v1-a1.ts", chunk)
	segments = append(segments, filepath) // use this to create the ffmpeg concat

	if _, err := os.Stat(filepath); err == nil {
		return err
	}

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return err
	}
	defer response.Body.Close()

	if response.StatusCode == 404 {
		return errors.New("End of content")
	}

	output, err := os.Create(filepath)
	if err != nil {
		fmt.Println("Error while creating", filepath, "-", err)
		return err
	}
	defer output.Close()

	_, err = io.Copy(output, response.Body)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return err
	}

	return nil
}
