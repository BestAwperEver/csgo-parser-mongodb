package app

import (
	"csgo-parser-mongodb/util/elias"
	"github.com/golang/geo/r2"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/common"
	"time"
)

type FramePositions struct {
	FrameNumber       int                   `bson:"FrameNumber"`
	PlayersPositions  []PlayerMovementInfo  `bson:"PlayerPositions"`
}

type FrameProjectiles struct {
	FrameNumber       int                   `bson:"FrameNumber"`
	GrenadesPositions []GrenadePositionInfo `bson:"GrenadePositions"`
}

type FrameInfernos struct {
	FrameNumber       int                   `bson:"FrameNumber"`
	CurrentInfernos   []InfernoInfo         `bson:"CurrentInfernos"`
}

type Int16Vector3 struct {
	X, Y, Z	int16
}

func NewInt16Vector3(v r3.Vector) Int16Vector3 {
	return Int16Vector3{int16(v.X), int16(v.Y), int16(v.Z)}
}

type Int16Vector2 struct {
	X, Y	int16
}

func NewInt16Vector2(v r2.Point) Int16Vector2 {
	return Int16Vector2{int16(v.X), int16(v.Y)}
}

type GrenadePositionInfo struct {
	UniqueID	int64			`bson:"UniqueID"`
	Position	Int16Vector3	`bson:"Position"`
}

type GrenadePositionInfoEncoded struct {
	StartFrame int
	EndFrame   int
	UniqueID   int64                    `bson:"UniqueID"`
	PositionX  elias.BitArrayWithLength `bson:"X"`
	PositionY  elias.BitArrayWithLength `bson:"Y"`
	PositionZ  elias.BitArrayWithLength `bson:"Z"`
}

type PlayerMovementInfo struct {
	SteamID		int64			`bson:"SteamID"`
	Position	Int16Vector3	`bson:"Position"`
	ViewX		int16			`bson:"ViewX"`
	ViewY		int16			`bson:"ViewY"`
}

type PlayerMovementInfoEncoded struct {
	StartFrame int
	EndFrame   int
	SteamID    int64                    `bson:"SteamID"`
	PositionX  elias.BitArrayWithLength `bson:"X"`
	PositionY  elias.BitArrayWithLength `bson:"Y"`
	PositionZ  elias.BitArrayWithLength `bson:"Z"`
	ViewX      elias.BitArrayWithLength `bson:"ViewXArray"`
	ViewY      elias.BitArrayWithLength `bson:"ViewYArray"`
}

type GrenadeProjectileWithStartFrame struct {
	StartFrame	int
	*common.GrenadeProjectile
}

type PlayerMovement struct {
	StartFrame	int
	EndFrame	int
	SteamID		int64	`bson:"SteamID"`
	PositionX	[]int	`bson:"X"`
	PositionY	[]int	`bson:"Y"`
	PositionZ	[]int	`bson:"Z"`
	ViewX		[]int	`bson:"ViewXArray"`
	ViewY		[]int	`bson:"ViewYArray"`
}

type RoundMovement struct {
	RoundNumber		int							`bson:"RoundNumber"`
	PlayerMovements	[]PlayerMovementInfoEncoded	`bson:"PlayerMovements"`
}

type PlayerStaticInfo struct {
	SteamID		int64	`bson:"SteamID"`
	Name		string	`bson:"Name"`
	EntityID	int		`bson:"EntityID"`
}

type AdditionalPlayerInfo struct {
	Score	int	`bson:"Score"`
	Assists	int	`bson:"Assists"`
	Deaths	int	`bson:"Deaths"`
	Kills	int	`bson:"Kills"`
	MVPs	int	`bson:"MVPs"`
}

func NewAdditionalPlayerInfo(info *common.AdditionalPlayerInformation) AdditionalPlayerInfo {
	return AdditionalPlayerInfo{
		info.Score,
		info.Assists,
		info.Deaths,
		info.Kills,
		info.MVPs,
	}
}

type PlayerStateInfo struct {
	SteamID					int64					`bson:"SteamID"`
	Team					common.Team				`bson:"Team"`
	FlashDuration			float32					`bson:"FlashDuration"`
	ActiveWeaponID			int64					`bson:"ActiveWeaponID"`
	//AmmoLeft				[32]int					`bson:"AmmoLeft"`	// always seems to be null
	Armor					int						`bson:"Armor"`
	CurrentEquipmentValue	int						`bson:"CurrentEquipmentValue"`
	HasDefuseKit			bool					`bson:"HasDefuseKit"`
	HasBomb					bool					`bson:"HasBomb"`
	HasHelmet				bool					`bson:"HasHelmet"`
	Hp						int						`bson:"Hp"`
	IsBot					bool					`bson:"IsBot"`
	IsConnected				bool					`bson:"IsConnected"`
	IsDefusing				bool					`bson:"IsDefusing"`
	IsDucking				bool					`bson:"IsDucking"`
	IsAlive					bool					`bson:"IsAlive"`
	IsBlinded				bool					`bson:"IsBlinded"`
	Money					int						`bson:"Money"`
	LastAlivePosition		Int16Vector3			`bson:"LastAlivePosition"`
	ViewX					int16					`bson:"ViewX"`
	ViewY					int16					`bson:"ViewY"`
	Inventory				[]EquipmentInfo			`bson:"Inventory"`
	AdditionalInfo			AdditionalPlayerInfo	`bson:"AdditionalInfo"`
}

