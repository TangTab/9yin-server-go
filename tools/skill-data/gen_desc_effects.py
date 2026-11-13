#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""gen_desc_effects.py —— 技能描述 → 受控词表 IR（desc_effects.lua）生成器。

数据源（默认自 NINEYIN_SERVER_ROOT 或脚本相对路径 ../../resources 探测）：
  text/chineses/string_skill_desc.idres   desc_<技能ID>_<等级>=机制描述
  text/chineses/string_buff_desc.idres    buf_<id>=状态中文名（反查）
  share/skill/buff_new.ini                buff 行（StaticData/LifeTime 佐证）

产物：scripts/module/skill/desc_effects.lua（幂等可重跑，勿手改）。
IR 权威定义：docs/skill-desc-ir.md（动词三分类：控制状态/位移/资源）。

用法：
  python gen_desc_effects.py                # 全量生成 + 覆盖率报表
  python gen_desc_effects.py --selftest     # 金样本断言（旋刀/太极剑/寒心恨雪族）
  python gen_desc_effects.py --root <dir>   # 显式指定 resources 根

设计要点：
  - 描述自带语义标注：<font color="#f9fc00">黄=动作词、#FFAE00/#FF3030=状态名；
    颜色优先于裸文本匹配（状态名只在引号/彩色段里反查 buff 配置）。
  - 解析不了的句子不猜：整段进 unparsed 留痕（含关键词却无模板命中的段落）。
  - 分级差集：全部等级各自解析，L1 为基准，数值字段变化进 by_level
    （路径平铺，如 max_hits / on_hit[1].apply_state.life_ms）。
