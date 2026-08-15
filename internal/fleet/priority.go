package fleet

import (
	"fmt"
	"sort"

	"gridflex/internal/model"
)

type Priority struct {
	VehicleID          string
	NeedWh             int64
	RemainingIntervals int
	RequiredPowerW     int64
	FairnessPPM        int64
	Weight             int64
	DepartureIndex     int
}

func (f *Fleet) ChargePriorities(index int, intervalMinutes int64, reserveWh int64) ([]Priority, error) {
	result := make([]Priority, 0, len(f.ids))
	for _, id := range f.ids {
		state := f.states[id]
		vehicle := state.Vehicle
		if index < vehicle.ArrivalIndex || index >= vehicle.DepartureIndex {
			continue
		}
		need, err := f.ChargeNeedWh(id, reserveWh)
		if err != nil {
			return nil, err
		}
		if need == 0 {
			continue
		}
		remaining := vehicle.DepartureIndex - index
		gridNeed, err := model.GridForChargeCeil(need, vehicle.ChargeEfficiencyPPM)
		if err != nil {
			return nil, err
		}
		requiredW, err := model.EnergyToPowerFloor(gridNeed, intervalMinutes*int64(remaining))
		if err != nil {
			return nil, err
		}
		if requiredW < 1 {
			requiredW = 1
		}
		initialNeed := vehicle.MinDepartureSOCWh - vehicle.InitialSOCWh
		if initialNeed < 1 {
			initialNeed = 1
		}
		progress := state.SOCWh - vehicle.InitialSOCWh
		if progress < 0 {
			progress = 0
		}
		fairness, err := model.MulDivFloor(progress, model.PartsPerMillion, initialNeed*vehicle.FairnessWeight)
		if err != nil {
			return nil, err
		}
		result = append(result, Priority{
			VehicleID:          id,
			NeedWh:             need,
			RemainingIntervals: remaining,
			RequiredPowerW:     requiredW,
			FairnessPPM:        fairness,
			Weight:             vehicle.FairnessWeight,
			DepartureIndex:     vehicle.DepartureIndex,
		})
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := result[i]
		right := result[j]
		if left.DepartureIndex != right.DepartureIndex {
			return left.DepartureIndex < right.DepartureIndex
		}
		if left.FairnessPPM != right.FairnessPPM {
			return left.FairnessPPM < right.FairnessPPM
		}
		if left.RequiredPowerW != right.RequiredPowerW {
			return left.RequiredPowerW > right.RequiredPowerW
		}
		return left.VehicleID < right.VehicleID
	})
	return result, nil
}

func (f *Fleet) DischargePriorities(index int, reserveWh int64) ([]Priority, error) {
	result := make([]Priority, 0, len(f.ids))
	for _, id := range f.ids {
		state := f.states[id]
		vehicle := state.Vehicle
		if index < vehicle.ArrivalIndex || index >= vehicle.DepartureIndex || !vehicle.AllowDischarge {
			continue
		}
		available, err := f.DischargeAvailableWh(id, reserveWh)
		if err != nil {
			return nil, err
		}
		if available == 0 {
			continue
		}
		result = append(result, Priority{
			VehicleID:          id,
			NeedWh:             available,
			RemainingIntervals: vehicle.DepartureIndex - index,
			Weight:             vehicle.FairnessWeight,
			DepartureIndex:     vehicle.DepartureIndex,
		})
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := result[i]
		right := result[j]
		if left.NeedWh != right.NeedWh {
			return left.NeedWh > right.NeedWh
		}
		if left.DepartureIndex != right.DepartureIndex {
			return left.DepartureIndex > right.DepartureIndex
		}
		return left.VehicleID < right.VehicleID
	})
	return result, nil
}

func (f *Fleet) AssertDeparture(index int) error {
	for _, id := range f.ids {
		state := f.states[id]
		if state.Vehicle.DepartureIndex == index && state.SOCWh < state.Vehicle.MinDepartureSOCWh {
			return fmt.Errorf("vehicle %q departs at SOC %d below target %d", id, state.SOCWh, state.Vehicle.MinDepartureSOCWh)
		}
	}
	return nil
}

func (f *Fleet) AllDepartureTargetsMet() bool {
	for _, id := range f.ids {
		state := f.states[id]
		if state.SOCWh < state.Vehicle.MinDepartureSOCWh {
			return false
		}
	}
	return true
}
