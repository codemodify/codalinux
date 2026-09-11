#!/usr/bin/env python3
"""Write a bright branded CodaLinux still (no extra image libs)."""
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


# 5x7 glyphs, rows top→bottom.
GLYPHS = {
    "C": ("01110", "10001", "10000", "10000", "10000", "10001", "01110"),
    "O": ("01110", "10001", "10001", "10001", "10001", "10001", "01110"),
    "D": ("11110", "10001", "10001", "10001", "10001", "10001", "11110"),
    "A": ("01110", "10001", "10001", "11111", "10001", "10001", "10001"),
    "L": ("10000", "10000", "10000", "10000", "10000", "10000", "11111"),
    "I": ("11111", "00100", "00100", "00100", "00100", "00100", "11111"),
    "N": ("10001", "11001", "10101", "10011", "10001", "10001", "10001"),
    "U": ("10001", "10001", "10001", "10001", "10001", "10001", "01110"),
    "X": ("10001", "10001", "01010", "00100", "01010", "10001", "10001"),
    " ": ("00000", "00000", "00000", "00000", "00000", "00000", "00000"),
}


def blit_text(
    pixels: bytearray,
    width: int,
    height: int,
    text: str,
    origin_x: int,
    origin_y: int,
    scale: int,
    color: tuple[int, int, int],
) -> None:
    cursor = origin_x
    for ch in text:
        glyph = GLYPHS.get(ch.upper(), GLYPHS[" "])
        for gy, row in enumerate(glyph):
            for gx, bit in enumerate(row):
                if bit != "1":
                    continue
                for oy in range(scale):
                    for ox in range(scale):
                        x = cursor + gx * scale + ox
                        y = origin_y + gy * scale + oy
                        if 0 <= x < width and 0 <= y < height:
                            i = (y * width + x) * 4
                            pixels[i : i + 4] = bytes((*color, 255))
        cursor += 6 * scale


def main() -> None:
    width, height = 1600, 900
    # Light steel → vivid teal. Must stay readable on a dark VBox capture.
    top = (176, 214, 228)
    bottom = (64, 168, 186)
    accent = (90, 230, 245)
    pixels = bytearray(width * height * 4)
    cx, cy, radius = int(width * 0.78), int(height * 0.28), 220
    for y in range(height):
        ty = y / (height - 1)
        row_r = lerp(top[0], bottom[0], ty)
        row_g = lerp(top[1], bottom[1], ty)
        row_b = lerp(top[2], bottom[2], ty)
        for x in range(width):
            dx = (x - cx) / radius
            dy = (y - cy) / radius
            dist = (dx * dx + dy * dy) ** 0.5
            glow = max(0.0, 1.0 - dist)
            r = lerp(row_r, accent[0], glow * 0.55)
            g = lerp(row_g, accent[1], glow * 0.55)
            b = lerp(row_b, accent[2], glow * 0.55)
            i = (y * width + x) * 4
            pixels[i : i + 4] = bytes((r, g, b, 255))
    blit_text(pixels, width, height, "CODA", 120, 360, 18, (18, 42, 52))
    blit_text(pixels, width, height, "CODA", 114, 354, 18, (255, 255, 255))
    blit_text(pixels, width, height, "LINUX", 120, 520, 8, (24, 64, 74))
    blit_text(pixels, width, height, "LINUX", 118, 518, 8, (232, 246, 248))
    dest = Path(__file__).resolve().parents[1] / "branding" / "wallpapers" / "default.png"
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_bytes(png_rgba(width, height, bytes(pixels)))
    print(f"wrote {dest} ({dest.stat().st_size} bytes)")


if __name__ == "__main__":
    main()
