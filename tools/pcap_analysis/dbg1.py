import os, sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pcaplib import read_pcapng
cnt = 0
for ts, lt, pkt in read_pcapng(r'E:\Workspace\project\game\adata\official\a.pcapng'):
    if cnt < 3:
        print('linktype=%s len=%d head=%s' % (lt, len(pkt), pkt[:40].hex()))
    cnt += 1
print('packets:', cnt)
