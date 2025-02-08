<p align="center">
<img src="/assets/logo.png" alt="ExpenseOwl Logo" width="250" height="250" /><br>
<h1 align="center">ExpenseOwl</h1><br>

<p align="center">
<a href="https://github.com/tanq16/expenseowl/actions/workflows/release.yml"><img src="https://github.com/tanq16/expenseowl/actions/workflows/release.yml/badge.svg" alt="Release Build"></a>&nbsp;<a href="https://goreportcard.com/report/github.com/tanq16/expenseowl"><img src="https://goreportcard.com/badge/github.com/tanq16/expenseowl" alt="Go Report Card"></a><br>
<a href="https://github.com/Tanq16/expenseowl/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/tanq16/expenseowl"></a>&nbsp;<a href="https://hub.docker.com/r/tanq16/expenseowl"><img alt="Docker Pulls" src="https://img.shields.io/docker/pulls/tanq16/expenseowl"></a>
</p>
</p>

`ExpenseOwl` is an extremely simple expense tracking system with a modern monthly pie-chart visualization. It tracks daily expenses, visualizes monthly spending patterns, and maintains an overview of financial habits.

---

- [Why Create This](#why-create-this)
- [Features](#features)
- [Screenshots](#screenshots)
- [Installation](#installation)
- [Usage](#usage)
- [Tech Stack](#technology-stack)

# Why Create This?

There are a ton of amazing projects for expense tracking across GitHub ([Actual](https://github.com/actualbudget/actual), [Firefly III](https://github.com/firefly-iii/firefly-iii), etc.). They're all incredible, but they aren't the *fastest* when trying to add expenses, and offer a ton of features which I don't use. Some use varying formats of data or complex APIs. *Don't get me wrong*, they're great when needed, but I wanted something dead simple that just gives me a pie chart per month and a tabular representation. NOTHING else!

Hence, I created this project, which I use in my homelab to track my expenses. The data is just JSON, so I can do whatever I want with that, including using `jq` to convert to CSV. The UI is elegant and mobile-friendly.

The intention of this app is to track spending across your categories in a simplistic manner. No complicated searching, no editing - just add, delete, and view! This intention is not going to change throughout the lifecycle of this project. This is not an app for budgeting, it's for tracking.

# Features

### Core Functionality

- Simple expense tracking with essential details only (optional name, date without time, amount, and category)
- UUID-based expense identification in the backend
- Flat file storage system (`data/expenses.json`)
- Multi-architecture Docker container with support for persistent storage
- REST API for expense management
- Single-user focused (mainly for a homelab deployment)
- CLI for both server and client (if needed) operations
- Custom categories via environment variable (`EXPENSE_CATEGORIES`) with sensible defaults
- Custom currency symbol in the frontend via environment variable (`CURRENCY`)

### Visualization

1. Dashboard with expense category breakdown (pie chart)
    - Click on a category to exclude it from the graph, click again to add it back
    - This helps visualize the breakdown without considering some categories like Rent
    - The legend shows a total expenditure of the month along with a total without the "Rent" category
2. Table view for detailed expense listing
    - This is where you can view individual expenses chronologically and delete them
    - You can use the browser's search to find a name if needed
3. Month-by-month navigation

### Progressive Web App (PWA)

The frontend of ExpenseOwl can be installed as a Progressive Web App on desktop and mobile devices. To install:

- Desktop: Click the install icon in your browser's address bar
- iOS: Use Safari's "Add to Home Screen" option in the share menu
- Android: Use Chrome's "Install" option in the menu

### Intention of Use

Reiterating that you should use this to just add expenses quickly. The default name for an expense is `unnamed` and the date is automatically set to current date. There's a default list of categories to choose from, which can be edited very easily.

In the ideal case, `enter the amount and choose category` - that's it!

For a bit more involved case, `enter the amount and name, choose the category, and select the date` - still very simple!

The application only allows addition and deletion, there's no need for editing. There are no tags, no wallet info, no budgeting, nothing! Plain and simple for the win.

# Screenshots

| | Desktop View | Mobile View |
| --- | --- | --- |
| Dashboard Light | <img src="/assets/dashboard-light.png" alt="Dashboard Light" /> | <img src="/assets/mobile-dashboard-light.png" alt="Mobile Dashboard Light" /> |
| Dashboard Dark | <img src="/assets/dashboard-dark.png" alt="Dashboard Dark" /> | <img src="/assets/mobile-dashboard-dark.png" alt="Mobile Dashboard Dark" /> |
| Table Light | <img src="/assets/table-light.png" alt="Table Light" /> | <img src="/assets/mobile-table-light.png" alt="Mobile Table Light" /> |
| Table Dark | <img src="/assets/table-dark.png" alt="Table Dark" /> | <img src="/assets/mobile-table-dark.png" alt="Mobile Table Dark" /> |

The interface automatically adapts to system preferences for themes.

# Installation

### Go Install

```bash
go install github.com/tanq16/expenseowl/cmd/expenseowl@latest
```

### Docker Installation

```bash
docker pull tanq16/expenseowl:main
```

```bash
docker run -d \
--name expenseowl \
-p 8080:8080 \
-e EXPENSE_CATEGORIES="Rent,Food,Transport,Fun,Bills" \
-e CURRENCY=jpy \
-v expenseowl_data:/app/data \
tanq16/expenseowl:main # EXPENSE_CATEGORIES, CURRENCY are optional configs
```

To use it with Docker compose or a container-management system like Portainer or Dockge, use this YAML definition:

```yaml
version: "3.8"
services:
  budgetlord:
    image: tanq16/expenseowl:main
    restart: unless-stopped
    ports:
      - 5006:8080
    volumes:
      - /home/tanq/expenseowl:/app/data # CHANGE DIR
```

### Building from Source

```bash
git clone https://github.com/tanq16/expenseowl.git && \
cd expenseowl
```

```bash
go build ./cmd/expenseowl
```

# Usage

Ideally, once deployed, just use the web interface and you're good to go. Access the web interface through your browser:

- Dashboard: `http://localhost:8080/`
- Table View: `http://localhost:8080/table`

> [!NOTE]
> This app has no authentication, so deploy carefully. It works very well with a reverse proxy like Nginx Proxy Manager and is mainly intended for homelab use.

If there are command-line automations that are required for use with the REST API, read on!

### CLI Mode

The application binary can run in either server or client mode:

Server Mode (Default):

```bash
./expenseowl
# or explicitly
./expenseowl --serve
```

Client Mode:

```bash
./expenseowl --client --addr localhost:8080
```

In client mode, you'll be prompted to enter the expense name, category (select from a list), amount, and date (in YYYY-MM-DD; optional, sets to current date when not provided).

### REST API

Add Expense:

```bash
curl -X PUT http://localhost:8080/expense \
-H "Content-Type: application/json" \
-d '{
    "name": "Groceries",
    "category": "Food",
    "amount": 75.50,
    "date": "2024-03-15T14:30:00Z"
}'
```

Get All Expenses:

```bash
curl http://localhost:8080/expenses
```

### Config Options

##### Currency Settings

ExpenseOwl supports multiple currencies through the CURRENCY environment variable. If not specified, it defaults to USD ($). Example, to run with Euro, use the following environment variable:

```bash
CURRENCY=eur ./expenseowl
```

Similarly, the environment variable can be set in a compose stack or using `-e` in the command line with a Docker command. The full list of currencies supported are present in [this file](https://github.com/Tanq16/ExpenseOwl/blob/main/internal/config/config.go#L27).

##### Category Settings

ExpenseOwl also supports custom categories, which can be specified through environment variables like so:

```bash
EXPENSE_CATEGORIES="Rent,Food,Transport,Fun,Bills" ./expenseowl
```

Similarly, it can be specified in a Docker compose stack of a Docker CLI command with the `-e` flag. Refer to the examples shown above in the README.

# Technology Stack

- Backend: Go
- Storage: JSON file system
- Frontend: Chart.js and vanilla web stack (HTML, JS, CSS)
- Interface: CLI + Web UI
