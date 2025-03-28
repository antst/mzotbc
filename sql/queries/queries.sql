
-- name: UpsertSensorValue :exec
INSERT INTO sensor (sensor_name, value, updated_at)
VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT (sensor_name)
    DO UPDATE SET value      = excluded.value,
                  updated_at = CURRENT_TIMESTAMP;

-- name: GetSensorValue :one
SELECT value
FROM sensor
WHERE sensor_name = ?;

-- name: UpsertZoneSetpoint :exec
INSERT INTO zone(zone_name, setpoint, updated_at)
VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(zone_name) DO UPDATE SET setpoint=excluded.setpoint,
                                     updated_at=CURRENT_TIMESTAMP;

-- name: GetZoneSetpoint :one
SELECT setpoint
from zone
where zone_name=?;


-- name: UpsertControllerValue :exec
INSERT INTO controller(name, value, updated_at)
VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(name) DO UPDATE SET value=excluded.value,
                                updated_at=CURRENT_TIMESTAMP;

-- name: GetControllerValue :one
SELECT value from controller where name=?;