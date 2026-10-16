import { FileText } from 'lucide-react';
import { useMemo, useState } from 'react';
import type { ImageAttachment } from '../shared/api';
import { imageSource, sizeOf } from '../state/attachments';
import type { Activity } from '../state/flow';
import type { SessionView, ToolCall } from '../state/session';
import { toolDetail } from '../state/tooldetail';
import { chipsOf, turnTouched, type TurnFile } from '../state/turn';
import { ActivityLine } from './Flow';
import { ToolDetail } from './ToolDetail';

type Chip = { kind: 'image'; image: ImageAttachment; key: string } | { kind: 'file'; file: TurnFile; key: string };

function baseName(path: string): string {
  return path.slice(path.lastIndexOf('/') + 1) || path;
}

function chipTitle(c: Chip): string {
  if (c.kind === 'image') return `${c.image.name} · ${sizeOf(c.image.bytes)}`;
  const f = c.file;
  return `${f.path} · +${f.added} −${f.removed}${f.failed ? ' · a última escrita falhou' : ''}`;
}

/**
 * The running turn's activity line with what the turn touched on its right
 * (D58): the pictures sent with it as thumbnails and the files it wrote as
 * chips, past the room a "+N". A file opens onto its diff under the line, the
 * same detail its tool line opens onto (D45).
 */
export function TurnActivity({
  session,
  activity,
  now,
  tick,
  compact,
  imagesAt,
}: {
  session: SessionView;
  activity: Activity;
  now: number;
  tick: number;
  compact: boolean;
  imagesAt: ReadonlyMap<number, ImageAttachment[]>;
}) {
  const touched = useMemo(() => turnTouched(session, imagesAt), [session, imagesAt]);
  const calls = useMemo(() => {
    const m = new Map<string, ToolCall>();
    for (const e of session.entries) if (e.kind === 'tool') m.set(e.call.id, e.call);
    return m;
  }, [session.entries]);
  const [open, setOpen] = useState<string | null>(null);
  const [all, setAll] = useState(false);
  const chips: Chip[] = [
    ...touched.images.map((image, i): Chip => ({ kind: 'image', image, key: `i-${i}-${image.name}` })),
    ...touched.files.map((file): Chip => ({ kind: 'file', file, key: `f-${file.path}` })),
  ];
  const room = compact ? 3 : 5;
  const { shown, more } = all ? { shown: chips, more: 0 } : chipsOf(chips, room);
  const openFile = open ? touched.files.find((f) => f.path === open) : undefined;
  const openCall = openFile ? calls.get(openFile.callId) : undefined;
  const detail = openCall ? toolDetail(openCall) : null;

  const cluster =
    chips.length === 0 ? undefined : (
      <span className="turn-touched" aria-label="O que este turno tocou">
        {shown.map((c) => {
          if (c.kind === 'image') {
            return <img key={c.key} className="turn-thumb" src={imageSource(c.image)} alt={c.image.name} title={chipTitle(c)} />;
          }
          const call = calls.get(c.file.callId);
          const opens = call ? toolDetail(call) !== null : false;
          const isOpen = open === c.file.path;
          return (
            <button
              key={c.key}
              type="button"
              className={`turn-chip${isOpen ? ' open' : ''}${c.file.failed ? ' failed' : ''}`}
              title={opens ? chipTitle(c) : `${chipTitle(c)} · ainda sem diff`}
              aria-expanded={opens ? isOpen : undefined}
              disabled={!opens}
              onClick={() => setOpen(isOpen ? null : c.file.path)}
            >
              <FileText size={11} aria-hidden="true" />
              <span className="turn-chip-name">{baseName(c.file.path)}</span>
            </button>
          );
        })}
        {more > 0 && (
          <button
            type="button"
            className="turn-chip more"
            title={chips.slice(shown.length).map(chipTitle).join('\n')}
            aria-label={`Mostrar mais ${more}`}
            onClick={() => setAll(true)}
          >
            +{more}
          </button>
        )}
        {all && chips.length > room && (
          <button type="button" className="turn-chip more" aria-label="Mostrar menos" title="Mostrar menos" onClick={() => setAll(false)}>
            −
          </button>
        )}
      </span>
    );

  return (
    <>
      {session.turn && <ActivityLine activity={activity} startedAt={session.turn.startedAt} now={now} tick={tick} touched={cluster} />}
      {openFile && detail && <ToolDetail id={`turn-diff-${openFile.callId}`} detail={detail} label={`O que mudou em ${openFile.path}`} />}
    </>
  );
}
