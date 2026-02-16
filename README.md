Esse sistema possui;

1 - Banco de dados baseado em arquivo de texto

2 - Feature que adiciona autometicamente header a pagina do livro, permitindo sua tradução.


Manual de uso:

Iniciar o servidor

./pdf-to-html -serve 8080

Abra no navegador: http://localhost:

Selecione o livro que deseja ler ou siga o passo seguinte para realizar o upload de um novo livro.

Para realizar o upload de um novo livro, na página principal do sistema, clique em "+ Upload PDF", selecione o arquivo PDF, selecione o idioma de origem do documento, clique no botão "Enviar PDF".

O sistema armazena em localStorage a última página lida pelo usuário, para ajudar o usuário a não esquecer onde parou de ler. 

Ao abrir uma página do PDF, o navegador Chrome vai detectar e sugerir uma tradução da página. Você pode selecionar a opção para traduzir todas as páginas visualizadas.