#!/usr/bin/env python3
"""Generate logo-512.png, og.png, social-preview.png, demo.gif in copper/soap language."""
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont
import math, shutil

ROOT = Path(__file__).resolve().parents[1]
SITE = ROOT / "site"
CONCRETE = (217, 221, 227)
INK = (20, 23, 28)
COPPER = (184, 115, 51)
COPPER_D = (138, 85, 36)
COPPER_HI = (224, 176, 122)
SOAP = (61, 184, 176)
SHEEN = (196, 90, 140)
SEAL = (31, 122, 107)
FAIL = (232, 93, 4)
BRASS = (196, 154, 92)
BRASS_D = (138, 106, 50)

SORA = "/usr/share/fonts/truetype/sand-box/google/Sora/Sora-VariableFont_wght.ttf"
WORK = "/usr/share/fonts/truetype/sand-box/google/Work Sans/WorkSans-VariableFont_wght.ttf"
MONO = "/usr/share/fonts/truetype/sand-box/google/Source Code Pro/SourceCodePro-VariableFont_wght.ttf"

def font(path, size):
    try:
        return ImageFont.truetype(path, size)
    except Exception:
        return ImageFont.load_default()

def draw_pipe_h(d, x0, y, x1, thick):
    r = thick // 2
    d.rounded_rectangle([x0, y - r, x1, y + r], radius=r, fill=COPPER)
    # specular
    d.rectangle([x0 + 4, y - r + 3, x1 - 4, y - r + 8], fill=COPPER_HI)
    # shade
    d.rectangle([x0 + 4, y + r - 7, x1 - 4, y + r - 2], fill=COPPER_D)

def draw_pipe_v(d, x, y0, y1, thick):
    r = thick // 2
    d.rounded_rectangle([x - r, y0, x + r, y1], radius=r, fill=COPPER)
    d.rectangle([x - r + 3, y0 + 4, x - r + 8, y1 - 4], fill=COPPER_HI)

