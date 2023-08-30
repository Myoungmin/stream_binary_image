package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	fps      = 60
	interval = int(1000000 / fps)
)

var (
	width  = 1024
	height = 1024
	frame  = 0
	// [2][]byte{createImage(0), createImage(1)} 배열 리터럴 생성
	// 이를 참조하는 슬라이스 images
	images = [][]byte{createImage(0), createImage(1)}
)

func main() {
	port := 8080

	http.HandleFunc("/", socketHandler)

	fmt.Printf("Listening on port %d...\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

// WebSocket 업그레이드 과정은 클라이언트가 HTTP 연결을 WebSocket 프로토콜로 업그레이드하도록 요청
// ReadBufferSize 및 WriteBufferSize 필드는 들어오고 나가는 데이터의 버퍼 크기를 관리하여 WebSocket 통신 중 성능과 메모리 사용을 최적화하는 역할
var upgrader = websocket.Upgrader{
	ReadBufferSize:  0,
	WriteBufferSize: 0,
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

	// goroutine 은 Go 런타임에 의해 관리되는 경량 쓰레드
	// 이벤트를 비동기적으로 처리하기 위해 goroutine 실행, 익명함수를 사용한 goroutine
	// 웹 소켓으로부터 수신된 이벤트를 읽어서 eventCh 채널을 통해 메인 goroutine으로 전송
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
			// 이미지 크기 수신 시 설정하여 이미지 재생성
			if strings.Contains(event, ",") {
				sizes := strings.Split(event, ",")
				width, _ = strconv.Atoi(sizes[0])
				height, _ = strconv.Atoi(sizes[1])
				images = [][]byte{createImage(0), createImage(1)}
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
			// 다른 case들이 모두 준비되지 않았을 때 실행
			// default:
			//  fmt.Printf("Select case not ready\n")
		}
	}
}

func createImage(i int) []byte {
	// make 함수로 슬라이스 생성
	// 0으로 이루어진 배열을 할당, 그리고 해당 배열을 참조하는 슬라이스 반환
	image := make([]byte, width*height)

	// if i == 0 {
	//  return image
	// }

	// roic := width / 10
	// for y := 0; y < height; y++ {
	//  for x := 0; x < width; x++ {
	//      image[y*width+x] = byte((x / roic) * 25)
	//  }
	// }

	// createImage 호출 시 랜덤 픽셀 이미지 생성
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			image[y*width+x] = byte(rand.Intn(256))
		}
	}

	return image
}
