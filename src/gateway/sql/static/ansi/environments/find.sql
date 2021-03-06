SELECT
  environments.api_id as api_id,
  environments.id as id,
  environments.name as name,
  environments.description as description,
  environments.data as data,
  environments.session_type as session_type,
  environments.session_header as session_header,
  environments.session_name as session_name,
  environments.session_auth_key as session_auth_key,
  environments.session_encryption_key as session_encryption_key,
  environments.session_auth_key_rotate as session_auth_key_rotate,
  environments.session_encryption_key_rotate as session_encryption_key_rotate,
  environments.show_javascript_errors as show_javascript_errors
FROM environments, apis
WHERE environments.id = ?
  AND environments.api_id = ?
  AND environments.api_id = apis.id
  AND apis.account_id = ?;
