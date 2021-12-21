package main

import (
	"errors"
	"fmt"
	"go.uber.org/atomic"
	"log"
	"sync"
)

type BattleHero struct {
	IsCardRefChange bool   //是否被动更改刷新卡牌规则技能
	IsSkill         bool   //是否使用了英雄技能
	Id              int64  //英雄id
	Skill           int32  //英雄技能
	HeroCoin        int32  //英雄技能发动需要的金币
	Health          int32  //英雄血量
	Armor           int32  //英雄具有的护甲量
	Remark          string //描述
}

type RoundStep struct {
	IsSneer      bool          //是否增加嘲讽
	IsSacred     bool          //是否增加圣盾
	Type         int8          // 1 攻防卡牌攻击 2 守方卡牌攻击 3 攻防英雄技能 4 守方英雄技能 5 战斗完成
	SkillId      int32         //发起的技能
	Attacker     int32         //发起方card id
	Harm         int32         //发起方收到的伤害
	AttackerHarm int32         //发起方打出的伤害
	Defender     int32         //防守方card id
	Winner       *Battler      //胜利方
	Health       int32         //加成血量
	Atk          int32         //加成的攻击
	Skill        int32         //加成的技能
	Receiver     []*BattleCard //受影响的卡牌
	CardAddn     []*BattleCard //召唤的卡牌
}

type Battler struct {
	Id int64
	IsFreeze          bool            //是否冻结当前酒馆的卡牌
	State             *atomic.Int32   //TODO 是否需要? 状态 0 未开局 1 准备中 2 战斗 3 离线 4 over
	Rank              int32           //战斗排名
	Level             int32           //酒馆级别
	Atk               int32           //攻击
	Harm              int32           //收到的伤害
	CardRefreshNum    int32           // 卡牌刷新次数
	CardWipeOutNum    int32           //卡牌消灭的数量
	CurrentCardBuyNum int32           //当前局购买的卡牌数量
	CurrentCardCost   int32           //一个技能周期花费金币的数量
	MpFreeze          int32           // 冻结酒馆卡牌花费的金币
	CardDieNum        int32           //卡牌刷新次数
	CostLevel         int32           //升级酒馆需要的金币
	CoinMax           int32           //可允许的最大金币
	Coin              int32           //剩余的金币
	CardRefreshCoin   int32           //刷新卡牌需要的金币
	CardCoin          int32           //购买卡牌需要的金币
	CardSellCoin      int32           //卖卡给的金币
	Name              string          //玩家称呼
	PubRoom           *PubRoom        //酒馆房间
	Rival             *Battler        //当前局对手方
	BattleCard        []*BattleCard   //拥有的卡牌
	AideCard          []*BattleCard   //出战的卡牌
	_AideCardBackup   []*BattleCard   //战斗中的卡牌列表
	Battles           []*Battle       //战斗列表,按照发生的先后顺序排列
	PubCard           []*BattleCard   //当前可选择的卡牌 不用map 是为了保证顺序
	PubCoin           []int32         //升级酒馆花费的金币 下标代表当前等级升级需要的金币
	GrantCoin         []int32         //每次开局给予的金币
	CardNum           []int           //每次开局刷新随从的数量
	HeroList          []*BattleHero   //可选英雄
	lock              sync.Mutex      //扣币的锁
	Hero              *BattleHero     //使用的英雄id,对应的技能加成 后期改成英雄卡牌对象
	RoundChan         chan *RoundStep //不同的玩家,取chan的效率不同
}

func BattlerCreate() *Battler {
	return &Battler{
		CardRefreshCoin: CardRefreshCostCoin,
		CardCoin:        CardBuyCostCoin, //购买卡牌需要花费的金币
		CardSellCoin:    CardSellCoin,
		State:           atomic.NewInt32(BattlerState_Init),
		Level:           0, //酒馆级别，从0开始
		PubCard:         make([]*BattleCard, 0, 6),
		BattleCard:      make([]*BattleCard, 0, 20),
		AideCard:        make([]*BattleCard, 0, 7),
		Battles:         make([]*Battle, 0, 10),
		lock:            sync.Mutex{},
		RoundChan:       make(chan *RoundStep, 20),
	}
}

