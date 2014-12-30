CREATE TABLE IF NOT EXISTS `proxy_endpoints` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `api_id` INTEGER NOT NULL,
  `group_id` INTEGER,
  `environment_id` INTEGER NOT NULL,
  `name` TEXT NOT NULL,
  `description` TEXT,
  `active` BOOLEAN NOT NULL DEFAULT 1,
  `cors_enabled` BOOLEAN NOT NULL DEFAULT 1,
  `cors_allow_override` TEXT,
  FOREIGN KEY(`api_id`) REFERENCES `apis`(`id`) ON DELETE CASCADE,
  FOREIGN KEY(`group_id`) REFERENCES `endpoint_groups`(`id`) ON DELETE SET NULL,
  FOREIGN KEY(`environment_id`) REFERENCES `environments`(`id`) ON DELETE RESTRICT
);
