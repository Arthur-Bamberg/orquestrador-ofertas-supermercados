# Backoffice

Superfície de administração do catálogo e do Canal. Quem se identifica aqui é o Operador — não o Contato.

## Language

**Operador**:
Pessoa do backoffice, distinta do Contato do Canal, identificada por um nome único e uma senha. Há vários; todos equivalentes. Não é entidade de cadastro na UI: a linha vive no banco.
_Avoid_: usuário, user, admin, Contato, cliente

**Sair**:
Operação do Operador que encerra a identificação no backoffice. Não desfaz Pareamento nem altera o Canal.
_Avoid_: logout (como sinónimo de Desparear), desconectar
