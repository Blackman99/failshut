#!/usr/bin/env python3
"""Generate the logo mark (logo-512.png, icon PNGs, favicon, logo.svg), og.png, social-preview.png and demo.gif in copper/soap language.

    python3 scripts/make_assets.py          # everything
    python3 scripts/make_assets.py logo     # logo mark + icons only
    python3 scripts/make_assets.py demo     # demo.gif only
"""
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

# --- logo mark -------------------------------------------------------------
# One geometry, in a 128-unit box, drives logo-512.png, the icon PNGs and the
# SVGs (logo.svg / favicon.svg), so every size shows the same centred mark:
# a copper 90° elbow with a flare nut on the bend and a soap bubble rising from
# that joint into the elbow's inner corner. Union bbox is centred on (64, 64).
LOGO_K = 1.12          # scale about the centre so the mark fills the canvas
PIPE_W = 24            # pipe outer width (units)
BEND_C = (66, 70)      # bend centre
BEND_R = 30            # bend centreline radius
PIPE_X0 = 32           # left end of the horizontal run
PIPE_Y1 = 28           # top end of the vertical run
NUT = (66 + BEND_R * math.cos(math.pi / 4), 70 + BEND_R * math.sin(math.pi / 4))
NUT_R = (15, 11, 6)
BUBBLE = (58, 58, 28)  # cx, cy, r — touches the nut, sits in the inner corner


def _u(v):
    return (v - 64) * LOGO_K + 64


def _hex(c):
    return "#%02x%02x%02x" % c


def _pipe_layers(simple=False):
    # (offset toward the outside of the bend, width, colour)
    if simple:  # favicon sizes: fewer, bolder bands so 16/32 px stay crisp
        return [(0, PIPE_W, (110, 63, 24)), (0.5, PIPE_W - 6, COPPER), (-3.5, 7, COPPER_HI)]
    return [
        (0, PIPE_W, (110, 63, 24)),
        (0.5, PIPE_W - 5, COPPER),
        (-4.5, 6, COPPER_HI),
        (6.5, 3.5, COPPER_D),
    ]


def _bubble_rgba(t):
    """Radial colour stops for the bubble, t in [0, 1] from the highlight focus."""
    stops = [(0.0, (159, 232, 226), 0.97), (0.4, SOAP, 0.92), (0.75, SHEEN, 0.80), (1.0, SOAP, 0.72)]
    for (t0, c0, a0), (t1, c1, a1) in zip(stops, stops[1:]):
        if t <= t1:
            f = (t - t0) / (t1 - t0) if t1 > t0 else 0
            c = tuple(int(c0[i] + (c1[i] - c0[i]) * f) for i in range(3))
            return c + (int(255 * (a0 + (a1 - a0) * f)),)
    return SOAP + (int(255 * 0.72),)


def render_mark(px, ss=8, simple=None):
    """Render the mark at px×px on a transparent canvas (supersampled ss×).

    simple (default: px <= 32) drops the thin shade band and the inner nut
    ring, and thickens the bubble rim, so favicon sizes don't turn to mush.
    """
    if simple is None:
        simple = px <= 32
    S = px * ss
    k = S / 128.0
    P = lambda v: _u(v) * k
    im = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    d = ImageDraw.Draw(im, "RGBA")
    cx, cy = BEND_C
    for off, w, col in _pipe_layers(simple):
        r = BEND_R + off
        half = w * LOGO_K * k / 2
        y = cy + r
        x = cx + r
        # horizontal run, bend, vertical run, round caps
        d.rectangle([P(PIPE_X0), P(y) - half, P(cx), P(y) + half], fill=col)
        d.rectangle([P(x) - half, P(PIPE_Y1), P(x) + half, P(cy)], fill=col)
        R = r * LOGO_K * k
        d.arc([P(cx) - R - half, P(cy) - R - half, P(cx) + R + half, P(cy) + R + half],
              0, 90, fill=col, width=int(round(2 * half)))
        for ex, ey in ((PIPE_X0, y), (x, PIPE_Y1)):
            d.ellipse([P(ex) - half, P(ey) - half, P(ex) + half, P(ey) + half], fill=col)
    nut = ((15, (110, 63, 24)), (9, COPPER)) if simple else zip(NUT_R, ((110, 63, 24), COPPER_D, COPPER))
    for rr, col in nut:
        R = rr * LOGO_K * k
        d.ellipse([P(NUT[0]) - R, P(NUT[1]) - R, P(NUT[0]) + R, P(NUT[1]) + R], fill=col)
    # bubble: smooth radial fill (no concentric outline rings → no moiré)
    bx, by, br = BUBBLE
    BX, BY, BR = P(bx), P(by), br * LOGO_K * k
    fx, fy = BX - 0.24 * BR, BY - 0.36 * BR
    reach = 1.45 * BR
    bub = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    x0, y0, x1, y1 = int(BX - BR) - 1, int(BY - BR) - 1, int(BX + BR) + 2, int(BY + BR) + 2
    pix = bub.load()
    for yy in range(max(0, y0), min(S, y1)):
        for xx in range(max(0, x0), min(S, x1)):
            if (xx - BX) ** 2 + (yy - BY) ** 2 <= BR * BR:
                t = min(1.0, math.hypot(xx - fx, yy - fy) / reach)
                pix[xx, yy] = _bubble_rgba(t)
    im = Image.alpha_composite(im, bub)
    d = ImageDraw.Draw(im, "RGBA")
    rim = max(1, int(round((3.2 if simple else 1.8) * LOGO_K * k)))
    d.ellipse([BX - BR, BY - BR, BX + BR, BY + BR], outline=SOAP + (255,), width=rim)
    hx, hy = P(bx - 10), P(by - 11)
    d.ellipse([hx - 8.5 * LOGO_K * k, hy - 4.8 * LOGO_K * k, hx + 8.5 * LOGO_K * k, hy + 4.8 * LOGO_K * k],
              fill=(255, 255, 255, 150))
    return im.resize((px, px), Image.LANCZOS)


