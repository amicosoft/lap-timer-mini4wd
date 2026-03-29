'use strict';

const els = {
  dot:         document.getElementById('status-dot'),
  statusText:  document.getElementById('status-text'),
  liveClock:   document.getElementById('live-clock'),
  raceTime:    document.getElementById('race-time'),
  totalTime:   document.getElementById('total-time'),
  lapNumber:   document.getElementById('lap-number'),
  lastLap:     document.getElementById('last-lap'),
  bestLap:     document.getElementById('best-lap'),
  lapList:     document.getElementById('lap-list'),
  actions:     document.getElementById('actions'),
};

// --- Clock ---

let clockInterval = null;
let prevStatus    = null;
let prevLapCount  = 0;
let prevBestLap   = 0;

// --- Sound ---

let _audioCtx = null;
function getAudioCtx() {
  if (!_audioCtx) _audioCtx = new (window.AudioContext || window.webkitAudioContext)();
  if (_audioCtx.state === 'suspended') _audioCtx.resume();
  return _audioCtx;
}

function soundEnabled() {
  return localStorage.getItem('soundEnabled') !== 'false';
}

// Square-wave tone with sharp attack and clean cut-off
function playTone(freq, duration, delay = 0, type = 'square', vol = 0.12) {
  const ctx = getAudioCtx();
  const osc = ctx.createOscillator();
  const gain = ctx.createGain();
  osc.connect(gain);
  gain.connect(ctx.destination);
  osc.type = type;
  osc.frequency.value = freq;
  const t = ctx.currentTime + delay;
  gain.gain.setValueAtTime(0, t);
  gain.gain.linearRampToValueAtTime(vol, t + 0.005);
  gain.gain.setValueAtTime(vol, t + duration - 0.01);
  gain.gain.linearRampToValueAtTime(0, t + duration);
  osc.start(t);
  osc.stop(t + duration + 0.01);
}

function playCountdown() {
  // 3 punchy beeps (square wave) like race start lights, then silence before GO
  playTone(880, 0.1, 0.0);
  playTone(880, 0.1, 0.4);
  playTone(880, 0.1, 0.8);
}

function playGo() {
  // Rising two-note burst: short low → longer high
  playTone(660,  0.1,  0,    'square', 0.14);
  playTone(1100, 0.28, 0.1,  'square', 0.14);
}

function playLapBeep() {
  // Bell ping: two sine oscillators (fundamental + harmonic) with instant attack and long decay
  const ctx = getAudioCtx();
  [[1047, 0.28], [2093, 0.12]].forEach(([freq, vol]) => {
    const osc  = ctx.createOscillator();
    const gain = ctx.createGain();
    osc.connect(gain);
    gain.connect(ctx.destination);
    osc.type = 'sine';
    osc.frequency.value = freq;
    gain.gain.setValueAtTime(vol, ctx.currentTime);
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.8);
    osc.start(ctx.currentTime);
    osc.stop(ctx.currentTime + 0.82);
  });
}

function playBestLap() {
  // Ascending arpeggio: C5-E5-G5-C6
  [523, 659, 784, 1047].forEach((freq, i) => playTone(freq, 0.13, i * 0.09, 'sine', 0.18));
}

function playPause() {
  // Descending two-tone — things slowing down
  playTone(880, 0.09, 0,    'square', 0.10);
  playTone(587, 0.13, 0.10, 'square', 0.10);
}

function playResume() {
  // Quick rising two-tone (softer than GO)
  playTone(587, 0.08, 0,    'square', 0.10);
  playTone(880, 0.14, 0.09, 'square', 0.10);
}

function playStop() {
  // Short low descending blip
  playTone(440, 0.08, 0,    'square', 0.10);
  playTone(294, 0.14, 0.09, 'square', 0.10);
}

function playResetTime() {
  // Soft single low tick
  playTone(330, 0.07, 0, 'square', 0.08);
}

function playReset() {
  // Descending arpeggio — clearing everything out
  [784, 659, 523, 392].forEach((freq, i) => playTone(freq, 0.10, i * 0.08, 'square', 0.10));
}

