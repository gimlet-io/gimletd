package customScm

type NonImpersonatedTokenManager interface {
	Token() (string, string, error)
}
