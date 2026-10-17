## en

- **The criteria lights move with each check (D54).** While a turn runs, the
  lights in the grid window, the maximized one and the Crew's Pronto tab are
  those of its last `done.checked`, not of the last turn that ended, with
  "ciclo N" beside them and, once it stalls, "parado s de L" against the
  ceiling the event carries. When the turn ends, `turn.completed` takes over
  again. A check that broke a criterion and made the loop put work back says so
  in the flow ("⟲ Ciclo 2 voltou: e2e quebrou, desfeito src/cart.ts"); before,
  the lights stood still until the turn was over and a rollback was invisible.
  Screens 02 and 03 have no check under way and stay at 0.00%.

## pt-BR

- **As luzes dos critérios andam a cada conferência (D54).** Enquanto o turno
  roda, as luzes da janela da grade, da maximizada e da aba Pronto da Crew são
  as do último `done.checked` dele, não as do último turno que terminou, com
  "ciclo N" ao lado e, quando para, "parado s de L" contra o teto que o evento
  traz. Quando o turno termina, o `turn.completed` volta a mandar. Conferência
  que quebrou um critério e fez o laço devolver trabalho diz isso no fluxo
  ("⟲ Ciclo 2 voltou: e2e quebrou, desfeito src/cart.ts"); antes, as luzes
  ficavam paradas até o fim do turno e um desfazer era invisível. As telas 02 e
  03 não têm conferência em curso e ficam a 0,00%.
