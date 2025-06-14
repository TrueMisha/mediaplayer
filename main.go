package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/eiannone/keyboard"
	"github.com/joho/godotenv"

	"soundcloud_player/player"
	"soundcloud_player/soundcloud"
	"soundcloud_player/ui"
)

func main() {
	ui.PrintHeader()
	err := godotenv.Load()
	if err != nil {
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

	if err := keyboard.Open(); err != nil {
		log.Fatalf("Ошибка открытия клавиатуры: %v", err)
	}
	defer keyboard.Close()
	keyChan := make(chan player.KeyEvent)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	player.StartKeyboardListener(ctx, keyChan)
	player.PlayWithControls(tracks, idx, keyChan)
}
