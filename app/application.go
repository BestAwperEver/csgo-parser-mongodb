package app

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"io"
	"reflect"
	"time"

	dem "github.com/markus-wa/demoinfocs-golang"
	"github.com/markus-wa/demoinfocs-golang/common"
	"github.com/markus-wa/demoinfocs-golang/events"
)

type Application struct {
	client *mongo.Client
	dbName string

	collectionNames map[ClIndex]string
	//clEvents		string
	//clEntities		string
	//clPlayers		string
	//clHeader		string
	//clGameState		string
	//clPositions		string
	//clInfernos		string
	//clProjectiles	string
	//clReplays		string

	//bulk operations
	bulkInserts                 map[ClIndex][]mongo.WriteModel
	collectionsForBulkInserting []ClIndex

	collections map[ClIndex]*mongo.Collection

	reader io.Reader
	parser *dem.Parser

	currentProjectiles map[int]*common.GrenadeProjectile
	equipmentElements  map[int64]EquipmentElementStaticInfo
	playersLoaded      bool
	saveGameStateFrameDenominator int
	savePositionsFrameDenominator int

	implicitlyProcessedEvents map[EvType]bool

	// for calculating deltas
	savePositionsAsDeltas         bool
	playersLastPositions          map[int64]PlayerMovementInfo
}

// collections' indices
type ClIndex int
const (
	ClEvents		ClIndex = 0
	ClEntities		ClIndex = 1
	ClPlayers		ClIndex = 2
	ClPositions		ClIndex = 3
	ClHeader		ClIndex = 4
	ClGameState		ClIndex = 5
	ClInfernos		ClIndex = 6
	ClProjectiles	ClIndex = 7
	ClReplays		ClIndex = 8
)

func NewApplication(
	reader io.Reader,
	client *mongo.Client,
	dbName string,
	collectionNames map[ClIndex]string,
	savePositionsAsDeltas bool,
	gameStateFreq int,
	positionsFreq int) Application {
	return Application{
		reader:                reader,
		client:                client,
		dbName:                dbName,
		collectionNames:       collectionNames,
		savePositionsAsDeltas: savePositionsAsDeltas,
		saveGameStateFrameDenominator: gameStateFreq,
		savePositionsFrameDenominator: positionsFreq,
		//clEvents: collectionNames[ClEvents],
		//clEntities: collectionNames[ClEntities],
		//clPlayers: collectionNames[ClPlayers],
		//clHeader: collectionNames[ClHeader],
		//clPositions: collectionNames[ClPositions],
		//clInfernos: collectionNames[ClInfernos],
		//clProjectiles: collectionNames[ClProjectiles],
		//clGameState: collectionNames[ClGameState],
		//clReplays:  collectionNames[ClReplays],
	}
}

func (app *Application) Init() {
	err := app.client.Ping(context.TODO(), readpref.Primary())
	checkError(err)

	app.equipmentElements = make(map[int64]EquipmentElementStaticInfo)
	app.currentProjectiles = make(map[int]*common.GrenadeProjectile)
	app.collections = make(map[ClIndex]*mongo.Collection)

	app.collections[ClEvents] = app.client.Database(app.dbName).Collection(app.collectionNames[ClEvents])
	app.collections[ClPositions] = app.client.Database(app.dbName).Collection(app.collectionNames[ClPositions])
	app.collections[ClInfernos] = app.client.Database(app.dbName).Collection(app.collectionNames[ClInfernos])
	app.collections[ClProjectiles] = app.client.Database(app.dbName).Collection(app.collectionNames[ClProjectiles])
	app.collections[ClHeader] = app.client.Database(app.dbName).Collection(app.collectionNames[ClHeader])
	app.collections[ClPlayers] = app.client.Database(app.dbName).Collection(app.collectionNames[ClPlayers])
	app.collections[ClEntities] = app.client.Database(app.dbName).Collection(app.collectionNames[ClEntities])
	app.collections[ClGameState] = app.client.Database(app.dbName).Collection(app.collectionNames[ClGameState])
	app.collections[ClReplays] = app.client.Database("meta_info").Collection(app.collectionNames[ClReplays])

	app.collectionsForBulkInserting = []ClIndex {
		ClEvents,
		ClPositions,
		ClInfernos,
		ClProjectiles,
		ClPlayers,
		ClEntities,
		ClGameState,
	}
	app.bulkInserts = make(map[ClIndex][]mongo.WriteModel)

	app.parser = dem.NewParser(app.reader)

	app.playersLoaded = false
	app.playersLastPositions = make(map[int64]PlayerMovementInfo)

	// events that are getting processed without dedicated handlers
	app.implicitlyProcessedEvents = map[EvType]bool {
		Footstep: true,

		WeaponFire: true,
		PlayerHurt: true,

		BombDefuseStart:   true,
		BombDefuseAborted: true,
		BombDefused:       true,
		BombDropped:       true,
		BombExplode:       true,
		BombPickup:        true,
		BombPlantBegin:    true,
		BombPlanted:       true,

		GrenadeProjectileBounce:  true,
		GrenadeProjectileDestroy: true,

		HeExplode: true,

		SmokeStart:   true,
		SmokeExpired: true,

		FireGrenadeStart:   true,
		FireGrenadeExpired: true,

		DecoyStart:   true,
		DecoyExpired: true,

		ItemPickup: true,
		ItemDrop:   true,
		ItemEquip:  true,

		PlayerDisconnected: true,

		BotTakenOver: true,

		ScoreUpdated: true,

		TeamSideSwitch: true,
	}
}

