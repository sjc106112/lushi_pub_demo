package main

import (
	"log"
	"math/rand"
)

type BattleCard struct {
	BattleAddnToSteady bool                // 是否战斗中的属性加成永久保留
	Id                 int32               //随机自增的的id
	Name string
	StarLevel          int32               //级别
	Owner              *Battler            //所属的主人
	BattleAddn         *BattleCardProperty //正常的属性
	BackUp             *BattleCardProperty //备份的属性
	IsFreeze           bool                //是否冻结的卡牌
	IsHidden           bool                //是否隐藏的卡牌，是单独显示的卡牌(PubRoom)
}

type AddnDetail struct {
	Num          int8   //触发次数
	BattleCardId int32  //发动技能的卡牌
	SkillId      int32  //产生该影响的技能id
	Atk          int32  //加成的攻击(有正负)
	Health       int32  //加成的血量(有正负)
	Remark       string //说明
}

type BattleCardProperty struct {
	IsDie     bool  //是否死亡
	Health    int32 //开始战斗的血量
	Atk       int32 //开始战斗的攻击
	Harm      int32 //受到的伤害
	IsSneer   bool  //是否嘲讽
	IsSacred  bool  //是否圣盾
	IsRoar    bool  //是否风吼
	IsTemp    bool  //是否战斗中临时召唤的卡牌
	Addn      map[int32]*AddnDetail
	Skills    map[int32]int8 //拥有的技能
	Round     int32          //攻击轮次
	AttackNum int32          //每次攻击几次
}

func (this *BattleCard) Attack(defender *Battler) {
	if defender.AideCard == nil || len(defender.AideCard) == 0 {
		return
	}
	var target = make([]*BattleCard, 0, len(defender.AideCard))
	for _, c := range defender.AideCard {
		if c.BattleAddn.IsSneer {
			target = append(target, c)
		}
	}
	if len(target) == 0 {
		target = defender.AideCard
	}
	var t = target[rand.Intn(len(target))]
	//开始攻击
	this.Owner._do_card_skill(nil, CardSkillStage_AttackBefor, this, t)
	t.Owner._do_card_skill(nil, CardSkillStage_DefendBefor, t, this)
	t.BattleAddn.Harm += this.BattleAddn.Atk
	if t.BattleAddn.Harm > t.BattleAddn.Health {
		t.BattleAddn.IsDie = true
	}
	this.BattleAddn.Harm += t.BattleAddn.Atk
	if this.BattleAddn.Harm > this.BattleAddn.Health {
		this.BattleAddn.IsDie = true
	}
	log.Println("card attack [%d,%d],defender[%d,%d]", this.Id, this.BattleAddn.Harm, t.Id, t.BattleAddn.Harm)
	//TODO Type类型未定义？
	round_step := &RoundStep{
		Type:         RoundStepType_GeneralAttack,
		Attacker:     this.Id,
		Defender:     t.Id,
		Harm:         t.BattleAddn.Atk,
		AttackerHarm: this.BattleAddn.Atk,
	}
	if this.Owner.State.Load() != BattlerState_Over {
		this.Owner.RoundChan <- round_step
	}

	if t.Owner.State.Load() != BattlerState_Over {
		t.Owner.RoundChan <- round_step
	}
	//攻击完成
	t.Owner._do_card_skill(nil, CardSkillStage_DefendAfter, t, this)
	this.Owner._do_card_skill(nil, CardSkillStage_AttackAfter, this, t)

	if t.BattleAddn.IsDie {
		t.Die(this.Owner)
	}
	if this.BattleAddn.IsDie {
		this.Die(defender)
	} else if this.BattleAddn.IsRoar {
		this.BattleAddn.AttackNum--
		if this.BattleAddn.AttackNum > 0 {
			this.Attack(defender)
			return
		}
	}
}

func (this *BattleCard) Die(defender *Battler) {
	//TODO 移除队列
	for i, c := range this.Owner.AideCard {
		if c.Id == this.Id {
			this.Owner.AideCard = append(this.Owner.AideCard[:i], this.Owner.AideCard[i+1:]...)
			//TODO 触发亡语
			//this.Skill(i, CardSkillStage_Die, this)
			this.Skill(&Event{
				Type:     SkillType_Card,
				Card:     this,
				Attacker: this.Owner,
				Defender: defender,
				Stage:    CardSkillStage_Die,
				Trigger:  c,
				Idx:      i,
			})
			return
		}
	}
}

