package player

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/eiannone/keyboard"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/fatih/color"

	"soundcloud_player/soundcloud"
	"soundcloud_player/ui"
	"soundcloud_player/utils"
)

type KeyEvent struct {
	Char rune
	Key  keyboard.Key
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

func formatDuration(d time.Duration) string {
	min := int(d.Minutes())
	sec := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", min, sec)
}

func readKeys(ctx context.Context, keyChan chan<- KeyEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			char, key, err := keyboard.GetKey()
			if err != nil {
				log.Printf("Ошибка клавиатуры: %v", err)
				continue
			}
			keyChan <- KeyEvent{Char: char, Key: key}
		}
	}
}

func PlayWithControls(tracks []soundcloud.Track, startIndex int, keyChan <-chan KeyEvent) {
	idx := startIndex

	for {
		if idx >= len(tracks) {
			fmt.Println("⛔ Треки закончились.")
			return
		}

		utils.ClearConsole()
		ui.PrintHeader()
		fmt.Printf("▶ Попытка воспроизведения: %s\n", tracks[idx].Title)

		streamURL := tracks[idx].StreamURL
		resp, err := http.Get(streamURL)
		if err != nil {
			log.Printf("Ошибка загрузки трека: %v\n", err)
			idx++
			continue
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("Ошибка чтения потока: %v\n", err)
			idx++
			continue
		}

		streamerOriginal, format, err := mp3.Decode(nopCloser{bytes.NewReader(data)})
		if err != nil {
			log.Printf("Ошибка декодирования: %v\n", err)
			idx++
			continue
		}

		buffer := beep.NewBuffer(format)
		buffer.Append(streamerOriginal)
		streamerOriginal.Close()
		streamer := buffer.Streamer(0, buffer.Len())

		speaker.Clear()
		if err := speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10)); err != nil {
			log.Printf("Ошибка speaker.Init: %v\n", err)
			idx++
			continue
		}

		ctrl := &beep.Ctrl{Streamer: streamer, Paused: false}
		done := make(chan bool)
		go func() {
			speaker.Play(beep.Seq(ctrl, beep.Callback(func() {
				done <- true
			})))
		}()

		utils.ClearConsole()
		ui.PrintHeader()
		fmt.Println("🎵 Сейчас играет:", color.New(color.FgGreen, color.Bold).Sprint(tracks[idx].Title))
		ui.PrintControls()

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				fmt.Println("\n✅ Трек завершён")
				speaker.Clear()
				goto nextTrack

			case keyEvent := <-keyChan:
				switch {
				case keyEvent.Char == 'q':
					fmt.Println("\n⏹ Выход...")
					speaker.Clear()
					return

				case keyEvent.Char == 'n':
					fmt.Println("\n⏭ Следующий трек...")
					speaker.Clear()
					goto nextTrack

				case keyEvent.Char == 'p' || keyEvent.Key == keyboard.KeySpace:
					speaker.Lock()
					ctrl.Paused = !ctrl.Paused
					speaker.Unlock()
					if ctrl.Paused {
						fmt.Println("\n⏸ Пауза")
					} else {
						fmt.Println("\n▶ Воспроизведение")
					}

				case keyEvent.Key == keyboard.KeyArrowLeft:
					speaker.Lock()
					step := format.SampleRate.N(time.Second * 30)
					newPos := streamer.Position() - step
					if newPos < 0 {
						newPos = 0
					}
					_ = streamer.Seek(newPos)
					speaker.Unlock()
					fmt.Println("\n⏪ Назад на 30 секунд")

				case keyEvent.Key == keyboard.KeyArrowRight:
					speaker.Lock()
					step := format.SampleRate.N(time.Second * 30)
					newPos := streamer.Position() + step
					if newPos >= buffer.Len() {
						newPos = buffer.Len() - 1
					}
					_ = streamer.Seek(newPos)
					speaker.Unlock()
					fmt.Println("\n⏩ Вперёд на 30 секунд")

				case keyEvent.Char == 's':
					speaker.Clear()
					fmt.Println("\n🔁 Новый поиск")
					fmt.Print("Введите новый запрос: ")
					var newQuery string
					fmt.Scanln(&newQuery)

					clientID := os.Getenv("SOUNDCLOUD_CLIENT_ID")
					newTracks, err := soundcloud.GetTracks(newQuery, clientID)
					if err != nil || len(newTracks) == 0 {
						ui.PrintError("Не удалось найти треки")
						continue
					}

					tracks = newTracks
					idx = 0
					goto nextTrack
				}

			case <-ticker.C:
				pos := streamer.Position()
				length := buffer.Len()
				posDur := time.Duration(pos) * time.Second / time.Duration(format.SampleRate)
				lenDur := time.Duration(length) * time.Second / time.Duration(format.SampleRate)

				fmt.Printf("\r⏳ %s / %s ", formatDuration(posDur), formatDuration(lenDur))
			}
		}

	nextTrack:
		idx++
	}
}

func StartKeyboardListener(ctx context.Context, keyChan chan KeyEvent) {
	go readKeys(ctx, keyChan)
}