function tickClock(sessionWall, lapWall) {
  if (clockInterval) { clearInterval(clockInterval); }
  clockInterval = setInterval(() => {
    els.liveClock.textContent = formatTime((Date.now() - lapWall) / 1000);
    els.raceTime.textContent  = formatTime((Date.now() - sessionWall) / 1000);
  }, 100);
}

function stopClock(lapText, raceText) {
  if (clockInterval) { clearInterval(clockInterval); clockInterval = null; }
  els.liveClock.textContent = lapText;
  els.raceTime.textContent  = raceText;
}

// --- WebSocket ---

let ws = null;
let reconnectTimer = null;

function connect() {
  ws = new WebSocket(`ws://${location.host}/ws`);

  ws.onopen = () => {
    els.dot.className = 'status-dot connected';
    els.statusText.textContent = 'Connected';
    if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null; }
  };

  ws.onmessage = (event) => {
    try { render(JSON.parse(event.data)); }
    catch (e) { console.error('parse error:', e); }
  };

  ws.onclose = ws.onerror = () => {
    els.dot.className = 'status-dot disconnected';
    els.statusText.textContent = 'Disconnected — reconnecting...';
    ws = null;
    reconnectTimer = setTimeout(connect, 2000);
  };
}

// --- Render ---

function render(state) {
  if (soundEnabled()) {
    if (state.status === 'armed'   && prevStatus === 'idle')    playCountdown();
    if (state.status === 'running' && prevStatus === 'armed')   playGo();
    if (state.status === 'running' && state.lapNumber > prevLapCount) {
      if (state.bestLap < prevBestLap || prevBestLap === 0) playBestLap();
      else playLapBeep();
    }
  }
  if (state.status === 'idle') { prevBestLap = 0; }

  els.lapNumber.textContent = state.lapNumber;
  els.totalTime.textContent = formatTime(state.totalTime);

  const completedTotal = (state.lapHistory || []).reduce((a, b) => a + b, 0);

  const now = Date.now();
  const sessionWall = now - state.totalTime * 1000;

  switch (state.status) {
    case 'running':
      tickClock(sessionWall, now - (state.totalTime - completedTotal) * 1000);
      break;
    case 'paused':
      stopClock(
        formatTime(state.totalTime - completedTotal),
        formatTime(state.totalTime)
      );
      break;
    case 'stopped':
      stopClock('0:00.0', formatTime(state.totalTime));
      break;
    default: // idle, armed
      stopClock('0:00.0', '0:00.0');
  }

  prevStatus   = state.status;
  prevLapCount = state.lapNumber;
  prevBestLap  = state.bestLap || 0;

  if (state.lapNumber > 0) {
    els.lastLap.textContent = state.lastLap.toFixed(2) + 's';
    els.bestLap.textContent = state.bestLap.toFixed(2) + 's';
  } else {
    els.lastLap.textContent = '-.--s';
    els.bestLap.textContent = '-.--s';
  }

  buildLapList(state.lapHistory, state.bestLap);
  renderButtons(state.status);
}

// --- Buttons ---

// Button definitions per status: [label, endpoint, style, soundFn]
const BUTTON_SETS = {
  idle:    [['Start', '/arm', 'btn-start', null]],
  armed:   [['Stop', '/stop', 'btn-stop', playStop], ['Reset', '/reset', 'btn-reset', playReset]],
  running: [['Pause', '/pause', 'btn-pause', playPause], ['Stop', '/stop', 'btn-stop', playStop]],
  paused:  [['Resume', '/arm', 'btn-start', playResume], ['Reset Time', '/reset-time', 'btn-reset-time', playResetTime], ['Stop', '/stop', 'btn-stop', playStop]],
  stopped: [['Reset', '/reset', 'btn-reset', playReset]],
};

function renderButtons(status) {
  const defs = BUTTON_SETS[status] || BUTTON_SETS.idle;
  els.actions.innerHTML = '';
  defs.forEach(([label, endpoint, cls, soundFn]) => {
    const btn = document.createElement('button');
    btn.textContent = label;
    btn.className = cls;
    btn.addEventListener('click', () => {
      if (soundEnabled() && soundFn) soundFn();
      fetch(endpoint, { method: 'POST' })
        .then(() => { if (endpoint === '/stop' || endpoint === '/reset') loadHistory(); })
        .catch(console.error);
    });
    els.actions.appendChild(btn);
  });
}