// makes a map from event for persistent saving
func (app *Application) getMap(event interface{}) map[string]interface{} {
	resultMap := make(map[string]interface{})
	reflectedEvent := reflect.ValueOf(event)
	if reflectedEvent.Kind() != reflect.Struct {
		panic("getMap received a non-struct object")
	}
	if reflectedEvent.NumField() == 0 {
		return nil
	}
	for i := 0; i < reflectedEvent.NumField(); i++ {
		if field := reflectedEvent.Field(i); field.CanInterface() {
			switch field.Kind() {
			case reflect.Ptr:
				if field.IsNil() == false {
					switch field := reflect.Indirect(field); field.Type().Name() {
					//default:
					//	data[reflectedEvent.Type().Field(i).Name] = reflectedEvent.Field(i).Interface()
					case "Player":
						P := field.Interface().(common.Player)
						resultMap[reflectedEvent.Type().Field(i).Name] = P.SteamID
					case "GrenadeProjectile":
						GP := field.Interface().(common.GrenadeProjectile)
						if _, ok := app.equipmentElements[GP.UniqueID()]; !ok {
							app.equipmentElements[GP.UniqueID()] = NewEquipmentElementStaticInfo(GP)
						}
						resultMap[reflectedEvent.Type().Field(i).Name] = GP.UniqueID()
					case "Equipment":
						Equip := field.Interface().(common.Equipment)
						EI := NewEquipmentInfo(Equip)
						//EI := getMap(Equip)
						resultMap[reflectedEvent.Type().Field(i).Name] = EI
					case "TeamState":
						TS := field.Interface().(common.TeamState)
						resultMap[reflectedEvent.Type().Field(i).Name] = TeamStateInfo{
							TS.ID,
							TS.Score,
							TS.ClanName,
							TS.Flag,
						}
					case "Inferno":
						INF := field.Interface().(common.Inferno)
						resultMap[reflectedEvent.Type().Field(i).Name] = INF.UniqueID()
					case "BombEvent":
						BE := field.Interface().(events.BombEvent)
						resultMap["Player"] = BE.Player.SteamID
						resultMap["Site"] = BE.Site
					}
				} else {
					resultMap[reflectedEvent.Type().Field(i).Name] = -1
				}
			case reflect.Struct:
				resultMap[reflectedEvent.Type().Field(i).Name] = app.getMap(field.Interface())
			default:
				resultMap[reflectedEvent.Type().Field(i).Name] = field.Interface()
			}
		}
		//if field := reflectedEvent.Field(i); field.Kind() != reflect.Ptr && field.Kind() != reflect.Struct {
		//	if field.CanInterface() {
		//		resultMap[reflectedEvent.Type().Field(i).Name] = field.Interface()
		//	}
		//} else if field.CanInterface() {
		//
		//}
	}

	return resultMap
}

