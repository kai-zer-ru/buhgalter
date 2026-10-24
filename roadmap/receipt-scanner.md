# Сканер чеков

Текущий код: [docs/receipt-scanner.md](../docs/receipt-scanner.md). Релиз — **v1.10.0** ([ROADMAP](../ROADMAP.md#v1100)): один пункт, Web + Android. QR на устройстве, BYO состав (proverkacheka), import и прикрепление к существующей операции. ИИ по фото и вложение снимка — [общие планы](../ROADMAP.md#общие-планы), не v1.10.

Feature flag: `receipt_scanner` (см. [feature-toggles.md](feature-toggles.md)). Модуль активен только с токеном API в админке — см. [README](../README.md#сканер-чеков-токен-proverkacheka).

## Принятые решения

- **Формат:** QR 54-ФЗ; состав — **proverkacheka.com** (токен инстанса в админке).
- **Импорт:** одна операция `expense` на позицию; общий счёт; категории на строках.
- **Прикрепление:** к уже созданному расходу; mismatch суммы → «Добавить всё равно» / «Отмена»; `POST /receipts/attach`.
- **Магазины:** из чека в `merchants` (тот же каталог, что у ручной операции). `store_id` по-прежнему пишется, отдельного поля и фильтра точки нет. Фильтры и блок «По магазинам» считают `merchant_id`.
- **Хранение:** JSON в `receipt_imports`; фото не хранится.

## Уроки экспериментальной ветки

Не опираться на `BUHGALTER_FNS_SESSION_ID`, `:9999` с VPS и silent fallback «Покупка по чеку». QR надёжнее на устройстве, не на серверном decode фото.

## Scope v1.10.0 (сканер)

Один функционал. Подробности — в [docs/receipt-scanner.md](../docs/receipt-scanner.md): QR на устройстве (Android + web), BYO proverkacheka в админке, import и attach, категории строк, дедуп, открепление, магазин чека.

ИИ по фото и вложение снимка — не этот релиз.

## Зависимости

- [Android-клиент](android-client.md), [mobile-web.md](mobile-web.md)
- [merchants-tags.md](merchants-tags.md), [category-rules-inbox.md](category-rules-inbox.md)
