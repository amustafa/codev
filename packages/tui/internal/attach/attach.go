package attach

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

const (
	frameControl = 0x00
	frameData    = 0x01
)

type controlMessage struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

// Run connects to a Tower terminal session via WebSocket and bridges
// the binary protocol (0x00 control / 0x01 data) to stdin/stdout.
// This lets an external terminal (like a Zellij pane) render the
// actual live PTY session managed by Tower.
func Run(towerURL, terminalID, workspace string) error {
	wsURL, err := buildWSURL(towerURL, terminalID, workspace)
	if err != nil {
		return err
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("websocket connect failed: %w", err)
	}
	defer conn.Close()

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to set raw terminal: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	sendResize(conn)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	go func() {
		for range sigCh {
			sendResize(conn)
		}
	}()

	done := make(chan struct{})

	// Server → stdout: read WebSocket frames, strip prefix, write raw bytes
	go func() {
		defer close(done)
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if len(msg) == 0 {
				continue
			}
			if msg[0] == frameData {
				os.Stdout.Write(msg[1:])
			}
		}
	}()

	// stdin → server: read raw bytes, prepend 0x01, send as binary frame
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				if err != io.EOF {
					fmt.Fprintf(os.Stderr, "stdin error: %v\r\n", err)
				}
				conn.Close()
				return
			}
			frame := make([]byte, 1+n)
			frame[0] = frameData
			copy(frame[1:], buf[:n])
			if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
				return
			}
		}
	}()

	<-done
	return nil
}

func buildWSURL(towerURL, terminalID, workspace string) (string, error) {
	u, err := url.Parse(towerURL)
	if err != nil {
		return "", err
	}

	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}

	if workspace != "" {
		encoded := base64.RawURLEncoding.EncodeToString([]byte(workspace))
		return fmt.Sprintf("%s://%s/workspace/%s/ws/terminal/%s",
			scheme, u.Host, encoded, terminalID), nil
	}
	return fmt.Sprintf("%s://%s/ws/terminal/%s", scheme, u.Host, terminalID), nil
}

func sendResize(conn *websocket.Conn) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return
	}
	msg := controlMessage{
		Type: "resize",
		Payload: map[string]any{
			"cols": w,
			"rows": h,
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	frame := make([]byte, 1+len(data))
	frame[0] = frameControl
	copy(frame[1:], data)
	conn.WriteMessage(websocket.BinaryMessage, frame)
}
