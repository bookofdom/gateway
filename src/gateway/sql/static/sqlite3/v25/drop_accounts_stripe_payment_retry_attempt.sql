ALTER TABLE `accounts` RENAME TO `_accounts`;

CREATE TABLE `accounts` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `created_at` DATETIME,
  `updated_at` DATETIME,
  `name` TEXT NOT NULL,
  `plan_id` INTEGER,
  `stripe_customer_id` TEXT,
  `stripe_subscription_id` TEXT,
  UNIQUE (`stripe_customer_id`) ON CONFLICT FAIL,
  UNIQUE (`stripe_subscription_id`) ON CONFLICT FAIL,
  FOREIGN KEY(`plan_id`) REFERENCES `plans`(`id`) ON DELETE SET NULL
);
CREATE INDEX idx_account_stripe_customer_id ON accounts(stripe_customer_id);
CREATE INDEX idx_account_stripe_subscription_id ON accounts(stripe_subscription_id);

INSERT INTO `accounts`
SELECT `id`, `created_at`, `updated_at`, `name`, `plan_id`, `stripe_customer_id`, `stripe_subscription_id`
FROM `_accounts`;

DROP TABLE `_accounts`;
