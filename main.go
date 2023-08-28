package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	width    = 1024
	height   = 1024
	fps      = 60
	interval = int(1000000 / fps)
)

var (
	frame  = 0
	images = [][]byte{createImage(0), createImage(1)}
)

func main() {
	port := 8080

	http.HandleFunc("/", socketHandler)
	http.HandleFunc("/image", imageHandler)

	fmt.Printf("Listening on port %d...\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func socketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("WebSocket upgrade error: %s\n", err.Error())
		return
	}
	defer conn.Close()

	fmt.Println("WebSocket opened.")

	ticker := time.NewTicker(time.Microsecond * time.Duration(interval))
	defer ticker.Stop()

	openSocket := false

	eventCh := make(chan string)

	// Start a goroutine to handle events asynchronously
	go func() {
		for {
			_, event, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("WebSocket error: %s\n", err.Error())
				return
			}
			eventCh <- string(event)
		}
	}()

	for {
		select {
		case <-ticker.C:
			if openSocket {
				frame++
				frameData := images[frame%2]
				err := conn.WriteMessage(websocket.BinaryMessage, frameData)
				if err != nil {
					fmt.Printf("Error writing frame: %s\n", err.Error())
					return
				}
			}
		case event := <-eventCh:
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
