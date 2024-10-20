package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"time"

	dem "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

type DemoData struct {
	Filename string
	MapName string
	Clutches[] Clutch
}

type Clutch struct {
	Clutcher ClutchPlayer // 
	Enemies[] ClutchPlayer //
	ClutchType string
	TickStart uint32
	TickEnd uint32
	Length float32
	IsClutcherWon bool
}

type ClutchPlayer struct {
	Name string
	Team string
	Health uint8
	Armor uint8
	Weapons[] string
}

type ClutchType int64
const (
	ClutchType2v1 ClutchType = 0
	ClutchType1v1 ClutchType = 1
)

var CounterTerroristsAlive uint8 = 5
var TerroristsAlive uint8 = 5

var Is2v1Clutch bool = false
var Is1v1Clutch bool = false 

var MatchDemoData DemoData
var Clutch2v1 Clutch
var Clutch1v1 Clutch

var CtTeam common.TeamState
var CtTeamMembers[]*common.Player

var TtTeam common.TeamState
var TtTeamMembers[]*common.Player

const Min2v1ClutchLength float32 = 5
const Max2v1ClutchLength float32 = 30
const Max1v1ClutchLength float32 = 15

var Clutches []Clutch

func main() {
	f, err := os.Open("---")
	if err != nil {
		log.Panic("failed to open demo file: ", err)
	}
	defer f.Close()

	p := dem.NewParser(f)
	defer p.Close()
	
	// RoundStart
	p.RegisterEventHandler(func (e events.RoundStart) {
		fmt.Printf("\n")
		fmt.Printf("\n[ROUND START]\n")
		DefualtVariablesOnRoundStart(p)
	})

	// PlayerKill
	p.RegisterEventHandler(func (e events.Kill) {
		switch e.Victim.Team {
			case common.TeamCounterTerrorists:
				CounterTerroristsAlive--
				RemovePlayerFromMembers(CtTeam, e.Victim.Name)
			case common.TeamTerrorists: 
				TerroristsAlive--
				RemovePlayerFromMembers(TtTeam, e.Victim.Name)
		}

		var teamInAdvantage string
		var diff uint8
		if CounterTerroristsAlive > TerroristsAlive {
			diff = CounterTerroristsAlive - TerroristsAlive
			teamInAdvantage = "CT"
		} else if TerroristsAlive > CounterTerroristsAlive {
			diff = TerroristsAlive - CounterTerroristsAlive
			teamInAdvantage = "TT"
		} else {
			diff = 0
			teamInAdvantage = "EQUAL"
		}

		fmt.Printf("[PLAYER KILL] [CT-ALIVE]: %v - %v: [TT-ALIVE] [DIFF: %v, LEADING: %v]", CounterTerroristsAlive, TerroristsAlive, diff, teamInAdvantage)	
		if TerroristsAlive == 1 || CounterTerroristsAlive == 1 {
			if diff <= 1 { // 2vs1
				if Clutch2v1.TickStart == 0 {
					switch (teamInAdvantage) {
						case "CT": 
							GetClutchPlayerInfo(TtTeamMembers, "Terrorists", 0)
							GetEnemiesPlayersInfo(CtTeamMembers, "Counter Terrorists", 0)
						case "TT": 
							GetClutchPlayerInfo(CtTeamMembers, "Counter Terrorists", 0)
							GetEnemiesPlayersInfo(TtTeamMembers, "Terrorists", 0)
					}
					fmt.Printf("\n -- [CLUTCH START 2vs1] - [ALONE PLAYER]: %v - [ENEMIES]: %v", Clutch2v1.Clutcher, Clutch2v1.Enemies)
					Clutch2v1.TickStart = GetGameTickFromTime(p.CurrentTime())
					Is2v1Clutch = true
				}
			}
		}

		if TerroristsAlive == 1 && CounterTerroristsAlive == 1 { // 1vs1
			Clutch1v1.TickStart = GetGameTickFromTime(p.CurrentTime())
			Clutch2v1.TickEnd = Clutch1v1.TickStart
			Is1v1Clutch = true
			
			switch (Clutch2v1.Clutcher.Team) {
				case "Counter Terrorists": 
					GetClutchPlayerInfo(CtTeamMembers, "Counter Terrorists", 1)
					GetEnemiesPlayersInfo(TtTeamMembers, "Terrorists", 1)
				case "Terrorists": 
					GetClutchPlayerInfo(TtTeamMembers, "Terrorists", 1)
					GetEnemiesPlayersInfo(CtTeamMembers, "Counter Terrorists", 1)
			}
			fmt.Printf(" -- [CLUTCH START 1vs1] - [ALONE PLAYER]: %v - [ENEMIES]: %v \n", Clutch1v1.Clutcher, Clutch1v1.Enemies) 
		}

		fmt.Printf("\n")
	})

	// RoundEnd
	p.RegisterEventHandler(func (e events.RoundEnd) {
		var gs dem.GameState = p.GameState()

		var roundEndTick uint32 = GetGameTickFromTime(p.CurrentTime())

		if Clutch2v1.TickStart != 0 && Clutch1v1.TickStart == 0 {
			Clutch2v1.TickEnd = roundEndTick
		}

		if (Clutch1v1.TickStart != 0) {
			Clutch1v1.TickEnd = roundEndTick
		}
		var winnerTeam string
		switch (e.Winner) {
			case common.TeamCounterTerrorists: winnerTeam = "Counter Terrorists"
			case common.TeamTerrorists: winnerTeam = "Terrorists"
			default: winnerTeam = "Tie"
		}
		IsClucherWon(winnerTeam)
		fmt.Printf("\n[ROUND END - %v win] [CT]: %v - %v : [TT]\n", winnerTeam, gs.TeamCounterTerrorists().Score(), gs.TeamTerrorists().Score())

		// No 2vs1 & 1vs1 clutch
		if reflect.DeepEqual(Clutch2v1, Clutch{}) && reflect.DeepEqual(Clutch1v1, Clutch{}) {
			fmt.Printf("No 1vs1 & 2vs1 clutch in this round")
			return
		}
		//fmt.Printf("\n[CLUTCH START 2vs1]: %v, [CLUTCH END]: %v \n", Clutch2v1.TickStart, Clutch2v1.TickEnd)
		//fmt.Printf("[CLUTCH START 1vs1]: %v, [CLUTCH END]: %v \n", Clutch1v1.TickStart, Clutch1v1.TickEnd)

		// Get 2vs1 clutch length in seconds
		if Clutch1v1.TickEnd != 0 {
			Clutch2v1.Length = GetLengthIsSeconds(Clutch2v1.TickStart, Clutch1v1.TickEnd)
		} else {
			Clutch2v1.Length = GetLengthIsSeconds(Clutch2v1.TickStart, roundEndTick)
		}

		if Clutch2v1.Length < 5 {
			fmt.Printf("[CLUTCH 2vs1] Length is too short (%v s) - skip. \n", Clutch2v1.Length)
			return
		} else if Clutch2v1.Length > Min2v1ClutchLength && Clutch2v1.Length < Max2v1ClutchLength {
			fmt.Printf("[CLUTCH 2vs1] Length is OK (%v) - save this as clip: %v \n", Clutch2v1.Length, Clutch2v1)
			Clutches = append(Clutches, Clutch2v1)
			return
		} else if Clutch2v1.Length > Max2v1ClutchLength {

			if Is1v1Clutch {
				Clutch1v1.Length = GetLengthIsSeconds(Clutch1v1.TickStart, Clutch1v1.TickEnd)
				fmt.Printf("[CLUTCH 2vs1] Length is too long (%v) - try to save as 1vs1 clip\n", Clutch2v1.Length)

				if Clutch1v1.Length < Max1v1ClutchLength {
					fmt.Printf("[CLUTCH 1vs1] Length is OK (%v) - no need to cut, save as clip: %v\v\n", Clutch1v1.Length, Clutch1v1)
					Clutches = append(Clutches, Clutch1v1)
					return
				} else {
					var oldClutchLength float32 = Clutch1v1.Length
					Clutch1v1.TickStart = Clutch1v1.TickEnd - 960
					Clutch1v1.Length = GetLengthIsSeconds(Clutch1v1.TickStart, Clutch1v1.TickEnd)
					fmt.Printf("[CLUTCH 1vs] Length was too long (%v) - now length is OK (%v) - save as clip: %v \n", oldClutchLength, Clutch1v1.Length, Clutch1v1)
					Clutches = append(Clutches, Clutch1v1)
					return
				}

			} else {
				fmt.Printf("[CLUTCH 2vs1] Length is too long (%v) - should be saved as 1vs1 clip but there was no 1vs1 clutch\n", Clutch2v1.Length)
			}
		}
	})
	
	err = p.ParseToEnd()
	if err != nil {
		log.Panic("failed to parse demo: ", err)
	}

	MatchDemoData.Clutches = Clutches
	MatchDemoData.MapName = p.Header().MapName
	MatchDemoData.Filename = f.Name()
	WriteClutchesToJson()
}

