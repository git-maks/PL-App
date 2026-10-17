"""Generates the PWA icons (pixel-art robot — the university's mascot is a robot).
Run: python3 scripts/build-app-icons.py  (outputs are committed to static/icons)."""
from pathlib import Path
from PIL import Image, ImageDraw

OUT = Path(__file__).resolve().parent.parent / "static" / "icons"
OUT.mkdir(parents=True, exist_ok=True)
NAVY, WHITE, GOLD, DARK = (31, 90, 166), (255, 255, 255), (245, 191, 72), (0, 27, 61)
ROBOT = [
    "................",
    ".......GG.......",
    ".......WW.......",
    "...WWWWWWWWWW...",
    "..WWWWWWWWWWWW..",
    "..WWNNWWWWNNWW..",
    "..WWNNWWWWNNWW..",
    "..WWWWWWWWWWWW..",
    "..WWWNNNNNNWWW..",
    "..WWWWWWWWWWWW..",
    "...WWWWWWWWWW...",
    ".....GGGGGG.....",
    "...GGGGGGGGGG...",
    "..GGGGWWWWGGGG..",
    "..GGGGWWWWGGGG..",
    "................",
]
COL = {"W": WHITE, "G": GOLD, "N": DARK}

def draw(size, art_ratio, rounded, bg=NAVY):
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    if rounded:
        d.rounded_rectangle([0, 0, size - 1, size - 1], radius=int(size * 0.22), fill=bg)
    else:
        d.rectangle([0, 0, size, size], fill=bg)
    px = int(size * art_ratio / 16)
    off = (size - px * 16) // 2
    for y, row in enumerate(ROBOT):
        for x, ch in enumerate(row):
            if ch in COL:
                d.rectangle([off + x * px, off + y * px, off + (x + 1) * px - 1, off + (y + 1) * px - 1], fill=COL[ch])
    return img

draw(192, 0.72, True).save(OUT / "icon-192.png")
draw(512, 0.72, True).save(OUT / "icon-512.png")
draw(512, 0.56, False).save(OUT / "maskable-512.png")   # content inside the 80% safe zone
draw(180, 0.66, False).convert("RGB").save(OUT / "apple-touch-icon.png")
# Monochrome badge for Android notifications (white on transparent).
badge = Image.new("RGBA", (72, 72), (0, 0, 0, 0))
bd = ImageDraw.Draw(badge)
px = 4; off = (72 - 64) // 2
for y, row in enumerate(ROBOT):
    for x, ch in enumerate(row):
        if ch in "WG":
            bd.rectangle([off + x*px, off + y*px, off + (x+1)*px - 1, off + (y+1)*px - 1], fill=(255, 255, 255, 255))
badge.save(OUT / "badge-72.png")

# SVG favicon
rects = "".join(f'<rect x="{x+2}" y="{y+2}" width="1" height="1" fill="#{"%02x%02x%02x" % COL[ch]}"/>'
                for y, row in enumerate(ROBOT) for x, ch in enumerate(row) if ch in COL)
(OUT / "icon.svg").write_text(f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" shape-rendering="crispEdges"><rect width="20" height="20" rx="4" fill="#1f5aa6"/>{rects}</svg>\n')
print("ok")
