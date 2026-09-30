// What the daemon sends is checked against the shape the generated types
// promise before anything reads it: the event stream is a system boundary, and
// a payload that does not match is a fact to show, not a value to guess from.
//
// A shape is written by hand, but it cannot drift from the generated type it
// describes: `Shape<T>` requires one entry per field of T, and each entry's
// optionality must agree with T's. A field added to, removed from or made
// optional in internal/protocol fails the typecheck here until the shape says
// the same.

export type Kind = 'string' | 'number' | 'boolean' | 'object' | 'array' | 'any';

export interface Field<Optional extends boolean = boolean> {
  readonly kind: Kind;
  readonly optional: Optional;
  /** For 'object', the nested shape; for 'array', each item's shape or kind. */
  readonly of?: AnyShape | Kind;
}

export type AnyShape = Readonly<Record<string, Field>>;

type IsOptional<V> = undefined extends V ? true : false;

export type Shape<T> = { readonly [K in keyof T]-?: Field<IsOptional<T[K]>> };

export function req(kind: Kind, of?: AnyShape | Kind): Field<false> {
  return of === undefined ? { kind, optional: false } : { kind, optional: false, of };
}

export function opt(kind: Kind, of?: AnyShape | Kind): Field<true> {
  return of === undefined ? { kind, optional: true } : { kind, optional: true, of };
}

export function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v);
}

function kindOf(v: unknown): Kind | 'null' | 'other' {
  if (v === null) return 'null';
  if (Array.isArray(v)) return 'array';
  switch (typeof v) {
    case 'string':
      return 'string';
    case 'number':
      return Number.isFinite(v) ? 'number' : 'other';
    case 'boolean':
      return 'boolean';
    case 'object':
      return 'object';
    default:
      return 'other';
  }
}

function checkValue(field: Field, v: unknown, path: string): string | null {
  if (field.kind === 'any') return null;
  const got = kindOf(v);
  // Go encodes a nil slice that is not omitempty as null. It is an empty list,
  // not a malformed one.
  if (got === 'null' && field.kind === 'array') return null;
  if (got !== field.kind) return `${path} deveria ser ${field.kind} e é ${got}`;
  if (field.kind === 'object' && field.of !== undefined && typeof field.of !== 'string') {
    return check(field.of, v, path);
  }
  if (field.kind === 'array' && field.of !== undefined && Array.isArray(v)) {
    for (let i = 0; i < v.length; i++) {
      const item = v[i];
      const why =
        typeof field.of === 'string'
          ? checkValue({ kind: field.of, optional: false }, item, `${path}[${i}]`)
          : check(field.of, item, `${path}[${i}]`);
      if (why) return why;
    }
  }
  return null;
}

/**
 * Returns why value does not match shape, or null when it does. Fields the
 * shape does not name are ignored: the core may add optional fields, and an
 * older desktop must keep reading what it knows.
 */
export function check(shape: AnyShape, value: unknown, path = 'payload'): string | null {
  if (!isRecord(value)) return `${path} não é um objeto`;
  for (const [name, field] of Object.entries(shape)) {
    const v = value[name];
    // An optional field that arrives as null says the same as one left out.
    if (v === undefined || (v === null && field.optional)) {
      if (field.optional) continue;
      return `${path}.${name} está ausente`;
    }
    const why = checkValue(field, v, `${path}.${name}`);
    if (why) return why;
  }
  return null;
}
