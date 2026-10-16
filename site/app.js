(function () {
  const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  const bubbles = [
    { el: document.querySelector(".b1"), r: 18, delay: 1200 },
    { el: document.querySelector(".b2"), r: 14, delay: 1450 },
    { el: document.querySelector(".b3"), r: 12, delay: 1700 },
    { el: document.querySelector(".b4"), r: 10, delay: 1950 },
  ];
  function grow(b) {
    if (!b.el) return;
    if (reduce) {
      b.el.setAttribute("r", String(b.r));
      b.el.setAttribute("opacity", "0.75");
      return;
    }
    const start = performance.now();
    const dur = 1100;
    function frame(t) {
      const p = Math.min(1, (t - start) / dur);
      const e = 1 - Math.pow(1 - p, 3);
      b.el.setAttribute("r", String(b.r * e));
      b.el.setAttribute("opacity", String(0.85 * e));
      // float up slightly
      const cy = parseFloat(b.el.getAttribute("data-cy") || b.el.getAttribute("cy"));
      if (!b.el.hasAttribute("data-cy")) b.el.setAttribute("data-cy", String(cy));
      b.el.setAttribute("cy", String(cy - 36 * e));
      if (p < 1) requestAnimationFrame(frame);
    }
    requestAnimationFrame(frame);
  }
  function start() {
    document.body.classList.add("ready");
    const wipe = document.querySelector(".soap-wipe");
    if (wipe) {
      if (reduce) wipe.setAttribute("width", "1120");
      else {
        wipe.setAttribute("width", "0");
        let w = 0;
        const t0 = performance.now();
        function wf(t) {
          const p = Math.min(1, (t - t0 - 400) / 1100);
          if (p > 0) {
            wipe.setAttribute("width", String(1120 * (1 - Math.pow(1 - p, 2))));
          }
          if (p < 1) requestAnimationFrame(wf);
        }
        requestAnimationFrame(wf);
      }
    }
    bubbles.forEach((b) => {
      if (reduce) grow(b);
      else setTimeout(() => grow(b), b.delay);
    });
  }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", start);
  else start();
})();
