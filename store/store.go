package store

import internal "gridflex/internal/store"

const InputFile = internal.InputFile
const BaselineFile = internal.BaselineFile
const PlanFile = internal.PlanFile
const SettlementFile = internal.SettlementFile
const AuditFile = internal.AuditFile
const AuditAlgorithm = internal.AuditAlgorithm
const AlgorithmVersion = internal.AlgorithmVersion

type Store = internal.Store
type ArtifactInfo = internal.ArtifactInfo

func New(directory string) (*Store, error) {
	return internal.New(directory)
}

func HashBytes(data []byte) string {
	return internal.HashBytes(data)
}
