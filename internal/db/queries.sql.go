// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: queries.sql

package db

import (
	"context"
)

const getControllerValue = `-- name: GetControllerValue :one
SELECT value from controller where name=?
`

func (q *Queries) GetControllerValue(ctx context.Context, name string) (string, error) {
	row := q.db.QueryRowContext(ctx, getControllerValue, name)
	var value string
	err := row.Scan(&value)
	return value, err
}

const getSensorValue = `-- name: GetSensorValue :one
SELECT value
FROM sensor
WHERE sensor_name = ?
`

func (q *Queries) GetSensorValue(ctx context.Context, sensorName string) (float64, error) {
	row := q.db.QueryRowContext(ctx, getSensorValue, sensorName)
	var value float64
	err := row.Scan(&value)
	return value, err
}

const getZoneSetpoint = `-- name: GetZoneSetpoint :one
SELECT setpoint
from zone
where zone_name=?
`

func (q *Queries) GetZoneSetpoint(ctx context.Context, zoneName string) (float64, error) {
	row := q.db.QueryRowContext(ctx, getZoneSetpoint, zoneName)
	var setpoint float64
	err := row.Scan(&setpoint)
	return setpoint, err
}

const upsertControllerValue = `-- name: UpsertControllerValue :exec
INSERT INTO controller(name, value, updated_at)
VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(name) DO UPDATE SET value=excluded.value,
                                updated_at=CURRENT_TIMESTAMP
`

type UpsertControllerValueParams struct {
	Name  string
	Value string
}

func (q *Queries) UpsertControllerValue(ctx context.Context, arg UpsertControllerValueParams) error {
	_, err := q.db.ExecContext(ctx, upsertControllerValue, arg.Name, arg.Value)
	return err
}

const upsertSensorValue = `-- name: UpsertSensorValue :exec
INSERT INTO sensor (sensor_name, value, updated_at)
VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT (sensor_name)
    DO UPDATE SET value      = excluded.value,
                  updated_at = CURRENT_TIMESTAMP
`

type UpsertSensorValueParams struct {
	SensorName string
	Value      float64
}

func (q *Queries) UpsertSensorValue(ctx context.Context, arg UpsertSensorValueParams) error {
	_, err := q.db.ExecContext(ctx, upsertSensorValue, arg.SensorName, arg.Value)
	return err
}

const upsertZoneSetpoint = `-- name: UpsertZoneSetpoint :exec
INSERT INTO zone(zone_name, setpoint, updated_at)
VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(zone_name) DO UPDATE SET setpoint=excluded.setpoint,
                                     updated_at=CURRENT_TIMESTAMP
`

type UpsertZoneSetpointParams struct {
	ZoneName string
	Setpoint float64
}

func (q *Queries) UpsertZoneSetpoint(ctx context.Context, arg UpsertZoneSetpointParams) error {
	_, err := q.db.ExecContext(ctx, upsertZoneSetpoint, arg.ZoneName, arg.Setpoint)
	return err
}
