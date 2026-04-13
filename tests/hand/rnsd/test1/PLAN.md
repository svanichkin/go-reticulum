# Build And Test Plan

Рабочая директория репозитория:

```bash
cd /Users/alien/Vault/Projects/Self/Golang/Reticulum/go-reticulum
```

Используем отдельную директорию конфига:

```bash
CFG="$PWD/tests/hand/rnsd/test1"
```

## 1. Сборка

Собрать `rnsd`:

```bash
mkdir -p ./bin
go build -o ./bin/rnsd ./cmd/rnsd
```

Опционально собрать `rnstatus` для локальной проверки:

```bash
go build -o ./bin/rnstatus ./cmd/rnstatus
```

## 2. Быстрая проверка CLI

```bash
./bin/rnsd --help
./bin/rnsd --version
./bin/rnsd --exampleconfig
```

Проверяем:

- help печатается без ошибки;
- версия выводится;
- example config печатается.

## 3. Старт в foreground

```bash
./bin/rnsd -config "$CFG" -vv
```

Проверяем в stdout:

- есть `Started rnsd version`;
- нет `connected to another shared local instance`;
- нет `Error starting rnsd`.

Остановить процесс: `Ctrl+C`.

## 4. Старт в service mode

Запуск:

```bash
./bin/rnsd -config "$CFG" -service
```

В отдельном терминале смотреть лог:

```bash
tail -f "$CFG/logfile"
```

Проверяем:

- файл `$CFG/logfile` создан;
- в нём есть `Started rnsd version`;
- нет критических ошибок старта.

## 5. Проверка shared instance через rnstatus

Пока `rnsd` работает:

```bash
./bin/rnstatus -config "$CFG" -a
```

или:

```bash
./bin/rnstatus -config "$CFG" -j
```

Проверяем:

- команда завершается с `exit 0`;
- нет `no shared RNS instance available`;
- вывод содержит статус инстанса;
- список интерфейсов может быть пустым, это нормально для этого конфига.

## 6. Проверка остановки

Остановить `rnsd`, затем снова выполнить:

```bash
./bin/rnstatus -config "$CFG" -a
```

Проверяем:

- shared instance больше недоступен;
- `rnstatus` сообщает, что локальный daemon не найден.

## 7. Что смотреть в логах

Полезные признаки:

- `Started rnsd version ...` — демон стартовал;
- `connected to another shared local instance` — поднят не свой daemon, а подключение к уже существующему;
- `unsupported interface type` — ошибка типа интерфейса в конфиге;
- `Could not locate external interface module` — конфиг ссылается на внешний Python interface, который Go-порт не поддерживает;
- `operation not permitted`, `permission denied`, ошибки bind/listen — проблемы окружения или прав.

## 8. Критерий прохождения

Тест считается успешным, если:

1. `rnsd` собирается.
2. `rnsd` стартует с этим конфигом.
3. `logfile` создаётся в service mode.
4. `rnstatus` видит shared instance при работающем `rnsd`.
5. После остановки `rnsd` shared instance исчезает.
