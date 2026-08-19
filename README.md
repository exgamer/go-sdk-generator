# gosdk-generator

CLI-генератор стартовых файлов для сервисов, построенных на [`go-sdk-rest-template`](https://git.mpinnovations.kz/mps/go-packages/go-sdk-rest-template) — шаблоне на базе `gosdk-core`, `gosdk-http-core`, `gosdk-postgres-core`, `gosdk-rabbit-core` и остальных пакетов из `git.mpinnovations.kz/mps/go-packages`.

Инструмент не является зависимостью генерируемого сервиса — используется только на этапе разработки (`go run .../cmd/codegen@version`), в `go.mod` целевого проекта не попадает.

## Команды

### `init`

Генерирует стартовые `main.go` и `internal/app/app.go` в пустом проекте (по образцу `go-sdk-rest-template`). Файлы, которые уже существуют, пропускаются — если не передан `--force`.

```bash
codegen init                       # без kernels — RegisterAndInitKernels() без аргументов
codegen init --kernels=http        # явно указанные kernels (postgres,http,rabbit)
codegen init --force               # перезаписать существующие main.go/app.go
codegen init --app-name "My Service"            # заголовок в swagger-аннотации main.go
```

Kernels по умолчанию не подключаются — нужно явно перечислить нужные через `--kernels`.

Команда всегда работает в текущем каталоге, путь модуля всегда читается из `go.mod` — предварительно выполните в нём `go mod init`.

После генерации (если файлы реально созданы, а не пропущены) `init` сам прогоняет `go get` для `gosdk-core` и пакетов выбранных kernels, затем `go mod tidy` — go.mod/go.sum сразу актуальны, руками ничего дописывать не нужно. Для этого шага нужен настроенный доступ к приватному GitLab (см. [`INSTALL.md`](INSTALL.md)).

### `kernel add`

Добавляет kernel(ы) в уже существующий `internal/app/app.go` — точечно дописывает импорт и аргумент в `RegisterAndInitKernels(...)`, не трогая остальной файл (ручные правки, регистрацию модулей и т.п. сохраняются). Уже зарегистрированные kernels пропускаются.

```bash
codegen kernel add http                 # один kernel
codegen kernel add postgres,rabbit      # несколько сразу
```

Как и `init`, сразу выполняет `go get` + `go mod tidy` для добавленных пакетов.

### `domain add`

Генерирует domain-слой нового модуля по образцу `city`/`product` из `go-sdk-rest-template`: `entity.go`, `dto.go` (`Search`), `repository.go` (интерфейс `Repository` с фиксированным CRUD-набором: `Paginated`/`GetById`/`Create`/`Update`/`Delete`/`Activate`/`Deactivate`), `service.go`.

```bash
codegen domain add --fields=Name:string,Price:float64,CategoryID:uint catalog/product
codegen domain add --fields=Name:string --force handbook/city
codegen domain add --fields=Name:string --methods=getbyid,create catalog/tag   # только часть CRUD
```

`<domain>/<module>` — позиционный аргумент, **после** флагов (так требует пакет `flag`). Поля задаются один раз через `--fields=Name:type,...` (допустимые типы: `string,bool,int,int64,uint,uint64,float32,float64`); `ID uint` и `Status int` добавляются в `entity.go` автоматически — их не нужно (и нельзя) указывать в `--fields`. В `Search` попадают `ID` + строковые поля (фильтр `ILIKE`) + поля типа uint/int с суффиксом `ID` в имени (точный фильтр) — как в `PostgresRepository.Paginated` из шаблона.

`--methods` управляет тем, какие методы попадут в `Repository`/`Service`: `paginated,getbyid,create,update,delete,activate,deactivate`. По умолчанию (флаг не передан) — все семь; при явном списке генерируются только они, а неиспользуемые импорты (`pagination` для `paginated`, `debug` для `getbyid`) исключаются автоматически.

Как и `init`/`kernel add`, сразу выполняет `go get` + `go mod tidy` — но только если это реально нужно: `gosdk-db-core` подтягивается, только если среди методов есть `Paginated`; иначе `go get` вообще не запускается.

### `infra add postgres`

Генерирует postgres-инфраструктуру для уже сгенерированного `domain add` модуля: `model.go` (GORM-модель), `mapper.go` (`modelToEntity`/`entityToModel`), `repository.go` (`PostgresRepository`, реализующий `Repository`).

```bash
codegen infra add postgres catalog/product
codegen infra add postgres --force handbook/city
```

Сущность/поля/методы **не указываются повторно** — команда читает `internal/domains/{domain}/{module}/entity.go` и `repository.go`, сгенерированные `domain add`, через `go/ast`: берёт оттуда список полей и ровно те методы `Repository`, что там объявлены (если `domain add` был вызван с `--methods=getbyid,create`, `infra add postgres` сгенерирует только `GetById`/`Create` — без лишних импортов). `<domain>/<module>` должен совпадать с тем, что передавался в `domain add`.

Имя таблицы — множественное число от `{module}` (`product` → `products`), колонки — snake_case от имён полей (`CategoryID` → `category_id`); полям поиска в `Search` при генерации `Paginated` соответствуют условия `{table}.{column} ILIKE ?` / `{table}.{column} = ?`, как в `PostgresRepository.Paginated` из шаблона.

Тоже сразу выполняет `go get` + `go mod tidy` — `gorm.io/gorm` всегда, `gosdk-db-core` только если среди методов есть `Paginated`.

### `infra add redis`

Генерирует кеширующий Redis-репозиторий по образцу `internal/infrastructure/redis/handbook/city/repository.go`: `RedisRepository` с методами `Set{Entity}`/`Get{Entity}ById` (кеш одной записи, `Helper.SetStruct`/`GetStruct`), `Set{Entity}List`/`Get{Entity}List` (кеш списка, `Helper.SetArray`/`GetArray`) и `Invalidate{Entity}Cache`. В отличие от `infra add postgres`, не реализует доменный `Repository` (в шаблоне он тоже не подключён к DI/фабрикам — это самостоятельный опциональный слой кэша), поэтому набор методов фиксирован и не зависит от `--methods`, указанных в `domain add`.

```bash
codegen infra add redis catalog/product
codegen infra add redis --force handbook/city
```

Как и `postgres`, сущность берётся из уже сгенерированного `domain add` (`entity.go` парсится через `go/ast`) — повторно указывать не нужно. Ключ кэша — `{domain}:{module}:{id}` (задокументированная в `SDK_REFERENCE.md` конвенция; в реальном `city` захардкожен `"service:city:"` — это специфика их окружения, не стали копировать буквально).

Сразу выполняет `go get` + `go mod tidy` (`gosdk-redis-core`, `github.com/redis/go-redis/v9`).

### `infra add http`

Генерирует HTTP-клиент к внешнему сервису по образцу `internal/infrastructure/http/handbook/city/`: `model.go` (модель ответа с `json`-тегами), `mapper.go` (`modelToEntity`), `repository.go` (`HttpRepository.GetById`, запрос через `gosdk-http-request-builder`). Как и `redis`, не реализует доменный `Repository` — самостоятельный клиент, набор методов фиксирован (сейчас только `GetById`, как в обоих референсах — реальном `city` и примере `product` из гайда).

```bash
codegen infra add http catalog/product
codegen infra add http --force handbook/city
```

URL и `product-service` в пути — заглушка (`http://{module}-service/api/v1/{module}s/%d`), помечена `TODO` в сгенерированном коде — нужно заменить на реальный адрес внешнего сервиса. Модель ответа по умолчанию включает все поля сущности (в реальном `city`/`product` внешний API возвращает только часть полей — но раз генератор не знает заранее, какие именно, он даёт полный набор, а лишнее убирается вручную).

Сразу выполняет `go get` + `go mod tidy` (`gosdk-http-core`, `gosdk-http-request-builder`).

Следующий слой в план — entrypoints (admin/http), дальше bootstrap.

## Установка

См. [`INSTALL.md`](INSTALL.md).

## Разработка

```bash
go build ./...
go vet ./...
go test ./...
```