//开局施放英雄技能
func (this *Battler) Open() {
	//释放英雄被动技能(开局)
	this.Atk = this.Level + 1
	this._do_hero_skill(&Event{
		Stage:    HeroSkillStage_Open,
		Attacker: this,
	})
	this.CostLevel = this.PubCoin[this.Level]
}

func (this *Battler) LeaveRoom() {
	if this.State.CAS(BattlerState_Ready, BattlerState_Over) {
		this.Rank = this.PubRoom.Rank.Sub(1)
		log.Println(fmt.Sprintf("玩家[%d]离开房间[%d],战绩[%d]", this.Id, this.PubRoom.Id, this.Rank))
		delete(Player2Room, this.Id)
		//close(this.RoundChan)
	}
}

//进入准备阶段
func (this *Battler) Ready() {
	//初始化金币
	idx := len(this.GrantCoin) - 1
	if idx > this.PubRoom.Round {
		idx = this.PubRoom.Round
	}
	this.Coin = this.GrantCoin[idx]
	this.CostLevel -= 1
	if this.CostLevel < 0 {
		this.CostLevel = 0
	}
	this.CurrentCardBuyNum = 0
	this.CurrentCardCost = 0
	//TODO 刷新卡牌,是否有被动技能更改刷新卡牌规则
	if !this.Hero.IsCardRefChange && !this.IsFreeze {
		nums := this.CardNum[len(this.CardNum)-1]
		if int(this.Level) < (len(this.CardNum) - 1) {
			nums = this.CardNum[this.Level]
		}
		this.CardRefresh(0, nums)
	}
	this.IsFreeze = false
	if this._AideCardBackup != nil && len(this._AideCardBackup) > 0 {
		for _, card := range this._AideCardBackup {
			if card.BackUp != nil {
				card.BattleAddn = card.BackUp
				card.BattleAddn.IsDie = false
				card.BattleAddn.Harm = 0
			}
		}
		this.AideCard = make([]*BattleCard, len(this._AideCardBackup))
		copy(this.AideCard, this._AideCardBackup)
	}
	//调用准备阶段英雄技能
	this._do_hero_skill(&Event{
		Stage:    HeroSkillStage_ReadyPassive,
		Attacker: this,
	})
	this._do_card_skill(nil, CardSkillStage_Ready, nil)
	this.Rival = nil
}

func (this *Battler) Battle() {
	for _, card := range this.AideCard {
		if !card.BattleAddnToSteady {
			card.BackUp = card.BattleAddn
		} else {
			card.BackUp = &BattleCardProperty{}
			*card.BackUp = *card.BattleAddn
			//card.BattleAddn.Addn = map[int32]*AddnDetail{}
		}
	}
	this._AideCardBackup = make([]*BattleCard, len(this.AideCard))
	copy(this._AideCardBackup, this.AideCard)
	//开战开始阶段英雄被动技能
	this._do_hero_skill(&Event{
		Stage:    HeroSkillStage_BattlePassive,
		Attacker: this,
	})
	//开战开始阶段卡牌技能
	this._do_card_skill(nil, CardSkillStage_Battle, nil)
}

//酒馆升级
func (this *Battler) PubUp() error {
	if int(this.Level) < len(this.PubCoin)-1 {
		this.lock.Lock()
		defer this.lock.Unlock()
		if err := this._docoin(this.CostLevel); err != nil {
			return err
		}
		this.Level += 1
		this.Atk = this.Level + 1 //TODO 英雄攻击等于酒馆等级?
		this.CostLevel = this.PubCoin[this.Level]
		//触发英雄技能
		this._do_hero_skill(&Event{
			Stage:    HeroSkillStage_Pubup,
			Attacker: this,
		})
		this._do_card_skill(nil, CardSkillStage_Pubup, nil)
		return nil
	}
	return errors.New(  "Maximum pub room level 6")
}

