package player

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/fatih/color"
	"github.com/gdamore/tcell/v2"

	"soundcloud_player/soundcloud"
	"soundcloud_player/ui"
	"soundcloud_player/utils"
)

type KeyEvent struct {
	Char rune
	Key  tcell.Key
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

var speakerMu sync.Mutex

func formatDuration(d time.Duration) string {
	min := int(d.Minutes())
	sec := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", min, sec)
}

func StartKeyboardListener(ctx context.Context, screen tcell.Screen, keyChan chan KeyEvent) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				ev := screen.PollEvent()
				switch tev := ev.(type) {
				case *tcell.EventKey:
					keyChan <- KeyEvent{
						Char: tev.Rune(),
						Key:  tev.Key(),
					}
				}
			}
		}
	}()
}

func PlayWithControls(tracks []soundcloud.Track, startIndex int, keyChan <-chan KeyEvent, screen tcell.Screen) {
	idx := startIndex

	for {
		if idx >= len(tracks) {
			fmt.Println("⛔ Треки закончились.")
			return
		}

		utils.ClearConsole()
		ui.PrintHeader()
		fmt.Printf("▶ Попытка воспроизведения: %s\n", tracks[idx].Title)

		resp, err := http.Get(tracks[idx].StreamURL)
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

		speakerMu.Lock()
		speaker.Clear()
		speaker.Close()
		if err := speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10)); err != nil {
			speakerMu.Unlock()
			log.Printf("Ошибка speaker.Init: %v\n", err)
			idx++
			continue
		}
		speakerMu.Unlock()

		ctrl := &beep.Ctrl{Streamer: streamer, Paused: false}
		done := make(chan bool)
		go func() {
			speakerMu.Lock()
			speaker.Play(beep.Seq(ctrl, beep.Callback(func() {
				done <- true
			})))
			speakerMu.Unlock()
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
				speakerMu.Lock()
				speaker.Clear()
				speakerMu.Unlock()
				goto nextTrack

			case keyEvent := <-keyChan:
				switch {
				case keyEvent.Char == 'q':
					fmt.Println("\n⏹ Выход...")
					speakerMu.Lock()
					speaker.Clear()
					speakerMu.Unlock()
					return

				case keyEvent.Char == 'n':
					fmt.Println("\n⏭ Следующий трек...")
					speakerMu.Lock()
					speaker.Clear()
					speakerMu.Unlock()
					goto nextTrack

				case keyEvent.Char == 'p' || (keyEvent.Key == tcell.KeyRune && keyEvent.Char == ' '):
					speakerMu.Lock()
					speaker.Lock()
					ctrl.Paused = !ctrl.Paused
					speaker.Unlock()
					speakerMu.Unlock()
					if ctrl.Paused {
						fmt.Println("\n⏸ Пауза")
					} else {
						fmt.Println("\n▶ Воспроизведение")
					}

				case keyEvent.Key == tcell.KeyLeft:
					speakerMu.Lock()
					speaker.Lock()
					step := format.SampleRate.N(time.Second * 30)
					newPos := streamer.Position() - step
					if newPos < 0 {
						newPos = 0
					}
					_ = streamer.Seek(newPos)
					speaker.Unlock()
					speakerMu.Unlock()
					fmt.Println("\n⏪ Назад на 30 секунд")

				case keyEvent.Key == tcell.KeyRight:
					speakerMu.Lock()
					speaker.Lock()
					step := format.SampleRate.N(time.Second * 30)
					newPos := streamer.Position() + step
					if newPos >= buffer.Len() {
						newPos = buffer.Len() - 1
					}
					_ = streamer.Seek(newPos)
					speaker.Unlock()
					speakerMu.Unlock()
					fmt.Println("\n⏩ Вперёд на 30 секунд")

				case keyEvent.Char == 's':
					speakerMu.Lock()
					speaker.Clear()
					speakerMu.Unlock()

					screen.Fini()

					fmt.Println("\n🔁 Новый поиск")
					fmt.Print("Введите новый запрос: ")
					var newQuery string
					fmt.Scanln(&newQuery)

					if err := screen.Init(); err != nil {
						log.Fatalf("Ошибка повторной инициализации экрана: %v", err)
					}

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
