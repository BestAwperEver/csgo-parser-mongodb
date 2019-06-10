package app

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"io"
	"reflect"
	"time"

	"github.com/golang/geo/r3"

	dem "github.com/markus-wa/demoinfocs-golang"
	"github.com/markus-wa/demoinfocs-golang/common"
	"github.com/markus-wa/demoinfocs-golang/events"
)

type Application struct {
	client		*mongo.Client
	dbName		string
	
	clEvents	string
	clEntities	string
	clPlayers	string
	clFrames	string
	clHeader	string
	clGameState	string
	
	collections	map[ClIndex]*mongo.Collection
	
	reader		io.Reader
	parser		*dem.Parser

	currentProjectiles	map[int]*common.GrenadeProjectile
	equipmentElements	map[int64]EquipmentElementStaticInfo
	
	implicitlyProcessedEvents	map[int64]EquipmentElementStaticInfo
}

func NewApplication(reader io.Reader, client *mongo.Client, dbName string, collectionNames map[ClIndex]string) Application {
	return Application{
		reader: reader,
		client: client,
		dbName: dbName,
		clEvents: collectionNames[ClEvents],
		clEntities: collectionNames[ClEntities],
		clPlayers: collectionNames[ClPlayers],
		clFrames: collectionNames[ClFrames],
		clHeader: collectionNames[ClHeader],
		clGameState: collectionNames[ClGameState],
	}
}

type ClIndex int

const (
	ClEvents	ClIndex = 0
	ClEntities	ClIndex = 1
	ClPlayers	ClIndex = 2
	ClFrames	ClIndex = 3
	ClHeader	ClIndex = 4
	ClGameState ClIndex = 5
)

func (app *Application) Init() {
	err := app.client.Ping(context.TODO(), readpref.Primary())
	checkError(err)

	app.equipmentElements = make(map[int64]EquipmentElementStaticInfo)
	app.currentProjectiles = make(map[int]*common.GrenadeProjectile)
	app.collections = make(map[ClIndex]*mongo.Collection)

	app.collections[ClEvents] = app.client.Database(app.dbName).Collection(app.clEvents)
	app.collections[ClFrames] = app.client.Database(app.dbName).Collection(app.clFrames)
	app.collections[ClHeader] = app.client.Database(app.dbName).Collection(app.clHeader)
	app.collections[ClPlayers] = app.client.Database(app.dbName).Collection(app.clPlayers)
	app.collections[ClEntities] = app.client.Database(app.dbName).Collection(app.clEntities)
	app.collections[ClGameState] = app.client.Database(app.dbName).Collection(app.clGameState)

	app.parser = dem.NewParser(app.reader)
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
// events that are getting processed without dedicated handlers
var ImplicitlyProcessedEvents = map[EvType]bool{
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

		_, err := app.collections[ClEvents].InsertOne(context.TODO(), data)
		checkError(err)
	})

	app.parser.RegisterEventHandler(func(e events.RoundEnd) {
		if app.parser.GameState().TeamTerrorists().Score == 15 && e.Winner == common.TeamTerrorists ||
			app.parser.GameState().TeamCounterTerrorists().Score == 15 && e.Winner == common.TeamCounterTerrorists ||
			app.parser.GameState().TotalRoundsPlayed() == 29 {
			//game is over (works for mm)
			for _, v := range app.equipmentElements {
				_, err := app.collections[ClEntities].InsertOne(context.TODO(), v)
				checkError(err)
			}
		}
	})

	app.parser.RegisterEventHandler(func(e events.MatchStartedChanged) {
		if e.NewIsStarted {
			for _, player := range app.parser.GameState().Participants().Playing() {
				_, err := app.collections[ClPlayers].InsertOne(context.TODO(),
					NewPlayerStaticInfo(*player))
				checkError(err)
			}
		}
	})

	app.parser.RegisterEventHandler(func(e events.RankUpdate) {
		var data = EventInfo{
			app.parser.CurrentFrame(),
			RankUpdate,
			app.getMap(e),
		}

		_, err := app.collections[ClEvents].InsertOne(context.TODO(), data)
		checkError(err)
	})

	app.parser.RegisterEventHandler(func(e events.FlashExplode) {
		if app.parser.GameState().IsWarmupPeriod() {
			return
		}

		type FlashExplodeInfo struct {
			UniqueID	int64		`bson:"UniqueID"`
			Position	r3.Vector	`bson:"Position"`
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
				e.Position,
			},
		}

		_, err := app.collections[ClEvents].InsertOne(context.TODO(), data)
		checkError(err)
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

		_, err := app.collections[ClEvents].InsertOne(context.TODO(), data)
		checkError(err)
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

		_, err := app.collections[ClEvents].InsertOne(context.TODO(), data)
		checkError(err)
	})
}