//使用卡牌
func (this *Battler) CardUse(card_id int32, index int, target ...*BattleCard) error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.PubRoom.State.Load() == PubRoomState_Ready {
		if index <= len(this.AideCard) && index >= 0 {
			for i, card := range this.BattleCard {
				if card.Id == card_id {
					this.BattleCard = append(this.BattleCard[:i], this.BattleCard[i+1:]...)
					card.Owner = this
					// 需要优化
					//this.AideCard = append(this.AideCard, nil)
					//copy(this.AideCard[index+1:], this.AideCard[index:])
					//this.AideCard[index] = card
					//this._do_card_skill(CardSkillStage_Use, card, target...)
					this._do_card_goout(card, index, target...)
					return nil
				}
			}
		} else {
			log.Println("invalid index[%d],aidecard[%d]", index, len(this.AideCard))
			return errors.New(  fmt.Sprintf("invalid index[%d],aidecard[%d]", index, len(this.AideCard)))
		}
		log.Println("card user not found card in battle card", card_id)
		return errors.New(  "not found card in battle card")
	}
	return errors.New( "cards cannot be used In battle")
}

//使用卡牌
func (this *Battler) CardMove(idx int, moved_idx int) {
	if idx == moved_idx {
		return
	}
	this.lock.Lock()
	defer this.lock.Unlock()

	if idx < moved_idx {
		tmp := this.AideCard[idx]
		copy(this.AideCard[idx:moved_idx], this.AideCard[idx+1:])
		this.AideCard[moved_idx] = tmp
	} else {
		tmp := this.AideCard[idx]
		copy(this.AideCard[moved_idx+1:], this.AideCard[moved_idx:idx])
		this.AideCard[moved_idx] = tmp
	}
}

//卖出卡牌
func (this *Battler) CardSell(card_id int32) error {
	this.lock.Lock()
	defer this.lock.Unlock()
	for i, card := range this.AideCard {
		if card.Id == card_id {
			this.BattleCard = append(this.AideCard[:i], this.AideCard[i+1:]...)
			//还卖的金币给用户
			if (this.Coin + this.CardSellCoin) > this.CoinMax {
				this.Coin = this.CoinMax
			} else {
				this.Coin += this.CardSellCoin
			}
			// 需要优化
			//TODO 还牌到酒馆
			this._do_card_skill(nil, CardSkillStage_Sell, card)
			return nil
		}
	}
	log.Println("card sell not found card in battle card", card_id)
	return errors.New("not found card in battle card")
}

func (this *Battler) _do_card_skill(event *Event, stage int, trigger *BattleCard, target ...*BattleCard) {
	for i, card := range this.AideCard {
		//TODO 执行 卡牌战吼技能
		//card.Skill(i, stage, trigger, target...)
		card.Skill(&Event{
			Type:     SkillType_Card,
			Card:     card,
			Attacker: this,
			Stage:    stage,
			Trigger:  trigger,
			Target:   target,
			Idx:      i,
			Parent:   event,
		})
	}
}

//设置英雄
func (this *Battler) HeroSet(hero *BattleHero) {
	if this.State.CAS(BattlerState_Init, BattlerState_Ready) {
		this.Hero = hero
		//释放英雄选择引用
		this.HeroList = nil
		this.PubRoom.Fsm()
	}
}

//主动释放英雄技能 准备阶段 主动触发
func (this *Battler) HeroSkill() error {
	if !this.Hero.IsSkill {
		this.lock.Lock()
		defer this.lock.Unlock()
		if !this.Hero.IsSkill {
			if err := this._docoin(this.Hero.HeroCoin); err != nil {
				return err
			}
			this.Hero.IsSkill = true
			//TODO 执行 英雄技能(立即触发的)
			this._do_hero_skill(&Event{
				Stage:    HeroSkillStage_Ready,
				Attacker: this,
			})
			//steps := _do_hero_skill_handle(this.Hero.Skill, HeroSkillStage_Ready, this, nil)
			//if steps != nil && len(steps) > 0 {
			//	for _, step := range steps {
			//		this.RoundChan <- &RoundStep{
			//			Type:     step.Type,
			//			SkillId:  step.SkillId,
			//			Attacker: step.Attacker,
			//			Harm:     step.Harm,
			//			Health:   step.Health,
			//			Atk:      step.Atk,
			//			Skill:    step.Skill,
			//			Receiver: step.Receiver,
			//		}
			//	}
			//}
		}
		return nil
	}
	return nil
}

