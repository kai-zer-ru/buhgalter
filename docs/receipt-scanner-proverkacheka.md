# Интеграция proverkacheka (сканер чеков)

Dev- и ops-документ к [receipt-scanner.md](receipt-scanner.md). Продуктовый поток и прикрепление к операциям — там же.

## Роль в архитектуре

| Компонент | Назначение |
|-----------|------------|
| Клиент (web / Android) | Сканирует QR 54-ФЗ, шлёт строку `qrraw` на `POST /api/v1/receipts/parse` или через scan-job после decode на сервере |
| Сервер | Хранит токен инстанса, вызывает `POST https://proverkacheka.com/api/v1/check/get`, нормализует ответ в `ParsedReceipt` |
| Админка | `PUT /api/v1/admin/settings/receipt-api-token` (write-only); `GET /api/v1/admin/settings` — `receipt_api_token_set: true/false` |

Эффективное включение модуля: `features.receipt_scanner === true` **и** `receipt_api_token_set === true`.

## Получение токена (для администратора инстанса)

См. **[README](../README.md#сканер-чеков-токен-proverkacheka)** — там пошагово: регистрация на proverkacheka, curl-проверка, поле в админке buhgalter.

## Проверка токена

**Снаружи buhgalter:**

```bash
curl -sS -X POST 'https://proverkacheka.com/api/v1/check/get' \
  -H 'Content-Type: application/json' \
  -d '{"token":"TOKEN","qrraw":"t=…&s=…&fn=…&i=…&fp=…&n=1"}'
```

**В buhgalter (после реализации):** кнопка «Проверить» в админке (тестовый QR из доки провайдера) и/или `go run ./cmd/receipt-services-probe` с флагом токена.

## Сервер (ориентир реализации)

- Колонка в `system_settings` (имя на усмотрение миграции, например `receipt_proverkacheka_token`), шифрование как у `notification_secret_key`.
- Пакет `internal/receipt/proverkacheka` — HTTP-клиент, маппинг JSON → `fns.ParsedReceipt`.
- `Fetcher` / цепочка lookup: при настроенном токене — только proverkacheka; без токена — `503` / `ERR_RECEIPT_API_NOT_CONFIGURED`, без silent fallback «Покупка по чеку».
- **Preview без лишнего HTTP:** после успешного `check/get` ответ сохраняется в `receipt_fiscal_previews` (ключ user + fn + i + fp). `ParseForUser` читает эту таблицу первой; fallback — `receipt_imports`, затем `receipt_scan_jobs`; к proverkacheka — **один** `check/get`, только если в БД нет валидного снимка.
- Middleware: `RequireFeature("receipt_scanner")` на группу `/receipts/*`.
- OpenAPI: новые коды ошибок, `POST /receipts/attach`, admin PUT token.

## Прикрепление к существующей операции

Отдельный use-case от `POST /receipts/import`:

| | Import | Attach |
|---|--------|--------|
| Создаёт операции | Да, N расходов | Нет |
| Пишет `receipt_imports` | Да | Да (или reuse по fn+i+fp) |
| Обновляет `transactions` | insert с `store_id` и `merchant_id` | `receipt_import_id`, опционально `store_id`; `merchant_id` только если у расхода его ещё не было |
| Duplicate fiscal key | 409 import | reuse import id |
| Сумма ≠ операции | N/A (суммы из позиций) | 409 без `acknowledge_sum_mismatch`; UI — «Добавить всё равно» |

Клиент при mismatch сначала показывает диалог; при согласии повторяет запрос с `"acknowledge_sum_mismatch": true`.

## Запасные провайдеры (не MVP)

См. BMAD [receipt-scanner-byo-api-brief](../bmad_docs/planning-artifacts/receipt-scanner-byo-api-brief.md): **code-qr.ru**, **OFD QR Cash** — после стабильного primary.

## Legacy

`BUHGALTER_FNS_SESSION_ID` и цепочка `:9999` / kkt-online — не расширять; после v1.10 оставить максимум как скрытый opt-in или удалить из доки пользователя.
