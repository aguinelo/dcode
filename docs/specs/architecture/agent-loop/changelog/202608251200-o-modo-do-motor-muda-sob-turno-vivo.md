# O modo do motor muda sob turno vivo

`Engine.SetMode` troca o par (sandbox, política) em tempo de execução e
`Engine.Mode` o lê. O turno em andamento **não** é interrompido: a próxima
chamada de ferramenta observa o par novo, e a que já está em voo termina sob o
que valia quando começou. Interromper transformaria um ajuste de autonomia em
cancelamento de trabalho.

É justamente por não interromper que a trava importa, e é aí que a primeira
versão errou. `SetMode` escreve da goroutine do handler HTTP enquanto o turno
roda. A avaliação de uma chamada foi posta sob o mutex; a montagem de um filho
delegado, não — e ela lê os mesmos dois campos, na goroutine do turno, que é
exatamente quem está vivo quando a troca chega. Sob `-race` é corrida, e
nenhum teste a via porque nenhum teste delegava enquanto trocava.

Todo leitor passa agora por `Engine.Mode`, que lê o par sob a mesma trava que o
escreve. E `childConfig` lê o par **uma vez** para a montagem inteira: um filho
montado a partir de duas leituras diferentes é um filho sob um par que ninguém
escolheu.

O invariante que fica é sobre o que se pergunta ao testar. O teste original
chamava `SetMode(full)` e assertava `cfg.Mode == full` — a forma que não pode
falhar, porque interroga o campo que acabou de escrever. Ele passaria com todos
os leitores segurando cópia velha, que é precisamente o defeito que existia.
O que faz `SetMode` ser verdade é o **veredito** da próxima chamada mudar, e a
delegação enxergar a mudança.
