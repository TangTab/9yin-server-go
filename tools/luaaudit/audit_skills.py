# audit_skills: skill_static.ini 中挂了 LuaScript 的静态行 → 归并到 skill_new.ini 的技能 id，
# 按脚本目录 / 流程脚本(SkillNormal|SkillLock) / 套路归类。只读。
import sys, re, collections
root = sys.argv[1]

def sections(path):
    cur, out = None, {}
    with open(path, 'rb') as f:
        for raw in f:
            line = raw.decode('utf-8', 'replace').rstrip('\r\n')
            if line.startswith('[') and line.endswith(']'):
                cur = line[1:-1]
                out.setdefault(cur, {})
            elif cur is not None and '=' in line and not line.startswith(';'):
                k, v = line.split('=', 1)
                out[cur][k.strip()] = v.strip()
    return out

static = sections(root + '/share/skill/skill_static.ini')
new = sections(root + '/share/skill/skill_new.ini')
# StaticData → [技能 id]
by_static = collections.defaultdict(list)
for sid, row in new.items():
    sd = row.get('StaticData', '')
    if sd:
        by_static[sd].append(sid)

lua_static = {sd: row for sd, row in static.items() if row.get('LuaScript', '').strip() not in ('', '#REF!')}
print('static rows total=%d  with LuaScript=%d' % (len(static), len(lua_static)))
skills = [(sd, sid) for sd in lua_static for sid in by_static.get(sd, [])]
print('skill ids bound to those static rows=%d (unbound static rows=%d)' % (
    len(skills), sum(1 for sd in lua_static if not by_static.get(sd))))

top = collections.Counter()
flow = collections.Counter()
taolu = collections.Counter()
orphan = []
for sd, sid in skills:
    script = lua_static[sd]['LuaScript'].replace(chr(92), '/').strip()
    parts = script.split('/')
    key = '/'.join(parts[:-1]) if len(parts) > 1 else '(root)'
    top[key] += 1
    flow[new[sid].get('script', '?')] += 1
    taolu[lua_static[sd].get('TaoLu', '')] += 1
for sd in lua_static:
    if not by_static.get(sd):
        orphan.append((sd, lua_static[sd]['LuaScript'], lua_static[sd].get('TaoLu', '')))
print('\n-- by script directory (skill id count) --')
for k, n in top.most_common():
    print('%5d  %s' % (n, k))
print('\n-- by flow script --')
for k, n in flow.most_common():
    print('%5d  %s' % (n, k))
print('\n-- by TaoLu (top 40) --')
for k, n in taolu.most_common(40):
    print('%5d  %s' % (n, k))
print('\n-- static rows with Lua but no skill_new binding (%d) --' % len(orphan))
for sd, s, t in orphan[:40]:
    print('   [%s] %s  TaoLu=%s' % (sd, s, t))
