## en

- **A configuration value that does not parse is said, not defaulted.**
  `allow_network = "talvez"` used to open a session with the network on — the
  default — and `DCODE_MAX_ITERATIONS=12abc` read as 12, the prefix a format
  scan stops at; neither said a word. Numbers are now read from the whole
  value, and a boolean or number that does not convert refuses the
  configuration naming the key, the value and the layer (file or environment
  variable): the CLI exits with it and the daemon refuses the session, as it
  already did for a project file it cannot read. The boolean vocabulary is
  unchanged.

## pt-BR

- **Valor de configuração que não converte é dito, não trocado pelo padrão.**
  `allow_network = "talvez"` abria a sessão com a rede ligada — o padrão — e
  `DCODE_MAX_ITERATIONS=12abc` valia 12, o prefixo em que uma leitura com
  formato para; nenhum dos dois dizia nada. Número passa a ser lido do valor
  inteiro, e booleano ou número que não converte recusa a configuração
  nomeando chave, valor e camada (arquivo ou variável de ambiente): a CLI sai
  com a mensagem e o daemon recusa a sessão, como já fazia com arquivo de
  projeto que não se lê. O vocabulário do booleano não muda.