/**
该方法会把剩余的牌还回去
coin 刷新卡牌花费的金币
刷新卡牌的数量
*/
func (this *Battler) CardRefresh(coin int32, nums int) error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if coin > 0 {
		if err := this._docoin(coin); err != nil {
			return err
		}
	}
	this.PubRoom.CardReturn(this.PubCard)
	this.PubCard = this.PubRoom.CardRefresh(this, nums)
	this.CardRefreshNum++
	this._do_hero_skill(&Event{
		Stage:    HeroSkillStage_CardRefresh,
		Attacker: this,
	})
	return nil
}

//注意,该方法只是发牌,不还牌
func (this *Battler) CardRefreshBySkill(nums int, level int32) {
	refreshcard := this.PubRoom.CardRefreshByLevel(nums, level)
	this.PubCard = append(this.PubCard, refreshcard...)
	this.CardRefreshNum++
	//TODO 有问题?
	this._do_hero_skill(&Event{
		Stage:    HeroSkillStage_CardRefresh,
		Attacker: this,
	})
}

//买卡
func (this *Battler) CardBuy(card_id int32) error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if err := this._docoin(this.CardCoin); err != nil {
		//tlog.Errorf("buy card[%d,%d],error %s ", this.Coin, this.CardCoin, err.Error())
		return err
	}
	for i, card := range this.PubCard {
		if card.Id == card_id {
			this.PubCard = append(this.PubCard[:i], this.PubCard[i+1:]...)
			this._do_addcard(card)
			return nil
		}
	}
	return errors.New("card not found")
}

//添加卡牌到手牌
func (this *Battler) _do_addcard(card *BattleCard) {
	//TODO 判断卡牌组合
	this.BattleCard = append(this.BattleCard, card)
	log.Println(fmt.Sprintf("battler[%d]有手牌[%d]张,新增[%s]", this.Id, len(this.BattleCard), card.Name))
	this._do_card_skill(nil, CardSkillStage_Buy, card)
}

//添加卡牌
func (this *Battler) _do_card_goout(card *BattleCard, idx int, target ...*BattleCard) {
	//TODO 判断卡牌组合
	this.AideCard = append(this.AideCard, nil)
	copy(this.AideCard[idx+1:], this.AideCard[idx:])
	this.AideCard[idx] = card
	//TODO 战斗中召唤的卡牌 触发 战吼？
	this._do_card_skill(nil, CardSkillStage_Use, card, target...)
	this._do_hero_skill(&Event{
		Stage:    HeroSkillStage_CallCard,
		Trigger:  card,
		Attacker: this,
		Idx:      idx,
		Target:   target,
	})
	log.Println(fmt.Sprintf("battler[%d]出战手牌[%d]张,新增[%s]", this.Id, len(this.AideCard), card.Name))
}

func (this *Battler) _docoin(coin int32) error {
	if this.Coin >= coin {
		this.Coin = this.Coin - coin
		return nil
	}
	//tlog.Error("扣除金币余额不足 ", this.PubRoom.Round, this.Player.Base.Id, this.Coin, coin)
	return errors.New(  "not enough gold coins")
}

