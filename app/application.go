package app

import (
	"context"
	"csgo-parser-mongodb/util/elias"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"io"
	"math"
	"reflect"
	"time"

	dem "github.com/markus-wa/demoinfocs-golang"
	"github.com/markus-wa/demoinfocs-golang/common"
	"github.com/markus-wa/demoinfocs-golang/events"
)

var dbgPrint bool

func init() {
	dbgPrint = false
}

type Application struct {
	client *mongo.Client
	dbName string

	collectionNames map[ClIndex]string

	//bulk operations
	bulkInserts map[ClIndex][]mongo.WriteModel

	collectionsForBulkInserting           []ClIndex
	collectionsForBulkInsertingEveryRound []ClIndex

	collections map[ClIndex]*mongo.Collection

	reader io.Reader
	parser *dem.Parser

	currentProjectiles            map[int64]GrenadeProjectileWithStartFrame
	equipmentElements             map[int64]EquipmentElementStaticInfo
	playersLoaded                 bool
	saveGameStateFrameDenominator int
	savePositionsFrameDenominator int
	frameRate                     int
	originalFramerate             int

	implicitlyProcessedEvents map[EvType]bool

	// for calculating deltas
	savePositionsAsDeltas bool
	playersLastPositions  map[int64]PlayerMovementInfo
	eliasEncodeDeltas     bool

	grenadesPositionsEncoded  []GrenadePositionInfoEncoded
	playersPositionsInRound   map[int64]*PlayerMovement // had to store pointers because go doesn't allow struct mutation when stored in a map
	roundNumber               int
	playerMovementEncodedData RoundMovement

	TerroristsAlive int
	CTsAlive        int

	playersStats []map[int64]*PlayerRoundStats // [round number][steam id]

	clutchPlayer   *PlayerRoundStats
	//clutchOpposite *PlayerRoundStats
	clutchEnemies  []*PlayerRoundStats
	clutchTeam     common.Team
	//clutchVs       int

	gameStarted bool
	openKill    bool // whether there was an open kill in the current round
	roundEnded  bool

	savedFrameNumber int
}

func (app *Application) clearPlayersInfo() {
	app.TerroristsAlive = 5
	app.CTsAlive = 5
	app.openKill = false
	app.clutchTeam = common.TeamUnassigned
	app.clutchPlayer = nil
	//app.clutchOpposite = nil
	//app.clutchVs = 0
	app.clutchEnemies = make([]*PlayerRoundStats, 0, 5)
}

// collections' indices
type ClIndex int
const (
	ClEvents	ClIndex = iota
	ClEntities
	ClPlayers
	ClPositions
	ClHeader
	ClGameState
	ClInfernos
	ClProjectiles
	ClReplays
)

const MAX_ROUNDS = 30

func NewApplication(
	reader io.Reader,
	client *mongo.Client,
	dbName string,
	collectionNames map[ClIndex]string,
	eliasEncoding bool,
	gameStateFreq int,
	frameRate int) Application {
	return Application {
		reader:							reader,
		client:							client,
		dbName:							dbName,
		collectionNames:				collectionNames,
		savePositionsAsDeltas:        	false, // doesn't help at all
		saveGameStateFrameDenominator:	gameStateFreq,
		frameRate:                    	frameRate,
		eliasEncodeDeltas:				eliasEncoding,
	}
}

