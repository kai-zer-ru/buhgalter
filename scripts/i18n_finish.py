#!/usr/bin/env python3
"""Finish i18n migration: fix error helpers, remaining Write -> WriteR."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1] / "server" / "internal"

HELPERS = [
    "writeAccountError",
    "writeDebtorError",
    "writeDebtError",
    "writeTxError",
    "writeCreditError",
    "writeCategoryError",
    "writeSubcategoryError",
    "writeUserLoadError",
]

MSG_TO_KEY = {
    "некорректный JSON": "ERR_INVALID_JSON",
    "логин и пароль обязательны": "ERR_LOGIN_PASSWORD_REQUIRED",
    "логин должен быть от 3 до 32 символов": "ERR_LOGIN_LENGTH",
    "укажите имя администратора": "ERR_DISPLAY_NAME_REQUIRED",
    "имя не должно быть длиннее 64 символов": "ERR_DISPLAY_NAME_LENGTH",
    "пароли не совпадают": "PASSWORDS_MISMATCH",
    "external_url должен начинаться с http:// или https://": "ERR_EXTERNAL_URL",
    "для восстановления введите RESTORE": "ERR_RESTORE_CONFIRM",
    "файл не загружен": "ERR_FILE_REQUIRED",
    "ожидается файл .db": "ERR_DB_FILE_REQUIRED",
    "некорректное время бэкапа": "ERR_BACKUP_TIME",
    "notification_secret_key не должен быть пустым": "ERR_SECRET_KEY_EMPTY",
    "нельзя удалить себя": "ERR_CANNOT_DELETE_SELF",
    "язык должен быть ru или en": "ERR_LANGUAGE",
    "валюта должна быть RUB, USD или EUR": "ERR_CURRENCY",
    "тема должна быть light или dark": "ERR_THEME",
    "некорректный часовой пояс": "ERR_TIMEZONE",
    "channel должен быть telegram или max": "ERR_CHANNEL",
    "канал уведомлений не настроен": "ERR_NOTIFICATION_CHANNEL",
    "group_by: day, week или month": "ERR_GROUP_BY",
    "некорректный формат даты периода": "ERR_PERIOD_DATE",
    "некорректный multipart запрос": "ERR_MULTIPART",
    "не указан id задачи": "ERR_JOB_ID",
    "нельзя удалить отдельную ногу перевода": "ERR_TRANSFER_DELETE",
    "счёт с таким именем уже существует": "CONFLICT_ACCOUNT_NAME",
    "категория с таким именем уже существует": "CONFLICT_CATEGORY_NAME",
    "подкатегория с таким именем уже существует": "CONFLICT_SUBCATEGORY_NAME",
    "должник с таким именем уже существует": "CONFLICT_DEBTOR_NAME",
    "у должника есть активные долги": "CONFLICT_DEBTOR_ACTIVE_DEBTS",
    "долг уже закрыт": "CONFLICT_DEBT_CLOSED",
    "нельзя взять в долг у того, кому вы уже дали — закройте активный долг «дал в долг»": "CONFLICT_DEBT_CANNOT_BORROW",
    "нельзя дать в долг тому, у кого вы уже взяли — закройте активный долг «взял в долг»": "CONFLICT_DEBT_CANNOT_LEND",
    "логин уже занят": "CONFLICT_LOGIN_TAKEN",
    "требуется авторизация": "UNAUTHORIZED",
    "неверный логин или пароль": "INVALID_CREDENTIALS",
    "слишком много попыток входа": "RATE_LIMITED",
    "начальная настройка уже выполнена": "ALREADY_CONFIGURED",
    "сервер временно недоступен": "SERVICE_UNAVAILABLE",
    "внутренняя ошибка сервера": "INTERNAL_ERROR",
    "пользователь не найден": "ERR_USER_NOT_FOUND",
    "недостаточно прав": "FORBIDDEN",
    "требуется API-токен": "ERR_API_TOKEN_REQUIRED",
    "недействительный API-токен": "ERR_API_TOKEN_INVALID",
    "API-токен истёк": "ERR_API_TOKEN_EXPIRED",
    "регистрация отключена": "REGISTRATION_DISABLED",
    "имя счёта: от 1 до 64 символов": "ERR_ACCOUNT_NAME_LENGTH",
    "тип счёта: cash или bank": "ERR_ACCOUNT_TYPE",
    "для банковского счёта укажите bank_id": "ERR_ACCOUNT_BANK_REQUIRED",
    "для наличного счёта bank_id не указывается": "ERR_ACCOUNT_BANK_FORBIDDEN",
    "банк не найден": "ERR_ACCOUNT_BANK_NOT_FOUND",
    "нельзя редактировать архивный счёт": "ERR_ACCOUNT_ARCHIVED_EDIT",
    "нельзя назначить основным архивный счёт": "ERR_ACCOUNT_PRIMARY_ARCHIVED",
    "некорректный начальный баланс": "ERR_ACCOUNT_INVALID_BALANCE",
    "некорректный баланс": "ERR_ACCOUNT_INVALID_BALANCE",
    "укажите имя должника": "ERR_DEBTOR_NAME_REQUIRED",
    "направление: lent или borrowed": "ERR_DEBT_DIRECTION",
    "некорректная сумма": "ERR_INVALID_AMOUNT",
    "некорректная дата возврата": "ERR_INVALID_DUE_DATE",
    "некорректная дата операции": "ERR_INVALID_DEBT_DATE",
    "укажите счёт при изменении баланса": "ERR_DEBT_ACCOUNT_REQUIRED",
    "счёт не найден": "ERR_ACCOUNT_NOT_FOUND",
    "счёт архивирован": "ERR_ACCOUNT_ARCHIVED",
    "сумма погашения превышает остаток долга": "ERR_SETTLE_AMOUNT",
    "укажите должника": "ERR_DEBTOR_REQUIRED",
    "тип операции: income или expense": "ERR_TX_TYPE",
    "сумма должна быть положительной": "ERR_TX_AMOUNT_POSITIVE",
    "категория не найдена": "ERR_CATEGORY_NOT_FOUND",
    "тип категории не совпадает с типом операции": "ERR_CATEGORY_TYPE_MISMATCH",
    "подкатегория не найдена": "ERR_SUBCATEGORY_NOT_FOUND",
    "счета перевода должны различаться": "ERR_TRANSFER_SAME_ACCOUNT",
    "для перевода используйте /transfers": "ERR_USE_TRANSFERS_ENDPOINT",
    "некорректный срок": "ERR_CREDIT_INVALID_TERM",
    "кредит уже закрыт": "ERR_CREDIT_CLOSED",
    "некорректная периодичность": "ERR_CREDIT_INVALID_INTERVAL",
    "нет неоплаченных платежей по графику": "ERR_CREDIT_NO_PENDING_PAYMENT",
    "нельзя удалить платёж, учтённый при добавлении": "ERR_CREDIT_CANNOT_REMOVE_RETRO",
    "укажите дату завершения": "ERR_CREDIT_COMPLETE_DATE",
    "некорректный status": "ERR_CREDIT_INVALID_STATUS",
    "некорректная сумма платежа": "ERR_CREDIT_INVALID_PAYMENT",
    "имя категории: от 1 до 64 символов": "ERR_CATEGORY_NAME_LENGTH",
    "системную категорию нельзя изменить": "ERR_CATEGORY_SYSTEM_READONLY",
    "некорректные данные категории": "ERR_CATEGORY_INVALID",
    "имя подкатегории: от 1 до 64 символов": "ERR_SUBCATEGORY_NAME_LENGTH",
    "у системной категории нельзя создавать подкатегории": "ERR_SUBCATEGORY_SYSTEM",
    "некорректный порядок категорий": "ERR_CATEGORY_REORDER",
    "некорректный порядок подкатегорий": "ERR_SUBCATEGORY_REORDER",
    "системную категорию нельзя удалить": "ERR_CATEGORY_SYSTEM_DELETE",
    "confirm=true обязателен": "ERR_IMPORT_CONFIRM",
    "имя токена обязательно": "ERR_TOKEN_NAME_REQUIRED",
    "некорректная дата истечения": "ERR_TOKEN_EXPIRES",
}

CODE_CONST = {
    '"INTERNAL_ERROR"': "apperror.InternalError",
    '"NOT_FOUND"': "apperror.NotFound",
    '"CONFLICT"': "apperror.Conflict",
    '"ALREADY_CONFIGURED"': "apperror.AlreadyConfigured",
    '"SERVICE_UNAVAILABLE"': "apperror.ServiceUnavailable",
    '"VALIDATION_ERROR"': "apperror.ValidationError",
}

WRITE_RE = re.compile(
    r'apperror\.Write\(w, (http\.Status[A-Za-z0-9]+), ([^,]+), ("(?:\\.|[^"\\])*"|apperror\.[A-Za-z]+)\)'
)


def rewrite_write(content: str) -> str:
    def repl(m: re.Match[str]) -> str:
        status, code, msg = m.group(1), m.group(2), m.group(3)
        code_out = CODE_CONST.get(code.strip(), code.strip())
        if msg.startswith("apperror."):
            return f"apperror.WriteR(w, r, {status}, {code_out})"
        raw = msg[1:-1]
        key = MSG_TO_KEY.get(raw)
        if key:
            if key in {
                "UNAUTHORIZED", "INVALID_CREDENTIALS", "RATE_LIMITED", "PASSWORDS_MISMATCH",
                "ALREADY_CONFIGURED", "SERVICE_UNAVAILABLE", "INTERNAL_ERROR", "NOT_FOUND",
                "CONFLICT", "FORBIDDEN", "REGISTRATION_DISABLED",
            } and code_out.startswith("apperror."):
                const = f"apperror.{key}" if key not in ("NOT_FOUND", "CONFLICT", "INTERNAL_ERROR", "ALREADY_CONFIGURED", "SERVICE_UNAVAILABLE") else code_out
                if key == "FORBIDDEN":
                    const = "apperror.Forbidden"
                elif key == "REGISTRATION_DISABLED":
                    const = "apperror.RegistrationDisabled"
                elif key == "UNAUTHORIZED":
                    const = "apperror.Unauthorized"
                elif key == "INVALID_CREDENTIALS":
                    const = "apperror.InvalidCredentials"
                elif key == "RATE_LIMITED":
                    const = "apperror.RateLimited"
                elif key == "PASSWORDS_MISMATCH":
                    const = "apperror.PasswordsMismatch"
                else:
                    const = code_out
                return f"apperror.WriteR(w, r, {status}, {const})"
            return f'apperror.WriteR(w, r, {status}, {code_out}, "{key}")'
        if code.strip() in ('"INTERNAL_ERROR"',):
            return f"apperror.WriteR(w, r, {status}, apperror.InternalError)"
        if code.strip() in ('"NOT_FOUND"',):
            return f"apperror.WriteR(w, r, {status}, apperror.NotFound)"
        if code.strip() in ('"CONFLICT"',):
            return f"apperror.WriteR(w, r, {status}, apperror.Conflict)"
        return m.group(0)

    return WRITE_RE.sub(repl, content)


def fix_helpers(content: str) -> str:
    for name in HELPERS:
        content = re.sub(
            rf"func {name}\(w http\.ResponseWriter, err error\)",
            f"func {name}(w http.ResponseWriter, r *http.Request, err error)",
            content,
        )
        content = re.sub(
            rf"{name}\(w, err\)",
            f"{name}(w, r, err)",
            content,
        )
    return content


def main() -> None:
    for path in ROOT.rglob("*.go"):
        if path.name.endswith("_test.go"):
            continue
        text = path.read_text(encoding="utf-8")
        new = fix_helpers(text)
        new = rewrite_write(new)
        if new != text:
            path.write_text(new, encoding="utf-8")
            print(path.relative_to(ROOT.parent.parent))


if __name__ == "__main__":
    main()
