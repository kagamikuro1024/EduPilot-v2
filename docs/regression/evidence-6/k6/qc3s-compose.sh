#!/bin/bash
# shim cho scan-pii-leak.sh: chỉ hỗ trợ `logs` (gateway + worker của stack qc3s)
docker logs qc3s-gw 2>&1; docker logs qc3s-worker 2>&1
