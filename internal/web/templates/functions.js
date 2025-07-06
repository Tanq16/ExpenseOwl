const colorPalette = [
    '#FF6B6B', '#4ECDC4', '#45B7D1', '#96CEB4', 
    '#FFBE0B', '#FF006E', '#8338EC', '#3A86FF', 
    '#FB5607', '#38B000', '#9B5DE5', '#F15BB5'
];
const currencyBehaviors = {
    "usd": { symbol: "$", useComma: false, useDecimals: true },
    "eur": { symbol: "€", useComma: true, useDecimals: true },
    "gbp": { symbol: "£", useComma: false, useDecimals: true },
    "jpy": { symbol: "¥", useComma: false, useDecimals: false },
    "cny": { symbol: "¥", useComma: false, useDecimals: true },
    "krw": { symbol: "₩", useComma: false, useDecimals: false },
    "inr": { symbol: "₹", useComma: false, useDecimals: true },
    "rub": { symbol: "₽", useComma: true, useDecimals: true },
    "brl": { symbol: "R$", useComma: true, useDecimals: true },
    "zar": { symbol: "R", useComma: false, useDecimals: true },
    "aed": { symbol: "AED", useComma: false, useDecimals: true },
    "aud": { symbol: "A$", useComma: false, useDecimals: true },
    "cad": { symbol: "C$", useComma: false, useDecimals: true },
    "chf": { symbol: "Fr", useComma: false, useDecimals: true },
    "hkd": { symbol: "HK$", useComma: false, useDecimals: true },
    "sgd": { symbol: "S$", useComma: false, useDecimals: true },
    "thb": { symbol: "฿", useComma: false, useDecimals: true },
    "try": { symbol: "₺", useComma: true, useDecimals: true },
    "mxn": { symbol: "Mex$", useComma: false, useDecimals: true },
    "php": { symbol: "₱", useComma: false, useDecimals: true },
    "pln": { symbol: "zł", useComma: true, useDecimals: true },
    "sek": { symbol: "kr", useComma: false, useDecimals: true },
    "nzd": { symbol: "NZ$", useComma: false, useDecimals: true },
    "dkk": { symbol: "kr.", useComma: true, useDecimals: true },
    "idr": { symbol: "Rp", useComma: false, useDecimals: true },
    "ils": { symbol: "₪", useComma: false, useDecimals: true },
    "vnd": { symbol: "₫", useComma: true, useDecimals: false },
    "myr": { symbol: "RM", useComma: false, useDecimals: true },
};

// let currentCurrency = 'usd';
// let startDate = 1;
// let currentDate = new Date();
// let allExpenses = [];
// let allTags = new Set();

function formatCurrency(amount) {
    const behavior = currencyBehaviors[currentCurrency] || { symbol: '$', useComma: false, useDecimals: true };
    const isNegative = amount < 0;
    const absAmount = Math.abs(amount);
    const options = {
        minimumFractionDigits: behavior.useDecimals ? 2 : 0,
        maximumFractionDigits: behavior.useDecimals ? 2 : 0,
    };
    let formattedAmount = new Intl.NumberFormat(behavior.useComma ? 'de-DE' : 'en-US', options).format(absAmount);
    const postfixCurrencies = new Set(['kr', 'kr.', 'Fr', 'zł']);
    let result = postfixCurrencies.has(behavior.symbol) ? `${formattedAmount} ${behavior.symbol}` : `${behavior.symbol}${formattedAmount}`;
    return isNegative ? `-${result}` : result;
}

function getUserTimeZone() {
    return Intl.DateTimeFormat().resolvedOptions().timeZone;
}

function formatMonth(date) {
    return date.toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'long',
        timeZone: getUserTimeZone()
    });
}

function getISODateWithLocalTime(dateInput) {
    const [year, month, day] = dateInput.split('-').map(Number);
    const now = new Date();
    const hours = now.getHours();
    const minutes = now.getMinutes();
    const seconds = now.getSeconds();
    const localDateTime = new Date(year, month - 1, day, hours, minutes, seconds);
    return localDateTime.toISOString();
}

function formatDateFromUTC(utcDateString) {
    const date = new Date(utcDateString);
    return date.toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        timeZoneName: 'short'
    });
}

function updateMonthDisplay() {
    const currentMonthEl = document.getElementById('currentMonth');
    if (currentMonthEl) {
        currentMonthEl.textContent = formatMonth(currentDate);
    }
}

function getMonthBounds(date) {
    const localDate = new Date(date);
    if (startDate === 1) {
        const startLocal = new Date(localDate.getFullYear(), localDate.getMonth(), 1);
        const endLocal = new Date(localDate.getFullYear(), localDate.getMonth() + 1, 0, 23, 59, 59, 999);
        return { start: new Date(startLocal.toISOString()), end: new Date(endLocal.toISOString()) };
    }
    let thisMonthStartDate = startDate;
    let prevMonthStartDate = startDate;

    const currentMonth = localDate.getMonth();
    const currentYear = localDate.getFullYear();
    const daysInCurrentMonth = new Date(currentYear, currentMonth + 1, 0).getDate();
    thisMonthStartDate = Math.min(thisMonthStartDate, daysInCurrentMonth);
    const prevMonth = currentMonth === 0 ? 11 : currentMonth - 1;
    const prevYear = currentMonth === 0 ? currentYear - 1 : currentYear;
    const daysInPrevMonth = new Date(prevYear, prevMonth + 1, 0).getDate();
    prevMonthStartDate = Math.min(prevMonthStartDate, daysInPrevMonth);

    if (localDate.getDate() < thisMonthStartDate) {
        const startLocal = new Date(prevYear, prevMonth, prevMonthStartDate);
        const endLocal = new Date(currentYear, currentMonth, thisMonthStartDate - 1, 23, 59, 59, 999);
        return { start: new Date(startLocal.toISOString()), end: new Date(endLocal.toISOString()) };
    } else {
        const nextMonth = currentMonth === 11 ? 0 : currentMonth + 1;
        const nextYear = currentMonth === 11 ? currentYear + 1 : currentYear;
        const daysInNextMonth = new Date(nextYear, nextMonth + 1, 0).getDate();
        let nextMonthStartDate = Math.min(startDate, daysInNextMonth);
        const startLocal = new Date(currentYear, currentMonth, thisMonthStartDate);
        const endLocal = new Date(nextYear, nextMonth, nextMonthStartDate - 1, 23, 59, 59, 999);
        return { start: new Date(startLocal.toISOString()), end: new Date(endLocal.toISOString()) };
    }
}

function getMonthExpenses(expenses) {
    const { start, end } = getMonthBounds(currentDate);
    return expenses.filter(exp => {
        const expDate = new Date(exp.date);
        return expDate >= start && expDate <= end;
    }).sort((a, b) => new Date(b.date) - new Date(a.date));
}

function escapeHTML(str) {
    if (typeof str !== 'string') return str;
    return str.replace(/[&<>'"]/g,
        tag => ({
            '&': '&amp;',
            '<': '&lt;',
            '>': '&gt;',
            "'": '&#39;',
            '"': '&quot;'
        }[tag] || tag)
    );
}
