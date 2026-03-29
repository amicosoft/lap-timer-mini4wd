# Mini 4WD Lap Timer — User Guide

## Overview

The lap timer tracks your Mini 4WD car's lap times in real time using an IR sensor connected to an Arduino. Open `http://localhost:8080` in your browser to access the timer.

![Full app layout](images/lap-timer-main-layout.png)

---

## Timers Layout

![Current Lap and Race timer display](images/lap-timers.png)

### Current Lap (large orange display)
Shows the time elapsed for the lap currently in progress. Resets to `0:00.0` at the start of each new lap.

### Race (blue display below)
Shows the total time elapsed for the entire race session, excluding any paused time.

### Scoreboard

![Scoreboard display](images/scoreboard.png)

<table border="1" cellpadding="8" cellspacing="0">
  <thead>
    <tr>
      <th>Field</th>
      <th>Location</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Total Time</strong></td>
      <td>Top left</td>
      <td>Sum of all completed laps</td>
    </tr>
    <tr>
      <td><strong>Lap</strong></td>
      <td>Top right</td>
      <td>Number of completed laps</td>
    </tr>
    <tr>
      <td><strong>Last Lap</strong></td>
      <td>Bottom left</td>
      <td>Time of the most recently completed lap</td>
    </tr>
    <tr>
      <td><strong>Best Lap</strong></td>
      <td>Bottom right</td>
      <td>Fastest lap of the session — highlighted in gold</td>
    </tr>
  </tbody>
</table>

### Lap History
A scrolling list of every completed lap. The best lap is marked with a ★.

---

## Buttons

### Start
Appears when the timer is idle. Arms the sensor and waits for the car to cross the IR beam for the first time. A 3-beep countdown plays to signal the timer is ready.

### Stop
Ends the race session immediately. The session is saved to history. A descending sound plays to confirm.

### Pause
Freezes both the current lap and race timers. The car's position is noted — use this when you need to interrupt a run.

### Resume
Appears after pausing. Clicking Resume puts the timer into a waiting state — **the timers do not start immediately**. Place the car at the track and let it pass the IR sensor; the moment it crosses, the race timer resumes and a fresh lap begins. A GO sound plays on the trigger.

### Reset Time
Appears when paused. Removes the current incomplete lap from the race timer without ending the session. Useful when you want to discard a partial lap before resuming.

### Reset
Clears all lap data and returns the timer to idle. The session is saved to history before resetting. A descending arpeggio plays to confirm.

---

## Race Sounds

Sounds can be toggled on or off in the **Settings** panel at the bottom of the page. The preference is saved between sessions.

<table border="1" cellpadding="8" cellspacing="0">
  <thead>
    <tr>
      <th>Event</th>
      <th>Sound</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Start armed</td>
      <td>3 short beeps (countdown)</td>
    </tr>
    <tr>
      <td>Car crosses start line</td>
      <td>Rising two-note GO burst</td>
    </tr>
    <tr>
      <td>Lap completed</td>
      <td>Bell ping</td>
    </tr>
    <tr>
      <td>New best lap</td>
      <td>Ascending arpeggio (C–E–G–C)</td>
    </tr>
    <tr>
      <td>Pause</td>
      <td>Descending two-tone</td>
    </tr>
    <tr>
      <td>Resume clicked</td>
      <td>Rising two-tone</td>
    </tr>
    <tr>
      <td>Stop</td>
      <td>Low descending blip</td>
    </tr>
    <tr>
      <td>Reset Time</td>
      <td>Soft tick</td>
    </tr>
    <tr>
      <td>Reset</td>
      <td>Descending arpeggio</td>
    </tr>
  </tbody>
</table>

---

## Typical Race Flow

1. Click **Start** — 3 beeps confirm the sensor is armed
2. Release the car — the timer starts the moment it crosses the IR beam
3. Each time the car completes a lap, the lap time is recorded automatically
4. Click **Stop** when the race is over — the session is saved
5. Click **Reset** to clear and start fresh for the next race

## Pausing Mid-Race

1. Click **Pause** — both timers freeze
2. Optionally click **Reset Time** to discard the current partial lap
3. Click **Resume** — the timer waits in a ready state
4. Send the car through the IR beam — race resumes with a fresh lap

---

## Session History

Past sessions are stored and accessible via the **Session History** panel at the bottom of the page. Click the panel header to expand it, then click any session to see its individual lap times.

![Session History panel](images/session-history.png)

---

## Connection Status

The indicator in the top-right corner shows the live connection to the server:

![Connected indicator](images/ir-connected.png)

- **Green dot** — connected, sensor active
- **Red dot** — disconnected, attempting to reconnect automatically
