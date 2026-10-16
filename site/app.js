(function () {
  const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  const needle = document.querySelector(".needle");
  const wipe = document.querySelector(".soap-wipe");
  const bubbles = Array.from(document.querySelectorAll(".bubble"));
  const stage = document.querySelector(".stage-readout");
  const gaugeState = document.querySelector(".gauge-state");
  const replayBtn = document.getElementById("replay");
  let running = false;

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

  function setStage(t) {
    if (stage) stage.textContent = t;
  }
  function setPsi(n) {
    if (gaugeState) gaugeState.textContent = String(n);
  }

  function easeOutCubic(t) {
    return 1 - Math.pow(1 - t, 3);
  }

  function animateNeedle(fromDeg, toDeg, dur, onDone) {
    if (!needle) { if (onDone) onDone(); return; }
    if (reduce) {
      needle.style.transform = "rotate(" + toDeg + "deg)";
      if (onDone) onDone();
      return;
    }
    const t0 = performance.now();
    function frame(now) {
      const p = Math.min(1, (now - t0) / dur);
      const e = easeOutCubic(p);
      const deg = fromDeg + (toDeg - fromDeg) * e;
      needle.style.transform = "rotate(" + deg + "deg)";
      setPsi(Math.round(((deg + 100) / 142) * 140));
      if (p < 1) requestAnimationFrame(frame);
      else if (onDone) onDone();
    }
    requestAnimationFrame(frame);
  }

  function animateWipe(dur, onDone) {
    if (!wipe) { if (onDone) onDone(); return; }
    const full = 1320;
    if (reduce) {
      wipe.setAttribute("width", String(full));
      if (onDone) onDone();
      return;
    }
    wipe.setAttribute("width", "0");
    const t0 = performance.now();
    function frame(now) {
      const p = Math.min(1, (now - t0) / dur);
      const e = 1 - Math.pow(1 - p, 2);
      wipe.setAttribute("width", String(full * e));
      if (p < 1) requestAnimationFrame(frame);
      else if (onDone) onDone();
    }
    requestAnimationFrame(frame);
  }

  function growBubble(el, target, onDone) {
    if (!el) { if (onDone) onDone(); return; }
    const baseCy = parseFloat(el.getAttribute("cy")) || 0;
    if (reduce) {
      el.setAttribute("r", String(target.r));
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
      el.setAttribute("r", String(target.r * e));
      el.setAttribute("cy", String(baseCy + (target.cy - baseCy) * e));
      el.setAttribute("opacity", String(0.15 + 0.7 * e));
      if (p < 1) requestAnimationFrame(frame);
      else if (onDone) onDone();
    }
    requestAnimationFrame(frame);
  }

  function resetBubbles() {
    bubbles.forEach(function (b, i) {
      b.setAttribute("r", "0");
      b.setAttribute("opacity", "0");
      const cx = [4, 26, -22, 14, -10, 32, -28, 8][i] || 0;
      b.setAttribute("cx", String(cx));
      b.setAttribute("cy", String(-8 - i * 4));
    });
  }

  function nucleate(onDone) {
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


  let floatRaf = null;
  function startFloat() {
    if (reduce) return;
    if (floatRaf) cancelAnimationFrame(floatRaf);
    const base = bubbles.map(function (el) {
      return {
        el: el,
        cx: parseFloat(el.getAttribute("cx")) || 0,
        cy: parseFloat(el.getAttribute("cy")) || 0,
        r: parseFloat(el.getAttribute("r")) || 10,
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
        // respawn soft
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
    resetBubbles();
    if (wipe) wipe.setAttribute("width", "0");
    if (needle) needle.style.transform = "rotate(-100deg)";
    setPsi(0);

    setStage("1 · PRESSURIZE");
    animateNeedle(-100, 42, reduce ? 0 : 1400, function () {
      setPsi(140);
      setStage("2 · WIPE SOAP");
      animateWipe(reduce ? 0 : 1100, function () {
        setStage("3 · BUBBLES ON FAIL-OPEN ONLY");
        nucleate(function () {
          setStage("LEAK FOUND · FAIL OPEN AT JOINT");
          startFloat();
          running = false;
          if (replayBtn) replayBtn.hidden = false;
        });
      });
    });
  }

  // fittings accordion
  document.querySelectorAll(".fitting").forEach(function (btn) {
    btn.addEventListener("click", function () {
      const open = btn.getAttribute("aria-expanded") === "true";
      document.querySelectorAll(".fitting").forEach(function (b) {
        b.setAttribute("aria-expanded", "false");
      });
      btn.setAttribute("aria-expanded", open ? "false" : "true");
    });
  });

  // copy buttons
  document.querySelectorAll("[data-copy]").forEach(function (btn) {
    btn.addEventListener("click", function () {
      const id = btn.getAttribute("data-copy");
      const el = document.getElementById(id);
      const text = el ? el.textContent : "";
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

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", runSequence);
  } else {
    runSequence();
  }
})();
