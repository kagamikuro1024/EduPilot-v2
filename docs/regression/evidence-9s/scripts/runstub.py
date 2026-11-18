#!/usr/bin/env python3
"""runstub.py <script.py> <CLS> [students…]: dựng lớp CLS, giữ khoá global + route GRADING→stub, chạy script con (env CLS), nhả khoá."""
import sys, os; sys.path.insert(0, os.path.dirname(__file__))
from q9 import *
script, cls = sys.argv[1], sys.argv[2]; sv = sys.argv[3:] or ["sv.gioi", "sv.kha", "sv.nguyco"]
mkcourse(cls, sv)
with stub_ctx():
    r = subprocess.run([sys.executable, "-u", os.path.join(os.path.dirname(__file__), script)], env=dict(os.environ, CLS=cls, GW=GW))
sys.exit(r.returncode)
