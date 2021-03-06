CREATE TABLE IF NOT EXISTS `proxy_endpoints` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `api_id` INTEGER NOT NULL,
  `endpoint_group_id` INTEGER,
  `environment_id` INTEGER NOT NULL,
  `name` TEXT NOT NULL,
  `description` TEXT,
  `active` BOOLEAN NOT NULL DEFAULT 1,
  `cors_enabled` BOOLEAN NOT NULL DEFAULT 1,
  `routes` TEXT,
  UNIQUE (`api_id`, `name`) ON CONFLICT FAIL,
  FOREIGN KEY(`api_id`) REFERENCES `apis`(`id`) ON DELETE CASCADE,
  FOREIGN KEY(`endpoint_group_id`) REFERENCES `endpoint_groups`(`id`) ON DELETE SET NULL,
  FOREIGN KEY(`environment_id`) REFERENCES `environments`(`id`)
);
