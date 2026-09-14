---
name: wakagym-pros
description: >-
  Adapted pro running sessions and the Set-my-fitness selector for wakagym.
  Use when rotating quality days, naming Josh Kerr / Kipchoge / Ingebrigtsen
  sessions, scaling Train Like the Pros workouts, or wiring race-result fitness.
---

# Pro sessions (adapted)

Do not copy elite volume. Pick one session from `frontend/src/lib/pro-sessions.js` for the existing slot (`r-int`, `r-hills`, `r-tempo`, `r-strides`, sometimes `r-long`). Scale with `volumeFactor(level)`. Notes name the athlete and say this is the athlete’s current build, not the pro’s splits.

## Fitness selector

Train Like the Pros uses one recent race (mile / 5k / 10k / half / marathon) as the source of all paces. In opengym that is `athlete.fitness = { distanceKey, timeSec }` plus `raceTimes`. Typed VO2 still wins. Then race times, then recent workouts (`vo2-est.js`). Setting fitness stamps `weekDirty` so Modify week can rebuild the rest of the week.

New runners still do not get 4×4 or Kerr sets until they can jog 10 minutes.

## Slot map

- `r-int` — VO2 / mixed speed / fartlek (Kerr sets, Bakken 45/15, Fisher bookends, Kipyegon, Farah ladder, Mona)
- `r-hills` — Magness 8s default in the catalog; rotation adds Jakob hills, Keely, Mara, Lydiard. Shin/Achilles skip hills.
- `r-tempo` — Jakob 6-min, Kipchoge rolling, Almgren 6×6, Paula tempo, Haile 2k, Grete 1k
- `r-strides` — El Guerrouj 400s, Rudisha 400/300, Bekele pairs, Lagat 400s
- `r-long` — Benoit flow miles only for advanced/expert

Hamstring: existing swap `r-int` → `r-hills` still applies, then a hill pro session may stamp.

Athletes: `references/athletes.md`.
