import sys,re
from lib3 import *
for f in sys.argv[1:]:
    last={}
    for l in open(f):
        m=re.match(r'(PASS|FAIL) (TC-[\w-]+) (.*)',l.rstrip('\n'))
        if m: last[m.group(2)]=(m.group(1)=='PASS',m.group(3))
    for tc,(ok,note) in last.items(): rec(tc,ok,note)
