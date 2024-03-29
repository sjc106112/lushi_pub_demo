package main

import (
	"fmt"
	"go.uber.org/atomic"
	"log"
	"math/rand"
	"sync"
	"time"
)

func CreatePubRoom(battlers []*Battler, heros []*BattleHero, card_map map[int32]*BattleCard) *PubRoom {
	room := &PubRoom{
		Battlers:  make([]*Battler, 0, len(battlers)),
		State:     atomic.NewInt32(PubRoomState_Init),
		Id:        time.Now().Unix(),
		StartTime: time.Now().Unix() + BattleChoseHeroTime,
		PubCoin:   []int32{6, 7, 8, 9, 10},
		GrantCoin: []int32{3, 4, 5, 6, 7, 8, 9, 10},
		CardNum:   []int{3, 4, 4, 5, 5, 6},
		cardLock:  sync.Mutex{},
		Rank:      atomic.NewInt32(int32(len(battlers)) + 1),
	}
	//TODO 初始化卡牌
	room.Cards = card_map
	for i, battler := range battlers {
		if battler != nil {
			battler.GrantCoin = room.GrantCoin
			battler.PubCoin = room.PubCoin
			battler.CardNum = room.CardNum
			battler.PubRoom = room
			battler.Id = int64(10000000 + i)
			//TODO 初始化能够选择的英雄
			battler.HeroList = make([]*BattleHero, 2)
			if len(heros) > 2 {
				idx := rand.Intn(len(heros))
				battler.HeroList[0] = heros[idx]
				heros = append(heros[:idx], heros[idx+1:]...)
				idx = rand.Intn(len(heros))
				battler.HeroList[1] = heros[idx]
				heros = append(heros[:idx], heros[idx+1:]...)
			} else {
				battler.HeroList = heros
			}
			room.Battlers = append(room.Battlers, battler)
			Player2Room[battler.Id] = battler
			log.Println(fmt.Sprintf("add pub room [%d,%d]", battler.Id, battler.PubRoom.Id))
		}
	}
	return room
}

type PubRoom struct {
	StartTime int64                 //开始时间
	State     *atomic.Int32         //状态 0 未开局 1 准备中 2 战斗中
	Id        int64                 //房间id
	Round     int                   //轮次
	Battlers  []*Battler            //开局所有玩家
	PubCoin   []int32               //升级酒馆花费的金币 下标代表当前等级升级需要的金币
	GrantCoin []int32               //每次开局给予的金币
	CardNum   []int                 //每次开局刷新随从的数量
	HeroList  []int64               //可选英雄
	Battles   []*Battle             //战斗列表
	Cards     map[int32]*BattleCard //当前牌局剩余的牌 等级以及对应的卡牌
	Remark    string                //族群 描述
	cardLock  sync.Mutex            //刷新的卡锁
	Rank      *atomic.Int32         //排名
}

func (this *PubRoom) Fsm() bool {
	if this.Rank.Load() <= 2 {
		for _, battler := range this.Battlers {
			if battler.State.Load() != BattlerState_Over {
				battler.LeaveRoom()
			}
		}
		return false
	}
	switch this.State.Load() {
	case PubRoomState_Init:
		return this.Open()
	case PubRoomState_Ready:
		return this.Battle()
	case PubRoomState_Battle:
		return this.Ready()
	}
	return false
}

//开局
func (this *PubRoom) Open() bool {
	if time.Now().Unix() < this.StartTime {
		for _, battler := range this.Battlers {
			if battler.State.Load() == BattlerState_Init {
				return false
			}
		}
	}
	if this.State.CAS(PubRoomState_Init, PubRoomState_Doing) {
		this.Round = 0 //第一局
		this.StartTime = time.Now().Unix() + BattleReadyStepTime
		for _, battler := range this.Battlers {
			if battler.State == nil || battler.State.Load() == BattlerState_Init {
				//TODO 判断用户是否选择了英雄,未选择,随机一个
				if battler.Hero == nil {
					battler.HeroSet(battler.HeroList[rand.Intn(1)])
				}
			}
			//触发英雄技能 开局
			battler.Open()
			battler.Ready()
			//TODO 判断用户是否在线
		}
		//TODO 匹配对战并排序
		this._domatch2battle()
		this.State.CAS(PubRoomState_Doing, PubRoomState_Ready)
		return true
	}
	if this.State.Load() == PubRoomState_Ready {
		return true
	} else {
		return false
	}
}

//开始准备阶段
func (this *PubRoom) Ready() bool {
	//if time.Now().Unix() < this.StartTime {
	for _, battle := range this.Battles {
		if battle.State != BattleState_Finish {
			return false
		}
	}
	//return false
	//}
	if this.State.CAS(PubRoomState_Battle, PubRoomState_Doing) {
		this.Round = this.Round + 1
		this.StartTime = time.Now().Unix() + BattleReadyStepTime
		for _, battler := range this.Battlers {
			if battler.State.Load() == BattlerState_Over {
				continue
			}
			if battler.Hero.Armor+battler.Hero.Health <= battler.Harm {
				battler.LeaveRoom()
				continue
			}
			if this.Rank.Load() <= 1 {
				battler.LeaveRoom()
				return true
			}
			battler.Ready()
		}
		this._domatch2battle()
		this.State.CAS(PubRoomState_Doing, PubRoomState_Ready)
		return true
	}
	return false
}

