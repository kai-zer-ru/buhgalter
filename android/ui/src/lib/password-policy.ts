export function passwordHasLetterAndDigit(password: string): boolean {
	let hasLetter = false;
	let hasDigit = false;
	for (const ch of password) {
		if (/[A-Za-zА-Яа-яЁё]/.test(ch)) {
			hasLetter = true;
		}
		if (/\d/.test(ch)) {
			hasDigit = true;
		}
	}
	return hasLetter && hasDigit;
}

export function isPasswordSameAsLogin(password: string, login: string): boolean {
	if (!login.trim()) return false;
	return password.trim().toLowerCase() === login.trim().toLowerCase();
}

export function validatePasswordPolicy(password: string, login: string): boolean {
	return (
		password.length >= 8 &&
		passwordHasLetterAndDigit(password) &&
		!isPasswordSameAsLogin(password, login)
	);
}