func (app *Application) Init() {
	err := app.client.Ping(context.TODO(), readpref.Primary())
	checkError(err)

	app.equipmentElements = make(map[int64]EquipmentElementStaticInfo)
	//app.currentProjectiles = make(map[int]*common.GrenadeProjectile)
	app.currentProjectiles = make(map[int64]GrenadeProjectileWithStartFrame)
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
	app.collectionsForBulkInsertingEveryRound = []ClIndex {
		ClEvents,
		ClPositions,
		ClInfernos,
		ClProjectiles,
		ClGameState,
	}
	app.bulkInserts = make(map[ClIndex][]mongo.WriteModel)

	app.parser = dem.NewParser(app.reader)

	app.playersLoaded = false
	app.playersLastPositions = make(map[int64]PlayerMovementInfo)
	app.playersPositionsInRound = make(map[int64]*PlayerMovement)
	app.playerMovementEncodedData = RoundMovement{
		0,
		make([]PlayerMovementInfoEncoded, 0, 20),
	}

	// events that are getting processed without dedicated handlers
	app.implicitlyProcessedEvents = map[EvType]bool {
		Footstep: true,

		WeaponFire: true,
		PlayerHurt: true,
		PlayerJump: true,

		PlayerDisconnected: true, // also manual handling

		BombDefuseStart:   true,
		BombDefuseAborted: true,
		BombDefused:       true,
		BombDropped:       true,
		BombExplode:       true,
		BombPickup:        true,
		BombPlantBegin:    true,
		BombPlanted:       true,

		GrenadeProjectileBounce:  true,
		GrenadeProjectileDestroy: true,	// also manual handling
		GrenadeProjectileThrow: true,	// also manual handling

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

		BotTakenOver: true,

		ScoreUpdated: true,

		RoundFreezetimeEnd: true,
		RoundMVPReason: true,
		RoundMVPAnnouncement: true,

		TeamSideSwitch: true,
	}

	app.gameStarted = false
	app.roundEnded = true

	app.playersStats = make([]map[int64]*PlayerRoundStats, MAX_ROUNDS)

	app.clearPlayersInfo()
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
					switch fieldValue := reflect.Indirect(field); fieldValue.Type().Name() {
					//default:
					//	data[reflectedEvent.Type().Field(i).Name] = reflectedEvent.Field(i).Interface()
					case "Player":
						P := fieldValue.Interface().(common.Player)
						resultMap[reflectedEvent.Type().Field(i).Name] = P.SteamID
					case "GrenadeProjectile":
						GP := field.Interface().(*common.GrenadeProjectile)
						if _, ok := app.equipmentElements[GP.UniqueID()]; !ok {
							app.equipmentElements[GP.UniqueID()] = NewEquipmentElementStaticInfo(GP)
						}
						resultMap[reflectedEvent.Type().Field(i).Name] = GP.UniqueID()
					case "Equipment":
						Equip := fieldValue.Interface().(common.Equipment)
						EI := NewEquipmentInfo(Equip)
						//EI := getMap(Equip)
						resultMap[reflectedEvent.Type().Field(i).Name] = EI
					case "TeamState":
						TS := fieldValue.Interface().(common.TeamState)
						resultMap[reflectedEvent.Type().Field(i).Name] = TeamStateInfo{
							TS.ID,
							TS.Score,
							TS.ClanName,
							TS.Flag,
						}
					case "Inferno":
						INF := fieldValue.Interface().(common.Inferno)
						resultMap[reflectedEvent.Type().Field(i).Name] = INF.UniqueID()
					case "BombEvent":
						BE := fieldValue.Interface().(events.BombEvent)
						resultMap["Player"] = BE.Player.SteamID
						resultMap["Site"] = BE.Site
					}
				} else {
					resultMap[reflectedEvent.Type().Field(i).Name] = -1
				}
			case reflect.Struct:
				resultMap[reflectedEvent.Type().Field(i).Name] = app.getMap(field.Interface())
			default:
				Name := reflectedEvent.Type().Field(i).Name
				// there is no consistent way of knowing UniqueID by EntityID in this particular case
				// so let's assume we don't need it :P
				// in reality, we don't really care about grenade's ID because we just need to know that
				// particular grenade event happened in some position
				// all the rest goes to knowing that projectile has been removed from the game
				// and we actually have another event for that, namely GrenadeProjectileDestroy
				if Name != "GrenadeEntityID" {
					resultMap[Name] = field.Interface()
				}
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

