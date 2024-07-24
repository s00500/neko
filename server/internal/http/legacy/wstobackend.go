package legacy

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"

	oldEvent "github.com/demodesk/neko/internal/http/legacy/event"
	oldMessage "github.com/demodesk/neko/internal/http/legacy/message"

	"github.com/demodesk/neko/internal/api/room"
	"github.com/demodesk/neko/internal/plugins/chat"
	"github.com/demodesk/neko/internal/plugins/filetransfer"
	"github.com/demodesk/neko/pkg/types"
	"github.com/demodesk/neko/pkg/types/event"
	"github.com/demodesk/neko/pkg/types/message"
)

func (s *session) wsToBackend(msg []byte) error {
	header := oldMessage.Message{}
	err := json.Unmarshal(msg, &header)
	if err != nil {
		return err
	}

	switch header.Event {
	// Signal Events
	case oldEvent.SIGNAL_OFFER:
		request := &oldMessage.SignalOffer{}
		err := json.Unmarshal(msg, request)
		if err != nil {
			return err
		}

		return s.toBackend(event.SIGNAL_OFFER, &message.SignalDescription{
			SDP: request.SDP,
		})

	case oldEvent.SIGNAL_ANSWER:
		request := &oldMessage.SignalAnswer{}
		err := json.Unmarshal(msg, request)
		if err != nil {
			return err
		}
´
		if request.DisplayName != "" {
			s.profile.Name = request.DisplayName

			err = s.apiReq(http.MethodPost, "/api/profile", s.profile, nil)
			if err != nil {
				return err
			}
		}

		return s.toBackend(event.SIGNAL_ANSWER, &message.SignalDescription{
			SDP: request.SDP,
		})

	case oldEvent.SIGNAL_CANDIDATE:
		request := &oldMessage.SignalCandidate{}
		err := json.Unmarshal(msg, request)
		if err != nil {
			return err
		}

		var candidate webrtc.ICECandidateInit
		err = json.Unmarshal([]byte(request.Data), &candidate)
		if err != nil {
			return err
		}

		return s.toBackend(event.SIGNAL_CANDIDATE, &message.SignalCandidate{
			ICECandidateInit: candidate,
		})

	// Control Events
	case oldEvent.CONTROL_RELEASE:
		return s.toBackend(event.CONTROL_RELEASE, nil)

	case oldEvent.CONTROL_REQUEST:
		return s.toBackend(event.CONTROL_REQUEST, nil)

	case oldEvent.CONTROL_GIVE:
		request := &oldMessage.Control{}
		err := json.Unmarshal(msg, request)
		if err != nil {
			return err
		}

		return s.apiReq(http.MethodPost, "/api/room/control/give/"+request.ID, nil, nil)

	case oldEvent.CONTROL_CLIPBOARD:
		request := &oldMessage.Clipboard{}
		err := json.Unmarshal(msg, request)
		if err != nil {
			return err
		}

		return s.toBackend(event.CLIPBOARD_SET, &message.ClipboardData{
			Text: request.Text,
		})

	case oldEvent.CONTROL_KEYBOARD:
		request := &oldMessage.Keyboard{}
		err := json.Unmarshal(msg, request)
		if err != nil {
			return err
		}

		if request.Layout != nil {
			err = s.toBackend(event.KEYBOARD_MAP, &message.KeyboardMap{
				KeyboardMap: types.KeyboardMap{
					Layout: *request.Layout,
				},
			})

			if err != nil {
				return err
			}
		}

		if request.CapsLock != nil || request.NumLock != nil || request.ScrollLock != nil {
			err = s.toBackend(event.KEYBOARD_MODIFIERS, &message.KeyboardModifiers{
				KeyboardModifiers: types.KeyboardModifiers{
					CapsLock: request.CapsLock,
					NumLock:  request.NumLock,
					// ScrollLock: request.ScrollLock, // ScrollLock is deprecated.
				},
			})

			if err != nil {
				return err
			}
		}

		return nil

	// Chat Events
	case oldEvent.CHAT_MESSAGE:
		request := &oldMessage.ChatReceive{}
		err := json.Unmarshal(msg, request)
		if err != nil {
			return err
		}

		return s.toBackend(chat.CHAT_MESSAGE, &chat.Content{
			Text: request.Content,
		})

	case oldEvent.CHAT_EMOTE:
		request := &oldMessage.EmoteReceive{}
		err := json.Unmarshal(msg, request)
		if err != nil {
			return err
		}

		// loopback emote
		msg, err := json.Marshal(&oldMessage.EmoteSend{
			Event: oldEvent.CHAT_EMOTE,
			ID:    s.id,
			Emote: request.Emote,
		})
		if err != nil {
			return err
		}

		// loopback emote
		err = s.connClient.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			return err
		}

		// broadcast emote to other users
		return s.toBackend(event.SEND_BROADCAST, &message.SendBroadcast{
			Sender:  s.id,
			Subject: "emote",
			Body:    request.Emote,
		})

	// File Transfer Events
	case oldEvent.FILETRANSFER_REFRESH:
		return s.toBackend(filetransfer.FILETRANSFER_UPDATE, nil)

	// Screen Events
	case oldEvent.SCREEN_RESOLUTION:
		// No WS equivalent, call HTTP API and return screen resolution.
		return fmt.Errorf("event not implemented: %s", header.Event)

	case oldEvent.SCREEN_CONFIGURATIONS:
		// No WS equivalent, call HTTP API and return screen configurations.
		return fmt.Errorf("event not implemented: %s", header.Event)

	case oldEvent.SCREEN_SET:
		request := &oldMessage.ScreenResolution{}
		err := json.Unmarshal(msg, request)
		if err != nil {
			return err
		}

		return s.toBackend(event.SCREEN_SET, &message.ScreenSize{
			ScreenSize: types.ScreenSize{
				Width:  request.Width,
				Height: request.Height,
				Rate:   request.Rate,
			},
		})

	// Broadcast Events
	case oldEvent.BROADCAST_CREATE:
		request := &oldMessage.BroadcastCreate{}
		err := json.Unmarshal(msg, request)
		if err != nil {
			return err
		}

		return s.apiReq(http.MethodPost, "/api/room/broadcast/start", room.BroadcastStatusPayload{
			URL:      request.URL,
			IsActive: true,
		}, nil)

	case oldEvent.BROADCAST_DESTROY:
		return s.apiReq(http.MethodPost, "/api/room/broadcast/stop", nil, nil)

	// Admin Events
	case oldEvent.ADMIN_LOCK:
		request := &oldMessage.AdminLock{}
		err := json.Unmarshal(msg, request)
		if err != nil {
			return err
		}

		data := map[string]any{}

		switch request.Resource {
		case "login":
			data["locked_logins"] = true
		case "control":
			data["locked_controls"] = true
		case "file_transfer":
			data["plugins"] = map[string]any{
				"filetransfer.enabled": false,
			}
		default:
			return fmt.Errorf("unknown resource: %s", request.Resource)
		}

		return s.apiReq(http.MethodPost, "/api/room/settings", data, nil)

	case oldEvent.ADMIN_UNLOCK:
		request := &oldMessage.AdminLock{}
		err := json.Unmarshal(msg, request)
		if err != nil {
			return err
		}

		data := map[string]any{}

		switch request.Resource {
		case "login":
			data["locked_logins"] = false
		case "control":
			data["locked_controls"] = false
		case "file_transfer":
			data["plugins"] = map[string]any{
				"filetransfer.enabled": true,
			}
		default:
			return fmt.Errorf("unknown resource: %s", request.Resource)
		}

		return s.apiReq(http.MethodPost, "/api/room/settings", data, nil)

	case oldEvent.ADMIN_CONTROL:
		return s.apiReq(http.MethodPost, "/api/room/control/take", nil, nil)

	case oldEvent.ADMIN_RELEASE:
		return s.apiReq(http.MethodPost, "/api/room/control/reset", nil, nil)

	case oldEvent.ADMIN_GIVE:
		request := &oldMessage.Admin{}
		err := json.Unmarshal(msg, request)
		if err != nil {
			return err
		}
		return s.apiReq(http.MethodPost, "/api/room/control/give/"+request.ID, nil, nil)

	case oldEvent.ADMIN_BAN:
		request := &oldMessage.Admin{}
		err := json.Unmarshal(msg, request)
		if err != nil {
			return err
		}

		// TODO: No WS equivalent, call HTTP API.
		return fmt.Errorf("event not implemented: %s", header.Event)

	case oldEvent.ADMIN_KICK:
		request := &oldMessage.Admin{}
		err := json.Unmarshal(msg, request)
		if err != nil {
			return err
		}

		// TODO: No WS equivalent, call HTTP API.
		return fmt.Errorf("event not implemented: %s", header.Event)

	case oldEvent.ADMIN_MUTE:
		request := &oldMessage.Admin{}
		err := json.Unmarshal(msg, request)
		if err != nil {
			return err
		}

		// TODO: No WS equivalent, call HTTP API.
		return fmt.Errorf("event not implemented: %s", header.Event)

	case oldEvent.ADMIN_UNMUTE:
		request := &oldMessage.Admin{}
		err := json.Unmarshal(msg, request)
		if err != nil {
			return err
		}

		// TODO: No WS equivalent, call HTTP API.
		return fmt.Errorf("event not implemented: %s", header.Event)

	default:
		return fmt.Errorf("unknown event type: %s", header.Event)
	}
}