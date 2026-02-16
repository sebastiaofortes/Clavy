Iniciar o servidor

./pdf-to-html -serve 8080

# Exemplos de curls:

# Converter página 1
```

curl "http://localhost:8080/convert?pdf=samples/teste.pdf&page=1" -o resultado.html

```

# Converter todas as páginas com zoom 2x
```
curl "http://localhost:8080/convert?pdf=samples/teste.pdf&zoom=2.0" -o resultado.html
```

# Abrir direto no navegador
```
open "http://localhost:8080/convert?pdf=samples/teste.pdf&page=1"
```