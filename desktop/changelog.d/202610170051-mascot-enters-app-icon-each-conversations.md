## en

- **The mascot enters the app: the icon and each conversation's state.** The
  owner chose the mascot of `refs/design/brand/mascote.webp` as dcode's face.
  It is drawn by hand as vectors, never cropped from the sheet: the app icon
  (`assets/icon/icon.svg`, front pose in Laranja on a graphite tile) and
  `npm run gen:icons`, which rasterises it in Playwright's Chromium and packs
  the `.icns`, `.ico` and `.png` with no platform tool; `forge.config.ts` gives
  it to the packager. Where the window showed a conversation's state with a
  character — a grid window, the maximized one, Crew's thread and session
  rows — it now shows the mascot's head in the same tone, and the eye and a
  mark say the state again in shape: Planejando…, Codando…, Verificando…,
  Deu certo!, Bug encontrado…, Esperando você, Hum…, Parado. The pose is read
  from what the window already folds (the conversation's state, the phase of
  the call running, the last seal), never from a field the protocol lacks.
  Empty screens show the whole figure; notices carry the "Bug encontrado…"
  head. Reduced motion still stops the breathing. References 02 and 03 move
  by 0.07% and 0.04%, inside the limit; the palette change stacked on this
  one recaptures them once (D49).

## pt-BR

- **O mascote entra no app: o ícone e o estado de cada conversa.** O dono
  escolheu o mascote de `refs/design/brand/mascote.webp` como a cara do dcode.
  Ele é desenhado à mão em vetor, nunca recortado da folha: o ícone do app
  (`assets/icon/icon.svg`, de frente, em Laranja, num ladrilho grafite) e o
  `npm run gen:icons`, que o rasteriza no Chromium do Playwright e monta o
  `.icns`, o `.ico` e o `.png` sem ferramenta de plataforma; o
  `forge.config.ts` o entrega ao empacotador. Onde a janela mostrava o estado
  de uma conversa com um caractere — janela da grade, a maximizada, a thread e
  as linhas de sessão do Crew —, agora mostra a cabeça do mascote no mesmo tom,
  e o olho e uma marca dizem o estado de novo em forma: Planejando…, Codando…,
  Verificando…, Deu certo!, Bug encontrado…, Esperando você, Hum…, Parado. A
  pose sai do que a janela já dobra (o estado da conversa, a fase da chamada
  que roda, o último selo), nunca de um campo que o protocolo não tem. Telas
  vazias mostram o mascote inteiro; avisos levam a cabeça de "Bug
  encontrado…". Menos movimento continua parando a respiração. As referências
  02 e 03 mudam 0,07% e 0,04%, dentro do limite; a troca de paleta empilhada
  sobre esta as recaptura uma vez só (D49).