// --- Lap list ---

function buildLapList(history, bestLap) {
  if (!history || history.length === 0) {
    els.lapList.innerHTML = '<li class="placeholder">Waiting for first car...</li>';
    return;
  }
  const frag = document.createDocumentFragment();
  history.forEach((lap, i) => {
    const li = document.createElement('li');
    if (lap === bestLap) li.classList.add('best-lap');
    const num = document.createElement('span');
    num.textContent = `Lap ${i + 1}${lap === bestLap ? ' ★' : ''}`;
    const time = document.createElement('span');
    time.textContent = lap.toFixed(2) + 's';
    li.appendChild(num);
    li.appendChild(time);
    frag.appendChild(li);
  });
  els.lapList.innerHTML = '';
  els.lapList.appendChild(frag);
  els.lapList.scrollTop = els.lapList.scrollHeight;
}

// --- Helpers ---

function formatTime(totalSeconds) {
  if (!totalSeconds || totalSeconds <= 0) return '0:00.0';
  const m = Math.floor(totalSeconds / 60);
  const s = (totalSeconds % 60).toFixed(1);
  return `${m}:${s.padStart(4, '0')}`;
}

// --- Session history ---

const historyToggle  = document.getElementById('history-toggle');
const historyPanel   = document.getElementById('history-panel');
const historyList    = document.getElementById('history-list');
const historyChevron = document.getElementById('history-chevron');

historyToggle.addEventListener('click', () => {
  const open = !historyPanel.classList.contains('hidden');
  historyPanel.classList.toggle('hidden', open);
  historyChevron.textContent = open ? '▼' : '▲';
  if (!open) loadHistory();
});

function loadHistory() {
  fetch('/history')
    .then(r => r.json())
    .then(renderHistory)
    .catch(console.error);
}

function renderHistory(sessions) {
  if (!sessions || sessions.length === 0) {
    historyList.innerHTML = '<span class="placeholder">No sessions saved yet.</span>';
    return;
  }
  historyList.innerHTML = '';
  sessions.forEach(s => {
    const entry = document.createElement('div');
    entry.className = 'session-entry';

    const date = new Date(s.startedAt);
    const dateStr = date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], {hour: '2-digit', minute: '2-digit'});

    const header = document.createElement('div');
    header.className = 'session-header';
    header.innerHTML = `
      <span class="session-date">${dateStr}</span>
      <div class="session-summary">
        <span>${s.lapCount} laps</span>
        <span class="s-best">Best: ${s.bestLap > 0 ? s.bestLap.toFixed(2) + 's' : '-'}</span>
        <span>Total: ${formatTime(s.totalTime)}</span>
      </div>`;

    const lapsDiv = document.createElement('div');
    lapsDiv.className = 'session-laps';
    const ol = document.createElement('ol');
    (s.laps || []).forEach(lap => {
      const li = document.createElement('li');
      li.textContent = lap.toFixed(2) + 's';
      if (lap === s.bestLap) li.classList.add('best-lap');
      ol.appendChild(li);
    });
    lapsDiv.appendChild(ol);

    header.addEventListener('click', () => lapsDiv.classList.toggle('open'));
    entry.appendChild(header);
    entry.appendChild(lapsDiv);
    historyList.appendChild(entry);
  });
}

// --- Settings ---

const settingsToggle  = document.getElementById('settings-toggle');
const settingsPanel   = document.getElementById('settings-panel');
const settingsChevron = document.getElementById('settings-chevron');
const soundToggle     = document.getElementById('sound-toggle');

soundToggle.checked = soundEnabled();

settingsToggle.addEventListener('click', () => {
  const open = !settingsPanel.classList.contains('hidden');
  settingsPanel.classList.toggle('hidden', open);
  settingsChevron.textContent = open ? '▼' : '▲';
});

soundToggle.addEventListener('change', () => {
  localStorage.setItem('soundEnabled', soundToggle.checked ? 'true' : 'false');
  // Warm up AudioContext on first user interaction
  if (soundToggle.checked) getAudioCtx();
});

renderButtons('idle');
connect();