"""

import argparse
import os
import re
import sys
from collections import defaultdict

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
DEFAULT_OUT = os.path.join(REPO_ROOT, "scripts", "module", "skill", "desc_effects.lua")

# ─────────────────────────── 数据装载 ───────────────────────────


def find_resources_root(explicit):
    """资源根 = 含 text/chineses 与 share/skill 的目录（兼容 …/resources 与
    …/resources/modern 两种布局）。探测序：--root > 环境变量 >
    env.server（部署脚本写法的 NINEYIN_SERVER_ROOT）> 仓内 resources。"""
    marker = os.path.join("text", "chineses", "string_skill_desc.idres")
    candidates = []
    if explicit:
        candidates.extend([explicit, os.path.join(explicit, "modern")])
    env = os.environ.get("NINEYIN_SERVER_ROOT", "")
    if not env:
        env = env_server_root()
    if env:
        candidates.extend([os.path.join(env, "resources"),
                           os.path.join(env, "resources", "modern")])
    candidates.extend([os.path.join(REPO_ROOT, "resources"),
                       os.path.join(REPO_ROOT, "resources", "modern")])
    for cand in candidates:
        if os.path.isfile(os.path.join(cand, marker)):
            return cand
    raise SystemExit("找不到资源根（需要 text/chineses/string_skill_desc.idres）；"
                     "用 --root 指定 <server>/resources 或 …/resources/modern 目录")


def env_server_root():
    """从仓内 env.server 读 NINEYIN_SERVER_ROOT（set "KEY=VALUE" 批处理写法）。"""
    path = os.path.join(REPO_ROOT, "env.server")
    if not os.path.isfile(path):
        return ""
    try:
        with open(path, "r", encoding="utf-8", errors="replace") as fh:
            for line in fh:
                m = re.search(r'NINEYIN_SERVER_ROOT=([^"\r\n]+)"?', line.strip())
                if m:
                    return m.group(1).strip().strip('"')
    except OSError:
        pass
    return ""


def load_idres(path):
    """idres 行式表（UTF-8 BOM + CRLF，KEY=VALUE）→ dict。"""
    table = {}
    with open(path, "r", encoding="utf-8-sig") as fh:
        for line in fh:
            line = line.rstrip("\r\n")
            if not line or line.startswith("=") or "=" not in line:
                continue
            key, _, value = line.partition("=")
            table[key.strip()] = value
    return table


def load_buff_name_index(path):
    """string_buff_desc：中文名 → [buf_id, …]（多对一由族前缀消歧）。"""
    index = defaultdict(list)
    for buf_id, name in load_idres(path).items():
        m = re.match(r"^(buf_[A-Za-z0-9_]+)$", buf_id)
        if m and name:
            index[name].append(buf_id)
    return index


# ─────────────────────── 描述预处理（颜色→语义） ───────────────────────

FONT_RE = re.compile(r'<font\s+color="#([0-9A-Fa-f]{6})"\s*>(.*?)</font>', re.S)
TAG_RE = re.compile(r"<[^>]+>")
PLACEHOLDER_RE = re.compile(r"\{@\d+:[^}]*\}")

ACTION_COLOR = "f9fc00"     # 黄 = 动作词（击飞/眩晕/倒地/拉近/霸体…）
STATE_COLORS = ("FFAE00", "FF3030")  # 橙/红 = 状态名（引号内）


def preprocess(text):
    """返回 (纯文本, 状态名列表, 动作词列表)。颜色标签折为纯文本但先抽取语义段。"""
    states, actions = [], []

    def grab(m):
        color, inner = m.group(1).lower(), m.group(2)
        if color == ACTION_COLOR:
            actions.append(inner)
        elif color in STATE_COLORS:
            states.append(inner)
        return inner

    plain = FONT_RE.sub(grab, text)
    plain = TAG_RE.sub("", plain)
    plain = plain.replace("\r", "").replace("\n", "")
    return plain, states, actions


def split_segments(plain):
    """按句读切分；<br> 已被 TAG_RE 去掉，按 。；！？ 与中文分号切段。"""
    raw = re.split(r"[。；！？]", plain)
    return [seg.strip() for seg in raw if seg.strip()]


# ─────────────────────── 数值与词表 ───────────────────────

NUM = r"(\d+(?:\.\d+)?)"

# 控制状态词表（kind → 触发词；词按长度降序避免前缀吞并）
CONTROL_KINDS = {
    "stun":        ["眩晕"],
    "coma":        ["昏迷"],
    "dizzy":       ["击晕"],
    "launch":      ["击飞"],
    "knockdown":   ["击倒", "倒地"],
    "root":        ["定身"],
    "silence":     ["封招"],
    "ban_qinggong": ["无法使用轻功", "禁轻功", "无法使用轻功"],
    "slow":        ["移动速度减缓", "减速"],
    "stiffen":     ["僵直"],
}
DISPLACEMENT_WORDS = {"击退": "push", "拉近": "pull"}
# 关键词监控表：段落含这些词但无模板命中 → unparsed
WATCH_WORDS = [
    "眩晕", "昏迷", "击晕", "击飞", "击倒", "倒地", "定身", "封招", "禁轻功",
    "轻功", "僵直", "击退", "拉近", "霸体", "无敌", "持续伤害", "持续损耗",
    "回复", "恢复", "内力", "怒气", "每击", "每秒", "叠加", "添加", "生成",
    "留下", "召唤", "空中", "招架", "破防", "几率", "概率", "层", "米",
]

# 套路口诀/纯描写段（绿色段开头"-…-"）与旁注括号的切分
PAREN_RE = re.compile(r"（([^）]*)）")


def parse_num(pattern, text, cast=float):
    """首个非空捕获组（多分支 alternation 时命中的组序不定）。"""
    m = re.search(pattern, text)
    if not m:
        return None
    raw = next((g for g in m.groups() if g is not None), None)
    if raw is None:
        return None
    value = float(raw)
    return value if cast is float else int(value)


def find_control(text):
    """文本中的控制词 → (kind, 触发词)。多词命中按词长优先取第一个。"""
    hits = []
    for kind, words in CONTROL_KINDS.items():
        for word in words:
            if word in text:
                hits.append((len(word), word, kind))
    if not hits:
        return None
    hits.sort(reverse=True)
    return hits[0][2], hits[0][1]


# ─────────────────────── 状态名 → buff 配置 ───────────────────────


def resolve_state(name, skill_id, buff_index):
    """状态中文名 → buf 配置 id。歧义消解：技能族前缀 > buf_CS_ > 首个。

    技能族：CS_by_xhmdl06 → 家族前缀 CS_by_xhmdl（同套路兄弟技的状态
    通常挂在族首个 buf 上，如"孽"→ buf_CS_by_xhmdl01）。
    """
    name = name.strip("“”\"『』「」 ")
    candidates = buff_index.get(name, [])
    if not candidates:
        return "", False
    if len(candidates) == 1:
        return candidates[0], True
    family = skill_id.rsplit("0", 1)[0] if re.search(r"\d+$", skill_id) else skill_id
    for prefix in ("buf_" + family, "buf_" + family[:8] + "0", "buf_CS_"):
        for cand in candidates:
            if cand.startswith(prefix):
                return cand, True
    return sorted(candidates)[0], True


# ─────────────────────── 段落模板匹配 ───────────────────────


def parse_segment(seg, skill_id, buff_index, entry, states=()):
    """对一个段落跑全部模板；命中则把规则/静态字段写进 entry。

    states=预处理收集的彩色状态名（#FFAE00/#FF3030 段，无引号形态的
    权威来源）。返回 True=有模板命中（段落被消费）；段落可被多模板
    消费（触发从句常与控制效果同段）。
    """
    matched = False
    trigger_claimed = False  # 专属触发（招架/破防/周期/自身/受击叠层）已认领本段

    # ── 旁注括号段（（…））：施法约束/消耗/轻功 ──
    for paren in PAREN_RE.findall(seg):
        if parse_paren_note(paren, entry):
            matched = True

    # ── 范围（aoe）：句内含"米"的形状表达 ──
    if parse_aoe(seg, entry):
        matched = True

    # ── 次数上限 ──
    hits = parse_max_hits(seg)
    if hits is not None:
        entry["compile"]["max_hits"] = hits
        matched = True

    # ── 施法霸体（非旁注位置出现的"红/黄霸体状态"） ──
    if "霸体" in seg and "状态" in seg:
        if "红霸体" in seg:
            entry["compile"]["cast_armor"] = "red"
            matched = True
        elif "黄霸体" in seg:
            entry["compile"]["cast_armor"] = "yellow"
            matched = True

    # ── 对空 / 空中限制（非旁注位置） ──
    if "只能针对空中目标" in seg or "只能对空中目标" in seg or "只能攻击空中目标" in seg:
        entry["compile"]["air"] = "only"
        matched = True
    if "无法在空中施展" in seg:
        entry["compile"]["air"] = "never"
        matched = True

    # ── 周期触发：每N秒给周围…添加 ──
    rule = parse_periodic(seg, skill_id, buff_index)
    if rule:
        entry["on_periodic"].append(rule)
        matched = True
        trigger_claimed = True

    # ── 自身状态：自身进入"X"状态 ──
    rule = parse_self_state(seg, skill_id, buff_index)
    if rule:
        entry["self_state"].append(rule)
        matched = True
        trigger_claimed = True

    # ── 招架触发：招架成功后 / 招架对方攻击 ──
    if re.search(r"招架成功后|招架对方攻击|每每.*招架", seg):
        rule = parse_rule_payload(seg, skill_id, buff_index, states)
        entry["on_parry"].append(rule)
        matched = True
        trigger_claimed = True

    # ── 破防触发：击破…招架（时） ──
    if re.search(r"击破.*招架|破防", seg) and re.search(r"击破", seg):
        rule = parse_rule_payload(seg, skill_id, buff_index, states)
        if re.search(r"(\d+)秒内", seg):
            rule["control_within_ms"] = parse_num(NUM + r"秒内", seg, int) * 1000
        entry["on_parry_break"].append(rule)
        matched = True
        trigger_claimed = True

    # ── 受击叠层（感染族）：存在"X"状态时，被攻击则叠加 ──
    m = re.search(r"存在[“\"]([^”\"]+)[”\"]状态时，被攻击则叠加", seg)
    if m:
        buff, ok = resolve_state(m.group(1), skill_id, buff_index)
        entry["on_victim"].append({
            "stack_buff": buff or m.group(1), "stack_resolved": ok,
            "stack_per_hit": 1,
        })
        matched = True
        trigger_claimed = True

    # ── 每击/命中触发（通用兜底）：专属触发已认领的段落不再双挂 ──
    if not trigger_claimed and any(trigger in seg for trigger in ("每击", "命中", "击退", "拉近", "添加", "击飞", "击倒", "眩晕", "昏迷", "定身", "封招", "禁轻功", "无法使用轻功", "流失内力")):
        rule = parse_rule_payload(seg, skill_id, buff_index, states)
        if rule_needs_emit(rule):
            if "每击" in seg:
                rule["per_hit"] = True
            entry["on_hit"].append(rule)
            matched = True

    # ── 条件从句：若命中已破防 / 若目标存有"X"状态(越多/N层)。
    #    附着顺序：本段 on_hit 规则 > 最近的破防规则（tjj05"若命中已破防
    #    则恢复内力"与触发段分句的形态）> 最近命中规则。 ──
    cond = parse_condition(seg)
    if cond:
        for target in (entry["on_hit"][::-1] + entry["on_parry_break"][::-1] if entry["on_hit"] or entry["on_parry_break"] else []):
            if "condition" not in target:
                target["condition"] = cond
                matched = True
                break

    return matched


def parse_paren_note(paren, entry):
    hit = False
    if "不可移动" in paren:
        entry["compile"]["can_move"] = False
        hit = True
    if "可施展轻功" in paren or "可利用轻功快速接近" in paren or "可以利用轻功快速接近" in paren:
        entry["compile"]["can_qinggong"] = True
        entry["compile"]["rush_note"] = True
        hit = True
    if "无法在空中施展" in paren:
        entry["compile"]["air"] = "never"
        hit = True
    if "只能针对空中目标" in paren or "只能对空中目标" in paren or "只能攻击空中目标" in paren:
        entry["compile"]["air"] = "only"
        hit = True
    for armor in ("红霸体", "黄霸体"):
        if armor in paren:
            entry["compile"]["cast_armor"] = armor[0]
            hit = True
    return hit


def parse_aoe(seg, entry):
    hit = False
    m = re.search(r"(?:前方|身前)" + NUM + r"米", seg)
    if m:
        entry["compile"]["aoe_shape"] = "sector"
        entry["compile"]["aoe_radius_m"] = float(m.group(1))
        hit = True
    m = re.search(r"周围" + NUM + r"米", seg)
    if m:
        entry["compile"].setdefault("aoe_shape", "circle")
        if entry["compile"]["aoe_shape"] == "circle":
            entry["compile"]["aoe_radius_m"] = float(m.group(1))
        hit = True
    m = re.search(NUM + r"米范围", seg)
    if m:
        entry["compile"].setdefault("aoe_shape", "circle")
        if entry["compile"]["aoe_shape"] == "circle":
            entry["compile"]["aoe_radius_m"] = float(m.group(1))
        hit = True
    m = re.search(r"宽" + NUM + r"米", seg)
    if m:
        entry["compile"]["aoe_shape"] = "rect"
        entry["compile"]["aoe_width_m"] = float(m.group(1))
        hit = True
    if "周围" in seg and "aoe_shape" not in entry["compile"]:
        entry["compile"]["aoe_shape"] = "circle"
        hit = True
    return hit


def parse_max_hits(seg):
    for pattern in (r"最多斩击" + NUM + r"次", r"最多攻击" + NUM + r"次",
                    r"最多" + NUM + r"次", r"共攻击" + NUM + r"次", r"连击" + NUM + r"次"):
        m = re.search(pattern, seg)
        if m:
            return int(float(m.group(1)))
    m = re.search(r"(\d+)连击", seg)
    if m:
        return int(m.group(1))
    if "双击" in seg:
        return 2
    return None


def parse_periodic(seg, skill_id, buff_index):
    m = re.search(r"每" + NUM + r"秒给自身周围", seg)
    if not m:
        return None
    rule = {"periodic_every_ms": int(float(m.group(1))) * 1000, "periodic_aoe": True}
    state = re.search(r"添加[“\"]?([^，。；:]{1,12})[”\"]?", seg[m.end():])
    if state:
        buff, ok = resolve_state(state.group(1), skill_id, buff_index)
        rule["apply_buff"] = buff or state.group(1)
        rule["apply_resolved"] = ok
    return rule


def parse_self_state(seg, skill_id, buff_index):
    m = re.search(r"自身进入[“\"]([^”\"]+)[”\"]状态", seg)
    if not m:
        return None
    buff, ok = resolve_state(m.group(1), skill_id, buff_index)
    rule = {"apply_buff": buff or m.group(1), "apply_resolved": ok}
    life = parse_num(r"持续" + NUM + r"秒|状态持续" + NUM + r"秒", seg, int)
    if life:
        rule["apply_life_ms"] = life * 1000
    return rule


def parse_condition(seg):
    cond = {}
    if re.search(r"未及招架|未招架|未被招架", seg):
        cond["unparried"] = True
    if re.search(r"已经被破防|已破防", seg):
        cond["parry_broken"] = True
    m = re.search(r"(?:若|当)?(?:目标|其|敌人)(?:身上)?存有[“\"]([^”\"]+)[”\"]", seg)
    if m:
        cond["target_state"] = m.group(1)
    m = re.search(r"无视招架", seg)
    if m:
        cond["unparryable"] = True
    return cond or None


def parse_rule_payload(seg, skill_id, buff_index, states=()):
    """从触发段提取效果载荷：控制/位移/资源/加状态/爆发/回复。

    states=彩色状态名列表（#FFAE00/#FF3030），是"添加X"里 X 的权威来源
    （旋刀族无引号全靠颜色）；无彩色段回退引号/短名正则。
    """
    rule = {}

    # 爆发段位置（其后出现的控制词属于 burst 效果，不重复挂规则级 control）
    burst_pos = seg.find("爆发")

    # 位移：击退（受害者位置后移）/ 拉近（向施法者）
    for word, kind in DISPLACEMENT_WORDS.items():
        if word in seg:
            rule["displacement_kind"] = kind
            dist = parse_num(word + NUM + r"米|" + NUM + r"米" + word, seg)
            rule["displacement_distance_m"] = dist or 0.0
            break

    # 资源：流失内力（目标减蓝）/ 恢复N点内力（无数字记 0）/ 恢复气血
    if "流失内力" in seg:
        rule["mp_drain"] = True
        dur = parse_num(NUM + r"秒内", seg, int)
        rule["mp_drain_dur_ms"] = dur * 1000 if dur else 0
    m = re.search(r"恢复自身?" + NUM + r"点内力", seg)
    if m:
        rule["mp_restore"] = int(float(m.group(1)))
    elif "恢复自身内力" in seg or "恢复内力" in seg:
        rule["mp_restore"] = 0
    if "恢复" in seg and "气血" in seg:
        rule["heal"] = True

    # 加状态：给(其|敌人|目标|攻击者)添加X / 添加N层X —— X 优先取彩色状态名
    name = None
    for cand in sorted(states, key=len, reverse=True):
        if cand and cand in seg:
            name = cand
            break
    if name is None:
        m = re.search(r"(?:给(?:其|敌人|目标|攻击者|自身周围敌人)?)?添加(?:\d+层)?[“\"]?([^”，。；：:]{1,16})[”\"]?", seg)
        if m:
            name = m.group(1).strip("“”\"『』")
    if name:
        buff, ok = resolve_state(name, skill_id, buff_index)
        rule["apply_buff"] = buff or name
        rule["apply_resolved"] = ok
        stacks = parse_num(r"添加" + NUM + r"层", seg, int)
        rule["apply_stacks"] = stacks if stacks else 1
        life = parse_num(r"持续" + NUM + r"秒", seg, int)
        if life:
            rule["apply_life_ms"] = life * 1000
        burst_at = parse_num(r"叠加至" + NUM + r"层后爆发", seg, int)
        if burst_at:
            rule["burst_at"] = burst_at
            burst_ctrl = find_control(seg[burst_pos:] if burst_pos >= 0 else seg)
            if burst_ctrl:
                rule["burst_kind"] = burst_ctrl[0]
                bdur = parse_num(NUM + r"秒", seg[burst_pos:], int)
                rule["burst_dur_ms"] = bdur * 1000 if bdur else 0

    # 控制状态 + 时长（优先长词）：造成N秒眩晕 / 使其…封招3秒 / N秒内无法使用轻功。
    # 爆发子句里的控制词（"叠加至N层后爆发，定身2秒"）只属于 burst。
    control = find_control(seg)
    if control:
        kind, word = control
        ctrl_pos = seg.find(word)
        beyond_burst = rule.get("burst_at") and burst_pos >= 0 and ctrl_pos >= 0 and ctrl_pos > burst_pos
        if not beyond_burst:
            rule["control_kind"] = kind
            dur = parse_num(word + NUM + r"秒|" + NUM + r"秒" + word + "|" + NUM + r"秒内" + word, seg, int)
            if dur is None:
                dur = parse_num(NUM + r"秒", seg, int)
            if dur:
                rule["control_dur_ms"] = dur * 1000

    # 概率：有N%几率/概率
    m = re.search(r"有" + NUM + r"%(?:的)?(?:几率|概率)", seg)
    if m:
        rule["chance"] = float(m.group(1)) / 100.0
    else:
        m = re.search(r"(\d+)%(?:几率|概率)", seg)
        if m:
            rule["chance"] = int(m.group(1)) / 100.0

    return rule


def rule_needs_emit(rule):
    """空载荷不发规则（模板命中但无效果字段的段落不算 on_hit）。

    按键存在性判断——mp_restore=0（"恢复内力"未标数值）是有效载荷。
    """
    keys = ("control_kind", "displacement_kind", "mp_drain", "mp_restore", "heal", "apply_buff")
    return any(k in rule for k in keys)


# ─────────────────────── 条目解析与差集 ───────────────────────


def new_entry():
    return {
        "compile": {}, "on_hit": [], "on_parry": [], "on_parry_break": [],
        "on_victim": [], "on_periodic": [], "self_state": [], "unparsed": [],
    }


NUMERIC_PATHS = [
    ("compile_max_hits", ("compile", "max_hits")),
    ("apply_life", ("apply_life_ms",)),
    ("apply_stacks", ("apply_stacks",)),
    ("control_dur", ("control_dur_ms",)),
    ("periodic_every", ("periodic_every_ms",)),
]


def entry_numeric_signature(entry):
    """数值指纹：(路径, 值) 平铺集合，用于等级间差集。"""
    sig = []

    def walk(prefix, obj):
        if isinstance(obj, dict):
            for key in sorted(obj):
                walk(prefix + [key], obj[key])
        elif isinstance(obj, list):
            for i, item in enumerate(obj):
                walk(prefix + [str(i + 1)], item)
        elif isinstance(obj, (int, float)) and not isinstance(obj, bool):
            sig.append((".".join(prefix), obj))

    walk([], entry)
    return sig


def structure_signature(entry):
    """结构指纹：数值置零后的形状（判断等级间是否只是数值变化）。"""
    return sorted(path for path, _ in entry_numeric_signature(entry))


def parse_one(text, skill_id, buff_index):
    entry = new_entry()
    plain, states, actions = preprocess(text)
    for seg in split_segments(plain):
        if not parse_segment(seg, skill_id, buff_index, entry, states):
            if any(w in seg for w in WATCH_WORDS):
                entry["unparsed"].append(seg[:48])
    return entry


def merge_levels(levels, skill_id, buff_index):
    """全部等级解析 → L1 基准 + by_level 数值差集。"""
    parsed = {}
    for level in sorted(levels):
        parsed[level] = parse_one(levels[level], skill_id, buff_index)
    base_level = min(parsed)
    base = parsed[base_level]
    base_paths = dict(entry_numeric_signature(base))
    by_level = {}
    for level, entry in parsed.items():
        if level == base_level:
            continue
        deltas = {}
        for path, value in entry_numeric_signature(entry):
            if base_paths.get(path) != value:
                deltas[path] = value
        if deltas:
            by_level[level] = deltas
    return base, by_level, len(parsed)


# ─────────────────────── Lua 输出 ───────────────────────


def lua_str(value):
    return '"%s"' % value.replace("\\", "\\\\").replace('"', '\\"')


def lua_field(key, value, indent):
    pad = " " * indent
    if isinstance(value, bool):
        return "%s%s = %s" % (pad, key, "true" if value else "false")
    if isinstance(value, int):
        return "%s%s = %d" % (pad, key, value)
    if isinstance(value, float):
        text = ("%g" % value)
        return "%s%s = %s" % (pad, key, text)
    if isinstance(value, str):
        return "%s%s = %s" % (pad, key, lua_str(value))
    return None


def rule_to_lua(rule, indent):
    """规则字典 → Lua 表体（平铺标量字段 + condition 嵌套表；空表返回 None）。"""
    fields = []
    for key in sorted(rule):
        value = rule[key]
        if key == "condition" and isinstance(value, dict) and value:
            inner = ", ".join("%s = %s" % (k, "true" if v else "false") if isinstance(v, bool)
                              else lua_str(v) for k, v in sorted(value.items()))
            fields.append("condition = { %s }" % inner)
            continue
        line = lua_field(key, value, 0)
        if line:
            fields.append(line.strip())
    if not fields:
        return None
    pad = " " * indent
    return "{\n" + ",\n".join(" " * (indent + 4) + f for f in fields) + "\n" + pad + "}"


def lua_number(value):
    return "%g" % value if isinstance(value, float) else str(value)


def entry_to_lua(skill_id, base, by_level, level_count):
    """条目 → Lua。字段间以逗号连接（生成合法表语法）；数值差集键带
    点路径，必须写成 ["path"] 引号键。"""
    fields = []

    compile_fields = {
        "max_hits": base["compile"].get("max_hits"),
        "cast_armor": base["compile"].get("cast_armor"),
        "can_move": base["compile"].get("can_move"),
        "can_qinggong": base["compile"].get("can_qinggong"),
        "rush_note": base["compile"].get("rush_note"),
        "air": base["compile"].get("air"),
    }
    aoe = {k: base["compile"].get(k) for k in ("aoe_shape", "aoe_radius_m", "aoe_width_m")}
    parts = []
    if any(v is not None and v != "" for v in aoe.values()):
        aoe_fields = [lua_field(k, v, 0) for k, v in aoe.items() if v is not None and v != ""]
        parts.append("aoe = { " + ", ".join(f.strip() for f in aoe_fields) + " }")
    if "can_move" in base["compile"]:
        parts.append("can_move = %s" % ("true" if base["compile"]["can_move"] else "false"))
    for key, value in compile_fields.items():
        if key == "can_move":
            continue
        if value is None or value is False:
            continue
        line = lua_field(key, value, 0)
        if line:
            parts.append(line.strip())
    if parts:
        fields.append("    compile = {\n        " + ",\n        ".join(parts) + "\n    }")

    for hooks_key in ("on_hit", "on_parry", "on_parry_break", "on_victim", "on_periodic", "self_state"):
        rules = [rule_to_lua(r, 8) for r in base[hooks_key]]
        rules = [r for r in rules if r]
        if rules:
            inner = ",\n".join("        " + r.lstrip() for r in rules)
            fields.append("    %s = {\n%s\n    }" % (hooks_key, inner))

    if by_level:
        level_lines = []
        for level in sorted(by_level):
            deltas = ", ".join('["%s"] = %s' % (path, lua_number(v))
                               for path, v in sorted(by_level[level].items()))
            level_lines.append("        [%d] = { %s }" % (level, deltas))
        fields.append("    by_level = {\n" + ",\n".join(level_lines) + "\n    }")

    if base["unparsed"]:
        quoted = ", ".join(lua_str(u) for u in base["unparsed"][:6])
        fields.append("    unparsed = { %s }" % quoted)

    return 'E["%s"] = {\n%s\n}' % (skill_id, ",\n".join(fields))


def emit_lua(entries):
    out = [
        "-- 本文件由 tools/skill-data/gen_desc_effects.py 生成（幂等可重跑，勿手改）。",
        "-- 数据源 string_skill_desc.idres（desc_<技能ID>_<等级>）；IR 权威定义",
        "-- docs/skill-desc-ir.md（动词三分类：控制状态/位移/资源）。消费方：",
        "-- internal/skillscript DescEffects（旁路数据层，优先级链不变：手写 >",
        "-- effects_data > INI；本表仅供查询与后续引擎接线）。",
        "local E = {}",
        "",
    ]
    emitted = 0
    for skill_id in sorted(entries):
        base, by_level, level_count = entries[skill_id]
        has_content = bool(base["compile"]) or base["unparsed"] or any(
            base[key] for key in ("on_hit", "on_parry", "on_parry_break", "on_victim", "on_periodic", "self_state"))
        if not has_content:
            continue  # 纯口诀/纯伤害技：无机制可解析，不占条目
        out.append(entry_to_lua(skill_id, base, by_level, level_count))
        out.append("")
        emitted += 1
    out.append("return E")
    out.append("")
    print("Lua 条目：%d（另 %d 技无机制内容不占条目）" % (emitted, len(entries) - emitted))
    return "\n".join(out)


# ─────────────────────── 覆盖率报表 ───────────────────────


def coverage_report(entries):
    total = len(entries)
    stats = defaultdict(int)
    unparsed_hist = defaultdict(int)
    for skill_id, (base, _, _) in entries.items():
        stats["skills"] += 1
        for key in ("on_hit", "on_parry", "on_parry_break", "on_victim", "on_periodic", "self_state"):
            if base[key]:
                stats[key] += 1
        if base["compile"].get("max_hits"):
            stats["max_hits"] += 1
        if base["compile"].get("aoe_shape"):
            stats["aoe"] += 1
        if base["compile"].get("cast_armor"):
            stats["cast_armor"] += 1
        if base["compile"].get("air"):
            stats["air"] += 1
        if base["unparsed"]:
            stats["with_unparsed"] += 1
        for u in base["unparsed"]:
            unparsed_hist[u] += 1
    lines = ["desc_effects 覆盖率报表（共 %d 技）:" % total]
    for key in sorted(stats):
        lines.append("  %-16s %5d  (%.1f%%)" % (key, stats[key], stats[key] * 100.0 / max(total, 1)))
    lines.append("  unparsed 高频段 Top15:")
    for phrase, count in sorted(unparsed_hist.items(), key=lambda kv: -kv[1])[:15]:
        lines.append("    %4d  %s" % (count, phrase))
    return "\n".join(lines)


# ─────────────────────── 自检（金样本） ───────────────────────


def selftest(resources_root):
    skill_desc = load_idres(os.path.join(resources_root, "text", "chineses", "string_skill_desc.idres"))
    buff_index = load_buff_name_index(os.path.join(resources_root, "text", "chineses", "string_buff_desc.idres"))

    grouped = defaultdict(dict)
    for key, value in skill_desc.items():
        m = re.match(r"^desc_(CS_[A-Za-z0-9_]+)_(\d+)$", key)
        if m:
            grouped[m.group(1)][int(m.group(2))] = value

    failures = []

    def check(cond, message):
        if not cond:
            failures.append(message)

    def one(skill_id):
        base, by_level, _ = merge_levels(grouped[skill_id], skill_id, buff_index)
        return base, by_level

    # 旋刀 01：前方20米扇形 + 命中未招架加"孽"（3层爆发昏迷2s）
    base, _ = one("CS_by_xhmdl01")
    check(base["compile"].get("aoe_shape") == "sector" and base["compile"].get("aoe_radius_m") == 20,
          "xhmdl01 aoe=%s" % base["compile"])
    hit = base["on_hit"][0] if base["on_hit"] else {}
    check(hit.get("apply_resolved") and hit.get("apply_buff") == "buf_CS_by_xhmdl01",
          "xhmdl01 apply_state=%s" % hit)
    check(hit.get("burst_at") == 3 and hit.get("burst_kind") == "coma" and hit.get("burst_dur_ms") == 2000,
          "xhmdl01 burst=%s" % hit)
    check((hit.get("condition") or {}).get("unparried"), "xhmdl01 cond=%s" % hit.get("condition"))

    # 旋刀 04：拉近 + 每击加孽 + 黄霸体 + 不可移动
    base, _ = one("CS_by_xhmdl04")
    check(base["compile"].get("cast_armor") == "yellow", "xhmdl04 armor=%s" % base["compile"])
    check("can_move" in base["compile"] and base["compile"]["can_move"] is False,
          "xhmdl04 can_move=%s" % base["compile"])
    hit = base["on_hit"][0] if base["on_hit"] else {}
    check(hit.get("displacement_kind") == "pull", "xhmdl04 pull=%s" % hit)
    check(hit.get("per_hit") is True, "xhmdl04 per_hit=%s" % hit)

    # 旋刀 05：击破招架 → 2层孽 + 6秒内出招后封招3秒
    base, _ = one("CS_by_xhmdl05")
    br = base["on_parry_break"][0] if base["on_parry_break"] else {}
    check(br.get("apply_stacks") == 2, "xhmdl05 stacks=%s" % br)
    check(br.get("control_kind") == "silence" and br.get("control_dur_ms") == 3000,
          "xhmdl05 silence=%s" % br)

    # 旋刀 06：招架成功 → 给攻击者"血刃"（8s，10层爆发定身2s）+ 被攻击叠层
    base, _ = one("CS_by_xhmdl06")
    par = base["on_parry"][0] if base["on_parry"] else {}
    check(par.get("apply_resolved") and par.get("apply_life_ms") == 8000,
          "xhmdl06 parry state=%s" % par)
    check(par.get("burst_at") == 10 and par.get("burst_kind") == "root", "xhmdl06 burst=%s" % par)
    victim = base["on_victim"][0] if base["on_victim"] else {}
    check(victim.get("stack_per_hit") == 1, "xhmdl06 stack=%s" % victim)

    # 旋刀 07：自身进入"横行无忌"（6s）+ 每2秒周围加孽
    base, _ = one("CS_by_xhmdl07")
    self_rule = base["self_state"][0] if base["self_state"] else {}
    check(self_rule.get("apply_life_ms") == 6000, "xhmdl07 self=%s" % self_rule)
    periodic = base["on_periodic"][0] if base["on_periodic"] else {}
    check(periodic.get("periodic_every_ms") == 2000 and periodic.get("periodic_aoe"),
          "xhmdl07 periodic=%s" % periodic)

    # 旋刀 08：每击加孽（无视招架）+ 红霸体
    base, _ = one("CS_by_xhmdl08")
    check(base["compile"].get("cast_armor") == "red", "xhmdl08 armor=%s" % base["compile"])
    hit = base["on_hit"][0] if base["on_hit"] else {}
    check(hit.get("per_hit") is True and (hit.get("condition") or {}).get("unparryable"),
          "xhmdl08 per_hit/unparryable=%s" % hit)

    # 太极剑 01：冲锋招式双击 → max_hits=2 + 轻功旁注
    base, _ = one("CS_wd_tjj01")
    check(base["compile"].get("max_hits") == 2, "tjj01 max_hits=%s" % base["compile"])
    check(base["compile"].get("rush_note") and base["compile"].get("can_qinggong"),
          "tjj01 qinggong=%s" % base["compile"])

    # 太极剑 05：击破招架（破防技标记）+ 命中已破防恢复内力（on_hit 条件规则）
    base, _ = one("CS_wd_tjj05")
    check(len(base["on_parry_break"]) >= 1, "tjj05 break=%s" % base["on_parry_break"])
    hit = base["on_hit"][0] if base["on_hit"] else {}
    check((hit.get("condition") or {}).get("parry_broken") and hit.get("mp_restore") is not None,
          "tjj05 payload=%s" % hit)

    # 太极剑 06：招架恢复4点内力
    base, _ = one("CS_wd_tjj06")
    par = base["on_parry"][0] if base["on_parry"] else {}
    check(par.get("mp_restore") == 4, "tjj06 mp=%s" % par)

    # 太极剑 08：击退（位移 push）+ 流失内力（减蓝）
    base, _ = one("CS_wd_tjj08")
    hit = base["on_hit"][0] if base["on_hit"] else {}
    check(hit.get("displacement_kind") == "push", "tjj08 push=%s" % hit)
    check(hit.get("mp_drain") is True, "tjj08 mp_drain=%s" % hit)

    # 太极剑 09：击落范围内 + 5秒内无法使用轻功
    base, _ = one("CS_wd_tjj09")
    hit = base["on_hit"][0] if base["on_hit"] else {}
    check(hit.get("control_kind") == "ban_qinggong" and hit.get("control_dur_ms") == 5000,
          "tjj09 ban_qinggong=%s" % hit)

    # 寒心恨雪：L1 最多斩击3次 → L10 8次（by_level 差集）
    base, by_level = one("CS_jh_hyjf08")
    check(base["compile"].get("max_hits") == 3, "hyjf08 L1 max_hits=%s" % base["compile"])
    check(by_level.get(10, {}).get("compile.max_hits") == 8,
          "hyjf08 by_level=%s" % by_level.get(10))

    if failures:
        print("SELFTEST FAILED (%d):" % len(failures))
        for failure in failures:
            print("  - %s" % failure)
        return 1
    print("selftest OK：金样本 13 技（旋刀8 + 太极剑5 + 寒心恨雪）全部断言通过")
    return 0


# ─────────────────────── 主流程 ───────────────────────


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", help="resources 根目录（<server>/resources）")
    parser.add_argument("--out", default=DEFAULT_OUT, help="输出 Lua 路径")
    parser.add_argument("--selftest", action="store_true", help="金样本断言后退出")
    parser.add_argument("--report-only", action="store_true", help="只打报表不写文件")
    args = parser.parse_args()

    resources_root = find_resources_root(args.root)
    skill_desc = load_idres(os.path.join(resources_root, "text", "chineses", "string_skill_desc.idres"))
    buff_index = load_buff_name_index(os.path.join(resources_root, "text", "chineses", "string_buff_desc.idres"))

    if args.selftest:
        sys.exit(selftest(resources_root))

    grouped = defaultdict(dict)
    for key, value in skill_desc.items():
        m = re.match(r"^desc_(CS_[A-Za-z0-9_]+)_(\d+)$", key)
        if m:
            grouped[m.group(1)][int(m.group(2))] = value

    entries = {}
    for skill_id in sorted(grouped):
        entries[skill_id] = merge_levels(grouped[skill_id], skill_id, buff_index)

    report = coverage_report(entries)
    print(report)
    if args.report_only:
        return

    lua = emit_lua(entries)
    with open(args.out, "w", encoding="utf-8", newline="\n") as fh:
        fh.write(lua)
    resolved = sum(1 for _, (base, _, _) in entries.items() for r in base["on_hit"] + base["on_parry"]
                   if r.get("apply_resolved"))
    total_states = sum(1 for _, (base, _, _) in entries.items()
                       for r in base["on_hit"] + base["on_parry"] + base["on_parry_break"]
                       + base["on_periodic"] + base["self_state"] if r.get("apply_buff"))
    print("已写 %s（%d 技；状态反查 %d/%d）" % (os.path.relpath(args.out, REPO_ROOT), len(entries), resolved, total_states))


if __name__ == "__main__":
    main()
