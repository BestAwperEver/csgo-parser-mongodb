package app

import (
	"github.com/golang/geo/r2"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/common"
)

type FramePositions struct {
	//FrameNumber			int						`bson:"FrameNumber"`
	//PlayerPositions		map[string]r3.Vector	`bson:"Positions"`
	//GrenadesPositions	[]GrenadePositionInfo	`bson:"GrenadesPositions"`
	FrameNumber       int                   `bson:"FrameNumber"`
	PlayersPositions  []PlayerMovementInfo  `bson:"PlayerPositions"`
	GrenadesPositions []GrenadePositionInfo `bson:"GrenadePositions"`
	CurrentInfernos   []InfernoInfo         `bson:"CurrentInfernos"`
}

type GrenadePositionInfo struct {
	UniqueID	int64		`bson:"UniqueID"`
	Position	r3.Vector	`bson:"Position"`
}

type PlayerMovementInfo struct {
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
	AmmoLeft				[32]int					`bson:"AmmoLeft"`
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
	LastAlivePosition		r3.Vector				`bson:"LastAlivePosition"`
	Inventory				[]EquipmentInfo			`bson:"Inventory"`
	AdditionalInfo			AdditionalPlayerInfo	`bson:"AdditionalInfo"`
}

func NewPlayerStateInfo(p *common.Player) PlayerStateInfo {
	PSI := PlayerStateInfo{
		-1,
		p.Team,
		p.FlashDuration,
		-1,
		p.AmmoLeft,
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
		p.LastAlivePosition,
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
	AmmoReserve		int 					`bson:"AmmoReserve"`
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

type GameStateInfo struct {
	FrameNumber	int					`bson:"FrameNumber"`
	Players		[]PlayerStateInfo	`bson:"Players"`
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