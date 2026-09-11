#!/usr/bin/env python3
"""Write a compact branded CodaLinux still (no extra image libs)."""
from __future__ import annotations

import struct
import zlib
from pathlib import Path


def png_rgba(width: int, height: int, pixels: bytes) -> bytes:
    def chunk(tag: bytes, data: bytes) -> bytes:
        return (
            struct.pack(">I", len(data))
            + tag
            + data
            + struct.pack(">I", zlib.crc32(tag + data) & 0xFFFFFFFF)
        )

    raw = b""
    stride = width * 4
    for y in range(height):
        raw += b"\x00" + pixels[y * stride : (y + 1) * stride]
    return b"".join(
        [
            b"\x89PNG\r\n\x1a\n",
            chunk(b"IHDR", struct.pack(">IIBBBBB", width, height, 8, 6, 0, 0, 0)),
            chunk(b"IDAT", zlib.compress(raw, 9)),
            chunk(b"IEND", b""),
        ]
    )


def lerp(a: int, b: int, t: float) -> int:
    return int(a + (b - a) * t)


def main() -> None:
    width, height = 1600, 900
    top = (22, 32, 42)
    bottom = (18, 78, 92)
    accent = (61, 214, 245)
    pixels = bytearray(width * height * 4)
    for y in range(height):
        ty = y / (height - 1)
        row_r = lerp(top[0], bottom[0], ty)
        row_g = lerp(top[1], bottom[1], ty)
        row_b = lerp(top[2], bottom[2], ty)
        for x in range(width):
            tx = x / (width - 1)
            # Soft diagonal wash toward the Coda teal.
            wash = max(0.0, 1.0 - abs((tx - 0.62) * 1.4 + (ty - 0.35)))
            r = lerp(row_r, accent[0], wash * 0.22)
            g = lerp(row_g, accent[1], wash * 0.22)
            b = lerp(row_b, accent[2], wash * 0.22)
            i = (y * width + x) * 4
            pixels[i : i + 4] = bytes((r, g, b, 255))
    dest = Path(__file__).resolve().parents[1] / "branding" / "wallpapers" / "default.png"
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_bytes(png_rgba(width, height, bytes(pixels)))
    print(f"wrote {dest} ({dest.stat().st_size} bytes)")


if __name__ == "__main__":
    main()
