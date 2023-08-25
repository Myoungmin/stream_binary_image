package main

import (
	"fmt"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)

const (
	width    = 2048
	height   = 2048
	fps      = 60
	interval = int(1000000 / fps)
)

var (
	frame  = 0
	images = [][]byte{createImage(0), createImage(1)}
)

func main() {
	port := 8080

	http.Handle("/", websocket.Handler(socketHandler))
	http.HandleFunc("/image", imageHandler)

	fmt.Printf("Listening on port %d...\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

func socketHandler(ws *websocket.Conn) {
	fmt.Println("WebSocket opened.")

	ticker := time.NewTicker(time.Microsecond * time.Duration(interval))
	defer ticker.Stop()

	openSocket := false

	for {
		select {
		case <-ticker.C:
			if openSocket {
				frame++
				frameData := images[frame%2]
				_, err := ws.Write(frameData)
				if err != nil {
					fmt.Printf("Error writing frame: %s\n", err.Error())
					return
				}
			}
		default:
			var event string
			err := websocket.Message.Receive(ws, &event)
			if err != nil {
				fmt.Printf("WebSocket error: %s\n", err.Error())
				return
			}

			if !openSocket {
				openSocket = true
			} else {
				if event == "start" {
					openSocket = true
				} else if event == "quit" {
					openSocket = false
				}
			}
		}
	}
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	imageIndex := 0
	if r.Method == http.MethodPost {
		imageIndex = 1
	}

	imageData := images[imageIndex]
	_, err := w.Write(imageData)
	if err != nil {
		fmt.Printf("Error writing image: %s\n", err.Error())
	}
}

func createImage(i int) []byte {
	image := make([]byte, width*height)

	if i == 0 {
		return image
	}

	roic := width / 10
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			image[y*width+x] = byte((x / roic) * 25)
		}
	}
	return image
}