func (this *Battler) _do_hero_skill(event *Event) {
	//steps := _do_hero_skill_handle(this.Hero.Skill, stage, trigger, this, nil)
	steps := EventSource(event)
	if steps != nil && len(steps) > 0 {
		for _, step := range steps {
			if step.IsSpecial {
				this.RoundChan <- &RoundStep{
					Type:     step.Type,
					SkillId:  step.SkillId,
					Attacker: step.Attacker,
					Harm:     step.Harm,
					Health:   step.Health,
					Atk:      step.Atk,
					Skill:    step.Skill,
					IsSneer:  step.IsSneer,
					IsSacred: step.IsSacred,
					Receiver: step.Receiver,
				}
			}
			//TODO 新召唤的卡牌
			if step.CardCall != nil && len(step.CardCall) > 0 {
				for _, card := range step.CardCall {
					this._do_card_goout(card, step.CardCallIdx)
				}
			}
			//受影响的已经存在的卡牌
			if step.Receiver != nil && len(step.Receiver) > 0 {
				for _, card := range step.Receiver {
					card._dosteps(step)
					//if step.Atk > 0 {
					//	this._do_card_skill(event, CardSkillStage_AddnAtk, card)
					//}
					//if step.Health > 0 {
					//	this._do_card_skill(event, CardSkillStage_AddnHealth, card)
					//}
					//if step.Atk > 0 || step.Health > 0 {
					//	this._do_card_skill(event, CardSkillStage_Addn, card)
					//}
				}
			}
			//for _, card := range step.Receiver {
			//	if card.BattleAddnToSteady || step.Forever {
			//		card.Steady.Skills[step.Skill] = step.SkillNum
			//		if step.Atk > 0 || step.Health > 0 {
			//			//TODO 属性加成 触发
			//			card.Steady.Atk += step.Atk
			//			card.Steady.Health += step.Health
			//			if addn := card.Steady.Addn[step.SkillId]; addn == nil {
			//				addn = &AddnDetail{
			//					SkillId: step.SkillId,
			//					Num:     1,
			//					Atk:     step.Atk,
			//					Health:  step.Health,
			//					Remark:  "", //TODO 加成描述
			//				}
			//				card.Steady.Addn[step.SkillId] = addn
			//			} else {
			//				card.Steady.Addn[step.SkillId].Atk += step.Atk
			//				card.Steady.Addn[step.SkillId].Health += step.Health
			//				card.Steady.Addn[step.SkillId].Num += 1
			//			}
			//		}
			//		if step.Skill > 0 {
			//			if val := card.Steady.Skills[step.Skill]; val == 0 {
			//				card.Steady.Skills[step.Skill] = 1 + step.SkillNum
			//			} else {
			//				if step.SkillNum > 0 {
			//					card.Steady.Skills[step.Skill] += step.SkillNum
			//				}
			//			}
			//		}
			//	}
			//	if !card.BattleAddnToSteady {
			//		card.BattleAddn.Skills[step.Skill] = step.SkillNum
			//		if step.Atk > 0 || step.Health > 0 {
			//			card.BattleAddn.Atk += step.Atk
			//			card.BattleAddn.Health += step.Health
			//			if addn := card.BattleAddn.Addn[step.SkillId]; addn == nil {
			//				addn = &AddnDetail{
			//					SkillId: step.SkillId,
			//					Num:     1,
			//					Atk:     step.Atk,
			//					Health:  step.Health,
			//					Remark:  "", //TODO 加成描述
			//				}
			//				card.BattleAddn.Addn[step.SkillId] = addn
			//			} else {
			//				card.BattleAddn.Addn[step.SkillId].Atk += step.Atk
			//				card.BattleAddn.Addn[step.SkillId].Health += step.Health
			//				card.BattleAddn.Addn[step.SkillId].Num += 1
			//			}
			//		}
			//		if step.Skill > 0 {
			//			if val := card.BattleAddn.Skills[step.Skill]; val == 0 {
			//				card.BattleAddn.Skills[step.Skill] = 1 + step.SkillNum
			//			} else {
			//				if step.SkillNum > 0 {
			//					card.BattleAddn.Skills[step.Skill] += step.SkillNum
			//				}
			//			}
			//		}
			//		if step.Forever {
			//			card.Steady.Skills[step.Skill] = step.SkillNum
			//			if step.Atk > 0 || step.Health > 0 {
			//				//TODO 属性加成 触发
			//				card.Steady.Atk += step.Atk
			//				card.Steady.Health += step.Health
			//				if addn := card.Steady.Addn[step.SkillId]; addn == nil {
			//					addn = &AddnDetail{
			//						SkillId: step.SkillId,
			//						Num:     1,
			//						Atk:     step.Atk,
			//						Health:  step.Health,
			//						Remark:  "", //TODO 加成描述
			//					}
			//					card.Steady.Addn[step.SkillId] = addn
			//				} else {
			//					card.Steady.Addn[step.SkillId].Atk += step.Atk
			//					card.Steady.Addn[step.SkillId].Health += step.Health
			//					card.Steady.Addn[step.SkillId].Num += 1
			//				}
			//			}
			//			if step.Skill > 0 {
			//				if val := card.Steady.Skills[step.Skill]; val == 0 {
			//					card.Steady.Skills[step.Skill] = 1 + step.SkillNum
			//				} else {
			//					if step.SkillNum > 0 {
			//						card.Steady.Skills[step.Skill] += step.SkillNum
			//					}
			//				}
			//			}
			//		}
			//	}
			//	//属性加载完成后触发一系列的
			//}
		}
	}
}
