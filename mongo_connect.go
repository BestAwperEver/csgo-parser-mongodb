package main

import (
	"context"
	"fmt"
	"github.com/golang/geo/r2"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/common"
	"github.com/markus-wa/demoinfocs-golang/events"
	"log"
	"os"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	//"github.com/mongodb/mongo-go-driver/bson"
	//"github.com/mongodb/mongo-go-driver/mongo"
	//"github.com/mongodb/mongo-go-driver/mongo/options"
	dem "github.com/markus-wa/demoinfocs-golang"
)

// You will be using this Trainer type later in the program
type Trainer struct {
	Name string
	Age  int
	City string
}

func main1() {

	//client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))

	if err != nil {
		log.Fatal(err)
	}

	// Check the connection
	err = client.Ping(ctx, readpref.Primary())

	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		err = client.Disconnect(ctx)

		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Connection to MongoDB closed.")
	}()

	fmt.Println("Connected to MongoDB!")

	collection := client.Database("test").Collection("trainers")

	ash := Trainer{"Ash", 10, "Pallet Town"}
	misty := Trainer{"Misty", 10, "Cerulean City"}
	brock := Trainer{"Brock", 15, "Pewter City"}

	insertResult, err := collection.InsertOne(context.TODO(), ash)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Inserted a single document: ", insertResult.InsertedID)

	trainers := []interface{}{misty, brock}

	insertManyResult, err := collection.InsertMany(context.TODO(), trainers)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Inserted multiple documents: ", insertManyResult.InsertedIDs)

	filter := bson.D{{"name", "Ash"}}

	update := bson.D{
		{"$inc", bson.D{
			{"age", 1},
		}},
	}

	updateResult, err := collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Matched %v documents and updated %v documents.\n",
		updateResult.MatchedCount, updateResult.ModifiedCount)

	// create a value into which the result can be decoded
	var result Trainer

	err = collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found a single document: %+v\n", result)

	// Pass these options to the Find method
	findOptions := options.Find()
	findOptions.SetLimit(2)

	// Here's an array in which you can store the decoded documents
	var results []*Trainer

	// Passing nil as the filter matches all documents in the collection
	cur, err := collection.Find(context.Background(), bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(context.Background())

	// Finding multiple documents returns a cursor
	// Iterating through the cursor allows us to decode documents one at a time
	for cur.Next(context.Background()) {

		// create a value into which the single document can be decoded
		var elem Trainer
		err := cur.Decode(&elem)
		if err != nil {
			log.Fatal(err)
		}
		results = append(results, &elem)
	}

	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found multiple documents (array of pointers): %+v\n", results)

	//deleteResult, err := collection.DeleteMany(context.TODO(), bson.D{})
	//if err != nil {
	//	log.Fatal(err)
	//}
	//fmt.Printf("Deleted %v documents in the trainers collection\n", deleteResult.DeletedCount)
}

func connect_to_mongo(URI string, timeout time.Duration) *mongo.Client {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(URI))

	if err != nil {
		log.Fatal(err)
	}

	// Check the connection
	err = client.Ping(ctx, readpref.Primary())

	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, readpref.Primary())

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB!")

	return client
}

func close_connection_to_mongo(client *mongo.Client) {
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)

	err := client.Disconnect(ctx)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connection to MongoDB closed.")
}

type Tick struct {
	TickNumber	int						`bson:"TickNumber"`
	Positions	map[string]r3.Vector	`bson:"Positions"`
	NadesPositions []NadeInfo			`bson:"NadesPositions"`
}

type NadeInfo struct {
	UniqueID	int64		`bson:"UniqueID"`
	Position	r3.Vector	`bson:"Position"`
}

type PlayerInfo struct {
	SteamID		int64		`bson:"SteamID"`
	Position	r3.Vector	`bson:"Position"`
	ViewX		float32		`bson:"ViewX"`
	ViewY		float32		`bson:"ViewY"`
}

type PlayerStaticInfo struct {
	SteamID		int64	`bson:"SteamID"`
	Name		string	`bson:"Name"`
	EntityID	int		`bson:"EntityID"`
}

func NewPlayerStaticInfo(player common.Player) PlayerStaticInfo {
	PST := PlayerStaticInfo{}
	PST.SteamID = player.SteamID
	PST.Name = player.Name
	PST.EntityID = player.EntityID
	return PST
}

