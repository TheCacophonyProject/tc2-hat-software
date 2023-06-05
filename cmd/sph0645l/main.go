package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/stianeikeland/go-rpio"
)

const (
	// Pin 17 is the GPIO17 pin on the Raspberry Pi
	pin            = rpio.Pin(17)
	duration       = 10
	arecordCmdPath = "/usr/bin/arecord"
	soxCmdPath     = "/usr/bin/sox"
	audioTrim      = 1
	gain           = 40
)

func main() {
	err := runMain()
	if err != nil {
		log.Println(err)
	}
}

func runMain() error {
	log.SetFlags(0)
	// Enable microphone by driving the CS pin on the LORA moduel high.
	err := rpio.Open()
	if err != nil {
		log.Fatalln(err)
	}
	defer rpio.Close()
	pin.Output()
	pin.High()
	time.Sleep(time.Second)

	tempDir := os.TempDir()

	log.Println("recording audio")
	rawWav := filepath.Join(tempDir, "raw.wav")
	//arecord -D plughw:0 -c1 -r 48000 -f S32_LE -t wav -V mono file.wav
	cmd := cmdFromStr("arecord -D plughw:0 -c1 -r 48000 -f S32_LE -t wav -V mono -d %d %s", duration+audioTrim, rawWav)
	//cmd := exec.Command(arecordCmdPath, "-D", "plughw:0", "-c1", "-r", "48000", "-d", strconv.Itoa(duration+audioTrim),
	//		"-f", "S32_LE", "-t", "wav", "-V", "mono", rawWav)
	err = cmd.Start()
	if err != nil {
		return err
	}

	// Wait for the recording to complete
	err = cmd.Wait()
	if err != nil {
		return err
	}

	// Stop disabling LORA module
	pin.Low()

	// Trim first 5 seconds
	log.Println("trimming the first 5 seconds")
	recordingTrimmed := filepath.Join(tempDir, "raw-trimmed.wav")
	//cmd = exec.Command(soxCmdPath, rawWav, recordingTrimmed, "trim", strconv.Itoa(audioTrim))
	cmd = cmdFromStr("sox %s %s trim %d", rawWav, recordingTrimmed, audioTrim)
	//log.Println(cmd.Args)
	if err := cmd.Run(); err != nil {
		log.Println("error running sox command:", err)
	}

	// Find DC offset
	log.Println("finding the DC offset")
	dcOffset := 0.0
	cmd = cmdFromStr("sox %s -n stat", recordingTrimmed)
	//cmd = exec.Command(soxCmdPath, recordingTrimmed, "-n", "stat")
	//log.Println(cmd.Args)
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(outBytes), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 3 && fields[0] == "Mean" && fields[1] == "amplitude:" {
			dcOffset, err = strconv.ParseFloat(fields[2], 64)
			if err != nil {
				return err
			}
		}
	}
	log.Println("the dc offset is ", dcOffset)

	// Remove DC offset
	log.Println("removing DC offset")
	removedDC := filepath.Join(tempDir, "raw-trimmed-remove-dc.wav")
	cmd = cmdFromStr("sox %s %s dcshift %f", recordingTrimmed, removedDC, -1*dcOffset)
	//cmd = exec.Command("sox", recordingTrimmed, removedDC, "dcshift", strconv.FormatFloat(-1*dcOffset, 'f', 6, 64))
	//log.Println(cmd.Args)
	if err := cmd.Run(); err != nil {
		log.Println("error running sox command:", err)
	}

	// Amplify recording
	log.Println("amplifying recording")
	ampRec := filepath.Join(tempDir, "raw-trimmed-remove-dc-amplified.wav")
	cmd = cmdFromStr("sox %s %s gain %d", removedDC, ampRec, gain)
	//cmd = exec.Command(soxCmdPath, removedDC, ampRec, "gain", "40")
	//log.Println(cmd.Args)
	if err := cmd.Run(); err != nil {
		log.Println("error running sox command:", err)
	}

	// Compress to mp4
	log.Println("compressing to mp4")
	mp4Rec := filepath.Join(tempDir, "recording.m4a")
	cmd = cmdFromStr("ffmpeg -i %s -c:a aac -q:a %d %s -y", ampRec, 6, mp4Rec)
	if err := cmd.Run(); err != nil {
		log.Println("error running ffmpeg command:", err)
	}
	log.Println("all done")

	return nil
}

// Makes a command from a string.
// Won't work with arguments that have spaces in them
func cmdFromStr(cmd string, vars ...interface{}) *exec.Cmd {
	commandStr := fmt.Sprintf(cmd, vars...)
	log.Println(commandStr)
	command := strings.Split(commandStr, " ")
	return exec.Command(command[0], command[1:]...)
}
