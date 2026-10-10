#!/usr/bin/env python3
"""Đo tỉ lệ ký tự / token (cl100k_base = tokenizer của text-embedding-3-small; o200k_base để so) cho văn bản Việt và Anh.
Chạy: /tmp/poc-venv/bin/python tokens.py <tệp...>   (cần `pip install tiktoken`)."""
import sys, unicodedata
import tiktoken

encs = {n: tiktoken.get_encoding(n) for n in ("cl100k_base", "o200k_base")}
for path in sys.argv[1:]:
    text = unicodedata.normalize("NFC", open(path, encoding="utf-8").read())
    chars, words = len(text), len(text.split())
    row = [f"{path.rsplit('/', 1)[-1]}: {chars} ký tự, {words} từ (tách khoảng trắng)"]
    for name, enc in encs.items():
        n = len(enc.encode(text))
        row.append(f"{name}: {n} token, {chars / n:.2f} ký tự/token, {n / words:.2f} token/từ")
    print(" | ".join(row))