type EquipmentInfo struct {
	UniqueID		int64					`bson:"UniqueID"`
	Weapon			common.EquipmentElement `bson:"Weapon"`
	OwnerID			int64					`bson:"OwnerID"`
	//AmmoType		int 					`bson:"AmmoType"`
	AmmoInMagazine	int 					`bson:"AmmoInMagazine"`
	AmmoReserve		int 					`bson:"AmmoReserve"`
	ZoomLevel		int 					`bson:"ZoomLevel"`
}

type TeamStateInfo struct {
	ID			int		`bson:"ID"`
	Score		int		`bson:"Score"`
	ClanName	string	`bson:"ClanName"`
	Flag		string	`bson:"Flag"`
}

type InfernoInfo struct {
	UniqueID		int64		`bson:"UniqueID"`
	ConvexHull2D	[]r2.Point	`bson:"ConvexHull2D"`
}

type EventInfo struct {
	FrameNumber int						`bson:"FrameNumber"`
	EventType	EvType					`bson:"EventType"`
	Data		map[string]interface{}	`bson:"Data, omitempty"`
}

type EquipmentElementStaticInfo struct {
	UniqueID	int64					`bson:"UniqueID"`
	EntityID	int						`bson:"EntityID"`
	OwnerID		int64					`bson:"OwnerID"`
	Weapon		common.EquipmentElement	`bson:"Weapon"`
}

var EquipmentElements = make(map[int64]EquipmentElementStaticInfo)

func NewEquipmentElementStaticInfo(GP common.GrenadeProjectile) EquipmentElementStaticInfo {
	GPI := EquipmentElementStaticInfo{
		GP.UniqueID(),
		GP.EntityID,
		//-1,
		-1,
		GP.Weapon,
	}
	//if GP.Owner != nil {
	//	GPI.OwnerID = GP.Owner.SteamID
	//}
	if GP.Thrower != nil {
		GPI.OwnerID = GP.Thrower.SteamID
	}
	return GPI
}

// makes a map from event for easier persistent saving
func getMap(event interface{}) map[string]interface{} {
	resultMap := make(map[string]interface{})
	reflectedEvent := reflect.ValueOf(event)
	if reflectedEvent.Kind() != reflect.Struct {
		panic("getMap received a non-struct object")
	}

	for i := 0; i < reflectedEvent.NumField(); i++ {
		if field := reflectedEvent.Field(i); field.Kind() != reflect.Ptr {
			if field.CanInterface() {
				resultMap[reflectedEvent.Type().Field(i).Name] = field.Interface()
			}
		} else if field.CanInterface() {
			if field.IsNil() == false {
				switch field := reflect.Indirect(field); field.Type().Name() {
				//default:
				//	data[reflectedEvent.Type().Field(i).Name] = reflectedEvent.Field(i).Interface()
				case "Player":
					P := field.Interface().(common.Player)
					resultMap[reflectedEvent.Type().Field(i).Name] = P.SteamID
				case "GrenadeProjectile":
					GP := field.Interface().(common.GrenadeProjectile)
					if _, ok := EquipmentElements[GP.UniqueID()]; !ok {
						EquipmentElements[GP.UniqueID()] = NewEquipmentElementStaticInfo(GP)
					}
					resultMap[reflectedEvent.Type().Field(i).Name] = GP.UniqueID()
				case "Equipment":
					Equip := field.Interface().(common.Equipment)
					EI := EquipmentInfo{
						Equip.UniqueID(),
						Equip.Weapon,
						-1,
						//Equip.AmmoType,
						Equip.AmmoInMagazine,
						0,
						Equip.ZoomLevel,
					}
					if Equip.Owner != nil {
						EI.OwnerID = Equip.Owner.SteamID
					}
					if Equip.AmmoReserve != 0 {
						EI.AmmoReserve = Equip.AmmoReserve
					}
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
				}
			} else {
				resultMap[reflectedEvent.Type().Field(i).Name] = -1
			}
		}
	}

	return resultMap
}

