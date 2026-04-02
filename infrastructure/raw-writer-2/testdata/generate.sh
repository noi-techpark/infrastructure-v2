#!/usr/bin/env bash
# SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
#
# SPDX-License-Identifier: CC0-1.0
#
# Generates test data files used by calls.http.
# Run once from the raw-writer-2 directory: ./testdata/generate.sh

set -euo pipefail
cd "$(dirname "$0")"

# small.bin — 4 raw bytes (tiny binary blob, exercises the small binary path)
printf '\xDE\xAD\xBE\xEF' > small.bin
echo "created small.bin (4 bytes)"

# small.bson — binary BSON encoding of {"hello":"world","value":42} (exercises the binary BSON path)
python3 -c "
import struct, sys

def bson_string(key, val):
    k = key.encode() + b'\x00'
    v = val.encode() + b'\x00'
    return b'\x02' + k + struct.pack('<i', len(v)) + v

def bson_int32(key, val):
    k = key.encode() + b'\x00'
    return b'\x10' + k + struct.pack('<i', val)

body = bson_string('hello', 'world') + bson_int32('value', 42)
doc = struct.pack('<i', 4 + 1 + len(body)) + body + b'\x00'
sys.stdout.buffer.write(doc)
" > small.bson
echo "created small.bson ($(wc -c < small.bson) bytes)"

# large.json — ~6 MB JSON array (exercises the large compressible path → S3 + zstd)
python3 -c "
import json, sys
sys.stdout.write(json.dumps([{'id': i, 'station': 'Bolzano-Center', 'temp_c': round(12.0 + i * 0.001, 3), 'value': i * 1.0} for i in range(200000)]))
" > large.json
echo "created large.json ($(wc -c < large.json) bytes)"

# large.bin — 6 MB of random bytes (exercises the large non-compressible path → S3 raw)
dd if=/dev/urandom bs=1M count=6 of=large.bin 2>/dev/null
echo "created large.bin ($(wc -c < large.bin) bytes)"
