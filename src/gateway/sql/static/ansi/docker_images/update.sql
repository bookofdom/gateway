UPDATE docker_images
SET
  name = ?,
  client_id = ?,
  updated_at = CURRENT_TIMESTAMP
WHERE docker_images.id = ?;