type EvType int
const (
	Kill EvType = 1
	AnnouncementFinalRound EvType = 2
	AnnouncementLastRoundHalf EvType = 3
	AnnouncementMatchStarted EvType = 4
	AnnouncementWinPanelMatch EvType = 5
	BombDefuseAborted EvType = 6
	BombDefuseStart EvType = 7
	BombDefused EvType = 8
	BombDropped EvType = 9
	BombEvent EvType = 10
	BombEventIf EvType = 11
	BombExplode EvType = 12
	BombPickup EvType = 13
	BombPlantBegin EvType = 14
	BombPlanted EvType = 15
	BotTakenOver EvType = 16
	ChatMessage EvType = 17
	DataTablesParsed EvType = 18
	DecoyExpired EvType = 19
	DecoyStart EvType = 20
	FireGrenadeExpired EvType = 21
	FireGrenadeStart EvType = 22
	FlashExplode EvType = 23
	Footstep EvType = 24
	GameHalfEnded EvType = 25
	GamePhaseChanged EvType = 26
	GenericGameEvent EvType = 27
	GrenadeEvent EvType = 28
	GrenadeEventIf EvType = 29
	GrenadeProjectileBounce EvType = 30
	GrenadeProjectileDestroy EvType = 31
	GrenadeProjectileThrow EvType = 32
	HeExplode EvType = 33
	HitGroup EvType = 34
	InfernoExpired EvType = 35
	InfernoStart EvType = 36
	IsWarmupPeriodChanged EvType = 37
	ItemDrop EvType = 38
	ItemEquip EvType = 39
	ItemPickup EvType = 40
	MatchStart EvType = 41
	MatchStartedChanged EvType = 42
	ParserWarn EvType = 43
	PlayerConnect EvType = 44
	PlayerDisconnected EvType = 45
	PlayerFlashed EvType = 46
	PlayerHurt EvType = 47
	PlayerJump EvType = 48
	PlayerTeamChange EvType = 49
	RankUpdate EvType = 50
	RoundEnd EvType = 51
	RoundEndOfficial EvType = 52
	RoundEndReason EvType = 53
	RoundFreezetimeEnd EvType = 54
	RoundMVPAnnouncement EvType = 55
	RoundMVPReason EvType = 56
	RoundStart EvType = 57
	SayText EvType = 58
	SayText2 EvType = 59
	ScoreUpdated EvType = 60
	SmokeExpired EvType = 61
	SmokeStart EvType = 62
	StringTableCreated EvType = 63
	TeamSideSwitch EvType = 64
	TickDone EvType = 65
	WeaponFire EvType = 66
)

var EvTypeIndex = map[string]EvType{
	"Kill": Kill,
	"AnnouncementFinalRound": AnnouncementFinalRound,
	"AnnouncementLastRoundHalf": AnnouncementLastRoundHalf,
	"AnnouncementMatchStarted": AnnouncementMatchStarted,
	"AnnouncementWinPanelMatch": AnnouncementWinPanelMatch,
	"BombDefuseAborted": BombDefuseAborted,
	"BombDefuseStart": BombDefuseStart,
	"BombDefused": BombDefused,
	"BombDropped": BombDropped,
	"BombEvent": BombEvent,
	"BombEventIf": BombEventIf,
	"BombExplode": BombExplode,
	"BombPickup": BombPickup,
	"BombPlantBegin": BombPlantBegin,
	"BombPlanted": BombPlanted,
	"BotTakenOver": BotTakenOver,
	"ChatMessage": ChatMessage,
	"DataTablesParsed": DataTablesParsed,
	"DecoyExpired": DecoyExpired,
	"DecoyStart": DecoyStart,
	"FireGrenadeExpired": FireGrenadeExpired,
	"FireGrenadeStart": FireGrenadeStart,
	"FlashExplode": FlashExplode,
	"Footstep": Footstep,
	"GameHalfEnded": GameHalfEnded,
	"GamePhaseChanged": GamePhaseChanged,
	"GenericGameEvent": GenericGameEvent,
	"GrenadeEvent": GrenadeEvent,
	"GrenadeEventIf": GrenadeEventIf,
	"GrenadeProjectileBounce": GrenadeProjectileBounce,
	"GrenadeProjectileDestroy": GrenadeProjectileDestroy,
	"GrenadeProjectileThrow": GrenadeProjectileThrow,
	"HeExplode": HeExplode,
	"HitGroup": HitGroup,
	"InfernoExpired": InfernoExpired,
	"InfernoStart": InfernoStart,
	"IsWarmupPeriodChanged": IsWarmupPeriodChanged,
	"ItemDrop": ItemDrop,
	"ItemEquip": ItemEquip,
	"ItemPickup": ItemPickup,
	"MatchStart": MatchStart,
	"MatchStartedChanged": MatchStartedChanged,
	"ParserWarn": ParserWarn,
	"PlayerConnect": PlayerConnect,
	"PlayerDisconnected": PlayerDisconnected,
	"PlayerFlashed": PlayerFlashed,
	"PlayerHurt": PlayerHurt,
	"PlayerJump": PlayerJump,
	"PlayerTeamChange": PlayerTeamChange,
	"RankUpdate": RankUpdate,
	"RoundEnd": RoundEnd,
	"RoundEndOfficial": RoundEndOfficial,
	"RoundEndReason": RoundEndReason,
	"RoundFreezetimeEnd": RoundFreezetimeEnd,
	"RoundMVPAnnouncement": RoundMVPAnnouncement,
	"RoundMVPReason": RoundMVPReason,
	"RoundStart": RoundStart,
	"SayText": SayText,
	"SayText2": SayText2,
	"ScoreUpdated": ScoreUpdated,
	"SmokeExpired": SmokeExpired,
	"SmokeStart": SmokeStart,
	"StringTableCreated": StringTableCreated,
	"TeamSideSwitch": TeamSideSwitch,
	"TickDone": TickDone,
	"WeaponFire": WeaponFire,
}

