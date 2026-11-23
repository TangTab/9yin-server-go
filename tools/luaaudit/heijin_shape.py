# heijin_shape: 量化黑金 ini/Skill 样本的 XML 形状（行数/属性并集/取值分布）。只读。
import sys, re, collections, os
root = sys.argv[1]
attr = re.compile(r'([A-Za-z_0-9]+)\s*=\s*"([^"]*)"')
def load(name):
    path = os.path.join(root, name)
    data = open(path, 'rb').read().decode('utf-8', 'replace')
    rows = []
    for m in re.finditer(r'<Property\b([^>]*?)/>', data, re.S):
        rows.append(dict(attr.findall(m.group(1))))
    return data, rows
for name in ['Skill_Config.xml', 'Skill_Normal_Attack.xml', 'skill_new.xml.xml', 'skill_static.xml', 'Buff_Config.xml', 'Buff_Modify.xml', 'buff_static.xml']:
    data, rows = load('ini/Skill/' + name)
    print('===== %s: %d Property rows, %d KB =====' % (name, len(rows), len(data) // 1024))
    union = collections.Counter()
    for r in rows:
        for k in r:
            union[k] += 1
    print('  attrs(%d): %s' % (len(union), ' '.join('%s(%d)' % kv for kv in union.most_common())))
    for key in ('script', 'Type', 'ItemType', 'SkillEditClass', 'QTypeID', 'job_sex'):
        c = collections.Counter(r.get(key, '') for r in rows if key in r)
        if c:
            print('  %s: %s' % (key, ' '.join('%s=%s' % kv for kv in c.most_common(12))))
    print('  sample: %s' % ' '.join('%s="%s"' % kv for kv in list(rows[0].items())[:16]) if rows else '')
