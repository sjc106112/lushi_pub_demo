package main

import (
	"sync"
)

const (
	BattlerState_Init    = 0
	BattlerState_Ready   = 1
	BattlerState_Battle  = 2
	BattlerState_Offline = 3
	BattlerState_Over    = 4

	PubRoomState_Init   = 0
	PubRoomState_Ready  = 1
	PubRoomState_Battle = 2
	PubRoomState_Doing  = 999

	HeroSkillStage_Open          = 101 //英雄技能 开局(被动)
	HeroSkillStage_Ready         = 102 //英雄技能 准备阶段 主动
	HeroSkillStage_Battle        = 103 //英雄技能 战斗开局
	HeroSkillStage_BattleFinish  = 104 //英雄技能 战斗结束
	HeroSkillStage_ReadyPassive  = 105 //英雄技能 准备阶段(被动)
	HeroSkillStage_BattlePassive = 106 //英雄技能 战斗开局
	HeroSkillStage_Pubup         = 107 //酒馆升级后触发英雄技能
	HeroSkillStage_CardRefresh   = 108 //酒馆升级后触发英雄技能
	HeroSkillStage_CallCard      = 109 //召唤卡牌后触发英雄技能

	CardSkillStage_Buy         = 201 //卡牌买
	CardSkillStage_Use         = 202 //卡牌战吼
	CardSkillStage_Ready       = 203 //卡牌准备阶段触发
	CardSkillStage_Sell        = 204 //卡牌卖出
	CardSkillStage_Battle      = 205 //战斗阶段开始
	CardSkillStage_Die         = 206 //卡牌亡语
	CardSkillStage_AttackBefor = 207 //卡牌攻击前
	CardSkillStage_AttackAfter = 208 //卡牌攻击后
	CardSkillStage_Pubup       = 209 //酒馆升级后触发
	CardSkillStage_DefendBefor = 210 //卡牌被攻击前
	CardSkillStage_DefendAfter = 211 //卡牌被攻击后
	CardSkillStage_AddnHealth  = 212 //卡牌加成生命
	CardSkillStage_AddnAtk     = 213 //卡牌加成攻击
	CardSkillStage_Addn        = 214 //卡牌加成攻击或者生命

	BattleState_Battling = 0
	BattleState_Finish   = 1

	BattleChoseHeroTime  = 5
	BattleReadyStepTime  = 10
	BattleBattleStepTime = 20

	CardRefreshCostCoin = 1 //刷新卡牌花费
	CardBuyCostCoin     = 3 //买卡牌花费
	CardSellCoin        = 1 //卖卡牌得到
	SkillType_Hero      = 1
	SkillType_Card      = 2

	RoundStepType_GeneralAttack = 1 //普通攻击
	RoundStepType_BattleOver    = 2 //战斗完成
)

var Player2Room = make(map[int64]*Battler)

type CardArray struct {
	data []*BattleCard
	lock sync.Mutex
}

func CardArrayCreate(size int) *CardArray {
	if size == 0 {
		size = 8
	}
	card_array := &CardArray{
		data: make([]*BattleCard, 0, size),
		lock: sync.Mutex{},
	}
	return card_array
}

func CardArrayCreateByArray(array []*BattleCard) *CardArray {
	card_array := &CardArray{
		data: array,
		lock: sync.Mutex{},
	}
	return card_array
}

func CardArrayCopy(card_array *CardArray) *CardArray {
	new_array := &CardArray{
		lock: sync.Mutex{},
		data: make([]*BattleCard, len(card_array.data)),
	}
	copy(new_array.data, card_array.data)
	return card_array
}

func (this *CardArray) Copy(card_array *CardArray) {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.data = make([]*BattleCard, 0, card_array.Size())
	card_array.Iteration(func(idx int, card *BattleCard) bool {
		if card != nil {
			//log.Println(fmt.Sprintf("卡牌copy %d %d", card.Id, card.Owner.Id))
			this.data = append(this.data, card)
		}
		return false
	})
}

func (this *CardArray) Get(idx int) *BattleCard {
	if this.IsEmpty() || this.Size() < idx {
		return nil
	}
	return this.data[idx]
}

func (this *CardArray) IsEmpty() bool {
	if this.data == nil || this.Size() <= 0 {
		return true
	}
	return false
}

func (this *CardArray) Size() int {
	return len(this.data)
}

func (this *CardArray) Add(card ...*BattleCard) {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.data = append(this.data, card...)
}

func (this *CardArray) RemoveByCardId(card_id int32) (int, *BattleCard) {
	this.lock.Lock()
	defer this.lock.Unlock()
	for i, datum := range this.data {
		if card_id == datum.Id {
			this.data = append(this.data[:i], this.data[i+1:]...)
			return i, datum
		}
	}
	return -1, nil
}

func (this *CardArray) RemoveByIdx(idx int) *BattleCard {
	this.lock.Lock()
	defer this.lock.Unlock()
	card := this.data[idx]
	this.data = append(this.data[:idx], this.data[idx+1:]...)
	return card
}

func (this *CardArray) Insert(card *BattleCard, idx int) {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.data = append(this.data, nil)
	copy(this.data[idx+1:], this.data[idx:])
	this.data[idx] = card
}

func (this *CardArray) Move(idx int, moved_idx int) {
	this.lock.Lock()
	defer this.lock.Unlock()
	if idx < moved_idx {
		tmp := this.data[idx]
		copy(this.data[idx:moved_idx], this.data[idx+1:])
		this.data[moved_idx] = tmp
	} else {
		tmp := this.data[idx]
		copy(this.data[moved_idx+1:], this.data[moved_idx:idx])
		this.data[moved_idx] = tmp
	}
}

func (this *CardArray) Array() []*BattleCard {
	return this.data
}

func (this *CardArray) Iteration(callback func(idx int, card *BattleCard) bool) (int, *BattleCard) {
	if !this.IsEmpty() {
		for idx, datum := range this.data {
			returned := callback(idx, datum)
			if !returned {
				return idx, datum
			}
		}
	}
	return -1, nil
}
