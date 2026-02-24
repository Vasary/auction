# E2E Runner (UI)

## 1. Назначение

`E2E Runner` на странице `/ui/e2e` выполняет функциональный сценарий поверх уже запущенного backend:

1. Создает несколько аукционов через HTTP.
2. Регистрирует участников.
3. Подключает несколько WebSocket-клиентов на каждый аукцион.
4. Отправляет ставки.
5. Дожидается завершения аукциона.
6. Валидирует финальное состояние (`winnerId`, `currentPrice`, история ставок, статус).

Главная цель: проверить бизнес-логику и консистентность состояния, а не только доступность API.

## 2. Где находится код

- UI-страница: `ui/src/pages/E2ERunnerPage.tsx`
- Основная логика сценария: `ui/src/components/E2ERunnerPanel.tsx`

## 3. Как симулируются "много клиентов"

Для каждого аукциона создается `clientsPerAuction` виртуальных участников.

Для каждого участника:

1. Генерируется `companyId` и `personId`.
2. Компания регистрируется через `POST /auctions/{tenderId}/participate`.
3. Открывается отдельный `WebSocket` на `/ws/{tenderId}`.

Именно отдельный сокет на участника и есть симуляция параллельных пользователей в браузере.

## 4. Тайминг и UTC

Все даты отправляются в backend строго в UTC (`toISOString()`):

- `startAt`
- `endAt`

В отчете также фиксируются UTC-метки, чтобы не было расхождений из-за локального timezone браузера.

## 5. Что делает один сценарий по аукциону

Для каждого аукциона:

1. Вычисляются `startAt/endAt`:
   - часть аукционов стартует "сейчас",
   - часть со сдвигом `startDelaySec`,
   - длительность берется из `shortDurationSec`/`longDurationSec`.
2. Отправляется `POST /auctions`.
3. Регистрируются участники (`participate`).
4. Подключаются WS-клиенты.
5. Скрипт дожидается старта аукциона.
6. Отправляются ставки раундами (`bidRounds`) с паузой `bidIntervalMs`.
7. Во время WS-обмена собирается ожидаемый результат:
   - `expectedWinnerId`
   - `expectedCurrentPrice`
   на основе `bid_result` с `accepted=true`.
8. Периодически опрашивается `GET /auctions/{tenderId}` до статуса `Finished` (или timeout).
9. Загружается история ставок `GET /auctions/{tenderId}/bids`.
10. Проводится валидация, формируется `passed/reasons`.

## 6. Что считается валидным (passed)

Аукцион считается `passed`, только если нет ни одной причины ошибки.

Проверки:

1. Аукцион дошел до `status=Finished`.
2. Подключился хотя бы 1 WS-клиент (`wsConnected > 0`).
3. Подключилось ровно ожидаемое количество клиентов (`wsConnected == clientsPerAuction`).
4. Была хотя бы одна попытка ставки (`bidsAttempted > 0`).
5. Был хотя бы один `bid_result` с `accepted=true` (`bidsAccepted > 0`).
6. В backend сохранилась хотя бы одна ставка (`bidsPersisted > 0`).
7. История ставок строго убывает по цене.
8. Разница между соседними ставками кратна `step`.
9. Если есть `accepted`-ставки, вычисленный `expectedWinnerId/expectedCurrentPrice` должен существовать.
10. `expectedWinnerId` совпадает с компанией последней ставки из `/bids`.
11. `expectedCurrentPrice` совпадает с суммой последней ставки из `/bids`.
12. `winnerId` из `GET /auctions/{id}` совпадает с компанией последней ставки.
13. `currentPrice` из `GET /auctions/{id}` совпадает с суммой последней ставки.
14. `winnerId/currentPrice` из `GET /auctions/{id}` совпадают с `expectedWinnerId/expectedCurrentPrice`.

## 7. Почему раньше могли быть "ложные успехи"

Если критерии мягкие, сценарий мог отмечаться успешным даже при:

- `wsConnected = 0`
- `bidsPersisted = 0`
- отсутствии принятых ставок

Теперь эти случаи дают `failed` с явной причиной в `reasons`.

## 8. Поля результата

По каждому аукциону выводятся:

- `tenderId`
- `startAt`, `endAt` (UTC)
- `wsConnected`
- `expectedWinnerId`
- `expectedCurrentPrice`
- `bidsAttempted`
- `bidsAccepted`
- `bidsPersisted`
- `winnerId`
- `currentPrice`
- `status`
- `passed`
- `reasons[]`

## 9. Ограничения сценария

1. Это браузерный функциональный тест, а не нагрузочный инструмент уровня k6/JMeter.
2. Большие значения `auctionsCount * clientsPerAuction` могут упираться в лимиты браузера/машины.
3. Тест выполняется с учетом реального времени, поэтому чувствителен к сетевым задержкам и нагрузке на backend/DB.
4. При слишком маленьком `bidIntervalMs` ставки могут отклоняться из-за rate-limit на backend.

## 10. Практические рекомендации

Базовый стабильный прогон:

- `auctionsCount = 4`
- `clientsPerAuction = 3..4`
- `startDelaySec = 30..60`
- `shortDurationSec = 60`
- `longDurationSec = 120`
- `bidRounds = 2..3`
- `bidIntervalMs >= 550` (если backend rate limit около 500ms на участника)

Если хотите тестировать именно отказоустойчивость и предельные режимы, постепенно увеличивайте только один параметр за раз.