func (app *Application) manualHandlerRegistering() {

	app.parser.RegisterEventHandler(func(e events.GrenadeProjectileThrow) {
		app.currentProjectiles[e.Projectile.EntityID] = e.Projectile
		app.equipmentElements[e.Projectile.UniqueID()] = NewEquipmentElementStaticInfo(*e.Projectile)
	})

	app.parser.RegisterEventHandler(func(e events.GameHalfEnded) {
		var data = EventInfo{
			app.parser.CurrentFrame(),
			GameHalfEnded,
			app.getMap(e),
		}

		model := mongo.NewInsertOneModel().SetDocument(data)
		app.bulkInserts[ClEvents] = append(app.bulkInserts[ClEvents], model)

		//_, err := app.collections[ClEvents].InsertOne(context.TODO(), data)
		//checkError(err)
	})

	app.parser.RegisterEventHandler(func(e events.RoundEnd) {
		if app.parser.GameState().TeamTerrorists().Score == 15 && e.Winner == common.TeamTerrorists ||
			app.parser.GameState().TeamCounterTerrorists().Score == 15 && e.Winner == common.TeamCounterTerrorists ||
			app.parser.GameState().TotalRoundsPlayed() == 29 {
			//game is over (works for mm) TODO: get it working for other match types
			for _, v := range app.equipmentElements {

				model := mongo.NewInsertOneModel().SetDocument(v)
				app.bulkInserts[ClEntities] = append(app.bulkInserts[ClEntities], model)

				//_, err := app.collections[ClEntities].InsertOne(context.TODO(), v)
				//checkError(err)
			}
		}
	})

	//app.parser.RegisterEventHandler(func(e events.MatchStartedChanged) {
	//	if e.NewIsStarted {
	//		for _, player := range app.parser.GameState().Participants().Playing() {
	//
	//			model := mongo.NewInsertOneModel().SetDocument(NewPlayerStaticInfo(*player))
	//			app.bulkInserts[ClPlayers] = append(app.bulkInserts[ClPlayers], model)
	//
	//			//_, err := app.collections[ClPlayers].InsertOne(context.TODO(),
	//			//	NewPlayerStaticInfo(*player))
	//			//checkError(err)
	//		}
	//	}
	//})

	app.parser.RegisterEventHandler(func(e events.RankUpdate) {
		var data = EventInfo{
			app.parser.CurrentFrame(),
			RankUpdate,
			app.getMap(e),
		}

		model := mongo.NewInsertOneModel().SetDocument(data)
		app.bulkInserts[ClEvents] = append(app.bulkInserts[ClEvents], model)

		//_, err := app.collections[ClEvents].InsertOne(context.TODO(), data)
		//checkError(err)
	})

	app.parser.RegisterEventHandler(func(e events.FlashExplode) {
		if app.parser.GameState().IsWarmupPeriod() {
			return
		}

		type FlashExplodeInfo struct {
			UniqueID	int64		`bson:"UniqueID"`
			Position	Int16Vector3	`bson:"Position"`
		}

		var data = struct {
			FrameNumber int              `bson:"FrameNumber"`
			EventType   EvType       `bson:"EventType"`
			Data        FlashExplodeInfo `bson:"Data"`
		}{
			app.parser.CurrentFrame(),
			FlashExplode,
			FlashExplodeInfo{
				app.currentProjectiles[e.GrenadeEntityID].UniqueID(),
				NewInt16Vector3(e.Position),
			},
		}

		model := mongo.NewInsertOneModel().SetDocument(data)
		app.bulkInserts[ClEvents] = append(app.bulkInserts[ClEvents], model)

		//_, err := app.collections[ClEvents].InsertOne(context.TODO(), data)
		//checkError(err)
	})

	app.parser.RegisterEventHandler(func(e events.Kill) {
		if app.parser.GameState().IsWarmupPeriod() {
			return
		}

		var data = EventInfo{
			app.parser.CurrentFrame(),
			Kill,
			app.getMap(e),
		}

		model := mongo.NewInsertOneModel().SetDocument(data)
		app.bulkInserts[ClEvents] = append(app.bulkInserts[ClEvents], model)

		//_, err := app.collections[ClEvents].InsertOne(context.TODO(), data)
		//checkError(err)
	})

	app.parser.RegisterEventHandler(func(e events.PlayerFlashed) {
		type PlayerFlashedInfo struct {
			AttackerID		int64			`bson:"AttackerID"`
			PlayerID		int64			`bson:"PlayerID"`
			FlashDuration	time.Duration	`bson:"FlashDuration"`
		}

		var data = struct {
			FrameNumber int               `bson:"FrameNumber"`
			EventType   EvType        `bson:"EventType"`
			Data        PlayerFlashedInfo `bson:"Data"`
		}{
			app.parser.CurrentFrame(),
			PlayerFlashed,
			PlayerFlashedInfo{
				e.Attacker.SteamID,
				e.Player.SteamID,
				e.FlashDuration(),
			},
		}

		model := mongo.NewInsertOneModel().SetDocument(data)
		app.bulkInserts[ClEvents] = append(app.bulkInserts[ClEvents], model)

		//_, err := app.collections[ClEvents].InsertOne(context.TODO(), data)
		//		//checkError(err)
	})
}

