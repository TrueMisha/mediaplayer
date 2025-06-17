package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/joho/godotenv"
	"math/rand"
	"os"
	"os/signal"
	"soundcloud_player/player"
	"soundcloud_player/soundcloud"
	"soundcloud_player/ui"
	"strings"
	"syscall"
	"time"
)

func main() {
	ui.PrintHeader()

	if err := godotenv.Load(); err != nil {
		ui.PrintError("Ошибка загрузки .env файла")
		return
	}

	clientID := os.Getenv("SOUNDCLOUD_CLIENT_ID")
	if clientID == "" {
		ui.PrintError("Переменная SOUNDCLOUD_CLIENT_ID не найдена в .env")
		return
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Введите запрос (или Enter для случайного): ")
	query, _ := reader.ReadString('\n')
	query = strings.TrimSpace(query)

	if query == "" {
		rand.Seed(time.Now().UnixNano())
		keywords := []string{"lofi", "hiphop", "ambient", "indie", "funk", "synthwave"}
		query = keywords[rand.Intn(len(keywords))]
		fmt.Println("🎲 Случайный запрос:", query)
	}

	tracks, err := soundcloud.GetTracks(query, clientID)
	if err != nil || len(tracks) == 0 {
		ui.PrintError("Не удалось получить треки")
		return
	}

	ui.PrintTracks(tracks)
	fmt.Print("\nВыбери трек (номер или Enter для случайного): ")
	choiceStr, _ := reader.ReadString('\n')
	choiceStr = strings.TrimSpace(choiceStr)

	var idx int
	if choiceStr == "" {
		rand.Seed(time.Now().UnixNano())
		idx = rand.Intn(len(tracks))
	} else {
		fmt.Sscanf(choiceStr, "%d", &idx)
		if idx < 1 || idx > len(tracks) {
			ui.PrintError("Неверный номер трека")
			return
		}
		idx--
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		ui.PrintError(fmt.Sprintf("Ошибка создания экрана: %v", err))
		return
	}
	if err := screen.Init(); err != nil {
		ui.PrintError(fmt.Sprintf("Ошибка инициализации экрана: %v", err))
		return
	}
	defer screen.Fini()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		screen.Fini()
		os.Exit(0)
	}()

	keyChan := make(chan player.KeyEvent)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	player.StartKeyboardListener(ctx, screen, keyChan)
	player.PlayWithControls(tracks, idx, keyChan, screen)
}
