# lushi_pub_demo
![image](https://github.com/sjc106112/lushi_pub_demo/blob/main/161200.png)
工作四天敲出的代码，无奈无人欣赏，上传github，自娱自乐。
整个代码模仿炉石传说酒馆对战体系,整个代码除了战斗流程的实现细节外，技能相关的东西都不涉及。当然，一些剧毒、圣盾等等忽略。
代码要表达的重点是:事件机制循环调用的解决方案，本方案将问题留在了事件层处理，不侵入到具体的战斗和技能中。
技能调用主要通过事件机制，通过一种特殊的方式解决了事件触发死循环的问题，这个问题在网上没有找到好的解决办法，抛砖引玉，希望大家指正。
技能实现参照 battle_skill.go,代码中实现了战吼加属性和属性变更再次加成属性的技能。
go run main.go 就可以

# 后续优化方案
 ## 这个版本可以支持属性加成类、嘲讽、风怒、召唤类技能
 ## 第二个版本：元数据版
         抽象出仇恨值来代替嘲讽概念、免伤来代替圣盾概念、攻击次数来代替风怒概念，这些属性和攻击、血量、盾、伤害、种族......, 统称为系统元数据；
         编辑器添加自定义元数据功能来支撑英雄技能（自定义元数据和系统元数据统一称为元数据）；
         自定义元数据可以在编辑器上根据事件节点配置 值加成、值初始化、值复原操作；
         元数据可以成为后续技能触发条件、目标筛选条件、目标属性......组成部分，编辑器提供表达式编辑功能；
 ## 第三个版本：组件版
        将房间、玩家、卡牌 作为聚合根，
        将金币、卡牌池、战斗、技能加成等等抽象成组件的概念，
        两者结合支撑多种战斗模式
 ## 第四个版本：深度定制化
        更加抽象的实现整个体系，将系统元数据去掉，统称为用户自定义元数据；
        将攻击、技能特效等等抽象为动作概念；
        
        
        
