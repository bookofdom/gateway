CREATE TABLE IF NOT EXISTS `proxy_endpoint_schemas` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `endpoint_id` INTEGER NOT NULL,
  `name` TEXT NOT NULL,
  `request_schema_id` INTEGER,
  `request_type` TEXT NOT NULL,
  `request_schema` TEXT,
  `response_same_as_request` BOOLEAN NOT NULL DEFAULT 0,
  `response_schema_id` INTEGER,
  `response_type` TEXT NOT NULL,
  `response_schema` TEXT,
  `data` TEXT,
  UNIQUE (`endpoint_id`, `name`) ON CONFLICT FAIL,
  FOREIGN KEY(`endpoint_id`) REFERENCES `proxy_endpoints`(`id`) ON DELETE CASCADE,
  FOREIGN KEY(`request_schema_id`) REFERENCES `schemas`(`id`) ON DELETE SET NULL,
  FOREIGN KEY(`response_schema_id`) REFERENCES `schemas`(`id`) ON DELETE SET NULL
);
