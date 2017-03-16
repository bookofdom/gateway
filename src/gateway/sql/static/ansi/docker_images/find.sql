SELECT
  docker_images.id as id,
  docker_images.created_at as created_at,
  docker_images.updated_at as updated_at,
  docker_images.client_id as client_id,
  docker_images.name as name
FROM docker_images
WHERE docker_images.name = ?
  AND docker_images.client_id = ?;
