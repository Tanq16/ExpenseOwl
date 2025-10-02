const colorPalette = [
    '#FF6B6B', '#4ECDC4', '#45B7D1', '#96CEB4', 
    '#FFBE0B', '#FF006E', '#8338EC', '#3A86FF', 
    '#FB5607', '#38B000', '#9B5DE5', '#F15BB5'
];

const currencyBehaviors = Object.create(null);

function loadCurrenciesBehavior(currencyCatalog) {
    currencyCatalog.forEach(obj => {
    currencyBehaviors[obj.code.toUpperCase()] = {
      symbol:       obj.symbol,
      decimals:     obj.decimals,
      commaDecimal: obj.commaDecimal,
      postFixCurrency: obj.postFixCurrency
    };
  });
}

function formatCurrency(amount, currency = defaultCurrency) {
    const behavior = currencyBehaviors[currency.toUpperCase()] || { symbol: '$', commaDecimal: false, decimals: 2 };
    const isNegative = amount < 0;
    const absAmount = Math.abs(amount);
    const options = {
        minimumFractionDigits: behavior.decimals,
        maximumFractionDigits: behavior.decimals
    };
    let formattedAmount = new Intl.NumberFormat(behavior.commaDecimal ? 'de-DE' : 'en-US', options).format(absAmount);
    let result = behavior.postFixCurrency ? `${formattedAmount} ${behavior.symbol}` : `${behavior.symbol}${formattedAmount}`;
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

// function getDateWithoutTime(dateObject) {
//   // Get the year, month, and day from the original Date object
//   const year = dateObject.getFullYear();
//   const month = dateObject.getMonth(); // Month is 0-indexed
//   const day = dateObject.getDate();

//   // Create a new Date object with only the year, month, and day
//   // Hours, minutes, seconds, and milliseconds will default to 0
//   return new Date(year, month, day);
// }

function formatToConvertedAmount(amount, rate, currency = defaultCurrency) {    
    return rate == 0 ? "" : formatCurrency(amount * rate, currency);
}

function extractRateParams(expenses, defaultQuote) {
  const ratesParams = Object.create(null);         
  defaultQuote = defaultQuote.toUpperCase();

  for (const exp of expenses) {
    // it is better to convert to a date object before in case the string was not in a good format
    const day  =  new Date(exp.date).toISOString().slice(0, 10);
    //const day  =  exp.date.slice(0, 10);
    const base = (exp.currency || "").toUpperCase();
    if (!base || base == defaultQuote) continue;                     // skip if unknown base

    if (!ratesParams[day]) ratesParams[day] = Object.create(null);
    ratesParams[day][base] = [defaultQuote];                      // {day:{base:quote}}
  }
  return ratesParams;
}
