package main

import (
	"fmt"
	"log"
	"math/rand"
)

func card_name(level int32) string {
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
			Health:   40,
			Armor:    0,
			Remark:   "",
		}
		ato++
	}
	ato = 1
	card_list := make(map[int32]*BattleCard, 150)
	for i := 0; i < 450; i++ {

		sneer := false
		if rand.Intn(1) > 0 {
			sneer = true
		}
		StarLevel := int32(1)
		if i/200 > 1 {
			StarLevel = int32(rand.Intn(5) + 1)
		}
		card_list[int32(1000+ato)] = &BattleCard{
			BattleAddnToSteady: false,
			Id:                 int32(1000 + ato),
			StarLevel:          StarLevel,
			Name:               card_name(StarLevel),
			BattleAddn: &BattleCardProperty{
				Health:  int32(rand.Intn(10)),
				Atk:     int32(rand.Intn(10)),
				IsSneer: sneer,
				Addn:    map[int32]*AddnDetail{},
			},
		}
		if i%3 == 0 {
			card_list[int32(1000+ato)].BattleAddn.Skills = map[int32]int8{1000010: 1, 1000020: 1}
		} else if i%5 == 0 {
			card_list[int32(1000+ato)].BattleAddn.Skills = map[int32]int8{1000010: 1, 1000020: 1}
		}
		ato++
	}
	battlers := make([]*Battler, 8)
	for i := 0; i < 8; i++ {
		battlers[i] = BattlerCreate()
	}
	room := CreatePubRoom(battlers, hero_card, card_list)
	for _, battler := range room.Battlers {
		battler.Harm = 30
	}
	//for _, battler := range room.Battlers {
	//	go func() {
	//		for{
	//			log.Println("=11111===")
	//			select {
	//			case step := <-battler.RoundChan:
	//				log.Println(step.Type,step.Attacker,step.Defender)
	//				if battler.State.Load() == BattlerState_Over {
	//					return
	//				}
	//			case <-time.After(200*time.Millisecond):
	//				//log.Println("=222===")
	//				if battler.State.Load() == BattlerState_Over {
	//					return
	//				}
	//			}
	//		}
	//	}()
	//}
	for {
		old_state := room.State.Load()
		if room.Fsm() {
			log.Println(fmt.Sprintf("room[%d]from[%d]to[%d]", room.Id, old_state, room.State.Load()))
			//if room.State.Load() == PubRoomState_Battle {
			//	for _, battler := range room.Battlers {
			//		//fmt.Printf(fmt.Sprintf("======玩家[%d]===出战卡牌", battler.Id))
			//		for _, card := range battler.AideCard {
			//			fmt.Print(card.Name)
			//		}
			//	}
			//	//fmt.Println("==========")
			//}
			if room.State.Load() == PubRoomState_Ready {
				for _, battler := range room.Battlers {
					if rand.Intn(1) > 0 {
						if err := battler.PubUp(); err != nil {
							log.Println(err.Error())
						}
					}
					//tlog.Info("当前用户金币余额:", room.Round, battler.Player.Base.Id, battler.Level, battler.Coin)
					card := battler.PubCard.Get(0)
					if card == nil {
						//log.Println("牌位nil")
					} else {
						if err := battler.CardBuy(card.Id); err == nil {
							battler.CardUse(battler.BattleCard.Get(0).Id, 0)
						}
					}
				}
			}
		}
		if room.Rank.Load() <= 1 {
			return
		}
	}
}
