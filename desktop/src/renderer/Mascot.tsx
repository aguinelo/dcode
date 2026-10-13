import type { ReactNode } from 'react';
import { MASCOT_WORDS, type MascotState } from '../state/mascot';

// The mascot (D49), drawn as vectors: the head block with its one big eye
// for a state at text size, and the whole figure for an empty screen. The
// head takes the tone of the state (currentColor), so the colour still says
// what the glyph said before; the eye and the mark beside it say it again in
// shape, for whoever does not read the colour. The eye keeps the mascot's
// own colours in both themes.

const INK = '#1A1A1A';
const EYE = '#F5F5F2';

/** The eye, looking where (dx, dy) points. */
function Eye({ dx = 0, dy = 0 }: { dx?: number; dy?: number }) {
  return (
    <>
      <circle cx="6.5" cy="9.2" r="3.4" fill={INK} />
      <circle cx="6.5" cy="9.2" r="2.4" fill={EYE} />
      <circle cx={6.5 + dx} cy={9.2 + dy} r="1.15" fill={INK} />
    </>
  );
}

const stroke = { stroke: 'currentColor', strokeWidth: 1.4, strokeLinecap: 'round' as const, fill: 'none' };

// The eye and the mark that make each pose, on a 16-unit square whose head
// fills the lower left and whose mark sits in the upper right.
const POSES: Record<MascotState, ReactNode> = {
  // Thinking: looking up at two rising dots.
  planejando: (
    <>
      <Eye dx={-0.6} dy={-0.7} />
      <circle cx="13.2" cy="3.9" r="0.95" fill="currentColor" />
      <circle cx="15" cy="1.4" r="1.05" fill="currentColor" />
    </>
  ),
  // At work: the sheet's two motion marks.
  codando: (
    <>
      <Eye />
      <path d="M12.6 3.4 L14.1 1.1 M13.7 5.4 L15.6 4.5" {...stroke} />
    </>
  ),
  // Checking: the eye turned to a lens.
  verificando: (
    <>
      <Eye dx={0.8} />
      <circle cx="14" cy="2.6" r="1.6" {...stroke} strokeWidth={1.1} />
      <path d="M12.8 3.8 L12 4.7" {...stroke} strokeWidth={1.1} />
    </>
  ),
  // Done: a check in the eye, and the marks.
  'deu-certo': (
    <>
      <circle cx="6.5" cy="9.2" r="3.4" fill={INK} />
      <path d="M4.9 9.3 L6.1 10.6 L8.3 8" stroke={EYE} strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round" fill="none" />
      <path d="M12.6 3.4 L14.1 1.1 M13.7 5.4 L15.6 4.5" {...stroke} />
    </>
  ),
  // A bug: the eye crossed out, and a dizzy squiggle.
  bug: (
    <>
      <circle cx="6.5" cy="9.2" r="3.4" fill={INK} />
      <path d="M5.2 7.9 L7.8 10.5 M7.8 7.9 L5.2 10.5" stroke={EYE} strokeWidth="1.2" strokeLinecap="round" />
      <path d="M12.3 4.8 L13.5 2.9 L14.4 4.4 L15.6 2.2" {...stroke} strokeWidth={1.2} strokeLinejoin="round" />
    </>
  ),
  // Waiting for you: eyes on you, and a mark that asks.
  esperando: (
    <>
      <Eye />
      <path d="M14.4 0.9 L14.4 3.6" {...stroke} strokeWidth={1.6} />
      <circle cx="14.4" cy="5.6" r="0.85" fill="currentColor" />
    </>
  ),
  // Not sure: the lid half down.
  hum: (
    <>
      <Eye dx={0.5} dy={0.4} />
      <path d="M2.9 9.2 a3.6 3.6 0 0 1 7.2 0 z" fill="currentColor" />
      <path d="M2.9 9.2 L10.1 9.2" stroke={INK} strokeWidth="0.9" />
    </>
  ),
  parado: <Eye />,
};

/**
 * The state of a conversation at text size, labelled with its word. `live`
 * makes it breathe (dc-breath), which reduced motion turns off; the pose
 * never depends on the motion.
 */
export function MascotGlyph({ state, live = false, size = 14, className = '' }: { state: MascotState; live?: boolean; size?: number; className?: string }) {
  const word = MASCOT_WORDS[state];
  return (
    <svg
      className={`mascot-glyph ${className}${live ? ' dc-breath' : ''}`}
      data-mascot={state}
      viewBox="0 0 16 16"
      width={size}
      height={size}
      role="img"
      aria-label={word}
    >
      <title>{word}</title>
      <rect x="0.5" y="4" width="12" height="10.6" rx="2.6" fill="currentColor" />
      {POSES[state]}
    </svg>
  );
}

// The arms and feet are the mascot's black; a hairline in the window's
// strongest line keeps them on a dark ground, where black alone disappears.
const LIMB = { stroke: 'var(--dc-line-strong)', strokeWidth: 1.5 };

/**
 * The whole figure, front pose, in Laranja: the one assets/icon/icon.svg puts
 * on its tile, on the same 130-unit grid. For a screen with nothing on it.
 */
export function MascotFigure({ size = 64 }: { size?: number }) {
  return (
    <svg className="mascot-figure" viewBox="0 8 130 134" width={size} height={(size * 134) / 130} aria-hidden>
      <path d="M101 24 L111 11 M106 35 L120 29" stroke="#FF6A00" strokeWidth="5" strokeLinecap="round" fill="none" />
      <rect x="12" y="74" width="20" height="30" rx="9" fill="#0A0A0A" style={LIMB} transform="rotate(12 22 89)" />
      <rect x="98" y="74" width="20" height="30" rx="9" fill="#0A0A0A" style={LIMB} transform="rotate(-12 108 89)" />
      <rect x="37" y="126" width="22" height="13" rx="6" fill="#0A0A0A" style={LIMB} />
      <rect x="71" y="126" width="22" height="13" rx="6" fill="#0A0A0A" style={LIMB} />
      <rect x="31" y="102" width="68" height="26" rx="9" fill="#FF6A00" />
      <rect x="37" y="105" width="56" height="4" rx="2" fill="#FB923C" />
      <rect x="27" y="72" width="76" height="28" rx="9" fill="#FF6A00" />
      <rect x="33" y="75" width="64" height="4" rx="2" fill="#FB923C" />
      <rect x="12" y="14" width="106" height="56" rx="15" fill="#FF6A00" />
      <rect x="22" y="18" width="86" height="5" rx="2.5" fill="#FB923C" />
      <circle cx="65" cy="42" r="21" fill={INK} />
      <circle cx="65" cy="42" r="14" fill={EYE} />
      <circle cx="65" cy="42" r="6.5" fill={INK} />
      <circle cx="61.5" cy="38.5" r="2.4" fill={EYE} />
    </svg>
  );
}
