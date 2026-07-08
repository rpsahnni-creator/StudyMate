// One-off: build app icons from the real Kiji logo (transparent PNG).
// iOS icons must be opaque, so we composite the logo onto a solid dark
// background with padding. Android adaptive foreground keeps transparency.
const fs = require("fs");
const path = require("path");
const { PNG } = require("pngjs");

const IMAGES = path.join(__dirname, "..", "assets", "images");
const logo = PNG.sync.read(fs.readFileSync(path.join(IMAGES, "kiji_logo.png")));

const BG = { r: 0x0e, g: 0x0e, b: 0x12 }; // dark charcoal like the shown splash

// Bilinear resize of an RGBA PNG into a target w/h, returns {data,width,height}.
function resize(src, tw, th) {
  const out = Buffer.alloc(tw * th * 4);
  const sx = src.width / tw;
  const sy = src.height / th;
  for (let y = 0; y < th; y++) {
    const fy = (y + 0.5) * sy - 0.5;
    const y0 = Math.max(0, Math.floor(fy));
    const y1 = Math.min(src.height - 1, y0 + 1);
    const wy = fy - y0;
    for (let x = 0; x < tw; x++) {
      const fx = (x + 0.5) * sx - 0.5;
      const x0 = Math.max(0, Math.floor(fx));
      const x1 = Math.min(src.width - 1, x0 + 1);
      const wx = fx - x0;
      const o = (y * tw + x) * 4;
      for (let c = 0; c < 4; c++) {
        const p00 = src.data[(y0 * src.width + x0) * 4 + c];
        const p01 = src.data[(y0 * src.width + x1) * 4 + c];
        const p10 = src.data[(y1 * src.width + x0) * 4 + c];
        const p11 = src.data[(y1 * src.width + x1) * 4 + c];
        const top = p00 + (p01 - p00) * wx;
        const bot = p10 + (p11 - p10) * wx;
        out[o + c] = Math.round(top + (bot - top) * wy);
      }
    }
  }
  return { data: out, width: tw, height: th };
}

// Build a size x size canvas; logo scaled to `scale` fraction, centered.
// opaque=true fills BG (iOS icon / splash), false leaves transparent (Android fg).
function build(size, scale, opaque) {
  const png = new PNG({ width: size, height: size });
  for (let i = 0; i < png.data.length; i += 4) {
    if (opaque) {
      png.data[i] = BG.r;
      png.data[i + 1] = BG.g;
      png.data[i + 2] = BG.b;
      png.data[i + 3] = 255;
    } else {
      png.data[i] = png.data[i + 1] = png.data[i + 2] = png.data[i + 3] = 0;
    }
  }
  const inner = Math.round(size * scale);
  const r = resize(logo, inner, inner);
  const off = Math.round((size - inner) / 2);
  for (let y = 0; y < inner; y++) {
    for (let x = 0; x < inner; x++) {
      const si = (y * inner + x) * 4;
      const a = r.data[si + 3] / 255;
      if (a === 0) continue;
      const di = ((y + off) * size + (x + off)) * 4;
      for (let c = 0; c < 3; c++) {
        png.data[di + c] = Math.round(r.data[si + c] * a + png.data[di + c] * (1 - a));
      }
      png.data[di + 3] = Math.max(png.data[di + 3], r.data[si + 3]);
    }
  }
  return PNG.sync.write(png);
}

fs.writeFileSync(path.join(IMAGES, "icon.png"), build(1024, 0.72, true));
fs.writeFileSync(path.join(IMAGES, "adaptive-icon.png"), build(1024, 0.6, false));
fs.writeFileSync(path.join(IMAGES, "splash-icon.png"), build(1024, 0.66, false));
console.log("icons written: icon.png (opaque), adaptive-icon.png (fg), splash-icon.png (fg)");
