package main

import (
	"fmt"
	"io"

	"gridflex/internal/config"
	"gridflex/internal/dispatch"
	"gridflex/internal/report"
	"gridflex/internal/settlement"
	"gridflex/internal/store"
)

type planResult struct {
	scenarioID string
	feasible   bool
	auditRoot  string
}

func executePlan(inputPath, outputDirectory string) (planResult, error) {
	scenario, err := config.LoadFile(inputPath)
	if err != nil {
		return planResult{}, err
	}
	baseline, err := dispatch.BuildBaseline(scenario)
	if err != nil {
		return planResult{}, fmt.Errorf("build baseline: %w", err)
	}
	planner, err := dispatch.New(scenario)
	if err != nil {
		return planResult{}, fmt.Errorf("create planner: %w", err)
	}
	plan, err := planner.Plan()
	if err != nil {
		return planResult{}, fmt.Errorf("create plan: %w", err)
	}
	engine, err := settlement.New(scenario)
	if err != nil {
		return planResult{}, fmt.Errorf("create settlement engine: %w", err)
	}
	settled, err := engine.Settle(baseline, plan)
	if err != nil {
		return planResult{}, fmt.Errorf("settle plan: %w", err)
	}
	if err := settlement.Reconcile(settled); err != nil {
		return planResult{}, fmt.Errorf("reconcile settlement: %w", err)
	}
	storage, err := store.New(outputDirectory)
	if err != nil {
		return planResult{}, err
	}
	if err := storage.SaveInput(scenario); err != nil {
		return planResult{}, err
	}
	if err := storage.SaveBaseline(baseline); err != nil {
		return planResult{}, err
	}
	if err := storage.SavePlan(plan); err != nil {
		return planResult{}, err
	}
	if err := storage.SaveSettlement(settled); err != nil {
		return planResult{}, err
	}
	audit, err := storage.Snapshot()
	if err != nil {
		return planResult{}, err
	}
	if err := storage.VerifyAudit(); err != nil {
		return planResult{}, fmt.Errorf("verify new audit: %w", err)
	}
	return planResult{scenarioID: scenario.ScenarioID, feasible: plan.Feasible, auditRoot: audit.RootSHA256}, nil
}

func renderStored(directory string, format report.Format, writer io.Writer) error {
	storage, err := store.New(directory)
	if err != nil {
		return err
	}
	if err := storage.VerifyAudit(); err != nil {
		return fmt.Errorf("verify audit: %w", err)
	}
	plan, err := storage.LoadPlan()
	if err != nil {
		return err
	}
	settled, err := storage.LoadSettlement()
	if err != nil {
		return err
	}
	audit, err := storage.LoadAudit()
	if err != nil {
		return err
	}
	document, err := report.Document(plan, settled, audit)
	if err != nil {
		return err
	}
	return report.Write(writer, format, document)
}
