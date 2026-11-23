# colunion: 各技能源表的列并集与出现次数（定映射表用）。只读。
import sys, collections
sys.stdout.reconfigure(encoding='utf-8', errors='backslashreplace')
root = sys.argv[1]
def union(path, maxrows=None):
    keys = collections.Counter(); rows = 0; first = None
    with open(path, 'rb') as f:
        for raw in f:
            line = raw.decode('utf-8', 'replace').rstrip('\r\n')
            if line.startswith('['):
                rows += 1
                if first is None: first = line
            elif '=' in line and not line.startswith(';'):
                k = line.split('=', 1)[0].strip()
                keys[k] += 1
    return rows, first, keys
for name in sys.argv[2:]:
    rows, first, keys = union(root + '/' + name)
    print('== %s: %d sections, first %s, %d keys' % (name, rows, first, len(keys)))
    print('   ' + ' '.join('%s(%d)' % kv for kv in keys.most_common()))
