INSERT INTO push_devices (
  push_channel_id,
  name,
  type,
  token,
  expires,
  data
)
VALUES (
  (SELECT push_channels.id
    FROM push_channels, remote_endpoints, apis
    WHERE push_channels.id = ?
      AND push_channels.remote_endpoint_id = ?
      AND push_channels.remote_endpoint_id = remote_endpoints.id
      AND remote_endpoints.api_id = ?
      AND remote_endpoints.api_id = apis.id
      AND apis.account_id = ?),
  ?,
  ?,
  ?,
  ?,
  ?
)