/**
 * idx 当前卡牌所在的位置下标
 * stage 对应trigger的状态
 * target 主动指定的目标
 */
func (this *BattleCard) Skill(event *Event) {
	//先触发当前牌的技能 优先战斗阶段,战斗为nil时触发准备阶段
	if this.BattleAddn != nil && this.BattleAddn.Skills != nil {
		if len(this.BattleAddn.Skills) > 0 {
			for skill_id, num := range this.BattleAddn.Skills {
				//战斗阶段
				for i := 0; i < int(num); i++ {
					event.SkillId = skill_id
					//idx, skill_id, trigger, stage, target...
					this._do_skill(event)
				}
			}
		}
	}
	//else if this.Steady.Skills != nil && len(this.Steady.Skills) > 0 {
	//	for skill_id, num := range this.Steady.Skills {
	//		//准备阶段
	//		for i := 0; i < int(num); i++ {
	//			event.SkillId = skill_id
	//			this._do_skill(event)
	//		}
	//	}
	//}

	////TODO 循环整个牌,触发技能
	//for _, card := range this.Owner.AideCard {
	//	if card.Steady.Skills != nil && len(card.Steady.Skills) > 0 {
	//		for skill_id, num := range card.Steady.Skills {
	//			//准备阶段
	//			for i := 0; i < int(num); i++ {
	//				card._do_skill(skill_id, this, stage)
	//			}
	//		}
	//	}
	//	if card.BattleAddn != nil && card.BattleAddn.Skills != nil && len(card.BattleAddn.Skills) > 0 {
	//		for skill_id, num := range card.BattleAddn.Skills {
	//			//准备阶段
	//			for i := 0; i < int(num); i++ {
	//				card._do_skill(skill_id, this, stage)
	//			}
	//		}
	//	}
	//}
}

//触发技能
func (this *BattleCard) _do_skill(event *Event) {
	steps := EventSource(event)
	if steps != nil && len(steps) > 0 {
		for _, step := range steps {
			//TODO 更新卡牌属性和AddnDetail 根据是否永久属性，更新不同的AddnDetail
			if step.IsSpecial {
				this.Owner.RoundChan <- &RoundStep{
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
					this.Owner._do_card_goout(card, step.CardCallIdx)
				}
			}
			//受影响的已经存在的卡牌
			if step.Receiver != nil && len(step.Receiver) > 0 {
				for _, card := range step.Receiver {
					card._dosteps(step)
					if step.Atk > 0 {
						this.Owner._do_card_skill(event, CardSkillStage_AddnAtk, card)
					}
					if step.Health > 0 {
						this.Owner._do_card_skill(event, CardSkillStage_AddnHealth, card)
					}
					if step.Atk > 0 || step.Health > 0 {
						log.Println("start addn 循环")
						this.Owner._do_card_skill(event, CardSkillStage_Addn, card)
					}
				}
			}
		}
	}
}

//属性临时强化
func (this *BattleCard) _dosteps(step *SkillStep) {
	this.BattleAddn._addproperty(step)
	if step.Forever && !this.BattleAddnToSteady && this.BackUp!=nil{
		this.BackUp._addproperty(step)
	}
}

func (this *BattleCardProperty) _addproperty(step *SkillStep) {
	if step.Atk > 0 || step.Health > 0 {
		this.Atk += step.Atk
		this.Health += step.Health
		if addn := this.Addn[step.SkillId]; addn == nil {
			addn = &AddnDetail{
				SkillId: step.SkillId,
				Num:     1,
				Atk:     step.Atk,
				Health:  step.Health,
				Remark:  "", //TODO 加成描述
			}
			this.Addn[step.SkillId] = addn
		} else {
			//只叠加次数
			this.Addn[step.SkillId].Num += 1
		}
	}
	if step.Skill > 0 {
		if val := this.Skills[step.Skill]; val == 0 {
			this.Skills[step.Skill] = 1 + step.SkillNum
		} else {
			if step.SkillNum > 0 {
				this.Skills[step.Skill] += step.SkillNum
			}
		}
	}
	if step.IsSneer {
		this.IsSneer = true
	}
	if step.IsSacred {
		this.IsSacred = true
	}
	if step.IsRoar {
		this.IsSacred = true
	}
}