func GetGameTickFromTime(time time.Duration) uint32 {
	// + 32 to just push a little bit forward
	return uint32 (time.Seconds() * 64) + 32 
}

func GetLengthIsSeconds(start uint32, end uint32) float32 {
	return (float32(end) - float32(start)) / 60
}

func RemovePlayerFromMembers(team common.TeamState, player string) {

	switch (team.Team()) {
		case common.TeamCounterTerrorists: 
			var newTeamMembers []*common.Player
			for i := 0; i < len(CtTeamMembers); i++ {
				if CtTeamMembers[i].Name == player {
					continue
				}
				newTeamMembers = append(newTeamMembers, CtTeamMembers[i])
			}
			CtTeamMembers = newTeamMembers

		case common.TeamTerrorists: 
			var newTeamMembers []*common.Player
			for i := 0; i < len(TtTeamMembers); i++ {
				if TtTeamMembers[i].Name == player {
					continue
				}
				newTeamMembers = append(newTeamMembers, TtTeamMembers[i])
			}
			TtTeamMembers = newTeamMembers
		}
}

func GetClutchPlayerInfo(team []*common.Player, clutcherTeam string, clutchType ClutchType) {
	for i := 0; i < len(team); i++ {
		var teamMember *common.Player = team[i]
		if teamMember.IsAlive() {
			switch (clutchType) {
				case 0: Clutch2v1.Clutcher = ClutchPlayer{Name: teamMember.Name, Team: clutcherTeam, Health: uint8(teamMember.Health()), Armor: uint8(teamMember.Armor()), Weapons: GetWeapons(teamMember.Weapons())}	
				case 1: Clutch1v1.Clutcher = ClutchPlayer{Name: teamMember.Name, Team: clutcherTeam, Health: uint8(teamMember.Health()), Armor: uint8(teamMember.Armor()), Weapons: GetWeapons(teamMember.Weapons())}
			}
			break
		}
	}
}