//开始战斗阶段
func (this *PubRoom) Battle() bool {
	if time.Now().Unix() < this.StartTime {
		return false
	}
	if this.State.CAS(PubRoomState_Ready, PubRoomState_Doing) {
		this.StartTime = time.Now().Unix() + BattleBattleStepTime
		log.Println(fmt.Sprintf("战斗开始，总战局 %d", len(this.Battles)))
		this.State.CAS(PubRoomState_Doing, PubRoomState_Battle)
		go func() {
			for _, battle := range this.Battles {
				battle.Attacker.Battle()
				battle.Defender.Battle()
				battle._doBattle()
			}
		}()
		return true
	}

	//if this.State.Load() == PubRoomState_Battle {
	//	return true
	//} else {
	//	return false
	//}
	return false
}

//战斗匹配
func (this *PubRoom) _domatch2battle() {
	//TODO 按照血量排序
	var less = func(battler1 *Battler, battler2 *Battler) bool {
		if battler1.Rank == battler2.Rank {
			return (battler1.Hero.Health + battler1.Hero.Armor) <= (battler2.Hero.Health + battler2.Hero.Armor)
		}
		return battler1.Rank > battler2.Rank
	}
	var swap = func(data []*Battler, i int, j int) {
		temp := data[j-1]
		data[j-1] = data[j]
		data[j] = temp
	}
	//for i := 4; i < len(this.Battlers); i++ {
	//	if less(this.Battlers[i-4], this.Battlers[i]) {
	//		swap(this.Battlers, i-4, i)
	//	}
	//}
	for i := 1; i < len(this.Battlers); i++ {
		//if this.Battlers[i].Hero.Health-this.Battlers[i].Harm <= 0 {
		//	break
		//}
		for j := i; j > 0 && less(this.Battlers[j-1], this.Battlers[j]); j-- {
			swap(this.Battlers, j-1, j)
		}
	}
	//匹配对手
	this.Battles = make([]*Battle, 0, 4)
	var lived []*Battler
	for i := len(this.Battlers) - 1; i >= 0; i-- {
		if this.Battlers[i].State.Load() == BattlerState_Ready {
			if (i & 1) == 0 {
				lived = this.Battlers[:i]
			} else {
				lived = this.Battlers[:i+1]
				lived[len(lived)-1].Rival = nil
			}
			break
		}
	}
	if len(lived) == 2 {
		lived[0].Rival = lived[1]
		lived[1].Rival = lived[0]
		this.Battles = append(this.Battles, &Battle{
			State:    BattleState_Battling,
			Attacker: lived[0],
			Defender: lived[0].Rival,
			Winner:   nil,
			Harm:     0,
		})
	} else {
		for _, battler := range lived {
			//TODO 需要优化
			if battler.Rival == nil {
				this._domatch(battler, lived)
			}
		}
	}
	return
}

func (this *PubRoom) _domatch(battler *Battler, lived []*Battler) {
	var rival *Battler
	for _, b := range lived {
		if b.Rival == nil && b != battler {
			rival = b
			if rand.Intn(len(lived)) > 0 {
				break
			}
		}
	}
	if rival == nil {
		log.Println(fmt.Sprintf("匹配对战失败%d", len(lived)))
		return
	}
	battler.Rival = rival
	rival.Rival = battler
	current := &Battle{
		State:    BattleState_Battling,
		Attacker: battler,
		Defender: rival,
		Winner:   nil,
		Harm:     0,
	}
	this.Battles = append(this.Battles, current)
	current.Attacker.Battles = append(current.Attacker.Battles, current)
	current.Defender.Battles = append(current.Defender.Battles, current)
}

func (this *PubRoom) CardRefresh(battler *Battler, nums int) []*BattleCard {
	this.cardLock.Lock()
	defer this.cardLock.Unlock()
	card_arr := make([]*BattleCard, 0, len(this.Cards))
	for i := range this.Cards {
		if this.Cards[i].StarLevel > (battler.Level + 1) {
			continue
		}
		card_arr = append(card_arr, this.Cards[i])
	}
	result := make([]*BattleCard, nums)
	if card_arr == nil || len(card_arr) <= 0 {
		log.Println("not found refresh card")
		return nil
	}
	for i := 0; i < nums; i++ {
		if len(card_arr) > 1 {
			idx := rand.Intn(len(card_arr))
			result[i] = card_arr[idx]
			card_arr = append(card_arr[:idx], card_arr[idx+1:]...)
			delete(this.Cards, result[i].Id)
		} else {
			result[i] = card_arr[0]
			delete(this.Cards, result[i].Id)
			break
		}
	}
	return result
}

