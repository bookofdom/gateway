ALTER TABLE `accounts` RENAME TO `_accounts`;

CREATE TABLE `accounts` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `name` TEXT UNIQUE NOT NULL
);

INSERT INTO `accounts`
SELECT `id`, `name`
FROM `_accounts`;

DROP TABLE `_accounts`;
