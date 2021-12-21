package main

import (
	"fmt"
	"log"
	"math/rand"
)
func card_name(level int32)string{
	switch level {
	case 1:
		return "愤怒编织者"
	case 2:
		return "无私的英雄"
	case 3:
		return "灵魂杂耍者"
	case 4:
		return "阿古斯防御者"
	case 5:
		return "致命的孢子"
	case 6:
		return "海鲜投喂手"
	default:
		return "融合怪"
	}
}
func main() {
	hero_card := make([]*BattleHero, 8)
	ato := 1
	for i := 0; i < 8; i++ {
		hero_card[i] = &BattleHero{
			Id:       int64(1000 + ato),
			HeroCoin: 2,
			Health:   0,
			Armor:    0,
			Remark:   "",
		}
		ato++
	}
	ato = 1
	card_list := make(map[int32]*BattleCard, 150)
	for i := 0; i < 150; i++ {
		star := rand.Intn(5)
		sneer := false
		if rand.Intn(1) > 0 {
			sneer = true
		}
		card_list[int32(1000+ato)] = &BattleCard{
			BattleAddnToSteady: false,
			Id:                 int32(1000 + ato),
			StarLevel:          int32(star + 1),
			Name:card_name(int32(star + 1)),
			BattleAddn: &BattleCardProperty{
				Health:  int32(rand.Intn(10)),
				Atk:     int32(rand.Intn(10)),
				IsSneer: sneer,
				Addn: map[int32]*AddnDetail{},
			},
		}
		if i%2==0{
			card_list[int32(1000+ato)].BattleAddn.Skills = map[int32]int8{1000010:1,1000020:1}
		}else{
			card_list[int32(1000+ato)].BattleAddn.Skills = map[int32]int8{1000010:1,1000020:1}
		}
		ato++
	}
	battlers := make([]*Battler,8)
	for i := 0; i < 8; i++ {
		battlers[i] = BattlerCreate()
	}
	room := CreatePubRoom(battlers,hero_card,card_list)
	//for _, battler := range room.Battlers {
	//	battler.Harm = 39
	//}
	for {
		old_state := room.State.Load()
		if room.Fsm() {
			log.Println(fmt.Sprintf("room[%d]from[%d]to[%d]", room.Id, old_state, room.State.Load()))
			if room.State.Load() == PubRoomState_Battle {
				for _, battler := range room.Battlers {
					fmt.Printf(fmt.Sprintf("======玩家[%d]===出战卡牌", battler.Id))
					for _, card := range battler.AideCard {
						fmt.Print(card.Name)
					}
				}
				fmt.Println("==========")
			}
			if room.State.Load() == PubRoomState_Ready {
				for _, battler := range room.Battlers {
					if rand.Intn(1) > 0 {
						if err := battler.PubUp(); err != nil {
							log.Println(err.Error())
						}
					}
					//tlog.Info("当前用户金币余额:", room.Round, battler.Player.Base.Id, battler.Level, battler.Coin)
					if err := battler.CardBuy(battler.PubCard[0].Id); err == nil {
						battler.CardUse(battler.BattleCard[0].Id, 0)
					}
				}
			}
		}
		if room.Rank.Load() <= 1 {
			return
		}
	}
}
