INSERT INTO docker_images (
  name, client_id,
  created_at
)
VALUES (
  ?, ?,
  CURRENT_TIMESTAMP
)
