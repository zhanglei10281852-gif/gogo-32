package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"gridflex/internal/config"
	"gridflex/internal/report"
)

const usageText = `gridflex - deterministic offline fleet energy planner

Usage:
  gridflex validate -input scenario.json
  gridflex plan     -input scenario.json -out artifacts
  gridflex report   -dir artifacts -format text|json
  gridflex run      -input scenario.json -out artifacts -format text|json
`

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "gridflex:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		_, _ = io.WriteString(stderr, usageText)
		return errors.New("command is required")
	}
	switch args[0] {
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "plan":
		return runPlan(args[1:], stdout, stderr)
	case "report":
		return runReport(args[1:], stdout, stderr)
	case "run":
		return runAll(args[1:], stdout, stderr)
	case "help", "-help", "--help", "-h":
		_, err := io.WriteString(stdout, usageText)
		return err
	default:
		_, _ = io.WriteString(stderr, usageText)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runValidate(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	input := flags.String("input", "", "path to scenario JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("validate does not accept positional arguments")
	}
	if *input == "" {
		return errors.New("validate requires -input")
	}
	scenario, err := config.LoadFile(*input)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "valid scenario %s: %d intervals, %d vehicles\n", scenario.ScenarioID, scenario.IntervalCount, len(scenario.Vehicles))
	return err
}

func runPlan(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("plan", flag.ContinueOnError)
	flags.SetOutput(stderr)
	input := flags.String("input", "", "path to scenario JSON")
	output := flags.String("out", "artifacts", "artifact directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("plan does not accept positional arguments")
	}
	if *input == "" {
		return errors.New("plan requires -input")
	}
	result, err := executePlan(*input, *output)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "planned scenario %s: feasible=%t audit=%s\n", result.scenarioID, result.feasible, result.auditRoot)
	return err
}

func runReport(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("report", flag.ContinueOnError)
	flags.SetOutput(stderr)
	directory := flags.String("dir", "artifacts", "artifact directory")
	formatValue := flags.String("format", "text", "text or json")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("report does not accept positional arguments")
	}
	format, err := report.ParseFormat(*formatValue)
	if err != nil {
		return err
	}
	return renderStored(*directory, format, stdout)
}

func runAll(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(stderr)
	input := flags.String("input", "", "path to scenario JSON")
	output := flags.String("out", "artifacts", "artifact directory")
	formatValue := flags.String("format", "text", "text or json")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("run does not accept positional arguments")
	}
	if *input == "" {
		return errors.New("run requires -input")
	}
	format, err := report.ParseFormat(*formatValue)
	if err != nil {
		return err
	}
	if _, err := executePlan(*input, *output); err != nil {
		return err
	}
	return renderStored(*output, format, stdout)
}