func GetWeapons (csWeapons[] *common.Equipment) []string {
	var weapons[] string
	for i := 0; i < len(csWeapons); i++ {
		var csWeapon *common.Equipment = csWeapons[i]
		if csWeapon.Type.String() == "Knife" {
			continue
		}
		weapons = append(weapons, csWeapon.Type.String())
	}
	return weapons
}

func GetEnemiesPlayersInfo(team []*common.Player, enemyTeam string, clutchType ClutchType) {
	for i := 0; i < len(team); i++ {
		var teamMember *common.Player = team[i]
		if teamMember.IsAlive() {
			switch (clutchType) {
			case 0: 
				Clutch2v1.Enemies = append(Clutch2v1.Enemies, ClutchPlayer{Name: teamMember.Name, Team: enemyTeam, Health: uint8(teamMember.Health()), Armor: uint8(teamMember.Armor()), Weapons: GetWeapons(teamMember.Weapons())})
			case 1: 
				Clutch1v1.Enemies = append(Clutch1v1.Enemies, ClutchPlayer{Name: teamMember.Name, Team: enemyTeam, Health: uint8(teamMember.Health()), Armor: uint8(teamMember.Armor()), Weapons: GetWeapons(teamMember.Weapons())})
			}
		}
	}
}

func IsClucherWon(winnerTeam string) {
	if winnerTeam == Clutch2v1.Clutcher.Team {
		Clutch2v1.IsClutcherWon = true
		Clutch1v1.IsClutcherWon = true
	} else {
		Clutch2v1.IsClutcherWon = false
		Clutch1v1.IsClutcherWon = false
	}
}

func DefualtVariablesOnRoundStart(p dem.Parser) {
	CounterTerroristsAlive, TerroristsAlive = 5, 5
	Is2v1Clutch, Is1v1Clutch = false, false
	MatchDemoData = DemoData{}
	Clutch2v1, Clutch1v1 = Clutch{}, Clutch{}
	Clutch2v1.ClutchType, Clutch1v1.ClutchType = "2vs1", "1vs1"
	CtTeam = *p.GameState().TeamCounterTerrorists()
	CtTeamMembers = CtTeam.Members()
	TtTeam = *p.GameState().TeamTerrorists()
	TtTeamMembers = TtTeam.Members()
}


func WriteClutchesToJson() {
		jsonData, err := json.MarshalIndent(MatchDemoData, "", "  ")
		if err != nil {
			log.Fatalf("Error serializing to JSON: %v", err)
		}
		err = os.WriteFile(MatchDemoData.Filename + ".json", jsonData, 0644)
		if err != nil {
			log.Fatalf("Error writing JSON to file: %v", err)
		}
}