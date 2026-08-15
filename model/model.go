package model

import internal "gridflex/internal/model"

const PartsPerMillion = internal.PartsPerMillion
const WattHoursPerKWh = internal.WattHoursPerKWh
const MinutesPerHour = internal.MinutesPerHour

type Scenario = internal.Scenario
type Site = internal.Site
type SiteCapacityEvent = internal.SiteCapacityEvent
type Options = internal.Options
type TariffPeriod = internal.TariffPeriod
type ForecastPoint = internal.ForecastPoint
type Vehicle = internal.Vehicle
type Interval = internal.Interval
type BaselineInterval = internal.BaselineInterval
type Baseline = internal.Baseline
type Action = internal.Action
type SiteInterval = internal.SiteInterval
type VehicleSummary = internal.VehicleSummary
type Plan = internal.Plan
type SettlementInterval = internal.SettlementInterval
type VehicleSettlement = internal.VehicleSettlement
type Settlement = internal.Settlement
type AuditEntry = internal.AuditEntry
type AuditSnapshot = internal.AuditSnapshot
type ReportDocument = internal.ReportDocument
type EnergyTotals = internal.EnergyTotals

var Min64 = internal.Min64
var SiteCapacityAt = internal.SiteCapacityAt
var Max64 = internal.Max64
var Clamp64 = internal.Clamp64
var Abs64 = internal.Abs64
var Sign64 = internal.Sign64
var MulDivFloor = internal.MulDivFloor
var MulDivCeil = internal.MulDivCeil
var PowerToEnergyFloor = internal.PowerToEnergyFloor
var PowerToEnergyCeil = internal.PowerToEnergyCeil
var EnergyToPowerFloor = internal.EnergyToPowerFloor
var ApplyChargeEfficiency = internal.ApplyChargeEfficiency
var GridForChargeCeil = internal.GridForChargeCeil
var GridFromDischargeFloor = internal.GridFromDischargeFloor
var BatteryForExportCeil = internal.BatteryForExportCeil
var CostMicro = internal.CostMicro
var CarbonGrams = internal.CarbonGrams
var AggregateActions = internal.AggregateActions
var ActionsByVehicle = internal.ActionsByVehicle
var ActionsByInterval = internal.ActionsByInterval
var VehicleSummaryByID = internal.VehicleSummaryByID
var SortedVehicleIDs = internal.SortedVehicleIDs
