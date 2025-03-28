/*
 * Copyright (c) 2023. Anton Starikov -- All Rights Reserved
 *
 * This file is part of MZOTBC project.
 *
 * MZOTBC is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as the Free Software Foundation,
 * either version 3 of the License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package db

import (
	"database/sql"
	_ "embed"

	"github.com/antst/mzotbc/internal/logger"

	_ "github.com/mattn/go-sqlite3"

	"github.com/antst/mzotbc/sql/schema"
)

func OpenDatabase(dbFile string) *Queries {
	sqlDB, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		logger.L().Panic(err)
	}

	if err := sqlDB.Ping(); err != nil {
		logger.L().Panicf("%s, %w", dbFile, err)
	}

	sqlDB.SetMaxOpenConns(100)

	// Create tables if they don't exist
	if _, err := sqlDB.Exec(schema.Schema); err != nil {
		logger.L().Panic(err)
	}

	queries := New(sqlDB)

	return queries
}

// Example functions using the generated code:

// Add similar functions for sensors and controllers