def mark_svg():
    cx, cy = BEND_C
    parts = []
    for off, w, col in _pipe_layers():
        r = BEND_R + off
        y, x = cy + r, cx + r
        dpath = "M%.2f %.2fH%.2fA%.2f %.2f 0 0 0 %.2f %.2fV%.2f" % (
            _u(PIPE_X0), _u(y), _u(cx), r * LOGO_K, r * LOGO_K, _u(x), _u(cy), _u(PIPE_Y1))
        parts.append('  <path d="%s" stroke="%s" stroke-width="%.2f" stroke-linecap="round"/>' % (dpath, _hex(col), w * LOGO_K))
    for rr, col in zip(NUT_R, ((110, 63, 24), COPPER_D, COPPER)):
        parts.append('  <circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s"/>' % (_u(NUT[0]), _u(NUT[1]), rr * LOGO_K, _hex(col)))
    bx, by, br = BUBBLE
    parts.append('  <circle cx="%.2f" cy="%.2f" r="%.2f" fill="url(#b)" stroke="%s" stroke-width="%.2f"/>' % (
        _u(bx), _u(by), br * LOGO_K, _hex(SOAP), 1.8 * LOGO_K))
    parts.append('  <ellipse cx="%.2f" cy="%.2f" rx="%.2f" ry="%.2f" fill="#fff" fill-opacity=".59"/>' % (
        _u(bx - 10), _u(by - 11), 8.5 * LOGO_K, 4.8 * LOGO_K))
    return """<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" fill="none">
  <!-- generated by scripts/make_assets.py (logo mark) -->
  <defs>
    <radialGradient id="b" cx="38%%" cy="32%%" r="72%%">
      <stop offset="0%%" stop-color="#9fe8e2" stop-opacity=".97"/>
      <stop offset="40%%" stop-color="%s" stop-opacity=".92"/>
      <stop offset="75%%" stop-color="%s" stop-opacity=".8"/>
      <stop offset="100%%" stop-color="%s" stop-opacity=".72"/>
    </radialGradient>
  </defs>
%s
</svg>
""" % (_hex(SOAP), _hex(SHEEN), _hex(SOAP), "\n".join(parts))


def logo512():
    big = render_mark(512)
    big.save(SITE / "logo-512.png", optimize=True)
    big.save(ROOT / "logo-512.png", optimize=True)
    render_mark(192).save(SITE / "icon-192.png", optimize=True)
    render_mark(180).save(SITE / "apple-touch-icon.png", optimize=True)
    i32, i16 = render_mark(32, ss=16), render_mark(16, ss=32)
    i32.save(SITE / "favicon-32.png", optimize=True)
    i16.save(SITE / "favicon-16.png", optimize=True)
    i32.save(SITE / "favicon.ico", sizes=[(16, 16), (32, 32)], append_images=[i16])
    svg = mark_svg()
    (SITE / "logo.svg").write_text(svg)
    (SITE / "favicon.svg").write_text(svg)
    print("logo-512.png, icon-192.png, apple-touch-icon.png, favicon-32/16.png, favicon.ico, logo.svg, favicon.svg")


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
        ("$ failshut audit --project ~/app", SOAP, mono),
        ("", CONCRETE, mono),
        ("FAIL OPEN — 1 finding(s) where policy may not hold:", FAIL, mono),
        ("", CONCRETE, mono),
        ("[1] hook-missing  (claude-code)", (255, 255, 255), mono),
        ("    file:     ~/app/.claude/settings.json", CONCRETE, mono_sm),
        ('    evidence: PreToolUse command', CONCRETE, mono_sm),
        ('              "$CLAUDE_PROJECT_DIR"/.claude/hooks/block-src-writes.sh', CONCRETE, mono_sm),
        ("              — missing", FAIL, mono_sm),
        ("    fix:      Restore the hook script. A hook that cannot", CONCRETE, mono_sm),
        ("              start lets the tool call through.", CONCRETE, mono_sm),
        ("", CONCRETE, mono),
        ("Summary: 1 FAIL OPEN · 5 FAIL SHUT checks · 1 config(s)", CONCRETE, mono),
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
                d.text((bbox2[2], y), " · 5 FAIL SHUT checks · 1 config(s)", fill=SOAP, font=fnt)
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
    import sys
    if sys.argv[1:] == ["demo"]:
        demo_gif()
        sys.exit(0)
    if sys.argv[1:] == ["logo"]:
        logo512()
        sys.exit(0)
    logo512()
    card(1200, 630, "og.png")
    card(1280, 640, "social-preview.png")
    shutil.copy(SITE / "og.png", SITE / "og-image.png")
    demo_gif()
