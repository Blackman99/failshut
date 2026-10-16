(function () {
  const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  const replayBtn = document.getElementById("replay");
  let running = false;
  let floatRaf = null;

  const BUBBLE_TARGETS = [
    { r: 26, cy: -32, delay: 0 },
    { r: 20, cy: -62, delay: 140 },
    { r: 18, cy: -88, delay: 260 },
    { r: 15, cy: -118, delay: 400 },
    { r: 13, cy: -148, delay: 520 },
    { r: 12, cy: -176, delay: 660 },
    { r: 11, cy: -200, delay: 800 },
    { r: 10, cy: -228, delay: 940 },
  ];

  function activeManifold() {
    const mob = window.matchMedia("(max-width: 720px)").matches;
    return {
      mob: mob,
      needle: document.querySelector(mob ? ".needle-mob" : ".needle"),
      wipe: document.querySelector(mob ? ".soap-wipe-mob" : ".soap-wipe:not(.soap-wipe-mob)"),
      bubbles: Array.from(document.querySelectorAll(mob ? ".bubble-mob" : ".manifold-desk .bubble")),
      stage: document.querySelector(mob ? ".stage-readout-mob" : ".manifold-desk .stage-readout"),
      gaugeState: document.querySelector(mob ? ".gauge-state-mob" : ".manifold-desk .gauge-state"),
      wipeIsVertical: mob,
    };
  }

  function setStage(el, t) {
    if (el) el.textContent = t;
  }
  function setPsi(el, n) {
    if (el) el.textContent = String(n);
  }

  function easeOutCubic(t) {
    return 1 - Math.pow(1 - t, 3);
  }

  function animateNeedle(needle, gaugeState, fromDeg, toDeg, dur, onDone) {
    if (!needle) { if (onDone) onDone(); return; }
    if (reduce) {
      needle.style.transform = "rotate(" + toDeg + "deg)";
      setPsi(gaugeState, 140);
      if (onDone) onDone();
      return;
    }
    const t0 = performance.now();
    function frame(now) {
      const p = Math.min(1, (now - t0) / dur);
      const e = easeOutCubic(p);
      const deg = fromDeg + (toDeg - fromDeg) * e;
      needle.style.transform = "rotate(" + deg + "deg)";
      setPsi(gaugeState, Math.round(((deg + 100) / 142) * 140));
      if (p < 1) requestAnimationFrame(frame);
      else if (onDone) onDone();
    }
    requestAnimationFrame(frame);
  }

  function animateWipe(wipe, vertical, dur, onDone) {
    if (!wipe) { if (onDone) onDone(); return; }
    const full = vertical ? 360 : 1320;
    const attr = vertical ? "height" : "width";
    if (reduce) {
      wipe.setAttribute(attr, String(full));
      if (onDone) onDone();
      return;
    }
    wipe.setAttribute(attr, "0");
    const t0 = performance.now();
    function frame(now) {
      const p = Math.min(1, (now - t0) / dur);
      const e = 1 - Math.pow(1 - p, 2);
      wipe.setAttribute(attr, String(full * e));
      if (p < 1) requestAnimationFrame(frame);
      else if (onDone) onDone();
    }
    requestAnimationFrame(frame);
  }

  function growBubble(el, target, onDone) {
    if (!el) { if (onDone) onDone(); return; }
    const baseCy = parseFloat(el.getAttribute("cy")) || 0;
    if (reduce) {
      el.setAttribute("r", String(target.r * 0.7));
      el.setAttribute("cy", String(target.cy));
      el.setAttribute("opacity", "0.8");
      if (onDone) onDone();
      return;
    }
    const t0 = performance.now();
    const dur = 900;
    function frame(now) {
      const p = Math.min(1, (now - t0) / dur);
      const e = easeOutCubic(p);
      el.setAttribute("r", String(target.r * 0.7 * e));
      el.setAttribute("cy", String(baseCy + (target.cy - baseCy) * e));
      el.setAttribute("opacity", String(0.15 + 0.7 * e));
      if (p < 1) requestAnimationFrame(frame);
      else if (onDone) onDone();
    }
    requestAnimationFrame(frame);
  }

  function resetBubbles(bubbles) {
    bubbles.forEach(function (b, i) {
      b.setAttribute("r", "0");
      b.setAttribute("opacity", "0");
      const cx = [4, 26, -22, 14, -10, 32, -28, 8][i] || 0;
      // mobile bubbles drift right of vertical pipe
      const mobCx = [18, 36, 48, 28, 52, 40, 58, 34][i] || 18;
      const isMob = b.classList.contains("bubble-mob");
      b.setAttribute("cx", String(isMob ? mobCx : cx));
      b.setAttribute("cy", String(-8 - i * 4));
    });
  }

  function nucleate(bubbles, onDone) {
    let left = bubbles.length;
    if (left === 0) { if (onDone) onDone(); return; }
    bubbles.forEach(function (el, i) {
      const target = BUBBLE_TARGETS[i] || BUBBLE_TARGETS[0];
      const start = function () {
        growBubble(el, target, function () {
          left -= 1;
          if (left <= 0 && onDone) onDone();
        });
      };
      if (reduce) start();
      else setTimeout(start, target.delay);
    });
  }

  function startFloat(bubbles) {
    if (reduce) return;
    if (floatRaf) cancelAnimationFrame(floatRaf);
    const base = bubbles.map(function (el) {
      return {
        el: el,
        cx: parseFloat(el.getAttribute("cx")) || 0,
        cy: parseFloat(el.getAttribute("cy")) || 0,
        phase: Math.random() * Math.PI * 2,
      };
    });
    const t0 = performance.now();
    function tick(now) {
      const t = (now - t0) / 1000;
      base.forEach(function (b, i) {
        const bob = Math.sin(t * 1.4 + b.phase) * 6;
        const drift = Math.sin(t * 0.7 + i) * 3;
        b.el.setAttribute("cy", String(b.cy + bob - t * 4));
        b.el.setAttribute("cx", String(b.cx + drift));
        if (parseFloat(b.el.getAttribute("cy")) < -200) {
          b.cy = -20;
          b.el.setAttribute("cy", "-20");
          b.el.setAttribute("r", String(8 + Math.random() * 12));
          b.el.setAttribute("opacity", "0.75");
        }
      });
      floatRaf = requestAnimationFrame(tick);
    }
    floatRaf = requestAnimationFrame(tick);
  }

  function runSequence() {
    if (running) return;
    if (floatRaf) { cancelAnimationFrame(floatRaf); floatRaf = null; }
    running = true;
    if (replayBtn) replayBtn.hidden = true;

    const m = activeManifold();
    resetBubbles(m.bubbles);
    if (m.wipe) {
      m.wipe.setAttribute(m.wipeIsVertical ? "height" : "width", "0");
    }
    if (m.needle) m.needle.style.transform = "rotate(-100deg)";
    setPsi(m.gaugeState, 0);

    setStage(m.stage, "1 · PRESSURIZE");
    animateNeedle(m.needle, m.gaugeState, -100, 42, reduce ? 0 : 1400, function () {
      setPsi(m.gaugeState, 140);
      setStage(m.stage, "2 · WIPE SOAP");
      animateWipe(m.wipe, m.wipeIsVertical, reduce ? 0 : 1100, function () {
        setStage(m.stage, "3 · BUBBLES ON FAIL-OPEN ONLY");
        nucleate(m.bubbles, function () {
          setStage(m.stage, "LEAK FOUND · FAIL OPEN AT JOINT");
          startFloat(m.bubbles);
          running = false;
          if (replayBtn) replayBtn.hidden = false;
        });
      });
    });
  }

  document.querySelectorAll(".fitting").forEach(function (btn) {
    btn.addEventListener("click", function () {
      const open = btn.getAttribute("aria-expanded") === "true";
      document.querySelectorAll(".fitting").forEach(function (b) {
        b.setAttribute("aria-expanded", "false");
      });
      btn.setAttribute("aria-expanded", open ? "false" : "true");
    });
  });

  // Copy: prefer data-full so shortened display still copies complete command
  document.querySelectorAll("[data-copy]").forEach(function (btn) {
    btn.addEventListener("click", function () {
      const id = btn.getAttribute("data-copy");
      const el = document.getElementById(id);
      let text = "";
      if (el) {
        text = el.getAttribute("data-full") || el.textContent || "";
      }
      if (!text) return;
      function done() {
        const prev = btn.textContent;
        btn.textContent = "Copied";
        setTimeout(function () { btn.textContent = prev; }, 1400);
      }
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text.trim()).then(done).catch(done);
      } else {
        done();
      }
    });
  });

  if (replayBtn) {
    replayBtn.addEventListener("click", runSequence);
  }

  // re-run when crossing mobile/desktop so the visible manifold animates
  let lastMob = window.matchMedia("(max-width: 720px)").matches;
  window.matchMedia("(max-width: 720px)").addEventListener("change", function (e) {
    if (e.matches !== lastMob) {
      lastMob = e.matches;
      running = false;
      if (floatRaf) { cancelAnimationFrame(floatRaf); floatRaf = null; }
      runSequence();
    }
  });

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", runSequence);
  } else {
    runSequence();
  }
})();
