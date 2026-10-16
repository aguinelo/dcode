// What the turn under way touched, for its activity line (D58).

import type { ImageAttachment } from '../shared/api';
import type { TouchedFile } from './context';
import type { SessionView } from './session';

export interface TurnFile extends TouchedFile {
  callId: string;
}

export interface TurnTouched {
  files: TurnFile[];
  images: ImageAttachment[];
}

export function turnTouched(_view: Pick<SessionView, 'turn' | 'entries'>, _imagesAt: ReadonlyMap<number, ImageAttachment[]>): TurnTouched {
  return { files: [], images: [] };
}

export function chipsOf<T>(items: readonly T[], _room: number): { shown: T[]; more: number } {
  return { shown: [...items], more: 0 };
}
