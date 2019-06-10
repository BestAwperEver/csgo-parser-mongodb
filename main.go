package main

import (
	"csgo-parser-mongodb/app"
	"os"
	"time"
)

var clNames = map[app.ClIndex]string {
	app.ClEntities: "entities",
	app.ClEvents: "events",
	app.ClPlayers: "players",
	app.ClFrames: "frames",
	app.ClHeader: "header",
	app.ClGameState: "game_states",
}

func main() {
	pathToDemoFile := "D:\\Games\\steamapps\\common\\Counter-Strike Global Offensive\\csgo\\replays\\match730_003349388754254037146_0607320178_181.dem"

	f, err := os.Open(pathToDemoFile)
	defer f.Close()
	checkError(err)

	dbName := "test"
	client := connect_to_mongo("mongodb://localhost:27017", 2*time.Second)
	defer close_connection_to_mongo(client)

	application := app.NewApplication(f, client, dbName, clNames)
	application.Init()
	application.Parse()
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
