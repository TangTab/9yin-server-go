-- 本文件由 tools/skill-data/gen_desc_effects.py 生成（幂等可重跑，勿手改）。
-- 数据源 string_skill_desc.idres（desc_<技能ID>_<等级>）；IR 权威定义
-- docs/skill-desc-ir.md（动词三分类：控制状态/位移/资源）。消费方：
-- internal/skillscript DescEffects（旁路数据层，优先级链不变：手写 >
-- effects_data > INI；本表仅供查询与后续引擎接线）。
local E = {}

E["CS_3times"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "stun"
        }
    }
}

E["CS_base_pgd02"] = {

}

E["CS_base_pgd03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_base_zfj02"] = {

}

E["CS_base_zfj03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_bg_wch01"] = {
    unparsed = { "格挡成功时：若自身当前气血百分比低于攻击目标，则获得一层“权”：每层使自身外功命中率提高3%，最" }
}

E["CS_bg_wch02"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "root"
        },
        {
            control_kind = "silence"
        }
    }
}

E["CS_bg_wch03"] = {
    on_parry_break = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_bg_wch05"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "silence"
        },
        {
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_bg_wch06"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_bg_wch07"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    }
}

E["CS_bg_wch08"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_bg_zlz02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_bg_zlz03"] = {
    on_hit = {
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_bg_zlz04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "root"
        }
    },
    unparsed = { "若未招架的敌人存在“引线”状态，则倒地3秒" }
}

E["CS_bg_zlz07"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_bg_zlz08_2",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 2000,
            control_kind = "stiffen"
        }
    }
}

E["CS_bg_zlz08"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_bg_zlz08_1",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_by_gxcj02"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "coma"
        }
    }
}

E["CS_by_gxcj03"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_by_gxcj03",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_by_gxcj04"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，对霸体无效，可被虚招克制）" }
}

E["CS_by_gxcj04_hide"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，对霸体无效，可被虚招克制）" }
}

E["CS_by_gxcj05"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_by_gxcj06"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_by_gxcj07"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_by_wgl01"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_by_wgl02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_by_wgl03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_by_wgl04"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_by_wgl04",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 5000,
            control_kind = "launch",
            displacement_distance_m = 3,
            displacement_kind = "push"
        }
    }
}

E["CS_by_wgl05"] = {
    unparsed = { "绕树三匝，何枝可依，招架成功时，降低攻击者1%气血上限（仅对玩家有效），提升自身本套路2%招式伤", "（提升招式功力，可增加状态叠加层数上限）" }
}

E["CS_by_wgl06"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "root"
        }
    }
}

E["CS_by_wgl07"] = {
    compile = {
        can_move = false,
        cast_armor = "yellow"
    }
}

E["CS_by_xhmdl01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_by_xhmdl01",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1,
            burst_at = 3,
            burst_dur_ms = 2000,
            burst_kind = "coma",
            condition = { unparried = true }
        }
    }
}

E["CS_by_xhmdl02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_by_xhmdl03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_by_xhmdl04"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        can_move = false,
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_by_xhmdl01",
            apply_resolved = true,
            apply_stacks = 1,
            displacement_distance_m = 0,
            displacement_kind = "pull",
            per_hit = true
        }
    }
}

E["CS_by_xhmdl05"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_by_xhmdl01",
            apply_resolved = true,
            apply_stacks = 2,
            control_dur_ms = 3000,
            control_kind = "silence",
            control_within_ms = 6000
        }
    }
}

E["CS_by_xhmdl06"] = {
    on_parry = {
        {
            apply_buff = "buf_CS_by_xhmdl06_01",
            apply_life_ms = 8000,
            apply_resolved = true,
            apply_stacks = 1,
            burst_at = 10,
            burst_dur_ms = 2000,
            burst_kind = "root"
        }
    },
    on_victim = {
        {
            stack_buff = "buf_CS_by_xhmdl06_01",
            stack_per_hit = 1,
            stack_resolved = true
        }
    },
    by_level = {
        [2] = { ["on_parry.1.apply_life_ms"] = 10000 },
        [3] = { ["on_parry.1.apply_life_ms"] = 12000 },
        [4] = { ["on_parry.1.apply_life_ms"] = 14000 },
        [5] = { ["on_parry.1.apply_life_ms"] = 16000 },
        [6] = { ["on_parry.1.apply_life_ms"] = 18000 }
    }
}

E["CS_by_xhmdl07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_periodic = {
        {
            apply_buff = "buf_CS_by_xhmdl01",
            apply_resolved = true,
            periodic_aoe = true,
            periodic_every_ms = 2000
        }
    },
    self_state = {
        {
            apply_buff = "buf_CS_by_xhmdl07",
            apply_life_ms = 6000,
            apply_resolved = true
        }
    }
}

E["CS_by_xhmdl08"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "red",
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "孽（无视招架）",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparryable = true },
            per_hit = true
        }
    }
}

E["CS_by_xlxf01"] = {
    unparsed = { "自身存在“苍穹灭”状态时：招式射程增加1倍，且有一定几率击晕目标" }
}

E["CS_by_xlxf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "自身存在“苍穹灭”状态时：招式射程增加1倍，且有一定几率击晕目标", "招式命中未招架目标时，减少自身所有招式10%回气时间" }
}

E["CS_by_xlxf03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "自身存在“苍穹灭”状态时：招式射程增加1倍，且有一定几率击晕目标", "招式命中未招架目标时，减少自身所有招式10%回气时间" }
}

E["CS_by_xlxf04"] = {
    on_parry_break = {
        {
            control_kind = "knockdown"
        }
    },
    unparsed = { "自身存在“苍穹灭”状态时：招式射程增加1倍，且有一定几率击晕目标" }
}

E["CS_by_xlxf05"] = {
    on_parry = {
        {
            control_dur_ms = 10000,
            control_kind = "coma"
        }
    },
    by_level = {
        [2] = { ["on_parry.1.control_dur_ms"] = 12000 },
        [3] = { ["on_parry.1.control_dur_ms"] = 14000 },
        [4] = { ["on_parry.1.control_dur_ms"] = 16000 },
        [5] = { ["on_parry.1.control_dur_ms"] = 18000 },
        [6] = { ["on_parry.1.control_dur_ms"] = 20000 }
    }
}

E["CS_by_xlxf06"] = {
    self_state = {
        {
            apply_buff = "苍穹灭",
            apply_resolved = false
        }
    },
    unparsed = { "8秒内，本套路冲锋招式命中未招架目标时，减少自身所有招式10%回气时间" }
}

E["CS_by_xlxf07"] = {
    unparsed = { "自身存在“苍穹灭”状态时：招式射程增加1倍，且有一定几率击晕目标", "招式命中未招架目标时，减少自身所有招式10%回气时间" }
}

E["CS_by_xyjj01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    }
}

E["CS_by_xyjj02"] = {
    on_parry_break = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_by_xyjj03"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_by_xyjj03_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_by_xyjj04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_by_xyjj05"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_by_xyjj06"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_by_xyjj07"] = {
    compile = {
        cast_armor = "yellow"
    },
    unparsed = { "【绞】或【压】之后快速连出本招，出招提升30%暴击几率" }
}

E["CS_cf_bwqf01"] = {
    unparsed = { "招式命中未招架敌人时，可窃取其4%外功命中率化为己用，该效果持续4秒，不可叠加" }
}

E["CS_cf_bwqf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_cf_bwqf03"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "stun"
        }
    }
}

E["CS_cf_bwqf04"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_cf_bwqf05"] = {
    unparsed = { "挑飞未及招架的目标" }
}

E["CS_cf_bwqf06"] = {
    on_parry_break = {
        {
            control_dur_ms = 2000,
            control_kind = "stun"
        }
    }
}

E["CS_cf_bwqf07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "时不利兮状态",
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 5000,
            control_kind = "root"
        },
        {
            control_kind = "root"
        }
    },
    by_level = {
        [6] = { ["on_hit.1.control_dur_ms"] = 6000 },
        [7] = { ["on_hit.1.control_dur_ms"] = 7000 },
        [8] = { ["on_hit.1.control_dur_ms"] = 8000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 9000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 10000 }
    }
}

E["CS_cf_yjqf01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_cf_yjqf01",
            apply_resolved = true,
            apply_stacks = 1,
            control_kind = "coma"
        },
        {
            apply_buff = "一层“雨柔",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 10000,
            control_kind = "knockdown"
        }
    }
}

E["CS_cf_yjqf01_hide"] = {
    on_hit = {
        {
            apply_buff = "一层“雨柔",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 10000,
            control_kind = "knockdown"
        }
    }
}

E["CS_cf_yjqf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_cf_yjqf03"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_dur_ms = 5000,
            control_kind = "launch"
        }
    }
}

E["CS_cf_yjqf04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_cf_yjqf05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_cf_yjqf05",
            apply_life_ms = 3000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "knockdown"
        }
    }
}

E["CS_cf_yjqf06"] = {
    self_state = {
        {
            apply_buff = "buf_CS_cf_yjqf06",
            apply_life_ms = 10000,
            apply_resolved = true
        }
    }
}

E["CS_cf_yjqf07"] = {
    unparsed = { "提枪邀敌，豪气冲天，成功招架敌人暴击，自身获得“壮怀激烈”状态：受到伤害降低20%，持续3秒", "成功招架时，额外获得一层“精忠”效果：【寒金鸣夜】每击可消耗一层获得短暂黄霸体效果（激活此效果需" }
}

E["CS_df_dfwfj01"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_df_dfwfj02"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 5000,
            control_kind = "slow"
        }
    }
}

E["CS_df_dfwfj03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "自身存在“五行”状态，则每击可打散未招架目标1点怒气" }
}

E["CS_df_dfwfj04"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_df_dfwfj04_npc"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_df_dfwfj05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            apply_buff = "五行状态",
            apply_life_ms = 7000,
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_df_dfwfj06"] = {
    unparsed = { "六道轮回，随剑御势，招架8次自身获得六道状态，持续15秒", "使用五行剑阵时消耗一层此状态对敌人点穴2秒", "使用七星剑阵后消耗一层此状态使5秒内使意图攻击自己的敌人点穴2秒", "自身存在六道状态时，招架不会触发计次状态" }
}

E["CS_df_dfwfj07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            apply_buff = "七星状态",
            apply_life_ms = 7000,
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_dh_trlx01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    }
}

E["CS_dh_trlx02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_dh_trlx03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_dh_trlx04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "青魇状态",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_dh_trlx05"] = {
    self_state = {
        {
            apply_buff = "buf_CS_dh_trlx05",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_dh_trlx06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_dh_trlx06",
            apply_life_ms = 8000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 8000,
            control_kind = "knockdown"
        }
    }
}

E["CS_dh_trlx07"] = {
    unparsed = { "（只能在自身闪避后5秒内使用，此招式可利用轻功快速接近对手" }
}

E["CS_dh_trlx07_hide"] = {
    unparsed = { "（只能在自身闪避后5秒内使用，此招式可利用轻功快速接近对手" }
}

E["CS_dh_txxhjp01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_dh_txxhjp01_01",
            apply_life_ms = 30000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        },
        {
            apply_buff = "buf_CS_dh_txxhjp01_01",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_dh_txxhjp02"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        },
        {
            apply_buff = "buf_CS_dh_txxhjp01_01",
            apply_resolved = true,
            apply_stacks = 1,
            control_kind = "launch"
        }
    }
}

E["CS_dh_txxhjp02_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        },
        {
            apply_buff = "buf_CS_dh_txxhjp01_01",
            apply_resolved = true,
            apply_stacks = 1,
            control_kind = "launch"
        }
    }
}

E["CS_dh_txxhjp03"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "stun",
            displacement_distance_m = 0,
            displacement_kind = "pull"
        },
        {
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_dh_txxhjp04"] = {
    on_parry_break = {
        {
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    }
}

E["CS_dh_txxhjp05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_dh_txxhjp01_01",
            apply_life_ms = 30000,
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.2
        }
    },
    by_level = {
        [2] = { ["on_hit.1.chance"] = 0.25 },
        [3] = { ["on_hit.1.chance"] = 0.3 },
        [4] = { ["on_hit.1.chance"] = 0.35 },
        [5] = { ["on_hit.1.chance"] = 0.4 },
        [6] = { ["on_hit.1.chance"] = 0.5 }
    }
}

E["CS_dh_txxhjp06"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_dh_txxhjp08"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_dh_txxhjp01_01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_dh_txxhjp08_hide"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_dh_txxhjp01_01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_dm_fmgf01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "盘缠翻摔式，对选中目标造成{@0:伤害包}，自身在空中时可施展该招式击落空中目标" }
}

E["CS_dm_fmgf01_sky"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "盘缠翻摔式，对选中目标造成{@0:伤害包}，自身在空中时可施展该招式击落空中目标" }
}

E["CS_dm_fmgf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_dm_fmgf03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    self_state = {
        {
            apply_buff = "buf_CS_dm_fmgf_03_01",
            apply_life_ms = 20000,
            apply_resolved = true
        }
    }
}

E["CS_dm_fmgf03_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    unparsed = { "该招式不可在空中施展" }
}

E["CS_dm_fmgf04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong",
            per_hit = true
        }
    }
}

E["CS_dm_fmgf05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        },
        {
            control_kind = "stiffen",
            per_hit = true
        }
    }
}

E["CS_dm_fmgf06"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 6000,
            control_kind = "stiffen",
            per_hit = true
        }
    }
}

E["CS_dm_fmgf07"] = {
    compile = {
        air = "never"
    },
    self_state = {
        {
            apply_buff = "buf_CS_dm_fmgf07_01",
            apply_resolved = true
        }
    }
}

E["CS_dm_fmgf08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "龙吞状态",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "coma",
            displacement_distance_m = 0,
            displacement_kind = "push",
            per_hit = true
        },
        {
            condition = { unparried = true },
            control_kind = "launch",
            per_hit = true
        }
    }
}

E["CS_dm_qys01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_dm_qys02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_dm_qys03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    unparsed = { "自身存在“佛影”时减少怒气效果翻倍" }
}

E["CS_dm_qys04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown",
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_dm_qys05"] = {
    unparsed = { "若见一切法，心不染着，是为无念，激发内力形成“佛影”附在自身" }
}

E["CS_dm_qys06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_dm_qys06",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_dm_qys07"] = {
    unparsed = { "自身存在“佛影”时，最后一击打散未招架目标一半当前怒气值" }
}

E["CS_dm_qys08"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_dy_sdyjl01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    }
}

E["CS_dy_sdyjl02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_dy_sdyjl03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_dy_sdyjl04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "silence"
        }
    }
}

E["CS_dy_sdyjl04_hide"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    }
}

E["CS_dy_sdyjl05"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_dy_sdyjl05_hide"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    },
    unparsed = { "使用完毕清除自身一层鸣刀状态" }
}

E["CS_dy_sdyjl06"] = {
    compile = {
        cast_armor = "yellow"
    }
}

E["CS_dy_sdyjl07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_dy_sdyjl08"] = {
    on_hit = {
        {
            apply_buff = "一层空灵状态",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 12000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 12000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 12000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 13000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 15000 }
    }
}

E["CS_dy_sdyjl09_hide"] = {
    unparsed = { "飞斩一击对目标造成{@0:伤害包}，使用完毕清除自身一层鸣刀状态" }
}

E["CS_em_jdmz01"] = {
    unparsed = { "隔空一击，对目标造成{@0:伤害包}，掌风可使对方重心不稳，减速20%（最多叠加3次）" }
}

E["CS_em_jdmz02"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_em_jdmz03"] = {

}

E["CS_em_jdmz04"] = {
    unparsed = { "防御架势，以巧劲卸力，招架时可让对方重心不稳，减速20%（最多叠加3次）" }
}

E["CS_em_jdmz05"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_em_jdmz07"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_em_lbc01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_em_lbc01",
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "dizzy",
            displacement_distance_m = 0,
            displacement_kind = "pull"
        },
        {
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_em_lbc01_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "dizzy",
            displacement_distance_m = 0,
            displacement_kind = "pull"
        },
        {
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_em_lbc02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "怒气需要的“离苦",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "同时可额外增加自身怒气，增加量随“离苦”状态叠加而增加", "锁定距离20米" }
}

E["CS_em_lbc02_hide"] = {
    unparsed = { "同时可额外增加自身怒气，增加量随“离多会少”状态叠加而增加" }
}

E["CS_em_lbc03"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "stun"
        },
        {
            condition = { unparried = true },
            control_kind = "stun"
        },
        {
            control_dur_ms = 1000,
            control_kind = "silence"
        }
    },
    unparsed = { "招式命中敌人时可获得一层“云愁”状态" }
}

E["CS_em_lbc03_hide"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "stun"
        },
        {
            condition = { unparried = true },
            control_kind = "stun"
        },
        {
            control_dur_ms = 1000,
            control_kind = "silence"
        }
    },
    unparsed = { "招式命中敌人时可获得一层“云愁”状态" }
}

E["CS_em_lbc03_sky"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "stun"
        }
    }
}

E["CS_em_lbc03_sky_hide"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "stun"
        },
        {
            control_dur_ms = 1000,
            control_kind = "silence"
        }
    }
}

E["CS_em_lbc04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 }
    },
    unparsed = { "破防成功时，打散敌人全部怒气，并根据打散的怒气量附加柔伤害" }
}

E["CS_em_lbc04_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    unparsed = { "破防成功时，打散敌人全部怒气，并根据打散的怒气量附加柔伤害" }
}

E["CS_em_lbc05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_em_lbc01",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "（只有学会招式【爱别离苦】后，招架才会增加“离苦”状态）同时激活全部【云烟】及【幻月】武技则不再", "“离多会少”状态：每层提高离别刺1%伤害及命中，可叠加20层" }
}

E["CS_em_lbc05_hide"] = {
    on_hit = {
        {
            apply_buff = "一层“离多会少",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "“离多会少”状态：每层提高离别刺1%伤害及命中，可叠加20层", "且额外增加自身1点怒气" }
}

E["CS_em_lbc06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "stun"
        },
        {
            apply_buff = "一层“飞燕",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_em_lbc06_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "一层“飞燕",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_em_lbc06_sky"] = {
    on_hit = {
        {
            control_kind = "stun"
        }
    }
}

E["CS_em_lbc07"] = {
    on_hit = {
        {
            apply_buff = "一层“绝俗",
            apply_resolved = false,
            apply_stacks = 1
        },
        {
            apply_buff = "一层“离愁",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    },
    unparsed = { "该状态叠加层数越高，伤害越高", "此状态下攻击敌人时，会根据自身“离苦”层数附加伤害" }
}

E["CS_em_lbc07_hide"] = {
    on_hit = {
        {
            apply_buff = "一层“离愁",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_em_lbc08"] = {
    on_hit = {
        {
            control_dur_ms = 1000,
            control_kind = "root"
        },
        {
            control_kind = "launch"
        }
    }
}

E["CS_em_pxcyz01"] = {
    on_hit = {
        {
            condition = { unparried = true },
            heal = true
        }
    }
}

E["CS_em_pxcyz02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_em_pxcyz03"] = {
    unparsed = { "冻碎对方招架时，可恢复自身气血，化解淤血" }
}

E["CS_em_pxcyz04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 2 }
    }
}

E["CS_em_pxcyz04_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 2 }
    }
}

E["CS_em_pxcyz05"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_em_pxcyz06"] = {
    on_hit = {
        {
            control_kind = "root"
        }
    }
}

E["CS_em_pxcyz07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "dizzy",
            per_hit = true
        }
    },
    unparsed = { "寒气入侵可使敌人移动迟缓，且无法施展轻功" }
}

E["CS_em_pxcyz07_a"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_em_pxcyz07_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "dizzy",
            per_hit = true
        }
    }
}

E["CS_em_pxcyz08"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "root"
        },
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_em_pxcyz08_hide"] = {
    on_hit = {
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_em_pxcyz08_hide_sky"] = {
    on_hit = {
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_em_pxcyz08_sky"] = {
    on_hit = {
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    },
    unparsed = { "造成{@0:伤害包}，跳起后在空中使用可从空中快速落下，使范围内受到飘雪攻击，减少身法" }
}

E["CS_em_qcc01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 10 }
    },
    unparsed = { "招式首次命中目标，则自身可获得1层“倾城”之意，最多叠加至4层，持续15秒" }
}

E["CS_em_qcc02"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_em_qcc02_01",
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.25,
            control_dur_ms = 2000,
            control_kind = "root"
        }
    }
}

E["CS_em_qcc03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_em_qcc04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    },
    unparsed = { "招式首次命中目标，则自身可获得1层“倾城”之意，最多叠加至4层，持续15秒" }
}

E["CS_em_qcc05"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "招式首次命中目标，则自身可获得1层“倾城”之意，最多叠加至4层，持续15秒" }
}

E["CS_em_qcc06"] = {
    compile = {
        cast_armor = "黄"
    },
    self_state = {
        {
            apply_buff = "buf_CS_em_qcc06",
            apply_resolved = true
        }
    }
}

E["CS_em_qcc07"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 },
        cast_armor = "黄"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "coma"
        }
    }
}

E["CS_em_ynjf01"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_em_ynjf02"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_em_ynjf02",
            apply_resolved = true,
            apply_stacks = 3,
            control_kind = "launch"
        }
    }
}

E["CS_em_ynjf03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_em_ynjf02",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_em_ynjf04"] = {
    compile = {
        air = "only"
    },
    on_hit = {
        {
            control_kind = "launch"
        },
        {
            apply_buff = "buf_CS_em_ynjf02",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_em_ynjf05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_em_ynjf02",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_em_ynjf06"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        air = "only"
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "knockdown"
        }
    },
    unparsed = { "秋水长天，落霞孤鹜，凭空落剑直刺空中目标，造成{@0:伤害包}", "同时根据自身“暗劲”层数附加额外伤害，使用后清除自身“暗劲”" }
}

E["CS_em_ynjf07"] = {
    on_parry = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_em_ynjf08"] = {
    unparsed = { "每击根据自身“暗劲”层数附加额外伤害" }
}

E["CS_em_yxc02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_em_yxc03"] = {
    on_parry_break = {
        {
            mp_restore = 0
        }
    }
}

E["CS_em_yxc04"] = {
    unparsed = { "防御架势，招架时以柔绵化劲方式，使攻击之敌体魄降低（降低20体魄持续15s）" }
}

E["CS_em_yxc06"] = {
    unparsed = { "挥刺连舞，特殊气劲可加速恢复自身及友方淤血" }
}

E["CS_em_yxc06_a"] = {
    unparsed = { "挥刺连舞，特殊气劲可加速恢复自身及友方气血" }
}

E["CS_em_yxc07"] = {
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_emqj_lbc01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_em_lbc01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_emqj_lbc02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "同时可额外增加自身怒气，增加量随“离苦”状态叠加而增加" }
}

E["CS_emqj_lbc03"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "stun"
        }
    }
}

E["CS_emqj_lbc03_sky"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "stun"
        }
    }
}

E["CS_emqj_lbc04"] = {
    unparsed = { "破防成功时，打散敌人全部怒气，并根据打散的怒气量附加柔伤害" }
}

E["CS_emqj_lbc05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_em_lbc01",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "（只有学会招式【爱别离苦】后，招架才会增加“离苦”状态）" }
}

E["CS_emqj_lbc06"] = {
    on_hit = {
        {
            control_kind = "stun"
        }
    }
}

E["CS_emqj_lbc06_sky"] = {
    on_hit = {
        {
            control_kind = "stun"
        }
    }
}

E["CS_emqj_lbc07"] = {
    on_hit = {
        {
            apply_buff = "一层“绝俗",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "该状态叠加层数越高，伤害越高", "此状态下攻击敌人时，会根据自身“离苦”层数附加伤害" }
}

E["CS_emqj_lbc08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "red"
    },
    on_hit = {
        {
            apply_buff = "一层“金顶佛光",
            apply_life_ms = 14000,
            apply_resolved = false,
            apply_stacks = 1
        },
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 20000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 20000 }
    }
}

E["CS_eqp_lzzy"] = {
    unparsed = { "调集内力抵御伤害，持续15秒，最多抵御自身气血上限20%的伤害" }
}

E["CS_eqp_zzcf"] = {
    unparsed = { "5秒内气血翻腾，锋不可当：命中和造成致命一击的几率都大幅提升，伤害忽视招架防御" }
}

E["CS_fire"] = {
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_fire_fb_aoe"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_gb_dgbf01"] = {
    on_hit = {
        {
            apply_buff = "挑拨",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_gb_dgbf02"] = {
    unparsed = { "突击斜刺，对目标造成{@0:伤害包}，并消耗目标身上1层“挑拨”，限制其使用虚招" }
}

E["CS_gb_dgbf03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "同时可消耗目标身上的1层“挑拨”，限制其使用招架" }
}

E["CS_gb_dgbf04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "同时可消耗目标身上的1层“挑拨”，限制其移动" }
}

E["CS_gb_dgbf05"] = {
    on_parry_break = {
        {
            apply_buff = "压扁",
            apply_resolved = false,
            apply_stacks = 1
        },
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong",
            control_within_ms = 3000
        }
    },
    by_level = {
        [2] = { ["on_parry_break.2.control_dur_ms"] = 4000, ["on_parry_break.2.control_within_ms"] = 4000 },
        [3] = { ["on_parry_break.2.control_dur_ms"] = 5000, ["on_parry_break.2.control_within_ms"] = 5000 },
        [4] = { ["on_parry_break.2.control_dur_ms"] = 5000, ["on_parry_break.2.control_within_ms"] = 5000 },
        [5] = { ["on_parry_break.2.control_dur_ms"] = 5000, ["on_parry_break.2.control_within_ms"] = 5000 },
        [6] = { ["on_parry_break.2.control_dur_ms"] = 5000, ["on_parry_break.2.control_within_ms"] = 5000 },
        [7] = { ["on_parry_break.2.control_dur_ms"] = 5000, ["on_parry_break.2.control_within_ms"] = 5000 },
        [8] = { ["on_parry_break.2.control_dur_ms"] = 5000, ["on_parry_break.2.control_within_ms"] = 5000 },
        [9] = { ["on_parry_break.2.control_dur_ms"] = 5000, ["on_parry_break.2.control_within_ms"] = 5000 },
        [10] = { ["on_parry_break.2.control_dur_ms"] = 5000, ["on_parry_break.2.control_within_ms"] = 5000 },
        [11] = { ["on_parry_break.2.control_dur_ms"] = 5000, ["on_parry_break.2.control_within_ms"] = 5000 },
        [12] = { ["on_parry_break.2.control_dur_ms"] = 5000, ["on_parry_break.2.control_within_ms"] = 5000 },
        [13] = { ["on_parry_break.2.control_dur_ms"] = 5000, ["on_parry_break.2.control_within_ms"] = 5000 },
        [14] = { ["on_parry_break.2.control_dur_ms"] = 5000, ["on_parry_break.2.control_within_ms"] = 5000 },
        [15] = { ["on_parry_break.2.control_dur_ms"] = 5000, ["on_parry_break.2.control_within_ms"] = 5000 },
        [16] = { ["on_parry_break.2.control_dur_ms"] = 5000, ["on_parry_break.2.control_within_ms"] = 5000 },
        [17] = { ["on_parry_break.2.control_dur_ms"] = 5000, ["on_parry_break.2.control_within_ms"] = 5000 },
        [18] = { ["on_parry_break.2.control_dur_ms"] = 5000, ["on_parry_break.2.control_within_ms"] = 5000 },
        [19] = { ["on_parry_break.2.control_dur_ms"] = 5000, ["on_parry_break.2.control_within_ms"] = 5000 }
    }
}

E["CS_gb_dgbf06"] = {
    on_hit = {
        {
            apply_buff = "挑拨",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "且有几率使攻击者被“缴械”：4秒内无法使用武器攻击", "（敌方身上“挑拨”层数越多，被缴械几率越大）" }
}

E["CS_gb_dgbf07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_gb_dgbf07",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_gb_dgbf08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 }
    }
}

E["CS_gb_dgbf08_a"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_gb_dgbf11"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_gb_ksbf01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_gb_ksbf02"] = {
    unparsed = { "向前方敌人喷吐“酒雾”，喷倒范围内目标，并降低其移动速度和命中几率" }
}

E["CS_gb_ksbf03"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_gb_ksbf04"] = {
    on_hit = {
        {
            mp_drain = true,
            mp_drain_dur_ms = 6000
        }
    },
    by_level = {
        [2] = { ["on_hit.1.mp_drain_dur_ms"] = 8000 },
        [3] = { ["on_hit.1.mp_drain_dur_ms"] = 10000 },
        [4] = { ["on_hit.1.mp_drain_dur_ms"] = 12000 },
        [5] = { ["on_hit.1.mp_drain_dur_ms"] = 15000 },
        [6] = { ["on_hit.1.mp_drain_dur_ms"] = 15000 },
        [7] = { ["on_hit.1.mp_drain_dur_ms"] = 15000 },
        [8] = { ["on_hit.1.mp_drain_dur_ms"] = 15000 },
        [9] = { ["on_hit.1.mp_drain_dur_ms"] = 15000 },
        [10] = { ["on_hit.1.mp_drain_dur_ms"] = 15000 },
        [11] = { ["on_hit.1.mp_drain_dur_ms"] = 15000 },
        [12] = { ["on_hit.1.mp_drain_dur_ms"] = 15000 },
        [13] = { ["on_hit.1.mp_drain_dur_ms"] = 15000 },
        [14] = { ["on_hit.1.mp_drain_dur_ms"] = 15000 },
        [15] = { ["on_hit.1.mp_drain_dur_ms"] = 15000 },
        [16] = { ["on_hit.1.mp_drain_dur_ms"] = 15000 },
        [17] = { ["on_hit.1.mp_drain_dur_ms"] = 15000 },
        [18] = { ["on_hit.1.mp_drain_dur_ms"] = 15000 },
        [19] = { ["on_hit.1.mp_drain_dur_ms"] = 15000 },
        [20] = { ["on_hit.1.mp_drain_dur_ms"] = 15000 }
    },
    unparsed = { "同时点燃目标身上的“酒雾”（自身醉酒层数越高，目标所受伤害越大）" }
}

E["CS_gb_ksbf05"] = {
    unparsed = { "防御状态，招架成功时有30%几率增加自身“醉意”" }
}

E["CS_gb_ksbf07"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    },
    unparsed = { "挥棍破空而击，气劲对空中目标造成{@0:伤害包}" }
}

E["CS_gb_lhz01"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    unparsed = { "同时激活全部【霸绝】及【逐犬】武技则自身存在“莲花真气”时，可消耗一层“莲花真气”使招式释放更为" }
}

E["CS_gb_lhz02"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_gb_lhz02a"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_gb_lhz04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 8 }
    },
    unparsed = { "招式释放后恢复调息时间5秒，每触发一次射程提高效果，此恢复调息效果便衰弱1秒（射程提高至最高时，", "破防成功额外获得1层“生莲”效果" }
}

E["CS_gb_lhz04_hide"] = {
    unparsed = { "每存在3层“莲花真气”均可使招式射程提高2米", "招式释放后恢复调息时间5秒，每触发一次射程提高效果，此恢复调息效果便衰弱1秒（射程提高至最高时，", "破防成功额外获得1层“生莲”效果" }
}

E["CS_gb_lhz05"] = {
    unparsed = { "防御架势，每次成功招架后，可化解对方之力转为“莲花真气”储存起来，最多可叠加3层", "同时激活全部【霸绝】及【逐犬】武技则招式调息时间提高至30秒，招式使用后的12秒内每秒源源不断的" }
}

E["CS_gb_lhz05_hide"] = {
    unparsed = { "防御架势，每次成功招架后，可化解对方之力转为“莲花真气”储存起来，最多可叠加3层", "招式调息时间提高至30秒，招式使用后的12秒内每秒源源不断的提供“莲花真气”效果(需在战斗中)" }
}

E["CS_gb_lhz06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "stiffen",
            per_hit = true
        }
    },
    unparsed = { "同时激活全部【霸绝】及【逐犬】武技则招式出手时自身若不存在“莲花真气”则将直接获得3层" }
}

E["CS_gb_lhz07"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        air = "only"
    }
}

E["CS_gb_lsbf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "冲锋招式，灵动摆击，对目标造成{@0:伤害包}，丐帮宗门弟子攻击身中“灵蛇奇毒”的未招架目标时，" }
}

E["CS_gb_lsbf03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_gb_lsbf04"] = {
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_gb_lsbf05"] = {
    unparsed = { "每次招架成功都会燃烧斗志，使自身攻击附加7点阳伤害（伤害受近身威力影响）", "丐帮弟子格挡后可额外获得一层“蛇势”：灵蛇棒法招式威力提高10%与10%的外功招架免伤，持续10" }
}

E["CS_gb_lsbf07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 5000,
            control_kind = "ban_qinggong"
        },
        {
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    },
    unparsed = { "丐帮弟子使用时可压制倒地敌人" }
}

E["CS_gb_xlsbzc01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "slow"
        }
    },
    by_level = {
        [7] = { ["compile.aoe_radius_m"] = 20 },
        [8] = { ["compile.aoe_radius_m"] = 20 },
        [9] = { ["compile.aoe_radius_m"] = 20 },
        [10] = { ["compile.aoe_radius_m"] = 20 }
    }
}

E["CS_gb_xlsbzc02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_gb_xlsbzc03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_gb_xlsbzc04"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_gb_xlsbzc05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_parry_break = {
        {
            apply_buff = "buf_CS_gb_xlsbzc05",
            apply_resolved = true,
            apply_stacks = 1,
            control_within_ms = 5000
        }
    },
    by_level = {
        [7] = { ["compile.aoe_radius_m"] = 15 },
        [8] = { ["compile.aoe_radius_m"] = 15 },
        [9] = { ["compile.aoe_radius_m"] = 15 },
        [10] = { ["compile.aoe_radius_m"] = 15 }
    }
}

E["CS_gb_xlsbzc06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "knockdown"
        }
    }
}

E["CS_gb_xlsbzc07"] = {
    on_hit = {
        {
            control_dur_ms = 1000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_gbqj_lsbf01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    }
}

E["CS_gbqj_lsbf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "冲锋招式，灵动摆击，对目标造成{@0:伤害包}，丐帮宗门弟子攻击身中“灵蛇奇毒”的未招架目标时，" }
}

E["CS_gbqj_lsbf03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_gbqj_lsbf04"] = {
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_gbqj_lsbf05"] = {
    unparsed = { "每次招架成功都会燃烧斗志，使自身攻击附加14点阳伤害（伤害受近身威力影响）", "丐帮弟子格挡后可额外获得一层“蛇势”：灵蛇棒法招式威力提高10%与10%的外功招架免伤，持续10" }
}

E["CS_gbqj_lsbf06"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "red"
    }
}

E["CS_gbqj_lsbf07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 5000,
            control_kind = "ban_qinggong"
        },
        {
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    },
    unparsed = { "丐帮弟子使用时可压制倒地敌人" }
}

E["CS_gbzp_dgbf01"] = {
    on_hit = {
        {
            apply_buff = "挑拨",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_gbzp_dgbf02"] = {
    unparsed = { "突击斜刺，对目标造成{@0:伤害包}，并消耗目标身上1层“挑拨”，限制其使用虚招" }
}

E["CS_gbzp_dgbf03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "同时可消耗目标身上的1层“挑拨”，限制其使用招架" }
}

E["CS_gbzp_dgbf04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "同时可消耗目标身上的1层“挑拨”，限制其移动" }
}

E["CS_gbzp_dgbf05"] = {
    on_parry_break = {
        {
            apply_buff = "压扁",
            apply_resolved = false,
            apply_stacks = 1
        },
        {
            control_dur_ms = 2000,
            control_kind = "ban_qinggong",
            control_within_ms = 2000
        }
    },
    by_level = {
        [3] = { ["on_parry_break.2.control_dur_ms"] = 3000, ["on_parry_break.2.control_within_ms"] = 3000 },
        [4] = { ["on_parry_break.2.control_dur_ms"] = 3000, ["on_parry_break.2.control_within_ms"] = 3000 },
        [5] = { ["on_parry_break.2.control_dur_ms"] = 4000, ["on_parry_break.2.control_within_ms"] = 4000 },
        [6] = { ["on_parry_break.2.control_dur_ms"] = 4000, ["on_parry_break.2.control_within_ms"] = 4000 },
        [7] = { ["on_parry_break.2.control_dur_ms"] = 4000, ["on_parry_break.2.control_within_ms"] = 4000 },
        [8] = { ["on_parry_break.2.control_dur_ms"] = 4000, ["on_parry_break.2.control_within_ms"] = 4000 },
        [9] = { ["on_parry_break.2.control_dur_ms"] = 4000, ["on_parry_break.2.control_within_ms"] = 4000 },
        [10] = { ["on_parry_break.2.control_dur_ms"] = 4000, ["on_parry_break.2.control_within_ms"] = 4000 },
        [11] = { ["on_parry_break.2.control_dur_ms"] = 4000, ["on_parry_break.2.control_within_ms"] = 4000 },
        [12] = { ["on_parry_break.2.control_dur_ms"] = 4000, ["on_parry_break.2.control_within_ms"] = 4000 },
        [13] = { ["on_parry_break.2.control_dur_ms"] = 4000, ["on_parry_break.2.control_within_ms"] = 4000 },
        [14] = { ["on_parry_break.2.control_dur_ms"] = 4000, ["on_parry_break.2.control_within_ms"] = 4000 },
        [15] = { ["on_parry_break.2.control_dur_ms"] = 4000, ["on_parry_break.2.control_within_ms"] = 4000 },
        [16] = { ["on_parry_break.2.control_dur_ms"] = 4000, ["on_parry_break.2.control_within_ms"] = 4000 },
        [17] = { ["on_parry_break.2.control_dur_ms"] = 4000, ["on_parry_break.2.control_within_ms"] = 4000 }
    }
}

E["CS_gbzp_dgbf06"] = {
    on_hit = {
        {
            apply_buff = "挑拨",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "且有几率使攻击者被“缴械”：3秒内无法使用武器攻击", "（敌方身上“挑拨”层数越多，被缴械几率越大）" }
}

E["CS_gbzp_dgbf07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_gbzp_dgbf07",
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            condition = { unparryable = true },
            control_kind = "launch"
        }
    }
}

E["CS_gbzp_dgbf08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 }
    },
    on_hit = {
        {
            condition = { unparryable = true },
            control_kind = "launch"
        }
    }
}

E["CS_gm_txjj01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_gm_txjj01",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_gm_txjj02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_gm_txjj03"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_gm_txjj04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_gm_txjj04",
            apply_life_ms = 15000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_gm_txjj05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_gm_txjj05",
            apply_life_ms = 2000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 2000,
            control_kind = "ban_qinggong",
            per_hit = true
        }
    }
}

E["CS_gm_txjj06"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    },
    on_parry_break = {
        {
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_gm_txjj07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_gm_txjj08"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push",
            heal = true
        }
    }
}

E["CS_gm_ynsxj01"] = {
    compile = {
        cast_armor = "yellow"
    }
}

E["CS_gm_ynsxj02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_gm_ynsxj04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_gm_ynsxj05"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_gm_ynsxj06"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_gm_ynsxj06",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_gm_ynsxj07"] = {
    unparsed = { "动离忧，泪难收，招架成功时，有一定几率获得“柳荫”状态：闪避大幅提升，不会被冲锋招式锁定，持续3" }
}

E["CS_hs_hsjf01"] = {
    on_hit = {
        {
            control_kind = "launch"
        },
        {
            control_dur_ms = 10000,
            control_kind = "launch"
        }
    }
}

E["CS_hs_hsjf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "腿攻下盘，倒地剖腹，对目标造成{@0:伤害包}" }
}

E["CS_hs_hsjf03"] = {

}

E["CS_hs_hsjf04"] = {
    compile = {
        air = "only"
    }
}

E["CS_hs_hsjf05"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_hs_hsjf05_hide"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_hs_hsjf06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_hs_hsjf07"] = {
    self_state = {
        {
            apply_buff = "buf_CS_hs_hsjf07",
            apply_life_ms = 2000,
            apply_resolved = true
        }
    },
    unparsed = { "（使用非锁定冲锋招式则可提前脱离该状态）（提升招式层数，可增加成功闪避后造成的伤害）" }
}

E["CS_hs_hsjf08"] = {
    on_hit = {
        {
            apply_buff = "落木",
            apply_life_ms = 8000,
            apply_resolved = false,
            apply_stacks = 1,
            per_hit = true
        }
    }
}

E["CS_hs_kfkj01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_hs_kfkj01_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_hs_kfkj02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_hs_kfkj03"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "coma"
        }
    }
}

E["CS_hs_kfkj04"] = {
    compile = {
        air = "never"
    },
    on_hit = {
        {
            control_dur_ms = 4000,
            control_kind = "coma"
        }
    },
    unparsed = { "每次施展该招式后的4秒内施展【纵风止燎】【风行草偃】【风驰霆击】将附带黄霸体效果" }
}

E["CS_hs_kfkj05"] = {
    on_parry_break = {
        {
            control_within_ms = 5000
        }
    }
}

E["CS_hs_kfkj06"] = {
    self_state = {
        {
            apply_buff = "buf_CS_hs_kfkj06",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_hs_kfkj07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_ahlj01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_ahlj02"] = {
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "coma",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_ahlj03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_dur_ms = 1000,
            control_kind = "coma",
            per_hit = true
        }
    }
}

E["CS_jh_ahlj04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_ahlj04_03",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 5000,
            control_kind = "launch"
        }
    }
}

E["CS_jh_ahlj04_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_ahlj05"] = {
    self_state = {
        {
            apply_buff = "心若冰清、天塌不惊",
            apply_life_ms = 5000,
            apply_resolved = false
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_jh_ahlj06"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch",
            per_hit = true
        }
    }
}

E["CS_jh_ahlj07"] = {
    on_hit = {
        {
            control_dur_ms = 1000,
            control_kind = "coma",
            per_hit = true
        }
    },
    unparsed = { "攻击或被攻击时获得额外怒气（2点额外怒气）" }
}

E["CS_jh_bgd01"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "buf_clone055_lljjc_008",
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.2
        },
        {
            control_kind = "coma"
        }
    }
}

E["CS_jh_bgd02"] = {
    on_hit = {
        {
            apply_buff = "地",
            apply_resolved = false,
            apply_stacks = 1,
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    unparsed = { "持续7秒，状态结束时，自身存在“山”效果则恢复至原状态（“地”触发时自身的气血内力值）" }
}

E["CS_jh_bgd03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_clone055_lljjc_008",
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_bgd04"] = {
    on_hit = {
        {
            apply_buff = "雷",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_bgd05"] = {
    on_hit = {
        {
            apply_buff = "buf_clone021_shui",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 2000,
            control_kind = "coma"
        }
    }
}

E["CS_jh_bgd06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 3 },
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_clone034_huo_wf_01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_bgd07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "泽",
            apply_life_ms = 8000,
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 8000,
            control_kind = "ban_qinggong"
        },
        {
            control_dur_ms = 2000,
            control_kind = "coma"
        }
    }
}

E["CS_jh_bgd08"] = {
    on_parry = {
        {
            apply_buff = "山",
            apply_life_ms = 8000,
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_bxjf01"] = {
    on_hit = {
        {
            control_kind = "stun"
        }
    },
    unparsed = { "命中目标时，自身获得1层“邪气”状态：增加移动速度和本套路招式冲锋距离，持续15秒，可叠加10层" }
}

E["CS_jh_bxjf02"] = {
    unparsed = { "菩提本无树，明镜亦非台，招架成功时，自身获得“邪气”状态：增加移动速度和本套路招式冲锋距离，持续" }
}

E["CS_jh_bxjf03"] = {
    on_hit = {
        {
            control_dur_ms = 15000,
            control_kind = "silence"
        }
    },
    unparsed = { "（最多可侵入1层“邪气”转为“风邪入侵”）（招式随自身移动速度的增加而提升伤害）（此套路必须是去" }
}

E["CS_jh_bxjf04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_bxjf05"] = {

}

E["CS_jh_bxjf06"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_bxjf06",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 10000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_bxjf07"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "再次按键，消耗10层“邪气”状态，可施展追击招式【万众伏诛】" }
}

E["CS_jh_bxjf07_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_bxjf08"] = {
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "root"
        }
    }
}

E["CS_jh_chz01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    by_level = {
        [12] = { ["on_hit.1.apply_stacks"] = 2, ["on_hit.1.control_dur_ms"] = 15000, ["on_hit.1.displacement_distance_m"] = 0 }
    },
    unparsed = { "出招时自身获得“指劲”状态：提升15%外功伤害减免忽视、15%内功伤害减免忽视、15%外功招架忽" }
}

E["CS_jh_chz02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    by_level = {
        [12] = { ["on_hit.1.apply_stacks"] = 2, ["on_hit.1.control_dur_ms"] = 15000, ["on_hit.1.displacement_distance_m"] = 0 }
    },
    unparsed = { "出招时自身获得“指劲”状态：提升15%外功伤害减免忽视、15%内功伤害减免忽视、15%外功招架忽" }
}

E["CS_jh_chz03"] = {
    compile = {
        cast_armor = "red"
    },
    by_level = {
        [12] = { ["on_hit.1.control_dur_ms"] = 15000 }
    }
}

E["CS_jh_chz04"] = {
    unparsed = { "出招时自身获得“指劲”状态：提升15%内/外功伤害减免忽视、15%外功招架忽视、15%内功化解忽" }
}

E["CS_jh_chz05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_chz05",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_chz06"] = {
    unparsed = { "招架成功时，自身获得“捻弹随喜”状态：增加招式【指矢天日】、招式【旁指曲谕】、招式【弹指倏生】、" }
}

E["CS_jh_chz07"] = {
    unparsed = { "目标获得“凌虚（敌）”状态：降低9点近身威力和内功威力，持续1分钟，可叠加5次" }
}

E["CS_jh_cjhyy01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_cjhyy08",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        },
        {
            control_dur_ms = 2000,
            control_kind = "coma"
        }
    }
}

E["CS_jh_cjhyy02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_cjhyy03"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_cjhyy08",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_cjhyy04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_cjhyy04",
            apply_life_ms = 3000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_jh_cjhyy05"] = {
    compile = {
        cast_armor = "yellow",
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "对范围内的敌人造成{@0:伤害包}，并使未招架的敌人缴械2秒" }
}

E["CS_jh_cjhyy06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_cjhyy06",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_cjhyy07"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_cjhyy08",
            apply_life_ms = 2000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_cjhyy08"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_cjhyy08",
            apply_life_ms = 2000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 3000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 3000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 4000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 4000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 5000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 5000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 8000 }
    }
}

E["CS_jh_cljf01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "对应灵犀(男)、灵犀(女)状态",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_cljf02"] = {

}

E["CS_jh_cljf03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    }
}

E["CS_jh_cljf04"] = {
    on_hit = {
        {
            apply_buff = "水乳交融状态",
            apply_life_ms = 7000,
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 11000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 12000 }
    }
}

E["CS_jh_cljf05"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "red"
    },
    on_hit = {
        {
            control_dur_ms = 1000,
            control_kind = "stun"
        },
        {
            apply_buff = "舞剑层数",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    by_level = {
        [3] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [4] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [5] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [6] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [7] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [8] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [13] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [14] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [15] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [16] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [17] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [18] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [19] = { ["on_hit.1.control_dur_ms"] = 3000 }
    },
    unparsed = { "同时自身进入舞剑状态：站立不动，并对周围敌人造成持续伤害" }
}

E["CS_jh_cljf06"] = {
    unparsed = { "剑势含情脉脉：可紊乱目标经络，限制其气血回复11点" }
}

E["CS_jh_cljf07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "相依状态",
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 1000,
            control_kind = "silence"
        }
    },
    by_level = {
        [5] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [6] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [7] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [8] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [13] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [14] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [15] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [16] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [17] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [18] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [19] = { ["on_hit.1.control_dur_ms"] = 2000 }
    }
}

E["CS_jh_cqgf01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_cqgf01a"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_cqgf02"] = {
    on_parry_break = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_cqgf03"] = {
    unparsed = { "防御架势，简单而实用，每次招架成功可额外减免3点伤害" }
}

E["CS_jh_cqgf04"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    },
    by_level = {
        [6] = { ["on_hit.2.apply_stacks"] = 1 },
        [7] = { ["on_hit.2.apply_stacks"] = 1 },
        [8] = { ["on_hit.2.apply_stacks"] = 1 },
        [9] = { ["on_hit.2.apply_stacks"] = 1 },
        [10] = { ["on_hit.2.apply_stacks"] = 1 },
        [11] = { ["on_hit.2.apply_stacks"] = 1 },
        [12] = { ["on_hit.2.apply_stacks"] = 1 },
        [13] = { ["on_hit.2.apply_stacks"] = 1 },
        [14] = { ["on_hit.2.apply_stacks"] = 1 },
        [15] = { ["on_hit.2.apply_stacks"] = 1 },
        [16] = { ["on_hit.2.apply_stacks"] = 1 }
    }
}

E["CS_jh_cqgf04_sky"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_cqgf07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "封招和键位错乱效果",
            apply_life_ms = 3000,
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    }
}

E["CS_jh_csdwg01"] = {
    on_hit = {
        {
            apply_buff = "5秒的风煞状态",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        },
        {
            apply_buff = "7秒的风行状态",
            apply_resolved = false,
            apply_stacks = 1
        },
        {
            apply_buff = "5秒的风煞状态",
            apply_life_ms = 10000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        },
        {
            control_dur_ms = 3000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_csdwg01_hide"] = {
    on_hit = {
        {
            apply_buff = "7秒的风行状态",
            apply_resolved = false,
            apply_stacks = 1
        },
        {
            apply_buff = "5秒的风煞状态",
            apply_life_ms = 10000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        },
        {
            control_dur_ms = 3000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_csdwg02"] = {
    on_parry_break = {
        {
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_jh_csdwg02_hide"] = {
    on_parry_break = {
        {
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_jh_csdwg03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "7秒的风行状态（激活此效果需与巫",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_csdwg04"] = {
    compile = {
        cast_armor = "yellow",
        air = "never"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "coma"
        }
    }
}

E["CS_jh_csdwg04_hide"] = {
    compile = {
        cast_armor = "yellow",
        air = "never"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "coma",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_csdwg05"] = {
    on_hit = {
        {
            control_dur_ms = 1000,
            control_kind = "stun"
        }
    },
    unparsed = { "掷棍对目标造成{@0:伤害包}，如目标未招架则将其拉进自身" }
}

E["CS_jh_csdwg06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "root"
        },
        {
            apply_buff = "一层赤煞",
            apply_resolved = false,
            apply_stacks = 1,
            control_kind = "knockdown"
        },
        {
            apply_buff = "赤煞）独武状态下无法获得赤煞招式",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_csdwg06_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "root"
        },
        {
            apply_buff = "一层赤煞",
            apply_resolved = false,
            apply_stacks = 1,
            control_kind = "knockdown"
        },
        {
            apply_buff = "赤煞）独武状态下无法获得赤煞招式",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_csdwg07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "赤水容状态",
            apply_life_ms = 8000,
            apply_resolved = false,
            apply_stacks = 1
        },
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "root"
        },
        {
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "coma"
        }
    },
    by_level = {
        [7] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 12000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 12000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 15000 }
    },
    unparsed = { "招式释放的同时给自己增加独武状态，持续5秒：套路伤害提高30%，每次使用【赤煞独武棍】招式会延长" }
}

E["CS_jh_csdwg07_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_jh_csdwg08"] = {
    unparsed = { "轰击目标造成{@0:伤害包}招式范围增加至6米（激活此效果需与巫墟宗派深处山鬼地窟的巫莎琪（77" }
}

E["CS_jh_csdwg08_hide"] = {
    unparsed = { "轰击目标造成{@0:伤害包}招式范围增加至6米（激活此效果需与巫墟宗派深处山鬼地窟的巫莎琪（77" }
}

E["CS_jh_csqns01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_csqns02"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_csqns03_01",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 5000,
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_csqns02_sky"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_csqns03_01",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 5000,
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_csqns03"] = {
    on_parry_break = {
        {
            control_dur_ms = 3000,
            control_kind = "root"
        },
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_csqns03_sky"] = {
    on_parry_break = {
        {
            control_dur_ms = 5000,
            control_kind = "root"
        }
    }
}

E["CS_jh_csqns04"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "launch"
        }
    }
}

E["CS_jh_csqns04_hide"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_csqns05"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_csqns05_hide"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_csqns05_sky"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_csqns06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 20 },
        cast_armor = "yellow"
    },
    self_state = {
        {
            apply_buff = "buf_CS_jh_csqns06_01",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制" }
}

E["CS_jh_csqns06_hide1"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 20 },
        cast_armor = "yellow"
    }
}

E["CS_jh_csqns06_hide2"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_cyjf01"] = {
    unparsed = { "若敌人未能及时招架，必会侵染“炎火剑气”：受到持续伤害，叠加层数越高伤害越大" }
}

E["CS_jh_cyjf02"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    },
    unparsed = { "韩似锦常用此招破除敌人的招架" }
}

E["CS_jh_cyjf03"] = {
    unparsed = { "残阳架势，凝聚“残阳剑气”护身：成功招架时，传递三道“离火真气”至敌方体内，同时回复自身一丝内力" }
}

E["CS_jh_cyjf04"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_cyjf06"] = {
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_cyjf07"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_cyjf08"] = {
    on_hit = {
        {
            control_kind = "launch"
        },
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_dgjj01"] = {
    on_hit = {
        {
            apply_buff = "硬直时",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_dgjj02"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_dgjj04"] = {
    unparsed = { "-寒蛟翻江，梭云摄月-荡剑萍空，巡弋突兀，击晕前方敌人，并造成{@0:伤害包}" }
}

E["CS_jh_dgjj05"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_dgjj07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_dgjj08"] = {
    on_hit = {
        {
            apply_buff = "无懈可击",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_dgjj09"] = {

}

E["CS_jh_dl01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    }
}

E["CS_jh_dl02"] = {
    unparsed = { "舒活经络恢复少量气血，当效果存在的时候，任何攻击自己的人都会受到内劲反噬，损失内力，效果持续6秒" }
}

E["CS_jh_dl03"] = {
    on_hit = {
        {
            apply_buff = "混元",
            apply_life_ms = 20000,
            apply_resolved = false,
            apply_stacks = 1
        },
        {
            apply_buff = "buf_CS_jh_dl03_01",
            apply_life_ms = 20000,
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.25,
            control_dur_ms = 20000,
            control_kind = "launch"
        }
    }
}

E["CS_jh_dl04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            chance = 0.25,
            control_kind = "silence"
        }
    }
}

E["CS_jh_dl05"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 6 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_dl06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "coma"
        }
    }
}

E["CS_jh_dl07"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 6 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_dl07",
            apply_life_ms = 20000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_dl08"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 10 }
    }
}

E["CS_jh_dldm01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_dldm01",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 15000,
            control_kind = "launch"
        }
    }
}

E["CS_jh_dldm02"] = {
    unparsed = { "破敌招架时，可使其“气血翻滚”：被铁爪攻击时遭受额外伤害" }
}

E["CS_jh_dldm03"] = {
    unparsed = { "若敌来不及招架，必被紧紧束缚，定住身形" }
}

E["CS_jh_dldm04"] = {
    on_parry = {
        {
            apply_buff = "buf_CS_jh_dldm04_1",
            apply_life_ms = 8000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 8000,
            control_kind = "slow"
        }
    },
    by_level = {
        [2] = { ["on_parry.1.apply_life_ms"] = 9000, ["on_parry.1.control_dur_ms"] = 9000 },
        [3] = { ["on_parry.1.apply_life_ms"] = 10000, ["on_parry.1.control_dur_ms"] = 10000 },
        [4] = { ["on_parry.1.apply_life_ms"] = 11000, ["on_parry.1.control_dur_ms"] = 11000 },
        [5] = { ["on_parry.1.apply_life_ms"] = 12000, ["on_parry.1.control_dur_ms"] = 12000 },
        [6] = { ["on_parry.1.apply_life_ms"] = 14000, ["on_parry.1.control_dur_ms"] = 14000 },
        [7] = { ["on_parry.1.apply_life_ms"] = 16000, ["on_parry.1.control_dur_ms"] = 16000 }
    }
}

E["CS_jh_dldm05"] = {
    unparsed = { "若敌人未及时招架，则将其拽至身边，并使其“气血混乱”" }
}

E["CS_jh_dldm06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_dldm07"] = {
    on_hit = {
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_dpgx01"] = {
    on_hit = {
        {
            control_kind = "stun"
        }
    },
    unparsed = { "（自身怒气越高，招式暴击几率越大）" }
}

E["CS_jh_dpgx02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "冲锋招式，迅如鬼魅，连斩目标造成{@0:伤害包}，当自身怒气大于60时，暴击伤害提高30%" }
}

E["CS_jh_dpgx03"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    },
    on_parry_break = {
        {
            control_kind = "knockdown"
        }
    },
    unparsed = { "（自身怒气越高，招式暴击几率越大）" }
}

E["CS_jh_dpgx04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "招式100%暴击空中的目标", "自身怒气大于60时，暴击伤害提高30%" }
}

E["CS_jh_dpgx05"] = {
    unparsed = { "防守架势，真气萦绕周身，招架成功时：降低目标韧性，使之更容易被暴击", "当自身怒气高于60时，下一次出招时不消耗气血" }
}

E["CS_jh_dpgx06"] = {
    unparsed = { "热血之招，逆行经脉，催逼45%内力转换为60点怒气，可持续7秒", "每次使用该招式时，可重置怒气持续时间", "(此招式获得的怒气无法叠加)" }
}

E["CS_jh_dpgx07"] = {
    unparsed = { "勾心异术，枯木回春，可将自身全部怒气都转换成气血，1点怒气回复0.2%的气血" }
}

E["CS_jh_dtd01"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_dur_ms = 10000,
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_dtd02"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_dtd02_01",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    on_parry_break = {
        {
            apply_buff = "5秒“离踝",
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 5000,
            control_kind = "root"
        }
    }
}

E["CS_jh_dtd03"] = {
    unparsed = { "缠头裹脑，分刀举势招架敌人攻击", "招架成功时，向攻击者传送一道“伤魂”刀劲：内功防御降低5%，气血回复降低5%，持续10秒，可叠加" }
}

E["CS_jh_dtd04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_dtd02_01",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_kind = "launch"
        }
    },
    unparsed = { "(此招式不能在空中使用)" }
}

E["CS_jh_dtd05"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "一层该状态",
            apply_resolved = false,
            apply_stacks = 1,
            per_hit = true
        }
    }
}

E["CS_jh_dtd06"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "5秒“折肘",
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 5000,
            control_kind = "silence"
        }
    }
}

E["CS_jh_fbg01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_fbg02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    }
}

E["CS_jh_fbg02_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_fbg02_hide_01",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 5000,
            control_kind = "slow"
        }
    }
}

E["CS_jh_fbg03"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_fbg05_01_2_stun",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 1000,
            control_kind = "silence"
        }
    }
}

E["CS_jh_fbg03_hide"] = {
    on_hit = {
        {
            control_dur_ms = 5000,
            control_kind = "root"
        }
    }
}

E["CS_jh_fbg03_hide_sky"] = {
    on_hit = {
        {
            apply_buff = "BUFF",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_fbg04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    self_state = {
        {
            apply_buff = "buf_CS_jh_fbg04_01",
            apply_resolved = true
        }
    },
    unparsed = { "（反击有效范围5米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_jh_fbg04_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "stun",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_fbg05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "ban_qinggong"
        }
    },
    self_state = {
        {
            apply_buff = "buf_CS_jh_fbg05_02",
            apply_resolved = true
        }
    }
}

E["CS_jh_fbg05_hide"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "root"
        }
    },
    unparsed = { "伫立凝视，接棍收势，内力外引，收回“风波棍罡”，削减招式【风波不定】5秒回气时间，同时清除自身的" }
}

E["CS_jh_fbg06"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_jh_fbg06_hide"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_fbg06_02",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "launch"
        }
    }
}

E["CS_jh_fgy01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_fgy01",
            apply_life_ms = 3000,
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.25
        }
    }
}

E["CS_jh_fgy02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_fgy04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_jh_fgy05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_fgy05_js",
            apply_life_ms = 3000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "slow"
        }
    }
}

E["CS_jh_fgy06"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    },
    unparsed = { "若目标未招架且存在“堕沙”状态，则目标将倒地不起3秒" }
}

E["CS_jh_fgy07"] = {
    unparsed = { "破防成功时，给目标加“堕沙”状态：移动速度降低50%，内外功命中率降低30%，持续5秒" }
}

E["CS_jh_fgy08"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_fgy08_1",
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.5
        }
    }
}

E["CS_jh_flxmz01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_flxmz01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_flxmz02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_flxmz01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_flxmz03"] = {
    on_hit = {
        {
            condition = { "雷隐" },
            control_dur_ms = 2000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_flxmz04"] = {
    on_parry_break = {
        {
            control_dur_ms = 1000,
            control_kind = "stiffen"
        }
    }
}

E["CS_jh_flxmz05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_flxmz05_01",
            apply_life_ms = 20000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 20000,
            control_kind = "launch"
        }
    },
    unparsed = { "雨天招架时，任何试图攻击自己的敌方都有可能被雷电灼伤" }
}

E["CS_jh_flxmz06"] = {
    unparsed = { "继而酝酿形成“风雷气场”：凝滞空气，使范围内敌人无法施展轻功，气场可持续10秒" }
}

E["CS_jh_flxmz07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { "雷隐" },
            control_kind = "launch"
        }
    }
}

E["CS_jh_flxmz08"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_fmzf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_fmzf02_sky"] = {
    unparsed = { "疯狂的打击目标，造成{@0:伤害包}，此招式可以利用轻功快速的接近对手" }
}

E["CS_jh_fmzf04"] = {
    unparsed = { "翻腾飞踢，踹飞身前敌人至空中，并造成{@0:伤害包}" }
}

E["CS_jh_fmzf05"] = {
    unparsed = { "敏感之式，招架成功时，自身血脉贲张，陷入疯狂，激发“疯意”：疯魔套路招式伤害提高20%，且攻击会" }
}

E["CS_jh_fmzf06"] = {

}

E["CS_jh_fmzf07"] = {
    unparsed = { "疯魔秘技，身形坠魔：消耗当前30%气血，增加50点怒气，瞬间把自己的气血、内力回复到最佳状态，同", "坠魔产生负面效应：自身内力消耗和所受伤害扩大300%，不能主动招架", "坠魔状态持续15秒，然后恢复使用之前的状态，坠魔期间若自身死亡，则直接死亡" }
}

E["CS_jh_fmzf09"] = {
    compile = {
        air = "only"
    }
}

E["CS_jh_ftbz01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_jh_ftbz01_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_ftbz02"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    }
}

E["CS_jh_ftbz03"] = {
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_ftbz05"] = {
    self_state = {
        {
            apply_buff = "buf_CS_jh_ftbz05",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_jh_ftbz06"] = {
    on_hit = {
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_ftbz07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    unparsed = { "水幕向前方行进并对接触到的友方赋予3秒黄霸体效果（60秒内只能触发一次）" }
}

E["CS_jh_fxjf01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    unparsed = { "若击中倒地目标则使自己进入独行之境：6秒内每2秒获取一层“峥嵘”状态" }
}

E["CS_jh_fxjf02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "峥嵘状态（峥嵘状态的级别和层数",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    on_parry_break = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_fxjf02_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "峥嵘状态（峥嵘状态的级别和层数",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    on_parry_break = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_fxjf03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "对目标造成{@0:伤害包}，启动时若自身血量高于目标血量，则可消耗一层“峥嵘”状态使该技能造成额" }
}

E["CS_jh_fxjf04"] = {
    unparsed = { "对目标造成{@0:伤害包}，目标未及招架则造成短暂僵直，最后一击挑飞未招架目标", "技能使用结束时若目标仍未招架则为自己增加一层“峥嵘”状态", "（需要学会招式【万里峥嵘】，“峥嵘”状态才能生效）招式范围增加至8米", "若自身存在3层及以上“峥嵘”状态，施展招式时将提升内功命中率", "招式造成的僵直效果延长1秒（激活此效果需与巫墟宗派深处山鬼地窟的巫莎琪（77,-124）咨询获取" }
}

E["CS_jh_fxjf04_hide"] = {
    unparsed = { "对目标造成{@0:伤害包}，目标未及招架则造成短暂僵直，最后一击挑飞未招架目标", "技能使用结束时若目标仍未招架则为自己增加一层“峥嵘”状态", "（需要学会招式【万里峥嵘】，“峥嵘”状态才能生效）招式范围增加至8米", "若自身存在3层及以上“峥嵘”状态，施展招式时将提升内功命中率", "招式造成的僵直效果延长1秒（激活此效果需与巫墟宗派深处山鬼地窟的巫莎琪（77,-124）咨询获取" }
}

E["CS_jh_fxjf05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 3 }
    },
    on_hit = {
        {
            apply_buff = "一层“峥嵘",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 11000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 12000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 13000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 13000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 15000 }
    }
}

E["CS_jh_fxjf06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "silence"
        }
    }
}

E["CS_jh_fxjf07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_fxjf08"] = {
    compile = {
        cast_armor = "yellow"
    }
}

E["CS_jh_gclhz01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_gclhz01",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 1000,
            control_kind = "slow"
        }
    },
    by_level = {
        [2] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [3] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [5] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [6] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [8] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [13] = { ["on_hit.1.control_dur_ms"] = 3000 }
    }
}

E["CS_jh_gclhz03"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_gclhz03_1",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_jh_gclhz05"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_gclhz06"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_gclhz06",
            apply_life_ms = 4000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 4000,
            control_kind = "slow",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 3000, ["on_hit.1.control_dur_ms"] = 3000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 2000, ["on_hit.1.control_dur_ms"] = 2000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 3000, ["on_hit.1.control_dur_ms"] = 3000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 2000, ["on_hit.1.control_dur_ms"] = 2000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 3000, ["on_hit.1.control_dur_ms"] = 3000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 2000, ["on_hit.1.control_dur_ms"] = 2000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 2000, ["on_hit.1.control_dur_ms"] = 2000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 2000, ["on_hit.1.control_dur_ms"] = 2000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 2000, ["on_hit.1.control_dur_ms"] = 2000 },
        [13] = { ["on_hit.1.apply_life_ms"] = 2000, ["on_hit.1.control_dur_ms"] = 2000 }
    }
}

E["CS_jh_ghqd02"] = {
    unparsed = { "同时给其附加“夺魂”内伤：每次施展轻功会损耗内力，内伤越多损耗越大" }
}

E["CS_jh_ghqd03"] = {
    on_hit = {
        {
            control_kind = "stun"
        }
    },
    unparsed = { "若果目标体内只存在“夺魄”内伤，则引爆内伤造成伤害和减速，并使其无法施展轻功" }
}

E["CS_jh_ghqd04"] = {
    unparsed = { "若目标存在“夺魄”内伤时，每下攻击会使其增加一层内伤", "（此招式可以利用轻功快速的接近目标）" }
}

E["CS_jh_ghqd05"] = {

}

E["CS_jh_ghqd06"] = {
    unparsed = { "勾魂之势，霸气外露，招架成功时：若对方体内存有“夺魄”内伤，则削减其气血", "若对方体内存有“夺魂”内伤，则削减其内力" }
}

E["CS_jh_ghqd07"] = {
    unparsed = { "每击都可引发目标体内的“夺魄”内伤和“夺魂”内伤，造成额外伤害" }
}

E["CS_jh_gwc01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_gwc01",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            heal = true
        }
    }
}

E["CS_jh_gwc02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_gwc03"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        },
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_gwc03_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_gwc04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_gwc04",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_gwc05"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_jh_gwc05",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 5000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_gwc06"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，对霸体无效，可被虚招克制）" }
}

E["CS_jh_gwc06_hide"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，对霸体无效，可被虚招克制）" }
}

E["CS_jh_gwc07"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        can_move = false,
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "launch",
            per_hit = true
        }
    }
}

E["CS_jh_gwzbd01"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "dizzy"
        }
    }
}

E["CS_jh_gwzbd01_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "dizzy"
        }
    }
}

E["CS_jh_gwzbd02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_gwzbd03",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 4
        },
        {
            apply_buff = "的“断筋",
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 15000,
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_gwzbd02_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_gwzbd03",
            apply_life_ms = 15000,
            apply_resolved = true,
            apply_stacks = 4,
            control_dur_ms = 15000,
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_gwzbd03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_gwzbd03",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        },
        {
            apply_buff = "buf_CS_jh_gwzbd03",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_gwzbd04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_gwzbd03",
            apply_life_ms = 15000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            per_hit = true
        }
    }
}

E["CS_jh_gwzbd05"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_gwzbd06"] = {
    self_state = {
        {
            apply_buff = "buf_CS_jh_gwzbd06",
            apply_resolved = true
        }
    }
}

E["CS_jh_gwzbd07"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "stiffen"
        }
    }
}

E["CS_jh_gwzbd07_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "stiffen"
        }
    }
}

E["CS_jh_gwzbd08"] = {
    on_hit = {
        {
            control_kind = "knockdown"
        }
    },
    unparsed = { "（需消耗4层“断筋”状态）" }
}

E["CS_jh_gxfhj01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_gxfhj01_new",
            apply_life_ms = 3000,
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.3
        }
    }
}

E["CS_jh_gxfhj03"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_gxfhj03",
            apply_life_ms = 2000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "slow"
        }
    }
}

E["CS_jh_gxfhj04"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_gxfhj04",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_gxfhj06"] = {
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "root"
        }
    }
}

E["CS_jh_gxfhj07"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_gxfhj07",
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.5
        }
    }
}

E["CS_jh_hjdf01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_hjdf01",
            apply_life_ms = 15000,
            apply_resolved = true,
            apply_stacks = 3,
            condition = { unparried = true }
        }
    }
}

E["CS_jh_hjdf01_hide"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_hjdf01",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 3
        }
    },
    on_parry_break = {
        {
            control_within_ms = 5000
        }
    }
}

E["CS_jh_hjdf02"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_jh_hjdf01",
            apply_resolved = true,
            apply_stacks = 3
        }
    }
}

E["CS_jh_hjdf02_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_hjdf01",
            apply_life_ms = 15000,
            apply_resolved = true,
            apply_stacks = 3,
            condition = { unparried = true }
        }
    }
}

E["CS_jh_hjdf03"] = {
    compile = {
        cast_armor = "yellow"
    }
}

E["CS_jh_hjdf04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "命中未招架目标时：根据自身“缓劲”层数附加额外伤害，每击清除一层" }
}

E["CS_jh_hjdf06"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "命中未招架目标时：根据自身“缓劲”层数附加额外伤害，每击清除一层" }
}

E["CS_jh_hjdf06_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_hjdf07",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    on_parry_break = {
        {
            control_within_ms = 5000
        }
    },
    unparsed = { "并根据自身“缓劲”层数附加额外伤害，每击清除一层" }
}

E["CS_jh_hjdf07"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_dur_ms = 1000,
            control_kind = "coma",
            per_hit = true
        }
    },
    unparsed = { "并根据自身“缓劲”层数附加额外伤害，每击清除一层" }
}

E["CS_jh_hlzmg01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_hlzmg01",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_hlzmg02"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_hlzmg02",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_hlzmg03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_hlzmg03",
            apply_resolved = true,
            apply_stacks = 1,
            per_hit = true
        }
    }
}

E["CS_jh_hlzmg04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_hlzmg03",
            apply_resolved = true,
            apply_stacks = 1,
            per_hit = true
        }
    }
}

E["CS_jh_hlzmg05"] = {
    self_state = {
        {
            apply_buff = "buf_CS_jh_hlzmg05",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    by_level = {
        [2] = { ["self_state.1.apply_life_ms"] = 6000 },
        [3] = { ["self_state.1.apply_life_ms"] = 7000 },
        [4] = { ["self_state.1.apply_life_ms"] = 8000 },
        [5] = { ["self_state.1.apply_life_ms"] = 9000 },
        [6] = { ["self_state.1.apply_life_ms"] = 10000 }
    }
}

E["CS_jh_hlzmg06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_hlzmg07"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_hlzmg03",
            apply_resolved = true,
            apply_stacks = 1,
            per_hit = true
        }
    }
}

E["CS_jh_hq01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_hq02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    on_parry_break = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_hq03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    }
}

E["CS_jh_hq04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "stiffen",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    unparsed = { "在“救命猴毛”状态下施展该招式时，该招式伤害会越来越高（每击叠加10%）" }
}

E["CS_jh_hq04_sky"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    },
    unparsed = { "在“救命猴毛”状态下施展该招式时，该招式伤害会越来越高（每击叠加100%）" }
}

E["CS_jh_hq05"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    },
    unparsed = { "摇头晃脑，跳步乱爪，对选中目标造成{@0:伤害包}，并使未招架的敌人视野降低、键位错乱，持续5秒" }
}

E["CS_jh_hq06"] = {
    unparsed = { "提膝挠腰，金鸡独立，使自身进入百变神猴状态：成功招架敌人攻击时，自身可获得1根“救命猴毛”，持续" }
}

E["CS_jh_hq07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_htxf05"] = {
    unparsed = { "锁定，目标未招架则使用结束后黄霸体2s" }
}

E["CS_jh_hyjf01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_hyjf01",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "（使用2次后第3次会击晕目标1秒）" }
}

E["CS_jh_hyjf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_hyjf03"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    unparsed = { "风寒入侵：限制周围敌人使用轻功，且大幅下降其移动速度（速度随时间慢慢恢复）" }
}

E["CS_jh_hyjf04"] = {
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_hyjf05"] = {
    on_hit = {
        {
            apply_buff = "状态",
            apply_life_ms = 6000,
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 1000,
            control_kind = "stun"
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [13] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [14] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [15] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [16] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [17] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [18] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [19] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 }
    }
}

E["CS_jh_hyjf06"] = {
    unparsed = { "凌寒架势，御剑傲雪：招架成功时，自身获得“傲雪”状态，最多可叠加3层：激发傲骨天性，使得【惊雪无", "涤荡平息怒火，减少【寒心恨雪】施展所消耗的怒气（每1层“傲雪”状态减少1点怒气消耗，每次攻击消耗" }
}

E["CS_jh_hyjf07"] = {
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_hyjf08"] = {
    compile = {
        max_hits = 3
    },
    by_level = {
        [2] = { ["compile.max_hits"] = 4 },
        [3] = { ["compile.max_hits"] = 5 },
        [4] = { ["compile.max_hits"] = 6 },
        [5] = { ["compile.max_hits"] = 7 },
        [6] = { ["compile.max_hits"] = 8 },
        [7] = { ["compile.max_hits"] = 8 },
        [8] = { ["compile.max_hits"] = 8 },
        [9] = { ["compile.max_hits"] = 8 },
        [10] = { ["compile.max_hits"] = 8 },
        [11] = { ["compile.max_hits"] = 8 },
        [12] = { ["compile.max_hits"] = 8 },
        [13] = { ["compile.max_hits"] = 8 },
        [14] = { ["compile.max_hits"] = 8 },
        [15] = { ["compile.max_hits"] = 8 },
        [16] = { ["compile.max_hits"] = 8 },
        [17] = { ["compile.max_hits"] = 8 },
        [18] = { ["compile.max_hits"] = 8 },
        [19] = { ["compile.max_hits"] = 8 }
    },
    unparsed = { "（施展此招式需要10点怒气，每攻击一次消耗10点怒气）" }
}

E["CS_jh_jgjf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_jgjf03"] = {
    on_parry_break = {
        {
            control_kind = "silence"
        }
    }
}

E["CS_jh_jgjf04"] = {
    unparsed = { "戴九履一，通悖乱逆，招架成功时：自身处于天时，则获得“遁甲”状态：反弹50%伤害，可叠加5层，受", "自身处于地利，则获得“伏吟”状态：攻击时附加伤害，可叠加5层，攻击则移除1层", "当自身不处于天时地利时，招架则可获得1道“剑光”，可叠加9层" }
}

E["CS_jh_jgjf05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            control_kind = "root"
        }
    },
    unparsed = { "在天盘剑阵中：自身及队友获得天时（增加5%移动速度），敌人获得天缺（施展轻功增加1点轻功值消耗、" }
}

E["CS_jh_jgjf06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_jgjf07_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_jgjf08"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_jgjf09"] = {
    compile = {
        cast_armor = "red"
    }
}

E["CS_jh_jgmnc01"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "一层“魔吟",
            apply_life_ms = 7000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    },
    unparsed = { "若目标未招架且在空中，则直接将其击落（击落效果10秒内只能触发一次）（激活此效果需与巫墟宗派深处" }
}

E["CS_jh_jgmnc02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    },
    on_parry_break = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_jgmnc03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "突刺目标造成{@0:伤害包}，若目标未招架，则最后一击将在原地留下一尊『魔影』『魔影』存在10秒" }
}

E["CS_jh_jgmnc04"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "昧魔状态",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            control_kind = "stiffen"
        },
        {
            control_kind = "launch"
        },
        {
            control_kind = "launch"
        }
    },
    unparsed = { "昧魔状态：减速10%，持续5秒，可叠加5层" }
}

E["CS_jh_jgmnc05"] = {
    compile = {
        air = "never"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jgmnc05_1",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "stiffen"
        }
    },
    unparsed = { "向身前冲刺，对路径上的敌人造成{@0:伤害包}并短暂僵直冲刺前在原地留下一尊『魔影』『魔影』存在" }
}

E["CS_jh_jgmnc05_hide"] = {
    compile = {
        air = "never"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jgmnc05_hide",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "stiffen"
        }
    }
}

E["CS_jh_jgmnc06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 20 },
        cast_armor = "yellow"
    }
}

E["CS_jh_jgmnc07"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）发起反击的最后一击将在原地留下一尊『魔影』『魔影』存在10秒，且同" }
}

E["CS_jh_jgmnc07_hide"] = {
    unparsed = { "招架，防反" }
}

E["CS_jh_jgmnc08"] = {
    unparsed = { "飞天斩击，对目标造成{@0:伤害包},若目标未招架，则最后一击将在原地留下一尊『魔影』『魔影』存" }
}

E["CS_jh_jgsj01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "silence"
        }
    },
    unparsed = { "刺击目标造成{@0:伤害包}，并吸取伤害35%的气血回复自身" }
}

E["CS_jh_jgsj01_npc"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "silence"
        }
    }
}

E["CS_jh_jgsj02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "dizzy"
        }
    }
}

E["CS_jh_jgsj02_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "dizzy"
        }
    }
}

E["CS_jh_jgsj02_npc"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_jh_jgsj03"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_parry_break = {
        {
            control_dur_ms = 5000,
            control_kind = "knockdown",
            control_within_ms = 5000
        }
    }
}

E["CS_jh_jgsj03_npc"] = {
    on_parry_break = {
        {
            control_within_ms = 5000
        }
    }
}

E["CS_jh_jgsj04"] = {
    unparsed = { "对目标造成{@0:伤害包}【“神降·东君” ：招式射程提高至15米，且释放此招式(东君焚轮)不会" }
}

E["CS_jh_jgsj05"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_jgsj05_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_jgsj05_hide1"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_jgsj06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_jgsj07"] = {
    on_hit = {
        {
            heal = true
        }
    }
}

E["CS_jh_jgsj07_hide"] = {
    unparsed = { "下3次使用【九歌神降】套路招式，均可恢复一定量气血（升级层数可提高恢复气血的数值）" }
}

E["CS_jh_jgsj08"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "stun"
        },
        {
            condition = { unparried = true },
            control_dur_ms = 6000,
            control_kind = "launch"
        }
    }
}

E["CS_jh_jgsj08_npc"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_jgsj09"] = {
    unparsed = { "消耗10%最大气血以祭神，获得1层“神降”：使用【九歌神降】套路可消耗一层“神降”唤起对应神降效" }
}

E["CS_jh_jgsj09_hide"] = {
    unparsed = { "获得1层“神降”：使用【九歌神降】套路可消耗一层“神降”唤起对应神降效果使用此技能后可直接施展【" }
}

E["CS_jh_jhdf01"] = {
    on_hit = {
        {
            control_kind = "launch"
        },
        {
            apply_buff = "一层“碎魂",
            apply_life_ms = 15000,
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_jhdf02"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    },
    unparsed = { "（空中目标不受影响，冲击范围10米）由此招可见拉忽尔脱技艺之高超，令人拜服", "处于心魔，挖宝，禁地玩法时，招式影响范围扩大2米，招式释放时获得一层“怒斩”状态：使自身【惊鸿刀", "持续45秒最多可叠加1层" }
}

E["CS_jh_jhdf03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_dur_ms = 1000,
            control_kind = "coma"
        },
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    },
    by_level = {
        [3] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [4] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [5] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [6] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [7] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [8] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [13] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [14] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [15] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [16] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [17] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [18] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [19] = { ["on_hit.1.control_dur_ms"] = 3000 }
    }
}

E["CS_jh_jhdf04"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "毒源",
            apply_resolved = false,
            apply_stacks = 1
        },
        {
            apply_buff = "一层“夺魄",
            apply_life_ms = 15000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_jh_jhdf05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jhdf05",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_jhdf06"] = {
    unparsed = { "横眉冷对，刁钻反击，招架成功且自身存在“无双”效果时有20%几率反弹30%伤害" }
}

E["CS_jh_jhdf08"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    },
    unparsed = { "招式释放时若处于“碎魂”标记下，则消耗“碎魂”标记，使自身【刀刃风暴】每秒减少【春泥护花】调息时", "招式释放时若处于“夺魄”标记下，则消耗“夺魄”标记，使自身【刀刃风暴】每秒回复一次气血" }
}

E["CS_jh_jhsyd02"] = {
    unparsed = { "招式命中未招架的敌人可使其内功防御降低15%，持续7秒" }
}

E["CS_jh_jhsyd03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "若目标处于破防不可招架状态，则对其造成大量阳属性伤害" }
}

E["CS_jh_jhsyd04"] = {
    on_parry_break = {
        {
            control_kind = "root"
        }
    },
    unparsed = { "同时将“浮香”暗劲增至满层" }
}

E["CS_jh_jhsyd05"] = {
    unparsed = { "招架成功时，向攻击者传送一道“浮香”暗劲，可叠加5层：每次被镜花水月刀招式攻击，都会触发一道暗劲" }
}

E["CS_jh_jhsyd06"] = {
    on_hit = {
        {
            apply_buff = "浮香",
            apply_resolved = false,
            apply_stacks = 5
        }
    },
    unparsed = { "使目标身体摇摇欲坠：发动攻击或使用轻功都有很大几率跌倒，在空中则会直接坠落" }
}

E["CS_jh_jhsyd07"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_jsjf01"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    },
    unparsed = { "招式触发的黄霸体招式变更为手动释放（激活此效果需与巫墟宗派深处山鬼地窟的巫莎琪（77,-124）" }
}

E["CS_jh_jsjf01_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_jsjf01_hide_1"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_jsjf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_jsjf03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 10 }
    },
    on_hit = {
        {
            control_kind = "dizzy",
            displacement_distance_m = 0,
            displacement_kind = "push"
        },
        {
            condition = { unparried = true },
            control_kind = "dizzy",
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_jh_jsjf03_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 10 }
    }
}

E["CS_jh_jsjf03_hide_1"] = {
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "pull"
        },
        {
            condition = { unparried = true },
            control_kind = "dizzy",
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_jh_jsjf04"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_jh_jsjf04_01",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 2000,
            control_kind = "root",
            control_within_ms = 5000
        }
    },
    by_level = {
        [2] = { ["on_parry_break.1.control_within_ms"] = 6000 },
        [3] = { ["on_parry_break.1.control_within_ms"] = 7000 },
        [4] = { ["on_parry_break.1.control_within_ms"] = 8000 },
        [5] = { ["on_parry_break.1.control_within_ms"] = 9000 },
        [6] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [7] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [8] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [9] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [10] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [11] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [12] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [13] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [14] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [15] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [16] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [17] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [18] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [19] = { ["on_parry_break.1.control_within_ms"] = 10000 }
    }
}

E["CS_jh_jsjf04_hide"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_jh_jsjf04_01",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 2000,
            control_kind = "root",
            control_within_ms = 5000
        }
    },
    by_level = {
        [2] = { ["on_parry_break.1.control_within_ms"] = 6000 },
        [3] = { ["on_parry_break.1.control_within_ms"] = 7000 },
        [4] = { ["on_parry_break.1.control_within_ms"] = 8000 },
        [5] = { ["on_parry_break.1.control_within_ms"] = 9000 },
        [6] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [7] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [8] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [9] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [10] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [11] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [12] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [13] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [14] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [15] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [16] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [17] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [18] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [19] = { ["on_parry_break.1.control_within_ms"] = 10000 }
    }
}

E["CS_jh_jsjf05"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_dur_ms = 7000,
            control_kind = "launch"
        }
    }
}

E["CS_jh_jsjf05_hide"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_dur_ms = 7000,
            control_kind = "launch"
        }
    }
}

E["CS_jh_jsjf06"] = {
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "root"
        }
    },
    by_level = {
        [3] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [4] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [5] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [6] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [7] = { ["on_hit.1.control_dur_ms"] = 5000 },
        [8] = { ["on_hit.1.control_dur_ms"] = 5000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 5000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 5000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 5000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 5000 },
        [13] = { ["on_hit.1.control_dur_ms"] = 5000 },
        [14] = { ["on_hit.1.control_dur_ms"] = 5000 },
        [15] = { ["on_hit.1.control_dur_ms"] = 5000 },
        [16] = { ["on_hit.1.control_dur_ms"] = 5000 },
        [17] = { ["on_hit.1.control_dur_ms"] = 5000 },
        [18] = { ["on_hit.1.control_dur_ms"] = 5000 },
        [19] = { ["on_hit.1.control_dur_ms"] = 5000 }
    }
}

E["CS_jh_jsjf07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jsjf07",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_jsjf07_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "stun"
        }
    }
}

E["CS_jh_jsjf07_hide_1"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_jsz01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jsz08",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_jh_jsz02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_jsz03"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_jsz04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jsz08",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_jsz05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jsz05",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_jsz06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jsz06",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_jsz07"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jsz08",
            apply_resolved = true,
            apply_stacks = 3,
            control_dur_ms = 2000,
            control_kind = "root"
        }
    }
}

E["CS_jh_jsz08"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jsz08",
            apply_life_ms = 3000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 4000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 4000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 5000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 5000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 9000 }
    },
    unparsed = { "若敌方存在9层“九思”状态，则目标将倒地并移除“九思”状态" }
}

E["CS_jh_jsz09"] = {
    compile = {
        cast_armor = "yellow",
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jsz08",
            apply_resolved = true,
            apply_stacks = 1,
            per_hit = true
        }
    }
}

E["CS_jh_jwdf01"] = {
    unparsed = { "挥刀横斩，气劲对前方敌人造成{@0:伤害包}，同时为自身蓄存一层“炎阳刀气”", "（若目标身上“炎”气数量是3的倍数或已达上限，则对其造成无视招架的额外伤害）" }
}

E["CS_jh_jwdf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_jwdf03"] = {
    on_hit = {
        {
            apply_buff = "一股紊乱真气",
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_jwdf05"] = {
    unparsed = { "刀锋灼伤目标给其注入一股“炎”气，目标运气施招时，“炎”气数量有几率爆发增加", "（若目标身上“炎”气数量是3的倍数或已达上限，则对其造成无视招架的额外伤害）" }
}

E["CS_jh_jwdf07"] = {
    unparsed = { "夸父逐日，炎火冲天，将前方敌人抛至空中，以气御刀，多段斩击造成{@0:伤害包}", "每次击中敌人身体，会激活一层其体内潜藏的“炎”气并使之爆炸，攻击结束时，会引爆剩余的“炎”气" }
}

E["CS_jh_jwdf08"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparryable = true },
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_jybgz05"] = {
    unparsed = { "常年修炼九阴白骨爪的极阴招数,会导致修炼者皮肤冰冷坚硬如铁,减少所受伤害,攻击者都会受到这股寒毒" }
}

E["CS_jh_jybgz06"] = {
    unparsed = { "九阴白骨爪独特心法,通过用对方头骨来提升功力,每杀死一个敌人就会让功力提升一层,每层都会让九阴白" }
}

E["CS_jh_jybgz07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_jyjf01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    }
}

E["CS_jh_jyjf02"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_jyjf03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_jyjf04"] = {

}

E["CS_jh_jyjf05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jyjf05",
            apply_life_ms = 8000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_jyjf06"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 10 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_jyjf07"] = {
    compile = {
        air = "only"
    },
    unparsed = { "跳斩空中目标造成{@0:伤害包}" }
}

E["CS_jh_jymg01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "stiffen"
        }
    }
}

E["CS_jh_jymg02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_jymg03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jymg08",
            apply_resolved = true,
            apply_stacks = 3,
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_jymg04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "root"
        }
    }
}

E["CS_jh_jymg05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_jymg06"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jymg08",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 5000,
            control_kind = "dizzy"
        }
    },
    unparsed = { "（阴兵借道状态的级别，与架招【阴兵借道】功力有关）（反击对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_jh_jymg06_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_jh_jymg07"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jymg08",
            apply_resolved = true,
            apply_stacks = 5,
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_jymg08"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jymg08",
            apply_life_ms = 4000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [3] = { ["on_hit.1.apply_life_ms"] = 5000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 5000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [13] = { ["on_hit.1.apply_life_ms"] = 10000 }
    }
}

E["CS_jh_jysg01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    },
    on_hit = {
        {
            apply_buff = "一层日昃之离",
            apply_resolved = false,
            apply_stacks = 1,
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_jysg02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 10 }
    },
    on_parry_break = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_jysg03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_jysg04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_jysg05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 },
        can_move = false,
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_jh_jysg06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "1秒黄霸体状态",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_jysg07"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围20米，对霸体无效，可被虚招克制）" }
}

E["CS_jh_jysg07_hide"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围20米，对霸体无效，可被虚招克制）" }
}

E["CS_jh_jysg08"] = {
    unparsed = { "脚震天地，力踏八方，施放招式获得“无初有终”，九阳真气护体，自身免伤15%，外功招架与内功化解提" }
}

E["CS_jh_jysg09"] = {
    compile = {
        cast_armor = "red"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 10000,
            control_kind = "launch"
        }
    },
    unparsed = { "出招时获得九阳真力，提升15%外功伤害减免忽视、15%内功伤害减免忽视、15%外功招架忽视、15" }
}

E["CS_jh_jysg09_hide"] = {
    compile = {
        cast_armor = "red"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_jysg09_npc"] = {
    compile = {
        cast_armor = "red"
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_jzbf01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jzbf01",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            per_hit = true
        }
    },
    by_level = {
        [3] = { ["on_hit.1.apply_stacks"] = 2 },
        [4] = { ["on_hit.1.apply_stacks"] = 2 },
        [5] = { ["on_hit.1.apply_stacks"] = 3 },
        [6] = { ["on_hit.1.apply_stacks"] = 3 },
        [7] = { ["on_hit.1.apply_stacks"] = 4 },
        [8] = { ["on_hit.1.apply_stacks"] = 4 },
        [9] = { ["on_hit.1.apply_stacks"] = 4 },
        [10] = { ["on_hit.1.apply_stacks"] = 4 },
        [11] = { ["on_hit.1.apply_stacks"] = 4 },
        [12] = { ["on_hit.1.apply_stacks"] = 4 }
    }
}

E["CS_jh_jzbf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "对存在“尖”状态的目标，可造成大量伤害（极大几率暴击），之后清除其尖状态" }
}

E["CS_jh_jzbf03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "若目标存在“尖”状态，招架本招式则会被震伤，受到额外伤害" }
}

E["CS_jh_jzbf04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jzbf04",
            apply_life_ms = 7000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 2000,
            control_kind = "silence"
        }
    },
    unparsed = { "（招式功力每提升1层，冷却时间减少1秒）" }
}

E["CS_jh_jzbf05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_jzbf05",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    self_state = {
        {
            apply_buff = "buf_CS_jh_jzbf05_1",
            apply_life_ms = 10000,
            apply_resolved = true
        }
    }
}

E["CS_jh_jzbf06"] = {

}

E["CS_jh_jzbf07"] = {
    on_hit = {
        {
            control_dur_ms = 4000,
            control_kind = "root"
        }
    },
    self_state = {
        {
            apply_buff = "buf_CS_jh_jzbf07",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制" }
}

E["CS_jh_jzbf08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            control_kind = "stun"
        },
        {
            control_dur_ms = 3000,
            control_kind = "stun"
        }
    },
    by_level = {
        [3] = { ["on_hit.1.control_dur_ms"] = 1000 },
        [4] = { ["on_hit.1.control_dur_ms"] = 1000 },
        [5] = { ["on_hit.1.control_dur_ms"] = 1000 },
        [6] = { ["on_hit.1.control_dur_ms"] = 1000 },
        [7] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [8] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 2000 }
    },
    unparsed = { "气场对范围内敌人造成持续伤害（距离气场中心越近，所受伤害越高）" }
}

E["CS_jh_kfdf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_kfdf03"] = {
    compile = {
        max_hits = 4
    }
}

E["CS_jh_kfdf05"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_kfdf05",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            control_dur_ms = 10000,
            control_kind = "root"
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 11000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 12000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 13000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 14000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 15000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 16000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 17000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 18000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 19000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 20000 }
    }
}

E["CS_jh_kfdf06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_kfdf06",
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.2
        }
    },
    by_level = {
        [2] = { ["on_hit.1.chance"] = 0.3 },
        [3] = { ["on_hit.1.chance"] = 0.4 },
        [4] = { ["on_hit.1.chance"] = 0.5 },
        [5] = { ["on_hit.1.chance"] = 0.6 },
        [6] = { ["on_hit.1.chance"] = 0.7 },
        [7] = { ["on_hit.1.chance"] = 0.8 },
        [8] = { ["on_hit.1.chance"] = 0.9 }
    }
}

E["CS_jh_kfdf07"] = {

}

E["CS_jh_klsjj01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            control_within_ms = 5000
        },
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_klsjj02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_klsjj04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_klsjj04_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_klsjj05"] = {
    self_state = {
        {
            apply_buff = "buf_CS_jh_klsjj05_01",
            apply_resolved = true
        }
    }
}

E["CS_jh_klsjj06"] = {
    compile = {
        air = "only"
    }
}

E["CS_jh_klsjj06_hide"] = {
    unparsed = { "目标在地面时，身形虚晃，腾挪闪移至目标身边，对目标造成{@0:伤害包}并使未招架的目标倒地目标在" }
}

E["CS_jh_klsjj06_hide1"] = {
    compile = {
        air = "only"
    }
}

E["CS_jh_klsjj07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_dur_ms = 8000,
            control_kind = "ban_qinggong"
        },
        {
            control_dur_ms = 8000,
            control_kind = "knockdown"
        },
        {
            condition = { unparried = true },
            control_dur_ms = 8000,
            control_kind = "launch"
        }
    }
}

E["CS_jh_klsjj07_sky"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 8000,
            control_kind = "launch"
        }
    }
}

E["CS_jh_klsjj08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    },
    unparsed = { "出招时，消耗额外怒气使自身获得“归阳”状态：提升本套路招式伤害，状态持续10秒", "人归阳：提升本套路5%的招式伤害，额外消耗5点怒气", "地归阳：提升本套路10%的招式伤害，额外消耗10点怒气", "天归阳：提升本套路15%的招式伤害，额外消耗15点怒气" }
}

E["CS_jh_klsjj08_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    },
    unparsed = { "出招时，消耗额外怒气使自身获得“归阳”状态：提升本套路招式伤害，状态持续10秒", "人归阳：提升本套路5%的招式伤害，额外消耗5点怒气", "地归阳：提升本套路10%的招式伤害，额外消耗10点怒气", "天归阳：提升本套路15%的招式伤害，额外消耗15点怒气" }
}

E["CS_jh_kspzj01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_kspzj02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    },
    unparsed = { "震地：在地面施展该招式时，可将暗劲渡入身前地面，一段时间后暗劲爆发伤敌：破除敌人招架时使其摔倒" }
}

E["CS_jh_kspzj02_hide"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    }
}

E["CS_jh_kspzj03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 10 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_kspzj04"] = {
    compile = {
        cast_armor = "yellow"
    },
    self_state = {
        {
            apply_buff = "buf_CS_jh_kspzj04_01",
            apply_resolved = true
        }
    },
    unparsed = { "（通过提升招式层数可削减招式回气时间）" }
}

E["CS_jh_kspzj05"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "root",
            displacement_distance_m = 0,
            displacement_kind = "push"
        },
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_kspzj06"] = {
    self_state = {
        {
            apply_buff = "buf_CS_jh_kspzj06_01",
            apply_resolved = true
        }
    }
}

E["CS_jh_kspzj07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown",
            per_hit = true
        }
    }
}

E["CS_jh_kxtd02_1"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "launch"
        }
    }
}

E["CS_jh_kxtd02_2"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_kxtd02_3"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_kxtd03"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    unparsed = { "对倒地目标，可造成额外伤害" }
}

E["CS_jh_kxtd04_1"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_kxtd05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_kxtd_05",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_kxtd06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_kxtd_05",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    self_state = {
        {
            apply_buff = "buf_CS_jh_kxtd_06_1",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_jh_kxtd06_hide"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_kxtd_05",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_kxtd07"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_kxtd_05",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    on_parry_break = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_lbqx01"] = {
    unparsed = { "对目标造成{@0:伤害包}，出招时，移除所有的“明察”状态，每移除1层“明察”状态，可使【沉浮】" }
}

E["CS_jh_lbqx02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_lbqx02",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "（“沉浮”、“未艾”、“醉局”状态总共可叠加3层，随机替换其他状态并且优先替换层数较少的状态）" }
}

E["CS_jh_lbqx03"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_lbqx03",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "（“沉浮”、“未艾”、“醉局”状态总共可叠加3层，随机替换其他状态并且优先替换层数较少的状态）" }
}

E["CS_jh_lbqx04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_lbqx04",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 5000,
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    unparsed = { "（“沉浮”、“未艾”、“醉局”状态总共可叠加3层，随机替换其他状态并且优先替换层数较少的状态）" }
}

E["CS_jh_lbqx05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 10 }
    },
    on_hit = {
        {
            control_dur_ms = 5000,
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_lbqx06"] = {
    unparsed = { "若自身存在2层“沉浮”，则使造成的暴击伤害提升15%", "若自身存在3层“沉浮”，则使造成的暴击伤害提升30%", "若自身存在2层“未艾”，则自身回复少许气血", "若自身存在3层“未艾”，则自身回复少量气血和少许内力", "若自身存在2层“醉局”，则自身5秒内移动速度提升30%", "若自身存在3层“醉局”，则自身5秒内移动速度提升50%且免疫移动限制效果（加速效果15秒内仅能生" }
}

E["CS_jh_lbqx07"] = {
    unparsed = { "对目标造成{@0:伤害包}，破防成功时，使其缴械3秒，同时自身获得3层“明察”" }
}

E["CS_jh_lbqx08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 10 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_lbqx08",
            apply_life_ms = 15000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_lbwb01"] = {
    unparsed = { "神行百变-无敌斩" }
}

E["CS_jh_lbwb02"] = {
    on_parry_break = {
        {
            control_dur_ms = 10000,
            control_kind = "launch",
            heal = true
        }
    }
}

E["CS_jh_ldg01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_ldg02"] = {
    on_hit = {
        {
            apply_buff = "一层“弓鸣",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "push",
            per_hit = true
        }
    }
}

E["CS_jh_ldg02_hide"] = {
    on_hit = {
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_ldg03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "push"
        },
        {
            condition = { unparryable = true },
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    unparsed = { "蓄力时间超过10%时，提升10%的伤害以及2米的射程", "蓄力时间超过50%时，提升20%的伤害以及4米的射程" }
}

E["CS_jh_ldg04"] = {
    on_hit = {
        {
            apply_buff = "破月状态",
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 2000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_ldg05"] = {
    unparsed = { "命中未招架目标使其硬直" }
}

E["CS_jh_ldg06"] = {
    on_parry = {
        {
            apply_buff = "惊弓状态",
            apply_life_ms = 15000,
            apply_resolved = false,
            apply_stacks = 1,
            chance = 0.3
        }
    },
    by_level = {
        [10] = { ["on_parry.1.apply_life_ms"] = 20000 },
        [11] = { ["on_parry.1.apply_life_ms"] = 25000 },
        [12] = { ["on_parry.1.apply_life_ms"] = 30000 },
        [13] = { ["on_parry.1.apply_life_ms"] = 30000 }
    }
}

E["CS_jh_ldg07"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_lhbwq01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_lhbwq01",
            apply_life_ms = 30000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_lhbwq02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "若自身存在10层扑食状态，则获得“化龙”状态：【龙游八荒】变招为【风虎云龙】，状态持续15秒" }
}

E["CS_jh_lhbwq02_speed"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "若自身存在10层扑食状态，则获得“化龙”状态：【龙游八荒】变招为【风虎云龙】，状态持续15秒" }
}

E["CS_jh_lhbwq03"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    unparsed = { "（招式【风虎云龙】威力更大，且不消耗内力）" }
}

E["CS_jh_lhbwq04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_dur_ms = 5000,
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_lhbwq05"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_lhbwq05",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 11000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 12000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 13000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 14000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 15000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 16000 }
    }
}

E["CS_jh_lhbwq06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_lhbwq0601",
            apply_life_ms = 15000,
            apply_resolved = true,
            apply_stacks = 1,
            per_hit = true
        }
    }
}

E["CS_jh_lhbwq07"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_jh_lhbwq07",
            apply_life_ms = 8000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_lhbwq08"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_lhbwq08",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "（招式连绵无绝，且不消耗内力）" }
}

E["CS_jh_liusha01"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 12000,
            control_kind = "stiffen"
        }
    },
    unparsed = { "甩出袖中折扇，以气御扇伤其身前目标造成{@0:伤害包}，有一定几率击中目标要穴使之一段时间内无法" }
}

E["CS_jh_liusha01_hide"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "stiffen"
        }
    }
}

E["CS_jh_liusha04"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_liusha04_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_liusha05"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_liusha05_hide"] = {
    unparsed = { "力灌剑身，猛插入地面，激起两道越发强大的气波，被第一道气波震伤的目标短时间内无法施展轻功，第二道" }
}

E["CS_jh_liusha06"] = {
    unparsed = { "凝聚内力护住全身，增强防御，每成功招架目标攻击都会令防御更加坚不可摧" }
}

E["CS_jh_liusha07"] = {

}

E["CS_jh_liusha08"] = {
    unparsed = { "狠狠的将目标踩进土里，持续5秒，每秒造成当前生命值5%的伤害（最多不高于138点伤害，可以受伤害" }
}

E["CS_jh_liusha09"] = {
    unparsed = { "平地刮起一阵龙卷风，对直线上的敌方造成{@0:伤害包}，将陷土里的目标卷至空中" }
}

E["CS_jh_lldf01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "一层“刺风",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_jh_lldf01_hide"] = {
    on_hit = {
        {
            apply_buff = "一层“束风",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    },
    unparsed = { "存在2层及以上“束风”状态，额外降低25%移速", "存在3层“束风”状态，额外降低20%伤害", "每次出招将消耗一层“山截海断”状态" }
}

E["CS_jh_lldf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "root"
        }
    }
}

E["CS_jh_lldf03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            apply_buff = "buf_CS_jh_lldf03",
            apply_life_ms = 3000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_lldf03_hide"] = {
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "coma"
        }
    },
    on_parry_break = {
        {
            control_dur_ms = 2000,
            control_kind = "root"
        }
    },
    unparsed = { "每次出招将消耗一层“山截海断”状态" }
}

E["CS_jh_lldf04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 10 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_lldf05"] = {
    unparsed = { "举头仰问天，天色但苍苍，招架成功自身获得“老树苍烟”状态：减少10%受到的伤害，持续5秒，可叠加", "受到攻击，若处于未招架状态，则消除一层“老树苍烟”状态" }
}

E["CS_jh_lldf06"] = {
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    },
    unparsed = { "万里卷潮来，无情送潮归，暴起全身的真气将刀震碎，自身激发出5层“山截海断”状态：每层状态使移速提" }
}

E["CS_jh_lldf07"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_lldf08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 },
        cast_armor = "yellow"
    }
}

E["CS_jh_lldf08_hide"] = {
    compile = {
        cast_armor = "red"
    },
    unparsed = { "出招时消耗所有的“山截海断”状态，至少存在一层“山截海断”状态才能出招，额外消耗的“山截海断”状" }
}

E["CS_jh_llt01"] = {
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "root"
        }
    },
    by_level = {
        [5] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [6] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [7] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [8] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 5000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 5000 }
    }
}

E["CS_jh_llt02"] = {
    on_parry_break = {
        {
            control_dur_ms = 4000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_llt03"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    },
    by_level = {
        [4] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [5] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [6] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [7] = { ["on_hit.1.control_dur_ms"] = 4000, ["on_hit.2.control_dur_ms"] = 4000 },
        [8] = { ["on_hit.1.control_dur_ms"] = 4000, ["on_hit.2.control_dur_ms"] = 4000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 5000, ["on_hit.2.control_dur_ms"] = 5000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 5000, ["on_hit.2.control_dur_ms"] = 5000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 5000, ["on_hit.2.control_dur_ms"] = 5000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 5000, ["on_hit.2.control_dur_ms"] = 5000 }
    }
}

E["CS_jh_llt05"] = {
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_llt06"] = {
    unparsed = { "自身处在掷点状态时：每击附加硬直效果（无视目标招架）、每击附带溅射100%伤害效果" }
}

E["CS_jh_lsbm01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_lsbm02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    }
}

E["CS_jh_lsbm03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    }
}

E["CS_jh_lsbm03_hide"] = {
    unparsed = { "挥击而出，对目标造成{@0:伤害包}，破防成功则使其外功防御降低30%，持续5秒" }
}

E["CS_jh_lsbm04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_lsbm05"] = {
    on_hit = {
        {
            apply_buff = "一层藏刀",
            apply_life_ms = 15000,
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_lsbm06"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "stiffen",
            per_hit = true
        }
    }
}

E["CS_jh_lsbm07"] = {
    unparsed = { "吐纳之法，提高本套路招式伤害10%，使招式【瞒天过海】变化：飞出的匕首更多，暴击率提高5%，但射" }
}

E["CS_jh_lybj01"] = {
    unparsed = { "双箭连环，对选中敌人造成{@0:伤害包}，可获得1层震地气劲" }
}

E["CS_jh_lybj03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 25 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_jh_lybj05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    unparsed = { "自身进入乱雾遮天状态：成功招架敌人攻击时，可获得震地气劲，持续24秒，最多可获得5层" }
}

E["CS_jh_lybj05_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_lybj06"] = {
    on_hit = {
        {
            apply_buff = "标记效果",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "coma"
        },
        {
            apply_buff = "等同于第一个目标的效果",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_lybj07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_lybj08"] = {
    unparsed = { "以飞花闪影般身法连续射箭，对选中敌人造成{@0:伤害包}，可获得2层震地气劲" }
}

E["CS_jh_lydgd01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_lydgd02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    unparsed = { "招式命中未招架的敌人时，自身可获得1层“蕴影”状态" }
}

E["CS_jh_lydgd03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "ban_qinggong"
        }
    },
    by_level = {
        [7] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [8] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 5000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 5000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 5000 },
        [13] = { ["on_hit.1.control_dur_ms"] = 5000 }
    }
}

E["CS_jh_lydgd04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "silence"
        }
    }
}

E["CS_jh_lydgd05"] = {
    by_level = {
        [4] = { ["on_hit.1.control_dur_ms"] = 1000 },
        [5] = { ["on_hit.1.control_dur_ms"] = 1000 },
        [6] = { ["on_hit.1.control_dur_ms"] = 1000 },
        [7] = { ["on_hit.1.control_dur_ms"] = 1000 },
        [8] = { ["on_hit.1.control_dur_ms"] = 1000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 1000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [13] = { ["on_hit.1.control_dur_ms"] = 2000 }
    },
    unparsed = { "撤步横架，自身进入韬光蕴影状态：成功招架敌人攻击时，自身获得1层“蕴影”状态，持续12秒，可叠加", "蕴影：自身在未招架的情况下受到攻击时，“蕴影”状态会转移至攻击者身上，叠加至6层时会对敌人造成外" }
}

E["CS_jh_lydgd06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        air = "never"
    },
    on_hit = {
        {
            control_dur_ms = 10000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_lydgd07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_lyjf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_lyjf02",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_lyjf03"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_lyjf03",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_jh_lyjf04"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    },
    unparsed = { "擎剑敛气，激射而出，对点选区域内敌人造成{@0:伤害包},射程20米" }
}

E["CS_jh_lyjf05"] = {
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_lyjf06"] = {
    unparsed = { "流云心诀，反守为攻，招架成功时，向敌方目标释放流云剑气：剑气弹射，对剑气影响的目标造成伤害及30" }
}

E["CS_jh_lyjf07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_lzxss01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_lzxss02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    }
}

E["CS_jh_lzxss03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 }
    }
}

E["CS_jh_lzxss04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "coma"
        }
    }
}

E["CS_jh_lzxss05"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "自身进入茹毛饮血状态，每次成功招架可获得1层血沸状态，持续8秒，可叠加5层", "自身存在血沸状态时，该招式可主动施展，扑向敌人，咬颈残杀，造成阴伤害，且每击附加10%的阴伤害（" }
}

E["CS_jh_lzxss05_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "扑向敌人，咬颈残杀，对选中目标造成{@0:伤害包}，并每击附加10%的阴伤害（无视防御，自身血沸" }
}

E["CS_jh_lzxss06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "coma"
        }
    }
}

E["CS_jh_lzxss07"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    }
}

E["CS_jh_mhz01"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_mhz02"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_mhz03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_mhz05"] = {
    unparsed = { "并使地面敌人僵直2秒，击落未招架的空中敌人" }
}

E["CS_jh_mhz06"] = {
    self_state = {
        {
            apply_buff = "buf_CS_jh_mhz06",
            apply_life_ms = 20000,
            apply_resolved = true
        }
    }
}

E["CS_jh_mhz07"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    },
    unparsed = { "交叉挥砍，向前方冲出8米，对范围内敌人造成{@0:伤害包}" }
}

E["CS_jh_mjh01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_mjh02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    unparsed = { "成功破除敌人招架时，使其“失势”：3秒内无法招架，且受到招式【破风】攻击时僵直" }
}

E["CS_jh_mjh03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "stiffen"
        },
        {
            control_kind = "stiffen"
        }
    },
    unparsed = { "破风：命中无法招架的目标时造成1.1倍伤害" }
}

E["CS_jh_mjh03_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "stiffen"
        }
    }
}

E["CS_jh_mjh04"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "coma"
        }
    }
}

E["CS_jh_mjh05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "黄"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_mjh06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "stiffen",
            per_hit = true
        }
    },
    unparsed = { "并可额外消耗怒气提升招式伤害：每额外消耗5点怒气，提高5%招式伤害" }
}

E["CS_jh_mjh07"] = {
    self_state = {
        {
            apply_buff = "buf_CS_jh_mjh07_01",
            apply_life_ms = 15000,
            apply_resolved = true
        }
    },
    by_level = {
        [6] = { ["self_state.1.apply_life_ms"] = 18000 },
        [7] = { ["self_state.1.apply_life_ms"] = 18000 },
        [8] = { ["self_state.1.apply_life_ms"] = 20000 },
        [9] = { ["self_state.1.apply_life_ms"] = 20000 },
        [10] = { ["self_state.1.apply_life_ms"] = 22000 },
        [11] = { ["self_state.1.apply_life_ms"] = 22000 },
        [12] = { ["self_state.1.apply_life_ms"] = 24000 },
        [13] = { ["self_state.1.apply_life_ms"] = 24000 }
    },
    unparsed = { "招式【追影】在足够的“噬气凝影”气劲支持下，可凝聚出刀影，提升【噬气】层数可提升刀影伤害" }
}

E["CS_jh_mjh08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "三层当前【伏晖】招式等级对应的【",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_jh_mlyqj01_f"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_mlyqj01_m"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_mlyqj02_f"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_mlyqj02_f",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            control_dur_ms = 1000,
            control_kind = "root"
        }
    },
    by_level = {
        [3] = { ["on_hit.2.control_dur_ms"] = 2000 },
        [4] = { ["on_hit.2.control_dur_ms"] = 2000 },
        [5] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [6] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [7] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [8] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [9] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [10] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [11] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [12] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [13] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [14] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [15] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [16] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [17] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [18] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [19] = { ["on_hit.2.control_dur_ms"] = 3000 }
    }
}

E["CS_jh_mlyqj02_m"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_mlyqj02_m",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            control_dur_ms = 1000,
            control_kind = "root"
        }
    },
    by_level = {
        [3] = { ["on_hit.2.control_dur_ms"] = 2000 },
        [4] = { ["on_hit.2.control_dur_ms"] = 2000 },
        [5] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [6] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [7] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [8] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [9] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [10] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [11] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [12] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [13] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [14] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [15] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [16] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [17] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [18] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [19] = { ["on_hit.2.control_dur_ms"] = 3000 }
    }
}

E["CS_jh_mlyqj03_f"] = {
    on_parry_break = {
        {
            apply_buff = "琴瑟调和状态",
            apply_life_ms = 6000,
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_mlyqj03_m"] = {
    on_parry_break = {
        {
            apply_buff = "琴瑟调和状态",
            apply_life_ms = 6000,
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_mlyqj04_f"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            apply_buff = "流水(女)状态",
            apply_life_ms = 15000,
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_mlyqj04_m"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            apply_buff = "流水(男)状态",
            apply_life_ms = 15000,
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_mlyqj05_m"] = {
    on_hit = {
        {
            apply_buff = "护花郎状态",
            apply_life_ms = 15000,
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 18000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 21000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 24000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 27000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 30000 }
    }
}

E["CS_jh_mlyqj06_f"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_mlyqj06_m"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "silence"
        }
    }
}

E["CS_jh_mlyqj07_f"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 },
        cast_armor = "red"
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_mlyqj07_m"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 },
        cast_armor = "red"
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_mlyqj08_f"] = {
    unparsed = { "招式可击落空中目标" }
}

E["CS_jh_mlyqj08_m"] = {
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_mmqfj01"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    },
    unparsed = { "（招式有一定几率造成毁容，将目标打成风华绝代）" }
}

E["CS_jh_mmqfj01_sky"] = {
    compile = {
        cast_armor = "yellow"
    }
}

E["CS_jh_mmqfj02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_mmqfj02_sky"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_mmqfj03"] = {
    on_parry_break = {
        {
            control_kind = "knockdown"
        }
    },
    unparsed = { "（招式有一定几率造成毁容，将目标打成风华绝代）" }
}

E["CS_jh_mmqfj03_sky"] = {
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_mmqfj04"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_mmqfj04_sky"] = {
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_mmqfj05"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        can_move = false,
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "风华绝代",
            apply_life_ms = 30000,
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 30000,
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_mmqfj05_sky"] = {
    unparsed = { "AOE，黄霸体，无数个脚影践踏地面：眼部变为熊猫眼，脸上有个红色脚印" }
}

E["CS_jh_mmqfj06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_mmqfj06",
            apply_life_ms = 4000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    },
    unparsed = { "（招式有一定几率造成毁容，将目标打成惨惨惨）" }
}

E["CS_jh_mmqfj06_sky"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_mmqfj06",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_mmqfj07"] = {
    on_hit = {
        {
            apply_buff = "惨惨惨",
            apply_life_ms = 30000,
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_mmqfj07_sky"] = {
    on_hit = {
        {
            apply_buff = "惨惨惨",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_mqc01"] = {
    on_hit = {
        {
            apply_buff = "‘花娇禁’状态",
            apply_life_ms = 10000,
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "‘花娇禁’状态叠加至3层后，可使招式变幻为【掌情绕】" }
}

E["CS_jh_mqc01_hide"] = {
    on_hit = {
        {
            apply_buff = "‘情绕’效果",
            apply_life_ms = 7000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        },
        {
            control_dur_ms = 1000,
            control_kind = "silence"
        }
    }
}

E["CS_jh_mqc02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_mqc03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_mqc04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_mqc05"] = {
    on_hit = {
        {
            apply_buff = "‘凝心慌’状态",
            apply_life_ms = 10000,
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 10000,
            control_kind = "launch"
        }
    }
}

E["CS_jh_mqc06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_mqc07"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "silence",
            per_hit = true
        }
    },
    unparsed = { "格挡成功3次可获得一层榴花妖状态：使用套路下【掌情绕】【鬓丝磨】【魑魅博】时附带额外效果", "最多可叠加1层" }
}

E["CS_jh_mqc08"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "silence"
        },
        {
            control_dur_ms = 2000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_myjf01"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "ban_qinggong"
        },
        {
            apply_buff = "一层“疏影",
            apply_resolved = false,
            apply_stacks = 1,
            control_kind = "ban_qinggong"
        }
    },
    by_level = {
        [3] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [4] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [5] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [6] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [7] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [8] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [13] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [14] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [15] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [16] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [17] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [18] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [19] = { ["on_hit.1.control_dur_ms"] = 4000 }
    },
    unparsed = { "该状态可叠加5层，持续2秒，每层减速14%" }
}

E["CS_jh_myjf02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            condition = { unparryable = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_myjf04"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_myjf05"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_parry_break = {
        {
            apply_buff = "buf_CS_jh_myjf05_01",
            apply_life_ms = 4000,
            apply_resolved = true,
            apply_stacks = 4,
            control_dur_ms = 4000,
            control_kind = "ban_qinggong"
        }
    },
    by_level = {
        [2] = { ["on_parry_break.1.apply_stacks"] = 5 },
        [3] = { ["on_parry_break.1.apply_life_ms"] = 5000, ["on_parry_break.1.apply_stacks"] = 5, ["on_parry_break.1.control_dur_ms"] = 5000 },
        [4] = { ["on_parry_break.1.apply_life_ms"] = 5000, ["on_parry_break.1.apply_stacks"] = 6, ["on_parry_break.1.control_dur_ms"] = 5000 },
        [5] = { ["on_parry_break.1.apply_life_ms"] = 6000, ["on_parry_break.1.apply_stacks"] = 6, ["on_parry_break.1.control_dur_ms"] = 6000 },
        [6] = { ["on_parry_break.1.apply_life_ms"] = 7000, ["on_parry_break.1.apply_stacks"] = 7, ["on_parry_break.1.control_dur_ms"] = 7000 },
        [7] = { ["on_parry_break.1.apply_life_ms"] = 7000, ["on_parry_break.1.apply_stacks"] = 7, ["on_parry_break.1.control_dur_ms"] = 7000 },
        [8] = { ["on_parry_break.1.apply_life_ms"] = 7000, ["on_parry_break.1.apply_stacks"] = 7, ["on_parry_break.1.control_dur_ms"] = 7000 },
        [9] = { ["on_parry_break.1.apply_life_ms"] = 7000, ["on_parry_break.1.apply_stacks"] = 7, ["on_parry_break.1.control_dur_ms"] = 7000 },
        [10] = { ["on_parry_break.1.apply_life_ms"] = 7000, ["on_parry_break.1.apply_stacks"] = 7, ["on_parry_break.1.control_dur_ms"] = 7000 },
        [11] = { ["on_parry_break.1.apply_life_ms"] = 7000, ["on_parry_break.1.apply_stacks"] = 7, ["on_parry_break.1.control_dur_ms"] = 7000 },
        [12] = { ["on_parry_break.1.apply_life_ms"] = 7000, ["on_parry_break.1.apply_stacks"] = 7, ["on_parry_break.1.control_dur_ms"] = 7000 },
        [13] = { ["on_parry_break.1.apply_life_ms"] = 7000, ["on_parry_break.1.apply_stacks"] = 7, ["on_parry_break.1.control_dur_ms"] = 7000 },
        [14] = { ["on_parry_break.1.apply_life_ms"] = 7000, ["on_parry_break.1.apply_stacks"] = 7, ["on_parry_break.1.control_dur_ms"] = 7000 },
        [15] = { ["on_parry_break.1.apply_life_ms"] = 7000, ["on_parry_break.1.apply_stacks"] = 7, ["on_parry_break.1.control_dur_ms"] = 7000 },
        [16] = { ["on_parry_break.1.apply_life_ms"] = 7000, ["on_parry_break.1.apply_stacks"] = 7, ["on_parry_break.1.control_dur_ms"] = 7000 },
        [17] = { ["on_parry_break.1.apply_life_ms"] = 7000, ["on_parry_break.1.apply_stacks"] = 7, ["on_parry_break.1.control_dur_ms"] = 7000 },
        [18] = { ["on_parry_break.1.apply_life_ms"] = 7000, ["on_parry_break.1.apply_stacks"] = 7, ["on_parry_break.1.control_dur_ms"] = 7000 },
        [19] = { ["on_parry_break.1.apply_life_ms"] = 7000, ["on_parry_break.1.apply_stacks"] = 7, ["on_parry_break.1.control_dur_ms"] = 7000 }
    }
}

E["CS_jh_myjf05_npc"] = {
    unparsed = { "可令魅影分身同步使用该招式，破防成功，目标在一段时间移动速度大幅下降，速度慢慢恢复，期间禁止使用" }
}

E["CS_jh_myjf06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_myjf06_03",
            apply_life_ms = 4000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [3] = { ["on_hit.1.apply_life_ms"] = 5000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 5000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 7000 }
    },
    unparsed = { "（自身脱离招架状态时，“灭影”状态立刻消失）（使用其他套路招式，则清除“韬形灭影”招架状态）" }
}

E["CS_jh_myjf07"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_myjf02",
            apply_life_ms = 2000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 2000,
            control_kind = "ban_qinggong",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    by_level = {
        [3] = { ["on_hit.1.apply_life_ms"] = 3000, ["on_hit.1.control_dur_ms"] = 3000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 3000, ["on_hit.1.control_dur_ms"] = 3000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 4000, ["on_hit.1.control_dur_ms"] = 4000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 4000, ["on_hit.1.control_dur_ms"] = 4000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 4000, ["on_hit.1.control_dur_ms"] = 4000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 4000, ["on_hit.1.control_dur_ms"] = 4000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 4000, ["on_hit.1.control_dur_ms"] = 4000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 4000, ["on_hit.1.control_dur_ms"] = 4000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 4000, ["on_hit.1.control_dur_ms"] = 4000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 4000, ["on_hit.1.control_dur_ms"] = 4000 },
        [13] = { ["on_hit.1.apply_life_ms"] = 4000, ["on_hit.1.control_dur_ms"] = 4000 },
        [14] = { ["on_hit.1.apply_life_ms"] = 4000, ["on_hit.1.control_dur_ms"] = 4000 },
        [15] = { ["on_hit.1.apply_life_ms"] = 4000, ["on_hit.1.control_dur_ms"] = 4000 },
        [16] = { ["on_hit.1.apply_life_ms"] = 4000, ["on_hit.1.control_dur_ms"] = 4000 },
        [17] = { ["on_hit.1.apply_life_ms"] = 4000, ["on_hit.1.control_dur_ms"] = 4000 },
        [18] = { ["on_hit.1.apply_life_ms"] = 4000, ["on_hit.1.control_dur_ms"] = 4000 },
        [19] = { ["on_hit.1.apply_life_ms"] = 4000, ["on_hit.1.control_dur_ms"] = 4000 }
    }
}

E["CS_jh_myjf08"] = {
    compile = {
        max_hits = 5
    }
}

E["CS_jh_mzjf01"] = {
    unparsed = { "攻击时额外获得1点怒气（敌未招架）" }
}

E["CS_jh_mzjf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "亦守亦攻：自身招架耐力大于75%时，攻击忽视目标5%外功伤害减免" }
}

E["CS_jh_mzjf03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "亦守亦攻：自身招架耐力大于75%时，攻击忽视目标5%外功伤害减免" }
}

E["CS_jh_mzjf04"] = {
    compile = {
        cast_armor = "yellow"
    },
    unparsed = { "截剑盈缩，怒极无为，化解怒气以恢复自身2000点招架耐力，并清除不可招架负面状态", "自身同时获得“剑气江湖”状态：施展墨子剑法招式击中目标时，每击附加当前招架值3%的伤害(最多附加" }
}

E["CS_jh_mzjf05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_mzjf05_01",
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            chance = 0.2,
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    },
    unparsed = { "（目标每施展一次招式，移除一层“非攻”状态）" }
}

E["CS_jh_mzjf06"] = {

}

E["CS_jh_mzjf07"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_mzjf07_02",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_mzjf08"] = {
    on_hit = {
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    unparsed = { "亦守亦攻：自身招架耐力大于75%时，攻击忽视目标5%外功伤害减免", "以攻养攻：若自身当前招架耐力低于30%时，施展招式完毕，可恢复20点怒气" }
}

E["CS_jh_mzmyc01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_mzmyc05",
            apply_resolved = true,
            apply_stacks = 3
        },
        {
            apply_buff = "此状态）",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "三十三层云天外，无上天魔俯身斩" }
}

E["CS_jh_mzmyc02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_mzmyc02_npc"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_mzmyc03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "knockdown"
        }
    },
    by_level = {
        [11] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 5000 }
    }
}

E["CS_jh_mzmyc04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_mzmyc05",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        },
        {
            apply_buff = "此状态）",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_mzmyc05"] = {
    on_hit = {
        {
            apply_buff = "一层“枯骨",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "每层减少目标4%移动速度，持续7秒，状态可叠加3层" }
}

E["CS_jh_mzmyc06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 },
        cast_armor = "yellow",
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "stun",
            per_hit = true
        }
    }
}

E["CS_jh_mzmyc07"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_ncd01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_ncd01_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_ncd02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_ncd02_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_ncd03"] = {
    compile = {
        cast_armor = "yellow"
    }
}

E["CS_jh_ncd03_hide"] = {
    compile = {
        cast_armor = "yellow"
    }
}

E["CS_jh_ncd04"] = {

}

E["CS_jh_ncd04_hide"] = {

}

E["CS_jh_ncd05"] = {
    self_state = {
        {
            apply_buff = "buf_CS_jh_ncd05_01",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_jh_ncd05_hide_01"] = {
    self_state = {
        {
            apply_buff = "以守为攻",
            apply_life_ms = 5000,
            apply_resolved = false
        }
    },
    unparsed = { "（防反招式，对霸体无效，可被虚招克制）" }
}

E["CS_jh_ncd06"] = {
    compile = {
        cast_armor = "red"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_ncd06",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_ncd06_hide"] = {
    compile = {
        cast_armor = "red"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_ncd06_02",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_ncd07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    unparsed = { "招式为【流纨素】时：招式可无视目标招架，对其造成伤害" }
}

E["CS_jh_pjjt01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_pjjt01",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_pjjt02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_pjjt07",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_pjjt03"] = {
    on_parry_break = {
        {
            control_dur_ms = 1000,
            control_kind = "stiffen"
        }
    }
}

E["CS_jh_pjjt04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_pjjt04",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1,
            displacement_distance_m = 0,
            displacement_kind = "push"
        },
        {
            apply_buff = "buf_CS_jh_pjjt07",
            apply_life_ms = 3000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_pjjt05"] = {
    on_hit = {
        {
            control_dur_ms = 5000,
            control_kind = "launch"
        }
    }
}

E["CS_jh_pjjt06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_pjjt06",
            apply_life_ms = 7000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 7000,
            control_kind = "knockdown",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    unparsed = { "自身于空中释放时，额外获得1秒黄霸体效果（激活此效果需与巫墟宗派深处山鬼地窟的巫莎琪（77,-1" }
}

E["CS_jh_pjjt07"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_pjjt07",
            apply_life_ms = 3000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_pjjt07_hide"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_pjjt07",
            apply_life_ms = 3000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_pjjt08"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_pjjt07",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_plbz01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_plbz01",
            apply_life_ms = 6000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "使用后消耗一层“破”" }
}

E["CS_jh_plbz02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "实破虚出：出招过程中可施展【明暗纷纭】打断本招式，同时获得1层“破”" }
}

E["CS_jh_plbz03"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_plbz03_hide",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparryable = true },
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    },
    unparsed = { "自身存在“破”时，2秒内可消耗一层“破”再次施展招式" }
}

E["CS_jh_plbz03_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_plbz03_hide",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparryable = true },
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    }
}

E["CS_jh_plbz04"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "以静制动",
            apply_life_ms = 6000,
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 12000 }
    }
}

E["CS_jh_plbz05"] = {
    compile = {
        cast_armor = "yellow"
    },
    unparsed = { "状态持续8秒，可叠加5层，触发后清除1层" }
}

E["CS_jh_plbz06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_parry_break = {
        {
            apply_buff = "buf_CS_jh_plbz06",
            apply_life_ms = 6000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "虚破实出：出招过程中可施展【虚实无定】、【见微知著】打断本招式，同时获得1层“破”" }
}

E["CS_jh_plbz07"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "实破虚出：出招过程中可施展【明暗纷纭】打断本招式，同时获得1层“破”" }
}

E["CS_jh_pz18"] = {
    on_hit = {
        {
            control_dur_ms = 5000,
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_qbgf01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparryable = true },
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_qbgf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_qbgf03"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_jh_qbgf03",
            apply_life_ms = 4000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_parry_break.1.apply_life_ms"] = 5000 },
        [3] = { ["on_parry_break.1.apply_life_ms"] = 6000 },
        [4] = { ["on_parry_break.1.apply_life_ms"] = 7000 },
        [5] = { ["on_parry_break.1.apply_life_ms"] = 8000 },
        [6] = { ["on_parry_break.1.apply_life_ms"] = 9000 },
        [7] = { ["on_parry_break.1.apply_life_ms"] = 9000 },
        [8] = { ["on_parry_break.1.apply_life_ms"] = 9000 },
        [9] = { ["on_parry_break.1.apply_life_ms"] = 9000 },
        [10] = { ["on_parry_break.1.apply_life_ms"] = 9000 },
        [11] = { ["on_parry_break.1.apply_life_ms"] = 9000 },
        [12] = { ["on_parry_break.1.apply_life_ms"] = 9000 },
        [13] = { ["on_parry_break.1.apply_life_ms"] = 9000 },
        [14] = { ["on_parry_break.1.apply_life_ms"] = 9000 },
        [15] = { ["on_parry_break.1.apply_life_ms"] = 9000 },
        [16] = { ["on_parry_break.1.apply_life_ms"] = 9000 },
        [17] = { ["on_parry_break.1.apply_life_ms"] = 9000 },
        [18] = { ["on_parry_break.1.apply_life_ms"] = 9000 },
        [19] = { ["on_parry_break.1.apply_life_ms"] = 9000 }
    }
}

E["CS_jh_qbgf04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_qbgf04_1_1",
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            apply_buff = "buf_CS_jh_qbgf04_2_1",
            apply_resolved = true,
            apply_stacks = 1,
            per_hit = true
        }
    }
}

E["CS_jh_qbgf05"] = {
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        },
        {
            control_kind = "launch"
        },
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_qbgf06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_qbgf06",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_qbgf07"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_qbgf08"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    unparsed = { "招式每击附加目标当前气血2%的伤害（仅对玩家有效）" }
}

E["CS_jh_qjjy01"] = {
    unparsed = { "处于禁地中时，出招后使自身10秒内【竹影诗瘦】的招式伤害提高10%，最多可叠加3层" }
}

E["CS_jh_qjjy02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_qjjy02_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_qjjy03"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "root"
        }
    },
    unparsed = { "对范围内的敌人造成{@0:伤害包}，未招架的敌人被击中将减速50%，持续3秒" }
}

E["CS_jh_qjjy04"] = {
    unparsed = { "处于禁地中时，出招时可消耗1层“青筠”，回复少量怒气" }
}

E["CS_jh_qjjy05"] = {
    compile = {
        cast_armor = "yellow",
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "对范围内的敌人造成{@0:伤害包}，并使未招架的敌人缴械2秒" }
}

E["CS_jh_qjjy06"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        },
        {
            control_dur_ms = 5000,
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_qjjy07"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    },
    unparsed = { "处于禁地中时，破防成功将使【风竹敲秋】的回气时间额外减少5秒，同时自身获得3层“青筠”（“青筠”" }
}

E["CS_jh_qjjy08"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_qjjy08_1",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [4] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 10000 }
    },
    unparsed = { "处于禁地中时，“青筠”获得额外效果：【青筠剑意】招式伤害提高28%，持续15秒，可叠加5层" }
}

E["CS_jh_qxj01"] = {
    unparsed = { "每次攻击自身获得一层“贪婪”状态：攻击吸血(伤害的1%)，可叠加10层，持续14秒" }
}

E["CS_jh_qxj02"] = {

}

E["CS_jh_qxj03"] = {
    unparsed = { "真人之魄，贵掌寿基，招架成功时：自身获得“禄存”状态：攻击吸血(伤害的10%)，提升本套路招式伤" }
}

E["CS_jh_qxj04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "若目标前5秒内招架成功自己的招式，则使自己下一击伤害提升50%" }
}

E["CS_jh_qxj05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_qxj05",
            apply_life_ms = 7000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_stacks"] = 2 },
        [3] = { ["on_hit.1.apply_stacks"] = 3 },
        [4] = { ["on_hit.1.apply_stacks"] = 3 },
        [5] = { ["on_hit.1.apply_stacks"] = 3 },
        [6] = { ["on_hit.1.apply_stacks"] = 3 },
        [7] = { ["on_hit.1.apply_stacks"] = 3 },
        [8] = { ["on_hit.1.apply_stacks"] = 3 },
        [9] = { ["on_hit.1.apply_stacks"] = 3 },
        [10] = { ["on_hit.1.apply_stacks"] = 3 }
    },
    unparsed = { "（单次恢复的气血值，不超过自身最大气血值的7%）" }
}

E["CS_jh_qxj06"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        can_move = false,
        cast_armor = "yellow"
    },
    unparsed = { "出招过程中，每次攻击（敌人未招架）或每次被攻击时，自身获得1层“武曲”状态：提升100点外功防御" }
}

E["CS_jh_qxj07"] = {
    unparsed = { "招式每次命中未招架目标都获得1层“武曲”状态：提升100点外功防御和10点内功防御，可叠加49层" }
}

E["CS_jh_qxz01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_qxz02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_qxz03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    unparsed = { "招式施展完毕自己向身前冲刺10米" }
}

E["CS_jh_qxz04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_qxz05"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围15米，对霸体无效，可被虚招克制）" }
}

E["CS_jh_qxz05_hide"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围20米，对霸体无效，可被虚招克制）" }
}

E["CS_jh_qxz06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    }
}

E["CS_jh_qxz07"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_qxz07",
            apply_life_ms = 3000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_qyzf01"] = {
    on_hit = {
        {
            control_kind = "coma"
        },
        {
            control_dur_ms = 3000,
            control_kind = "coma"
        }
    }
}

E["CS_jh_qyzf01_hide"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "coma"
        }
    }
}

E["CS_jh_qyzf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "铁劲缠身并且清除自身【沧海一现】",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "千里追魂，烈焰缠身，冲击敌人对之造成{@0:伤害包}，招式命中未格挡目标可消耗其身上三层“铁劲缠" }
}

E["CS_jh_qyzf02_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "铁劲缠身并且清除自身【沧海一现】",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_qyzf03"] = {
    unparsed = { "狂龙厉行，铁甲之威，生出一道内劲，增加外功防御，每招架对方的攻击，都会给敌方传递铁劲缠身效果", "处于心魔，挖宝，禁地玩法时，每存在1层铁劲缠身受到【青云掌法】招式的伤害提高10%此招源于青云堡" }
}

E["CS_jh_qyzf04"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        },
        {
            apply_buff = "buf_CS_jh_qyzf03",
            apply_resolved = true,
            apply_stacks = 2,
            control_kind = "launch"
        }
    }
}

E["CS_jh_qyzf05"] = {
    on_hit = {
        {
            control_kind = "coma"
        },
        {
            apply_buff = "铁劲缠身原是青云堡老管家的厉害手",
            apply_resolved = false,
            apply_stacks = 2,
            control_kind = "coma"
        }
    }
}

E["CS_jh_qyzf06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "铁劲缠身本是青云堡老管家的看家功",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_qyzf06_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_qyzf03",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_qyzf07"] = {
    on_hit = {
        {
            control_kind = "ban_qinggong"
        },
        {
            apply_buff = "铁劲缠身此乃青云堡吴真的成名绝技",
            apply_resolved = false,
            apply_stacks = 1,
            control_kind = "root"
        }
    }
}

E["CS_jh_qyzf07_hide"] = {
    on_hit = {
        {
            apply_buff = "铁劲缠身此乃青云堡吴真的成名绝技",
            apply_resolved = false,
            apply_stacks = 1,
            control_kind = "coma"
        }
    }
}

E["CS_jh_qzjf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_qzjf03"] = {
    compile = {
        cast_armor = "red"
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_qzjf04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_qzjf05"] = {
    on_parry_break = {
        {
            control_within_ms = 3000
        }
    }
}

E["CS_jh_qzjf06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 3 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    },
    by_level = {
        [7] = { ["compile.aoe_radius_m"] = 4 },
        [8] = { ["compile.aoe_radius_m"] = 5 },
        [9] = { ["compile.aoe_radius_m"] = 5, ["on_hit.1.control_dur_ms"] = 4000 },
        [10] = { ["compile.aoe_radius_m"] = 5, ["on_hit.1.control_dur_ms"] = 5000 }
    }
}

E["CS_jh_qzjf07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        air = "never"
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    },
    self_state = {
        {
            apply_buff = "buf_CS_jh_qzjf07",
            apply_resolved = true
        }
    }
}

E["CS_jh_qzjf07_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_sbz01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_sbz01",
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            apply_buff = "buf_CS_jh_sbz01_2",
            apply_life_ms = 20000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_sbz02"] = {
    unparsed = { "若目标存在“贱龙”标志，则使其陷入“无手”状态：键位错乱，状态持续6秒（根据“贱龙”叠加次数延长" }
}

E["CS_jh_sbz03"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_sbz03",
            apply_life_ms = 3000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 4000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 5000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [13] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [14] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [15] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [16] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [17] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [18] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [19] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [20] = { ["on_hit.1.apply_life_ms"] = 8000 }
    }
}

E["CS_jh_sbz04"] = {
    on_hit = {
        {
            control_kind = "coma"
        }
    }
}

E["CS_jh_sf01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_sf02"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_sf03"] = {
    on_parry_break = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong",
            control_within_ms = 3000
        }
    }
}

E["CS_jh_sf04"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_sf04",
            apply_life_ms = 15000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_jh_sf05"] = {
    on_hit = {
        {
            apply_buff = "一层独善其身状态",
            apply_life_ms = 8000,
            apply_resolved = false,
            apply_stacks = 1,
            chance = 0.3
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.chance"] = 0.35 },
        [3] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.chance"] = 0.35 },
        [4] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.chance"] = 0.4 },
        [5] = { ["on_hit.1.apply_life_ms"] = 11000, ["on_hit.1.chance"] = 0.4 },
        [6] = { ["on_hit.1.apply_life_ms"] = 12000, ["on_hit.1.chance"] = 0.4 },
        [7] = { ["on_hit.1.apply_life_ms"] = 12000, ["on_hit.1.chance"] = 0.45 },
        [8] = { ["on_hit.1.apply_life_ms"] = 12000, ["on_hit.1.chance"] = 0.45 },
        [9] = { ["on_hit.1.apply_life_ms"] = 12000, ["on_hit.1.chance"] = 0.5 },
        [10] = { ["on_hit.1.apply_life_ms"] = 12000, ["on_hit.1.chance"] = 0.5 },
        [11] = { ["on_hit.1.apply_life_ms"] = 12000, ["on_hit.1.chance"] = 0.55 },
        [12] = { ["on_hit.1.apply_life_ms"] = 15000, ["on_hit.1.chance"] = 0.6 }
    }
}

E["CS_jh_sf06"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_jh_sf06_hide"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_jh_sf06_hide_npc"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_jh_sf07"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow",
        air = "never"
    }
}

E["CS_jh_sf08"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow",
        air = "never"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "dizzy",
            per_hit = true
        }
    }
}

E["CS_jh_sfgp01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    by_level = {
        [6] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [7] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [8] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [9] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [10] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [11] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [12] = { ["on_hit.1.displacement_distance_m"] = 0 }
    }
}

E["CS_jh_sfgp02"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_sfgp03"] = {
    on_parry_break = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_sfgp04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 20 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_sfgp04",
            apply_life_ms = 15000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        },
        {
            apply_buff = "纵横状态",
            apply_life_ms = 10000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    },
    by_level = {
        [6] = { ["on_hit.1.displacement_distance_m"] = 0, ["on_hit.2.apply_life_ms"] = 15000, ["on_hit.3.apply_life_ms"] = 10000, ["on_hit.3.apply_stacks"] = 1 },
        [7] = { ["on_hit.1.displacement_distance_m"] = 0, ["on_hit.2.apply_life_ms"] = 15000, ["on_hit.3.apply_life_ms"] = 10000, ["on_hit.3.apply_stacks"] = 1 },
        [8] = { ["on_hit.1.displacement_distance_m"] = 0, ["on_hit.2.apply_life_ms"] = 15000, ["on_hit.3.apply_life_ms"] = 10000, ["on_hit.3.apply_stacks"] = 1 },
        [9] = { ["on_hit.1.displacement_distance_m"] = 0, ["on_hit.2.apply_life_ms"] = 15000, ["on_hit.3.apply_life_ms"] = 10000, ["on_hit.3.apply_stacks"] = 1 },
        [10] = { ["on_hit.1.displacement_distance_m"] = 0, ["on_hit.2.apply_life_ms"] = 15000, ["on_hit.3.apply_life_ms"] = 10000, ["on_hit.3.apply_stacks"] = 1 },
        [11] = { ["on_hit.1.displacement_distance_m"] = 0, ["on_hit.2.apply_life_ms"] = 15000, ["on_hit.3.apply_life_ms"] = 10000, ["on_hit.3.apply_stacks"] = 1 },
        [12] = { ["on_hit.1.displacement_distance_m"] = 0, ["on_hit.2.apply_life_ms"] = 15000, ["on_hit.3.apply_life_ms"] = 10000, ["on_hit.3.apply_stacks"] = 1 }
    }
}

E["CS_jh_sfgp05"] = {
    on_hit = {
        {
            apply_buff = "一层独善其身状态",
            apply_life_ms = 10000,
            apply_resolved = false,
            apply_stacks = 1,
            chance = 0.35
        }
    },
    by_level = {
        [2] = { ["on_hit.1.chance"] = 0.4 },
        [3] = { ["on_hit.1.apply_life_ms"] = 11000, ["on_hit.1.chance"] = 0.4 },
        [4] = { ["on_hit.1.apply_life_ms"] = 12000, ["on_hit.1.chance"] = 0.4 },
        [5] = { ["on_hit.1.apply_life_ms"] = 12000, ["on_hit.1.chance"] = 0.45 },
        [6] = { ["on_hit.1.apply_life_ms"] = 12000, ["on_hit.1.chance"] = 0.45 },
        [7] = { ["on_hit.1.apply_life_ms"] = 12000, ["on_hit.1.chance"] = 0.5 },
        [8] = { ["on_hit.1.apply_life_ms"] = 12000, ["on_hit.1.chance"] = 0.5 },
        [9] = { ["on_hit.1.apply_life_ms"] = 12000, ["on_hit.1.chance"] = 0.55 },
        [10] = { ["on_hit.1.apply_life_ms"] = 60000, ["on_hit.1.chance"] = 0.6 },
        [11] = { ["on_hit.1.apply_life_ms"] = 60000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 60000 }
    }
}

E["CS_jh_sfgp06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 3 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_jh_sfgp06_hide"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_jh_sfgp06_hide_npc"] = {
    on_hit = {
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_jh_sfgp07"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "red",
        air = "never"
    },
    by_level = {
        [6] = { ["on_hit.1.apply_life_ms"] = 18000, ["on_hit.1.apply_stacks"] = 1 },
        [7] = { ["on_hit.1.apply_life_ms"] = 20000, ["on_hit.1.apply_stacks"] = 1 },
        [8] = { ["on_hit.1.apply_life_ms"] = 20000, ["on_hit.1.apply_stacks"] = 1 },
        [9] = { ["on_hit.1.apply_life_ms"] = 25000, ["on_hit.1.apply_stacks"] = 1 },
        [10] = { ["on_hit.1.apply_life_ms"] = 30000, ["on_hit.1.apply_stacks"] = 1 },
        [11] = { ["compile.aoe_radius_m"] = 20, ["on_hit.1.apply_stacks"] = 1, ["on_hit.2.displacement_distance_m"] = 0, ["on_hit.3.apply_life_ms"] = 20000, ["on_hit.3.apply_stacks"] = 1, ["on_hit.4.apply_stacks"] = 1, ["on_hit.4.control_dur_ms"] = 3000, ["on_hit.4.displacement_distance_m"] = 0, ["on_hit.5.apply_stacks"] = 1, ["on_hit.5.displacement_distance_m"] = 0 },
        [12] = { ["compile.aoe_radius_m"] = 20, ["on_hit.1.apply_stacks"] = 1, ["on_hit.2.displacement_distance_m"] = 0, ["on_hit.3.apply_life_ms"] = 20000, ["on_hit.3.apply_stacks"] = 1, ["on_hit.4.apply_stacks"] = 1, ["on_hit.4.control_dur_ms"] = 3000, ["on_hit.4.displacement_distance_m"] = 0, ["on_hit.5.apply_stacks"] = 1, ["on_hit.5.displacement_distance_m"] = 0 }
    }
}

E["CS_jh_sfgp07_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "red",
        air = "never"
    },
    by_level = {
        [6] = { ["on_hit.1.apply_life_ms"] = 18000, ["on_hit.1.apply_stacks"] = 1 },
        [7] = { ["on_hit.1.apply_life_ms"] = 20000, ["on_hit.1.apply_stacks"] = 1 },
        [8] = { ["on_hit.1.apply_life_ms"] = 20000, ["on_hit.1.apply_stacks"] = 1 },
        [9] = { ["on_hit.1.apply_life_ms"] = 25000, ["on_hit.1.apply_stacks"] = 1 },
        [10] = { ["on_hit.1.apply_life_ms"] = 30000, ["on_hit.1.apply_stacks"] = 1 },
        [11] = { ["compile.aoe_radius_m"] = 20, ["on_hit.1.apply_stacks"] = 1, ["on_hit.2.displacement_distance_m"] = 0, ["on_hit.3.apply_life_ms"] = 20000, ["on_hit.3.apply_stacks"] = 1, ["on_hit.4.apply_stacks"] = 1, ["on_hit.4.control_dur_ms"] = 3000, ["on_hit.4.displacement_distance_m"] = 0, ["on_hit.5.apply_stacks"] = 1, ["on_hit.5.displacement_distance_m"] = 0 },
        [12] = { ["compile.aoe_radius_m"] = 20, ["on_hit.1.apply_stacks"] = 1, ["on_hit.2.displacement_distance_m"] = 0, ["on_hit.3.apply_life_ms"] = 20000, ["on_hit.3.apply_stacks"] = 1, ["on_hit.4.apply_stacks"] = 1, ["on_hit.4.control_dur_ms"] = 3000, ["on_hit.4.displacement_distance_m"] = 0, ["on_hit.5.apply_stacks"] = 1, ["on_hit.5.displacement_distance_m"] = 0 }
    }
}

E["CS_jh_sfgp08"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "red",
        air = "never"
    },
    on_hit = {
        {
            condition = { unparried = true, unparryable = true },
            control_kind = "launch",
            per_hit = true
        }
    }
}

E["CS_jh_sfj01"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_sfj01_1"] = {
    unparsed = { "弗嗔弗喜，神锥横凿，击晕前方敌人，并造成{@0:伤害包}" }
}

E["CS_jh_sfj01_2"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_sfj02"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_sfj02_1",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "root"
        }
    },
    unparsed = { "同时自身获得“飞廉（盈）”状态：增加移动速度10%、轻功施展距离1米、冲锋距离10米，持续5秒" }
}

E["CS_jh_sfj03"] = {
    on_parry_break = {
        {
            control_dur_ms = 3000,
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_sfj04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_sfj04_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "弄月",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_sfj05"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_sfj05_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_sfj05_1",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_sfj06"] = {
    self_state = {
        {
            apply_buff = "buf_CS_jh_sfj06",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_jh_sfj07"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        },
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_sfj08"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    }
}

E["CS_jh_sfsj01"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_sfsj02"] = {
    unparsed = { "破防成功时额外僵直0.7秒并在5秒内被暴击几率提高15%" }
}

E["CS_jh_sfsj03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_sfsj04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "silence"
        },
        {
            apply_buff = "一层持续5秒的\"天命昭昭\"状态）",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_sfsj05"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_sfsj06"] = {
    unparsed = { "舞扇对目标造成{@0:伤害包}，命中已被\"天命昭昭\"所扰的未招架敌方时，额外向敌方传入一道\"飒风" }
}

E["CS_jh_sfsj07"] = {
    on_hit = {
        {
            apply_buff = "一层\"天命昭昭\"状态",
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 10000,
            control_kind = "slow"
        }
    }
}

E["CS_jh_sfsj08"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "root",
            heal = true
        }
    }
}

E["CS_jh_shlkd01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    }
}

E["CS_jh_shlkd02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_shlkd03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "silence"
        }
    }
}

E["CS_jh_shlkd04"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_shlkd05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "silence",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_shlkd06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_shlkd07",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "stun"
        },
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_shlkd07"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_shlkd07",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 2000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_shlkd08"] = {
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_jh_shlkd08_hide"] = {
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_jh_sjjc01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_sjjc02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    }
}

E["CS_jh_sjjc03"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_sjjc04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_sjjc04_01",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_jh_sjjc05"] = {
    unparsed = { "向8到15米距离的选中区域内连续射出大量箭枝，对范围内的敌人造成{@0:伤害包}，并使未招架的敌", "招式在空中施展时，效果变弱：箭雨变少，伤害降低" }
}

E["CS_jh_sjjc05_sky"] = {
    unparsed = { "向8到15米距离的选中区域内连续射出大量箭枝，对范围内的敌人造成{@0:伤害包}，并使未招架的敌", "招式在空中施展时，效果变弱：箭雨变少，伤害降低" }
}

E["CS_jh_sjjc06"] = {
    unparsed = { "对选中敌人造成{@0:伤害包}，直接击落空中目标，该招式命中率极高", "击中未招架且已被灼伤的玩家将使目标玩家5秒内气血混乱（每秒造成当前气血1%的伤害且气血恢复速率降" }
}

E["CS_jh_sjjc07"] = {
    unparsed = { "弓藏火引，借劲燃之，借招架反震之力点燃箭簇之技", "成功招架敌人攻击时，自身获得“流星飞火”状态：持续12秒，可叠加3层，增强套路下所有射箭招式，使", "灼伤：每秒造成17点阳伤害，持续5秒，可叠加5层，层数越高伤害越高" }
}

E["CS_jh_sjjc08"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        },
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    },
    by_level = {
        [8] = { ["compile.aoe_radius_m"] = 25 },
        [9] = { ["compile.aoe_radius_m"] = 25 },
        [10] = { ["compile.aoe_radius_m"] = 25 }
    }
}

E["CS_jh_sjjc08_sky"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        },
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    },
    by_level = {
        [8] = { ["compile.aoe_radius_m"] = 25 },
        [9] = { ["compile.aoe_radius_m"] = 25 },
        [10] = { ["compile.aoe_radius_m"] = 25 }
    }
}

E["CS_jh_sjjc09"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "coma",
            per_hit = true
        }
    },
    unparsed = { "招式在空中施展时，效果变弱：伤害变低，跳闪距离变短" }
}

E["CS_jh_sjjc09_sky"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "coma",
            per_hit = true
        }
    },
    unparsed = { "招式在空中施展时，效果变弱：伤害变低，跳闪距离变短" }
}

E["CS_jh_slbwj01"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_slbwj02"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_slbwj04"] = {
    on_hit = {
        {
            control_kind = "stun"
        }
    },
    on_parry_break = {
        {
            apply_buff = "buf_CS_jh_slbwj04",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 1000,
            control_kind = "stun"
        }
    },
    by_level = {
        [4] = { ["on_parry_break.1.control_dur_ms"] = 2000 },
        [5] = { ["on_parry_break.1.control_dur_ms"] = 2000 },
        [6] = { ["on_parry_break.1.control_dur_ms"] = 2000 },
        [7] = { ["on_parry_break.1.control_dur_ms"] = 2000 },
        [8] = { ["on_parry_break.1.control_dur_ms"] = 2000 },
        [9] = { ["on_parry_break.1.control_dur_ms"] = 2000 },
        [10] = { ["on_parry_break.1.control_dur_ms"] = 3000 },
        [11] = { ["on_parry_break.1.control_dur_ms"] = 3000 },
        [12] = { ["on_parry_break.1.control_dur_ms"] = 3000 },
        [13] = { ["on_parry_break.1.control_dur_ms"] = 3000 },
        [14] = { ["on_parry_break.1.control_dur_ms"] = 3000 },
        [15] = { ["on_parry_break.1.control_dur_ms"] = 3000 },
        [16] = { ["on_parry_break.1.control_dur_ms"] = 3000 },
        [17] = { ["on_parry_break.1.control_dur_ms"] = 3000 },
        [18] = { ["on_parry_break.1.control_dur_ms"] = 3000 },
        [19] = { ["on_parry_break.1.control_dur_ms"] = 3000 }
    }
}

E["CS_jh_slbwj05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_slbwj05_01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_slbwj06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_slbwj07"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    }
}

E["CS_jh_slwx_jctq"] = {
    on_hit = {
        {
            control_dur_ms = 5000,
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_sxjj01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_sxjj05_01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_sxjj02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_sxjj05_01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_sxjj03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "launch"
        },
        {
            apply_buff = "buf_CS_jh_sxjj05_01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_sxjj04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_sxjj05_01",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    on_parry_break = {
        {
            apply_buff = "buf_CS_jh_sxjj05_01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_sxjj05"] = {
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "coma"
        }
    },
    unparsed = { "拧身旋剑，密雨无间，招架成功，使没有“无间”状态的目标获得1层“无间”状态，持续12秒" }
}

E["CS_jh_sxjj06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "ban_qinggong"
        },
        {
            apply_buff = "buf_CS_jh_sxjj05_01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_sxjj07"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_sxjj05_01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_szzhg01"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_jh_szzhg02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true,
        air = "only"
    },
    unparsed = { "目标未招架则将其击落" }
}

E["CS_jh_szzhg02_sky"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_szzhg03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_szzhg04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 10 }
    }
}

E["CS_jh_szzhg05"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_szzhg07"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_szzhg07_01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_szzhg08"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_jh_szzhg08",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 5000,
            control_kind = "launch"
        }
    }
}

E["CS_jh_szzhg09"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow",
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_tgf01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_tgf02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_jh_tgf03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_tgf04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_tgf05"] = {
    on_hit = {
        {
            apply_buff = "一层“横担金斧",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 11000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 12000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 13000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 13000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 15000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 15000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 20000 }
    }
}

E["CS_jh_tgf06"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    }
}

E["CS_jh_tgf07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_tlq02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_tlq03"] = {
    on_hit = {
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        },
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_tlq04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_tlq04",
            apply_resolved = true,
            apply_stacks = 1,
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    unparsed = { "蓄力招式，凝聚内力后破空而击，对目标造成{@0:伤害包}" }
}

E["CS_jh_tlq04_sky"] = {
    unparsed = { "蓄力1.5秒奋力一击，震伤目标使之3秒内无法招架" }
}

E["CS_jh_tlq05"] = {
    on_parry_break = {
        {
            control_within_ms = 2000,
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    by_level = {
        [3] = { ["on_parry_break.1.control_within_ms"] = 3000 },
        [4] = { ["on_parry_break.1.control_within_ms"] = 3000 },
        [5] = { ["on_parry_break.1.control_within_ms"] = 4000 },
        [6] = { ["on_parry_break.1.control_within_ms"] = 4000 },
        [7] = { ["on_parry_break.1.control_within_ms"] = 4000 },
        [8] = { ["on_parry_break.1.control_within_ms"] = 4000 },
        [9] = { ["on_parry_break.1.control_within_ms"] = 4000 },
        [10] = { ["on_parry_break.1.control_within_ms"] = 4000 },
        [11] = { ["on_parry_break.1.control_within_ms"] = 4000 },
        [12] = { ["on_parry_break.1.control_within_ms"] = 4000 },
        [13] = { ["on_parry_break.1.control_within_ms"] = 4000 },
        [14] = { ["on_parry_break.1.control_within_ms"] = 4000 },
        [15] = { ["on_parry_break.1.control_within_ms"] = 4000 },
        [16] = { ["on_parry_break.1.control_within_ms"] = 4000 },
        [17] = { ["on_parry_break.1.control_within_ms"] = 4000 },
        [18] = { ["on_parry_break.1.control_within_ms"] = 4000 },
        [19] = { ["on_parry_break.1.control_within_ms"] = 4000 }
    }
}

E["CS_jh_tlq06"] = {
    unparsed = { "沉身定气，展臂护体，招架成功时有30%几率激发潜能：使招式【螳臂破车】威力提升30%，并可凌空瞬" }
}

E["CS_jh_tlw01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_tlw01",
            apply_life_ms = 8000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 2000,
            control_kind = "root"
        }
    }
}

E["CS_jh_tlw02"] = {
    unparsed = { "（目标在空中时，吸引无效，只将其击落）（吸引效果只对玩家有效）" }
}

E["CS_jh_tlw03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_tlw03_sky"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_tlw04"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_tlw05"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_jh_tlw05",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_tlw06"] = {
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_tlw07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    unparsed = { "同时自身获得2层“撩蝉纱”状态：再次施招时，自身可与『舞影』互换位置，并对自身周围的敌人造成伤害", "该状态持续10秒，可叠加2层，每次互换消耗1层" }
}

E["CS_jh_tlw07_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    }
}

E["CS_jh_tmby01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "旋琴弄弦，对选中目标造成{@0:伤害包}，消耗1层“余音”可降低未招架的敌人5%的招式伤害，持续" }
}

E["CS_jh_tmby02"] = {
    unparsed = { "竖琴弹弦，对选中目标造成{@0:伤害包}，消耗1层“余音”可降低未招架的敌人5%的招式伤害，持续" }
}

E["CS_jh_tmby03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "silence"
        }
    },
    unparsed = { "消耗1层“余音”可向身前弹出一道“余音音波”：替触碰到的友方目标回复气血（受近身威力和内功威力加" }
}

E["CS_jh_tmby04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    },
    by_level = {
        [11] = { ["on_hit.1.control_dur_ms"] = 2000, ["on_hit.2.control_dur_ms"] = 3000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 2000, ["on_hit.2.control_dur_ms"] = 3000 }
    },
    unparsed = { "消耗1层“余音”可向身前弹出一道“余音音波”：对触碰到的敌人造成柔伤害（受近身威力和内功威力加成" }
}

E["CS_jh_tmby05"] = {
    self_state = {
        {
            apply_buff = "buf_CS_jh_tmby05_01",
            apply_life_ms = 12000,
            apply_resolved = true
        }
    },
    by_level = {
        [5] = { ["self_state.1.apply_life_ms"] = 10000 },
        [7] = { ["self_state.1.apply_life_ms"] = 14000 },
        [8] = { ["self_state.1.apply_life_ms"] = 16000 },
        [9] = { ["self_state.1.apply_life_ms"] = 18000 },
        [10] = { ["self_state.1.apply_life_ms"] = 20000 },
        [11] = { ["self_state.1.apply_life_ms"] = 25000 },
        [12] = { ["self_state.1.apply_life_ms"] = 30000 }
    }
}

E["CS_jh_tmby06"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 10 },
        cast_armor = "red"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        },
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_tmby06_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 10 },
        cast_armor = "red"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        },
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_tmby07"] = {
    by_level = {
        [5] = { ["compile.aoe_radius_m"] = 6 },
        [6] = { ["compile.aoe_radius_m"] = 7 },
        [7] = { ["compile.aoe_radius_m"] = 8 },
        [8] = { ["compile.aoe_radius_m"] = 9 },
        [9] = { ["compile.aoe_radius_m"] = 10 },
        [10] = { ["compile.aoe_radius_m"] = 12 },
        [11] = { ["compile.aoe_radius_m"] = 14 },
        [12] = { ["compile.aoe_radius_m"] = 15 }
    },
    unparsed = { "曲终琴抱，余音袅袅，使自身承受的伤害降低10%，且受到的减益状态无法叠层（该效果生效2次），持续" }
}

E["CS_jh_tmby08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 },
        cast_armor = "red"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "silence",
            per_hit = true
        }
    },
    by_level = {
        [4] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [5] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [6] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 2000 }
    }
}

E["CS_jh_tmjt03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_tmjt04"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_jh_tmjt04",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_tmjt05"] = {
    on_hit = {
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_tmjt06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_tmjt07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_tqjf01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_tqjf01",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "（使用2次后第3次会击晕目标1秒）" }
}

E["CS_jh_tqjf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_tqjf03"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_tqjf04"] = {
    on_hit = {
        {
            control_kind = "ban_qinggong"
        },
        {
            apply_buff = "三层“临渊",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_tqjf05"] = {
    on_hit = {
        {
            apply_buff = "状态",
            apply_life_ms = 6000,
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 1000,
            control_kind = "stun"
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [13] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [14] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [15] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [16] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [17] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [18] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 },
        [19] = { ["on_hit.1.apply_life_ms"] = 10000, ["on_hit.1.control_dur_ms"] = 2000 }
    }
}

E["CS_jh_tqjf06"] = {
    unparsed = { "招架成功时，自身获得“临渊”状态，最多可叠加3层：激发傲骨天性，使得【风岚无相】【穹崖九仞】【太", "涤荡平息怒火，减少【剑归太虚】施展所消耗的怒气（每1层“临渊”状态减少1点怒气消耗，每次攻击消耗" }
}

E["CS_jh_tqjf07"] = {
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_tqjf08"] = {
    compile = {
        max_hits = 3
    },
    by_level = {
        [2] = { ["compile.max_hits"] = 4 },
        [3] = { ["compile.max_hits"] = 5 },
        [4] = { ["compile.max_hits"] = 6 },
        [5] = { ["compile.max_hits"] = 7 },
        [6] = { ["compile.max_hits"] = 8 },
        [7] = { ["compile.max_hits"] = 8 },
        [8] = { ["compile.max_hits"] = 8 },
        [9] = { ["compile.max_hits"] = 8 },
        [10] = { ["compile.max_hits"] = 8 },
        [11] = { ["compile.max_hits"] = 8 },
        [12] = { ["compile.max_hits"] = 8 },
        [13] = { ["compile.max_hits"] = 8 },
        [14] = { ["compile.max_hits"] = 8 },
        [15] = { ["compile.max_hits"] = 8 },
        [16] = { ["compile.max_hits"] = 8 },
        [17] = { ["compile.max_hits"] = 8 },
        [18] = { ["compile.max_hits"] = 8 },
        [19] = { ["compile.max_hits"] = 8 }
    },
    unparsed = { "（施展此招式需要10点怒气，每攻击一次消耗10点怒气）" }
}

E["CS_jh_tslyz01"] = {
    unparsed = { "运功于掌，向敌人急速射出乱心尘，没入敌身顷刻发作，使中招之人生命内力不断流失，身形迟缓" }
}

E["CS_jh_tslyz03"] = {
    unparsed = { "若目标体内带有乱心尘，则以阳刚内力引发乱心尘发作造成阳伤害" }
}

E["CS_jh_tslyz04"] = {
    unparsed = { "若目标体内带有乱心尘，则以阴柔内力引发乱心尘发作造成阴伤害" }
}

E["CS_jh_tslyz06"] = {
    unparsed = { "坚守之式，以纯阳真气催动内力循环，护体盘身", "成功格挡住敌人进攻时，可将攻向自己的气劲，部分化为己用，增强自身内力" }
}

E["CS_jh_tslyz07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_tsz01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_tsz",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        },
        {
            control_dur_ms = 3000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_tsz02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_tsz",
            apply_resolved = true,
            apply_stacks = 3
        },
        {
            control_dur_ms = 3000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_tsz03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_tsz",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_jh_tsz04"] = {
    unparsed = { "获得5秒的洪渎印效果：被攻击时若攻击者存在5层以上“咒”，则反弹30%伤害（单次上限不超过自身内" }
}

E["CS_jh_tsz06"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "dizzy"
        },
        {
            control_dur_ms = 15000,
            control_kind = "launch",
            per_hit = true
        }
    }
}

E["CS_jh_tsz07"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_ttg01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_ttg01",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "每层状态增加5%伤害，最多可叠加3层，持续15秒" }
}

E["CS_jh_ttg02"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    },
    unparsed = { "（招式最大冲锋距离为20米）" }
}

E["CS_jh_ttg03"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_ttg04"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_jh_ttg04",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_ttg05"] = {
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_jh_ttg06"] = {

}

E["CS_jh_txd01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    }
}

E["CS_jh_txd02"] = {
    unparsed = { "对目标造成{@0:伤害包}（此招式可用轻功快速接近对手）" }
}

E["CS_jh_txd03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 10 }
    },
    on_parry_break = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_txd04"] = {
    unparsed = { "格挡成功使自身【吞象典】招式伤害提高5%持续5秒最多可叠加1层" }
}

E["CS_jh_txd05"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_txd06"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_txd07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_txd08"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_tzp01"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_tzp02"] = {
    compile = {
        cast_armor = "yellow",
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_tzp03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_jh_tzp03_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_jh_tzp04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_tzp04",
            apply_life_ms = 15000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_tzp05"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_parry_break = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_tzp05_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_parry_break = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_tzp06"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown",
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_jh_tzp07"] = {
    compile = {
        cast_armor = "yellow"
    },
    unparsed = { "对目标造成{@0:伤害包}处于禁地中时，招式第一击命中未招架目标时，若目标身上不存在“横天”状态" }
}

E["CS_jh_tzp08"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "red"
    },
    unparsed = { "存在10层以上“还空”且气血低于25%时招式使用后立即恢复30%气血，同时使自身进入2分钟的“反" }
}

E["CS_jh_whjf01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_whjf01",
            apply_life_ms = 15000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 15000,
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_whjf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_whjf03"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow",
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_whjf04"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_jh_whjf04",
            apply_life_ms = 6000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 6000,
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_whjf05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_whjf05_2",
            apply_life_ms = 15000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_whjf06"] = {
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_whjf07"] = {
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_whjf08"] = {
    unparsed = { "醉里挑灯看剑，梦回吹角连营，使自身进入贪欢状态：吸收伤害，15秒内状态未被打破，则恢复自身气血（" }
}

E["CS_jh_wlbgg01"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_wlbgg01",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 2000,
            control_kind = "root"
        }
    }
}

E["CS_jh_wlbgg02"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_jh_wlbgg02",
            apply_life_ms = 4000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 4000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_wlbgg03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_wlbgg04"] = {
    unparsed = { "埋身退步，避锐懈敌，招架成功时，自身获得1层“蛰伏不鸣”状态，持续15秒", "该状态叠加至10层后，可激发“一鸣惊人”状态：10秒内，成功施展招式【挑纵手】后，可再次施展【挑" }
}

E["CS_jh_wlbgg05"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_wlbgg06"] = {
    compile = {
        cast_armor = "yellow",
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_wlbgg07"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_wlp01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_wlp02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "对选中目标造成{@0:伤害包}，在虎鹤双形状态下，招式冲刺速度、冲刺距离和命中几率都得到提升" }
}

E["CS_jh_wlp02_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "对选中目标造成{@0:伤害包}，在虎鹤双形状态下，招式冲刺速度、冲刺距离和命中几率都得到提升" }
}

E["CS_jh_wlp03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    },
    unparsed = { "蛇鹤双行状态下可使妄图招架的敌人沾染蛇毒：5秒内移动速度越来越慢" }
}

E["CS_jh_wlp04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    unparsed = { "消耗3层三转兽劲可在施展招式时获得黄霸体", "虎鹤双形状态下腿势密集，招式影响范围内的敌人无法招架" }
}

E["CS_jh_wlp05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_jh_wlp05_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_jh_wlp06"] = {
    unparsed = { "对选中目标造成{@0:伤害包}，每次命中敌人，皆可打散自身1层三转兽劲" }
}

E["CS_jh_wsgf01"] = {
    unparsed = { "桃园金兰誓，弟兄山海盟，招架成功时，有一定几率激活“结义”状态：恢复自身和队友的气血、内力（气血" }
}

E["CS_jh_wsgf02"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    unparsed = { "重伤敌人（仅限玩家）时：清除招式【过关斩将】回气时间，并使招式【过关斩将】暴击几率极大提升" }
}

E["CS_jh_wsgf03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_wsgf05"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_wsgf05_01",
            apply_life_ms = 12000,
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.08,
            control_dur_ms = 4000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_wsgf06"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_parry_break = {
        {
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_jh_wsgf07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    },
    unparsed = { "每重伤一个目标（仅限玩家）：恢复自身气血、内力（气血内力最大值的20%），并重置关公显圣持续时间" }
}

E["CS_jh_wsgf08"] = {
    compile = {
        cast_armor = "red"
    }
}

E["CS_jh_wshtz01"] = {
    on_hit = {
        {
            apply_buff = "荒腾状态",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            control_kind = "slow"
        }
    }
}

E["CS_jh_wshtz02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_wshtz03"] = {

}

E["CS_jh_wshtz04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_wshtz05"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_wshtz06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_wshtz07"] = {
    on_hit = {
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_jh_wshtz08"] = {
    on_hit = {
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_jh_wxjj01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    },
    on_hit = {
        {
            apply_buff = "流水状态",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 5000,
            control_kind = "slow"
        }
    },
    by_level = {
        [3] = { ["on_hit.1.apply_life_ms"] = 6000, ["on_hit.1.control_dur_ms"] = 6000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 6000, ["on_hit.1.control_dur_ms"] = 6000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 7000, ["on_hit.1.control_dur_ms"] = 7000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 7000, ["on_hit.1.control_dur_ms"] = 7000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 8000, ["on_hit.1.control_dur_ms"] = 8000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 8000, ["on_hit.1.control_dur_ms"] = 8000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 8000, ["on_hit.1.control_dur_ms"] = 8000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 8000, ["on_hit.1.control_dur_ms"] = 8000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 8000, ["on_hit.1.control_dur_ms"] = 8000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 8000, ["on_hit.1.control_dur_ms"] = 8000 }
    }
}

E["CS_jh_wxjj02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            apply_buff = "厚土状态",
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_wxjj03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "灿金状态",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_jh_wxjj04"] = {
    unparsed = { "舞剑自守，自身进入五行化一状态，成功招架敌人攻击时，自身获得五行汇聚状态，增加自身移动速度10%" }
}

E["CS_jh_wxjj05"] = {
    on_hit = {
        {
            apply_buff = "木盾状态",
            apply_life_ms = 8000,
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_wxjj06"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow",
        air = "never"
    }
}

E["CS_jh_wxjj07"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_xfsyt01"] = {
    compile = {
        cast_armor = "yellow"
    },
    unparsed = { "同时自身获得“倚云”状态：持续15秒，提升【倚云听风】招式伤害，可叠加10层" }
}

E["CS_jh_xfsyt01_sky"] = {
    on_hit = {
        {
            apply_buff = "npc标记buff",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_xfsyt02"] = {
    on_parry_break = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_xfsyt02_sky"] = {
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_xfsyt03"] = {
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "root"
        }
    },
    unparsed = { "金鳞蛰伏，风云化龙，招架成功时：自身获得“独霸”状态：提升旋风威力，持续8秒" }
}

E["CS_jh_xfsyt03_sky"] = {
    on_hit = {
        {
            apply_buff = "buff2——“霸腿",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_xfsyt04"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_dur_ms = 1000,
            control_kind = "coma"
        },
        {
            apply_buff = "定身状态",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparryable = true },
            control_dur_ms = 5000,
            control_kind = "root"
        }
    }
}

E["CS_jh_xfsyt04_sky"] = {
    on_hit = {
        {
            apply_buff = "buff3\"扫叶\"定身3s",
            apply_resolved = false,
            apply_stacks = 1,
            control_kind = "root"
        }
    }
}

E["CS_jh_xfsyt05"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "root"
        },
        {
            condition = { unparryable = true },
            control_kind = "root"
        }
    }
}

E["CS_jh_xfsyt05_sky"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_xfsyt06"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "root"
        },
        {
            condition = { unparryable = true },
            control_kind = "root"
        }
    }
}

E["CS_jh_xfsyt06_sky"] = {
    unparsed = { "1hit，跳起冲锋，若自身已在空中则直接向下斜踢，造成{@0:伤害包}" }
}

E["CS_jh_xfsyt07"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_xfsyt07",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 1000,
            control_kind = "coma"
        },
        {
            control_kind = "coma"
        }
    },
    by_level = {
        [7] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [8] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 2000 }
    },
    unparsed = { "招式根据自身“倚云”层数附加额外伤害" }
}

E["CS_jh_xfz01"] = {
    unparsed = { "出招后移除自身1层“金刚加持”" }
}

E["CS_jh_xfz02"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "coma"
        },
        {
            apply_buff = "3秒黄霸体",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "出招后移除自身1层“金刚加持”" }
}

E["CS_jh_xfz02_hide"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "coma"
        },
        {
            apply_buff = "3秒黄霸体",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "出招后移除自身1层“金刚加持”" }
}

E["CS_jh_xfz03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 40 }
    },
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    },
    unparsed = { "同时加速自身气血回复，化解淤血，持续5秒" }
}

E["CS_jh_xfz04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 40 }
    }
}

E["CS_jh_xfz05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 20 }
    },
    on_hit = {
        {
            control_dur_ms = 5000,
            control_kind = "root"
        }
    },
    unparsed = { "出招后移除自身1层“金刚加持”" }
}

E["CS_jh_xfz06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    unparsed = { "处于“寂灭”之境的敌人每次被攻击，提升攻击者1%伤害，可叠加10层，持续1分钟" }
}

E["CS_jh_xfz07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 20 }
    },
    on_hit = {
        {
            control_dur_ms = 9000,
            control_kind = "launch"
        }
    },
    unparsed = { "招架成功时，自身获得加1层“金刚加持”：提升招式【非常无常】、【空相无相】、【天鼓雷音如来印】2" }
}

E["CS_jh_xld01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_xld01",
            apply_life_ms = 8000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_xld02"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_xld02",
            apply_life_ms = 15000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_xld03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            control_kind = "launch"
        },
        {
            control_dur_ms = 3000,
            control_kind = "launch"
        }
    }
}

E["CS_jh_xld05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_xld05",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 10000,
            control_kind = "launch"
        }
    }
}

E["CS_jh_xld06"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_xld06_01",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "stun",
            per_hit = true
        }
    }
}

E["CS_jh_xld08"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow",
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_xlj01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_dur_ms = 5000,
            control_kind = "slow"
        }
    }
}

E["CS_jh_xlj02"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_xlj02",
            apply_life_ms = 6000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [3] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 11000 }
    }
}

E["CS_jh_xlj03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_jh_xlj04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_jh_xlj05"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_jh_xlj06"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_xlj07"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_xlj08"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    },
    unparsed = { "稚元:怒气消耗降低5点" }
}

E["CS_jh_xlsbz03"] = {
    unparsed = { "沉身架势，气海盘龙，招架成功时，有20%几率清除目标身上1个增益状态，调息时间30秒" }
}

E["CS_jh_xlsbz04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    by_level = {
        [5] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [6] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [7] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [8] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [9] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [10] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [11] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [12] = { ["on_hit.1.displacement_distance_m"] = 0 }
    }
}

E["CS_jh_xlsbz05"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    },
    on_parry_break = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_xlsbz06"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    by_level = {
        [5] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [6] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [7] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [8] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [9] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [10] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [11] = { ["on_hit.1.displacement_distance_m"] = 0 },
        [12] = { ["on_hit.1.displacement_distance_m"] = 0 }
    }
}

E["CS_jh_xlsbz07"] = {
    on_hit = {
        {
            control_dur_ms = 1000,
            control_kind = "coma",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    by_level = {
        [5] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [6] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [7] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [8] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 3000 }
    }
}

E["CS_jh_xlsbz07_hs"] = {
    on_hit = {
        {
            control_dur_ms = 1000,
            control_kind = "coma",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    by_level = {
        [5] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [6] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [7] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [8] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 3000 }
    }
}

E["CS_jh_xlsbz10"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    },
    by_level = {
        [10] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 2000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 2000 }
    }
}

E["CS_jh_xlsbz11"] = {
    unparsed = { "对目标造成{@0:伤害包}，此招为由下而上的攻敌之术，与飞龙在天相为反生，能击落空中的目标，目标" }
}

E["CS_jh_xlsbz13"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 7 }
    }
}

E["CS_jh_xlsbz14"] = {
    unparsed = { "引导，快速的回复自身的连击点“劲力”，霸体" }
}

E["CS_jh_xlsbz15"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 },
        cast_armor = "red"
    },
    on_hit = {
        {
            control_kind = "coma"
        }
    }
}

E["CS_jh_xlsbz16"] = {
    unparsed = { "对身前目标造成{@0:伤害包}，破防成功震伤敌人致其内伤，使其使用招式有几率被反噬，受到伤害且出" }
}

E["CS_jh_xlsbz17"] = {
    unparsed = { "对目标造成{@0:伤害包}，集中所有的力量与一点，若目标招架则加速其招架耐力的消耗" }
}

E["CS_jh_xlsbz18"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "launch"
        },
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_xlsbz18_sky"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "launch"
        },
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_xxsyp01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_xxsyp01_1",
            apply_life_ms = 6000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [4] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 10000 }
    }
}

E["CS_jh_xxsyp02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_xxsyp04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "silence"
        },
        {
            control_dur_ms = 1000,
            control_kind = "stun"
        }
    },
    by_level = {
        [4] = { ["on_hit.2.control_dur_ms"] = 2000 },
        [5] = { ["on_hit.2.control_dur_ms"] = 2000 },
        [6] = { ["on_hit.2.control_dur_ms"] = 2000 },
        [7] = { ["on_hit.2.control_dur_ms"] = 2000 },
        [8] = { ["on_hit.2.control_dur_ms"] = 2000 },
        [9] = { ["on_hit.2.control_dur_ms"] = 2000 },
        [10] = { ["on_hit.2.control_dur_ms"] = 2000 },
        [11] = { ["on_hit.2.control_dur_ms"] = 2000 },
        [12] = { ["on_hit.2.control_dur_ms"] = 2000 }
    }
}

E["CS_jh_xxsyp05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_xxsyp06"] = {
    on_hit = {
        {
            apply_buff = "一层“寒江",
            apply_life_ms = 8000,
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    by_level = {
        [3] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 11000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 12000 }
    },
    unparsed = { "每层“寒江”降低目标【雁行功】和【穿云纵】的速度和距离2%", "每次施展轻功清除一层“寒江”" }
}

E["CS_jh_xxsyp07"] = {
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_xxsyp08"] = {
    on_hit = {
        {
            control_kind = "stiffen"
        }
    }
}

E["CS_jh_xyyys01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    }
}

E["CS_jh_xyyys02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_xyyys03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_xyyys04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_xyyys05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "silence",
            per_hit = true
        }
    },
    unparsed = { "招式收招后将消耗血月附体状态并使自身后续5秒内使用【蛊怨之瞳】【烬鸦召来】均有概率诱导毒瘴爆发：" }
}

E["CS_jh_xyyys07"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_xyyys08"] = {
    unparsed = { "招架成功时可将毒瘴打入攻击者体内", "若自身未处于血月附体状态则有30%概率祈求成功（祈求成功后进入60秒调息时间，升级招式层数可降低" }
}

E["CS_jh_yfjb01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_yfjb02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "云倾状态",
            apply_resolved = false,
            apply_stacks = 2,
            control_dur_ms = 2000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_yfjb03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_yfjb04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "silence",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_yfjb04_1"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    unparsed = { "使未招架目标僵直" }
}

E["CS_jh_yfjb04_2"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        },
        {
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    }
}

E["CS_jh_yfjb05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_yfjb05_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_yfjb06"] = {
    on_hit = {
        {
            control_dur_ms = 10000,
            control_kind = "ban_qinggong"
        }
    },
    by_level = {
        [11] = { ["on_hit.1.control_dur_ms"] = 11000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 12000 }
    }
}

E["CS_jh_yfjb07"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制）" }
}

E["CS_jh_yfjb07_hide"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制）" }
}

E["CS_jh_yfjb08"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 8 },
        can_move = false,
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch",
            per_hit = true
        }
    }
}

E["CS_jh_ymcxs01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    }
}

E["CS_jh_ymcxs02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "若自身存在移动速度降低的负面效果，会使目标6秒内每次使用轻功额外消耗4点轻功值" }
}

E["CS_jh_ymcxs03"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_ymcxs04"] = {
    on_hit = {
        {
            apply_buff = "一层续魄",
            apply_life_ms = 15000,
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "续魄叠满5层时可再次施展该招式，提高自身5%外功招架忽视，效果持续7秒，同时可清除自身大部分负面" }
}

E["CS_jh_ymcxs05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "silence"
        }
    }
}

E["CS_jh_ymcxs06"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_yqq02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_yqq03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_yqq04"] = {
    on_parry_break = {
        {
            apply_buff = "状态",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_yqq05"] = {

}

E["CS_jh_yqq06"] = {
    on_hit = {
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jh_yrst01"] = {
    on_hit = {
        {
            apply_buff = "一层最低等级的“撩火",
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 15000,
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_yrst02"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_yrst03"] = {
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_yrst04"] = {
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_yrst05"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_yrst06"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_yrst07"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_parry = {
        {
            apply_buff = "buf_CS_jh_yrst07_01",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_parry.1.apply_life_ms"] = 6000 },
        [3] = { ["on_parry.1.apply_life_ms"] = 7000 },
        [4] = { ["on_parry.1.apply_life_ms"] = 8000 },
        [5] = { ["on_parry.1.apply_life_ms"] = 9000 },
        [6] = { ["on_parry.1.apply_life_ms"] = 10000 },
        [7] = { ["on_parry.1.apply_life_ms"] = 10000 },
        [8] = { ["on_parry.1.apply_life_ms"] = 11000 },
        [9] = { ["on_parry.1.apply_life_ms"] = 13000 },
        [10] = { ["on_parry.1.apply_life_ms"] = 14000 },
        [11] = { ["on_parry.1.apply_life_ms"] = 15000 },
        [12] = { ["on_parry.1.apply_life_ms"] = 15000 }
    }
}

E["CS_jh_yss01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    }
}

E["CS_jh_yss02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "素练横空溅腐骨，毒涏蚀甲透岩层" }
}

E["CS_jh_yss03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_yss04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "dizzy"
        }
    }
}

E["CS_jh_yss05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "root"
        }
    },
    unparsed = { "若自身存在“白煞”状态，则未招架的敌人将受到冰髓雾的蛇缚效果" }
}

E["CS_jh_yss06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_yss07"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    }
}

E["CS_jh_yss08"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_yss08_b",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            apply_buff = "buf_CS_jh_yss08_q",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [3] = { ["on_hit.1.apply_life_ms"] = 11000, ["on_hit.2.apply_life_ms"] = 11000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 11000, ["on_hit.2.apply_life_ms"] = 11000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 12000, ["on_hit.2.apply_life_ms"] = 12000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 12000, ["on_hit.2.apply_life_ms"] = 12000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 13000, ["on_hit.2.apply_life_ms"] = 13000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 13000, ["on_hit.2.apply_life_ms"] = 13000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 14000, ["on_hit.2.apply_life_ms"] = 14000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 14000, ["on_hit.2.apply_life_ms"] = 14000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 15000, ["on_hit.2.apply_life_ms"] = 15000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 15000, ["on_hit.2.apply_life_ms"] = 15000 },
        [13] = { ["on_hit.1.apply_life_ms"] = 15000, ["on_hit.2.apply_life_ms"] = 15000 }
    }
}

E["CS_jh_yszd02"] = {
    unparsed = { "尝试安抚野兽，有几率使野兽变得安静" }
}

E["CS_jh_yszd03"] = {
    unparsed = { "（只有圣兽亲密度排行榜第一名的驯兽大师才有可能召唤成功" }
}

E["CS_jh_yxfj01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_yxfj02"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_yxfj02_01",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_yxfj03"] = {
    unparsed = { "踏雪泥——拔剑飞鸿斩，自身进入拔剑状态：处于招架状态被敌人攻击时，可拆解敌招，并反击敌人，状态可", "（防反招式，有效范围20米，对霸体无效，可被虚招克制）（拔剑状态下，自身招架时无法移动)" }
}

E["CS_jh_yxfj04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_yxfj04_hide"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_yxfj05"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_yxfj06"] = {
    on_parry_break = {
        {
            control_dur_ms = 3000,
            control_kind = "dizzy"
        }
    },
    unparsed = { "自身处于拔剑状态时，变换出招方式为：突进至鼠标点选范围内破防", "（鼠标点选最大距离为20米）" }
}

E["CS_jh_yxfj06_hide"] = {
    on_parry_break = {
        {
            control_dur_ms = 3000,
            control_kind = "dizzy"
        }
    },
    unparsed = { "自身处于拔剑状态时，变换出招方式为：突进至鼠标点选范围内破防", "（鼠标点选最大距离为20米）" }
}

E["CS_jh_yxfj07"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    self_state = {
        {
            apply_buff = "buf_CS_jh_yxfj07",
            apply_resolved = true
        }
    }
}

E["CS_jh_yybtbf01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_yybtbf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_yybtbf03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_yybtbf04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_yybtbf06"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown",
            per_hit = true
        }
    },
    self_state = {
        {
            apply_buff = "叹西峡",
            apply_resolved = false
        }
    }
}

E["CS_jh_yybtbf06_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown",
            per_hit = true
        }
    }
}

E["CS_jh_yybtbf07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    }
}

E["CS_jh_yybtbf08"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "coma",
            per_hit = true
        }
    }
}

E["CS_jh_yybtbf08_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_yydf01"] = {
    unparsed = { "【“仁者无敌” ：招式变为冲锋招式，伤害更高】" }
}

E["CS_jh_yydf01_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_yydf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_yydf02_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_yydf03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 20 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "coma"
        },
        {
            control_dur_ms = 3000,
            control_kind = "coma"
        }
    }
}

E["CS_jh_yydf03_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 20 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "coma"
        }
    }
}

E["CS_jh_yydf04"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "red",
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_yydf04_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "red",
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_yydf05"] = {
    compile = {
        cast_armor = "yellow"
    },
    unparsed = { "鸳刀扑朔，鸯刀迷离，双刀合一，进入“仁者无敌” 之境：增强本套路其他招式" }
}

E["CS_jh_yydf06"] = {
    self_state = {
        {
            apply_buff = "buf_CS_jh_yydf06",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式功力，增加反击伤害）【“仁者无敌” ：招式伤害增加，防反有效范围变为25米】" }
}

E["CS_jh_yydf06_hide"] = {
    self_state = {
        {
            apply_buff = "buf_CS_jh_yydf06_hide",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围25米，对霸体无效，可被虚招克制" }
}

E["CS_jh_yydf06_hide_1"] = {
    self_state = {
        {
            apply_buff = "buf_CS_jh_yydf06",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围25米，对霸体无效，可被虚招克制" }
}

E["CS_jh_yydf07"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    },
    on_parry_break = {
        {
            control_dur_ms = 5000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_yydf07_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    },
    on_parry_break = {
        {
            control_dur_ms = 5000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jh_yydld01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_yydld01_1",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            apply_buff = "buf_CS_jh_yydld01_2",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_yydld02"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_yydld02_hide"] = {
    self_state = {
        {
            apply_buff = "buf_CS_jh_yydld02_hide",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_jh_yydld03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_yydld04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_yydld04",
            apply_life_ms = 8000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        },
        {
            apply_buff = "buf_CS_jh_yydld04_1",
            apply_life_ms = 8000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_jh_yydld05"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_yydld05_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_yydld05_hide",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_jh_yydld06"] = {
    on_hit = {
        {
            apply_buff = "灼日状态",
            apply_resolved = false,
            apply_stacks = 1
        },
        {
            apply_buff = "飞鸿状态",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_yylcd01"] = {
    unparsed = { "右手切换使用时使自己恢复2%的内力" }
}

E["CS_jh_yylcd02"] = {
    on_parry_break = {
        {
            control_within_ms = 3000
        }
    }
}

E["CS_jh_yylcd03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "silence"
        }
    }
}

E["CS_jh_yylcd04"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_yylcd05"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "stun"
        }
    }
}

E["CS_jh_yylcd06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_yylcd06_01",
            apply_life_ms = 8000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 12000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 14000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 16000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 20000 }
    }
}

E["CS_jh_yywd01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_yywd04",
            apply_life_ms = 8000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_yywd02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_yywd03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_yywd04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_yywd04",
            apply_life_ms = 8000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jh_yywd05"] = {
    compile = {
        can_move = false,
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_yywd06"] = {
    on_parry_break = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jh_yywd07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_yywd04",
            apply_life_ms = 8000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 2000,
            control_kind = "coma",
            per_hit = true
        }
    }
}

E["CS_jh_zf01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_zf01_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_zf02"] = {
    unparsed = { "处于禁地中时，此招式亦将为选定目标范围内的所有队友5秒内持续回复怒气及内力" }
}

E["CS_jh_zf03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 7 }
    },
    by_level = {
        [12] = { ["compile.aoe_radius_m"] = 20, ["on_hit.1.control_dur_ms"] = 5000, ["on_hit.1.displacement_distance_m"] = 0 }
    }
}

E["CS_jh_zf03_1"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jh_zf03_2"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    }
}

E["CS_jh_zf04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_zf04_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_zf05"] = {
    unparsed = { "格挡成功后使自身5秒内使用【醉赋】招式回复部分气血(升级招式可提高气血回复量)" }
}

E["CS_jh_zf07"] = {
    unparsed = { "对选中范围内的目标回复120点气血" }
}

E["CS_jh_zq01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_zq01",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            apply_buff = "一层",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_jh_zq02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "dizzy"
        }
    },
    self_state = {
        {
            apply_buff = "buf_CS_jh_zq02",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（反击有效范围5米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）移除恢复淤血效果" }
}

E["CS_jh_zq02_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "黄霸体反击目标",
            apply_resolved = false,
            apply_stacks = 1,
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_zq02_hide1"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "黄霸体反击目标",
            apply_resolved = false,
            apply_stacks = 1,
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_zq02_hide2"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "黄霸体反击目标",
            apply_resolved = false,
            apply_stacks = 1,
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_zq02_hide3"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "黄霸体反击目标",
            apply_resolved = false,
            apply_stacks = 1,
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_zq02_hide4"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "黄霸体反击目标",
            apply_resolved = false,
            apply_stacks = 1,
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_zq03"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_zq03",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            control_kind = "launch"
        },
        {
            apply_buff = "一层",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "招式范围增加至10米（采和拦腰状态的级别，与架招【采和拦腰】功力有关）（激活此效果需与巫墟宗派深" }
}

E["CS_jh_zq03_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_zq03",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            control_kind = "launch"
        },
        {
            apply_buff = "一层",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "招式范围增加至10米（采和拦腰状态的级别，与架招【采和拦腰】功力有关）（激活此效果需与巫墟宗派深" }
}

E["CS_jh_zq04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_zq04",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 5000,
            control_kind = "launch"
        }
    },
    unparsed = { "（自身每存在一层“采和拦腰”，该状态持续时间延长1秒）招式调整为单体目标，出招更加迅捷（激活此效" }
}

E["CS_jh_zq04_hide"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_zq04",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 5000,
            control_kind = "launch"
        }
    },
    unparsed = { "（自身每存在一层“采和拦腰”，该状态持续时间延长1秒）（激活此效果需与巫墟宗派深处山鬼地窟的巫莎" }
}

E["CS_jh_zq05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_zq05",
            apply_life_ms = 8000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 11000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 12000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 13000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 14000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 15000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 16000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 17000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 18000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 19000 }
    },
    unparsed = { "（自身存在“仙姑献酒”时无法获得“采和拦腰”状态，存在该架招状态时可使自身在倒地休憩时额外恢复内" }
}

E["CS_jh_zq05_hide"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_zq05",
            apply_life_ms = 8000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "（自身存在“仙姑献酒”时无法获得“采和拦腰”状态，存在该架招状态时可使自身在倒地休憩时额外恢复内" }
}

E["CS_jh_zq06"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_zq06_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_zq07"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            chance = 0.1,
            condition = { unparried = true },
            control_kind = "knockdown"
        },
        {
            control_kind = "knockdown"
        },
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jh_zq08"] = {
    unparsed = { "弹腰献酒，一醉方休，自身获得“仙姑献酒”状态：移动迟缓，提升闪避几率，持续8秒", "酒量为小，则移动速度降低20%，闪避几率提升5%", "酒量为中，则移动速度降低15%，闪避几率提升10%", "酒量为大，则移动速度降低10%，闪避几率提升15%", "酒量为海量，则移动速度降低5%，闪避几率提升20%", "）成功闪避对手技能则自动清除该状态，并获得3层“采和拦腰”状态" }
}

E["CS_jh_zq08_hide"] = {
    unparsed = { "弹腰献酒，一醉方休，自身获得“仙姑献酒”状态：移动迟缓，持续恢复淤血效果，持续8秒", "受到攻击时，将恢复双倍的淤血并为攻击者增加“醉酒”状态：招式伤害降低5%，持续10秒，最多可叠加", "）成功释放后可直接获得3层“采和拦腰”状态", "（采和拦腰状态的级别，与架招【采和拦腰】功力有关，未开启架招则无法获得3层采和拦腰状态）（激活此" }
}

E["CS_jh_ztj01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    }
}

E["CS_jh_ztj02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jh_ztj03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jh_ztj04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    unparsed = { "若自身存在3层及以上“魁杓”状态，招式命中几率大幅提升" }
}

E["CS_jh_ztj05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    },
    unparsed = { "若自身存在3层及以上“魁杓”状态，可消耗3层“魁杓”变更为击晕未招架的敌人" }
}

E["CS_jh_ztj06"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    unparsed = { "破防成功时，使其缴械3秒" }
}

E["CS_jh_ztj07"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jh_ztj07_1",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 11000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 12000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 13000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 14000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 15000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 16000 }
    }
}

E["CS_jh_ztj08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 },
        cast_armor = "yellow"
    },
    unparsed = { "每存在一层“魁杓”状态，造成的伤害提升5%" }
}

E["CS_jhgp_gxfhj01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jhgp_gxfhj01",
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.3
        }
    }
}

E["CS_jhgp_gxfhj02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jhgp_gxfhj01",
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.1
        }
    }
}

E["CS_jhgp_gxfhj03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jhgp_gxfhj03",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "root"
        },
        {
            control_dur_ms = 3000,
            control_kind = "coma"
        }
    }
}

E["CS_jhgp_gxfhj04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 20 },
        cast_armor = "red"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jhgp_gxfhj04",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jhgp_gxfhj05"] = {
    compile = {
        cast_armor = "red"
    }
}

E["CS_jhgp_gxfhj06"] = {
    unparsed = { "破防成功时，给目标加“花袭”状态（眼花缭乱，下次攻击有较大几率被闪避）和“花香”状态（长时间不可" }
}

E["CS_jhgp_gxfhj07"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jhgp_gxfhj07_1",
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.1,
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    by_level = {
        [3] = { ["on_hit.1.chance"] = 0.15 },
        [4] = { ["on_hit.1.chance"] = 0.15 },
        [5] = { ["on_hit.1.chance"] = 0.2 },
        [6] = { ["on_hit.1.chance"] = 0.2 },
        [7] = { ["on_hit.1.chance"] = 0.25 },
        [8] = { ["on_hit.1.chance"] = 0.25 },
        [9] = { ["on_hit.1.chance"] = 0.3 },
        [10] = { ["on_hit.1.chance"] = 0.3 }
    }
}

E["CS_jhgp_wjj01"] = {

}

E["CS_jhgp_wjj02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "一层星图状态",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_jhgp_wjj03"] = {
    unparsed = { "自身持续获得星图状态，此状态下若自身存在10层星图可消耗所有星图状态，释放【天工合卷·变】" }
}

E["CS_jhgp_wjj04"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "root"
        }
    },
    by_level = {
        [9] = { ["on_hit.1.apply_stacks"] = 1, ["on_hit.1.control_dur_ms"] = 5000 },
        [10] = { ["on_hit.1.apply_stacks"] = 1, ["on_hit.1.control_dur_ms"] = 5000 },
        [11] = { ["on_hit.1.apply_stacks"] = 1, ["on_hit.1.control_dur_ms"] = 5000 },
        [12] = { ["on_hit.1.apply_stacks"] = 1, ["on_hit.1.control_dur_ms"] = 5000 }
    }
}

E["CS_jhgp_wjj05"] = {
    on_hit = {
        {
            apply_buff = "一层星图状态",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    },
    by_level = {
        [8] = { ["on_hit.2.control_dur_ms"] = 15000 },
        [9] = { ["on_hit.2.control_dur_ms"] = 15000 },
        [10] = { ["on_hit.2.control_dur_ms"] = 15000 },
        [11] = { ["on_hit.2.control_dur_ms"] = 15000 },
        [12] = { ["on_hit.2.control_dur_ms"] = 15000 }
    }
}

E["CS_jhgp_wjj06"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "stun"
        }
    },
    by_level = {
        [9] = { ["on_hit.1.apply_stacks"] = 1, ["on_hit.2.control_dur_ms"] = 2000 },
        [10] = { ["on_hit.1.apply_stacks"] = 1, ["on_hit.2.control_dur_ms"] = 2000 },
        [11] = { ["on_hit.1.apply_stacks"] = 1, ["on_hit.2.control_dur_ms"] = 2000 },
        [12] = { ["on_hit.1.apply_stacks"] = 1, ["on_hit.2.control_dur_ms"] = 2000 }
    },
    unparsed = { "招式出手时，每层星图状态额外提高招式命中率，10层时招式会心释放-此星愈胜，此剑愈烈-（游戏时辰" }
}

E["CS_jhgp_wjj07"] = {
    on_hit = {
        {
            apply_buff = "一层星图状态",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "每层星图状态提升自身1%护盾穿透，降低【天工合卷】怒气消耗2点" }
}

E["CS_jhgp_wjj08"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "一层星图状态",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_jhgp_wjj09"] = {
    compile = {
        cast_armor = "red",
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "一层星图状态",
            apply_resolved = false,
            apply_stacks = 1,
            chance = 0.4,
            condition = { unparried = true },
            per_hit = true
        }
    }
}

E["CS_jhgp_wjj09_hide"] = {
    compile = {
        cast_armor = "red"
    },
    by_level = {
        [12] = { ["on_hit.1.apply_stacks"] = 1 }
    }
}

E["CS_jl_bmjs01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong",
            control_within_ms = 3000
        }
    }
}

E["CS_jl_bmjs02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jl_bmjs03"] = {
    unparsed = { "点选区域划出刀气，对选定半径5米区域内目标造成{@0:伤害包}，并使其摔倒" }
}

E["CS_jl_bmjs04"] = {
    unparsed = { "跳步悬空斩，对身前范围5米内的敌人造成{@0:伤害包}" }
}

E["CS_jl_bmjs05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jl_bmjs05",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "并使【开天辟地】出招时为黄霸体" }
}

E["CS_jl_bmjs06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jl_bmjs06",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "每击消耗1层，共可叠加5层" }
}

E["CS_jl_bmjs07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    },
    unparsed = { "若自身存在“生肉枯骨”，则出招时为黄霸体", "稚元:再次施展时，若自身存在“生肉枯骨”，招式命中目标后额外多回复10%淤血" }
}

E["CS_jl_bmjs07_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jl_bmjs07",
            apply_life_ms = 3000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    },
    unparsed = { "若自身存在“生肉枯骨”，则出招时为黄霸体招式命中目标后回复10%淤血", "稚元:命中敌人时，若自身存在“生肉枯骨”，招式命中目标后额外多回复10%淤血" }
}

E["CS_jl_dmssj02"] = {
    on_hit = {
        {
            control_kind = "stun"
        }
    }
}

E["CS_jl_dmssj03"] = {
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jl_dmssj04"] = {
    unparsed = { "防守架势，收敛剑身，每次招架成功时，有20%几率减少攻击者2点怒气值", "自身同时激活所有的【噬魂】武技及【森罗】武技，则格挡成功时可获得一层“森罗”状态" }
}

E["CS_jl_dmssj05"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jl_dmssj06"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "silence"
        }
    },
    unparsed = { "招式最后一击可将未招架敌人挑飞" }
}

E["CS_jl_dmssj07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "stun"
        }
    },
    unparsed = { "（出招若被目标招架或躲闪，则下次发动招式冷却时间减半）" }
}

E["CS_jl_dmssj08"] = {
    on_hit = {
        {
            control_kind = "root"
        }
    }
}

E["CS_jl_fgz01"] = {
    unparsed = { "自身同时激活所有的【噬魂】武技及【森罗】武技，则招式命中未招架敌人时可获得一层“森罗”状态" }
}

E["CS_jl_fgz02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jl_fgz02",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jl_fgz03"] = {
    unparsed = { "破防成功后，给敌人注入毒素“捣髓”：内力不断流失", "使用招式时，根据消耗内力值受到等同伤害，使用招式亦可催动毒素发作，刷新毒素持续时间" }
}

E["CS_jl_fgz03_hide"] = {
    unparsed = { "破防成功后，给敌人注入毒素“捣髓·远”：内力不断流失", "使用招式时，根据消耗内力值受到等同伤害，使用招式亦可催动毒素发作，刷新毒素持续时间" }
}

E["CS_jl_fgz04"] = {
    unparsed = { "防御架势，以毒护心，招架成功时，会令攻击者陷入“蚀心”状态：每秒2点毒性伤害，持续7秒，该效果可" }
}

E["CS_jl_fgz06"] = {
    unparsed = { "腐骨秘技，向点选区域投掷毒瓶(所有由【蚀骨摧心掌】生成的毒瓶至多只能存在一个)，爆发奇异毒雾：敌" }
}

E["CS_jl_fgz07"] = {
    unparsed = { "可击落空中目标，使其陷入“骨折”状态：移动迟缓，使用轻功时会遭受伤害并加重伤势，增加“骨折”的时" }
}

E["CS_jl_fgz08"] = {
    unparsed = { "腐骨秘技，向自身脚下投掷毒瓶，爆发奇异烟雾保护自身(所有由【蚀骨摧心掌】生成的毒瓶至多只能存在一", "处于烟雾中时，受到伤害时将有概率重新获得护身效果（使8秒内下一击受到的伤害降低30%）此招式与【" }
}

E["CS_jl_shuangci01"] = {
    on_hit = {
        {
            mp_drain = true,
            mp_drain_dur_ms = 0
        },
        {
            mp_drain = true,
            mp_drain_dur_ms = 0
        }
    }
}

E["CS_jl_shuangci02"] = {
    unparsed = { "释放玄毒激发潜能,提升刺法威力，同时毒损导致自身降低200点气血上限值，可叠加3次" }
}

E["CS_jl_shuangci03"] = {
    unparsed = { "周边被恶蛊波及贴附的敌人，使用轻功时会不断损失气血" }
}

E["CS_jl_shuangci04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jl_shuangci06"] = {

}

E["CS_jl_shuangci07"] = {
    unparsed = { "并使在施放【阴魂出窍】时不再有怜悯之心，获得“霸体”效果" }
}

E["CS_jl_shuangci08"] = {
    unparsed = { "以恶蛊附于武器之上，每次招架敌人攻击，都会汲取其2点鲜血（汲取值受内功威力影响）" }
}

E["CS_jl_zfd02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jl_zfd04"] = {
    unparsed = { "妄图招架之敌，会被气劲“震伤”并暂时无力出招" }
}

E["CS_jl_zfd05"] = {
    unparsed = { "防御架势，招架成功时可激发潜能，使自身攻击附加7点刚伤害（伤害受内功影响）" }
}

E["CS_jl_zfd07"] = {
    unparsed = { "空中逐电：气劲如电，切击点选的区域，伤害更高，同时使“震伤”的敌人暂时无法出招，并遭受额外刚性伤" }
}

E["CS_jlqj_zfd02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jlqj_zfd04"] = {
    unparsed = { "妄图招架之敌，会被气劲“震伤”并暂时无力出招" }
}

E["CS_jlqj_zfd05"] = {
    unparsed = { "防御架势，招架成功时可激发潜能，使自身攻击附加14点刚伤害（伤害受内功影响）" }
}

E["CS_jlqj_zfd06"] = {
    compile = {
        cast_armor = "red"
    }
}

E["CS_jlqj_zfd07"] = {
    unparsed = { "空中逐电：气劲如电，切击点选的区域，伤害更高，同时使“震伤”的敌人暂时无法出招，并遭受额外刚性伤" }
}

E["CS_jw_yrz01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 10 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_jw_yrz02"] = {
    unparsed = { "并使妄图招架之敌倒地（装备薙刀发挥100%招式威力，装备长棍只能发挥70%招式威力）" }
}

E["CS_jw_yrz04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jw_yrz05_hide"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_jw_yrz07"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "pull"
        },
        {
            condition = { unparryable = true },
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    },
    unparsed = { "突击目标造成{@0:伤害包}，若目标未招架则击晕其3秒" }
}

E["CS_jw_yrz08"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "red",
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "root"
        },
        {
            condition = { unparried = true },
            control_kind = "dizzy",
            per_hit = true
        }
    }
}

E["CS_jy_dqd03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "若目标处于破防不可招架状态，则对其造成大量刚属性伤害" }
}

E["CS_jy_dqd04"] = {
    on_parry_break = {
        {
            control_kind = "root"
        }
    }
}

E["CS_jy_dqd05"] = {
    unparsed = { "无情架势，招架成功时，向攻击者传送一道“断情”刀劲，可叠加5层：每次被断情招式攻击，都会触发一道" }
}

E["CS_jy_dqd06"] = {
    unparsed = { "使目标身体摇摇欲坠：发动攻击或使用轻功都有很大几率跌倒，在空中则会直接坠落" }
}

E["CS_jy_dqd07"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jy_jgd01"] = {
    compile = {
        max_hits = 3
    }
}

E["CS_jy_jgd02"] = {
    unparsed = { "两连击，每击造成{@0:伤害包}" }
}

E["CS_jy_jgd03"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jy_jgd04"] = {

}

E["CS_jy_jgd06"] = {
    unparsed = { "使自身释放技能所消耗内力增加，但同时技能伤害也增加，效果持续一段时间" }
}

E["CS_jy_jgd07"] = {
    unparsed = { "华丽的单体输出，每击造成{@0:伤害包}" }
}

E["CS_jy_shd01"] = {
    unparsed = { "凝内力于刀锋，潇洒一击，气劲击中的敌人受到{@0:伤害包}" }
}

E["CS_jy_shd03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jy_shd04"] = {
    unparsed = { "对方妄图招架只会被击破，并受到失魂刀法刀伤，在一段时间内持续受到伤害" }
}

E["CS_jy_shd05"] = {
    unparsed = { "暗藏杀机，招架时趁机反伤敌方，使其一段时间内持续受到伤害（伤害受近身威力影响）" }
}

E["CS_jy_shd07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "失魂刀法刀伤",
            apply_resolved = false,
            apply_stacks = 1,
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jy_shd07_sky"] = {
    on_hit = {
        {
            apply_buff = "失魂刀法刀伤",
            apply_resolved = false,
            apply_stacks = 1,
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_jy_tdshs01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jy_tdshs01",
            apply_life_ms = 7000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 7000,
            control_kind = "slow"
        }
    }
}

E["CS_jy_tdshs03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "root"
        },
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_jy_tdshs04"] = {
    on_parry_break = {
        {
            control_dur_ms = 2000,
            control_kind = "knockdown"
        },
        {
            control_dur_ms = 2000,
            control_kind = "knockdown"
        }
    }
}

E["CS_jy_tdshs05"] = {
    unparsed = { "飞索将会缠绕未招架的敌人，成功缠绕后可施展招式【疏而不漏】：将被缠绕的敌人扯近后，甩飞至点选区域", "成功甩飞目标后，处于\"经天纬地\"状态且自身同时激活所有的【千魂】武技及【行刑】武技，则移除【血洗" }
}

E["CS_jy_tdshs05_hide"] = {
    unparsed = { "成功甩飞目标后，处于\"经天纬地\"状态且自身同时激活所有的【千魂】武技及【行刑】武技，则移除【血洗" }
}

E["CS_jy_tdshs05_hide_1"] = {
    unparsed = { "成功甩飞目标后，处于\"经天纬地\"状态且自身同时激活所有的【千魂】武技及【行刑】武技，则移除【血洗" }
}

E["CS_jy_tdshs06"] = {
    compile = {
        cast_armor = "yellow"
    },
    self_state = {
        {
            apply_buff = "buf_CS_jy_tdshs06",
            apply_life_ms = 8000,
            apply_resolved = true
        }
    },
    by_level = {
        [2] = { ["self_state.1.apply_life_ms"] = 9000 },
        [3] = { ["self_state.1.apply_life_ms"] = 10000 },
        [4] = { ["self_state.1.apply_life_ms"] = 11000 },
        [5] = { ["self_state.1.apply_life_ms"] = 12000 },
        [6] = { ["self_state.1.apply_life_ms"] = 13000 },
        [7] = { ["self_state.1.apply_life_ms"] = 14000 },
        [8] = { ["self_state.1.apply_life_ms"] = 15000 },
        [9] = { ["self_state.1.apply_life_ms"] = 16000 },
        [10] = { ["self_state.1.apply_life_ms"] = 17000 }
    }
}

E["CS_jy_xsd01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "red"
    },
    on_hit = {
        {
            apply_buff = "一层“血战八方",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_jy_xsd02"] = {
    on_hit = {
        {
            apply_buff = "一层“血战八方",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    },
    unparsed = { "若招式被招架，则无法利用飞爪将自己牵引至目标身边" }
}

E["CS_jy_xsd03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jy_xsd04"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jy_xsd04_mark08",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 10000,
            control_kind = "silence"
        }
    }
}

E["CS_jy_xsd05"] = {
    on_hit = {
        {
            apply_buff = "buf_shp_2024621_029",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jy_xsd05_hide"] = {
    on_hit = {
        {
            apply_buff = "buf_shp_2024621_029",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jy_xsd06"] = {
    unparsed = { "睚眦必报，血债血偿，每次格挡成功会获得1层“贪婪欲念”（可叠加3层）：使本套路攻击附加20%吸血", "格挡成功时会额外获得一层“血战八方”状态：可叠加10层持续30秒" }
}

E["CS_jy_xsd08"] = {
    unparsed = { "同时汲取敌人鲜血恢复自身伤势，吸血效果20%（攻击被招架则无吸血效果）" }
}

E["CS_jy_yzq02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jy_yzq03"] = {
    unparsed = { "鹰爪凶残心法，使贪婪之心附加于攻击，产生“啄目争珠”效果：被击中者会气血外泄，减速，使用轻功有几" }
}

E["CS_jy_yzq04"] = {
    on_hit = {
        {
            control_kind = "slow"
        }
    }
}

E["CS_jy_yzq05"] = {

}

E["CS_jy_yzq06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jy_yzq07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jy_yzq07_sky"] = {
    unparsed = { "在空中冲刺向目标所在的区域，撕裂范围内的单位，造成{@0:伤害包}且附加“啄目争珠”相同的效果" }
}

E["CS_jy_yzq08"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "冲锋招式，只针对末路穷途的倒地目标，绝望抓撕对其造成{@0:伤害包}" }
}

E["CS_jyqj_yzq02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jyqj_yzq03"] = {
    unparsed = { "鹰爪凶残心法，使贪婪之心附加于攻击，产生“啄目争珠”效果：被击中者会气血外泄，减速，使用轻功有几" }
}

E["CS_jyqj_yzq04"] = {
    on_hit = {
        {
            control_kind = "slow"
        }
    }
}

E["CS_jyqj_yzq05"] = {

}

E["CS_jyqj_yzq06"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "red"
    }
}

E["CS_jyqj_yzq07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jyqj_yzq07_sky"] = {
    unparsed = { "在空中冲刺向目标所在的区域，撕裂范围内的单位，造成{@0:伤害包}且附加“啄目争珠”相同的效果" }
}

E["CS_jyqj_yzq08"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "冲锋招式，只针对末路穷途的倒地目标，绝望抓撕对其造成{@0:伤害包}" }
}

E["CS_jz_lyfh01"] = {
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jz_lyfh01_a"] = {
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jz_lyfh02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "当目标下盘不稳时，可将其挑飞到空中" }
}

E["CS_jz_lyfh02_a"] = {
    unparsed = { "对目标造成{@0:伤害包}，当目标下盘不稳时，可以用剑将目标挑飞到空中;此招式可以利用轻功快速地" }
}

E["CS_jz_lyfh03"] = {
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jz_lyfh03_a"] = {
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jz_lyfh04"] = {
    on_hit = {
        {
            chance = 0.25,
            control_dur_ms = 5000,
            control_kind = "root"
        }
    }
}

E["CS_jz_lyfh05"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jz_lyfh06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jh_hq02_01",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 10000,
            control_kind = "stiffen"
        }
    },
    unparsed = { "飞花意境，皓雪飘零，周身创出内劲气场：增强自身外功威力、内功威力、气血恢复能力" }
}

E["CS_jz_lyfh07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jz_wyjf01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_tyhg_yszyq01_1",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_jz_wyjf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jz_wyjf03"] = {
    unparsed = { "（此心法需要内力维持，每秒消耗1%内力）" }
}

E["CS_jz_wyjf04"] = {

}

E["CS_jz_wyjf05"] = {
    unparsed = { "防御架势，化解敌方攻势之时，快速恢复自身内力（50%几率，每秒2%内力恢复，持续4s）" }
}

E["CS_jz_wyjf06"] = {
    unparsed = { "无涯高深心法，御剑一怒而发，产生雄浑无涯气场：使自身和受影响的队友大幅提升柔伤害，并可快速恢复内" }
}

E["CS_jz_wyjf07"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_jz_xytf01"] = {
    unparsed = { "三路快攻，叠加至敌，对前方敌人造成{@0:伤害包}，配合“逍遥身法”可发挥更大威力" }
}

E["CS_jz_xytf01x"] = {
    compile = {
        max_hits = 3
    }
}

E["CS_jz_xytf01y"] = {
    compile = {
        max_hits = 3
    }
}

E["CS_jz_xytf01y_1"] = {
    compile = {
        max_hits = 3
    }
}

E["CS_jz_xytf01y_2"] = {
    compile = {
        max_hits = 3
    }
}

E["CS_jz_xytf01y_3"] = {
    compile = {
        max_hits = 3
    }
}

E["CS_jz_xytf01z"] = {
    compile = {
        max_hits = 3
    }
}

E["CS_jz_xytf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jz_xytf02_sky"] = {
    unparsed = { "凌厉的腿法连续攻向敌人2次造成{@0:伤害包},配合逍遥身法可以发挥出更大的威力;此招式可以利用" }
}

E["CS_jz_xytf04"] = {
    on_parry_break = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jz_xytf04_sky"] = {
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_jz_xytf05"] = {
    unparsed = { "防御架势，招架成功时会激发腿部潜能“逍遥身法”（提高自身移动速度和7点阴伤害，提升整个逍遥腿法的" }
}

E["CS_jz_xytf07"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "若自身存在“逍遥身法”，凌空攻击空中目标时，将大幅提升招式伤害倍增的几率" }
}

E["CS_jz_xytf07_sky"] = {
    unparsed = { "连续快速的踢向目标造成{@0:伤害包}，当自身格挡成功激活逍遥身法之后，若自己和敌人都在空中时攻" }
}

E["CS_jz_yxjf01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jz_yxjf01",
            apply_resolved = true,
            apply_stacks = 1,
            control_kind = "slow"
        }
    }
}

E["CS_jz_yxjf01_hide"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jz_yxjf01",
            apply_resolved = true,
            apply_stacks = 1,
            control_kind = "slow"
        }
    }
}

E["CS_jz_yxjf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jz_yxjf03"] = {
    on_hit = {
        {
            control_dur_ms = 1000,
            control_kind = "stun"
        }
    }
}

E["CS_jz_yxjf04"] = {
    on_hit = {
        {
            control_kind = "root"
        }
    }
}

E["CS_jz_yxjf05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jz_yxjf05_01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jz_yxjf07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    unparsed = { "音波扰敌耳聪，使其进入“迷魂”状态：攻击力下降、按键效果发生错乱（上下左右颠倒），空中之敌会失控" }
}

E["CS_jz_yydbf01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jz_yydbf01",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 5000,
            control_kind = "slow"
        }
    },
    unparsed = { "自身存在“宫”时，招式可击落空中目标" }
}

E["CS_jz_yydbf02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 }
    },
    unparsed = { "（装备箫发挥100%招式威力，装备单剑只能发挥70%招式威力）稚元:破防成功时，目标受到的僵直效" }
}

E["CS_jz_yydbf03"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jz_yydbf03",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_jz_yydbf04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_jz_yydbf03",
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.22
        }
    },
    by_level = {
        [2] = { ["on_hit.1.chance"] = 0.24 },
        [3] = { ["on_hit.1.chance"] = 0.26 },
        [4] = { ["on_hit.1.chance"] = 0.28 },
        [5] = { ["on_hit.1.chance"] = 0.3 },
        [6] = { ["on_hit.1.chance"] = 0.32 },
        [7] = { ["on_hit.1.chance"] = 0.34 },
        [8] = { ["on_hit.1.chance"] = 0.36 },
        [9] = { ["on_hit.1.chance"] = 0.38 },
        [10] = { ["on_hit.1.chance"] = 0.4 }
    },
    unparsed = { "“箫音”叠加到9层后则变为“箫音醉”状态" }
}

E["CS_jz_yydbf05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_jz_yydbf05_01",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        },
        {
            control_dur_ms = 2000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jz_yydbf06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 }
    },
    on_hit = {
        {
            control_kind = "launch"
        },
        {
            control_kind = "launch",
            per_hit = true
        }
    }
}

E["CS_jz_yydbf07"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_jz_yydbf09"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        },
        {
            apply_buff = "buf_CS_jz_yydbf05_01",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_jzqj_lyfh01"] = {
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jzqj_lyfh02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "当目标下盘不稳时，可将其挑飞到空中" }
}

E["CS_jzqj_lyfh03"] = {
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jzqj_lyfh04"] = {
    on_hit = {
        {
            chance = 0.25,
            control_dur_ms = 5000,
            control_kind = "root"
        }
    }
}

E["CS_jzqj_lyfh05"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_jzqj_lyfh06"] = {
    unparsed = { "飞花意境，皓雪飘零，周身创出内劲气场：增强自身外功威力、内功威力、气血恢复能力" }
}

E["CS_jzqj_lyfh07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_jzqj_lyfh08"] = {
    compile = {
        cast_armor = "red"
    }
}

E["CS_kf_emqj_lbc01"] = {

}

E["CS_kf_emqj_lbc03"] = {

}

E["CS_kf_emqj_lbc04"] = {

}

E["CS_kf_emqj_lbc06"] = {

}

E["CS_kf_gbqj_lsbf01"] = {

}

E["CS_kf_gbqj_lsbf02"] = {

}

E["CS_kf_gbqj_lsbf03"] = {

}

E["CS_kf_gbqj_lsbf04"] = {

}

E["CS_kf_jlqj_zfd01"] = {

}

E["CS_kf_jlqj_zfd02"] = {

}

E["CS_kf_jlqj_zfd03"] = {

}

E["CS_kf_jlqj_zfd04"] = {

}

E["CS_kf_jlqj_zfd07"] = {

}

E["CS_kf_jyqj_yzq01"] = {

}

E["CS_kf_jyqj_yzq02"] = {

}

E["CS_kf_jyqj_yzq04"] = {

}

E["CS_kf_jyqj_yzq08"] = {

}

E["CS_kf_jzqj_lyfh01"] = {

}

E["CS_kf_jzqj_lyfh02"] = {

}

E["CS_kf_jzqj_lyfh03"] = {

}

E["CS_kf_jzqj_lyfh05"] = {

}

E["CS_kf_slqj_djgq02"] = {

}

E["CS_kf_slqj_djgq03"] = {

}

E["CS_kf_slqj_djgq07"] = {

}

E["CS_kf_slqj_djgq08"] = {

}

E["CS_kf_tmqj_mhb01"] = {

}

E["CS_kf_tmqj_mhb02"] = {

}

E["CS_kf_tmqj_mhb03"] = {

}

E["CS_kf_tmqj_mhb04"] = {

}

E["CS_kf_wdqj_tjj01"] = {

}

E["CS_kf_wdqj_tjj04"] = {

}

E["CS_kf_wdqj_tjj05"] = {

}

E["CS_kf_wdqj_tjj08"] = {

}

E["CS_kl_hqxxp01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            apply_buff = "一层自慎",
            apply_life_ms = 10000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 10000,
            control_kind = "slow"
        },
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_kl_hqxxp02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_kl_hqxxp03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 10 }
    },
    on_parry_break = {
        {
            apply_buff = "远近相安状态",
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "root",
            control_within_ms = 5000
        }
    }
}

E["CS_kl_hqxxp04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_kl_hqxxp05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_kl_hqxxp05_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_kl_hqxxp06"] = {
    unparsed = { "对目标造成{@0:伤害包}，若目标未招架且未处于知音难觅状态下，则将目标与自身互相共鸣，进入10", "肝肠断状态叠加至3层时会额外使其命中率降低100%，持续至10秒内下一次出招", "肝肠断状态叠加至3层或双方距离超过15米，移除双方的知音难觅状态" }
}

E["CS_kl_hqxxp06_hide"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_kl_hqxxp06_hide",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "对队友使用，恢复其一定量气血并将目标与自身互相共鸣，进入10秒的知音难觅（友）状态知音难觅（友）", "琴瑟和鸣状态叠加至3层时，额外使双方10秒内下一次释放昆仑琴类武学怒气招式免受负面状态影响", "琴瑟和鸣状态叠加至3层时或双方距离超过15米，移除双方的知音难觅（友）状态" }
}

E["CS_kl_hqxxp07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_kl_hyjf01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            apply_buff = "一层濯缨",
            apply_life_ms = 15000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 15000,
            control_kind = "slow"
        }
    }
}

E["CS_kl_hyjf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_kl_hyjf03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "五层濯缨致盲",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_kl_hyjf04"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）致盲:丢失目标并被剥夺视野" }
}

E["CS_kl_hyjf04_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）致盲:丢失目标并被剥夺视野" }
}

E["CS_kl_hyjf05"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "一层濯缨",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            control_kind = "knockdown",
            per_hit = true
        }
    }
}

E["CS_kl_hyjf06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_kl_hyjf01",
            apply_resolved = true,
            apply_stacks = 2,
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "silence"
        }
    }
}

E["CS_kl_hyjf07"] = {
    unparsed = { "若目标未招架则根据目标身上的濯缨层数触发额外效果", "一层及以上：出招时能够恢复少量气血，濯缨层数越多，恢复效果越好二层及以上：出招时暴击率提高，濯缨", "消耗的层数越多，致盲时间越长" }
}

E["CS_kl_syl01"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch",
            per_hit = true
        }
    }
}

E["CS_kl_syl01_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_kl_syl02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 },
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_kl_syl03"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_kl_syl03_hide"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_kl_syl04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_kl_syl04_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_kl_syl05"] = {
    on_hit = {
        {
            apply_buff = "一层言愁",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_kl_syl06"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        },
        {
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_kl_syl06_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_kl_syl07"] = {
    unparsed = { "浓墨焕彩状态下额外获得3层登楼", "每层登楼降低招式内力消耗30%，每3秒衰减一层，每次衰减额外获得一层点睛之笔" }
}

E["CS_kl_syl08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        },
        {
            condition = { unparried = true },
            control_dur_ms = 10000,
            control_kind = "launch"
        }
    },
    unparsed = { "同时在脚下画下一条苍龙印记并使自身进入浓墨焕彩状态：此状态下本套路招式效果发生变化每次使用都将累", "5层以上点睛之笔则此次额外伤害无视目标防御）" }
}

E["CS_kl_syl08_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    },
    unparsed = { "5层以上点睛之笔则此次额外伤害无视目标防御）" }
}

E["CS_kl_wltf01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_kl_wltf05_1",
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.15,
            condition = { unparried = true }
        }
    }
}

E["CS_kl_wltf01_sky"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_kl_wltf05_1",
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.15,
            condition = { unparried = true }
        }
    }
}

E["CS_kl_wltf02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_kl_wltf02_sky"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_kl_wltf03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_kl_wltf03_sky"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_kl_wltf04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 10000,
            control_kind = "knockdown"
        }
    }
}

E["CS_kl_wltf04_sky"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 10000,
            control_kind = "knockdown"
        }
    }
}

E["CS_kl_wltf05"] = {
    unparsed = { "格挡成功时积蓄一缕威灵之力持续15秒可叠加5层", "使用本套路【目极千里】成功破防后或本套路【飘摇八极】命中未招架目标后，威灵之力将爆发，每层威灵之" }
}

E["CS_kl_wltf05_sky"] = {
    unparsed = { "格挡成功时积蓄一缕威灵之力持续15秒可叠加5层", "使用本套路【目极千里】成功破防后或本套路【飘摇八极】命中未招架目标后，威灵之力将爆发，每层威灵之" }
}

E["CS_kl_wltf06"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 10 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_kl_wltf06_sky"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 10 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_kl_ymchj01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    },
    on_hit = {
        {
            apply_buff = "月溯效果",
            apply_life_ms = 10000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_kl_ymchj02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "silence"
        }
    }
}

E["CS_kl_ymchj03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_kl_ymchj04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_kl_ymchj05"] = {
    on_hit = {
        {
            apply_buff = "一层浸月",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_kl_ymchj06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_kl_ymchj07"] = {
    on_hit = {
        {
            apply_buff = "一层月溯",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "花月美人状态持续3秒最多叠加3层，招式使用结束时清除" }
}

E["CS_kl_ymchj08"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    }
}

E["CS_lzy_gbqj_lsbf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "冲锋招式，灵动摆击，对目标造成{@0:伤害包}，丐帮宗门弟子攻击身中“灵蛇奇毒”的未招架目标时，" }
}

E["CS_lzy_gbqj_lsbf03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_lzy_jlqj_zfd02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_lzy_jyqj_yzq02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_lzy_jyqj_yzq08"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "冲锋招式，只针对末路穷途的倒地目标，绝望抓撕对其造成{@0:伤害包}" }
}

E["CS_lzy_jzqj_lyfh02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "当目标下盘不稳时，可将其挑飞到空中" }
}

E["CS_lzy_jzqj_lyfh05"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_lzy_sjhrc_mryl002"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_lzy_sjhrc_mryl002_new"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_lzy_sjhrc_mryl003"] = {
    on_hit = {
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_lzy_sjhrc_mryl004"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "root"
        }
    },
    unparsed = { "若未招架的敌人存在“引线”状态，则倒地3秒" }
}

E["CS_lzy_sjhrc_mryl007"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_bg_zlz08_2",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 2000,
            control_kind = "stiffen"
        }
    }
}

E["CS_lzy_sjhrc_mryl008"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_bg_zlz08_1",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_lzy_slqj_djgq07"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_lzy_tmqj_mhb04"] = {
    unparsed = { "妄图抵抗的敌人徒耗更多招架耐力" }
}

E["CS_lzy_tyhgrc_pbxb001"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_tyhg_mwybj01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_lzy_tyhgrc_pbxb002"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_tyhg_mwybj01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_lzy_tyhgrc_pbxb003"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "铁画银钩，对目标造成{@0:伤害包}，若目标存在“采薇”状态且未招架时，该招式额外造成0.3倍伤" }
}

E["CS_lzy_tyhgrc_pbxb004"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_tyhg_mwybj05",
            apply_resolved = true,
            apply_stacks = 1,
            control_kind = "launch"
        }
    }
}

E["CS_lzy_wdqj_tjj01"] = {
    compile = {
        max_hits = 2,
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_lzy_xmgrc_gxtlj01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 10 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "slow"
        }
    }
}

E["CS_lzy_xmgrc_gxtlj02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            control_kind = "root"
        }
    }
}

E["CS_lzy_xmgrc_gxtlj03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    unparsed = { "技能释放完成时剑气倾泻，将仍处于自身周围的未招架敌人远远震开此招式只能在地面使用且出招时无法移动" }
}

E["CS_mj_rylhq01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "ban_qinggong"
        },
        {
            condition = { unparried = true },
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_mj_rylhq01_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "slow"
        }
    }
}

E["CS_mj_rylhq02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_parry_break = {
        {
            control_dur_ms = 1000,
            control_kind = "stiffen"
        }
    }
}

E["CS_mj_rylhq02_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            control_dur_ms = 1000,
            control_kind = "stiffen"
        }
    }
}

E["CS_mj_rylhq03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_mj_rylhq03_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_mj_rylhq04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_mj_rylhq05"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        can_move = false,
        cast_armor = "yellow",
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "launch",
            per_hit = true
        },
        {
            condition = { unparried = true },
            control_kind = "launch",
            per_hit = true
        }
    }
}

E["CS_mj_rylhq05_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        can_move = false,
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "launch",
            per_hit = true
        }
    }
}

E["CS_mj_rylhq06"] = {
    unparsed = { "使用招式消耗断枪可将长枪拆解为“双枪姿态” ，可叠加7层，使用一次招式，消耗一层“双枪姿态” ，" }
}

E["CS_mj_rylhq07"] = {
    unparsed = { "招架成功可获得断枪，可叠加2层再次使用招式可将双枪合为长枪，清除双枪姿态" }
}

E["CS_mj_rylhq08"] = {
    compile = {
        can_move = false,
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "stun"
        }
    }
}

E["CS_mj_rylhq08_hide"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "stun"
        }
    }
}

E["CS_mj_typzd01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_mj_typzd02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    }
}

E["CS_mj_typzd03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "灼伤状态",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    },
    on_parry_break = {
        {
            apply_buff = "灼伤状态",
            apply_life_ms = 12000,
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_mj_typzd04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown",
            per_hit = true
        }
    }
}

E["CS_mj_typzd05"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            apply_buff = "灼伤状态",
            apply_life_ms = 12000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            per_hit = true
        }
    }
}

E["CS_mj_typzd06"] = {
    self_state = {
        {
            apply_buff = "buf_CS_mj_typzd0601",
            apply_life_ms = 12000,
            apply_resolved = true
        }
    }
}

E["CS_mj_typzd07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown",
            per_hit = true
        }
    }
}

E["CS_mr_wxqyr01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_mr_wxqyr02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_mr_wxqyr02",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_mr_wxqyr03"] = {
    unparsed = { "挥臂前突，对前方目标造成伤害{@0:伤害包}，若目标处于倒地状态，则额外造成12%伤害" }
}

E["CS_mr_wxqyr04"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_mr_wxqyr05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_mr_wxqyr05",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_mr_wxqyr06"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_newdataosha_dingzw_001"] = {
    unparsed = { "使用之后立即恢复10%气血" }
}

E["CS_newdataosha_dingzw_002"] = {
    unparsed = { "使用之后立即恢复50%气血" }
}

E["CS_newdataosha_dingzw_003"] = {
    unparsed = { "使用之后立即恢复10%内力" }
}

E["CS_newdataosha_dingzw_004"] = {
    unparsed = { "使用之后立即恢复50%内力" }
}

E["CS_ng_gsd01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_ng_gsd02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_ng_gsd03"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_ng_gsd04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown",
            per_hit = true
        }
    }
}

E["CS_ng_gsd05"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 },
        cast_armor = "yellow",
        air = "never"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "dizzy",
            per_hit = true
        }
    },
    by_level = {
        [6] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [7] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [8] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [9] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [10] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [11] = { ["on_hit.2.control_dur_ms"] = 3000 },
        [12] = { ["on_hit.2.control_dur_ms"] = 3000 }
    }
}

E["CS_ng_gsd05_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    by_level = {
        [6] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [7] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [8] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 3000 }
    }
}

E["CS_ng_gsd06"] = {
    self_state = {
        {
            apply_buff = "buf_CS_ng_gsd06_01",
            apply_life_ms = 30000,
            apply_resolved = true
        }
    },
    unparsed = { "每次开合扇时可消耗2层扇劲获得增益效果：开扇切合扇：提升8%外功招架忽视、8%内功化解忽视，持续" }
}

E["CS_ng_jh_306_01"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_ng_jh_306_02"] = {
    unparsed = { "-探地通幽，茹毛饮血-魔性乍现，推敌倒地，落拳重轰，对目标造成{@0:伤害包}", "魔性迸发，裂敌肌骨，最后一击汲取目标大量气血和内力" }
}

E["CS_ng_jh_306_03"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    unparsed = { "（功力偷取上限为30层，持续时间内自身功力精进爆发）爆发过后，因功力冲突，内息紊乱反噬，走火入魔" }
}

E["CS_ng_jh_507_01"] = {
    on_hit = {
        {
            control_dur_ms = 1000,
            control_kind = "root"
        }
    },
    unparsed = { "最后一针刺伤筋脉，使目标5秒内无法施展轻功" }
}

E["CS_ng_jh_507_03"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_nj_tq01"] = {
    compile = {
        cast_armor = "yellow"
    }
}

E["CS_nj_tq01_sky"] = {
    compile = {
        cast_armor = "yellow"
    }
}

E["CS_nj_tq02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_nj_tq02_sky"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_nj_tq03"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_nj_tq03",
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            control_dur_ms = 4000,
            control_kind = "launch"
        }
    }
}

E["CS_nj_tq03_hide"] = {
    on_hit = {
        {
            control_dur_ms = 4000,
            control_kind = "launch"
        }
    }
}

E["CS_nj_tq03_hide_sky"] = {
    on_hit = {
        {
            control_dur_ms = 4000,
            control_kind = "launch"
        }
    }
}

E["CS_nj_tq04"] = {
    on_parry_break = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_nj_tq04_sky"] = {
    on_parry_break = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_nj_tq05"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_nj_tq05_hide"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_nj_tq05_hide_sky"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_nlb_tmw01"] = {
    on_hit = {
        {
            apply_buff = "【天魔三相】",
            apply_resolved = false,
            apply_stacks = 1,
            chance = 0.05
        }
    },
    by_level = {
        [2] = { ["on_hit.1.chance"] = 0.1 },
        [3] = { ["on_hit.1.chance"] = 0.15 },
        [4] = { ["on_hit.1.chance"] = 0.2 },
        [5] = { ["on_hit.1.chance"] = 0.25 },
        [6] = { ["on_hit.1.chance"] = 0.3 },
        [7] = { ["on_hit.1.chance"] = 0.35 },
        [8] = { ["on_hit.1.chance"] = 0.4 },
        [9] = { ["on_hit.1.chance"] = 0.45 },
        [10] = { ["on_hit.1.chance"] = 0.5 },
        [11] = { ["on_hit.1.chance"] = 0.55 },
        [12] = { ["on_hit.1.chance"] = 0.6 },
        [13] = { ["on_hit.1.chance"] = 0.65 },
        [14] = { ["on_hit.1.chance"] = 0.7 },
        [15] = { ["on_hit.1.chance"] = 0.75 }
    }
}

E["CS_nlb_tmw02"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_nlb_tmw02_tmyB",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "盘膝坐卧，掌破防御，造成{@0:伤害包}" }
}

E["CS_nlb_tmw03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_nlb_tmw02_tmyB",
            apply_life_ms = 4000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    },
    by_level = {
        [4] = { ["on_hit.1.apply_life_ms"] = 5000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 5000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 5000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [13] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [14] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [15] = { ["on_hit.1.apply_life_ms"] = 7000 }
    }
}

E["CS_nlb_tmw04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_nlb_tmw02_tmyB",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_nlb_tmw05"] = {
    compile = {
        cast_armor = "yellow"
    },
    unparsed = { "以内力裹送暗器，震击敌人，对其造成{@0:伤害包}，若敌人身中“天魔印”，且不在防御状态，则该招" }
}

E["CS_nlb_tmw06"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "stun"
        }
    }
}

E["CS_nlb_tmw07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_nlb_tmw09"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 },
        cast_armor = "yellow"
    }
}

E["CS_nlb_yzx01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    }
}

E["CS_nlb_yzx02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_nlb_yzx03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_nlb_yzx03",
            apply_life_ms = 30000,
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            control_dur_ms = 2000,
            control_kind = "root"
        }
    }
}

E["CS_nlb_yzx03_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "root"
        }
    }
}

E["CS_nlb_yzx04"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_nlb_yzx04_01",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 2000,
            control_kind = "root",
            control_within_ms = 8000
        }
    },
    by_level = {
        [2] = { ["on_parry_break.1.control_within_ms"] = 9000 },
        [3] = { ["on_parry_break.1.control_within_ms"] = 10000 },
        [4] = { ["on_parry_break.1.control_within_ms"] = 11000 },
        [5] = { ["on_parry_break.1.control_within_ms"] = 11000 },
        [6] = { ["on_parry_break.1.control_within_ms"] = 12000 },
        [7] = { ["on_parry_break.1.control_within_ms"] = 13000 },
        [8] = { ["on_parry_break.1.control_within_ms"] = 14000 },
        [9] = { ["on_parry_break.1.control_within_ms"] = 14000 },
        [10] = { ["on_parry_break.1.control_within_ms"] = 15000 },
        [11] = { ["on_parry_break.1.control_within_ms"] = 15000 },
        [12] = { ["on_parry_break.1.control_within_ms"] = 16000 }
    }
}

E["CS_nlb_yzx05"] = {
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "launch"
        }
    }
}

E["CS_nlb_yzx06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "root"
        }
    }
}

E["CS_nlb_yzx07"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "stun"
        }
    }
}

E["CS_nlb_yzx08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 20 }
    },
    on_hit = {
        {
            control_kind = "root"
        }
    }
}

E["CS_qjss_jy01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_qy_lyjyj01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "一层“卧听幽篁",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    },
    unparsed = { "招式范围增加至15米（激活此效果需与巫墟宗派深处山鬼地窟的巫莎琪（77,-124）咨询获取“筋经" }
}

E["CS_qy_lyjyj01_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            apply_buff = "一层“卧听幽篁",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    },
    unparsed = { "招式范围增加至15米（激活此效果需与巫墟宗派深处山鬼地窟的巫莎琪（77,-124）咨询获取“筋经" }
}

E["CS_qy_lyjyj02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "一层“箭草齐发",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_qy_lyjyj03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 10 },
        air = "never"
    }
}

E["CS_qy_lyjyj04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    self_state = {
        {
            apply_buff = "buf_CS_qy_lyjyj04",
            apply_life_ms = 6000,
            apply_resolved = true
        }
    }
}

E["CS_qy_lyjyj05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_qy_lyjyj05",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "ban_qinggong"
        },
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "root"
        },
        {
            condition = { unparried = true },
            control_kind = "stiffen"
        }
    }
}

E["CS_qy_lyjyj06"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_qy_lyjyj07"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    }
}

E["CS_qy_lyjyj07_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    }
}

E["CS_qy_lyjyj08"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_qy_lyjyj08_hide"] = {
    unparsed = { "敬亭之山黄索漠，兀如断岸无棱角，招架，防反" }
}

E["CS_single_bg2"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "knockdown"
        }
    }
}

E["CS_single_bg3"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_single_bg3",
            apply_life_ms = 15000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 15000,
            control_kind = "knockdown"
        }
    }
}

E["CS_single_bpyz"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "root"
        }
    },
    by_level = {
        [2] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [3] = { ["on_hit.1.control_dur_ms"] = 4000 }
    }
}

E["CS_single_bsbc"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_single_cfdh"] = {
    unparsed = { "东海特有的轻功能力，招式使用后使自身在水中施展轻功时效果增强且有几率会回复一定的轻功值，效果持续" }
}

E["CS_single_cfh"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 10 }
    },
    on_hit = {
        {
            apply_buff = "雷霆陷阵杀效果",
            apply_life_ms = 10000,
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 12000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 15000 }
    }
}

E["CS_single_cwqxz"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "buf_dm_cwqxz_01",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_single_cyj"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        air = "never"
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_single_dbgx"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_single_dbgx",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 10000,
            control_kind = "slow"
        }
    }
}

E["CS_single_fcqt"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_single_fmjgz"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 10 }
    }
}

E["CS_single_fzqx"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_single_fzqx01",
            apply_life_ms = 60000,
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.25
        },
        {
            chance = 0.05,
            control_dur_ms = 5000,
            control_kind = "coma"
        }
    },
    by_level = {
        [2] = { ["on_hit.1.chance"] = 0.3, ["on_hit.2.chance"] = 0.04 },
        [3] = { ["on_hit.1.chance"] = 0.35, ["on_hit.2.chance"] = 0.03 }
    },
    unparsed = { "20%概率获得增益状态：移动增加20%，持续20秒", "5%概率获得增益状态：免疫晕眩，持续10秒", "25%概率获得减益状态：全属性降低5，持续30秒", "20%概率获得减益状态：移动降低10%，持续10秒" }
}

E["CS_single_gb01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_single_gb02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_single_gx01"] = {
    unparsed = { "“傩面·河伯” 臂力提高45点，同时额外获得300点破防，出招时自身臂力低于300点则移除此额外" }
}

E["CS_single_gxx"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    unparsed = { "拧身推掌，水袖飞缠，神水宫独门秘技，以水一般的内力控制绸带，遥缠选定区域，造成{@0:伤害包}" }
}

E["CS_single_hfkl"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 },
        cast_armor = "yellow"
    }
}

E["CS_single_hg"] = {
    compile = {
        cast_armor = "yellow"
    }
}

E["CS_single_hqes"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 },
        cast_armor = "red"
    },
    on_hit = {
        {
            apply_buff = "一道“黄泉",
            apply_resolved = false,
            apply_stacks = 1,
            control_kind = "ban_qinggong"
        },
        {
            control_dur_ms = 3000,
            control_kind = "coma"
        }
    },
    by_level = {
        [2] = { ["on_hit.2.control_dur_ms"] = 5000 },
        [3] = { ["on_hit.2.control_dur_ms"] = 10000 }
    },
    unparsed = { "同时在自身脚下留下一道“黄泉”" }
}

E["CS_single_hqes_npc"] = {
    on_hit = {
        {
            control_kind = "coma"
        }
    }
}

E["CS_single_hsp_fyj"] = {
    unparsed = { "华山派剑法轻功：将内力灌注与佩剑之中，配合独门轻功，与目标气息相引，从而快速接近目标", "出招过程黄霸体", "若招式被招架，则无法移动至目标身边" }
}

E["CS_single_hsp_jwsx"] = {
    on_hit = {
        {
            control_kind = "launch"
        },
        {
            apply_buff = "buf_ng_hsp_gdj_cf_02",
            apply_life_ms = 30000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 30000,
            control_kind = "launch"
        }
    }
}

E["CS_single_hsp_yjtf"] = {
    on_hit = {
        {
            control_kind = "knockdown"
        },
        {
            apply_buff = "buf_CS_jh_qzjf07",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_single_huagmz"] = {
    on_hit = {
        {
            control_dur_ms = 6000,
            control_kind = "ban_qinggong"
        }
    },
    by_level = {
        [2] = { ["on_hit.1.control_dur_ms"] = 7000 },
        [3] = { ["on_hit.1.control_dur_ms"] = 8000 },
        [4] = { ["on_hit.1.control_dur_ms"] = 9000 },
        [5] = { ["on_hit.1.control_dur_ms"] = 10000 }
    }
}

E["CS_single_hyxj"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "root"
        },
        {
            apply_buff = "buf_CS_single_hyxj02",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 5000,
            control_kind = "ban_qinggong"
        },
        {
            apply_buff = "黏油效果）",
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    }
}

E["CS_single_jctq"] = {
    on_hit = {
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_single_jctq_hd"] = {
    on_hit = {
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_single_jjg"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_single_jjg",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_single_jl02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_single_jl03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_single_jlj"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 }
    }
}

E["CS_single_js"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    },
    by_level = {
        [2] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [3] = { ["on_hit.1.control_dur_ms"] = 5000 }
    }
}

E["CS_single_jy01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_single_jy02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_single_jz03"] = {
    unparsed = { "可驱散自身及选中范围8米内友方（最多5个）身上1个由禁地守卫施加的特殊招式效果" }
}

E["CS_single_kl03"] = {
    unparsed = { "可驱散自身及选中范围8米内友方（最多5个）身上1个由禁地守卫施加的特殊招式效果" }
}

E["CS_single_lh"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    },
    by_level = {
        [2] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [3] = { ["on_hit.1.control_dur_ms"] = 5000 }
    }
}

E["CS_single_lyq"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "root"
        }
    },
    by_level = {
        [2] = { ["on_hit.1.control_dur_ms"] = 3000 },
        [3] = { ["on_hit.1.control_dur_ms"] = 4000 }
    }
}

E["CS_single_msg"] = {
    unparsed = { "迷神噬心生涅槃，苗疆蛊出无命还，对未招架目标放出迷神蛊虫造成{@0:伤害包}，使其迷失神智，朝向", "（对空中目标无效，且迷失神智只对玩家生效）" }
}

E["CS_single_mshc"] = {
    unparsed = { "回春妙手，逆转阴阳，返血生精，造化新生，将目标的宝贝接植复原，恢复常态，并使目标获得虚弱状态：攻" }
}

E["CS_single_mwnpz"] = {
    on_hit = {
        {
            apply_buff = "buf_dm_mwnpz_01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_single_mz"] = {
    unparsed = { "（只适用于马类坐骑,将召唤人物属性-坐骑界面中已勾选且有时效的马匹,使用任意招式尝试攻击后的12" }
}

E["CS_single_mzy"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_ssg_single_mzy01_01",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 2000,
            control_kind = "coma"
        }
    }
}

E["CS_single_newts01"] = {
    on_hit = {
        {
            control_dur_ms = 1000,
            control_kind = "stun"
        }
    }
}

E["CS_single_newts02"] = {
    unparsed = { "起死回生：受到致命伤害时，回复40%气血" }
}

E["CS_single_newts03"] = {
    on_hit = {
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_single_newts05"] = {
    unparsed = { "禁空领域：禁止使用轻功、移动速度降低20%，离开范围晕眩2秒" }
}

E["CS_single_newts06"] = {
    unparsed = { "身轻如燕：此状态移动速度加强，轻功消耗降低，轻功强化" }
}

E["CS_single_nljqh"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 }
    }
}

E["CS_single_qfww01"] = {
    unparsed = { "江湖流传已久的秘诀，可以恢复自身的气血" }
}

E["CS_single_qfww02"] = {
    unparsed = { "江湖流传已久的秘诀，可以恢复一定的内力" }
}

E["CS_single_qiankyz"] = {
    unparsed = { "一掷赌乾坤，将10两碎银掷出，对正面20米扇形范围内目标造成{@0:伤害包}" }
}

E["CS_single_qmdj"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 },
        cast_armor = "red"
    }
}

E["CS_single_qyg1"] = {
    unparsed = { "使天机珠变换为碧耀模式：自身造成伤害时若敌方气血小于30%则使敌方进入碧耀：气血恢复速率降低25" }
}

E["CS_single_qyg2"] = {
    unparsed = { "使天机珠变换为为连莚模式：自身造成伤害时若敌方存在护体类增益效果（如吸收伤害等直接减免伤害的状态" }
}

E["CS_single_qyg4"] = {
    unparsed = { "创建一道25米的傲倪气场，使气场内所有友方暴击率提高5%，所有伤害提高10%" }
}

E["CS_single_rssq"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "push"
        },
        {
            control_dur_ms = 8000,
            control_kind = "silence"
        }
    },
    by_level = {
        [2] = { ["on_hit.2.control_dur_ms"] = 10000 },
        [3] = { ["on_hit.2.control_dur_ms"] = 12000 }
    },
    unparsed = { "未及招架的敌人将获得“弱水三千”状态：大幅度降低其闪避几率，持续8秒" }
}

E["CS_single_sjhb"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "red"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_single_sjjz"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 10 }
    },
    unparsed = { "同时周围每个神机营队友都会为自己增加1层神机防御：提高2%外功防御，最多10层，持续15秒" }
}

E["CS_single_sl01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_single_sl03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_single_tianmjt"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_single_tls2"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_single_tly"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    }
}

E["CS_single_ts01"] = {
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "stun"
        }
    }
}

E["CS_single_txg"] = {
    on_hit = {
        {
            apply_buff = "同心蛊状态",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 10000 }
    }
}

E["CS_single_txg_hide"] = {
    on_hit = {
        {
            apply_buff = "同心蛊状态",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 10000 }
    }
}

E["CS_single_txsj"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_single_tyhg1"] = {
    unparsed = { "10秒内提高自身5%的内息并回复5%的内力" }
}

E["CS_single_tyhg2"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 10 }
    },
    on_hit = {
        {
            apply_buff = "10秒的瀑水成虹状态",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 10000,
            control_kind = "knockdown"
        }
    }
}

E["CS_single_tyhg3"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "coma"
        }
    }
}

E["CS_single_tzst"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_single_wd02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_single_wd03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    }
}

E["CS_single_wh1"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_wh_hyj_mark_cd",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "coma",
            heal = true
        }
    }
}

E["CS_single_wh2"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 },
        cast_armor = "yellow"
    }
}

E["CS_single_wh3"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 }
    }
}

E["CS_single_wh4"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 }
    }
}

E["CS_single_wjdbz"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "launch"
        },
        {
            apply_buff = "buf_dm_wjdbz_02",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_single_wjxc"] = {
    unparsed = { "品茗于心，成功施展招式可恢复自身最大内力值的20%" }
}

E["CS_single_wnwx"] = {
    on_hit = {
        {
            control_dur_ms = 6000,
            control_kind = "silence"
        }
    },
    by_level = {
        [2] = { ["on_hit.1.control_dur_ms"] = 9000 },
        [3] = { ["on_hit.1.control_dur_ms"] = 12000 }
    }
}

E["CS_single_wsdy"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 10 }
    }
}

E["CS_single_wshj"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 10 }
    }
}

E["CS_single_wxd"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 5000,
            control_kind = "ban_qinggong"
        }
    },
    by_level = {
        [2] = { ["on_hit.1.control_dur_ms"] = 6000 },
        [3] = { ["on_hit.1.control_dur_ms"] = 7000 }
    }
}

E["CS_single_wxp01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_single_wxp02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_single_xd"] = {
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "stun"
        }
    }
}

E["CS_single_xhb"] = {
    on_hit = {
        {
            apply_buff = "销魂效果",
            apply_life_ms = 8000,
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_life_ms"] = 10000 },
        [3] = { ["on_hit.1.apply_life_ms"] = 12000 }
    }
}

E["CS_single_xm1"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 7 }
    },
    on_hit = {
        {
            apply_buff = "对应的“磐隐暗劲",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_hit.1.apply_stacks"] = 2 },
        [3] = { ["on_hit.1.apply_stacks"] = 3 }
    }
}

E["CS_single_xm2"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_single_xm3"] = {
    unparsed = { "挥毫泼墨，对目标造成{@0:伤害包}若目标未招架则击晕1秒，随后在目标脸部挥墨，给其留下墨印" }
}

E["CS_single_xsmq"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 10 }
    }
}

E["CS_single_xt01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_single_xt03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    }
}

E["CS_single_xzqw"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_single_yfs"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    }
}

E["CS_single_yfz"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "stun"
        }
    }
}

E["CS_single_ywdj"] = {
    unparsed = { "达摩派独门轻功秘技，折苇枝投入水中，再以独门轻身法门踏苇渡水" }
}

E["CS_single_zdhl"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "stun"
        }
    }
}

E["CS_sj_syxz01"] = {
    unparsed = { "每次命中未招架目标，自身叠加1层“金针”状态：减少1点【天女散花】怒气消耗，状态可叠加18层，持" }
}

E["CS_sj_syxz02"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_sj_syxz02",
            apply_life_ms = 3000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_parry_break.1.apply_life_ms"] = 4000 },
        [3] = { ["on_parry_break.1.apply_life_ms"] = 5000 },
        [4] = { ["on_parry_break.1.apply_life_ms"] = 6000 },
        [5] = { ["on_parry_break.1.apply_life_ms"] = 7000 },
        [6] = { ["on_parry_break.1.apply_life_ms"] = 8000 },
        [7] = { ["on_parry_break.1.apply_life_ms"] = 9000 },
        [8] = { ["on_parry_break.1.apply_life_ms"] = 10000 },
        [9] = { ["on_parry_break.1.apply_life_ms"] = 11000 },
        [10] = { ["on_parry_break.1.apply_life_ms"] = 12000 },
        [11] = { ["on_parry_break.1.apply_life_ms"] = 13000 },
        [12] = { ["on_parry_break.1.apply_life_ms"] = 14000 },
        [13] = { ["on_parry_break.1.apply_life_ms"] = 15000 }
    }
}

E["CS_sj_syxz03"] = {
    on_parry = {
        {
            apply_buff = "buf_CS_sj_syxz03",
            apply_life_ms = 18000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [2] = { ["on_parry.1.apply_life_ms"] = 19000 },
        [3] = { ["on_parry.1.apply_life_ms"] = 20000 },
        [4] = { ["on_parry.1.apply_life_ms"] = 21000 },
        [5] = { ["on_parry.1.apply_life_ms"] = 22000 },
        [6] = { ["on_parry.1.apply_life_ms"] = 23000 },
        [7] = { ["on_parry.1.apply_life_ms"] = 24000 },
        [8] = { ["on_parry.1.apply_life_ms"] = 25000 },
        [9] = { ["on_parry.1.apply_life_ms"] = 26000 },
        [10] = { ["on_parry.1.apply_life_ms"] = 27000 },
        [11] = { ["on_parry.1.apply_life_ms"] = 28000 },
        [12] = { ["on_parry.1.apply_life_ms"] = 29000 },
        [13] = { ["on_parry.1.apply_life_ms"] = 30000 }
    },
    unparsed = { "（每施展一次本套路招式，清除一层识络状态）" }
}

E["CS_sj_syxz04"] = {
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_sj_syxz05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_sj_syxz05_2",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_sj_syxz05_hide"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_sj_syxz05_2",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_sj_syxz06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_sj_syxz06",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            heal = true
        }
    },
    unparsed = { "（蓄力时间为5秒，目标持续回血状态只可同时存在一个，该恢复不受淤血限制）（自身医疗值越高，招式功" }
}

E["CS_sj_syxz07"] = {
    unparsed = { "蓄力招式，枯木逢雨，妙手回春，若蓄力成功：可将重伤倒地目标救起，并恢复其15%最大气血内力值", "（招式蓄力时间3秒，招式只能对重伤倒地队友使用）" }
}

E["CS_sj_syxz08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 },
        cast_armor = "yellow"
    }
}

E["CS_sjy_dkd01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_sjy_dkd02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_sjy_dkd02",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    }
}

E["CS_sjy_dkd03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "惊蛇",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            control_kind = "slow"
        }
    },
    unparsed = { "同时自身可获得1层“战意”状态" }
}

E["CS_sjy_dkd04"] = {
    self_state = {
        {
            apply_buff = "buf_CS_sjy_dkd04",
            apply_life_ms = 6000,
            apply_resolved = true
        }
    },
    by_level = {
        [2] = { ["self_state.1.apply_life_ms"] = 7000 },
        [3] = { ["self_state.1.apply_life_ms"] = 8000 },
        [4] = { ["self_state.1.apply_life_ms"] = 9000 },
        [5] = { ["self_state.1.apply_life_ms"] = 10000 },
        [6] = { ["self_state.1.apply_life_ms"] = 12000 }
    }
}

E["CS_sjy_dkd05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_sjy_dkd06"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_sjy_dkd07"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_sjy_dkd07",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        },
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    },
    unparsed = { "（无法在空中使用）" }
}

E["CS_sjy_klbx01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "自身存在1层且只有1层回马状态时，招式最后一击可挑飞目标" }
}

E["CS_sjy_klbx01_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "身似铁桥，仰身出枪，对选中目标造成{@0:伤害包}，招式第二击可挑飞未招架的敌人" }
}

E["CS_sjy_klbx02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 6 }
    },
    unparsed = { "当招式命中未招架的敌人时，自身获得1层回马：存在1层且只有1层回马状态时，【怒龙锁月】招式可挑飞" }
}

E["CS_sjy_klbx03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    }
}

E["CS_sjy_klbx04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    },
    unparsed = { "（施展该招式时自身移动速度大幅提升且可自由移动，但不可使用轻功）（装备长枪发挥100%招式威力，" }
}

E["CS_sjy_klbx05"] = {
    unparsed = { "使自身进入反守为攻状态（持续5秒）：处于招架状态被敌人攻击时，可直接触发回马枪，但无挑飞效果", "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）（装备长枪发挥100%招式威力，装备长棍只能发挥70%招式威力）" }
}

E["CS_sjy_klbx05_hide"] = {
    unparsed = { "使自身进入反守为攻状态（持续5秒）：处于招架状态被敌人攻击时，可直接触发回马枪，但无挑飞效果", "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）（装备长枪发挥100%招式威力，装备长棍只能发挥70%招式威力）" }
}

E["CS_sjy_klbx06"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_sjy_klbx07"] = {
    unparsed = { "蓄势发出一声狂啸，同时以独特心法激发自身潜力：自身气血值百分比高于或等于50%时，狂啸过后会消耗", "自身气血值百分比低于50%时，狂啸过后会消耗当前内力的50%以获得惊龙回光状态：回复已损失气血和" }
}

E["CS_sl_chpdg01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    unparsed = { "若自身存在“五蕴皆空”状态，本招式暴击几率大幅度增加" }
}

E["CS_sl_chpdg02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_sl_chpdg03"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_sl_chpdg03_01",
            apply_life_ms = 8000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "招架成功减少招式【五蕴皆空】1秒回气时间" }
}

E["CS_sl_chpdg04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_sl_chpdg05"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_sl_chpdg06"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            control_dur_ms = 1000,
            control_kind = "stiffen",
            control_within_ms = 5000
        }
    }
}

E["CS_sl_chpdg07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_sl_chpdg08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow",
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "knockdown"
        }
    },
    unparsed = { "使倒地状态的敌人无法起身" }
}

E["CS_sl_djgq02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "coma"
        }
    }
}

E["CS_sl_djgq03"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_sl_djgq04"] = {
    on_hit = {
        {
            apply_buff = "buf_wuji_CS_sl_djgq04",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "敌若妄图招架则会被摔的更惨（受到额外刚性伤害）" }
}

E["CS_sl_djgq05"] = {
    unparsed = { "防御状态，招架成功时，使攻击者受到7点刚性反伤（伤害受内功威力影响）" }
}

E["CS_sl_djgq07"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_sl_dmgf01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    }
}

E["CS_sl_dmgf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "冲锋招式，将臂力灌注棍端攻向目标，造成{@0:伤害包}，“佛光普照”状态下，每击将附加伤害，自身" }
}

E["CS_sl_dmgf03"] = {
    on_hit = {
        {
            apply_buff = "一层“普渡四方",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    }
}

E["CS_sl_dmgf04"] = {
    self_state = {
        {
            apply_buff = "buf_CS_sl_dmgf04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_sl_dmgf05"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_sl_dmgf06"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    },
    by_level = {
        [8] = { ["on_hit.1.control_dur_ms"] = 15000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 15000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 15000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 15000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 15000 },
        [13] = { ["on_hit.1.control_dur_ms"] = 15000 },
        [14] = { ["on_hit.1.control_dur_ms"] = 15000 },
        [15] = { ["on_hit.1.control_dur_ms"] = 15000 },
        [16] = { ["on_hit.1.control_dur_ms"] = 15000 },
        [17] = { ["on_hit.1.control_dur_ms"] = 15000 },
        [18] = { ["on_hit.1.control_dur_ms"] = 15000 },
        [19] = { ["on_hit.1.control_dur_ms"] = 15000 },
        [20] = { ["on_hit.1.control_dur_ms"] = 15000 }
    }
}

E["CS_sl_dmgf07"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "stun"
        },
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "stun"
        }
    },
    unparsed = { "处于地面释放时，若命中已招架目标，则清除自身所有的“佛光普照”状态" }
}

E["CS_sl_dmgf07_sky"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "stun"
        }
    }
}

E["CS_sl_lhq02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_sl_lhq03"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_sl_lhq03",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_sl_lhq04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_sl_lhq04",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_sl_lhq05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_sl_lhq04",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "幻相无生，拈花一指，运行“拈花”真气于周身，当自身存在此真气时：使用罗汉拳招式【野马追风】或【连", "每使用一次招式【连环三挑】或【野马追风】减少一层“拈花”状态" }
}

E["CS_sl_lhq07"] = {
    unparsed = { "猛虎之势，直扑空中目标，将其击落并造成{@0:伤害包}" }
}

E["CS_sl_lzs01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_sl_lzs01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_sl_lzs02"] = {
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_sl_lzs03"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_sl_lzs04",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    by_level = {
        [11] = { ["compile.aoe_radius_m"] = 15 },
        [12] = { ["compile.aoe_radius_m"] = 15 },
        [13] = { ["compile.aoe_radius_m"] = 15 },
        [14] = { ["compile.aoe_radius_m"] = 15 },
        [15] = { ["compile.aoe_radius_m"] = 15 },
        [16] = { ["compile.aoe_radius_m"] = 15 },
        [17] = { ["compile.aoe_radius_m"] = 15 },
        [18] = { ["compile.aoe_radius_m"] = 15 },
        [19] = { ["compile.aoe_radius_m"] = 15 }
    }
}

E["CS_sl_lzs04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_sl_lzs04",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_sl_lzs05"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "stun"
        }
    }
}

E["CS_sl_lzs06"] = {
    on_hit = {
        {
            apply_buff = "一层“虚",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "若在地面攻击空中的目标，则会击落目标并附加额外伤害" }
}

E["CS_sl_wtgf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_sl_wtgf03"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_sl_wtgf03",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_sl_wtgf04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_sl_wtgf04_01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_sl_wtgf05"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "stun"
        }
    }
}

E["CS_slqj_djgq02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "coma"
        }
    }
}

E["CS_slqj_djgq03"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_slqj_djgq04"] = {
    on_hit = {
        {
            apply_buff = "buf_wuji_CS_sl_djgq04",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "敌若妄图招架则会被摔的更惨（受到额外刚性伤害）" }
}

E["CS_slqj_djgq05"] = {
    unparsed = { "防御状态，招架成功时，使攻击者受到14点刚性反伤（伤害受内功威力影响）" }
}

E["CS_slqj_djgq06"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "red"
    }
}

E["CS_slqj_djgq07"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_slqj_djgq08"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    }
}

E["CS_slzp_lzs01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_slzp_lzs01",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "同时激活全部【佛宗】及【禅心】武技则目标在空中时招式变为远程单体招式" }
}

E["CS_slzp_lzs01_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_sl_lzs01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_slzp_lzs02"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "ban_qinggong"
        }
    },
    unparsed = { "同时激活全部【佛宗】及【禅心】武技则目标在空中时招式射程提高5米" }
}

E["CS_slzp_lzs02_hide"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_slzp_lzs03"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_slzp_lzs04",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_slzp_lzs03_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_parry_break = {
        {
            apply_buff = "buf_CS_sl_lzs04",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "破防成功后自身额外获得30%穿透效果（可穿透大部分招式内功及装备的免伤和吸收效果，造成伤害）并使" }
}

E["CS_slzp_lzs04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_slzp_lzs04",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_slzp_lzs05"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "root"
        }
    }
}

E["CS_slzp_lzs06"] = {
    on_hit = {
        {
            apply_buff = "一层“虚",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "若在地面攻击空中的目标，则会击落目标并附加额外伤害" }
}

E["CS_slzp_lzs07"] = {
    on_hit = {
        {
            apply_buff = "一层“缺",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    },
    unparsed = { "同时激活全部【佛宗】及【禅心】武技则目标在空中时招式射程提高10米", "若招式被打断或闪避，则获得一层“灭风”状态：下次【腾风】攻击空中目标时怒气消耗降低30点" }
}

E["CS_slzp_lzs07_hide"] = {
    on_hit = {
        {
            apply_buff = "一层“缺",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    },
    unparsed = { "若招式被打断或闪避，则获得一层“灭风”状态：下次【腾风】攻击空中目标时怒气消耗降低30点" }
}

E["CS_slzp_lzs08"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 26000,
            control_kind = "launch"
        },
        {
            apply_buff = "一层“搬山",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_ssg_bhjtz01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    },
    unparsed = { "掌影纷纷推狂浪，劲力层层势滔天" }
}

E["CS_ssg_bhjtz02"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "coma"
        }
    }
}

E["CS_ssg_bhjtz03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_ssg_bhjtz04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 10 }
    }
}

E["CS_ssg_bhjtz06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_ssg_bhjtz06_01",
            apply_life_ms = 6000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 6000,
            control_kind = "silence"
        }
    },
    by_level = {
        [3] = { ["on_hit.1.apply_life_ms"] = 7000, ["on_hit.1.control_dur_ms"] = 7000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 7000, ["on_hit.1.control_dur_ms"] = 7000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 8000, ["on_hit.1.control_dur_ms"] = 8000 }
    }
}

E["CS_ssg_bhjtz07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 },
        cast_armor = "yellow"
    },
    self_state = {
        {
            apply_buff = "buf_CS_ssg_bhjtz07_smd",
            apply_resolved = true
        }
    },
    unparsed = { "“云封”、“雾锁”状态持续6秒，可叠加5层" }
}

E["CS_ssg_bhjtz08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 }
    },
    self_state = {
        {
            apply_buff = "buf_CS_ssg_bhjtz08_01",
            apply_resolved = true
        }
    }
}

E["CS_ssg_bhjtz09"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_ssg_bhjtz09_01",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 2000,
            control_kind = "root"
        }
    }
}

E["CS_ssg_jxc01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_ssg_jxc02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_ssg_jxc02",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_ssg_jxc03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            control_within_ms = 3000
        }
    }
}

E["CS_ssg_jxc04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_ssg_jxc05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_ssg_jxc06"] = {
    on_hit = {
        {
            apply_buff = "破雪而立状态",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_ssg_jxc07"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "coma"
        }
    }
}

E["CS_ssg_jxc08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "coma"
        }
    }
}

E["CS_th_bhcs01"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "音律羽",
            apply_resolved = false,
            apply_stacks = 1
        },
        {
            apply_buff = "buf_CS_th_bhcs01",
            apply_life_ms = 15000,
            apply_resolved = true,
            apply_stacks = 1,
            per_hit = true
        }
    },
    unparsed = { "若目标身上浩渺状态叠加至6层，则目标遭受内伤" }
}

E["CS_th_bhcs02"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "音律徵",
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_th_bhcs03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 },
        cast_armor = "yellow"
    }
}

E["CS_th_bhcs04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "音律商",
            apply_resolved = false,
            apply_stacks = 1
        },
        {
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_th_bhcs05"] = {
    on_hit = {
        {
            apply_buff = "风起澜生状态",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "招架成功时，自身获得“风飞”状态：移动速度提升5%，可叠加10层，持续15秒" }
}

E["CS_th_bhcs06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            apply_buff = "音律角",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_th_bhcs07"] = {
    on_hit = {
        {
            apply_buff = "音律宫",
            apply_resolved = false,
            apply_stacks = 1
        },
        {
            apply_buff = "buf_CS_th_bhcs01",
            apply_life_ms = 15000,
            apply_resolved = true,
            apply_stacks = 1,
            per_hit = true
        }
    },
    unparsed = { "若目标身上浩渺状态叠加至6层，则目标遭受内伤" }
}

E["CS_th_bhcs08"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_th_bhcs08",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "若音律为角商羽，则招式附带效果：对目标造成伤害，招式可利用轻功快速接近对手" }
}

E["CS_th_bhcs08_hide01"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_th_bhcs08_hide01",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_th_bhcs08_hide02"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_th_bhcs08_hide01",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_th_bhcs08_hide03"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_th_bhcs08_hide01",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_tls_xycd01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_tls_xycd02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 5000,
            control_kind = "knockdown"
        }
    }
}

E["CS_tls_xycd03"] = {
    on_hit = {
        {
            apply_buff = "最低级的“不虚不实",
            apply_life_ms = 10000,
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 10000,
            control_kind = "knockdown"
        }
    }
}

E["CS_tls_xycd04"] = {
    on_hit = {
        {
            apply_buff = "7秒的“不喜不悲",
            apply_resolved = false,
            apply_stacks = 1
        },
        {
            control_dur_ms = 3000,
            control_kind = "stun"
        }
    }
}

E["CS_tls_xycd05"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch",
            per_hit = true
        }
    },
    by_level = {
        [11] = { ["compile.aoe_radius_m"] = 12 },
        [12] = { ["compile.aoe_radius_m"] = 12 }
    }
}

E["CS_tls_xycd07"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    }
}

E["CS_tls_xycd08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        },
        {
            control_dur_ms = 2000,
            control_kind = "knockdown"
        }
    }
}

E["CS_tls_xycd08_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "knockdown"
        }
    }
}

E["CS_tm_fgzyc02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_tm_fgzyc03"] = {
    unparsed = { "招式可挑碎目标招架，并使其腿足虚麻，移动速度降低" }
}

E["CS_tm_fgzyc04"] = {

}

E["CS_tm_fgzyc05"] = {
    on_hit = {
        {
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_tm_fgzyc06_a"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_tm_fgzyc07"] = {
    on_hit = {
        {
            control_kind = "root"
        }
    },
    unparsed = { "若目标在地面，则使其流失气血，暂时无法施展轻功" }
}

E["CS_tm_jsc01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_tm_jsc04"] = {
    unparsed = { "妄图招架之敌，会被强力震飞且下身麻痹（减速30%，持续8秒）" }
}

E["CS_tm_jsc05"] = {
    unparsed = { "招架成功时，化对方之力纳为己用，使攻击附加7点阴伤害，持续10秒（伤害受内功威力影响）" }
}

E["CS_tm_jsc06"] = {
    unparsed = { "招式配合“随行”身法使用时，可极大提升暴击几率" }
}

E["CS_tm_mhb04"] = {
    unparsed = { "妄图抵抗的敌人徒耗更多招架耐力" }
}

E["CS_tm_mhb05"] = {

}

E["CS_tm_mhb06"] = {
    unparsed = { "如果对方勉强招架，会额外消耗其招架耐力" }
}

E["CS_tm_tmzhd01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_tm_tmzhd01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_tm_tmzhd02"] = {
    unparsed = { "飞身扬击，追魂夺魄，对目标造成{@0:伤害包}（伤害随“追魂”叠加次数和与目标距离的拉远而提高）" }
}

E["CS_tm_tmzhd03"] = {
    unparsed = { "刀劲刺伤敌穴，使其使用轻功时额外损耗内力" }
}

E["CS_tm_tmzhd04"] = {
    on_parry_break = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_tm_tmzhd05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_tm_tmzhd01",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "防御架势，身影如魅，招架成功时，可瞬间大幅提升移动能力，持续2秒" }
}

E["CS_tm_tmzhd06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_tm_ykr01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    unparsed = { "命中未招架的敌人会窃取其10%的闪避几率，该效果持续12秒" }
}

E["CS_tm_ykr02"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_tm_ykr02_01",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            control_kind = "launch"
        }
    }
}

E["CS_tm_ykr03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_tm_ykr04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "ban_qinggong"
        },
        {
            control_kind = "ban_qinggong"
        }
    },
    unparsed = { "神嚎：造成持续伤害，并每秒获得25%的减速效果" }
}

E["CS_tm_ykr06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "ban_qinggong"
        },
        {
            control_kind = "ban_qinggong"
        }
    },
    self_state = {
        {
            apply_buff = "buf_CS_tm_ykr06_01",
            apply_resolved = true
        }
    },
    unparsed = { "神嚎：造成持续伤害，并每秒获得25%的减速效果" }
}

E["CS_tm_ykr07"] = {
    unparsed = { "瞬身闪击，瞬移到敌人身后，造成{@0:伤害包}，并使其僵直1秒", "稚元:射程增加至17米" }
}

E["CS_tm_ykr08"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    },
    unparsed = { "向身前掷出唐门独门暗器“轮回刃”，“轮回刃”在飞行15米之后会自动飞回" }
}

E["CS_tm_ywt01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 10 }
    }
}

E["CS_tm_ywt01_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 10 }
    }
}

E["CS_tm_ywt02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_tm_ywt03"] = {
    unparsed = { "同时根据目标存有“冤魂”状态的层数，附加额外伤害" }
}

E["CS_tm_ywt04"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    unparsed = { "对陷阱内敌人造成持续伤害（若目标身上存有“冤魂”状态，可造成更多伤害，层数越多，伤害越高）" }
}

E["CS_tm_ywt05"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "stun"
        }
    },
    by_level = {
        [3] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [4] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [5] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [6] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [7] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [8] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [11] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [12] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [13] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [14] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [15] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [16] = { ["on_hit.1.control_dur_ms"] = 4000 },
        [17] = { ["on_hit.1.control_dur_ms"] = 4000 }
    }
}

E["CS_tm_ywt06"] = {
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "root"
        },
        {
            apply_buff = "buf_CS_tm_ywt06_03",
            apply_resolved = true,
            apply_stacks = 1
        }
    },
    unparsed = { "陷阱内敌人持续减速20%并限制轻功（持续效果不叠加）" }
}

E["CS_tm_ywt07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 }
    }
}

E["CS_tm_ywt09"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_tmqj_mhb01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_tmqj_mhb02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_tmqj_mhb04"] = {
    unparsed = { "妄图抵抗的敌人徒耗更多招架耐力" }
}

E["CS_tmqj_mhb05"] = {

}

E["CS_tmqj_mhb06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "red"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    },
    unparsed = { "如果对方勉强招架，会额外消耗其招架耐力" }
}

E["CS_ts_tsjf01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_ts_tsjf01",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_ts_tsjf02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_ts_tsjf03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_ts_tsjf01",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_ts_tsjf05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_ts_tsjf02",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_ts_tsjf06"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "root"
        }
    },
    unparsed = { "寒气凛然，滴水成冰，用天山独有的内力激荡剑身在地面挑起一道冰墙造成{@0:伤害包}" }
}

E["CS_ts_tsjf07"] = {
    unparsed = { "独自高山凭落寞，千层冰魄映长天，招架获得一层冰寒之气，可叠加10层，10层后可主动释放，利用冰寒", "施展后清除叠加的冰寒之气并进入调息时间20秒使用其他套路清除此状态" }
}

E["CS_ts_tsjf08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 },
        can_move = false,
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_ts_tyjf01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_ts_tyjf01",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_ts_tyjf02"] = {
    on_hit = {
        {
            apply_buff = "雪落状态",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_ts_tyjf03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_ts_tyjf04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "一层落雪效果",
            apply_resolved = false,
            apply_stacks = 1,
            per_hit = true
        },
        {
            condition = { unparried = true },
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_ts_tyjf04_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    unparsed = { "震碎落雪剑时，自身每存在一层落雪效果，5秒内自身受到的伤害减少10%（最多50%）并移除落雪效果" }
}

E["CS_ts_tyjf05"] = {
    compile = {
        cast_armor = "yellow"
    }
}

E["CS_ts_tyjf06"] = {
    unparsed = { "成功招架使敌方获得清辉状态：减速5%，内功防御降低3%，清辉存在5秒可叠加5层" }
}

E["CS_ts_tyjf07"] = {
    on_hit = {
        {
            control_kind = "slow"
        }
    },
    unparsed = { "稚元:射程增加至5米" }
}

E["CS_ts_tyjf08"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    },
    unparsed = { "稚元:命中未招架目标后，提高自身30%命中率，持续5秒" }
}

E["CS_ts_wcjf01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            mp_drain = true,
            mp_drain_dur_ms = 0
        }
    }
}

E["CS_ts_wcjf02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    }
}

E["CS_ts_wcjf03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_ts_wcjf04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_ts_wcjf05"] = {
    on_hit = {
        {
            apply_buff = "销魂状态",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true },
            control_kind = "stiffen",
            per_hit = true
        }
    }
}

E["CS_ts_wcjf06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_ts_wcjf06_01_1",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 5000,
            control_kind = "ban_qinggong"
        }
    },
    unparsed = { "汨罗江水长连绵，朝朝暮暮勿停歇，楚人歌，离骚响乾坤，使用招式获得“转乾”状态，再次使用可获得“转" }
}

E["CS_ts_wcjf07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch",
            per_hit = true
        }
    }
}

E["CS_ts_wcjf08"] = {
    unparsed = { "每击都可附加0.1倍目标已损失内力伤害（只对玩家有效）" }
}

E["CS_tyhg_mwybj01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_tyhg_mwybj01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_tyhg_mwybj02"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_tyhg_mwybj01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_tyhg_mwybj03"] = {
    self_state = {
        {
            apply_buff = "buf_CS_tyhg_mwybj02_01",
            apply_resolved = true
        }
    }
}

E["CS_tyhg_mwybj04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_tyhg_mwybj03_02",
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            apply_buff = "buf_CS_tyhg_mwybj03_03_01",
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            control_dur_ms = 8000,
            control_kind = "root"
        }
    },
    unparsed = { "奋笔疾书，对点选地面创建半径5米的阵法区域，区域开启时造成{@0:伤害包}" }
}

E["CS_tyhg_mwybj05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_tyhg_mwybj04_01",
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            apply_buff = "buf_CS_tyhg_mwybj04_03",
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            control_dur_ms = 8000,
            control_kind = "knockdown"
        }
    }
}

E["CS_tyhg_mwybj06"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "铁画银钩，对目标造成{@0:伤害包}，若目标存在“采薇”状态且未招架时，该招式额外造成0.3倍伤" }
}

E["CS_tyhg_mwybj07"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_tyhg_mwybj05",
            apply_resolved = true,
            apply_stacks = 1,
            control_kind = "launch"
        }
    }
}

E["CS_tyhg_mwybj08"] = {
    unparsed = { "笔意肆意，对点选半径5米的区域内敌方目标造成{@0:伤害包}，并对未格挡目标造成短暂硬直", "之后再选定区域展开阵法，阵内友方目标回复大量气血（禁地，心魔中额外回复气血）", "若身处禁地或心魔场景时，区域内友方目标获得“酒浓春入梦”状态：持续恢复怒气，阵法存在5秒" }
}

E["CS_tyhg_yszyq01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_tyhg_yszyq01",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            apply_buff = "沧海效果",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_tyhg_yszyq02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_tyhg_yszyq03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 5000,
            control_kind = "launch",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_tyhg_yszyq04"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            displacement_distance_m = 0,
            displacement_kind = "push"
        },
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "root"
        }
    },
    unparsed = { "若目标存在浮天，则每存在1层浮天，使目标减速25%，持续3秒" }
}

E["CS_tyhg_yszyq05"] = {
    compile = {
        cast_armor = "yellow"
    },
    unparsed = { "对目标造成{@0:伤害包}，最后一击时，自身每存在1层云海，则使未招架目标键位错乱1秒，最多3秒" }
}

E["CS_tyhg_yszyq06"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "silence"
        }
    },
    unparsed = { "5秒内自身持续获得虚无缥缈状态：每层提升10%内功命中率，最多可叠加5层，虚无缥缈状态可持续3秒" }
}

E["CS_tyhg_yszyq07"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_tyhg_yszyq08"] = {
    self_state = {
        {
            apply_buff = "云海相望",
            apply_life_ms = 9000,
            apply_resolved = false
        }
    },
    by_level = {
        [3] = { ["self_state.1.apply_life_ms"] = 10000 },
        [4] = { ["self_state.1.apply_life_ms"] = 10000 },
        [5] = { ["self_state.1.apply_life_ms"] = 11000 },
        [6] = { ["self_state.1.apply_life_ms"] = 11000 },
        [7] = { ["self_state.1.apply_life_ms"] = 12000 },
        [8] = { ["self_state.1.apply_life_ms"] = 12000 },
        [9] = { ["self_state.1.apply_life_ms"] = 13000 },
        [10] = { ["self_state.1.apply_life_ms"] = 13000 },
        [11] = { ["self_state.1.apply_life_ms"] = 14000 },
        [12] = { ["self_state.1.apply_life_ms"] = 15000 }
    },
    unparsed = { "每存在1层云海状态，释放【浮天沧海】和【秋宵梦觉】时可额外获得5%内功暴击率" }
}

E["CS_wd_lyjf01"] = {
    on_hit = {
        {
            control_kind = "coma"
        }
    },
    unparsed = { "“阳切阴”时会将纯阳真气转换为内力并对目标造成伤害" }
}

E["CS_wd_lyjf03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_wd_lyjf04"] = {
    unparsed = { "挑剑扬击，对前方敌人造成{@0:伤害包}，打破敌人招架时可使其重心不稳，移动速度降低" }
}

E["CS_wd_lyjf05"] = {

}

E["CS_wd_lyjf07"] = {
    unparsed = { "阴阳剑气交错纷飞，对目标造成{@0:伤害包}，凌空使用可击落空中目标，同时：“阳”状态下凌空攻击", "“阴”状态下凌空攻击空中目标可增加自身怒气" }
}

E["CS_wd_qfjf01"] = {
    unparsed = { "简略扬击，凌厉剑气横切前方敌人造成{@0:伤害包}，并使其每次使用轻功都遭受额外伤害" }
}

E["CS_wd_qfjf02"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_wd_qfjf02",
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            control_kind = "knockdown"
        },
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_wd_qfjf03"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 20000,
            control_kind = "launch"
        }
    },
    unparsed = { "当目标被“浊”气所困时，招式余劲可使其3秒内无法招架" }
}

E["CS_wd_qfjf04"] = {
    unparsed = { "妄图招架之敌，会感染“浊”气并被摔倒" }
}

E["CS_wd_qfjf05"] = {
    unparsed = { "防御架势，涤荡心神，每次成功招架都使自身“清风”剑气更盛，持续10秒" }
}

E["CS_wd_qfjf07"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "当目标无法招架时将受到额外伤害" }
}

E["CS_wd_rzrj01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    },
    unparsed = { "命中未招架的敌人会让自己获得梦魂，提高自身命中率10%，效果持续10秒" }
}

E["CS_wd_rzrj02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "根据自身的似水层数会使攻击附加额外伤害，每次命中目标清除一层似水" }
}

E["CS_wd_rzrj03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_wd_rzrj03_1",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "launch"
        }
    }
}

E["CS_wd_rzrj04"] = {
    unparsed = { "跃步回剑斩，快速移动至目标身后，击晕其3秒并造成{@0:伤害包}", "同时自身8秒内内功命中率提升30%根据自身的似水层数会使攻击附加额外伤害，每次命中目标清除一层似" }
}

E["CS_wd_rzrj05"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_wd_rzrj06"] = {
    on_hit = {
        {
            apply_buff = "一层似水",
            apply_life_ms = 15000,
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_wd_rzrj07"] = {
    unparsed = { "根据自身的似水层数会使攻击附加额外伤害，每次命中目标清除一层似水" }
}

E["CS_wd_rzrj08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "knockdown"
        },
        {
            condition = { unparried = true },
            control_dur_ms = 5000,
            control_kind = "knockdown"
        }
    }
}

E["CS_wd_tjj01"] = {
    compile = {
        max_hits = 2,
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_wd_tjj02"] = {
    unparsed = { "独特调息心法，可将气血和内力的比例互换" }
}

E["CS_wd_tjj03"] = {
    unparsed = { "独特吐纳心法，可在短时间内迅速恢复一定内力" }
}

E["CS_wd_tjj04"] = {
    unparsed = { "纯阳一击，剑气波及到的敌人受到{@0:伤害包}，并致其内伤：移动速度减缓，生命持续损耗" }
}

E["CS_wd_tjj04z"] = {
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_wd_tjj05"] = {
    on_hit = {
        {
            condition = { parry_broken = true },
            mp_restore = 0
        }
    }
}

E["CS_wd_tjj06"] = {
    on_parry = {
        {
            mp_restore = 4
        }
    },
    by_level = {
        [2] = { ["on_parry.1.mp_restore"] = 5 },
        [3] = { ["on_parry.1.mp_restore"] = 7 },
        [4] = { ["on_parry.1.mp_restore"] = 9 },
        [5] = { ["on_parry.1.mp_restore"] = 11 },
        [6] = { ["on_parry.1.mp_restore"] = 13 },
        [7] = { ["on_parry.1.mp_restore"] = 15 },
        [8] = { ["on_parry.1.mp_restore"] = 18 },
        [9] = { ["on_parry.1.mp_restore"] = 21 },
        [10] = { ["on_parry.1.mp_restore"] = 23 },
        [11] = { ["on_parry.1.mp_restore"] = 26 },
        [12] = { ["on_parry.1.mp_restore"] = 30 },
        [13] = { ["on_parry.1.mp_restore"] = 33 },
        [14] = { ["on_parry.1.mp_restore"] = 36 },
        [15] = { ["on_parry.1.mp_restore"] = 40 },
        [16] = { ["on_parry.1.mp_restore"] = 43 },
        [17] = { ["on_parry.1.mp_restore"] = 47 },
        [18] = { ["on_parry.1.mp_restore"] = 51 },
        [19] = { ["on_parry.1.mp_restore"] = 55 },
        [20] = { ["on_parry.1.mp_restore"] = 59 }
    }
}

E["CS_wd_tjj08"] = {
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push",
            mp_drain = true,
            mp_drain_dur_ms = 0
        }
    }
}

E["CS_wd_tjj09"] = {
    on_hit = {
        {
            control_dur_ms = 5000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_wd_tjq01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_wd_tjq01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_wd_tjq02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_dur_ms = 5000,
            control_kind = "launch"
        }
    }
}

E["CS_wd_tjq03"] = {
    unparsed = { "若在自身格挡成功后5秒内使用，可延长：敌人被破防后无法招架的时间" }
}

E["CS_wd_tjq04"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_wd_tjq04_3",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_wd_tjq05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_wd_tjq05",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_wd_tjq06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    unparsed = { "强劲力道可击落所有妄图近身的空中单位", "若在成功格挡后5秒内出招，可使敌人双足虚软：无法跳跃和使用轻功" }
}

E["CS_wd_tjq07"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_wd_tjq07",
            apply_resolved = true,
            apply_stacks = 1,
            heal = true,
            mp_restore = 0
        }
    }
}

E["CS_wdqj_tjj01"] = {
    compile = {
        max_hits = 2,
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_wdqj_tjj02"] = {
    unparsed = { "独特调息心法，可将气血和内力的比例互换" }
}

E["CS_wdqj_tjj03"] = {
    unparsed = { "独特吐纳心法，可在短时间内迅速恢复一定内力" }
}

E["CS_wdqj_tjj04"] = {
    unparsed = { "纯阳一击，剑气波及到的敌人受到{@0:伤害包}，并致其内伤：移动速度减缓，生命持续损耗" }
}

E["CS_wdqj_tjj05"] = {
    on_hit = {
        {
            condition = { parry_broken = true },
            mp_restore = 0
        }
    }
}

E["CS_wdqj_tjj06"] = {
    on_parry = {
        {
            mp_restore = 7
        }
    },
    by_level = {
        [2] = { ["on_parry.1.mp_restore"] = 11 },
        [3] = { ["on_parry.1.mp_restore"] = 15 }
    }
}

E["CS_wdqj_tjj07"] = {
    compile = {
        cast_armor = "red"
    }
}

E["CS_wdqj_tjj08"] = {
    on_hit = {
        {
            displacement_distance_m = 0,
            displacement_kind = "push",
            mp_drain = true,
            mp_drain_dur_ms = 0
        }
    }
}

E["CS_wdqj_tjj09"] = {
    on_hit = {
        {
            control_dur_ms = 5000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_wdzp_tjq01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_wdzp_tjq01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_wdzp_tjq02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 5000,
            control_kind = "launch"
        }
    }
}

E["CS_wdzp_tjq03"] = {
    unparsed = { "若在自身格挡成功后5秒内使用，可延长：敌人被破防后无法招架的时间" }
}

E["CS_wdzp_tjq04"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_wdzp_tjq04_3",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_wdzp_tjq05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_wdzp_tjq05",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_wdzp_tjq06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    unparsed = { "强劲力道可击落所有妄图近身的空中单位", "若在成功格挡后5秒内出招，可使敌人双足虚软：无法跳跃和使用轻功" }
}

E["CS_wdzp_tjq07"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_wdzp_tjq07",
            apply_resolved = true,
            apply_stacks = 1,
            heal = true,
            mp_restore = 0
        }
    }
}

E["CS_wh_hxw01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "一层“胡旋",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_wh_hxw02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_wh_hxw03"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_wh_hxw04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_wh_hxw05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 },
        cast_armor = "yellow"
    }
}

E["CS_wh_hxw06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "一层“胡旋",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_wh_hxw07"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            apply_buff = "buf_CS_wh_hxw08",
            apply_resolved = true,
            apply_stacks = 3,
            control_dur_ms = 2000,
            control_kind = "root"
        }
    }
}

E["CS_wh_hxw08"] = {
    on_hit = {
        {
            apply_buff = "一层“胡旋",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    by_level = {
        [3] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [4] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [5] = { ["on_hit.1.apply_life_ms"] = 6000 },
        [6] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [7] = { ["on_hit.1.apply_life_ms"] = 7000 },
        [8] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [9] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [10] = { ["on_hit.1.apply_life_ms"] = 8000 },
        [11] = { ["on_hit.1.apply_life_ms"] = 9000 },
        [12] = { ["on_hit.1.apply_life_ms"] = 10000 }
    }
}

E["CS_wh_hyj01"] = {
    on_hit = {
        {
            control_kind = "silence"
        }
    }
}

E["CS_wh_hyj02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_wh_hyj_mark_cd",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 1000,
            control_kind = "coma"
        }
    }
}

E["CS_wh_hyj03"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_wh_hyj04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "一层“幻欲",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_wh_hyj05"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_wh_hyj05",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 1000,
            control_kind = "coma"
        }
    }
}

E["CS_wh_hyj06"] = {
    on_hit = {
        {
            apply_buff = "一层“幻欲",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_wh_hyj07"] = {
    on_hit = {
        {
            apply_buff = "一层“幻欲",
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_wh_hyj08"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_wh_hyj_mark_cd",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "coma"
        }
    },
    unparsed = { "最大冲刺距离15米）" }
}

E["CS_wqb_gmfs01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            apply_buff = "一层啄目效果",
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        },
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_wqb_gmfs01_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_wqb_gmfs02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "root"
        }
    }
}

E["CS_wqb_gmfs03"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_wqb_gmfs04"] = {
    on_hit = {
        {
            apply_buff = "一层雷霆效果",
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 9000,
            control_kind = "slow",
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    unparsed = { "召唤大型傀儡护持己身：每次格挡成功时，获得1层护持" }
}

E["CS_wqb_gmfs05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        },
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_wqb_gmfs06"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    }
}

E["CS_wqb_gmfs08"] = {
    compile = {
        cast_armor = "yellow"
    }
}

E["CS_wxj_gyxhj01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_wxj_gyxhj02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_wxj_gyxhj02_01",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_wxj_gyxhj03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_wxj_gyxhj03_01",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_wxj_gyxhj04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    self_state = {
        {
            apply_buff = "buf_CS_wxj_gyxhj04_01",
            apply_life_ms = 12000,
            apply_resolved = true
        }
    },
    unparsed = { "亦可将真气打回敌人体内，震退未招架的目标一小段距离，使其3秒内无法招架并无法出招" }
}

E["CS_wxj_gyxhj04_hide"] = {
    unparsed = { "聚势挥掌，旋气外击，将真气打回敌人体内，造成{@0:伤害包}，震退未招架的目标一小段距离，使其3" }
}

E["CS_wxj_gyxhj05"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    },
    unparsed = { "（此招式不可在空中施展）（装备双环发挥100%招式威力，装备双刺只能发挥70%招式威力）" }
}

E["CS_wxj_gyxhj05_hide_01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    },
    unparsed = { "（此招式不可在空中施展）（装备双环发挥100%招式威力，装备双刺只能发挥70%招式威力）" }
}

E["CS_wxj_gyxhj05_hide_02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    },
    unparsed = { "（此招式不可在空中施展）（装备双环发挥100%招式威力，装备双刺只能发挥70%招式威力）" }
}

E["CS_wxj_gyxhj06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "灵蛊噬体",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_wxj_qzwds01"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_wxj_qzwds02"] = {
    on_hit = {
        {
            apply_buff = "毒蛛吐丝持续6秒",
            apply_life_ms = 6000,
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "毒蛛吐丝叠加3次后，3秒内若目标尝试出招则被打断" }
}

E["CS_wxj_qzwds03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_wxj_qzwds04"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    }
}

E["CS_wxj_qzwds06"] = {
    on_parry = {
        {
            apply_buff = "buf_CS_wxj_qzwds06",
            apply_life_ms = 7000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_wxj_qzwds07"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_wxj_qzwds07",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1
        },
        {
            apply_buff = "buf_CS_wxj_qzwds02_1",
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_wxj_qzwds08"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_xd_dyshd02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    unparsed = { "招式首次命中目标，自身获得1层“魂”状态，每层“魂”可化解一击伤害的50%" }
}

E["CS_xd_dyshd03"] = {
    unparsed = { "摄威擅势，公道不行，招架成功时，自身获得“威”状态：本套路招式伤害提升2%，可叠加2层" }
}

E["CS_xd_dyshd04"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_parry_break = {
        {
            control_dur_ms = 4000,
            control_kind = "root"
        }
    }
}

E["CS_xd_dyshd05"] = {
    unparsed = { "12秒内，攻克玩家恢复自身损失气血的50%和损耗内力的50%，并获得3层“魂”状态，每层“魂”可" }
}

E["CS_xd_dyshd06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    unparsed = { "当伤害目标之后，自身将获得2层“魂”状态，每层“魂”可化解一击伤害的50%" }
}

E["CS_xd_dyshd07"] = {
    unparsed = { "招式首次命中目标，自身获得2层“魂”状态，每层“魂”可化解一击伤害的50%" }
}

E["CS_xd_xdj02"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        }
    }
}

E["CS_xd_xdj03"] = {

}

E["CS_xd_xdj04"] = {
    on_hit = {
        {
            control_kind = "stun"
        }
    },
    unparsed = { "招架成功次数越多，招式【血·歌】伤害越高" }
}

E["CS_xd_xdj04_hide"] = {
    on_hit = {
        {
            control_kind = "stun"
        }
    }
}

E["CS_xd_xdj05"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_xd_xdj06"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_xd_xdj07"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow",
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "stun"
        }
    }
}

E["CS_xj01"] = {
    on_hit = {
        {
            control_kind = "root"
        }
    }
}

E["CS_xjz_wmcf01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_xjz_wmcf01",
            apply_resolved = true,
            apply_stacks = 1,
            control_kind = "coma"
        }
    }
}

E["CS_xjz_wmcf02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_xjz_wmcf03"] = {
    self_state = {
        {
            apply_buff = "buf_CS_xjz_wmcf03",
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，可持续5秒，对霸体无效，可被虚招克制）（提升招式层数，可增加反击伤害）（装备尺发挥1" }
}

E["CS_xjz_wmcf03_hide"] = {
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_xjz_wmcf04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_xjz_wmcf04",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 5000,
            control_kind = "ban_qinggong"
        }
    },
    on_parry_break = {
        {
            heal = true
        }
    }
}

E["CS_xjz_wmcf05"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "root"
        }
    }
}

E["CS_xjz_wmcf06"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 20 }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_xjz_wmcf07"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            control_kind = "coma"
        }
    }
}

E["CS_xmg_pyj01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 10 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 5000,
            control_kind = "slow"
        }
    }
}

E["CS_xmg_pyj02"] = {
    unparsed = { "招架成功时将一缕“磐隐”暗劲打入敌方体内潜伏10秒，最多可叠加7层", "“磐隐”暗劲：被【横卧沧海】攻击时，将被短暂僵直" }
}

E["CS_xmg_pyj03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_xmg_pyj04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_xmg_pyj05"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_xmg_pyj06"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_xmg_pyj07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    unparsed = { "技能释放完成时剑气倾泻，将仍处于自身周围的未招架敌人远远震开" }
}

E["CS_xmg_pyj08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "stiffen",
            per_hit = true
        }
    }
}

E["CS_xmg_xtxjf01"] = {
    unparsed = { "若目标身上“星河鹭起”状态叠加到5层则会引爆，造成伤害并不辨方向3秒" }
}

E["CS_xmg_xtxjf02"] = {
    on_parry_break = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_xmg_xtxjf03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_xmg_xtxjf04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_xmg_xtxjf05"] = {
    on_hit = {
        {
            apply_buff = "落晓星沉",
            apply_life_ms = 10000,
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "root"
        }
    }
}

E["CS_xmg_xtxjf06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_xmg_xtxjf07_01",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_xmg_xtxjf07"] = {
    on_hit = {
        {
            control_dur_ms = 5000,
            control_kind = "slow"
        }
    },
    by_level = {
        [2] = { ["on_hit.1.control_dur_ms"] = 6000 },
        [3] = { ["on_hit.1.control_dur_ms"] = 6000 },
        [4] = { ["on_hit.1.control_dur_ms"] = 7000 },
        [5] = { ["on_hit.1.control_dur_ms"] = 7000 },
        [6] = { ["on_hit.1.control_dur_ms"] = 8000 },
        [7] = { ["on_hit.1.control_dur_ms"] = 8000 },
        [8] = { ["on_hit.1.control_dur_ms"] = 9000 },
        [9] = { ["on_hit.1.control_dur_ms"] = 9000 },
        [10] = { ["on_hit.1.control_dur_ms"] = 10000 }
    }
}

E["CS_xt_gqml01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "一层“复山河",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_xt_gqml02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_xt_gqml02",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_xt_gqml03"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_xt_gqml03_2",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "coma",
            heal = true
        },
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        },
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "silence"
        }
    }
}

E["CS_xt_gqml03_hide"] = {
    compile = {
        air = "never"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_xt_gqml03_sky"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "silence"
        }
    }
}

E["CS_xt_gqml04"] = {
    unparsed = { "快速对周身范围内的敌人造成{@0:伤害包}，若敌人存在“重击撕裂”效果，则所拥有的“重击撕裂”的" }
}

E["CS_xt_gqml05"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_xt_gqml05",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_kind = "dizzy"
        },
        {
            apply_buff = "buf_CS_xt_gqml05",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_kind = "dizzy"
        },
        {
            apply_buff = "buf_CS_xt_gqml02",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 5000,
            control_kind = "knockdown"
        },
        {
            control_kind = "coma"
        }
    }
}

E["CS_xt_gqml05_hide1"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_xt_gqml05",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_kind = "dizzy"
        }
    }
}

E["CS_xt_gqml05_hide2"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_xt_gqml02",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true },
            control_dur_ms = 5000,
            control_kind = "knockdown"
        }
    }
}

E["CS_xt_gqml06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 1000,
            control_kind = "launch"
        },
        {
            apply_buff = "不同层数的“重击撕裂",
            apply_life_ms = 10000,
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_xt_gqml07"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_parry_break = {
        {
            apply_buff = "buf_CS_xt_gqml05",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 2000,
            control_kind = "silence"
        }
    }
}

E["CS_xt_gqml08"] = {
    unparsed = { "招架成功自身获得“志斗战”状态：减少10%受到的外功伤害，持续5秒，可叠加1层" }
}

E["CS_xt_gqml09"] = {
    on_hit = {
        {
            apply_buff = "15秒的“定太平",
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 1000,
            control_kind = "knockdown"
        }
    }
}

E["CS_xt_jtq01"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_xt_jtq02",
            apply_resolved = true,
            apply_stacks = 3,
            control_kind = "knockdown"
        }
    }
}

E["CS_xt_jtq02"] = {
    on_hit = {
        {
            apply_buff = "一层“拳踢相应",
            apply_life_ms = 10000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_xt_jtq03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_xt_jtq04"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_xt_jtq05"] = {
    on_hit = {
        {
            apply_buff = "一层“拳踢相应",
            apply_resolved = false,
            apply_stacks = 1,
            chance = 0.2
        }
    },
    by_level = {
        [2] = { ["on_hit.1.chance"] = 0.25 },
        [3] = { ["on_hit.1.chance"] = 0.3 },
        [4] = { ["on_hit.1.chance"] = 0.35 },
        [5] = { ["on_hit.1.chance"] = 0.4 },
        [6] = { ["on_hit.1.chance"] = 0.5 },
        [7] = { ["on_hit.1.chance"] = 0.55 },
        [8] = { ["on_hit.1.chance"] = 0.6 },
        [9] = { ["on_hit.1.chance"] = 0.65 },
        [10] = { ["on_hit.1.chance"] = 0.7 },
        [11] = { ["on_hit.1.chance"] = 0.75 },
        [12] = { ["on_hit.1.chance"] = 0.8 },
        [13] = { ["on_hit.1.chance"] = 0.9 }
    }
}

E["CS_xt_jtq06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    },
    unparsed = { "若目标未招架且身中3层以上“拳踢相应”则招式第一击可使其短暂僵直" }
}

E["CS_xt_jtq07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_xt_xjj01"] = {
    on_hit = {
        {
            apply_buff = "一层“黄沙百战",
            apply_life_ms = 5000,
            apply_resolved = false,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_xt_xjj02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_xt_xjj03"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "dizzy"
        },
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_xt_xjj03_sky"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_xt_xjj04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 5000,
            control_kind = "slow"
        }
    }
}

E["CS_xt_xjj05"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_xt_xjj05",
            apply_life_ms = 10000,
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        },
        {
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    }
}

E["CS_xt_xjj06"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_xt_xjj07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    }
}

E["CS_xt_xjj08"] = {
    self_state = {
        {
            apply_buff = "buf_CS_by_gxcj04",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "（防反招式，有效范围6米，对霸体无效，可被虚招克制", "提升招式层数，可增加反击伤害）" }
}

E["CS_xt_xjj08_hide"] = {
    unparsed = { "招架，防反" }
}

E["CS_xt_xjl01"] = {
    on_parry_break = {
        {
            apply_buff = "一层“回浪连拍",
            apply_resolved = false,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_xt_xjl02"] = {
    unparsed = { "对前方敌人造成{@0:伤害包}，自身存在“甲兵突进”状态时命中未招架目标使其缴械2秒（每15秒才" }
}

E["CS_xt_xjl03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "一层持续10秒的“甲兵突进",
            apply_life_ms = 10000,
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_xt_xjl04"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "launch"
        },
        {
            condition = { unparried = true },
            control_kind = "dizzy",
            per_hit = true
        }
    }
}

E["CS_xt_xjl05"] = {
    on_hit = {
        {
            apply_buff = "一层持续10秒的“甲兵突进",
            apply_life_ms = 10000,
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    self_state = {
        {
            apply_buff = "buf_CS_xt_xjl05",
            apply_life_ms = 5000,
            apply_resolved = true
        }
    },
    unparsed = { "提升招式层数，可增加反击伤害）" }
}

E["CS_xt_xjl05_hide"] = {
    on_hit = {
        {
            apply_buff = "一层持续10秒的“甲兵突进",
            apply_life_ms = 10000,
            apply_resolved = false,
            apply_stacks = 1
        }
    }
}

E["CS_xt_xjl06"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_xt_xjl06",
            apply_resolved = true,
            apply_stacks = 1,
            condition = { unparried = true }
        }
    }
}

E["CS_xt_xjl07"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        },
        {
            apply_buff = "buf_CS_xt_xjl06",
            apply_resolved = true,
            apply_stacks = 1,
            control_kind = "knockdown"
        }
    }
}

E["CS_xt_xjl08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 },
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 3000,
            control_kind = "root"
        }
    },
    unparsed = { "稚元:怒气消耗降低5点" }
}

E["CS_xt_yjjx01"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 5000,
            control_kind = "knockdown"
        }
    }
}

E["CS_xt_yjjx01_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    }
}

E["CS_xt_yjjx02"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        },
        {
            control_dur_ms = 2000,
            control_kind = "silence"
        }
    }
}

E["CS_xt_yjjx02_hide"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_xt_yjjx03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_xt_yjjx03_hide"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_xt_yjjx04"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    },
    unparsed = { "【“双持状态” ：招式变为15米单体技能】" }
}

E["CS_xt_yjjx04_hide"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "silence",
            displacement_distance_m = 0,
            displacement_kind = "pull"
        }
    }
}

E["CS_xt_yjjx05"] = {
    on_hit = {
        {
            apply_buff = "一层“风云变",
            apply_resolved = false,
            apply_stacks = 1
        }
    },
    unparsed = { "反震伤害受内功威力加成，提升招式层数可提高反震伤害）" }
}

E["CS_xt_yjjx06"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    unparsed = { "拆解手中长斧进入“双持状态”：套路部分招式发生变化且在15秒内每次使用变化的招式都会获得一层“风" }
}

E["CS_xt_yjjx06_hide"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    unparsed = { "解除“双持状态”且在15秒内使用本套路任意招式都可获得一层“风云变”状态" }
}

E["CS_xt_yjjx07"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_xt_yjjx08"] = {
    compile = {
        cast_armor = "yellow"
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "dizzy",
            per_hit = true
        }
    },
    unparsed = { "三层“风云变”状态下可使用恢复自身150点气血（受内功威力加成），15秒内套路内部分招式可消耗“" }
}

E["CS_xt_yjjx09"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "silence"
        }
    }
}

E["CS_xt_yjjx09_hide"] = {
    on_hit = {
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_xy_jlsd01"] = {
    unparsed = { "招式命中敌人自身获得“雷霆”状态：本套路招式伤害提高10%，可以叠加3层，持续7秒" }
}

E["CS_xy_jlsd02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_xy_jlsd03"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            condition = { unparried = true },
            control_dur_ms = 2000,
            control_kind = "coma"
        }
    }
}

E["CS_xy_jlsd04"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_dur_ms = 4000,
            control_kind = "launch"
        }
    }
}

E["CS_xy_jlsd04_1"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_dur_ms = 4000,
            control_kind = "launch"
        }
    }
}

E["CS_xy_jlsd04_2"] = {
    compile = {
        aoe = { aoe_shape = "circle" }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_xy_jlsd05"] = {
    on_parry_break = {
        {
            control_dur_ms = 3000,
            control_kind = "silence"
        }
    }
}

E["CS_xy_jlsd06"] = {
    compile = {
        cast_armor = "yellow"
    }
}

E["CS_xy_jlsd07"] = {
    compile = {
        cast_armor = "yellow"
    },
    self_state = {
        {
            apply_buff = "buf_CS_xy_jlsd07",
            apply_life_ms = 10000,
            apply_resolved = true
        }
    }
}

E["CS_xy_jlsd07_hide"] = {
    compile = {
        cast_armor = "yellow"
    }
}

E["CS_xy_ktdj02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_xy_ktdj03"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_xy_ktdj03_1",
            apply_life_ms = 7000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_xy_ktdj04"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_xy_ktdj04",
            apply_life_ms = 7000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_xy_ktdj05"] = {
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "silence"
        }
    }
}

E["CS_xy_ktdj06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "root",
            displacement_distance_m = 0,
            displacement_kind = "push"
        },
        {
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    }
}

E["CS_xy_ktdj07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 },
        cast_armor = "yellow",
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            apply_buff = "buf_CS_xy_ktdj07_01",
            apply_life_ms = 2000,
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 2000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_xy_nrjbd02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_xy_nrjbd03"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_xy_nrjbd03",
            apply_life_ms = 20000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_xy_nrjbd04"] = {
    compile = {
        aoe = { aoe_shape = "circle" },
        cast_armor = "yellow",
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "stun"
        },
        {
            condition = { unparried = true },
            control_kind = "knockdown"
        }
    }
}

E["CS_xy_nrjbd05"] = {
    on_parry_break = {
        {
            control_kind = "knockdown"
        }
    }
}

E["CS_xy_nrjbd06"] = {
    unparsed = { "一抔净土掩风流，招架成功时，有一定几率清除本套路所有招式回气时间" }
}

E["CS_xy_nrjbd07"] = {
    on_hit = {
        {
            control_dur_ms = 4000,
            control_kind = "ban_qinggong"
        }
    }
}

E["CS_xy_shl01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_xy_shl04",
            apply_life_ms = 20000,
            apply_resolved = true,
            apply_stacks = 2
        }
    }
}

E["CS_xy_shl01_1"] = {
    unparsed = { "施展招式时将掷出一团火焰，火焰飞行中碰触的敌人将被击晕2秒" }
}

E["CS_xy_shl02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_xy_shl03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_xy_shl04"] = {
    on_parry_break = {
        {
            apply_buff = "buf_CS_xy_shl04",
            apply_life_ms = 20000,
            apply_resolved = true,
            apply_stacks = 2
        }
    },
    by_level = {
        [3] = { ["on_parry_break.1.apply_stacks"] = 3 },
        [4] = { ["on_parry_break.1.apply_stacks"] = 3 },
        [5] = { ["on_parry_break.1.apply_stacks"] = 4 },
        [6] = { ["on_parry_break.1.apply_stacks"] = 4 },
        [7] = { ["on_parry_break.1.apply_stacks"] = 5 },
        [8] = { ["on_parry_break.1.apply_stacks"] = 5 },
        [9] = { ["on_parry_break.1.apply_stacks"] = 5 },
        [10] = { ["on_parry_break.1.apply_stacks"] = 5 },
        [11] = { ["on_parry_break.1.apply_stacks"] = 5 },
        [12] = { ["on_parry_break.1.apply_stacks"] = 5 }
    }
}

E["CS_xy_shl05"] = {
    unparsed = { "-处所庄严皆清净，诸恶不净彼元无-沉水入火，自取灭亡，招架成功6次后：反弹自身所受伤害的20%至" }
}

E["CS_xy_shl06"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 6 }
    },
    unparsed = { "施展招式时将掷出一团火焰，火焰飞行中碰触的敌人将被伤害并击晕3秒" }
}

E["CS_xy_shl07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 15 }
    },
    on_hit = {
        {
            control_dur_ms = 2000,
            control_kind = "root"
        },
        {
            control_kind = "launch"
        }
    }
}

E["CS_xy_shl07_hide"] = {
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_xy_shl08"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 6 },
        cast_armor = "red",
        can_qinggong = true,
        rush_note = true
    },
    on_hit = {
        {
            control_kind = "stun"
        }
    }
}

E["CS_yh_hsqs01"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_yh_hsqs01",
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.1
        }
    },
    by_level = {
        [3] = { ["on_hit.1.chance"] = 0.15 },
        [4] = { ["on_hit.1.chance"] = 0.15 },
        [5] = { ["on_hit.1.chance"] = 0.2 },
        [6] = { ["on_hit.1.chance"] = 0.2 },
        [7] = { ["on_hit.1.chance"] = 0.25 },
        [8] = { ["on_hit.1.chance"] = 0.25 },
        [9] = { ["on_hit.1.chance"] = 0.3 },
        [10] = { ["on_hit.1.chance"] = 0.3 },
        [11] = { ["on_hit.1.chance"] = 0.3 },
        [12] = { ["on_hit.1.chance"] = 0.3 },
        [13] = { ["on_hit.1.chance"] = 0.3 },
        [14] = { ["on_hit.1.chance"] = 0.3 },
        [15] = { ["on_hit.1.chance"] = 0.3 },
        [16] = { ["on_hit.1.chance"] = 0.3 },
        [17] = { ["on_hit.1.chance"] = 0.3 },
        [18] = { ["on_hit.1.chance"] = 0.3 },
        [19] = { ["on_hit.1.chance"] = 0.3 }
    }
}

E["CS_yh_hsqs03"] = {
    unparsed = { "破防成功时，给目标加“蝶芒”状态（眼花缭乱，下次攻击有较大几率被闪避）和“花颦”状态（长时间不可" }
}

E["CS_yh_hsqs04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_yh_hsqs04_1",
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.1,
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    by_level = {
        [3] = { ["on_hit.1.chance"] = 0.15 },
        [4] = { ["on_hit.1.chance"] = 0.15 },
        [5] = { ["on_hit.1.chance"] = 0.2 },
        [6] = { ["on_hit.1.chance"] = 0.2 },
        [7] = { ["on_hit.1.chance"] = 0.25 },
        [8] = { ["on_hit.1.chance"] = 0.25 },
        [9] = { ["on_hit.1.chance"] = 0.3 },
        [10] = { ["on_hit.1.chance"] = 0.35 }
    }
}

E["CS_yh_hsqs05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 10 },
        cast_armor = "yellow",
        air = "never"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_yh_hsqs05",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_yh_hsqs06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_yh_hsqs06",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "root"
        },
        {
            control_dur_ms = 3000,
            control_kind = "stun"
        }
    }
}

E["CS_yhwq_hsqs01"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_yhwq_hsqs01",
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.3
        }
    }
}

E["CS_yhwq_hsqs02"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_yhwq_hsqs01",
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.1
        }
    }
}

E["CS_yhwq_hsqs03"] = {
    unparsed = { "破防成功时，给目标加“蝶芒”状态（眼花缭乱，下次攻击有较大几率被闪避）和“花颦”状态（长时间不可" }
}

E["CS_yhwq_hsqs04"] = {
    on_hit = {
        {
            apply_buff = "buf_CS_yhwq_hsqs04_1",
            apply_resolved = true,
            apply_stacks = 1,
            chance = 0.1,
            displacement_distance_m = 0,
            displacement_kind = "push"
        }
    },
    by_level = {
        [3] = { ["on_hit.1.chance"] = 0.15 },
        [4] = { ["on_hit.1.chance"] = 0.15 },
        [5] = { ["on_hit.1.chance"] = 0.2 },
        [6] = { ["on_hit.1.chance"] = 0.2 },
        [7] = { ["on_hit.1.chance"] = 0.25 },
        [8] = { ["on_hit.1.chance"] = 0.25 },
        [9] = { ["on_hit.1.chance"] = 0.3 },
        [10] = { ["on_hit.1.chance"] = 0.3 }
    }
}

E["CS_yhwq_hsqs05"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 20 },
        cast_armor = "red"
    },
    on_hit = {
        {
            apply_buff = "buf_CS_yhwq_hsqs05",
            apply_life_ms = 5000,
            apply_resolved = true,
            apply_stacks = 1
        }
    }
}

E["CS_yhwq_hsqs06"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 }
    },
    on_hit = {
        {
            apply_buff = "buf_CS_yhwq_hsqs06",
            apply_resolved = true,
            apply_stacks = 1,
            control_dur_ms = 3000,
            control_kind = "root"
        },
        {
            control_dur_ms = 3000,
            control_kind = "stun"
        }
    }
}

E["CS_ym_mwcq01"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    unparsed = { "命中未招架目标可打散其25%淤血" }
}

E["CS_ym_mwcq02"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_ym_mwcq03"] = {
    compile = {
        can_qinggong = true,
        rush_note = true
    }
}

E["CS_ym_mwcq04"] = {
    on_hit = {
        {
            apply_buff = "一层“化木",
            apply_life_ms = 15000,
            apply_resolved = false,
            apply_stacks = 1,
            heal = true
        }
    },
    unparsed = { "当自身存在8层“化木”时，再次施展招式，清空自身“化木”层数，可清除自身大部分负面状态，并恢复自" }
}

E["CS_ym_mwcq04_hide"] = {
    on_hit = {
        {
            apply_buff = "一层“化木",
            apply_life_ms = 15000,
            apply_resolved = false,
            apply_stacks = 1,
            heal = true
        }
    },
    unparsed = { "当自身存在8层“化木”时，再次施展招式，清空自身“化木”层数，可清除自身大部分负面状态，并恢复自" }
}

E["CS_ym_mwcq05"] = {
    on_hit = {
        {
            heal = true
        }
    },
    unparsed = { "笔势化劲点击目标造成{@0:伤害包}，命中未招架敌人使其命中率降低15%，效果持续6秒，期间会持" }
}

E["CS_ym_mwcq05_hide"] = {
    on_hit = {
        {
            heal = true
        }
    },
    unparsed = { "笔势化劲点击目标造成伤害，命中未招架敌人使其命中率降低15%，效果持续6秒，期间会持续流失气血" }
}

E["CS_ym_mwcq06"] = {
    compile = {
        aoe = { aoe_shape = "sector", aoe_radius_m = 5 }
    },
    on_hit = {
        {
            control_kind = "launch"
        }
    }
}

E["CS_ym_mwcq07"] = {
    compile = {
        aoe = { aoe_shape = "circle", aoe_radius_m = 8 },
        cast_armor = "yellow"
    },
    unparsed = { "气场范围内敌对目标使用轻功有几率摔倒（滞留阵法时间越长，摔倒概率越大），且每秒会受到持续伤害" }
}

E["CS_ym_mwcq08"] = {
    on_hit = {
        {
            control_dur_ms = 3000,
            control_kind = "ban_qinggong",
            heal = true
        }
    },
    unparsed = { "御笔画地，在原地布下一个半径6米的气场，持续10秒", "当自身存在“化木”时，可在气场中持续恢复气血，每次恢复会消耗一层“化木”" }
}

return E