func (app *Application) Parse() {

	header, err := app.parser.ParseHeader()
	checkError(err)
	headerMap := app.getMap(header)
	fmt.Println("Header:", headerMap)

	_, err =  app.collections[ClHeader].InsertOne(context.TODO(), headerMap)
	checkError(err)

	next, err := app.parser.ParseNextFrame()
	checkError(err)

	// general handler function
	app.parser.RegisterEventHandler(func(e interface{}) {
		if app.parser.GameState().IsWarmupPeriod() {
			return
		}
		reflectedEvent := reflect.ValueOf(e)

		if evType := EvTypeIndex[reflectedEvent.Type().Name()]; ImplicitlyProcessedEvents[evType] {
			var data = EventInfo {
				app.parser.CurrentFrame(),
				evType,
				map[string]interface{}{},
			}

			data.Data = app.getMap(e)

			_, err :=  app.collections[ClEvents].InsertOne(context.TODO(), data)
			checkError(err)
		}
	})

	var saveTicks = true

	for next {
		next, err = app.parser.ParseNextFrame()
		checkError(err)
		if app.parser.GameState().IsWarmupPeriod() || app.parser.GameState().IsMatchStarted() == false {
			continue
		}
		//if app.parser.GameState().TotalRoundsPlayed() > 3 {
		//	break
		//}
		if app.parser.CurrentFrame() % 30 == 0 {
			//saving the whole game state

			var data = GameStateInfo{
				app.parser.CurrentFrame(),
				make([]PlayerStateInfo, 0, len(app.parser.GameState().Participants().Playing())),
			}

			for _, p := range app.parser.GameState().Participants().Playing() {
				data.Players = append(data.Players, NewPlayerStateInfo(p))
			}

			_, err :=  app.collections[ClGameState].InsertOne(context.TODO(), data)
			checkError(err)
		}
		if saveTicks {
			playersPos := make([]PlayerMovementInfo, 0, len(app.parser.GameState().Participants().Playing()))
			grenadesPos := make([]GrenadePositionInfo, 0, len(app.parser.GameState().GrenadeProjectiles()))
			currentInfernos := make([]InfernoInfo, 0, len(app.parser.GameState().Infernos()))
			
			for _, v := range app.parser.GameState().Participants().Playing() {
				playersPos = append(playersPos, PlayerMovementInfo{
					v.SteamID,
					v.Position,
					v.ViewDirectionX,
					v.ViewDirectionY,
				})
			}
			for _, v := range app.parser.GameState().GrenadeProjectiles() {
				grenadesPos = append(grenadesPos, GrenadePositionInfo{
					v.UniqueID(),
					v.Position,
				})
			}
			for _, v := range app.parser.GameState().Infernos() {
				currentInfernos = append(currentInfernos, InfernoInfo{
					v.UniqueID(),
					v.Active().ConvexHull2D(),
				})
			}
			frameInfo := FramePositions{
				app.parser.CurrentFrame(),
				playersPos,
				grenadesPos,
				currentInfernos,
			}

			_, err :=  app.collections[ClFrames].InsertOne(context.TODO(), frameInfo)

			checkError(err)
		}
	}
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}