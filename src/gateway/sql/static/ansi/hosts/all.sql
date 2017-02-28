SELECT
  hosts.api_id as api_id,
  hosts.id as id,
  hosts.name as name,
  hosts.hostname as hostname,
  hosts.cert as cert,
  hosts.private_key as private_key,
  hosts.force_ssl as force_ssl
FROM hosts, apis
WHERE hosts.api_id = ?
  AND hosts.api_id = apis.id
  AND apis.account_id = ?
ORDER BY hosts.name ASC;
