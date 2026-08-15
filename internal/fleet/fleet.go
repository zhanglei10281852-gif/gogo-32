package fleet

import (
	"errors"
	"fmt"
	"sort"

	"gridflex/internal/model"
)

type State struct {
	Vehicle             model.Vehicle
	SOCWh               int64
	ChargedBatteryWh    int64
	DischargedBatteryWh int64
	GridImportWh        int64
	GridExportWh        int64
}

type Fleet struct {
	states map[string]*State
	ids    []string
}

func New(vehicles []model.Vehicle) (*Fleet, error) {
	if len(vehicles) == 0 {
		return nil, errors.New("fleet must contain at least one vehicle")
	}
	states := make(map[string]*State, len(vehicles))
	ids := make([]string, 0, len(vehicles))
	for _, vehicle := range vehicles {
		if vehicle.ID == "" {
			return nil, errors.New("vehicle ID is required")
		}
		if _, exists := states[vehicle.ID]; exists {
			return nil, fmt.Errorf("duplicate vehicle ID %q", vehicle.ID)
		}
		if vehicle.InitialSOCWh < vehicle.MinSOCWh || vehicle.InitialSOCWh > vehicle.MaxSOCWh {
			return nil, fmt.Errorf("vehicle %q initial SOC outside bounds", vehicle.ID)
		}
		copyVehicle := vehicle
		states[vehicle.ID] = &State{Vehicle: copyVehicle, SOCWh: vehicle.InitialSOCWh}
		ids = append(ids, vehicle.ID)
	}
	sort.Strings(ids)
	return &Fleet{states: states, ids: ids}, nil
}

func (f *Fleet) IDs() []string {
	result := make([]string, len(f.ids))
	copy(result, f.ids)
	return result
}

func (f *Fleet) Len() int {
	return len(f.ids)
}

func (f *Fleet) State(id string) (State, error) {
	state, ok := f.states[id]
	if !ok {
		return State{}, fmt.Errorf("unknown vehicle %q", id)
	}
	return *state, nil
}

func (f *Fleet) SOC(id string) (int64, error) {
	state, ok := f.states[id]
	if !ok {
		return 0, fmt.Errorf("unknown vehicle %q", id)
	}
	return state.SOCWh, nil
}

func (f *Fleet) AvailableIDs(index int) []string {
	result := make([]string, 0, len(f.ids))
	for _, id := range f.ids {
		vehicle := f.states[id].Vehicle
		if index >= vehicle.ArrivalIndex && index < vehicle.DepartureIndex {
			result = append(result, id)
		}
	}
	return result
}

func (f *Fleet) ChargeNeedWh(id string, reserveWh int64) (int64, error) {
	state, ok := f.states[id]
	if !ok {
		return 0, fmt.Errorf("unknown vehicle %q", id)
	}
	target := state.Vehicle.MinDepartureSOCWh + reserveWh
	if target > state.Vehicle.MaxSOCWh {
		target = state.Vehicle.MaxSOCWh
	}
	if state.SOCWh >= target {
		return 0, nil
	}
	return target - state.SOCWh, nil
}

func (f *Fleet) CapacityRemainingWh(id string) (int64, error) {
	state, ok := f.states[id]
	if !ok {
		return 0, fmt.Errorf("unknown vehicle %q", id)
	}
	return state.Vehicle.MaxSOCWh - state.SOCWh, nil
}

func (f *Fleet) DischargeAvailableWh(id string, reserveWh int64) (int64, error) {
	state, ok := f.states[id]
	if !ok {
		return 0, fmt.Errorf("unknown vehicle %q", id)
	}
	floor := state.Vehicle.MinDepartureSOCWh + reserveWh
	if floor < state.Vehicle.MinSOCWh {
		floor = state.Vehicle.MinSOCWh
	}
	if floor > state.Vehicle.MaxSOCWh {
		floor = state.Vehicle.MaxSOCWh
	}
	if state.SOCWh <= floor {
		return 0, nil
	}
	return state.SOCWh - floor, nil
}

func (f *Fleet) Charge(id string, gridWh int64) (batteryWh int64, before int64, after int64, err error) {
	if gridWh < 0 {
		return 0, 0, 0, errors.New("charge grid energy must be non-negative")
	}
	state, ok := f.states[id]
	if !ok {
		return 0, 0, 0, fmt.Errorf("unknown vehicle %q", id)
	}
	batteryWh, err = model.ApplyChargeEfficiency(gridWh, state.Vehicle.ChargeEfficiencyPPM)
	if err != nil {
		return 0, 0, 0, err
	}
	if state.SOCWh+batteryWh > state.Vehicle.MaxSOCWh {
		return 0, 0, 0, fmt.Errorf("vehicle %q charge exceeds max SOC", id)
	}
	before = state.SOCWh
	state.SOCWh += batteryWh
	state.ChargedBatteryWh += batteryWh
	state.GridImportWh += gridWh
	return batteryWh, before, state.SOCWh, nil
}

func (f *Fleet) Discharge(id string, gridWh int64) (batteryWh int64, before int64, after int64, err error) {
	if gridWh < 0 {
		return 0, 0, 0, errors.New("discharge grid energy must be non-negative")
	}
	state, ok := f.states[id]
	if !ok {
		return 0, 0, 0, fmt.Errorf("unknown vehicle %q", id)
	}
	if !state.Vehicle.AllowDischarge {
		return 0, 0, 0, fmt.Errorf("vehicle %q does not allow discharge", id)
	}
	batteryWh, err = model.BatteryForExportCeil(gridWh, state.Vehicle.DischargeEfficiencyPPM)
	if err != nil {
		return 0, 0, 0, err
	}
	if state.SOCWh-batteryWh < state.Vehicle.MinSOCWh {
		return 0, 0, 0, fmt.Errorf("vehicle %q discharge violates min SOC", id)
	}
	before = state.SOCWh
	state.SOCWh -= batteryWh
	state.DischargedBatteryWh += batteryWh
	state.GridExportWh += gridWh
	return batteryWh, before, state.SOCWh, nil
}

func (f *Fleet) Summaries() []model.VehicleSummary {
	result := make([]model.VehicleSummary, 0, len(f.ids))
	for _, id := range f.ids {
		state := f.states[id]
		need := state.Vehicle.MinDepartureSOCWh - state.Vehicle.InitialSOCWh
		if need < 1 {
			need = 1
		}
		progress := state.SOCWh - state.Vehicle.InitialSOCWh
		if progress < 0 {
			progress = 0
		}
		score, _ := model.MulDivFloor(progress, model.PartsPerMillion, need)
		result = append(result, model.VehicleSummary{
			VehicleID:           id,
			InitialSOCWh:        state.Vehicle.InitialSOCWh,
			FinalSOCWh:          state.SOCWh,
			TargetSOCWh:         state.Vehicle.MinDepartureSOCWh,
			ChargedBatteryWh:    state.ChargedBatteryWh,
			DischargedBatteryWh: state.DischargedBatteryWh,
			GridImportWh:        state.GridImportWh,
			GridExportWh:        state.GridExportWh,
			FairnessScorePPM:    score,
		})
	}
	return result
}

func (f *Fleet) Clone() *Fleet {
	states := make(map[string]*State, len(f.states))
	for id, state := range f.states {
		copyState := *state
		states[id] = &copyState
	}
	return &Fleet{states: states, ids: append([]string(nil), f.ids...)}
}
