# audit_buffs: buff_static.ini 中挂了 LuaScript 的 buff → 归并 buff_new.ini 的 buff id，按脚本目录/名字族归类。只读。
import sys, collections, os
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

static = sections(root + '/share/skill/buff_static.ini')
new = sections(root + '/share/skill/buff_new.ini')
by_static = collections.defaultdict(list)
for bid, row in new.items():
    sd = row.get('StaticData', '')
    if sd:
        by_static[sd].append(bid)
lua_static = {sd: row for sd, row in static.items() if row.get('LuaScript', '').strip() not in ('', '#REF!')}
print('buff_static rows=%d  with LuaScript=%d ; buff_new rows=%d' % (len(static), len(lua_static), len(new)))
buffs = [(sd, bid) for sd in lua_static for bid in by_static.get(sd, [])]
print('buff ids bound=%d  unbound static rows=%d' % (len(buffs), sum(1 for sd in lua_static if not by_static.get(sd))))
top = collections.Counter()
family = collections.Counter()
missing = collections.Counter()
for sd, bid in buffs:
    script = lua_static[sd]['LuaScript'].replace(chr(92), '/').strip()
    parts = script.split('/')
    top['/'.join(parts[:-1]) if len(parts) > 1 else '(root)'] += 1
    stem = parts[-1]
    fam = stem.split('_')[0:2]
    family['_'.join(fam)] += 1
    path = os.path.join(root, 'lua', script)
    if not os.path.isfile(path):
        missing[script] += 1
print('\n-- by script directory --')
for k, n in top.most_common():
    print('%5d  %s' % (n, k))
print('\n-- by script name family (top 40) --')
for k, n in family.most_common(40):
    print('%5d  %s' % (n, k))
print('\n-- scripts referenced but file absent under Res/lua: %d refs / %d files' % (sum(missing.values()), len(missing)))
for k, n in missing.most_common(15):
    print('   %3d  %s' % (n, k))
# 名字族含 ng / neigong 的 buff（全量，不限 Lua）
ng = [b for b in new if b.lower().startswith('buf_ng') or 'neigong' in b.lower()]
nglua = [b for b in ng if new[b].get('StaticData', '') in lua_static]
print('\n-- buff ids named buf_ng*/neigong*: %d (with Lua: %d) --' % (len(ng), len(nglua)))
print('   sample:', ', '.join(sorted(ng)[:12]))
