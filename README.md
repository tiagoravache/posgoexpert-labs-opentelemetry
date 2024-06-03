# goexpert-labs-open-telemetry
Resposta do Lab de Open Telemetry da pós Go Expert.

Para execução dos serviços, rodar o seguinte comando na raiz do projeto: 

```bash
docker-compose up -d 
```

Para testar alguns cenários, pode-se executar o arquivo `test.http` que se encontra na raiz do projeto.

Para visualizar os traces, acessar o serviço do zipkin no endereço: `http://localhost:9411/` e apertar o botão `Run Query`.