func (app *Application) registerHandlersForStats() {

	app.parser.RegisterEventHandler(func(e events.MatchStart) {
		app.gameStarted = true
	})

	//app.parser.RegisterEventHandler(func(e events.PlayerConnect) {
	//	//Player := app.getPlayerStats(e.Player.SteamID)
	//	//if _, ok := app.playersStats[app.roundNumber][e.Player.SteamID]; !ok {
	//	//	app.playersStats[app.roundNumber][e.Player.SteamID] = NewPlayerStats(e.Player.SteamID)
	//	//}
	//})

	app.parser.RegisterEventHandler(func(e events.RoundStart) {
		app.clearPlayersInfo()
		app.roundEnded = false
	})

	app.parser.RegisterEventHandler(func(e events.Kill) {
		if app.gameStarted == false {
			return
		}
		if e.Killer == nil {
			return
		}
		PS := app.getPlayerStats(e.Killer)
		PS.kills++
		if e.IsHeadshot {
			PS.headshots++
		}
		if app.openKill == false {
			PS.openKill = true
			app.openKill = true
		}

		if app.clutchTeam == common.TeamUnassigned {
			aliveTerrorists := app.AliveMembers(common.TeamTerrorists)
			aliveCounterTerrorists := app.AliveMembers(common.TeamCounterTerrorists)
			if len(aliveCounterTerrorists) == 1 {
				app.clutchTeam = common.TeamCounterTerrorists
				app.clutchPlayer = app.getPlayerStats(aliveCounterTerrorists[0])
				for _, p := range aliveTerrorists {
					app.clutchEnemies = append(app.clutchEnemies, app.getPlayerStats(p))
				}
			} else if len(aliveTerrorists) == 1 {
				app.clutchTeam = common.TeamTerrorists
				app.clutchPlayer = app.getPlayerStats(aliveTerrorists[0])
				for _, p := range aliveCounterTerrorists {
					app.clutchEnemies = append(app.clutchEnemies, app.getPlayerStats(p))
				}
			}
		}
	})

	app.parser.RegisterEventHandler(func(e events.PlayerHurt) {
		if app.gameStarted == false {
			return
		}
		if e.Attacker == nil {
			return
		}
		Attacker := app.getPlayerStats(e.Attacker)
		if e.Player.Team == e.Attacker.Team {
			Attacker.teamDamage += e.HealthDamage
		} else {
			Attacker.damage += e.HealthDamage
			if e.Weapon.Weapon == common.EqHE {
				Attacker.heDamage += e.HealthDamage
			}
		}
	})

	app.parser.RegisterEventHandler(func(e events.WeaponFire) {
		if app.gameStarted == false {
			return
		}
		Player := app.getPlayerStats(e.Shooter)
		Player.weaponFires[e.Weapon.Weapon]++
	})

	app.parser.RegisterEventHandler(func(e events.PlayerFlashed) {
		if app.gameStarted == false {
			return
		}
		Attacker := app.getPlayerStats(e.Player)
		if e.Attacker == e.Player {
			Attacker.selfFlash += e.FlashDuration()
		}
		if e.Attacker.Team == e.Player.Team {
			Attacker.selfFlash += e.FlashDuration()
		} else {
			Attacker.enemyFlash += e.FlashDuration()
			if 2 * e.FlashDuration() > time.Second {
				Attacker.enemyFlashed++
			}
		}
	})

	app.parser.RegisterEventHandler(func(e events.RoundEnd) {
		if app.gameStarted == false {
			return
		}
		// handle current clutch
		if app.clutchTeam != common.TeamUnassigned {
			if e.Winner == app.clutchTeam {
				app.clutchPlayer.clutch[len(app.clutchEnemies)-1] = true
				for _, p := range app.clutchEnemies {
					if p != nil {
						p.clLoose = true
					}
				}
				dbgLog(fmt.Sprintf("%s won a clutch against %d enemies", app.clutchPlayer.Name, len(app.clutchEnemies)))
			} else if len(app.clutchEnemies) == 1 {
				app.clutchPlayer.clLoose = true
				app.clutchEnemies[0].clutch[0] = true
				dbgLog(fmt.Sprintf("%s won a clutch against %s", app.clutchEnemies[0].Name, app.clutchPlayer.Name))
			} else {
				app.clutchEnemies[0].clLoose = true
				dbgLog(fmt.Sprintf("%s lost a clutch against %d enemies", app.clutchPlayer.Name, len(app.clutchEnemies)))
			}
		}
	})
}

