package legacy

import (
	"fmt"
	"log"
	"net/http"

	"github.com/demodesk/neko/pkg/types"
	"github.com/demodesk/neko/pkg/utils"
	"github.com/gorilla/websocket"
)

var (
	// DefaultUpgrader specifies the parameters for upgrading an HTTP
	// connection to a WebSocket connection.
	DefaultUpgrader = &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// DefaultDialer is a dialer with all fields set to the default zero values.
	DefaultDialer = websocket.DefaultDialer
)

type LegacyHandler struct {
}

func New() *LegacyHandler {
	// Init

	return &LegacyHandler{}
}

func (h *LegacyHandler) Route(r types.Router) {
	log.Println("legacy handler route")

	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) error {
		connBackend, _, err := DefaultDialer.Dial("ws://127.0.0.1:8080/api/ws?token="+r.URL.Query().Get("token"), nil)
		if err != nil {
			return utils.HttpError(http.StatusServiceUnavailable).
				WithInternalErr(err).
				Msg("couldn't dial to remote backend url")
		}
		defer connBackend.Close()

		connClient, err := DefaultUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return utils.HttpError(http.StatusInternalServerError).
				WithInternalErr(err).
				Msg("couldn't upgrade connection to websocket")
		}
		defer connClient.Close()

		errClient := make(chan error, 1)
		errBackend := make(chan error, 1)
		replicateWebsocketConn := func(dst, src *websocket.Conn, errc chan error, rewriteTextMessage func([]byte) ([]byte, error)) {
			for {
				msgType, msg, err := src.ReadMessage()
				if err != nil {
					m := websocket.FormatCloseMessage(websocket.CloseNormalClosure, fmt.Sprintf("%v", err))
					if e, ok := err.(*websocket.CloseError); ok {
						if e.Code != websocket.CloseNoStatusReceived {
							m = websocket.FormatCloseMessage(e.Code, e.Text)
						}
					}
					errc <- err
					dst.WriteMessage(websocket.CloseMessage, m)
					break
				}
				if msgType == websocket.TextMessage {
					msg, err = rewriteTextMessage(msg)
					if err != nil {
						log.Printf("websocketproxy: Error when rewriting message: %v", err)
						continue
					}
				}
				err = dst.WriteMessage(msgType, msg)
				if err != nil {
					errc <- err
					break
				}
			}
		}

		// client -> backend
		go replicateWebsocketConn(connClient, connBackend, errClient, h.wsToBackend)

		// backend -> client
		go replicateWebsocketConn(connBackend, connClient, errBackend, h.wsToClient)

		var message string
		select {
		case err = <-errClient:
			message = "websocketproxy: Error when copying from backend to client: %v"
		case err = <-errBackend:
			message = "websocketproxy: Error when copying from client to backend: %v"

		}

		if e, ok := err.(*websocket.CloseError); !ok || e.Code == websocket.CloseAbnormalClosure {
			log.Printf(message, err)
		}

		return nil
	})

	/*
		r.Get("/stats", func(w http.ResponseWriter, r *http.Request) error {
			password := r.URL.Query().Get("pwd")
			isAdmin, err := webSocketHandler.IsAdmin(password)
			if err != nil {
				return utils.HttpForbidden(err)
			}

			if !isAdmin {
				return utils.HttpUnauthorized().Msg("bad authorization")
			}

			w.Header().Set("Content-Type", "application/json")

			stats := webSocketHandler.Stats()
			return json.NewEncoder(w).Encode(stats)
		})

		r.Get("/screenshot.jpg", func(w http.ResponseWriter, r *http.Request) error {
			password := r.URL.Query().Get("pwd")
			isAdmin, err := webSocketHandler.IsAdmin(password)
			if err != nil {
				return utils.HttpForbidden(err)
			}

			if !isAdmin {
				return utils.HttpUnauthorized().Msg("bad authorization")
			}

			if webSocketHandler.IsLocked("login") {
				return utils.HttpError(http.StatusLocked).Msg("room is locked")
			}

			quality, err := strconv.Atoi(r.URL.Query().Get("quality"))
			if err != nil {
				quality = 90
			}

			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Content-Type", "image/jpeg")

			img := desktop.GetScreenshotImage()
			if err := jpeg.Encode(w, img, &jpeg.Options{Quality: quality}); err != nil {
				return utils.HttpInternalServerError().WithInternalErr(err)
			}

			return nil
		})

		// allow downloading and uploading files
		if webSocketHandler.FileTransferEnabled() {
			r.Get("/file", func(w http.ResponseWriter, r *http.Request) error {
				return nil
			})

			r.Post("/file", func(w http.ResponseWriter, r *http.Request) error {
				return nil
			})
		}
	*/

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) error {
		_, err := w.Write([]byte("true"))
		return err
	})
}
