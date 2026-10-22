# Сканер чеков

Текущий код (v1.3+): [docs/receipt-scanner.md](../docs/receipt-scanner.md). Целевой переработанный релиз — **v1.10.0** ([ROADMAP](../ROADMAP.md#v1100)): QR на устройстве, BYO состав (proverkacheka), прикрепление к существующим операциям, опционально ИИ и вложение фото. Web + Android.

Feature flag: `receipt_scanner` (см. [feature-toggles.md](feature-toggles.md)). Модуль активен только с токеном API в админке — см. [README](../README.md#сканер-чеков-токен-proverkacheka).

## Принятые решения

- **Формат:** QR 54-ФЗ; состав — **proverkacheka.com** (токен инстанса в админке).
- **Импорт:** одна операция `expense` на позицию; общий счёт; категории на строках.
- **Прикрепление:** к уже созданному расходу; mismatch суммы → «Добавить всё равно» / «Отмена»; `POST /receipts/attach`.
- **Магазины:** из чека в `stores`; фильтры и статистика при наличии данных.
- **Хранение:** JSON в `receipt_imports`; фото не хранится.

## Уроки экспериментальной ветки

Не опираться на `BUHGALTER_FNS_SESSION_ID`, `:9999` с VPS и silent fallback «Покупка по чеку». QR надёжнее на устройстве, не на серверном decode фото.

## Scope v1.10.0 (сканер)

1. QR на устройстве (Android + web).
2. BYO proverkacheka в админке.
3. Attach к существующей операции.
4. Опционально: ИИ по фото, вложение снимка.

## Зависимости

- [Android-клиент](android-client.md), [mobile-web.md](mobile-web.md)
- [merchants-tags.md](merchants-tags.md), [category-rules-inbox.md](category-rules-inbox.md)
