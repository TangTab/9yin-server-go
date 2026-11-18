--开始使用某项技能，命令格式：unsigned cmdid, object skill
COMMAND_SKILL_BEGIN = 600

--使用技能尝试攻击：unsigned cmdid, object target, object skill
COMMAND_SKILL_TRY_ATTACK = 605

--对目标使用增益技能，命令格式：unsigned cmdid, object target
COMMAND_ENHANCETARGET = 609
--攻击目标，命令格式：unsigned cmdid, object target
COMMAND_ATTACKTARGET = 610
--伤害目标之前，命令格式：unsigned cmdid, object target
COMMAND_BEFORE_DAMAGETARGET = 611
--被伤害之前，命令格式：unsigned cmdid(sender:src)
COMMAND_BEFORE_BEDAMAGE = 612
--伤害目标，命令格式：unsigned cmdid, object target, int damage, int damagetype
COMMAND_DAMAGETARGET = 613
--被攻击，命令格式：unsigned cmdid, int damage, int damagetype(sender:src)
COMMAND_BEDAMAGE = 614
--击杀目标，命令格式：unsigned cmdid, object target
COMMAND_KILLTARGET = 615
--被击杀，命令格式：unsigned cmdid(sender:src)
COMMAND_BEKILL = 616