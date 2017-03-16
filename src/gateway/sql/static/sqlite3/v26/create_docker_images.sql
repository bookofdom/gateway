CREATE TABLE IF NOT EXISTS `docker_images` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `created_at` DATETIME,
  `updated_at` DATETIME,
  `client_id` TEXT NOT NULL,
  `name` TEXT NOT NULL,
  UNIQUE (`client_id`, `name`) ON CONFLICT FAIL
);
CREATE INDEX idx_docker_images_updated_at ON docker_images(updated_at);
CREATE INDEX idx_docker_images_client_id ON docker_images(client_id);
CREATE INDEX idx_docker_images_name ON docker_images(name);
