package main

import (
	"csgo-parser-mongodb/app"
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

func main() {
	var pathToDemoFile, mongoUri, dbName string
	var gameStateFreq, positionsFreq int

	flag.StringVar(&pathToDemoFile,"dpath", "none", "path to the .dem file to parse")
	flag.StringVar(&mongoUri, "uri", "localhost:27017", "mongodb connection URI")
	flag.StringVar(&dbName, "dbname", "test", "database name for parsed data")
	flag.IntVar(&gameStateFreq, "gamestate", 30, "save a full game state every _ frames")
	flag.IntVar(&positionsFreq, "positions", 1, "save players' and grenades' positions every _ frames")

	flag.Parse()

	if pathToDemoFile == "none" {
		pathToDemoFile = "D:\\Games\\steamapps\\common\\Counter-Strike Global Offensive\\csgo\\replays\\match730_003349388754254037146_0607320178_181.dem"
	}

	mongoUri = "mongodb://" + mongoUri

	f, err := os.Open(pathToDemoFile)
	defer f.Close()
	checkError(err)

	client := connect_to_mongo(mongoUri, 2*time.Second)
	defer close_connection_to_mongo(client)

	application := app.NewApplication(f, client, dbName, clNames, false, gameStateFreq, positionsFreq)
	application.Init()
	application.Parse()
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
