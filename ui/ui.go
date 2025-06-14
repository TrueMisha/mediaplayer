package ui

import (
	"fmt"
	"os"

	"github.com/common-nighthawk/go-figure"
	"github.com/fatih/color"

	"soundcloud_player/soundcloud"
)

func PrintHeader() {
	figure.NewFigure("MediaPlayer", "", true).Print()
	color.Cyan("Coded by TrueMisha  tg@TrueMishaaa✨\n")
}

func PrintTracks(tracks []soundcloud.Track) {
	color.Yellow("\n🎵 Найденные треки:")
	for i, track := range tracks {
		fmt.Printf("%s %s\n",
			color.New(color.FgCyan).Sprintf("[%2d]", i+1),
			color.New(color.FgWhite, color.Bold).Sprintf(track.Title),
		)
	}
}

func PrintError(msg string) {
	color.New(color.FgRed).Fprintf(os.Stderr, "[-] %s\n", msg)
}

func PrintControls() {
	color.New(color.FgYellow).Print(`

🎛 Управление:
  [ ]⏸ Пауза / ▶ Воспроизвести
  [n] ⏭ Следующий трек
  [s] 🔁 Новый поиск
  [←] ⏪ Назад 30с
  [→] ⏩ Вперёд 30с
  [q] ⏹ Выход

`)
}
