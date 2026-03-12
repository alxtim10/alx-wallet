# alx-wallet 💳

A **Digital Wallet Backend** built with **Go**, **PostgreSQL**, and **Docker**, implementing a **double-entry accounting ledger system** similar to architectures used by modern fintech systems (e.g. Stripe, Coinbase, Square).

This project demonstrates how to build a **financially safe wallet system** where money **cannot disappear or be duplicated** due to API failures or concurrency issues.

---

# Overview

Digital wallets require **strong financial consistency guarantees**.
A naive wallet design that stores balances directly in an `accounts.balance` column is **unsafe**.

Instead, this system implements a **double-entry ledger** where every transaction records:

```
debit = credit
```

This ensures that:

* money cannot be created
* money cannot be destroyed
* every transfer is auditable

---

# Tech Stack

Backend

* **Go**
* **Gin Web Framework**

Database

* **PostgreSQL**

Infrastructure

* **Docker**
* **Docker Compose**

Architecture

* Clean Architecture
* Repository Pattern
* Service Layer
* REST API

---

# System Architecture

```
Client
   │
   ▼
HTTP API (Gin)
   │
   ▼
Service Layer
   │
   ▼
Repository Layer
   │
   ▼
PostgreSQL Ledger
```

Responsibilities:

| Layer      | Responsibility            |
| ---------- | ------------------------- |
| Handler    | HTTP requests & responses |
| Service    | Business logic            |
| Repository | Database queries          |
| Database   | Source of financial truth |

---

# Project Structure

```
alx-wallet
│
├── cmd
│   └── main.go
│
├── internal
│   ├── handler
│   │   └── wallet_handler.go
│   │
│   ├── service
│   │   └── wallet_service.go
│   │
│   └── repository
│       ├── account_repository.go
│       └── ledger_repository.go
│
├── migrations
│   └── 001_init.sql
│
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── README.md
```

---

# Database Design

The system uses a **double-entry ledger**.

Tables:

```
accounts
journal_entries
ledger_entries
```

---

## accounts

Represents wallet accounts.

```
accounts
---------
id
user_id
type
created_at
```

Example account types:

* wallet
* system
* escrow

---

## journal_entries

Represents a **logical financial transaction**.

```
journal_entries
---------------
id
reference_id
description
created_at
```

Example:

```
transfer-123
deposit-456
withdrawal-789
```

---

## ledger_entries

Represents the **actual financial movement**.

```
ledger_entries
--------------
id
journal_id
user_id
amount
entry_type
created_at
```

`entry_type`:

```
debit
credit
```

---

# Double Entry Accounting

Every financial transaction creates **two ledger entries**.

Example transfer:

User A sends **500** to User B.

Ledger records:

```
User A → debit 500
User B → credit 500
```

This guarantees:

```
total debit = total credit
```

Which means the ledger **always balances**.

---

# Example Ledger Flow

Before transfer

```
User A balance: 1000
User B balance: 0
```

Transfer

```
500
```

Ledger entries created:

```
journal_entries
----------------
transfer-001

ledger_entries
----------------
User A  debit   500
User B  credit  500
```

Balances become:

```
User A = 500
User B = 500
```

---

# Running the Project

## Requirements

Install:

* Docker
* Docker Compose

---

## Start Services

```
docker compose up --build
```

This will start:

```
PostgreSQL
Wallet API
```

---

# Database Initialization

Database schema is automatically loaded using:

```
/docker-entrypoint-initdb.d
```

The `migrations` folder is mounted into the container.

```
./migrations:/docker-entrypoint-initdb.d
```

Postgres will execute all `.sql` files on first initialization.

---

# API Endpoints

Base URL

```
http://localhost:8080
```

---

# Create Account

```
POST /accounts
```

Request

```json
{
  "user_id": "11111111-1111-1111-1111-111111111111",
  "type": "wallet"
}
```

Response

```json
{
  "user_id": "uuid"
}
```

---

# Get Balance

```
GET /accounts/{user_id}/balance
```

Example

```
GET /accounts/uuid/balance
```

Response

```json
{
  "user_id": "uuid",
  "balance": 1000
}
```

Balance is calculated from:

```
ledger_entries
```

---

# Transfer Funds

```
POST /transfer
```

Request

```json
{
  "from_user_id": "uuid",
  "to_user_id": "uuid",
  "amount": 500,
  "reference_id": "transfer-001"
}
```

Response

```json
{
  "status": "success"
}
```

---

# Financial Consistency

Transfers must execute inside **database transactions**.

Example pattern:

```
BEGIN;

insert journal entry
insert debit entry
insert credit entry

COMMIT;
```

If any step fails:

```
ROLLBACK
```

This guarantees **atomic financial operations**.

---

# Why Double Entry Matters

A naive wallet design:

```
accounts
balance
```

Problems:

* race conditions
* lost money
* audit difficulty
* no financial traceability

Double-entry systems solve this by keeping an **immutable financial history**.

---

# Security Considerations

Future improvements:

* idempotency keys
* distributed locks
* transaction states
* fraud detection
* rate limiting

---

# Future Improvements

This project can be extended to include:

* Idempotency keys
* Transaction table
* Redis balance cache
* Event-driven architecture
* Fraud detection
* KYC integration
* Payment rails integration

---

# License

MIT

---

# Author

Alexander Tim

---

If you want, I can also help you add **a very impressive section most fintech repos include**, which is:

**“Financial Integrity Guarantees”**

This explains **why the ledger can never lose money**, and it makes the repo look **much more senior-level when recruiters or engineers review it**.
