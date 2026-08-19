## Установка

Пакет размещён на приватном GitLab — перед установкой нужно настроить доступ (см. также `INSTALL.md` в `gosdk-core`, если уже настраивали раньше — шаги идентичны).

Тот же доступ нужен и целевому проекту: после генерации файлов `codegen init` (и `codegen kernel add`) сам выполняет `go get` + `go mod tidy` для `gosdk-core` и пакетов выбранных kernels (`gosdk-http-core`, `gosdk-postgres-core`, `gosdk-rabbit-core`) — без шагов 1–4 ниже эти команды упадут на авторизации.

**1. Personal Access Token**

Создать на `https://git.mpinnovations.kz` в **Settings → Access Tokens** с правом `read_repository`.

**2. ~/.gitconfig** — для git-операций (все ОС)

```bash
git config --global url."https://oauth2:YOUR_TOKEN@git.mpinnovations.kz/".insteadOf "https://git.mpinnovations.kz/"
```

**3. Файл netrc** — для HTTP-запросов Go

<details>
<summary>macOS / Linux</summary>

```bash
echo "machine git.mpinnovations.kz login oauth2 password YOUR_TOKEN" >> ~/.netrc
chmod 600 ~/.netrc
```

</details>

<details>
<summary>Windows (PowerShell)</summary>

```powershell
Add-Content "$env:USERPROFILE\_netrc" "machine git.mpinnovations.kz login oauth2 password YOUR_TOKEN"
```

</details>

**4. GOPRIVATE** — чтобы Go не обращался к публичному прокси (все ОС)

```bash
go env -w GOPRIVATE=git.mpinnovations.kz/*
```

**5. Запуск (без установки, привязка к версии)**

```bash
go run git.mpinnovations.kz/mps/go-packages/gosdk-generator/cmd/codegen@v0.1.0 init
```

Не добавляет `gosdk-generator` в зависимости целевого сервиса — модуль скачивается только в module cache вызывающей машины.