func NewPlayerStateInfo(p *common.Player) PlayerStateInfo {
	PSI := PlayerStateInfo{
		-1,
		p.Team,
		p.FlashDuration,
		-1,
		//p.AmmoLeft,
		p.Armor,
		p.CurrentEquipmentValue,
		p.HasDefuseKit,
		false,
		p.HasHelmet,
		p.Hp,
		p.IsBot,
		p.IsConnected,
		p.IsDefusing,
		p.IsDucking,
		p.IsAlive(),
		p.IsBlinded(),
		p.Money,
		NewInt16Vector3(p.LastAlivePosition),
		int16(p.ViewDirectionX),
		int16(p.ViewDirectionY),
		make([]EquipmentInfo, 0, 7),
		NewAdditionalPlayerInfo(p.AdditionalPlayerInformation),
	}
	if p.ActiveWeapon() != nil {
		PSI.ActiveWeaponID = p.ActiveWeapon().UniqueID()
	}
	if PSI.IsBot == false {
		PSI.SteamID = p.SteamID
	}
	for _, Equip := range p.Weapons() {
		PSI.Inventory = append(PSI.Inventory, NewEquipmentInfo(*Equip))
		if Equip.Weapon == common.EqBomb {
			PSI.HasBomb = true
		}
	}
	return PSI
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
	AmmoReserve		int 					`bson:"AmmoReserve"`	// Seems to be 0 all the time for some reason
	ZoomLevel		int 					`bson:"ZoomLevel"`
}

func NewEquipmentInfo(Equip common.Equipment) EquipmentInfo {
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

	return EI
}

type PlayerRoundStats struct {
	SteamID int64
	Name	string
	rounds  int // = 1

	clutch	[5]bool
	clLoose bool

	kills     int
	headshots int

	damage     int
	teamDamage int

	openKill	bool

	teamFlash	time.Duration
	selfFlash	time.Duration
	enemyFlash	time.Duration
	enemyFlashed int

	weaponFires map[common.EquipmentElement]int
	//weaponDamage	map[common.EquipmentElement]int

	//hegrenades int
	heDamage     int

	//molotovs   int
	//molotovdamage	int
	//flashbangs int
}

func NewPlayerStats(SteamID int64, Name string) *PlayerRoundStats {
	return &PlayerRoundStats{
		SteamID: SteamID,
		Name: Name,
		weaponFires: make(map[common.EquipmentElement]int),
		rounds: 1,
	}
	//PS.headshots = 0
	//PS.rounds = 1
	//PS.k1 = 0
	//PS.k2 = 0
	//PS.k3 = 0
	//PS.k4 = 0
	//PS.k5 = 0
	//PS.clLoose = 0
	//PS.cl1 = 0
	//PS.cl2 = 0
	//PS.cl3 = 0
	//PS.cl4 = 0
	//PS.cl5 = 0
	//PS.damage = 0
	//PS.teamDamage = 0
	//PS.selfFlash = 0
	//PS.teamFlash = 0
	//PS.enemyFlash = 0
	//PS.openKills = 0
	//PS.hegrenades = 0
	//PS.molotovs = 0
	//PS.molotovdamage = 0
	//PS.heDamage = 0
	//PS.flashbangs = 0
	//PS.weaponFires = make(map[common.EquipmentElement]int)
	//return &PS
}

type GameStateInfo struct {
	FrameNumber	int					`bson:"FrameNumber"`
	Players		[]PlayerStateInfo	`bson:"Players"`
}

type PlayerFlashedInfo struct {
	AttackerID		int64			`bson:"AttackerID"`
	PlayerID		int64			`bson:"PlayerID"`
	FlashDuration	time.Duration	`bson:"FlashDuration"`
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

type FlashExplodeInfo struct {
	//UniqueID	int64			`bson:"UniqueID"`
	Position	Int16Vector3	`bson:"Position"`
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

func NewEquipmentElementStaticInfo(eq interface{}) EquipmentElementStaticInfo {
	if GP, ok := eq.(*common.GrenadeProjectile); ok {
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
	} else if EQ, ok := eq.(common.Equipment); ok {
		EESI := EquipmentElementStaticInfo{
			EQ.UniqueID(),
			EQ.EntityID,
			-1,
			EQ.Weapon,
		}
		if EQ.Owner != nil {
			EESI.OwnerID = GP.Owner.SteamID
		}
		return EESI
	} else {
		// shouldn't be anything else
		panic("NewEquipmentElementStaticInfo recieved non-equipment value")
	}
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