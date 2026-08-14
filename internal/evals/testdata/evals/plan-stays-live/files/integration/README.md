# Integração

Roda contra o **banco de staging**, alcançado pelo utilitário `dcode-testdb`.

`make integration` sobe a conexão e chama a suíte. Sem esse utilitário nada
aqui roda: a suíte existe para exercitar o caminho real de pagamento, e um
duplo de teste mediria outra coisa.
