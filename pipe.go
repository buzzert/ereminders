package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"
	"syscall"
)

const pipesDirectory string = "/tmp/ereminders_pipes"
const commandPipeName string = "commands"
const commandResultPipeName string = "recv"

func getTemporaryDirectory() (string, error) {
	err := os.MkdirAll(pipesDirectory, 0755)
	if err != nil {
		log.Println("Error: Unable to create temporary directory")
		return "", err
	}

	return pipesDirectory, nil
}

func makeNamedPipe(name string) string {
    tempDir, _ := getTemporaryDirectory()
	namedPipeFilename := filepath.Join(tempDir, name)
	syscall.Mkfifo(namedPipeFilename, 0600)

	return namedPipeFilename
}

func openPipe(pipeName string, flags int) *os.File {
	filename := makeNamedPipe(pipeName)
	pipe, err := os.OpenFile(filename, flags, 0600)
	if err != nil {
		log.Print("Error opening pipe: ")
		log.Println(err)
		return nil
	}

	return pipe
}

// ReadCommand blocks on the command pipe until data is received
func ReadCommand() string {
	commandPipe := openPipe(commandPipeName, os.O_RDONLY)

	var commandBuf bytes.Buffer
	io.Copy(&commandBuf, commandPipe)
	commandPipe.Close()

	return commandBuf.String()
}

// SendCommand sends a command string to the command named pipe
func SendCommand(command string) {
	commandPipe := openPipe(commandPipeName, os.O_WRONLY)
	commandPipe.WriteString(command)
	commandPipe.Close()
}

// SendResponse sends the response to the tx/rx pipe
func SendResponse(response string) {
	txPipe := openPipe(commandResultPipeName, os.O_WRONLY)
	txPipe.WriteString(response)
	txPipe.Close()
}

// ReadResponse blocks while it waits for a response on the tx/rx named pipe
func ReadResponse() string {
	rxPipe := openPipe(commandResultPipeName, os.O_RDONLY)

	var rxBuf bytes.Buffer
	io.Copy(&rxBuf, rxPipe)
	rxPipe.Close()

	return rxBuf.String()
}

// TransmitCommand sends the command to the command pipe, then waits for a response from the
// recv pipe
func TransmitCommand(command string) string {
	SendCommand(command)
	return ReadResponse()
}