var EvTypeByIndex = map[EvType]string{
	Kill: "Kill",
	AnnouncementFinalRound: "AnnouncementFinalRound",
	AnnouncementLastRoundHalf: "AnnouncementLastRoundHalf",
	AnnouncementMatchStarted: "AnnouncementMatchStarted",
	AnnouncementWinPanelMatch: "AnnouncementWinPanelMatch",
	BombDefuseAborted: "BombDefuseAborted",
	BombDefuseStart: "BombDefuseStart",
	BombDefused: "BombDefused",
	BombDropped: "BombDropped",
	BombEvent: "BombEvent",
	BombEventIf: "BombEventIf",
	BombExplode: "BombExplode",
	BombPickup: "BombPickup",
	BombPlantBegin: "BombPlantBegin",
	BombPlanted: "BombPlanted",
	BotTakenOver: "BotTakenOver",
	ChatMessage: "ChatMessage",
	DataTablesParsed: "DataTablesParsed",
	DecoyExpired: "DecoyExpired",
	DecoyStart: "DecoyStart",
	FireGrenadeExpired: "FireGrenadeExpired",
	FireGrenadeStart: "FireGrenadeStart",
	FlashExplode: "FlashExplode",
	Footstep: "Footstep",
	GameHalfEnded: "GameHalfEnded",
	GamePhaseChanged: "GamePhaseChanged",
	GenericGameEvent: "GenericGameEvent",
	GrenadeEvent: "GrenadeEvent",
	GrenadeEventIf: "GrenadeEventIf",
	GrenadeProjectileBounce: "GrenadeProjectileBounce",
	GrenadeProjectileDestroy: "GrenadeProjectileDestroy",
	GrenadeProjectileThrow: "GrenadeProjectileThrow",
	HeExplode: "HeExplode",
	HitGroup: "HitGroup",
	InfernoExpired: "InfernoExpired",
	InfernoStart: "InfernoStart",
	IsWarmupPeriodChanged: "IsWarmupPeriodChanged",
	ItemDrop: "ItemDrop",
	ItemEquip: "ItemEquip",
	ItemPickup: "ItemPickup",
	MatchStart: "MatchStart",
	MatchStartedChanged: "MatchStartedChanged",
	ParserWarn: "ParserWarn",
	PlayerConnect: "PlayerConnect",
	PlayerDisconnected: "PlayerDisconnected",
	PlayerFlashed: "PlayerFlashed",
	PlayerHurt: "PlayerHurt",
	PlayerJump: "PlayerJump",
	PlayerTeamChange: "PlayerTeamChange",
	RankUpdate: "RankUpdate",
	RoundEnd: "RoundEnd",
	RoundEndOfficial: "RoundEndOfficial",
	RoundEndReason: "RoundEndReason",
	RoundFreezetimeEnd: "RoundFreezetimeEnd",
	RoundMVPAnnouncement: "RoundMVPAnnouncement",
	RoundMVPReason: "RoundMVPReason",
	RoundStart: "RoundStart",
	SayText: "SayText",
	SayText2: "SayText2",
	ScoreUpdated: "ScoreUpdated",
	SmokeExpired: "SmokeExpired",
	SmokeStart: "SmokeStart",
	StringTableCreated: "StringTableCreated",
	TeamSideSwitch: "TeamSideSwitch",
	TickDone: "TickDone",
	WeaponFire: "WeaponFire",
}

