// Seeded PRNG for per-VU Terraform-like variation (mulberry32).

/**
 * @param {number} seed integer seed
 * @returns {() => number} float in [0, 1)
 */
export function mulberry32(seed) {
  let a = seed >>> 0;
  return function next() {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/**
 * @param {() => number} rnd
 * @param {number} min inclusive
 * @param {number} max inclusive
 */
export function randInt(rnd, min, max) {
  return min + Math.floor(rnd() * (max - min + 1));
}

/**
 * @param {() => number} rnd
 * @param {number} min
 * @param {number} max
 */
export function randFloat(rnd, min, max) {
  return min + rnd() * (max - min);
}

/**
 * Stable seed from VU + optional run id.
 * @param {number} vu
 * @param {string} [runId]
 */
export function seedForVu(vu, runId) {
  const s = `${runId || 'tf'}:${vu}`;
  let h = 2166136261;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return h >>> 0;
}
