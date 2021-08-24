package store

import (
	"database/sql"
	"github.com/gimlet-io/gimletd/model"
	queries "github.com/gimlet-io/gimletd/store/sql"
	"github.com/russross/meddler"
)

func (db *Store) SaveOrUpdateGitopsCommit(gitopsCommit model.GitopsCommit) error {
	stmt := queries.Stmt(db.driver, queries.SelectGitopsCommitBySha)
	savedGitopsCommit := new(model.GitopsCommit)
	err := meddler.QueryRow(db, savedGitopsCommit, stmt, gitopsCommit.Sha)
	if err == sql.ErrNoRows {
		return meddler.Insert(db, "gitops_commits", gitopsCommit)
	} else if err != nil {
		return err
	}

	savedGitopsCommit.Status = gitopsCommit.Status
	savedGitopsCommit.StatusDesc = gitopsCommit.StatusDesc
	return meddler.Update(db, "gitops_commits", savedGitopsCommit)
}