var ImplicitlyProcessedEvents = map[EvType]bool{
	Footstep: true,
	WeaponFire: true,
}

func main2() {
	client := connect_to_mongo("mongodb://localhost:27017", 2*time.Second)

	defer close_connection_to_mongo(client)

	pathToDemoFile := "D:\\Games\\steamapps\\common\\Counter-Strike Global Offensive\\csgo\\replays\\match730_003347728584037892268_1491137544_186.dem"

	f, err := os.Open(pathToDemoFile)
	defer f.Close()
	checkError(err)

	p := dem.NewParser(f)

	header, err := p.ParseHeader()
	checkError(err)
	fmt.Println("Header:", getMap(header))

	next, err := p.ParseNextFrame()
	checkError(err)

	databaseName := "test"

	collectionTicks := client.Database(databaseName).Collection("ticks")
	collectionEvents := client.Database(databaseName).Collection("events")
	collectionPlayers := client.Database(databaseName).Collection("players")
	collectionEntities := client.Database(databaseName).Collection("entities")

	CurrentProjectiles := make(map[int]*common.GrenadeProjectile)

	p.RegisterEventHandler(func(e events.GrenadeProjectileThrow) {
		CurrentProjectiles[e.Projectile.EntityID] = e.Projectile
		EquipmentElements[e.Projectile.UniqueID()] = NewEquipmentElementStaticInfo(*e.Projectile)
	})

	p.RegisterEventHandler(func(e events.GameHalfEnded) {
		var data = EventInfo{
			p.CurrentFrame(),
			GameHalfEnded,
			getMap(e),
		}

		_, err := collectionEvents.InsertOne(context.TODO(), data)
		checkError(err)
	})

	p.RegisterEventHandler(func(e events.RoundEnd) {
		if p.GameState().TeamTerrorists().Score == 15 && e.Winner == common.TeamTerrorists ||
			p.GameState().TeamCounterTerrorists().Score == 15 && e.Winner == common.TeamCounterTerrorists ||
			p.GameState().TotalRoundsPlayed() == 29 {
			//game is over (works for mm)
			for _, v := range EquipmentElements {
				_, err := collectionEntities.InsertOne(context.TODO(), v)
				checkError(err)
			}
		}
	})

	p.RegisterEventHandler(func(e events.MatchStartedChanged) {
		if e.NewIsStarted {
			for _, player := range p.GameState().Participants().Playing() {
				_, err := collectionPlayers.InsertOne(context.TODO(),
					NewPlayerStaticInfo(*player))
				checkError(err)
			}
		}
	})

	p.RegisterEventHandler(func(e events.RankUpdate) {
		var data = EventInfo{
			p.CurrentFrame(),
			RankUpdate,
			getMap(e),
		}

		_, err := collectionEvents.InsertOne(context.TODO(), data)
		checkError(err)
	})

	p.RegisterEventHandler(func(e events.FlashExplode) {
		if p.GameState().IsWarmupPeriod() {
			return
		}

		type FlashExplodeInfo struct {
			UniqueID	int64		`bson:"UniqueID"`
			Position	r3.Vector	`bson:"Position"`
		}

		var data = struct {
			FrameNumber	int					`bson:"FrameNumber"`
			EventType	EvType				`bson:"EventType"`
			Data		FlashExplodeInfo	`bson:"Data"`
		}{
			p.CurrentFrame(),
			FlashExplode,
			FlashExplodeInfo{
				CurrentProjectiles[e.GrenadeEntityID].UniqueID(),
				e.Position,
			},
		}

		_, err := collectionEvents.InsertOne(context.TODO(), data)
		checkError(err)
	})

	p.RegisterEventHandler(func(e events.Kill) {
		if p.GameState().IsWarmupPeriod() {
			return
		}

		var data = EventInfo{
			p.CurrentFrame(),
			Kill,
			getMap(e),
		}

		_, err := collectionEvents.InsertOne(context.TODO(), data)
		checkError(err)
	})

	p.RegisterEventHandler(func(e events.PlayerFlashed) {
		type PlayerFlashedInfo struct {
			AttackerID		int64			`bson:"AttackerID"`
			PlayerID		int64			`bson:"PlayerID"`
			FlashDuration	time.Duration	`bson:"FlashDuration"`
		}

		var data = struct {
			FrameNumber	int					`bson:"FrameNumber"`
			EventType	EvType				`bson:"EventType"`
			Data		PlayerFlashedInfo	`bson:"Data"`
		}{
			p.CurrentFrame(),
			PlayerFlashed,
			PlayerFlashedInfo{
				e.Attacker.SteamID,
				e.Player.SteamID,
				e.FlashDuration(),
			},
		}

		_, err := collectionEvents.InsertOne(context.TODO(), data)
		checkError(err)
	})

	// general handler function
	p.RegisterEventHandler(func(e interface{}) {
		if p.GameState().IsWarmupPeriod() {
			return
		}
		reflectedEvent := reflect.ValueOf(e)

		if evType := EvTypeIndex[reflectedEvent.Type().Name()]; ImplicitlyProcessedEvents[evType] {
			var data = EventInfo {
				p.CurrentFrame(),
				evType,
				getMap(reflectedEvent.Interface()),
			}

			_, err := collectionEvents.InsertOne(context.TODO(), data)
			checkError(err)
		}
	})

	var saveTicks = true

	for next {
		next, err = p.ParseNextFrame()
		checkError(err)
		if p.GameState().IsWarmupPeriod() || p.GameState().IsMatchStarted() == false {
			continue
		}
		if p.GameState().TotalRoundsPlayed() > 3 {
			break
		}
		if saveTicks {
			PlayersPos := make([]PlayerInfo, 0, len(p.GameState().Participants().Playing()))
			NadesPos := make([]NadeInfo, 0, len(p.GameState().GrenadeProjectiles()))
			CurrentInfernos := make([]InfernoInfo, 0, len(p.GameState().Infernos()))
			for _, v := range p.GameState().Participants().Playing() {
				PlayersPos = append(PlayersPos, PlayerInfo{
					v.SteamID,
					v.Position,
					v.ViewDirectionX,
					v.ViewDirectionY,
				})
			}
			for _, v := range p.GameState().GrenadeProjectiles() {
				NadesPos = append(NadesPos, NadeInfo{
					v.UniqueID(),
					v.Position,
				})
			}
			for _, v := range p.GameState().Infernos() {
				CurrentInfernos = append(CurrentInfernos, InfernoInfo{
					v.UniqueID(),
					v.Active().ConvexHull2D(),
				})
			}
			//tickInfo := Tick{p.CurrentFrame(), PlayersPos, NadesPos}
			//fmt.Println(tickInfo)
			_, err := collectionTicks.InsertOne(context.TODO(), struct {
				FrameNumber      int           `bson:"FrameNumber"`
				PlayersPositions []PlayerInfo  `bson:"PlayerPositions"`
				NadesPositions   []NadeInfo    `bson:"GrenadePositions"`
				CurrentInfernos  []InfernoInfo `bson:"CurrentInfernos"`
			}{
				p.CurrentFrame(),
				PlayersPos,
				NadesPos,
				CurrentInfernos,
			})
			checkError(err)
		}
	}
}

func main3() {
	e := struct{Foo int; Foo2 string; Bar common.Player}{
		5,
		"asdf",
		common.Player{},
	}
	reflectedEvent := reflect.ValueOf(e)
	//values := make([]interface{}, reflected_event.NumField())
	data := make(map[string]interface{})

	for i := 0; i < reflectedEvent.NumField(); i++ {
		//values[i] = reflected_event.Field(i).Interface()
		//reflected_event.Type().Field(i).Name
		switch n := reflectedEvent.Field(i).Type().Name(); n {
		default:
			data[reflectedEvent.Type().Field(i).Name] = reflectedEvent.Field(i).Interface()
		case "Player":
			reflectedEvent2 := reflect.ValueOf(reflectedEvent.Field(i).Interface())
			data[reflectedEvent.Type().Field(i).Name] = reflectedEvent2.FieldByName("SteamID")
		}
	}

	fmt.Print(data)
}

func main() {
	main2()
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}