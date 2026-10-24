# Android — сканер чеков (54-ФЗ)

Parity с web по [receipt-scanner.md](receipt-scanner.md). **Только online** (нет офлайн-outbox для чеков).

## Включение

Как на сервере: feature `receipt_scanner` + токен proverkacheka в админке. Клиент смотрит `GET /api/v1/features`: `receipt_scanner` и при наличии поля — `receipt_api_token_set`.

UI скрыт, если модуль выключен (`isReceiptScannerEnabled` в `android/ui/src/lib/receipt/receipt-module.ts`).

## Маршруты

| Путь | Назначение |
|------|------------|
| `/receipts/import?from=&account=` | Import: N расходов из чека |
| `/receipts/attach?attach_tx=&from=` | Прикрепить чек к существующему расходу |
| `/receipts/[id]?transaction_id=` | Просмотр чека, открепление |

Gate: `featureRequiredForPath` — префикс `/receipts` → `receipt_scanner` ([features.ts](../android/ui/src/lib/features.ts)).

Хелперы: `receiptImportPath`, `receiptAttachPath`, `receiptViewPath` — `android/ui/src/lib/android/form-routes.ts`.

## Точки входа

- **«Чек»** (`receipt.title`) в блоке быстрых кнопок на главной, `/transactions`, `/accounts/[id]` — переход на import (при включённом модуле).
- Меню операции: **«Прикрепить чек»**, **«Данные чека»**, **«Удалить чек»** — [TransactionList.svelte](../android/ui/src/lib/components/TransactionList.svelte).

## UX (import / attach)

Один экранный поток [ReceiptScanForm.svelte](../android/ui/src/lib/components/ReceiptScanForm.svelte) (`FormPageShell`, не модалка):

1. Две кнопки: камера (`html5-qrcode`, [parse-qr.ts](../android/ui/src/lib/receipt/parse-qr.ts)) и загрузка файла (`BarcodeDetector`, иначе `scanFile`). Раскладка — [ниже](#кнопки). QR читается на телефоне, дальше `POST /receipts/parse` со строкой. Картинка на сервер не уходит. Камера: `CAMERA` в `AndroidManifest` (`required=false`). Если код на файле не прочитался — `receipt.parseError`. Если запрос чека не дошёл до сервера — `receipt.requestNotSent`, не «Сессия сохранена».
2. Альтернатива: строка QR → `POST /receipts/parse`.
3. Подтверждение: счёт (import), список позиций без горизонтального скролла — название, сумма и одна кнопка ([ReceiptItemCategoryButton.svelte](../android/ui/src/lib/components/ReceiptItemCategoryButton.svelte)). Кнопка открывает окно с категорией, подкатегорией и «+» ([ReceiptInlineSubcategoryField.svelte](../android/ui/src/lib/components/ReceiptInlineSubcategoryField.svelte)). «Общая категория» свёрнута в спойлер (подсказка, выбор и «Проставить без категории» внутри). Нижние кнопки — [ниже](#кнопки).
4. Дубликат: только сообщение `already_imported` / `already_in_ledger` + «Назад» (без таблицы).
5. Save → API import/attach с `items[]`; `dataRefreshTick` + `leaveForm(from)`.

Attach: сумма операции vs чек — диалог «Добавить всё равно» при mismatch (`acknowledge_sum_mismatch`).

## Кнопки

Тот же `ReceiptScanForm` на import и attach. Веб-модалка чека этим разделом не описывается.

Шаг сканирования и экран дубликата (на дубликате «Назад» остаётся в теле страницы):

- «Сканировать QR» (пока камера открыта — «Остановить») и «Загрузить фото» стоят в одной строке по центру. Если строка не влезает, перенос тоже по центру. На всю ширину их не растягивать.
- Нижней полосы нет. `form-page-footer` рисуется только вместе с кнопками подтверждения.

Подтверждение, когда чек не дубликат:

- «Назад» и «Сохранить» (import) или «Прикрепить» (attach) — прямые кнопки футера `FormPageShell`, как «Отмена» и «Сохранить» на создании операции: две равные колонки на всю ширину, одна высота, подпись в одну строку. Числа операций в подписи нет.
- «Назад», «Прикрепить» и условия, когда сохранение недоступно, те же. «Проставить без категории» остаётся в спойлере общей категории.

## API-клиент

`android/ui/src/lib/api/client.ts` — те же контракты, что web: `importReceipt`, `attachReceipt`, `detachReceipt`, `getReceipt`, scan-jobs, флаги preview.

## i18n

Ключи `receipt.*` в `android/ui/src/lib/i18n/{ru,en}.json` (копия web; CI: `make copy-ui-i18n` → `server/ui_locales`).

## Отличия от web

| | Web | Android |
|---|-----|---------|
| Оболочка | ModalShell | FormPageShell + `leaveForm` |
| Import | Модалка на странице | Отдельный маршрут `/receipts/import` |
| Офлайн | — | `requireOnline()` на входе import/attach |

## Тесты

- `android/ui/src/lib/receipt/receipt-module.test.ts`
- `form-routes.test.ts` — пути receipt

Сборка UI: `cd android/ui && npm run check`.

См. также [android-client-ui.md](android-client-ui.md), [android-client.md](android-client.md).
