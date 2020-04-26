package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
)

var segments []string

func main() {
	chunkCount := flag.Int("chunks", 0, "number of chunks the stream is split into")
	cleanupChunk := flag.Bool("clean", false, "clean up chunk files")
	title := flag.String("title", "output", "filename for the output")
	flag.Parse()

	fmt.Println(*chunkCount)

	// grab all of the segments
	if *chunkCount != 0 {
		for i := 1; i < *chunkCount+1; i++ {
			_ = makeRequest(i)
		}
	} else {
		counter := 1
		for {
			err := makeRequest(counter)
			if err != nil {
				break
			}
			counter++
		}
	}

	for i, seg := range segments {
		// convert all segments to working version of vlc ts
		cmd := exec.Command("ffmpeg", "-i", seg, "-c", "copy", seg+".fixed.ts")
		fmt.Println(cmd.String())
		err := cmd.Run()
		if err != nil {
			log.Fatalf("[ error ] error converting to fixed version: %s\n", err)
		}

		segments[i] = seg + ".fixed.ts"

		// convert to mp4
		cmd = exec.Command("ffmpeg", "-i", seg, "-c", "copy", seg+".mp4")
		fmt.Println(cmd.String())
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
	fmt.Println("[ info ] command to be run: " + cmd.String())
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

func makeRequest(chunk int) error {
	url := fmt.Sprintf("", chunk)

	filepath := fmt.Sprintf("seg-%d-v1-a1.ts", chunk)
	segments = append(segments, filepath) // use this to create the ffmpeg concat

	if _, err := os.Stat(filepath); err == nil {
		return err
	}

	output, err := os.Create(filepath)
	if err != nil {
		fmt.Println("Error while creating", filepath, "-", err)
		return err
	}
	defer output.Close()

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return err
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return err
	}

	fmt.Println(n, "bytes downloaded.")
	return nil
}