func (app *Application) Parse() {

	header, err := app.parser.ParseHeader()
	checkError(err)
	headerMap := app.getMap(header)
	fmt.Println("Header:", headerMap)

	_, err =  app.collections[ClHeader].InsertOne(context.TODO(), headerMap)
	checkError(err)

	if app.dbName != "test" {
		_, err =  app.collections[ClReplays].InsertOne(context.TODO(), struct {
			DBname		string
			Timestamp	time.Time
		}{
			app.dbName,
			time.Now(),
		})
		checkError(err)
	}

	next, err := app.parser.ParseNextFrame()
	checkError(err)

	app.manualHandlerRegistering()

	// general handler function
	app.parser.RegisterEventHandler(func(e interface{}) {
		if app.parser.GameState().IsWarmupPeriod() {
			return
		}
		reflectedEvent := reflect.ValueOf(e)

		if evType := EvTypeIndex[reflectedEvent.Type().Name()]; app.implicitlyProcessedEvents[evType] {
			var data = EventInfo {
				app.parser.CurrentFrame(),
				evType,
				map[string]interface{}{},
			}

			data.Data = app.getMap(e)

			model := mongo.NewInsertOneModel().SetDocument(data)
			app.bulkInserts[ClEvents] = append(app.bulkInserts[ClEvents], model)

			//_, err :=  app.collections[ClEvents].InsertOne(context.TODO(), data)
			//checkError(err)
		}
	})

	for next {
		next, err = app.parser.ParseNextFrame()
		checkError(err)
		if app.parser.GameState().IsWarmupPeriod() || app.parser.GameState().IsMatchStarted() == false {
			continue
		}
		//if app.parser.GameState().TotalRoundsPlayed() < 2 {
		//	continue
		//}
		//if app.parser.GameState().TotalRoundsPlayed() > 3 {
		//	break
		//}

		//saving the whole game state
		if app.parser.CurrentFrame() % app.saveGameStateFrameDenominator == 0 {
		//if true {
			//saving the whole game state

			//saving general player infos
			if app.playersLoaded == false {
				allPlayersAreHumans := true
				for _, player := range app.parser.GameState().Participants().Playing() {
					if player.IsBot == true {
						allPlayersAreHumans = false
						break
					}
				}
				if allPlayersAreHumans {
					app.playersLoaded = true
					for _, player := range app.parser.GameState().Participants().Playing() {
						model := mongo.NewInsertOneModel().SetDocument(NewPlayerStaticInfo(*player))
						app.bulkInserts[ClPlayers] = append(app.bulkInserts[ClPlayers], model)
					}
				}
			}


			var data = GameStateInfo{
				app.parser.CurrentFrame(),
				make([]PlayerStateInfo, 0, len(app.parser.GameState().Participants().Playing())),
			}

			for _, p := range app.parser.GameState().Participants().Playing() {
				data.Players = append(data.Players, NewPlayerStateInfo(p))
			}

			model := mongo.NewInsertOneModel().SetDocument(data)
			app.bulkInserts[ClGameState] = append(app.bulkInserts[ClGameState], model)

			//_, err :=  app.collections[ClGameState].InsertOne(context.TODO(), data)
			//checkError(err)
		}
		if app.parser.CurrentFrame() % app.savePositionsFrameDenominator == 0 {

			playersPos := make([]PlayerMovementInfo, 0, len(app.parser.GameState().Participants().Playing()))
			grenadesPos := make([]GrenadePositionInfo, 0, len(app.parser.GameState().GrenadeProjectiles()))
			currentInfernos := make([]InfernoInfo, 0, len(app.parser.GameState().Infernos()))
			
			for _, v := range app.parser.GameState().Participants().Playing() {
				var data PlayerMovementInfo
				if app.savePositionsAsDeltas {
					data = app.calculateDelta(v)
				} else {
					data = PlayerMovementInfo{
						v.SteamID,
						NewInt16Vector3(v.Position),
						int16(v.ViewDirectionX),
						int16(v.ViewDirectionY),
					}
				}
				playersPos = append(playersPos, data)
			}
			for _, v := range app.parser.GameState().GrenadeProjectiles() {
				grenadesPos = append(grenadesPos, GrenadePositionInfo{
					v.UniqueID(),
					NewInt16Vector3(v.Position),
				})
			}
			for _, v := range app.parser.GameState().Infernos() {
				currentInfernos = append(currentInfernos, InfernoInfo{
					v.UniqueID(),
					v.Active().ConvexHull2D(),
				})
			}

			if len(playersPos) > 0 {
				data := FramePositions{
					app.parser.CurrentFrame(),
					playersPos,
				}

				model := mongo.NewInsertOneModel().SetDocument(data)
				app.bulkInserts[ClPositions] = append(app.bulkInserts[ClPositions], model)

				//_, err := app.collections[ClPositions].InsertOne(context.TODO(), data)
				//checkError(err)
			}

			if len(grenadesPos) > 0 {
				data := FrameProjectiles{
					app.parser.CurrentFrame(),
					grenadesPos,
				}

				model := mongo.NewInsertOneModel().SetDocument(data)
				app.bulkInserts[ClProjectiles] = append(app.bulkInserts[ClProjectiles], model)

				//_, err = app.collections[ClProjectiles].InsertOne(context.TODO(), data)
				//checkError(err)
			}

			if len(currentInfernos) > 0 {
				data := FrameInfernos{
					app.parser.CurrentFrame(),
					currentInfernos,
				}

				model := mongo.NewInsertOneModel().SetDocument(data)
				app.bulkInserts[ClInfernos] = append(app.bulkInserts[ClInfernos], model)

				//_, err = app.collections[ClInfernos].InsertOne(context.TODO(), data)
				//checkError(err)
			}
		}
	}

	for _, collectionIndex := range app.collectionsForBulkInserting {
		data := app.bulkInserts[collectionIndex]
		fmt.Printf("Length of %s: %d\n", app.collectionNames[collectionIndex], len(data))
		_, err := app.collections[collectionIndex].BulkWrite(context.TODO(), data)
		checkError(err)
	}

}

func (app *Application) calculateDelta(player *common.Player) PlayerMovementInfo {
	PMI := PlayerMovementInfo{
		SteamID: player.SteamID,
	}
	if _, ok := app.playersLastPositions[PMI.SteamID]; !ok {
		app.playersLastPositions[PMI.SteamID] = PlayerMovementInfo{
			SteamID:	player.SteamID,
			Position:	NewInt16Vector3(player.Position),
			ViewX:		int16(player.ViewDirectionX),
			ViewY:		int16(player.ViewDirectionY),
		}
		return app.playersLastPositions[PMI.SteamID]
	} else {
		PMI.Position = Int16Vector3{
			int16(player.Position.X) - app.playersLastPositions[player.SteamID].Position.X,
			int16(player.Position.Y) - app.playersLastPositions[player.SteamID].Position.Y,
			int16(player.Position.X) - app.playersLastPositions[player.SteamID].Position.Z,
		}
		PMI.ViewX = int16(player.ViewDirectionX) - app.playersLastPositions[player.SteamID].ViewX
		PMI.ViewY = int16(player.ViewDirectionY) - app.playersLastPositions[player.SteamID].ViewY
	}
	return PMI
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}