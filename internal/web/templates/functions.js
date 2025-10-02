const colorPalette = [
    '#FF6B6B', '#4ECDC4', '#45B7D1', '#96CEB4', 
    '#FFBE0B', '#FF006E', '#8338EC', '#3A86FF', 
    '#FB5607', '#38B000', '#9B5DE5', '#F15BB5'
];
const currencyBehaviors = {
    usd: {symbol: "$", useComma: false, useDecimals: true, useSpace: false, right: false},
    eur: {symbol: "€", useComma: true, useDecimals: true, useSpace: false, right: false},
    gbp: {symbol: "£", useComma: false, useDecimals: true, useSpace: false, right: false},
    jpy: {symbol: "¥", useComma: false, useDecimals: false, useSpace: false, right: false},
    cny: {symbol: "¥", useComma: false, useDecimals: true, useSpace: false, right: false},
    krw: {symbol: "₩", useComma: false, useDecimals: false, useSpace: false, right: false},
    inr: {symbol: "₹", useComma: false, useDecimals: true, useSpace: false, right: false},
    rub: {symbol: "₽", useComma: true, useDecimals: true, useSpace: false, right: false},
    brl: {symbol: "R$", useComma: true, useDecimals: true, useSpace: false, right: false},
    zar: {symbol: "R", useComma: false, useDecimals: true, useSpace: true, right: true},
    aed: {symbol: "AED", useComma: false, useDecimals: true, useSpace: true, right: true},
    aud: {symbol: "A$", useComma: false, useDecimals: true, useSpace: false, right: false},
    cad: {symbol: "C$", useComma: false, useDecimals: true, useSpace: false, right: false},
    chf: {symbol: "Fr", useComma: false, useDecimals: true, useSpace: true, right: true},
    hkd: {symbol: "HK$", useComma: false, useDecimals: true, useSpace: false, right: false},
    bdt: {symbol: "৳", useComma: false, useDecimals: true, useSpace: false, right: false},
    sgd: {symbol: "S$", useComma: false, useDecimals: true, useSpace: false, right: false},
    thb: {symbol: "฿", useComma: false, useDecimals: true, useSpace: false, right: false},
    try: {symbol: "₺", useComma: true, useDecimals: true, useSpace: false, right: false},
    mxn: {symbol: "Mex$", useComma: false, useDecimals: true, useSpace: false, right: false},
    php: {symbol: "₱", useComma: false, useDecimals: true, useSpace: false, right: false},
    pln: {symbol: "zł", useComma: true, useDecimals: true, useSpace: true, right: true},
    sek: {symbol: "kr", useComma: false, useDecimals: true, useSpace: true, right: true},
    nzd: {symbol: "NZ$", useComma: false, useDecimals: true, useSpace: false, right: false},
    dkk: {symbol: "kr.", useComma: true, useDecimals: true, useSpace: true, right: true},
    idr: {symbol: "Rp", useComma: false, useDecimals: true, useSpace: true, right: true},
    ils: {symbol: "₪", useComma: false, useDecimals: true, useSpace: false, right: false},
    vnd: {symbol: "₫", useComma: true, useDecimals: false, useSpace: true, right: true},
    myr: {symbol: "RM", useComma: false, useDecimals: true, useSpace: false, right: false},
    mad: {symbol: "DH", useComma: false, useDecimals: true, useSpace: true, right: true},
};

function formatCurrency(amount) {
    const behavior = currencyBehaviors[currentCurrency] || {
        symbol: "$",
        useComma: false,
        useDecimals: true,
        useSpace: false,
        right: false,
    };
    const isNegative = amount < 0;
    const absAmount = Math.abs(amount);
    const options = {
        minimumFractionDigits: behavior.useDecimals ? 2 : 0,
        maximumFractionDigits: behavior.useDecimals ? 2 : 0,
    };
    let formattedAmount = new Intl.NumberFormat(behavior.useComma ? "de-DE" : "en-US",options).format(absAmount);
    let result = behavior.right
        ? `${formattedAmount}${behavior.useSpace ? " " : ""}${behavior.symbol}`
        : `${behavior.symbol}${behavior.useSpace ? " " : ""}${formattedAmount}`;
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
