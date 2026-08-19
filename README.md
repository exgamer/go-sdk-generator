# gosdk-generator

CLI-генератор файлов для сервисов на [`go-sdk-rest-template`](https://github.com/exgamer/go-sdk-rest-template).

Не является зависимостью генерируемого сервиса, в `go.mod` целевого проекта не попадает — используется через `go run` (см. [`INSTALL.md`](INSTALL.md)).

Все команды выполняются из корня целевого проекта (там, где `go.mod`). После генерации сами прогоняют `go get` + `go mod tidy` для нужных пакетов.

## Порядок генерации модуля

```bash
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 init --kernels=postgres,http,rabbit,redis

go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 domain add --fields=Name:string,Price:float64,CategoryID:uint catalog/product

go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 infra add postgres catalog/product
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 infra add redis catalog/product      # опционально
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 infra add http catalog/product       # опционально

go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 entrypoints add http admin catalog/product
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 entrypoints add rabbit catalog/product  # опционально

go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 bootstrap add catalog/product
```

## Команды

### `init`

Генерирует `main.go`, `internal/app/app.go`, `docs/docs.go` (плейсхолдер до `swag init`).

```bash
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 init
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 init --kernels=postgres,http,rabbit,redis
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 init --app-name "My Service"
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 init --force
```

- `--kernels` — comma-list: `postgres,http,rabbit,redis`. По умолчанию пусто.
- `--app-name` — заголовок в swagger-аннотации `main.go`. По умолчанию — последний сегмент module path.
- `--force` — перезаписать существующие `main.go`/`app.go`. `docs/docs.go` не перезаписывается никогда.

### `kernel add`

Добавляет kernel(ы) в существующий `internal/app/app.go`.

```bash
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 kernel add http
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 kernel add postgres,rabbit,redis
```

Уже зарегистрированные kernels пропускаются.

### `domain add`

Генерирует domain-слой: `entity.go`, `dto.go` (`Search`), `repository.go` (интерфейс `Repository`), `service.go`.

```bash
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 domain add --fields=Name:string,Price:float64,CategoryID:uint catalog/product
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 domain add --fields=Name:string --methods=getbyid,create catalog/tag
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 domain add --fields=Name:string --force handbook/city
```

- `--fields=Name:type,...` — поля сущности. Типы: `string,bool,int,int64,uint,uint64,float32,float64`. `ID`/`Status` добавляются автоматически, указывать нельзя.
- `--methods` — подмножество `paginated,getbyid,create,update,delete,activate,deactivate`. По умолчанию — все.
- `--force`
- `<domain>/<module>` — позиционный аргумент, идёт **после** флагов.

### `infra add postgres`

Генерирует `model.go`, `mapper.go`, `repository.go` (`PostgresRepository`, реализует `Repository`) для модуля, уже созданного `domain add`.

```bash
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 infra add postgres catalog/product
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 infra add postgres --force handbook/city
```

Поля и методы читаются из уже сгенерированного domain-слоя — повторно указывать не нужно.

### `infra add redis`

Генерирует `RedisRepository`: `Set{Entity}`/`Get{Entity}ById`, `Set{Entity}List`/`Get{Entity}List`, `Invalidate{Entity}Cache`.

```bash
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 infra add redis catalog/product
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 infra add redis --force handbook/city
```

### `infra add http`

Генерирует HTTP-клиент к внешнему сервису: `model.go`, `mapper.go`, `repository.go` (`HttpRepository.GetById`), `internal/configs/{module}_config.go` (хост через `mapstructure:"{MODULE}_SERVICE_HOST"`).

```bash
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 infra add http catalog/product
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 infra add http --force handbook/city
```

### `entrypoints add http`

Генерирует `request.go`, `response.go`, `mapper.go`, `handler.go` (со swagger-аннотациями), `routes.go`.

```bash
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 entrypoints add http admin catalog/product
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 entrypoints add http client --prefix=/api catalog/product
```

- `<admin|client>` — позиционный, определяет директорию (`internal/entrypoints/{admin|client}/http/...`).
- `--prefix` — префикс группы маршрутов. По умолчанию `/{последний сегмент module path}`.
- `--force`

Набор HTTP-методов (`Index/View/Create/Update/Delete`) зависит от того, какие `--methods` были у `domain add`.

### `entrypoints add rabbit`

Генерирует `{module}_consumer.go` (`Consumer.Consume`, JSON → `{module}Message`) и `consumer_registry.go` (`GetConsumers`).

```bash
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 entrypoints add rabbit catalog/product
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 entrypoints add rabbit --force handbook/city
```

### `bootstrap add`

Генерирует `internal/app/bootstrap/{domain}/{module}/` (`repositories_factory.go`, `services_factory.go`, `handlers_factory.go`, `module.go`, `consumers_factory.go` если есть rabbit) и регистрирует модуль в `internal/app/app.go`.

```bash
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 bootstrap add catalog/product
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 bootstrap add --force handbook/city
```

Ничего не указывается вручную — какие слои подключать, определяется по тому, что уже сгенерировано на диске для `<domain>/<module>`:

- **обязательно**: `infra add postgres` (реализация `domain.Repository`) и хотя бы один из `entrypoints add http admin`/`client`.
- **опционально**: `infra add redis`, `infra add http`, `entrypoints add rabbit` — подключаются, если найдены.

`--force` перезаписывает файлы фабрик/`module.go`; регистрация в `app.go` идемпотентна независимо от `--force`.

### `app-name`

Меняет `@title`/`@description` в swagger-аннотации `main.go`.

```bash
go run github.com/exgamer/go-sdk-generator/cmd/codegen@v0.0.3 app-name "My Service"
```

## Установка

См. [`INSTALL.md`](INSTALL.md).

## Разработка

```bash
go build ./...
go vet ./...
go test ./...
```
