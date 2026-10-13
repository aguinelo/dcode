// The keys the help lists (D48), grouped as the window answers them. The window's
// own are useWindowKeys' (D31, D36), the field's are Field's, and an approval's
// are approval.ts' (D22, D39): a key added there is a line to add here.

export interface HelpKey {
  key: string;
  what: string;
}

export interface HelpGroup {
  title: string;
  keys: HelpKey[];
}

export const HELP_KEYS: readonly HelpGroup[] = [
  {
    title: 'Janela',
    keys: [
      { key: '⌘K', what: 'procura uma conversa' },
      { key: '⌘N', what: 'abre uma conversa nova' },
      { key: '⌘0', what: 'volta para a grade' },
      { key: '⌘1–9', what: 'põe o cursor na janela n' },
      { key: '⌘↵', what: 'maximiza a janela em foco, e a traz de volta' },
      { key: '⇧⇥', what: 'troca o modo da conversa em foco' },
      { key: '← → ↑ ↓', what: 'fora de um campo: muda a janela em foco' },
      { key: 'esc', what: 'sai do campo; maximizada, volta para a grade' },
    ],
  },
  {
    title: 'Campo',
    keys: [
      { key: '↵', what: 'envia; fora do campo, põe o cursor na janela em foco' },
      { key: '⇧↵', what: 'quebra a linha' },
      { key: '↑ ↓', what: 'no começo ou no fim do campo, volta ao que já se pediu nesta conversa' },
      { key: '⌥↵', what: 'durante um turno, guarda a mensagem para depois dele' },
      { key: '⌃X', what: 'tira da fila a última mensagem guardada' },
      { key: '/', what: 'no começo do campo, abre os comandos: ↑ ↓ escolhem, ↵ ou ⇥ completam, esc fecha' },
      { key: '?', what: 'com o campo vazio, abre esta ajuda' },
      { key: '!', what: 'no começo, roda o resto como comando na sessão' },
    ],
  },
  {
    title: 'Aprovação',
    keys: [
      { key: '1–4', what: 'a resposta de mesmo número no cartão' },
      { key: '↵ esc', what: 'nega' },
    ],
  },
];
