#!/usr/bin/env bash

# patch.sh — Porcupine key-validation bypass (dynamic offset discovery)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

BINARY="$SCRIPT_DIR/node_modules/@picovoice/porcupine-node/lib/linux/x86_64/pv_porcupine.node"
BACKUP="$BINARY.orig"
MODEL="$SCRIPT_DIR/model/synthara.ppn"

# xor eax,eax; ret; nop×4 makes it always returns PV_STATUS_SUCCESS (0)

PATCH_BYTES="31c0c390909090"

# push r15; mov r15,rsi; push r14 is the known key_validate prologue (safety check)

ORIG_BYTES="41574989f74156"

# helpers

die() { echo "[error] $*" >&2; exit 1; }
info() { echo "[*] $*"; }

# dependency check

check_deps() {

    for cmd in python3 node; do
        command -v "$cmd" &>/dev/null || die "'$cmd' not found; install it first."

    done

}

# dynamic offset discovery

find_patch_offset() {

    python3 - "$BINARY" "$PATCH_BYTES" "$ORIG_BYTES" <<'PY'
import sys, struct

def die(msg):
    print(f"[error] {msg}", file=sys.stderr)
    sys.exit(1)

path, patch_hex, orig_hex = sys.argv[1:]
data  = open(path, 'rb').read()
PATCH = bytes.fromhex(patch_hex)
ORIG  = bytes.fromhex(orig_hex)

# parse ELF64 header

if data[:4] != b'\x7fELF' or data[4] != 2:
    die("not a 64-bit ELF file")

e_shoff,     = struct.unpack_from('<Q', data, 40)
e_shentsize, = struct.unpack_from('<H', data, 58)
e_shnum,     = struct.unpack_from('<H', data, 60)
e_shstrndx,  = struct.unpack_from('<H', data, 62)

shstr_off, = struct.unpack_from('<Q', data, e_shoff + e_shstrndx * e_shentsize + 24)

# index all sections

sections = {}
for i in range(e_shnum):
    hdr        = e_shoff + i * e_shentsize
    name_off,  = struct.unpack_from('<I', data, hdr)
    name       = data[shstr_off + name_off :].split(b'\x00')[0].decode()
    sh_addr,   = struct.unpack_from('<Q', data, hdr + 16)
    sh_offset, = struct.unpack_from('<Q', data, hdr + 24)
    sh_size,   = struct.unpack_from('<Q', data, hdr + 32)
    sections[name] = (sh_addr, sh_offset, sh_size)

for name in ('.text', '.dynsym', '.dynstr'):
    if name not in sections:
        die(f"{name} section not found in ELF")

text_addr, text_off, text_size = sections['.text']
text = data[text_off : text_off + text_size]

# locate pv_porcupine_init via the dynamic symbol table

dynsym_addr, dynsym_off, dynsym_sz = sections['.dynsym']
dynstr_addr, dynstr_off, dynstr_sz = sections['.dynstr']

text_sym_vas = []
init_va = None

for i in range(dynsym_sz // 24):
    sym_off   = dynsym_off + i * 24
    st_name,  = struct.unpack_from('<I', data, sym_off)
    st_value, = struct.unpack_from('<Q', data, sym_off + 8)
    sym_name  = data[dynstr_off + st_name :].split(b'\x00')[0].decode(errors='replace')
    if st_value and text_addr <= st_value < text_addr + text_size:
        text_sym_vas.append(st_value)
    if sym_name == 'pv_porcupine_init':
        init_va = st_value

if init_va is None:
    die("'pv_porcupine_init' not found in .dynsym")

# bound pv_porcupine_init with the next exported symbol

text_sym_vas.sort()
try:
    end_va = next(va for va in text_sym_vas if va > init_va)
except StopIteration:
    end_va = text_addr + text_size   # last exported function; scan to .text end

# scan pv_porcupine_init's body for a call whose target has the known prologue

start_pos = init_va - text_addr
end_pos   = end_va  - text_addr

for j in range(start_pos, end_pos - 4):

    if text[j] != 0xe8:
        continue

    disp, = struct.unpack_from('<i', text, j + 1)
    target_va = (text_addr + j + 5 + disp) & 0xFFFFFFFFFFFFFFFF

    if not (text_addr <= target_va < text_addr + text_size):
        continue

    target_off   = text_off + (target_va - text_addr)
    target_bytes = data[target_off : target_off + len(ORIG)]

    if target_bytes == ORIG or target_bytes == PATCH:
        print(f"{target_off:x}")
        sys.exit(0)

die("key_validate not found: no call in pv_porcupine_init targets a function with the expected prologue")
PY
}

# patch application

apply_patch() {

    [[ -f "$BINARY" ]] || die "Binary not found: $BINARY"

    info "Scanning binary for key_validate offset…"

    local offset_hex
    offset_hex=$(find_patch_offset)

    info "Located key_validate at file offset 0x${offset_hex}"

    [[ -f "$BACKUP" ]] || cp "$BINARY" "$BACKUP"

    python3 - "$BINARY" "0x${offset_hex}" "$PATCH_BYTES" "$ORIG_BYTES" <<'PY'

import sys

path, offset_s, patch_hex, orig_hex = sys.argv[1:]
offset = int(offset_s, 16)
patch  = bytes.fromhex(patch_hex)
orig   = bytes.fromhex(orig_hex)

with open(path, 'r+b') as f:

    f.seek(offset)
    current = f.read(len(patch))

    if current == patch:
        print("  → already patched, nothing to do")
        sys.exit(0)

    if current != orig:
        print(f"  [!] unexpected bytes at 0x{offset:x}: {current.hex()}")
        print(f"      expected original: {orig_hex}")
        print(f"      patching anyway — restore from backup first if concerned")

    f.seek(offset)
    f.write(patch)

    print(f"  -> patched 0x{offset:x}: {orig_hex} -> {patch_hex}")

PY
}


# patch restoration

restore_patch() {

    if [[ ! -f "$BACKUP" ]]; then

        info "No backup found — nothing to restore."
        return

    fi

    info "Restoring original binary from backup…"
    cp "$BACKUP" "$BINARY"
    info "Restored."

}


# verification

verify_patch() {

    [[ -f "$BINARY" ]] || die "Binary not found: $BINARY"

    local offset_hex
    offset_hex=$(find_patch_offset)

    local actual
    actual=$(python3 - "$BINARY" "0x${offset_hex}" <<'PY'

import sys

data = open(sys.argv[1], 'rb').read()
offset = int(sys.argv[2], 16)

print(data[offset : offset + 7].hex())
PY
)

    if [[ "$actual" == "$PATCH_BYTES" ]]; then

        info "Patch verified  (0x${offset_hex}: ${actual})"

    else

        die "Patch not applied — at 0x${offset_hex} got: ${actual}  expected: ${PATCH_BYTES}"

    fi

}


# usage

usage() {
    cat <<EOF
Usage: $0 [command]

Commands:
  patch     Discover offset and apply the binary patch (default)
  restore   Restore the original binary from backup
  verify    Verify the patch is currently applied
  run       Apply patch if needed, then start the wakeword detector
  help      Show this message

EOF
}


# main

cmd="${1:-patch}"

case "$cmd" in

    patch)

        check_deps
        apply_patch
        verify_patch
        info "Done."

        ;;

    restore)

        restore_patch

        ;;

    verify)

        verify_patch

        ;;

    run)

        check_deps
        [[ -f "$MODEL" ]] || die "Model not found: $MODEL"

        apply_patch
        verify_patch

        export KEY="BYPASSED"

        info "Starting wakeword detector (say 'Synthara' to trigger)…"
        info "Press Ctrl-C to stop."

        echo ""

        cd "$SCRIPT_DIR"
        exec npm start

        ;;

    help | --help | -h)

        usage
        ;;

    *)

        die "Unknown command '$cmd'.  Run '$0 help' for usage."
        ;;

esac
