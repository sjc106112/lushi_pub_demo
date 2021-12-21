package main

import (
	"encoding/json"
	"fmt"
	"log"
)

type SkillStep struct {
	IsSneer         bool          //是否增加嘲讽
	IsSacred        bool          //是否增加圣盾
	IsRoar          bool          //是否风吼
	IsSpecial       bool          //是否特效展示，如果是，将推到前台
	Forever         bool          //是否永久的
	Type            int8          //1 攻方卡牌攻击 2 守方卡牌攻击 3 攻防英雄技能 4 守方英雄技能
	SkillId         int32         //发起的技能
	Attacker        int32         //发起方
	Receiver        []*BattleCard //受影响的卡牌
	HeroHarm        int32         //攻击伤害 英雄(己方)
	Harm            int32         //攻击伤害 卡牌
	Health          int32         //加成血量
	Atk             int32         //加成的攻击
	Skill           int32         //加成的技能
	SkillNum        int8          //加成的技能攻击次数,0表示不叠加
	CostLevel       int32         //影响升级酒馆需要的金币
	Coin            int32         //影响剩余的金币
	CardRefreshCoin int32         //影响刷新卡牌需要的金币
	CardCoin        int32         //影响购买卡牌需要的金币
	CardCall        []*BattleCard //召唤的卡牌
	CardCallIdx     int           //召唤的卡牌插入的位置,如果为-1 则代表着亡语牌的位置插入
}

type Event struct {
	SkillId  int32         //技能id
	Type     int8          // 1 hero 2 card
	Card     *BattleCard   //技能的拥有者,事件关心者
	Attacker *Battler      // 攻击者,如果Type为1则代表着攻击者是技能的拥有者
	Defender *Battler      //防守者
	Stage    int           //事件类型
	Trigger  *BattleCard   //事件触发者,如果不为null stage 对应trigger 的状态
	Target   []*BattleCard //主动指定的目标
	Parent   *Event        //父事件
	Idx      int           // card存在的位置下标
	//Result   []*SkillStep  // 当前时间执行的合集
}

func (this *Event) equals(event *Event) bool {
	if event == nil {
		return false
	}
	if this == event {
		return true
	}
	if this.SkillId != event.SkillId || this.Stage != event.Stage || this.Type != event.Type ||
		this.Card != event.Card || this.Attacker != event.Attacker ||
		this.Defender != event.Defender || this.Trigger != event.Trigger {
		return false
	}
	return true
}

func (this *Event) String() string {
	data, _ := json.Marshal(this)
	return string(data)
}


func EventSource(event *Event) []*SkillStep {
	//循环问题
	var event_temp = event.Parent
	for event_temp != nil {
		if event.equals(event_temp) {
			log.Println(fmt.Sprintf("技能触发死循环 %d %d",event.SkillId,event.Stage))
			return nil
		}
		event_temp = event_temp.Parent
	}
	//TODO 监听 ？？
	switch event.Type {
	case SkillType_Hero:
		return _do_hero_skill_handle(event.SkillId, event.Stage, event.Trigger, event.Attacker, event.Defender, event.Target...)
	case SkillType_Card:
		return _do_card_skill_handle(event.SkillId, event.Card, event.Trigger, event.Stage, event.Attacker, event.Defender, event.Target...)
	}
	return nil
}

type skill_handle func(skill_id int32, stage int, attacker *Battler, defender *Battler) []*SkillStep

/**
 * trigger stage 对应trigger的状态
 * target 主动指定的目标
 * attacker 攻击的玩家
 * defender 防守的玩家
 */
func _do_hero_skill_handle(skill_id int32, stage int, trigger *BattleCard, attacker *Battler, defender *Battler, target ...*BattleCard) []*SkillStep {
	return nil
}

/**
 * card 技能skill_id拥有的卡牌
 * trigger stage 对应trigger的状态
 * target 主动指定的卡牌目标
 * attacker 攻击的玩家
 * defender 防守的玩家
*/
func _do_card_skill_handle(skill_id int32, card *BattleCard, trigger *BattleCard, stage int, attacker *Battler, defender *Battler, target ...*BattleCard) []*SkillStep {
	//log.Println("卡牌技能 :",skill_id,card.Name,stage,card.Id)
	return skill_property(skill_id,card,trigger,stage,attacker,defender)
}

func skill_property(skill_id int32,card *BattleCard, trigger *BattleCard,stage int, attacker *Battler, defender *Battler, target ...*BattleCard)[]*SkillStep{
	steps := make([]*SkillStep,0,10)
	switch skill_id {
	case 1000010://本身加属性
		if stage == CardSkillStage_Use{
			return append(steps,&SkillStep{
				Forever:         true,
				SkillId:         1000010,
				Receiver:        []*BattleCard{card},
				Health:          1,
				Atk:             1,
			})
		}
		return nil
	case 1000020://属性变更给对方加属性
		if stage == CardSkillStage_Addn&& card == trigger{
			return append(steps,&SkillStep{
				Forever:         true,
				SkillId:         1000020,
				Receiver:        []*BattleCard{card},
				Health:          2,
				Atk:             2,
			})
		}
	}
	return nil
}