func (this *PubRoom) CardRefreshByLevel(nums int, level int32) []*BattleCard {
	this.cardLock.Lock()
	defer this.cardLock.Unlock()
	card_arr := make([]*BattleCard, 0, len(this.Cards))
	for i := range this.Cards {
		if this.Cards[i].StarLevel == level {
			card_arr = append(card_arr, this.Cards[i])
		}
	}
	result := make([]*BattleCard, nums)
	for i := 0; i < nums; i++ {
		idx := rand.Intn(len(card_arr))
		result[i] = card_arr[idx]
		card_arr = append(card_arr[:idx], card_arr[idx+1:]...)
		delete(this.Cards, result[i].Id)
	}
	return result
}

func (this *PubRoom) CardReturn(cards ...*BattleCard) {
	this.cardLock.Lock()
	defer this.cardLock.Unlock()
	for _, card := range cards {
		if card != nil {
			card.Owner = nil
			this.Cards[card.Id] = card
		}
	}
}

type Battle struct {
	State    int8     //1 战斗中 2 战斗完成
	Attacker *Battler //攻方
	Defender *Battler //防守
	Winner   *Battler //胜者
	Harm     int32    //造成的伤害
}

func (this *Battle) _doBattle() {
	if this.Defender.AideCard.IsEmpty() && this.Attacker.AideCard.IsEmpty() {
		this.State = BattleState_Finish
		//TODO 添加 RoundChan
		log.Println("战斗结束 [%d,%d],伤害[%d]", this.Attacker.Id, this.Defender.Id, this.Harm)
		//TODO Type类型未定义？
		//round_step := &RoundStep{
		//	Type: RoundStepType_BattleOver,
		//	Harm: 0,
		//}
		//if this.Attacker.State.Load() != BattlerState_Over {
		//	this.Attacker.RoundChan <- round_step
		//}
		//if this.Defender.State.Load() != BattlerState_Over {
		//	this.Defender.RoundChan <- round_step
		//}
		return
	}
	if this.Defender.AideCard.Size() > this.Attacker.AideCard.Size() {
		temp := this.Attacker
		this.Attacker = this.Defender
		this.Defender = temp
	} else if this.Defender.AideCard.Size() == this.Attacker.AideCard.Size() && rand.Intn(1) == 1 {
		temp := this.Attacker
		this.Attacker = this.Defender
		this.Defender = temp
	}
	var round1 = 1
	var round2 = 1
	var idx1 = 0
	var idx2 = 0
	for {
		idx1 = this._do_attack(round1, idx1)
		if this.State == BattleState_Finish {
			return
		}
		if idx1 < 0 {
			idx1 = 0
			round1++
		}
		temp := this.Attacker
		this.Attacker = this.Defender
		this.Defender = temp
		idx2 = this._do_attack(round2, idx2)
		if idx2 < 0 {
			round2++
			idx2 = 0
		}
		if this.State == BattleState_Finish {
			return
		}
	}
}

/**
 * 递归攻击
 */
func (this *Battle) _do_attack(round int, idx int) int {
	if this.Defender.AideCard == nil || this.Defender.AideCard.IsEmpty() {
		this.State = BattleState_Finish
		if !this.Attacker.AideCard.IsEmpty() {
			this.Winner = this.Attacker
			this.Harm = this.Attacker.Atk
			this.Attacker.AideCard.Iteration(func(idx int, card *BattleCard) bool {
				this.Harm += card.StarLevel
				return true
			})
			if this.Defender.State.Load() != BattlerState_Over {
				this.Defender.Harm += this.Harm
				if this.Defender.Hero.Health <= this.Defender.Harm {
					this.Defender.LeaveRoom()
				}
			}
		}
		//TODO Type类型未定义？
		//round_step := &RoundStep{
		//	Type:   RoundStepType_BattleOver,
		//	Harm:   this.Harm,
		//	Winner: this.Winner,
		//}
		//if this.Attacker.State.Load() != BattlerState_Over {
		//	this.Attacker.RoundChan <- round_step
		//}
		//if this.Defender.State.Load() != BattlerState_Over {
		//	this.Defender.RoundChan <- round_step
		//}
		if this.Winner != nil {
			log.Println(fmt.Sprintf("战斗结束 [%d,%d],伤害[%d,%d,%d],胜者[%d]", this.Attacker.Id, this.Defender.Id, this.Harm, this.Defender.Harm, this.Defender.Hero.Health, this.Winner.Id))
		} else {
			log.Println(fmt.Sprintf("战斗结束 [%d,%d],伤害[%d,%d,%d]", this.Attacker.Id, this.Defender.Id, this.Defender.Harm, this.Defender.Hero.Health, this.Harm))
		}
		return 0
	}
	i, attacker := this.Attacker.AideCard.Iteration(func(idx int, card *BattleCard) bool {
		if card.BattleAddn.Round != int32(round) {
			return false
		}
		return true
	})
	if attacker != nil {
		if attacker.Owner != this.Attacker {
			log.Println(fmt.Sprintf("卡牌错误 %d %d %d", attacker.Id, attacker.Owner.Id, this.Attacker.Id))
		}
		attacker.Attack(this.Defender)
		attacker.BattleAddn.Round = int32(round)
		return i
	}
	return -1
}
