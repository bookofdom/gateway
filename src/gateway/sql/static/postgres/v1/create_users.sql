CREATE TABLE IF NOT EXISTS "users" (
  "id" SERIAL PRIMARY KEY,
  "account_id" INTEGER NOT NULL,
  "name" TEXT NOT NULL,
  "email" TEXT UNIQUE NOT NULL,
  "hashed_password" TEXT NOT NULL,
  FOREIGN KEY("account_id") REFERENCES "accounts"("id") ON DELETE CASCADE
);
