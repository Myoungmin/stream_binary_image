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

// WebSocket 업그레이드 과정은 클라이언트가 HTTP 연결을 WebSocket 프로토콜로 업그레이드하도록 요청
// ReadBufferSize 및 WriteBufferSize 필드는 들어오고 나가는 데이터의 버퍼 크기를 관리하여 WebSocket 통신 중 성능과 메모리 사용을 최적화하는 역할
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func socketHandler(w http.ResponseWriter, r *http.Request) {
	// HTTP 연결을 WebSocket 프로토콜로 업그레이드
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

	// Channel은 채널 연산자인 <- 을 통해 값을 주고 받을 수 있는 하나의 분리된 통로
	// Channel은 map과 slice처럼 사용하기 전에 생성
	// 전송과 수신은 다른 한 쪽이 준비될 때까지 block 상태
	// 명시적인 lock이나 조건 변수 없이 goroutine이 synchronous하게 작업될 수 있도록한다.
	eventCh := make(chan string)

	// 이벤트를 비동기적으로 처리하기 위해 goroutine 실행
	go func() {
		for {
			_, event, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("WebSocket error: %s\n", err.Error())
				return
			}
			// 채널 eventCh에 이벤트를 전송
			eventCh <- string(event)
		}
	}()

	for {
		// select: goroutine이 다중 커뮤니케이션 연산에서 대기할 수 있게 한다.
		// case들 중 하나가 실행될 때까지 block
		// 다수의 case가 준비되는 경우에는 select가 무작위로 하나를 선택
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
		// goroutine에서 비동기적으로 이벤트를 수신하였을 때
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
		// 다른 case들이 모두 준비되지 않았을 때 실행
		default:
			fmt.Printf("Select case not ready\n")
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
