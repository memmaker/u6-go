# internal/ending — END.EXE

Port of `~/Projects/u6-re/src-end` (END.C, STARS.C, STARFLD.ASM, GRLIB). Entry: `ending.Run(fsys, Args)`.

Args (GAME.EXE `execl("end.exe", param_str, y_str, m_str, d_str)`, seg_0903.c C_0903_0A1E):
param_str = video letter + `M`/`K` + sound letter (+ port) → `Mouse`, `Music`; y/m/d = game date
minus 4 July 161 → `Years`, `Months`, `Days`. CONFIG.U6 is skipped: param_str always overrides it.

On exit END.EXE stops the music, sets text mode 3 and returns to DOS; it execs nothing.

## Deviations
- VGA only; the video letter is ignored. No mouse pointer is drawn (only buttons count).
- Machine speed fixed at a fast machine (calcSpeed), loops paced one per host tick, as in intro.
- fix: years/months/days outside 0..29 (garbage in the original) are written as digits.
- fix: the star field's shift registers were addressed through CS in the original.
- GRLIB is UL.LIB's twin: text, fades and dissolve follow internal/intro's port (intro's are
  methods on its unexported state, so they are mirrored here rather than imported).
