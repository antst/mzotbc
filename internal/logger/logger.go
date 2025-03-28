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

package logger

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.SugaredLogger
	dlevel = zap.NewAtomicLevelAt(zapcore.DebugLevel)
)

func init() {
	cfg := zap.Config{
		Level:            dlevel,
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stdout"},
		// NOTE: set this false to enable stack trace
		DisableStacktrace: true,
	}

	l, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	logger = l.Sugar()

	L().Debugf("Logger initialized")
}

func L() *zap.SugaredLogger {
	if logger == nil {
		panic("Logger is not initialized")
	}
	return logger
}

func Close() {
	if err := L().Sync(); err != nil {
		L().Error(errors.WithMessage(err, "failed to close logger"))
	}
}

func SetLogLevel(level zapcore.Level) {
	dlevel.SetLevel(level)
}
