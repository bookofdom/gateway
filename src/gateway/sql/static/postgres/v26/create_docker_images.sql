CREATE TABLE IF NOT EXISTS "docker_images" (
  "id" SERIAL PRIMARY KEY,
  "created_at" TIMESTAMPTZ,
  "updated_at" TIMESTAMPTZ,
  "client_id" TEXT NOT NULL,
  "name" TEXT NOT NULL,
  UNIQUE ("client_id", "name")
);
CREATE INDEX idx_docker_images_updated_at ON docker_images USING btree(updated_at);
CREATE INDEX idx_docker_images_client_id ON docker_images USING btree(client_id);
CREATE INDEX idx_docker_images_name ON docker_images USING btree(name);
