# -*- coding: utf-8 -*-
"""一次性数据搬运：从 2433 仓（b-line）cmd/protocol-probe 的建号 starter
常量提取 StarterSet.xml（技能/轻功/任务门/采集/内功五节）。

用法: python -I tools/gen_starter_set.py
输出: testdata/ini/CreateRole/StarterSet.xml
"""
import os
import re

B = r'E:\Workspace\project\game\9yin-go-server1-2433\cmd\protocol-probe'
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
OUT = os.path.join(ROOT, 'testdata', 'ini', 'CreateRole', 'StarterSet.xml')


def read(name):
    return open(os.path.join(B, name), encoding='utf-8').read()


def skill_rows():
    text = read('skill_view.go')
    m = re.search(r'var starterSkillViews = \[\]learnedSkillView\{(.*?)\n\}', text, re.S)
    rows = []
    for rm in re.finditer(
            r'\{configID:\s*"([^"]+)",\s*staticData:\s*(\d+),\s*itemType:\s*(\d+),\s*level:\s*(\d+),\s*maxLevel:\s*(\d+)'
            r'(?:,\s*total:\s*([\d.]+))?(?:,\s*pauseTime:\s*([^,}]+))?', m.group(1)):
        cid, sd, it, lv, ml, total, pause = rm.groups()
        rows.append((cid, sd, it, lv, ml, total or '', (pause or '').strip()))
    return rows


def qinggong_ids():
    text = read('qinggong_records.go')

    def grab(var):
        m = re.search(r'var %s = \[\]string\{(.*?)\}' % var, text, re.S)
        return re.findall(r'"([^"]+)"', m.group(1))
    return grab('starterQingGongIDs'), grab('starterQingGongSkillIDs')


def task_gates():
    text = read('qinggong_records.go')
    out = {}

    def const(name):
        m = re.search(r'%s\s+uint32\s*=\s*(\d+)' % name, text)
        return int(m.group(1)) if m else None
    for n in ('functionUnlockTaskID', 'functionOpenGateTaskID', 'facultyAltTaskID', 'facultyActAltTaskID',
              'facultyGuideTaskFirst', 'facultyGuideTaskLast', 'facultySchoolLeadTaskFirst',
              'facultySchoolLeadTaskLast', 'facultyActTaskFirst', 'facultyActTaskLast'):
        out[n] = const(n)
    m = re.search(r'var funcGateTaskIDs = \[\]uint32\{(.*?)\}', text, re.S)
    out['funcGateTaskIDs'] = [int(x) for x in re.findall(r'\d+', m.group(1))]
    return out


def collect_ids():
    text = read('funcopen_cluster.go')
    m = re.search(r'var collectSkillIDs = \[\]string\{(.*?)\n\}', text, re.S)
    return re.findall(r'"([^"]+)"', m.group(1))


def neigong_rows():
    text = read('player_progress.go')
    m = re.search(r'books := \[\]learnedNeiGong\{(.*?)\n\t\}\n', text, re.S)
    rows = []
    for rm in re.finditer(
            r'\{configID:\s*"([^"]+)",\s*staticData:\s*(\d+),\s*itemType:\s*(\d+),\s*level:\s*(\d+),\s*maxLevel:\s*(\d+),'
            r'\s*total:\s*([\d.]+),\s*power:\s*([\d.]+),\s*maxPower:\s*([\d.]+)(.*?)\},', m.group(1) + '\n},', re.S):
        cid, sd, it, lv, ml, total, pw, mpw, tail = rm.groups()
        extra = dict(re.findall(r'(\w+):\s*([\d.]+)', tail))
        rows.append((cid, sd, it, lv, ml, total, pw, mpw, extra))
    return rows


def main():
    lines = ['<?xml version="1.0" encoding="utf-8"?>',
             '<!-- 建号 starter 授予集（b-line cmd/protocol-probe 常量提取，',
             '     tools/gen_starter_set.py 生成；数值权威=客户端资源 ini，',
             '     本表为无资源目录时的建号兜底口径）。-->', '<Object>']
    # 技能
    lines.append('\t<!-- 初始武学 25 项：三脚猫五件套 + 花神七式 + 心佛掌 + 太极拳古谱 -->')
    lines.append('\t<StarterSkills>')
    for cid, sd, it, lv, ml, total, pause in skill_rows():
        attrs = 'ConfigID="%s" StaticData="%s" ItemType="%s" Level="%s" MaxLevel="%s"' % (cid, sd, it, lv, ml)
        if total:
            attrs += ' Total="%d"' % int(float(total))
        if pause:
            attrs += ' PauseTime="%s"' % pause
        lines.append('\t\t<Skill %s/>' % attrs)
    lines.append('\t</StarterSkills>')
    # 轻功
    cats, insts = qinggong_ids()
    lines.append('\t<!-- 轻功：类别 18（QingGongRec 解锁）+ 实例 51（View46 容器行） -->')
    lines.append('\t<QingGong>')
    lines.append('\t\t<Category IDs="%s"/>' % ','.join(cats))
    lines.append('\t\t<Instance IDs="%s"/>' % ','.join(insts))
    lines.append('\t</QingGong>')
    # 任务门
    g = task_gates()
    lines.append('\t<!-- 功能/功能开放门任务：Task_Accepted 行集 + Task_Completed 位图集 -->')
    lines.append('\t<TaskGates>')
    lines.append('\t\t<Unlock FunctionUnlock="%d" FacultyAlt="%d" FacultyActAlt="%d" OpenGate="%d"/>' %
                 (g['functionUnlockTaskID'], g['facultyAltTaskID'], g['facultyActAltTaskID'], g['functionOpenGateTaskID']))
    lines.append('\t\t<Guide First="%d" Last="%d"/>' % (g['facultyGuideTaskFirst'], g['facultyGuideTaskLast']))
    lines.append('\t\t<SchoolLead First="%d" Last="%d"/>' % (g['facultySchoolLeadTaskFirst'], g['facultySchoolLeadTaskLast']))
    lines.append('\t\t<Act First="%d" Last="%d"/>' % (g['facultyActTaskFirst'], g['facultyActTaskLast']))
    lines.append('\t\t<FuncGate IDs="%s"/>' % ','.join(str(x) for x in g['funcGateTaskIDs']))
    lines.append('\t</TaskGates>')
    # 采集
    lines.append('\t<!-- 采集技能 %d 项（CollectSkillRec 行） -->' % len(collect_ids()))
    lines.append('\t<CollectSkills IDs="%s"/>' % ','.join(collect_ids()))
    # 内功
    lines.append('\t<!-- 初始内功书 6 本（CurNeiGong=第一本）；Add 系为 30 级 varprop 值 -->')
    lines.append('\t<NeiGong>')
    for cid, sd, it, lv, ml, total, pw, mpw, extra in neigong_rows():
        attrs = 'ConfigID="%s" StaticData="%s" ItemType="%s" Level="%s" MaxLevel="%s" Total="%s" Power="%s" MaxPower="%s"' % (
            cid, sd, it, lv, ml, int(float(total)), int(float(pw)), int(float(mpw)))
        for k, v in extra.items():
            attrs += ' %s="%d"' % (k, int(float(v)))
        lines.append('\t\t<Book %s/>' % attrs)
    lines.append('\t</NeiGong>')
    lines.append('</Object>')
    os.makedirs(os.path.dirname(OUT), exist_ok=True)
    with open(OUT, 'w', encoding='utf-8') as f:
        f.write('\n'.join(lines) + '\n')
    print('written', OUT)


if __name__ == '__main__':
    main()
