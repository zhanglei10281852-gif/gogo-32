package model

import (
	"errors"
	"math"
)

const PartsPerMillion int64 = 1_000_000
const WattHoursPerKWh int64 = 1_000
const MinutesPerHour int64 = 60

func Min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func Max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func Clamp64(value, low, high int64) int64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func Abs64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func Sign64(value int64) int64 {
	if value < 0 {
		return -1
	}
	if value > 0 {
		return 1
	}
	return 0
}

func MulDivFloor(a, b, divisor int64) (int64, error) {
	if divisor <= 0 {
		return 0, errors.New("divisor must be positive")
	}
	if a < 0 || b < 0 {
		return 0, errors.New("floor operands must be non-negative")
	}
	if a != 0 && b > math.MaxInt64/a {
		return 0, errors.New("integer multiplication overflow")
	}
	return a * b / divisor, nil
}

func MulDivCeil(a, b, divisor int64) (int64, error) {
	if divisor <= 0 {
		return 0, errors.New("divisor must be positive")
	}
	if a < 0 || b < 0 {
		return 0, errors.New("ceiling operands must be non-negative")
	}
	if a != 0 && b > math.MaxInt64/a {
		return 0, errors.New("integer multiplication overflow")
	}
	product := a * b
	if product == 0 {
		return 0, nil
	}
	return 1 + (product-1)/divisor, nil
}

func PowerToEnergyFloor(powerW, minutes int64) (int64, error) {
	return MulDivFloor(powerW, minutes, MinutesPerHour)
}

func PowerToEnergyCeil(powerW, minutes int64) (int64, error) {
	return MulDivCeil(powerW, minutes, MinutesPerHour)
}

func EnergyToPowerFloor(energyWh, minutes int64) (int64, error) {
	return MulDivFloor(energyWh, MinutesPerHour, minutes)
}

func ApplyChargeEfficiency(gridWh, efficiencyPPM int64) (int64, error) {
	return MulDivFloor(gridWh, efficiencyPPM, PartsPerMillion)
}

func GridForChargeCeil(batteryWh, efficiencyPPM int64) (int64, error) {
	return MulDivCeil(batteryWh, PartsPerMillion, efficiencyPPM)
}

func GridFromDischargeFloor(batteryWh, efficiencyPPM int64) (int64, error) {
	return MulDivFloor(batteryWh, efficiencyPPM, PartsPerMillion)
}

func BatteryForExportCeil(gridWh, efficiencyPPM int64) (int64, error) {
	return MulDivCeil(gridWh, PartsPerMillion, efficiencyPPM)
}

func CostMicro(energyWh, priceMicroPerKWh int64) (int64, error) {
	return MulDivFloor(energyWh, priceMicroPerKWh, WattHoursPerKWh)
}

func CarbonGrams(energyWh, gramsPerKWh int64) (int64, error) {
	return MulDivFloor(energyWh, gramsPerKWh, WattHoursPerKWh)
}
