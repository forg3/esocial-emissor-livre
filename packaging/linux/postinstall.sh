#!/bin/sh
# Aviso pós-instalação do Validador eSocial.
# O binário é auto-contido: cria o banco SQLite e abre o navegador na primeira execução.
if [ -t 1 ]; then
  echo "Validador eSocial instalado em /usr/bin/validador-esocial"
  echo "Execute: validador-esocial            # abre http://localhost:8000"
  echo "Ou:      validador-esocial -senha SUA_SENHA -host 0.0.0.0"
  echo "A senha local de acesso é exibida no terminal na primeira execução."
fi
exit 0
