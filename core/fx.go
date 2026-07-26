package core

import (
	"github.com/KNattawat89/hospital-middleware-system/core/patient"
	"github.com/KNattawat89/hospital-middleware-system/core/staff"
	"go.uber.org/fx"
)

var Modules = fx.Module(
	"core",
	patient.Module,
	staff.Module,
)