def draw_joint(d, cx, cy, leak=False, r=28):
    d.ellipse([cx - r - 6, cy - r - 6, cx + r + 6, cy + r + 6], fill=(110, 63, 24))
    d.ellipse([cx - r, cy - r, cx + r, cy + r], fill=COPPER_D)
    col = FAIL if leak else SEAL
    ir = r - 10
    d.ellipse([cx - ir, cy - ir, cx + ir, cy + ir], fill=col)
    if leak:
        for ox, oy, br in ((4, -42, 26), (26, -72, 18), (-18, -96, 14), (10, -124, 11)):
            bx, by = cx + ox, cy + oy
            for i in range(br, 0, -2):
                t = i / br
                rr = int(SOAP[0] * t + SHEEN[0] * (1 - t))
                gg = int(SOAP[1] * t + SHEEN[1] * (1 - t))
                bb = int(SOAP[2] * t + SHEEN[2] * (1 - t))
                d.ellipse([bx - i, by - i, bx + i, by + i], outline=(rr, gg, bb))
            d.ellipse([bx - br, by - br, bx + br, by + br], outline=SOAP, width=2)
            d.ellipse([bx - br // 3, by - br // 2, bx, by - br // 4], fill=(255, 255, 255, 140))

def elbow(draw, cx, cy, scale=1.0):
    w = int(22 * scale)
    draw.rounded_rectangle(
        [cx - int(70 * scale), cy - w // 2, cx + int(10 * scale), cy + w // 2],
        radius=w // 2, fill=COPPER,
    )
    draw.rounded_rectangle(
        [cx - w // 2, cy - int(70 * scale), cx + w // 2, cy + int(10 * scale)],
        radius=w // 2, fill=COPPER,
    )
    draw.arc(
        [cx - int(8 * scale), cy - int(8 * scale), cx + int(8 * scale), cy + int(8 * scale)],
        0, 360, fill=COPPER_D, width=max(2, int(3 * scale)),
    )
    br = int(28 * scale)
    bx, by = cx + int(32 * scale), cy - int(28 * scale)
    for i in range(br, 0, -1):
        t = i / br
        r = int(SOAP[0] * t + SHEEN[0] * (1 - t))
        g = int(SOAP[1] * t + SHEEN[1] * (1 - t))
        b = int(SOAP[2] * t + SHEEN[2] * (1 - t))
        draw.ellipse([bx - i, by - i, bx + i, by + i], outline=(r, g, b, 200))
    draw.ellipse([bx - br, by - br, bx + br, by + br], outline=SOAP, width=max(2, int(2 * scale)))
    draw.ellipse(
        [bx - int(10 * scale), by - int(14 * scale), bx - int(2 * scale), by - int(6 * scale)],
        fill=(255, 255, 255, 160),
    )

def logo512():
    s = 512
    im = Image.new("RGBA", (s, s), (0, 0, 0, 0))
    d = ImageDraw.Draw(im, "RGBA")
    elbow(d, 220, 300, scale=2.8)
    im.save(SITE / "logo-512.png", optimize=True)
    im.save(ROOT / "logo-512.png", optimize=True)
    print("logo-512.png")

def card(w, h, path):
    im = Image.new("RGB", (w, h), CONCRETE)
    d = ImageDraw.Draw(im, "RGBA")
    # concrete flecks
    for i in range(0, w, 37):
        for j in range(0, h, 41):
            d.point((i + (j % 7), j + (i % 5)), fill=(196, 200, 206))
    # vertical risers
    for x in (int(w * 0.22), int(w * 0.50), int(w * 0.78)):
        draw_pipe_v(d, x, 90, h // 2 + 10, 36)
        # flare nut
        d.rectangle([x - 26, h // 2 - 70, x + 26, h // 2 - 48], fill=COPPER_D)
    # main header
    draw_pipe_h(d, 40, h // 2 + 10, w - 40, 44)
    # couplings
    for x in (int(w * 0.35), int(w * 0.65)):
        d.rectangle([x - 8, h // 2 - 14, x + 8, h // 2 + 34], fill=COPPER_D)
    # joints
    draw_joint(d, int(w * 0.22), h // 2 + 10, leak=False, r=26)
    draw_joint(d, int(w * 0.50), h // 2 + 10, leak=True, r=30)
    draw_joint(d, int(w * 0.78), h // 2 + 10, leak=False, r=26)
    # brass nameplate
    tw, th = int(w * 0.52), 86
    tx, ty = (w - tw) // 2, 36
    d.rounded_rectangle([tx, ty, tx + tw, ty + th], radius=6, fill=BRASS, outline=COPPER_D, width=4)
    d.rounded_rectangle([tx + 8, ty + 8, tx + tw - 8, ty + th - 8], radius=2, outline=BRASS_D, width=1)
    # rivets
    for rx, ry in ((tx + 16, ty + 16), (tx + tw - 16, ty + 16), (tx + 16, ty + th - 16), (tx + tw - 16, ty + th - 16)):
        d.ellipse([rx - 5, ry - 5, rx + 5, ry + 5], fill=COPPER_D)
        d.ellipse([rx - 2, ry - 2, rx + 2, ry + 2], fill=BRASS)
    f1 = font(SORA, 34 if w >= 1200 else 28)
    f2 = font(WORK, 15)
    d.text((w // 2, ty + 36), "FAILSHUT — LEAK CHECK", fill=INK, font=f1, anchor="mm")
    d.text((w // 2, ty + 64), "PRESSURIZE · WIPE SOAP · WATCH JOINTS", fill=COPPER_D, font=f2, anchor="mm")
    # tagline
    f3 = font(WORK, 20)
    d.text((w // 2, h - 42), "Config says DENY. Runtime says ALLOW? Pressurize.", fill=INK, font=f3, anchor="mm")
    # gauge
    gx, gy = w - 110, 150
    d.ellipse([gx - 48, gy - 48, gx + 48, gy + 48], fill=INK)
    d.ellipse([gx - 42, gy - 42, gx + 42, gy + 42], outline=COPPER, width=4)
    d.ellipse([gx - 36, gy - 36, gx + 36, gy + 36], fill=CONCRETE)
    # needle at pressure
    ang = math.radians(-40)
    d.line([(gx, gy), (gx + int(28 * math.sin(ang)), gy - int(28 * math.cos(ang)))], fill=FAIL, width=3)
    d.ellipse([gx - 4, gy - 4, gx + 4, gy + 4], fill=INK)
    im.save(SITE / path, optimize=True)
    print(path)

def demo_gif():
    """Terminal replay matching Bay 03 — concrete bezel so it never reads as a dead black void."""
    frames = []
    W, H = 760, 380
    pad = 20
    tw, th = W - 2 * pad, H - 2 * pad
    mono = font(MONO, 15)
    mono_sm = font(MONO, 13)
    title_f = font(SORA, 13)
    lines = [
        ("$ failshut audit", SOAP, mono),
        ("", CONCRETE, mono),
        ("FAIL OPEN — 1 finding(s) where policy may not hold:", FAIL, mono),
        ("", CONCRETE, mono),
        ("[1] ask-vs-effective  (codex)", (255, 255, 255), mono),
        ("    file:     ~/.codex/config.toml", CONCRETE, mono_sm),
        ('    evidence: approval_policy="ask"', CONCRETE, mono_sm),
        ("              but sandbox_approval=false", CONCRETE, mono_sm),
        ("    fix:      Set sandbox_approval=true", CONCRETE, mono_sm),
        ("              so Ask cannot be skipped", CONCRETE, mono_sm),
        ("", CONCRETE, mono),
        ("Summary: 1 FAIL OPEN · 5 FAIL SHUT · 1 config", CONCRETE, mono),
    ]
    steps = list(range(1, len(lines) + 1)) + [len(lines)] * 12
    for n in steps:
        im = Image.new("RGB", (W, H), CONCRETE)
        d = ImageDraw.Draw(im)
        for i in range(0, W, 37):
            for j in range(0, H, 41):
                d.point((i + (j % 7), j + (i % 5)), fill=(196, 200, 206))
        d.rectangle([pad - 4, pad - 4, pad + tw + 4, pad + th + 4], outline=COPPER, width=4)
        d.rectangle([pad, pad, pad + tw, pad + th], fill=INK)
        d.rectangle([pad, pad, pad + tw, pad + 28], fill=COPPER)
        d.rectangle([pad, pad + 28, pad + tw, pad + 32], fill=COPPER_D)
        d.text((pad + 12, pad + 14), "JOB #FS-001  ·  failshut audit", fill=(255, 245, 230), font=title_f, anchor="lm")
        d.rectangle([pad + tw - 118, pad + 4, pad + tw - 8, pad + 24], outline=(255, 200, 160), width=2)
        d.text((pad + tw - 63, pad + 14), "FAIL OPEN", fill=(255, 220, 190), font=title_f, anchor="mm")
        y = pad + 48
        for line, col, fnt in lines[:n]:
            if "Summary" in line:
                d.text((pad + 20, y), "Summary: 1 ", fill=CONCRETE, font=fnt)
                bbox = d.textbbox((pad + 20, y), "Summary: 1 ", font=fnt)
                d.text((bbox[2], y), "FAIL OPEN", fill=FAIL, font=fnt)
                bbox2 = d.textbbox((bbox[2], y), "FAIL OPEN", font=fnt)
                d.text((bbox2[2], y), " · 5 FAIL SHUT · 1 config", fill=SOAP, font=fnt)
            else:
                d.text((pad + 20, y), line, fill=col, font=fnt)
            y += 20 if fnt is mono_sm else 22
        if n < len(lines):
            d.rectangle([pad + 20, y, pad + 28, y + 14], fill=SOAP)
        frames.append(im)
    frames[0].save(
        SITE / "demo.gif",
        save_all=True,
        append_images=frames[1:],
        duration=160,
        loop=0,
        optimize=False,
    )
    print("demo.gif", len(frames), "frames")

if __name__ == "__main__":
    logo512()
    card(1200, 630, "og.png")
    card(1280, 640, "social-preview.png")
    shutil.copy(SITE / "og.png", SITE / "og-image.png")
    demo_gif()
