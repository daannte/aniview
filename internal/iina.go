package internal

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

// Connect to the MPV IPC server
func connectToPipe(ipcSocketPath string) (net.Conn, error) {
	// Connect to the UNIX socket
	conn, err := net.Dial("unix", ipcSocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the socket: %v", err)
	}
	return conn, nil
}

// MPVSendCommand sends a command to the MPV IPC server and returns the result
func MPVSendCommand(ipcSocketPath string, command []interface{}) (interface{}, error) {
	// Connect to the IPC socket
	conn, err := connectToPipe(ipcSocketPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Marshal the command to JSON
	commandStr, err := json.Marshal(map[string]interface{}{
		"command": command,
	})
	if err != nil {
		return nil, err
	}

	// Send the command to the IPC server
	_, err = conn.Write(append(commandStr, '\n'))
	if err != nil {
		return nil, err
	}

	// Receive the response
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	// Parse the response
	var response map[string]interface{}
	if err := json.Unmarshal(buf[:n], &response); err != nil {
		return nil, err
	}

	// Return the response data
	if data, exists := response["data"]; exists {
		return data, nil
	}

	return nil, nil
}

func getVideoDuration(ipcSocketPath string) (float64, error) {
	// Send the 'get_property' command to MPV to get the duration of the video
	command := []interface{}{"get_property", "duration"}
	data, err := MPVSendCommand(ipcSocketPath, command)
	if err != nil {
		return 0, err
	}

	// Assert the data to float64 (MPV will return a float representing the duration in seconds)
	if videoDuration, ok := data.(float64); ok {
		return videoDuration, nil
	}

	return 0, nil
}

func getCurrentPlaybackTime(ipcSocketPath string) (float64, error) {
	// Send the 'get_property' command to MPV to get the current time-pos
	command := []interface{}{"get_property", "time-pos"}
	data, err := MPVSendCommand(ipcSocketPath, command)
	if err != nil {
		return 0, err
	}

	// Assert the data to float64 (MPV will return a float representing time in seconds)
	if timePos, ok := data.(float64); ok {
		return timePos, nil
	}

	return 0, nil
}

func getPausedState(ipcSocketPath string) (bool, error) {
	// Send the 'get_property' command to MPV to get the 'pause' property
	command := []interface{}{"get_property", "pause"}
	data, err := MPVSendCommand(ipcSocketPath, command)
	if err != nil {
		return false, err
	}

	// Assert the data to a boolean (MPV will return true/false for pause state)
	if paused, ok := data.(bool); ok {
		return paused, nil
	}

	return false, nil
}

func CleanupSocket(socketPath string) {
	if err := os.Remove(socketPath); err != nil {
		fmt.Printf("Error removing socket: %v\n", err)
	}
}

func waitForSocket(socketPath string) error {
	for {
		_, err := os.Stat(socketPath)
		if err == nil {
			// File exists, proceed
			return nil
		}

		if os.IsNotExist(err) {
			// File does not exist, keep waiting
			time.Sleep(1 * time.Second)
			continue
		}

		// An unexpected error occurred (e.g., permission issues)
		return fmt.Errorf("error checking socket file: %v", err)
	}
}
