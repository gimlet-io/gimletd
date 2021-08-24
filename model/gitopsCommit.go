package model

const Progressing = "Progressing"
const ReconciliationSucceeded = "ReconciliationSucceeded"
const ValidationFailed = "ValidationFailed"
const ReconciliationFailed = "ReconciliationFailed"

type GitopsCommit struct {
	ID         string `json:"id,omitempty"  meddler:"id"`
	Sha        string `json:"sha,omitempty"  meddler:"sha"`
	Status     string `json:"status,omitempty"  meddler:"status"`
	StatusDesc string `json:"statusDesc,omitempty"  meddler:"statusDesc"`
}
