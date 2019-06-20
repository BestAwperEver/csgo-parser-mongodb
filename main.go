package main

import (
	"csgo-parser-mongodb/app"
	"fmt"
	"os"
	"flag"
	"time"
)

var clNames = map[app.ClIndex]string {
	app.ClEntities: "entities",
	app.ClEvents: "events",
	app.ClPlayers: "players",
	app.ClPositions: "players_positions",
	app.ClProjectiles: "grenades_positions",
	app.ClInfernos: "current_infernos",
	app.ClHeader: "header",
	app.ClGameState: "game_states",
	app.ClReplays: "replays",
}

func correctFramerate(x int) bool {
	return (x >= 16) && (x <= 128) && (x != 0) && ((x & (x - 1)) == 0)
}

func main() {
	var pathToDemoFile, mongoUri, dbName string
	var gameStateFreq, frameRate int

	flag.StringVar(&pathToDemoFile,"dpath", "none", "Path to the .dem file to parse.")
	flag.StringVar(&mongoUri, "uri", "localhost:27017", "MongoDB connection URI.")
	flag.StringVar(&dbName, "dbname", "test", "Database name for parsed data.")

	flag.IntVar(&frameRate,"framerate", 32, "Saves players' and grenades' positions with specified framerate. Possible values: 16, 32, 64 or 128. Cannot be greater than demo's original framerate.")
	flag.IntVar(&gameStateFreq, "gamestate", 32, "Saves a full game state every _ frames.")

	flag.Parse()

	if !correctFramerate(frameRate) {
		fmt.Printf("Incorrect requested framerate: %d. Must be 16, 32, 64 or 128.", frameRate)
	}

	if pathToDemoFile == "none" {
		//pathToDemoFile = "D:\\Games\\steamapps\\common\\Counter-Strike Global Offensive\\csgo\\replays\\match730_003221901158402490704_1843732364_900.dem"
		pathToDemoFile = "D:\\Games\\steamapps\\common\\Counter-Strike Global Offensive\\csgo\\replays\\match730_003349388754254037146_0607320178_181.dem"
	}

	mongoUri = "mongodb://" + mongoUri

	f, err := os.Open(pathToDemoFile)
	defer f.Close()
	checkError(err)

	client := connect_to_mongo(mongoUri, 2*time.Second)
	defer close_connection_to_mongo(client)

	application := app.NewApplication(f, client, dbName, clNames, false, gameStateFreq, frameRate)
	application.Init()
	t1 := time.Now()
	application.Parse()
	t2 := time.Now()
	diff := t2.Sub(t1)
	fmt.Printf("Parsing process took %f seconds.\n", diff.Seconds())
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
