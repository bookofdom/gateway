UPDATE hosts
SET name = ?,
    hostname = ?,
    cert = ?,
    private_key = ?,
    force_ssl = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE hosts.id = ?
  AND hosts.api_id IN
    (SELECT id FROM apis WHERE id = ? AND account_id = ?)
