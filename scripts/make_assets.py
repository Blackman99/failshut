#!/usr/bin/env python3
"""Generate logo-512.png, og.png, social-preview.png, demo.gif in copper/soap language."""
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont
import math, subprocess, os

ROOT = Path(__file__).resolve().parents[1]
SITE = ROOT / "site"
CONCRETE = (217, 221, 227)
INK = (20, 23, 28)
COPPER = (184, 115, 51)
COPPER_D = (138, 85, 36)
SOAP = (61, 184, 176)
SHEEN = (196, 90, 140)
SEAL = (31, 122, 107)
FAIL = (232, 93, 4)
BRASS = (196, 154, 92)

SORA = "/usr/share/fonts/truetype/sand-box/google/Sora/Sora-VariableFont_wght.ttf"
WORK = "/usr/share/fonts/truetype/sand-box/google/Work Sans/WorkSans-VariableFont_wght.ttf"
MONO = "/usr/share/fonts/truetype/sand-box/google/Source Code Pro/SourceCodePro-VariableFont_wght.ttf"

def font(path, size, weight=400):
    try:
        return ImageFont.truetype(path, size)
    except Exception:
        return ImageFont.load_default()

def elbow(draw, cx, cy, scale=1.0):
    # thick copper elbow using arcs/rects
    w = int(22 * scale)
    # horizontal
    draw.rounded_rectangle([cx - int(70*scale), cy - w//2, cx + int(10*scale), cy + w//2],
                           radius=w//2, fill=COPPER)
    # vertical
    draw.rounded_rectangle([cx - w//2, cy - int(70*scale), cx + w//2, cy + int(10*scale)],
                           radius=w//2, fill=COPPER)
    # highlight
    draw.arc([cx - int(8*scale), cy - int(8*scale), cx + int(8*scale), cy + int(8*scale)],
             0, 360, fill=COPPER_D, width=max(2, int(3*scale)))
    # bubble
    br = int(28 * scale)
    bx, by = cx + int(32*scale), cy - int(28*scale)
    for i in range(br, 0, -1):
        t = i / br
        r = int(SOAP[0] * t + SHEEN[0] * (1-t))
        g = int(SOAP[1] * t + SHEEN[1] * (1-t))
        b = int(SOAP[2] * t + SHEEN[2] * (1-t))
        draw.ellipse([bx-i, by-i, bx+i, by+i], outline=(r, g, b, 200))
    draw.ellipse([bx - br, by - br, bx + br, by + br], outline=SOAP, width=max(2, int(2*scale)))
    # shine
    draw.ellipse([bx - int(10*scale), by - int(14*scale), bx - int(2*scale), by - int(6*scale)],
                 fill=(255, 255, 255, 160))

def logo512():
    s = 512
    im = Image.new("RGBA", (s, s), (0, 0, 0, 0))
    d = ImageDraw.Draw(im, "RGBA")
    elbow(d, 220, 300, scale=2.8)
    im.save(SITE / "logo-512.png")
    im.save(ROOT / "logo-512.png")
    print("logo-512.png")

def card(w, h, path):
    im = Image.new("RGB", (w, h), CONCRETE)
    d = ImageDraw.Draw(im, "RGBA")
    # soft copper specular strips
    for i in range(0, w, 48):
        d.rectangle([i, 0, i + 8, h], fill=(200, 180, 160))
    # manifold bar
    d.rounded_rectangle([40, h//2 - 18, w - 40, h//2 + 18], radius=18, fill=COPPER)
    d.rectangle([40, h//2 - 14, w - 40, h//2 - 6], fill=(232, 196, 154))
    # joints
    for x, leak in ((w*0.25, False), (w*0.5, True), (w*0.75, False)):
        d.ellipse([x-28, h//2-28, x+28, h//2+28], fill=COPPER_D)
        col = FAIL if leak else SEAL
        d.ellipse([x-16, h//2-16, x+16, h//2+16], fill=col)
        if leak:
            for j, (ox, oy, r) in enumerate(((0, -50, 22), (20, -78, 16), (-16, -100, 12))):
                bx, by = x + ox, h//2 + oy
                d.ellipse([bx-r, by-r, bx+r, by+r], outline=SOAP, width=3)
                d.ellipse([bx-r+3, by-r+3, bx+r-3, by-r+8], fill=(255,255,255,80))
    # brass tag
    tw, th = int(w*0.55), 78
    tx, ty = (w - tw)//2, 48
    d.rounded_rectangle([tx, ty, tx+tw, ty+th], radius=6, fill=BRASS, outline=COPPER_D, width=3)
    f1 = font(SORA, 36 if w >= 1200 else 32)
    f2 = font(WORK, 16)
    d.text((w//2, ty + 30), "FAILSHUT — LEAK CHECK", fill=INK, font=f1, anchor="mm")
    d.text((w//2, ty + 58), "permission hooks · FAIL OPEN or FAIL SHUT", fill=COPPER_D, font=f2, anchor="mm")
    # tagline
    f3 = font(WORK, 22)
    d.text((w//2, h - 56), "Config says DENY. Runtime says ALLOW? Pressurize.", fill=INK, font=f3, anchor="mm")
    im.save(SITE / path)
    print(path)

def demo_gif():
    # Simple terminal-like frames
    frames = []
    w, h = 720, 240
    mono = font(MONO, 16)
    lines = [
        "$ failshut audit",
        "",
        "FAIL OPEN — 1 finding(s) where policy may not hold:",
        "",
        "[1] ask-vs-effective  (codex)",
        "    file:     ~/.codex/config.toml",
        "    evidence: approval_policy=\"ask\" but sandbox_approval=false",
        "    fix:      Set sandbox_approval=true so Ask cannot be skipped",
    ]
    for n in range(1, len(lines) + 1):
        im = Image.new("RGB", (w, h), INK)
        d = ImageDraw.Draw(im)
        d.rectangle([0, 0, w, 8], fill=COPPER)
        y = 24
        for line in lines[:n]:
            col = FAIL if "FAIL OPEN" in line else (SOAP if line.startswith("$") else (217, 221, 227))
            if line.startswith("[1]"):
                col = (255, 255, 255)
            d.text((20, y), line[:90], fill=col, font=mono)
            y += 22
        frames.append(im)
    # hold last
    frames += [frames[-1]] * 6
    frames[0].save(SITE / "demo.gif", save_all=True, append_images=frames[1:], duration=280, loop=0, optimize=True)
    print("demo.gif")

if __name__ == "__main__":
    logo512()
    card(1200, 630, "og.png")
    card(1280, 640, "social-preview.png")
    # alias
    import shutil
    shutil.copy(SITE / "og.png", SITE / "og-image.png")
    demo_gif()