func (app *Application) manualHandlerRegistering() {

	app.registerHandlersForStats()

	app.parser.RegisterEventHandler(func(e events.ItemPickup) {
		if _, ok := app.equipmentElements[e.Weapon.UniqueID()]; !ok {
			app.equipmentElements[e.Weapon.UniqueID()] = NewEquipmentElementStaticInfo(e.Weapon)
		}
	})

	app.parser.RegisterEventHandler(func(e events.GrenadeProjectileThrow) {
		app.currentProjectiles[e.Projectile.UniqueID()] = GrenadeProjectileWithStartFrame {
			app.savedFrameNumber,
			e.Projectile,
		}
		app.equipmentElements[e.Projectile.UniqueID()] = NewEquipmentElementStaticInfo(e.Projectile)
	})

	app.parser.RegisterEventHandler(func(e events.GameHalfEnded) {
		var data = EventInfo{
			app.savedFrameNumber,
			GameHalfEnded,
			app.getMap(e),
		}

		model := mongo.NewInsertOneModel().SetDocument(data)
		app.bulkInserts[ClEvents] = append(app.bulkInserts[ClEvents], model)

		//_, err := app.collections[ClEvents].InsertOne(context.TODO(), data)
		//checkError(err)
	})

	app.parser.RegisterEventHandler(func(e events.RoundEnd) {
		//if app.parser.GameState().TeamTerrorists().Score == 15 && e.Winner == common.TeamTerrorists ||
		//	app.parser.GameState().TeamCounterTerrorists().Score == 15 && e.Winner == common.TeamCounterTerrorists ||
		//	app.parser.GameState().TotalRoundsPlayed() == 29 {
		//	//game is over (works for mm) TODO: get it working for other match types
		//	for _, v := range app.equipmentElements {
		//
		//		model := mongo.NewInsertOneModel().SetDocument(v)
		//		app.bulkInserts[ClEntities] = append(app.bulkInserts[ClEntities], model)
		//
		//		//_, err := app.collections[ClEntities].InsertOne(context.TODO(), v)
		//		//checkError(err)
		//	}
		//}
		fmt.Printf("%d\n", app.roundNumber)
	})

	app.parser.RegisterEventHandler(func(e events.RoundStart){
		if app.eliasEncodeDeltas {
			app.playerMovementEncodedData.RoundNumber = app.roundNumber

			for k, v := range app.playersPositionsInRound {
				if v.StartFrame == 0 {
					continue // no movement
				}
				PMIE := app.encodePlayerMovement(k, v, true)
				app.playerMovementEncodedData.PlayerMovements = append(app.playerMovementEncodedData.PlayerMovements, PMIE)
			}

			if len(app.playerMovementEncodedData.PlayerMovements) > 0 {
				model := mongo.NewInsertOneModel().SetDocument(app.playerMovementEncodedData)
				app.bulkInserts[ClPositions] = append(app.bulkInserts[ClPositions], model)
			}

			app.playerMovementEncodedData.PlayerMovements = app.playerMovementEncodedData.PlayerMovements[:0]
		}
		app.saveDataToMongo()
		app.roundNumber++
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

	if app.eliasEncodeDeltas {

		app.parser.RegisterEventHandler(func(e events.GrenadeProjectileDestroy) {
			X := make([]int, len(e.Projectile.Trajectory))
			Y := make([]int, len(e.Projectile.Trajectory))
			Z := make([]int, len(e.Projectile.Trajectory))
			for i, v := range e.Projectile.Trajectory {
				X[i] = int(v.X)
				Y[i] = int(v.Y)
				Z[i] = int(v.Z)
			}
			data := GrenadePositionInfoEncoded {
				app.currentProjectiles[e.Projectile.UniqueID()].StartFrame,
				app.savedFrameNumber,
				e.Projectile.UniqueID(),
				elias.EliasGammaNegative(elias.ArrayToDeltas(X)...),
				elias.EliasGammaNegative(elias.ArrayToDeltas(Y)...),
				elias.EliasGammaNegative(elias.ArrayToDeltas(Z)...),
			}
			model := mongo.NewInsertOneModel().SetDocument(data)
			app.bulkInserts[ClProjectiles] = append(app.bulkInserts[ClProjectiles], model)
		})

		app.parser.RegisterEventHandler(func(e events.PlayerDisconnected) {
			// player has disconnected and his movement wasn't reset
			if PM, ok := app.playersPositionsInRound[e.Player.SteamID]; ok && PM.StartFrame != 0 {
				PM.EndFrame = app.savedFrameNumber
				app.playerMovementEncodedData.PlayerMovements = append(app.playerMovementEncodedData.PlayerMovements, app.encodePlayerMovement(e.Player.SteamID, PM, true))
			}
		})

		app.parser.RegisterEventHandler(func(e events.Kill) {
			// player was killed and his movement wasn't reset
			if PM, ok := app.playersPositionsInRound[e.Victim.SteamID]; ok && PM.StartFrame != 0 {
				PM.EndFrame = app.savedFrameNumber
				app.playerMovementEncodedData.PlayerMovements = append(app.playerMovementEncodedData.PlayerMovements, app.encodePlayerMovement(e.Victim.SteamID, PM, true))
			}
		})
	}

	app.parser.RegisterEventHandler(func(e events.RankUpdate) {
		var data = EventInfo{
			app.savedFrameNumber,
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

		var data = struct {
			FrameNumber int					`bson:"FrameNumber"`
			EventType   EvType				`bson:"EventType"`
			Data        FlashExplodeInfo	`bson:"Data"`
		}{
			app.savedFrameNumber,
			FlashExplode,
			FlashExplodeInfo{
				//app.currentProjectiles[e.GrenadeEntityID].UniqueID(), // doesn't matter, it's projectile ID, not item's
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

		var data = EventInfo {
			app.savedFrameNumber,
			Kill,
			app.getMap(e),
		}

		if app.eliasEncodeDeltas {
			if PM, ok := app.playersPositionsInRound[e.Victim.SteamID]; ok && PM.EndFrame == 0 {
				PM.EndFrame = app.savedFrameNumber
			} else if !ok { // wtf? kill event with non-existed victim? let's panic!
				panic("Kill event with non-existing victim")
			}
		}

		model := mongo.NewInsertOneModel().SetDocument(data)
		app.bulkInserts[ClEvents] = append(app.bulkInserts[ClEvents], model)

		//_, err := app.collections[ClEvents].InsertOne(context.TODO(), data)
		//checkError(err)
	})

	app.parser.RegisterEventHandler(func(e events.PlayerFlashed) {

		var data = struct {
			FrameNumber int					`bson:"FrameNumber"`
			EventType   EvType				`bson:"EventType"`
			Data        PlayerFlashedInfo	`bson:"Data"`
		}{
			app.savedFrameNumber,
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
	app.originalFramerate = int(math.Round(float64(header.PlaybackFrames) / header.PlaybackTime.Seconds() / 16) * 16)
	fmt.Printf("Original demo framerate: %d frames per second.\n", app.originalFramerate)
	if app.originalFramerate < app.frameRate {
		app.savePositionsFrameDenominator = 1
		fmt.Printf("Requested framerate (%d) is greater than original. Setting framerate to %d.\n", app.frameRate, app.originalFramerate)
	} else {
		app.savePositionsFrameDenominator = app.originalFramerate / app.frameRate
		fmt.Printf("Saving players' and grenades' positions every %d frame(s).\n", app.savePositionsFrameDenominator)
	}


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
				app.savedFrameNumber,
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

	fmt.Println("Parsing started")

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
		if app.parser.CurrentFrame() % (app.saveGameStateFrameDenominator * app.savePositionsFrameDenominator) == 0 {
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
				app.savedFrameNumber,
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

			app.savedFrameNumber++

			playersPos := make([]PlayerMovementInfo, 0, len(app.parser.GameState().Participants().Playing()))
			grenadesPos := make([]GrenadePositionInfo, 0, len(app.parser.GameState().GrenadeProjectiles()))
			currentInfernos := make([]InfernoInfo, 0, len(app.parser.GameState().Infernos()))

			for _, v := range app.parser.GameState().Participants().Playing() {
				if v.IsAlive() { // saving only positions of alive players
					if !app.eliasEncodeDeltas {
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
					} else {
						PM, ok := app.playersPositionsInRound[v.SteamID]
						if !ok { // player didn't exist; first round or new spawned bot?
							app.playersPositionsInRound[v.SteamID] = &PlayerMovement{}
							PM = app.playersPositionsInRound[v.SteamID]
							PM.StartFrame = app.savedFrameNumber
						}
						if PM.StartFrame == 0 {
							// probably reconnected
							PM.StartFrame = app.savedFrameNumber
						}
						PM.PositionX = append(PM.PositionX, int(v.Position.X))
						PM.PositionY = append(PM.PositionY, int(v.Position.Y))
						PM.PositionZ = append(PM.PositionZ, int(v.Position.Z))
						PM.ViewX = append(PM.ViewX, int(v.ViewDirectionX))
						PM.ViewY = append(PM.ViewY, int(v.ViewDirectionY))
					}
				}
			}

			if !app.eliasEncodeDeltas {
				for _, v := range app.parser.GameState().GrenadeProjectiles() {
					grenadesPos = append(grenadesPos, GrenadePositionInfo{
						v.UniqueID(),
						NewInt16Vector3(v.Position),
					})
				}
			}

			for _, v := range app.parser.GameState().Infernos() {
				currentInfernos = append(currentInfernos, InfernoInfo{
					v.UniqueID(),
					v.Active().ConvexHull2D(),
				})
			}

			if !app.eliasEncodeDeltas {
				if len(playersPos) > 0 {
					data := FramePositions{
						app.savedFrameNumber,
						playersPos,
					}

					model := mongo.NewInsertOneModel().SetDocument(data)
					app.bulkInserts[ClPositions] = append(app.bulkInserts[ClPositions], model)

					//_, err := app.collections[ClPositions].InsertOne(context.TODO(), data)
					//checkError(err)
				}
			}

			if !app.eliasEncodeDeltas {
				if len(grenadesPos) > 0 {
					data := FrameProjectiles{
						app.savedFrameNumber,
						grenadesPos,
					}

					model := mongo.NewInsertOneModel().SetDocument(data)
					app.bulkInserts[ClProjectiles] = append(app.bulkInserts[ClProjectiles], model)

					//_, err = app.collections[ClProjectiles].InsertOne(context.TODO(), data)
					//checkError(err)
				}
			}

			if len(currentInfernos) > 0 {
				data := FrameInfernos{
					app.savedFrameNumber,
					currentInfernos,
				}

				model := mongo.NewInsertOneModel().SetDocument(data)
				app.bulkInserts[ClInfernos] = append(app.bulkInserts[ClInfernos], model)

				//_, err = app.collections[ClInfernos].InsertOne(context.TODO(), data)
				//checkError(err)
			}
		}
	}

	fmt.Println("Parsing ended")

	for _, v := range app.equipmentElements {

		model := mongo.NewInsertOneModel().SetDocument(v)
		app.bulkInserts[ClEntities] = append(app.bulkInserts[ClEntities], model)

		//_, err := app.collections[ClEntities].InsertOne(context.TODO(), v)
		//checkError(err)
	}

	for _, collectionIndex := range app.collectionsForBulkInserting {
		data := app.bulkInserts[collectionIndex]
		if dbgPrint {
			fmt.Printf("Length of %s: %d\n", app.collectionNames[collectionIndex], len(data))
		}
		if len(data) > 0 {
			_, err := app.collections[collectionIndex].BulkWrite(context.TODO(), data)
			checkError(err)
		}
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

func (app *Application) saveDataToMongo() {
	for _, collectionIndex := range app.collectionsForBulkInsertingEveryRound {
		data := app.bulkInserts[collectionIndex]
		//if dbgPrint {
		//	fmt.Printf("Length of %s: %d\n", app.collectionNames[collectionIndex], len(data))
		//}
		if len(data) > 0 {
			_, err := app.collections[collectionIndex].BulkWrite(context.TODO(), data)
			checkError(err)
		}
		app.bulkInserts[collectionIndex] = nil
	}
	//runtime.GC() // doesn't seem to be helpful at all -__-
}

func (app *Application) encodePlayerMovement(SteamID int64, playerMovement *PlayerMovement, reset bool) PlayerMovementInfoEncoded {
	PMIE := PlayerMovementInfoEncoded {
		playerMovement.StartFrame,
		app.savedFrameNumber,
		SteamID,
		elias.EliasGammaNegative(elias.ArrayToDeltas(playerMovement.PositionX)...),
		elias.EliasGammaNegative(elias.ArrayToDeltas(playerMovement.PositionY)...),
		elias.EliasGammaNegative(elias.ArrayToDeltas(playerMovement.PositionZ)...),
		elias.EliasGammaNegative(elias.ArrayToDeltas(playerMovement.ViewX)...),
		elias.EliasGammaNegative(elias.ArrayToDeltas(playerMovement.ViewY)...),
	}
	if playerMovement.EndFrame == 0 { // player didn't get killed or disconnected
		PMIE.EndFrame = app.savedFrameNumber
	}

	if reset {
		// reset movement data
		//playerMovement.StartFrame = app.savedFrameNumber + 1 // moved to Parse()
		playerMovement.StartFrame = 0
		playerMovement.EndFrame = 0
		playerMovement.PositionX = playerMovement.PositionX[:0]
		playerMovement.PositionY = playerMovement.PositionY[:0]
		playerMovement.PositionZ = playerMovement.PositionX[:0]
		playerMovement.ViewX = playerMovement.PositionX[:0]
		playerMovement.ViewY = playerMovement.ViewY[:0]
	}

	return PMIE
}

func (app *Application) getPlayerStats(player *common.Player) *PlayerRoundStats {
	if app.playersStats[app.roundNumber-1] == nil {
		app.playersStats[app.roundNumber-1] = make(map[int64]*PlayerRoundStats)
	}
	PS, ok := app.playersStats[app.roundNumber-1][player.SteamID]
	if !ok {
		PS = NewPlayerStats(player.SteamID, player.Name)
		app.playersStats[app.roundNumber-1][player.SteamID] = PS
	}
	return PS
}

func (app *Application) AliveMembers(team common.Team) []*common.Player {
	res := make([]*common.Player, 0, 5)
	for _, p := range app.parser.GameState().Participants().TeamMembers(team) {
		if p.IsAlive() {
			res = append(res, p)
		}
	}
	return res
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func dbgLog(s string) {
	if dbgPrint {
		fmt.Println(s)
	}
}